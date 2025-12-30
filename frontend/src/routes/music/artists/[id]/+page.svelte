<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getArtist, type ArtistDetail } from '$lib/api';

	let artist: ArtistDetail | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			artist = await getArtist(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load artist';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{artist?.name || 'Artist'} - Outpost</title>
</svelte:head>

{#if loading}
	<p class="text-gray-400">Loading artist...</p>
{:else if error}
	<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded">
		{error}
	</div>
{:else if artist}
	<div class="space-y-6">
		<div class="flex items-start gap-6">
			<div class="w-48 h-48 bg-gray-800 rounded-lg flex-shrink-0 overflow-hidden">
				{#if artist.imagePath}
					<img
						src="/images/{artist.imagePath}"
						alt={artist.name}
						class="w-full h-full object-cover"
					/>
				{:else}
					<div class="w-full h-full flex items-center justify-center text-gray-600">
						<svg class="w-24 h-24" fill="currentColor" viewBox="0 0 24 24">
							<path d="M12 14.25c2.485 0 4.5-2.015 4.5-4.5s-2.015-4.5-4.5-4.5-4.5 2.015-4.5 4.5 2.015 4.5 4.5 4.5zm0 1.5c-3.315 0-9 1.665-9 4.98v.27c0 .825.675 1.5 1.5 1.5h15c.825 0 1.5-.675 1.5-1.5v-.27c0-3.315-5.685-4.98-9-4.98z"/>
						</svg>
					</div>
				{/if}
			</div>
			<div>
				<h1 class="text-3xl font-bold">{artist.name}</h1>
				{#if artist.sortName && artist.sortName !== artist.name}
					<p class="text-gray-400">{artist.sortName}</p>
				{/if}
				<p class="text-gray-400 mt-2">{artist.albums.length} album{artist.albums.length !== 1 ? 's' : ''}</p>
				{#if artist.overview}
					<p class="text-gray-300 mt-4 max-w-2xl">{artist.overview}</p>
				{/if}
			</div>
		</div>

		<div>
			<h2 class="text-xl font-semibold mb-4">Albums</h2>
			{#if artist.albums.length === 0}
				<p class="text-gray-400">No albums found.</p>
			{:else}
				<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
					{#each artist.albums as album}
						<a href="/music/albums/{album.id}" class="group">
							<div class="aspect-square bg-gray-800 rounded-lg overflow-hidden">
								{#if album.coverPath}
									<img
										src="/images/{album.coverPath}"
										alt={album.title}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-gray-600">
										<svg class="w-16 h-16" fill="currentColor" viewBox="0 0 24 24">
											<path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
										</svg>
									</div>
								{/if}
							</div>
							<h3 class="mt-2 font-medium truncate group-hover:text-blue-400">{album.title}</h3>
							{#if album.year}
								<p class="text-sm text-gray-400">{album.year}</p>
							{/if}
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/if}
