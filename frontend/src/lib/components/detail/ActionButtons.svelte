<script lang="ts">
	interface Props {
		inLibrary?: boolean;
		inWatchlist?: boolean;
		watched?: boolean;
		hasTrailer?: boolean;
		onPlay?: () => void;
		onTrailer?: () => void;
		onToggleWatchlist?: () => void;
		onToggleWatched?: () => void;
		onMore?: () => void;
		disabled?: boolean;
		class?: string;
	}

	let {
		inLibrary = false,
		inWatchlist = false,
		watched = false,
		hasTrailer = false,
		onPlay,
		onTrailer,
		onToggleWatchlist,
		onToggleWatched,
		onMore,
		disabled = false,
		class: className = ''
	}: Props = $props();
</script>

<div class="flex items-center justify-center gap-2 {className}">
	<!-- In Library / Watchlist -->
	<button
		onclick={onToggleWatchlist}
		{disabled}
		class="{inLibrary || inWatchlist ? 'btn-action-success' : 'btn-action-glass'}"
		title={inLibrary ? 'In Library' : inWatchlist ? 'In Watchlist' : 'Add to Watchlist'}
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
		</svg>
	</button>

	<!-- Watched -->
	<button
		onclick={onToggleWatched}
		{disabled}
		class="{watched ? 'btn-action-success' : 'btn-action-glass'}"
		title={watched ? 'Watched' : 'Mark as Watched'}
	>
		<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
			<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
		</svg>
	</button>

	<!-- Play (larger, amber) -->
	{#if onPlay}
		<button onclick={onPlay} {disabled} class="btn-action-primary" title="Play">
			<svg class="w-5 h-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
				<path d="M8 5v14l11-7z" />
			</svg>
		</button>
	{/if}

	<!-- Trailer (red) -->
	{#if hasTrailer && onTrailer}
		<button onclick={onTrailer} class="btn-action-trailer" title="Watch Trailer">
			<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
				<path d="M19.615 3.184c-3.604-.246-11.631-.245-15.23 0-3.897.266-4.356 2.62-4.385 8.816.029 6.185.484 8.549 4.385 8.816 3.6.245 11.626.246 15.23 0 3.897-.266 4.356-2.62 4.385-8.816-.029-6.185-.484-8.549-4.385-8.816zm-10.615 12.816v-8l8 3.993-8 4.007z"/>
			</svg>
		</button>
	{/if}

	<!-- More options -->
	{#if onMore}
		<button onclick={onMore} class="btn-action-glass" title="More Options">
			<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
				<path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/>
			</svg>
		</button>
	{/if}
</div>
