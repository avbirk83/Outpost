<script lang="ts">
	import { getImageUrl } from '$lib/api';

	interface Props {
		backdropPath?: string | null;
		posterPath?: string | null;
		title: string;
		tagline?: string | null;
		focalX?: number;
		focalY?: number;
		children?: import('svelte').Snippet;
		posterChildren?: import('svelte').Snippet;
	}

	let { backdropPath, posterPath, title, tagline, focalX, focalY, children, posterChildren }: Props = $props();

	function getFocalPosition(fx?: number, fy?: number): string {
		const x = fx !== undefined ? Math.round(fx * 100) : 50;
		const y = fy !== undefined ? Math.round(fy * 100) : 25;
		return `${x}% ${y}%`;
	}
</script>

<!-- Full viewport hero - no scroll -->
<div class="fixed inset-0 overflow-hidden">
	<!-- Backdrop -->
	{#if backdropPath}
		<img
			src={getImageUrl(backdropPath)}
			alt=""
			class="absolute inset-0 w-full h-full object-cover"
			style="object-position: {getFocalPosition(focalX, focalY)};"
		/>
	{/if}

	<!-- Gradient overlays -->
	<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/85 to-bg-primary/40"></div>
	<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/60"></div>

	<!-- Content grid -->
	<div class="relative h-full flex flex-col">
		<!-- Main content area -->
		<div class="flex-1 flex gap-6 lg:gap-10 p-6 pt-4">
			<!-- Poster column -->
			<div class="flex-shrink-0 w-44 lg:w-56 flex flex-col">
				{#if posterPath}
					<img
						src={getImageUrl(posterPath)}
						alt={title}
						class="w-full rounded-lg shadow-2xl"
					/>
				{:else}
					<div class="w-full aspect-[2/3] bg-bg-elevated rounded-lg flex items-center justify-center">
						<svg class="w-12 h-12 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
						</svg>
					</div>
				{/if}
				<!-- Poster slot for buttons etc -->
				{#if posterChildren}
					<div class="mt-3">
						{@render posterChildren()}
					</div>
				{/if}
			</div>

			<!-- Info column -->
			<div class="flex-1 flex flex-col min-w-0">
				{#if children}
					{@render children()}
				{/if}
			</div>
		</div>
	</div>
</div>
