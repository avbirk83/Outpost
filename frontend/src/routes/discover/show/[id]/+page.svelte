<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import {
		getDiscoverShowDetail,
		createRequest,
		addToWatchlist,
		removeFromWatchlist,
		isInWatchlist,
		type DiscoverShowDetailWithStatus
	} from '$lib/api';
	import { toast } from '$lib/stores/toast';
	import MediaDetail from '$lib/components/MediaDetail.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import TrailerModal from '$lib/components/TrailerModal.svelte';
	import RequestModal from '$lib/components/RequestModal.svelte';

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
		{#snippet actionButtons()}
			<!-- Watchlist -->
			<IconButton
				onclick={toggleWatchlist}
				disabled={watchlistLoading}
				active={inWatchlist}
				title={inWatchlist ? 'In Watchlist' : 'Add to Watchlist'}
			>
				{#if watchlistLoading}
					<div class="spinner-md text-cream"></div>
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
				<IconButton href="/tv/{show.libraryId}" variant="green" title="View in Library">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</IconButton>
			{:else if show.requestStatus === 'approved'}
				<IconButton variant="green" disabled title="Available in Library">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</IconButton>
			{:else if requested || show.requested}
				<IconButton variant="yellow" disabled title="Request Pending">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</IconButton>
			{:else}
				<IconButton onclick={openRequestModal} disabled={requesting} title="Request">
					{#if requesting}
						<div class="spinner-md text-cream"></div>
					{:else}
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
					{/if}
				</IconButton>
			{/if}

			<!-- Trailer button -->
			{#if show.trailerKey}
				<IconButton onclick={() => showTrailer = true} title="Watch Trailer">
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
				overview: show.overview
			}}
			mode="request"
			onConfirm={handleRequest}
			onCancel={() => showRequestModal = false}
		/>
	{/if}
{/if}
