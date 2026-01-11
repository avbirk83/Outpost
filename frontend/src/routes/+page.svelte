<script lang="ts">
	import { goto } from '$app/navigation';
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
		createRequest,
		addToWatchlist,
		isInWatchlist,
		getCollections,
		type DiscoverItem,
		type Movie,
		type Show,
		type ContinueWatchingItem,
		type Request,
		type WatchlistItem,
		type Collection
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import { getYear, formatTimeLeft } from '$lib/utils';
	import ScrollSection from '$lib/components/containers/ScrollSection.svelte';
	import MediaCard from '$lib/components/media/MediaCard.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';

	function formatShortDate(dateStr: string | undefined): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function isNew(addedAt: string | undefined): boolean {
		if (!addedAt) return false;
		const added = new Date(addedAt);
		const now = new Date();
		const diffDays = (now.getTime() - added.getTime()) / (1000 * 60 * 60 * 24);
		return diffDays <= 7;
	}

	let recentMovies: Movie[] = $state([]);
	let recentShows: Show[] = $state([]);
	let continueWatching: ContinueWatchingItem[] = $state([]);
	let recentRequests: Request[] = $state([]);
	let watchlistItems: WatchlistItem[] = $state([]);
	let collections: Collection[] = $state([]);

	// Derived watchlist filters
	const libraryWatchlist = $derived(watchlistItems.filter(item => item.inLibrary));
	const wantedItems = $derived(watchlistItems.filter(item => !item.inLibrary));

	// Hero carousel state
	let heroItems: DiscoverItem[] = $state([]);
	let heroIndex = $state(0);
	let autoplayTimer: ReturnType<typeof setInterval> | null = null;
	const AUTOPLAY_INTERVAL = 8000;

	// Hero action states
	let heroWatchlistStatus: Map<string, boolean> = $state(new Map());
	let requestingHero = $state(false);
	let togglingWatchlist = $state(false);

	let loading = $state(true);
	let error: string | null = $state(null);
	let showTrailerModal = $state(false);
	let trailerHero: DiscoverItem | null = $state(null); // Capture hero when opening trailer

	const currentHero = $derived(() => heroItems[heroIndex] || null);

	// Combined recently added (movies + shows, sorted by addedAt, limited to 20)
	const recentlyAdded = $derived(() => {
		const combined: Array<{ type: 'movie' | 'tv'; item: Movie | Show }> = [
			...recentMovies.map(m => ({ type: 'movie' as const, item: m })),
			...recentShows.map(s => ({ type: 'tv' as const, item: s }))
		];
		// Sort by addedAt descending
		combined.sort((a, b) => {
			const aDate = new Date(a.item.addedAt || 0).getTime();
			const bDate = new Date(b.item.addedAt || 0).getTime();
			return bDate - aDate;
		});
		return combined.slice(0, 20);
	});

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
			const [trendingMoviesRes, trendingShowsRes, movies, shows, cw, requests, watchlist, colls] = await Promise.all([
				getTrendingMovies(),
				getTrendingShows(),
				getMovies().catch(() => []),
				getShows().catch(() => []),
				getContinueWatching().catch(() => []),
				getRequests().catch(() => []),
				getWatchlist().catch(() => []),
				getCollections().catch(() => [])
			]);

			const trendingMovies = (trendingMoviesRes?.results || []).slice(0, 10);
			const trendingShows = (trendingShowsRes?.results || []).slice(0, 10);

			const combined: DiscoverItem[] = [];
			const maxLen = Math.max(trendingMovies.length, trendingShows.length);
			for (let i = 0; i < maxLen; i++) {
				if (i < trendingMovies.length) combined.push({ ...trendingMovies[i], mediaType: 'movie' });
				if (i < trendingShows.length) combined.push({ ...trendingShows[i], mediaType: 'tv' });
			}

			const filteredHeroes = combined.filter(item => item.backdropPath).slice(0, 15);

			heroItems = filteredHeroes.map(item => {
				const matchingRequest = requests.find(
					req => req.tmdbId === item.id &&
						((req.type === 'movie' && item.mediaType === 'movie') ||
						 (req.type === 'show' && item.mediaType === 'tv'))
				);
				if (matchingRequest) {
					const status = matchingRequest.status === 'requested' ? 'pending' :
								   matchingRequest.status === 'available' ? 'approved' :
								   matchingRequest.status;
					return { ...item, requestStatus: status };
				}
				return item;
			});

			const watchlistChecks = heroItems.map(async (item) => {
				const key = getHeroWatchlistKey(item);
				try {
					const inWl = await isInWatchlist(item.id, item.mediaType === 'movie' ? 'movie' : 'tv');
					heroWatchlistStatus.set(key, inWl);
				} catch {
					heroWatchlistStatus.set(key, false);
				}
			});
			await Promise.all(watchlistChecks);
			heroWatchlistStatus = new Map(heroWatchlistStatus);

			recentMovies = movies.slice(0, 25);
			recentShows = shows.slice(0, 25);
			continueWatching = cw.slice(0, 25);
			// Get all watchlist items - we'll filter for different sections
			watchlistItems = watchlist.slice(0, 50);
			recentRequests = requests.slice(0, 10);
			collections = colls.slice(0, 20);

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

	function getHeroWatchlistKey(hero: DiscoverItem): string {
		return `${hero.mediaType}-${hero.id}`;
	}

	function isHeroInWatchlist(hero: DiscoverItem): boolean {
		return heroWatchlistStatus.get(getHeroWatchlistKey(hero)) || false;
	}

	async function handleHeroRequest() {
		const hero = currentHero();
		if (!hero || requestingHero) return;

		requestingHero = true;
		try {
			const title = hero.title || hero.name || '';
			const dateStr = hero.releaseDate || hero.firstAirDate;
			const year = dateStr ? parseInt(dateStr.substring(0, 4)) : undefined;

			await createRequest({
				type: hero.mediaType === 'movie' ? 'movie' : 'show',
				tmdbId: hero.id,
				title,
				year,
				overview: hero.overview || undefined,
				posterPath: hero.posterPath || undefined
			});
			heroItems = heroItems.map(item => {
				if (item.id === hero.id && item.mediaType === hero.mediaType) {
					return { ...item, requestStatus: 'pending' };
				}
				return item;
			});
			toast.success('Request submitted');
		} catch (e) {
			if (e instanceof Error && e.message === 'Already requested') {
				heroItems = heroItems.map(item => {
					if (item.id === hero.id && item.mediaType === hero.mediaType) {
						return { ...item, requestStatus: 'pending' };
					}
					return item;
				});
				toast.info('Already requested');
			} else {
				console.error('Failed to create request:', e);
				toast.error('Failed to create request');
			}
		} finally {
			requestingHero = false;
		}
	}

	async function handleHeroWatchlist() {
		const hero = currentHero();
		if (!hero || togglingWatchlist) return;

		togglingWatchlist = true;
		const key = getHeroWatchlistKey(hero);
		const inWatchlist = heroWatchlistStatus.get(key) || false;

		try {
			if (inWatchlist) {
				await removeFromWatchlist(hero.id, hero.mediaType === 'movie' ? 'movie' : 'tv');
				heroWatchlistStatus.set(key, false);
				watchlistItems = watchlistItems.filter(
					item => !(item.tmdbId === hero.id && item.mediaType === hero.mediaType)
				);
				toast.success('Removed from watchlist');
			} else {
				await addToWatchlist(hero.id, hero.mediaType === 'movie' ? 'movie' : 'tv');
				heroWatchlistStatus.set(key, true);
				toast.success('Added to watchlist');
			}
			heroWatchlistStatus = new Map(heroWatchlistStatus);
		} catch (e) {
			console.error('Failed to toggle watchlist:', e);
			toast.error('Failed to update watchlist');
		} finally {
			togglingWatchlist = false;
		}
	}

	function handleHeroTrailer() {
		trailerHero = currentHero(); // Capture current hero before opening
		showTrailerModal = true;
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
		return item.mediaType === 'movie' ? `/explore/movie/${item.tmdbId}` : `/explore/show/${item.tmdbId}`;
	}
