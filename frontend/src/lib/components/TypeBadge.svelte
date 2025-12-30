<script lang="ts">
	interface Props {
		type: 'movie' | 'series' | 'episode' | 'anime' | 'music' | 'book' | string;
	}

	let { type }: Props = $props();

	function getLabel(t: string): string {
		switch (t) {
			case 'movie': return 'MOVIE';
			case 'series':
			case 'show':
			case 'tv': return 'SERIES';
			case 'episode': return 'EPISODE';
			case 'anime': return 'ANIME';
			case 'music': return 'MUSIC';
			case 'book': return 'BOOK';
			default: return t.toUpperCase();
		}
	}
</script>

<div class="liquid-badge-sm !bg-black/90 text-white">
	{getLabel(type)}
</div>
