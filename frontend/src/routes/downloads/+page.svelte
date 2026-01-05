<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { getDownloads, type Download } from '$lib/api';

	let downloads: Download[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let refreshInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		await loadDownloads();
		refreshInterval = setInterval(loadDownloads, 5000);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	async function loadDownloads() {
		try {
			downloads = await getDownloads();
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load downloads';
		} finally {
			loading = false;
		}
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatSpeed(bytesPerSec: number): string {
		return formatSize(bytesPerSec) + '/s';
	}

	function formatEta(seconds: number): string {
		if (seconds <= 0 || seconds > 86400 * 365) return '--';
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		if (seconds < 86400) {
			const hours = Math.floor(seconds / 3600);
			const mins = Math.floor((seconds % 3600) / 60);
			return `${hours}h ${mins}m`;
		}
		return `${Math.floor(seconds / 86400)}d`;
	}

	function getProgressBarColor(status: string): string {
		switch (status) {
			case 'downloading': return 'bg-white-400';
			case 'completed': return 'bg-green-500';
			case 'paused': return 'bg-white/50';
			case 'error': return 'bg-white/30';
			default: return 'bg-text-muted';
		}
	}

	let activeDownloads = $derived(downloads.filter(d => d.status === 'downloading' || d.status === 'queued' || d.status === 'paused'));
	let completedDownloads = $derived(downloads.filter(d => d.status === 'completed'));
	let errorDownloads = $derived(downloads.filter(d => d.status === 'error'));
</script>

<svelte:head>
	<title>Downloads - Outpost</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-text-primary">Downloads</h1>
		<div class="inline-flex items-center p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
			<button
				class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg text-white/60 hover:text-white hover:bg-white/5 transition-colors"
				onclick={loadDownloads}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Refresh
			</button>
		</div>
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
			<div class="flex items-center gap-3">
				<div class="spinner-lg text-cream"></div>
				<p class="text-text-secondary">Loading downloads...</p>
			</div>
		</div>
	{:else if downloads.length === 0}
		<div class="glass-card p-12 text-center">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
				<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-text-primary mb-2">No active downloads</h2>
			<p class="text-text-secondary">Downloads from your configured clients will appear here.</p>
		</div>
	{:else}
		<!-- Active Downloads -->
		{#if activeDownloads.length > 0}
			<section class="space-y-3">
				<h2 class="text-lg font-semibold text-text-primary flex items-center gap-2">
					<span class="w-2 h-2 rounded-full bg-white-400 animate-pulse"></span>
					Active ({activeDownloads.length})
				</h2>
				{#each activeDownloads as download}
					<div class="glass-card p-4">
						<div class="flex items-start justify-between mb-3">
							<div class="flex-1 min-w-0 mr-4">
								<h3 class="font-medium text-text-primary truncate">{download.name}</h3>
								<p class="text-sm text-text-secondary">
									{download.clientName}
									{#if download.category}
										<span class="mx-1 text-text-muted">|</span>
										{download.category}
									{/if}
								</p>
							</div>
							<span class="liquid-badge-sm capitalize
								{download.status === 'downloading' ? '!bg-white-500/20 !border-t-white-400/40 text-white-400' : ''}
								{download.status === 'paused' ? '!bg-white/5 text-text-secondary' : ''}
								{download.status === 'queued' ? '!bg-white/5 text-text-muted' : ''}
							">
								{download.status}
							</span>
						</div>

						<div class="mb-3">
							<div class="h-2 bg-bg-elevated rounded-full overflow-hidden">
								<div
									class="h-full transition-all {getProgressBarColor(download.status)}"
									style="width: {download.progress}%"
								></div>
							</div>
						</div>

						<div class="flex items-center justify-between text-sm text-text-secondary">
							<span>
								{download.progress.toFixed(1)}%
								{#if download.size > 0}
									<span class="mx-1 text-text-muted">|</span>
									{formatSize(download.downloaded)} / {formatSize(download.size)}
								{/if}
							</span>
							{#if download.status === 'downloading'}
								<span class="flex items-center gap-2">
									<svg class="w-4 h-4 text-white-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
									</svg>
									{formatSpeed(download.speed)}
									{#if download.eta > 0}
										<span class="text-text-muted">|</span>
										ETA: {formatEta(download.eta)}
									{/if}
								</span>
							{/if}
						</div>
					</div>
				{/each}
			</section>
		{/if}

		<!-- Completed Downloads -->
		{#if completedDownloads.length > 0}
			<section class="space-y-3">
				<h2 class="text-lg font-semibold text-text-primary flex items-center gap-2">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					Completed ({completedDownloads.length})
				</h2>
				{#each completedDownloads as download}
					<div class="glass-card p-4">
						<div class="flex items-center justify-between">
							<div class="flex-1 min-w-0 mr-4">
								<h3 class="font-medium text-green-400 truncate">{download.name}</h3>
								<p class="text-sm text-text-secondary">
									{download.clientName}
									{#if download.size > 0}
										<span class="mx-1 text-text-muted">|</span>
										{formatSize(download.size)}
									{/if}
								</p>
							</div>
							<span class="liquid-badge-sm !bg-green-500/20 !border-t-green-400/40 text-green-400">Completed</span>
						</div>
					</div>
				{/each}
			</section>
		{/if}

		<!-- Error Downloads -->
		{#if errorDownloads.length > 0}
			<section class="space-y-3">
				<h2 class="text-lg font-semibold text-text-secondary flex items-center gap-2">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
					</svg>
					Errors ({errorDownloads.length})
				</h2>
				{#each errorDownloads as download}
					<div class="glass-card p-4 border border-white/10">
						<div class="flex items-center justify-between">
							<div class="flex-1 min-w-0 mr-4">
								<h3 class="font-medium text-text-secondary truncate">{download.name}</h3>
								<p class="text-sm text-text-muted">{download.clientName}</p>
							</div>
							<span class="liquid-badge-sm !bg-white/5 text-text-secondary">Error</span>
						</div>
					</div>
				{/each}
			</section>
		{/if}
	{/if}
</div>