</script>

<svelte:head>
	<title>Home - Outpost</title>
</svelte:head>

<div class="space-y-8">
	{#if error}
		<div class="mx-[60px] mt-6 bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-lg">
			{error}
			<button class="ml-2 underline" onclick={() => (error = null)}>Dismiss</button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center h-96">
			<div class="flex items-center gap-3">
				<div class="spinner-lg text-text-muted"></div>
				<p class="text-text-secondary">Loading...</p>
			</div>
		</div>
	{:else}
		<!-- Hero Carousel Section -->
		{#if heroItems.length > 0}
			{@const hero = currentHero()}
			{#if hero}
				<section class="relative h-[50vh] min-h-[400px] overflow-hidden">
					<!-- Backdrop image -->
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
					<div class="absolute inset-0 bg-gradient-to-r from-[#0a0a0a] via-[#0a0a0a]/70 to-transparent pointer-events-none"></div>
					<div class="absolute inset-0 bg-gradient-to-t from-[#0a0a0a] via-transparent to-[#0a0a0a]/30 pointer-events-none"></div>

					<!-- Content -->
					<div class="relative h-full flex items-end pb-16 px-[60px]">
						<div class="max-w-2xl">
							{#key heroIndex}
								<div class="animate-fade-in">
									<div class="flex items-center gap-2 mb-4">
										<span class="px-3 py-1.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
											Trending Now
										</span>
										<span class="px-3 py-1.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
											{hero.mediaType === 'movie' ? 'Movie' : 'TV Series'}
										</span>
									</div>
									<h1 class="text-4xl md:text-5xl font-bold text-text-primary mb-3">{hero.title || hero.name}</h1>
									<p class="text-text-secondary text-lg mb-2">
										{getYear(hero.releaseDate || hero.firstAirDate)}
										{#if hero.rating}
											<span class="mx-2">·</span>
											<span class="text-text-primary">{hero.rating.toFixed(1)}</span>
										{/if}
									</p>
									{#if hero.overview}
										<p class="text-text-secondary line-clamp-2 mb-6 max-w-xl">{hero.overview}</p>
									{/if}
								</div>
							{/key}

							<!-- Action buttons -->
							<div class="flex items-center gap-2">
								<!-- Details -->
								<button
									onclick={() => goto(hero.mediaType === 'movie' ? `/explore/movie/${hero.id}` : `/explore/show/${hero.id}`)}
									class="btn-icon-glass-lg"
									title="View Details"
								>
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
								</button>

								<!-- Request -->
								{#if hero.requestStatus === 'approved'}
									<button class="btn-hero-success" title="Available in Library" disabled>
										<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
										</svg>
									</button>
								{:else if hero.requestStatus === 'pending'}
									<button class="btn-hero-pending" title="Request Pending" disabled>
										<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
									</button>
								{:else}
									<button
										onclick={handleHeroRequest}
										disabled={requestingHero}
										class="btn-icon-glass-lg"
										title="Request"
									>
										{#if requestingHero}
											<div class="spinner-md text-text-muted"></div>
										{:else}
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
											</svg>
										{/if}
									</button>
								{/if}

								<!-- Watchlist -->
								<button
									onclick={handleHeroWatchlist}
									disabled={togglingWatchlist}
									class="{isHeroInWatchlist(hero) ? 'btn-hero-success' : 'btn-icon-glass-lg'}"
									title={isHeroInWatchlist(hero) ? 'In Watchlist' : 'Add to Watchlist'}
								>
									{#if togglingWatchlist}
										<div class="spinner-md"></div>
									{:else if isHeroInWatchlist(hero)}
										<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
										</svg>
									{:else}
										<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
									{/if}
								</button>

								<!-- Trailer -->
								<IconButton onclick={handleHeroTrailer} title="Watch Trailer">
									<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
										<path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
									</svg>
								</IconButton>
							</div>
						</div>

						<!-- Carousel navigation -->
						<div class="absolute bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-3">
							<button onclick={prevHero} class="carousel-nav" aria-label="Previous">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
								</svg>
							</button>
							<div class="flex items-center gap-2">
								{#each heroItems as _, i}
									<button
										onclick={() => goToHero(i)}
										class="carousel-dot transition-all {i === heroIndex ? '!bg-text-primary !w-6' : 'hover:bg-text-secondary'}"
										aria-label="Go to slide {i + 1}"
									></button>
								{/each}
							</div>
							<button onclick={nextHero} class="carousel-nav" aria-label="Next">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</button>
						</div>
					</div>
				</section>
			{/if}
		{/if}

		<!-- Content sections -->
		<div class="space-y-8 pb-8">
			<!-- Continue Watching / Your List -->
			{#if continueWatching.length > 0 || libraryWatchlist.length > 0}
				<ScrollSection title="Watchlist" linkText="View All" linkHref="/library">
					{#each continueWatching as item}
						<MediaCard
							type="landscape"
							title={item.title}
							subtitle={item.subtitle}
							imagePath={item.backdropPath ? getImageUrl(item.backdropPath) : (item.posterPath ? getImageUrl(item.posterPath) : undefined)}
							isLocal={true}
							progress={item.progressPercent}
							href={item.mediaType === 'movie' ? `/movies/${item.mediaId}` : `/tv/${item.mediaId}`}
						/>
					{/each}
					{#each libraryWatchlist as item}
						<MediaCard
							type="landscape"
							title={item.title}
							subtitle={item.year ? item.year.toString() : ''}
							imagePath={item.backdropPath ? getImageUrl(item.backdropPath) : undefined}
							isLocal={true}
							progress={item.progress}
							href={getWatchlistItemHref(item)}
						/>
					{/each}
				</ScrollSection>
			{/if}

			<!-- On Your Radar (Watchlist items not yet in library) -->
			{#if wantedItems.length > 0}
				<ScrollSection title="On Your Radar" linkText="View All" linkHref="/explore">
					{#each wantedItems as item}
						<button
							onclick={() => goto(`/explore/${item.mediaType === 'movie' ? 'movie' : 'show'}/${item.tmdbId}`)}
							class="flex-shrink-0 flex gap-3.5 p-3.5 w-[300px] bg-glass backdrop-blur-xl border border-border-subtle rounded-xl cursor-pointer transition-all hover:bg-glass-hover hover:translate-x-1 scroll-snap-align-start text-left"
						>
							<div class="w-[60px] h-[90px] rounded-lg overflow-hidden flex-shrink-0 bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44]">
								{#if item.posterPath}
									<img
										src={getTmdbImageUrl(item.posterPath, 'w154')}
										alt={item.title}
										class="w-full h-full object-cover"
									/>
								{/if}
							</div>
							<div class="flex-1 flex flex-col justify-center min-w-0">
								<div class="text-[11px] text-outpost-400 font-semibold uppercase tracking-wider mb-1">
									On Watchlist
								</div>
								<div class="text-sm font-semibold text-text-primary truncate mb-1">
									{item.title}
								</div>
								<div class="text-xs text-text-muted">
									{item.mediaType === 'movie' ? 'Movie' : 'TV Show'} • {item.year || ''}
								</div>
							</div>
						</button>
					{/each}
				</ScrollSection>
			{/if}

			<!-- Recent Requests -->
			{#if recentRequests.length > 0}
				<ScrollSection title="Recent Requests" linkText="View All" linkHref="/requests">
					{#each recentRequests as request}
						<MediaCard
							type="poster"
							title={request.title}
							subtitle={request.username || 'Unknown'}
							imagePath={request.posterPath ? getTmdbImageUrl(request.posterPath, 'w300') : undefined}
							isLocal={false}
							requestStatus={request.status === 'approved' ? 'available' : request.status === 'requested' ? 'pending' : undefined}
							href={request.type === 'movie' ? `/explore/movie/${request.tmdbId}` : `/explore/show/${request.tmdbId}`}
						/>
					{/each}
				</ScrollSection>
			{/if}

			<!-- Recently Added -->
			{#if recentlyAdded().length > 0}
				<ScrollSection title="Recently Added" linkText="View All" linkHref="/library">
					{#each recentlyAdded() as { type, item }}
						<MediaCard
							type="poster"
							title={item.title}
							subtitle={item.year?.toString()}
							imagePath={item.posterPath}
							isLocal={true}
							runtime={type === 'movie' ? (item as Movie).runtime : undefined}
							contentRating={(item as Movie | Show).contentRating}
							badge={isNew(item.addedAt) ? 'New' : undefined}
							badgeVariant={isNew(item.addedAt) ? 'new' : 'default'}
							href={type === 'movie' ? `/movies/${item.id}` : `/tv/${item.id}`}
						/>
					{/each}
				</ScrollSection>
			{/if}

			<!-- Collections -->
			{#if collections.length > 0}
				<ScrollSection title="Collections" linkText="View All" linkHref="/library?tab=collections">
					{#each collections as collection}
						<a
							href="/collections/{collection.id}"
							class="flex-shrink-0 w-[180px] group scroll-snap-align-start"
						>
							<div class="relative aspect-[2/3] rounded-xl overflow-hidden bg-bg-card border border-border-subtle mb-2">
								{#if collection.posterPath}
									<img
										src={getImageUrl(collection.posterPath)}
										alt={collection.name}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44]">
										<span class="text-5xl">📚</span>
									</div>
								{/if}
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent"></div>
								<div class="absolute bottom-0 left-0 right-0 p-3">
									<p class="text-xs text-text-muted">
										{collection.ownedCount}/{collection.itemCount} in library
									</p>
								</div>
								{#if collection.isAuto}
									<div class="absolute top-2 right-2 px-2 py-0.5 rounded text-[10px] font-medium bg-accent-primary/80 text-white">
										TMDB
									</div>
								{/if}
							</div>
							<h3 class="text-sm font-medium text-text-primary truncate group-hover:text-accent-primary transition-colors">
								{collection.name}
							</h3>
						</a>
					{/each}
				</ScrollSection>
			{/if}
		</div>
	{/if}
</div>

<!-- Trailer Modal -->
{#if trailerHero}
	<TrailerModal
		bind:open={showTrailerModal}
		tmdbId={trailerHero.id}
		mediaType={trailerHero.mediaType === 'movie' ? 'movie' : 'tv'}
		title={trailerHero.title || trailerHero.name}
	/>
{/if}
