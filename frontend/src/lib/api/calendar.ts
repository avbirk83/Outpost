import { API_BASE, apiFetch } from './core';

export interface CalendarItem {
	date: string;        // YYYY-MM-DD
	type: 'episode' | 'movie';
	title: string;
	subtitle: string;
	tmdbId: number;
	mediaId: number | null;
	posterPath: string | null;
	inLibrary: boolean;
	isWanted: boolean;
	airTime?: string;
}

export type CalendarFilter = 'all' | 'movies' | 'tv' | 'library' | 'wanted';

export async function getCalendarItems(
	start: string,
	end: string,
	filter: CalendarFilter = 'all'
): Promise<CalendarItem[]> {
	const params = new URLSearchParams({ start, end, filter });
	const response = await apiFetch(`${API_BASE}/calendar?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}
