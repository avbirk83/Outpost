<script lang="ts">
	interface Props {
		tmdbId?: number | null;
		imdbId?: string | null;
		mediaType: 'movie' | 'tv';
		class?: string;
	}

	let { tmdbId, imdbId, mediaType, class: className = '' }: Props = $props();

	const tmdbPath = $derived(mediaType === 'movie' ? 'movie' : 'tv');
	const traktPath = $derived(mediaType === 'movie' ? 'movies' : 'shows');
</script>

<div class="flex justify-center gap-3 {className}">
	{#if tmdbId}
		<a
			href="https://www.themoviedb.org/{tmdbPath}/{tmdbId}"
			target="_blank"
			class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
			title="View on TMDB"
		>
			<img src="/icons/tmdb.svg" alt="TMDB" class="w-7 h-7" />
		</a>
	{/if}
	{#if imdbId}
		<a
			href="https://www.imdb.com/title/{imdbId}"
			target="_blank"
			class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
			title="View on IMDb"
		>
			<img src="/icons/imdb.svg" alt="IMDb" class="w-7 h-7" />
		</a>
	{/if}
	<a
		href="https://trakt.tv/search/imdb/{imdbId || ''}"
		target="_blank"
		class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
		title="View on Trakt"
	>
		<img src="/icons/trakt.svg" alt="Trakt" class="w-7 h-7" />
	</a>
	{#if mediaType === 'movie' && tmdbId}
		<a
			href="https://letterboxd.com/tmdb/{tmdbId}"
			target="_blank"
			class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden"
			title="View on Letterboxd"
		>
			<img src="/icons/letterboxd.svg" alt="Letterboxd" class="w-7 h-7" />
		</a>
	{/if}
</div>
