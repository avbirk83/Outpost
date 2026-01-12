import { API_BASE, apiFetch } from './core';

export function getStreamUrl(type: 'movie' | 'episode', id: number): string {
	return `${API_BASE}/stream/${type}/${id}`;
}

export interface VideoStream {
	index: number;
	codec: string;
	profile?: string;
	width: number;
	height: number;
	aspectRatio?: string;
	frameRate?: string;
	bitRate?: number;
	pixelFormat?: string;
	default: boolean;
}

export interface AudioStream {
	index: number;
	codec: string;
	channels: number;
	channelLayout?: string;
	sampleRate?: number;
	bitRate?: number;
	language?: string;
	title?: string;
	default: boolean;
}

export interface SubtitleTrack {
	index: number;
	language: string;
	title: string;
	codec: string;
	default: boolean;
	forced: boolean;
	external: boolean;
	filePath?: string;
}

export interface MediaInfo {
	duration: number;
	fileSize?: number;
	bitRate?: number;
	container?: string;
	videoStreams: VideoStream[];
	audioStreams: AudioStream[];
	subtitleTracks: SubtitleTrack[];
}

export async function getMediaInfo(type: 'movie' | 'episode', id: number): Promise<MediaInfo> {
	const response = await apiFetch(`${API_BASE}/media-info/${type}/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getSubtitleTracks(mediaType: 'movie' | 'episode', mediaId: number): Promise<SubtitleTrack[]> {
	const response = await apiFetch(`${API_BASE}/subtitles/${mediaType}/${mediaId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export function getSubtitleTrackUrl(mediaType: 'movie' | 'episode', mediaId: number, trackIndex: number): string {
	return `${API_BASE}/subtitles/${mediaType}/${mediaId}/track/${trackIndex}`;
}

export interface Chapter {
	index: number;
	title: string;
	startTime: number;
	endTime: number;
}

export async function getChapters(mediaType: 'movie' | 'episode', mediaId: number): Promise<Chapter[]> {
	const response = await apiFetch(`${API_BASE}/chapters/${mediaType}/${mediaId}`);
	if (!response.ok) {
		if (response.status === 404) {
			return []; // No chapters found
		}
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export interface SkipSegment {
	startTime: number;
	endTime: number;
}

export interface SkipSegments {
	intro?: SkipSegment;
	credits?: SkipSegment;
}

export async function getSkipSegments(showId: number): Promise<SkipSegments> {
	const response = await apiFetch(`${API_BASE}/skip-segments/${showId}`);
	if (!response.ok) {
		if (response.status === 404) {
			return {}; // No skip segments
		}
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveSkipSegment(
	showId: number,
	type: 'intro' | 'credits',
	startTime: number,
	endTime: number
): Promise<void> {
	const response = await apiFetch(`${API_BASE}/skip-segments/${showId}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ type, startTime, endTime })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function deleteSkipSegment(showId: number, type: 'intro' | 'credits'): Promise<void> {
	const response = await apiFetch(`${API_BASE}/skip-segments/${showId}/${type}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Per-episode media segments (intro/credits detection)
export interface MediaSegment {
	id: number;
	episodeId: number;
	segmentType: 'intro' | 'credits' | 'recap' | 'preview';
	startSeconds: number;
	endSeconds: number;
	confidence: number;
	source: 'chapter' | 'fingerprint' | 'blackframe' | 'user';
	createdAt: string;
}

export async function getMediaSegments(episodeId: number): Promise<MediaSegment[]> {
	const response = await apiFetch(`${API_BASE}/episodes/${episodeId}/segments`);
	if (!response.ok) {
		if (response.status === 404) {
			return [];
		}
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveMediaSegment(
	episodeId: number,
	segmentType: 'intro' | 'credits' | 'recap' | 'preview',
	startSeconds: number,
	endSeconds: number
): Promise<MediaSegment> {
	const response = await apiFetch(`${API_BASE}/episodes/${episodeId}/segments`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ segmentType, startSeconds, endSeconds })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteMediaSegment(episodeId: number, segmentId: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/episodes/${episodeId}/segments/${segmentId}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}
