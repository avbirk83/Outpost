<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getShow, refreshShowMetadata, getImageUrl, getTmdbImageUrl, createWantedItem,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getMediaInfo, getShowSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		type ShowDetail, type QualityProfile, type MediaInfo, type TMDBShowResult
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import Dropdown from '$lib/components/Dropdown.svelte';
	import PersonModal from '$lib/components/PersonModal.svelte';

	let show: ShowDetail | null = $state(null);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let profiles: QualityProfile[] = $state([]);
	let user = $state<{ role: string } | null>(null);
	let watchedEpisodes: Set<number> = $state(new Set());
	let togglingEpisode: number | null = $state(null);
	let showManageMenu = $state(false);
	let inWatchlist = $state(false);
	let watchlistLoading = $state(false);
	let showTrailerModal = $state(false);
	let selectedSeasonIndex = $state(0);
	let recommendations: TMDBShowResult[] = $state([]);
	let castScrollContainer: HTMLElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);
	let crewScrollContainer: HTMLElement;
	let canScrollCrewLeft = $state(false);
	let canScrollCrewRight = $state(true);
	let selectedPersonId = $state<number | null>(null);
	let selectedPersonName = $state<string>('');

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

	function handlePersonClick(person: { id?: number; name: string }) {
		if (person.id) {
			selectedPersonId = person.id;
			selectedPersonName = person.name;
		}
	}

	function closePersonModal() {
		selectedPersonId = null;
		selectedPersonName = '';
	}

	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			show = await getShow(id);
			if (user?.role === 'admin') {
				profiles = await getQualityProfiles();
			}
			// Check watchlist status using TMDB ID
			if (show?.tmdbId) {
				inWatchlist = await isInWatchlist(show.tmdbId, 'tv').catch(() => false);
			}
			// Load suggestions
			if (show) {
				try {
					const suggestResult = await getShowSuggestions(show.id);
					recommendations = suggestResult.results.slice(0, 12);
				} catch { /* Suggestions are optional */ }
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load show';
		} finally {
			loading = false;
		}
	});

	async function handleRefresh() {
		if (!show) return;
		showManageMenu = false;
		refreshing = true;
		try {
			show = await refreshShowMetadata(show.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh';
		} finally {
			refreshing = false;
		}
	}

	function handlePlayNext() {
		if (!show?.seasons) return;
		// Find first unwatched episode
		for (const season of show.seasons) {
			for (const ep of season.episodes || []) {
				if (!watchedEpisodes.has(ep.id)) {
					goto(`/watch/episode/${ep.id}`);
					return;
				}
			}
		}
		// All watched, play first episode
		const firstEp = show.seasons[0]?.episodes?.[0];
		if (firstEp) {
			goto(`/watch/episode/${firstEp.id}`);
		}
	}

	async function handleToggleEpisodeWatched(episodeId: number, runtime?: number) {
		togglingEpisode = episodeId;
		try {
			if (watchedEpisodes.has(episodeId)) {
				await markAsUnwatched('episode', episodeId);
				watchedEpisodes.delete(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			} else {
				await markAsWatched('episode', episodeId, runtime ? runtime * 60 : 2400);
				watchedEpisodes.add(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update watch status';
		} finally {
			togglingEpisode = null;
		}
	}

	async function handleToggleWatchlist() {
		if (!show?.tmdbId) return;
		watchlistLoading = true;
		try {
			if (inWatchlist) {
				await removeFromWatchlist(show.tmdbId, 'tv');
				inWatchlist = false;
			} else {
				await addToWatchlist(show.tmdbId, 'tv');
				inWatchlist = true;
			}
		} catch (e) {
			console.error('Failed to update watchlist:', e);
		} finally {
			watchlistLoading = false;
		}
	}

	const selectedSeason = $derived(show?.seasons?.[selectedSeasonIndex]);

	const nextEpisode = $derived(() => {
		if (!show?.seasons) return null;
		for (const season of show.seasons) {
			for (const ep of season.episodes || []) {
				if (!watchedEpisodes.has(ep.id)) {
					return { season: season.seasonNumber, episode: ep.episodeNumber };
				}
			}
		}
		return null;
	});

	const totalEpisodes = $derived(() => {
		if (!show?.seasons) return 0;
		return show.seasons.reduce((sum, s) => sum + (s.episodes?.length || 0), 0);
	});

	function parseGenres(g?: string): string[] {
		if (!g) return [];
		try { return JSON.parse(g) || []; } catch { return []; }
	}

	function parseCast(c?: string): Array<{ name: string; character: string; profile_path?: string }> {
		if (!c) return [];
		try { return JSON.parse(c) || []; } catch { return []; }
	}

	function parseCrew(c?: string): Array<{ name: string; job: string; department: string; profile_path?: string }> {
		if (!c) return [];
		try { return JSON.parse(c) || []; } catch { return []; }
	}

	function parseTrailers(t?: string): Array<{ key: string; name: string; type: string; site?: string; official?: boolean }> {
		if (!t) return [];
		try { return JSON.parse(t) || []; } catch { return []; }
	}

	function getOfficialTrailer() {
		const trailers = parseTrailers(show?.trailers);
		const official = trailers.find(t => t.type === 'Trailer' && (t.site === 'YouTube' || !t.site) && t.official !== false);
		return official || trailers[0];
	}

	function getLanguageName(code?: string): string {
		if (!code || code === 'und') return 'Unknown';
		try {
			const displayNames = new Intl.DisplayNames(['en'], { type: 'language' });
			return displayNames.of(code) || code;
		} catch {
			return code;
		}
	}

	function getStatusColor(status?: string): string {
		switch (status?.toLowerCase()) {
			case 'returning series':
			case 'in production':
				return 'text-green-400';
			case 'ended':
				return 'text-yellow-400';
			case 'canceled':
				return 'text-red-400';
			default:
				return 'text-text-secondary';
		}
	}

	function getCountryFlag(code?: string): string {
		if (!code || code.length !== 2) return '';
		return code.toUpperCase().split('').map(c => String.fromCodePoint(127397 + c.charCodeAt(0))).join('');
	}

	function getCountryName(code?: string): string {
		if (!code) return '';
		try {
			const displayNames = new Intl.DisplayNames(['en'], { type: 'region' });
			return displayNames.of(code.toUpperCase()) || code;
		} catch {
			return code;
		}
	}

	function getTotalRuntime(): string {
		if (!show?.seasons) return '-';
		let totalMinutes = 0;
		for (const season of show.seasons) {
			if (!season?.episodes) continue;
			for (const ep of season.episodes) {
				totalMinutes += ep.runtime || 0;
			}
		}
		if (totalMinutes === 0) return '-';
		const hours = Math.floor(totalMinutes / 60);
		const mins = totalMinutes % 60;
		if (hours === 0) return `${mins}m`;
		return `${hours}h ${mins}m`;
	}

	function getCreators(): string[] {
		const crew = parseCrew(show?.crew);
		return crew
			.filter(c => c.job === 'Creator' || c.job === 'Executive Producer' || c.job === 'Showrunner')
			.map(c => c.name)
			.slice(0, 3);
	}

	function getNextAirDate(): string | null {
		if (!show?.seasons || show.status?.toLowerCase() !== 'returning series') return null;
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		for (const season of show.seasons) {
			if (!season?.episodes) continue;
			for (const ep of season.episodes) {
				if (ep.airDate) {
					const airDate = new Date(ep.airDate);
					if (airDate >= today) {
						return ep.airDate;
					}
				}
			}
		}
		return null;
	}

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString('en-US', {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}

	function getTags(): string[] {
		return parseGenres(show?.genres);
	}
</script>

<svelte:head>
	<title>{show?.title || 'TV Show'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-96">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Loading show...</p>
		</div>
	</div>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-lg">
		{error}
		<button class="ml-2 underline" onclick={() => error = null}>Dismiss</button>
	</div>
{:else if show}
	<div class="space-y-6 -mt-22 -mx-6">
		<!-- ============================================
		     1. HERO SECTION
		     ============================================ -->
		<section class="relative min-h-[500px]">
			<!-- Backdrop -->
			{#if show.backdropPath}
				<img
					src={getImageUrl(show.backdropPath)}
					alt=""
					class="absolute inset-0 w-full h-full object-cover pointer-events-none"
					style="object-position: center 25%;"
					draggable="false"
				/>
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
							{#if show.posterPath}
								<img
									src={getImageUrl(show.posterPath)}
									alt={show.title}
									class="w-full h-full object-cover"
								/>
							{:else}
								<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
									<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
								</div>
							{/if}
							<!-- Status Badge -->
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
							{#if show.rating}
								<a href="https://www.themoviedb.org/tv/{show.tmdbId}" target="_blank" class="flex items-center gap-1.5 hover:opacity-80 transition-opacity" title="TMDB Rating">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6 rounded" />
									<span class="text-base font-bold text-white">{show.rating.toFixed(1)}</span>
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
						{show.title}
						{#if show.year}
							<span class="text-text-secondary font-normal">({show.year})</span>
						{/if}
					</h1>

					<!-- Meta line -->
					<div class="flex items-center gap-2 text-text-secondary mb-4">
						{#if show.contentRating}
							<span class="px-2 py-0.5 border border-white/30 text-xs font-medium">{show.contentRating}</span>
						{/if}
						<span>{show.seasons?.length || 0} Seasons</span>
						<span>•</span>
						<span>{totalEpisodes()} Episodes</span>
						{#if getTags().length > 0}
							<span>•</span>
							<span>{getTags().join(', ')}</span>
						{/if}
					</div>

					<!-- Tags (clickable pills) -->
					{#if getTags().length > 0}
						<div class="flex flex-wrap gap-2 mb-4">
							{#each getTags() as tag}
								<a href="/discover/show?genre={encodeURIComponent(tag)}" class="liquid-tag text-sm">
									{tag}
								</a>
							{/each}
						</div>
					{/if}

					<!-- Tagline -->
					{#if show.tagline}
						<p class="text-text-secondary italic mb-4">"{show.tagline}"</p>
					{/if}

					<!-- Overview -->
					{#if show.overview}
						<p class="text-text-secondary leading-relaxed max-w-2xl mb-5">
							{show.overview}
						</p>
					{/if}

					<!-- Icon bubble controls -->
					<div class="flex items-center gap-3 mb-5">
						<!-- Play button -->
						<button
							onclick={handlePlayNext}
							class="h-11 px-6 rounded-full bg-white text-black font-semibold flex items-center gap-2 hover:bg-white/90 transition-all"
						>
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M8 5v14l11-7z" />
							</svg>
							{#if nextEpisode()}
								Play S{nextEpisode()?.season} E{nextEpisode()?.episode}
							{:else}
								Play S1 E1
							{/if}
						</button>

						<button
							onclick={handleToggleWatchlist}
							disabled={watchlistLoading}
							class="w-11 h-11 rounded-full flex items-center justify-center transition-all border disabled:opacity-50 {inWatchlist ? 'bg-blue-600 border-blue-500 text-white' : 'bg-white/10 border-white/20 text-white hover:bg-white/20'}"
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
								onclick={() => showManageMenu = !showManageMenu}
								class="w-11 h-11 rounded-full bg-white/10 border border-white/20 text-white flex items-center justify-center hover:bg-white/20 transition-all"
								title="Manage"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
								</svg>
							</button>
							{#if showManageMenu}
								<button
									type="button"
									class="fixed inset-0 z-[55] cursor-default"
									onclick={() => showManageMenu = false}
									aria-label="Close menu"
								></button>
								<div class="absolute left-0 mt-2 w-48 py-1 z-[60] bg-[#141416] border border-white/10 rounded-2xl shadow-xl overflow-hidden">
									<button
										onclick={handleRefresh}
										class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors"
									>
										{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
									</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
									<button class="w-full text-left px-4 py-2.5 text-sm text-white/80 hover:bg-white/10 hover:text-white transition-colors" onclick={() => showManageMenu = false}>Edit Images</button>
									<div class="border-t border-white/10 my-1"></div>
									<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={() => showManageMenu = false}>Delete</button>
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- RIGHT: Info Panel Card -->
				<div class="flex-shrink-0 w-72">
					<div class="liquid-card p-4 space-y-3 text-sm">
						<!-- Status -->
						<div class="flex justify-between">
							<span class="text-text-muted">Status</span>
							<span class="{getStatusColor(show.status)} font-medium">{show.status || 'Unknown'}</span>
						</div>

						<!-- Network -->
						{#if show.network}
							<div class="flex justify-between">
								<span class="text-text-muted">Network</span>
								<span>{show.network}</span>
							</div>
						{/if}

						<!-- Created By -->
						{#if getCreators().length > 0}
							<div class="flex justify-between">
								<span class="text-text-muted">Created By</span>
								<span class="text-right max-w-[180px] truncate" title={getCreators().join(', ')}>{getCreators().join(', ')}</span>
							</div>
						{/if}

						<!-- Next Air Date -->
						{#if getNextAirDate()}
							<div class="flex justify-between">
								<span class="text-text-muted">Next Episode</span>
								<span class="text-green-400">{formatDate(getNextAirDate())}</span>
							</div>
						{/if}

						<!-- Year -->
						{#if show.year}
							<div class="flex justify-between">
								<span class="text-text-muted">Year</span>
								<span>{show.year}</span>
							</div>
						{/if}

						<!-- Seasons -->
						<div class="flex justify-between">
							<span class="text-text-muted">Seasons</span>
							<span>{show.seasons?.length || 0}</span>
						</div>

						<!-- Episodes -->
						<div class="flex justify-between">
							<span class="text-text-muted">Episodes</span>
							<span>{totalEpisodes()}</span>
						</div>

						<!-- Total Runtime -->
						<div class="flex justify-between">
							<span class="text-text-muted">Total Runtime</span>
							<span>{getTotalRuntime()}</span>
						</div>

						<!-- Language -->
						{#if show.originalLanguage}
							<div class="flex justify-between">
								<span class="text-text-muted">Language</span>
								<span>{getLanguageName(show.originalLanguage)}</span>
							</div>
						{/if}

						<!-- Country -->
						{#if show.country}
							<div class="flex justify-between">
								<span class="text-text-muted">Country</span>
								<span>{getCountryFlag(show.country)} {getCountryName(show.country)}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- Parental -->
						{#if show.contentRating}
							<div class="flex justify-between items-center">
								<span class="text-text-muted">Parental</span>
								<span class="flex items-center gap-2">
									<span class="px-1.5 py-0.5 bg-white/10 rounded text-xs font-medium">{show.contentRating}</span>
									{#if show.imdbId}
										<a href="https://www.imdb.com/title/{show.imdbId}/parentalguide" target="_blank" class="text-sky-400 hover:text-sky-300 text-xs">
											View ↗
										</a>
									{/if}
								</span>
							</div>
						{/if}

						<!-- Added date -->
						{#if show.addedAt}
							<div class="flex justify-between">
								<span class="text-text-muted">Added</span>
								<span class="text-xs">{new Date(show.addedAt).toLocaleDateString()}</span>
							</div>
						{/if}

						<div class="border-t border-white/10 my-2"></div>

						<!-- External Links -->
						<div class="flex justify-center gap-3">
							{#if show.tmdbId}
								<a href="https://www.themoviedb.org/tv/{show.tmdbId}" target="_blank"
								   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on TMDB">
									<img src="/icons/tmdb.svg" alt="TMDB" class="w-7 h-7" />
								</a>
							{/if}
							{#if show.imdbId}
								<a href="https://www.imdb.com/title/{show.imdbId}" target="_blank"
								   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on IMDb">
									<img src="/icons/imdb.svg" alt="IMDb" class="w-7 h-7" />
								</a>
							{/if}
							<a href="https://trakt.tv/search/tmdb/{show.tmdbId}?id_type=show" target="_blank"
							   class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Trakt">
								<img src="/icons/trakt.svg" alt="Trakt" class="w-7 h-7" />
							</a>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- ============================================
		     2. EPISODES SECTION
		     ============================================ -->
		<section class="px-6">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-text-primary">Episodes</h2>
				{#if show.seasons && show.seasons.length > 0}
					<Dropdown
						options={show.seasons.map((season, i) => ({ value: i, label: `Season ${season.seasonNumber}` }))}
						value={selectedSeasonIndex}
						onchange={(v) => selectedSeasonIndex = v as number}
					/>
				{/if}
			</div>

			{#if selectedSeason?.episodes && selectedSeason.episodes.length > 0}
				<div class="flex gap-4 overflow-x-auto pb-2 scrollbar-thin">
					{#each selectedSeason.episodes as episode}
						<button
							onclick={() => goto(`/watch/episode/${episode.id}`)}
							class="group relative text-left flex-shrink-0 w-64"
						>
							<div class="relative aspect-video bg-bg-card overflow-hidden rounded-xl">
								<!-- Episode still/backdrop -->
								{#if episode.stillPath}
									<img
										src={getImageUrl(episode.stillPath)}
										alt={episode.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else if show.backdropPath}
									<img
										src={getImageUrl(show.backdropPath)}
										alt={episode.title}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105 opacity-50"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
										<svg class="w-10 h-10 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
									</div>
								{/if}

								<!-- Gradient overlay -->
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>

								<!-- Play button on hover -->
								<div class="absolute inset-0 flex items-center justify-center">
									<div class="w-12 h-12 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transform scale-75 group-hover:scale-100 transition-all duration-300 border border-white/30">
										<svg class="w-6 h-6 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
											<path d="M8 5v14l11-7z" />
										</svg>
									</div>
								</div>

								<!-- Episode number badge -->
								<div class="absolute top-1 left-1">
									<div class="px-2 py-0.5 rounded bg-black/70 text-white text-xs font-medium">
										E{episode.episodeNumber}
									</div>
								</div>

								<!-- Watched indicator -->
								{#if watchedEpisodes.has(episode.id)}
									<div class="absolute top-1 right-1">
										<div class="w-5 h-5 rounded-full bg-green-600 flex items-center justify-center">
											<svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
											</svg>
										</div>
									</div>
								{/if}

								<!-- Episode title at bottom -->
								<div class="absolute bottom-0 left-0 right-0 p-3">
									<p class="text-sm font-medium text-white truncate">{episode.title || `Episode ${episode.episodeNumber}`}</p>
									{#if episode.runtime}
										<p class="text-xs text-white/60">{episode.runtime}m</p>
									{/if}
								</div>
							</div>
						</button>
					{/each}
				</div>
			{:else}
				<div class="text-center py-12 text-text-muted">
					<p>No episodes found for this season.</p>
				</div>
			{/if}
		</section>

		<!-- ============================================
		     3. CAST
		     ============================================ -->
		{#if parseCast(show.cast).length > 0}
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
					class="flex gap-5 overflow-x-auto pt-1 pl-1 pb-2 -ml-1 scrollbar-thin"
				>
					{#each parseCast(show.cast) as actor}
						<button
							onclick={() => handlePersonClick(actor)}
							class="flex-shrink-0 w-28 text-center cursor-pointer group"
						>
							<div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10 group-hover:ring-white/30 transition-all">
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
							<p class="mt-2 text-sm font-medium text-text-primary truncate group-hover:text-white transition-colors">{actor.name}</p>
							<p class="text-xs text-text-muted truncate">{actor.character}</p>
						</button>
					{/each}
				</div>
			</section>
		{/if}

		<!-- ============================================
		     4. CREW
		     ============================================ -->
		{#if parseCrew(show.crew).length > 0}
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
					class="flex gap-5 overflow-x-auto pt-1 pl-1 pb-2 -ml-1 scrollbar-thin"
				>
					{#each parseCrew(show.crew) as member}
						<button
							onclick={() => handlePersonClick(member)}
							class="flex-shrink-0 w-28 text-center cursor-pointer group"
						>
							<div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10 group-hover:ring-white/30 transition-all">
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
							<p class="mt-2 text-sm font-medium text-text-primary truncate group-hover:text-white transition-colors">{member.name}</p>
							<p class="text-xs text-text-muted truncate">{member.job}</p>
						</button>
					{/each}
				</div>
			</section>
		{/if}

		<!-- ============================================
		     5. SUGGESTIONS ROW
		     ============================================ -->
		<section class="px-6 pb-8">
			<h2 class="text-lg font-semibold text-text-primary mb-3">More Like This</h2>
			{#if recommendations.length > 0}
				<div class="flex gap-4 overflow-x-auto pb-2 scrollbar-thin">
					{#each recommendations as rec}
						<a href="/discover/show/{rec.id}" class="flex-shrink-0 w-32 group">
							<div class="relative aspect-[2/3] rounded-lg overflow-hidden bg-bg-card">
								{#if rec.poster_path}
									<img
										src={getTmdbImageUrl(rec.poster_path, 'w342')}
										alt={rec.name}
										class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
										<svg class="w-10 h-10 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
										</svg>
									</div>
								{/if}
								<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
							</div>
							<p class="mt-2 text-xs text-text-primary truncate">{rec.name}</p>
							<p class="text-[10px] text-text-muted">{rec.first_air_date?.split('-')[0] || ''}</p>
						</a>
					{/each}
				</div>
			{:else}
				<div class="flex gap-2">
					{#each getTags().slice(0, 3) as genre}
						<a href="/discover/show?genre={encodeURIComponent(genre)}" class="liquid-btn-sm">
							Browse {genre} →
						</a>
					{/each}
				</div>
			{/if}
		</section>
	</div>

	<!-- Trailer Modal -->
	{#if showTrailerModal && getOfficialTrailer()}
		{@const trailer = getOfficialTrailer()}
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<button
				class="absolute inset-0 bg-black/90"
				onclick={() => showTrailerModal = false}
				aria-label="Close"
			></button>
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

	<!-- Person Modal -->
	<PersonModal
		personId={selectedPersonId}
		personName={selectedPersonName}
		onClose={closePersonModal}
	/>
{/if}
