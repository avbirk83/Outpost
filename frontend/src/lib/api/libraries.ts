import { API_BASE, apiFetch } from './core';

export interface Library {
	id: number;
	name: string;
	path: string;
	type: 'movies' | 'tv' | 'anime' | 'music' | 'books';
	scanInterval: number;
}

export interface ScanProgress {
	scanning: boolean;
	library: string;
	phase: string;
	current: number;
	total: number;
	percent: number;
	lastLibrary?: string;
	lastAdded: number;
	lastSkipped: number;
	lastErrors: number;
	lastScanAt?: string;
}

export async function getLibraries(): Promise<Library[]> {
	const response = await apiFetch(`${API_BASE}/libraries`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createLibrary(library: Omit<Library, 'id'>): Promise<Library> {
	const response = await apiFetch(`${API_BASE}/libraries`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(library)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteLibrary(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/libraries/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function scanLibrary(id: number): Promise<{ status: string; message: string }> {
	const response = await apiFetch(`${API_BASE}/libraries/${id}/scan`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getScanProgress(): Promise<ScanProgress> {
	const response = await apiFetch(`${API_BASE}/scan/progress`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}
