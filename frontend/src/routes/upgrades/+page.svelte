<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getUpgrades,
		searchUpgrade,
		searchAllUpgrades,
		resetUpgradeSearch,
		getTmdbImageUrl,
		type UpgradeableItem,
		type UpgradesSummary
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { LoadingSpinner, EmptyState, Button } from '$lib/components/ui';

	type Tab = 'movies' | 'episodes';
	let activeTab: Tab = $state('movies');

	let summary: UpgradesSummary | null = $state(null);
	let loading = $state(true);
	let searchingIds: Set<number> = $state(new Set());
	let searchingAll = $state(false);
	let resettingIds: Set<number> = $state(new Set());

	onMount(async () => {
		await loadUpgrades();
	});

	async function loadUpgrades() {
		loading = true;
		try {
			summary = await getUpgrades();
		} catch (e) {
			console.error('Failed to load upgrades:', e);
			toast.error('Failed to load upgrades');
		} finally {
			loading = false;
		}
	}

	async function handleSearch(item: UpgradeableItem) {
		searchingIds.add(item.id);
		searchingIds = searchingIds;
		try {
			await searchUpgrade(item.type, item.id);
			toast.success(`Searching for upgrade: ${item.title}`);
			// Reload to update last searched
			await loadUpgrades();
		} catch (e) {
			toast.error('Search failed');
		} finally {
			searchingIds.delete(item.id);
			searchingIds = searchingIds;
		}
	}

	async function handleSearchAll(mediaType?: 'movie' | 'episode') {
		searchingAll = true;
		try {
			const result = await searchAllUpgrades(10, mediaType);
			toast.success(result.message);
			await loadUpgrades();
		} catch (e) {
			toast.error('Bulk search failed');
		} finally {
			searchingAll = false;
		}
	}

	async function handleResetSearch(item: UpgradeableItem) {
		resettingIds.add(item.id);
		resettingIds = resettingIds;
		try {
			await resetUpgradeSearch(item.type, item.id);
			toast.success(`Reset search for: ${item.title}`);
			await loadUpgrades();
		} catch (e) {
			toast.error('Reset failed');
		} finally {
			resettingIds.delete(item.id);
			resettingIds = resettingIds;
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

	function formatRelativeTimeFuture(dateStr: string): string {
		const diff = new Date(dateStr).getTime() - Date.now();
		if (diff <= 0) return 'now';
		const mins = Math.floor(diff / 60000);
		if (mins < 60) return `in ${mins}m`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `in ${hours}h`;
		return `in ${Math.floor(hours / 24)}d`;
	}

	let movies = $derived((summary as UpgradesSummary | null)?.movies || []);
	let episodes = $derived((summary as UpgradesSummary | null)?.episodes || []);
	let counts = $derived({
		movies: movies.length,
		episodes: episodes.length,
		total: (summary as UpgradesSummary | null)?.totalCount || 0
	});
</script>

<svelte:head>
	<title>Upgrades - Outpost</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-text-primary">Quality Upgrades</h1>
			{#if summary}
				<p class="text-sm text-text-muted mt-1">
					{counts.total} items below quality cutoff · {formatBytes(summary.totalSize)} potential savings
				</p>
				{#if summary.searching > 0 || summary.pendingRetry > 0}
					<div class="flex items-center gap-3 mt-1.5">
						{#if summary.searching > 0}
							<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-400 text-xs">
								<div class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></div>
								{summary.searching} searching
							</span>
						{/if}
						{#if summary.pendingRetry > 0}
							<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-yellow-500/20 text-yellow-400 text-xs">
								<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
								{summary.pendingRetry} waiting
							</span>
						{/if}
					</div>
				{/if}
			{/if}
		</div>
		<div class="flex items-center gap-2">
			{#if loading}
				<div class="spinner-sm text-text-muted"></div>
			{/if}
			<button
				class="px-4 py-2 rounded-lg bg-amber-400 text-black text-sm font-medium hover:bg-amber-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
				onclick={() => handleSearchAll()}
				disabled={searchingAll || counts.total === 0}
			>
				{#if searchingAll}
					<div class="spinner-sm"></div>
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				{/if}
				Search All
			</button>
		</div>
	</div>

	<!-- Tabs -->
	<div class="flex flex-wrap gap-2">
		<button
			class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeTab === 'movies' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}"
			onclick={() => activeTab = 'movies'}
		>
			Movies
			{#if counts.movies > 0}
				<span class="px-1.5 py-0.5 text-xs rounded-full {activeTab === 'movies' ? 'bg-black/20' : 'bg-blue-500 text-white'}">{counts.movies}</span>
			{/if}
		</button>
		<button
			class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeTab === 'episodes' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}"
			onclick={() => activeTab = 'episodes'}
		>
			Episodes
			{#if counts.episodes > 0}
				<span class="px-1.5 py-0.5 text-xs rounded-full {activeTab === 'episodes' ? 'bg-black/20' : 'bg-green-500 text-white'}">{counts.episodes}</span>
			{/if}
		</button>
	</div>

	{#if loading}
		<LoadingSpinner size="lg" fullPage />
	{:else}
		{@const items = activeTab === 'movies' ? movies : episodes}
		{#if items.length === 0}
			<EmptyState
				icon="M5 13l4 4L19 7"
				title={activeTab === 'movies' ? 'All movies meet quality targets' : 'All episodes meet quality targets'}
				description="No upgrades available at this time"
				compact
			/>
		{:else}
			<div class="space-y-2">
				{#each items as item (item.id)}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4">
						<div class="flex items-start gap-3">
							<!-- Poster -->
							{#if item.posterPath}
								<img
									src={getTmdbImageUrl(item.posterPath, 'w92')}
									alt=""
									class="w-12 h-18 rounded-lg object-cover flex-shrink-0"
								/>
							{:else}
								<div class="w-12 h-18 rounded-lg bg-bg-elevated flex items-center justify-center flex-shrink-0">
									<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
									</svg>
								</div>
							{/if}

							<!-- Content -->
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="min-w-0">
										<a
											href={item.type === 'movie' ? `/movies/${item.id}` : `/tv/${item.id}`}
											class="font-medium text-text-primary hover:text-amber-400 truncate block"
										>
											{item.title}
											{#if item.year}
												<span class="text-text-muted">({item.year})</span>
											{/if}
										</a>
										{#if item.type === 'episode' && item.showTitle}
											<p class="text-xs text-text-muted truncate mt-0.5">
												{item.showTitle} - S{String(item.seasonNumber).padStart(2, '0')}E{String(item.episodeNumber).padStart(2, '0')}
											</p>
										{/if}
									</div>

									<!-- Action buttons -->
									<div class="flex items-center gap-2">
										{#if item.searchStatus === 'searching'}
											<span class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg bg-blue-500/20 text-blue-400 text-xs">
												<div class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></div>
												Searching
											</span>
										{:else if item.searchStatus === 'pending_retry'}
											<span class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg bg-yellow-500/20 text-yellow-400 text-xs" title="Retries: {item.searchAttempts || 0}">
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
												{item.nextSearchAt ? formatRelativeTimeFuture(item.nextSearchAt) : 'waiting'}
											</span>
											<button
												class="px-2 py-1 rounded-lg text-xs font-medium bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
												onclick={() => handleResetSearch(item)}
												disabled={resettingIds.has(item.id)}
												title="Reset backoff and search again"
											>
												{#if resettingIds.has(item.id)}
													<div class="spinner-sm"></div>
												{:else}
													<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
													</svg>
												{/if}
											</button>
										{:else}
											<button
												class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50 flex items-center gap-1.5"
												onclick={() => handleSearch(item)}
												disabled={searchingIds.has(item.id)}
											>
												{#if searchingIds.has(item.id)}
													<div class="spinner-sm"></div>
													Searching...
												{:else}
													<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
													</svg>
													Search
												{/if}
											</button>
										{/if}
									</div>
								</div>

								<!-- Quality comparison -->
								<div class="flex items-center flex-wrap gap-x-3 gap-y-1 mt-2">
									<div class="flex items-center gap-2 text-xs">
										<span class="px-2 py-0.5 rounded bg-orange-500/20 text-orange-400 font-medium">
											{item.currentQuality || 'Unknown'}
										</span>
										<svg class="w-3 h-3 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" />
										</svg>
										<span class="px-2 py-0.5 rounded bg-green-500/20 text-green-400 font-medium">
											{item.cutoffQuality || 'Better'}
										</span>
									</div>
									<span class="text-xs text-text-muted">
										Score: {item.currentScore} → {item.cutoffScore}
									</span>
								</div>

								<!-- Meta info -->
								<div class="flex items-center gap-2 mt-2 text-xs text-text-muted">
									<span>{formatBytes(item.size)}</span>
									{#if item.lastSearched}
										<span>·</span>
										<span>Searched {formatRelativeTime(item.lastSearched)}</span>
									{:else}
										<span>·</span>
										<span>Never searched</span>
									{/if}
									{#if item.searchAttempts && item.searchAttempts > 0}
										<span>·</span>
										<span>{item.searchAttempts} attempt{item.searchAttempts > 1 ? 's' : ''}</span>
									{/if}
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>

			<!-- Search type button -->
			<div class="flex justify-center pt-4">
				<button
					class="px-4 py-2 rounded-lg bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary text-sm font-medium transition-colors disabled:opacity-50 flex items-center gap-2"
					onclick={() => handleSearchAll(activeTab === 'movies' ? 'movie' : 'episode')}
					disabled={searchingAll}
				>
					{#if searchingAll}
						<div class="spinner-sm"></div>
					{:else}
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
					{/if}
					Search Top 10 {activeTab === 'movies' ? 'Movies' : 'Episodes'}
				</button>
			</div>
		{/if}
	{/if}
</div>
