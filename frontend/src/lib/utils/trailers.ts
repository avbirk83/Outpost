/**
 * Shared trailer utilities
 * Available: parseTrailers(), getOfficialTrailer(), Trailer type
 */

export interface Trailer {
	key: string;
	name: string;
	type: string;
	site?: string;
	official?: boolean;
}

/**
 * Parse trailers JSON string from API response
 */
export function parseTrailers(trailersJson?: string): Trailer[] {
	if (!trailersJson) return [];
	try {
		return JSON.parse(trailersJson);
	} catch {
		return [];
	}
}

/**
 * Get the best official trailer from a list
 * Prioritizes: official YouTube trailers, then any trailer
 */
export function getOfficialTrailer(trailersJson?: string): Trailer | undefined {
	const trailers = parseTrailers(trailersJson);
	if (trailers.length === 0) return undefined;

	// Find official YouTube trailer
	const official = trailers.find(
		t => t.type === 'Trailer' && (t.site === 'YouTube' || !t.site) && t.official !== false
	);

	return official || trailers[0];
}

/**
 * Get YouTube embed URL for a trailer
 */
export function getTrailerEmbedUrl(trailer: Trailer, autoplay = true): string {
	return `https://www.youtube.com/embed/${trailer.key}${autoplay ? '?autoplay=1' : ''}`;
}
