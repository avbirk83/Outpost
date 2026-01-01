<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import {
		getDiscoverMovieDetail,
		createRequest,
		getTmdbImageUrl,
		addToWatchlist,
		removeFromWatchlist,
		isInWatchlist,
		type DiscoverMovieDetailWithStatus
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { formatRuntime, getYear } from '$lib/utils';
	import PersonModal from '$lib/components/PersonModal.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';

	let movie: DiscoverMovieDetailWithStatus | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let requesting = $state(false);
	let requested = $state(false);
	let inWatchlist = $state(false);
	let watchlistLoading = $state(false);
	let showTrailerModal = $state(false);
	let castScrollContainer: HTMLElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);
	let selectedPersonId = $state<number | null>(null);
	let selectedPersonName = $state<string>('');

	function updateCastScrollState() {
		if (!castScrollContainer) return;
		canScrollLeft = castScrollContainer.scrollLeft > 0;
		canScrollRight = castScrollContainer.scrollLeft < castScrollContainer.scrollWidth - castScrollContainer.clientWidth - 10;
	}

	function scrollCast(direction: 'left' | 'right') {
		if (!castScrollContainer) return;
		const scrollAmount = 300;
		castScrollContainer.scrollBy({
			left: direction === 'left' ? -scrollAmount : scrollAmount,
			behavior: 'smooth'
		});
		setTimeout(updateCastScrollState, 350);
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

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			const [movieData, watchlistStatus] = await Promise.all([
				getDiscoverMovieDetail(id),
				isInWatchlist(id, 'movie').catch(() => false)
			]);
			movie = movieData as DiscoverMovieDetailWithStatus;
			inWatchlist = watchlistStatus;
			if (movie.requested) {
				requested = true;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load movie details';
		} finally {
			loading = false;
		}
	});

	async function handleRequest() {
		if (!movie) return;
		requesting = true;

		try {
			const year = movie.releaseDate ? parseInt(movie.releaseDate.substring(0, 4)) : undefined;
			await createRequest({
				type: 'movie',
				tmdbId: movie.id,
				title: movie.title,
				year,
				overview: movie.overview || undefined,
				posterPath: movie.posterPath || undefined
			});
			requested = true;
			toast.success('Request submitted');
		} catch (e) {
			if (e instanceof Error && e.message === 'Already requested') {
				requested = true;
				toast.info('Already requested');
			} else {
				toast.error('Failed to create request');
			}
		} finally {
			requesting = false;
		}
	}

	async function toggleWatchlist() {
		if (!movie) return;
		watchlistLoading = true;
		try {
			if (inWatchlist) {
				await removeFromWatchlist(movie.id, 'movie');
				inWatchlist = false;
				toast.success('Removed from watchlist');
			} else {
				await addToWatchlist(movie.id, 'movie');
				inWatchlist = true;
				toast.success('Added to watchlist');
			}
		} catch (e) {
			console.error('Failed to update watchlist:', e);
			toast.error('Failed to update watchlist');
		} finally {
			watchlistLoading = false;
		}
	}

	function getStatusColor(status?: string): string {
		switch (status?.toLowerCase()) {
			case 'released':
				return 'text-green-400';
			case 'in production':
			case 'post production':
				return 'text-yellow-400';
			default:
				return 'text-green-400';
		}
	}

	function getLanguageName(code?: string): string {
		if (!code) return '';
		try {
			const displayNames = new Intl.DisplayNames(['en'], { type: 'language' });
			return displayNames.of(code) || code;
		} catch {
			return code;
		}
	}

	function getCountryFlag(code?: string): string {
		if (!code || code.length !== 2) return '';
		return code.toUpperCase().split('').map(c => String.fromCodePoint(127397 + c.charCodeAt(0))).join('');
	}

	function getCountryName(code?: string): string {
		if (!code) return '';
		try {
			const displayNames = new Intl.DisplayNames(['en'], { type: 'region' });
			return displayNames.of(code.toUpperCase()) || code;
		} catch {
			return code;
		}
	}
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
	</div>
{:else if movie}
	<div class="space-y-6 -mt-22 -mx-6">
		<!-- Hero Section -->
		<section class="relative min-h-[500px]">
			<!-- Backdrop -->
			{#if movie.backdropPath}
				<img
					src={getTmdbImageUrl(movie.backdropPath, 'w1280')}
					alt=""
					class="absolute inset-0 w-full h-full object-cover pointer-events-none"
					style="object-position: center 25%;"
					draggable="false"
				/>
				<div class="absolute inset-0 bg-gradient-to-r from-[#0a0a0a] via-[#0a0a0a]/80 to-transparent pointer-events-none"></div>
				<div class="absolute inset-0 bg-gradient-to-t from-[#0a0a0a] via-transparent to-[#0a0a0a]/30 pointer-events-none"></div>
			{/if}

			<!-- Hero Content: 3 columns -->
			<div class="relative z-10 px-6 pt-32 pb-8 flex gap-6">
				<!-- LEFT: Poster Card -->
				<div class="flex-shrink-0 w-64 mt-8">
					<div class="liquid-card overflow-hidden">
						<div class="relative aspect-[2/3] bg-bg-card">
							{#if movie.posterPath}
								<img
									src={getTmdbImageUrl(movie.posterPath, 'w500')}
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
							<!-- Status Badge -->
							{#if movie.inLibrary}
								<div class="absolute top-3 right-3">
									<div class="w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="In Library">
										<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
										</svg>
									</div>
								</div>
							{:else if requested || movie.requested}
								<div class="absolute top-3 right-3">
									<div class="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center" title="Requested">
										<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
									</div>
								</div>
							{/if}
						</div>
						<!-- Ratings Row -->
						<div class="p-3 flex justify-around items-center border-t border-white/10">
							{#if movie.rating}
								<a href="https://www.themoviedb.org/movie/{movie.id}" target="_blank" class="flex items-center gap-1.5 hover:opacity-80 transition-opacity" title="TMDB Rating">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6 rounded" />
									<span class="text-base font-bold text-white">{movie.rating.toFixed(1)}</span>
								</a>
							{/if}
							<div class="flex items-center gap-1.5 opacity-40" title="Rotten Tomatoes">
								<img src="/icons/rottentomatoes.svg" alt="Rotten Tomatoes" class="w-6 h-6" />
								<span class="text-base font-bold">--</span>
							</div>
							<div class="flex items-center gap-1.5 opacity-40" title="Metacritic">
								<img src="/icons/metacritic.svg" alt="Metacritic" class="w-6 h-6 rounded" />
								<span class="text-base font-bold">--</span>
							</div>
						</div>
					</div>
				</div>

				<!-- CENTER: Title, Tags, Overview, Controls -->
				<div class="flex-1 min-w-0 py-4">
					<!-- Title -->
					<h1 class="text-4xl md:text-5xl font-bold text-white mb-2">
						{movie.title}
						{#if movie.releaseDate}
							<span class="text-text-secondary font-normal">({getYear(movie.releaseDate)})</span>
						{/if}
					</h1>

					<!-- Meta line -->
					<div class="flex items-center gap-2 text-text-secondary mb-4">
						{#if movie.runtime}
							<span>{formatRuntime(movie.runtime)}</span>
						{/if}
						{#if movie.genres && movie.genres.length > 0}
							<span>•</span>
							<span>{movie.genres.join(', ')}</span>
						{/if}
					</div>

					<!-- Tags (clickable pills) -->
					{#if movie.genres && movie.genres.length > 0}
						<div class="flex flex-wrap gap-2 mb-4">
							{#each movie.genres as genre}
								<a href="/discover?tab=movies&genre={encodeURIComponent(genre)}" class="liquid-tag text-sm">
									{genre}
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
						<!-- Details (link to this page - could link to library if available) -->
						{#if movie.inLibrary && movie.libraryId}
							<IconButton href="/movies/{movie.libraryId}" variant="green" title="View in Library">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							</IconButton>
						{:else if movie.requestStatus === 'approved'}
							<IconButton variant="green" disabled title="Available in Library">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							</IconButton>
						{:else if requested || movie.requested}
							<IconButton variant="yellow" disabled title="Request Pending">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							</IconButton>
						{:else}
							<IconButton onclick={handleRequest} disabled={requesting} title="Request">
								{#if requesting}
									<div class="w-5 h-5 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
								{:else}
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
									</svg>
								{/if}
							</IconButton>
						{/if}

						<!-- Watchlist -->
						<IconButton
							onclick={toggleWatchlist}
							disabled={watchlistLoading}
							active={inWatchlist}
							title={inWatchlist ? 'In Watchlist' : 'Add to Watchlist'}
						>
							{#if watchlistLoading}
								<div class="w-5 h-5 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
							{:else if inWatchlist}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							{:else}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
							{/if}
						</IconButton>

						<!-- Trailer button -->
						{#if movie.trailerKey}
							<IconButton onclick={() => showTrailerModal = true} variant="red" title="Watch Trailer">
								<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M8 5v14l11-7z" />
								</svg>
							</IconButton>
						{/if}
					</div>
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
						{#if movie.releaseDate}
							<div class="flex justify-between">
								<span class="text-text-muted">Released</span>
								<span>{getYear(movie.releaseDate)}</span>
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

						<!-- Director -->
						{#if movie.director}
							<div class="flex justify-between">
								<span class="text-text-muted">Director</span>
								<span class="text-right">{movie.director}</span>
							</div>
						{/if}

						<!-- Language -->
						{#if movie.originalLanguage}
							<div class="flex justify-between">
								<span class="text-text-muted">Language</span>
								<span>{getLanguageName(movie.originalLanguage)}</span>
							</div>
						{/if}

						<!-- Country -->
						{#if movie.productionCountries && movie.productionCountries.length > 0}
							<div class="flex justify-between items-center">
								<span class="text-text-muted">Country</span>
								<span class="flex items-center gap-1.5">
									<span class="text-base">{getCountryFlag(movie.productionCountries[0])}</span>
									<span>{getCountryName(movie.productionCountries[0])}</span>
								</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- External Links -->
						<div class="flex justify-center gap-3">
							<a href="https://www.themoviedb.org/movie/{movie.id}" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on TMDB">
								<img src="/icons/tmdb.svg" alt="TMDB" class="w-7 h-7" />
							</a>
							{#if movie.imdbId}
								<a href="https://www.imdb.com/title/{movie.imdbId}" target="_blank"
								   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on IMDb">
									<img src="/icons/imdb.svg" alt="IMDb" class="w-7 h-7" />
								</a>
							{/if}
							<a href="https://trakt.tv/search/tmdb/{movie.id}?id_type=movie" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Trakt">
								<img src="/icons/trakt.svg" alt="Trakt" class="w-7 h-7" />
							</a>
							<a href="https://letterboxd.com/tmdb/{movie.id}" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Letterboxd">
								<img src="/icons/letterboxd.svg" alt="Letterboxd" class="w-7 h-7" />
							</a>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- Cast Section -->
		{#if movie.cast && movie.cast.length > 0}
			<section class="px-6">
				<div class="flex items-center justify-between mb-3">
					<h2 class="text-lg font-semibold text-text-primary">Cast</h2>
					<div class="flex gap-1">
						<button
							onclick={() => scrollCast('left')}
							disabled={!canScrollLeft}
							class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
							aria-label="Scroll left"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
							</svg>
						</button>
						<button
							onclick={() => scrollCast('right')}
							disabled={!canScrollRight}
							class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
							aria-label="Scroll right"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
							</svg>
						</button>
					</div>
				</div>
				<div
					bind:this={castScrollContainer}
					onscroll={updateCastScrollState}
					class="flex gap-5 overflow-x-auto pt-1 pl-1 pb-2 -ml-1 scrollbar-thin"
				>
					{#each movie.cast as actor}
						<button
							onclick={() => handlePersonClick({ id: actor.id, name: actor.name })}
							class="flex-shrink-0 w-28 text-center cursor-pointer group"
						>
							<div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10 group-hover:ring-white/30 transition-all">
								{#if actor.photo}
									<img
										src={getTmdbImageUrl(actor.photo, 'w185')}
										alt={actor.name}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-3xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
										{actor.name.charAt(0)}
									</div>
								{/if}
							</div>
							<p class="mt-2 text-sm font-medium text-text-primary truncate group-hover:text-white transition-colors">{actor.name}</p>
							<p class="text-xs text-text-muted truncate">{actor.character}</p>
						</button>
					{/each}
				</div>
			</section>
		{/if}
	</div>

	<!-- Person Modal -->
	<PersonModal
		personId={selectedPersonId}
		personName={selectedPersonName}
		onClose={closePersonModal}
	/>

	<!-- Trailer Modal -->
	{#if movie.trailerKey}
		<TrailerModal
			open={showTrailerModal}
			trailers={[{ key: movie.trailerKey, name: 'Trailer', type: 'Trailer', site: 'YouTube' }]}
			onClose={() => showTrailerModal = false}
		/>
	{/if}
{/if}
