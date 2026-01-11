import { API_BASE, apiFetch } from './core';

export interface RuleCondition {
	field: string;
	operator: 'eq' | 'gte' | 'lte' | 'contains' | 'not_contains' | 'within';
	value: string | number | boolean;
}

export interface PlaylistRules {
	match: 'all' | 'any';
	conditions: RuleCondition[];
}

export interface SmartPlaylist {
	id: number;
	userId?: number;
	name: string;
	description?: string;
	rules: string;
	sortBy: 'added' | 'title' | 'year' | 'rating' | 'runtime';
	sortOrder: 'asc' | 'desc';
	limitCount?: number;
	mediaType: 'movies' | 'shows' | 'both';
	autoRefresh: boolean;
	isSystem: boolean;
	itemCount?: number;
	lastRefreshed?: string;
	createdAt: string;
}

export interface SmartPlaylistItem {
	id: number;
	mediaType: 'movie' | 'show';
	title: string;
	year?: number;
	posterPath?: string;
	rating?: number;
	runtime?: number;
	addedAt?: string;
}

export interface SmartPlaylistDetail extends SmartPlaylist {
	items: SmartPlaylistItem[];
}

// Supported fields for rule conditions
export const RULE_FIELDS = [
	{ value: 'genre', label: 'Genre', type: 'text' },
	{ value: 'year', label: 'Year', type: 'number' },
	{ value: 'rating', label: 'Rating', type: 'number' },
	{ value: 'runtime', label: 'Runtime (min)', type: 'number' },
	{ value: 'resolution', label: 'Resolution', type: 'select', options: ['2160p', '1080p', '720p', '480p'] },
	{ value: 'codec', label: 'Codec', type: 'select', options: ['hevc', 'h264', 'av1', 'mpeg4'] },
	{ value: 'added', label: 'Added', type: 'duration' },
	{ value: 'watched', label: 'Watched', type: 'boolean' },
	{ value: 'library', label: 'Library', type: 'number' },
	{ value: 'collection', label: 'Collection', type: 'number' },
	{ value: 'actor', label: 'Actor', type: 'text' },
	{ value: 'director', label: 'Director', type: 'text' },
	{ value: 'studio', label: 'Studio', type: 'text' }
] as const;

// Operators for different field types
export const OPERATORS = {
	text: [
		{ value: 'contains', label: 'Contains' },
		{ value: 'not_contains', label: 'Does not contain' },
		{ value: 'eq', label: 'Equals' }
	],
	number: [
		{ value: 'eq', label: 'Equals' },
		{ value: 'gte', label: 'Greater than or equal' },
		{ value: 'lte', label: 'Less than or equal' }
	],
	select: [
		{ value: 'eq', label: 'Is' },
		{ value: 'contains', label: 'Contains' }
	],
	duration: [{ value: 'within', label: 'Within last' }],
	boolean: [{ value: 'eq', label: 'Is' }]
} as const;

export async function getSmartPlaylists(): Promise<SmartPlaylist[]> {
	const response = await apiFetch(`${API_BASE}/smart-playlists`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getSmartPlaylist(id: number): Promise<SmartPlaylistDetail> {
	const response = await apiFetch(`${API_BASE}/smart-playlists/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function createSmartPlaylist(data: {
	name: string;
	description?: string;
	rules: string;
	sortBy?: string;
	sortOrder?: string;
	limitCount?: number;
	mediaType?: string;
	autoRefresh?: boolean;
}): Promise<SmartPlaylist> {
	const response = await apiFetch(`${API_BASE}/smart-playlists`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function updateSmartPlaylist(
	id: number,
	data: Partial<{
		name: string;
		description: string;
		rules: string;
		sortBy: string;
		sortOrder: string;
		limitCount: number;
		mediaType: string;
		autoRefresh: boolean;
	}>
): Promise<SmartPlaylist> {
	const response = await apiFetch(`${API_BASE}/smart-playlists/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteSmartPlaylist(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/smart-playlists/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function refreshSmartPlaylist(
	id: number
): Promise<{ items: SmartPlaylistItem[]; itemCount: number }> {
	const response = await apiFetch(`${API_BASE}/smart-playlists/${id}/refresh`, {
		method: 'POST'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function previewSmartPlaylist(data: {
	rules: string;
	sortBy?: string;
	sortOrder?: string;
	limitCount?: number;
	mediaType?: string;
}): Promise<{ items: SmartPlaylistItem[]; itemCount: number }> {
	const response = await apiFetch(`${API_BASE}/smart-playlists/preview`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Helper to parse rules JSON safely
export function parseRules(rulesJson: string): PlaylistRules {
	try {
		return JSON.parse(rulesJson);
	} catch {
		return { match: 'all', conditions: [] };
	}
}

// Helper to stringify rules
export function stringifyRules(rules: PlaylistRules): string {
	return JSON.stringify(rules);
}
