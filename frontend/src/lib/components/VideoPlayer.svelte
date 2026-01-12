<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getProgress,
		saveProgress,
		getMediaInfo,
		getSubtitleTracks,
		getSubtitleTrackUrl,
		getChapters,
		getSkipSegments,
		getMediaSegments,
		type SubtitleTrack,
		type MediaInfo,
		type Chapter,
		type SkipSegments,
		type MediaSegment
	} from '$lib/api';
	import { formatTime } from '$lib/utils';
	import {
		PlaybackInfoOverlay,
		NextEpisodePopup,
		PlayerProgressBar,
		PlayerVolumeControl,
		SubtitleMenu,
		AudioMenu,
		SettingsMenu,
		type PlaybackInfo
	} from './player';

	interface Props {
		src: string;
		title: string;
		subtitle?: string;
		mediaType: 'movie' | 'episode';
		mediaId: number;
		showId?: number | null;
		onClose?: () => void;
		initialSubtitle?: number | null;
		nextEpisodeData?: NextEpisode | null;
		onNextEpisode?: (episodeId: number) => void;
	}

	let { src, title, subtitle = '', mediaType, mediaId, showId = null, onClose, initialSubtitle = null, nextEpisodeData = null, onNextEpisode }: Props = $props();

	// Core video state
	let video: HTMLVideoElement;
	let container: HTMLDivElement;
	let playing = $state(false);
	let currentTime = $state(0);
	let duration = $state(0);
	let totalDuration = $state(0);
	let buffered = $state(0);
	let volume = $state(1);
	let muted = $state(false);
	let fullscreen = $state(false);
	let showControls = $state(true);
	let controlsTimeout: ReturnType<typeof setTimeout>;
	let progressInterval: ReturnType<typeof setInterval>;
	let seekOffset = $state(0);
	let currentSrc = $state(src);
	let loading = $state(true);
	let isTranscoded = false;

	// Playback settings
	let playbackSpeed = $state(1);
	let savedPlaybackSpeed = 1; // For long press restore
	type AspectRatio = 'fit' | 'fill' | '16:9' | '4:3' | '21:9';
	let aspectRatio = $state<AspectRatio>('fit');

	// UI states
	let showShortcutsOverlay = $state(false);
	let theaterMode = $state(false);
	let pipActive = $state(false);
	let showTimeRemaining = $state(false);

	// Speed boost (long press)
	let longPressTimer: ReturnType<typeof setTimeout> | null = null;
	let isSpeedBoosting = $state(false);

	// Touch gestures
	let touchGestureActive = false; // Track if gesture started on valid area (not buttons)
	let touchStartX = 0;
	let touchStartY = 0;
	let touchStartTime = 0;
	let touchStartVolume = 0;
	let lastTapTime = 0;
	let lastTapX = 0;
	let doubleTapTimeout: ReturnType<typeof setTimeout> | null = null;
	let seekIndicator = $state<{ show: boolean; amount: number; x: number }>({ show: false, amount: 0, x: 0 });
	let swipeIndicator = $state<{ show: boolean; type: 'volume' | 'brightness'; level: number }>({ show: false, type: 'volume', level: 0 });

	// Menu states
	let showSettingsMenu = $state(false);
	let showSubtitleMenu = $state(false);
	let showAudioMenu = $state(false);
	let showPlaybackInfo = $state(false);

	// Playback info
	let mediaInfoData = $state<MediaInfo | null>(null);
	let playbackInfo = $state<PlaybackInfo | null>(null);
	let playbackInfoInterval: ReturnType<typeof setInterval> | null = null;

	// Next episode
	interface NextEpisode {
		id: number;
		title: string;
		seasonNumber: number;
		episodeNumber: number;
	}
	let nextEpisode = $state<NextEpisode | null>(null);
	let showNextEpisode = $state(false);
	let nextEpisodeCountdown = $state(10);
	let nextEpisodeTimer: ReturnType<typeof setInterval> | null = null;

	$effect(() => {
		nextEpisode = nextEpisodeData;
	});

	// Audio
	interface AudioTrackInfo {
		index: number;
		language: string;
		label: string;
	}
	let audioTracks = $state<AudioTrackInfo[]>([]);
	let selectedAudioTrack = $state(0);
	let audioSync = $state(0);
	let audioContext: AudioContext | null = null;
	let audioSource: MediaElementAudioSourceNode | null = null;
	let delayNode: DelayNode | null = null;
	let audioSyncInitialized = false;

	// Subtitles
	let subtitleTracks = $state<SubtitleTrack[]>([]);
	let selectedSubtitle = $state<number | null>(null);
	let originalVttText = new Map<number, string>();
	let pendingSubtitleRestore: number | null = null;
	let subtitleOffset = $state(0);

	// Chapters
	let chapters = $state<Chapter[]>([]);
	let currentChapter = $state<Chapter | null>(null);
	let showChapterList = $state(false);

	// Skip segments (intro/credits)
	let skipSegments = $state<SkipSegments>({});
	let showSkipIntro = $state(false);
	let showSkipCredits = $state(false);
	let skipIntroTimeout: ReturnType<typeof setTimeout> | null = null;
	let inCreditsRange = $state(false);
	let autoSkipIntro = $state(false);
	let autoSkipCredits = $state(false);
	let hasAutoSkippedIntro = false; // Prevent multiple auto-skips

	// Custom subtitle rendering
	interface SubtitleCue {
		start: number;
		end: number;
		text: string;
		isSoundEffect: boolean;
		speaker: string | null;
	}
	let subtitleCues = $state<SubtitleCue[]>([]);
	let currentSubtitleText = $state<string>('');
	let currentSpeaker = $state<string | null>(null);
	let lastSubtitleTime = $state(0);
	const MIN_SPEECH_DURATION = 1.8;
	const MIN_EFFECT_DURATION = 2.5;

	// Double-click handling
	let clickTimeout: ReturnType<typeof setTimeout> | null = null;
	let clickCount = 0;

	onMount(async () => {
		// Load saved volume from localStorage
		const savedVolume = localStorage.getItem('outpost_player_volume');
		if (savedVolume !== null) {
			volume = parseFloat(savedVolume);
		}

		// Load saved time display preference
		const savedTimeDisplay = localStorage.getItem('outpost_player_time_remaining');
		if (savedTimeDisplay === 'true') {
			showTimeRemaining = true;
		}

		// Load auto-skip preferences
		autoSkipIntro = localStorage.getItem('outpost_auto_skip_intro') === 'true';
		autoSkipCredits = localStorage.getItem('outpost_auto_skip_credits') === 'true';

		// Load playback speed for episodes (per show) or reset for movies
		if (mediaType === 'episode' && showId) {
			const savedSpeed = localStorage.getItem(`outpost_playback_speed_${showId}`);
			if (savedSpeed !== null) {
				playbackSpeed = parseFloat(savedSpeed);
			}
		} else {
			playbackSpeed = 1; // Reset to 1x for movies
		}

		const [mediaInfoResult, progressResult, subtitlesResult, chaptersResult] = await Promise.allSettled([
			getMediaInfo(mediaType, mediaId),
			getProgress(mediaType, mediaId),
			getSubtitleTracks(mediaType, mediaId),
			getChapters(mediaType, mediaId)
		]);

		if (mediaInfoResult.status === 'fulfilled') {
			mediaInfoData = mediaInfoResult.value;
			totalDuration = mediaInfoResult.value.duration;
		}

		if (subtitlesResult.status === 'fulfilled') {
			subtitleTracks = subtitlesResult.value;
			if (subtitleTracks.length > 0) {
				preloadSubtitle(subtitleTracks[0].index);
			}
		}

		if (chaptersResult.status === 'fulfilled') {
			chapters = chaptersResult.value;
		}

		// Load skip segments for episodes
		console.log('VideoPlayer mediaType:', mediaType, 'mediaId:', mediaId);
		if (mediaType === 'episode') {
			try {
				// Try episode-level segments first (from chapter detection)
				console.log('Fetching segments for episode:', mediaId);
				const episodeSegments = await getMediaSegments(mediaId);
				console.log('Got segments:', episodeSegments);
				if (episodeSegments.length > 0) {
					// Convert MediaSegment[] to SkipSegments format
					skipSegments = {};
					for (const seg of episodeSegments) {
						if (seg.segmentType === 'intro') {
							skipSegments.intro = { startTime: seg.startSeconds, endTime: seg.endSeconds };
						} else if (seg.segmentType === 'credits') {
							skipSegments.credits = { startTime: seg.startSeconds, endTime: seg.endSeconds };
						}
					}
				} else if (showId) {
					// Fall back to show-level segments
					skipSegments = await getSkipSegments(showId);
				}
			} catch (e) {
				// Ignore errors - skip segments are optional
			}
		}

		let savedPosition = 0;
		if (progressResult.status === 'fulfilled') {
			const progress = progressResult.value;
			if (progress.position > 0 && progress.position < (totalDuration || progress.duration) - 10) {
				savedPosition = progress.position;
			}
		}

		loading = false;
		if (savedPosition > 0) {
			seekToTime(savedPosition);
		}

		progressInterval = setInterval(() => {
			if (video && !video.paused && totalDuration > 0) {
				saveProgress({ mediaType, mediaId, position: getActualTime(), duration: totalDuration });
			}
		}, 10000);

		document.addEventListener('keydown', handleKeydown);

		// Listen for PiP changes
		video?.addEventListener('enterpictureinpicture', () => { pipActive = true; });
		video?.addEventListener('leavepictureinpicture', () => { pipActive = false; });
	});

	onDestroy(() => {
		clearInterval(progressInterval);
		clearTimeout(controlsTimeout);
		if (clickTimeout) clearTimeout(clickTimeout);
		if (nextEpisodeTimer) clearInterval(nextEpisodeTimer);
		if (playbackInfoInterval) clearInterval(playbackInfoInterval);
		if (skipIntroTimeout) clearTimeout(skipIntroTimeout);
		if (longPressTimer) clearTimeout(longPressTimer);
		if (doubleTapTimeout) clearTimeout(doubleTapTimeout);
		document.removeEventListener('keydown', handleKeydown);
		originalVttText.clear();
		if (audioContext) {
			audioContext.close();
			audioContext = null;
		}
		if (totalDuration > 0) {
			saveProgress({ mediaType, mediaId, position: getActualTime(), duration: totalDuration });
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

		// Close shortcuts overlay on any key if open (except '?')
		if (showShortcutsOverlay && e.key !== '?') {
			showShortcutsOverlay = false;
		}

		switch (e.key) {
			case ' ': case 'k': case 'K': e.preventDefault(); togglePlay(); break;
			case 'ArrowLeft': case 'j': case 'J': e.preventDefault(); seek(-10); break;
			case 'ArrowRight': case 'l': case 'L': e.preventDefault(); seek(10); break;
			case 'ArrowUp': e.preventDefault(); setVolume(Math.min(1, volume + 0.05)); break;
			case 'ArrowDown': e.preventDefault(); setVolume(Math.max(0, volume - 0.05)); break;
			case 'f': case 'F': e.preventDefault(); toggleFullscreen(); break;
			case 'm': case 'M': e.preventDefault(); muted = !muted; break;
			case 'c': case 'C': e.preventDefault(); showChapterList = !showChapterList; break;
			case 'g': e.preventDefault(); adjustSubtitleOffset(0.5); break;
			case 'G': e.preventDefault(); adjustSubtitleOffset(-0.5); break;
			case 'h': case 'H': e.preventDefault(); subtitleOffset = 0; break;
			case 'Escape':
				if (showShortcutsOverlay) showShortcutsOverlay = false;
				else if (theaterMode) toggleTheaterMode();
				else if (fullscreen) document.exitFullscreen();
				else if (onClose) onClose();
				break;
			case 'Home': e.preventDefault(); seekToTime(0); break;
			case 'End': e.preventDefault(); seekToTime(getDuration() - 1); break;
			case '0': case '1': case '2': case '3': case '4':
			case '5': case '6': case '7': case '8': case '9':
				e.preventDefault(); seekToTime(getDuration() * (parseInt(e.key) / 10)); break;
			case 'i': case 'I': e.preventDefault(); togglePlaybackInfo(); break;
			case 'a': case 'A': e.preventDefault(); cycleAspectRatio(); break;
			case 'n': case 'N': e.preventDefault(); if (nextEpisode && onNextEpisode) playNextEpisode(); break;
			case ',': case '<': e.preventDefault(); previousChapter(); break;
			case '.': case '>': e.preventDefault(); nextChapter(); break;
			case 'p': case 'P': e.preventDefault(); togglePiP(); break;
			case 't': case 'T': e.preventDefault(); toggleTheaterMode(); break;
			case 's':
				e.preventDefault();
				if (showSkipIntro) skipIntro();
				else if (showSkipCredits) skipCredits();
				break;
			case '?': e.preventDefault(); showShortcutsOverlay = !showShortcutsOverlay; break;
		}
	}

	function togglePlay() {
		if (!video) return;
		if (video.paused) video.play(); else video.pause();
	}

	function toggleSubtitles() {
		if (selectedSubtitle !== null) selectSubtitle(null);
		else if (subtitleTracks.length > 0) selectSubtitle(subtitleTracks[0].index);
	}

	function cycleSubtitleTrack() {
		if (subtitleTracks.length === 0) return;
		if (selectedSubtitle === null) {
			selectSubtitle(subtitleTracks[0].index);
		} else {
			const idx = subtitleTracks.findIndex(t => t.index === selectedSubtitle);
			selectSubtitle(idx === subtitleTracks.length - 1 ? null : subtitleTracks[idx + 1].index);
		}
	}

	function seek(seconds: number) {
		const dur = totalDuration || duration;
		seekToTime(Math.max(0, Math.min(dur, getActualTime() + seconds)));
	}

	function seekToTime(targetTime: number) {
		if (!video) return;
		if (!isTranscoded && video.readyState >= 1) {
			const seekable = video.seekable;
			for (let i = 0; i < seekable.length; i++) {
				if (targetTime >= seekable.start(i) && targetTime <= seekable.end(i)) {
					video.currentTime = targetTime;
					return;
				}
			}
			isTranscoded = true;
		}

		seekOffset = targetTime;
		currentTime = 0;
		pendingSubtitleRestore = selectedSubtitle;
		clearSubtitles();

		const baseUrl = src.split('?')[0];
		video.pause();
		video.src = `${baseUrl}?t=${Math.floor(targetTime)}`;
		currentSrc = video.src;
		video.load();
		video.addEventListener('canplay', () => {
			video.play().catch(() => {});
			if (pendingSubtitleRestore !== null) {
				selectSubtitle(pendingSubtitleRestore);
				pendingSubtitleRestore = null;
			}
		}, { once: true });
	}

	function toggleFullscreen() {
		if (!container) return;
		if (!document.fullscreenElement) container.requestFullscreen();
		else document.exitFullscreen();
	}

	// Volume with localStorage persistence
	function setVolume(newVolume: number) {
		volume = newVolume;
		localStorage.setItem('outpost_player_volume', String(newVolume));
	}

	// Theater mode
	function toggleTheaterMode() {
		theaterMode = !theaterMode;
		// Dispatch event for parent to handle
		container?.dispatchEvent(new CustomEvent('theatermode', { detail: theaterMode, bubbles: true }));
	}

	// Picture-in-Picture
	async function togglePiP() {
		if (!video) return;
		try {
			if (document.pictureInPictureElement) {
				await document.exitPictureInPicture();
			} else if (document.pictureInPictureEnabled) {
				await video.requestPictureInPicture();
			}
		} catch (e) {
			console.error('PiP failed:', e);
		}
	}

	// Time display toggle
	function toggleTimeDisplay() {
		showTimeRemaining = !showTimeRemaining;
		localStorage.setItem('outpost_player_time_remaining', String(showTimeRemaining));
	}

	// Touch gesture handlers
	function handleTouchStart(e: TouchEvent) {
		if (e.touches.length !== 1) return;

		// Ignore touches on interactive elements
		const target = e.target as HTMLElement;
		if (target.closest('button') || target.closest('.player-controls') || target.closest('.player-top-bar') ||
			target.closest('.settings-menu') || target.closest('.subtitle-menu') || target.closest('.audio-menu') ||
			target.closest('.chapter-list-panel') || target.closest('.shortcuts-overlay')) {
			touchGestureActive = false;
			return;
		}

		// Mark gesture as active for this touch sequence
		touchGestureActive = true;

		const touch = e.touches[0];
		touchStartX = touch.clientX;
		touchStartY = touch.clientY;
		touchStartTime = Date.now();
		touchStartVolume = volume;

		// Long press detection
		longPressTimer = setTimeout(() => {
			if (!isSpeedBoosting) {
				savedPlaybackSpeed = playbackSpeed;
				playbackSpeed = 2;
				if (video) video.playbackRate = 2;
				isSpeedBoosting = true;
			}
		}, 500);
	}

	function handleTouchMove(e: TouchEvent) {
		// Only process if gesture started on valid area (not buttons)
		if (!touchGestureActive || e.touches.length !== 1 || !container) return;

		const touch = e.touches[0];
		const rect = container.getBoundingClientRect();
		const deltaX = touch.clientX - touchStartX;
		const deltaY = touch.clientY - touchStartY;

		// Cancel long press if moved significantly
		if (Math.abs(deltaX) > 10 || Math.abs(deltaY) > 10) {
			if (longPressTimer) {
				clearTimeout(longPressTimer);
				longPressTimer = null;
			}
		}

		// Vertical swipe for volume (right half of screen)
		const isRightHalf = touchStartX > rect.width / 2;
		if (isRightHalf && Math.abs(deltaY) > 20 && Math.abs(deltaY) > Math.abs(deltaX)) {
			e.preventDefault();
			const volumeDelta = -deltaY / rect.height;
			const newVolume = Math.max(0, Math.min(1, touchStartVolume + volumeDelta));
			setVolume(newVolume);
			swipeIndicator = { show: true, type: 'volume', level: newVolume };
		}
	}

	function handleTouchEnd(e: TouchEvent) {
		// Clear long press timer
		if (longPressTimer) {
			clearTimeout(longPressTimer);
			longPressTimer = null;
		}

		// Restore speed if was boosting
		if (isSpeedBoosting) {
			playbackSpeed = savedPlaybackSpeed;
			if (video) video.playbackRate = savedPlaybackSpeed;
			isSpeedBoosting = false;
		}

		// Hide swipe indicator
		if (swipeIndicator.show) {
			setTimeout(() => { swipeIndicator = { ...swipeIndicator, show: false }; }, 300);
		}

		// If gesture wasn't active (touch started on button), don't process
		if (!touchGestureActive) {
			return;
		}

		// Reset gesture flag
		touchGestureActive = false;

		// Double-tap detection
		const now = Date.now();
		const touch = e.changedTouches[0];
		if (!touch || !container) return;

		const rect = container.getBoundingClientRect();
		const x = touch.clientX - rect.left;
		const third = rect.width / 3;

		// Check if it's a quick tap (not a swipe)
		const deltaX = touch.clientX - touchStartX;
		const deltaY = touch.clientY - touchStartY;
		const tapDuration = now - touchStartTime;

		if (tapDuration < 300 && Math.abs(deltaX) < 20 && Math.abs(deltaY) < 20) {
			// Check for double tap
			if (now - lastTapTime < 300 && Math.abs(x - lastTapX) < 50) {
				// Double tap detected
				e.preventDefault();
				if (doubleTapTimeout) clearTimeout(doubleTapTimeout);

				if (x < third) {
					// Left third - seek back
					seek(-10);
					showSeekIndicator(-10, x);
				} else if (x > third * 2) {
					// Right third - seek forward
					seek(10);
					showSeekIndicator(10, x + 40);
				} else {
					// Center - toggle fullscreen
					toggleFullscreen();
				}
				lastTapTime = 0;
			} else {
				// First tap - wait for potential double tap
				lastTapTime = now;
				lastTapX = x;
				doubleTapTimeout = setTimeout(() => {
					// Single tap - toggle play
					togglePlay();
				}, 300);
			}
		}
	}

	function showSeekIndicator(amount: number, x: number) {
		seekIndicator = { show: true, amount, x };
		setTimeout(() => { seekIndicator = { ...seekIndicator, show: false }; }, 600);
	}

	function getActualTime(): number { return seekOffset + currentTime; }
	function getDuration(): number { return totalDuration || duration || 0; }

	function getEndTime(): string {
		const remaining = getDuration() - getActualTime();
		if (remaining <= 0 || !isFinite(remaining)) return '';
		const end = new Date(Date.now() + remaining * 1000);
		return end.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function handleTimeUpdate() {
		if (!video) return;
		currentTime = video.currentTime;
		try {
			if (video.buffered && video.buffered.length > 0) {
				buffered = video.buffered.end(video.buffered.length - 1);
			}
		} catch (e) {
			// Ignore buffered access errors
		}
		updateCurrentSubtitle();
		updateCurrentChapter();
		updateSkipSegments();
		if (nextEpisode && mediaType === 'episode') {
			const remaining = getDuration() - getActualTime();
			if (remaining > 0 && remaining <= 30 && !showNextEpisode && !inCreditsRange) startNextEpisodeCountdown();
			else if (remaining > 30 && showNextEpisode) clearNextEpisodeTimer();
		}
	}

	function updateSkipSegments() {
		const time = getActualTime();

		// Check intro segment
		if (skipSegments.intro) {
			const inIntro = time >= skipSegments.intro.startTime && time < skipSegments.intro.endTime;
			if (inIntro && !showSkipIntro) {
				// Auto-skip if enabled and hasn't already skipped this intro
				if (autoSkipIntro && !hasAutoSkippedIntro) {
					hasAutoSkippedIntro = true;
					skipIntro();
					return;
				}
				showSkipIntro = true;
				// Auto-hide after 5 seconds
				if (skipIntroTimeout) clearTimeout(skipIntroTimeout);
				skipIntroTimeout = setTimeout(() => {
					showSkipIntro = false;
				}, 5000);
			} else if (!inIntro && showSkipIntro) {
				showSkipIntro = false;
				if (skipIntroTimeout) {
					clearTimeout(skipIntroTimeout);
					skipIntroTimeout = null;
				}
			}
		}

		// Check credits segment
		if (skipSegments.credits) {
			const wasInCredits = inCreditsRange;
			inCreditsRange = time >= skipSegments.credits.startTime && time < skipSegments.credits.endTime;

			if (inCreditsRange && !wasInCredits) {
				// Entered credits range
				if (nextEpisode && onNextEpisode) {
					// Start next episode countdown (or auto-skip to it)
					startNextEpisodeCountdown();
				} else if (autoSkipCredits) {
					// Auto-skip credits if enabled and no next episode
					skipCredits();
				} else {
					// Show skip credits button
					showSkipCredits = true;
				}
			} else if (!inCreditsRange && wasInCredits) {
				// Left credits range
				showSkipCredits = false;
				if (!nextEpisode) {
					clearNextEpisodeTimer();
				}
			}
		}
	}

	function skipIntro() {
		if (skipSegments.intro) {
			seekToTime(skipSegments.intro.endTime);
			showSkipIntro = false;
		}
	}

	function skipCredits() {
		if (skipSegments.credits) {
			seekToTime(skipSegments.credits.endTime);
			showSkipCredits = false;
		}
	}

	function handleLoadedMetadata() {
		if (!video) return;
		duration = video.duration;
		video.playbackRate = playbackSpeed;
		detectAudioTracks();
	}

	function handlePlay() {
		playing = true;
		if (audioContext?.state === 'suspended') audioContext.resume();
	}

	function handlePause() { playing = false; }

	function handleMouseMove() {
		showControls = true;
		clearTimeout(controlsTimeout);
		controlsTimeout = setTimeout(() => {
			if (playing) {
				showControls = false;
				showSubtitleMenu = false;
				showSettingsMenu = false;
				showAudioMenu = false;
			}
		}, 3000);
	}

	function handleVideoClick(e: MouseEvent) {
		clickCount++;
		if (clickTimeout) clearTimeout(clickTimeout);
		clickTimeout = setTimeout(() => {
			if (clickCount === 1) togglePlay();
			clickCount = 0;
		}, 250);

		if (clickCount === 2) {
			clearTimeout(clickTimeout);
			clickCount = 0;
			const rect = container.getBoundingClientRect();
			const x = e.clientX - rect.left;
			const third = rect.width / 3;
			if (x < third) seek(-10);
			else if (x > third * 2) seek(10);
			else toggleFullscreen();
		}
	}

	function handleContainerClick(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.settings-menu')) showSettingsMenu = false;
		if (!target.closest('.subtitle-menu')) showSubtitleMenu = false;
		if (!target.closest('.audio-menu')) showAudioMenu = false;
		if (target.closest('.player-overlay') && !target.closest('.player-controls') && !target.closest('.player-top-bar') && !target.closest('button')) {
			handleVideoClick(e);
		}
	}

	// Aspect ratio
	function cycleAspectRatio() {
		const options: AspectRatio[] = ['fit', 'fill', '16:9', '4:3', '21:9'];
		aspectRatio = options[(options.indexOf(aspectRatio) + 1) % options.length];
	}

	function getAspectRatioStyle(): string {
		switch (aspectRatio) {
			case 'fill': return 'object-fit: cover;';
			case '16:9': return 'object-fit: contain; aspect-ratio: 16/9;';
			case '4:3': return 'object-fit: contain; aspect-ratio: 4/3;';
			case '21:9': return 'object-fit: contain; aspect-ratio: 21/9;';
			default: return 'object-fit: contain;';
		}
	}

	// Playback info
	function togglePlaybackInfo() {
		showPlaybackInfo = !showPlaybackInfo;
		if (showPlaybackInfo) {
			updatePlaybackInfo();
			playbackInfoInterval = setInterval(updatePlaybackInfo, 1000);
		} else if (playbackInfoInterval) {
			clearInterval(playbackInfoInterval);
			playbackInfoInterval = null;
		}
	}

	function updatePlaybackInfo() {
		if (!video) return;
		let container = mediaInfoData?.container || 'Unknown';
		if (container === 'Unknown') {
			const filename = src.split('?')[0].split('/').pop() || '';
			if (filename.includes('.')) container = filename.split('.').pop()?.toUpperCase() || 'Unknown';
		}

		let droppedFrames = 0, totalFrames = 0;
		const quality = (video as any).getVideoPlaybackQuality?.();
		if (quality) { droppedFrames = quality.droppedVideoFrames || 0; totalFrames = quality.totalVideoFrames || 0; }

		let videoCodec = 'N/A', audioCodec = 'N/A', audioChannels = '', resolution = video.videoWidth && video.videoHeight ? `${video.videoWidth}×${video.videoHeight}` : 'Unknown', bitrate = 'N/A';

		if (mediaInfoData) {
			const vs = mediaInfoData.videoStreams?.[0];
			if (vs) {
				videoCodec = vs.codec?.toUpperCase() || 'N/A';
				if (vs.profile) videoCodec += ` (${vs.profile})`;
				resolution = `${vs.width}×${vs.height}`;
				if (vs.frameRate) resolution += ` @ ${vs.frameRate}`;
			}
			const as = mediaInfoData.audioStreams?.[0];
			if (as) { audioCodec = as.codec?.toUpperCase() || 'N/A'; audioChannels = as.channelLayout || `${as.channels}ch`; }
			if (mediaInfoData.bitRate) bitrate = `${(mediaInfoData.bitRate / 1000000).toFixed(1)} Mbps`;
		}

		playbackInfo = { playMethod: isTranscoded ? 'Transcode' : 'Direct Play', container, resolution, videoCodec, audioCodec, audioChannels, droppedFrames, totalFrames, bitrate };
	}

	// Next episode
	function playNextEpisode() {
		if (nextEpisode && onNextEpisode) { clearNextEpisodeTimer(); onNextEpisode(nextEpisode.id); }
	}

	function clearNextEpisodeTimer() {
		if (nextEpisodeTimer) { clearInterval(nextEpisodeTimer); nextEpisodeTimer = null; }
		showNextEpisode = false;
		nextEpisodeCountdown = 10;
	}

	function startNextEpisodeCountdown() {
		if (!nextEpisode || nextEpisodeTimer) return;
		showNextEpisode = true;
		nextEpisodeCountdown = 10;
		nextEpisodeTimer = setInterval(() => {
			nextEpisodeCountdown--;
			if (nextEpisodeCountdown <= 0) playNextEpisode();
		}, 1000);
	}

	// Audio tracks
	function detectAudioTracks() {
		if (!video) return;
		const tracks: AudioTrackInfo[] = [];
		const vat = (video as any).audioTracks;
		if (vat?.length > 0) {
			for (let i = 0; i < vat.length; i++) {
				tracks.push({ index: i, language: vat[i].language || 'Unknown', label: vat[i].label || `Track ${i + 1}` });
			}
		}
		audioTracks = tracks;
	}

	function selectAudioTrack(index: number) {
		if (!video) return;
		const vat = (video as any).audioTracks;
		if (vat) { for (let i = 0; i < vat.length; i++) vat[i].enabled = (i === index); }
		selectedAudioTrack = index;
	}

	// Audio sync
	function initAudioSync() {
		if (audioSyncInitialized || !video) return;
		try {
			audioContext = new AudioContext();
			audioSource = audioContext.createMediaElementSource(video);
			delayNode = audioContext.createDelay(5);
			delayNode.delayTime.value = 0;
			audioSource.connect(delayNode);
			delayNode.connect(audioContext.destination);
			audioSyncInitialized = true;
		} catch (e) { console.error('Audio sync setup failed:', e); }
	}

	function adjustAudioSync(delta: number) {
		if (!audioSyncInitialized) initAudioSync();
		audioSync = Math.max(-0.5, Math.min(3, audioSync + delta));
		if (delayNode) delayNode.delayTime.value = Math.max(0, audioSync);
	}

	function resetAudioSync() { audioSync = 0; if (delayNode) delayNode.delayTime.value = 0; }

	// Subtitles
	function clearSubtitles() { subtitleCues = []; currentSubtitleText = ''; currentSpeaker = null; lastSubtitleTime = 0; }

	function adjustSubtitleOffset(delta: number) { subtitleOffset += delta; updateCurrentSubtitle(); }

	function updateCurrentSubtitle() {
		if (subtitleCues.length === 0) { currentSubtitleText = ''; currentSpeaker = null; return; }
		const time = getActualTime() + subtitleOffset;
		const cue = subtitleCues.find(c => time >= c.start && time <= c.end);

		if (cue) {
			if (currentSubtitleText !== cue.text) lastSubtitleTime = time;
			currentSubtitleText = cue.text;
			currentSpeaker = cue.speaker;
		} else {
			const prevIdx = subtitleCues.findIndex((c, i) => time > c.end && (!subtitleCues[i + 1] || time < subtitleCues[i + 1].start));
			if (prevIdx >= 0 && currentSubtitleText) {
				const prev = subtitleCues[prevIdx];
				const minDur = prev.isSoundEffect ? MIN_EFFECT_DURATION : MIN_SPEECH_DURATION;
				if (time - lastSubtitleTime < minDur) return;
			}
			currentSubtitleText = '';
			currentSpeaker = null;
		}
	}

	async function preloadSubtitle(trackIndex: number) {
		if (originalVttText.has(trackIndex)) return;
		try {
			const url = getSubtitleTrackUrl(mediaType, mediaId, trackIndex);
			const res = await fetch(url, { credentials: 'include' });
			if (res.ok) originalVttText.set(trackIndex, await res.text());
		} catch (e) { console.error('Failed to preload subtitle:', e); }
	}

	async function selectSubtitle(trackIndex: number | null) {
		selectedSubtitle = trackIndex;
		showSubtitleMenu = false;
		clearSubtitles();

		if (trackIndex !== null) {
			const track = subtitleTracks.find(t => t.index === trackIndex);
			if (track) {
				let vtt = originalVttText.get(track.index);
				if (!vtt) {
					const url = getSubtitleTrackUrl(mediaType, mediaId, track.index);
					const res = await fetch(url, { credentials: 'include' });
					if (!res.ok) return;
					vtt = await res.text();
					originalVttText.set(track.index, vtt);
				}
				subtitleCues = parseVtt(vtt);
				updateCurrentSubtitle();
			}
		}
	}

	function parseVtt(vtt: string): SubtitleCue[] {
		const cues: SubtitleCue[] = [];
		const lines = vtt.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
		let i = 0;
		while (i < lines.length && !lines[i].includes('-->')) i++;

		while (i < lines.length) {
			const line = lines[i].trim();
			if (line.includes('-->')) {
				const match = line.match(/(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})\s*-->\s*(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})/);
				if (match) {
					const start = parseTimestamp(match[1]), end = parseTimestamp(match[2]);
					const textLines: string[] = [];
					i++;
					while (i < lines.length && lines[i].trim() !== '' && !lines[i].includes('-->')) {
						if (!/^\d+$/.test(lines[i].trim())) textLines.push(lines[i].trim());
						i++;
					}
					if (textLines.length > 0) {
						const raw = textLines.join('\n').replace(/<[^>]+>/g, '');
						const { cleanText, speaker, isSoundEffect } = detectSpeakerAndEffects(raw);
						cues.push({ start, end, text: cleanText, speaker, isSoundEffect });
					}
				} else i++;
			} else i++;
		}
		return cues;
	}

	function parseTimestamp(ts: string): number {
		const parts = ts.replace(',', '.').split(':');
		if (parts.length === 3) return parseFloat(parts[0]) * 3600 + parseFloat(parts[1]) * 60 + parseFloat(parts[2]);
		return parseFloat(parts[0]) * 60 + parseFloat(parts[1]);
	}

	function detectSpeakerAndEffects(text: string): { cleanText: string; speaker: string | null; isSoundEffect: boolean } {
		let cleanText = text, speaker: string | null = null, isSoundEffect = false;
		const trimmed = text.trim();
		if (/^\[.+\]$/.test(trimmed) || /^\(.+\)$/.test(trimmed) || /^\*.+\*$/.test(trimmed) || trimmed.startsWith('[') || trimmed.startsWith('(') || trimmed.startsWith('♪')) {
			isSoundEffect = true;
		}
		const speakerMatch = cleanText.match(/^-?\s*([A-Z][A-Z\s]{1,20}):\s*/) || cleanText.match(/^-?\s*([A-Z][a-z]+(?:\s[A-Z][a-z]+)?):\s*/);
		if (speakerMatch) { speaker = speakerMatch[1].trim(); cleanText = cleanText.replace(speakerMatch[0], '').trim(); }
		else if (cleanText.startsWith('- ')) cleanText = cleanText.substring(2).trim();
		return { cleanText, speaker, isSoundEffect };
	}

	// Chapters
	function updateCurrentChapter() {
		if (chapters.length === 0) { currentChapter = null; return; }
		const time = getActualTime();
		const chapter = chapters.find(c => time >= c.startTime && time < c.endTime);
		currentChapter = chapter || null;
	}

	function nextChapter() {
		if (chapters.length === 0) return;
		const time = getActualTime();
		const next = chapters.find(c => c.startTime > time + 0.5);
		if (next) seekToTime(next.startTime);
	}

	function previousChapter() {
		if (chapters.length === 0) return;
		const time = getActualTime();
		// If more than 3 seconds into current chapter, go to start of current
		// Otherwise, go to previous chapter
		const current = chapters.find(c => time >= c.startTime && time < c.endTime);
		if (current && time - current.startTime > 3) {
			seekToTime(current.startTime);
		} else {
			const prev = [...chapters].reverse().find(c => c.startTime < time - 0.5);
			if (prev) seekToTime(prev.startTime);
		}
	}

	function seekToChapter(chapter: Chapter) {
		seekToTime(chapter.startTime);
		showChapterList = false;
	}

	// Effects
	$effect(() => { if (video) { video.volume = volume; video.muted = muted; } });

	let initialSubtitleApplied = false;
	$effect(() => {
		if (video && subtitleTracks.length > 0 && initialSubtitle !== null && !initialSubtitleApplied) {
			if (subtitleTracks.some(t => t.index === initialSubtitle)) {
				initialSubtitleApplied = true;
				selectSubtitle(initialSubtitle);
			}
		}
	});
</script>

<svelte:document onfullscreenchange={() => fullscreen = !!document.fullscreenElement} />

<div bind:this={container} class="player-container {theaterMode ? 'theater-mode' : ''}" onmousemove={handleMouseMove} onclick={handleContainerClick} ontouchstart={handleTouchStart} ontouchmove={handleTouchMove} ontouchend={handleTouchEnd} role="application" aria-label="Video player">
	{#if loading}
		<div class="player-loading"><div class="loading-spinner"></div></div>
	{/if}

	<video bind:this={video} src={currentSrc} class="video-element" style={getAspectRatioStyle()}
		ontimeupdate={handleTimeUpdate} onloadedmetadata={handleLoadedMetadata}
		onplay={handlePlay} onpause={handlePause} onclick={handleVideoClick}
		crossorigin="anonymous" autoplay>
	</video>

	<!-- Keyboard shortcuts overlay -->
	{#if showShortcutsOverlay}
		<div class="shortcuts-overlay" onclick={() => showShortcutsOverlay = false} role="dialog" aria-label="Keyboard shortcuts">
			<div class="shortcuts-panel" onclick={(e) => e.stopPropagation()}>
				<div class="shortcuts-header">
					<h3>Keyboard Shortcuts</h3>
					<button class="shortcuts-close" onclick={() => showShortcutsOverlay = false}>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
				<div class="shortcuts-columns">
					<div class="shortcuts-column">
						<h4>Playback</h4>
						<div class="shortcut-item"><span class="shortcut-key">Space / K</span><span>Play / Pause</span></div>
						<div class="shortcut-item"><span class="shortcut-key">← / J</span><span>Seek -10s</span></div>
						<div class="shortcut-item"><span class="shortcut-key">→ / L</span><span>Seek +10s</span></div>
						<div class="shortcut-item"><span class="shortcut-key">↑ / ↓</span><span>Volume</span></div>
						<div class="shortcut-item"><span class="shortcut-key">M</span><span>Mute</span></div>
						<div class="shortcut-item"><span class="shortcut-key">0-9</span><span>Seek to %</span></div>
						<div class="shortcut-item"><span class="shortcut-key">S</span><span>Skip intro/credits</span></div>
					</div>
					<div class="shortcuts-column">
						<h4>Navigation</h4>
						<div class="shortcut-item"><span class="shortcut-key">F</span><span>Fullscreen</span></div>
						<div class="shortcut-item"><span class="shortcut-key">T</span><span>Theater mode</span></div>
						<div class="shortcut-item"><span class="shortcut-key">P</span><span>Picture-in-Picture</span></div>
						<div class="shortcut-item"><span class="shortcut-key">, / .</span><span>Prev/Next chapter</span></div>
						<div class="shortcut-item"><span class="shortcut-key">C</span><span>Chapter list</span></div>
						<div class="shortcut-item"><span class="shortcut-key">I</span><span>Playback info</span></div>
						<div class="shortcut-item"><span class="shortcut-key">?</span><span>Show/hide this</span></div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Speed boost indicator -->
	{#if isSpeedBoosting}
		<div class="speed-boost-indicator">2×</div>
	{/if}

	<!-- Double-tap seek indicator -->
	{#if seekIndicator.show}
		<div class="seek-indicator" style="left: {seekIndicator.x}px">
			<div class="seek-indicator-ripple"></div>
			<span>{seekIndicator.amount > 0 ? '+' : ''}{seekIndicator.amount}s</span>
		</div>
	{/if}

	<!-- Volume swipe indicator -->
	{#if swipeIndicator.show}
		<div class="swipe-indicator">
			<svg class="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
				{#if swipeIndicator.type === 'volume'}
					<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
				{/if}
			</svg>
			<div class="swipe-level-bar">
				<div class="swipe-level-fill" style="height: {swipeIndicator.level * 100}%"></div>
			</div>
		</div>
	{/if}

	{#if showPlaybackInfo && playbackInfo}
		<PlaybackInfoOverlay info={playbackInfo} />
	{/if}

	{#if showNextEpisode && nextEpisode}
		<NextEpisodePopup
			seasonNumber={nextEpisode.seasonNumber}
			episodeNumber={nextEpisode.episodeNumber}
			title={nextEpisode.title}
			countdown={nextEpisodeCountdown}
			onCancel={clearNextEpisodeTimer}
			onPlay={playNextEpisode}
		/>
	{/if}

	{#if currentSubtitleText}
		<div class="subtitle-overlay">
			<div class="subtitle-content">
				{#if currentSpeaker}<span class="subtitle-speaker">{currentSpeaker}</span>{/if}
				<div class="subtitle-text">{@html currentSubtitleText.replace(/\n/g, '<br>')}</div>
			</div>
		</div>
	{/if}

	{#if showSkipIntro}
		<button class="skip-button" onclick={skipIntro}>
			Skip Intro
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
			</svg>
		</button>
	{/if}

	{#if showSkipCredits && !nextEpisode}
		<button class="skip-button" onclick={skipCredits}>
			Skip Credits
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
			</svg>
		</button>
	{/if}

	{#if showChapterList && chapters && chapters.length > 0}
		<div class="chapter-list-panel">
			<div class="chapter-list-header">
				<h3>Chapters</h3>
				<button class="chapter-list-close" onclick={() => showChapterList = false} aria-label="Close chapters">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
			<div class="chapter-list-items">
				{#each chapters as chapter}
					<button
						class="chapter-item {currentChapter?.index === chapter.index ? 'active' : ''}"
						onclick={() => seekToChapter(chapter)}
					>
						<span class="chapter-time">{formatTime(chapter.startTime)}</span>
						<span class="chapter-name">{chapter.title || `Chapter ${chapter.index + 1}`}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<div class="player-overlay {showControls ? 'visible' : ''}">
		<div class="player-top-gradient"></div>
		<div class="player-top-bar">
			{#if onClose}
				<button class="player-btn" onclick={onClose} aria-label="Go back">
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
				</button>
			{:else}<div class="w-10"></div>{/if}
			<div class="player-title-container">
				<h2 class="player-title">{title}</h2>
				{#if subtitle}<p class="player-subtitle-text">{subtitle}</p>{/if}
			</div>
			<div class="w-10"></div>
		</div>

		<div class="player-bottom-gradient"></div>
		<div class="player-controls">
			{#if currentChapter}
				<div class="current-chapter-display">
					<span class="chapter-label">Chapter:</span>
					<span class="chapter-title">{currentChapter.title}</span>
				</div>
			{/if}
			<PlayerProgressBar
				currentTime={getActualTime()}
				duration={getDuration()}
				{buffered}
				{chapters}
				endTime={getEndTime()}
				showRemaining={showTimeRemaining}
				onSeek={seekToTime}
				onToggleTimeDisplay={toggleTimeDisplay}
			/>

			<div class="controls-row">
				<div class="controls-left">
					<button class="player-btn" onclick={() => seek(-10)} aria-label="Skip back 10s">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M12.5 3C17.15 3 21.08 6.03 22.47 10.22L20.1 11C19.05 7.81 16.04 5.5 12.5 5.5C10.54 5.5 8.77 6.22 7.38 7.38L10 10H3V3L5.6 5.6C7.45 4 9.85 3 12.5 3M10 12V22H8V14H6V12H10M18 14V20C18 21.11 17.11 22 16 22H14C12.9 22 12 21.1 12 20V14C12 12.9 12.9 12 14 12H16C17.11 12 18 12.9 18 14M14 14V20H16V14H14Z"/></svg>
					</button>
					<button class="player-btn player-btn-play" onclick={togglePlay} aria-label={playing ? 'Pause' : 'Play'}>
						{#if playing}
							<svg class="w-7 h-7" fill="currentColor" viewBox="0 0 24 24"><path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" /></svg>
						{:else}
							<svg class="w-7 h-7" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z" /></svg>
						{/if}
					</button>
					<button class="player-btn" onclick={() => seek(10)} aria-label="Skip forward 10s">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M11.5 3C6.85 3 2.92 6.03 1.53 10.22L3.9 11C4.95 7.81 7.96 5.5 11.5 5.5C13.46 5.5 15.23 6.22 16.62 7.38L14 10H21V3L18.4 5.6C16.55 4 14.15 3 11.5 3M10 12V22H8V14H6V12H10M18 14V20C18 21.11 17.11 22 16 22H14C12.9 22 12 21.1 12 20V14C12 12.9 12.9 12 14 12H16C17.11 12 18 12.9 18 14M14 14V20H16V14H14Z"/></svg>
					</button>
					<PlayerVolumeControl {volume} {muted} onVolumeChange={(v) => { setVolume(v); muted = false; }} onMuteToggle={() => muted = !muted} />
					{#if chapters && chapters.length > 0}
						<div class="chapter-nav-divider"></div>
						<button class="player-btn" onclick={previousChapter} aria-label="Previous chapter" title="Previous Chapter (,)">
							<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M6 6h2v12H6V6zm3.5 6l8.5 6V6l-8.5 6z"/></svg>
						</button>
						<button class="player-btn {showChapterList ? 'active' : ''}" onclick={() => { showChapterList = !showChapterList; }} aria-label="Chapters" title="Chapters (P)">
							<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M3 13h2v-2H3v2zm0 4h2v-2H3v2zm0-8h2V7H3v2zm4 4h14v-2H7v2zm0 4h14v-2H7v2zM7 7v2h14V7H7z"/></svg>
						</button>
						<button class="player-btn" onclick={nextChapter} aria-label="Next chapter" title="Next Chapter (.)">
							<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M6 18l8.5-6L6 6v12zm2-8.14L11.03 12 8 14.14V9.86zM16 6h2v12h-2V6z"/></svg>
						</button>
					{/if}
				</div>

				<div class="controls-right">
					<button class="player-btn {showPlaybackInfo ? 'active' : ''}" onclick={togglePlaybackInfo} aria-label="Playback info" title="Playback Info (I)">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
					</button>

					<SubtitleMenu
						tracks={subtitleTracks}
						selectedIndex={selectedSubtitle}
						offset={subtitleOffset}
						open={showSubtitleMenu}
						onToggle={() => { showSubtitleMenu = !showSubtitleMenu; showSettingsMenu = false; showAudioMenu = false; }}
						onSelect={selectSubtitle}
						onOffsetChange={adjustSubtitleOffset}
						onOffsetReset={() => { subtitleOffset = 0; updateCurrentSubtitle(); }}
					/>

					<AudioMenu
						tracks={audioTracks}
						selectedIndex={selectedAudioTrack}
						{audioSync}
						open={showAudioMenu}
						onToggle={() => { showAudioMenu = !showAudioMenu; showSubtitleMenu = false; showSettingsMenu = false; }}
						onTrackSelect={selectAudioTrack}
						onSyncChange={adjustAudioSync}
						onSyncReset={resetAudioSync}
					/>

					<SettingsMenu
						open={showSettingsMenu}
						{playbackSpeed}
						{aspectRatio}
						{autoSkipIntro}
						{autoSkipCredits}
						onToggle={() => { showSettingsMenu = !showSettingsMenu; showSubtitleMenu = false; showAudioMenu = false; }}
						onSpeedChange={(s) => {
							playbackSpeed = s;
							if (video) video.playbackRate = s;
							// Save speed per show for episodes
							if (mediaType === 'episode' && showId) {
								localStorage.setItem(`outpost_playback_speed_${showId}`, String(s));
							}
						}}
						onAspectChange={(r) => aspectRatio = r}
						onAutoSkipIntroChange={(enabled) => {
							autoSkipIntro = enabled;
							localStorage.setItem('outpost_auto_skip_intro', String(enabled));
						}}
						onAutoSkipCreditsChange={(enabled) => {
							autoSkipCredits = enabled;
							localStorage.setItem('outpost_auto_skip_credits', String(enabled));
						}}
					/>

					<!-- Speed indicator badge -->
					{#if playbackSpeed !== 1}
						<div class="speed-badge">{playbackSpeed}×</div>
					{/if}

					<!-- Picture-in-Picture button -->
					<button class="player-btn {pipActive ? 'active' : ''}" onclick={togglePiP} aria-label="Picture-in-Picture" title="Picture-in-Picture (P)">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							{#if pipActive}
								<path d="M19 7h-8v6h8V7zm2-4H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14z"/>
							{:else}
								<path d="M19 7h-8v6h8V7zm4-4H1v17h22V3zm-2 15H3V5h18v13z"/>
							{/if}
						</svg>
					</button>

					<!-- Theater mode button -->
					<button class="player-btn {theaterMode ? 'active' : ''}" onclick={toggleTheaterMode} aria-label="Theater mode" title="Theater Mode (T)">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							{#if theaterMode}
								<path d="M19 6H5c-1.1 0-2 .9-2 2v8c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 10H5V8h14v8z"/>
							{:else}
								<path d="M4 6h16v2H4zm0 5h16v2H4zm0 5h16v2H4z"/>
							{/if}
						</svg>
					</button>

					<button class="player-btn" onclick={toggleFullscreen} aria-label={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}>
						{#if fullscreen}
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/></svg>
						{:else}
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/></svg>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.player-container { position: relative; width: 100%; height: 100%; background: #0a0a0a; overflow: hidden; }
	.player-loading { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: rgba(10, 10, 10, 0.6); z-index: 50; }
	.loading-spinner { width: 48px; height: 48px; border: 3px solid rgba(245, 230, 200, 0.2); border-top-color: #E8A849; border-radius: 50%; animation: spin 1s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	.video-element { width: 100%; height: 100%; background: #000; }

	.player-overlay { position: absolute; inset: 0; display: flex; flex-direction: column; justify-content: space-between; opacity: 0; transition: opacity 0.3s; pointer-events: none; }
	.player-overlay.visible { opacity: 1; pointer-events: auto; }

	.player-top-gradient { position: absolute; top: 0; left: 0; right: 0; height: 120px; background: linear-gradient(to bottom, rgba(10, 10, 10, 0.8), transparent); pointer-events: none; }
	.player-bottom-gradient { position: absolute; bottom: 0; left: 0; right: 0; height: 200px; background: linear-gradient(to top, rgba(10, 10, 10, 0.95), transparent); pointer-events: none; }

	.player-top-bar { position: relative; z-index: 10; display: flex; align-items: center; justify-content: space-between; padding: 20px 24px; }
	.player-title-container { text-align: center; }
	.player-title { color: #F5E6C8; font-size: 18px; font-weight: 500; }
	.player-subtitle-text { color: rgba(245, 230, 200, 0.7); font-size: 14px; }

	.player-controls { position: relative; z-index: 10; padding: 0 24px 20px; display: flex; flex-direction: column; gap: 12px; }
	.controls-row { display: flex; align-items: center; justify-content: space-between; }
	.controls-left, .controls-right { display: flex; align-items: center; gap: 4px; }

	.player-btn { background: none; border: none; color: #F5E6C8; padding: 10px; cursor: pointer; border-radius: 50%; transition: all 0.2s; display: flex; align-items: center; justify-content: center; }
	.player-btn:hover { color: #E8A849; background: rgba(255, 255, 255, 0.06); }
	.player-btn.active { color: #E8A849; }
	.player-btn-play { padding: 12px; }

	.subtitle-overlay { position: absolute; bottom: 100px; left: 0; right: 0; display: flex; justify-content: center; pointer-events: none; z-index: 30; padding: 0 40px; }
	.subtitle-content { display: flex; flex-direction: column; align-items: center; gap: 4px; max-width: 85%; }
	.subtitle-speaker { color: #E8A849; font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif; font-size: 16px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px; text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.95), -1px -1px 2px rgba(0, 0, 0, 0.95), 0 0 4px rgba(0, 0, 0, 0.7); }
	.subtitle-text { color: #F5E6C8; font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif; font-size: 28px; font-weight: 600; line-height: 1.4; text-align: center; text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.95), -1px -1px 2px rgba(0, 0, 0, 0.95), 1px -1px 2px rgba(0, 0, 0, 0.95), -1px 1px 2px rgba(0, 0, 0, 0.95), 0 0 6px rgba(0, 0, 0, 0.7); }

	/* Skip button styles */
	.skip-button {
		position: absolute;
		right: 24px;
		bottom: 140px;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 20px;
		background: rgba(232, 168, 73, 0.9);
		color: #0a0a0a;
		border: none;
		border-radius: 24px;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		z-index: 35;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	}
	.skip-button:hover {
		background: #E8A849;
		transform: scale(1.05);
	}
	.skip-button svg {
		width: 16px;
		height: 16px;
	}

	/* Chapter styles */
	.current-chapter-display { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
	.chapter-label { color: rgba(245, 230, 200, 0.5); font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
	.chapter-title { color: #E8A849; font-size: 13px; font-weight: 500; }

	.chapter-nav-divider { width: 1px; height: 20px; background: rgba(245, 230, 200, 0.2); margin: 0 4px; }

	.chapter-list-panel { position: absolute; right: 24px; bottom: 120px; width: 320px; max-height: 400px; background: rgba(10, 10, 10, 0.95); backdrop-filter: blur(12px); border: 1px solid rgba(245, 230, 200, 0.15); border-radius: 12px; z-index: 40; display: flex; flex-direction: column; overflow: hidden; }
	.chapter-list-header { display: flex; align-items: center; justify-content: space-between; padding: 16px; border-bottom: 1px solid rgba(245, 230, 200, 0.1); }
	.chapter-list-header h3 { color: #F5E6C8; font-size: 14px; font-weight: 600; margin: 0; }
	.chapter-list-close { background: none; border: none; color: rgba(245, 230, 200, 0.6); cursor: pointer; padding: 4px; border-radius: 4px; display: flex; align-items: center; justify-content: center; }
	.chapter-list-close:hover { color: #F5E6C8; background: rgba(255, 255, 255, 0.1); }
	.chapter-list-items { overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 2px; }
	.chapter-item { display: flex; align-items: center; gap: 12px; padding: 10px 12px; background: none; border: none; border-radius: 8px; cursor: pointer; text-align: left; width: 100%; transition: background 0.2s; }
	.chapter-item:hover { background: rgba(255, 255, 255, 0.08); }
	.chapter-item.active { background: rgba(232, 168, 73, 0.15); }
	.chapter-item.active .chapter-time { color: #E8A849; }
	.chapter-time { color: rgba(245, 230, 200, 0.5); font-size: 12px; font-variant-numeric: tabular-nums; min-width: 50px; }
	.chapter-name { color: #F5E6C8; font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

	/* Keyboard shortcuts overlay */
	.shortcuts-overlay {
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0.8);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 60;
		backdrop-filter: blur(4px);
	}

	.shortcuts-panel {
		background: rgba(10, 10, 10, 0.95);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 16px;
		padding: 24px;
		max-width: 600px;
		width: 90%;
	}

	.shortcuts-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 20px;
		padding-bottom: 12px;
		border-bottom: 1px solid rgba(245, 230, 200, 0.1);
	}

	.shortcuts-header h3 {
		color: #F5E6C8;
		font-size: 18px;
		font-weight: 600;
		margin: 0;
	}

	.shortcuts-close {
		background: none;
		border: none;
		color: rgba(245, 230, 200, 0.6);
		cursor: pointer;
		padding: 6px;
		border-radius: 6px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.shortcuts-close:hover {
		color: #F5E6C8;
		background: rgba(255, 255, 255, 0.1);
	}

	.shortcuts-columns {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 32px;
	}

	.shortcuts-column h4 {
		color: #E8A849;
		font-size: 13px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin: 0 0 12px 0;
	}

	.shortcut-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 0;
		color: rgba(245, 230, 200, 0.8);
		font-size: 13px;
	}

	.shortcut-key {
		background: rgba(245, 230, 200, 0.1);
		color: #F5E6C8;
		padding: 4px 8px;
		border-radius: 4px;
		font-family: monospace;
		font-size: 12px;
		font-weight: 500;
	}

	/* Speed badge */
	.speed-badge {
		background: rgba(232, 168, 73, 0.2);
		color: #E8A849;
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
	}

	/* Speed boost indicator */
	.speed-boost-indicator {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		background: rgba(232, 168, 73, 0.9);
		color: #0a0a0a;
		padding: 16px 32px;
		border-radius: 12px;
		font-size: 32px;
		font-weight: 700;
		z-index: 45;
		pointer-events: none;
		animation: pulse 0.5s ease-in-out infinite alternate;
	}

	@keyframes pulse {
		from { transform: translate(-50%, -50%) scale(1); }
		to { transform: translate(-50%, -50%) scale(1.05); }
	}

	/* Seek indicator (double-tap) */
	.seek-indicator {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		color: #F5E6C8;
		font-size: 24px;
		font-weight: 600;
		z-index: 45;
		pointer-events: none;
	}

	.seek-indicator-ripple {
		position: absolute;
		width: 80px;
		height: 80px;
		border-radius: 50%;
		background: rgba(245, 230, 200, 0.3);
		animation: ripple 0.6s ease-out forwards;
	}

	@keyframes ripple {
		from {
			transform: scale(0.5);
			opacity: 1;
		}
		to {
			transform: scale(1.5);
			opacity: 0;
		}
	}

	/* Swipe indicator (volume/brightness) */
	.swipe-indicator {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		color: #F5E6C8;
		z-index: 45;
		pointer-events: none;
	}

	.swipe-level-bar {
		width: 6px;
		height: 100px;
		background: rgba(245, 230, 200, 0.2);
		border-radius: 3px;
		overflow: hidden;
		position: relative;
	}

	.swipe-level-fill {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		background: #E8A849;
		border-radius: 3px;
		transition: height 0.1s;
	}

	/* Theater mode */
	.player-container.theater-mode {
		position: fixed;
		inset: 0;
		z-index: 9999;
		background: #000;
	}

	@media (max-width: 768px) {
		.subtitle-overlay { bottom: 80px; padding: 0 16px; }
		.subtitle-text { font-size: 18px; }
		.player-top-bar { padding: 12px 16px; }
		.player-controls { padding: 0 16px 16px; }
		.player-title { font-size: 14px; }
	}

	@media (min-width: 1920px) { .subtitle-text { font-size: 36px; } }
</style>
