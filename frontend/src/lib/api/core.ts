export const API_BASE = '/api';

// Helper to ensure credentials are always included
export async function apiFetch(url: string, options: RequestInit = {}): Promise<Response> {
	return fetch(url, {
		...options,
		credentials: 'include'
	});
}

export interface HealthResponse {
	status: string;
}

export async function getHealth(): Promise<HealthResponse> {
	const response = await apiFetch(`${API_BASE}/health`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Image URL helper - for library items (served through our API)
export function getImageUrl(path: string | undefined): string | undefined {
	if (!path) return undefined;
	if (path.startsWith('/images/')) {
		return path;
	}
	const cleanPath = path.startsWith('/') ? path : `/${path}`;
	return `/images${cleanPath}`;
}

// TMDB image URL helper
export function getTmdbImageUrl(path: string | undefined, size = 'w500'): string {
	if (!path) return '';
	return `https://image.tmdb.org/t/p/${size}${path}`;
}
