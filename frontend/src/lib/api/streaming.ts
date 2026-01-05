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
