<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getDownloadItems,
		deleteDownloadItem,
		getWantedItems,
		deleteWantedItem,
		searchWantedItem,
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
		type Request,
		type SystemStatus,
		type GrabHistoryItem,
		type BlocklistEntry
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { auth } from '$lib/stores/auth';
	import QueueCard, { type QueueState } from '$lib/components/QueueCard.svelte';
	import { LoadingSpinner, EmptyState } from '$lib/components/ui';

	type Filter = 'queue' | 'history' | 'blocklist';
	let activeFilter: Filter = $state('queue');

	let user = $state<{ role: string } | null>(null);
	auth.subscribe((value) => { user = value; });
	let systemStatus: SystemStatus | null = $state(null);

	let downloads: DownloadItem[] = $state([]);
	let wantedItems: WantedItem[] = $state([]);
	let requests: Request[] = $state([]);
	let deniedRequests: Request[] = $state([]);
	let grabHistory: GrabHistoryItem[] = $state([]);
	let blocklist: BlocklistEntry[] = $state([]);

	let loading = $state(true);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let processingIds: Set<string> = $state(new Set());
	let searchingIds: Set<number> = $state(new Set());
	let confirmingCancel: string | null = $state(null);
	let confirmingRemove: string | null = $state(null);

	// Unified queue item - merges request, wanted, download by tmdbId
	interface QueueItem {
		id: string;
		tmdbId: number;
		title: string;
		year?: number;
		type: 'movie' | 'show';
		posterPath?: string | null;
		state: QueueState;
		progress?: number;
		downloaded?: number;
		size?: number;
		speed?: number;
		eta?: number;
		quality?: string;
		username?: string;
		timestamp?: string;
		error?: string;
		seasons?: string; // JSON array of season numbers (for TV shows)
		// Source IDs for actions
		requestId?: number;
		wantedId?: number;
		downloadId?: number;
	}

	// Map download state to simplified QueueState
	function mapDownloadState(state: string): QueueState {
		switch (state) {
			case 'queued':
			case 'downloading': return 'downloading';
			case 'paused': return 'paused';
			case 'stalled': return 'stalled';
			case 'importing':
			case 'import_pending': return 'importing';
			case 'failed':
			case 'import_blocked': return 'failed';
			default: return 'downloading';
		}
	}

	// Merge all sources into unified queue items
	let queueItems = $derived.by(() => {
		const itemMap = new Map<number, QueueItem>();

		// First add requests (lowest priority - will be overwritten)
		for (const r of requests) {
			if (r.status === 'denied') continue; // Skip denied
			// approved requests show as "searching" since that's what happens after approval
			const state: QueueState = r.status === 'requested' ? 'pending' : 'searching';
			itemMap.set(r.tmdbId, {
				id: `req-${r.id}`,
				tmdbId: r.tmdbId,
				title: r.title,
				year: r.year,
				type: r.type as 'movie' | 'show',
				posterPath: r.posterPath,
				state,
				username: r.username,
				timestamp: formatRelativeTime(r.requestedAt),
				seasons: r.seasons,
				requestId: r.id
			});
		}

		// Then add wanted items - all show as "searching" (system is looking for downloads)
		for (const w of wantedItems) {
			const existing = itemMap.get(w.tmdbId);
			itemMap.set(w.tmdbId, {
				id: `wanted-${w.id}`,
				tmdbId: w.tmdbId,
				title: w.title,
				year: w.year,
				type: w.type as 'movie' | 'show',
				posterPath: w.posterPath,
				state: 'searching',
				username: existing?.username,
				timestamp: existing?.timestamp,
				seasons: w.seasons || existing?.seasons,
				requestId: existing?.requestId,
				wantedId: w.id
			});
		}

		// Finally add downloads (highest priority) - skip imported (they're in history)
		for (const d of downloads) {
			if (d.state === 'imported') continue;

			let key: number;
			let existing: QueueItem | undefined;

			if (d.tmdbId) {
				// Download has tmdbId - use it directly
				key = d.tmdbId;
				existing = itemMap.get(d.tmdbId);
			} else {
				// Download has no tmdbId - try to find a matching wanted/request item by title
				// This handles cases where the download wasn't linked to grab history
				// Normalize release title: replace periods/underscores with spaces, lowercase
				const normalizedTitle = d.title.toLowerCase().replace(/[._]/g, ' ');
				let matchedKey: number | undefined;
				for (const [k, item] of itemMap.entries()) {
					if (item.title) {
						// Get first few words of the wanted item title
						const itemWords = item.title.toLowerCase().split(' ').filter(w => w.length > 0);
						// Check if the normalized download title starts with these words
						const searchPattern = itemWords.slice(0, Math.min(3, itemWords.length)).join(' ');
						if (normalizedTitle.includes(searchPattern)) {
							matchedKey = k;
							existing = item;
							break;
						}
					}
				}
				key = matchedKey || -d.id;
			}

			itemMap.set(key, {
				id: `dl-${d.id}`,
				tmdbId: existing?.tmdbId || d.tmdbId || 0,
				title: existing?.title || d.title,
				year: existing?.year || d.year,
				type: existing?.type || (d.mediaType || 'movie') as 'movie' | 'show',
				posterPath: existing?.posterPath || d.posterPath,
				state: mapDownloadState(d.state),
				progress: d.progress,
				downloaded: d.downloaded,
				size: d.size,
				speed: d.speed,
				eta: d.eta,
				quality: d.quality,
				username: existing?.username,
				timestamp: existing?.timestamp,
				seasons: existing?.seasons,
				error: d.errors?.[0] || d.importBlockReason,
				requestId: existing?.requestId,
				wantedId: existing?.wantedId,
				downloadId: d.id
			});
		}

		// Convert to array and sort by priority (active items first)
		return Array.from(itemMap.values()).sort((a, b) => {
			const priority = (item: QueueItem) => {
				switch (item.state) {
					case 'downloading': case 'importing': return 0;
					case 'paused': case 'stalled': return 1;
					case 'failed': return 2;
					case 'searching': return 3;
					case 'pending': return 4;
					default: return 5;
				}
			};
			return priority(a) - priority(b);
		});
	});

	// Filter to only show active queue items (not denied)
	let activeQueueItems = $derived(queueItems.filter(q => q.state !== 'denied'));

	let counts = $derived({
		queue: activeQueueItems.length,
		history: grabHistory.length + deniedRequests.length,
		blocklist: blocklist.length
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
			const [dlData, wantedData, reqData, deniedData, statusData, histData, blockData] = await Promise.all([
				getDownloadItems(), getWantedItems(), getRequests(),
				getRequests('denied'), getSystemStatus(), getGrabHistory(100), getBlocklist()
			]);
			downloads = dlData || [];
			wantedItems = wantedData || [];
			requests = reqData || [];
			deniedRequests = deniedData || [];
			systemStatus = statusData;
			grabHistory = histData || [];
			blocklist = blockData || [];
		} catch (e) {
			console.error('Failed to load activity:', e);
		} finally {
			loading = false;
		}
	}

	async function handleApprove(item: QueueItem) {
		if (!item.requestId) return;
		const key = `req-${item.requestId}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await updateRequest(item.requestId, 'approved');
			await loadAll();
			toast.success('Approved - searching...');
		} catch (e) {
			toast.error('Failed to approve');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function handleDeny(item: QueueItem) {
		if (!item.requestId) return;
		const key = `req-${item.requestId}`;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			await updateRequest(item.requestId, 'denied');
			await loadAll();
			toast.info('Request denied');
		} catch (e) {
			toast.error('Failed to deny');
		} finally {
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function handleSearch(item: QueueItem) {
		if (!item.wantedId) return;
		searchingIds.add(item.wantedId);
		searchingIds = searchingIds;
		try {
			await searchWantedItem(item.wantedId);
			await loadAll();
		} catch (e) {
			toast.error('Search failed');
		} finally {
			searchingIds.delete(item.wantedId);
			searchingIds = searchingIds;
		}
	}

	async function handleCancel(item: QueueItem) {
		console.log('handleCancel called:', { id: item.id, downloadId: item.downloadId, state: item.state });
		if (!item.downloadId) {
			console.log('handleCancel: no downloadId, returning early');
			return;
		}
		// First click shows confirm, second click cancels
		if (confirmingCancel !== item.id) {
			console.log('handleCancel: setting confirm state');
			confirmingCancel = item.id;
			setTimeout(() => { if (confirmingCancel === item.id) confirmingCancel = null; }, 3000);
			return;
		}
		console.log('handleCancel: proceeding with delete');
		const key = item.id;
		processingIds.add(key);
		processingIds = processingIds;
		try {
			// Cancel download and delete files
			console.log('handleCancel: calling deleteDownloadItem with id:', item.downloadId);
			await deleteDownloadItem(item.downloadId, { deleteFiles: true });
			console.log('handleCancel: deleteDownloadItem succeeded');
			// Also remove from wanted list if present
			if (item.wantedId) {
				await deleteWantedItem(item.wantedId);
			}
			// Also remove the request if present
			if (item.requestId) {
				await deleteRequest(item.requestId);
			}
			await loadAll();
			toast.success('Download cancelled');
		} catch (e) {
			console.error('handleCancel error:', e);
			toast.error(`Failed to cancel: ${e instanceof Error ? e.message : 'Unknown error'}`);
		} finally {
			confirmingCancel = null;
			processingIds.delete(key);
			processingIds = processingIds;
		}
	}

	async function handleRemove(item: QueueItem) {
		// Remove ALL related items: download, wanted, AND request
		const key = item.id;
		processingIds.add(key);
		processingIds = processingIds;

		try {
			// Remove download first (includes files)
			if (item.downloadId) {
				await deleteDownloadItem(item.downloadId, { deleteFiles: true });
			}
			// Then remove from wanted list
			if (item.wantedId) {
				await deleteWantedItem(item.wantedId);
			}
			// Finally remove the request
			if (item.requestId) {
				await deleteRequest(item.requestId);
			}
			await loadAll();
			toast.success('Removed');
		} catch (e) {
			toast.error('Failed to remove');
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

	function formatRelativeTime(dateStr: string): string {
		const diff = Date.now() - new Date(dateStr).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		return `${Math.floor(hours / 24)}d ago`;
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

	function isProcessing(item: QueueItem): boolean {
		if (item.requestId && processingIds.has(`req-${item.requestId}`)) return true;
		if (item.wantedId && processingIds.has(`wanted-${item.wantedId}`)) return true;
		if (item.downloadId && processingIds.has(`dl-${item.downloadId}`)) return true;
		return false;
	}

	function isSearching(item: QueueItem): boolean {
		return item.wantedId ? searchingIds.has(item.wantedId) : false;
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

	<div class="flex flex-wrap gap-2">
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'queue' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'queue'}>
			Queue {#if counts.queue > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'queue' ? 'bg-black/20' : 'bg-blue-500 text-white'}">{counts.queue}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'history' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'history'}>
			History {#if counts.history > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'history' ? 'bg-black/20' : 'bg-green-500 text-white'}">{counts.history}</span>{/if}
		</button>
		<button class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeFilter === 'blocklist' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}" onclick={() => activeFilter = 'blocklist'}>
			Blocklist {#if counts.blocklist > 0}<span class="px-1.5 py-0.5 text-xs rounded-full {activeFilter === 'blocklist' ? 'bg-black/20' : 'bg-red-500 text-white'}">{counts.blocklist}</span>{/if}
		</button>
	</div>

	{#if loading}
		<LoadingSpinner size="lg" fullPage />
	{:else if activeFilter === 'queue'}
		{#if activeQueueItems.length === 0}
			<EmptyState
				icon="M5 13l4 4L19 7"
				title="Queue is empty"
				description="No pending downloads or requests"
				compact
			/>
		{:else}
			<div class="space-y-2">
				{#each activeQueueItems as item (item.id)}
					<QueueCard
						id={item.id}
						tmdbId={item.tmdbId}
						title={item.title}
						year={item.year}
						type={item.type}
						posterPath={item.posterPath}
						state={item.state}
						seasons={item.seasons}
						progress={item.progress}
						downloaded={item.downloaded}
						size={item.size}
						speed={item.speed}
						eta={item.eta}
						quality={item.quality}
						username={item.username}
						timestamp={item.timestamp}
						error={item.error}
						onApprove={() => handleApprove(item)}
						onDeny={() => handleDeny(item)}
						onSearch={() => handleSearch(item)}
						onCancel={() => handleCancel(item)}
						onRemove={() => handleRemove(item)}
						processing={isProcessing(item)}
						searching={isSearching(item)}
						isAdmin={user?.role === 'admin'}
						confirmingCancel={confirmingCancel === item.id}
					/>
				{/each}
			</div>
		{/if}
	{:else if activeFilter === 'history'}
		{#if grabHistory.length === 0 && deniedRequests.length === 0}
			<EmptyState
				icon="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
				title="No history yet"
				description="Downloads and requests will appear here"
				compact
			/>
		{:else}
			<div class="space-y-4">
				<!-- Denied Requests -->
				{#if deniedRequests.length > 0}
					<div class="space-y-2">
						<h3 class="text-sm font-medium text-text-muted uppercase tracking-wide px-1">Denied Requests</h3>
						{#each deniedRequests as req}
							<div class="bg-bg-card border border-border-subtle rounded-xl p-4 border-l-2 border-l-red-500">
								<div class="flex items-start gap-3">
									{#if req.posterPath}
										<img src={getTmdbImageUrl(req.posterPath, 'w92')} alt="" class="w-10 h-14 rounded-lg object-cover flex-shrink-0" />
									{:else}
										<div class="w-10 h-14 rounded-lg bg-red-500/20 flex items-center justify-center flex-shrink-0">
											<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<h3 class="font-medium text-text-primary truncate">{req.title}{#if req.year}<span class="text-text-muted"> ({req.year})</span>{/if}</h3>
										<div class="flex items-center flex-wrap gap-x-2 gap-y-0.5 mt-1 text-xs text-text-muted">
											<span class="text-red-400">Denied</span>
											<span>·</span>
											<span class="capitalize">{req.type}</span>
											{#if req.username}<span>·</span><span>by {req.username}</span>{/if}
											<span>·</span><span>{formatRelativeTime(req.updatedAt)}</span>
										</div>
										{#if req.statusReason}<p class="text-xs text-text-muted mt-1">{req.statusReason}</p>{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>
				{/if}

				<!-- Grab History -->
				{#if grabHistory.length > 0}
					<div class="space-y-2">
						{#if deniedRequests.length > 0}
							<h3 class="text-sm font-medium text-text-muted uppercase tracking-wide px-1">Downloads</h3>
						{/if}
						{#each grabHistory as h}
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
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	{:else if activeFilter === 'blocklist'}
		{#if blocklist.length === 0}
			<EmptyState
				icon="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
				title="Blocklist is empty"
				description="No blocked releases"
				compact
			/>
		{:else}
			<div class="space-y-2">
				{#each blocklist as b}
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
				{/each}
			</div>
		{/if}
	{/if}
</div>
