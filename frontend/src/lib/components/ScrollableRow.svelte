<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		children: Snippet;
		class?: string;
		padding?: string;
	}

	let { title, children, class: className = '', padding = 'px-6' }: Props = $props();

	let scrollContainer: HTMLElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);

	function updateScrollState() {
		if (!scrollContainer) return;
		canScrollLeft = scrollContainer.scrollLeft > 0;
		canScrollRight = scrollContainer.scrollLeft < scrollContainer.scrollWidth - scrollContainer.clientWidth - 10;
	}

	function scroll(direction: 'left' | 'right') {
		if (!scrollContainer) return;
		const scrollAmount = 300;
		scrollContainer.scrollBy({
			left: direction === 'left' ? -scrollAmount : scrollAmount,
			behavior: 'smooth'
		});
		setTimeout(updateScrollState, 350);
	}
</script>

<section class="{padding} {className}">
	<div class="flex items-center justify-between mb-3">
		<h2 class="text-lg font-semibold text-text-primary">{title}</h2>
		<div class="flex gap-1">
			<button
				onclick={() => scroll('left')}
				disabled={!canScrollLeft}
				class="liquid-btn-icon !w-8 !h-8 !rounded-full disabled:opacity-30 disabled:cursor-not-allowed"
				aria-label="Scroll left"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<button
				onclick={() => scroll('right')}
				disabled={!canScrollRight}
				class="liquid-btn-icon !w-8 !h-8 !rounded-full disabled:opacity-30 disabled:cursor-not-allowed"
				aria-label="Scroll right"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>
	</div>
	<div
		bind:this={scrollContainer}
		onscroll={updateScrollState}
		class="flex gap-5 overflow-x-auto pt-1 pl-1 pb-2 -ml-1 scrollbar-thin"
	>
		{@render children()}
	</div>
</section>
