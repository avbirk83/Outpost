<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getMovie, refreshMovieMetadata, deleteMovie, getImageUrl,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getMediaInfo, getMovieSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		getMovieQuality, setMovieQuality,
		type Movie, type QualityProfile, type MediaInfo, type TMDBMovieResult, type QualityInfo
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import { toast } from '$lib/stores/toast';
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
	let confirmingDelete = $state(false);
	let deleting = $state(false);
	let showTrailerModal = $state(false);
	let selectedVideo = $state(0);
	let recommendations: TMDBMovieResult[] = $state([]);
	let qualityInfo: QualityInfo | null = $state(null);
	let monitored = $state(true);
	let monitoringLoading = $state(false);

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
			// Load quality/monitoring info
			try {
				qualityInfo = await getMovieQuality(id);
				if (qualityInfo?.override) {
					monitored = qualityInfo.override.monitored;
				}
			} catch { /* Quality info is optional */ }
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

	function handleDeleteClick() {
		showManageMenu = false;
		confirmingDelete = true;
	}

	async function handleConfirmDelete() {
		if (!movie) return;
		deleting = true;
		try {
			await deleteMovie(movie.id);
			goto('/library');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete';
			confirmingDelete = false;
		} finally {
			deleting = false;
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

	async function handleToggleMonitoring() {
		if (!movie) return;
		monitoringLoading = true;
		try {
			const newMonitored = !monitored;
			await setMovieQuality(movie.id, { monitored: newMonitored });
			monitored = newMonitored;
			toast.success(newMonitored ? 'Monitoring enabled' : 'Monitoring disabled');
		} catch (e) {
			console.error('Failed to update monitoring:', e);
			toast.error('Failed to update monitoring');
		} finally {
			monitoringLoading = false;
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
			<div class="spinner-lg text-cream"></div>
			<p class="text-text-secondary">Loading movie...</p>
		</div>
	</div>
{:else if error}
	<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-lg">
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
			<!-- Watchlist -->
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

			<!-- Mark Watched -->
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
							class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors"
						>
							{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
						</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
						<div class="border-t border-border-subtle my-1"></div>
						<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={handleDeleteClick}>Delete</button>
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
						<span class="text-text-muted">|</span>
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
						<span class="text-text-muted">|</span>
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
					<h2 class="text-lg font-semibold text-text-primary mb-4">Files</h2>
					<div class="flex gap-4">
						<div class="relative w-72 md:w-80">
							<!-- Monitor badge -->
							<button
								onclick={handleToggleMonitoring}
								disabled={monitoringLoading}
								class="absolute top-2 right-2 z-10 px-2 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1 {monitored ? 'bg-black/70 text-white' : 'bg-black/50 text-white/50'} hover:bg-black/80"
								title={monitored ? 'Monitored (click to disable)' : 'Not monitored (click to enable)'}
							>
								{#if monitoringLoading}
									<div class="spinner-sm"></div>
								{:else if monitored}
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
									</svg>
								{:else}
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
									</svg>
								{/if}
								<span>{monitored ? 'Monitored' : 'Not Monitored'}</span>
							</button>
							<button onclick={handlePlay} class="group w-full">
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

	<!-- Delete Confirmation Modal -->
	{#if confirmingDelete}
		<div class="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
			<div class="bg-bg-card border border-border-subtle rounded-2xl p-6 max-w-md w-full">
				<h3 class="text-lg font-semibold text-text-primary mb-2">Delete Movie</h3>
				<p class="text-text-secondary mb-4">
					Are you sure you want to delete "{movie?.title}"? This will remove the file from disk and cannot be undone.
				</p>
				<div class="flex gap-3 justify-end">
					<button
						class="px-4 py-2 rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors"
						onclick={() => confirmingDelete = false}
						disabled={deleting}
					>
						Cancel
					</button>
					<button
						class="px-4 py-2 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors disabled:opacity-50"
						onclick={handleConfirmDelete}
						disabled={deleting}
					>
						{deleting ? 'Deleting...' : 'Delete'}
					</button>
				</div>
			</div>
		</div>
	{/if}
{/if}
