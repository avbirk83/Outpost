<script lang="ts">
	interface Props {
		tmdbId?: number | null;
		tmdbRating?: number | null;
		mediaType: 'movie' | 'tv';
		rottenTomatoes?: number | null;
		metacritic?: number | null;
		class?: string;
	}

	let { tmdbId, tmdbRating, mediaType, rottenTomatoes, metacritic, class: className = '' }: Props = $props();

	const tmdbPath = $derived(mediaType === 'movie' ? 'movie' : 'tv');
</script>

<div class="flex justify-around items-center {className}">
	{#if tmdbRating}
		<a
			href="https://www.themoviedb.org/{tmdbPath}/{tmdbId}"
			target="_blank"
			class="flex items-center gap-1.5 hover:opacity-80 transition-opacity"
			title="TMDB Rating"
		>
			<img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6 rounded" />
			<span class="text-base font-bold text-text-primary">{tmdbRating.toFixed(1)}</span>
		</a>
	{/if}
	<div
		class="flex items-center gap-1.5 {rottenTomatoes ? '' : 'opacity-40'}"
		title={rottenTomatoes ? 'Rotten Tomatoes Score' : 'Rotten Tomatoes (coming soon)'}
	>
		<img src="/icons/rottentomatoes.svg" alt="Rotten Tomatoes" class="w-6 h-6" />
		<span class="text-base font-bold">{rottenTomatoes ?? '--'}</span>
	</div>
	<div
		class="flex items-center gap-1.5 {metacritic ? '' : 'opacity-40'}"
		title={metacritic ? 'Metacritic Score' : 'Metacritic (coming soon)'}
	>
		<img src="/icons/metacritic.svg" alt="Metacritic" class="w-6 h-6 rounded" />
		<span class="text-base font-bold">{metacritic ?? '--'}</span>
	</div>
</div>
