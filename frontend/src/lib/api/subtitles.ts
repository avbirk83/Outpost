import { API_BASE, apiFetch } from './core';

export interface SubtitleResult {
	id: string;
	fileId: number;
	language: string;
	languageName?: string;
	fileName: string;
	release: string;
	uploadDate: string;
	downloads: number;
	fps: number;
	hearingImpaired: boolean;
	aiTranslated: boolean;
	fromTrusted: boolean;
	featureTitle: string;
	featureYear: number;
}

export interface SubtitleLanguage {
	language_code: string;
	language_name: string;
}

export interface SubtitleSearchParams {
	query?: string;
	tmdbId?: number;
	imdbId?: string;
	year?: number;
	season?: number;
	episode?: number;
	languages?: string[];
	hearingImpaired?: boolean | null;
}

export interface SubtitleDownloadParams {
	fileId: number;
	mediaType: 'movie' | 'episode';
	mediaId: number;
	language: string;
	episodeId?: number;
}

export interface SubtitleDownloadResult {
	success: boolean;
	path: string;
	message: string;
	remaining: number;
}

export async function searchSubtitles(params: SubtitleSearchParams): Promise<SubtitleResult[]> {
	const response = await apiFetch(`${API_BASE}/opensubtitles/search`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(params)
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

export async function downloadSubtitle(params: SubtitleDownloadParams): Promise<SubtitleDownloadResult> {
	const response = await apiFetch(`${API_BASE}/opensubtitles/download`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(params)
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

export async function getSubtitleLanguages(): Promise<SubtitleLanguage[]> {
	const response = await apiFetch(`${API_BASE}/opensubtitles/languages`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function testOpenSubtitlesConnection(apiKey?: string): Promise<{ success: boolean; error?: string; message?: string }> {
	const response = await apiFetch(`${API_BASE}/opensubtitles/test`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ apiKey: apiKey || '' })
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Common subtitle languages for the UI selector
export const COMMON_LANGUAGES = [
	{ code: 'en', name: 'English' },
	{ code: 'es', name: 'Spanish' },
	{ code: 'fr', name: 'French' },
	{ code: 'de', name: 'German' },
	{ code: 'it', name: 'Italian' },
	{ code: 'pt', name: 'Portuguese' },
	{ code: 'nl', name: 'Dutch' },
	{ code: 'pl', name: 'Polish' },
	{ code: 'ru', name: 'Russian' },
	{ code: 'ja', name: 'Japanese' },
	{ code: 'ko', name: 'Korean' },
	{ code: 'zh', name: 'Chinese' },
	{ code: 'ar', name: 'Arabic' },
	{ code: 'hi', name: 'Hindi' },
	{ code: 'sv', name: 'Swedish' },
	{ code: 'da', name: 'Danish' },
	{ code: 'fi', name: 'Finnish' },
	{ code: 'no', name: 'Norwegian' },
	{ code: 'tr', name: 'Turkish' },
	{ code: 'el', name: 'Greek' }
];

// Helper to format upload date
export function formatSubtitleDate(dateStr: string): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	return date.toLocaleDateString();
}

// Helper to format download count
export function formatDownloads(count: number): string {
	if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
	if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
	return count.toString();
}
