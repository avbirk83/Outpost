<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getAlbum, type AlbumDetail, type Track } from '$lib/api';

	let album: AlbumDetail | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let currentTrack: Track | null = $state(null);
	let isPlaying = $state(false);
	let audioElement: HTMLAudioElement | null = $state(null);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			album = await getAlbum(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load album';
		} finally {
			loading = false;
		}
	});

	function formatDuration(seconds: number): string {
		if (!seconds) return '--:--';
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function playTrack(track: Track) {
		if (currentTrack?.id === track.id) {
			if (isPlaying) {
				audioElement?.pause();
			} else {
				audioElement?.play();
			}
		} else {
			currentTrack = track;
			isPlaying = true;
		}
	}

	function handleAudioPlay() {
		isPlaying = true;
	}

	function handleAudioPause() {
		isPlaying = false;
	}

	function handleAudioEnded() {
		isPlaying = false;
		// Play next track
		if (album && currentTrack) {
			const currentIndex = album.tracks.findIndex(t => t.id === currentTrack!.id);
			if (currentIndex < album.tracks.length - 1) {
				playTrack(album.tracks[currentIndex + 1]);
			}
		}
	}
</script>

<svelte:head>
	<title>{album?.title || 'Album'} - Outpost</title>
</svelte:head>

{#if loading}
	<p class="text-gray-400">Loading album...</p>
{:else if error}
	<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded">
		{error}
	</div>
{:else if album}
	<div class="space-y-6">
		<div class="flex items-start gap-6">
			<div class="w-48 h-48 bg-gray-800 rounded-lg flex-shrink-0 overflow-hidden">
				{#if album.coverPath}
					<img
						src="/images/{album.coverPath}"
						alt={album.title}
						class="w-full h-full object-cover"
					/>
				{:else}
					<div class="w-full h-full flex items-center justify-center text-gray-600">
						<svg class="w-24 h-24" fill="currentColor" viewBox="0 0 24 24">
							<path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
						</svg>
					</div>
				{/if}
			</div>
			<div>
				<h1 class="text-3xl font-bold">{album.title}</h1>
				{#if album.artist}
					<a href="/music/artists/{album.artist.id}" class="text-blue-400 hover:underline text-lg">
						{album.artist.name}
					</a>
				{/if}
				{#if album.year}
					<p class="text-gray-400 mt-1">{album.year}</p>
				{/if}
				<p class="text-gray-400 mt-2">{album.tracks.length} track{album.tracks.length !== 1 ? 's' : ''}</p>
				{#if album.overview}
					<p class="text-gray-300 mt-4 max-w-2xl">{album.overview}</p>
				{/if}
			</div>
		</div>

		<div>
			<h2 class="text-xl font-semibold mb-4">Tracks</h2>
			{#if album.tracks.length === 0}
				<p class="text-gray-400">No tracks found.</p>
			{:else}
				<div class="bg-gray-800 rounded-lg overflow-hidden">
					{#each album.tracks as track, index}
						<button
							class="w-full flex items-center gap-4 px-4 py-3 hover:bg-gray-700 transition-colors text-left {currentTrack?.id === track.id ? 'bg-gray-700' : ''}"
							onclick={() => playTrack(track)}
						>
							<span class="w-8 text-gray-400 text-center">
								{#if currentTrack?.id === track.id && isPlaying}
									<svg class="w-4 h-4 mx-auto text-blue-400" fill="currentColor" viewBox="0 0 24 24">
										<path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
									</svg>
								{:else}
									{track.trackNumber}
								{/if}
							</span>
							<div class="flex-1 min-w-0">
								<p class="truncate {currentTrack?.id === track.id ? 'text-blue-400' : ''}">{track.title}</p>
							</div>
							<span class="text-gray-400 text-sm">{formatDuration(track.duration)}</span>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Audio Player -->
	{#if currentTrack}
		<div class="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-700 p-4">
			<div class="max-w-7xl mx-auto flex items-center gap-4">
				<div class="w-12 h-12 bg-gray-800 rounded flex-shrink-0 overflow-hidden">
					{#if album.coverPath}
						<img src="/images/{album.coverPath}" alt={album.title} class="w-full h-full object-cover" />
					{:else}
						<div class="w-full h-full flex items-center justify-center text-gray-600">
							<svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
								<path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
							</svg>
						</div>
					{/if}
				</div>
				<div class="flex-1 min-w-0">
					<p class="font-medium truncate">{currentTrack.title}</p>
					<p class="text-sm text-gray-400 truncate">{album.artist?.name}</p>
				</div>
				<audio
					bind:this={audioElement}
					src="/api/stream/track/{currentTrack.id}"
					autoplay
					onplay={handleAudioPlay}
					onpause={handleAudioPause}
					onended={handleAudioEnded}
					controls
					class="w-64"
				></audio>
			</div>
		</div>
	{/if}
{/if}
