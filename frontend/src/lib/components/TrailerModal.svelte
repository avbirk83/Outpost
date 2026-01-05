<script lang="ts">
	/**
	 * TrailerModal - Shared trailer modal component
	 *
	 * Usage:
	 * 1. With pre-loaded trailer data (library items):
	 *    <TrailerModal bind:open trailersJson={movie.trailers} title={movie.title} />
	 *
	 * 2. With TMDB ID (discover/home items - will fetch on open):
	 *    <TrailerModal bind:open tmdbId={item.id} mediaType="movie" title={item.title} />
	 */
	import { getTrailers, type TrailerInfo } from '$lib/api';
	import { getOfficialTrailer, parseTrailers, type Trailer } from '$lib/utils';

	interface Props {
		open: boolean;
		title?: string;
		// Option 1: Pre-loaded trailer JSON string (from library items)
		trailersJson?: string;
		// Option 2: Fetch from TMDB by ID
		tmdbId?: number;
		mediaType?: 'movie' | 'tv';
	}

	let { open = $bindable(), title = '', trailersJson, tmdbId, mediaType }: Props = $props();

	let trailer: Trailer | undefined = $state(undefined);
	let loading = $state(false);
	let error: string | null = $state(null);

	// When trailersJson is provided, parse and find the best trailer
	$effect(() => {
		if (trailersJson) {
			trailer = getOfficialTrailer(trailersJson);
		}
	});

	// When modal opens and we need to fetch from TMDB
	$effect(() => {
		if (open && tmdbId && mediaType && !trailersJson) {
			fetchTrailer();
		}
	});

	async function fetchTrailer() {
		if (!tmdbId || !mediaType) return;

		loading = true;
		error = null;

		try {
			const trailers = await getTrailers(tmdbId, mediaType);
			if (trailers.length > 0) {
				// Find official trailer
				const official = trailers.find(
					t => t.type === 'Trailer' && (t.site === 'YouTube' || !t.site) && t.official !== false
				);
				trailer = official || trailers[0];
			} else {
				error = 'No trailer available';
			}
		} catch (e) {
			error = 'Failed to load trailer';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function close() {
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			close();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			close();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div
		class="modal-overlay z-[150]"
		onclick={handleBackdropClick}
		role="dialog"
		aria-modal="true"
		aria-label="Trailer for {title}"
	>
		<div class="relative w-full max-w-4xl mx-4">
			{#if loading}
				<div class="aspect-video bg-black/50 rounded-xl flex items-center justify-center">
					<div class="flex items-center gap-3 text-white">
						<div class="spinner-lg text-cream"></div>
						<span>Loading trailer...</span>
					</div>
				</div>
			{:else if error}
				<div class="aspect-video bg-black/50 rounded-xl flex items-center justify-center">
					<div class="text-center text-white">
						<svg class="w-12 h-12 mx-auto mb-2 text-white/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
						<p>{error}</p>
					</div>
				</div>
			{:else if trailer}
				<iframe
					class="w-full aspect-video rounded-xl"
					src="https://www.youtube.com/embed/{trailer.key}?autoplay=1"
					title={trailer.name || title}
					frameborder="0"
					allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
					allowfullscreen
				></iframe>
			{/if}

			<!-- Close button -->
			<button
				onclick={close}
				class="absolute -top-12 right-0 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors"
				aria-label="Close trailer"
			>
				<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	</div>
{/if}
