<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getDownloadItems,
		getImportHistory,
		deleteDownloadItem,
		type DownloadItem,
		type ImportHistoryItem
	} from '$lib/api';

	let downloads: DownloadItem[] = $state([]);
	let importHistory: ImportHistoryItem[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;

	// Group downloads by status
	let downloading = $derived(downloads.filter(d => d.status === 'downloading'));
	let importing = $derived(downloads.filter(d => d.status === 'importing'));
	let completed = $derived(downloads.filter(d => d.status === 'imported').slice(0, 20));
	let needsAttention = $derived(downloads.filter(d => d.status === 'failed' || d.status === 'unmatched'));

	onMount(async () => {
		await loadData();
		// Refresh every 5 seconds
		refreshInterval = setInterval(loadData, 5000);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	async function loadData() {
		try {
			const [dlData, histData] = await Promise.all([
				getDownloadItems(),
				getImportHistory(30)
			]);
			downloads = dlData || [];
			importHistory = histData || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load activity';
		} finally {
			loading = false;
		}
	}

	async function handleRemoveDownload(id: number) {
		try {
			await deleteDownloadItem(id);
			downloads = downloads.filter(d => d.id !== id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove download';
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatRelativeTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'downloading': return 'text-blue-400';
			case 'importing': return 'text-yellow-400';
			case 'imported': return 'text-green-400';
			case 'completed': return 'text-green-400';
			case 'failed': return 'text-red-400';
			case 'unmatched': return 'text-orange-400';
			default: return 'text-text-muted';
		}
	}

	function getStatusIcon(status: string): string {
		switch (status) {
			case 'downloading': return 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4';
			case 'importing': return 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15';
			case 'imported': return 'M5 13l4 4L19 7';
			case 'completed': return 'M5 13l4 4L19 7';
			case 'failed': return 'M6 18L18 6M6 6l12 12';
			case 'unmatched': return 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';
			default: return 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z';
		}
	}
</script>

<svelte:head>
	<title>Activity - Outpost</title>
</svelte:head>

<div class="space-y-8 max-w-4xl mx-auto">
	<div>
		<h1 class="text-3xl font-bold text-text-primary">Activity</h1>
		<p class="text-text-secondary mt-1">Monitor downloads and imports</p>
	</div>

	{#if error}
		<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
			<span>{error}</span>
			<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)}>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="w-8 h-8 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"></div>
		</div>
	{:else}
		<!-- Downloading -->
		{#if downloading.length > 0}
			<section class="glass-card p-6 space-y-4">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-xl bg-blue-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
					</div>
					<h2 class="text-lg font-semibold text-text-primary">Downloading</h2>
				</div>

				<div class="space-y-3">
					{#each downloading as dl}
						<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
							<div class="flex items-center justify-between mb-2">
								<span class="font-medium text-text-primary truncate flex-1 mr-4">{dl.title}</span>
								<span class="text-sm text-text-secondary">{formatBytes(dl.size)}</span>
							</div>
							<div class="w-full bg-bg-card rounded-full h-2 overflow-hidden">
								<div
									class="bg-blue-500 h-full transition-all duration-300"
									style="width: {dl.progress}%"
								></div>
							</div>
							<div class="flex justify-between mt-1.5 text-xs text-text-muted">
								<span>{dl.progress.toFixed(1)}%</span>
								<span>{formatRelativeTime(dl.updatedAt)}</span>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Importing -->
		{#if importing.length > 0}
			<section class="glass-card p-6 space-y-4">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-xl bg-yellow-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-yellow-400 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
						</svg>
					</div>
					<h2 class="text-lg font-semibold text-text-primary">Importing</h2>
				</div>

				<div class="space-y-3">
					{#each importing as dl}
						<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
							<div class="flex items-center gap-3">
								<div class="w-5 h-5 border-2 border-yellow-400 border-t-transparent rounded-full animate-spin"></div>
								<span class="font-medium text-text-primary">{dl.title}</span>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Needs Attention -->
		{#if needsAttention.length > 0}
			<section class="glass-card p-6 space-y-4 border-l-4 border-l-red-500">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-xl bg-red-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
						</svg>
					</div>
					<h2 class="text-lg font-semibold text-text-primary">Needs Attention</h2>
				</div>

				<div class="space-y-3">
					{#each needsAttention as dl}
						<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
							<div class="flex items-center justify-between">
								<div class="flex-1">
									<div class="flex items-center gap-2">
										<span class="font-medium text-text-primary">{dl.title}</span>
										<span class="px-2 py-0.5 text-xs rounded-lg {dl.status === 'failed' ? 'bg-red-900/50 text-red-300' : 'bg-orange-900/50 text-orange-300'}">
											{dl.status === 'failed' ? 'Failed' : 'Unmatched'}
										</span>
									</div>
									{#if dl.error}
										<p class="text-sm text-text-muted mt-1">{dl.error}</p>
									{/if}
									{#if dl.importedPath}
										<p class="text-xs text-text-muted mt-1 truncate">{dl.importedPath}</p>
									{/if}
								</div>
								<div class="flex gap-2 ml-4">
									<button
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
										onclick={() => handleRemoveDownload(dl.id)}
									>
										Dismiss
									</button>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Recently Imported -->
		<section class="glass-card p-6 space-y-4">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-green-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<h2 class="text-lg font-semibold text-text-primary">Recently Imported</h2>
			</div>

			{#if completed.length === 0 && importHistory.length === 0}
				<p class="text-text-muted py-4">No recent imports.</p>
			{:else}
				<div class="space-y-2">
					{#each completed as dl}
						<div class="p-3 bg-bg-elevated/50 rounded-xl border border-white/5 flex items-center justify-between">
							<div class="flex-1 min-w-0">
								<span class="font-medium text-text-primary block truncate">{dl.title}</span>
								{#if dl.importedPath}
									<span class="text-xs text-text-muted block truncate">{dl.importedPath}</span>
								{/if}
							</div>
							<span class="text-xs text-text-muted ml-4 flex-shrink-0">{formatRelativeTime(dl.updatedAt)}</span>
						</div>
					{/each}
					{#each importHistory.filter(h => h.success) as hist}
						<div class="p-3 bg-bg-elevated/50 rounded-xl border border-white/5 flex items-center justify-between">
							<div class="flex-1 min-w-0">
								<span class="text-text-secondary block truncate">{hist.destPath.split('/').pop() || hist.destPath.split('\\').pop()}</span>
								<span class="text-xs text-text-muted block truncate">{hist.destPath}</span>
							</div>
							<span class="text-xs text-text-muted ml-4 flex-shrink-0">{formatRelativeTime(hist.createdAt)}</span>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<!-- No activity state -->
		{#if downloading.length === 0 && importing.length === 0 && completed.length === 0 && needsAttention.length === 0 && importHistory.length === 0}
			<div class="glass-card p-12 text-center">
				<div class="w-16 h-16 rounded-full bg-bg-elevated/50 flex items-center justify-center mx-auto mb-4">
					<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
				<h3 class="text-lg font-medium text-text-primary mb-2">No Activity</h3>
				<p class="text-text-secondary">Downloads and imports will appear here.</p>
			</div>
		{/if}
	{/if}
</div>
