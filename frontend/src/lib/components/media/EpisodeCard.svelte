<script lang="ts">
	import { getTmdbImageUrl, getImageUrl } from '$lib/api';

	interface Props {
		number: number;
		title: string;
		duration?: string;
		date?: string;
		imagePath?: string | null;
		isLocal?: boolean;
		watched?: boolean;
		progress?: number;
		inProgress?: boolean;
		onclick?: () => void;
		class?: string;
	}

	let {
		number,
		title,
		duration,
		date,
		imagePath,
		isLocal = false,
		watched = false,
		progress,
		inProgress = false,
		onclick,
		class: className = ''
	}: Props = $props();

	function getImageSrc() {
		if (!imagePath) return null;
		return isLocal ? getImageUrl(imagePath) : getTmdbImageUrl(imagePath, 'w300');
	}
</script>

<button
	{onclick}
	class="group flex-shrink-0 w-60 cursor-pointer transition-all focus:outline-none focus-visible:outline-2 focus-visible:outline-border-focus focus-visible:outline-offset-2 {watched ? 'opacity-60 hover:opacity-100' : ''} {className}"
>
	<!-- Thumbnail -->
	<div class="relative w-full aspect-video bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44] rounded-xl overflow-hidden {inProgress ? 'ring-2 ring-amber-400' : ''}">
		{#if getImageSrc()}
			<img
				src={getImageSrc()}
				alt={title}
				class="w-full h-full object-cover"
				loading="lazy"
			/>
		{/if}

		<!-- Episode number badge -->
		<div class="absolute top-2 left-2 w-7 h-7 rounded-full bg-black/70 flex items-center justify-center text-xs font-semibold">
			{number}
		</div>

		<!-- Duration badge -->
		{#if duration}
			<div class="absolute bottom-2 right-2 px-1.5 py-0.5 bg-black/70 rounded text-[11px]">
				{duration}
			</div>
		{/if}

		<!-- Watched checkmark -->
		{#if watched}
			<div class="status-circle status-watched absolute top-2 right-2 !text-black">
				<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{/if}

		<!-- Play overlay on hover -->
		<div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
			<svg class="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24">
				<path d="M8 5v14l11-7z" />
			</svg>
		</div>

		<!-- Progress bar -->
		{#if progress !== undefined && progress > 0}
			<div class="absolute bottom-0 left-0 right-0 h-[3px] bg-black/50">
				<div class="h-full bg-amber-400" style="width: {progress}%"></div>
			</div>
		{/if}
	</div>

	<!-- Info -->
	<div class="py-2.5 px-1">
		<h4 class="text-sm font-medium text-text-primary truncate">{title}</h4>
		{#if date}
			<p class="text-xs text-text-muted">{date}</p>
		{/if}
	</div>
</button>
