import { API_BASE, apiFetch } from './core';

// Progress types

export interface Progress {
	mediaType: string;
	mediaId: number;
	position: number;
	duration: number;
}

export interface ContinueWatchingItem {
	mediaType: 'movie' | 'episode';
	mediaId: number;
	title: string;
	subtitle?: string;
	showTitle?: string;
	season?: number;
	episode?: number;
	posterPath?: string;
	backdropPath?: string;
	position: number;
	duration: number;
	progressPercent: number;
	updatedAt: string;
}

export interface WatchStatus {
	watched: boolean;
	progress: number;
}

// Progress API functions

export async function getProgress(type: string, id: number): Promise<Progress> {
	const response = await apiFetch(`${API_BASE}/progress/${type}/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveProgress(progress: Progress): Promise<void> {
	const response = await apiFetch(`${API_BASE}/progress`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(progress)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Continue Watching API functions

export async function getContinueWatching(): Promise<ContinueWatchingItem[]> {
	const response = await apiFetch(`${API_BASE}/continue-watching`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function removeContinueWatching(mediaType: string, mediaId: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/continue-watching/${mediaType}/${mediaId}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Watch state API functions

export async function getWatchStatus(mediaType: 'movie' | 'episode', mediaId: number): Promise<WatchStatus> {
	const response = await apiFetch(`${API_BASE}/watched/${mediaType}/${mediaId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function markAsWatched(mediaType: 'movie' | 'episode', mediaId: number, duration?: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/watched/${mediaType}/${mediaId}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ duration: duration || 3600 })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function markAsUnwatched(mediaType: 'movie' | 'episode', mediaId: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/watched/${mediaType}/${mediaId}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}
