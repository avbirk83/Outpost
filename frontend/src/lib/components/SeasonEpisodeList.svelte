<script lang="ts">
	import { getSeasonDetails, type SeasonSummary, type SeasonDetails, type EpisodeInfo } from '$lib/api/discover';
	import EpisodeCard from './EpisodeCard.svelte';

	// For library pages - episodes are already loaded
	interface LibrarySeason {
		seasonNumber: number;
		episodes: Array<{
			id: number;
			episodeNumber: number;
			title?: string;
			overview?: string | null;
			airDate?: string | null;
			runtime?: number | null;
			stillPath?: string | null;
		}>;
	}

	interface Props {
		// For explore - we fetch from TMDB
		tmdbId?: number;
		seasonSummaries?: SeasonSummary[];
		// For library - episodes are pre-loaded
		librarySeasons?: LibrarySeason[];
		// Common
		showBackdrop?: string | null;
		isLibrary?: boolean;
		// Library callbacks
		watchedEpisodes?: Set<number>;
		onPlay?: (episodeId: number) => void;
		onToggleWatched?: (episodeId: number) => void;
		onSubtitleSearch?: (episode: { id: number; title: string; seasonNumber: number; episodeNumber: number }) => void;
		onDelete?: (episodeId: number) => void;
		isAdmin?: boolean;
		togglingEpisodeId?: number | null;
		// Season monitoring (library only)
		monitoredSeasons?: Set<number>;
		onToggleSeasonMonitor?: (seasonNumber: number) => void;
		togglingSeasonMonitor?: number | null;
	}

	let {
		tmdbId,
		seasonSummaries = [],
		librarySeasons = [],
		showBackdrop,
		isLibrary = false,
		watchedEpisodes = new Set(),
		onPlay,
		onToggleWatched,
		onSubtitleSearch,
		onDelete,
		isAdmin = false,
		togglingEpisodeId = null,
		monitoredSeasons = new Set(),
		onToggleSeasonMonitor,
		togglingSeasonMonitor = null
	}: Props = $props();

	let selectedSeasonIndex = $state(0);
	let loadingSeasons = $state<Set<number>>(new Set());
	let seasonCache = $state<Map<number, SeasonDetails>>(new Map());
	// Track which episode is expanded (by episode number within current season)
	let expandedEpisodeNumber = $state<number | null>(null);

	// Get available seasons
	const seasons = $derived(
		isLibrary
			? librarySeasons.map(s => ({ seasonNumber: s.seasonNumber, episodeCount: s.episodes.length }))
			: seasonSummaries.filter(s => s.season_number > 0).map(s => ({ seasonNumber: s.season_number, episodeCount: s.episode_count }))
	);

	const selectedSeasonNumber = $derived(seasons[selectedSeasonIndex]?.seasonNumber || 1);

	// Extended episode info that includes local still path
	interface ExtendedEpisodeInfo extends EpisodeInfo {
		localStillPath?: string | null;
	}

	// Get episodes for selected season
	const selectedEpisodes = $derived.by((): ExtendedEpisodeInfo[] => {
		if (isLibrary) {
			const libSeason = librarySeasons.find(s => s.seasonNumber === selectedSeasonNumber);
			if (!libSeason) return [];

			// Check if we have TMDB data cached for this season
			const tmdbSeason = seasonCache.get(selectedSeasonNumber);

			// Convert library episodes to EpisodeInfo format, merging with TMDB data if available
			return libSeason.episodes.map(ep => {
				// Find matching TMDB episode by episode number
				const tmdbEp = tmdbSeason?.episodes?.find(te => te.episode_number === ep.episodeNumber);

				return {
					id: ep.id, // Keep library ID for actions
					episode_number: ep.episodeNumber,
					name: ep.title || tmdbEp?.name || `Episode ${ep.episodeNumber}`,
					overview: ep.overview || tmdbEp?.overview || '',
					air_date: ep.airDate || tmdbEp?.air_date || '',
					runtime: ep.runtime || tmdbEp?.runtime || 0,
					// Keep local and TMDB paths separate
					localStillPath: ep.stillPath || null,
					still_path: tmdbEp?.still_path || null,
					vote_average: tmdbEp?.vote_average || 0,
					guest_stars: tmdbEp?.guest_stars || [],
					crew: tmdbEp?.crew || []
				} as ExtendedEpisodeInfo;
			});
		} else {
			const cached = seasonCache.get(selectedSeasonNumber);
			return cached?.episodes || [];
		}
	});

	// Fetch season details when switching seasons
	// For explore: always fetch
	// For library: fetch if we have a tmdbId (to get episode details like overview, guest stars)
	$effect(() => {
		if (tmdbId && selectedSeasonNumber && !seasonCache.has(selectedSeasonNumber) && !loadingSeasons.has(selectedSeasonNumber)) {
			fetchSeasonDetails(selectedSeasonNumber);
		}
	});

	async function fetchSeasonDetails(seasonNum: number) {
		if (!tmdbId) return;

		loadingSeasons = new Set([...loadingSeasons, seasonNum]);
		try {
			const details = await getSeasonDetails(tmdbId, seasonNum);
			seasonCache = new Map(seasonCache).set(seasonNum, details);
		} catch (e) {
			console.error(`Failed to fetch season ${seasonNum}:`, e);
		} finally {
			const newLoading = new Set(loadingSeasons);
			newLoading.delete(seasonNum);
			loadingSeasons = newLoading;
		}
	}

	function selectSeason(index: number) {
		selectedSeasonIndex = index;
		// Reset expanded episode when changing seasons - will auto-expand next to watch
		expandedEpisodeNumber = null;
	}

	// For library, we show episodes immediately (from local data) while TMDB data loads in background
	// For explore, we wait for TMDB data
	const isLoading = $derived(!isLibrary && loadingSeasons.has(selectedSeasonNumber));

	// Check if we have TMDB data loaded for the current season
	const hasTmdbData = $derived(seasonCache.has(selectedSeasonNumber));

	// Calculate the "next episode to watch" for library mode
	// This is the first unwatched episode in order, or the first episode if all watched/new series
	const nextToWatchEpisodeNumber = $derived.by(() => {
		if (!isLibrary || selectedEpisodes.length === 0) return null;

		// Find first unwatched episode
		const firstUnwatched = selectedEpisodes.find(ep => !watchedEpisodes.has(ep.id));
		if (firstUnwatched) {
			return firstUnwatched.episode_number;
		}

		// All watched - return first episode
		return selectedEpisodes[0]?.episode_number || null;
	});

	// Auto-expand the next episode to watch when season changes or on initial load
	// Only auto-expand when we have TMDB data (so expanded content isn't blank)
	$effect(() => {
		// Only auto-expand if nothing is currently expanded AND we have TMDB data loaded
		if (expandedEpisodeNumber === null && nextToWatchEpisodeNumber !== null && hasTmdbData) {
			expandedEpisodeNumber = nextToWatchEpisodeNumber;
		}
	});

	function toggleEpisodeExpand(episodeNumber: number) {
		if (expandedEpisodeNumber === episodeNumber) {
			expandedEpisodeNumber = null;
		} else {
			expandedEpisodeNumber = episodeNumber;
		}
	}
