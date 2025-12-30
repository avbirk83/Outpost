<script lang="ts">
	import { getTmdbImageUrl } from '$lib/api';

	interface CastMember {
		name: string;
		character: string;
		profile_path?: string | null;
		id?: number;
	}

	interface Props {
		cast: CastMember[];
		onActorClick?: (actor: CastMember) => void;
	}

	let { cast, onActorClick }: Props = $props();

	let scrollContainer: HTMLDivElement;
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
		setTimeout(updateScrollState, 300);
	}

	function handleClick(actor: CastMember) {
		onActorClick?.(actor);
	}
</script>

<div class="relative">
	<!-- Header -->
	<div class="flex items-center justify-between mb-3">
		<h3 class="text-xs font-medium text-text-muted uppercase tracking-wide">Cast</h3>
		<div class="flex gap-1">
			<button
				onclick={() => scroll('left')}
				class="p-1 rounded bg-white/5 hover:bg-white/10 text-text-muted hover:text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
				disabled={!canScrollLeft}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<button
				onclick={() => scroll('right')}
				class="p-1 rounded bg-white/5 hover:bg-white/10 text-text-muted hover:text-white transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
				disabled={!canScrollRight}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>
	</div>

	<!-- Scrollable row -->
	<div
		bind:this={scrollContainer}
		onscroll={updateScrollState}
		class="flex gap-4 overflow-x-auto scrollbar-hide pb-1"
	>
		{#each cast as actor}
			<button
				onclick={() => handleClick(actor)}
				class="flex-shrink-0 text-center group cursor-pointer focus:outline-none"
			>
				<div class="w-14 h-14 lg:w-16 lg:h-16 rounded-full overflow-hidden bg-bg-elevated ring-2 ring-transparent group-hover:ring-white/30 transition-all">
					{#if actor.profile_path}
						<img
							src={getTmdbImageUrl(actor.profile_path, 'w185')}
							alt={actor.name}
							class="w-full h-full object-cover"
						/>
					{:else}
						<div class="w-full h-full flex items-center justify-center text-text-muted text-lg font-medium">
							{actor.name.charAt(0)}
						</div>
					{/if}
				</div>
				<p class="text-text-primary text-xs mt-1.5 w-14 lg:w-16 truncate group-hover:text-white transition-colors">{actor.name}</p>
				<p class="text-text-muted text-xs w-14 lg:w-16 truncate">{actor.character}</p>
			</button>
		{/each}
	</div>
</div>

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
</style>
