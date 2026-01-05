<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getShow, refreshShowMetadata, getImageUrl,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getShowSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		deleteEpisode,
		type ShowDetail, type QualityProfile, type TMDBShowResult
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import { getOfficialTrailer, parseGenres, parseCast, parseCrew } from '$lib/utils';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import MediaDetail from '$lib/components/MediaDetail.svelte';

	let show: ShowDetail | null = $state(null);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let profiles: QualityProfile[] = $state([]);
	let user = $state<{ role: string } | null>(null);
	let watchedEpisodes: Set<number> = $state(new Set());
	let togglingEpisode: number | null = $state(null);
	let showManageMenu = $state(false);
	let inWatchlist = $state(false);
	let watchlistLoading = $state(false);
	let showTrailerModal = $state(false);
	let selectedSeasonIndex = $state(0);
	let recommendations: TMDBShowResult[] = $state([]);

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
			// Check watchlist status using TMDB ID
			if (show?.tmdbId) {
				inWatchlist = await isInWatchlist(show.tmdbId, 'tv').catch(() => false);
			}
			// Load suggestions
			if (show) {
				try {
					const suggestResult = await getShowSuggestions(show.id);
					recommendations = suggestResult.results.slice(0, 20);
				} catch { /* Suggestions are optional */ }
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load show';
		} finally {
			loading = false;
		}
	});

	async function handleRefresh() {
		if (!show) return;
		showManageMenu = false;
		refreshing = true;
		try {
			show = await refreshShowMetadata(show.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh';
		} finally {
			refreshing = false;
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

	async function handleToggleEpisodeWatched(episodeId: number, runtime?: number, e?: Event) {
		e?.stopPropagation();
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

	let deletingEpisode: number | null = $state(null);
	let confirmingDeleteEpisodeId: number | null = $state(null);

	function handleDeleteEpisodeClick(episodeId: number, e: Event) {
		e.stopPropagation();
		confirmingDeleteEpisodeId = episodeId;
	}

	function cancelDeleteEpisode(e: Event) {
		e.stopPropagation();
		confirmingDeleteEpisodeId = null;
	}

	async function confirmDeleteEpisode(episodeId: number, e: Event) {
		e.stopPropagation();
		deletingEpisode = episodeId;
		confirmingDeleteEpisodeId = null;
		try {
			await deleteEpisode(episodeId);
			// Refresh show data to update the episode list
			if (show) {
				show = await getShow(show.id);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete episode';
		} finally {
			deletingEpisode = null;
		}
	}

	async function handleToggleWatchlist() {
		if (!show?.tmdbId) return;
		watchlistLoading = true;
		try {
			if (inWatchlist) {
				await removeFromWatchlist(show.tmdbId, 'tv');
				inWatchlist = false;
			} else {
				await addToWatchlist(show.tmdbId, 'tv');
				inWatchlist = true;
			}
		} catch (e) {
			console.error('Failed to update watchlist:', e);
		} finally {
			watchlistLoading = false;
		}
	}

	const selectedSeason = $derived(show?.seasons?.[selectedSeasonIndex]);

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

	function getTotalRuntime(): string {
		if (!show?.seasons) return '-';
		let totalMinutes = 0;
		for (const season of show.seasons) {
			if (!season?.episodes) continue;
			for (const ep of season.episodes) {
				totalMinutes += ep.runtime || 0;
			}
		}
		if (totalMinutes === 0) return '-';
		const hours = Math.floor(totalMinutes / 60);
		const mins = totalMinutes % 60;
		if (hours === 0) return `${mins}m`;
		return `${hours}h ${mins}m`;
	}

	function getCreators(): string[] {
		const crew = parseCrew(show?.crew);
		return crew
			.filter(c => c.job === 'Creator' || c.job === 'Executive Producer' || c.job === 'Showrunner')
			.map(c => c.name)
			.slice(0, 3);
	}

	function getNextAirDate(): string | null {
		if (!show?.seasons || show.status?.toLowerCase() !== 'returning series') return null;
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		for (const season of show.seasons) {
			if (!season?.episodes) continue;
			for (const ep of season.episodes) {
				if (ep.airDate) {
					const airDate = new Date(ep.airDate);
					if (airDate >= today) {
						return ep.airDate;
					}
				}
			}
		}
		return null;
	}

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString('en-US', {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}

	// Get tags from genres
	const tags = $derived(parseGenres(show?.genres));

	// Transform cast/crew for MediaDetail
	const castList = $derived(parseCast(show?.cast).map(c => ({
		id: c.id,
		name: c.name,
		character: c.character,
		profile_path: c.profile_path
	})));

	const crewList = $derived(parseCrew(show?.crew).map(c => ({
		id: c.id,
		name: c.name,
		job: c.job,
		profile_path: c.profile_path
	})));

	// Transform recommendations for MediaDetail
	const recsList = $derived(recommendations.map(r => ({
		id: r.id,
		title: r.name,
		poster_path: r.poster_path,
		release_date: r.first_air_date,
		vote_average: r.vote_average
	})));
</script>

<svelte:head>
	<title>{show?.title || 'TV Show'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-96">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Loading show...</p>
		</div>
	</div>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-lg">
		{error}
		<button class="ml-2 underline" onclick={() => error = null}>Dismiss</button>
	</div>
{:else if show}
	<MediaDetail
		title={show.title}
		year={show.year}
		overview={show.overview}
		tagline={show.tagline}
		posterPath={show.posterPath}
		backdropPath={show.backdropPath}
		genres={tags}
		tmdbId={show.tmdbId}
		imdbId={show.imdbId}
		mediaType="tv"
		source="library"
		seasons={show.seasons?.length || 0}
		episodes={totalEpisodes()}
		networks={show.network ? [show.network] : []}
		status={show.status}
		contentRating={show.contentRating}
		originalLanguage={show.originalLanguage}
		country={show.country}
		rating={show.rating}
		cast={castList}
		crew={crewList}
		recommendations={recsList}
		addedAt={show.addedAt}
		useLocalImages={true}
		posterClickable={true}
		onPosterClick={handlePlayNext}
		trailersJson={show.trailers}
	>
		{#snippet actionButtons()}
			<IconButton
				onclick={handleToggleWatchlist}
				disabled={watchlistLoading}
				active={inWatchlist}
				title="{inWatchlist ? 'Remove from' : 'Add to'} Watchlist"
			>
				{#if inWatchlist}
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{:else}
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
				{/if}
			</IconButton>

			<IconButton
				onclick={handlePlayNext}
				variant="yellow"
				title="Play {nextEpisode() ? `S${nextEpisode()?.season} E${nextEpisode()?.episode}` : 'S1 E1'}"
			>
				<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M8 5v14l11-7z" />
				</svg>
			</IconButton>

			{#if getOfficialTrailer(show?.trailers)}
				<IconButton
					onclick={() => showTrailerModal = true}
					title="Watch Trailer"
				>
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
						<path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
					</svg>
				</IconButton>
			{/if}

			<!-- Manage dropdown -->
			<div class="relative">
				<IconButton
					onclick={() => showManageMenu = !showManageMenu}
					title="More options"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
					</svg>
				</IconButton>
				{#if showManageMenu}
					<button
						type="button"
						class="fixed inset-0 z-[55] cursor-default"
						onclick={() => showManageMenu = false}
						aria-label="Close menu"
					></button>
					<div class="absolute left-1/2 -translate-x-1/2 mt-2 w-48 py-1 z-[60] bg-[#141416] border border-white/10 rounded-2xl shadow-xl overflow-hidden">
						<button
							onclick={() => { handleRefresh(); showManageMenu = false; }}
							class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors"
						>
							{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
						</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
						<div class="border-t border-white/10 my-1"></div>
						<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={() => showManageMenu = false}>Delete</button>
					</div>
				{/if}
			</div>
		{/snippet}

		{#snippet extraInfoRows()}
			<!-- Created By -->
			{#if getCreators().length > 0}
				<div class="flex justify-between">
					<span class="text-text-muted">Created By</span>
					<span class="text-right max-w-[180px] truncate" title={getCreators().join(', ')}>{getCreators().join(', ')}</span>
				</div>
			{/if}

			<!-- Next Air Date -->
			{#if getNextAirDate()}
				<div class="flex justify-between">
					<span class="text-text-muted">Next Episode</span>
					<span class="text-green-400">{formatDate(getNextAirDate() || '')}</span>
				</div>
			{/if}

			<!-- Total Runtime -->
			<div class="flex justify-between">
				<span class="text-text-muted">Total Runtime</span>
				<span>{getTotalRuntime()}</span>
			</div>

			<!-- Added date -->
			{#if show.addedAt}
				<div class="flex justify-between">
					<span class="text-text-muted">Added</span>
					<span class="text-xs">{new Date(show.addedAt).toLocaleDateString()}</span>
				</div>
			{/if}
		{/snippet}

		{#snippet extraSections()}
			<!-- Episodes Section -->
			<section class="px-[60px]">
				<h2 class="text-lg font-semibold text-text-primary mb-4">Episodes</h2>

				<!-- Season Pills -->
				{#if show.seasons && show.seasons.length > 1}
					<div class="flex gap-2 mb-4 overflow-x-auto pb-2 scrollbar-thin">
						{#each show.seasons as season, i}
							<button
								onclick={() => selectedSeasonIndex = i}
								class="flex-shrink-0 px-4 py-2 rounded-full text-sm font-medium transition-all {selectedSeasonIndex === i
									? 'bg-[#f5f5dc] text-black'
									: 'bg-glass border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
							>
								Season {season.seasonNumber}
							</button>
						{/each}
					</div>
				{/if}

				{#if selectedSeason?.episodes && selectedSeason.episodes.length > 0}
					<div class="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
						{#each selectedSeason.episodes as episode}
							<div
								class="group flex-shrink-0 w-64 rounded-xl overflow-hidden bg-bg-elevated cursor-pointer transition-all duration-300 hover:translate-y-[-6px] hover:shadow-lg"
							>
								<!-- Image container -->
								<button
									onclick={() => goto(`/watch/episode/${episode.id}`)}
									class="relative aspect-video bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44] w-full"
								>
									{#if episode.stillPath}
										<img
											src={getImageUrl(episode.stillPath)}
											alt={episode.title}
											class="w-full h-full object-cover"
										/>
									{:else if show.backdropPath}
										<img
											src={getImageUrl(show.backdropPath)}
											alt={episode.title}
											class="w-full h-full object-cover opacity-50"
										/>
									{/if}

									<!-- Watched badge -->
									{#if watchedEpisodes.has(episode.id)}
										<div class="absolute top-2 right-2 px-2 py-1 rounded text-[10px] font-bold uppercase bg-green-500 text-black">
											Watched
										</div>
									{/if}

									<!-- Play overlay on hover -->
									<div class="absolute inset-0 bg-black/40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
										<div class="w-12 h-12 rounded-full bg-glass border border-border-subtle backdrop-blur-xl flex items-center justify-center">
											<svg class="w-6 h-6 text-text-primary ml-0.5" fill="currentColor" viewBox="0 0 24 24">
												<path d="M8 5v14l11-7z" />
											</svg>
										</div>
									</div>
								</button>

								<!-- Info section -->
								<div class="p-3">
									<div class="flex items-center gap-2 mb-1">
										<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
											S{selectedSeason.seasonNumber} E{episode.episodeNumber}
										</span>
										{#if episode.runtime}
											<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
												{episode.runtime}m
											</span>
										{/if}
										{#if show.contentRating}
											<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
												{show.contentRating}
											</span>
										{/if}
									</div>
									<div class="flex items-center justify-between gap-2">
										<h3 class="text-sm font-semibold text-text-primary truncate flex-1">
											{episode.title || `Episode ${episode.episodeNumber}`}
										</h3>
										<!-- Episode actions -->
										<div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
											<button
												onclick={(e) => handleToggleEpisodeWatched(episode.id, episode.runtime, e)}
												disabled={togglingEpisode === episode.id}
												class="p-1.5 rounded-full transition-colors {watchedEpisodes.has(episode.id) ? 'text-green-400 hover:bg-green-500/20' : 'text-text-muted hover:bg-white/10 hover:text-white'}"
												title={watchedEpisodes.has(episode.id) ? 'Mark as unwatched' : 'Mark as watched'}
											>
												{#if togglingEpisode === episode.id}
													<div class="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
												{:else if watchedEpisodes.has(episode.id)}
													<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
													</svg>
												{:else}
													<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
													</svg>
												{/if}
											</button>
											{#if user?.role === 'admin'}
												{#if confirmingDeleteEpisodeId === episode.id}
													<!-- Confirm button (checkmark) -->
													<button
														onclick={(e) => confirmDeleteEpisode(episode.id, e)}
														disabled={deletingEpisode === episode.id}
														class="p-1.5 rounded-full text-green-400 hover:bg-green-500/20 transition-colors"
														title="Confirm delete"
													>
														{#if deletingEpisode === episode.id}
															<div class="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
														{:else}
															<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
															</svg>
														{/if}
													</button>
													<!-- Cancel button (X) -->
													<button
														onclick={(e) => cancelDeleteEpisode(e)}
														class="p-1.5 rounded-full text-text-muted hover:bg-white/10 hover:text-white transition-colors"
														title="Cancel"
													>
														<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
														</svg>
													</button>
												{:else}
													<button
														onclick={(e) => handleDeleteEpisodeClick(episode.id, e)}
														disabled={deletingEpisode === episode.id}
														class="p-1.5 rounded-full text-text-muted hover:bg-red-500/20 hover:text-red-400 transition-colors"
														title="Delete episode"
													>
														<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
														</svg>
													</button>
												{/if}
											{/if}
										</div>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="text-center py-12 text-text-muted">
						<p>No episodes found for this season.</p>
					</div>
				{/if}
			</section>
		{/snippet}
	</MediaDetail>

	<!-- Trailer Modal -->
	<TrailerModal
		bind:open={showTrailerModal}
		trailersJson={show?.trailers}
		title={show?.title}
	/>
{/if}
