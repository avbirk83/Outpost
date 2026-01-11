<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		getStorageAnalytics,
		type StorageAnalytics,
		type LibrarySize,
		type QualitySize,
		type LargestItem,
		type DuplicateItem
	} from '$lib/api';

	let analytics: StorageAnalytics | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let expandedDuplicates: Set<number> = $state(new Set());

	onMount(() => {
		loadAnalytics();
	});

	async function loadAnalytics() {
		try {
			loading = true;
			error = null;
			analytics = await getStorageAnalytics();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load storage analytics';
		} finally {
			loading = false;
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function getUsagePercent(used: number, total: number): number {
		if (total === 0) return 0;
		return Math.round((used / total) * 100);
	}

	function getUsageColor(percent: number): string {
		if (percent < 70) return 'bg-green-500';
		if (percent < 90) return 'bg-amber-500';
		return 'bg-red-500';
	}

	function getUsageTextColor(percent: number): string {
		if (percent < 70) return 'text-green-400';
		if (percent < 90) return 'text-amber-400';
		return 'text-red-400';
	}

	function getLibraryColor(type: string): string {
		switch (type) {
			case 'movies': return 'bg-blue-500';
			case 'tv': return 'bg-purple-500';
			case 'anime': return 'bg-pink-500';
			case 'music': return 'bg-green-500';
			case 'books': return 'bg-amber-500';
			default: return 'bg-gray-500';
		}
	}

	function getQualityColor(quality: string): string {
		switch (quality) {
			case '2160p': return 'bg-purple-500';
			case '1080p': return 'bg-blue-500';
			case '720p': return 'bg-green-500';
			case '480p': return 'bg-amber-500';
			default: return 'bg-gray-500';
		}
	}

	function toggleDuplicate(tmdbId: number) {
		const newSet = new Set(expandedDuplicates);
		if (newSet.has(tmdbId)) {
			newSet.delete(tmdbId);
		} else {
			newSet.add(tmdbId);
		}
		expandedDuplicates = newSet;
	}

	function navigateToItem(item: LargestItem) {
		if (item.type === 'movie') {
			goto(`/movies/${item.id}`);
		} else {
			// For episodes, we'd need to navigate to the show/episode
			// For now, just log it
			console.log('Navigate to episode', item.id);
		}
	}

	// Calculate total library size for percentage bars
	const totalLibrarySize = $derived.by(() => {
		if (!analytics) return 0;
		return analytics.byLibrary.reduce((sum: number, lib: LibrarySize) => sum + lib.size, 0);
	});

	// Calculate total quality size for percentage bars
	const totalQualitySize = $derived.by(() => {
		if (!analytics) return 0;
		return analytics.byQuality.reduce((sum: number, q: QualitySize) => sum + q.size, 0);
	});
</script>

<div class="space-y-6">
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<svg class="w-8 h-8 animate-spin text-text-muted" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
			<span class="ml-3 text-text-muted">Loading storage analytics...</span>
		</div>
	{:else if error}
		<div class="bg-red-900/20 border border-red-600/30 text-red-400 px-4 py-3 rounded-xl">
			{error}
			<button class="ml-4 underline" onclick={loadAnalytics}>Retry</button>
		</div>
	{:else if analytics}
		<!-- Overview Cards -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
			<!-- Total Storage -->
			<div class="glass-card p-6">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-xl bg-blue-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
						</svg>
					</div>
					<div>
						<p class="text-sm text-text-muted">Total Storage</p>
						<p class="text-2xl font-bold text-text-primary">{formatBytes(analytics.total)}</p>
					</div>
				</div>
			</div>

			<!-- Used Storage -->
			<div class="glass-card p-6">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-xl bg-amber-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
						</svg>
					</div>
					<div>
						<p class="text-sm text-text-muted">Used</p>
						<p class="text-2xl font-bold {getUsageTextColor(getUsagePercent(analytics.used, analytics.total))}">
							{formatBytes(analytics.used)}
						</p>
					</div>
				</div>
				<div class="w-full h-2 bg-white/5 rounded-full overflow-hidden">
					<div
						class="h-full {getUsageColor(getUsagePercent(analytics.used, analytics.total))} transition-all duration-500"
						style="width: {getUsagePercent(analytics.used, analytics.total)}%"
					></div>
				</div>
				<p class="text-xs text-text-muted mt-2">{getUsagePercent(analytics.used, analytics.total)}% used</p>
			</div>

			<!-- Free Storage -->
			<div class="glass-card p-6">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-xl bg-green-600/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
					</div>
					<div>
						<p class="text-sm text-text-muted">Free</p>
						<p class="text-2xl font-bold text-green-400">{formatBytes(analytics.free)}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- By Library and By Quality Charts -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- By Library -->
			<div class="glass-card p-6">
				<h3 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
					<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					Storage by Library
				</h3>

				{#if analytics.byLibrary.length === 0}
					<p class="text-text-muted text-sm">No libraries found</p>
				{:else}
					<div class="space-y-4">
						{#each analytics.byLibrary as lib}
							{@const percent = totalLibrarySize > 0 ? (lib.size / totalLibrarySize) * 100 : 0}
							<div>
								<div class="flex items-center justify-between mb-1">
									<div class="flex items-center gap-2">
										<span class="w-3 h-3 rounded-full {getLibraryColor(lib.type)}"></span>
										<span class="text-sm text-text-primary">{lib.name}</span>
										<span class="text-xs text-text-muted">({lib.count} items)</span>
									</div>
									<span class="text-sm font-medium text-text-secondary">{formatBytes(lib.size)}</span>
								</div>
								<div class="w-full h-2 bg-white/5 rounded-full overflow-hidden">
									<div
										class="h-full {getLibraryColor(lib.type)} transition-all duration-500"
										style="width: {percent}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- By Quality -->
			<div class="glass-card p-6">
				<h3 class="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
					<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
					</svg>
					Storage by Quality
				</h3>

				{#if analytics.byQuality.length === 0}
					<p class="text-text-muted text-sm">No quality data available</p>
				{:else}
					<div class="space-y-4">
						{#each analytics.byQuality as quality}
							{@const percent = totalQualitySize > 0 ? (quality.size / totalQualitySize) * 100 : 0}
							<div>
								<div class="flex items-center justify-between mb-1">
									<div class="flex items-center gap-2">
										<span class="w-3 h-3 rounded-full {getQualityColor(quality.quality)}"></span>
										<span class="text-sm text-text-primary">{quality.quality}</span>
										<span class="text-xs text-text-muted">({quality.count} files)</span>
									</div>
									<span class="text-sm font-medium text-text-secondary">{formatBytes(quality.size)}</span>
								</div>
								<div class="w-full h-2 bg-white/5 rounded-full overflow-hidden">
									<div
										class="h-full {getQualityColor(quality.quality)} transition-all duration-500"
										style="width: {percent}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>

		<!-- Duplicates Alert -->
		{#if analytics.duplicates.length > 0}
			<div class="glass-card p-6 border-l-4 border-amber-500">
				<div class="flex items-start gap-3">
					<div class="w-10 h-10 rounded-xl bg-amber-600/20 flex items-center justify-center flex-shrink-0">
						<svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
						</svg>
					</div>
					<div class="flex-1">
						<h3 class="text-lg font-semibold text-amber-400 mb-1">
							{analytics.duplicates.length} Duplicate{analytics.duplicates.length !== 1 ? 's' : ''} Found
						</h3>
						<p class="text-sm text-text-secondary mb-4">
							The following items have multiple copies. Consider removing duplicates to save space.
						</p>

						<div class="space-y-2">
							{#each analytics.duplicates as dup}
								<div class="bg-white/5 rounded-lg overflow-hidden">
									<button
										class="w-full px-4 py-3 flex items-center justify-between text-left hover:bg-white/5 transition-colors"
										onclick={() => toggleDuplicate(dup.tmdbId)}
									>
										<div>
											<span class="text-text-primary font-medium">{dup.title}</span>
											<span class="text-text-muted ml-2">({dup.year})</span>
											<span class="text-xs text-amber-400 ml-2">{dup.copies.length} copies</span>
										</div>
										<svg
											class="w-5 h-5 text-text-muted transition-transform {expandedDuplicates.has(dup.tmdbId) ? 'rotate-180' : ''}"
											fill="none" stroke="currentColor" viewBox="0 0 24 24"
										>
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
										</svg>
									</button>

									{#if expandedDuplicates.has(dup.tmdbId)}
										<div class="px-4 pb-3 space-y-2">
											{#each dup.copies as copy, i}
												<div class="flex items-center justify-between py-2 px-3 bg-white/5 rounded-lg text-sm">
													<div class="flex items-center gap-3">
														<span class="px-2 py-0.5 text-xs rounded bg-white/10 text-text-muted">
															{copy.quality}
														</span>
														<span class="text-text-secondary truncate max-w-md" title={copy.path}>
															{copy.path}
														</span>
													</div>
													<span class="text-text-primary font-medium">{formatBytes(copy.size)}</span>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Largest Files Table -->
		<div class="glass-card p-6">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold text-text-primary flex items-center gap-2">
					<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
					</svg>
					Largest Files
				</h3>
				<span class="text-xs text-text-muted">Top 20</span>
			</div>

			{#if analytics.largest.length === 0}
				<p class="text-text-muted text-sm">No files found</p>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="text-left text-text-muted border-b border-white/5">
								<th class="pb-3 font-medium">Title</th>
								<th class="pb-3 font-medium">Year</th>
								<th class="pb-3 font-medium">Quality</th>
								<th class="pb-3 font-medium text-right">Size</th>
							</tr>
						</thead>
						<tbody>
							{#each analytics.largest as item, i}
								<tr
									class="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
									onclick={() => navigateToItem(item)}
								>
									<td class="py-3">
										<div class="flex items-center gap-2">
											<span class="text-xs px-1.5 py-0.5 rounded bg-white/10 text-text-muted uppercase">
												{item.type === 'movie' ? 'M' : 'E'}
											</span>
											<span class="text-text-primary">{item.title}</span>
										</div>
									</td>
									<td class="py-3 text-text-secondary">{item.year}</td>
									<td class="py-3">
										<span class="px-2 py-0.5 text-xs rounded {getQualityColor(item.quality.split(' ')[0])} text-white">
											{item.quality}
										</span>
									</td>
									<td class="py-3 text-right font-medium text-text-primary">{formatBytes(item.size)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>

		<!-- By Year (Collapsed by default) -->
		{#if analytics.byYear.length > 0}
			<details class="glass-card">
				<summary class="p-6 cursor-pointer flex items-center justify-between hover:bg-white/5 transition-colors">
					<h3 class="text-lg font-semibold text-text-primary flex items-center gap-2">
						<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
						Storage by Release Year
					</h3>
					<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
					</svg>
				</summary>
				<div class="px-6 pb-6">
					<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
						{#each analytics.byYear.slice(0, 24) as year}
							<div class="bg-white/5 rounded-lg p-3 text-center">
								<p class="text-lg font-bold text-text-primary">{year.year}</p>
								<p class="text-xs text-text-muted">{year.count} items</p>
								<p class="text-sm text-text-secondary">{formatBytes(year.size)}</p>
							</div>
						{/each}
					</div>
				</div>
			</details>
		{/if}

		<!-- Refresh Button -->
		<div class="flex justify-center">
			<button
				class="liquid-btn flex items-center gap-2"
				onclick={loadAnalytics}
				disabled={loading}
			>
				<svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Refresh Analytics
			</button>
		</div>
	{/if}
</div>
