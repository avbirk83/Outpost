<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getTrendingMovies,
		getTrendingShows,
		getImageUrl,
		getTmdbImageUrl,
		getMovies,
		getShows,
		getContinueWatching,
		removeContinueWatching,
		getRequests,
		getWatchlist,
		removeFromWatchlist,
		type DiscoverItem,
		type Movie,
		type Show,
		type ContinueWatchingItem,
		type Request,
		type WatchlistItem
	} from '$lib/api';
	import MediaRow from '$lib/components/MediaRow.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import ContinueCard from '$lib/components/ContinueCard.svelte';

	function getYear(dateStr: string | undefined): string {
		if (!dateStr) return '';
		return dateStr.substring(0, 4);
	}

	let recentMovies: Movie[] = $state([]);
	let recentShows: Show[] = $state([]);
	let continueWatching: ContinueWatchingItem[] = $state([]);
	let recentRequests: Request[] = $state([]);
	let watchlistItems: WatchlistItem[] = $state([]);

	// Hero carousel state
	let heroItems: DiscoverItem[] = $state([]);
	let heroIndex = $state(0);
	let autoplayTimer: ReturnType<typeof setInterval> | null = null;
	const AUTOPLAY_INTERVAL = 8000; // 8 seconds

	let loading = $state(true);
	let error: string | null = $state(null);

	// Current hero item
	const currentHero = $derived(() => heroItems[heroIndex] || null);

	function nextHero() {
		if (heroItems.length > 0) {
			heroIndex = (heroIndex + 1) % heroItems.length;
		}
		resetAutoplay();
	}

	function prevHero() {
		if (heroItems.length > 0) {
			heroIndex = (heroIndex - 1 + heroItems.length) % heroItems.length;
		}
		resetAutoplay();
	}

	function goToHero(index: number) {
		heroIndex = index;
		resetAutoplay();
	}

	function resetAutoplay() {
		if (autoplayTimer) {
			clearInterval(autoplayTimer);
		}
		autoplayTimer = setInterval(() => {
			if (heroItems.length > 0) {
				heroIndex = (heroIndex + 1) % heroItems.length;
			}
		}, AUTOPLAY_INTERVAL);
	}

	onMount(async () => {
		try {
			const [trendingMoviesRes, trendingShowsRes, movies, shows, cw, requests, watchlist] = await Promise.all([
				getTrendingMovies(),
				getTrendingShows(),
				getMovies().catch(() => []),
				getShows().catch(() => []),
				getContinueWatching().catch(() => []),
				getRequests().catch(() => []),
				getWatchlist().catch(() => [])
			]);

			// Combine trending movies and shows for hero carousel
			// Take top 10 movies and top 10 shows, interleave them, filter for backdrop
			const trendingMovies = trendingMoviesRes.results.slice(0, 10);
			const trendingShows = trendingShowsRes.results.slice(0, 10);

			// Interleave movies and shows for variety, explicitly setting mediaType
			const combined: DiscoverItem[] = [];
			const maxLen = Math.max(trendingMovies.length, trendingShows.length);
			for (let i = 0; i < maxLen; i++) {
				if (i < trendingMovies.length) combined.push({ ...trendingMovies[i], mediaType: 'movie' });
				if (i < trendingShows.length) combined.push({ ...trendingShows[i], mediaType: 'tv' });
			}

			// Filter for items with backdrop, limit to 15
			heroItems = combined.filter(item => item.backdropPath).slice(0, 15);

			// Recent additions sorted by ID (newest first)
			recentMovies = movies.slice(0, 12);
			recentShows = shows.slice(0, 12);
			continueWatching = cw;
			// Recent requests (limit to 10, newest first)
			recentRequests = requests.slice(0, 10);
			watchlistItems = watchlist;

			// Start autoplay
			resetAutoplay();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load content';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		if (autoplayTimer) {
			clearInterval(autoplayTimer);
		}
	});

	function getStatusColor(status: string): string {
		switch (status) {
			case 'approved': return 'bg-blue-500';
			case 'pending': return 'bg-white/50';
			case 'rejected': return 'bg-red-500';
			default: return 'bg-gray-500';
		}
	}

	function getStatusLabel(status: string): string {
		switch (status) {
			case 'approved': return 'Available';
			case 'pending': return 'Requested';
			case 'rejected': return 'Rejected';
			default: return status;
		}
	}

	function formatTimeLeft(position: number, duration: number): string {
		const remaining = Math.max(0, duration - position);
		const minutes = Math.floor(remaining / 60);
		if (minutes >= 60) {
			const hours = Math.floor(minutes / 60);
			const mins = minutes % 60;
			return `${hours}h ${mins}m left`;
		}
		return `${minutes}m left`;
	}

	async function handleRemoveFromContinue(item: ContinueWatchingItem) {
		try {
			await removeContinueWatching(item.mediaType, item.mediaId);
			continueWatching = continueWatching.filter(
				i => !(i.mediaType === item.mediaType && i.mediaId === item.mediaId)
			);
		} catch (e) {
			console.error('Failed to remove from continue watching:', e);
		}
	}

	async function handleRemoveFromWatchlist(item: WatchlistItem) {
		try {
			await removeFromWatchlist(item.tmdbId, item.mediaType);
			watchlistItems = watchlistItems.filter(
				i => !(i.tmdbId === item.tmdbId && i.mediaType === item.mediaType)
			);
		} catch (e) {
			console.error('Failed to remove from watchlist:', e);
		}
	}

	function getWatchlistItemHref(item: WatchlistItem): string {
		if (item.inLibrary && item.libraryId) {
			return item.mediaType === 'movie' ? `/movies/${item.libraryId}` : `/tv/${item.libraryId}`;
		}
		return item.mediaType === 'movie' ? `/discover/movie/${item.tmdbId}` : `/discover/show/${item.tmdbId}`;
	}
