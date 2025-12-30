<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		getMovies,
		getShows,
		getArtists,
		getAlbums,
		getBooks,
		getImageUrl,
		type Movie,
		type Show,
		type Artist,
		type Album,
		type Book
	} from '$lib/api';
	import PosterCard from '$lib/components/PosterCard.svelte';

	// Tab state from URL
	type TabType = 'movies' | 'tv' | 'music' | 'books';
	let activeTab = $state<TabType>('movies');

	// Data states
	let movies: Movie[] = $state([]);
	let shows: Show[] = $state([]);
	let artists: Artist[] = $state([]);
	let albums: Album[] = $state([]);
	let books: Book[] = $state([]);

	// Loading states per tab
	let loadingMovies = $state(false);
	let loadingShows = $state(false);
	let loadingMusic = $state(false);
	let loadingBooks = $state(false);
	let loadedTabs = $state<Set<TabType>>(new Set());

	let error: string | null = $state(null);

	// Filter/sort states
	let movieSearch = $state('');
	let movieSort = $state<'title' | 'year' | 'rating' | 'added'>('added');
	let movieViewMode = $state<'grid' | 'rows'>('grid');
	let movieGenreFilter = $state<string>('all');
	let movieContentRatings = $state<Set<string>>(new Set());
	let movieRuntimeMin = $state(0);
	let movieRuntimeMax = $state(300);
	let movieRatingMin = $state(0);
	let movieRatingMax = $state(10);
	let movieFiltersExpanded = $state(false);

	let showSearch = $state('');
	let showSort = $state<'title' | 'year' | 'rating' | 'added'>('added');
	let showStatus = $state<'all' | 'continuing' | 'ended'>('all');
	let showViewMode = $state<'grid' | 'rows'>('grid');
	let showGenreFilter = $state<string>('all');
	let showContentRatings = $state<Set<string>>(new Set());
	let showRatingMin = $state(0);
	let showRatingMax = $state(10);
	let showFiltersExpanded = $state(false);

	let musicView = $state<'artists' | 'albums'>('artists');

	let bookSearch = $state('');

	// Content rating options
	const movieContentRatingOptions = ['G', 'PG', 'PG-13', 'R', 'NC-17', 'NR'];
	const showContentRatingOptions = ['TV-Y', 'TV-Y7', 'TV-G', 'TV-PG', 'TV-14', 'TV-MA', 'NR'];

	function toggleMovieContentRating(rating: string) {
		const newSet = new Set(movieContentRatings);
		if (newSet.has(rating)) {
			newSet.delete(rating);
		} else {
			newSet.add(rating);
		}
		movieContentRatings = newSet;
	}

	function toggleShowContentRating(rating: string) {
		const newSet = new Set(showContentRatings);
		if (newSet.has(rating)) {
			newSet.delete(rating);
		} else {
			newSet.add(rating);
		}
		showContentRatings = newSet;
	}

	function clearMovieFilters() {
		movieGenreFilter = 'all';
		movieContentRatings = new Set();
		movieRuntimeMin = 0;
		movieRuntimeMax = 300;
		movieRatingMin = 0;
		movieRatingMax = 10;
		movieSearch = '';
	}

	function clearShowFilters() {
		showGenreFilter = 'all';
		showContentRatings = new Set();
		showRatingMin = 0;
		showRatingMax = 10;
		showSearch = '';
		showStatus = 'all';
	}

	const hasActiveMovieFilters = $derived(() => {
		return movieGenreFilter !== 'all' ||
			movieContentRatings.size > 0 ||
			movieRuntimeMin > 0 ||
			movieRuntimeMax < 300 ||
			movieRatingMin > 0 ||
			movieRatingMax < 10;
	});

	const hasActiveShowFilters = $derived(() => {
		return showGenreFilter !== 'all' ||
			showContentRatings.size > 0 ||
			showRatingMin > 0 ||
			showRatingMax < 10 ||
			showStatus !== 'all';
	});

	// Parse genres from JSON string
	function parseGenres(genresStr: string | undefined): string[] {
		if (!genresStr) return [];
		try {
			return JSON.parse(genresStr);
		} catch {
			return [];
		}
	}

	// Group movies by genre
	const moviesByGenre = $derived(() => {
		const genreMap = new Map<string, Movie[]>();
		for (const movie of movies) {
			const genres = parseGenres(movie.genres);
			if (genres.length === 0) {
				const uncategorized = genreMap.get('Uncategorized') || [];
				uncategorized.push(movie);
				genreMap.set('Uncategorized', uncategorized);
			} else {
				for (const genre of genres) {
					const list = genreMap.get(genre) || [];
					list.push(movie);
					genreMap.set(genre, list);
				}
			}
		}
		// Sort genres by count, but keep common genres first
		const priorityGenres = ['Action', 'Comedy', 'Drama', 'Horror', 'Thriller', 'Science Fiction', 'Romance', 'Adventure', 'Animation', 'Documentary'];
		const sortedEntries = Array.from(genreMap.entries()).sort((a, b) => {
			const aIdx = priorityGenres.indexOf(a[0]);
			const bIdx = priorityGenres.indexOf(b[0]);
			if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx;
			if (aIdx !== -1) return -1;
			if (bIdx !== -1) return 1;
			return b[1].length - a[1].length;
		});
		return sortedEntries;
	});

	// Group shows by genre
	const showsByGenre = $derived(() => {
		const genreMap = new Map<string, Show[]>();
		for (const show of shows) {
			const genres = parseGenres(show.genres);
			if (genres.length === 0) {
				const uncategorized = genreMap.get('Uncategorized') || [];
				uncategorized.push(show);
				genreMap.set('Uncategorized', uncategorized);
			} else {
				for (const genre of genres) {
					const list = genreMap.get(genre) || [];
					list.push(show);
					genreMap.set(genre, list);
				}
			}
		}
		const priorityGenres = ['Action & Adventure', 'Comedy', 'Drama', 'Crime', 'Mystery', 'Sci-Fi & Fantasy', 'Documentary', 'Animation', 'Reality'];
		const sortedEntries = Array.from(genreMap.entries()).sort((a, b) => {
			const aIdx = priorityGenres.indexOf(a[0]);
			const bIdx = priorityGenres.indexOf(b[0]);
			if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx;
			if (aIdx !== -1) return -1;
			if (bIdx !== -1) return 1;
			return b[1].length - a[1].length;
		});
		return sortedEntries;
	});

	// Read tab from URL on mount
	onMount(() => {
		const urlTab = $page.url.searchParams.get('tab') as TabType | null;
		if (urlTab && ['movies', 'tv', 'music', 'books'].includes(urlTab)) {
			activeTab = urlTab;
		}
		loadTabData(activeTab);
	});

	// Update URL when tab changes
	function setTab(tab: TabType) {
		activeTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set('tab', tab);
		goto(url.pathname + url.search, { replaceState: true, keepFocus: true });
		loadTabData(tab);
	}

	// Lazy load data for tab
	async function loadTabData(tab: TabType) {
		if (loadedTabs.has(tab)) return;

		try {
			switch (tab) {
				case 'movies':
					loadingMovies = true;
					movies = await getMovies();
					break;
				case 'tv':
					loadingShows = true;
					shows = await getShows();
					break;
				case 'music':
					loadingMusic = true;
					const [a, al] = await Promise.all([getArtists(), getAlbums()]);
					artists = a;
					albums = al;
					break;
				case 'books':
					loadingBooks = true;
					books = await getBooks();
					break;
			}
			loadedTabs.add(tab);
		} catch (e) {
			error = e instanceof Error ? e.message : `Failed to load ${tab}`;
		} finally {
			loadingMovies = false;
			loadingShows = false;
			loadingMusic = false;
			loadingBooks = false;
		}
	}

	// Get unique genres from movies
	const allMovieGenres = $derived(() => {
		const genres = new Set<string>();
		for (const movie of movies) {
			for (const g of parseGenres(movie.genres)) {
				genres.add(g);
			}
		}
		return Array.from(genres).sort();
	});

	// Get unique genres from shows
	const allShowGenres = $derived(() => {
		const genres = new Set<string>();
		for (const show of shows) {
			for (const g of parseGenres(show.genres)) {
				genres.add(g);
			}
		}
		return Array.from(genres).sort();
	});

	// Filtered/sorted data
	const filteredMovies = $derived(() => {
		let result = [...movies];
		// Genre filter
		if (movieGenreFilter !== 'all') {
			result = result.filter(m => parseGenres(m.genres).includes(movieGenreFilter));
		}
		// Content rating filter
		if (movieContentRatings.size > 0) {
			result = result.filter(m => {
				const rating = m.contentRating || 'NR';
				return movieContentRatings.has(rating) || (rating === '' && movieContentRatings.has('NR'));
			});
		}
		// Runtime filter
		if (movieRuntimeMin > 0 || movieRuntimeMax < 300) {
			result = result.filter(m => {
				const runtime = m.runtime || 0;
				return runtime >= movieRuntimeMin && runtime <= movieRuntimeMax;
			});
		}
		// Rating filter
		if (movieRatingMin > 0 || movieRatingMax < 10) {
			result = result.filter(m => {
				const rating = m.rating || 0;
				return rating >= movieRatingMin && rating <= movieRatingMax;
			});
		}
		// Search filter
		if (movieSearch) {
			const query = movieSearch.toLowerCase();
			result = result.filter(m =>
				m.title.toLowerCase().includes(query) ||
				(m.year && m.year.toString().includes(query))
			);
		}
		switch (movieSort) {
			case 'title':
				result.sort((a, b) => a.title.localeCompare(b.title));
				break;
			case 'year':
				result.sort((a, b) => (b.year || 0) - (a.year || 0));
				break;
			case 'rating':
				result.sort((a, b) => (b.rating || 0) - (a.rating || 0));
				break;
			case 'added':
				result.sort((a, b) => b.id - a.id);
				break;
		}
		return result;
	});

	const filteredShows = $derived(() => {
		let result = [...shows];
		// Genre filter
		if (showGenreFilter !== 'all') {
			result = result.filter(s => parseGenres(s.genres).includes(showGenreFilter));
		}
		// Content rating filter
		if (showContentRatings.size > 0) {
			result = result.filter(s => {
				const rating = s.contentRating || 'NR';
				return showContentRatings.has(rating) || (rating === '' && showContentRatings.has('NR'));
			});
		}
		// Rating filter
		if (showRatingMin > 0 || showRatingMax < 10) {
			result = result.filter(s => {
				const rating = s.rating || 0;
				return rating >= showRatingMin && rating <= showRatingMax;
			});
		}
		// Search filter
		if (showSearch) {
			const query = showSearch.toLowerCase();
			result = result.filter(s =>
				s.title.toLowerCase().includes(query) ||
				(s.year && s.year.toString().includes(query)) ||
				(s.network && s.network.toLowerCase().includes(query))
			);
		}
		// Status filter
		if (showStatus === 'continuing') {
			result = result.filter(s =>
				s.status?.toLowerCase() === 'returning series' ||
				s.status?.toLowerCase() === 'in production'
			);
		} else if (showStatus === 'ended') {
			result = result.filter(s =>
				s.status?.toLowerCase() === 'ended' ||
				s.status?.toLowerCase() === 'canceled'
			);
		}
		switch (showSort) {
			case 'title':
				result.sort((a, b) => a.title.localeCompare(b.title));
				break;
			case 'year':
				result.sort((a, b) => (b.year || 0) - (a.year || 0));
				break;
			case 'rating':
				result.sort((a, b) => (b.rating || 0) - (a.rating || 0));
				break;
			case 'added':
				result.sort((a, b) => b.id - a.id);
				break;
		}
		return result;
	});

	const filteredBooks = $derived(() => {
		if (!bookSearch) return books;
		return books.filter(b =>
			b.title.toLowerCase().includes(bookSearch.toLowerCase()) ||
			(b.author && b.author.toLowerCase().includes(bookSearch.toLowerCase()))
		);
	});

	function getStatusColor(status: string | undefined): string {
		switch (status?.toLowerCase()) {
			case 'returning series':
			case 'in production':
				return 'bg-green-500/80';
			case 'ended':
			case 'canceled':
				return 'bg-white/50';
			default:
				return 'bg-white/30';
		}
	}

	function getFormatColor(format: string): string {
		switch (format.toLowerCase()) {
			case 'epub':
				return 'bg-green-900 text-green-300';
			case 'pdf':
				return 'bg-white/20 text-white';
			case 'mobi':
			case 'azw':
			case 'azw3':
				return 'bg-yellow-900 text-yellow-300';
			case 'cbz':
			case 'cbr':
				return 'bg-purple-900 text-purple-300';
			default:
				return 'bg-gray-700 text-gray-300';
		}
	}

	const tabs = $derived([
		{ id: 'movies' as TabType, label: 'Movies', count: movies.length },
		{ id: 'tv' as TabType, label: 'TV Shows', count: shows.length },
		{ id: 'music' as TabType, label: 'Music', count: artists.length + albums.length },
		{ id: 'books' as TabType, label: 'Books', count: books.length },
	]);
