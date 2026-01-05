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

// Download Tracking

export interface DownloadItem {
	id: number;
	downloadClientId: number | null;
	externalId: string;
	mediaId: number | null;
	mediaType: string | null;
	title: string;
	size: number;
	status: 'downloading' | 'completed' | 'importing' | 'imported' | 'failed' | 'unmatched';
	progress: number;
	downloadPath: string | null;
	importedPath: string | null;
	error: string | null;
	createdAt: string;
	updatedAt: string;
}

export async function getDownloadItems(): Promise<DownloadItem[]> {
	const response = await apiFetch(`${API_BASE}/download-items`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteDownloadItem(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/download-items/${id}`, {
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
