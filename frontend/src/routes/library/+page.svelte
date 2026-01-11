<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import {
		getMovies,
		getShows,
		getCollections,
		createCollection,
		getSmartPlaylists,
		getImageUrl,
		type Movie,
		type Show,
		type Collection,
		type SmartPlaylist
	} from '$lib/api';
	import MediaCard from '$lib/components/media/MediaCard.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { auth } from '$lib/stores/auth';

	// Tab state from URL
	type TabType = 'movies' | 'tv' | 'collections' | 'playlists';
	let activeTab = $state<TabType>('movies');

	// Data states
	let movies: Movie[] = $state([]);
	let shows: Show[] = $state([]);
	let collections: Collection[] = $state([]);
	let playlists: SmartPlaylist[] = $state([]);

	// Loading states per tab
	let loadingMovies = $state(false);
	let loadingShows = $state(false);
	let loadingCollections = $state(false);
	let loadingPlaylists = $state(false);
	let loadedTabs = $state<Set<TabType>>(new Set());

	// Get current user for admin check
	const user = $derived($auth);

	let error: string | null = $state(null);

	// Collection creation modal state
	let showCreateModal = $state(false);
	let newCollectionName = $state('');
	let newCollectionDescription = $state('');
	let creatingCollection = $state(false);

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

	// Valid tabs for URL handling
	const validTabs: TabType[] = ['movies', 'tv', 'collections', 'playlists'];

	// Reactively sync tab from URL (handles back/forward navigation)
	$effect(() => {
		const urlTab = $page.url.searchParams.get('tab') as TabType | null;
		if (urlTab && validTabs.includes(urlTab)) {
			if (activeTab !== urlTab) {
				activeTab = urlTab;
				loadTabData(urlTab);
			}
		}
	});

	// Read tab from URL on mount
	onMount(() => {
		const urlTab = $page.url.searchParams.get('tab') as TabType | null;
		if (urlTab && validTabs.includes(urlTab)) {
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

	// Update URL when tab changes (don't use replaceState so back button works)
	function setTab(tab: TabType) {
		if (activeTab === tab) return;
		activeTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set('tab', tab);
		goto(url.pathname + url.search, { keepFocus: true });
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
				case 'collections':
					loadingCollections = true;
					collections = await getCollections();
					break;
				case 'playlists':
					loadingPlaylists = true;
					playlists = await getSmartPlaylists();
					break;
			}
			loadedTabs.add(tab);
		} catch (e) {
			error = e instanceof Error ? e.message : `Failed to load ${tab}`;
		} finally {
			loadingMovies = false;
			loadingShows = false;
			loadingCollections = false;
			loadingPlaylists = false;
		}
	}

	// Handle collection creation
	async function handleCreateCollection(e: Event) {
		e.preventDefault();
		if (!newCollectionName.trim() || creatingCollection) return;

		creatingCollection = true;
		try {
			const newColl = await createCollection({
				name: newCollectionName.trim(),
				description: newCollectionDescription.trim() || undefined
			});
			collections = [newColl, ...collections];
			showCreateModal = false;
			newCollectionName = '';
			newCollectionDescription = '';
			// Navigate to the new collection
			goto(`/collections/${newColl.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create collection';
		} finally {
			creatingCollection = false;
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

	// Genre options for Select components
	const movieGenreOptions = $derived(() => {
		return [{ value: 'all', label: 'All Genres' }, ...allMovieGenres().map(g => ({ value: g, label: g }))];
	});

	const showGenreOptions = $derived(() => {
		return [{ value: 'all', label: 'All Genres' }, ...allShowGenres().map(g => ({ value: g, label: g }))];
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
		{ id: 'collections' as TabType, label: 'Collections', count: collections.length },
		{ id: 'playlists' as TabType, label: 'Playlists', count: playlists.length },
	]);
</script>

<svelte:head>
	<title>Library - Outpost</title>
</svelte:head>

<div class="space-y-6">
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
				<div class="absolute inset-0 bg-gradient-to-t from-[#0a0a0a] via-[#0a0a0a]/60 to-[#0a0a0a]/30 pointer-events-none"></div>
			</section>
		{/if}
	{/if}

	<div class="px-[60px] space-y-6 {(activeTab === 'movies' || activeTab === 'tv') && heroItems().length > 0 ? '-mt-56 relative z-10' : ''}">
		<!-- Filter Pills -->
		<div class="flex flex-wrap gap-2">
			<!-- Media type pills -->
			{#each tabs as tab}
				<button
					onclick={() => setTab(tab.id)}
					class="px-4 py-2.5 text-sm font-medium transition-all rounded-full min-h-[44px] flex items-center
						{activeTab === tab.id
							? 'bg-text-primary text-black border border-text-primary'
							: 'bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
				>
					{tab.label}
					{#if loadedTabs.has(tab.id) && tab.count > 0}
						<span class="ml-1.5 text-xs opacity-60">({tab.count})</span>
					{/if}
				</button>
			{/each}

			<div class="w-px h-8 bg-border-subtle self-center mx-1"></div>

			<!-- Genre quick-filter pills -->
			{#if activeTab === 'movies'}
				{#each ['Action', 'Comedy', 'Drama', 'Sci-Fi', 'Horror', 'Thriller'] as genre}
					<button
						onclick={() => movieGenreFilter = movieGenreFilter === genre ? 'all' : genre}
						class="px-4 py-2.5 text-sm font-medium transition-all rounded-full min-h-[44px] flex items-center
							{movieGenreFilter === genre
								? 'bg-text-primary text-black border border-text-primary'
								: 'bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
					>
						{genre}
					</button>
				{/each}
			{:else if activeTab === 'tv'}
				{#each ['Action & Adventure', 'Comedy', 'Drama', 'Crime', 'Sci-Fi & Fantasy', 'Mystery'] as genre}
					<button
						onclick={() => showGenreFilter = showGenreFilter === genre ? 'all' : genre}
						class="px-4 py-2.5 text-sm font-medium transition-all rounded-full min-h-[44px] flex items-center
							{showGenreFilter === genre
								? 'bg-text-primary text-black border border-text-primary'
								: 'bg-glass backdrop-blur-xl border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
					>
						{genre}
					</button>
				{/each}
			{/if}
		</div>

		{#if error}
		<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-lg">
			{error}
			<button class="ml-2 underline" onclick={() => (error = null)}>Dismiss</button>
		</div>
	{/if}

	<!-- Movies Tab -->
	{#if activeTab === 'movies'}
		<div class="space-y-6">
			<!-- Controls -->
			<div class="space-y-4">
				<div class="flex flex-wrap items-center gap-3 relative z-20">
					{#if movieViewMode === 'grid'}
						<div class="inline-flex items-center gap-2 p-1.5 rounded-xl bg-bg-card border border-border-subtle">
							<div class="relative">
								<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
								</svg>
								<input
									type="text"
									placeholder="Search..."
									bind:value={movieSearch}
									class="bg-transparent pl-9 pr-3 py-1.5 w-40 text-sm text-text-primary placeholder-text-muted focus:outline-none"
								/>
							</div>
							<div class="w-px h-6 bg-border-subtle"></div>
							<Select
								bind:value={movieSort}
								options={[
									{ value: 'added', label: 'Recently Added' },
									{ value: 'title', label: 'Title A-Z' },
									{ value: 'year', label: 'Year' },
									{ value: 'rating', label: 'Rating' }
								]}
								class="w-40"
							/>
							<div class="w-px h-6 bg-border-subtle"></div>
							<button
								onclick={() => movieFiltersExpanded = !movieFiltersExpanded}
								class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-full transition-colors {hasActiveMovieFilters() || movieFiltersExpanded ? 'bg-glass-hover text-text-primary' : 'text-text-muted hover:text-text-primary hover:bg-glass-hover'}"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
								</svg>
								Filters
								{#if hasActiveMovieFilters()}
									<span class="w-1.5 h-1.5 rounded-full bg-text-primary"></span>
								{/if}
							</button>
						</div>
					{/if}

				</div>

				<!-- Inline filter panel -->
				{#if movieFiltersExpanded && movieViewMode === 'grid'}
					<div class="liquid-panel p-4 space-y-4">
						<!-- Genre -->
						<div>
							<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Genre</label>
							<Select
								bind:value={movieGenreFilter}
								options={movieGenreOptions()}
								class="w-48"
							/>
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
								class="text-sm text-text-secondary hover:text-text-primary transition-colors"
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
						<div class="spinner-lg text-cream"></div>
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
					<button onclick={() => goto('/settings')} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-glass border border-border-subtle text-text-primary hover:bg-glass-hover transition-all">
						Go to Settings
					</button>
				</div>
			{:else if movieViewMode === 'grid'}
				{#if filteredMovies().length === 0}
					<div class="glass-card p-8 text-center">
						<p class="text-text-secondary">No movies match your search.</p>
						<button onclick={() => (movieSearch = '')} class="mt-2 text-text-secondary hover:text-text-primary transition-colors">
							Clear search
						</button>
					</div>
				{:else}
					<div class="flex items-center justify-between mb-4">
						<h2 class="text-xl font-semibold text-text-primary">
							{movieSearch || movieGenreFilter !== 'all' || hasActiveMovieFilters() ? 'Results' : (movieSort === 'added' ? 'Recently Added' : movieSort === 'title' ? 'A-Z' : movieSort === 'year' ? 'By Year' : 'Top Rated')}
							<span class="text-sm text-text-muted font-normal ml-2">({filteredMovies().length})</span>
						</h2>
					</div>
					<div class="grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-3">
						{#each filteredMovies() as movie}
							<MediaCard
								type="poster"
								fill={true}
								href="/movies/{movie.id}"
								title={movie.title}
								subtitle={movie.year?.toString() || 'Unknown year'}
								imagePath={movie.posterPath}
								isLocal={true}
								runtime={movie.runtime}
								contentRating={movie.contentRating}
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
									<MediaCard
										type="poster"
										href="/movies/{movie.id}"
										title={movie.title}
										subtitle={movie.year?.toString() || 'Unknown year'}
										imagePath={movie.posterPath}
										isLocal={true}
										runtime={movie.runtime}
										contentRating={movie.contentRating}
										watchState={movie.watchState}
									/>
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
				<div class="flex flex-wrap items-center gap-3 relative z-20">
					{#if showViewMode === 'grid'}
						<div class="inline-flex items-center gap-2 p-1.5 rounded-xl bg-bg-card border border-border-subtle">
							<div class="relative">
								<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
								</svg>
								<input
									type="text"
									placeholder="Search..."
									bind:value={showSearch}
									class="bg-transparent pl-9 pr-3 py-1.5 w-40 text-sm text-text-primary placeholder-text-muted focus:outline-none"
								/>
							</div>
							<div class="w-px h-6 bg-border-subtle"></div>
							<Select
								bind:value={showSort}
								options={[
									{ value: 'added', label: 'Recently Added' },
									{ value: 'title', label: 'Title A-Z' },
									{ value: 'year', label: 'Year' },
									{ value: 'rating', label: 'Rating' }
								]}
								class="w-40"
							/>
							<div class="w-px h-6 bg-border-subtle"></div>
							<button
								onclick={() => showFiltersExpanded = !showFiltersExpanded}
								class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-full transition-colors {hasActiveShowFilters() || showFiltersExpanded ? 'bg-glass-hover text-text-primary' : 'text-text-muted hover:text-text-primary hover:bg-glass-hover'}"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
								</svg>
								Filters
								{#if hasActiveShowFilters()}
									<span class="w-1.5 h-1.5 rounded-full bg-text-primary"></span>
								{/if}
							</button>
						</div>
					{/if}

				</div>

				<!-- Inline filter panel -->
				{#if showFiltersExpanded && showViewMode === 'grid'}
					<div class="liquid-panel p-4 space-y-4">
						<!-- Genre + Status row -->
						<div class="flex flex-wrap gap-4">
							<div>
								<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Genre</label>
								<Select
									bind:value={showGenreFilter}
									options={showGenreOptions()}
									class="w-48"
								/>
							</div>
							<div>
								<label class="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2 block">Status</label>
								<Select
									bind:value={showStatus}
									options={[
										{ value: 'all', label: 'All Status' },
										{ value: 'continuing', label: 'Continuing' },
										{ value: 'ended', label: 'Ended' }
									]}
									class="w-40"
								/>
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
								class="text-sm text-text-secondary hover:text-text-primary transition-colors"
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
						<div class="spinner-lg text-cream"></div>
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
					<button onclick={() => goto('/settings')} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-glass border border-border-subtle text-text-primary hover:bg-glass-hover transition-all">
						Go to Settings
					</button>
				</div>
			{:else if showViewMode === 'grid'}
				{#if filteredShows().length === 0}
					<div class="glass-card p-8 text-center">
						<p class="text-text-secondary">No shows match your filters.</p>
						<button onclick={() => { showSearch = ''; showStatus = 'all'; }} class="mt-2 text-text-secondary hover:text-text-primary transition-colors">
							Clear filters
						</button>
					</div>
				{:else}
					<div class="flex items-center justify-between mb-4">
						<h2 class="text-xl font-semibold text-text-primary">
							{showSearch || showGenreFilter !== 'all' || hasActiveShowFilters() ? 'Results' : (showSort === 'added' ? 'Recently Added' : showSort === 'title' ? 'A-Z' : showSort === 'year' ? 'By Year' : 'Top Rated')}
							<span class="text-sm text-text-muted font-normal ml-2">({filteredShows().length})</span>
						</h2>
					</div>
					<div class="grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-3">
						{#each filteredShows() as show}
							<MediaCard
								type="poster"
								fill={true}
								href="/tv/{show.id}"
								title={show.title}
								subtitle={show.year?.toString() || 'Unknown year'}
								imagePath={show.posterPath}
								isLocal={true}
								contentRating={show.contentRating}
								watchState={show.watchState}
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
									<MediaCard
										type="poster"
										href="/tv/{show.id}"
										title={show.title}
										subtitle={show.year?.toString() || 'Unknown year'}
										imagePath={show.posterPath}
										isLocal={true}
										contentRating={show.contentRating}
										watchState={show.watchState}
									/>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Collections Tab -->
	{#if activeTab === 'collections'}
		<div class="space-y-6">
			<!-- Header with Create button -->
			<div class="flex items-center justify-between">
				<h2 class="text-xl font-semibold text-text-primary">
					Collections
					<span class="text-sm text-text-muted font-normal ml-2">({collections.length})</span>
				</h2>
				{#if user?.role === 'admin'}
					<button
						onclick={() => showCreateModal = true}
						class="px-4 py-2 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors flex items-center gap-2"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						Create Collection
					</button>
				{/if}
			</div>

			{#if loadingCollections}
				<div class="flex items-center justify-center py-20">
					<div class="animate-spin w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full"></div>
				</div>
			{:else if collections.length === 0}
				<div class="text-center py-20">
					<div class="text-6xl mb-4">📚</div>
					<h3 class="text-xl font-medium text-text-primary mb-2">No Collections Yet</h3>
					<p class="text-text-muted mb-6">Collections are automatically created when you add movies from franchises,<br/>or you can create custom collections to organize your library.</p>
					{#if user?.role === 'admin'}
						<button
							onclick={() => showCreateModal = true}
							class="px-6 py-3 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors"
						>
							Create Your First Collection
						</button>
					{/if}
				</div>
			{:else}
				<div class="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-4">
					{#each collections as collection}
						<a
							href="/collections/{collection.id}"
							class="group bg-bg-card border border-border-subtle rounded-xl overflow-hidden hover:border-border-hover transition-all hover:shadow-lg"
						>
							<!-- Collection Poster/Backdrop -->
							<div class="relative aspect-video bg-bg-tertiary overflow-hidden">
								{#if collection.backdropPath || collection.posterPath}
									<img
										src={getImageUrl(collection.backdropPath || collection.posterPath)}
										alt={collection.name}
										class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-4xl text-text-muted">📚</div>
								{/if}
								<!-- Gradient overlay -->
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>
								<!-- Auto badge -->
								{#if collection.isAuto}
									<div class="absolute top-3 left-3 px-2 py-1 rounded bg-accent-primary/90 text-white text-xs font-medium">
										TMDB Collection
									</div>
								{/if}
								<!-- Count badge -->
								<div class="absolute bottom-3 right-3 px-2 py-1 rounded bg-black/60 text-white text-sm font-medium">
									{collection.ownedCount}/{collection.itemCount}
								</div>
							</div>
							<!-- Info -->
							<div class="p-4">
								<h3 class="font-semibold text-text-primary group-hover:text-accent-primary transition-colors truncate">
									{collection.name}
								</h3>
								{#if collection.description}
									<p class="text-sm text-text-muted mt-1 line-clamp-2">{collection.description}</p>
								{/if}
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Playlists Tab -->
	{#if activeTab === 'playlists'}
		<div class="space-y-6">
			<!-- Header with Create button -->
			<div class="flex items-center justify-between">
				<h2 class="text-xl font-semibold text-text-primary">
					Smart Playlists
					<span class="text-sm text-text-muted font-normal ml-2">({playlists.length})</span>
				</h2>
				<a
					href="/playlists/new"
					class="px-4 py-2 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors flex items-center gap-2"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Create Playlist
				</a>
			</div>

			{#if loadingPlaylists}
				<div class="flex items-center justify-center py-20">
					<div class="animate-spin w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full"></div>
				</div>
			{:else if playlists.length === 0}
				<div class="text-center py-20">
					<div class="text-6xl mb-4">🎬</div>
					<h3 class="text-xl font-medium text-text-primary mb-2">No Playlists Yet</h3>
					<p class="text-text-muted mb-6">Create smart playlists with custom rules to organize your media.<br/>Filter by genre, year, rating, and more.</p>
					<a
						href="/playlists/new"
						class="inline-block px-6 py-3 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors"
					>
						Create Your First Playlist
					</a>
				</div>
			{:else}
				<div class="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-4">
					{#each playlists as playlist}
						<a
							href="/playlists/{playlist.id}"
							class="group bg-bg-card border border-border-subtle rounded-xl overflow-hidden hover:border-border-hover transition-all hover:shadow-lg"
						>
							<!-- Playlist preview images -->
							<div class="relative aspect-video bg-bg-tertiary overflow-hidden">
								{#if playlist.previewItems && playlist.previewItems.length > 0}
									<div class="grid grid-cols-3 h-full">
										{#each playlist.previewItems.slice(0, 3) as item, i}
											<div class="relative overflow-hidden {i === 1 ? 'border-x border-bg-tertiary' : ''}">
												{#if item.posterPath}
													<img
														src={getImageUrl(item.posterPath)}
														alt=""
														class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
													/>
												{:else}
													<div class="w-full h-full bg-bg-elevated"></div>
												{/if}
											</div>
										{/each}
									</div>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-4xl text-text-muted">🎬</div>
								{/if}
								<!-- Gradient overlay -->
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>
								<!-- System badge -->
								{#if playlist.isSystem}
									<div class="absolute top-3 left-3 px-2 py-1 rounded bg-blue-500/90 text-white text-xs font-medium">
										System
									</div>
								{/if}
								<!-- Count badge -->
								<div class="absolute bottom-3 right-3 px-2 py-1 rounded bg-black/60 text-white text-sm font-medium">
									{playlist.itemCount} items
								</div>
							</div>
							<!-- Info -->
							<div class="p-4">
								<h3 class="font-semibold text-text-primary group-hover:text-accent-primary transition-colors truncate">
									{playlist.name}
								</h3>
								{#if playlist.description}
									<p class="text-sm text-text-muted mt-1 line-clamp-2">{playlist.description}</p>
								{/if}
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	</div>
</div>

<!-- Create Collection Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
		onclick={(e) => { if (e.target === e.currentTarget) showCreateModal = false; }}
		onkeydown={(e) => { if (e.key === 'Escape') showCreateModal = false; }}
		role="dialog"
		tabindex="-1"
	>
		<div class="bg-bg-primary border border-border-subtle rounded-xl w-full max-w-md">
			<div class="p-6 border-b border-border-subtle">
				<h2 class="text-xl font-semibold text-text-primary">Create Collection</h2>
			</div>
			<form onsubmit={handleCreateCollection} class="p-6 space-y-4">
				<div>
					<label for="collName" class="block text-sm font-medium text-text-secondary mb-2">Name</label>
					<input
						id="collName"
						type="text"
						bind:value={newCollectionName}
						placeholder="My Collection"
						required
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary"
					/>
				</div>
				<div>
					<label for="collDesc" class="block text-sm font-medium text-text-secondary mb-2">Description (optional)</label>
					<textarea
						id="collDesc"
						bind:value={newCollectionDescription}
						placeholder="Add a description..."
						rows="3"
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary resize-none"
					></textarea>
				</div>
				<div class="flex gap-3 pt-2">
					<button
						type="button"
						onclick={() => showCreateModal = false}
						class="flex-1 px-4 py-2.5 rounded-lg border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={!newCollectionName.trim() || creatingCollection}
						class="flex-1 px-4 py-2.5 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{creatingCollection ? 'Creating...' : 'Create'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
