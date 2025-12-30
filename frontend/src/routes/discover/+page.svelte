<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getTrendingMovies,
		getPopularMovies,
		getUpcomingMovies,
		getTopRatedMovies,
		getTrendingShows,
		getPopularShows,
		getTopRatedShows,
		getMovieGenres,
		getTVGenres,
		getMoviesByGenre,
		getTVByGenre,
		createRequest,
		type DiscoverItem,
		type Genre
	} from '$lib/api';
	import MediaRow from '$lib/components/MediaRow.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';

	// Row data
	let trendingMovies: DiscoverItem[] = $state([]);
	let popularMovies: DiscoverItem[] = $state([]);
	let upcomingMovies: DiscoverItem[] = $state([]);
	let topRatedMovies: DiscoverItem[] = $state([]);
	let trendingShows: DiscoverItem[] = $state([]);
	let popularShows: DiscoverItem[] = $state([]);
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

	onMount(async () => {
		try {
			// Load base content and genres in parallel
			const [
				trendingM,
				popularM,
				upcomingM,
				topRatedM,
				trendingS,
				popularS,
				topRatedS,
				movieGenresRes,
				tvGenresRes
			] = await Promise.all([
				getTrendingMovies(),
				getPopularMovies(),
				getUpcomingMovies(),
				getTopRatedMovies(),
				getTrendingShows(),
				getPopularShows(),
				getTopRatedShows(),
				getMovieGenres().catch(() => ({ genres: [] })),
				getTVGenres().catch(() => ({ genres: [] }))
			]);

			trendingMovies = trendingM.results;
			popularMovies = popularM.results;
			upcomingMovies = upcomingM.results;
			topRatedMovies = topRatedM.results;
			trendingShows = trendingS.results;
			popularShows = popularS.results;
			topRatedShows = topRatedS.results;
			movieGenres = movieGenresRes.genres;
			tvGenres = tvGenresRes.genres;

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
			upcomingMovies = updateItems(upcomingMovies);
			topRatedMovies = updateItems(topRatedMovies);
			trendingShows = updateItems(trendingShows);
			popularShows = updateItems(popularShows);
			topRatedShows = updateItems(topRatedShows);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create request';
		}
	}

	function getImageUrl(path: string | undefined): string {
		if (!path) return '';
		return `https://image.tmdb.org/t/p/w300${path}`;
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

<div class="space-y-8">
	<!-- Header with tabs -->
	<div class="flex flex-col gap-4">
		<h1 class="text-3xl font-bold text-text-primary">Discover</h1>
		<p class="text-text-secondary">Find new movies and shows to add to your library</p>

		<!-- Tab bar -->
		<div class="flex gap-2">
			<button
				onclick={() => activeTab = 'movies'}
				class="px-4 py-2 text-sm font-medium transition-all rounded-xl
					{activeTab === 'movies'
						? 'liquid-glass text-white'
						: 'text-white/50 hover:text-white hover:bg-white/5'}"
			>
				Movies
			</button>
			<button
				onclick={() => activeTab = 'shows'}
				class="px-4 py-2 text-sm font-medium transition-all rounded-xl
					{activeTab === 'shows'
						? 'liquid-glass text-white'
						: 'text-white/50 hover:text-white hover:bg-white/5'}"
			>
				TV Shows
			</button>
		</div>
	</div>

	{#if error}
		<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
			<span>{error}</span>
			<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)} title="Dismiss">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center h-64">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white-400 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading discover content...</p>
			</div>
		</div>
	{:else}
		<!-- Movies Tab -->
		{#if activeTab === 'movies'}
			<div class="space-y-8">
				<!-- Trending Movies -->
				{#if trendingMovies.length > 0}
					<MediaRow title="Trending Movies">
						{#each trendingMovies.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType={item.type === 'movie' ? 'movie' : 'series'}
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Popular Movies -->
				{#if popularMovies.length > 0}
					<MediaRow title="Popular Movies">
						{#each popularMovies.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType={item.type === 'movie' ? 'movie' : 'series'}
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Upcoming Movies -->
				{#if upcomingMovies.length > 0}
					<MediaRow title="Upcoming Movies">
						{#each upcomingMovies.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType={item.type === 'movie' ? 'movie' : 'series'}
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Top Rated Movies -->
				{#if topRatedMovies.length > 0}
					<MediaRow title="Top Rated Movies">
						{#each topRatedMovies.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType={item.type === 'movie' ? 'movie' : 'series'}
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Genre Rows -->
				{#each priorityMovieGenreIds as genreId}
					{#if movieGenreContent.get(genreId)?.length}
						<MediaRow title={getGenreName(genreId, 'movie')}>
							{#each (movieGenreContent.get(genreId) || []).slice(0, 12) as item}
								<div class="flex-shrink-0 w-32 sm:w-36">
									<PosterCard
										href={getDetailUrl(item)}
										title={item.title || item.name || ''}
										subtitle={item.releaseDate?.substring(0, 4)}
										posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
										mediaType={item.type === 'movie' ? 'movie' : 'series'}
										inLibrary={item.inLibrary}
										requested={item.requested}
										requestStatus={item.requestStatus}
										onRequest={(e) => handleRequest(e, item)}
									/>
								</div>
							{/each}
						</MediaRow>
					{/if}
				{/each}
			</div>
		{/if}

		<!-- Shows Tab -->
		{#if activeTab === 'shows'}
			<div class="space-y-8">
				<!-- Trending Shows -->
				{#if trendingShows.length > 0}
					<MediaRow title="Trending TV Shows">
						{#each trendingShows.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType="series"
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Popular Shows -->
				{#if popularShows.length > 0}
					<MediaRow title="Popular TV Shows">
						{#each popularShows.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType="series"
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Top Rated Shows -->
				{#if topRatedShows.length > 0}
					<MediaRow title="Top Rated TV Shows">
						{#each topRatedShows.slice(0, 12) as item}
							<div class="flex-shrink-0 w-32 sm:w-36">
								<PosterCard
									href={getDetailUrl(item)}
									title={item.title || item.name || ''}
									subtitle={item.releaseDate?.substring(0, 4)}
									posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
									mediaType="series"
									inLibrary={item.inLibrary}
									requested={item.requested}
									requestStatus={item.requestStatus}
									onRequest={(e) => handleRequest(e, item)}
								/>
							</div>
						{/each}
					</MediaRow>
				{/if}

				<!-- Genre Rows -->
				{#each priorityTVGenreIds as genreId}
					{#if tvGenreContent.get(genreId)?.length}
						<MediaRow title={getGenreName(genreId, 'tv')}>
							{#each (tvGenreContent.get(genreId) || []).slice(0, 12) as item}
								<div class="flex-shrink-0 w-32 sm:w-36">
									<PosterCard
										href={getDetailUrl(item)}
										title={item.title || item.name || ''}
										subtitle={item.releaseDate?.substring(0, 4)}
										posterUrl={item.posterPath ? getImageUrl(item.posterPath) : undefined}
										mediaType="series"
										inLibrary={item.inLibrary}
										requested={item.requested}
										requestStatus={item.requestStatus}
										onRequest={(e) => handleRequest(e, item)}
									/>
								</div>
							{/each}
						</MediaRow>
					{/if}
				{/each}
			</div>
		{/if}
	{/if}
</div>
