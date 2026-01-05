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
		onClose?: () => void;
		initialSubtitle?: number | null;
		nextEpisodeData?: NextEpisode | null;
		onNextEpisode?: (episodeId: number) => void;
	}

	let { src, title, subtitle = '', mediaType, mediaId, onClose, initialSubtitle = null, nextEpisodeData = null, onNextEpisode }: Props = $props();

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
	type AspectRatio = 'fit' | 'fill' | '16:9' | '4:3' | '21:9';
	let aspectRatio = $state<AspectRatio>('fit');

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
		const [mediaInfoResult, progressResult, subtitlesResult] = await Promise.allSettled([
			getMediaInfo(mediaType, mediaId),
			getProgress(mediaType, mediaId),
			getSubtitleTracks(mediaType, mediaId)
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
	});

	onDestroy(() => {
		clearInterval(progressInterval);
		clearTimeout(controlsTimeout);
		if (clickTimeout) clearTimeout(clickTimeout);
		if (nextEpisodeTimer) clearInterval(nextEpisodeTimer);
		if (playbackInfoInterval) clearInterval(playbackInfoInterval);
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

		switch (e.key) {
			case ' ': case 'k': case 'K': e.preventDefault(); togglePlay(); break;
			case 'ArrowLeft': case 'j': case 'J': e.preventDefault(); seek(-10); break;
			case 'ArrowRight': case 'l': case 'L': e.preventDefault(); seek(10); break;
			case 'ArrowUp': e.preventDefault(); volume = Math.min(1, volume + 0.05); break;
			case 'ArrowDown': e.preventDefault(); volume = Math.max(0, volume - 0.05); break;
			case 'f': case 'F': e.preventDefault(); toggleFullscreen(); break;
			case 'm': case 'M': e.preventDefault(); muted = !muted; break;
			case 'c': case 'C': e.preventDefault(); toggleSubtitles(); break;
			case 's': case 'S': e.preventDefault(); cycleSubtitleTrack(); break;
			case 'g': e.preventDefault(); adjustSubtitleOffset(0.5); break;
			case 'G': e.preventDefault(); adjustSubtitleOffset(-0.5); break;
			case 'h': case 'H': e.preventDefault(); subtitleOffset = 0; break;
			case 'Escape': if (fullscreen) document.exitFullscreen(); else if (onClose) onClose(); break;
			case 'Home': e.preventDefault(); seekToTime(0); break;
			case 'End': e.preventDefault(); seekToTime(getDuration() - 1); break;
			case '0': case '1': case '2': case '3': case '4':
			case '5': case '6': case '7': case '8': case '9':
				e.preventDefault(); seekToTime(getDuration() * (parseInt(e.key) / 10)); break;
			case 'i': case 'I': e.preventDefault(); togglePlaybackInfo(); break;
			case 'a': case 'A': e.preventDefault(); cycleAspectRatio(); break;
			case 'n': case 'N': e.preventDefault(); if (nextEpisode && onNextEpisode) playNextEpisode(); break;
		}
	}

	function togglePlay() {
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
		if (!document.fullscreenElement) container.requestFullscreen();
		else document.exitFullscreen();
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
		currentTime = video.currentTime;
		if (video.buffered.length > 0) buffered = video.buffered.end(video.buffered.length - 1);
		updateCurrentSubtitle();
		if (nextEpisode && mediaType === 'episode') {
			const remaining = getDuration() - getActualTime();
			if (remaining > 0 && remaining <= 30 && !showNextEpisode) startNextEpisodeCountdown();
			else if (remaining > 30 && showNextEpisode) clearNextEpisodeTimer();
		}
	}

	function handleLoadedMetadata() {
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

<div bind:this={container} class="player-container" onmousemove={handleMouseMove} onclick={handleContainerClick} role="application" aria-label="Video player">
	{#if loading}
		<div class="player-loading"><div class="loading-spinner"></div></div>
	{/if}

	<video bind:this={video} src={currentSrc} class="video-element" style={getAspectRatioStyle()}
		ontimeupdate={handleTimeUpdate} onloadedmetadata={handleLoadedMetadata}
		onplay={handlePlay} onpause={handlePause} onclick={handleVideoClick}
		crossorigin="anonymous" autoplay>
	</video>

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
			<PlayerProgressBar
				currentTime={getActualTime()}
				duration={getDuration()}
				{buffered}
				endTime={getEndTime()}
				onSeek={seekToTime}
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
					<PlayerVolumeControl {volume} {muted} onVolumeChange={(v) => { volume = v; muted = false; }} onMuteToggle={() => muted = !muted} />
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
						onToggle={() => { showSettingsMenu = !showSettingsMenu; showSubtitleMenu = false; showAudioMenu = false; }}
						onSpeedChange={(s) => { playbackSpeed = s; if (video) video.playbackRate = s; }}
						onAspectChange={(r) => aspectRatio = r}
					/>

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

	@media (max-width: 768px) {
		.subtitle-overlay { bottom: 80px; padding: 0 16px; }
		.subtitle-text { font-size: 18px; }
		.player-top-bar { padding: 12px 16px; }
		.player-controls { padding: 0 16px 16px; }
		.player-title { font-size: 14px; }
	}

	@media (min-width: 1920px) { .subtitle-text { font-size: 36px; } }
</style>
