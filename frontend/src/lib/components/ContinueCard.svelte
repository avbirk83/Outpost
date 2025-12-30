<script lang="ts">
	import TypeBadge from './TypeBadge.svelte';

	interface Props {
		id: number;
		title: string;
		subtitle?: string;
		backdropUrl?: string;
		posterUrl?: string;
		progress: number;
		duration?: string;
		type: 'movie' | 'episode';
		href?: string;
		onRemove?: () => void;
	}

	let { id, title, subtitle, backdropUrl, posterUrl, progress, duration, type, href, onRemove }: Props = $props();

	const link = href ?? (type === 'movie' ? `/watch/movie/${id}` : `/watch/episode/${id}`);
</script>

<div class="group relative">
	<a href={link} class="block">
		<div class="relative aspect-video bg-bg-card overflow-hidden rounded-xl">
			<!-- Backdrop/Poster image -->
			{#if backdropUrl}
				<img
					src={backdropUrl}
					alt={title}
					class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
					loading="lazy"
				/>
			{:else if posterUrl}
				<img
					src={posterUrl}
					alt={title}
					class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
					loading="lazy"
				/>
			{:else}
				<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
					<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
			{/if}

			<!-- Gradient overlay -->
			<div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>

			<!-- Play button - centered, appears on hover -->
			<div class="absolute inset-0 flex items-center justify-center">
				<div class="w-14 h-14 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transform scale-75 group-hover:scale-100 transition-all duration-300 border border-white/30">
					<svg class="w-7 h-7 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
						<path d="M8 5v14l11-7z" />
					</svg>
				</div>
			</div>

			<!-- Type badge - top left -->
			<div class="absolute top-3 left-3">
				<TypeBadge type={type} />
			</div>

			<!-- Content overlay at bottom -->
			<div class="absolute bottom-0 left-0 right-0 p-4">
				<h3 class="text-base font-semibold text-white truncate">{title}</h3>
				{#if subtitle}
					<p class="text-sm text-white/70 truncate mt-0.5">{subtitle}</p>
				{/if}
				<!-- Progress bar -->
				<div class="mt-3 flex items-center gap-3">
					<div class="flex-1 h-1 bg-white/30 rounded-full overflow-hidden">
						<div
							class="h-full bg-white/50 rounded-full transition-all duration-300"
							style="width: {progress}%"
						></div>
					</div>
					{#if duration}
						<span class="text-xs text-white/70 whitespace-nowrap">{duration}</span>
					{/if}
				</div>
			</div>
		</div>
	</a>

	<!-- Remove button - top right, on hover -->
	{#if onRemove}
		<button
			onclick={(e) => { e.preventDefault(); e.stopPropagation(); onRemove(); }}
			class="absolute top-3 right-3 w-7 h-7 rounded-full bg-black/80 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-black"
			title="Remove from Continue Watching"
		>
			<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	{/if}
</div>
