<script lang="ts">
	import type { VideoStream, AudioStream, SubtitleTrack } from '$lib/api';
	import Dropdown from './Dropdown.svelte';

	interface Props {
		videoStreams?: VideoStream[];
		audioStreams?: AudioStream[];
		subtitleTracks?: SubtitleTrack[];
		selectedSubtitle?: number | null;
		onSubtitleChange?: (index: number | null) => void;
		compact?: boolean;
	}

	let { videoStreams = [], audioStreams = [], subtitleTracks = [], selectedSubtitle = null, onSubtitleChange, compact = false }: Props = $props();

	let internalSubtitle = $state(selectedSubtitle);
	let selectedVideo = $state(0);
	let selectedAudio = $state(0);

	$effect(() => {
		internalSubtitle = selectedSubtitle;
	});

	function handleSubtitleChange(value: string | number | null) {
		const newValue = value === null || value === '' ? null : typeof value === 'number' ? value : parseInt(value);
		internalSubtitle = newValue;
		onSubtitleChange?.(newValue);
	}

	function formatResolution(width: number, height: number): string {
		// Check both width and height to handle widescreen content
		if (width >= 3840 || height >= 2160) return '4K';
		if (width >= 2560 || height >= 1440) return '1440p';
		if (width >= 1920 || height >= 1080) return '1080p';
		if (width >= 1280 || height >= 720) return '720p';
		return `${height}p`;
	}

	function formatCodec(codec: string): string {
		const map: Record<string, string> = {
			'hevc': 'HEVC', 'h265': 'HEVC', 'h264': 'H.264', 'av1': 'AV1',
			'aac': 'AAC', 'ac3': 'AC3', 'eac3': 'E-AC3', 'dts': 'DTS',
			'truehd': 'TrueHD', 'flac': 'FLAC', 'opus': 'Opus'
		};
		return map[codec.toLowerCase()] || codec.toUpperCase();
	}

	function formatChannels(channels: number, layout?: string): string {
		if (layout?.includes('atmos')) return 'Atmos';
		if (channels === 8) return '7.1';
		if (channels === 6) return '5.1';
		if (channels === 2) return 'Stereo';
		if (channels === 1) return 'Mono';
		return `${channels}ch`;
	}

	function getLanguageName(code?: string): string {
		if (!code) return 'Unknown';
		const map: Record<string, string> = {
			'eng': 'English', 'spa': 'Spanish', 'fre': 'French', 'fra': 'French',
			'deu': 'German', 'ger': 'German', 'ita': 'Italian', 'por': 'Portuguese',
			'jpn': 'Japanese', 'kor': 'Korean', 'zho': 'Chinese', 'chi': 'Chinese',
			'rus': 'Russian', 'ara': 'Arabic', 'hin': 'Hindi', 'und': 'Unknown'
		};
		return map[code] || code.toUpperCase();
	}

	function formatSize(bytes?: number): string {
		if (!bytes) return '';
		const gb = bytes / (1024 * 1024 * 1024);
		return gb >= 1 ? `${gb.toFixed(1)}GB` : `${(bytes / (1024 * 1024)).toFixed(0)}MB`;
	}

	function getVideoLabel(v: VideoStream): string {
		const parts = [formatResolution(v.width, v.height)];
		if (v.hdr) parts.push(v.hdr);
		parts.push(formatCodec(v.codec));
		return parts.join(' ');
	}

	function getAudioLabel(a: AudioStream): string {
		const parts = [getLanguageName(a.language)];
		parts.push(`(${formatCodec(a.codec)} ${formatChannels(a.channels, a.channelLayout)})`);
		if (a.title) parts.push(`- ${a.title}`);
		return parts.join(' ');
	}

	function getSubtitleLabel(t: SubtitleTrack): string {
		const parts = [t.title || getLanguageName(t.language)];
		if (t.forced) parts.push('(Forced)');
		if (t.sdh) parts.push('(SDH)');
		return parts.join(' ');
	}
</script>

<div class="flex flex-wrap items-center gap-3">
	<!-- Video -->
	{#if videoStreams.length > 0}
		{#if videoStreams.length === 1}
			<span class="liquid-glass px-4 py-2 rounded-xl text-sm text-text-primary flex items-center gap-2">
				<svg class="w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
				{getVideoLabel(videoStreams[0])}
			</span>
		{:else}
			<Dropdown
				icon="video"
				options={videoStreams.map(v => ({ value: v.index, label: getVideoLabel(v) }))}
				value={selectedVideo}
				onchange={(v) => selectedVideo = v as number}
			/>
		{/if}
	{/if}

	<!-- Audio -->
	{#if audioStreams.length > 0}
		{#if audioStreams.length === 1}
			<span class="liquid-glass px-4 py-2 rounded-xl text-sm text-text-primary flex items-center gap-2">
				<svg class="w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
				</svg>
				{getAudioLabel(audioStreams[0])}
			</span>
		{:else}
			<Dropdown
				icon="audio"
				options={audioStreams.map(a => ({ value: a.index, label: getAudioLabel(a) }))}
				value={selectedAudio}
				onchange={(v) => selectedAudio = v as number}
			/>
		{/if}
	{/if}

	<!-- Subtitles -->
	{#if subtitleTracks.length > 0}
		<Dropdown
			icon="subtitles"
			options={[{ value: null, label: 'Off' }, ...subtitleTracks.map(t => ({ value: t.index, label: getSubtitleLabel(t) }))]}
			value={internalSubtitle}
			onchange={handleSubtitleChange}
		/>
	{/if}
</div>
