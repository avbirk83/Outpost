import { API_BASE, apiFetch } from './core';

export interface UpgradeableItem {
	id: number;
	type: 'movie' | 'episode';
	title: string;
	year?: number;
	seasonNumber?: number;
	episodeNumber?: number;
	showTitle?: string;
	currentQuality: string;
	currentScore: number;
	cutoffQuality: string;
	cutoffScore: number;
	posterPath?: string;
	size: number;
	lastSearched?: string;
}

export interface UpgradesSummary {
	movies: UpgradeableItem[];
	episodes: UpgradeableItem[];
	totalCount: number;
	totalSize: number;
}

export async function getUpgrades(): Promise<UpgradesSummary> {
	const response = await apiFetch(`${API_BASE}/upgrades`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function searchUpgrade(
	mediaType: 'movie' | 'episode',
	mediaId: number
): Promise<{ success: boolean; message: string }> {
	const response = await apiFetch(`${API_BASE}/upgrades/search`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ mediaType, mediaId })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function searchAllUpgrades(
	limit?: number,
	mediaType?: 'movie' | 'episode'
): Promise<{ success: boolean; queued: number; message: string }> {
	const response = await apiFetch(`${API_BASE}/upgrades/search-all`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ limit: limit || 10, mediaType: mediaType || '' })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Helper to format quality comparison
export function formatQualityComparison(item: UpgradeableItem): string {
	return `${item.currentQuality} → ${item.cutoffQuality}`;
}

// Helper to format score difference
export function getScoreDifference(item: UpgradeableItem): number {
	return item.cutoffScore - item.currentScore;
}
