<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { getMovie, getStreamUrl, type Movie } from '$lib/api';
	import VideoPlayer from '$lib/components/VideoPlayer.svelte';

	let movie: Movie | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let initialSubtitle: number | null = null;

	onMount(async () => {
		const id = parseInt($page.params.id);
		// Check for subtitle parameter
		const subParam = $page.url.searchParams.get('sub');
		if (subParam !== null) {
			initialSubtitle = parseInt(subParam);
		}
		try {
			movie = await getMovie(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load movie';
		} finally {
			loading = false;
		}
	});

	function handleClose() {
		goto('/library');
	}
</script>

<svelte:head>
	<title>{movie?.title || 'Watch'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-screen bg-black">
		<p class="text-gray-400">Loading...</p>
	</div>
{:else if error}
	<div class="flex items-center justify-center h-screen bg-black">
		<div class="text-center">
			<p class="text-text-secondary mb-4">{error}</p>
			<button onclick={() => goto('/library')} class="text-text-secondary hover:text-text-primary">Back to Library</button>
		</div>
	</div>
{:else if movie}
	<div class="fixed inset-0 bg-black z-50">
		<VideoPlayer
			src={getStreamUrl('movie', movie.id)}
			title={movie.title}
			mediaType="movie"
			mediaId={movie.id}
			onClose={handleClose}
			{initialSubtitle}
		/>
	</div>
{/if}