</script>

<svelte:head>
	<title>Library - Outpost</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header with tabs -->
	<div class="flex flex-col gap-4">
		<h1 class="text-3xl font-bold text-text-primary">Library</h1>

		<!-- Tab bar -->
		<div class="flex gap-2">
			{#each tabs as tab}
				<button
					onclick={() => setTab(tab.id)}
					class="px-4 py-2 text-sm font-medium transition-all rounded-xl
						{activeTab === tab.id
							? 'liquid-glass text-white'
							: 'text-white/50 hover:text-white hover:bg-white/5'}"
				>
					{tab.label}
					{#if loadedTabs.has(tab.id) && tab.count > 0}
						<span class="ml-1.5 text-xs text-white/40">({tab.count})</span>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	{#if error}
		<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-lg">
			{error}
			<button class="ml-2 underline" onclick={() => (error = null)}>Dismiss</button>
		</div>
	{/if}

	<!-- Movies Tab -->
	{#if activeTab === 'movies'}
		<div class="space-y-6">
			<!-- Controls -->
			<div class="space-y-4">
				<div class="flex flex-wrap items-center gap-3">
					{#if movieViewMode === 'grid'}
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
							</svg>
							<input
								type="text"
								placeholder="Search movies..."
								bind:value={movieSearch}
								class="liquid-input pl-10 pr-4 py-2 w-48 sm:w-64"
							/>
						</div>
						<select
							bind:value={movieSort}
							class="liquid-select px-4 py-2"
						>
							<option value="added">Recently Added</option>
							<option value="title">Title A-Z</option>
							<option value="year">Year</option>
							<option value="rating">Rating</option>
						</select>
						<button
							onclick={() => movieFiltersExpanded = !movieFiltersExpanded}
							class="liquid-btn px-4 py-2 flex items-center gap-2 {hasActiveMovieFilters() ? '!border-t-white/20 !border-l-white/10 !bg-white/5' : ''}"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
							</svg>
							Filters
							{#if hasActiveMovieFilters()}
								<span class="w-2 h-2 rounded-full bg-white/50"></span>
							{/if}
							<svg class="w-4 h-4 transition-transform {movieFiltersExpanded ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
							</svg>
						</button>
					{/if}

					<!-- View toggle -->
					<div class="flex gap-1 ml-auto">
						<button
							onclick={() => movieViewMode = 'grid'}
							class="liquid-btn-icon {movieViewMode === 'grid' ? '!bg-white/10 !border-t-white/20 text-white' : ''}"
							title="Grid view"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							</svg>
						</button>
						<button
							onclick={() => movieViewMode = 'rows'}
							class="liquid-btn-icon {movieViewMode === 'rows' ? '!bg-white/10 !border-t-white/20 text-white' : ''}"
							title="Rows by genre"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
							</svg>
						</button>
					</div>
				</div>

				<!-- Inline filter panel -->
				{#if movieFiltersExpanded && movieViewMode === 'grid'}
					<div class="liquid-panel p-4 space-y-4">
						<!-- Genre -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Genre</label>
							<select
								bind:value={movieGenreFilter}
								class="liquid-select px-3 py-1.5 text-sm"
							>
								<option value="all">All Genres</option>
								{#each allMovieGenres() as genre}
									<option value={genre}>{genre}</option>
								{/each}
							</select>
						</div>

						<!-- Content Rating -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Content Rating</label>
							<div class="flex flex-wrap gap-2">
								{#each movieContentRatingOptions as rating}
									<button
										onclick={() => toggleMovieContentRating(rating)}
										class="{movieContentRatings.has(rating) ? 'liquid-chip-active' : 'liquid-chip'}"
									>
										{rating}
									</button>
								{/each}
							</div>
						</div>

						<!-- Runtime -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">
								Runtime: {movieRuntimeMin}m - {movieRuntimeMax === 300 ? '300+' : movieRuntimeMax + 'm'}
							</label>
							<div class="flex items-center gap-4">
								<input
									type="range"
									min="0"
									max="300"
									step="15"
									bind:value={movieRuntimeMin}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
								<span class="text-text-muted text-sm">to</span>
								<input
									type="range"
									min="0"
									max="300"
									step="15"
									bind:value={movieRuntimeMax}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
							</div>
						</div>

						<!-- TMDB Score -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">
								TMDB Score: {movieRatingMin.toFixed(1)} - {movieRatingMax.toFixed(1)}
							</label>
							<div class="flex items-center gap-4">
								<input
									type="range"
									min="0"
									max="10"
									step="0.5"
									bind:value={movieRatingMin}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
								<span class="text-text-muted text-sm">to</span>
								<input
									type="range"
									min="0"
									max="10"
									step="0.5"
									bind:value={movieRatingMax}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
							</div>
						</div>

						<!-- Clear filters -->
						{#if hasActiveMovieFilters()}
							<button
								onclick={clearMovieFilters}
								class="text-sm text-white/70 hover:text-white transition-colors"
							>
								Clear all filters
							</button>
						{/if}
					</div>
				{/if}
			</div>

			{#if loadingMovies}
				<div class="flex items-center justify-center h-64">
					<div class="flex items-center gap-3">
						<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
						<p class="text-text-secondary">Loading movies...</p>
					</div>
				</div>
			{:else if movies.length === 0}
				<div class="glass-card p-12 text-center">
					<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
						<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
						</svg>
					</div>
					<h2 class="text-xl font-semibold text-text-primary mb-2">No movies found</h2>
					<p class="text-text-secondary mb-6">Add a library and scan it in Settings to get started.</p>
					<a href="/settings" class="liquid-btn inline-flex items-center gap-2">
						Go to Settings
					</a>
				</div>
			{:else if movieViewMode === 'grid'}
				{#if filteredMovies().length === 0}
					<div class="glass-card p-8 text-center">
						<p class="text-text-secondary">No movies match your search.</p>
						<button onclick={() => (movieSearch = '')} class="mt-2 text-white/70 hover:text-white transition-colors">
							Clear search
						</button>
					</div>
				{:else}
					<div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 2xl:grid-cols-8 gap-4">
						{#each filteredMovies() as movie}
							<PosterCard
								href="/movies/{movie.id}"
								title={movie.title}
								subtitle={movie.year?.toString() || 'Unknown year'}
								posterUrl={movie.posterPath ? getImageUrl(movie.posterPath) : undefined}
								rating={movie.rating}
								mediaType="movie"
								watchState={movie.watchState}
							/>
						{/each}
					</div>
				{/if}
			{:else}
				<!-- Rows by genre view -->
				<div class="space-y-8">
					{#each moviesByGenre() as [genre, genreMovies]}
						<section>
							<h2 class="text-xl font-semibold text-text-primary mb-4">{genre} <span class="text-sm text-text-muted font-normal">({genreMovies.length})</span></h2>
							<div class="flex gap-3 overflow-x-auto pb-4 scrollbar-thin">
								{#each genreMovies as movie}
									<div class="flex-shrink-0 w-32 sm:w-36">
										<PosterCard
											href="/movies/{movie.id}"
											title={movie.title}
											subtitle={movie.year?.toString() || 'Unknown year'}
											posterUrl={movie.posterPath ? getImageUrl(movie.posterPath) : undefined}
											rating={movie.rating}
											mediaType="movie"
											watchState={movie.watchState}
										/>
									</div>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- TV Shows Tab -->
	{#if activeTab === 'tv'}
		<div class="space-y-6">
			<!-- Controls -->
			<div class="space-y-4">
				<div class="flex flex-wrap items-center gap-3">
					{#if showViewMode === 'grid'}
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
							</svg>
							<input
								type="text"
								placeholder="Search shows..."
								bind:value={showSearch}
								class="liquid-input pl-10 pr-4 py-2 w-48 sm:w-64"
							/>
						</div>
						<select
							bind:value={showSort}
							class="liquid-select px-4 py-2"
						>
							<option value="added">Recently Added</option>
							<option value="title">Title A-Z</option>
							<option value="year">Year</option>
							<option value="rating">Rating</option>
						</select>
						<button
							onclick={() => showFiltersExpanded = !showFiltersExpanded}
							class="liquid-btn px-4 py-2 flex items-center gap-2 {hasActiveShowFilters() ? '!border-t-white/20 !border-l-white/10 !bg-white/5' : ''}"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
							</svg>
							Filters
							{#if hasActiveShowFilters()}
								<span class="w-2 h-2 rounded-full bg-white/50"></span>
							{/if}
							<svg class="w-4 h-4 transition-transform {showFiltersExpanded ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
							</svg>
						</button>
					{/if}

					<!-- View toggle -->
					<div class="flex gap-1 ml-auto">
						<button
							onclick={() => showViewMode = 'grid'}
							class="liquid-btn-icon {showViewMode === 'grid' ? '!bg-white/10 !border-t-white/20 text-white' : ''}"
							title="Grid view"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							</svg>
						</button>
						<button
							onclick={() => showViewMode = 'rows'}
							class="liquid-btn-icon {showViewMode === 'rows' ? '!bg-white/10 !border-t-white/20 text-white' : ''}"
							title="Rows by genre"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
							</svg>
						</button>
					</div>
				</div>

				<!-- Inline filter panel -->
				{#if showFiltersExpanded && showViewMode === 'grid'}
					<div class="liquid-panel p-4 space-y-4">
						<!-- Genre + Status row -->
						<div class="flex flex-wrap gap-4">
							<div>
								<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Genre</label>
								<select
									bind:value={showGenreFilter}
									class="liquid-select px-3 py-1.5 text-sm"
								>
									<option value="all">All Genres</option>
									{#each allShowGenres() as genre}
										<option value={genre}>{genre}</option>
									{/each}
								</select>
							</div>
							<div>
								<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Status</label>
								<select
									bind:value={showStatus}
									class="liquid-select px-3 py-1.5 text-sm"
								>
									<option value="all">All Status</option>
									<option value="continuing">Continuing</option>
									<option value="ended">Ended</option>
								</select>
							</div>
						</div>

						<!-- Content Rating -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Content Rating</label>
							<div class="flex flex-wrap gap-2">
								{#each showContentRatingOptions as rating}
									<button
										onclick={() => toggleShowContentRating(rating)}
										class="{showContentRatings.has(rating) ? 'liquid-chip-active' : 'liquid-chip'}"
									>
										{rating}
									</button>
								{/each}
							</div>
						</div>

						<!-- TMDB Score -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">
								TMDB Score: {showRatingMin.toFixed(1)} - {showRatingMax.toFixed(1)}
							</label>
							<div class="flex items-center gap-4">
								<input
									type="range"
									min="0"
									max="10"
									step="0.5"
									bind:value={showRatingMin}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
								<span class="text-text-muted text-sm">to</span>
								<input
									type="range"
									min="0"
									max="10"
									step="0.5"
									bind:value={showRatingMax}
									class="flex-1 h-1.5 bg-bg-elevated rounded-full appearance-none cursor-pointer accent-white"
								/>
							</div>
						</div>

						<!-- Clear filters -->
						{#if hasActiveShowFilters()}
							<button
								onclick={clearShowFilters}
								class="text-sm text-white/70 hover:text-white transition-colors"
							>
								Clear all filters
							</button>
						{/if}
					</div>
				{/if}
			</div>

			{#if loadingShows}
				<div class="flex items-center justify-center h-64">
					<div class="flex items-center gap-3">
						<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
						<p class="text-text-secondary">Loading shows...</p>
					</div>
				</div>
			{:else if shows.length === 0}
				<div class="glass-card p-12 text-center">
					<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
						<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
					</div>
					<h2 class="text-xl font-semibold text-text-primary mb-2">No TV shows found</h2>
					<p class="text-text-secondary mb-6">Add a library and scan it in Settings to get started.</p>
					<a href="/settings" class="liquid-btn inline-flex items-center gap-2">
						Go to Settings
					</a>
				</div>
			{:else if showViewMode === 'grid'}
				{#if filteredShows().length === 0}
					<div class="glass-card p-8 text-center">
						<p class="text-text-secondary">No shows match your filters.</p>
						<button onclick={() => { showSearch = ''; showStatus = 'all'; }} class="mt-2 text-white/70 hover:text-white transition-colors">
							Clear filters
						</button>
					</div>
				{:else}
					<div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 2xl:grid-cols-8 gap-4">
						{#each filteredShows() as show}
							<PosterCard
								href="/tv/{show.id}"
								title={show.title}
								subtitle={show.network ? `${show.year || ''} - ${show.network}` : (show.year?.toString() || 'Unknown year')}
								posterUrl={show.posterPath ? getImageUrl(show.posterPath) : undefined}
								rating={show.rating}
								mediaType="series"
								watchState={show.watchState}
								watchedEpisodes={show.watchedEpisodes}
								totalEpisodes={show.totalEpisodes}
							/>
						{/each}
					</div>
				{/if}
			{:else}
				<!-- Rows by genre view -->
				<div class="space-y-8">
					{#each showsByGenre() as [genre, genreShows]}
						<section>
							<h2 class="text-xl font-semibold text-text-primary mb-4">{genre} <span class="text-sm text-text-muted font-normal">({genreShows.length})</span></h2>
							<div class="flex gap-3 overflow-x-auto pb-4 scrollbar-thin">
								{#each genreShows as show}
									<div class="flex-shrink-0 w-32 sm:w-36">
										<PosterCard
											href="/tv/{show.id}"
											title={show.title}
											subtitle={show.network ? `${show.year || ''} - ${show.network}` : (show.year?.toString() || 'Unknown year')}
											posterUrl={show.posterPath ? getImageUrl(show.posterPath) : undefined}
											rating={show.rating}
											mediaType="series"
											watchState={show.watchState}
											watchedEpisodes={show.watchedEpisodes}
											totalEpisodes={show.totalEpisodes}
										/>
									</div>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Music Tab -->
	{#if activeTab === 'music'}
		<div class="space-y-6">
			<!-- View toggle -->
			<div class="flex gap-2">
				<button
					class="{musicView === 'artists' ? 'liquid-chip-active' : 'liquid-chip'} px-4 py-2"
					onclick={() => musicView = 'artists'}
				>
					Artists
				</button>
				<button
					class="{musicView === 'albums' ? 'liquid-chip-active' : 'liquid-chip'} px-4 py-2"
					onclick={() => musicView = 'albums'}
				>
					Albums
				</button>
			</div>

			{#if loadingMusic}
				<div class="flex items-center justify-center h-64">
					<div class="flex items-center gap-3">
						<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
						<p class="text-text-secondary">Loading music...</p>
					</div>
				</div>
			{:else if musicView === 'artists'}
				{#if artists.length === 0}
					<div class="glass-card p-12 text-center">
						<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
							<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
							</svg>
						</div>
						<h2 class="text-xl font-semibold text-text-primary mb-2">No artists found</h2>
						<p class="text-text-secondary mb-6">Add a music library in Settings to scan for music.</p>
						<a href="/settings" class="liquid-btn inline-flex items-center gap-2">
							Go to Settings
						</a>
					</div>
				{:else}
					<div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 2xl:grid-cols-8 gap-4">
						{#each artists as artist}
							<a href="/music/artists/{artist.id}" class="group">
								<div class="aspect-square bg-bg-card rounded-lg flex items-center justify-center overflow-hidden">
									{#if artist.imagePath}
										<img
											src="/images/{artist.imagePath}"
											alt={artist.name}
											class="w-full h-full object-cover"
										/>
									{:else}
										<div class="text-text-muted">
											<svg class="w-10 h-10" fill="currentColor" viewBox="0 0 24 24">
												<path d="M12 14.25c2.485 0 4.5-2.015 4.5-4.5s-2.015-4.5-4.5-4.5-4.5 2.015-4.5 4.5 2.015 4.5 4.5 4.5zm0 1.5c-3.315 0-9 1.665-9 4.98v.27c0 .825.675 1.5 1.5 1.5h15c.825 0 1.5-.675 1.5-1.5v-.27c0-3.315-5.685-4.98-9-4.98z"/>
											</svg>
										</div>
									{/if}
								</div>
								<h3 class="mt-2 font-medium truncate text-text-primary group-hover:text-white transition-colors">{artist.name}</h3>
							</a>
						{/each}
					</div>
				{/if}
			{:else}
				{#if albums.length === 0}
					<div class="glass-card p-12 text-center">
						<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
							<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
							</svg>
						</div>
						<h2 class="text-xl font-semibold text-text-primary mb-2">No albums found</h2>
						<p class="text-text-secondary mb-6">Add a music library in Settings to scan for music.</p>
						<a href="/settings" class="liquid-btn inline-flex items-center gap-2">
							Go to Settings
						</a>
					</div>
				{:else}
					<div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 2xl:grid-cols-8 gap-4">
						{#each albums as album}
							<a href="/music/albums/{album.id}" class="group">
								<div class="aspect-square bg-bg-card rounded-lg flex items-center justify-center overflow-hidden">
									{#if album.coverPath}
										<img
											src="/images/{album.coverPath}"
											alt={album.title}
											class="w-full h-full object-cover"
										/>
									{:else}
										<div class="text-text-muted">
											<svg class="w-10 h-10" fill="currentColor" viewBox="0 0 24 24">
												<path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
											</svg>
										</div>
									{/if}
								</div>
								<h3 class="mt-2 font-medium truncate text-text-primary group-hover:text-white transition-colors">{album.title}</h3>
								{#if album.year}
									<p class="text-sm text-text-secondary">{album.year}</p>
								{/if}
							</a>
						{/each}
					</div>
				{/if}
			{/if}
		</div>
	{/if}

	<!-- Books Tab -->
	{#if activeTab === 'books'}
		<div class="space-y-6">
			<!-- Controls -->
			<div class="flex flex-wrap items-center gap-3">
				<div class="relative">
					<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
					<input
						type="text"
						placeholder="Search books..."
						bind:value={bookSearch}
						class="liquid-input pl-10 pr-4 py-2 w-48 sm:w-64"
					/>
				</div>
			</div>

			{#if loadingBooks}
				<div class="flex items-center justify-center h-64">
					<div class="flex items-center gap-3">
						<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
						<p class="text-text-secondary">Loading books...</p>
					</div>
				</div>
			{:else if books.length === 0}
				<div class="glass-card p-12 text-center">
					<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
						<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
						</svg>
					</div>
					<h2 class="text-xl font-semibold text-text-primary mb-2">No books found</h2>
					<p class="text-text-secondary mb-6">Add a books library in Settings to scan for books.</p>
					<a href="/settings" class="liquid-btn inline-flex items-center gap-2">
						Go to Settings
					</a>
				</div>
			{:else if filteredBooks().length === 0}
				<div class="glass-card p-8 text-center">
					<p class="text-text-secondary">No books match your search.</p>
					<button onclick={() => (bookSearch = '')} class="mt-2 text-white/70 hover:text-white transition-colors">
						Clear search
					</button>
				</div>
			{:else}
				<div class="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 2xl:grid-cols-8 gap-4">
					{#each filteredBooks() as book}
						<a href="/books/{book.id}" class="group">
							<div class="aspect-[2/3] bg-bg-card rounded-lg overflow-hidden relative">
								{#if book.coverPath}
									<img
										src="/images/{book.coverPath}"
										alt={book.title}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-text-muted p-4">
										<div class="text-center">
											<svg class="w-12 h-12 mx-auto mb-2" fill="currentColor" viewBox="0 0 24 24">
												<path d="M18 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zM6 4h5v8l-2.5-1.5L6 12V4z"/>
											</svg>
											<span class="text-xs">{book.format.toUpperCase()}</span>
										</div>
									</div>
								{/if}
								<div class="absolute top-2 right-2">
									<span class="px-2 py-0.5 text-xs rounded {getFormatColor(book.format)}">
										{book.format.toUpperCase()}
									</span>
								</div>
							</div>
							<h3 class="mt-2 font-medium text-sm truncate text-text-primary group-hover:text-white transition-colors">{book.title}</h3>
							{#if book.author}
								<p class="text-xs text-text-secondary truncate">{book.author}</p>
							{/if}
						</a>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>
