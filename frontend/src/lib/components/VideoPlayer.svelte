<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { getProgress, saveProgress, getMediaInfo, getSubtitleTracks, getSubtitleTrackUrl, type SubtitleTrack } from '$lib/api';

	interface Props {
		src: string;
		title: string;
		mediaType: 'movie' | 'episode';
		mediaId: number;
		onClose?: () => void;
		initialSubtitle?: number | null;
	}

	let { src, title, mediaType, mediaId, onClose, initialSubtitle = null }: Props = $props();

	let video: HTMLVideoElement;
	let container: HTMLDivElement;
	let playing = $state(false);
	let currentTime = $state(0);
	let duration = $state(0);
	let totalDuration = $state(0); // Actual duration from ffprobe
	let volume = $state(1);
	let muted = $state(false);
	let fullscreen = $state(false);
	let showControls = $state(true);
	let controlsTimeout: ReturnType<typeof setTimeout>;
	let progressInterval: ReturnType<typeof setInterval>;
	let seekOffset = $state(0); // Track seek offset for transcoded streams
	let currentSrc = $state(src);
	let loading = $state(true);
	// Use plain variable (not $state) to avoid reactivity issues during video reload
	let isTranscoded = false; // Set when we first need to seek (transcoded streams can't native seek)

	// Subtitle state
	let subtitleTracks = $state<SubtitleTrack[]>([]);
	let selectedSubtitle = $state<number | null>(null); // null = off, number = track index
	let showSubtitleMenu = $state(false);
	// Cache for prefetched subtitle blob URLs and original VTT text
	let subtitleBlobUrls = new Map<number, string>();
	let originalVttText = new Map<number, string>();
	// Track pending subtitle restoration to prevent duplicates
	let pendingSubtitleRestore: number | null = null;

	onMount(async () => {
		// Get actual media duration first
		try {
			const mediaInfo = await getMediaInfo(mediaType, mediaId);
			totalDuration = mediaInfo.duration;
		} catch (e) {
			console.error('Failed to get media info:', e);
		}

		// Load saved progress
		let savedPosition = 0;
		try {
			const progress = await getProgress(mediaType, mediaId);
			if (progress.position > 0 && progress.position < (totalDuration || progress.duration) - 10) {
				savedPosition = progress.position;
			}
		} catch (e) {
			// No saved progress
		}

		loading = false;

		// If we have a saved position, seek to it
		if (savedPosition > 0) {
			seekToTime(savedPosition);
		}

		// Load subtitle tracks AFTER video has started (to avoid reload issues)
		setTimeout(async () => {
			try {
				const loadedTracks = await getSubtitleTracks(mediaType, mediaId);
				console.log('Loaded subtitle tracks:', loadedTracks);
				subtitleTracks = loadedTracks;
				// Auto-select initial subtitle if specified
				if (initialSubtitle !== null && loadedTracks.some(t => t.index === initialSubtitle)) {
					selectSubtitle(initialSubtitle);
				}
			} catch (e) {
				console.error('Failed to get subtitle tracks:', e);
			}
		}, 1000);

		// Save progress periodically
		progressInterval = setInterval(() => {
			if (video && !video.paused && totalDuration > 0) {
				saveProgress({
					mediaType,
					mediaId,
					position: getActualTime(),
					duration: totalDuration
				});
			}
		}, 10000); // Save every 10 seconds

		// Handle keyboard shortcuts
		document.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		clearInterval(progressInterval);
		clearTimeout(controlsTimeout);
		document.removeEventListener('keydown', handleKeydown);

		// Clean up blob URLs for subtitles
		subtitleBlobUrls.forEach((url) => URL.revokeObjectURL(url));
		subtitleBlobUrls.clear();
		originalVttText.clear();

		// Save final progress
		if (totalDuration > 0) {
			saveProgress({
				mediaType,
				mediaId,
				position: getActualTime(),
				duration: totalDuration
			});
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		switch (e.key) {
			case ' ':
			case 'k':
				e.preventDefault();
				togglePlay();
				break;
			case 'ArrowLeft':
				e.preventDefault();
				seek(-10);
				break;
			case 'ArrowRight':
				e.preventDefault();
				seek(10);
				break;
			case 'ArrowUp':
				e.preventDefault();
				adjustVolume(0.1);
				break;
			case 'ArrowDown':
				e.preventDefault();
				adjustVolume(-0.1);
				break;
			case 'f':
				e.preventDefault();
				toggleFullscreen();
				break;
			case 'm':
				e.preventDefault();
				muted = !muted;
				break;
			case 'Escape':
				if (fullscreen) {
					document.exitFullscreen();
				} else if (onClose) {
					onClose();
				}
				break;
		}
	}

	function togglePlay() {
		if (video.paused) {
			video.play();
		} else {
			video.pause();
		}
	}

	function seek(seconds: number) {
		const dur = totalDuration || duration;
		const newTime = Math.max(0, Math.min(dur, getActualTime() + seconds));
		seekToTime(newTime);
	}

	function seekToTime(targetTime: number) {
		if (!video) return;

		console.log('seekToTime:', targetTime, 'isTranscoded:', isTranscoded);

		// Try native seeking first if not already in transcoded mode
		if (!isTranscoded && video.readyState >= 1) {
			// Check if we can actually seek to this position
			const seekable = video.seekable;
			let canSeek = false;
			for (let i = 0; i < seekable.length; i++) {
				if (targetTime >= seekable.start(i) && targetTime <= seekable.end(i)) {
					canSeek = true;
					break;
				}
			}

			if (canSeek) {
				console.log('Using native seek to:', targetTime);
				video.currentTime = targetTime;
				return;
			} else {
				// Can't native seek, switch to transcoded mode
				console.log('Cannot native seek, switching to transcoded mode');
				isTranscoded = true;
			}
		}

		// Transcoded: reload video with new start time
		seekOffset = targetTime;
		currentTime = 0;

		// Save subtitle to restore after reload (use pending flag to prevent duplicates)
		pendingSubtitleRestore = selectedSubtitle;

		// Clear current subtitle before reload to prevent duplicates
		clearAllSubtitles();

		// Build new URL with seek parameter
		const baseUrl = src.split('?')[0];
		const newSrc = `${baseUrl}?t=${Math.floor(targetTime)}`;
		console.log('Transcoded seek to:', targetTime, 'newSrc:', newSrc);

		// Directly set video source and reload
		video.pause();
		video.src = newSrc;
		currentSrc = newSrc;
		video.load();

		// Resume playback when ready and restore subtitle
		video.addEventListener('canplay', () => {
			console.log('Video ready after seek, playing...');
			video.play().catch(() => {});

			// Restore subtitle selection after seek (only if still pending)
			const subtitleToRestore = pendingSubtitleRestore;
			pendingSubtitleRestore = null;
			if (subtitleToRestore !== null) {
				selectSubtitle(subtitleToRestore);
			}
		}, { once: true });
	}

	function adjustVolume(delta: number) {
		volume = Math.max(0, Math.min(1, volume + delta));
	}

	function toggleFullscreen() {
		if (!document.fullscreenElement) {
			container.requestFullscreen();
		} else {
			document.exitFullscreen();
		}
	}

	function handleTimeUpdate() {
		currentTime = video.currentTime;
	}

	function handleLoadedMetadata() {
		duration = video.duration;
		console.log('Video loaded - duration:', video.duration, 'isTranscoded:', isTranscoded);
	}

	function handlePlay() {
		playing = true;
	}

	function handlePause() {
		playing = false;
	}

	function handleSeek(e: MouseEvent) {
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const percent = (e.clientX - rect.left) / rect.width;
		const dur = totalDuration || duration;
		const targetTime = percent * dur;
		seekToTime(targetTime);
	}

	// Get the actual current playback time including seek offset
	function getActualTime(): number {
		// Always add seekOffset - it's 0 for direct play, >0 for transcoded seek
		return seekOffset + currentTime;
	}

	// Get the total duration
	function getDuration(): number {
		return totalDuration || duration || 0;
	}

	function handleMouseMove() {
		showControls = true;
		clearTimeout(controlsTimeout);
		controlsTimeout = setTimeout(() => {
			if (playing) {
				showControls = false;
			}
		}, 3000);
	}

	function handleFullscreenChange() {
		fullscreen = !!document.fullscreenElement;
	}

	function formatTime(seconds: number): string {
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = Math.floor(seconds % 60);
		if (h > 0) {
			return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
		}
		return `${m}:${s.toString().padStart(2, '0')}`;
	}

	// Subtitle helpers
	function clearAllSubtitles() {
		if (!video) return;

		// Remove all track DOM elements
		const existingTracks = video.querySelectorAll('track');
		existingTracks.forEach(t => t.remove());

		// Disable all text tracks
		for (let i = 0; i < video.textTracks.length; i++) {
			video.textTracks[i].mode = 'disabled';
		}

		// Revoke any existing blob URLs
		subtitleBlobUrls.forEach((url) => URL.revokeObjectURL(url));
		subtitleBlobUrls.clear();
	}

	function getSubtitleLabel(track: SubtitleTrack): string {
		const parts: string[] = [];
		if (track.title) {
			parts.push(track.title);
		} else if (track.language) {
			// Convert language code to name
			const langNames: Record<string, string> = {
				eng: 'English',
				spa: 'Spanish',
				fre: 'French',
				fra: 'French',
				deu: 'German',
				ger: 'German',
				ita: 'Italian',
				por: 'Portuguese',
				jpn: 'Japanese',
				kor: 'Korean',
				zho: 'Chinese',
				chi: 'Chinese',
				rus: 'Russian',
				ara: 'Arabic',
				hin: 'Hindi',
				dut: 'Dutch',
				pol: 'Polish',
				swe: 'Swedish',
				dan: 'Danish',
				fin: 'Finnish',
				nor: 'Norwegian',
				cze: 'Czech',
				hun: 'Hungarian',
				gre: 'Greek',
				heb: 'Hebrew',
				tha: 'Thai',
				tur: 'Turkish',
				vie: 'Vietnamese',
				ind: 'Indonesian',
				und: 'Unknown'
			};
			parts.push(langNames[track.language] || track.language.toUpperCase());
		} else {
			parts.push(`Track ${track.index + 1}`);
		}
		if (track.forced) parts.push('(Forced)');
		if (track.external) parts.push('(External)');
		return parts.join(' ');
	}

	// Adjust VTT timestamps by an offset (for transcoded seek)
	function adjustVttTimestamps(vttText: string, offsetSeconds: number): string {
		if (offsetSeconds === 0) return vttText;

		// Parse VTT timestamp: 00:01:30.500 (HH:MM:SS.mmm) or 01:30.500 (MM:SS.mmm)
		function parseTimestamp(ts: string): number {
			const parts = ts.split(':');
			if (parts.length === 3) {
				// HH:MM:SS.mmm
				return parseFloat(parts[0]) * 3600 + parseFloat(parts[1]) * 60 + parseFloat(parts[2]);
			} else if (parts.length === 2) {
				// MM:SS.mmm
				return parseFloat(parts[0]) * 60 + parseFloat(parts[1]);
			}
			return 0;
		}

		// Format seconds back to VTT timestamp (always use HH:MM:SS.mmm for consistency)
		function formatTimestamp(seconds: number): string {
			if (seconds < 0) seconds = 0;
			const h = Math.floor(seconds / 3600);
			const m = Math.floor((seconds % 3600) / 60);
			const s = (seconds % 60).toFixed(3);
			return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.padStart(6, '0')}`;
		}

		// Match VTT cue timestamps - both formats:
		// HH:MM:SS.mmm --> HH:MM:SS.mmm (e.g., 01:00:01.223 --> 01:00:03.934)
		// MM:SS.mmm --> MM:SS.mmm (e.g., 00:39.498 --> 00:42.960)
		return vttText.replace(
			/(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})\s*-->\s*(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})/g,
			(match, start, end) => {
				const startSec = parseTimestamp(start.replace(',', '.')) - offsetSeconds;
				const endSec = parseTimestamp(end.replace(',', '.')) - offsetSeconds;
				// Skip cues that would have negative end time (they're in the past)
				if (endSec < 0) return match; // Will be filtered by browser anyway
				return `${formatTimestamp(Math.max(0, startSec))} --> ${formatTimestamp(endSec)}`;
			}
		);
	}

	async function selectSubtitle(trackIndex: number | null) {
		console.log('selectSubtitle called with:', trackIndex, 'seekOffset:', seekOffset, 'isTranscoded:', isTranscoded);
		selectedSubtitle = trackIndex;
		showSubtitleMenu = false;

		if (!video) {
			console.log('No video element');
			return;
		}

		// Clear all existing subtitles first
		clearAllSubtitles();

		// Add the selected subtitle track
		if (trackIndex !== null) {
			const track = subtitleTracks.find(t => t.index === trackIndex);
			if (track) {
				try {
					// Fetch original VTT if not cached
					let vttText = originalVttText.get(track.index);
					if (!vttText) {
						const subtitleUrl = getSubtitleTrackUrl(mediaType, mediaId, track.index);
						console.log('Fetching subtitle from:', subtitleUrl);
						const response = await fetch(subtitleUrl, { credentials: 'include' });
						if (!response.ok) {
							throw new Error(`Failed to fetch subtitle: ${response.status}`);
						}
						vttText = await response.text();
						originalVttText.set(track.index, vttText);
					}

					// Adjust timestamps based on current seek offset (for transcoded streams)
					const currentSeekOffset = seekOffset;
					const adjustedVtt = currentSeekOffset > 0 ? adjustVttTimestamps(vttText, currentSeekOffset) : vttText;

					// Create blob URL for the subtitle track
					const blob = new Blob([adjustedVtt], { type: 'text/vtt' });
					const blobUrl = URL.createObjectURL(blob);
					subtitleBlobUrls.set(track.index, blobUrl); // Track for cleanup

					const trackEl = document.createElement('track');
					trackEl.kind = 'subtitles';
					trackEl.src = blobUrl;
					trackEl.srclang = track.language || 'en';
					trackEl.label = getSubtitleLabel(track);

					// Handle load errors
					trackEl.addEventListener('error', (e) => {
						console.error('Failed to load subtitle track:', e);
					});

					video.appendChild(trackEl);

					// Enable the track after it's added
					if (trackEl.track) {
						trackEl.track.mode = 'showing';
					}
				} catch (e) {
					console.error('Failed to load subtitle:', e);
				}
			}
		}
	}

	function toggleSubtitleMenu() {
		showSubtitleMenu = !showSubtitleMenu;
	}

	$effect(() => {
		if (video) {
			video.volume = volume;
			video.muted = muted;
		}
	});


</script>

<svelte:document onfullscreenchange={handleFullscreenChange} />

<div
	bind:this={container}
	class="relative bg-black w-full h-full"
	onmousemove={handleMouseMove}
	role="application"
	aria-label="Video player"
>
	<!-- Video element -->
	<video
		bind:this={video}
		src={currentSrc}
		class="w-full h-full"
		ontimeupdate={handleTimeUpdate}
		onloadedmetadata={handleLoadedMetadata}
		onplay={handlePlay}
		onpause={handlePause}
		onclick={togglePlay}
		crossorigin="anonymous"
		autoplay
	>
	</video>

	<!-- Controls overlay -->
	<div
		class="absolute inset-0 flex flex-col justify-end transition-opacity duration-300 {showControls
			? 'opacity-100'
			: 'opacity-0'}"
	>
		<!-- Gradient background -->
		<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40 pointer-events-none"></div>

		<!-- Top bar -->
		<div class="absolute top-0 left-0 right-0 p-4 flex items-center justify-between">
			{#if onClose}
				<button class="liquid-btn-icon !bg-white/10 hover:!bg-white/20" onclick={onClose}>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
				</button>
			{:else}
				<div></div>
			{/if}
			<h2 class="text-white text-lg font-medium liquid-badge !bg-black/30">{title}</h2>
			<div></div>
		</div>

		<!-- Bottom controls -->
		<div class="relative p-4 space-y-3">
			<!-- Progress bar -->
			<div
				class="h-1.5 bg-white/30 rounded-full cursor-pointer group"
				onclick={handleSeek}
				role="slider"
				aria-label="Seek"
				aria-valuemin={0}
				aria-valuemax={getDuration()}
				aria-valuenow={getActualTime()}
				tabindex="0"
			>
				<div
					class="h-full bg-white rounded-full relative"
					style="width: {getDuration() ? (getActualTime() / getDuration()) * 100 : 0}%"
				>
					<div class="absolute right-0 top-1/2 -translate-y-1/2 w-3.5 h-3.5 bg-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity shadow-lg shadow-white/50"></div>
				</div>
			</div>

			<!-- Control buttons -->
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<!-- Play/Pause -->
					<button class="liquid-btn-icon !w-12 !h-12 !bg-white/10 hover:!bg-white/20" onclick={togglePlay}>
						{#if playing}
							<svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
								<path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
							</svg>
						{:else}
							<svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
								<path d="M8 5v14l11-7z" />
							</svg>
						{/if}
					</button>

					<!-- Skip buttons -->
					<button class="liquid-btn-icon !bg-white/10 hover:!bg-white/20" onclick={() => seek(-10)}>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12.066 11.2a1 1 0 000 1.6l5.334 4A1 1 0 0019 16V8a1 1 0 00-1.6-.8l-5.333 4zM4.066 11.2a1 1 0 000 1.6l5.334 4A1 1 0 0011 16V8a1 1 0 00-1.6-.8l-5.334 4z" />
						</svg>
					</button>
					<button class="liquid-btn-icon !bg-white/10 hover:!bg-white/20" onclick={() => seek(10)}>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.933 12.8a1 1 0 000-1.6L6.6 7.2A1 1 0 005 8v8a1 1 0 001.6.8l5.333-4zM19.933 12.8a1 1 0 000-1.6l-5.333-4A1 1 0 0013 8v8a1 1 0 001.6.8l5.333-4z" />
						</svg>
					</button>

					<!-- Volume -->
					<div class="flex items-center gap-2">
						<button class="liquid-btn-icon !bg-white/10 hover:!bg-white/20" onclick={() => (muted = !muted)}>
							{#if muted || volume === 0}
								<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51A8.796 8.796 0 0021 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06a8.99 8.99 0 003.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z" />
								</svg>
							{:else if volume < 0.5}
								<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M18.5 12A4.5 4.5 0 0014 7.97v8.05c2.48-.5 4.5-2.66 4.5-4.02zM5 9v6h4l5 5V4L9 9H5z" />
								</svg>
							{:else}
								<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
									<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z" />
								</svg>
							{/if}
						</button>
						<input
							type="range"
							min="0"
							max="1"
							step="0.1"
							bind:value={volume}
							class="w-20 h-1.5 bg-white/20 rounded-full appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:shadow-lg"
						/>
					</div>

					<!-- Time -->
					<span class="liquid-badge-sm !bg-black/30 text-white/80">
						{formatTime(getActualTime())} / {formatTime(getDuration())}
					</span>
				</div>

				<div class="flex items-center gap-3">
					<!-- Subtitles -->
					{#if subtitleTracks.length > 0}
						<div class="relative">
							<button
								class="liquid-btn-icon !bg-white/10 hover:!bg-white/20 {selectedSubtitle !== null ? '!bg-white/20' : ''}"
								onclick={toggleSubtitleMenu}
								title="Subtitles"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
								</svg>
							</button>

							<!-- Subtitle menu -->
							{#if showSubtitleMenu}
								<div class="absolute bottom-full right-0 mb-2 bg-black/90 rounded-lg border border-white/10 overflow-hidden min-w-[160px] shadow-xl">
									<button
										class="w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10 transition-colors flex items-center justify-between {selectedSubtitle === null ? 'bg-white/20' : ''}"
										onclick={() => selectSubtitle(null)}
									>
										<span>Off</span>
										{#if selectedSubtitle === null}
											<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
												<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
											</svg>
										{/if}
									</button>
									{#each subtitleTracks as track (track.index)}
										<button
											class="w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10 transition-colors flex items-center justify-between {selectedSubtitle === track.index ? 'bg-white/20' : ''}"
											onclick={() => selectSubtitle(track.index)}
										>
											<span>{getSubtitleLabel(track)}</span>
											{#if selectedSubtitle === track.index}
												<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
													<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
												</svg>
											{/if}
										</button>
									{/each}
								</div>
							{/if}
						</div>
					{/if}

					<!-- Fullscreen -->
					<button class="liquid-btn-icon !bg-white/10 hover:!bg-white/20" onclick={toggleFullscreen} title="Fullscreen">
						{#if fullscreen}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						{:else}
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
							</svg>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	/* Subtitle styling using ::cue pseudo-element - must be global */
	:global(video::cue) {
		background-color: rgba(0, 0, 0, 0.8);
		color: white;
		font-size: 1.3em;
		font-family: system-ui, -apple-system, sans-serif;
		line-height: 1.4;
		padding: 0.15em 0.4em;
		border-radius: 3px;
		text-shadow: 1px 1px 3px black, -1px -1px 3px black;
	}
</style>
