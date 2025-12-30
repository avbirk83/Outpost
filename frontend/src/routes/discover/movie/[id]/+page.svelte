<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import {
		getDiscoverMovieDetail,
		createRequest,
		getTmdbImageUrl,
		type DiscoverMovieDetail
	} from '$lib/api';

	let movie: DiscoverMovieDetail | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let requesting = $state(false);
	let requested = $state(false);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			movie = await getDiscoverMovieDetail(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load movie details';
		} finally {
			loading = false;
		}
	});

	async function handleRequest() {
		if (!movie) return;
		requesting = true;

		try {
			const year = movie.releaseDate ? parseInt(movie.releaseDate.substring(0, 4)) : undefined;
			await createRequest({
				type: 'movie',
				tmdbId: movie.id,
				title: movie.title,
				year,
				overview: movie.overview || undefined,
				posterPath: movie.posterPath || undefined
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

	function formatRuntime(minutes: number): string {
		if (!minutes) return '';
		const h = Math.floor(minutes / 60);
		const m = minutes % 60;
		return h > 0 ? `${h}h ${m}m` : `${m}m`;
	}

	function getYear(dateStr: string | undefined): string {
		if (!dateStr) return '';
		return dateStr.substring(0, 4);
	}
</script>

<svelte:head>
	<title>{movie?.title || 'Movie'} - Outpost</title>
</svelte:head>

{#if loading}
	<p class="text-gray-400">Loading movie details...</p>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded">
		{error}
	</div>
{:else if movie}
	<!-- Backdrop -->
	{#if movie.backdropPath}
		<div class="absolute inset-x-0 top-0 h-96 -z-10">
			<img
				src={getTmdbImageUrl(movie.backdropPath, 'w1280')}
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
				{#if movie.posterPath}
					<img
						src={getTmdbImageUrl(movie.posterPath, 'w500')}
						alt={movie.title}
						class="w-full h-full object-cover"
					/>
				{:else}
					<div class="w-full h-full flex items-center justify-center text-gray-600">
						<svg class="w-24 h-24" fill="currentColor" viewBox="0 0 24 24">
							<path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
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
				<h1 class="text-4xl font-bold">{movie.title}</h1>
				{#if movie.tagline}
					<p class="text-gray-400 text-lg mt-1 italic">"{movie.tagline}"</p>
				{/if}
			</div>

			<div class="flex flex-wrap items-center gap-4 text-sm">
				{#if movie.rating}
					<div class="flex items-center gap-1">
						<span class="text-yellow-400 text-lg">*</span>
						<span class="text-lg font-semibold">{movie.rating.toFixed(1)}</span>
						<span class="text-gray-400">/10</span>
					</div>
				{/if}
				{#if movie.releaseDate}
					<span class="text-gray-300">{getYear(movie.releaseDate)}</span>
				{/if}
				{#if movie.runtime}
					<span class="text-gray-300">{formatRuntime(movie.runtime)}</span>
				{/if}
			</div>

			{#if movie.genres && movie.genres.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each movie.genres as genre}
						<span class="liquid-tag">{genre}</span>
					{/each}
				</div>
			{/if}

			{#if movie.director}
				<div>
					<span class="text-gray-400">Directed by</span>
					<span class="ml-2">{movie.director}</span>
				</div>
			{/if}

			{#if movie.overview}
				<div>
					<h2 class="text-lg font-semibold mb-2">Overview</h2>
					<p class="text-gray-300 leading-relaxed">{movie.overview}</p>
				</div>
			{/if}

			{#if movie.cast && movie.cast.length > 0}
				<div>
					<h2 class="text-lg font-semibold mb-4">Cast</h2>
					<div class="flex gap-4 overflow-x-auto pb-4">
						{#each movie.cast as person}
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
