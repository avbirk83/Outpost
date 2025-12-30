<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import {
		getDiscoverShowDetail,
		createRequest,
		getTmdbImageUrl,
		type DiscoverShowDetail
	} from '$lib/api';

	let show: DiscoverShowDetail | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let requesting = $state(false);
	let requested = $state(false);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			show = await getDiscoverShowDetail(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load show details';
		} finally {
			loading = false;
		}
	});

	async function handleRequest() {
		if (!show) return;
		requesting = true;

		try {
			const year = show.firstAirDate ? parseInt(show.firstAirDate.substring(0, 4)) : undefined;
			await createRequest({
				type: 'show',
				tmdbId: show.id,
				title: show.title,
				year,
				overview: show.overview || undefined,
				posterPath: show.posterPath || undefined
			});
			requested = true;
		} catch (e) {
			if (e instanceof Error && e.message === 'Already requested') {
				requested = true;
			} else {
				error = e instanceof Error ? e.message : 'Failed to create request';
			}
		} finally {
			requesting = false;
		}
	}

	function getYear(dateStr: string | undefined): string {
		if (!dateStr) return '';
		return dateStr.substring(0, 4);
	}
</script>

<svelte:head>
	<title>{show?.title || 'TV Show'} - Outpost</title>
</svelte:head>

{#if loading}
	<p class="text-gray-400">Loading show details...</p>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded">
		{error}
	</div>
{:else if show}
	<!-- Backdrop -->
	{#if show.backdropPath}
		<div class="absolute inset-x-0 top-0 h-96 -z-10">
			<img
				src={getTmdbImageUrl(show.backdropPath, 'w1280')}
				alt=""
				class="w-full h-full object-cover" style="object-position: center 25%;"
			/>
			<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent"></div>
		</div>
	{/if}

	<div class="flex flex-col md:flex-row gap-8 pt-8">
		<!-- Poster -->
		<div class="flex-shrink-0">
			<div class="w-64 aspect-[2/3] bg-gray-800 rounded-lg overflow-hidden">
				{#if show.posterPath}
					<img
						src={getTmdbImageUrl(show.posterPath, 'w500')}
						alt={show.title}
						class="w-full h-full object-cover"
					/>
				{:else}
					<div class="w-full h-full flex items-center justify-center text-gray-600">
						<svg class="w-24 h-24" fill="currentColor" viewBox="0 0 24 24">
							<path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
						</svg>
					</div>
				{/if}
			</div>
			<div class="mt-4 space-y-2">
				{#if requested}
					<button
						disabled
						class="liquid-btn w-full !bg-green-500/20 !border-t-green-400/40 text-green-400"
					>
						Requested
					</button>
				{:else}
					<button
						onclick={handleRequest}
						disabled={requesting}
						class="liquid-btn w-full disabled:opacity-50"
					>
						{requesting ? 'Requesting...' : 'Request'}
					</button>
				{/if}
			</div>
		</div>

		<!-- Details -->
		<div class="flex-1 space-y-6">
			<div>
				<h1 class="text-4xl font-bold">{show.title}</h1>
			</div>

			<div class="flex flex-wrap items-center gap-4 text-sm">
				{#if show.rating}
					<div class="flex items-center gap-1">
						<span class="text-yellow-400 text-lg">*</span>
						<span class="text-lg font-semibold">{show.rating.toFixed(1)}</span>
						<span class="text-gray-400">/10</span>
					</div>
				{/if}
				{#if show.firstAirDate}
					<span class="text-gray-300">{getYear(show.firstAirDate)}</span>
				{/if}
				{#if show.seasons}
					<span class="text-gray-300">{show.seasons} season{show.seasons !== 1 ? 's' : ''}</span>
				{/if}
				{#if show.status}
					<span class="liquid-badge-sm">{show.status}</span>
				{/if}
			</div>

			{#if show.genres && show.genres.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each show.genres as genre}
						<span class="liquid-tag">{genre}</span>
					{/each}
				</div>
			{/if}

			{#if show.networks && show.networks.length > 0}
				<div>
					<span class="text-gray-400">Network</span>
					<span class="ml-2">{show.networks.join(', ')}</span>
				</div>
			{/if}

			{#if show.overview}
				<div>
					<h2 class="text-lg font-semibold mb-2">Overview</h2>
					<p class="text-gray-300 leading-relaxed">{show.overview}</p>
				</div>
			{/if}

			{#if show.cast && show.cast.length > 0}
				<div>
					<h2 class="text-lg font-semibold mb-4">Cast</h2>
					<div class="flex gap-4 overflow-x-auto pb-4">
						{#each show.cast as person}
							<div class="flex-shrink-0 w-24 text-center">
								<div class="w-24 h-24 rounded-full bg-gray-700 overflow-hidden mx-auto">
									{#if person.photo}
										<img
											src={getTmdbImageUrl(person.photo, 'w185')}
											alt={person.name}
											class="w-full h-full object-cover"
										/>
									{:else}
										<div class="w-full h-full flex items-center justify-center text-gray-500">
											<svg class="w-12 h-12" fill="currentColor" viewBox="0 0 24 24">
												<path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
											</svg>
										</div>
									{/if}
								</div>
								<p class="mt-2 text-sm font-medium truncate">{person.name}</p>
								<p class="text-xs text-gray-400 truncate">{person.character}</p>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}