</script>

{#if seasons.length > 0}
	<div class="space-y-4">
		<!-- Season selector pills -->
		<div class="flex gap-2 overflow-x-auto pb-2 scrollbar-thin">
			{#each seasons as season, i}
				<div class="flex-shrink-0 flex items-center gap-1">
					<button
						onclick={() => selectSeason(i)}
						class="px-4 py-2 text-sm font-medium transition-all {isLibrary && onToggleSeasonMonitor ? 'rounded-l-full' : 'rounded-full'} {selectedSeasonIndex === i
							? 'bg-cream text-black'
							: 'bg-glass border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'} {isLibrary && onToggleSeasonMonitor ? 'border-r-0' : ''}"
					>
						Season {season.seasonNumber}
						<span class="ml-1 text-xs opacity-70">({season.episodeCount})</span>
					</button>

					{#if isLibrary && onToggleSeasonMonitor}
						<button
							onclick={() => onToggleSeasonMonitor?.(season.seasonNumber)}
							disabled={togglingSeasonMonitor === season.seasonNumber}
							class="px-2 py-2 rounded-r-full text-sm transition-all border border-l-0 {selectedSeasonIndex === i
								? 'bg-cream text-black border-cream'
								: 'bg-glass border-border-subtle text-text-secondary hover:bg-glass-hover'} {monitoredSeasons.has(season.seasonNumber) ? '' : 'opacity-50'}"
							title={monitoredSeasons.has(season.seasonNumber) ? 'Monitored (click to disable)' : 'Not monitored (click to enable)'}
						>
							{#if togglingSeasonMonitor === season.seasonNumber}
								<div class="spinner-sm"></div>
							{:else if monitoredSeasons.has(season.seasonNumber)}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{:else}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
								</svg>
							{/if}
						</button>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Episodes -->
		{#if isLoading}
			<div class="flex items-center justify-center py-12">
				<div class="spinner-lg text-text-muted"></div>
			</div>
		{:else if selectedEpisodes.length > 0}
			<div class="flex items-start gap-3 overflow-x-auto pb-2 scrollbar-thin">
				{#each selectedEpisodes as episode (episode.id || episode.episode_number)}
					<EpisodeCard
						{episode}
						seasonNumber={selectedSeasonNumber}
						{showBackdrop}
						localStillPath={isLibrary ? episode.localStillPath : null}
						{isLibrary}
						isWatched={watchedEpisodes.has(episode.id)}
						episodeId={episode.id}
						onPlay={() => onPlay?.(episode.id)}
						onToggleWatched={() => onToggleWatched?.(episode.id)}
						onSubtitleSearch={() => onSubtitleSearch?.({
							id: episode.id,
							title: episode.name,
							seasonNumber: selectedSeasonNumber,
							episodeNumber: episode.episode_number
						})}
						onDelete={() => onDelete?.(episode.id)}
						{isAdmin}
						togglingWatched={togglingEpisodeId === episode.id}
						isExpanded={expandedEpisodeNumber === episode.episode_number}
						onToggleExpand={() => toggleEpisodeExpand(episode.episode_number)}
					/>
				{/each}
			</div>
		{:else}
			<div class="text-center py-8 text-text-muted">
				No episodes available for this season
			</div>
		{/if}
	</div>
{:else}
	<div class="text-center py-8 text-text-muted">
		No seasons available
	</div>
{/if}
