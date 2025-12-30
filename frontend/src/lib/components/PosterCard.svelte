<script lang="ts">
	import TypeBadge from './TypeBadge.svelte';

	interface Props {
		href: string;
		title: string;
		subtitle?: string;
		posterUrl?: string;
		rating?: number;
		progress?: number;
		badge?: string;
		badgeColor?: string;
		// New Phase 11 props
		mediaType?: 'movie' | 'series' | 'anime' | 'music' | 'book';
		inLibrary?: boolean;
		requested?: boolean;
		requestStatus?: 'pending' | 'approved' | 'rejected';
		watchState?: 'unwatched' | 'partial' | 'watched';
		downloadProgress?: number; // 0-100, shown when downloading
		episodeProgress?: string; // e.g., "4 of 12"
		// Episode counts (used to compute episodeProgress if not provided directly)
		watchedEpisodes?: number;
		totalEpisodes?: number;
		// Request callback for discover cards
		onRequest?: (e: MouseEvent) => void;
	}

	let {
		href,
		title,
		subtitle,
		posterUrl,
		rating,
		progress,
		badge,
		badgeColor = 'bg-black/70',
		mediaType,
		inLibrary,
		requested,
		requestStatus,
		watchState,
		downloadProgress,
		episodeProgress,
		watchedEpisodes,
		totalEpisodes,
		onRequest
	}: Props = $props();

	// Compute episode progress string if not directly provided
	const computedEpisodeProgress = $derived(() => {
		if (episodeProgress) return episodeProgress;
		if (totalEpisodes && totalEpisodes > 0) {
			const watched = watchedEpisodes || 0;
			return `${watched}/${totalEpisodes}`;
		}
		return undefined;
	});

</script>

<a
	{href}
	class="group block poster-card hover-lift"
>
	<!-- Poster image -->
	<div class="relative w-full aspect-[2/3] bg-bg-card overflow-hidden rounded-lg">
		{#if posterUrl}
			<img
				src={posterUrl}
				alt={title}
				class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
				loading="lazy"
			/>
		{:else}
			<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
				<svg class="w-12 h-12 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
			</div>
		{/if}

		<!-- Gradient overlay on hover -->
		<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>

		<!-- Play button overlay -->
		<div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300">
			<div class="w-14 h-14 rounded-full bg-white/30 flex items-center justify-center transform scale-75 group-hover:scale-100 transition-transform duration-300">
				<svg class="w-6 h-6 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
					<path d="M8 5v14l11-7z" />
				</svg>
			</div>
		</div>

		<!-- Type badge - top left -->
		{#if mediaType}
			<div class="absolute top-1 left-1">
				<TypeBadge type={mediaType} />
			</div>
		{:else if badge}
			<!-- Legacy badge support -->
			<div class="absolute top-1 left-1">
				<TypeBadge type={badge} />
			</div>
		{/if}

		<!-- Rating badge - top right -->
		{#if rating}
			<div class="absolute top-1 right-1 liquid-badge-sm !bg-black/90 !gap-1">
				<svg class="w-2.5 h-2.5 text-white" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
				</svg>
				{rating.toFixed(1)}
			</div>
		{/if}

		<!-- Watch state indicator - bottom right -->
		{#if watchState === 'watched'}
			<div class="absolute bottom-1 right-1 w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="Watched">
				<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{:else if watchState === 'partial'}
			<div class="absolute bottom-1 right-1 w-6 h-6 rounded-full bg-amber-500 flex items-center justify-center" title="In Progress">
				<svg class="w-3.5 h-3.5 text-white" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5z" />
				</svg>
			</div>
		{:else if inLibrary}
			<div class="absolute bottom-1 right-1 w-6 h-6 rounded-full bg-green-600 flex items-center justify-center" title="In Library">
				<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{:else if requestStatus === 'approved' && downloadProgress !== undefined}
			<div class="absolute bottom-1 right-1 px-1.5 py-0.5 bg-blue-600 rounded text-[10px] font-bold text-white flex items-center gap-1" title="Downloading {downloadProgress}%">
				<svg class="w-3 h-3 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
				{downloadProgress}%
			</div>
		{:else if requestStatus === 'approved'}
			<div class="absolute bottom-1 right-1 w-6 h-6 rounded-full bg-blue-600 flex items-center justify-center" title="Downloading">
				<svg class="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
			</div>
		{:else if requested}
			<div class="absolute bottom-1 right-1 w-6 h-6 rounded-full bg-amber-500 flex items-center justify-center" title="Requested">
				<svg class="w-3.5 h-3.5 text-white" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z" />
				</svg>
			</div>
		{/if}

		<!-- Request button on hover (for discover cards) -->
		{#if onRequest && !inLibrary && !requested}
			<button
				onclick={(e) => { e.preventDefault(); e.stopPropagation(); onRequest(e); }}
				class="absolute bottom-1 right-1 w-8 h-8 rounded-full bg-white/30 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-white/40"
				title="Request"
			>
				<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
			</button>
		{/if}

		<!-- Episode progress badge - bottom left -->
		{#if computedEpisodeProgress()}
			<div class="absolute bottom-1 left-1 px-1.5 py-0.5 bg-black/80 rounded text-[10px] font-medium text-white flex items-center gap-1">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
				</svg>
				{computedEpisodeProgress()}
			</div>
		{/if}

		<!-- Progress bar (watch progress) -->
		{#if progress !== undefined && progress > 0}
			<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/50">
				<div
					class="h-full bg-outpost-500 transition-all duration-300"
					style="width: {progress}%"
				></div>
			</div>
		{/if}
	</div>

	<!-- Title and subtitle -->
	<div class="mt-2 px-0.5">
		<h3 class="text-sm font-medium text-text-primary truncate group-hover:text-outpost-400 transition-colors">
			{title}
		</h3>
		{#if subtitle}
			<p class="text-xs text-text-secondary mt-0.5 truncate">{subtitle}</p>
		{/if}
	</div>
</a>
