<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { getLogs, downloadLogs, type LogEntry, type LogsResponse } from '$lib/api';

	let logs: LogEntry[] = $state([]);
	let total = $state(0);
	let hasMore = $state(false);
	let loading = $state(true);
	let error: string | null = $state(null);
	let downloading = $state(false);

	// Filters
	let levelFilter = $state('all');
	let sourceFilter = $state('all');
	let searchQuery = $state('');
	let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	// Auto-refresh
	let autoRefresh = $state(false);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;

	// Auto-scroll
	let logContainer: HTMLDivElement | undefined = $state();
	let isAtBottom = $state(true);

	const levelOptions = [
		{ value: 'all', label: 'All Levels' },
		{ value: 'DEBUG', label: 'Debug' },
		{ value: 'INFO', label: 'Info' },
		{ value: 'WARN', label: 'Warn' },
		{ value: 'ERROR', label: 'Error' }
	];

	const sourceOptions = [
		{ value: 'all', label: 'All Sources' },
		{ value: 'api', label: 'API' },
		{ value: 'scheduler', label: 'Scheduler' },
		{ value: 'indexer', label: 'Indexer' },
		{ value: 'importer', label: 'Importer' },
		{ value: 'download', label: 'Download' },
		{ value: 'scanner', label: 'Scanner' },
		{ value: 'metadata', label: 'Metadata' },
		{ value: 'auth', label: 'Auth' }
	];

	onMount(() => {
		loadLogs();
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
		if (searchDebounceTimer) {
			clearTimeout(searchDebounceTimer);
		}
	});

	async function loadLogs() {
		try {
			loading = true;
			error = null;

			const response = await getLogs({
				level: levelFilter !== 'all' ? levelFilter : undefined,
				source: sourceFilter !== 'all' ? sourceFilter : undefined,
				search: searchQuery || undefined,
				limit: 500
			});

			logs = response.entries;
			total = response.total;
			hasMore = response.hasMore;

			// Auto-scroll to bottom if we were at bottom
			if (isAtBottom && logContainer) {
				setTimeout(() => {
					if (logContainer) {
						logContainer.scrollTop = logContainer.scrollHeight;
					}
				}, 0);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load logs';
		} finally {
			loading = false;
		}
	}

	function handleScroll() {
		if (logContainer) {
			const threshold = 50;
			isAtBottom = logContainer.scrollHeight - logContainer.scrollTop - logContainer.clientHeight < threshold;
		}
	}

	function toggleAutoRefresh() {
		autoRefresh = !autoRefresh;
		if (autoRefresh) {
			refreshInterval = setInterval(loadLogs, 5000);
		} else if (refreshInterval) {
			clearInterval(refreshInterval);
			refreshInterval = null;
		}
	}

	function handleSearchInput(e: Event) {
		const target = e.target as HTMLInputElement;
		const value = target.value;

		if (searchDebounceTimer) {
			clearTimeout(searchDebounceTimer);
		}

		searchDebounceTimer = setTimeout(() => {
			searchQuery = value;
			loadLogs();
		}, 300);
	}

	async function handleDownload() {
		try {
			downloading = true;
			await downloadLogs();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to download logs';
		} finally {
			downloading = false;
		}
	}

	function getLevelColor(level: string): string {
		switch (level) {
			case 'DEBUG':
				return 'text-text-muted';
			case 'INFO':
				return 'text-cream';
			case 'WARN':
				return 'text-amber-400';
			case 'ERROR':
				return 'text-red-400';
			default:
				return 'text-text-secondary';
		}
	}

	function getLevelBadgeClass(level: string): string {
		switch (level) {
			case 'DEBUG':
				return 'bg-white/5 text-text-muted';
			case 'INFO':
				return 'bg-blue-900/30 text-blue-300';
			case 'WARN':
				return 'bg-amber-900/30 text-amber-300';
			case 'ERROR':
				return 'bg-red-900/30 text-red-400';
			default:
				return 'bg-white/5 text-text-muted';
		}
	}

	function formatTimestamp(ts: string): string {
		const date = new Date(ts);
		return date.toLocaleString('en-US', {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			hour12: false
		});
	}

	// Watch for filter changes
	$effect(() => {
		// Track dependencies
		levelFilter;
		sourceFilter;
		// Reload when filters change (but not search, which is debounced)
		loadLogs();
	});
</script>

<div class="space-y-4">
	<!-- Filter Bar -->
	<div class="glass-card p-4">
		<div class="flex flex-wrap items-center gap-4">
			<!-- Level Filter -->
			<div class="w-36">
				<Select
					id="level-filter"
					bind:value={levelFilter}
					options={levelOptions}
				/>
			</div>

			<!-- Source Filter -->
			<div class="w-40">
				<Select
					id="source-filter"
					bind:value={sourceFilter}
					options={sourceOptions}
				/>
			</div>

			<!-- Search -->
			<div class="flex-1 min-w-48">
				<input
					type="text"
					placeholder="Search logs..."
					class="liquid-input w-full px-4 py-2"
					value={searchQuery}
					oninput={handleSearchInput}
				/>
			</div>

			<!-- Auto-refresh Toggle -->
			<button
				type="button"
				class="flex items-center gap-2 px-3 py-2 rounded-lg transition-colors {autoRefresh ? 'bg-green-600/20 text-green-400 border border-green-600/30' : 'bg-white/5 text-text-muted border border-white/10 hover:text-text-secondary'}"
				onclick={toggleAutoRefresh}
			>
				<svg class="w-4 h-4 {autoRefresh ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				<span class="text-sm">Auto-refresh</span>
			</button>

			<!-- Download Button -->
			<button
				type="button"
				class="liquid-btn-sm"
				onclick={handleDownload}
				disabled={downloading}
			>
				{#if downloading}
					<svg class="w-4 h-4 animate-spin mr-2" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
				{:else}
					<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				{/if}
				Download Logs
			</button>
		</div>

		{#if hasMore}
			<p class="text-xs text-text-muted mt-2">
				Showing {logs.length} of {total} entries (limited to 500)
			</p>
		{/if}
	</div>

	{#if error}
		<div class="bg-red-900/20 border border-red-600/30 text-red-400 px-4 py-3 rounded-xl">
			{error}
		</div>
	{/if}

	<!-- Log Display -->
	<div class="glass-card overflow-hidden">
		<div
			bind:this={logContainer}
			class="log-container h-[600px] overflow-y-auto p-4 font-mono text-sm"
			onscroll={handleScroll}
		>
			{#if loading && logs.length === 0}
				<div class="flex items-center justify-center h-full text-text-muted">
					<svg class="w-6 h-6 animate-spin mr-2" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Loading logs...
				</div>
			{:else if logs.length === 0}
				<div class="flex flex-col items-center justify-center h-full text-text-muted">
					<svg class="w-12 h-12 mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
					{#if searchQuery || levelFilter !== 'all' || sourceFilter !== 'all'}
						<p>No logs matching filters</p>
					{:else}
						<p>No logs captured yet</p>
					{/if}
				</div>
			{:else}
				<div class="space-y-1">
					{#each logs as log}
						<div class="log-entry flex items-start gap-2 py-1 px-2 rounded hover:bg-white/5 transition-colors group">
							<!-- Timestamp -->
							<span class="text-text-muted whitespace-nowrap flex-shrink-0">
								[{formatTimestamp(log.timestamp)}]
							</span>

							<!-- Level Badge -->
							<span class="px-1.5 py-0.5 text-xs rounded {getLevelBadgeClass(log.level)} flex-shrink-0">
								{log.level}
							</span>

							<!-- Source -->
							<span class="text-text-muted whitespace-nowrap flex-shrink-0">
								[{log.source}]
							</span>

							<!-- Message -->
							<span class="{getLevelColor(log.level)} break-all">
								{log.message}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Scroll to Bottom Button -->
	{#if !isAtBottom && logs.length > 0}
		<button
			type="button"
			class="fixed bottom-8 right-8 bg-accent-primary text-white px-4 py-2 rounded-full shadow-lg hover:bg-accent-primary/90 transition-colors flex items-center gap-2"
			onclick={() => {
				if (logContainer) {
					logContainer.scrollTop = logContainer.scrollHeight;
				}
			}}
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
			</svg>
			Scroll to bottom
		</button>
	{/if}
</div>

<style>
	.log-container {
		background: #0d0d0d;
		scrollbar-width: thin;
		scrollbar-color: rgba(255, 255, 255, 0.2) transparent;
	}

	.log-container::-webkit-scrollbar {
		width: 8px;
	}

	.log-container::-webkit-scrollbar-track {
		background: transparent;
	}

	.log-container::-webkit-scrollbar-thumb {
		background: rgba(255, 255, 255, 0.2);
		border-radius: 4px;
	}

	.log-container::-webkit-scrollbar-thumb:hover {
		background: rgba(255, 255, 255, 0.3);
	}

	.log-entry {
		font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Mono', 'Droid Sans Mono', 'Source Code Pro', monospace;
		font-size: 12px;
		line-height: 1.5;
	}
</style>
