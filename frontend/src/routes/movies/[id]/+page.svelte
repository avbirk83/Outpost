<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getMovie, refreshMovieMetadata, getImageUrl, getTmdbImageUrl,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getMediaInfo, getMovieSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		type Movie, type QualityProfile, type MediaInfo, type TMDBMovieResult
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import {
		formatRuntime, getOfficialTrailer, parseGenres, parseCast, parseCrew,
		formatMoneyFull, getLanguageName, getCountryName, getCountryFlag,
		formatResolution, formatAudioChannels, getStatusColor
	} from '$lib/utils';
	import Dropdown from '$lib/components/Dropdown.svelte';
	import PersonModal from '$lib/components/PersonModal.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import ScrollableRow from '$lib/components/ScrollableRow.svelte';
	import PersonCard from '$lib/components/PersonCard.svelte';
	import RatingsRow from '$lib/components/RatingsRow.svelte';
	import ExternalLinks from '$lib/components/ExternalLinks.svelte';

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
	let recsScrollContainer: HTMLElement;
	let canScrollRecsLeft = $state(false);
	let canScrollRecsRight = $state(true);
	let selectedPersonId = $state<number | null>(null);
	let selectedPersonName = $state<string>('');

	function updateRecsScrollState() {
		if (!recsScrollContainer) return;
		canScrollRecsLeft = recsScrollContainer.scrollLeft > 0;
		canScrollRecsRight = recsScrollContainer.scrollLeft < recsScrollContainer.scrollWidth - recsScrollContainer.clientWidth - 10;
	}

	function scrollRecs(direction: 'left' | 'right') {
		if (!recsScrollContainer) return;
		const scrollAmount = 300;
		recsScrollContainer.scrollBy({
			left: direction === 'left' ? -scrollAmount : scrollAmount,
			behavior: 'smooth'
		});
		setTimeout(updateRecsScrollState, 350);
	}

	function handlePersonClick(person: { id?: number; name: string }) {
		if (person.id) {
			selectedPersonId = person.id;
			selectedPersonName = person.name;
		}
	}

	function closePersonModal() {
		selectedPersonId = null;
		selectedPersonName = '';
	}

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
		console.log('handleRefresh called', movie?.id);
		if (!movie) {
			console.log('No movie, returning');
			return;
		}
		showManageMenu = false;
		refreshing = true;
		try {
			console.log('Calling refreshMovieMetadata...');
			movie = await refreshMovieMetadata(movie.id);
			console.log('Refresh complete', movie);
		} catch (e) {
			console.error('Refresh failed', e);
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
	<div class="space-y-6 -mt-22 -mx-6">
		<!-- ============================================
		     1. HERO SECTION
		     ============================================ -->
		<section class="relative min-h-[500px]">
			<!-- Backdrop - same as home page -->
			{#if movie.backdropPath}
				<img
					src={getImageUrl(movie.backdropPath)}
					alt=""
					class="absolute inset-0 w-full h-full object-cover pointer-events-none"
					style="object-position: center 25%;"
					draggable="false"
				/>
				<!-- Gradient overlays - same as home page -->
				<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/80 to-transparent pointer-events-none"></div>
				<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/30 pointer-events-none"></div>
			{/if}

			<!-- Hero Content: 3 columns -->
			<div class="relative z-10 px-6 pt-32 pb-8 flex gap-6">
				<!-- LEFT: Poster Card -->
				<div class="flex-shrink-0 w-64 mt-8">
					<div class="liquid-card overflow-hidden">
						<!-- Poster -->
						<div class="relative aspect-[2/3] bg-bg-card">
							{#if movie.posterPath}
								<img
									src={getImageUrl(movie.posterPath)}
									alt={movie.title}
									class="w-full h-full object-cover"
								/>
							{:else}
								<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
									<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
									</svg>
								</div>
							{/if}
							<!-- Status Badge - same green checkmark as PosterCard -->
							<div class="absolute top-3 right-3">
								<div class="w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="In Library">
									<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
									</svg>
								</div>
							</div>
						</div>
						<!-- Ratings Row -->
						<RatingsRow
							tmdbId={movie.tmdbId}
							tmdbRating={movie.rating}
							mediaType="movie"
						/>
					</div>
				</div>

				<!-- CENTER: Title, Tags, Overview, Controls -->
				<div class="flex-1 min-w-0 py-4">
					<!-- Title -->
					<h1 class="text-4xl md:text-5xl font-bold text-white mb-2">
						{movie.title}
						<span class="text-text-secondary font-normal">({movie.year})</span>
					</h1>

					<!-- Meta line -->
					<div class="flex items-center gap-2 text-text-secondary mb-4">
						{#if movie.contentRating}
							<span class="px-2 py-0.5 border border-white/30 text-xs font-medium">{movie.contentRating}</span>
						{/if}
						{#if movie.runtime}
							<span>{formatRuntime(movie.runtime)}</span>
						{/if}
						{#if tags.length > 0}
							<span>•</span>
							<span>{tags.join(', ')}</span>
						{/if}
					</div>

					<!-- Tags (clickable pills) -->
					{#if tags.length > 0}
						<div class="flex flex-wrap gap-2 mb-4">
							{#each tags as tag}
								<a href="/discover/movie?genre={encodeURIComponent(tag)}" class="liquid-tag text-sm">
									{tag}
								</a>
							{/each}
						</div>
					{/if}

					<!-- Tagline -->
					{#if movie.tagline}
						<p class="text-text-secondary italic mb-4">"{movie.tagline}"</p>
					{/if}

					<!-- Overview -->
					{#if movie.overview}
						<p class="text-text-secondary leading-relaxed max-w-2xl mb-5">
							{movie.overview}
						</p>
					{/if}

					<!-- Icon bubble controls -->
					<div class="flex items-center gap-2 mb-5">
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

						{#if getOfficialTrailer(movie?.trailers)}
							<IconButton
								onclick={() => showTrailerModal = true}
								variant="red"
								title="Watch Trailer"
							>
								<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M8 5v14l11-7z" />
								</svg>
							</IconButton>
						{/if}

						<!-- Manage dropdown -->
						<div class="relative">
							<IconButton
								onclick={() => { console.log('Toggle menu', !showManageMenu); showManageMenu = !showManageMenu; }}
								title="Manage"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
								</svg>
							</IconButton>
							{#if showManageMenu}
								<!-- Backdrop to close menu -->
								<button
									type="button"
									class="fixed inset-0 z-[55] cursor-default"
									onclick={() => showManageMenu = false}
									aria-label="Close menu"
								></button>
								<div class="absolute left-0 mt-2 w-48 py-1 z-[60] bg-[#141416] border border-white/10 rounded-2xl shadow-xl overflow-hidden">
									<button
										onclick={() => { console.log('Refresh button clicked'); handleRefresh(); }}
										class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors"
									>
										{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
									</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Images</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Find Subtitles</button>
									<div class="border-t border-white/10 my-1"></div>
									<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={() => showManageMenu = false}>Delete</button>
								</div>
							{/if}
						</div>
					</div>

					<!-- Playback selectors -->
					{#if mediaInfo}
						<div class="inline-flex items-center p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
							{#if mediaInfo.videoStreams?.length}
								<Dropdown
									icon="video"
									options={mediaInfo.videoStreams.map((v, i) => ({ value: i, label: `${formatResolution(v.width, v.height)} ${v.codec?.toUpperCase() || ''}` }))}
									value={selectedVideo}
									onchange={(v) => selectedVideo = v as number}
									inline={true}
								/>
							{/if}
							{#if mediaInfo.audioStreams?.length}
								{#if mediaInfo.videoStreams?.length}
									<div class="w-px h-6 bg-white/10"></div>
								{/if}
								<Dropdown
									icon="audio"
									options={mediaInfo.audioStreams.map((a, i) => ({ value: i, label: `${a.language?.toUpperCase() || 'UNK'} ${a.codec?.toUpperCase() || ''} ${formatAudioChannels(a.channels)}` }))}
									value={selectedAudio}
									onchange={(v) => selectedAudio = v as number}
									inline={true}
								/>
							{/if}
							{#if mediaInfo.videoStreams?.length || mediaInfo.audioStreams?.length}
								<div class="w-px h-6 bg-white/10"></div>
							{/if}
							<Dropdown
								icon="subtitles"
								options={[{ value: null, label: 'Off' }, ...(mediaInfo.subtitleTracks || []).map(s => ({ value: s.index, label: s.title || getLanguageName(s.language) }))]}
								value={selectedSubtitle}
								onchange={(v) => selectedSubtitle = v as number | null}
								inline={true}
							/>
						</div>
					{/if}
				</div>

				<!-- RIGHT: Info Panel Card -->
				<div class="flex-shrink-0 w-72 mt-8">
					<div class="liquid-card p-4 space-y-2.5 text-sm">
						<!-- Status -->
						<div class="flex justify-between">
							<span class="text-text-muted">Status</span>
							<span class="{getStatusColor(movie.status)} font-medium">{movie.status || 'Released'}</span>
						</div>

						<!-- Release Date -->
						{#if movie.year}
							<div class="flex justify-between">
								<span class="text-text-muted">Released</span>
								<span>{movie.year}</span>
							</div>
						{/if}

						<!-- Runtime -->
						{#if movie.runtime}
							<div class="flex justify-between">
								<span class="text-text-muted">Runtime</span>
								<span>{formatRuntime(movie.runtime)}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- Budget -->
						{#if movie.budget}
							<div class="flex justify-between">
								<span class="text-text-muted">Budget</span>
								<span>{formatMoneyFull(movie.budget)}</span>
							</div>
						{/if}

						<!-- Revenue -->
						{#if movie.revenue}
							<div class="flex justify-between">
								<span class="text-text-muted">Revenue</span>
								<span class="{movie.revenue > (movie.budget || 0) ? 'text-green-400' : 'text-red-400'}">{formatMoneyFull(movie.revenue)}</span>
							</div>
						{/if}

						{#if movie.budget || movie.revenue}
							<div class="border-t border-white/10 my-2"></div>
						{/if}

						<!-- Language -->
						{#if movie.originalLanguage}
							<div class="flex justify-between">
								<span class="text-text-muted">Language</span>
								<span>{getLanguageName(movie.originalLanguage)}</span>
							</div>
						{/if}

						<!-- Country -->
						{#if movie.country}
							<div class="flex justify-between items-center">
								<span class="text-text-muted">Country</span>
								<span class="flex items-center gap-1.5">
									<span class="text-base">{getCountryFlag(movie.country)}</span>
									<span>{getCountryName(movie.country)}</span>
								</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- Parental -->
						{#if movie.contentRating}
							<div class="flex justify-between items-center">
								<span class="text-text-muted">Parental</span>
								<span class="flex items-center gap-2">
									<span class="px-1.5 py-0.5 bg-white/10 rounded text-xs font-medium">{movie.contentRating}</span>
									{#if movie.imdbId}
										<a href="https://www.imdb.com/title/{movie.imdbId}/parentalguide" target="_blank" class="text-sky-400 hover:text-sky-300 text-xs">
											View ↗
										</a>
									{/if}
								</span>
							</div>
						{/if}

						<!-- Date Added -->
						{#if movie.addedAt}
							<div class="flex justify-between">
								<span class="text-text-muted">Added</span>
								<span>{new Date(movie.addedAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- External Links -->
						<ExternalLinks
							tmdbId={movie.tmdbId}
							imdbId={movie.imdbId}
							mediaType="movie"
						/>
					</div>
				</div>
			</div>
		</section>

		<!-- ============================================
		     3. FILES SECTION
		     ============================================ -->
		{#if mediaInfo}
			<section class="px-6">
				<h2 class="text-lg font-semibold text-text-primary mb-3">Files</h2>
				<div class="flex gap-4">
					<!-- File card - same style as ContinueCard -->
					<button onclick={handlePlay} class="group relative w-72 md:w-80">
						<div class="relative aspect-video bg-bg-card overflow-hidden rounded-xl">
							<!-- Backdrop image -->
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

							<!-- Gradient overlay -->
							<div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>

							<!-- Play button - centered, on hover -->
							<div class="absolute inset-0 flex items-center justify-center">
								<div class="w-14 h-14 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transform scale-75 group-hover:scale-100 transition-all duration-300 border border-white/30">
									<svg class="w-7 h-7 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
										<path d="M8 5v14l11-7z" />
									</svg>
								</div>
							</div>

							<!-- Quality badge - top left -->
							{#if mediaInfo.videoStreams?.[0]}
								<div class="absolute top-3 left-3">
									<div class="liquid-badge-sm">
										{formatResolution(mediaInfo.videoStreams[0].width, mediaInfo.videoStreams[0].height)}
									</div>
								</div>
							{/if}

							<!-- Content overlay at bottom -->
							<div class="absolute bottom-0 left-0 right-0 p-4">
								<h3 class="text-base font-semibold text-white truncate">{movie.title}</h3>
								<p class="text-sm text-white/70 truncate mt-0.5">
									{#if mediaInfo.videoStreams?.[0]}
										{mediaInfo.videoStreams[0].codec?.toUpperCase()}
									{/if}
									{#if mediaInfo.audioStreams?.[0]}
										• {mediaInfo.audioStreams[0].codec?.toUpperCase()} {formatAudioChannels(mediaInfo.audioStreams[0].channels)}
									{/if}
								</p>
							</div>
						</div>
					</button>
				</div>
			</section>
		{/if}

		<!-- ============================================
		     4. CAST & CREW
		     ============================================ -->
		<!-- Cast -->
		{#if parseCast(movie.cast).length > 0}
			<ScrollableRow title="Cast">
				{#each parseCast(movie.cast) as actor}
					<PersonCard
						name={actor.name}
						role={actor.character}
						profilePath={actor.profile_path}
						onclick={() => handlePersonClick(actor)}
					/>
				{/each}
			</ScrollableRow>
		{/if}

		<!-- Crew -->
		{#if parseCrew(movie.crew).length > 0}
			<ScrollableRow title="Crew">
				{#each parseCrew(movie.crew) as member}
					<PersonCard
						name={member.name}
						role={member.job}
						profilePath={member.profile_path}
						onclick={() => handlePersonClick(member)}
					/>
				{/each}
			</ScrollableRow>
		{/if}

		<!-- ============================================
		     5. SUGGESTIONS ROW
		     ============================================ -->
		<section class="px-6">
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-lg font-semibold text-text-primary">More Like This</h2>
				{#if recommendations.length > 0}
					<div class="flex gap-1">
						<button
							onclick={() => scrollRecs('left')}
							disabled={!canScrollRecsLeft}
							class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
							aria-label="Scroll left"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
							</svg>
						</button>
						<button
							onclick={() => scrollRecs('right')}
							disabled={!canScrollRecsRight}
							class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
							aria-label="Scroll right"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
							</svg>
						</button>
					</div>
				{/if}
			</div>
			{#if recommendations.length > 0}
				<div
					bind:this={recsScrollContainer}
					onscroll={updateRecsScrollState}
					class="flex gap-4 overflow-x-auto pb-2 scrollbar-thin"
				>
					{#each recommendations as rec}
						<a href="/discover/movie/{rec.id}" class="flex-shrink-0 w-32 group">
							<div class="relative aspect-[2/3] rounded-lg overflow-hidden bg-bg-card">
								{#if rec.poster_path}
									<img
										src={getTmdbImageUrl(rec.poster_path, 'w342')}
										alt={rec.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
										<svg class="w-10 h-10 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
										</svg>
									</div>
								{/if}
								<!-- Hover overlay -->
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
							</div>
							<p class="mt-2 text-xs text-text-primary truncate">{rec.title}</p>
							<p class="text-[10px] text-text-muted">{rec.release_date?.split('-')[0] || ''}</p>
						</a>
					{/each}
				</div>
			{:else}
				<div class="flex gap-2">
					{#each tags.slice(0, 3) as genre}
						<a href="/discover/movie?genre={encodeURIComponent(genre)}" class="liquid-btn-sm">
							Browse {genre} →
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- ============================================
		     6. FOOTER (File Info)
		     ============================================ -->
		{#if mediaInfo}
			<footer class="px-6 pb-8">
				<div class="text-xs text-text-muted text-center">
					{#if mediaInfo.videoStreams?.[0]}
						{mediaInfo.videoStreams[0].codec?.toUpperCase()} {formatResolution(mediaInfo.videoStreams[0].width, mediaInfo.videoStreams[0].height)}
					{/if}
					{#if mediaInfo.audioStreams?.[0]}
						• {mediaInfo.audioStreams[0].codec?.toUpperCase()} {formatAudioChannels(mediaInfo.audioStreams[0].channels)}
					{/if}
					{#if mediaInfo.container}
						• {mediaInfo.container.toUpperCase()}
					{/if}
				</div>
			</footer>
		{/if}
	</div>

	<!-- Trailer Modal -->
	<TrailerModal
		bind:open={showTrailerModal}
		trailersJson={movie?.trailers}
		title={movie?.title}
	/>

	<!-- Person Modal -->
	<PersonModal
		personId={selectedPersonId}
		personName={selectedPersonName}
		onClose={closePersonModal}
	/>

{/if}
