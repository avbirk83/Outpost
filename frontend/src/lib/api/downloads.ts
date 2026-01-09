import { API_BASE, apiFetch } from './core';

// Download Client types

export interface DownloadClient {
	id: number;
	name: string;
	type: 'qbittorrent' | 'transmission' | 'sabnzbd' | 'nzbget';
	host: string;
	port: number;
	username?: string;
	password?: string;
	apiKey?: string;
	useTls: boolean;
	category?: string;
	priority: number;
	enabled: boolean;
}

export interface Download {
	id: string;
	name: string;
	size: number;
	downloaded: number;
	progress: number;
	speed: number;
	eta: number;
	status: 'downloading' | 'paused' | 'completed' | 'error' | 'queued';
	savePath: string;
	category: string;
	clientId: number;
	clientName: string;
	clientType: string;
}

export interface TestConnectionResult {
	success: boolean;
	message?: string;
	error?: string;
}

// Download Client API functions

export async function getDownloadClients(): Promise<DownloadClient[]> {
	const response = await apiFetch(`${API_BASE}/download-clients`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createDownloadClient(client: Omit<DownloadClient, 'id'>): Promise<DownloadClient> {
	const response = await apiFetch(`${API_BASE}/download-clients`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(client)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateDownloadClient(id: number, client: Partial<DownloadClient>): Promise<DownloadClient> {
	const response = await apiFetch(`${API_BASE}/download-clients/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(client)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteDownloadClient(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/download-clients/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function testDownloadClient(id: number): Promise<TestConnectionResult> {
	const response = await apiFetch(`${API_BASE}/download-clients/${id}/test`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getDownloads(): Promise<Download[]> {
	const response = await apiFetch(`${API_BASE}/downloads`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// TrackedDownload states
export type DownloadState =
	| 'queued'
	| 'downloading'
	| 'paused'
	| 'stalled'
	| 'completed'
	| 'import_pending'
	| 'importing'
	| 'imported'
	| 'import_blocked'
	| 'failed'
	| 'ignored';

// Download Tracking (TrackedDownload from backend)
export interface TrackedDownload {
	id: number;
	downloadClientId: number;
	externalId: string;
	requestId?: number;
	mediaId?: number;
	mediaType: string;
	tmdbId?: number;
	posterPath?: string;
	year?: number;

	state: DownloadState;
	previousState?: DownloadState;
	stateChangedAt: string;

	title: string;
	parsedInfo?: object;

	size: number;
	downloaded: number;
	progress: number;
	speed: number;
	eta: number;
	seeders: number;

	downloadPath?: string;
	importPath?: string;

	quality?: string;
	customFormatScore: number;

	grabbedAt: string;
	completedAt?: string;
	importedAt?: string;

	warnings?: string[];
	errors?: string[];
	importBlockReason?: string;

	ratio: number;
	seedingTime: number;
	canRemove: boolean;

	createdAt: string;
	updatedAt: string;
}

// Legacy alias for compatibility
export type DownloadItem = TrackedDownload;

export async function getDownloadItems(): Promise<DownloadItem[]> {
	const response = await apiFetch(`${API_BASE}/download-items`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export interface DeleteDownloadOptions {
	deleteFromClient?: boolean; // default: true
	deleteFiles?: boolean; // default: false
}

export async function deleteDownloadItem(id: number, options?: DeleteDownloadOptions): Promise<void> {
	const params = new URLSearchParams();
	if (options?.deleteFromClient === false) params.set('deleteFromClient', 'false');
	if (options?.deleteFiles) params.set('deleteFiles', 'true');

	const url = params.toString()
		? `${API_BASE}/download-items/${id}?${params}`
		: `${API_BASE}/download-items/${id}`;

	const response = await apiFetch(url, {
		method: 'DELETE',
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

// Import History

export interface ImportHistoryItem {
	id: number;
	downloadId: number | null;
	sourcePath: string;
	destPath: string;
	mediaId: number | null;
	mediaType: string | null;
	success: boolean;
	error: string | null;
	createdAt: string;
}

export async function getImportHistory(limit = 50): Promise<ImportHistoryItem[]> {
	const response = await apiFetch(`${API_BASE}/imports/history?limit=${limit}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Grab History

export interface GrabHistoryItem {
	id: number;
	mediaId: number;
	mediaType: string;
	releaseTitle: string;
	indexerId: number | null;
	indexerName: string | null;
	qualityResolution: string | null;
	qualitySource: string | null;
	qualityCodec: string | null;
	qualityAudio: string | null;
	qualityHdr: string | null;
	releaseGroup: string | null;
	size: number;
	downloadClientId: number | null;
	downloadId: string | null;
	status: 'grabbed' | 'imported' | 'failed';
	errorMessage: string | null;
	grabbedAt: string;
	importedAt: string | null;
}

export async function getGrabHistory(limit = 100): Promise<GrabHistoryItem[]> {
	const response = await apiFetch(`${API_BASE}/grab-history?limit=${limit}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Blocklist

export interface BlocklistEntry {
	id: number;
	mediaId: number | null;
	mediaType: string | null;
	releaseTitle: string;
	releaseGroup: string | null;
	indexerId: number | null;
	reason: string;
	errorMessage: string | null;
	expiresAt: string | null;
	createdAt: string;
}

export async function getBlocklist(): Promise<BlocklistEntry[]> {
	const response = await apiFetch(`${API_BASE}/blocklist`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function addToBlocklist(entry: Omit<BlocklistEntry, 'id' | 'createdAt'>): Promise<BlocklistEntry> {
	const response = await apiFetch(`${API_BASE}/blocklist`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(entry)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function removeFromBlocklist(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/blocklist/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}
