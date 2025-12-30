<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getShow, refreshShowMetadata, getImageUrl, createWantedItem,
		getQualityProfiles, markAsWatched, markAsUnwatched, getMediaInfo,
		type ShowDetail, type QualityProfile, type MediaInfo
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import CastRow from '$lib/components/CastRow.svelte';
	import MediaSelector from '$lib/components/MediaSelector.svelte';
	import EpisodeGrid from '$lib/components/EpisodeGrid.svelte';
	import Dropdown from '$lib/components/Dropdown.svelte';

	let show: ShowDetail | null = $state(null);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let addingToWanted = $state(false);
	let addedToWanted = $state(false);
	let profiles: QualityProfile[] = $state([]);
	let user = $state<{ role: string } | null>(null);
	let watchedEpisodes: Set<number> = $state(new Set());
	let togglingEpisode: number | null = $state(null);
	let markingAllWatched = $state(false);
	let selectedSeasonIndex = $state(0);
	let mediaInfo: MediaInfo | null = $state(null);

	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			show = await getShow(id);
			if (user?.role === 'admin') {
				profiles = await getQualityProfiles();
			}
			// Get media info for first episode if available
			if (show.seasons?.[0]?.episodes?.[0]) {
				try {
					mediaInfo = await getMediaInfo('episode', show.seasons[0].episodes[0].id);
				} catch {}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load show';
		} finally {
			loading = false;
		}
	});

	async function handleRefresh() {
		if (!show) return;
		refreshing = true;
		try {
			show = await refreshShowMetadata(show.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh metadata';
		} finally {
			refreshing = false;
		}
	}

	async function handleAddToWanted() {
		if (!show || !show.tmdbId) return;
		addingToWanted = true;
		try {
			await createWantedItem({
				type: 'show',
				tmdbId: show.tmdbId,
				title: show.title,
				year: show.year,
				posterPath: show.posterPath || undefined,
				qualityProfileId: profiles.length > 0 ? profiles[0].id : 1,
				monitored: true,
				seasons: '[]'
			});
			addedToWanted = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add to wanted';
		} finally {
			addingToWanted = false;
		}
	}

	async function handleToggleEpisodeWatched(episodeId: number, runtime?: number) {
		togglingEpisode = episodeId;
		try {
			if (watchedEpisodes.has(episodeId)) {
				await markAsUnwatched('episode', episodeId);
				watchedEpisodes.delete(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			} else {
				await markAsWatched('episode', episodeId, runtime ? runtime * 60 : 2400);
				watchedEpisodes.add(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update watch status';
		} finally {
			togglingEpisode = null;
		}
	}

	async function handleMarkAllWatched() {
		if (!selectedSeason?.episodes) return;
		markingAllWatched = true;
		try {
			const allWatched = selectedSeason.episodes.every(ep => watchedEpisodes.has(ep.id));
			for (const episode of selectedSeason.episodes) {
				if (allWatched) {
					await markAsUnwatched('episode', episode.id);
					watchedEpisodes.delete(episode.id);
				} else if (!watchedEpisodes.has(episode.id)) {
					await markAsWatched('episode', episode.id, episode.runtime ? episode.runtime * 60 : 2400);
					watchedEpisodes.add(episode.id);
				}
			}
			watchedEpisodes = new Set(watchedEpisodes);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update watch status';
		} finally {
			markingAllWatched = false;
		}
	}

	function handlePlayNext() {
		if (!show?.seasons) return;
		// Find first unwatched episode
		for (const season of show.seasons) {
			for (const ep of season.episodes || []) {
				if (!watchedEpisodes.has(ep.id)) {
					goto(`/watch/episode/${ep.id}`);
					return;
				}
			}
		}
		// All watched, play first episode
		const firstEp = show.seasons[0]?.episodes?.[0];
		if (firstEp) {
			goto(`/watch/episode/${firstEp.id}`);
		}
	}

	const selectedSeason = $derived(show?.seasons?.[selectedSeasonIndex]);

	const seasonWatchedCount = $derived(() => {
		if (!selectedSeason?.episodes) return 0;
		return selectedSeason.episodes.filter(ep => watchedEpisodes.has(ep.id)).length;
	});

	const nextEpisode = $derived(() => {
		if (!show?.seasons) return null;
		for (const season of show.seasons) {
			for (const ep of season.episodes || []) {
				if (!watchedEpisodes.has(ep.id)) {
					return { season: season.seasonNumber, episode: ep.episodeNumber };
				}
			}
		}
		return null;
	});

	const totalEpisodes = $derived(() => {
		if (!show?.seasons) return 0;
		return show.seasons.reduce((sum, s) => sum + (s.episodes?.length || 0), 0);
	});

	function parseGenres(genres: string | undefined): string[] {
		if (!genres) return [];
		try { return JSON.parse(genres); } catch { return []; }
	}

	function parseCast(cast: string | undefined): Array<{ name: string; character: string; profile_path?: string }> {
		if (!cast) return [];
		try { return JSON.parse(cast); } catch { return []; }
	}

	function getFocalPosition(focalX?: number, focalY?: number): string {
		const x = focalX !== undefined ? Math.round(focalX * 100) : 50;
		const y = focalY !== undefined ? Math.round(focalY * 100) : 25;
		return `${x}% ${y}%`;
	}

	function getStatusColor(status?: string): string {
		switch (status?.toLowerCase()) {
			case 'returning series':
			case 'in production':
				return 'text-green-400';
			default:
				return 'text-text-secondary';
		}
	}
</script>

<svelte:head>
	<title>{show?.title || 'TV Show'} - Outpost</title>
</svelte:head>

<!-- Full viewport - no scroll -->
<div class="absolute inset-0 overflow-hidden">
	{#if error}
		<div class="absolute top-4 left-1/2 -translate-x-1/2 z-50 bg-red-500/20 border border-red-500/50 text-red-200 px-4 py-2 rounded-lg text-sm">
			{error}
			<button class="ml-2 underline" onclick={() => (error = null)}>Dismiss</button>
		</div>
	{/if}

	{#if loading}
		<div class="h-full flex items-center justify-center">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white/40 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading...</p>
			</div>
		</div>
	{:else if show}
		<!-- Backdrop -->
		{#if show.backdropPath}
			<img
				src={getImageUrl(show.backdropPath)}
				alt=""
				class="absolute inset-0 w-full h-full object-cover"
				style="object-position: {getFocalPosition(show.focalX, show.focalY)};"
			/>
		{/if}

		<!-- Gradient overlays -->
		<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/90 to-bg-primary/50"></div>
		<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-bg-primary/50 to-bg-primary/70"></div>

		<!-- Content layout -->
		<div class="relative h-full flex flex-col">
			<!-- Top bar -->
			<div class="flex-shrink-0 flex items-center justify-between px-6 py-4">
				<a href="/library" class="flex items-center gap-2 text-text-secondary hover:text-white text-sm transition-colors">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
					Back
				</a>

				<div class="flex items-center gap-3">
					<button
						onclick={handleRefresh}
						disabled={refreshing}
						class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-text-secondary hover:text-white transition-colors disabled:opacity-50"
						title="Refresh metadata"
					>
						<svg class="w-4 h-4 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
						</svg>
					</button>
				</div>
			</div>

			<!-- Main content - poster + info -->
			<div class="flex-1 flex gap-6 lg:gap-8 px-6 min-h-0">
				<!-- Left: Poster -->
				<div class="flex-shrink-0 w-40 lg:w-48 flex flex-col">
					{#if show.posterPath}
						<img
							src={getImageUrl(show.posterPath)}
							alt={show.title}
							class="w-full rounded-lg shadow-2xl"
						/>
					{:else}
						<div class="w-full aspect-[2/3] bg-bg-elevated rounded-lg flex items-center justify-center">
							<svg class="w-12 h-12 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
							</svg>
						</div>
					{/if}
				</div>

				<!-- Right: Info -->
				<div class="flex-1 flex flex-col min-w-0 overflow-hidden">
					<!-- Title row with ratings -->
					<div class="flex items-start justify-between gap-4">
						<h1 class="text-2xl lg:text-3xl font-bold text-white leading-tight">{show.title}</h1>

						<div class="flex-shrink-0 flex items-center gap-3">
							{#if show.rating}
								<a href="https://www.themoviedb.org/tv/{show.tmdbId}" target="_blank" class="flex items-center gap-1.5 hover:opacity-80 transition-opacity" title="TMDB Rating">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-5 h-5 rounded" />
									<span class="text-sm font-medium text-white">{show.rating.toFixed(1)}</span>
								</a>
							{/if}
						</div>
					</div>

					<!-- Meta row -->
					<div class="flex flex-wrap items-center gap-2 mt-2 text-sm text-text-secondary">
						{#if show.year}<span>{show.year}</span>{/if}
						<span class="text-text-muted">•</span>
						<span>{show.seasons?.length || 0} Seasons</span>
						<span class="text-text-muted">•</span>
						<span>{totalEpisodes()} Episodes</span>
						{#if show.contentRating}
							<span class="text-text-muted">•</span>
							<span class="px-1.5 py-0.5 border border-white/20 rounded text-xs">{show.contentRating}</span>
						{/if}
						{#if show.status}
							<span class="text-text-muted">•</span>
							<span class={getStatusColor(show.status)}>{show.status}</span>
						{/if}
					</div>

					<!-- Genres inline -->
					{#if parseGenres(show.genres).length > 0}
						<div class="text-sm text-text-secondary mt-1">
							{parseGenres(show.genres).join(', ')}
						</div>
					{/if}

					<!-- Crew info -->
					<div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 mt-3 text-sm">
						{#if show.network}
							<span class="text-text-muted uppercase text-xs tracking-wide">Network</span>
							<span class="text-text-primary">{show.network}</span>
						{/if}
					</div>

					<!-- Media selectors -->
					{#if mediaInfo && (mediaInfo.audioStreams?.length || mediaInfo.subtitleTracks?.length)}
						<div class="mt-3">
							<MediaSelector
								audioStreams={mediaInfo.audioStreams}
								subtitleTracks={mediaInfo.subtitleTracks}
							/>
						</div>
					{/if}

					<!-- Overview -->
					{#if show.overview}
						<p class="text-text-secondary text-sm leading-relaxed mt-3 line-clamp-2">{show.overview}</p>
					{/if}

					<!-- Action buttons -->
					<div class="flex flex-wrap items-center gap-3 mt-4">
						<button
							onclick={handlePlayNext}
							class="px-5 py-2 rounded-lg bg-white text-black font-medium flex items-center gap-2 hover:bg-white/90 transition-colors"
						>
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M8 5v14l11-7z" />
							</svg>
							{#if nextEpisode()}
								Play S{nextEpisode()?.season} E{nextEpisode()?.episode}
							{:else}
								Play S1 E1
							{/if}
						</button>

						<button
							class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-text-secondary hover:text-white transition-colors"
							title="Add to List"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
							</svg>
						</button>

						<button
							class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-text-secondary hover:text-white transition-colors"
							title="Watch Trailer"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
							</svg>
						</button>

						{#if user?.role === 'admin' && show.tmdbId && !addedToWanted}
							<button
								onclick={handleAddToWanted}
								disabled={addingToWanted}
								class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-text-secondary hover:text-white transition-colors disabled:opacity-50"
								title="Monitor for new episodes"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
								</svg>
							</button>
						{/if}
					</div>

					<!-- External links -->
					<div class="flex items-center gap-2 mt-3">
						{#if show.tmdbId}
							<a
								href="https://www.themoviedb.org/tv/{show.tmdbId}"
								target="_blank"
								rel="noopener noreferrer"
								class="w-8 h-8 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
								title="View on TMDB"
							>
								<img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6" />
							</a>
						{/if}
						<a
							href="https://trakt.tv/search/tmdb/{show.tmdbId}?id_type=show"
							target="_blank"
							rel="noopener noreferrer"
							class="w-8 h-8 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
							title="View on Trakt"
						>
							<img src="/icons/trakt.svg" alt="Trakt" class="w-6 h-6" />
						</a>
						{#if show.imdbId}
							<a
								href="https://www.imdb.com/title/{show.imdbId}"
								target="_blank"
								rel="noopener noreferrer"
								class="w-8 h-8 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
								title="View on IMDb"
							>
								<img src="/icons/imdb.svg" alt="IMDb" class="w-6 h-6" />
							</a>
						{/if}
					</div>
				</div>
			</div>

			<!-- Episodes section -->
			<div class="flex-shrink-0 px-6 py-3 border-t border-white/10">
				<div class="flex items-center justify-between mb-3">
					<div class="flex items-center gap-4">
						<h3 class="text-xs font-medium text-text-muted uppercase tracking-wide">Episodes</h3>
						{#if show.seasons && show.seasons.length > 0}
							<Dropdown
								options={show.seasons.map((season, i) => ({ value: i, label: `Season ${season.seasonNumber}` }))}
								value={selectedSeasonIndex}
								onchange={(v) => selectedSeasonIndex = v as number}
							/>
						{/if}
						<span class="text-xs text-text-muted">
							{seasonWatchedCount()} of {selectedSeason?.episodes?.length || 0} watched
						</span>
					</div>
					<button
						onclick={handleMarkAllWatched}
						disabled={markingAllWatched}
						class="px-3 py-1 text-xs rounded bg-white/5 hover:bg-white/10 text-text-secondary hover:text-white transition-colors disabled:opacity-50"
					>
						{markingAllWatched ? 'Updating...' : 'Mark All'}
					</button>
				</div>

				<!-- Episode grid - scrollable if needed -->
				<div class="max-h-36 overflow-y-auto">
					{#if selectedSeason?.episodes}
						<EpisodeGrid
							episodes={selectedSeason.episodes}
							{watchedEpisodes}
							onToggleWatched={handleToggleEpisodeWatched}
							{togglingEpisode}
						/>
					{:else}
						<p class="text-text-muted text-sm">No episodes found.</p>
					{/if}
				</div>
			</div>

			<!-- Cast row -->
			{#if parseCast(show.cast).length > 0}
				<div class="flex-shrink-0 px-6 pb-4 border-t border-white/10 pt-3">
					<CastRow cast={parseCast(show.cast).slice(0, 10)} />
				</div>
			{/if}
		</div>
	{/if}
</div>
