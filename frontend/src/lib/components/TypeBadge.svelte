<script lang="ts">
	interface Props {
		type: 'movie' | 'series' | 'episode' | 'anime' | string;
	}

	let { type }: Props = $props();

	function getLabel(t: string): string {
		switch (t) {
			case 'movie': return 'MOVIE';
			case 'series':
			case 'show':
			case 'tv':
			case 'episode': return 'SERIES';
			case 'anime': return 'ANIME';
			default: return t.toUpperCase();
		}
	}
</script>

<div class="liquid-badge-sm">
	{getLabel(type)}
</div>
