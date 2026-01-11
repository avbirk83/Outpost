import { API_BASE, apiFetch } from './core';

export interface Collection {
	id: number;
	name: string;
	description?: string;
	tmdbCollectionId?: number;
	posterPath?: string;
	backdropPath?: string;
	isAuto: boolean;
	sortOrder: 'release' | 'added' | 'title' | 'custom';
	itemCount: number;
	ownedCount: number;
	createdAt: string;
	updatedAt: string;
}

export interface CollectionItem {
	id: number;
	collectionId: number;
	mediaType: 'movie' | 'show';
	mediaId?: number;
	tmdbId: number;
	title: string;
	year?: number;
	posterPath?: string;
	sortOrder: number;
	inLibrary: boolean;
	addedAt: string;
}

export interface CollectionDetail extends Collection {
	items: CollectionItem[];
}

export async function getCollections(): Promise<Collection[]> {
	const response = await apiFetch(`${API_BASE}/collections`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getCollection(id: number): Promise<CollectionDetail> {
	const response = await apiFetch(`${API_BASE}/collections/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function createCollection(data: {
	name: string;
	description?: string;
}): Promise<Collection> {
	const response = await apiFetch(`${API_BASE}/collections`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function updateCollection(
	id: number,
	data: Partial<{ name: string; description: string; sortOrder: string }>
): Promise<Collection> {
	const response = await apiFetch(`${API_BASE}/collections/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteCollection(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/collections/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function addCollectionItem(
	collectionId: number,
	item: {
		mediaType: 'movie' | 'show';
		tmdbId: number;
		title: string;
		year?: number;
		posterPath?: string;
	}
): Promise<CollectionItem> {
	const response = await apiFetch(`${API_BASE}/collections/${collectionId}/items`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(item)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function removeCollectionItem(
	collectionId: number,
	tmdbId: number,
	mediaType: string
): Promise<void> {
	const response = await apiFetch(
		`${API_BASE}/collections/${collectionId}/items?tmdbId=${tmdbId}&mediaType=${mediaType}`,
		{ method: 'DELETE' }
	);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function reorderCollectionItems(collectionId: number, itemIds: number[]): Promise<void> {
	const response = await apiFetch(`${API_BASE}/collections/${collectionId}/reorder`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ itemIds })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function getMediaCollections(tmdbId: number, mediaType: 'movie' | 'show'): Promise<Collection[]> {
	const response = await apiFetch(`${API_BASE}/collections?tmdbId=${tmdbId}&mediaType=${mediaType}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}
