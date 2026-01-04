<script lang="ts">
	import ExternalLinks from '$lib/components/ExternalLinks.svelte';

	interface InfoRow {
		label: string;
		value: string;
		variant?: 'default' | 'accent' | 'success';
		link?: string;
	}

	interface CriticScore {
		source: 'tmdb' | 'rt' | 'metacritic';
		score: number | null;
		icon: string;
	}

	interface Props {
		scores?: CriticScore[];
		rows: InfoRow[];
		tmdbId?: number | null;
		imdbId?: string | null;
		mediaType: 'movie' | 'tv';
		class?: string;
	}

	let { scores = [], rows, tmdbId, imdbId, mediaType, class: className = '' }: Props = $props();

	const variantColors = {
		default: 'text-text-primary',
		accent: 'text-amber-400',
		success: 'text-green-400'
	};
</script>

<div class="bg-glass backdrop-blur-xl border border-border-subtle rounded-lg p-4 {className}" style="background: rgba(255, 255, 255, 0.06);">
	<!-- Critic Scores -->
	{#if scores.length > 0}
		<div class="flex justify-around py-2">
			{#each scores as score}
				<div class="flex flex-col items-center gap-1">
					<span class="text-xl">{score.icon}</span>
					<span class="text-sm font-semibold">
						{score.score !== null ? score.score.toFixed(1) : '--'}
					</span>
				</div>
			{/each}
		</div>
		<div class="h-px bg-border-subtle my-2"></div>
	{/if}

	<!-- Info Rows -->
	{#each rows as row, i}
		{#if row.label === '---'}
			<div class="h-px bg-border-subtle my-2"></div>
		{:else}
			<div class="flex justify-between items-center py-2">
				<span class="text-sm text-text-muted">{row.label}</span>
				<span class="text-sm font-medium text-right {variantColors[row.variant || 'default']}">
					{#if row.link}
						<a href={row.link} target="_blank" class="text-amber-400 hover:underline">
							{row.value}
						</a>
					{:else}
						{row.value}
					{/if}
				</span>
			</div>
		{/if}
	{/each}

	<!-- External Links -->
	{#if tmdbId || imdbId}
		<div class="h-px bg-border-subtle my-2"></div>
		<div class="pt-3">
			<ExternalLinks {tmdbId} {imdbId} {mediaType} />
		</div>
	{/if}
</div>
