<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import {
		getMovies,
		getShows,
		getImageUrl,
		type Movie,
		type Show
	} from '$lib/api';
	import PosterCard from '$lib/components/PosterCard.svelte';

	// Tab state from URL
	type TabType = 'movies' | 'tv';
	let activeTab = $state<TabType>('movies');

	// Data states
	let movies: Movie[] = $state([]);
	let shows: Show[] = $state([]);

	// Loading states per tab
	let loadingMovies = $state(false);
	let loadingShows = $state(false);
	let loadedTabs = $state<Set<TabType>>(new Set());

	let error: string | null = $state(null);

	// Hero carousel state
	let heroIndex = $state(0);
	let autoplayTimer: ReturnType<typeof setInterval> | null = null;
	const AUTOPLAY_INTERVAL = 15000; // 15 seconds

	// Get hero items based on active tab - only movies and TV have backdrops
	const heroItems = $derived(() => {
		if (activeTab === 'movies') {
			return movies.filter(m => m.backdropPath).slice(0, 10);
		} else if (activeTab === 'tv') {
			return shows.filter(s => s.backdropPath).slice(0, 10);
		}
		return [];
	});

	// Current hero item
	const currentHero = $derived(() => {
		const items = heroItems();
		return items[heroIndex] || null;
	});

	// Reset hero index when tab changes
	$effect(() => {
		// When activeTab changes, reset hero index
		activeTab;
		heroIndex = 0;
	});

	function nextHero() {
		const items = heroItems();
		if (items.length > 0) {
			heroIndex = (heroIndex + 1) % items.length;
		}
		resetAutoplay();
	}

	function prevHero() {
		const items = heroItems();
		if (items.length > 0) {
			heroIndex = (heroIndex - 1 + items.length) % items.length;
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
			const items = heroItems();
			if (items.length > 0) {
				heroIndex = (heroIndex + 1) % items.length;
			}
		}, AUTOPLAY_INTERVAL);
	}

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
		if (urlTab && ['movies', 'tv'].includes(urlTab)) {
			activeTab = urlTab;
		}
		loadTabData(activeTab);
		// Start hero autoplay
		resetAutoplay();
	});

	onDestroy(() => {
		if (autoplayTimer) {
			clearInterval(autoplayTimer);
		}
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
			}
			loadedTabs.add(tab);
		} catch (e) {
			error = e instanceof Error ? e.message : `Failed to load ${tab}`;
		} finally {
			loadingMovies = false;
			loadingShows = false;
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

	const tabs = $derived([
		{ id: 'movies' as TabType, label: 'Movies', count: movies.length },
		{ id: 'tv' as TabType, label: 'TV Shows', count: shows.length },
	]);
</script>

<svelte:head>
	<title>Library - Outpost</title>
</svelte:head>

<div class="space-y-6 -mt-22 -mx-6">
	<!-- Hero Background - Only for Movies and TV -->
	{#if (activeTab === 'movies' || activeTab === 'tv') && heroItems().length > 0}
		{@const hero = currentHero()}
		{#if hero}
			<section class="relative h-[35vh] min-h-[280px] overflow-hidden">
				<!-- Backdrop image with fade transition -->
				{#key `${activeTab}-${heroIndex}`}
					<img
						src={getImageUrl(hero.backdropPath || '')}
						alt=""
						class="absolute inset-0 w-full h-full object-cover animate-fade-in pointer-events-none"
						style="object-position: center {hero.focalY ?? 25}%;"
						draggable="false"
					/>
				{/key}

				<!-- Gradient overlays -->
				<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-bg-primary/60 to-bg-primary/30 pointer-events-none"></div>
			</section>
		{/if}
	{/if}

	<div class="px-6 space-y-6 {(activeTab === 'movies' || activeTab === 'tv') && heroItems().length > 0 ? '-mt-56 relative z-10' : ''}">
		<!-- Tab bar -->
		<div class="inline-flex gap-1 p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
			{#each tabs as tab}
				<button
					onclick={() => setTab(tab.id)}
					class="px-4 py-2 text-sm font-medium transition-all rounded-lg
						{activeTab === tab.id
							? 'bg-white/15 text-white'
							: 'text-white/60 hover:text-white hover:bg-white/5'}"
				>
					{tab.label}
					{#if loadedTabs.has(tab.id) && tab.count > 0}
						<span class="ml-1.5 text-xs text-white/40">({tab.count})</span>
					{/if}
				</button>
			{/each}
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
						<div class="inline-flex items-center gap-2 p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
							<div class="relative">
								<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
								</svg>
								<input
									type="text"
									placeholder="Search..."
									bind:value={movieSearch}
									class="bg-transparent pl-9 pr-3 py-1.5 w-40 text-sm text-white placeholder-white/40 focus:outline-none"
								/>
							</div>
							<div class="w-px h-6 bg-white/10"></div>
							<select
								bind:value={movieSort}
								class="bg-transparent px-3 py-1.5 text-sm text-white/80 focus:outline-none cursor-pointer"
							>
								<option value="added" class="bg-zinc-900">Recently Added</option>
								<option value="title" class="bg-zinc-900">Title A-Z</option>
								<option value="year" class="bg-zinc-900">Year</option>
								<option value="rating" class="bg-zinc-900">Rating</option>
							</select>
							<div class="w-px h-6 bg-white/10"></div>
							<button
								onclick={() => movieFiltersExpanded = !movieFiltersExpanded}
								class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg transition-colors {hasActiveMovieFilters() || movieFiltersExpanded ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
								</svg>
								Filters
								{#if hasActiveMovieFilters()}
									<span class="w-1.5 h-1.5 rounded-full bg-white"></span>
								{/if}
							</button>
						</div>
					{/if}

					<!-- View toggle -->
					<div class="inline-flex gap-1 p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10 ml-auto">
						<button
							onclick={() => movieViewMode = 'grid'}
							class="p-2 rounded-lg transition-colors {movieViewMode === 'grid' ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
							title="Grid view"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							</svg>
						</button>
						<button
							onclick={() => movieViewMode = 'rows'}
							class="p-2 rounded-lg transition-colors {movieViewMode === 'rows' ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
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
						<div class="inline-flex items-center gap-2 p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
							<div class="relative">
								<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
								</svg>
								<input
									type="text"
									placeholder="Search..."
									bind:value={showSearch}
									class="bg-transparent pl-9 pr-3 py-1.5 w-40 text-sm text-white placeholder-white/40 focus:outline-none"
								/>
							</div>
							<div class="w-px h-6 bg-white/10"></div>
							<select
								bind:value={showSort}
								class="bg-transparent px-3 py-1.5 text-sm text-white/80 focus:outline-none cursor-pointer"
							>
								<option value="added" class="bg-zinc-900">Recently Added</option>
								<option value="title" class="bg-zinc-900">Title A-Z</option>
								<option value="year" class="bg-zinc-900">Year</option>
								<option value="rating" class="bg-zinc-900">Rating</option>
							</select>
							<div class="w-px h-6 bg-white/10"></div>
							<button
								onclick={() => showFiltersExpanded = !showFiltersExpanded}
								class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg transition-colors {hasActiveShowFilters() || showFiltersExpanded ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
								</svg>
								Filters
								{#if hasActiveShowFilters()}
									<span class="w-1.5 h-1.5 rounded-full bg-white"></span>
								{/if}
							</button>
						</div>
					{/if}

					<!-- View toggle -->
					<div class="inline-flex gap-1 p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10 ml-auto">
						<button
							onclick={() => showViewMode = 'grid'}
							class="p-2 rounded-lg transition-colors {showViewMode === 'grid' ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
							title="Grid view"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
							</svg>
						</button>
						<button
							onclick={() => showViewMode = 'rows'}
							class="p-2 rounded-lg transition-colors {showViewMode === 'rows' ? 'bg-white/15 text-white' : 'text-white/60 hover:text-white hover:bg-white/5'}"
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

	</div>
</div>
