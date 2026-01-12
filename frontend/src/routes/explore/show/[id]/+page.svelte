<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import {
		getDiscoverShowDetail,
		createRequest,
		addToWatchlist,
		removeFromWatchlist,
		isInWatchlist,
		getTmdbImageUrl,
		type DiscoverShowDetailWithStatus,
		type SeasonSummary
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import MediaDetail from '$lib/components/MediaDetail.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import RequestModal from '$lib/components/RequestModal.svelte';
	import ScrollableRow from '$lib/components/ScrollableRow.svelte';

	// Season expansion state
	let expandedSeason = $state<number | null>(null);

	function toggleSeason(seasonNumber: number) {
		if (expandedSeason === seasonNumber) {
			expandedSeason = null;
		} else {
			expandedSeason = seasonNumber;
		}
	}

	function formatAirDate(date: string | undefined): string {
		if (!date) return '';
		try {
			return new Date(date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
		} catch {
			return date;
		}
	}

	let show: DiscoverShowDetailWithStatus | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let requesting = $state(false);
	let requested = $state(false);
	let inWatchlist = $state(false);
	let watchlistLoading = $state(false);
	let showTrailer = $state(false);
	let showRequestModal = $state(false);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			const [showData, watchlistStatus] = await Promise.all([
				getDiscoverShowDetail(id),
				isInWatchlist(id, 'tv').catch(() => false)
			]);
			show = showData as DiscoverShowDetailWithStatus;
			inWatchlist = watchlistStatus;
			if (show.requested) {
				requested = true;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load show details';
		} finally {
			loading = false;
		}
	});

	function openRequestModal() {
		showRequestModal = true;
	}

	async function handleRequest(qualityPresetId: number) {
		if (!show) return;
		showRequestModal = false;
		requesting = true;

		try {
			const year = show.firstAirDate ? parseInt(show.firstAirDate.substring(0, 4)) : undefined;
			await createRequest({
				type: 'show',
				tmdbId: show.id,
				title: show.title,
				year,
				overview: show.overview || undefined,
				posterPath: show.posterPath || undefined,
				backdropPath: show.backdropPath || undefined,
				qualityPresetId
			});
			requested = true;
			toast.success('Request submitted! It will be searched once approved.');
		} catch (e) {
			if (e instanceof Error && e.message === 'Already requested') {
				requested = true;
				toast.info('Already requested');
			} else {
				toast.error('Failed to create request');
			}
		} finally {
			requesting = false;
		}
	}

	async function toggleWatchlist() {
		if (!show) return;
		watchlistLoading = true;
		try {
			if (inWatchlist) {
				await removeFromWatchlist(show.id, 'tv');
				inWatchlist = false;
				toast.success('Removed from watchlist');
			} else {
				await addToWatchlist(show.id, 'tv');
				inWatchlist = true;
				toast.success('Added to watchlist');
			}
		} catch (e) {
			console.error('Failed to update watchlist:', e);
			toast.error('Failed to update watchlist');
		} finally {
			watchlistLoading = false;
		}
	}
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
	</div>
{:else if show}
	<MediaDetail
		title={show.title}
		overview={show.overview}
		posterPath={show.posterPath}
		backdropPath={show.backdropPath}
		genres={show.genres}
		tmdbId={show.id}
		imdbId={show.imdbId}
		mediaType="tv"
		source="discover"
		contentRating={show.contentRating}
		firstAirDate={show.firstAirDate}
		status={show.status}
		seasons={show.seasons}
		episodes={show.episodes}
		networks={show.networks}
		originalLanguage={show.originalLanguage}
		productionCountries={show.productionCountries}
		rating={show.rating}
		cast={show.cast}
		crew={show.crew}
		recommendations={show.recommendations}
		trailerKey={show.trailerKey}
		useLocalImages={false}
	>
		{#snippet extraSections()}
			{#if show.seasonDetails && show.seasonDetails.length > 0}
				<section class="px-[60px] py-6">
					<h2 class="text-xl font-semibold text-text-primary mb-4">Seasons</h2>
					<div class="space-y-3">
						{#each show.seasonDetails.filter(s => s.season_number > 0) as season}
							<div class="liquid-card overflow-hidden">
								<button
									onclick={() => toggleSeason(season.season_number)}
									class="w-full p-4 flex items-center gap-4 hover:bg-white/5 transition-colors text-left"
								>
									<!-- Season Poster -->
									{#if season.poster_path}
										<img
											src={getTmdbImageUrl(season.poster_path, 'w92')}
											alt={season.name}
											class="w-12 h-18 object-cover rounded-lg flex-shrink-0"
										/>
									{:else}
										<div class="w-12 h-18 bg-bg-elevated rounded-lg flex items-center justify-center flex-shrink-0">
											<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
											</svg>
										</div>
									{/if}

									<!-- Season Info -->
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<h3 class="font-medium text-text-primary">{season.name}</h3>
											<span class="text-sm text-text-muted">{season.episode_count} episodes</span>
										</div>
										{#if season.air_date}
											<p class="text-sm text-text-secondary mt-0.5">{formatAirDate(season.air_date)}</p>
										{/if}
									</div>

									<!-- Expand Arrow -->
									<svg
										class="w-5 h-5 text-text-muted transition-transform {expandedSeason === season.season_number ? 'rotate-180' : ''}"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
									</svg>
								</button>

								<!-- Expanded Overview -->
								{#if expandedSeason === season.season_number && season.overview}
									<div class="px-4 pb-4 pt-0">
										<div class="pl-16">
											<p class="text-sm text-text-secondary leading-relaxed">{season.overview}</p>
										</div>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				</section>
			{/if}
		{/snippet}

		{#snippet actionButtons()}
			<!-- Watchlist -->
			<IconButton
				onclick={toggleWatchlist}
				disabled={watchlistLoading}
				active={inWatchlist}
				compact
				title={inWatchlist ? 'In Watchlist' : 'Add to Watchlist'}
			>
				{#if watchlistLoading}
					<div class="spinner-sm text-cream"></div>
				{:else if inWatchlist}
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{:else}
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
				{/if}
			</IconButton>

			<!-- Request / In Library -->
			{#if show.inLibrary && show.libraryId}
				<IconButton href="/tv/{show.libraryId}" variant="green" compact title="View in Library">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</IconButton>
			{:else if show.requestStatus === 'approved'}
				<IconButton variant="green" compact disabled title="Available in Library">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</IconButton>
			{:else if requested || show.requested}
				<IconButton variant="yellow" compact disabled title="Request Pending">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</IconButton>
			{:else}
				<IconButton onclick={openRequestModal} compact disabled={requesting} title="Request">
					{#if requesting}
						<div class="spinner-sm text-cream"></div>
					{:else}
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
					{/if}
				</IconButton>
			{/if}

			<!-- Trailer button -->
			{#if show.trailerKey}
				<IconButton onclick={() => showTrailer = true} compact title="Watch Trailer">
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
						<path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
					</svg>
				</IconButton>
			{/if}
		{/snippet}
	</MediaDetail>

	<!-- Trailer Modal -->
	<TrailerModal
		bind:open={showTrailer}
		tmdbId={show.id}
		mediaType="tv"
		title={show.title}
	/>

	<!-- Request Modal -->
	{#if showRequestModal}
		<RequestModal
			item={{
				title: show.title,
				year: show.firstAirDate ? parseInt(show.firstAirDate.substring(0, 4)) : undefined,
				type: 'show',
				posterPath: show.posterPath,
				backdropPath: show.backdropPath,
				overview: show.overview,
				tmdbId: show.id
			}}
			mode="request"
			onConfirm={handleRequest}
			onCancel={() => showRequestModal = false}
		/>
	{/if}
{/if}
