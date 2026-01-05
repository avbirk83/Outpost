<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatRuntime } from '$lib/utils';

	interface Episode {
		id: number;
		episodeNumber: number;
		title?: string;
		runtime?: number;
		stillPath?: string | null;
		overview?: string;
	}

	interface Props {
		episodes: Episode[];
		watchedEpisodes: Set<number>;
		progressMap?: Map<number, number>; // episodeId -> progress percentage
		onToggleWatched: (episodeId: number, runtime?: number) => void;
		togglingEpisode?: number | null;
	}

	let { episodes, watchedEpisodes, progressMap = new Map(), onToggleWatched, togglingEpisode = null }: Props = $props();

	function handlePlay(episodeId: number) {
		goto(`/watch/episode/${episodeId}`);
	}

	function handleToggle(e: MouseEvent, episodeId: number, runtime?: number) {
		e.preventDefault();
		e.stopPropagation();
		onToggleWatched(episodeId, runtime);
	}
</script>

<div class="grid grid-cols-2 gap-1">
	{#each episodes as episode}
		{@const isWatched = watchedEpisodes.has(episode.id)}
		{@const progress = progressMap.get(episode.id)}
		<div
			role="button"
			tabindex="0"
			onclick={() => handlePlay(episode.id)}
			onkeydown={(e) => e.key === 'Enter' && handlePlay(episode.id)}
			class="group flex items-center gap-3 p-2 rounded-lg hover:bg-white/5 transition-colors cursor-pointer"
		>
			<!-- Episode number -->
			<span class="w-6 text-center text-sm font-medium text-text-muted">{episode.episodeNumber}</span>

			<!-- Title -->
			<span class="flex-1 text-sm text-text-primary truncate group-hover:text-cream transition-colors">
				{episode.title || `Episode ${episode.episodeNumber}`}
			</span>

			<!-- Runtime -->
			{#if episode.runtime}
				<span class="text-xs text-text-muted">{formatRuntime(episode.runtime)}</span>
			{/if}

			<!-- Watch status -->
			<button
				onclick={(e) => handleToggle(e, episode.id, episode.runtime)}
				disabled={togglingEpisode === episode.id}
				class="w-6 h-6 flex items-center justify-center flex-shrink-0 rounded transition-colors
					{isWatched ? 'text-green-400' : progress ? 'text-amber-400' : 'text-text-muted hover:text-text-primary'}"
				title={isWatched ? 'Watched' : progress ? `${progress}% watched` : 'Not watched'}
			>
				{#if togglingEpisode === episode.id}
					<div class="spinner-xs"></div>
				{:else if isWatched}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{:else if progress}
					<!-- Progress indicator -->
					<div class="w-4 h-4 rounded-full border-2 border-current relative">
						<div
							class="absolute inset-0.5 rounded-full bg-current"
							style="clip-path: polygon(50% 50%, 50% 0%, {50 + 50 * Math.sin((progress / 100) * 2 * Math.PI)}% {50 - 50 * Math.cos((progress / 100) * 2 * Math.PI)}%, 50% 50%);"
						></div>
					</div>
				{:else}
					<div class="w-4 h-4 rounded-full border-2 border-current opacity-30"></div>
				{/if}
			</button>
		</div>
	{/each}
</div>
