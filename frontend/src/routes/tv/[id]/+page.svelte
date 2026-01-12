<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getShow, refreshShowMetadata, getImageUrl, getTmdbImageUrl,
		getQualityProfiles, getWatchStatus, markAsWatched, markAsUnwatched,
		getShowSuggestions, addToWatchlist, removeFromWatchlist, isInWatchlist,
		deleteEpisode, getShowQuality, setShowQuality, getQualityPresets,
		getSkipSegments, saveSkipSegment, deleteSkipSegment,
		getMissingEpisodes, requestMissingEpisodes, detectShowIntros,
		type ShowDetail, type QualityProfile, type TMDBShowResult, type QualityInfo, type QualityPreset, type SkipSegments,
		type MissingEpisodesResult, type MissingEpisode
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import { toast } from '$lib/stores/toast';
	import { getOfficialTrailer, parseGenres, parseCast, parseCrew } from '$lib/utils';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import MediaDetail from '$lib/components/MediaDetail.svelte';
	import Dropdown from '$lib/components/Dropdown.svelte';
	import AddToCollectionButton from '$lib/components/AddToCollectionButton.svelte';
	import SubtitleSearchModal from '$lib/components/SubtitleSearchModal.svelte';
	import SeasonEpisodeList from '$lib/components/SeasonEpisodeList.svelte';

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
	let qualityInfo: QualityInfo | null = $state(null);
	let monitored = $state(true);
	let monitoringLoading = $state(false);
	let monitoredSeasons = $state<Set<number>>(new Set());
	let togglingSeasonMonitor: number | null = $state(null);
	let qualityPresets: QualityPreset[] = $state([]);
	let selectedPresetId: number | null = $state(null);
	let preferredAudioLang = $state<string>('');
	let preferredSubtitleLang = $state<string>('');

	// Subtitle search
	let subtitleSearchEpisode = $state<{ id: number; title: string; seasonNumber: number; episodeNumber: number } | null>(null);

	// Skip segments
	let skipSegments = $state<SkipSegments>({});
	let introStartInput = $state('');
	let introEndInput = $state('');
	let creditsStartInput = $state('');
	let creditsEndInput = $state('');
	let savingIntro = $state(false);
	let savingCredits = $state(false);
	let deletingIntro = $state(false);
	let deletingCredits = $state(false);

	// Missing episodes
	let missingData = $state<MissingEpisodesResult | null>(null);
	let loadingMissing = $state(false);
	let requestingMissing = $state(false);
	let requestingSeasonMissing = $state<number | null>(null);
	let showMissingSection = $state(false);

	// Intro detection
	let detectingIntros = $state(false);

	// Common language options for audio/subtitle preferences
	const languageOptions = [
		{ value: '', label: 'Default' },
		{ value: 'en', label: 'English' },
		{ value: 'ja', label: 'Japanese' },
		{ value: 'es', label: 'Spanish' },
		{ value: 'fr', label: 'French' },
		{ value: 'de', label: 'German' },
		{ value: 'it', label: 'Italian' },
		{ value: 'pt', label: 'Portuguese' },
		{ value: 'ko', label: 'Korean' },
		{ value: 'zh', label: 'Chinese' },
		{ value: 'ru', label: 'Russian' },
	];

	const subtitleOptions = [
		{ value: '', label: 'Default' },
		{ value: 'off', label: 'Off' },
		{ value: 'en', label: 'English' },
		{ value: 'ja', label: 'Japanese' },
		{ value: 'es', label: 'Spanish' },
		{ value: 'fr', label: 'French' },
		{ value: 'de', label: 'German' },
		{ value: 'it', label: 'Italian' },
		{ value: 'pt', label: 'Portuguese' },
		{ value: 'ko', label: 'Korean' },
		{ value: 'zh', label: 'Chinese' },
		{ value: 'ru', label: 'Russian' },
	];

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
			// Load quality presets
			try {
				const allPresets = await getQualityPresets();
				qualityPresets = allPresets.filter(p => p.enabled && p.mediaType === 'tv');
			} catch { /* Presets are optional */ }

			// Load quality/monitoring info
			try {
				qualityInfo = await getShowQuality(id);
				if (qualityInfo?.override) {
					monitored = qualityInfo.override.monitored;
					selectedPresetId = qualityInfo.override.presetId ?? null;
					preferredAudioLang = qualityInfo.override.preferredAudioLang ?? '';
					preferredSubtitleLang = qualityInfo.override.preferredSubtitleLang ?? '';
					// Parse monitored seasons
					if (qualityInfo.override.monitoredSeasons) {
						try {
							const seasons = JSON.parse(qualityInfo.override.monitoredSeasons);
							monitoredSeasons = new Set(seasons);
						} catch { /* Invalid JSON, treat as all monitored */ }
					}
				}
				// If no monitored seasons set, default to all seasons
				if (monitoredSeasons.size === 0 && show?.seasons) {
					monitoredSeasons = new Set(show.seasons.map(s => s.seasonNumber));
				}
			} catch { /* Quality info is optional */ }

			// Set default preset if none selected
			if (!selectedPresetId && qualityPresets.length > 0) {
				const defaultPreset = qualityPresets.find(p => p.isDefault);
				selectedPresetId = defaultPreset?.id ?? qualityPresets[0].id;
			}

			// Load skip segments
			try {
				skipSegments = await getSkipSegments(id);
				if (skipSegments.intro) {
					introStartInput = formatTimeInput(skipSegments.intro.startTime);
					introEndInput = formatTimeInput(skipSegments.intro.endTime);
				}
				if (skipSegments.credits) {
					creditsStartInput = formatTimeInput(skipSegments.credits.startTime);
					creditsEndInput = formatTimeInput(skipSegments.credits.endTime);
				}
			} catch { /* Skip segments are optional */ }

			// Load suggestions
			if (show) {
				try {
					const suggestResult = await getShowSuggestions(show.id);
					recommendations = suggestResult.results.slice(0, 20);
				} catch { /* Suggestions are optional */ }
			}

			// Load missing episodes (for shows with TMDB ID)
			if (show?.tmdbId) {
				loadMissingEpisodes();
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

	async function handleToggleEpisodeWatched(episodeId: number) {
		togglingEpisode = episodeId;
		// Find the episode runtime
		let runtime = 2400; // default 40 minutes
		if (show?.seasons) {
			for (const season of show.seasons) {
				const ep = season.episodes.find(e => e.id === episodeId);
				if (ep?.runtime) {
					runtime = ep.runtime * 60;
					break;
				}
			}
		}
		try {
			if (watchedEpisodes.has(episodeId)) {
				await markAsUnwatched('episode', episodeId);
				watchedEpisodes.delete(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			} else {
				await markAsWatched('episode', episodeId, runtime);
				watchedEpisodes.add(episodeId);
				watchedEpisodes = new Set(watchedEpisodes);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update watch status';
		} finally {
			togglingEpisode = null;
		}
	}

	let deletingEpisode: number | null = $state(null);

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

	async function handleToggleMonitoring() {
		if (!show) return;
		monitoringLoading = true;
		try {
			const newMonitored = !monitored;
			await setShowQuality(show.id, {
				monitored: newMonitored,
				monitoredSeasons: JSON.stringify(Array.from(monitoredSeasons)),
				presetId: selectedPresetId
			});
			monitored = newMonitored;
			toast.success(newMonitored ? 'Monitoring enabled' : 'Monitoring disabled');
		} catch (e) {
			console.error('Failed to update monitoring:', e);
			toast.error('Failed to update monitoring');
		} finally {
			monitoringLoading = false;
		}
	}

	async function handlePresetChange(presetId: number) {
		if (!show) return;
		selectedPresetId = presetId;
		try {
			await setShowQuality(show.id, {
				monitored,
				monitoredSeasons: JSON.stringify(Array.from(monitoredSeasons)),
				presetId,
				preferredAudioLang,
				preferredSubtitleLang
			});
			toast.success('Quality preset updated');
		} catch (e) {
			console.error('Failed to update quality preset:', e);
			toast.error('Failed to update quality preset');
		}
	}

	async function handleAudioLangChange(lang: string) {
		if (!show) return;
		preferredAudioLang = lang;
		try {
			await setShowQuality(show.id, {
				monitored,
				monitoredSeasons: JSON.stringify(Array.from(monitoredSeasons)),
				presetId: selectedPresetId,
				preferredAudioLang: lang,
				preferredSubtitleLang
			});
			toast.success('Audio preference updated');
		} catch (e) {
			console.error('Failed to update audio preference:', e);
			toast.error('Failed to update audio preference');
		}
	}

	async function handleSubtitleLangChange(lang: string) {
		if (!show) return;
		preferredSubtitleLang = lang;
		try {
			await setShowQuality(show.id, {
				monitored,
				monitoredSeasons: JSON.stringify(Array.from(monitoredSeasons)),
				presetId: selectedPresetId,
				preferredAudioLang,
				preferredSubtitleLang: lang
			});
			toast.success('Subtitle preference updated');
		} catch (e) {
			console.error('Failed to update subtitle preference:', e);
			toast.error('Failed to update subtitle preference');
		}
	}

	function handleEpisodePlay(episodeId: number) {
		goto(`/watch/episode/${episodeId}`);
	}

	function handleEpisodeSubtitleSearch(episode: { id: number; title: string; seasonNumber: number; episodeNumber: number }) {
		subtitleSearchEpisode = episode;
	}

	async function handleEpisodeDelete(episodeId: number) {
		deletingEpisode = episodeId;
		try {
			await deleteEpisode(episodeId);
			// Refresh show data to update the episode list
			if (show) {
				show = await getShow(show.id);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete episode';
		} finally {
			deletingEpisode = null;
		}
	}

	async function handleToggleSeasonMonitoring(seasonNumber: number) {
		if (!show) return;
		togglingSeasonMonitor = seasonNumber;
		try {
			const newMonitoredSeasons = new Set(monitoredSeasons);
			if (newMonitoredSeasons.has(seasonNumber)) {
				newMonitoredSeasons.delete(seasonNumber);
			} else {
				newMonitoredSeasons.add(seasonNumber);
			}
			await setShowQuality(show.id, {
				monitored: monitored,
				monitoredSeasons: JSON.stringify(Array.from(newMonitoredSeasons)),
				presetId: selectedPresetId
			});
			monitoredSeasons = newMonitoredSeasons;
		} catch (e) {
			console.error('Failed to update season monitoring:', e);
			toast.error('Failed to update season monitoring');
		} finally {
			togglingSeasonMonitor = null;
		}
	}

	const selectedSeason = $derived(show?.seasons?.[selectedSeasonIndex]);

	// Transform seasons for SeasonEpisodeList component
	const librarySeasons = $derived(
		show?.seasons?.map(s => ({
			seasonNumber: s.seasonNumber,
			episodes: s.episodes.map(ep => ({
				id: ep.id,
				episodeNumber: ep.episodeNumber,
				title: ep.title,
				overview: ep.overview,
				airDate: ep.airDate,
				runtime: ep.runtime,
				stillPath: ep.stillPath
			}))
		})) || []
	);

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

	// Skip segment helper functions
	function formatTimeInput(seconds: number): string {
		const mins = Math.floor(seconds / 60);
		const secs = Math.floor(seconds % 60);
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function parseTimeInput(input: string): number | null {
		const trimmed = input.trim();
		if (!trimmed) return null;

		// Try MM:SS format
		const colonMatch = trimmed.match(/^(\d+):(\d{1,2})$/);
		if (colonMatch) {
			const mins = parseInt(colonMatch[1]);
			const secs = parseInt(colonMatch[2]);
			if (secs < 60) {
				return mins * 60 + secs;
			}
		}

		// Try plain seconds
		const num = parseFloat(trimmed);
		if (!isNaN(num) && num >= 0) {
			return num;
		}

		return null;
	}

	async function handleSaveIntro() {
		if (!show) return;
		const start = parseTimeInput(introStartInput);
		const end = parseTimeInput(introEndInput);

		if (start === null || end === null) {
			toast.error('Invalid time format. Use MM:SS or seconds.');
			return;
		}
		if (start >= end) {
			toast.error('Start time must be before end time.');
			return;
		}

		savingIntro = true;
		try {
			await saveSkipSegment(show.id, 'intro', start, end);
			skipSegments = { ...skipSegments, intro: { startTime: start, endTime: end } };
			toast.success('Intro skip saved');
		} catch (e) {
			toast.error('Failed to save intro skip');
		} finally {
			savingIntro = false;
		}
	}

	async function handleDeleteIntro() {
		if (!show) return;
		deletingIntro = true;
		try {
			await deleteSkipSegment(show.id, 'intro');
			skipSegments = { ...skipSegments, intro: undefined };
			introStartInput = '';
			introEndInput = '';
			toast.success('Intro skip removed');
		} catch (e) {
			toast.error('Failed to delete intro skip');
		} finally {
			deletingIntro = false;
		}
	}

	async function handleSaveCredits() {
		if (!show) return;
		const start = parseTimeInput(creditsStartInput);
		const end = parseTimeInput(creditsEndInput);

		if (start === null || end === null) {
			toast.error('Invalid time format. Use MM:SS or seconds.');
			return;
		}
		if (start >= end) {
			toast.error('Start time must be before end time.');
			return;
		}

		savingCredits = true;
		try {
			await saveSkipSegment(show.id, 'credits', start, end);
			skipSegments = { ...skipSegments, credits: { startTime: start, endTime: end } };
			toast.success('Credits skip saved');
		} catch (e) {
			toast.error('Failed to save credits skip');
		} finally {
			savingCredits = false;
		}
	}

	async function handleDeleteCredits() {
		if (!show) return;
		deletingCredits = true;
		try {
			await deleteSkipSegment(show.id, 'credits');
			skipSegments = { ...skipSegments, credits: undefined };
			creditsStartInput = '';
			creditsEndInput = '';
			toast.success('Credits skip removed');
		} catch (e) {
			toast.error('Failed to delete credits skip');
		} finally {
			deletingCredits = false;
		}
	}

	async function loadMissingEpisodes() {
		if (!show?.tmdbId) return;
		loadingMissing = true;
		try {
			missingData = await getMissingEpisodes(show.id);
			showMissingSection = missingData.missing.length > 0;
		} catch (e) {
			console.error('Failed to load missing episodes:', e);
		} finally {
			loadingMissing = false;
		}
	}

	async function handleRequestAllMissing() {
		if (!show) return;
		requestingMissing = true;
		try {
			const result = await requestMissingEpisodes(show.id);
			toast.success(`Added ${result.addedCount} episodes to wanted list`);
			// Refresh missing data
			await loadMissingEpisodes();
		} catch (e) {
			toast.error('Failed to request missing episodes');
		} finally {
			requestingMissing = false;
		}
	}

	async function handleRequestSeasonMissing(seasonNumber: number) {
		if (!show) return;
		requestingSeasonMissing = seasonNumber;
		try {
			const result = await requestMissingEpisodes(show.id, seasonNumber);
			toast.success(`Added ${result.addedCount} episodes to wanted list`);
			// Refresh missing data
			await loadMissingEpisodes();
		} catch (e) {
			toast.error('Failed to request missing episodes');
		} finally {
			requestingSeasonMissing = null;
		}
	}

	async function handleDetectIntros() {
		if (!show) return;
		detectingIntros = true;
		try {
			const result = await detectShowIntros(show.id);
			if (result.success) {
				const completed = result.results.filter(r => r.status === 'completed').length;
				const failed = result.results.filter(r => r.status === 'failed').length;
				if (failed > 0) {
					toast.warning();
				} else {
					toast.success();
				}
			}
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Failed to detect intros');
		} finally {
			detectingIntros = false;
		}
	}

	// Get tags from genres
	const tags = $derived(parseGenres(show?.genres));

	// Transform cast/crew for MediaDetail
	const castList = $derived(parseCast(show?.cast).map(c => ({
		id: c.id,
		name: c.name,
		character: c.character,
		profile_path: c.profile_path
	})));

	const crewList = $derived(parseCrew(show?.crew).map(c => ({
		id: c.id,
		name: c.name,
		job: c.job,
		profile_path: c.profile_path
	})));

	// Transform recommendations for MediaDetail
	const recsList = $derived(recommendations.map(r => ({
		id: r.id,
		title: r.name,
		poster_path: r.poster_path,
		release_date: r.first_air_date,
		vote_average: r.vote_average
	})));
</script>

<svelte:head>
	<title>{show?.title || 'TV Show'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-96">
		<div class="flex items-center gap-3">
			<div class="spinner-lg text-cream"></div>
			<p class="text-text-secondary">Loading show...</p>
		</div>
	</div>
{:else if error}
	<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-lg">
		{error}
		<button class="ml-2 underline" onclick={() => error = null}>Dismiss</button>
	</div>
{:else if show}
	<MediaDetail
		title={show.title}
		year={show.year}
		overview={show.overview}
		tagline={show.tagline}
		posterPath={show.posterPath}
		backdropPath={show.backdropPath}
		genres={tags}
		tmdbId={show.tmdbId}
		imdbId={show.imdbId}
		mediaType="tv"
		source="library"
		seasons={show.seasons?.length || 0}
		episodes={totalEpisodes()}
		networks={show.network ? [show.network] : []}
		status={show.status}
		contentRating={show.contentRating}
		originalLanguage={show.originalLanguage}
		country={show.country}
		rating={show.rating}
		cast={castList}
		crew={crewList}
		recommendations={recsList}
		addedAt={show.addedAt}
		useLocalImages={true}
		posterClickable={true}
		onPosterClick={handlePlayNext}
		trailersJson={show.trailers}
	>
		{#snippet actionButtons()}
			<!-- Play (primary action) -->
			<IconButton
				onclick={handlePlayNext}
				variant="yellow"
				compact
				title="Play {nextEpisode() ? `S${nextEpisode()?.season} E${nextEpisode()?.episode}` : 'S1 E1'}"
			>
				<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M8 5v14l11-7z" />
				</svg>
			</IconButton>

			<!-- Watchlist -->
			<IconButton
				onclick={handleToggleWatchlist}
				disabled={watchlistLoading}
				active={inWatchlist}
				compact
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
			</IconButton>

			{#if getOfficialTrailer(show?.trailers)}
				<IconButton
					onclick={() => showTrailerModal = true}
					compact
					title="Watch Trailer"
				>
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
						<path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
					</svg>
				</IconButton>
			{/if}

			<!-- More menu -->
			<div class="relative">
				<IconButton
					onclick={(e: MouseEvent) => { e.stopPropagation(); showManageMenu = !showManageMenu; }}
					compact
					title="More options"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
					</svg>
				</IconButton>
				{#if showManageMenu}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<div
						class="fixed inset-0 z-[55]"
						onclick={() => showManageMenu = false}
					></div>
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<div
						class="absolute left-1/2 -translate-x-1/2 mt-2 w-48 py-1 z-[60] bg-bg-dropdown border border-white/10 rounded-2xl shadow-xl overflow-hidden"
						onclick={(e: MouseEvent) => e.stopPropagation()}
					>
						{#if show?.tmdbId}
							<AddToCollectionButton
								tmdbId={show.tmdbId}
								mediaType="show"
								title={show.title}
								year={show.year}
								posterPath={show.posterPath}
								asMenuItem
								onSelect={() => showManageMenu = false}
							/>
						{/if}
						<button
							onclick={() => { handleRefresh(); showManageMenu = false; }}
							class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors"
						>
							{refreshing ? 'Refreshing...' : 'Refresh Metadata'}
						</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors" onclick={() => showManageMenu = false}>Edit Metadata</button>
						<button class="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-white/10 hover:text-text-primary transition-colors" onclick={() => showManageMenu = false}>Fix Match</button>
						<div class="border-t border-border-subtle my-1"></div>
						<button class="w-full text-left px-4 py-2.5 text-sm text-red-400 hover:bg-white/10 hover:text-red-300 transition-colors" onclick={() => showManageMenu = false}>Delete</button>
					</div>
				{/if}
			</div>
		{/snippet}

		{#snippet extraInfoRows()}
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
					<span class="text-green-400">{formatDate(getNextAirDate() || '')}</span>
				</div>
			{/if}

			<!-- Total Runtime -->
			<div class="flex justify-between">
				<span class="text-text-muted">Total Runtime</span>
				<span>{getTotalRuntime()}</span>
			</div>

			<!-- Added date -->
			{#if show.addedAt}
				<div class="flex justify-between">
					<span class="text-text-muted">Added</span>
					<span class="text-xs">{new Date(show.addedAt).toLocaleDateString()}</span>
				</div>
			{/if}
		{/snippet}

		{#snippet centerExtra()}
			<div class="flex items-center gap-2">
				<div class="w-[110px]">
					<Dropdown
						icon="audio"
						options={languageOptions}
						value={preferredAudioLang}
						onchange={(v) => handleAudioLangChange(v as string)}
						placeholder="Audio"
					/>
				</div>
				<div class="w-[110px]">
					<Dropdown
						icon="subtitles"
						options={subtitleOptions}
						value={preferredSubtitleLang}
						onchange={(v) => handleSubtitleLangChange(v as string)}
						placeholder="Subtitles"
					/>
				</div>
				{#if qualityPresets.length > 0}
					<div class="w-[140px]">
						<Dropdown
							icon="quality"
							options={qualityPresets.map(p => ({ value: p.id, label: p.name + (p.isDefault ? ' ★' : '') }))}
							value={selectedPresetId ?? 0}
							onchange={(v) => handlePresetChange(v as number)}
						/>
					</div>
				{/if}
				{#if qualityInfo?.status && !qualityInfo.status.targetMet}
					<a
						href="/upgrades"
						class="ml-2 px-2 py-1 rounded-lg text-xs font-medium bg-orange-500/20 text-orange-400 hover:bg-orange-500/30 transition-colors flex items-center gap-1.5"
						title="Quality is below cutoff - click to view upgrades"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
						</svg>
						Upgrade Available
					</a>
				{/if}
			</div>
		{/snippet}

		{#snippet extraSections()}
			<!-- Episodes Section -->
			<section class="px-[60px] py-6">
				<h2 class="text-xl font-semibold text-text-primary mb-4">Episodes</h2>
				<SeasonEpisodeList
					tmdbId={show.tmdbId}
					librarySeasons={librarySeasons}
					showBackdrop={show.backdropPath}
					isLibrary={true}
					watchedEpisodes={watchedEpisodes}
					onPlay={handleEpisodePlay}
					onToggleWatched={handleToggleEpisodeWatched}
					onSubtitleSearch={handleEpisodeSubtitleSearch}
					onDelete={handleEpisodeDelete}
					isAdmin={user?.role === 'admin'}
					togglingEpisodeId={togglingEpisode}
					monitoredSeasons={monitoredSeasons}
					onToggleSeasonMonitor={handleToggleSeasonMonitoring}
					togglingSeasonMonitor={togglingSeasonMonitor}
				/>
			</section>

			<!-- Missing Episodes Section -->
			{#if showMissingSection && missingData && missingData.missing.length > 0}
				<section class="px-[60px] mt-8">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-3">
							<h2 class="text-lg font-semibold text-text-primary">Missing Episodes</h2>
							<span class="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/20 text-amber-400">
								{missingData.missing.length} missing
							</span>
						</div>
						{#if user?.role === 'admin'}
							<button
								onclick={handleRequestAllMissing}
								disabled={requestingMissing}
								class="px-4 py-2 bg-cream text-black text-sm font-medium rounded-lg hover:bg-cream/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
							>
								{#if requestingMissing}
									<div class="spinner-sm"></div>
									Requesting...
								{:else}
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
									</svg>
									Request All Missing
								{/if}
							</button>
						{/if}
					</div>

					<!-- Season summary cards -->
					{#if missingData.missingBySeason.length > 0}
						<div class="flex gap-3 mb-4 overflow-x-auto pb-2 scrollbar-thin">
							{#each missingData.missingBySeason.filter(s => s.missingCount > 0) as seasonSummary}
								<div class="flex-shrink-0 bg-glass border border-border-subtle rounded-xl p-3 min-w-[160px]">
									<div class="flex items-center justify-between gap-3">
										<div>
											<p class="text-xs text-text-muted">Season {seasonSummary.seasonNumber}</p>
											<p class="text-sm text-text-primary font-medium">
												{seasonSummary.missingCount} / {seasonSummary.totalEpisodes} missing
											</p>
										</div>
										{#if user?.role === 'admin'}
											<button
												onclick={() => handleRequestSeasonMissing(seasonSummary.seasonNumber)}
												disabled={requestingSeasonMissing === seasonSummary.seasonNumber}
												class="p-2 rounded-lg bg-cream/10 text-cream hover:bg-cream/20 disabled:opacity-50 transition-colors"
												title="Request season {seasonSummary.seasonNumber}"
											>
												{#if requestingSeasonMissing === seasonSummary.seasonNumber}
													<div class="spinner-sm"></div>
												{:else}
													<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
													</svg>
												{/if}
											</button>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}

					<!-- Missing episodes list -->
					<div class="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
						{#each missingData.missing.slice(0, 20) as episode}
							<div class="flex-shrink-0 w-64 rounded-xl overflow-hidden bg-bg-elevated border border-border-subtle">
								<!-- Image container -->
								<div class="relative aspect-video bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44]">
									{#if episode.stillPath}
										<img
											src={getTmdbImageUrl(episode.stillPath, 'w300')}
											alt={episode.title}
											class="w-full h-full object-cover opacity-60"
										/>
									{:else if show?.backdropPath}
										<img
											src={getImageUrl(show.backdropPath)}
											alt={episode.title}
											class="w-full h-full object-cover opacity-30"
										/>
									{/if}
									<!-- Missing badge -->
									<div class="absolute top-2 right-2 px-2 py-1 rounded text-[10px] font-bold uppercase bg-amber-500 text-black">
										Missing
									</div>
								</div>

								<!-- Info section -->
								<div class="p-3">
									<div class="flex items-center gap-2 mb-1">
										<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
											S{episode.seasonNumber} E{episode.episodeNumber}
										</span>
										{#if episode.airDate}
											<span class="text-[10px] text-text-muted">
												{new Date(episode.airDate).toLocaleDateString()}
											</span>
										{/if}
									</div>
									<h3 class="text-sm font-semibold text-text-primary truncate" title={episode.title}>
										{episode.title || `Episode ${episode.episodeNumber}`}
									</h3>
									{#if episode.overview}
										<p class="text-xs text-text-muted mt-1 line-clamp-2">{episode.overview}</p>
									{/if}
								</div>
							</div>
						{/each}
						{#if missingData.missing.length > 20}
							<div class="flex-shrink-0 w-64 rounded-xl bg-bg-elevated border border-border-subtle flex items-center justify-center">
								<p class="text-sm text-text-muted">+{missingData.missing.length - 20} more</p>
							</div>
						{/if}
					</div>
				</section>
			{/if}

			<!-- Skip Segments Section -->
			{#if user?.role === 'admin'}
				<section class="px-[60px] mt-8">
					<div class="flex items-center justify-between mb-4">
						<h2 class="text-lg font-semibold text-text-primary">Playback Settings</h2>
						<button
							onclick={handleDetectIntros}
							disabled={detectingIntros}
							class="px-4 py-2 bg-purple-600 text-white text-sm font-medium rounded-lg hover:bg-purple-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
						>
							{#if detectingIntros}
								<div class="spinner-sm"></div>
								Detecting...
							{:else}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
								</svg>
								Auto-Detect Intros
							{/if}
						</button>
					</div>
					<p class="text-sm text-text-muted mb-4">These skip times apply to all episodes of this show.</p>

					<div class="space-y-4 max-w-xl">
						<!-- Intro Skip -->
						<div class="bg-glass border border-border-subtle rounded-xl p-4">
							<h3 class="text-sm font-semibold text-text-primary mb-3">Skip Intro</h3>
							<div class="flex items-center gap-3">
								<div class="flex-1">
									<label class="text-xs text-text-muted block mb-1">Start (MM:SS)</label>
									<input
										type="text"
										bind:value={introStartInput}
										placeholder="0:30"
										class="w-full px-3 py-2 bg-bg-input border border-border-subtle rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:border-cream focus:outline-none"
									/>
								</div>
								<div class="flex-1">
									<label class="text-xs text-text-muted block mb-1">End (MM:SS)</label>
									<input
										type="text"
										bind:value={introEndInput}
										placeholder="1:30"
										class="w-full px-3 py-2 bg-bg-input border border-border-subtle rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:border-cream focus:outline-none"
									/>
								</div>
								<div class="flex items-end gap-2 pb-0.5">
									<button
										onclick={handleSaveIntro}
										disabled={savingIntro || !introStartInput || !introEndInput}
										class="px-4 py-2 bg-cream text-black text-sm font-medium rounded-lg hover:bg-cream/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
									>
										{savingIntro ? 'Saving...' : 'Save'}
									</button>
									{#if skipSegments.intro}
										<button
											onclick={handleDeleteIntro}
											disabled={deletingIntro}
											class="px-3 py-2 text-red-400 hover:bg-red-500/20 rounded-lg transition-colors"
											title="Delete intro skip"
										>
											{#if deletingIntro}
												<div class="spinner-sm"></div>
											{:else}
												<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
												</svg>
											{/if}
										</button>
									{/if}
								</div>
							</div>
						</div>

						<!-- Credits Skip -->
						<div class="bg-glass border border-border-subtle rounded-xl p-4">
							<h3 class="text-sm font-semibold text-text-primary mb-3">Skip Credits</h3>
							<div class="flex items-center gap-3">
								<div class="flex-1">
									<label class="text-xs text-text-muted block mb-1">Start (MM:SS)</label>
									<input
										type="text"
										bind:value={creditsStartInput}
										placeholder="21:00"
										class="w-full px-3 py-2 bg-bg-input border border-border-subtle rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:border-cream focus:outline-none"
									/>
								</div>
								<div class="flex-1">
									<label class="text-xs text-text-muted block mb-1">End (MM:SS)</label>
									<input
										type="text"
										bind:value={creditsEndInput}
										placeholder="22:00"
										class="w-full px-3 py-2 bg-bg-input border border-border-subtle rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:border-cream focus:outline-none"
									/>
								</div>
								<div class="flex items-end gap-2 pb-0.5">
									<button
										onclick={handleSaveCredits}
										disabled={savingCredits || !creditsStartInput || !creditsEndInput}
										class="px-4 py-2 bg-cream text-black text-sm font-medium rounded-lg hover:bg-cream/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
									>
										{savingCredits ? 'Saving...' : 'Save'}
									</button>
									{#if skipSegments.credits}
										<button
											onclick={handleDeleteCredits}
											disabled={deletingCredits}
											class="px-3 py-2 text-red-400 hover:bg-red-500/20 rounded-lg transition-colors"
											title="Delete credits skip"
										>
											{#if deletingCredits}
												<div class="spinner-sm"></div>
											{:else}
												<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
												</svg>
											{/if}
										</button>
									{/if}
								</div>
							</div>
						</div>
					</div>
				</section>
			{/if}
		{/snippet}
	</MediaDetail>

	<!-- Trailer Modal -->
	<TrailerModal
		bind:open={showTrailerModal}
		trailersJson={show?.trailers}
		title={show?.title}
	/>

	<!-- Subtitle Search Modal -->
	{#if subtitleSearchEpisode && show}
		<SubtitleSearchModal
			media={{
				type: 'episode',
				mediaId: show.id,
				title: subtitleSearchEpisode.title,
				tmdbId: show.tmdbId ?? undefined,
				episodeId: subtitleSearchEpisode.id,
				seasonNumber: subtitleSearchEpisode.seasonNumber,
				episodeNumber: subtitleSearchEpisode.episodeNumber,
				showTitle: show.title
			}}
			onClose={() => subtitleSearchEpisode = null}
			onDownloaded={() => {
				toast.success('Subtitle downloaded successfully');
			}}
		/>
	{/if}
{/if}
