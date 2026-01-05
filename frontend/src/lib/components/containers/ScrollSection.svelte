<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		linkText?: string;
		linkHref?: string;
		meta?: string;
		children: Snippet;
		headerExtra?: Snippet;
		class?: string;
	}

	let { title, linkText, linkHref, meta, children, headerExtra, class: className = '' }: Props = $props();

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

<section class="{className}">
	<!-- Header -->
	<div class="flex items-center justify-between mb-4 px-[60px] relative z-10">
		<div class="flex items-center gap-4">
			<h2 class="text-xl font-semibold text-text-primary">{title}</h2>
			{#if meta}
				<span class="text-sm text-text-muted">{meta}</span>
			{/if}
		</div>

		<div class="flex items-center gap-3">
			{#if headerExtra}
				{@render headerExtra()}
			{/if}

			{#if linkText && linkHref}
				<a
					href={linkHref}
					class="text-sm text-text-secondary hover:text-text-primary px-4 py-2 rounded-full hover:bg-glass transition-all min-h-[44px] flex items-center"
				>
					{linkText}
				</a>
			{/if}

			<div class="flex gap-1">
				<button onclick={() => scroll('left')} disabled={!canScrollLeft} class="btn-icon-circle-xs" aria-label="Scroll left">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button onclick={() => scroll('right')} disabled={!canScrollRight} class="btn-icon-circle-xs" aria-label="Scroll right">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		</div>
	</div>

	<!-- Scroll Row -->
	<div
		bind:this={scrollContainer}
		onscroll={updateScrollState}
		class="flex gap-4 overflow-x-auto px-[60px] pb-6 scroll-snap-x scrollbar-hide"
		style="scroll-snap-type: x mandatory;"
	>
		{@render children()}
	</div>
</section>
