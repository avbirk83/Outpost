<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { createRequest, getImageUrl, getSystemStatus, getNotifications, getUnreadCount, markRead, markAllRead, type SystemStatus, type Movie, type Show, type Artist, type Book, type DiscoverItem, type Notification, type Profile, type TmdbMovieResult, type TmdbTVResult, AVATARS } from '$lib/api';
	import { profileStore } from '$lib/stores/profile';
	import { onMount, onDestroy } from 'svelte';
	import { normalizeText, searchScore } from '$lib/utils/search';

	interface Props {
		username?: string;
		isAdmin?: boolean;
		activeProfile?: Profile | null;
		onLogout?: () => void;
	}

	let { username = '', isAdmin = false, activeProfile = null, onLogout }: Props = $props();

	let showProfileMenu = $state(false);
	let profiles = $state<Profile[]>([]);

	// Subscribe to profile store for profile list
	profileStore.subscribe((state) => {
		profiles = state.profiles;
	});

	let showUserMenu = $state(false);

	// Notifications
	let showNotifications = $state(false);
	let notifications = $state<Notification[]>([]);
	let unreadCount = $state(0);
	let notificationInterval: ReturnType<typeof setInterval> | null = null;
	let loadingNotifications = $state(false);

	// System status
	let systemStatus: SystemStatus | null = $state(null);
	let statusInterval: ReturnType<typeof setInterval> | null = null;
	let prevPendingRequests = $state(-1); // -1 indicates initial load not done yet
	let hasNewRequest = $state(false);

	onMount(() => {
		loadSystemStatus();
		statusInterval = setInterval(loadSystemStatus, 5000); // Check more frequently

		// Load notification count on mount and poll every 30 seconds
		loadUnreadCount();
		notificationInterval = setInterval(loadUnreadCount, 30000);
	});

	onDestroy(() => {
		if (statusInterval) clearInterval(statusInterval);
		if (notificationInterval) clearInterval(notificationInterval);
	});

	async function loadUnreadCount() {
		try {
			unreadCount = await getUnreadCount();
		} catch (e) {
			console.debug('Failed to load unread count:', e);
		}
	}

	async function loadNotifications() {
		loadingNotifications = true;
		try {
			notifications = await getNotifications(false, 20);
		} catch (e) {
			console.debug('Failed to load notifications:', e);
		}
		loadingNotifications = false;
	}

	function toggleNotifications() {
		showNotifications = !showNotifications;
		if (showNotifications) {
			loadNotifications();
		}
	}

	function closeNotifications() {
		showNotifications = false;
	}

	async function handleMarkAllRead() {
		try {
			await markAllRead();
			notifications = notifications.map((n) => ({ ...n, read: true }));
			unreadCount = 0;
		} catch (e) {
			console.debug('Failed to mark all read:', e);
		}
	}

	async function handleNotificationClick(notification: Notification) {
		// Mark as read
		if (!notification.read) {
			try {
				await markRead(notification.id);
				notifications = notifications.map((n) =>
					n.id === notification.id ? { ...n, read: true } : n
				);
				unreadCount = Math.max(0, unreadCount - 1);
			} catch (e) {
				// Continue anyway
			}
		}

		// Navigate and close
		if (notification.link) {
			goto(notification.link);
		}
		showNotifications = false;
	}

	function getNotificationIcon(type: Notification['type']): { path: string; color: string } {
		switch (type) {
			case 'new_content':
			case 'request_approved':
				return {
					path: 'M5 13l4 4L19 7',
					color: 'text-green-400'
				};
			case 'request_denied':
				return {
					path: 'M6 18L18 6M6 6l12 12',
					color: 'text-red-400'
				};
			case 'download_complete':
				return {
					path: 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
					color: 'text-blue-400'
				};
			case 'download_failed':
				return {
					path: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z',
					color: 'text-red-400'
				};
			default:
				return {
					path: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9',
					color: 'text-text-muted'
				};
		}
	}

	function formatTimeAgo(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	async function loadSystemStatus() {
		try {
			const newStatus = await getSystemStatus();
			// Check if pending requests increased (skip first load when prevPendingRequests is -1)
			if (prevPendingRequests >= 0 && newStatus.pendingRequests > prevPendingRequests) {
				hasNewRequest = true;
			}
			prevPendingRequests = newStatus.pendingRequests;
			systemStatus = newStatus;
		} catch (e) {
			console.debug('Failed to load system status:', e);
		}
	}

	function handleActivityClick() {
		hasNewRequest = false;
		goto('/activity');
	}

	// Search result types
	type LibrarySearchResult = (Movie | Show | Artist | Book) & { type: string; source: string; score: number };
	type DiscoverSearchResult = DiscoverItem & { source: string };

	// Search state
	let query = $state('');
	let libraryResults = $state<LibrarySearchResult[]>([]);
	let discoverResults = $state<DiscoverSearchResult[]>([]);
	let loading = $state(false);
	let selectedIndex = $state(0);
	let showSearchDropdown = $state(false);
	let searchInputRef: HTMLInputElement;
	let debounceTimer: ReturnType<typeof setTimeout>;
	let searchFilter = $state<'all' | 'movies' | 'tv' | 'music' | 'books'>('all');

	// Cache library data
	let cachedMovies: Movie[] | null = null;
	let cachedShows: Show[] | null = null;
	let cachedArtists: Artist[] | null = null;
	let cachedBooks: Book[] | null = null;
	let cacheTime = 0;
	const CACHE_DURATION = 60000;

	let allResults = $derived([...libraryResults, ...discoverResults]);

	// Navigation items
	const navItems = [
		{ href: '/', label: 'Home', match: (path: string) => path === '/' },
		{ href: '/library', label: 'Library', match: (path: string) => path.startsWith('/library') || path.startsWith('/movies') || path.startsWith('/tv') || path.startsWith('/music') || path.startsWith('/books') },
		{ href: '/playlists', label: 'Playlists', match: (path: string) => path.startsWith('/playlists') },
		{ href: '/explore', label: 'Explore', match: (path: string) => path.startsWith('/explore') },
	];

	const isCalendarActive = $derived($page.url.pathname.startsWith('/calendar'));

	// Close dropdowns on navigation
	$effect(() => {
		const url = $page.url.pathname;
		showSearchDropdown = false;
		showUserMenu = false;
		query = '';
	});

	const filterOptions = [
		{ id: 'all', label: 'All' },
		{ id: 'movies', label: 'Movies' },
		{ id: 'tv', label: 'TV' },
	] as const;

	function handleLogout() {
		showUserMenu = false;
		onLogout?.();
	}

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
	}

	function closeUserMenu() {
		showUserMenu = false;
	}

	function toggleProfileMenu() {
		showProfileMenu = !showProfileMenu;
	}

	function closeProfileMenu() {
		showProfileMenu = false;
	}

	async function switchProfile(profile: Profile) {
		showProfileMenu = false;
		await profileStore.select(profile.id);
	}

	function getProfileAvatarUrl(profile: Profile | null): string {
		return profile?.avatarUrl || AVATARS[0];
	}

	function isActive(item: typeof navItems[0]): boolean {
		return item.match($page.url.pathname);
	}

	// Search functions
	function handleSearchFocus() {
		showSearchDropdown = true;
	}

	function handleSearchBlur(e: FocusEvent) {
		setTimeout(() => {
			if (!document.activeElement?.closest('.search-dropdown')) {
				showSearchDropdown = false;
			}
		}, 200);
	}

	async function loadLibraryCache() {
		const now = Date.now();
		if (cachedMovies && cachedShows && (now - cacheTime) < CACHE_DURATION) {
			return;
		}
		const [moviesRes, showsRes, artistsRes, booksRes] = await Promise.all([
			fetch(`/api/movies`, { credentials: 'include' }),
			fetch(`/api/shows`, { credentials: 'include' }),
			fetch(`/api/artists`, { credentials: 'include' }),
			fetch(`/api/books`, { credentials: 'include' }),
		]);
		cachedMovies = moviesRes.ok ? await moviesRes.json() : [];
		cachedShows = showsRes.ok ? await showsRes.json() : [];
		cachedArtists = artistsRes.ok ? await artistsRes.json() : [];
		cachedBooks = booksRes.ok ? await booksRes.json() : [];
		cacheTime = now;
	}

	function handleSearchInput() {
		clearTimeout(debounceTimer);
		selectedIndex = 0;
		showSearchDropdown = true;

		if (!query.trim()) {
			libraryResults = [];
			discoverResults = [];
			return;
		}

		debounceTimer = setTimeout(async () => {
			loading = true;
			try {
				const includeMovies = searchFilter === 'all' || searchFilter === 'movies';
				const includeTV = searchFilter === 'all' || searchFilter === 'tv';
				const includeMusic = searchFilter === 'all' || searchFilter === 'music';
				const includeBooks = searchFilter === 'all' || searchFilter === 'books';

				const tmdbPromises: Promise<Response>[] = [];
				if (isAdmin && includeMovies) {
					tmdbPromises.push(fetch(`/api/tmdb/search/movie?q=${encodeURIComponent(query)}`, { credentials: 'include' }));
				}
				if (isAdmin && includeTV) {
					tmdbPromises.push(fetch(`/api/tmdb/search/tv?q=${encodeURIComponent(query)}`, { credentials: 'include' }));
				}

				await loadLibraryCache();
				const tmdbResponses = await Promise.all(tmdbPromises);

				let tmdbMovies: TmdbMovieResult[] = [];
				let tmdbShows: TmdbTVResult[] = [];
				let responseIdx = 0;
				if (isAdmin && includeMovies && tmdbResponses[responseIdx]) {
					tmdbMovies = tmdbResponses[responseIdx].ok ? await tmdbResponses[responseIdx].json() : [];
					responseIdx++;
				}
				if (isAdmin && includeTV && tmdbResponses[responseIdx]) {
					tmdbShows = tmdbResponses[responseIdx].ok ? await tmdbResponses[responseIdx].json() : [];
				}

				const searchTerm = query.trim();

				const scoredMovies = includeMovies ? (cachedMovies || [])
					.map((m) => ({ ...m, type: 'movie' as const, source: 'library' as const, score: searchScore(m.title, searchTerm) }))
					.filter((m) => m.score > 0) : [];
				const scoredShows = includeTV ? (cachedShows || [])
					.map((s) => ({ ...s, type: 'show' as const, source: 'library' as const, score: searchScore(s.title, searchTerm) }))
					.filter((s) => s.score > 0) : [];
				const scoredArtists = includeMusic ? (cachedArtists || [])
					.map((a) => ({ ...a, type: 'artist' as const, source: 'library' as const, score: searchScore(a.name, searchTerm) }))
					.filter((a) => a.score > 0) : [];
				const scoredBooks = includeBooks ? (cachedBooks || [])
					.map((b) => ({ ...b, type: 'book' as const, source: 'library' as const, score: searchScore(b.title, searchTerm) }))
					.filter((b) => b.score > 0) : [];

				libraryResults = [...scoredMovies, ...scoredShows, ...scoredArtists, ...scoredBooks]
					.sort((a, b) => b.score - a.score)
					.slice(0, 10);

				const libraryTmdbIds = new Set([
					...(cachedMovies || []).map((m) => m.tmdbId),
					...(cachedShows || []).map((s) => s.tmdbId),
				]);

				const allDiscoverResults = [
					...tmdbMovies
						.filter((m) => !libraryTmdbIds.has(m.id))
						.map((m) => ({ ...m, type: 'movie' as const, source: 'discover' as const, popularity: m.popularity || 0 })),
					...tmdbShows
						.filter((s) => !libraryTmdbIds.has(s.id))
						.map((s) => ({ ...s, type: 'show' as const, source: 'discover' as const, popularity: s.popularity || 0 })),
				]
					.sort((a, b) => b.popularity - a.popularity)
					.slice(0, 10);

				discoverResults = allDiscoverResults;
			} catch (err) {
				console.error('Search error:', err);
				libraryResults = [];
				discoverResults = [];
			} finally {
				loading = false;
			}
		}, 300);
	}

	function setFilter(filter: typeof searchFilter) {
		searchFilter = filter;
		if (query.trim()) {
			handleSearchInput();
		}
	}

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			showSearchDropdown = false;
			query = '';
			searchInputRef?.blur();
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = Math.min(selectedIndex + 1, allResults.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = Math.max(selectedIndex - 1, 0);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (allResults.length > 0 && allResults[selectedIndex]) {
				navigateToResult(allResults[selectedIndex]);
			}
		}
	}

	function navigateToResult(item: LibrarySearchResult | DiscoverSearchResult) {
		showSearchDropdown = false;
		query = '';
		let path = '';
		if (item.source === 'library') {
			if (item.type === 'movie') path = `/movies/${item.id}`;
			else if (item.type === 'show') path = `/tv/${item.id}`;
			else if (item.type === 'artist') path = `/music/artists/${item.id}`;
			else if (item.type === 'book') path = `/books/${item.id}`;
		} else {
			if (item.type === 'movie') path = `/explore/movie/${item.id}`;
			else if (item.type === 'show') path = `/explore/show/${item.id}`;
		}
		if (path) goto(path);
	}

	function getTypeLabel(type: string): string {
		switch (type) {
			case 'movie': return 'Movie';
			case 'show': return 'TV Show';
			case 'artist': return 'Artist';
			case 'book': return 'Book';
			default: return type;
		}
	}

	function getTypeIcon(type: string): string {
		switch (type) {
			case 'movie': return 'M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z';
			case 'show': return 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z';
			case 'artist': return 'M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3';
			case 'book': return 'M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253';
			default: return 'M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z';
		}
	}

	async function handleInlineRequest(e: MouseEvent, item: DiscoverSearchResult) {
		e.preventDefault();
		e.stopPropagation();
		const dateStr = item.release_date || item.first_air_date;
		try {
			await createRequest({
				type: item.type === 'movie' ? 'movie' : 'show',
				tmdbId: item.id,
				title: item.title || item.name,
				year: dateStr ? parseInt(dateStr.substring(0, 4)) : undefined,
				overview: item.overview,
				posterPath: item.poster_path
			});
			discoverResults = discoverResults.map(r =>
				r.id === item.id ? { ...r, requested: true } : r
			);
		} catch (err) {
			console.error('Failed to create request:', err);
		}
	}
