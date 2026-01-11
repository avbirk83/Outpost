<script lang="ts">
	import type { ContentRating } from '$lib/api';

	interface Props {
		rating?: string | null;
		size?: 'sm' | 'md' | 'lg';
	}

	let { rating, size = 'md' }: Props = $props();

	// Normalize rating to standard MPAA ratings
	const normalizedRating = $derived.by(() => {
		if (!rating) return null;

		// Already normalized
		if (['G', 'PG', 'PG-13', 'R', 'NC-17'].includes(rating)) {
			return rating as ContentRating;
		}

		// US TV ratings
		if (['TV-Y', 'TV-Y7', 'TV-G'].includes(rating)) return 'G';
		if (rating === 'TV-PG') return 'PG';
		if (rating === 'TV-14') return 'PG-13';
		if (rating === 'TV-MA') return 'R';

		// UK ratings
		if (['U', 'Uc'].includes(rating)) return 'G';
		if (['12', '12A'].includes(rating)) return 'PG-13';
		if (rating === '15') return 'R';
		if (['18', 'R18'].includes(rating)) return 'NC-17';

		// Return original if unrecognized
		return rating;
	});

	const colorClass = $derived.by(() => {
		switch (normalizedRating) {
			case 'G':
				return 'green';
			case 'PG':
				return 'green';
			case 'PG-13':
				return 'yellow';
			case 'R':
				return 'orange';
			case 'NC-17':
				return 'red';
			default:
				return 'gray';
		}
	});

	const sizeClass = $derived.by(() => {
		switch (size) {
			case 'sm':
				return 'text-xs px-1.5 py-0.5';
			case 'lg':
				return 'text-sm px-3 py-1';
			default:
				return 'text-xs px-2 py-0.5';
		}
	});
</script>

{#if rating}
	<span class="rating-badge {colorClass} {sizeClass}">
		{normalizedRating || rating}
	</span>
{/if}

<style>
	.rating-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 600;
		border-radius: 4px;
		white-space: nowrap;
	}

	.green {
		background: rgba(34, 197, 94, 0.15);
		color: #22c55e;
		border: 1px solid rgba(34, 197, 94, 0.3);
	}

	.yellow {
		background: rgba(234, 179, 8, 0.15);
		color: #eab308;
		border: 1px solid rgba(234, 179, 8, 0.3);
	}

	.orange {
		background: rgba(249, 115, 22, 0.15);
		color: #f97316;
		border: 1px solid rgba(249, 115, 22, 0.3);
	}

	.red {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
		border: 1px solid rgba(239, 68, 68, 0.3);
	}

	.gray {
		background: rgba(156, 163, 175, 0.15);
		color: #9ca3af;
		border: 1px solid rgba(156, 163, 175, 0.3);
	}
</style>
