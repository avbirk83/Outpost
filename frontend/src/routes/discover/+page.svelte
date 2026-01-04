<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getTrendingMovies,
		getPopularMovies,
		getTheatricalReleases,
		getTopRatedMovies,
		getTrendingShows,
		getPopularShows,
		getTopRatedShows,
		getUpcomingShows,
		getMovieGenres,
		getTVGenres,
		getMoviesByGenre,
		getTVByGenre,
		createRequest,
		addToWatchlist,
		type DiscoverItem,
		type Genre
	} from '$lib/api';
	import ScrollSection from '$lib/components/containers/ScrollSection.svelte';
	import MediaCard from '$lib/components/media/MediaCard.svelte';

	// Row data
	let trendingMovies: DiscoverItem[] = $state([]);
	let popularMovies: DiscoverItem[] = $state([]);
	let theatricalReleases: DiscoverItem[] = $state([]);
	let topRatedMovies: DiscoverItem[] = $state([]);
	let trendingShows: DiscoverItem[] = $state([]);
	let popularShows: DiscoverItem[] = $state([]);
	let upcomingShows: DiscoverItem[] = $state([]);
	let topRatedShows: DiscoverItem[] = $state([]);

	// Genre data
	let movieGenres: Genre[] = $state([]);
	let tvGenres: Genre[] = $state([]);
	let movieGenreContent: Map<number, DiscoverItem[]> = $state(new Map());
	let tvGenreContent: Map<number, DiscoverItem[]> = $state(new Map());


	// Priority genres to show (common ones)
	const priorityMovieGenreIds = [28, 35, 27, 878, 10749, 12]; // Action, Comedy, Horror, Sci-Fi, Romance, Adventure
	const priorityTVGenreIds = [10759, 35, 18, 80, 10765, 16]; // Action & Adventure, Comedy, Drama, Crime, Sci-Fi & Fantasy, Animation

	let loading = $state(true);
	let error: string | null = $state(null);
	let activeTab = $state<'movies' | 'shows'>('movies');

	async function loadTheatricalReleases() {
		try {
			// Fetch 3 pages for more content (~60 items)
			const [page1, page2, page3] = await Promise.all([
				getTheatricalReleases('', 1),
				getTheatricalReleases('', 2),
				getTheatricalReleases('', 3)
			]);
			theatricalReleases = [...page1.results, ...page2.results, ...page3.results];
		} catch (e) {
			console.error('Failed to load theatrical releases:', e);
		}
	}

	onMount(async () => {
		try {
			// Load base content and genres in parallel
			const [
				trendingM,
				popularM,
				theatricalM1,
				theatricalM2,
				theatricalM3,
				topRatedM,
				trendingS,
				popularS,
				upcomingS,
				topRatedS,
				movieGenresRes,
				tvGenresRes
			] = await Promise.all([
				getTrendingMovies(),
				getPopularMovies(),
				getTheatricalReleases('', 1),
				getTheatricalReleases('', 2),
				getTheatricalReleases('', 3),
				getTopRatedMovies(),
				getTrendingShows(),
				getPopularShows(),
				getUpcomingShows(),
				getTopRatedShows(),
				getMovieGenres().catch(() => ({ genres: [] })),
				getTVGenres().catch(() => ({ genres: [] }))
			]);

			trendingMovies = trendingM.results;
			popularMovies = popularM.results;
			theatricalReleases = [...theatricalM1.results, ...theatricalM2.results, ...theatricalM3.results];
			topRatedMovies = topRatedM.results;
			trendingShows = trendingS.results;
			popularShows = popularS.results;
			upcomingShows = upcomingS.results;
			topRatedShows = topRatedS.results;
			movieGenres = movieGenresRes.genres;
			tvGenres = tvGenresRes.genres;

			// Select heroes for each tab (top 10 with backdrops)
			movieHeroes = trendingMovies.filter(m => m.backdropPath).slice(0, 10);
			showHeroes = trendingShows.filter(s => s.backdropPath).slice(0, 10);

			// Load content for priority genres
			await loadGenreContent();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load discover content';
		} finally {
			loading = false;
		}
	});

	async function loadGenreContent() {
		// Load movie genres
		const moviePromises = priorityMovieGenreIds.map(async (id) => {
			try {
				const result = await getMoviesByGenre(id);
				return { id, items: result.results };
			} catch {
				return { id, items: [] };
			}
		});

		// Load TV genres
		const tvPromises = priorityTVGenreIds.map(async (id) => {
			try {
				const result = await getTVByGenre(id);
				return { id, items: result.results };
			} catch {
				return { id, items: [] };
			}
		});

		const [movieResults, tvResults] = await Promise.all([
			Promise.all(moviePromises),
			Promise.all(tvPromises)
		]);

		const newMovieContent = new Map<number, DiscoverItem[]>();
		for (const { id, items } of movieResults) {
			if (items.length > 0) {
				newMovieContent.set(id, items);
			}
		}
		movieGenreContent = newMovieContent;

		const newTVContent = new Map<number, DiscoverItem[]>();
		for (const { id, items } of tvResults) {
			if (items.length > 0) {
				newTVContent.set(id, items);
			}
		}
		tvGenreContent = newTVContent;
	}

	function getGenreName(genreId: number, type: 'movie' | 'tv'): string {
		const genres = type === 'movie' ? movieGenres : tvGenres;
		return genres.find(g => g.id === genreId)?.name || 'Unknown';
	}

	async function handleRequest(e: MouseEvent, item: DiscoverItem) {
		e.preventDefault();
		e.stopPropagation();

		try {
			await createRequest({
				type: item.type === 'movie' ? 'movie' : 'show',
				tmdbId: item.id,
				title: item.title || item.name || '',
				year: item.releaseDate ? parseInt(item.releaseDate.substring(0, 4)) : undefined,
				overview: item.overview,
				posterPath: item.posterPath
			});

			// Update local state
			const updateItems = (items: DiscoverItem[]) =>
				items.map(i => i.id === item.id ? { ...i, requested: true } : i);

			trendingMovies = updateItems(trendingMovies);
			popularMovies = updateItems(popularMovies);
			theatricalReleases = updateItems(theatricalReleases);
			topRatedMovies = updateItems(topRatedMovies);
			trendingShows = updateItems(trendingShows);
			popularShows = updateItems(popularShows);
			upcomingShows = updateItems(upcomingShows);
			topRatedShows = updateItems(topRatedShows);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create request';
		}
	}

	function getImageUrl(path: string | undefined): string {
		if (!path) return '';
		return `https://image.tmdb.org/t/p/w300${path}`;
	}

	function getBackdropUrl(path: string | undefined): string {
		if (!path) return '';
		return `https://image.tmdb.org/t/p/w1280${path}`;
	}

	// Hero carousel items (multiple per tab)
	let movieHeroes: DiscoverItem[] = $state([]);
	let showHeroes: DiscoverItem[] = $state([]);
	let movieHeroIndex = $state(0);
	let showHeroIndex = $state(0);

	// Current hero based on active tab
	let currentHeroes = $derived(activeTab === 'movies' ? movieHeroes : showHeroes);
	let currentHeroIndex = $derived(activeTab === 'movies' ? movieHeroIndex : showHeroIndex);
	let currentHero = $derived(currentHeroes[currentHeroIndex] || null);

	function nextHero() {
		if (activeTab === 'movies') {
			movieHeroIndex = (movieHeroIndex + 1) % movieHeroes.length;
		} else {
			showHeroIndex = (showHeroIndex + 1) % showHeroes.length;
		}
	}

	function prevHero() {
		if (activeTab === 'movies') {
			movieHeroIndex = (movieHeroIndex - 1 + movieHeroes.length) % movieHeroes.length;
		} else {
			showHeroIndex = (showHeroIndex - 1 + showHeroes.length) % showHeroes.length;
		}
	}

	function goToHero(index: number) {
		if (activeTab === 'movies') {
			movieHeroIndex = index;
		} else {
			showHeroIndex = index;
		}
	}

	function getDetailUrl(item: DiscoverItem): string {
		if (item.inLibrary) {
			return item.type === 'movie' ? `/movies/${item.libraryId}` : `/tv/${item.libraryId}`;
		}
		return item.type === 'movie' ? `/discover/movie/${item.id}` : `/discover/show/${item.id}`;
	}
