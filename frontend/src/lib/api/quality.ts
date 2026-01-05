import { API_BASE, apiFetch } from './core';

// Quality Profile types

export interface QualityProfile {
	id: number;
	name: string;
	upgradeAllowed: boolean;
	upgradeUntilScore: number;
	minFormatScore: number;
	cutoffFormatScore: number;
	qualities: string; // JSON array
	customFormats: string; // JSON object of format_id -> score
}

export interface CustomFormat {
	id: number;
	name: string;
	conditions: string; // JSON array
}

export interface ParsedRelease {
	title: string;
	year?: number;
	resolution?: string;
	source?: string;
	codec?: string;
	audioCodec?: string;
	audioFeature?: string;
	hdr?: string[];
	releaseGroup?: string;
	proper?: boolean;
	repack?: boolean;
	edition?: string;
	season?: number;
	episode?: number;
	quality?: string;
}

// Quality Profile API functions

export async function getQualityProfiles(): Promise<QualityProfile[]> {
	const response = await apiFetch(`${API_BASE}/quality-profiles`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createQualityProfile(profile: Omit<QualityProfile, 'id'>): Promise<QualityProfile> {
	const response = await apiFetch(`${API_BASE}/quality-profiles`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(profile)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateQualityProfile(id: number, profile: Partial<QualityProfile>): Promise<QualityProfile> {
	const response = await apiFetch(`${API_BASE}/quality-profiles/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(profile)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteQualityProfile(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/quality-profiles/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Custom Format API functions

export async function getCustomFormats(): Promise<CustomFormat[]> {
	const response = await apiFetch(`${API_BASE}/custom-formats`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createCustomFormat(format: Omit<CustomFormat, 'id'>): Promise<CustomFormat> {
	const response = await apiFetch(`${API_BASE}/custom-formats`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(format)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateCustomFormat(id: number, format: Partial<CustomFormat>): Promise<CustomFormat> {
	const response = await apiFetch(`${API_BASE}/custom-formats/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(format)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteCustomFormat(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/custom-formats/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function parseReleaseName(name: string): Promise<ParsedRelease> {
	const response = await apiFetch(`${API_BASE}/releases/parse`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Quality Preset types and functions

export interface QualityPreset {
	id: number;
	name: string;
	isDefault: boolean;
	isBuiltIn: boolean;
	enabled: boolean;
	priority: number;
	resolution: string;
	source: string;
	hdrFormats: string | null;
	codec: string;
	audioFormats: string | null;
	preferredEdition: string;
	minSeeders: number;
	preferSeasonPacks: boolean;
	autoUpgrade: boolean;
	createdAt: string;
	updatedAt: string;
}

export async function getQualityPresets(): Promise<QualityPreset[]> {
	const response = await apiFetch(`${API_BASE}/quality/presets`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getQualityPreset(id: number): Promise<QualityPreset> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function createQualityPreset(preset: Partial<QualityPreset>): Promise<QualityPreset> {
	const response = await apiFetch(`${API_BASE}/quality/presets`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(preset),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function updateQualityPreset(id: number, preset: Partial<QualityPreset>): Promise<QualityPreset> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(preset),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteQualityPreset(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}`, {
		method: 'DELETE',
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function toggleQualityPresetEnabled(id: number, enabled: boolean): Promise<void> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}/toggle`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ enabled }),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function updateQualityPresetPriority(id: number, priority: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}/priority`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ priority }),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function setDefaultQualityPreset(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/quality/presets/${id}/default`, {
		method: 'POST',
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}
