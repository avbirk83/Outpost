<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getUpgrades,
		searchUpgrade,
		searchAllUpgrades,
		resetUpgradeSearch,
		pauseUpgrade,
		getImageUrl,
		type UpgradeableItem,
		type UpgradesSummary
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { LoadingSpinner, EmptyState } from '$lib/components/ui';

	type Tab = 'movies' | 'tv';
	let activeTab: Tab = $state('movies');

	let summary: UpgradesSummary | null = $state(null);
	let loading = $state(true);
	let searchingIds: Set<number> = $state(new Set());
	let searchingAll = $state(false);
	let resettingIds: Set<number> = $state(new Set());
	let pausingIds: Set<number> = $state(new Set());
	let expandedShows: Set<number> = $state(new Set());

	// Group episodes by show
	interface ShowGroup {
		showId: number;
		showTitle: string;
		posterPath?: string;
		episodes: UpgradeableItem[];
		totalSize: number;
	}

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

	async function handlePause(item: UpgradeableItem, pause: boolean) {
		pausingIds.add(item.id);
		pausingIds = pausingIds;
		try {
			await pauseUpgrade(item.type, item.id, pause);
			toast.success(pause ? `Paused upgrades for: ${item.title}` : `Resumed upgrades for: ${item.title}`);
			await loadUpgrades();
		} catch (e) {
			toast.error('Failed to update pause status');
		} finally {
			pausingIds.delete(item.id);
			pausingIds = pausingIds;
		}
	}

	function toggleShow(showId: number) {
		if (expandedShows.has(showId)) {
			expandedShows.delete(showId);
		} else {
			expandedShows.add(showId);
		}
		expandedShows = expandedShows;
	}

	function formatBytes(bytes: number): string {
		if (!bytes || bytes <= 0 || !isFinite(bytes)) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		if (i < 0 || i >= sizes.length || !isFinite(i)) return '0 B';
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

	let movies = $derived(summary?.movies ?? []);
	let episodes = $derived(summary?.episodes ?? []);

	// Group episodes by show
	let showGroups = $derived.by(() => {
		const groups = new Map<number, ShowGroup>();
		for (const ep of episodes) {
			const showId = ep.showId ?? 0;
			if (!groups.has(showId)) {
				groups.set(showId, {
					showId,
					showTitle: ep.showTitle ?? 'Unknown Show',
					posterPath: ep.posterPath,
					episodes: [],
					totalSize: 0
				});
			}
			const group = groups.get(showId)!;
			group.episodes.push(ep);
			group.totalSize += ep.size || 0;
		}
		// Sort episodes within each group by season/episode
		for (const group of groups.values()) {
			group.episodes.sort((a, b) => {
				const seasonDiff = (a.seasonNumber ?? 0) - (b.seasonNumber ?? 0);
				if (seasonDiff !== 0) return seasonDiff;
				return (a.episodeNumber ?? 0) - (b.episodeNumber ?? 0);
			});
		}
		return Array.from(groups.values()).sort((a, b) => b.episodes.length - a.episodes.length);
	});

	let counts = $derived({
		movies: movies.length,
		shows: showGroups.length,
		episodes: episodes.length,
		total: summary?.totalCount ?? 0
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
					{counts.total} items below quality cutoff · {formatBytes(summary.totalSize)} current storage
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
			class="px-3 py-1.5 rounded-lg text-sm font-medium transition-all flex items-center gap-1.5 {activeTab === 'tv' ? 'bg-amber-400 text-black' : 'bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary'}"
			onclick={() => activeTab = 'tv'}
		>
			TV
			{#if counts.shows > 0}
				<span class="px-1.5 py-0.5 text-xs rounded-full {activeTab === 'tv' ? 'bg-black/20' : 'bg-green-500 text-white'}">{counts.shows}</span>
			{/if}
		</button>
	</div>

	{#if loading}
		<LoadingSpinner size="lg" fullPage />
	{:else if activeTab === 'movies'}
		{#if movies.length === 0}
			<EmptyState
				icon="M5 13l4 4L19 7"
				title="All movies meet quality targets"
				description="No upgrades available at this time"
				compact
			/>
		{:else}
			<div class="space-y-2">
				{#each movies as item (item.id)}
					<div class="bg-bg-card border border-border-subtle rounded-xl p-4">
						<div class="flex items-start gap-3">
							<!-- Poster -->
							{#if item.posterPath}
								<img
									src={getImageUrl(item.posterPath)}
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
											href="/movies/{item.id}"
											class="font-medium text-text-primary hover:text-amber-400 truncate block"
										>
											{item.title}
											{#if item.year}
												<span class="text-text-muted">({item.year})</span>
											{/if}
										</a>
									</div>

									<!-- Action buttons -->
									<div class="flex items-center gap-2">
										{#if item.searchStatus === 'paused' || item.upgradePaused}
											<span class="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg bg-gray-500/20 text-gray-400 text-xs">
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
												Paused
											</span>
											<button
												class="px-2 py-1 rounded-lg text-xs font-medium bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors disabled:opacity-50"
												onclick={() => handlePause(item, false)}
												disabled={pausingIds.has(item.id)}
												title="Resume upgrade search"
											>
												{#if pausingIds.has(item.id)}
													<div class="spinner-sm"></div>
												{:else}
													<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
													</svg>
												{/if}
											</button>
										{:else if item.searchStatus === 'searching'}
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
											<button
												class="px-2 py-1 rounded-lg text-xs font-medium bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
												onclick={() => handlePause(item, true)}
												disabled={pausingIds.has(item.id)}
												title="Pause upgrade search"
											>
												{#if pausingIds.has(item.id)}
													<div class="spinner-sm"></div>
												{:else}
													<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
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
											<button
												class="px-2 py-1 rounded-lg text-xs font-medium bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
												onclick={() => handlePause(item, true)}
												disabled={pausingIds.has(item.id)}
												title="Pause upgrade search"
											>
												{#if pausingIds.has(item.id)}
													<div class="spinner-sm"></div>
												{:else}
													<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
													</svg>
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
					onclick={() => handleSearchAll('movie')}
					disabled={searchingAll}
				>
					{#if searchingAll}
						<div class="spinner-sm"></div>
					{:else}
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
					{/if}
					Search Top 10 Movies
				</button>
			</div>
		{/if}
	{:else}
		<!-- TV Shows Tab -->
		{#if showGroups.length === 0}
			<EmptyState
				icon="M5 13l4 4L19 7"
				title="All TV shows meet quality targets"
				description="No upgrades available at this time"
				compact
			/>
		{:else}
			<div class="space-y-2">
				{#each showGroups as group (group.showId)}
					<div class="bg-bg-card border border-border-subtle rounded-xl overflow-hidden">
						<!-- Show Header (clickable to expand) -->
						<button
							class="w-full p-4 flex items-center gap-3 hover:bg-white/5 transition-colors text-left"
							onclick={() => toggleShow(group.showId)}
						>
							<!-- Poster -->
							{#if group.posterPath}
								<img
									src={getImageUrl(group.posterPath)}
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

							<!-- Show Info -->
							<div class="flex-1 min-w-0">
								<div class="flex items-center justify-between gap-2">
									<a
										href="/tv/{group.showId}"
										class="font-medium text-text-primary hover:text-amber-400 truncate"
										onclick={(e) => e.stopPropagation()}
									>
										{group.showTitle}
									</a>
									<div class="flex items-center gap-2">
										<span class="px-2 py-0.5 text-xs rounded-full bg-green-500/20 text-green-400">
											{group.episodes.length} episode{group.episodes.length > 1 ? 's' : ''}
										</span>
										<svg
											class="w-4 h-4 text-text-muted transition-transform {expandedShows.has(group.showId) ? 'rotate-180' : ''}"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
										</svg>
									</div>
								</div>
								<p class="text-xs text-text-muted mt-1">{formatBytes(group.totalSize)} total</p>
							</div>
						</button>

						<!-- Episodes List (collapsible) -->
						{#if expandedShows.has(group.showId)}
							<div class="border-t border-border-subtle">
								{#each group.episodes as item (item.id)}
									<div class="p-3 pl-8 border-b border-border-subtle last:border-b-0 hover:bg-white/5">
										<div class="flex items-center justify-between gap-2">
											<div class="min-w-0 flex-1">
												<div class="flex items-center gap-2">
													<span class="text-xs text-text-muted font-mono">
														S{String(item.seasonNumber).padStart(2, '0')}E{String(item.episodeNumber).padStart(2, '0')}
													</span>
													<span class="text-sm text-text-primary truncate">{item.title}</span>
												</div>
												<div class="flex items-center gap-2 mt-1">
													<span class="px-1.5 py-0.5 rounded text-xs bg-orange-500/20 text-orange-400">
														{item.currentQuality || 'Unknown'}
													</span>
													<svg class="w-2.5 h-2.5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" />
													</svg>
													<span class="px-1.5 py-0.5 rounded text-xs bg-green-500/20 text-green-400">
														{item.cutoffQuality || 'Better'}
													</span>
													<span class="text-xs text-text-muted ml-2">{formatBytes(item.size)}</span>
												</div>
											</div>

											<!-- Action buttons -->
											<div class="flex items-center gap-2">
												{#if item.searchStatus === 'paused' || item.upgradePaused}
													<span class="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-gray-500/20 text-gray-400 text-xs">
														<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
														</svg>
														Paused
													</span>
													<button
														class="p-1 rounded-lg text-xs bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors disabled:opacity-50"
														onclick={() => handlePause(item, false)}
														disabled={pausingIds.has(item.id)}
														title="Resume"
													>
														{#if pausingIds.has(item.id)}
															<div class="spinner-sm"></div>
														{:else}
															<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
															</svg>
														{/if}
													</button>
												{:else if item.searchStatus === 'searching'}
													<span class="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-blue-500/20 text-blue-400 text-xs">
														<div class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></div>
														Searching
													</span>
												{:else if item.searchStatus === 'pending_retry'}
													<span class="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-yellow-500/20 text-yellow-400 text-xs">
														<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
														</svg>
														{item.nextSearchAt ? formatRelativeTimeFuture(item.nextSearchAt) : 'waiting'}
													</span>
													<button
														class="p-1 rounded-lg text-xs bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
														onclick={() => handleResetSearch(item)}
														disabled={resettingIds.has(item.id)}
														title="Reset backoff"
													>
														{#if resettingIds.has(item.id)}
															<div class="spinner-sm"></div>
														{:else}
															<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
															</svg>
														{/if}
													</button>
													<button
														class="p-1 rounded-lg text-xs bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
														onclick={() => handlePause(item, true)}
														disabled={pausingIds.has(item.id)}
														title="Pause"
													>
														{#if pausingIds.has(item.id)}
															<div class="spinner-sm"></div>
														{:else}
															<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
															</svg>
														{/if}
													</button>
												{:else}
													<button
														class="px-2 py-1 rounded-lg text-xs font-medium bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50 flex items-center gap-1"
														onclick={() => handleSearch(item)}
														disabled={searchingIds.has(item.id)}
													>
														{#if searchingIds.has(item.id)}
															<div class="spinner-sm"></div>
														{:else}
															<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
															</svg>
															Search
														{/if}
													</button>
													<button
														class="p-1 rounded-lg text-xs bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
														onclick={() => handlePause(item, true)}
														disabled={pausingIds.has(item.id)}
														title="Pause"
													>
														{#if pausingIds.has(item.id)}
															<div class="spinner-sm"></div>
														{:else}
															<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
															</svg>
														{/if}
													</button>
												{/if}
											</div>
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>

			<!-- Search type button -->
			<div class="flex justify-center pt-4">
				<button
					class="px-4 py-2 rounded-lg bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary text-sm font-medium transition-colors disabled:opacity-50 flex items-center gap-2"
					onclick={() => handleSearchAll('episode')}
					disabled={searchingAll}
				>
					{#if searchingAll}
						<div class="spinner-sm"></div>
					{:else}
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
					{/if}
					Search Top 10 Episodes
				</button>
			</div>
		{/if}
	{/if}
</div>
