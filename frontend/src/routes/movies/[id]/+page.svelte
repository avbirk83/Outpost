<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getMovie, refreshMovieMetadata, getImageUrl, getTmdbImageUrl, createWantedItem,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getMediaInfo, getMovieSuggestions, type Movie, type QualityProfile, type MediaInfo, type TMDBMovieResult
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import Dropdown from '$lib/components/Dropdown.svelte';

	let movie: Movie | null = $state(null);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let profiles: QualityProfile[] = $state([]);
	let user = $state<{ role: string } | null>(null);
	let isWatched = $state(false);
	let togglingWatched = $state(false);
	let mediaInfo: MediaInfo | null = $state(null);
	let selectedSubtitle = $state<number | null>(null);
	let selectedAudio = $state<number>(0);
	let showManageMenu = $state(false);
	let inWatchlist = $state(false);
	let showTrailerModal = $state(false);
	let selectedVideo = $state(0);
	let recommendations: TMDBMovieResult[] = $state([]);
	let castScrollContainer: HTMLElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);
	let crewScrollContainer: HTMLElement;
	let canScrollCrewLeft = $state(false);
	let canScrollCrewRight = $state(true);

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

	function updateCrewScrollState() {
		if (!crewScrollContainer) return;
		canScrollCrewLeft = crewScrollContainer.scrollLeft > 0;
		canScrollCrewRight = crewScrollContainer.scrollLeft < crewScrollContainer.scrollWidth - crewScrollContainer.clientWidth - 10;
	}

	function scrollCrew(direction: 'left' | 'right') {
		if (!crewScrollContainer) return;
		const scrollAmount = 300;
		crewScrollContainer.scrollBy({
			left: direction === 'left' ? -scrollAmount : scrollAmount,
			behavior: 'smooth'
		});
		setTimeout(updateCrewScrollState, 350);
	}

	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			movie = await getMovie(id);
			if (user?.role === 'admin') {
				profiles = await getQualityProfiles();
			}
			const watchStatus = await getWatchStatus('movie', id);
			isWatched = watchStatus.watched;
			mediaInfo = await getMediaInfo('movie', id);
			const defaultSub = mediaInfo.subtitleTracks?.find(t => t.default);
			if (defaultSub) selectedSubtitle = defaultSub.index;
			// Load suggestions based on genres, excluding library items
			if (movie) {
				try {
					const suggestResult = await getMovieSuggestions(movie.id);
					recommendations = suggestResult.results.slice(0, 12);
				} catch { /* Suggestions are optional */ }
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load movie';
		} finally {
			loading = false;
		}
	});

	async function handleRefresh() {
		console.log('handleRefresh called', movie?.id);
		if (!movie) {
			console.log('No movie, returning');
			return;
		}
		showManageMenu = false;
		refreshing = true;
		try {
			console.log('Calling refreshMovieMetadata...');
			movie = await refreshMovieMetadata(movie.id);
			console.log('Refresh complete', movie);
		} catch (e) {
			console.error('Refresh failed', e);
			error = e instanceof Error ? e.message : 'Failed to refresh';
		} finally {
			refreshing = false;
		}
	}

	function handlePlay() {
		if (movie) {
			const params = new URLSearchParams();
			if (selectedSubtitle !== null) params.set('sub', selectedSubtitle.toString());
			goto(`/watch/movie/${movie.id}${params.toString() ? '?' + params.toString() : ''}`);
		}
	}

	async function handleToggleWatched() {
		if (!movie) return;
		togglingWatched = true;
		try {
			if (isWatched) {
				await markAsUnwatched('movie', movie.id);
				isWatched = false;
			} else {
				await markAsWatched('movie', movie.id, movie.runtime ? movie.runtime * 60 : 3600);
				isWatched = true;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update';
		} finally {
			togglingWatched = false;
		}
	}

	function formatRuntime(minutes?: number): string {
		if (!minutes) return '';
		const h = Math.floor(minutes / 60);
		const m = minutes % 60;
		return h > 0 ? `${h}h ${m}m` : `${m}m`;
	}

	function parseGenres(g?: string): string[] {
		if (!g) return [];
		try { return JSON.parse(g); } catch { return []; }
	}

	function parseCast(c?: string): Array<{ name: string; character: string; profile_path?: string }> {
		if (!c) return [];
		try { return JSON.parse(c); } catch { return []; }
	}

	function parseCrew(c?: string): Array<{ name: string; job: string; department: string; profile_path?: string }> {
		if (!c) return [];
		try { return JSON.parse(c); } catch { return []; }
	}

	function formatMoney(amount?: number): string {
		if (!amount || amount === 0) return '-';
		if (amount >= 1_000_000_000) return `$${(amount / 1_000_000_000).toFixed(1)}B`;
		if (amount >= 1_000_000) return `$${(amount / 1_000_000).toFixed(0)}M`;
		return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(amount);
	}

	function parseTrailers(t?: string): Array<{ key: string; name: string; type: string; site?: string; official?: boolean }> {
		if (!t) return [];
		try { return JSON.parse(t); } catch { return []; }
	}

	function getOfficialTrailer() {
		const trailers = parseTrailers(movie?.trailers);
		// Filter for official YouTube trailers
		const official = trailers.find(t => t.type === 'Trailer' && (t.site === 'YouTube' || !t.site) && t.official !== false);
		return official || trailers[0];
	}

	function getLanguageName(code?: string): string {
		if (!code) return '-';
		const languages: Record<string, string> = {
			en: 'English', es: 'Spanish', fr: 'French', de: 'German', it: 'Italian',
			pt: 'Portuguese', ja: 'Japanese', ko: 'Korean', zh: 'Chinese', ru: 'Russian',
			hi: 'Hindi', ar: 'Arabic', nl: 'Dutch', sv: 'Swedish', pl: 'Polish'
		};
		return languages[code] || code.toUpperCase();
	}

	function formatResolution(height?: number): string {
		if (!height) return '';
		if (height >= 2160) return '4K';
		if (height >= 1080) return '1080p';
		if (height >= 720) return '720p';
		return `${height}p`;
	}

	function formatAudioChannels(channels?: number): string {
		if (!channels) return '';
		if (channels >= 8) return '7.1';
		if (channels >= 6) return '5.1';
		if (channels === 2) return 'Stereo';
		return `${channels}ch`;
	}

	function formatFileSize(bytes?: number): string {
		if (!bytes) return '';
		if (bytes >= 1_000_000_000) return `${(bytes / 1_000_000_000).toFixed(1)} GB`;
		if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(0)} MB`;
		return `${bytes} bytes`;
	}

	function getCountryFlag(code?: string): string {
		if (!code) return '';
		const flags: Record<string, string> = {
			'US': '🇺🇸', 'USA': '🇺🇸', 'United States': '🇺🇸', 'United States of America': '🇺🇸',
			'UK': '🇬🇧', 'GB': '🇬🇧', 'United Kingdom': '🇬🇧',
			'CA': '🇨🇦', 'Canada': '🇨🇦',
			'AU': '🇦🇺', 'Australia': '🇦🇺',
			'FR': '🇫🇷', 'France': '🇫🇷',
			'DE': '🇩🇪', 'Germany': '🇩🇪',
			'JP': '🇯🇵', 'Japan': '🇯🇵',
			'KR': '🇰🇷', 'South Korea': '🇰🇷',
			'CN': '🇨🇳', 'China': '🇨🇳',
			'IN': '🇮🇳', 'India': '🇮🇳',
			'IT': '🇮🇹', 'Italy': '🇮🇹',
			'ES': '🇪🇸', 'Spain': '🇪🇸',
			'MX': '🇲🇽', 'Mexico': '🇲🇽',
			'BR': '🇧🇷', 'Brazil': '🇧🇷',
			'NZ': '🇳🇿', 'New Zealand': '🇳🇿'
		};
		return flags[code] || '';
	}

	// Get tags from genres
	function getTags(): string[] {
		return parseGenres(movie?.genres);
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
		<button class="ml-2 underline" onclick={() => error = null}>Dismiss</button>
	</div>
{:else if movie}
	<div class="space-y-6 -mt-22 -mx-6">
		<!-- ============================================
		     1. HERO SECTION
		     ============================================ -->
		<section class="relative min-h-[500px]">
			<!-- Backdrop - same as home page -->
			{#if movie.backdropPath}
				<img
					src={getImageUrl(movie.backdropPath)}
					alt=""
					class="absolute inset-0 w-full h-full object-cover pointer-events-none"
					style="object-position: center 25%;"
					draggable="false"
				/>
				<!-- Gradient overlays - same as home page -->
				<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/80 to-transparent pointer-events-none"></div>
				<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/30 pointer-events-none"></div>
			{/if}

			<!-- Hero Content: 3 columns -->
			<div class="relative z-10 px-6 pt-24 pb-8 flex gap-6">
				<!-- LEFT: Poster Card -->
				<div class="flex-shrink-0 w-64">
					<div class="liquid-card overflow-hidden">
						<!-- Poster -->
						<div class="relative aspect-[2/3] bg-bg-card">
							{#if movie.posterPath}
								<img
									src={getImageUrl(movie.posterPath)}
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
							<!-- Status Badge - same green checkmark as PosterCard -->
							<div class="absolute top-3 right-3">
								<div class="w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="In Library">
									<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
									</svg>
								</div>
							</div>
						</div>
						<!-- Ratings Row -->
						<div class="p-3 flex justify-around items-center border-t border-white/10">
							{#if movie.rating}
								<a href="https://www.themoviedb.org/movie/{movie.tmdbId}" target="_blank" class="flex items-center gap-1.5 hover:opacity-80 transition-opacity" title="TMDB Rating">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6 rounded" />
									<span class="text-base font-bold text-white">{movie.rating.toFixed(1)}</span>
								</a>
							{/if}
							<div class="flex items-center gap-1.5 opacity-40" title="Rotten Tomatoes (coming soon)">
								<img src="/icons/rottentomatoes.svg" alt="Rotten Tomatoes" class="w-6 h-6" />
								<span class="text-base font-bold">--</span>
							</div>
							<div class="flex items-center gap-1.5 opacity-40" title="Metacritic (coming soon)">
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
						<span class="text-text-secondary font-normal">({movie.year})</span>
					</h1>

					<!-- Meta line -->
					<div class="flex items-center gap-2 text-text-secondary mb-4">
						{#if movie.contentRating}
							<span class="px-2 py-0.5 border border-white/30 text-xs font-medium">{movie.contentRating}</span>
						{/if}
						{#if movie.runtime}
							<span>{formatRuntime(movie.runtime)}</span>
						{/if}
						{#if getTags().length > 0}
							<span>•</span>
							<span>{getTags().join(', ')}</span>
						{/if}
					</div>

					<!-- Tags (clickable pills) -->
					{#if getTags().length > 0}
						<div class="flex flex-wrap gap-2 mb-4">
							{#each getTags() as tag}
								<a href="/discover/movie?genre={encodeURIComponent(tag)}" class="liquid-tag text-sm">
									{tag}
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
					<div class="flex items-center gap-3 mb-5">
						<button
							onclick={() => inWatchlist = !inWatchlist}
							class="w-11 h-11 rounded-full flex items-center justify-center transition-all border {inWatchlist ? 'bg-blue-600 border-blue-500 text-white' : 'bg-white/10 border-white/20 text-white hover:bg-white/20'}"
							title="{inWatchlist ? 'Remove from' : 'Add to'} Watchlist"
						>
							{#if inWatchlist}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							{:else}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
							{/if}
						</button>

						<button
							onclick={handleToggleWatched}
							disabled={togglingWatched}
							class="w-11 h-11 rounded-full flex items-center justify-center transition-all border {isWatched ? 'bg-green-600 border-green-500 text-white' : 'bg-white/10 border-white/20 text-white hover:bg-white/20'}"
							title="{isWatched ? 'Mark as unwatched' : 'Mark as watched'}"
						>
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
							</svg>
						</button>

						{#if getOfficialTrailer()}
							<button
								onclick={() => showTrailerModal = true}
								class="w-11 h-11 rounded-full bg-red-600 border border-red-500 text-white flex items-center justify-center hover:bg-red-500 transition-all"
								title="Watch Trailer"
							>
								<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M8 5v14l11-7z" />
								</svg>
							</button>
						{/if}

						<!-- Manage dropdown -->
						<div class="relative">
							<button
								onclick={() => { console.log('Toggle menu', !showManageMenu); showManageMenu = !showManageMenu; }}
								class="w-11 h-11 rounded-full bg-white/10 border border-white/20 text-white flex items-center justify-center hover:bg-white/20 transition-all"
								title="Manage"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
								</svg>
							</button>
							{#if showManageMenu}
								<!-- Backdrop to close menu -->
								<button
									type="button"
									class="fixed inset-0 z-[55] cursor-default"
									onclick={() => showManageMenu = false}
									aria-label="Close menu"
								></button>
								<div class="absolute left-0 mt-2 w-48 py-1 z-[60] bg-[#141416] border border-white/10 rounded-2xl shadow-xl overflow-hidden">
									<button
										onclick={() => { console.log('Refresh button clicked'); handleRefresh(); }}
										class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors"
									>
										{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
									</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Images</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Find Subtitles</button>
									<div class="border-t border-white/10 my-1"></div>
									<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={() => showManageMenu = false}>Delete</button>
								</div>
							{/if}
						</div>
					</div>

					<!-- Playback selectors -->
					{#if mediaInfo}
						<div class="flex flex-wrap items-center gap-3">
							{#if mediaInfo.videoStreams?.length}
								<Dropdown
									icon="video"
									options={mediaInfo.videoStreams.map((v, i) => ({ value: i, label: `${formatResolution(v.height)} ${v.codec?.toUpperCase() || ''}` }))}
									value={selectedVideo}
									onchange={(v) => selectedVideo = v as number}
								/>
							{/if}
							{#if mediaInfo.audioStreams?.length}
								<Dropdown
									icon="audio"
									options={mediaInfo.audioStreams.map((a, i) => ({ value: i, label: `${a.language?.toUpperCase() || 'UNK'} ${a.codec?.toUpperCase() || ''} ${formatAudioChannels(a.channels)}` }))}
									value={selectedAudio}
									onchange={(v) => selectedAudio = v as number}
								/>
							{/if}
							<Dropdown
								icon="subtitles"
								options={[{ value: null, label: 'Off' }, ...(mediaInfo.subtitleTracks || []).map(s => ({ value: s.index, label: s.title || s.language || 'Unknown' }))]}
								value={selectedSubtitle}
								onchange={(v) => selectedSubtitle = v as number | null}
							/>
						</div>
					{/if}
				</div>

				<!-- RIGHT: Info Panel Card -->
				<div class="flex-shrink-0 w-72">
					<div class="liquid-card p-4 space-y-3 text-sm">
						<!-- Status -->
						<div class="flex justify-between">
							<span class="text-text-muted">Status</span>
							<span class="text-green-400 font-medium">{movie.status || 'Released'}</span>
						</div>

						<!-- Year -->
						{#if movie.year}
							<div class="flex justify-between">
								<span class="text-text-muted">🎬 Year</span>
								<span>{movie.year}</span>
							</div>
						{/if}

						<!-- Runtime -->
						{#if movie.runtime}
							<div class="flex justify-between">
								<span class="text-text-muted">Runtime</span>
								<span>{formatRuntime(movie.runtime)}</span>
							</div>
						{/if}

						<!-- Budget -->
						{#if movie.budget}
							<div class="flex justify-between">
								<span class="text-text-muted">Budget</span>
								<span>{formatMoney(movie.budget)}</span>
							</div>
						{/if}

						<!-- Revenue -->
						{#if movie.revenue}
							<div class="flex justify-between">
								<span class="text-text-muted">Revenue</span>
								<span class="text-green-400">{formatMoney(movie.revenue)}</span>
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
						{#if movie.country}
							<div class="flex justify-between">
								<span class="text-text-muted">Country</span>
								<span>{getCountryFlag(movie.country)} {movie.country}</span>
							</div>
						{/if}

						<!-- Director -->
						{#if movie.director}
							<div class="flex justify-between">
								<span class="text-text-muted">Director</span>
								<span class="truncate ml-2">{movie.director}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- Parental -->
						{#if movie.contentRating}
							<div class="flex justify-between items-center">
								<span class="text-text-muted">Parental</span>
								<span class="flex items-center gap-2">
									{movie.contentRating}
									{#if movie.imdbId}
										<a href="https://www.imdb.com/title/{movie.imdbId}/parentalguide" target="_blank" class="text-sky-400 hover:underline text-xs">
											↗
										</a>
									{/if}
								</span>
							</div>
						{/if}

						<!-- Added date -->
						{#if movie.addedAt}
							<div class="flex justify-between">
								<span class="text-text-muted">Added</span>
								<span class="text-xs">{new Date(movie.addedAt).toLocaleDateString()}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- External Links -->
						<div class="flex justify-center gap-3">
							{#if movie.tmdbId}
								<a href="https://www.themoviedb.org/movie/{movie.tmdbId}" target="_blank"
								   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on TMDB">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-7 h-7" />
								</a>
							{/if}
							{#if movie.imdbId}
								<a href="https://www.imdb.com/title/{movie.imdbId}" target="_blank"
								   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on IMDb">
									<img src="/icons/imdb.svg" alt="IMDb" class="w-7 h-7" />
								</a>
							{/if}
							<a href="https://trakt.tv/search/imdb/{movie.imdbId || ''}" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Trakt">
								<img src="/icons/trakt.svg" alt="Trakt" class="w-7 h-7" />
							</a>
							<a href="https://letterboxd.com/tmdb/{movie.tmdbId}" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Letterboxd">
								<img src="/icons/letterboxd.svg" alt="Letterboxd" class="w-7 h-7" />
							</a>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- ============================================
		     3. FILES SECTION
		     ============================================ -->
		{#if mediaInfo}
			<section class="px-6">
				<h2 class="text-lg font-semibold text-text-primary mb-3">Files</h2>
				<div class="flex gap-4">
					<!-- File card - same style as ContinueCard -->
					<button onclick={handlePlay} class="group relative w-72 md:w-80">
						<div class="relative aspect-video bg-bg-card overflow-hidden rounded-xl">
							<!-- Backdrop image -->
							{#if movie.backdropPath}
								<img
									src={getImageUrl(movie.backdropPath)}
									alt={movie.title}
									class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
								/>
							{:else if movie.posterPath}
								<img
									src={getImageUrl(movie.posterPath)}
									alt={movie.title}
									class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
								/>
							{:else}
								<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
									<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
								</div>
							{/if}

							<!-- Gradient overlay -->
							<div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>

							<!-- Play button - centered, on hover -->
							<div class="absolute inset-0 flex items-center justify-center">
								<div class="w-14 h-14 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transform scale-75 group-hover:scale-100 transition-all duration-300 border border-white/30">
									<svg class="w-7 h-7 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
										<path d="M8 5v14l11-7z" />
									</svg>
								</div>
							</div>

							<!-- Quality badge - top left -->
							{#if mediaInfo.videoStreams?.[0]}
								<div class="absolute top-3 left-3">
									<div class="liquid-badge-sm !bg-black/90 text-white">
										{formatResolution(mediaInfo.videoStreams[0].height)}
									</div>
								</div>
							{/if}

							<!-- Content overlay at bottom -->
							<div class="absolute bottom-0 left-0 right-0 p-4">
								<h3 class="text-base font-semibold text-white truncate">{movie.title}</h3>
								<p class="text-sm text-white/70 truncate mt-0.5">
									{#if mediaInfo.videoStreams?.[0]}
										{mediaInfo.videoStreams[0].codec?.toUpperCase()}
									{/if}
									{#if mediaInfo.audioStreams?.[0]}
										• {mediaInfo.audioStreams[0].codec?.toUpperCase()} {formatAudioChannels(mediaInfo.audioStreams[0].channels)}
									{/if}
								</p>
							</div>
						</div>
					</button>
				</div>
			</section>
		{/if}

		<!-- ============================================
		     4. CAST & CREW
		     ============================================ -->
		<!-- Cast - full width -->
		{#if parseCast(movie.cast).length > 0}
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
					class="flex gap-5 overflow-x-auto pb-2 scrollbar-thin"
				>
					{#each parseCast(movie.cast) as actor}
						<div class="flex-shrink-0 w-28 text-center">
							<div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10">
								{#if actor.profile_path}
									<img
										src={getTmdbImageUrl(actor.profile_path, 'w185')}
										alt={actor.name}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-3xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
										{actor.name.charAt(0)}
									</div>
								{/if}
							</div>
							<p class="mt-2 text-sm font-medium text-text-primary truncate">{actor.name}</p>
							<p class="text-xs text-text-muted truncate">{actor.character}</p>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Crew - full width -->
		{#if parseCrew(movie.crew).length > 0}
			<section class="px-6">
				<div class="flex items-center justify-between mb-3">
					<h2 class="text-lg font-semibold text-text-primary">Crew</h2>
					<div class="flex gap-1">
						<button
							onclick={() => scrollCrew('left')}
							disabled={!canScrollCrewLeft}
							class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
							aria-label="Scroll left"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
							</svg>
						</button>
						<button
							onclick={() => scrollCrew('right')}
							disabled={!canScrollCrewRight}
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
					bind:this={crewScrollContainer}
					onscroll={updateCrewScrollState}
					class="flex gap-5 overflow-x-auto pb-2 scrollbar-thin"
				>
					{#each parseCrew(movie.crew) as member}
						<div class="flex-shrink-0 w-28 text-center">
							<div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10">
								{#if member.profile_path}
									<img
										src={getTmdbImageUrl(member.profile_path, 'w185')}
										alt={member.name}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-3xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
										{member.name.charAt(0)}
									</div>
								{/if}
							</div>
							<p class="mt-2 text-sm font-medium text-text-primary truncate">{member.name}</p>
							<p class="text-xs text-text-muted truncate">{member.job}</p>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- ============================================
		     5. SUGGESTIONS ROW
		     ============================================ -->
		<section class="px-6">
			<h2 class="text-lg font-semibold text-text-primary mb-3">More Like This</h2>
			{#if recommendations.length > 0}
				<div class="flex gap-4 overflow-x-auto pb-2 scrollbar-thin">
					{#each recommendations as rec}
						<a href="/discover/movie/{rec.id}" class="flex-shrink-0 w-32 group">
							<div class="relative aspect-[2/3] rounded-lg overflow-hidden bg-bg-card">
								{#if rec.poster_path}
									<img
										src={getTmdbImageUrl(rec.poster_path, 'w342')}
										alt={rec.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
										<svg class="w-10 h-10 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
										</svg>
									</div>
								{/if}
								<!-- Hover overlay -->
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
							</div>
							<p class="mt-2 text-xs text-text-primary truncate">{rec.title}</p>
							<p class="text-[10px] text-text-muted">{rec.release_date?.split('-')[0] || ''}</p>
						</a>
					{/each}
				</div>
			{:else}
				<div class="flex gap-2">
					{#each getTags().slice(0, 3) as genre}
						<a href="/discover/movie?genre={encodeURIComponent(genre)}" class="liquid-btn-sm">
							Browse {genre} →
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<!-- ============================================
		     6. FOOTER (File Info)
		     ============================================ -->
		{#if mediaInfo}
			<footer class="px-6 pb-8">
				<div class="text-xs text-text-muted text-center">
					{#if mediaInfo.videoStreams?.[0]}
						{mediaInfo.videoStreams[0].codec?.toUpperCase()} {formatResolution(mediaInfo.videoStreams[0].height)}
					{/if}
					{#if mediaInfo.audioStreams?.[0]}
						• {mediaInfo.audioStreams[0].codec?.toUpperCase()} {formatAudioChannels(mediaInfo.audioStreams[0].channels)}
					{/if}
					{#if mediaInfo.container}
						• {mediaInfo.container.toUpperCase()}
					{/if}
				</div>
			</footer>
		{/if}
	</div>

	<!-- Trailer Modal -->
	{#if showTrailerModal && getOfficialTrailer()}
		{@const trailer = getOfficialTrailer()}
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<!-- Backdrop -->
			<button
				class="absolute inset-0 bg-black/90"
				onclick={() => showTrailerModal = false}
				aria-label="Close"
			></button>
			<!-- Modal -->
			<div class="relative w-full max-w-4xl mx-4 aspect-video">
				<iframe
					src="https://www.youtube.com/embed/{trailer.key}?autoplay=1"
					title={trailer.name}
					class="w-full h-full rounded-lg"
					frameborder="0"
					allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
					allowfullscreen
				></iframe>
				<button
					onclick={() => showTrailerModal = false}
					class="absolute -top-12 right-0 text-white hover:text-text-secondary"
				>
					✕ Close
				</button>
			</div>
		</div>
	{/if}

{/if}
