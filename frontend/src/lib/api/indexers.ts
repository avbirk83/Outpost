import { API_BASE, apiFetch } from './core';
import type { TestConnectionResult } from './downloads';

// Indexer types

export interface Indexer {
	id: number;
	name: string;
	type: 'torznab' | 'newznab' | 'prowlarr';
	url: string;
	apiKey?: string;
	categories?: string;
	priority: number;
	enabled: boolean;
	prowlarrId?: number;
	syncedFromProwlarr?: boolean;
	protocol?: string;
	supportsMovies?: boolean;
	supportsTV?: boolean;
	supportsMusic?: boolean;
	supportsBooks?: boolean;
	supportsAnime?: boolean;
	supportsImdb?: boolean;
	supportsTmdb?: boolean;
	supportsTvdb?: boolean;
}

export interface ProwlarrConfig {
	id?: number;
	url: string;
	apiKey?: string;
	autoSync: boolean;
	syncIntervalHours: number;
	lastSync?: string;
	createdAt?: string;
}

export interface IndexerTag {
	id: number;
	prowlarrId: number;
	name: string;
	indexerCount?: number;
}

export interface IndexerCapabilities {
	searchAvailable: boolean;
	movieSearchAvailable: boolean;
	tvSearchAvailable: boolean;
	musicSearchAvailable: boolean;
	bookSearchAvailable: boolean;
	categories: { id: number; name: string }[];
	supportsImdbSearch: boolean;
	supportsTvdbSearch: boolean;
	supportsTmdbSearch: boolean;
}

// Search types

export interface SearchResult {
	indexerId: number;
	indexerName: string;
	indexerType: string;
	title: string;
	guid: string;
	link: string;
	magnetLink?: string;
	size: number;
	seeders: number;
	leechers: number;
	publishDate: string;
	category: string;
	categoryId: string;
	imdbId?: string;
	tvdbId?: string;
	infoUrl?: string;
}

export interface CustomFormatHit {
	name: string;
	score: number;
}

export interface ScoredSearchResult extends SearchResult {
	quality?: string;
	resolution?: string;
	source?: string;
	codec?: string;
	audioCodec?: string;
	audioFeature?: string;
	hdr?: string[];
	releaseGroup?: string;
	proper?: boolean;
	repack?: boolean;
	baseScore: number;
	customFormatHits?: CustomFormatHit[];
	totalScore: number;
	rejected: boolean;
	rejectionReason?: string;
}

export interface SearchParams {
	q?: string;
	type?: 'movie' | 'tvsearch' | 'music' | 'book' | 'search';
	imdbId?: string;
	tvdbId?: string;
	tmdbId?: string;
	season?: number;
	episode?: number;
	categories?: number[];
	limit?: number;
	profileId?: number;
}

export interface GrabParams {
	link: string;
	magnetLink?: string;
	indexerType: string;
	category?: string;
}

// Indexer API functions

export async function getIndexers(): Promise<Indexer[]> {
	const response = await apiFetch(`${API_BASE}/indexers`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createIndexer(indexer: Omit<Indexer, 'id'>): Promise<Indexer> {
	const response = await apiFetch(`${API_BASE}/indexers`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(indexer)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateIndexer(id: number, indexer: Partial<Indexer>): Promise<Indexer> {
	const response = await apiFetch(`${API_BASE}/indexers/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(indexer)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteIndexer(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/indexers/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function testIndexer(id: number): Promise<TestConnectionResult> {
	const response = await apiFetch(`${API_BASE}/indexers/${id}/test`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getIndexerCapabilities(id: number): Promise<IndexerCapabilities> {
	const response = await apiFetch(`${API_BASE}/indexers/${id}/capabilities`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Prowlarr functions

export async function getProwlarrConfig(): Promise<ProwlarrConfig | null> {
	const response = await apiFetch(`${API_BASE}/prowlarr/config`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	const data = await response.json();
	return data.url ? data : null;
}

export async function saveProwlarrConfig(config: Partial<ProwlarrConfig>): Promise<ProwlarrConfig> {
	const response = await apiFetch(`${API_BASE}/prowlarr/config`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(config)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export interface ProwlarrTestResult {
	success: boolean;
	error?: string;
	indexerCount?: number;
}

export async function testProwlarrConnection(url: string, apiKey: string): Promise<ProwlarrTestResult> {
	const response = await apiFetch(`${API_BASE}/prowlarr/test`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ url, apiKey })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export interface ProwlarrSyncResult {
	success: boolean;
	error?: string;
	synced?: number;
	indexers?: Indexer[];
}

export async function syncProwlarr(): Promise<ProwlarrSyncResult> {
	const response = await apiFetch(`${API_BASE}/prowlarr/sync`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getIndexerTags(): Promise<IndexerTag[]> {
	const response = await apiFetch(`${API_BASE}/indexer-tags`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Search functions

export async function searchIndexers(params: SearchParams): Promise<SearchResult[]> {
	const searchParams = new URLSearchParams();
	if (params.q) searchParams.set('q', params.q);
	if (params.type) searchParams.set('type', params.type);
	if (params.imdbId) searchParams.set('imdbId', params.imdbId);
	if (params.tvdbId) searchParams.set('tvdbId', params.tvdbId);
	if (params.tmdbId) searchParams.set('tmdbId', params.tmdbId);
	if (params.season) searchParams.set('season', params.season.toString());
	if (params.episode) searchParams.set('episode', params.episode.toString());
	if (params.categories?.length) searchParams.set('categories', params.categories.join(','));
	if (params.limit) searchParams.set('limit', params.limit.toString());

	const response = await apiFetch(`${API_BASE}/search?${searchParams}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function searchIndexersScored(params: SearchParams): Promise<ScoredSearchResult[]> {
	const searchParams = new URLSearchParams();
	if (params.q) searchParams.set('q', params.q);
	if (params.type) searchParams.set('type', params.type);
	if (params.imdbId) searchParams.set('imdbId', params.imdbId);
	if (params.tvdbId) searchParams.set('tvdbId', params.tvdbId);
	if (params.tmdbId) searchParams.set('tmdbId', params.tmdbId);
	if (params.season) searchParams.set('season', params.season.toString());
	if (params.episode) searchParams.set('episode', params.episode.toString());
	if (params.categories?.length) searchParams.set('categories', params.categories.join(','));
	if (params.limit) searchParams.set('limit', params.limit.toString());
	if (params.profileId) searchParams.set('profileId', params.profileId.toString());

	const response = await apiFetch(`${API_BASE}/search/scored?${searchParams}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function grabRelease(params: GrabParams): Promise<{ success: boolean; message: string; client?: string }> {
	const response = await apiFetch(`${API_BASE}/grab`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(params)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}
