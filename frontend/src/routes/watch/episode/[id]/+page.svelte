<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { getStreamUrl, getEpisode, type EpisodeDetail } from '$lib/api';
	import VideoPlayer from '$lib/components/VideoPlayer.svelte';

	let episodeId: number;
	let episode = $state<EpisodeDetail | null>(null);
	let loading = $state(true);

	onMount(async () => {
		episodeId = parseInt($page.params.id);
		try {
			episode = await getEpisode(episodeId);
		} catch (e) {
			console.error('Failed to fetch episode:', e);
		}
		loading = false;
	});

	function handleClose() {
		goto('/tv');
	}
</script>

<svelte:head>
	<title>Watch Episode - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-screen bg-black">
		<p class="text-gray-400">Loading...</p>
	</div>
{:else}
	<div class="fixed inset-0 bg-black z-50">
		<VideoPlayer
			src={getStreamUrl('episode', episodeId)}
			title={episode?.title || 'Episode'}
			mediaType="episode"
			mediaId={episodeId}
			showId={episode?.showId}
			onClose={handleClose}
		/>
	</div>
{/if}
