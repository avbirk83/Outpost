<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getMovie, refreshMovieMetadata, getImageUrl,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getMediaInfo, getMovieSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		type Movie, type QualityProfile, type MediaInfo, type TMDBMovieResult
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import {
		formatRuntime, getOfficialTrailer, parseGenres, parseCast, parseCrew,
		formatResolution, formatAudioChannels, getLanguageName
	} from '$lib/utils';
	import Dropdown from '$lib/components/Dropdown.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import MediaDetail from '$lib/components/MediaDetail.svelte';

	let movie: Movie | null = $state(null);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let profiles: QualityProfile[] = $state([]);
	let user = $state<{ role: string } | null>(null);
	let isWatched = $state(false);
	let togglingWatched = $state(false);
	let mediaInfo: MediaInfo | null = $state(null);
	let selectedSubtitle = $state<number | null>(null);
	let selectedAudio = $state<number>(0);
	let showManageMenu = $state(false);
	let inWatchlist = $state(false);
	let watchlistLoading = $state(false);
	let showTrailerModal = $state(false);
	let selectedVideo = $state(0);
	let recommendations: TMDBMovieResult[] = $state([]);

	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			movie = await getMovie(id);
			if (user?.role === 'admin') {
				profiles = await getQualityProfiles();
			}
			const watchStatus = await getWatchStatus('movie', id);
			isWatched = watchStatus.watched;
			mediaInfo = await getMediaInfo('movie', id);
			const defaultSub = mediaInfo.subtitleTracks?.find(t => t.default);
			if (defaultSub) selectedSubtitle = defaultSub.index;
			// Check watchlist status using TMDB ID
			if (movie?.tmdbId) {
				inWatchlist = await isInWatchlist(movie.tmdbId, 'movie').catch(() => false);
			}
			// Load suggestions based on genres, excluding library items
			if (movie) {
				try {
					const suggestResult = await getMovieSuggestions(movie.id);
					recommendations = suggestResult.results.slice(0, 20);
				} catch { /* Suggestions are optional */ }
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load movie';
		} finally {
			loading = false;
		}
	});

	async function handleRefresh() {
		if (!movie) return;
		showManageMenu = false;
		refreshing = true;
		try {
			movie = await refreshMovieMetadata(movie.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh';
		} finally {
			refreshing = false;
		}
	}

	function handlePlay() {
		if (movie) {
			const params = new URLSearchParams();
			if (selectedSubtitle !== null) params.set('sub', selectedSubtitle.toString());
			goto(`/watch/movie/${movie.id}${params.toString() ? '?' + params.toString() : ''}`);
		}
	}

	async function handleToggleWatched() {
		if (!movie) return;
		togglingWatched = true;
		try {
			if (isWatched) {
				await markAsUnwatched('movie', movie.id);
				isWatched = false;
			} else {
				await markAsWatched('movie', movie.id, movie.runtime ? movie.runtime * 60 : 3600);
				isWatched = true;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update';
		} finally {
			togglingWatched = false;
		}
	}

	async function handleToggleWatchlist() {
		if (!movie?.tmdbId) return;
		watchlistLoading = true;
		try {
			if (inWatchlist) {
				await removeFromWatchlist(movie.tmdbId, 'movie');
				inWatchlist = false;
			} else {
				await addToWatchlist(movie.tmdbId, 'movie');
				inWatchlist = true;
			}
		} catch (e) {
			console.error('Failed to update watchlist:', e);
		} finally {
			watchlistLoading = false;
		}
	}

	// Get tags from genres
	const tags = $derived(parseGenres(movie?.genres));

	// Transform cast/crew for MediaDetail
	const castList = $derived(parseCast(movie?.cast).map(c => ({
		id: c.id,
		name: c.name,
		character: c.character,
		profile_path: c.profile_path
	})));

	const crewList = $derived(parseCrew(movie?.crew).map(c => ({
		id: c.id,
		name: c.name,
		job: c.job,
		profile_path: c.profile_path
	})));

	// Transform recommendations for MediaDetail
	const recsList = $derived(recommendations.map(r => ({
		id: r.id,
		title: r.title,
		poster_path: r.poster_path,
		release_date: r.release_date,
		vote_average: r.vote_average
	})));
</script>

<svelte:head>
	<title>{movie?.title || 'Movie'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-96">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Loading movie...</p>
		</div>
	</div>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-lg">
		{error}
		<button class="ml-2 underline" onclick={() => error = null}>Dismiss</button>
	</div>
{:else if movie}
	<MediaDetail
		title={movie.title}
		year={movie.year}
		overview={movie.overview}
		tagline={movie.tagline}
		posterPath={movie.posterPath}
		backdropPath={movie.backdropPath}
		genres={tags}
		tmdbId={movie.tmdbId}
		imdbId={movie.imdbId}
		mediaType="movie"
		source="library"
		runtime={movie.runtime}
		budget={movie.budget}
		revenue={movie.revenue}
		theatricalRelease={movie.theatricalRelease}
		digitalRelease={movie.digitalRelease}
		contentRating={movie.contentRating}
		status={movie.status}
		originalLanguage={movie.originalLanguage}
		country={movie.country}
		studios={movie.studios}
		rating={movie.rating}
		cast={castList}
		crew={crewList}
		recommendations={recsList}
		addedAt={movie.addedAt}
		lastWatchedAt={movie.lastWatchedAt}
		playCount={movie.playCount}
		useLocalImages={true}
		posterClickable={true}
		onPosterClick={handlePlay}
		trailersJson={movie.trailers}
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
				onclick={handleToggleWatched}
				disabled={togglingWatched}
				active={isWatched}
				title="{isWatched ? 'Mark as unwatched' : 'Mark as watched'}"
			>
				<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
				</svg>
			</IconButton>

			<IconButton
				onclick={handlePlay}
				variant="yellow"
				title="Play"
			>
				<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M8 5v14l11-7z" />
				</svg>
			</IconButton>

			{#if getOfficialTrailer(movie?.trailers)}
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

		{#snippet centerExtra()}
			{#if mediaInfo}
				<div class="flex items-center gap-3">
					{#if mediaInfo.videoStreams?.length}
						<div class="w-[120px]">
							<Dropdown
								icon="video"
								options={mediaInfo.videoStreams.map((v, i) => ({ value: i, label: `${formatResolution(v.width, v.height)} ${v.codec?.toUpperCase() || ''}` }))}
								value={selectedVideo}
								onchange={(v) => selectedVideo = v as number}
								inline={true}
							/>
						</div>
					{/if}
					{#if mediaInfo.videoStreams?.length && mediaInfo.audioStreams?.length}
						<span class="text-white/20">|</span>
					{/if}
					{#if mediaInfo.audioStreams?.length}
						<div class="w-[140px]">
							<Dropdown
								icon="audio"
								options={mediaInfo.audioStreams.map((a, i) => ({ value: i, label: `${a.language?.toUpperCase() || 'UNK'} ${a.codec?.toUpperCase() || ''} ${formatAudioChannels(a.channels)}` }))}
								value={selectedAudio}
								onchange={(v) => selectedAudio = v as number}
								inline={true}
							/>
						</div>
					{/if}
					{#if (mediaInfo.videoStreams?.length || mediaInfo.audioStreams?.length)}
						<span class="text-white/20">|</span>
					{/if}
					<div class="w-[110px]">
						<Dropdown
							icon="subtitles"
							options={[{ value: null, label: 'Off' }, ...(mediaInfo.subtitleTracks || []).map(s => ({ value: s.index, label: s.title || getLanguageName(s.language) }))]}
							value={selectedSubtitle}
							onchange={(v) => selectedSubtitle = v as number | null}
							inline={true}
						/>
					</div>
				</div>
			{/if}
		{/snippet}

		{#snippet extraInfoRows()}
			{#if movie.addedAt}
				<div class="flex justify-between">
					<span class="text-text-muted">Added</span>
					<span>{new Date(movie.addedAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
				</div>
			{/if}
			{#if movie.lastWatchedAt}
				<div class="flex justify-between">
					<span class="text-text-muted">Last Watched</span>
					<span>{new Date(movie.lastWatchedAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
				</div>
			{/if}
			{#if movie.playCount > 0}
				<div class="flex justify-between">
					<span class="text-text-muted">Play Count</span>
					<span>{movie.playCount}</span>
				</div>
			{/if}
		{/snippet}

		{#snippet extraSections()}
			{#if mediaInfo}
				<section class="px-[60px]">
					<h2 class="text-lg font-semibold text-text-primary mb-3">Files</h2>
					<div class="flex gap-4">
						<button onclick={handlePlay} class="group relative w-72 md:w-80">
							<div class="relative aspect-video bg-bg-card overflow-hidden rounded-xl">
								{#if movie.backdropPath}
									<img
										src={getImageUrl(movie.backdropPath)}
										alt={movie.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else if movie.posterPath}
									<img
										src={getImageUrl(movie.posterPath)}
										alt={movie.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
										<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
									</div>
								{/if}

								<div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>

								<div class="absolute inset-0 flex items-center justify-center">
									<div class="w-14 h-14 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transform scale-75 group-hover:scale-100 transition-all duration-300 border border-white/30">
										<svg class="w-7 h-7 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
											<path d="M8 5v14l11-7z" />
										</svg>
									</div>
								</div>

								<div class="absolute bottom-0 left-0 right-0 p-4">
									<h3 class="text-base font-semibold text-white truncate">{movie.title}</h3>
									<div class="flex flex-wrap gap-1.5 mt-2">
										{#if mediaInfo.videoStreams?.[0]}
											<span class="px-2 py-0.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
												{formatResolution(mediaInfo.videoStreams[0].width, mediaInfo.videoStreams[0].height)}
											</span>
										{/if}
										{#if mediaInfo.videoStreams?.[0]?.codec}
											<span class="px-2 py-0.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
												{mediaInfo.videoStreams[0].codec.toUpperCase()}
											</span>
										{/if}
										{#if mediaInfo.audioStreams?.[0]?.codec}
											<span class="px-2 py-0.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
												{mediaInfo.audioStreams[0].codec.toUpperCase()} {formatAudioChannels(mediaInfo.audioStreams[0].channels)}
											</span>
										{/if}
										{#if mediaInfo.container}
											<span class="px-2 py-0.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
												{mediaInfo.container.toUpperCase()}
											</span>
										{/if}
									</div>
								</div>
							</div>
						</button>
					</div>
				</section>
			{/if}
		{/snippet}
	</MediaDetail>

	<!-- Trailer Modal -->
	<TrailerModal
		bind:open={showTrailerModal}
		trailersJson={movie?.trailers}
		title={movie?.title}
	/>
{/if}
