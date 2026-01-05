<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import { createRequest } from '$lib/api';

	interface Props {
		open?: boolean;
		onClose?: () => void;
	}

	let { open = false, onClose }: Props = $props();

	let query = $state('');
	let libraryResults = $state<any[]>([]);
	let discoverResults = $state<any[]>([]);
	let loading = $state(false);
	let selectedIndex = $state(0);
	let inputRef: HTMLInputElement;
	let debounceTimer: ReturnType<typeof setTimeout>;
	let portalTarget: HTMLElement | null = null;
	let overlayEl: HTMLElement;

	// Create portal container on mount
	onMount(() => {
		portalTarget = document.createElement('div');
		portalTarget.id = 'search-portal';
		document.body.appendChild(portalTarget);
	});

	onDestroy(() => {
		if (portalTarget && portalTarget.parentNode) {
			portalTarget.parentNode.removeChild(portalTarget);
		}
	});

	// Move overlay to portal when it exists
	$effect(() => {
		if (open && overlayEl && portalTarget) {
			portalTarget.appendChild(overlayEl);
		}
	});

	// Combined results for keyboard navigation
	let allResults = $derived([...libraryResults, ...discoverResults]);

	// Quick actions for empty state
	const quickActions = [
		{ label: 'Browse Library', href: '/library', icon: 'film' },
		{ label: 'Discover Movies', href: '/', icon: 'search' },
		{ label: 'View Requests', href: '/requests', icon: 'plus' },
	];

	$effect(() => {
		if (open && inputRef) {
			inputRef.focus();
			query = '';
			libraryResults = [];
			discoverResults = [];
			selectedIndex = 0;
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose?.();
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			const maxIndex = allResults.length > 0 ? allResults.length - 1 : quickActions.length - 1;
			selectedIndex = Math.min(selectedIndex + 1, maxIndex);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = Math.max(selectedIndex - 1, 0);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (allResults.length > 0 && allResults[selectedIndex]) {
				navigateToResult(allResults[selectedIndex]);
			} else if (quickActions[selectedIndex]) {
				goto(quickActions[selectedIndex].href);
				onClose?.();
			}
		}
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		selectedIndex = 0;

		if (!query.trim()) {
			libraryResults = [];
			discoverResults = [];
			return;
		}

		debounceTimer = setTimeout(async () => {
			loading = true;
			try {
				// Search local library AND TMDB discover in parallel
				const [moviesRes, showsRes, tmdbMoviesRes, tmdbShowsRes] = await Promise.all([
					fetch(`/api/movies?search=${encodeURIComponent(query)}&limit=5`, { credentials: 'include' }),
					fetch(`/api/shows?search=${encodeURIComponent(query)}&limit=5`, { credentials: 'include' }),
					fetch(`/api/discover/search/movie?query=${encodeURIComponent(query)}`, { credentials: 'include' }),
					fetch(`/api/discover/search/tv?query=${encodeURIComponent(query)}`, { credentials: 'include' }),
				]);

				const movies = moviesRes.ok ? await moviesRes.json() : [];
				const shows = showsRes.ok ? await showsRes.json() : [];
				const tmdbMovies = tmdbMoviesRes.ok ? (await tmdbMoviesRes.json()).results || [] : [];
				const tmdbShows = tmdbShowsRes.ok ? (await tmdbShowsRes.json()).results || [] : [];

				// Library results (items you own)
				libraryResults = [
					...movies.map((m: any) => ({ ...m, type: 'movie', source: 'library' })),
					...shows.map((s: any) => ({ ...s, type: 'show', source: 'library' })),
				].slice(0, 5);

				// Get library TMDB IDs to filter discover results
				const libraryTmdbIds = new Set([
					...movies.map((m: any) => m.tmdb_id),
					...shows.map((s: any) => s.tmdb_id),
				]);

				// Discover results (items you can request) - filter out library items
				discoverResults = [
					...tmdbMovies
						.filter((m: any) => !libraryTmdbIds.has(m.id) && !m.inLibrary)
						.map((m: any) => ({ ...m, type: 'movie', source: 'discover' })),
					...tmdbShows
						.filter((s: any) => !libraryTmdbIds.has(s.id) && !s.inLibrary)
						.map((s: any) => ({ ...s, type: 'show', source: 'discover' })),
				].slice(0, 5);
			} catch (err) {
				console.error('Search error:', err);
				libraryResults = [];
				discoverResults = [];
			} finally {
				loading = false;
			}
		}, 300);
	}

	function navigateToResult(item: any) {
		if (item.source === 'library') {
			if (item.type === 'movie') {
				goto(`/movies/${item.id}`);
			} else if (item.type === 'show') {
				goto(`/tv/${item.id}`);
			}
		} else {
			// Discover item - go to discover detail page
			if (item.type === 'movie') {
				goto(`/discover/movie/${item.id}`);
			} else if (item.type === 'show') {
				goto(`/discover/show/${item.id}`);
			}
		}
		onClose?.();
	}

	async function handleInlineRequest(e: MouseEvent, item: any) {
		e.preventDefault();
		e.stopPropagation();

		try {
			await createRequest({
				type: item.type === 'movie' ? 'movie' : 'show',
				tmdbId: item.id,
				title: item.title || item.name,
				year: item.releaseDate ? parseInt(item.releaseDate.substring(0, 4)) : undefined,
				overview: item.overview,
				posterPath: item.posterPath
			});

			// Update local state to show requested
			discoverResults = discoverResults.map(r =>
				r.id === item.id ? { ...r, requested: true } : r
			);
		} catch (err) {
			console.error('Failed to create request:', err);
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose?.();
		}
	}

	function getSelectedIndexForSection(item: any, index: number): number {
		if (item.source === 'library') {
			return index;
		} else {
			return libraryResults.length + index;
		}
	}
</script>

{#if open}
	<!-- Backdrop with blur - portaled to body -->
	<div
		bind:this={overlayEl}
		class="search-backdrop animate-fade-in"
		onclick={handleBackdropClick}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<!-- Search panel -->
		<div class="max-w-2xl mx-auto mt-[15vh] animate-slide-down px-4">
			<div class="search-panel overflow-hidden">
				<!-- Search input -->
				<div class="flex items-center gap-4 p-4 border-b border-white/5">
					<svg class="w-5 h-5 text-text-secondary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
					<input
						bind:this={inputRef}
						bind:value={query}
						oninput={handleInput}
						onkeydown={handleKeydown}
						type="text"
						placeholder="Search your library..."
						class="flex-1 bg-transparent text-white placeholder-white/40 outline-none text-lg"
					/>
					{#if loading}
						<div class="spinner-md text-cream"></div>
					{:else}
						<kbd class="liquid-badge-sm">ESC</kbd>
					{/if}
				</div>

				<!-- Results or quick actions -->
				<div class="max-h-[60vh] overflow-y-auto scrollbar-thin">
					{#if libraryResults.length > 0 || discoverResults.length > 0}
						<!-- Library results section -->
						{#if libraryResults.length > 0}
							<div class="p-2">
								<p class="px-3 py-2 text-xs text-text-muted uppercase tracking-wider flex items-center gap-2">
									<svg class="w-3.5 h-3.5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
									In Your Library
								</p>
								{#each libraryResults as item, i}
									<button
										onclick={() => navigateToResult(item)}
										class="w-full flex items-center gap-4 p-3 rounded-lg transition-all
											{selectedIndex === i ? 'bg-white/10' : 'hover:bg-white/5'}"
									>
										<!-- Poster thumbnail -->
										<div class="w-12 h-16 bg-bg-card rounded overflow-hidden flex-shrink-0 relative">
											{#if item.poster_path || item.posterPath}
												<img
													src={`/api/images/poster${item.poster_path || item.posterPath}`}
													alt=""
													class="w-full h-full object-cover"
												/>
											{:else}
												<div class="w-full h-full flex items-center justify-center text-text-muted">
													<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
													</svg>
												</div>
											{/if}
											<!-- Green library badge -->
											<div class="absolute -top-1 -right-1 w-4 h-4 bg-green-500 rounded-full flex items-center justify-center">
												<svg class="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
												</svg>
											</div>
										</div>

										<!-- Info -->
										<div class="flex-1 text-left">
											<p class="text-white font-medium">{item.title || item.name}</p>
											<p class="text-sm text-text-secondary">
												{item.type === 'movie' ? 'Movie' : 'TV Show'}
												{#if item.release_date || item.first_air_date}
													<span class="mx-1">·</span>
													{new Date(item.release_date || item.first_air_date).getFullYear()}
												{/if}
											</p>
										</div>

										<!-- Play icon -->
										<div class="liquid-btn-icon !p-2 !rounded-full !bg-white/10 !border-t-white/20">
											<svg class="w-4 h-4 text-white ml-0.5" fill="currentColor" viewBox="0 0 24 24">
												<path d="M8 5v14l11-7z" />
											</svg>
										</div>
									</button>
								{/each}
							</div>
						{/if}

						<!-- Discover results section (available to request) -->
						{#if discoverResults.length > 0}
							<div class="p-2 {libraryResults.length > 0 ? 'border-t border-white/10' : ''}">
								<p class="px-3 py-2 text-xs text-text-muted uppercase tracking-wider flex items-center gap-2">
									<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
									</svg>
									Available to Request
								</p>
								{#each discoverResults as item, i}
									{@const globalIndex = libraryResults.length + i}
									<div
										class="flex items-center gap-4 p-3 rounded-lg transition-all cursor-pointer
											{selectedIndex === globalIndex ? 'bg-white/10' : 'hover:bg-white/5'}"
										onclick={() => navigateToResult(item)}
										onkeydown={(e) => e.key === 'Enter' && navigateToResult(item)}
										role="button"
										tabindex="0"
									>
										<!-- Poster thumbnail -->
										<div class="w-12 h-16 bg-bg-card rounded overflow-hidden flex-shrink-0 relative">
											{#if item.posterPath}
												<img
													src={`https://image.tmdb.org/t/p/w92${item.posterPath}`}
													alt=""
													class="w-full h-full object-cover"
												/>
											{:else}
												<div class="w-full h-full flex items-center justify-center text-text-muted">
													<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
													</svg>
												</div>
											{/if}
											{#if item.requested}
												<!-- Requested badge -->
												<div class="absolute -top-1 -right-1 w-4 h-4 bg-white/50 rounded-full flex items-center justify-center">
													<svg class="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
													</svg>
												</div>
											{/if}
										</div>

										<!-- Info -->
										<div class="flex-1 text-left">
											<p class="text-white font-medium">{item.title || item.name}</p>
											<p class="text-sm text-text-secondary">
												{item.type === 'movie' ? 'Movie' : 'TV Show'}
												{#if item.releaseDate}
													<span class="mx-1">·</span>
													{item.releaseDate.substring(0, 4)}
												{/if}
											</p>
										</div>

										<!-- Request button or requested indicator -->
										{#if item.requested}
											<span class="liquid-tag !py-1 !px-3 !text-xs !bg-white/10 !border-t-white/20 text-white">
												Requested
											</span>
										{:else}
											<button
												onclick={(e) => handleInlineRequest(e, item)}
												class="liquid-btn-icon !p-2 !rounded-full"
												title="Request"
											>
												<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
												</svg>
											</button>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					{:else if query.trim() && !loading}
						<!-- No results -->
						<div class="p-8 text-center">
							<p class="text-text-secondary">No results found for "{query}"</p>
							<p class="mt-2 text-sm text-text-muted">Try searching with different keywords</p>
						</div>
					{:else}
						<!-- Quick actions -->
						<div class="p-2">
							<p class="px-3 py-2 text-xs text-text-muted uppercase tracking-wider">Quick Actions</p>
							{#each quickActions as action, i}
								<a
									href={action.href}
									onclick={() => onClose?.()}
									class="flex items-center gap-4 p-3 rounded-lg transition-all
										{selectedIndex === i ? 'bg-white/10' : 'hover:bg-white/5'}"
								>
									<div class="liquid-btn-icon !rounded-lg text-white/70">
										{#if action.icon === 'film'}
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
											</svg>
										{:else if action.icon === 'tv'}
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
											</svg>
										{:else if action.icon === 'music'}
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
											</svg>
										{:else if action.icon === 'plus'}
											<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
											</svg>
										{/if}
									</div>
									<span class="text-white">{action.label}</span>
									<svg class="ml-auto w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
									</svg>
								</a>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<!-- Keyboard hints -->
			<div class="flex items-center justify-center gap-6 mt-4 text-xs text-white/50">
				<span class="flex items-center gap-1">
					<kbd class="liquid-badge-sm">↑</kbd>
					<kbd class="liquid-badge-sm">↓</kbd>
					to navigate
				</span>
				<span class="flex items-center gap-1">
					<kbd class="liquid-badge-sm">Enter</kbd>
					to select
				</span>
				<span class="flex items-center gap-1">
					<kbd class="liquid-badge-sm">Esc</kbd>
					to close
				</span>
			</div>
		</div>
	</div>
{/if}
