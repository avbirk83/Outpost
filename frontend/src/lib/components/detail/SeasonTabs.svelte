<script lang="ts">
	interface Season {
		number: number;
		name?: string;
		episodeCount?: number;
	}

	interface Props {
		seasons: Season[];
		selected: number;
		onChange: (seasonNumber: number) => void;
		class?: string;
	}

	let { seasons, selected, onChange, class: className = '' }: Props = $props();
</script>

<div class="flex gap-1 flex-wrap {className}">
	{#each seasons as season}
		<button
			onclick={() => onChange(season.number)}
			class="px-5 py-2.5 rounded-full text-sm font-medium transition-all
				{selected === season.number
					? 'bg-amber-400 border-amber-400 text-black'
					: 'bg-glass border border-border-subtle text-text-secondary hover:bg-glass-hover hover:text-text-primary'}"
		>
			{season.name || `Season ${season.number}`}
		</button>
	{/each}
</div>
