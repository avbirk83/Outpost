<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		viewAllHref?: string;
		children: Snippet;
	}

	let { title, viewAllHref, children }: Props = $props();

	let scrollContainer: HTMLDivElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);

	function updateScrollState() {
		if (!scrollContainer) return;
		canScrollLeft = scrollContainer.scrollLeft > 0;
		canScrollRight = scrollContainer.scrollLeft < scrollContainer.scrollWidth - scrollContainer.clientWidth - 10;
	}

	function scrollLeft() {
		scrollContainer?.scrollBy({ left: -400, behavior: 'smooth' });
		setTimeout(updateScrollState, 350);
	}

	function scrollRight() {
		scrollContainer?.scrollBy({ left: 400, behavior: 'smooth' });
		setTimeout(updateScrollState, 350);
	}
</script>

<section class="relative">
	<!-- Header -->
	<div class="flex items-center justify-between mb-4">
		<h2 class="text-xl font-semibold text-text-primary">{title}</h2>

		<div class="flex items-center gap-2">
			{#if viewAllHref}
				<a
					href={viewAllHref}
					class="text-sm text-text-secondary hover:text-outpost-400 transition-colors"
				>
					View All
				</a>
			{/if}

			<!-- Scroll buttons -->
			<div class="flex items-center gap-1 ml-2">
				<button
					onclick={scrollLeft}
					disabled={!canScrollLeft}
					class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
					aria-label="Scroll left"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button
					onclick={scrollRight}
					disabled={!canScrollRight}
					class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
					aria-label="Scroll right"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		</div>
	</div>

	<!-- Scrollable row -->
	<div
		bind:this={scrollContainer}
		onscroll={updateScrollState}
		class="flex gap-4 overflow-x-auto scrollbar-hide pb-2 -mx-6 px-6"
	>
		{@render children()}
	</div>
</section>
