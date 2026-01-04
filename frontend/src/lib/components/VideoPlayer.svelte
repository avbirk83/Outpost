<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		getProgress,
		saveProgress,
		getMediaInfo,
		getSubtitleTracks,
		getSubtitleTrackUrl,
		type SubtitleTrack,
		type MediaInfo
	} from '$lib/api';
	import { formatTime } from '$lib/utils';

	interface Props {
		src: string;
		title: string;
		subtitle?: string;
		mediaType: 'movie' | 'episode';
		mediaId: number;
		onClose?: () => void;
		initialSubtitle?: number | null;
		nextEpisodeData?: NextEpisode | null;
		onNextEpisode?: (episodeId: number) => void;
	}

	let { src, title, subtitle = '', mediaType, mediaId, onClose, initialSubtitle = null, nextEpisodeData = null, onNextEpisode }: Props = $props();

	// Set next episode from props
	$effect(() => {
		nextEpisode = nextEpisodeData;
	});

	let video: HTMLVideoElement;
	let container: HTMLDivElement;
	let progressBar: HTMLDivElement;
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

	// Hover states
	let hoverTime = $state<number | null>(null);
	let hoverX = $state(0);
	let showVolumeSlider = $state(false);

	// Playback speed
	let playbackSpeed = $state(1);
	let showSettingsMenu = $state(false);
	const speedOptions = [0.5, 0.75, 1, 1.25, 1.5, 2];

	// Aspect ratio
	type AspectRatio = 'fit' | 'fill' | '16:9' | '4:3' | '21:9';
	let aspectRatio = $state<AspectRatio>('fit');
	const aspectRatioOptions: { value: AspectRatio; label: string }[] = [
		{ value: 'fit', label: 'Fit' },
		{ value: 'fill', label: 'Fill' },
		{ value: '16:9', label: '16:9' },
		{ value: '4:3', label: '4:3' },
		{ value: '21:9', label: '21:9 (Ultrawide)' }
	];

	// Playback info overlay
	let showPlaybackInfo = $state(false);
	let mediaInfoData = $state<MediaInfo | null>(null);
	interface PlaybackInfo {
		playMethod: string;
		container: string;
		resolution: string;
		videoCodec: string;
		audioCodec: string;
		audioChannels: string;
		droppedFrames: number;
		totalFrames: number;
		bitrate: string;
	}
	let playbackInfo = $state<PlaybackInfo | null>(null);
	let playbackInfoInterval: ReturnType<typeof setInterval> | null = null;

	// Next episode (for TV)
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

	// Audio sync (in seconds, positive = delay audio)
	let audioSync = $state(0);
	let audioContext: AudioContext | null = null;
	let audioSource: MediaElementAudioSourceNode | null = null;
	let delayNode: DelayNode | null = null;
	let audioSyncInitialized = false;

	// Audio track selection
	interface AudioTrackInfo {
		index: number;
		language: string;
		label: string;
	}
	let audioTracks = $state<AudioTrackInfo[]>([]);
	let selectedAudioTrack = $state(0);
	let showAudioMenu = $state(false);

	// Subtitle state
	let subtitleTracks = $state<SubtitleTrack[]>([]);
	let selectedSubtitle = $state<number | null>(null);
	let showSubtitleMenu = $state(false);
	let originalVttText = new Map<number, string>();
	let pendingSubtitleRestore: number | null = null;

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
	let subtitleOffset = $state(0); // No default offset - adjust per video as needed
	let lastSubtitleTime = $state(0); // Track when subtitle was shown

	// Minimum display durations
	const MIN_SPEECH_DURATION = 1.8; // seconds
	const MIN_EFFECT_DURATION = 2.5; // seconds


	// Double-click handling
	let clickTimeout: ReturnType<typeof setTimeout> | null = null;
	let clickCount = 0;

	onMount(async () => {
		// Load media info, progress, and subtitles in parallel
		const [mediaInfoResult, progressResult, subtitlesResult] = await Promise.allSettled([
			getMediaInfo(mediaType, mediaId),
			getProgress(mediaType, mediaId),
			getSubtitleTracks(mediaType, mediaId)
		]);

		// Handle media info
		if (mediaInfoResult.status === 'fulfilled') {
			mediaInfoData = mediaInfoResult.value;
			totalDuration = mediaInfoResult.value.duration;
		} else {
			console.error('Failed to get media info:', mediaInfoResult.reason);
		}

		// Handle subtitles
		if (subtitlesResult.status === 'fulfilled') {
			subtitleTracks = subtitlesResult.value;
			// Pre-load first subtitle track in background
			if (subtitleTracks.length > 0) {
				preloadSubtitle(subtitleTracks[0].index);
			}
		}

		// Handle saved progress
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
				saveProgress({
					mediaType,
					mediaId,
					position: getActualTime(),
					duration: totalDuration
				});
			}
		}, 10000);

		document.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		clearInterval(progressInterval);
		clearTimeout(controlsTimeout);
		if (clickTimeout) clearTimeout(clickTimeout);
		if (nextEpisodeTimer) clearInterval(nextEpisodeTimer);
		if (playbackInfoInterval) clearInterval(playbackInfoInterval);
		document.removeEventListener('keydown', handleKeydown);

		originalVttText.clear();

		// Cleanup audio sync
		if (audioContext) {
			audioContext.close();
			audioContext = null;
		}

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
		// Ignore if typing in an input
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

		switch (e.key) {
			case ' ':
			case 'k':
			case 'K':
				e.preventDefault();
				togglePlay();
				break;
			case 'ArrowLeft':
			case 'j':
			case 'J':
				e.preventDefault();
				seek(-10);
				break;
			case 'ArrowRight':
			case 'l':
			case 'L':
				e.preventDefault();
				seek(10);
				break;
			case 'ArrowUp':
				e.preventDefault();
				adjustVolume(0.05);
				break;
			case 'ArrowDown':
				e.preventDefault();
				adjustVolume(-0.05);
				break;
			case 'f':
			case 'F':
				e.preventDefault();
				toggleFullscreen();
				break;
			case 'm':
			case 'M':
				e.preventDefault();
				muted = !muted;
				break;
			case 'c':
			case 'C':
				e.preventDefault();
				toggleSubtitles();
				break;
			case 's':
			case 'S':
				e.preventDefault();
				cycleSubtitleTrack();
				break;
			case 'g':
				e.preventDefault();
				adjustSubtitleOffset(0.5); // Subtitles earlier (if audio ahead of subs)
				break;
			case 'G':
				e.preventDefault();
				adjustSubtitleOffset(-0.5); // Subtitles later (if audio behind subs)
				break;
			case 'h':
			case 'H':
				e.preventDefault();
				subtitleOffset = 0; // Reset offset
				break;
			case 'Escape':
				if (fullscreen) {
					document.exitFullscreen();
				} else if (onClose) {
					onClose();
				}
				break;
			case 'Home':
				e.preventDefault();
				seekToTime(0);
				break;
			case 'End':
				e.preventDefault();
				seekToTime(getDuration() - 1);
				break;
			case '0': case '1': case '2': case '3': case '4':
			case '5': case '6': case '7': case '8': case '9':
				e.preventDefault();
				const percent = parseInt(e.key) / 10;
				seekToTime(getDuration() * percent);
				break;
			case 'i':
			case 'I':
				e.preventDefault();
				togglePlaybackInfo();
				break;
			case 'a':
			case 'A':
				e.preventDefault();
				cycleAspectRatio();
				break;
			case 'n':
			case 'N':
				e.preventDefault();
				if (nextEpisode && onNextEpisode) {
					playNextEpisode();
				}
				break;
		}
	}

	function toggleSubtitles() {
		if (selectedSubtitle !== null) {
			selectSubtitle(null);
		} else if (subtitleTracks.length > 0) {
			selectSubtitle(subtitleTracks[0].index);
		}
	}

	function cycleSubtitleTrack() {
		if (subtitleTracks.length === 0) return;

		if (selectedSubtitle === null) {
			selectSubtitle(subtitleTracks[0].index);
		} else {
			const currentIdx = subtitleTracks.findIndex(t => t.index === selectedSubtitle);
			if (currentIdx === subtitleTracks.length - 1) {
				selectSubtitle(null);
			} else {
				selectSubtitle(subtitleTracks[currentIdx + 1].index);
			}
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

		if (!isTranscoded && video.readyState >= 1) {
			const seekable = video.seekable;
			let canSeek = false;
			for (let i = 0; i < seekable.length; i++) {
				if (targetTime >= seekable.start(i) && targetTime <= seekable.end(i)) {
					canSeek = true;
					break;
				}
			}

			if (canSeek) {
				video.currentTime = targetTime;
				return;
			} else {
				isTranscoded = true;
			}
		}

		seekOffset = targetTime;
		currentTime = 0;
		pendingSubtitleRestore = selectedSubtitle;
		clearSubtitles();

		const baseUrl = src.split('?')[0];
		const newSrc = `${baseUrl}?t=${Math.floor(targetTime)}`;

		video.pause();
		video.src = newSrc;
		currentSrc = newSrc;
		video.load();

		video.addEventListener('canplay', () => {
			video.play().catch(() => {});
			const subtitleToRestore = pendingSubtitleRestore;
			pendingSubtitleRestore = null;
			if (subtitleToRestore !== null) {
				selectSubtitle(subtitleToRestore);
			}
		}, { once: true });
	}

	function adjustVolume(delta: number) {
		volume = Math.max(0, Math.min(1, volume + delta));
		muted = false;
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
		// Update buffered
		if (video.buffered.length > 0) {
			buffered = video.buffered.end(video.buffered.length - 1);
		}
		// Update current subtitle
		updateCurrentSubtitle();

		// Check for next episode (show countdown in last 30 seconds)
		if (nextEpisode && mediaType === 'episode') {
			const remaining = getDuration() - getActualTime();
			if (remaining > 0 && remaining <= 30 && !showNextEpisode) {
				startNextEpisodeCountdown();
			} else if (remaining > 30 && showNextEpisode) {
				clearNextEpisodeTimer();
			}
		}
	}

	function updateCurrentSubtitle() {
		if (subtitleCues.length === 0) {
			currentSubtitleText = '';
			currentSpeaker = null;
			return;
		}

		const time = getActualTime() + subtitleOffset;

		// Find the cue that should be showing
		const cueIndex = subtitleCues.findIndex(c => time >= c.start && time <= c.end);
		const cue = cueIndex >= 0 ? subtitleCues[cueIndex] : null;

		// Debug: log occasionally
		if (Math.floor(time * 2) % 10 === 0) {
			console.log('Subtitle update:', { time, cueIndex, cue: cue?.text?.substring(0, 30), first: subtitleCues[0] });
		}

		if (cue) {
			// We have an active cue
			if (currentSubtitleText !== cue.text) {
				lastSubtitleTime = time;
			}
			currentSubtitleText = cue.text;
			currentSpeaker = cue.speaker;
		} else {
			// No active cue - check if we should keep showing the last one
			const prevCueIndex = subtitleCues.findIndex((c, i) => {
				const next = subtitleCues[i + 1];
				return time > c.end && (!next || time < next.start);
			});

			if (prevCueIndex >= 0 && currentSubtitleText) {
				const prevCue = subtitleCues[prevCueIndex];
				const minDuration = prevCue.isSoundEffect ? MIN_EFFECT_DURATION : MIN_SPEECH_DURATION;
				const elapsed = time - lastSubtitleTime;

				// Keep showing if minimum duration hasn't passed
				if (elapsed < minDuration) {
					return; // Keep current subtitle
				}
			}

			currentSubtitleText = '';
			currentSpeaker = null;
		}
	}

	function adjustSubtitleOffset(delta: number) {
		subtitleOffset += delta;
		updateCurrentSubtitle();
	}

	function handleLoadedMetadata() {
		duration = video.duration;
		video.playbackRate = playbackSpeed;
		detectAudioTracks();
	}

	function initAudioSync() {
		if (audioSyncInitialized || !video) return;

		try {
			audioContext = new AudioContext();
			audioSource = audioContext.createMediaElementSource(video);
			delayNode = audioContext.createDelay(5); // Max 5 second delay
			delayNode.delayTime.value = 0; // Start with no delay

			// Connect: video -> delay -> speakers
			audioSource.connect(delayNode);
			delayNode.connect(audioContext.destination);
			audioSyncInitialized = true;
		} catch (e) {
			console.error('Failed to setup audio sync:', e);
		}
	}

	function adjustAudioSync(delta: number) {
		// Initialize audio routing on first adjustment
		if (!audioSyncInitialized) {
			initAudioSync();
		}

		// Allow -500ms to +3000ms (negative requires video delay which we fake with smaller audio delay)
		audioSync = Math.max(-0.5, Math.min(3, audioSync + delta));
		applyAudioSync();
	}

	function setAudioSync(value: number) {
		if (!audioSyncInitialized) {
			initAudioSync();
		}
		audioSync = Math.max(-0.5, Math.min(3, value));
		applyAudioSync();
	}

	function applyAudioSync() {
		if (!delayNode) return;
		// For positive values: delay audio (audio plays later)
		// For negative values: we can't make audio earlier, but we set delay to 0
		const delay = Math.max(0, audioSync);
		delayNode.delayTime.value = delay;
	}

	function detectAudioTracks() {
		if (!video) return;

		const tracks: AudioTrackInfo[] = [];

		// Try to get audio tracks from the video element (browser support varies)
		const videoAudioTracks = (video as any).audioTracks;
		if (videoAudioTracks && videoAudioTracks.length > 0) {
			for (let i = 0; i < videoAudioTracks.length; i++) {
				const track = videoAudioTracks[i];
				tracks.push({
					index: i,
					language: track.language || 'Unknown',
					label: track.label || `Track ${i + 1}`
				});
			}
		}

		audioTracks = tracks;
	}

	function selectAudioTrack(index: number) {
		if (!video) return;

		const videoAudioTracks = (video as any).audioTracks;
		if (videoAudioTracks) {
			for (let i = 0; i < videoAudioTracks.length; i++) {
				videoAudioTracks[i].enabled = (i === index);
			}
		}

		selectedAudioTrack = index;
	}

	function getAudioTrackLabel(track: AudioTrackInfo): string {
		const langNames: Record<string, string> = {
			'eng': 'English', 'en': 'English',
			'spa': 'Spanish', 'es': 'Spanish',
			'fra': 'French', 'fr': 'French',
			'deu': 'German', 'de': 'German',
			'ita': 'Italian', 'it': 'Italian',
			'jpn': 'Japanese', 'ja': 'Japanese',
			'kor': 'Korean', 'ko': 'Korean',
			'por': 'Portuguese', 'pt': 'Portuguese',
			'rus': 'Russian', 'ru': 'Russian',
			'zho': 'Chinese', 'zh': 'Chinese',
			'und': 'Unknown'
		};

		if (track.label && track.label !== `Track ${track.index + 1}`) {
			return track.label;
		}

		return langNames[track.language] || track.language || `Track ${track.index + 1}`;
	}

	function handlePlay() {
		playing = true;
		// Resume audio context if suspended (browser autoplay policy)
		if (audioContext?.state === 'suspended') {
			audioContext.resume();
		}
	}

	function handlePause() {
		playing = false;
	}

	function handleVideoClick(e: MouseEvent) {
		clickCount++;

		if (clickTimeout) {
			clearTimeout(clickTimeout);
		}

		clickTimeout = setTimeout(() => {
			if (clickCount === 1) {
				// Single click - toggle play
				togglePlay();
			}
			clickCount = 0;
		}, 250);

		if (clickCount === 2) {
			// Double click - handle based on position
			clearTimeout(clickTimeout);
			clickCount = 0;

			const rect = container.getBoundingClientRect();
			const x = e.clientX - rect.left;
			const third = rect.width / 3;

			if (x < third) {
				// Left third - seek back
				seek(-10);
			} else if (x > third * 2) {
				// Right third - seek forward
				seek(10);
			} else {
				// Center - fullscreen
				toggleFullscreen();
			}
		}
	}

	function handleProgressClick(e: MouseEvent) {
		const rect = progressBar.getBoundingClientRect();
		const percent = (e.clientX - rect.left) / rect.width;
		const dur = totalDuration || duration;
		const targetTime = percent * dur;
		seekToTime(targetTime);
	}

	function handleProgressHover(e: MouseEvent) {
		if (!progressBar) return;
		const rect = progressBar.getBoundingClientRect();
		const percent = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
		const dur = totalDuration || duration;
		hoverTime = percent * dur;
		hoverX = e.clientX - rect.left;
	}

	function handleProgressLeave() {
		hoverTime = null;
	}

	function getActualTime(): number {
		return seekOffset + currentTime;
	}

	function getDuration(): number {
		return totalDuration || duration || 0;
	}

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

	function handleFullscreenChange() {
		fullscreen = !!document.fullscreenElement;
	}

	function setPlaybackSpeed(speed: number) {
		playbackSpeed = speed;
		if (video) {
			video.playbackRate = speed;
		}
		showSettingsMenu = false;
	}

	function setAspectRatio(ratio: AspectRatio) {
		aspectRatio = ratio;
	}

	function cycleAspectRatio() {
		const currentIndex = aspectRatioOptions.findIndex(o => o.value === aspectRatio);
		const nextIndex = (currentIndex + 1) % aspectRatioOptions.length;
		aspectRatio = aspectRatioOptions[nextIndex].value;
	}

	function getAspectRatioStyle(): string {
		switch (aspectRatio) {
			case 'fill':
				return 'object-fit: cover;';
			case '16:9':
				return 'object-fit: contain; aspect-ratio: 16/9;';
			case '4:3':
				return 'object-fit: contain; aspect-ratio: 4/3;';
			case '21:9':
				return 'object-fit: contain; aspect-ratio: 21/9;';
			default:
				return 'object-fit: contain;';
		}
	}

	function togglePlaybackInfo() {
		showPlaybackInfo = !showPlaybackInfo;
		if (showPlaybackInfo) {
			updatePlaybackInfo();
			// Update dropped frames periodically
			playbackInfoInterval = setInterval(updatePlaybackInfo, 1000);
		} else {
			if (playbackInfoInterval) {
				clearInterval(playbackInfoInterval);
				playbackInfoInterval = null;
			}
		}
	}

	function updatePlaybackInfo() {
		if (!video) return;

		// Get container from media info or fallback to URL extension
		let container = mediaInfoData?.container || 'Unknown';
		if (container === 'Unknown') {
			const urlPath = src.split('?')[0];
			const filename = urlPath.split('/').pop() || '';
			if (filename.includes('.')) {
				container = filename.split('.').pop()?.toUpperCase() || 'Unknown';
			}
		}

		// Get playback quality (dropped frames)
		let droppedFrames = 0;
		let totalFrames = 0;
		const quality = (video as any).getVideoPlaybackQuality?.();
		if (quality) {
			droppedFrames = quality.droppedVideoFrames || 0;
			totalFrames = quality.totalVideoFrames || 0;
		}

		// Determine play method
		const playMethod = isTranscoded ? 'Transcode' : 'Direct Play';

		// Get codec info from media info
		let videoCodec = 'N/A';
		let audioCodec = 'N/A';
		let audioChannels = '';
		let resolution = video.videoWidth && video.videoHeight
			? `${video.videoWidth}×${video.videoHeight}`
			: 'Unknown';
		let bitrate = 'N/A';

		if (mediaInfoData) {
			// Video stream info
			const videoStream = mediaInfoData.videoStreams?.[0];
			if (videoStream) {
				videoCodec = videoStream.codec?.toUpperCase() || 'N/A';
				if (videoStream.profile) {
					videoCodec += ` (${videoStream.profile})`;
				}
				resolution = `${videoStream.width}×${videoStream.height}`;
				if (videoStream.frameRate) {
					resolution += ` @ ${videoStream.frameRate}`;
				}
			}

			// Audio stream info
			const audioStream = mediaInfoData.audioStreams?.[0];
			if (audioStream) {
				audioCodec = audioStream.codec?.toUpperCase() || 'N/A';
				audioChannels = audioStream.channelLayout || `${audioStream.channels}ch`;
			}

			// Bitrate
			if (mediaInfoData.bitRate) {
				const mbps = (mediaInfoData.bitRate / 1000000).toFixed(1);
				bitrate = `${mbps} Mbps`;
			}
		}

		playbackInfo = {
			playMethod,
			container,
			resolution,
			videoCodec,
			audioCodec,
			audioChannels,
			droppedFrames,
			totalFrames,
			bitrate
		};
	}

	function getEndTime(): string {
		const remaining = getDuration() - getActualTime();
		if (remaining <= 0 || !isFinite(remaining)) return '';

		const now = new Date();
		const endTime = new Date(now.getTime() + remaining * 1000);
		return endTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function playNextEpisode() {
		if (nextEpisode && onNextEpisode) {
			clearNextEpisodeTimer();
			onNextEpisode(nextEpisode.id);
		}
	}

	function clearNextEpisodeTimer() {
		if (nextEpisodeTimer) {
			clearInterval(nextEpisodeTimer);
			nextEpisodeTimer = null;
		}
		showNextEpisode = false;
		nextEpisodeCountdown = 10;
	}

	function startNextEpisodeCountdown() {
		if (!nextEpisode || nextEpisodeTimer) return;

		showNextEpisode = true;
		nextEpisodeCountdown = 10;

		nextEpisodeTimer = setInterval(() => {
			nextEpisodeCountdown--;
			if (nextEpisodeCountdown <= 0) {
				playNextEpisode();
			}
		}, 1000);
	}

	// Subtitle helpers
	function clearSubtitles() {
		subtitleCues = [];
		currentSubtitleText = '';
		currentSpeaker = null;
		lastSubtitleTime = 0;
	}

	function detectSpeakerAndEffects(text: string): { cleanText: string; speaker: string | null; isSoundEffect: boolean } {
		let cleanText = text;
		let speaker: string | null = null;
		let isSoundEffect = false;

		// Detect sound effects: [text], (text), *text*, ♪text♪
		const effectPatterns = [
			/^\[.+\]$/,           // [door slams]
			/^\(.+\)$/,           // (gunshot)
			/^\*.+\*$/,           // *sighs*
			/^♪.+♪$/,             // ♪music♪
			/^\[.+\]$/m,          // Multi-line with brackets
		];

		const trimmed = text.trim();
		if (effectPatterns.some(p => p.test(trimmed)) ||
			trimmed.startsWith('[') || trimmed.startsWith('(') || trimmed.startsWith('♪')) {
			isSoundEffect = true;
		}

		// Detect speaker patterns: "NAME:", "NAME:", "<i>NAME:</i>", "- NAME:"
		const speakerPatterns = [
			/^-?\s*([A-Z][A-Z\s]{1,20}):\s*/,           // JOHN: or - JOHN:
			/^-?\s*([A-Z][a-z]+(?:\s[A-Z][a-z]+)?):\s*/, // John: or John Smith:
			/^<i>([^<]+):<\/i>\s*/,                      // <i>John:</i>
		];

		for (const pattern of speakerPatterns) {
			const match = cleanText.match(pattern);
			if (match) {
				speaker = match[1].trim();
				cleanText = cleanText.replace(pattern, '').trim();
				break;
			}
		}

		// Also check for "- " at start (dialogue indicator)
		if (!speaker && cleanText.startsWith('- ')) {
			cleanText = cleanText.substring(2).trim();
		}

		return { cleanText, speaker, isSoundEffect };
	}

	function parseVtt(vttText: string): SubtitleCue[] {
		const cues: SubtitleCue[] = [];
		// Normalize line endings and split
		const lines = vttText.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
		let i = 0;

		// Skip WEBVTT header and any metadata
		while (i < lines.length && !lines[i].includes('-->')) {
			i++;
		}

		while (i < lines.length) {
			const line = lines[i].trim();

			// Look for timestamp line
			if (line.includes('-->')) {
				const match = line.match(/(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})\s*-->\s*(\d{1,2}:\d{2}(?::\d{2})?[.,]\d{3})/);
				if (match) {
					const start = parseVttTimestamp(match[1]);
					const end = parseVttTimestamp(match[2]);

					// Collect text lines until empty line or next cue
					const textLines: string[] = [];
					i++;
					while (i < lines.length && lines[i].trim() !== '' && !lines[i].includes('-->')) {
						// Skip cue identifiers (numeric lines before timestamp)
						if (!/^\d+$/.test(lines[i].trim())) {
							textLines.push(lines[i].trim());
						}
						i++;
					}

					if (textLines.length > 0) {
						// Strip VTT formatting tags like <i>, </i>, etc.
						const rawText = textLines.join('\n').replace(/<[^>]+>/g, '');
						const { cleanText, speaker, isSoundEffect } = detectSpeakerAndEffects(rawText);
						cues.push({ start, end, text: cleanText, speaker, isSoundEffect });
					}
				} else {
					i++;
				}
			} else {
				i++;
			}
		}
		return cues;
	}

	function parseVttTimestamp(ts: string): number {
		const parts = ts.replace(',', '.').split(':');
		if (parts.length === 3) {
			return parseFloat(parts[0]) * 3600 + parseFloat(parts[1]) * 60 + parseFloat(parts[2]);
		} else if (parts.length === 2) {
			return parseFloat(parts[0]) * 60 + parseFloat(parts[1]);
		}
		return 0;
	}

	function formatOffset(offset: number): string {
		if (offset === 0) return '0s';
		const sign = offset > 0 ? '+' : '';
		return `${sign}${offset.toFixed(1)}s`;
	}

	function getSubtitleLabel(track: SubtitleTrack): string {
		const parts: string[] = [];
		if (track.title) {
			parts.push(track.title);
		} else if (track.language) {
			const langNames: Record<string, string> = {
				eng: 'English', spa: 'Spanish', fre: 'French', fra: 'French',
				deu: 'German', ger: 'German', ita: 'Italian', por: 'Portuguese',
				jpn: 'Japanese', kor: 'Korean', zho: 'Chinese', chi: 'Chinese',
				rus: 'Russian', ara: 'Arabic', hin: 'Hindi', dut: 'Dutch',
				pol: 'Polish', swe: 'Swedish', dan: 'Danish', fin: 'Finnish',
				nor: 'Norwegian', cze: 'Czech', hun: 'Hungarian', gre: 'Greek',
				heb: 'Hebrew', tha: 'Thai', tur: 'Turkish', vie: 'Vietnamese',
				ind: 'Indonesian', und: 'Unknown'
			};
			parts.push(langNames[track.language] || track.language.toUpperCase());
		} else {
			parts.push(`Track ${track.index + 1}`);
		}
		if (track.forced) parts.push('(Forced)');
		if (track.external) parts.push('(External)');
		return parts.join(' ');
	}

	// Pre-load subtitle in background
	async function preloadSubtitle(trackIndex: number) {
		if (originalVttText.has(trackIndex)) return;
		try {
			const subtitleUrl = getSubtitleTrackUrl(mediaType, mediaId, trackIndex);
			console.log('Pre-loading subtitle:', subtitleUrl);
			const response = await fetch(subtitleUrl, { credentials: 'include' });
			if (response.ok) {
				const vttText = await response.text();
				originalVttText.set(trackIndex, vttText);
				console.log('Pre-loaded subtitle, length:', vttText.length);
			}
		} catch (e) {
			console.error('Failed to pre-load subtitle:', e);
		}
	}

	let subtitleLoading = $state(false);
	async function selectSubtitle(trackIndex: number | null) {
		console.log('selectSubtitle:', trackIndex);
		selectedSubtitle = trackIndex;
		showSubtitleMenu = false;
		clearSubtitles();

		if (trackIndex !== null) {
			const track = subtitleTracks.find(t => t.index === trackIndex);
			if (track) {
				subtitleLoading = true;
				try {
					let vttText = originalVttText.get(track.index);
					if (!vttText) {
						const subtitleUrl = getSubtitleTrackUrl(mediaType, mediaId, track.index);
						console.log('Fetching:', subtitleUrl);
						const response = await fetch(subtitleUrl, { credentials: 'include' });
						console.log('Response status:', response.status);
						if (!response.ok) throw new Error(`Failed to fetch subtitle: ${response.status}`);
						vttText = await response.text();
						console.log('Got VTT, length:', vttText.length);
						originalVttText.set(track.index, vttText);
					}

					// Parse VTT into cues for custom rendering
					subtitleCues = parseVtt(vttText);
					console.log('Parsed cues:', subtitleCues.length);
					updateCurrentSubtitle();
				} catch (e) {
					console.error('Failed to load subtitle:', e);
				} finally {
					subtitleLoading = false;
				}
			}
		}
	}

	$effect(() => {
		if (video) {
			video.volume = volume;
			video.muted = muted;
		}
	});

	// Apply initial subtitle when video and tracks are ready
	let initialSubtitleApplied = false;
	$effect(() => {
		if (video && subtitleTracks.length > 0 && initialSubtitle !== null && !initialSubtitleApplied) {
			if (subtitleTracks.some(t => t.index === initialSubtitle)) {
				initialSubtitleApplied = true;
				selectSubtitle(initialSubtitle);
			}
		}
	});

	// Close menus when clicking outside, and handle play/pause on video area
	function handleContainerClick(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.settings-menu') && !target.closest('.settings-btn')) {
			showSettingsMenu = false;
		}
		if (!target.closest('.subtitle-menu') && !target.closest('.subtitle-btn')) {
			showSubtitleMenu = false;
		}
		if (!target.closest('.audio-menu') && !target.closest('.audio-btn')) {
			showAudioMenu = false;
		}

		// If clicking on overlay (not on controls), trigger video click behavior
		if (target.closest('.player-overlay') &&
			!target.closest('.player-controls') &&
			!target.closest('.player-top-bar') &&
			!target.closest('button')) {
			handleVideoClick(e);
		}
	}