</script>

<header class="fixed top-0 left-0 right-0 h-16 flex items-center justify-between px-10 z-40" style="background: linear-gradient(to bottom, rgba(10,10,10,0.95) 0%, rgba(10,10,10,0.8) 60%, transparent 100%); backdrop-filter: blur(10px);">
	<!-- Left: Logo + Nav -->
	<div class="flex items-center gap-8">
		<!-- Logo Banner -->
		<button onclick={() => goto('/')} class="flex items-center mr-4">
			<img src="/outpost-banner.png" alt="Outpost" class="h-10 w-auto object-contain" />
		</button>

		<!-- Navigation Tabs -->
		<nav class="flex items-center gap-1">
			{#each navItems as item}
				<button
					onclick={() => goto(item.href)}
					class="px-5 py-2 text-sm font-medium rounded-full transition-all flex items-center
						{isActive(item)
							? 'text-black bg-amber-400'
							: 'text-text-secondary hover:text-cream hover:bg-cream/10'}"
				>
					{item.label}
				</button>
			{/each}
		</nav>
	</div>

	<!-- Center: Search -->
	<div class="flex-1 max-w-2xl mx-8 relative">
		<div class="relative z-50">
			<div class="flex items-center h-11 w-full px-5 gap-3 bg-bg-input rounded-2xl {showSearchDropdown ? 'rounded-b-none' : ''}">
				<svg class="w-4 h-4 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<input
					bind:this={searchInputRef}
					bind:value={query}
					oninput={handleSearchInput}
					onfocus={handleSearchFocus}
					onblur={handleSearchBlur}
					onkeydown={handleSearchKeydown}
					type="text"
					placeholder="Search..."
					class="flex-1 bg-transparent text-text-primary placeholder-text-muted text-sm [outline:none] [border:none] focus:[outline:none] focus:[box-shadow:none]"
				/>
				{#if loading}
					<div class="spinner-sm text-text-muted"></div>
				{/if}
			</div>

			<!-- Search Dropdown -->
			{#if showSearchDropdown}
				<button
					class="fixed inset-0 z-40"
					onclick={() => { showSearchDropdown = false; }}
					aria-label="Close search"
				></button>
				<div class="search-dropdown absolute left-0 right-0 top-full bg-bg-input rounded-b-2xl z-50 shadow-2xl overflow-hidden">
					<!-- Filter Tabs -->
					<div class="flex items-center gap-1 p-3 border-t border-white/5">
						{#each filterOptions as option}
							<button
								onclick={() => setFilter(option.id)}
								class="px-4 py-1.5 text-xs font-medium rounded-full transition-all {searchFilter === option.id ? 'bg-amber-400 text-black' : 'text-text-muted hover:text-cream hover:bg-cream/10'}"
							>
								{option.label}
							</button>
						{/each}
					</div>
					<!-- Results -->
					<div class="search-results max-h-[60vh] overflow-y-auto">
						{#if loading}
							<div class="p-6 flex items-center justify-center gap-2">
								<div class="spinner-md text-text-muted"></div>
								<span class="text-sm text-text-secondary">Searching...</span>
							</div>
						{:else if !query.trim()}
							<div class="p-6 text-center text-text-muted text-sm">
								Type to search your library{isAdmin ? ' and discover new content' : ''}
							</div>
						{:else if libraryResults.length > 0 || discoverResults.length > 0}
							<!-- Library results -->
							{#if libraryResults.length > 0}
								<div class="p-2">
									<p class="px-3 py-1.5 text-xs text-text-muted uppercase tracking-wider flex items-center gap-2">
										<svg class="w-3 h-3 text-success" fill="currentColor" viewBox="0 0 24 24">
											<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
										</svg>
										In Library
									</p>
									{#each libraryResults as item, i}
										<button
											onclick={() => navigateToResult(item)}
											class="w-full flex items-center gap-3 p-2 rounded-lg transition-all
												{selectedIndex === i ? 'bg-cream/10' : 'hover:bg-cream/10'}"
										>
											<div class="w-10 h-14 bg-glass rounded-lg overflow-hidden flex-shrink-0 {item.type === 'artist' ? '!rounded-full !w-10 !h-10' : ''}">
												{#if item.posterPath || item.imagePath || item.coverPath}
													<img
														src={getImageUrl(item.posterPath || item.imagePath || item.coverPath)}
														alt=""
														class="w-full h-full object-cover"
													/>
												{:else}
													<div class="w-full h-full flex items-center justify-center text-text-muted">
														<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getTypeIcon(item.type)} />
														</svg>
													</div>
												{/if}
											</div>
											<div class="flex-1 text-left min-w-0">
												<p class="text-text-primary text-sm font-medium truncate">{item.title || item.name}</p>
												<p class="text-xs text-text-muted">
													{getTypeLabel(item.type)}
													{#if item.year || item.release_date || item.first_air_date}
														· {item.year || new Date(item.release_date || item.first_air_date).getFullYear()}
													{/if}
												</p>
											</div>
											<svg class="w-4 h-4 text-success flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
												<path d="M8 5v14l11-7z" />
											</svg>
										</button>
									{/each}
								</div>
							{/if}

							<!-- Discover results -->
							{#if discoverResults.length > 0}
								<div class="p-2 {libraryResults.length > 0 ? 'border-t border-border-subtle' : ''}">
									<p class="px-3 py-1.5 text-xs text-text-muted uppercase tracking-wider flex items-center gap-2">
										<svg class="w-3 h-3 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
										Request
									</p>
									{#each discoverResults as item, i}
										{@const globalIndex = libraryResults.length + i}
										<div
											class="flex items-center gap-3 p-2 rounded-lg transition-all cursor-pointer
												{selectedIndex === globalIndex ? 'bg-cream/10' : 'hover:bg-cream/10'}"
											onclick={() => navigateToResult(item)}
											role="button"
											tabindex="0"
										>
											<div class="w-10 h-14 bg-glass rounded-lg overflow-hidden flex-shrink-0">
												{#if item.poster_path}
													<img
														src={`https://image.tmdb.org/t/p/w92${item.poster_path}`}
														alt=""
														class="w-full h-full object-cover"
													/>
												{:else}
													<div class="w-full h-full flex items-center justify-center text-text-muted">
														<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
														</svg>
													</div>
												{/if}
											</div>
											<div class="flex-1 text-left min-w-0">
												<p class="text-text-primary text-sm font-medium truncate">{item.title || item.name}</p>
												<p class="text-xs text-text-muted">
													{item.type === 'movie' ? 'Movie' : 'TV Show'}
													{#if item.release_date || item.first_air_date}
														· {(item.release_date || item.first_air_date).substring(0, 4)}
													{/if}
												</p>
											</div>
											{#if item.requested}
												<span class="text-xs text-text-muted">Requested</span>
											{:else}
												<button onclick={(e) => handleInlineRequest(e, item)} class="btn-icon-circle-sm" title="Request">
													<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
													</svg>
												</button>
											{/if}
										</div>
									{/each}
								</div>
							{/if}
						{:else}
							<div class="p-6 text-center">
								<svg class="w-12 h-12 mx-auto text-text-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
								</svg>
								<p class="text-text-secondary text-sm">No results found for "{query}"</p>
								<p class="text-text-muted text-xs mt-1">Try a different search term or filter</p>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Right: Actions + User -->
	<div class="flex items-center gap-2">
		<!-- Activity Indicator (running tasks) -->
		{#if systemStatus && systemStatus.runningTasks.length > 0}
			<div
				class="btn-icon-circle !text-blue-400"
				title={"Running: " + systemStatus.runningTasks.join(", ")}
			>
				<div class="spinner-sm text-blue-400"></div>
			</div>
		{/if}

		<!-- Activity (combined: queue, wanted, requests) -->
		<button
			onclick={handleActivityClick}
			class="btn-icon-circle relative {hasNewRequest ? 'ring-2 ring-amber-400 ring-offset-2 ring-offset-[#0a0a0a]' : ''}"
			title="Activity"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
			</svg>
			{#if (systemStatus?.activeDownloads || 0) + (systemStatus?.pendingRequests || 0) > 0}
				<span class="absolute -top-1 -right-1 min-w-[18px] h-[18px] rounded-full bg-amber-500 text-white text-[10px] font-bold flex items-center justify-center px-1 {hasNewRequest ? 'animate-pulse' : ''}">
					{(systemStatus?.activeDownloads || 0) + (systemStatus?.pendingRequests || 0)}
				</span>
			{/if}
		</button>

		<!-- Settings (admin only) -->
		{#if isAdmin}
			<button
				onclick={() => goto('/settings')}
				class="btn-icon-circle"
				title="Settings"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
			</button>
		{/if}

		<!-- Calendar -->
		<button
			onclick={() => goto('/calendar')}
			class="btn-icon-circle {isCalendarActive ? '!bg-amber-400 !text-black' : ''}"
			title="Calendar"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
			</svg>
		</button>

		<!-- Notifications -->
		<div class="relative">
			<button
				onclick={toggleNotifications}
				class="btn-icon-circle relative"
				title="Notifications"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
				</svg>
				{#if unreadCount > 0}
					<span class="absolute -top-1 -right-1 min-w-[18px] h-[18px] flex items-center justify-center bg-red-500 text-white text-[10px] font-semibold rounded-full px-1 animate-pulse">
						{unreadCount > 99 ? '99+' : unreadCount}
					</span>
				{/if}
			</button>

			{#if showNotifications}
				<button
					class="fixed inset-0 z-40"
					onclick={closeNotifications}
					aria-label="Close notifications"
				></button>

				<div class="absolute right-0 top-full mt-2 w-[360px] max-h-[480px] rounded-xl bg-bg-card backdrop-blur-xl border border-border-subtle z-50 shadow-2xl overflow-hidden">
					<div class="flex items-center justify-between px-4 py-3 border-b border-border-subtle">
						<h3 class="text-sm font-semibold text-text-primary">Notifications</h3>
						{#if unreadCount > 0}
							<button
								onclick={handleMarkAllRead}
								class="text-xs text-text-muted hover:text-cream transition-colors"
							>
								Mark all read
							</button>
						{/if}
					</div>

					<div class="overflow-y-auto max-h-[400px]">
						{#if loadingNotifications}
							<div class="flex items-center justify-center py-8">
								<div class="w-5 h-5 border-2 border-cream/30 border-t-cream rounded-full animate-spin"></div>
							</div>
						{:else if notifications.length === 0}
							<div class="flex flex-col items-center justify-center py-8 text-text-muted">
								<svg class="w-10 h-10 mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
								</svg>
								<span class="text-sm">No notifications yet</span>
							</div>
						{:else}
							{#each notifications as notification}
								{@const icon = getNotificationIcon(notification.type)}
								<button
									onclick={() => handleNotificationClick(notification)}
									class="w-full flex items-start gap-3 px-4 py-3 hover:bg-cream/5 transition-colors text-left {!notification.read ? 'bg-cream/5' : ''}"
								>
									<div class="flex-shrink-0 w-8 h-8 rounded-full bg-bg-elevated flex items-center justify-center {icon.color}">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={icon.path} />
										</svg>
									</div>
									<div class="flex-1 min-w-0">
										<p class="text-sm font-medium text-text-primary truncate">{notification.title}</p>
										<p class="text-xs text-text-muted line-clamp-2">{notification.message}</p>
										<p class="text-xs text-text-muted/60 mt-1">{formatTimeAgo(notification.createdAt)}</p>
									</div>
									{#if !notification.read}
										<div class="flex-shrink-0 w-2 h-2 rounded-full bg-blue-500 mt-2"></div>
									{/if}
								</button>
							{/each}
						{/if}
					</div>
				</div>
			{/if}
		</div>

		<!-- GitHub Sponsor -->
		<a
			href="https://github.com/sponsors/avbirk83"
			target="_blank"
			rel="noopener noreferrer"
			class="btn-icon-circle hover:!text-pink-400 hover:!bg-pink-500/20 group"
			title="Support Outpost"
		>
			<svg class="w-[18px] h-[18px] group-hover:fill-pink-400 transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
			</svg>
		</a>

		<!-- Profile Avatar/Menu -->
		<div class="relative">
			<button
				onclick={toggleProfileMenu}
				class="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-cream/10 transition-all"
				title={activeProfile ? activeProfile.name : 'Select Profile'}
			>
				<div class="w-8 h-8 rounded-lg overflow-hidden ring-2 ring-cream/30 flex-shrink-0">
					{#if activeProfile?.avatarUrl}
						<img src={activeProfile.avatarUrl} alt="" class="w-full h-full object-cover" />
					{:else}
						<div class="w-full h-full bg-bg-tertiary flex items-center justify-center text-text-muted">
							<span class="text-sm font-semibold uppercase">
								{activeProfile?.name?.charAt(0) || username?.charAt(0) || 'U'}
							</span>
						</div>
					{/if}
				</div>
				<svg class="w-3 h-3 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>

			{#if showProfileMenu}
				<button
					class="fixed inset-0 z-40"
					onclick={closeProfileMenu}
					aria-label="Close menu"
				></button>

				<div class="absolute right-0 top-full mt-2 min-w-[220px] rounded-xl bg-bg-card backdrop-blur-xl border border-border-subtle py-1 z-50 shadow-2xl">
					<!-- Current user info -->
					<div class="px-3 py-2 border-b border-border-subtle">
						<p class="text-sm font-medium text-text-primary">{username}</p>
						<p class="text-xs text-text-muted">{isAdmin ? 'Administrator' : 'User'}</p>
					</div>

					<!-- Profile switcher -->
					{#if profiles.length > 1}
						<div class="py-1 border-b border-border-subtle">
							<p class="px-3 py-1 text-xs text-text-muted uppercase tracking-wider">Switch Profile</p>
							{#each profiles.filter(p => p.id !== activeProfile?.id) as profile}
								<button
									onclick={() => switchProfile(profile)}
									class="w-full flex items-center gap-3 px-3 py-2 mx-1 rounded-lg text-text-secondary hover:text-cream hover:bg-cream/10 transition-all"
									style="width: calc(100% - 8px);"
								>
									<div class="w-7 h-7 rounded-lg overflow-hidden flex-shrink-0">
										{#if profile.avatarUrl}
											<img src={profile.avatarUrl} alt="" class="w-full h-full object-cover" />
										{:else}
											<div class="w-full h-full bg-bg-tertiary flex items-center justify-center text-text-muted text-xs font-semibold">
												{profile.name.charAt(0).toUpperCase()}
											</div>
										{/if}
									</div>
									<span class="text-sm">{profile.name}</span>
									{#if profile.isKid}
										<span class="text-[10px] px-1.5 py-0.5 bg-green-600/20 text-green-400 rounded">Kids</span>
									{/if}
								</button>
							{/each}
						</div>
					{/if}

					<!-- Manage profiles link -->
					<button
						onclick={() => { closeProfileMenu(); goto('/profiles'); }}
						class="w-full flex items-center gap-2.5 px-3 py-2 mx-1 mt-1 rounded-lg text-text-muted hover:text-cream hover:bg-cream/10 transition-all"
						style="width: calc(100% - 8px);"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
						</svg>
						<span class="text-sm">Manage Profiles</span>
					</button>

					{#if isAdmin}
						<button
							onclick={() => { closeProfileMenu(); goto('/users'); }}
							class="w-full flex items-center gap-2.5 px-3 py-2 mx-1 rounded-lg text-text-muted hover:text-cream hover:bg-cream/10 transition-all"
							style="width: calc(100% - 8px);"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
							</svg>
							<span class="text-sm">Manage Users</span>
						</button>
					{/if}

					<div class="my-1 mx-2 h-px bg-border-subtle"></div>
					<button
						onclick={handleLogout}
						class="w-full flex items-center gap-2.5 px-3 py-2 mx-1 rounded-lg text-text-muted hover:text-cream hover:bg-cream/10 transition-all"
						style="width: calc(100% - 8px);"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
						</svg>
						<span class="text-sm">Sign out</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>

<style>
	.search-results {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.search-results::-webkit-scrollbar {
		display: none;
	}
</style>
