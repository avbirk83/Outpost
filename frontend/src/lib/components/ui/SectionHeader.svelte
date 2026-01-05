<script lang="ts">
	interface Props {
		title: string;
		linkText?: string;
		linkHref?: string;
		onScrollLeft?: () => void;
		onScrollRight?: () => void;
		canScrollLeft?: boolean;
		canScrollRight?: boolean;
		class?: string;
	}

	let {
		title,
		linkText,
		linkHref,
		onScrollLeft,
		onScrollRight,
		canScrollLeft = true,
		canScrollRight = true,
		class: className = ''
	}: Props = $props();
</script>

<div class="flex items-center justify-between mb-4 {className}">
	<h2 class="text-xl font-semibold text-text-primary">{title}</h2>

	<div class="flex items-center gap-4">
		{#if linkText && linkHref}
			<a
				href={linkHref}
				class="text-sm text-text-secondary hover:text-text-primary px-4 py-2 rounded-full hover:bg-glass transition-all"
			>
				{linkText}
			</a>
		{/if}

		{#if onScrollLeft && onScrollRight}
			<div class="flex gap-1">
				<button onclick={onScrollLeft} disabled={!canScrollLeft} class="btn-icon-circle-xs" aria-label="Scroll left">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button onclick={onScrollRight} disabled={!canScrollRight} class="btn-icon-circle-xs" aria-label="Scroll right">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		{/if}
	</div>
</div>