</script>

<svelte:document onfullscreenchange={handleFullscreenChange} />

<div
	bind:this={container}
	class="player-container"
	onmousemove={handleMouseMove}
	onclick={handleContainerClick}
	role="application"
	aria-label="Video player"
>
	<!-- Loading overlay -->
	{#if loading}
		<div class="player-loading">
			<div class="loading-spinner"></div>
		</div>
	{/if}

	<!-- Video element -->
	<video
		bind:this={video}
		src={currentSrc}
		class="w-full h-full video-player"
		style={getAspectRatioStyle()}
		ontimeupdate={handleTimeUpdate}
		onloadedmetadata={handleLoadedMetadata}
		onplay={handlePlay}
		onpause={handlePause}
		onclick={handleVideoClick}
		crossorigin="anonymous"
		autoplay
	>
	</video>

	<!-- Playback Info Overlay -->
	{#if showPlaybackInfo && playbackInfo}
		<div class="playback-info-overlay">
			<div class="info-title">Playback Info</div>

			<div class="info-section">
				<div class="info-section-title">Playback</div>
				<div class="info-row" class:good={playbackInfo.playMethod === 'Direct Play'} class:warning={playbackInfo.playMethod === 'Transcode'}>
					<span>Play Method</span>
					<span>{playbackInfo.playMethod}</span>
				</div>
				<div class="info-row">
					<span>Container</span>
					<span>{playbackInfo.container}</span>
				</div>
				{#if playbackInfo.bitrate !== 'N/A'}
					<div class="info-row">
						<span>Bitrate</span>
						<span>{playbackInfo.bitrate}</span>
					</div>
				{/if}
			</div>

			<div class="info-section">
				<div class="info-section-title">Video</div>
				<div class="info-row">
					<span>Resolution</span>
					<span>{playbackInfo.resolution}</span>
				</div>
				<div class="info-row">
					<span>Codec</span>
					<span>{playbackInfo.videoCodec}</span>
				</div>
				<div class="info-row" class:good={playbackInfo.droppedFrames === 0} class:warning={playbackInfo.droppedFrames > 0 && playbackInfo.droppedFrames < 10} class:bad={playbackInfo.droppedFrames >= 10}>
					<span>Dropped Frames</span>
					<span>{playbackInfo.droppedFrames}{playbackInfo.totalFrames > 0 ? ` / ${playbackInfo.totalFrames}` : ''}</span>
				</div>
			</div>

			<div class="info-section">
				<div class="info-section-title">Audio</div>
				<div class="info-row">
					<span>Codec</span>
					<span>{playbackInfo.audioCodec}{playbackInfo.audioChannels ? ` (${playbackInfo.audioChannels})` : ''}</span>
				</div>
			</div>

			<div class="info-hint">Press 'i' to close</div>
		</div>
	{/if}

	<!-- Next Episode Popup -->
	{#if showNextEpisode && nextEpisode}
		<div class="next-episode-popup">
			<div class="next-episode-content">
				<div class="next-episode-label">Up Next</div>
				<div class="next-episode-title">S{nextEpisode.seasonNumber}E{nextEpisode.episodeNumber}: {nextEpisode.title || 'Next Episode'}</div>
				<div class="next-episode-actions">
					<button class="next-episode-btn cancel" onclick={clearNextEpisodeTimer}>
						Cancel
					</button>
					<button class="next-episode-btn play" onclick={playNextEpisode}>
						Play Now ({nextEpisodeCountdown}s)
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Custom subtitle overlay -->
	{#if currentSubtitleText}
		<div class="subtitle-overlay">
			<div class="subtitle-content">
				{#if currentSpeaker}
					<span class="subtitle-speaker">{currentSpeaker}</span>
				{/if}
				<div class="subtitle-text">
					{@html currentSubtitleText.replace(/\n/g, '<br>')}
				</div>
			</div>
		</div>
	{/if}

	<!-- Controls overlay -->
	<div class="player-overlay {showControls ? 'visible' : ''}">
		<!-- Top gradient -->
		<div class="player-top-gradient"></div>

		<!-- Top bar -->
		<div class="player-top-bar">
			{#if onClose}
				<button class="player-btn" onclick={onClose} aria-label="Go back">
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
				</button>
			{:else}
				<div class="w-10"></div>
			{/if}
			<div class="player-title-container">
				<h2 class="player-title">{title}</h2>
				{#if subtitle}
					<p class="player-subtitle-text">{subtitle}</p>
				{/if}
			</div>
			<div class="w-10"></div>
		</div>

		<!-- Bottom gradient -->
		<div class="player-bottom-gradient"></div>

		<!-- Bottom controls -->
		<div class="player-controls">
			<!-- End time above progress bar -->
			{#if getEndTime()}
				<div class="end-time-row">
					<span class="player-end-time">Ends {getEndTime()}</span>
				</div>
			{/if}

			<!-- Progress bar -->
			<div class="progress-row">
				<span class="player-time">{formatTime(getActualTime())}</span>

				<div
					bind:this={progressBar}
					class="progress-container"
					onclick={handleProgressClick}
					onmousemove={handleProgressHover}
					onmouseleave={handleProgressLeave}
					role="slider"
					aria-label="Seek"
					aria-valuemin={0}
					aria-valuemax={getDuration()}
					aria-valuenow={getActualTime()}
					tabindex="0"
				>
					<!-- Buffered -->
					<div class="progress-buffered" style="width: {getDuration() ? (buffered / getDuration()) * 100 : 0}%"></div>
					<!-- Played -->
					<div class="progress-played" style="width: {getDuration() ? (getActualTime() / getDuration()) * 100 : 0}%"></div>
					<!-- Handle -->
					<div class="progress-handle" style="left: {getDuration() ? (getActualTime() / getDuration()) * 100 : 0}%"></div>

					<!-- Hover preview -->
					{#if hoverTime !== null}
						<div class="timestamp-preview" style="left: {hoverX}px">
							{formatTime(hoverTime)}
						</div>
					{/if}
				</div>

				<span class="player-time">{formatTime(getDuration())}</span>
			</div>

			<!-- Control buttons -->
			<div class="controls-row">
				<div class="controls-left">
					<!-- Skip back -->
					<button class="player-btn" onclick={() => seek(-10)} aria-label="Skip back 10 seconds">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M12.5 3C17.15 3 21.08 6.03 22.47 10.22L20.1 11C19.05 7.81 16.04 5.5 12.5 5.5C10.54 5.5 8.77 6.22 7.38 7.38L10 10H3V3L5.6 5.6C7.45 4 9.85 3 12.5 3M10 12V22H8V14H6V12H10M18 14V20C18 21.11 17.11 22 16 22H14C12.9 22 12 21.1 12 20V14C12 12.9 12.9 12 14 12H16C17.11 12 18 12.9 18 14M14 14V20H16V14H14Z"/>
						</svg>
					</button>

					<!-- Play/Pause -->
					<button class="player-btn player-btn-play" onclick={togglePlay} aria-label={playing ? 'Pause' : 'Play'}>
						{#if playing}
							<svg class="w-7 h-7" fill="currentColor" viewBox="0 0 24 24">
								<path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
							</svg>
						{:else}
							<svg class="w-7 h-7" fill="currentColor" viewBox="0 0 24 24">
								<path d="M8 5v14l11-7z" />
							</svg>
						{/if}
					</button>

					<!-- Skip forward -->
					<button class="player-btn" onclick={() => seek(10)} aria-label="Skip forward 10 seconds">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M11.5 3C6.85 3 2.92 6.03 1.53 10.22L3.9 11C4.95 7.81 7.96 5.5 11.5 5.5C13.46 5.5 15.23 6.22 16.62 7.38L14 10H21V3L18.4 5.6C16.55 4 14.15 3 11.5 3M10 12V22H8V14H6V12H10M18 14V20C18 21.11 17.11 22 16 22H14C12.9 22 12 21.1 12 20V14C12 12.9 12.9 12 14 12H16C17.11 12 18 12.9 18 14M14 14V20H16V14H14Z"/>
						</svg>
					</button>

					<!-- Volume -->
					<div
						class="volume-container"
						onmouseenter={() => showVolumeSlider = true}
						onmouseleave={() => showVolumeSlider = false}
					>
						<button class="player-btn" onclick={() => muted = !muted} aria-label={muted ? 'Unmute' : 'Mute'}>
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
						<div class="volume-slider {showVolumeSlider ? 'expanded' : ''}">
							<div class="volume-bar">
								<div class="volume-level" style="width: {muted ? 0 : volume * 100}%"></div>
							</div>
							<input
								type="range"
								min="0"
								max="1"
								step="0.05"
								bind:value={volume}
								class="volume-input"
								aria-label="Volume"
							/>
						</div>
					</div>
				</div>

				<div class="controls-right">
					<!-- Playback Info -->
					<button
						class="player-btn {showPlaybackInfo ? 'active' : ''}"
						onclick={togglePlaybackInfo}
						aria-label="Playback info"
						title="Playback Info (I)"
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
						</svg>
					</button>

					<!-- Subtitles -->
					{#if subtitleTracks.length > 0}
					<div class="relative subtitle-menu">
							<button
								class="player-btn subtitle-btn {selectedSubtitle !== null ? 'active' : ''}"
								onclick={() => { showSubtitleMenu = !showSubtitleMenu; showSettingsMenu = false; showAudioMenu = false; }}
								aria-label="Subtitles"
							>
								<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
									<path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z"/>
								</svg>
							</button>

							{#if showSubtitleMenu}
								<div class="track-dropdown">
									<button
										class="track-item {selectedSubtitle === null ? 'active' : ''}"
										onclick={() => selectSubtitle(null)}
									>
										<span class="track-item-check">
											{#if selectedSubtitle === null}
												<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
													<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
												</svg>
											{/if}
										</span>
										<span class="track-item-label">Off</span>
									</button>
									{#each subtitleTracks as track (track.index)}
										<button
											class="track-item {selectedSubtitle === track.index ? 'active' : ''}"
											onclick={() => selectSubtitle(track.index)}
										>
											<span class="track-item-check">
												{#if selectedSubtitle === track.index}
													<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
														<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
													</svg>
												{/if}
											</span>
											<span class="track-item-label">{getSubtitleLabel(track)}</span>
										</button>
									{/each}

									<!-- Timing controls -->
									{#if selectedSubtitle !== null}
										<div class="timing-divider"></div>
										<div class="timing-header">Timing Offset</div>
										<div class="timing-controls">
											<button
												class="timing-btn"
												onclick={() => adjustSubtitleOffset(-0.5)}
												title="Delay subtitles (G)"
											>
												-0.5s
											</button>
											<span class="timing-value">{formatOffset(subtitleOffset)}</span>
											<button
												class="timing-btn"
												onclick={() => adjustSubtitleOffset(0.5)}
												title="Earlier subtitles (Shift+G)"
											>
												+0.5s
											</button>
											<button
												class="timing-btn timing-reset"
												onclick={() => { subtitleOffset = 0; updateCurrentSubtitle(); }}
												title="Reset (H)"
											>
												Reset
											</button>
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/if}

					<!-- Audio -->
					<div class="relative audio-menu">
						<button
							class="player-btn audio-btn"
							onclick={() => { showAudioMenu = !showAudioMenu; showSubtitleMenu = false; showSettingsMenu = false; }}
							aria-label="Audio settings"
						>
							<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
								<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
							</svg>
						</button>

						{#if showAudioMenu}
							<div class="track-dropdown audio-dropdown">
								{#if audioTracks.length > 1}
									<div class="settings-header">Audio Track</div>
									{#each audioTracks as track (track.index)}
										<button
											class="track-item {selectedAudioTrack === track.index ? 'active' : ''}"
											onclick={() => selectAudioTrack(track.index)}
										>
											<span class="track-item-check">
												{#if selectedAudioTrack === track.index}
													<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
														<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
													</svg>
												{/if}
											</span>
											<span class="track-item-label">{getAudioTrackLabel(track)}</span>
										</button>
									{/each}
									<div class="settings-divider"></div>
								{/if}

								<div class="settings-header">Audio Sync</div>
								<div class="audio-sync-control">
									<button
										class="sync-btn"
										onclick={() => adjustAudioSync(-0.05)}
										aria-label="Audio earlier"
									>−</button>
									<span class="sync-value" class:sync-negative={audioSync < 0} class:sync-positive={audioSync > 0}>
										{audioSync >= 0 ? '+' : ''}{(audioSync * 1000).toFixed(0)}ms
									</span>
									<button
										class="sync-btn"
										onclick={() => adjustAudioSync(0.05)}
										aria-label="Audio later"
									>+</button>
									{#if audioSync !== 0}
										<button
											class="sync-reset"
											onclick={() => setAudioSync(0)}
											aria-label="Reset audio sync"
										>Reset</button>
									{/if}
								</div>
								<div class="sync-hint">− earlier / + later</div>
							</div>
						{/if}
					</div>

					<!-- Settings -->
					<div class="relative settings-menu">
						<button
							class="player-btn settings-btn"
							onclick={() => { showSettingsMenu = !showSettingsMenu; showSubtitleMenu = false; showAudioMenu = false; }}
							aria-label="Settings"
						>
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
							</svg>
						</button>

						{#if showSettingsMenu}
							<div class="settings-dropdown">
								<div class="settings-header">Playback Speed</div>
								{#each speedOptions as speed}
									<button
										class="settings-item {playbackSpeed === speed ? 'active' : ''}"
										onclick={() => setPlaybackSpeed(speed)}
									>
										<span>{speed === 1 ? 'Normal' : speed + 'x'}</span>
										{#if playbackSpeed === speed}
											<svg class="w-4 h-4 text-amber" fill="currentColor" viewBox="0 0 24 24">
												<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
											</svg>
										{/if}
									</button>
								{/each}

								<div class="settings-divider"></div>
								<div class="settings-header">Aspect Ratio</div>
								{#each aspectRatioOptions as option}
									<button
										class="settings-item {aspectRatio === option.value ? 'active' : ''}"
										onclick={() => setAspectRatio(option.value)}
									>
										<span>{option.label}</span>
										{#if aspectRatio === option.value}
											<svg class="w-4 h-4 text-amber" fill="currentColor" viewBox="0 0 24 24">
												<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
											</svg>
										{/if}
									</button>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Fullscreen -->
					<button class="player-btn" onclick={toggleFullscreen} aria-label={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}>
						{#if fullscreen}
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/>
							</svg>
						{:else}
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
								<path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/>
							</svg>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	/* Color variables */
	:root {
		--player-cream: #F5E6C8;
		--player-cream-secondary: rgba(245, 230, 200, 0.7);
		--player-cream-muted: rgba(245, 230, 200, 0.5);
		--player-amber: #E8A849;
		--player-bg: #0a0a0a;
		--player-glass: rgba(255, 255, 255, 0.06);
		--player-glass-hover: rgba(255, 255, 255, 0.1);
		--player-glass-border: rgba(255, 255, 255, 0.1);
	}

	.player-container {
		position: relative;
		width: 100%;
		height: 100%;
		background: var(--player-bg);
		overflow: hidden;
	}

	/* Loading */
	.player-loading {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(10, 10, 10, 0.6);
		z-index: 50;
	}

	.loading-spinner {
		width: 48px;
		height: 48px;
		border: 3px solid rgba(245, 230, 200, 0.2);
		border-top-color: var(--player-amber);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Overlay */
	.player-overlay {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		opacity: 0;
		transition: opacity 0.3s;
		pointer-events: none;
	}

	.player-overlay.visible {
		opacity: 1;
		pointer-events: auto;
	}

	/* Gradients */
	.player-top-gradient {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 120px;
		background: linear-gradient(to bottom, rgba(10, 10, 10, 0.8), transparent);
		pointer-events: none;
	}

	.player-bottom-gradient {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 200px;
		background: linear-gradient(to top, rgba(10, 10, 10, 0.95), transparent);
		pointer-events: none;
	}

	/* Top bar */
	.player-top-bar {
		position: relative;
		z-index: 10;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 20px 24px;
	}

	.player-title-container {
		text-align: center;
	}

	.player-title {
		color: var(--player-cream);
		font-size: 18px;
		font-weight: 500;
	}

	.player-subtitle-text {
		color: var(--player-cream-secondary);
		font-size: 14px;
	}

	/* Bottom controls */
	.player-controls {
		position: relative;
		z-index: 10;
		padding: 0 24px 20px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	/* Progress row */
	.progress-row {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.player-time {
		color: var(--player-cream-secondary);
		font-size: 13px;
		font-weight: 500;
		font-variant-numeric: tabular-nums;
		min-width: 50px;
	}

	.player-time:last-child {
		text-align: right;
	}

	/* Progress bar */
	.progress-container {
		flex: 1;
		position: relative;
		height: 4px;
		background: rgba(245, 230, 200, 0.2);
		border-radius: 2px;
		cursor: pointer;
		transition: height 0.2s;
	}

	.progress-container:hover {
		height: 8px;
	}

	.progress-buffered {
		position: absolute;
		height: 100%;
		background: rgba(245, 230, 200, 0.4);
		border-radius: 2px;
		pointer-events: none;
	}

	.progress-played {
		position: absolute;
		height: 100%;
		background: var(--player-amber);
		border-radius: 2px;
		pointer-events: none;
	}

	.progress-handle {
		position: absolute;
		width: 16px;
		height: 16px;
		background: var(--player-amber);
		border: 2px solid var(--player-cream);
		border-radius: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		opacity: 0;
		transition: opacity 0.2s;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
		pointer-events: none;
	}

	.progress-container:hover .progress-handle {
		opacity: 1;
	}

	.timestamp-preview {
		position: absolute;
		bottom: 100%;
		margin-bottom: 8px;
		transform: translateX(-50%);
		padding: 6px 12px;
		background: rgba(10, 10, 10, 0.9);
		backdrop-filter: blur(8px);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 6px;
		color: var(--player-cream);
		font-size: 13px;
		font-weight: 500;
		white-space: nowrap;
		pointer-events: none;
	}

	/* Controls row */
	.controls-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.controls-left, .controls-right {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	/* Buttons */
	.player-btn {
		background: none;
		border: none;
		color: var(--player-cream);
		padding: 10px;
		cursor: pointer;
		border-radius: 50%;
		transition: all 0.2s;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.player-btn:hover {
		color: var(--player-amber);
		background: var(--player-glass);
	}

	.player-btn.active {
		color: var(--player-amber);
	}

	.player-btn-play {
		padding: 12px;
	}

	/* Volume */
	.volume-container {
		display: flex;
		align-items: center;
	}

	.volume-slider {
		width: 0;
		overflow: hidden;
		transition: width 0.2s;
		display: flex;
		align-items: center;
		position: relative;
	}

	.volume-slider.expanded {
		width: 80px;
		margin-left: 4px;
	}

	.volume-bar {
		position: absolute;
		width: 100%;
		height: 4px;
		background: rgba(245, 230, 200, 0.2);
		border-radius: 2px;
		pointer-events: none;
	}

	.volume-level {
		height: 100%;
		background: var(--player-cream);
		border-radius: 2px;
	}

	.volume-input {
		width: 100%;
		height: 4px;
		background: transparent;
		appearance: none;
		cursor: pointer;
		position: relative;
		z-index: 1;
	}

	.volume-input::-webkit-slider-thumb {
		appearance: none;
		width: 12px;
		height: 12px;
		background: var(--player-cream);
		border-radius: 50%;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
	}

	/* Dropdowns */
	.track-dropdown, .settings-dropdown {
		position: absolute;
		bottom: 100%;
		right: 0;
		margin-bottom: 12px;
		min-width: 200px;
		max-height: 300px;
		overflow-y: auto;
		background: rgba(10, 10, 10, 0.95);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		padding: 8px 0;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
	}

	.track-item, .settings-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		color: var(--player-cream);
		font-size: 14px;
		cursor: pointer;
		transition: background 0.2s;
		background: none;
		border: none;
		width: 100%;
		text-align: left;
	}

	.track-item:hover, .settings-item:hover {
		background: var(--player-glass);
	}

	.track-item.active, .settings-item.active {
		color: var(--player-amber);
	}

	.track-item-check {
		width: 16px;
		color: var(--player-amber);
	}

	.track-item-label {
		flex: 1;
	}

	.settings-header {
		padding: 8px 16px;
		color: var(--player-cream-muted);
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.settings-divider {
		height: 1px;
		background: rgba(245, 230, 200, 0.1);
		margin: 8px 0;
	}

	.audio-sync-control {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
	}

	.sync-btn {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: var(--player-cream);
		font-size: 18px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sync-btn:hover {
		background: rgba(255, 255, 255, 0.2);
	}

	.sync-value {
		min-width: 70px;
		text-align: center;
		font-size: 14px;
		font-weight: 500;
		color: var(--player-cream);
		font-family: monospace;
	}

	.sync-negative {
		color: #4ade80;
	}

	.sync-positive {
		color: #f97316;
	}

	.sync-reset {
		padding: 4px 10px;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.1);
		border: none;
		color: var(--player-cream-muted);
		font-size: 12px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.sync-reset:hover {
		background: rgba(255, 255, 255, 0.2);
		color: var(--player-cream);
	}

	.sync-hint {
		padding: 4px 16px 8px;
		font-size: 11px;
		color: var(--player-cream-muted);
		text-align: center;
	}

	.settings-item {
		justify-content: space-between;
	}

	/* Scrollbar for dropdowns */
	.track-dropdown::-webkit-scrollbar {
		width: 6px;
	}

	.track-dropdown::-webkit-scrollbar-track {
		background: transparent;
	}

	.track-dropdown::-webkit-scrollbar-thumb {
		background: rgba(245, 230, 200, 0.2);
		border-radius: 3px;
	}

	/* Custom subtitle overlay */
	.subtitle-overlay {
		position: absolute;
		bottom: 100px;
		left: 0;
		right: 0;
		display: flex;
		justify-content: center;
		pointer-events: none;
		z-index: 30;
		padding: 0 40px;
	}

	.subtitle-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		max-width: 85%;
	}

	.subtitle-speaker {
		color: var(--player-amber);
		font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif;
		font-size: 16px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 1px;
		text-shadow:
			1px 1px 2px rgba(0, 0, 0, 0.95),
			-1px -1px 2px rgba(0, 0, 0, 0.95),
			0 0 4px rgba(0, 0, 0, 0.7);
	}

	.subtitle-text {
		color: var(--player-cream);
		font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, Roboto, sans-serif;
		font-size: 28px;
		font-weight: 600;
		line-height: 1.4;
		text-align: center;
		/* Clean text outline for readability */
		text-shadow:
			1px 1px 2px rgba(0, 0, 0, 0.95),
			-1px -1px 2px rgba(0, 0, 0, 0.95),
			1px -1px 2px rgba(0, 0, 0, 0.95),
			-1px 1px 2px rgba(0, 0, 0, 0.95),
			0 0 6px rgba(0, 0, 0, 0.7);
	}

	/* Timing controls in subtitle menu */
	.timing-divider {
		height: 1px;
		background: rgba(245, 230, 200, 0.15);
		margin: 8px 0;
	}

	.timing-header {
		padding: 6px 16px;
		color: var(--player-cream-muted);
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.timing-controls {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
	}

	.timing-btn {
		background: var(--player-glass);
		border: 1px solid rgba(245, 230, 200, 0.2);
		color: var(--player-cream);
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.timing-btn:hover {
		background: var(--player-glass-hover);
		border-color: rgba(245, 230, 200, 0.3);
	}

	.timing-value {
		color: var(--player-amber);
		font-size: 13px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		min-width: 50px;
		text-align: center;
	}

	.timing-reset {
		margin-left: auto;
		color: var(--player-cream-muted);
	}

	.timing-reset:hover {
		color: var(--player-cream);
	}

	@media (max-width: 768px) {
		.subtitle-overlay {
			bottom: 80px;
			padding: 0 16px;
		}

		.subtitle-text {
			font-size: 18px;
		}

		.player-top-bar {
			padding: 12px 16px;
		}

		.player-controls {
			padding: 0 16px 16px;
		}

		.player-title {
			font-size: 14px;
		}
	}

	@media (min-width: 1920px) {
		.subtitle-text {
			font-size: 36px;
		}
	}

	/* Text color helper */
	.text-amber {
		color: var(--player-amber);
	}

	/* End time display */
	.end-time-row {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 4px;
	}

	.player-end-time {
		color: var(--player-cream-muted);
		font-size: 12px;
		font-weight: 500;
	}

	/* Video player aspect ratio */
	.video-player {
		width: 100%;
		height: 100%;
		background: #000;
	}

	/* Playback Info Overlay */
	.playback-info-overlay {
		position: absolute;
		top: 80px;
		left: 24px;
		background: rgba(0, 0, 0, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(245, 230, 200, 0.15);
		border-radius: 12px;
		padding: 16px 20px;
		z-index: 40;
		min-width: 280px;
		font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
	}

	.info-title {
		color: var(--player-amber);
		font-size: 13px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 1px;
		margin-bottom: 12px;
		padding-bottom: 8px;
		border-bottom: 1px solid rgba(245, 230, 200, 0.15);
	}

	.info-section {
		margin-bottom: 12px;
	}

	.info-section:last-of-type {
		margin-bottom: 0;
	}

	.info-section-title {
		color: var(--player-cream-muted);
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 6px;
	}

	.info-row {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		padding: 4px 0;
		font-size: 12px;
	}

	.info-row span:first-child {
		color: var(--player-cream-muted);
	}

	.info-row span:last-child {
		color: var(--player-cream);
		font-weight: 500;
		text-align: right;
	}

	.info-row.good span:last-child {
		color: #4ade80;
	}

	.info-row.warning span:last-child {
		color: #fbbf24;
	}

	.info-row.bad span:last-child {
		color: #f87171;
	}

	.info-hint {
		color: var(--player-cream-muted);
		font-size: 10px;
		margin-top: 12px;
		padding-top: 8px;
		border-top: 1px solid rgba(245, 230, 200, 0.1);
		text-align: center;
	}

	/* Next Episode Popup */
	.next-episode-popup {
		position: absolute;
		bottom: 120px;
		right: 24px;
		background: rgba(0, 0, 0, 0.9);
		backdrop-filter: blur(16px);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 16px;
		padding: 20px 24px;
		z-index: 45;
		min-width: 300px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
		animation: slideIn 0.3s ease-out;
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateX(20px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	.next-episode-content {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.next-episode-label {
		color: var(--player-cream-muted);
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 1px;
	}

	.next-episode-title {
		color: var(--player-cream);
		font-size: 16px;
		font-weight: 600;
		line-height: 1.3;
	}

	.next-episode-actions {
		display: flex;
		gap: 12px;
		margin-top: 4px;
	}

	.next-episode-btn {
		flex: 1;
		padding: 12px 16px;
		border-radius: 10px;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		border: none;
	}

	.next-episode-btn.cancel {
		background: rgba(255, 255, 255, 0.1);
		color: var(--player-cream);
		border: 1px solid rgba(245, 230, 200, 0.2);
	}

	.next-episode-btn.cancel:hover {
		background: rgba(255, 255, 255, 0.2);
	}

	.next-episode-btn.play {
		background: var(--player-amber);
		color: #0a0a0a;
	}

	.next-episode-btn.play:hover {
		background: #f0b85d;
	}
</style>
