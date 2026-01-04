<script lang="ts">
	import { getImageUrl } from '$lib/api';

	interface Props {
		filename: string;
		codec?: string;
		audio?: string;
		size?: string;
		badge?: string;
		thumbnailPath?: string | null;
		onclick?: () => void;
		class?: string;
	}

	let {
		filename,
		codec,
		audio,
		size,
		badge,
		thumbnailPath,
		onclick,
		class: className = ''
	}: Props = $props();

	const meta = [codec, audio, size].filter(Boolean).join(' • ');
</script>

<button
	{onclick}
	class="w-72 bg-glass backdrop-blur-xl border border-border-subtle rounded-xl overflow-hidden cursor-pointer hover:bg-glass-hover transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus {className}"
>
	<!-- Thumbnail -->
	<div class="h-24 bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44] relative">
		{#if thumbnailPath}
			<img
				src={getImageUrl(thumbnailPath)}
				alt=""
				class="w-full h-full object-cover"
			/>
		{/if}

		{#if badge}
			<div class="absolute top-2 left-2 px-1.5 py-0.5 bg-black/60 rounded text-[10px] font-semibold">
				{badge}
			</div>
		{/if}
	</div>

	<!-- Info -->
	<div class="p-3">
		<h4 class="text-sm font-medium text-text-primary truncate">{filename}</h4>
		{#if meta}
			<p class="text-xs text-text-muted">{meta}</p>
		{/if}
	</div>
</button>
