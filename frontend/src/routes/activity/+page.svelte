<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getDownloadItems,
		deleteDownloadItem,
		getWantedItems,
		deleteWantedItem,
		searchWantedItem,
		grabRelease,
		getQualityProfiles,
		getIndexers,
		getRequests,
		updateRequest,
		deleteRequest,
		getSystemStatus,
		getGrabHistory,
		getBlocklist,
		removeFromBlocklist,
		getTmdbImageUrl,
		type DownloadItem,
		type WantedItem,
		type QualityProfile,
		type ScoredSearchResult,
		type Request,
		type SystemStatus,
		type GrabHistoryItem,
		type BlocklistEntry
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { statusBar } from '$lib/stores/statusBar';
	import { auth } from '$lib/stores/auth';

	type Filter = 'all' | 'active' | 'wanted' | 'requests' | 'history' | 'blocklist' | 'failed';
	let activeFilter: Filter = $state('all');

	let user = $state<{ role: string } | null>(null);
	auth.subscribe((value) => { user = value; });
	let systemStatus: SystemStatus | null = $state(null);

	let downloads: DownloadItem[] = $state([]);
	let wantedItems: WantedItem[] = $state([]);
	let requests: Request[] = $state([]);
	let profiles: QualityProfile[] = $state([]);
	let grabHistory: GrabHistoryItem[] = $state([]);
	let blocklist: BlocklistEntry[] = $state([]);

	let loading = $state(true);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let processingIds: Set<string> = $state(new Set());
	let searchingIds: Set<number> = $state(new Set());
	let searchResults: Record<number, ScoredSearchResult[]> = $state({});
	let expandedSearchId: number | null = $state(null);
	let grabbingKeys: Set<string> = $state(new Set());

	let activeDownloads = $derived(downloads.filter(d =>
		['queued', 'downloading', 'paused', 'stalled', 'importing', 'import_pending'].includes(d.state)
	));
	let failedDownloads = $derived(downloads.filter(d => ['failed', 'import_blocked'].includes(d.state)));
	let pendingRequests = $derived(requests.filter(r => r.status === 'requested'));

	let counts = $derived({
		active: activeDownloads.length,
		wanted: wantedItems.filter(w => w.monitored).length,
		requests: pendingRequests.length,
		history: grabHistory.length,
		blocklist: blocklist.length,
		failed: failedDownloads.length
	});

	type ActivityItem =
		| { type: 'download'; data: DownloadItem }
		| { type: 'wanted'; data: WantedItem }
		| { type: 'request'; data: Request }
		| { type: 'history'; data: GrabHistoryItem }
		| { type: 'blocklist'; data: BlocklistEntry };

	let filteredItems = $derived.by(() => {
		let items: ActivityItem[] = [];
		downloads.forEach(d => {
			if (activeFilter === 'all' ||
				(activeFilter === 'active' && ['queued', 'downloading', 'paused', 'stalled', 'importing', 'import_pending'].includes(d.state)) ||
				(activeFilter === 'failed' && ['failed', 'import_blocked'].includes(d.state))) {
				items.push({ type: 'download', data: d });
			}
		});
		if (activeFilter === 'all' || activeFilter === 'wanted') {
			wantedItems.forEach(w => items.push({ type: 'wanted', data: w }));
		}
		if (activeFilter === 'all' || activeFilter === 'requests') {
			requests.forEach(r => {
				if (activeFilter === 'all' && r.status !== 'requested') return;
				items.push({ type: 'request', data: r });
			});
		}
		if (activeFilter === 'history') {
			grabHistory.forEach(h => items.push({ type: 'history', data: h }));
		}
		if (activeFilter === 'blocklist') {
			blocklist.forEach(b => items.push({ type: 'blocklist', data: b }));
		}
		return items.sort((a, b) => {
			const priority = (item: ActivityItem) => {
				if (item.type === 'download') {
					if (['downloading', 'importing'].includes(item.data.state)) return 0;
					if (['queued', 'import_pending', 'paused', 'stalled'].includes(item.data.state)) return 1;
					if (['failed', 'import_blocked'].includes(item.data.state)) return 2;
					return 5;
				}
				if (item.type === 'request' && item.data.status === 'requested') return 3;
				if (item.type === 'wanted') return 4;
				return 6;
			};
			return priority(a) - priority(b);
		});
	});

	onMount(async () => {
		await loadAll();
		refreshInterval = setInterval(loadAll, 3000);
	});

	onDestroy(() => {
		if (refreshInterval) clearInterval(refreshInterval);
	});

	async function loadAll() {
		try {
			const [dlData, wantedData, reqData, profData, statusData, histData, blockData] = await Promise.all([
				getDownloadItems(), getWantedItems(), getRequests(), getQualityProfiles(),
				getSystemStatus(), getGrabHistory(100), getBlocklist()
			]);
			downloads = dlData || [];
			wantedItems = wantedData || [];
			requests = reqData || [];
			profiles = profData || [];
			systemStatus = statusData;
			grabHistory = histData || [];
			blocklist = blockData || [];
		} catch (e) {
			console.error('Failed to load activity:', e);
		} finally {
			loading = false;
		}
	}

	async function cancelDownload(id: number) {
		const key = `dl-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await deleteDownloadItem(id);
			downloads = downloads.filter(d => d.id !== id);
			toast.success('Download cancelled');
		} catch (e) {
			toast.error('Failed to cancel download');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function searchWanted(id: number) {
		const item = wantedItems.find(i => i.id === id);
		if (!item) return;
		searchingIds.add(id);
		searchingIds = searchingIds;
		expandedSearchId = id;
		delete searchResults[id];
		try {
			let indexerNames: string[] = [];
			try {
				const indexers = await getIndexers();
				indexerNames = indexers.filter(i => i.enabled).map(i => i.name);
			} catch { indexerNames = ['Indexers']; }
			statusBar.startSearch(item.title, indexerNames);
			for (const name of indexerNames) statusBar.updateIndexer(name, 'searching');
			const results = await searchWantedItem(id);
			searchResults[id] = results;
			searchResults = searchResults;
			statusBar.endSearch(results.length);
		} catch (e) {
			toast.error('Search failed');
			statusBar.endSearch(0);
		} finally {
			searchingIds.delete(id);
			searchingIds = searchingIds;
		}
	}

	async function grabResult(result: ScoredSearchResult, wantedId: number) {
		const key = `${wantedId}-${result.guid}`;
		grabbingKeys.add(key);
		grabbingKeys = grabbingKeys;
		try {
			const response = await grabRelease({
				link: result.link, magnetLink: result.magnetLink,
				indexerType: result.indexerType, category: result.category
			});
			if (response.success) {
				toast.success('Sent to download client');
				if (searchResults[wantedId]) {
					searchResults[wantedId] = searchResults[wantedId].filter(r => r.guid !== result.guid);
				}
			} else {
				toast.error(response.message || 'Grab failed');
			}
		} catch (e) {
			toast.error('Failed to grab');
		} finally {
			grabbingKeys.delete(key);
			grabbingKeys = grabbingKeys;
		}
	}

	async function removeWanted(id: number) {
		const key = `wanted-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await deleteWantedItem(id);
			wantedItems = wantedItems.filter(w => w.id !== id);
			toast.success('Removed from wanted');
		} catch (e) {
			toast.error('Failed to remove');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function approveRequest(id: number) {
		const key = `req-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await updateRequest(id, 'approved');
			await loadAll();
			toast.success('Approved - searching...');
		} catch (e) {
			toast.error('Failed to approve');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function denyRequest(id: number) {
		const key = `req-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await updateRequest(id, 'denied');
			await loadAll();
			toast.info('Request denied');
		} catch (e) {
			toast.error('Failed to deny');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function deleteReq(id: number) {
		const key = `req-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await deleteRequest(id);
			requests = requests.filter(r => r.id !== id);
			toast.success('Request deleted');
		} catch (e) {
			toast.error('Failed to delete');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function unblock(id: number) {
		const key = `block-${id}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await removeFromBlocklist(id);
			blocklist = blocklist.filter(b => b.id !== id);
			toast.success('Removed from blocklist');
		} catch (e) {
			toast.error('Failed to remove');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}

	function formatSpeed(bytes: number): string { return formatBytes(bytes) + '/s'; }

	function formatEta(seconds: number): string {
		if (!seconds || seconds <= 0) return '';
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
	}

	function formatRelativeTime(dateStr: string): string {
		const diff = Date.now() - new Date(dateStr).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		return `${Math.floor(hours / 24)}d ago`;
	}

	function getProfileName(id: number): string { return profiles.find(p => p.id === id)?.name || ''; }

	function getStateColor(state: string): string {
		switch (state) {
			case 'downloading': return 'text-blue-400';
			case 'importing': case 'import_pending': return 'text-yellow-400';
			case 'imported': return 'text-green-400';
			case 'queued': return 'text-gray-400';
			case 'paused': return 'text-yellow-500';
			case 'stalled': return 'text-orange-400';
			case 'failed': case 'import_blocked': return 'text-red-400';
			default: return 'text-text-muted';
		}
	}

	function getStateBg(state: string): string {
		switch (state) {
			case 'downloading': return 'bg-blue-500/20';
			case 'importing': case 'import_pending': return 'bg-yellow-500/20';
			case 'imported': return 'bg-green-500/20';
			case 'failed': case 'import_blocked': return 'bg-red-500/20';
			default: return 'bg-white/5';
		}
	}

	function getHistoryStatusColor(status: string): string {
		switch (status) {
			case 'grabbed': return 'text-blue-400';
			case 'imported': return 'text-green-400';
			case 'failed': return 'text-red-400';
			default: return 'text-text-muted';
		}
	}

	function getHistoryStatusBg(status: string): string {
		switch (status) {
			case 'grabbed': return 'bg-blue-500/20';
			case 'imported': return 'bg-green-500/20';
			case 'failed': return 'bg-red-500/20';
			default: return 'bg-white/5';
		}
	}
</script>

<svelte:head>
	<title>Activity - Outpost</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 space-y-4">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-text-primary">Activity</h1>
		{#if loading}<div class="spinner-sm text-text-muted"></div>{/if}
	</div>

	{#if systemStatus && (systemStatus.runningTasks.length > 0 || activeDownloads.some(d => d.state === 'downloading'))}
		<div class="bg-bg-card border border-border-subtle rounded-xl p-3">
			<div class="flex items-center gap-3">
				<div class="spinner-sm text-blue-400"></div>
				<div class="flex-1 min-w-0">
					{#if systemStatus.runningTasks.length > 0}
						<span class="text-sm text-text-primary">{systemStatus.runningTasks.join(', ')}</span>
					{:else}
						{@const dl = activeDownloads.find(d => d.state === 'downloading')}
						{#if dl}
							<div class="flex items-center justify-between">
								<span class="text-sm text-text-primary truncate">{dl.title}</span>
								<span class="text-xs text-text-muted ml-2">{dl.progress.toFixed(0)}% · {formatSpeed(dl.speed)}</span>
							</div>
							<div class="w-full bg-bg-elevated rounded-full h-1 mt-1.5 overflow-hidden">
								<div class="bg-blue-500 h-full transition-all" style="width: {dl.progress}%"></div>
							</div>
						{/if}
					{/if}
				</div>
			</div>
		</div>
	{/if}

	<div class="flex flex-wrap gap-2">
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all {activeFilter === 'all' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'all'}>All</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'active' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'active'}>
			Active {#if counts.active > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'active' ? 'bg-black/20' : 'bg-blue-500 text-white'}">{counts.active}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'wanted' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'wanted'}>
			Wanted {#if counts.wanted > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'wanted' ? 'bg-black/20' : 'bg-purple-500 text-white'}">{counts.wanted}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'requests' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'requests'}>
			Requests {#if counts.requests > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'requests' ? 'bg-black/20' : 'bg-amber-500 text-white'}">{counts.requests}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'history' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'history'}>
			History {#if counts.history > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'history' ? 'bg-black/20' : 'bg-green-500 text-white'}">{counts.history}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'blocklist' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'blocklist'}>
			Blocklist {#if counts.blocklist > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'blocklist' ? 'bg-black/20' : 'bg-red-500 text-white'}">{counts.blocklist}</span>{/if}
		</button>
		{#if counts.failed > 0}
			<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'failed' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'failed'}>
				Failed <span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'failed' ? 'bg-black/20' : 'bg-red-500 text-white'}">{counts.failed}</span>
			</button>
		{/if}
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-16"><div class="spinner-lg text-amber-400"></div></div>
	{:else if filteredItems.length === 0}
		<div class="glass-card p-12 text-center">
			<div class="w-14 h-14 rounded-full bg-bg-elevated flex items-center justify-center mx-auto mb-3">
				<svg class="w-7 h-7 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 13l4 4L19 7" /></svg>
			</div>
			<p class="text-text-secondary">{#if activeFilter === 'all'}No activity{:else}No {activeFilter} items{/if}</p>
		</div>
	{:else}
		<div class="space-y-2">
			{#each filteredItems as item}
				{#if item.type === 'download'}
					{@const dl = item.data}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4 {['failed', 'import_blocked'].includes(dl.state) ? 'border-l-2 border-l-red-500' : ''}">
						<div class="flex items-start gap-3">
							<div class="w-8 h-8 rounded-lg {getStateBg(dl.state)} flex items-center justify-center flex-shrink-0 mt-0.5">
								{#if dl.state === 'downloading'}
									<svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
								{:else if dl.state === 'importing' || dl.state === 'import_pending'}
									<div class="spinner-sm text-yellow-400"></div>
								{:else if dl.state === 'imported'}
									<svg class="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
								{:else if dl.state === 'failed' || dl.state === 'import_blocked'}
									<svg class="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
								{:else}
									<svg class="w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
								{/if}
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="min-w-0">
										<h3 class="font-medium text-text-primary truncate">{dl.title}</h3>
										<div class="flex items-center gap-2 mt-1 text-xs text-text-muted">
											<span class="{getStateColor(dl.state)} capitalize">{dl.state.replace('_', ' ')}</span>
											{#if dl.size > 0}<span>·</span><span>{formatBytes(dl.size)}</span>{/if}
											<span>·</span><span>{formatRelativeTime(dl.updatedAt)}</span>
										</div>
									</div>
									{#if ['queued', 'downloading', 'paused', 'stalled'].includes(dl.state)}
										<button class="px-2 py-1 text-xs rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors disabled:opacity-50" onclick={() => cancelDownload(dl.id)} disabled={processingIds.has(`dl-${dl.id}`)}>{processingIds.has(`dl-${dl.id}`) ? '...' : 'Cancel'}</button>
									{:else if ['failed', 'import_blocked', 'imported'].includes(dl.state)}
										<button class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50" onclick={() => cancelDownload(dl.id)} disabled={processingIds.has(`dl-${dl.id}`)}>{processingIds.has(`dl-${dl.id}`) ? '...' : 'Remove'}</button>
									{/if}
								</div>
								{#if dl.state === 'downloading'}
									<div class="mt-2">
										<div class="w-full bg-bg-elevated rounded-full h-1.5 overflow-hidden"><div class="bg-blue-500 h-full transition-all duration-300" style="width: {dl.progress}%"></div></div>
										<div class="flex justify-between mt-1 text-xs text-text-muted">
											<span>{dl.progress.toFixed(1)}% · {formatBytes(dl.downloaded)} / {formatBytes(dl.size)}</span>
											<span>{formatSpeed(dl.speed)}{#if dl.eta > 0} · {formatEta(dl.eta)}{/if}</span>
										</div>
									</div>
								{/if}
								{#if dl.errors && dl.errors.length > 0}<p class="text-xs text-red-400 mt-2">{dl.errors[0]}</p>{/if}
								{#if dl.importBlockReason}<p class="text-xs text-orange-400 mt-2">{dl.importBlockReason}</p>{/if}
							</div>
						</div>
					</div>
				{:else if item.type === 'wanted'}
					{@const w = item.data}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4">
						<div class="flex items-start gap-3">
							<div class="w-8 h-8 rounded-lg bg-purple-500/20 flex items-center justify-center flex-shrink-0 mt-0.5">
								<svg class="w-4 h-4 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="min-w-0">
										<h3 class="font-medium text-text-primary truncate">{w.title}{#if w.year}<span class="text-text-muted"> ({w.year})</span>{/if}</h3>
										<div class="flex items-center gap-2 mt-1 text-xs text-text-muted">
											<span class="text-purple-400">Wanted</span><span>·</span><span class="capitalize">{w.type}</span>
											{#if getProfileName(w.qualityProfileId)}<span>·</span><span>{getProfileName(w.qualityProfileId)}</span>{/if}
										</div>
									</div>
									<div class="flex items-center gap-2">
										<button class="px-2 py-1 text-xs rounded-lg bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors disabled:opacity-50" onclick={() => searchWanted(w.id)} disabled={searchingIds.has(w.id)}>{searchingIds.has(w.id) ? 'Searching...' : 'Search'}</button>
										<button class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50" onclick={() => removeWanted(w.id)} disabled={processingIds.has(`wanted-${w.id}`)}>Remove</button>
									</div>
								</div>
								{#if searchResults[w.id] && searchResults[w.id].length > 0 && expandedSearchId === w.id}
									<div class="mt-3 pt-3 border-t border-border-subtle">
										<div class="text-xs text-text-muted mb-2">{searchResults[w.id].length} results</div>
										<div class="space-y-1 max-h-48 overflow-y-auto">
											{#each searchResults[w.id].slice(0, 5) as result}
												{@const key = `${w.id}-${result.guid}`}
												<div class="flex items-center justify-between gap-2 p-2 rounded-lg bg-bg-elevated/50 text-xs">
													<div class="min-w-0 flex-1">
														<p class="truncate text-text-primary" title={result.title}>{result.title}</p>
														<p class="text-text-muted">{result.indexerName} · {formatBytes(result.size)}</p>
													</div>
													<button class="px-2 py-1 rounded bg-green-500/20 text-green-400 hover:bg-green-500/30 disabled:opacity-50 flex-shrink-0" onclick={() => grabResult(result, w.id)} disabled={grabbingKeys.has(key)}>{grabbingKeys.has(key) ? '...' : 'Grab'}</button>
												</div>
											{/each}
										</div>
									</div>
								{:else if searchResults[w.id] && searchResults[w.id].length === 0 && expandedSearchId === w.id}
									<p class="text-xs text-text-muted mt-2">No results found</p>
								{/if}
							</div>
						</div>
					</div>
				{:else if item.type === 'request'}
					{@const r = item.data}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4">
						<div class="flex items-start gap-3">
							{#if r.posterPath}
								<img src={getTmdbImageUrl(r.posterPath, 'w92')} alt="" class="w-10 h-14 rounded-lg object-cover flex-shrink-0" />
							{:else}
								<div class="w-10 h-14 rounded-lg bg-bg-elevated flex items-center justify-center flex-shrink-0">
									<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="min-w-0">
										<h3 class="font-medium text-text-primary truncate">{r.title}{#if r.year}<span class="text-text-muted"> ({r.year})</span>{/if}</h3>
										<div class="flex items-center gap-2 mt-1 text-xs text-text-muted">
											<span class="{r.status === 'requested' ? 'text-amber-400' : r.status === 'approved' ? 'text-green-400' : 'text-text-muted'} capitalize">{r.status}</span>
											<span>·</span><span class="capitalize">{r.type}</span>
											{#if r.username}<span>·</span><span>by {r.username}</span>{/if}
											<span>·</span><span>{formatRelativeTime(r.requestedAt)}</span>
										</div>
									</div>
									{#if r.status === 'requested' && user?.role === 'admin'}
										<div class="flex items-center gap-1">
											<button class="px-2 py-1 text-xs rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors disabled:opacity-50" onclick={() => approveRequest(r.id)} disabled={processingIds.has(`req-${r.id}`)}>Approve</button>
											<button class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50" onclick={() => denyRequest(r.id)} disabled={processingIds.has(`req-${r.id}`)}>Deny</button>
										</div>
									{:else}
										<button class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50" onclick={() => deleteReq(r.id)} disabled={processingIds.has(`req-${r.id}`)}>Remove</button>
									{/if}
								</div>
							</div>
						</div>
					</div>
				{:else if item.type === 'history'}
					{@const h = item.data}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4">
						<div class="flex items-start gap-3">
							<div class="w-8 h-8 rounded-lg {getHistoryStatusBg(h.status)} flex items-center justify-center flex-shrink-0 mt-0.5">
								{#if h.status === 'grabbed'}
									<svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
								{:else if h.status === 'imported'}
									<svg class="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
								{:else}
									<svg class="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
								{/if}
							</div>
							<div class="flex-1 min-w-0">
								<h3 class="font-medium text-text-primary truncate">{h.releaseTitle}</h3>
								<div class="flex items-center flex-wrap gap-x-2 gap-y-0.5 mt-1 text-xs text-text-muted">
									<span class="{getHistoryStatusColor(h.status)} capitalize">{h.status}</span>
									{#if h.indexerName}<span>·</span><span>{h.indexerName}</span>{/if}
									{#if h.qualityResolution}<span>·</span><span>{h.qualityResolution}</span>{/if}
									{#if h.size > 0}<span>·</span><span>{formatBytes(h.size)}</span>{/if}
									<span>·</span><span>{formatRelativeTime(h.grabbedAt)}</span>
								</div>
								{#if h.errorMessage}<p class="text-xs text-red-400 mt-2">{h.errorMessage}</p>{/if}
							</div>
						</div>
					</div>
				{:else if item.type === 'blocklist'}
					{@const b = item.data}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4 border-l-2 border-l-red-500">
						<div class="flex items-start gap-3">
							<div class="w-8 h-8 rounded-lg bg-red-500/20 flex items-center justify-center flex-shrink-0 mt-0.5">
								<svg class="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="min-w-0">
										<h3 class="font-medium text-text-primary truncate">{b.releaseTitle}</h3>
										<div class="flex items-center gap-2 mt-1 text-xs text-text-muted">
											<span class="text-red-400">{b.reason}</span>
											{#if b.releaseGroup}<span>·</span><span>{b.releaseGroup}</span>{/if}
											<span>·</span><span>{formatRelativeTime(b.createdAt)}</span>
										</div>
										{#if b.errorMessage}<p class="text-xs text-text-muted mt-1">{b.errorMessage}</p>{/if}
									</div>
									<button class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50" onclick={() => unblock(b.id)} disabled={processingIds.has(`block-${b.id}`)}>{processingIds.has(`block-${b.id}`) ? '...' : 'Remove'}</button>
								</div>
							</div>
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>
