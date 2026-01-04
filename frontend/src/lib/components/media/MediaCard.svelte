<script lang="ts">
	import { getTmdbImageUrl, getImageUrl } from '$lib/api';

	interface Props {
		type?: 'poster' | 'landscape';
		fill?: boolean;
		title: string;
		subtitle?: string;
		imagePath?: string | null;
		isLocal?: boolean;
		href?: string;
		onclick?: () => void;
		class?: string;

		// Media type indicator
		mediaType?: 'movie' | 'tv';

		// Media info
		runtime?: number;
		contentRating?: string;

		// Status badges (icon-based)
		inLibrary?: boolean;
		requested?: boolean;
		requestStatus?: 'pending' | 'approved' | 'available';

		// Watch state (icon-based indicators)
		watchState?: 'unwatched' | 'partial' | 'watched';

		// Legacy badge support (for special cases like "New")
		badge?: string;
		badgeVariant?: 'default' | 'success' | 'new';

		// Progress
		progress?: number;
	}

	let {
		type = 'poster',
		fill = false,
		title,
		subtitle,
		imagePath,
		isLocal = false,
		href,
		onclick,
		class: className = '',
		mediaType,
		runtime,
		contentRating,
		inLibrary = false,
		requested = false,
		requestStatus,
		watchState,
		badge,
		badgeVariant = 'default',
		progress
	}: Props = $props();

	function formatRuntime(minutes: number): string {
		if (minutes < 60) return `${minutes}m`;
		const hours = Math.floor(minutes / 60);
		const mins = minutes % 60;
		return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
	}

	const widthClass = $derived(fill ? 'w-full' : (type === 'poster' ? 'w-[200px]' : 'w-72'));
	const aspectClass = $derived(type === 'poster' ? 'aspect-[2/3]' : 'aspect-video');

	// Auto-detect media type from href if not explicitly provided
	const detectedMediaType = $derived(() => {
		if (mediaType) return mediaType;
		if (!href) return undefined;
		if (href.includes('/movie') || href.includes('/movies')) return 'movie';
		if (href.includes('/tv') || href.includes('/show')) return 'tv';
		return undefined;
	});

	function getImageSrc() {
		if (!imagePath) return null;
		return isLocal ? getImageUrl(imagePath) : getTmdbImageUrl(imagePath, type === 'poster' ? 'w342' : 'w500');
	}

	const badgeColors = {
		default: 'bg-white/20 text-white',
		success: 'bg-green-500 text-black',
		new: 'bg-green-500 text-black'
	};
</script>

<svelte:element
	this={href ? 'a' : 'button'}
	{href}
	{onclick}
	class="group flex-shrink-0 {widthClass} rounded-xl overflow-hidden bg-bg-elevated cursor-pointer transition-all duration-300 hover:scale-[1.02] hover:shadow-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus {className}"
>
	<!-- Image container -->
	<div class="relative {aspectClass} bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44]">
		{#if getImageSrc()}
			<img
				src={getImageSrc()}
				alt={title}
				class="w-full h-full object-cover"
				loading="lazy"
			/>
		{/if}

		<!-- Custom badge - top left -->
		{#if badge}
			<div class="absolute top-2 left-2 px-2 py-1 rounded text-[10px] font-bold uppercase {badgeColors[badgeVariant]}">
				{badge}
			</div>
		{/if}

		<!-- Status indicator - bottom right (circular icons) -->
		{#if watchState === 'watched'}
			<div class="absolute bottom-2 right-2 w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="Watched">
				<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{:else if watchState === 'partial'}
			<div class="absolute bottom-2 right-2 w-6 h-6 rounded-full bg-amber-500 flex items-center justify-center" title="In Progress">
				<svg class="w-3.5 h-3.5 text-white" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5z" />
				</svg>
			</div>
		{:else if requestStatus === 'approved' || requestStatus === 'available' || inLibrary}
			<div class="absolute bottom-2 right-2 w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="Available">
				<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{:else if requestStatus === 'pending' || requested}
			<div class="absolute bottom-2 right-2 w-6 h-6 rounded-full bg-amber-500 flex items-center justify-center" title="Requested">
				<svg class="w-3.5 h-3.5 text-white" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z" />
				</svg>
			</div>
		{/if}

		<!-- Hover overlay -->
		<div class="absolute inset-0 bg-black/30 opacity-0 group-hover:opacity-100 transition-opacity"></div>

		<!-- Progress bar -->
		{#if progress !== undefined && progress > 0}
			<div class="absolute bottom-0 left-0 right-0 h-[3px] bg-black/50">
				<div class="h-full bg-amber-400" style="width: {progress}%"></div>
			</div>
		{/if}
	</div>

	<!-- Info -->
	<div class="p-3">
		<h3 class="text-sm font-semibold text-text-primary truncate">{title}</h3>
		<div class="flex items-center gap-1.5 mt-1 flex-wrap">
			{#if detectedMediaType()}
				<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
					{detectedMediaType() === 'movie' ? 'Movie' : 'TV'}
				</span>
			{/if}
			{#if runtime}
				<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
					{formatRuntime(runtime)}
				</span>
			{/if}
			{#if contentRating}
				<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
					{contentRating}
				</span>
			{/if}
			{#if subtitle}
				<p class="text-xs text-text-muted truncate">{subtitle}</p>
			{/if}
		</div>
	</div>
</svelte:element>