</script>

<svelte:head>
	<title>Discover - Outpost</title>
</svelte:head>

<div class="space-y-8 -mt-22 -mx-6">
	{#if loading}
		<div class="flex items-center justify-center h-96">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading discover content...</p>
			</div>
		</div>
	{:else}
		<!-- Hero Carousel Section - changes based on active tab -->
		{#if currentHero}
			<section class="relative h-[45vh] min-h-[380px] overflow-hidden">
				<!-- Backdrop image with fade transition -->
				{#key currentHero.id}
					<img
						src={getBackdropUrl(currentHero.backdropPath)}
						alt={currentHero.title || currentHero.name}
						class="absolute inset-0 w-full h-full object-cover animate-fade-in pointer-events-none"
						style="object-position: center 25%;"
						draggable="false"
					/>
				{/key}

				<!-- Gradient overlays (matching home page) -->
				<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/80 to-transparent pointer-events-none"></div>
				<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/30 pointer-events-none"></div>

				<!-- Content -->
				<div class="relative h-full flex items-end pb-12 px-[60px]">
					<div class="max-w-2xl">
						{#key currentHero.id}
							<div class="animate-fade-in">
								<div class="flex items-center gap-2 mb-4">
									<span class="px-3 py-1.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
										Trending
									</span>
									<span class="px-3 py-1.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary">
										{activeTab === 'movies' ? 'Movie' : 'TV Series'}
									</span>
								</div>
								<h1 class="text-4xl md:text-5xl font-bold text-white mb-3">{currentHero.title || currentHero.name}</h1>
								<p class="text-text-secondary text-lg mb-2">
									{currentHero.releaseDate?.substring(0, 4) || ''}
									{#if currentHero.voteAverage}
										<span class="mx-2">•</span>
										<span class="text-white">{currentHero.voteAverage.toFixed(1)}</span>
									{/if}
								</p>
								{#if currentHero.overview}
									<p class="text-text-secondary line-clamp-2 mb-6 max-w-xl">{currentHero.overview}</p>
								{/if}
							</div>
						{/key}
						<!-- Circular action buttons -->
						<div class="flex items-center gap-2">
							<!-- Details -->
							<a
								href={activeTab === 'movies' ? `/discover/movie/${currentHero.id}` : `/discover/show/${currentHero.id}`}
								class="w-11 h-11 rounded-full bg-glass border border-border-subtle text-text-primary flex items-center justify-center hover:bg-glass-hover transition-all"
								title="View Details"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							</a>
							<!-- Request -->
							{#if currentHero.inLibrary}
								<a
									href={activeTab === 'movies' ? `/movies/${currentHero.libraryId}` : `/tv/${currentHero.libraryId}`}
									class="w-11 h-11 rounded-full bg-green-600/80 border border-green-500/50 text-white flex items-center justify-center hover:bg-green-500/80 transition-all"
									title="In Library"
								>
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								</a>
							{:else if currentHero.requested || currentHero.requestStatus}
								<button
									class="w-11 h-11 rounded-full bg-yellow-600/80 border border-yellow-500/50 text-white flex items-center justify-center transition-all cursor-default"
									title="Request Pending"
									disabled
								>
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
								</button>
							{:else}
								<button
									onclick={(e) => handleRequest(e, currentHero)}
									class="w-11 h-11 rounded-full bg-glass border border-border-subtle text-text-primary flex items-center justify-center hover:bg-glass-hover transition-all"
									title="Request"
								>
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
									</svg>
								</button>
							{/if}
						</div>
					</div>

					<!-- Carousel navigation - center bottom with arrows -->
					<div class="absolute bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-3">
						<button
							onclick={prevHero}
							class="p-1.5 rounded-full bg-glass hover:bg-glass-hover text-text-primary transition-colors border border-border-subtle"
							aria-label="Previous"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
							</svg>
						</button>
						<div class="flex items-center gap-2">
							{#each currentHeroes as _, i}
								<button
									onclick={() => goToHero(i)}
									class="w-2 h-2 rounded-full transition-all {i === currentHeroIndex ? 'bg-text-primary w-6' : 'bg-text-muted hover:bg-text-secondary'}"
									aria-label="Go to slide {i + 1}"
								></button>
							{/each}
						</div>
						<button
							onclick={nextHero}
							class="p-1.5 rounded-full bg-glass hover:bg-glass-hover text-text-primary transition-colors border border-border-subtle"
							aria-label="Next"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
							</svg>
						</button>
					</div>
				</div>
			</section>
		{/if}

		<!-- Content sections -->
		<div class="space-y-8 pb-8">
			{#if error}
				<div class="mx-[60px] bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
					<span>{error}</span>
					<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)} title="Dismiss">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			{/if}

			<!-- Filter Pills -->
			<div class="mx-[60px] flex flex-wrap gap-2">
				<!-- Media type pills -->
				<button
					onclick={() => activeTab = 'movies'}
					class="px-4 py-2.5 text-sm font-medium transition-all rounded-full min-h-[44px] flex items-center
						{activeTab === 'movies'
							? 'bg-white text-black border border-white'
							: 'bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
				>
					Movies
				</button>
				<button
					onclick={() => activeTab = 'shows'}
					class="px-4 py-2.5 text-sm font-medium transition-all rounded-full min-h-[44px] flex items-center
						{activeTab === 'shows'
							? 'bg-white text-black border border-white'
							: 'bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
				>
					TV Shows
				</button>

				<!-- Separator -->
				<div class="w-px h-8 bg-border-subtle self-center mx-1"></div>

				<!-- Genre pills -->
				{#if activeTab === 'movies'}
					{#each priorityMovieGenreIds as genreId}
						{@const genre = movieGenres.find(g => g.id === genreId)}
						{#if genre && movieGenreContent.get(genreId)?.length}
							<button
								onclick={() => document.getElementById(`genre-movie-${genreId}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
								class="px-3 py-1.5 text-xs font-medium transition-all rounded-full bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary"
							>
								{genre.name}
							</button>
						{/if}
					{/each}
				{:else}
					{#each priorityTVGenreIds as genreId}
						{@const genre = tvGenres.find(g => g.id === genreId)}
						{#if genre && tvGenreContent.get(genreId)?.length}
							<button
								onclick={() => document.getElementById(`genre-tv-${genreId}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
								class="px-3 py-1.5 text-xs font-medium transition-all rounded-full bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary"
							>
								{genre.name}
							</button>
						{/if}
					{/each}
				{/if}
			</div>

			<!-- Movies Tab Content -->
			{#if activeTab === 'movies'}
				<!-- Trending Movies -->
				{#if trendingMovies.length > 0}
					<ScrollSection title="Trending Movies">
						{#each trendingMovies.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="movie"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Popular Movies -->
				{#if popularMovies.length > 0}
					<ScrollSection title="Popular Movies">
						{#each popularMovies.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="movie"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Coming to Theaters -->
				<ScrollSection title="Coming to Theaters">
					{#if theatricalReleases.length > 0}
						{#each theatricalReleases.slice(0, 40) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="movie"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					{:else}
						<div class="py-8 text-center text-text-muted w-full">
							No upcoming theatrical releases found
						</div>
					{/if}
				</ScrollSection>

				<!-- Top Rated Movies -->
				{#if topRatedMovies.length > 0}
					<ScrollSection title="Top Rated Movies">
						{#each topRatedMovies.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="movie"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Genre Rows -->
				{#each priorityMovieGenreIds as genreId}
					{#if movieGenreContent.get(genreId)?.length}
						<div id="genre-movie-{genreId}" class="scroll-mt-24">
							<ScrollSection title={getGenreName(genreId, 'movie')}>
								{#each (movieGenreContent.get(genreId) || []).slice(0, 25) as item}
									<MediaCard
										type="poster"
										title={item.title || item.name || ''}
										subtitle={item.releaseDate?.substring(0, 4)}
										imagePath={item.posterPath}
										isLocal={false}
										mediaType="movie"
										inLibrary={item.inLibrary}
										requested={item.requested}
										href={getDetailUrl(item)}
									/>
								{/each}
							</ScrollSection>
						</div>
					{/if}
				{/each}
			{/if}

			<!-- Shows Tab Content -->
			{#if activeTab === 'shows'}
				<!-- Trending Shows -->
				{#if trendingShows.length > 0}
					<ScrollSection title="Trending TV Shows">
						{#each trendingShows.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="tv"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Popular Shows -->
				{#if popularShows.length > 0}
					<ScrollSection title="Popular TV Shows">
						{#each popularShows.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="tv"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Upcoming Shows -->
				{#if upcomingShows.length > 0}
					<ScrollSection title="Upcoming TV Shows">
						{#each upcomingShows.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="tv"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Top Rated Shows -->
				{#if topRatedShows.length > 0}
					<ScrollSection title="Top Rated TV Shows">
						{#each topRatedShows.slice(0, 25) as item}
							<MediaCard
								type="poster"
								title={item.title || item.name || ''}
								subtitle={item.releaseDate?.substring(0, 4)}
								imagePath={item.posterPath}
								isLocal={false}
								mediaType="tv"
								inLibrary={item.inLibrary}
								requested={item.requested}
								href={getDetailUrl(item)}
							/>
						{/each}
					</ScrollSection>
				{/if}

				<!-- Genre Rows -->
				{#each priorityTVGenreIds as genreId}
					{#if tvGenreContent.get(genreId)?.length}
						<div id="genre-tv-{genreId}" class="scroll-mt-24">
							<ScrollSection title={getGenreName(genreId, 'tv')}>
								{#each (tvGenreContent.get(genreId) || []).slice(0, 25) as item}
									<MediaCard
										type="poster"
										title={item.title || item.name || ''}
										subtitle={item.releaseDate?.substring(0, 4)}
										imagePath={item.posterPath}
										isLocal={false}
										mediaType="tv"
										inLibrary={item.inLibrary}
										requested={item.requested}
										href={getDetailUrl(item)}
									/>
								{/each}
							</ScrollSection>
						</div>
					{/if}
				{/each}
			{/if}
		</div>
	{/if}
</div>