</script>

<svelte:head>
	<title>Home - Outpost</title>
</svelte:head>

<div class="space-y-8 -mt-22 -mx-6">
	{#if error}
		<div class="mx-6 mt-6 bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-lg">
			{error}
			<button class="ml-2 underline" onclick={() => (error = null)}>Dismiss</button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center h-96">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading discover content...</p>
			</div>
		</div>
	{:else}
		<!-- Hero Carousel Section -->
		{#if heroItems.length > 0}
			{@const hero = currentHero()}
			{#if hero}
				<section class="relative h-[45vh] min-h-[380px] overflow-hidden">
					<!-- Backdrop image with fade transition -->
					{#key heroIndex}
						<img
							src={getTmdbImageUrl(hero.backdropPath || '', 'original')}
							alt={hero.title || hero.name}
							class="absolute inset-0 w-full h-full object-cover animate-fade-in pointer-events-none"
							style="object-position: center 25%;"
							draggable="false"
						/>
					{/key}

					<!-- Gradient overlays -->
					<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/80 to-transparent pointer-events-none"></div>
					<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/30 pointer-events-none"></div>

					<!-- Content -->
					<div class="relative h-full flex items-end pb-12 px-6">
						<div class="max-w-2xl">
							{#key heroIndex}
								<div class="animate-fade-in">
									<div class="flex items-center gap-2 mb-4">
										<span class="liquid-tag !bg-white/10 !border-t-white/20 !border-l-white/10">
											Trending Now
										</span>
										<span class="liquid-tag">
											{hero.mediaType === 'movie' ? 'Movie' : 'TV Series'}
										</span>
									</div>
									<h1 class="text-4xl md:text-5xl font-bold text-white mb-3">{hero.title || hero.name}</h1>
									<p class="text-text-secondary text-lg mb-2">
										{getYear(hero.releaseDate || hero.firstAirDate)}
										{#if hero.rating}
											<span class="mx-2">•</span>
											<span class="text-white">{hero.rating.toFixed(1)}</span>
										{/if}
									</p>
									{#if hero.overview}
										<p class="text-text-secondary line-clamp-2 mb-6 max-w-xl">{hero.overview}</p>
									{/if}
								</div>
							{/key}
							<div class="flex items-center gap-3">
								<a
									href={hero.mediaType === 'movie' ? `/discover/movie/${hero.id}` : `/discover/show/${hero.id}`}
									class="liquid-btn inline-flex items-center gap-2"
								>
									<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
										<path d="M8 5v14l11-7z" />
									</svg>
									View Details
								</a>
								<button
									class="liquid-btn-icon"
									aria-label="Add to watchlist"
								>
									<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
									</svg>
								</button>
							</div>
						</div>

						<!-- Carousel navigation - center bottom with arrows -->
						<div class="absolute bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-3">
							<button
								onclick={prevHero}
								class="w-8 h-8 rounded-full bg-black/60 hover:bg-black/80 flex items-center justify-center transition-colors border border-white/10"
								aria-label="Previous"
							>
								<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
								</svg>
							</button>
							<div class="flex items-center gap-2">
								{#each heroItems as _, i}
									<button
										onclick={() => goToHero(i)}
										class="w-2 h-2 rounded-full transition-all {i === heroIndex ? 'bg-white w-6' : 'bg-white/30 hover:bg-white/50'}"
										aria-label="Go to slide {i + 1}"
									></button>
								{/each}
							</div>
							<button
								onclick={nextHero}
								class="w-8 h-8 rounded-full bg-black/60 hover:bg-black/80 flex items-center justify-center transition-colors border border-white/10"
								aria-label="Next"
							>
								<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</button>
						</div>
					</div>
				</section>
			{/if}
		{/if}

		<div class="space-y-10 px-6 pb-8">
			<!-- Watchlist - combines continue watching and manually added items -->
			{#if continueWatching.length > 0 || watchlistItems.length > 0}
				<MediaRow title="Watchlist">
					<!-- Continue Watching items first (have progress) -->
					{#each continueWatching as item}
						<div class="flex-shrink-0 w-72 md:w-80">
							<ContinueCard
								id={item.mediaId}
								title={item.title}
								subtitle={item.subtitle}
								backdropUrl={item.backdropPath ? getImageUrl(item.backdropPath) : undefined}
								posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
								progress={item.progressPercent}
								duration={formatTimeLeft(item.position, item.duration)}
								type={item.mediaType === 'movie' ? 'movie' : 'episode'}
								onRemove={() => handleRemoveFromContinue(item)}
							/>
						</div>
					{/each}
					<!-- Manually added watchlist items -->
					{#each watchlistItems as item}
						<div class="flex-shrink-0 w-72 md:w-80">
							<ContinueCard
								id={item.libraryId || item.tmdbId}
								title={item.title}
								subtitle={item.year?.toString() || ''}
								backdropUrl={item.backdropPath ? (item.inLibrary ? getImageUrl(item.backdropPath) : getTmdbImageUrl(item.backdropPath, 'w780')) : undefined}
								posterUrl={item.posterPath ? (item.inLibrary ? getImageUrl(item.posterPath) : getTmdbImageUrl(item.posterPath, 'w300')) : undefined}
								progress={item.progress}
								type={item.mediaType === 'movie' ? 'movie' : 'series'}
								href={getWatchlistItemHref(item)}
								onRemove={() => handleRemoveFromWatchlist(item)}
							/>
						</div>
					{/each}
				</MediaRow>
			{/if}

			<!-- Recent Requests -->
			{#if recentRequests.length > 0}
				<MediaRow title="Recent Requests" viewAllHref="/requests">
					{#each recentRequests as request}
						<div class="flex-shrink-0 w-36 md:w-40">
							<PosterCard
								href={request.type === 'movie' ? `/discover/movie/${request.tmdbId}` : `/discover/show/${request.tmdbId}`}
								title={request.title}
								subtitle={request.userName || 'Unknown'}
								posterUrl={request.posterPath ? getTmdbImageUrl(request.posterPath, 'w300') : undefined}
								mediaType={request.type === 'movie' ? 'movie' : 'series'}
								requested={true}
								requestStatus={request.status}
							/>
						</div>
					{/each}
				</MediaRow>
			{/if}

			<!-- Recently Added to Library -->
			{#if recentMovies.length > 0 || recentShows.length > 0}
				<MediaRow title="Recently Added" viewAllHref="/library">
					{#each recentMovies as movie}
						<div class="flex-shrink-0 w-36 md:w-40">
							<PosterCard
								href="/movies/{movie.id}"
								title={movie.title}
								subtitle={movie.year?.toString() || ''}
								posterUrl={movie.posterPath ? getImageUrl(movie.posterPath) : undefined}
								mediaType="movie"
								rating={movie.rating}
							/>
						</div>
					{/each}
					{#each recentShows as show}
						<div class="flex-shrink-0 w-36 md:w-40">
							<PosterCard
								href="/tv/{show.id}"
								title={show.title}
								subtitle={show.year?.toString() || ''}
								posterUrl={show.posterPath ? getImageUrl(show.posterPath) : undefined}
								mediaType="series"
								rating={show.rating}
							/>
						</div>
					{/each}
				</MediaRow>
			{/if}
		</div>
	{/if}
</div>
