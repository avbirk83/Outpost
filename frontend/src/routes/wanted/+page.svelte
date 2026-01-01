<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getWantedItems,
		deleteWantedItem,
		updateWantedItem,
		searchWantedItem,
		grabRelease,
		getQualityProfiles,
		type WantedItem,
		type QualityProfile,
		type ScoredSearchResult
	} from '$lib/api';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	let items: WantedItem[] = $state([]);
	let profiles: QualityProfile[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let searching: Record<number, boolean> = $state({});
	let searchResults: Record<number, ScoredSearchResult[]> = $state({});
	let grabbing: Record<string, boolean> = $state({});
	let grabResults: Record<string, { success: boolean; message: string }> = $state({});

	onMount(async () => {
		await Promise.all([loadItems(), loadProfiles()]);
	});

	async function loadItems() {
		try {
			loading = true;
			items = await getWantedItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load wanted items';
		} finally {
			loading = false;
		}
	}

	async function loadProfiles() {
		try {
			profiles = await getQualityProfiles();
		} catch (e) {
			console.error('Failed to load profiles:', e);
		}
	}

	async function handleDelete(id: number) {
		if (!confirm('Remove this item from the wanted list?')) return;
		try {
			await deleteWantedItem(id);
			await loadItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete item';
		}
	}

	async function handleToggleMonitored(item: WantedItem) {
		try {
			await updateWantedItem(item.id, { monitored: !item.monitored });
			await loadItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update item';
		}
	}

	async function handleSearch(id: number) {
		try {
			searching[id] = true;
			delete searchResults[id];
			const results = await searchWantedItem(id);
			searchResults[id] = results;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Search failed';
		} finally {
			searching[id] = false;
		}
	}

	async function handleGrab(result: ScoredSearchResult, wantedId: number) {
		const key = `${wantedId}-${result.guid}`;
		grabbing[key] = true;
		delete grabResults[key];

		try {
			const response = await grabRelease({
				link: result.link,
				magnetLink: result.magnetLink,
				indexerType: result.indexerType,
				category: result.category
			});
			grabResults[key] = { success: response.success, message: response.message };
		} catch (e) {
			grabResults[key] = { success: false, message: e instanceof Error ? e.message : 'Grab failed' };
		} finally {
			grabbing[key] = false;
		}
	}

	function getProfileName(id: number): string {
		const profile = profiles.find(p => p.id === id);
		return profile?.name || 'Unknown';
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		try {
			const date = new Date(dateStr);
			return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		} catch {
			return dateStr;
		}
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatScore(score: number): string {
		if (score >= 1000) {
			return (score / 1000).toFixed(0) + 'k';
		}
		return score.toString();
	}

	function getQualityColor(quality: string | undefined): string {
		if (!quality) return 'bg-bg-elevated';
		if (quality.includes('Remux')) return 'bg-purple-600';
		if (quality.includes('2160p')) return 'bg-blue-600';
		if (quality.includes('1080p')) return 'bg-green-600';
		if (quality.includes('720p')) return 'bg-yellow-600';
		return 'bg-bg-elevated';
	}

	function getPosterUrl(path: string | undefined): string {
		if (!path) return '';
		return `https://image.tmdb.org/t/p/w92${path}`;
	}
</script>

<svelte:head>
	<title>Wanted - Outpost</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<h1 class="text-2xl font-bold text-text-primary">Wanted</h1>
			{#if items.length > 0}
				<div class="inline-flex items-center px-3 py-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10 text-sm text-white/60">
					{items.filter(i => i.monitored).length} monitored
				</div>
			{/if}
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
				<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading wanted items...</p>
			</div>
		</div>
	{:else if items.length === 0}
		<div class="glass-card p-12 text-center">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
				<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-text-primary mb-2">No items in your wanted list</h2>
			<p class="text-text-secondary">Add movies or TV shows from their detail pages to monitor them for upgrades.</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each items as item}
				<div class="glass-card overflow-hidden">
					<div class="flex items-start gap-4 p-4">
						{#if item.posterPath}
							<img
								src={getPosterUrl(item.posterPath)}
								alt={item.title}
								class="w-16 h-24 object-cover rounded-lg flex-shrink-0"
							/>
						{:else}
							<div class="w-16 h-24 bg-bg-elevated rounded-lg flex items-center justify-center flex-shrink-0">
								<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
								</svg>
							</div>
						{/if}

						<div class="flex-1 min-w-0">
							<div class="flex items-start justify-between gap-4">
								<div>
									<h3 class="font-medium text-lg text-text-primary">
										{item.title}
										{#if item.year}
											<span class="text-text-secondary">({item.year})</span>
										{/if}
									</h3>
									<div class="flex items-center gap-2 mt-1.5 text-sm flex-wrap">
										<TypeBadge type={item.type} />
										<span class="liquid-badge-sm">
											{getProfileName(item.qualityProfileId)}
										</span>
										{#if item.monitored}
											<span class="liquid-badge-sm !bg-green-500/20 !border-t-green-400/40 text-green-400">Monitored</span>
										{:else}
											<span class="liquid-badge-sm !bg-white/5 text-text-muted">Not Monitored</span>
										{/if}
									</div>
									<p class="text-xs text-text-muted mt-1.5">
										Added: {formatDate(item.addedAt)}
										{#if item.lastSearched}
											<span class="mx-1">|</span>
											Last searched: {formatDate(item.lastSearched)}
										{/if}
									</p>
								</div>

								<div class="flex items-center gap-2 flex-shrink-0">
									<button
										class="w-12 h-7 rounded-full relative transition-colors {item.monitored ? 'bg-green-600' : 'bg-bg-elevated'}"
										onclick={() => handleToggleMonitored(item)}
										title={item.monitored ? 'Monitoring' : 'Not monitoring'}
									>
										<span
											class="absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform {item.monitored ? 'left-6' : 'left-1'}"
										></span>
									</button>
									<button
										class="liquid-btn-sm disabled:opacity-50"
										onclick={() => handleSearch(item.id)}
										disabled={searching[item.id]}
									>
										{searching[item.id] ? 'Searching...' : 'Search'}
									</button>
									<button
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
										onclick={() => handleDelete(item.id)}
									>
										Remove
									</button>
								</div>
							</div>
						</div>
					</div>

					{#if searchResults[item.id] && searchResults[item.id].length > 0}
						<div class="border-t border-white/10 p-4 bg-bg-primary/50">
							<h4 class="text-sm font-medium text-text-secondary mb-3">Search Results ({searchResults[item.id].length})</h4>
							<div class="overflow-x-auto">
								<table class="w-full text-sm">
									<thead class="text-xs text-text-muted border-b border-white/10">
										<tr>
											<th class="py-2 px-2 text-left">Title</th>
											<th class="py-2 px-2 text-left">Quality</th>
											<th class="py-2 px-2 text-center">Score</th>
											<th class="py-2 px-2 text-right">Size</th>
											<th class="py-2 px-2 text-center">S/L</th>
											<th class="py-2 px-2"></th>
										</tr>
									</thead>
									<tbody class="divide-y divide-white/5">
										{#each searchResults[item.id].slice(0, 10) as result}
											{@const key = `${item.id}-${result.guid}`}
											<tr class="hover:bg-white/5 transition-colors {result.rejected ? 'opacity-50' : ''}">
												<td class="py-2 px-2">
													<p class="truncate max-w-xs text-text-primary" title={result.title}>{result.title}</p>
													<span class="text-xs text-text-muted">{result.indexerName}</span>
												</td>
												<td class="py-2 px-2">
													<div class="flex flex-wrap gap-1">
														{#if result.quality}
															<span class="px-1.5 py-0.5 text-xs rounded-lg {getQualityColor(result.quality)} text-white font-medium">{result.quality}</span>
														{/if}
														{#if result.codec}
															<span class="px-1.5 py-0.5 text-xs rounded-lg bg-bg-elevated text-text-secondary">{result.codec}</span>
														{/if}
													</div>
												</td>
												<td class="py-2 px-2 text-center">
													<span class="font-bold {result.rejected ? 'text-text-secondary' : 'text-green-400'}">{formatScore(result.totalScore)}</span>
												</td>
												<td class="py-2 px-2 text-right text-text-secondary whitespace-nowrap">{formatSize(result.size)}</td>
												<td class="py-2 px-2 text-center whitespace-nowrap">
													{#if result.seeders > 0}
														<span class="text-green-400">{result.seeders}</span><span class="text-text-muted">/</span><span class="text-text-secondary">{result.leechers}</span>
													{:else}
														<span class="text-text-muted">-</span>
													{/if}
												</td>
												<td class="py-2 px-2">
													{#if grabResults[key]}
														<span class="text-xs flex items-center gap-1 {grabResults[key].success ? 'text-green-400' : 'text-text-secondary'}">
															<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																{#if grabResults[key].success}
																	<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
																{:else}
																	<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
																{/if}
															</svg>
															{grabResults[key].success ? 'Added' : 'Failed'}
														</span>
													{:else}
														<button
															class="liquid-btn-sm !py-1 !px-2 text-xs !bg-green-500/20 !border-t-green-400/40 text-green-400 hover:!bg-green-500/30 disabled:opacity-50"
															onclick={() => handleGrab(result, item.id)}
															disabled={grabbing[key]}
														>
															{grabbing[key] ? '...' : 'Grab'}
														</button>
													{/if}
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
								{#if searchResults[item.id].length > 10}
									<p class="text-xs text-text-muted mt-2 text-center">Showing top 10 of {searchResults[item.id].length} results</p>
								{/if}
							</div>
						</div>
					{:else if searchResults[item.id] && searchResults[item.id].length === 0}
						<div class="border-t border-white/10 p-4 text-center text-text-muted text-sm">
							No results found
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
