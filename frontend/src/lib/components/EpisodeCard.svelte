<script lang="ts">
	import { getTmdbImageUrl, getImageUrl } from '$lib/api';
	import type { EpisodeInfo, EpisodeGuestStar, EpisodeCrew } from '$lib/api/discover';

	interface Props {
		episode: EpisodeInfo;
		seasonNumber: number;
		showBackdrop?: string | null;
		// For library episodes - has local still path
		localStillPath?: string | null;
		// Library-specific features
		isLibrary?: boolean;
		isWatched?: boolean;
		episodeId?: number;
		onPlay?: () => void;
		onToggleWatched?: () => void;
		onSubtitleSearch?: () => void;
		onDelete?: () => void;
		isAdmin?: boolean;
		togglingWatched?: boolean;
		// Controlled expansion state
		isExpanded?: boolean;
		onToggleExpand?: () => void;
	}

	let {
		episode,
		seasonNumber,
		showBackdrop,
		localStillPath,
		isLibrary = false,
		isWatched = false,
		episodeId,
		onPlay,
		onToggleWatched,
		onSubtitleSearch,
		onDelete,
		isAdmin = false,
		togglingWatched = false,
		isExpanded = false,
		onToggleExpand
	}: Props = $props();

	// Use controlled state if callback provided, otherwise internal state
	let internalExpanded = $state(false);
	const expanded = $derived(onToggleExpand ? isExpanded : internalExpanded);

	// Get the image URL - prefer local path for library, otherwise TMDB
	function getEpisodeImage(): string | null {
		if (localStillPath) return getImageUrl(localStillPath);
		if (episode.still_path) return getTmdbImageUrl(episode.still_path, 'w300');
		if (showBackdrop) return isLibrary ? getImageUrl(showBackdrop) : getTmdbImageUrl(showBackdrop, 'w780');
		return null;
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function toggleExpand(e: MouseEvent) {
		// Don't expand if clicking action buttons
		const target = e.target as HTMLElement;
		if (target.closest('.action-btn')) return;

		if (onToggleExpand) {
			onToggleExpand();
		} else {
			internalExpanded = !internalExpanded;
		}
	}

	function handlePlay(e: MouseEvent) {
		e.stopPropagation();
		if (isLibrary && onPlay) {
			onPlay();
		}
	}

	// Get key crew members (director, writer)
	const directors = $derived(episode.crew?.filter(c => c.job === 'Director') || []);
	const writers = $derived(episode.crew?.filter(c => c.job === 'Writer' || c.department === 'Writing') || []);
	const hasExpandableContent = $derived(
		(episode.overview && episode.overview.length > 0) ||
		(episode.guest_stars && episode.guest_stars.length > 0) ||
		directors.length > 0 ||
		writers.length > 0
	);

	const episodeImage = $derived(getEpisodeImage());
</script>

<button
	type="button"
	onclick={toggleExpand}
	class="group flex-shrink-0 rounded-xl overflow-hidden bg-bg-elevated text-left {expanded ? 'w-96' : 'w-72'} {hasExpandableContent ? 'cursor-pointer' : ''}"
>
	<!-- Image container -->
	<div class="relative aspect-video bg-gradient-to-br from-[#1a1a2e] to-[#2d2d44]">
		{#if episodeImage}
			<img
				src={episodeImage}
				alt={episode.name}
				class="w-full h-full object-cover {!localStillPath && showBackdrop && !episode.still_path ? 'opacity-50' : ''}"
			/>
		{/if}

		<!-- Play button overlay (library only) -->
		{#if isLibrary && onPlay}
			<button
				onclick={handlePlay}
				class="action-btn absolute inset-0 bg-black/40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
				aria-label="Play episode"
			>
				<div class="w-14 h-14 rounded-full bg-cream/90 flex items-center justify-center shadow-lg">
					<svg class="w-7 h-7 text-black ml-1" fill="currentColor" viewBox="0 0 24 24">
						<path d="M8 5v14l11-7z" />
					</svg>
				</div>
			</button>
		{/if}

		<!-- Watched badge (library only) - top right -->
		{#if isLibrary && isWatched}
			<div class="absolute top-2 right-2 px-2 py-1 rounded text-[10px] font-bold uppercase bg-green-500 text-black">
				Watched
			</div>
		{/if}

		<!-- Air date badge (explore only) - top right -->
		{#if !isLibrary && episode.air_date}
			<div class="absolute top-2 right-2 px-2 py-1 rounded text-[10px] font-medium bg-black/60 text-white">
				{formatDate(episode.air_date)}
			</div>
		{/if}
	</div>

	<!-- Info section -->
	<div class="p-3">
		<div class="flex items-center gap-2 mb-1.5">
			<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
				S{seasonNumber} E{episode.episode_number}
			</span>
			{#if episode.runtime}
				<span class="text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded bg-white/10 text-text-secondary">
					{episode.runtime}m
				</span>
			{/if}
			{#if episode.vote_average > 0}
				<span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-400">
					★ {episode.vote_average.toFixed(1)}
				</span>
			{/if}
		</div>

		<div class="flex items-center justify-between gap-2">
			<h3 class="text-sm font-semibold text-text-primary truncate flex-1">
				{episode.name || `Episode ${episode.episode_number}`}
			</h3>

			<!-- Actions (visible on hover for library) -->
			{#if isLibrary}
				<div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
					{#if onToggleWatched}
						<button
							onclick={(e) => { e.stopPropagation(); onToggleWatched?.(); }}
							disabled={togglingWatched}
							class="action-btn p-1.5 rounded-full transition-colors {isWatched ? 'text-green-400 hover:bg-green-500/20' : 'text-text-muted hover:bg-white/10 hover:text-text-primary'}"
							title={isWatched ? 'Mark as unwatched' : 'Mark as watched'}
						>
							{#if togglingWatched}
								<div class="spinner-sm"></div>
							{:else if isWatched}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							{:else}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{/if}
						</button>
					{/if}

					{#if isAdmin && onSubtitleSearch}
						<button
							onclick={(e) => { e.stopPropagation(); onSubtitleSearch?.(); }}
							class="action-btn p-1.5 rounded-full text-text-muted hover:bg-purple-500/20 hover:text-purple-400 transition-colors"
							title="Search subtitles"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
							</svg>
						</button>
					{/if}

					{#if isAdmin && onDelete}
						<button
							onclick={(e) => { e.stopPropagation(); onDelete?.(); }}
							class="action-btn p-1.5 rounded-full text-text-muted hover:bg-red-500/20 hover:text-red-400 transition-colors"
							title="Delete episode"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Expand indicator -->
		{#if hasExpandableContent}
			<div class="w-full mt-2 py-1 flex items-center justify-center">
				<svg class="w-4 h-4 text-text-muted transition-transform {expanded ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</div>
		{/if}

		<!-- Expandable details -->
		{#if expanded && hasExpandableContent}
			<div class="mt-3 pt-3 border-t border-border-subtle space-y-3">
				<!-- Overview -->
				{#if episode.overview}
					<p class="text-xs text-text-secondary leading-relaxed">{episode.overview}</p>
				{/if}

				<!-- Air date (for library, show if available) -->
				{#if isLibrary && episode.air_date}
					<div class="text-xs">
						<span class="text-text-muted">Aired:</span>
						<span class="text-text-secondary ml-1">{formatDate(episode.air_date)}</span>
					</div>
				{/if}

				<!-- Crew (Director, Writers) -->
				{#if directors.length > 0 || writers.length > 0}
					<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs">
						{#if directors.length > 0}
							<div>
								<span class="text-text-muted">Director:</span>
								<span class="text-text-secondary ml-1">{directors.map(d => d.name).join(', ')}</span>
							</div>
						{/if}
						{#if writers.length > 0}
							<div>
								<span class="text-text-muted">Writer:</span>
								<span class="text-text-secondary ml-1">{writers.slice(0, 2).map(w => w.name).join(', ')}</span>
							</div>
						{/if}
					</div>
				{/if}

				<!-- Guest Stars -->
				{#if episode.guest_stars && episode.guest_stars.length > 0}
					<div>
						<h4 class="text-xs font-medium text-text-muted mb-2">Guest Stars</h4>
						<div class="flex gap-2 overflow-x-auto pb-1 scrollbar-thin">
							{#each episode.guest_stars.slice(0, 6) as guest}
								<div class="flex-shrink-0 w-16 text-center">
									{#if guest.profile_path}
										<img
											src={getTmdbImageUrl(guest.profile_path, 'w92')}
											alt={guest.name}
											class="w-12 h-12 rounded-full mx-auto object-cover bg-bg-base"
										/>
									{:else}
										<div class="w-12 h-12 rounded-full mx-auto bg-bg-base flex items-center justify-center">
											<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
											</svg>
										</div>
									{/if}
									<p class="text-[10px] text-text-primary mt-1 truncate">{guest.name}</p>
									<p class="text-[9px] text-text-muted truncate">{guest.character}</p>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</button>
