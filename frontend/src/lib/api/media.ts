import { API_BASE, apiFetch } from './core';

// Movie types

export interface Movie {
	id: number;
	libraryId: number;
	tmdbId?: number;
	imdbId?: string;
	title: string;
	originalTitle?: string;
	year: number;
	overview?: string;
	tagline?: string;
	runtime?: number;
	rating?: number;
	contentRating?: string;
	genres?: string;
	cast?: string;
	crew?: string;
	director?: string;
	writer?: string;
	editor?: string;
	producers?: string;
	status?: string;
	budget?: number;
	revenue?: number;
	country?: string;
	originalLanguage?: string;
	trailers?: string;
	posterPath?: string;
	backdropPath?: string;
	focalX?: number;
	focalY?: number;
	path: string;
	size: number;
	addedAt: string;
	watchState?: 'unwatched' | 'partial' | 'watched';
	progress?: number;
}

// Show types

export interface Show {
	id: number;
	libraryId: number;
	tmdbId?: number;
	tvdbId?: number;
	imdbId?: string;
	title: string;
	originalTitle?: string;
	year: number;
	overview?: string;
	status?: string;
	rating?: number;
	contentRating?: string;
	genres?: string;
	cast?: string;
	crew?: string;
	network?: string;
	posterPath?: string;
	backdropPath?: string;
	focalX?: number;
	focalY?: number;
	path: string;
	addedAt?: string;
	watchState?: 'unwatched' | 'partial' | 'watched';
	watchedEpisodes?: number;
	totalEpisodes?: number;
}

export interface Season {
	id: number;
	showId: number;
	seasonNumber: number;
	name?: string;
	overview?: string;
	posterPath?: string;
	airDate?: string;
	episodes: Episode[];
}

export interface Episode {
	id: number;
	seasonId: number;
	episodeNumber: number;
	title: string;
	overview?: string;
	airDate?: string;
	runtime?: number;
	stillPath?: string;
	path: string;
	size: number;
}

export interface ShowDetail extends Show {
	seasons: Season[];
}

// Music types

export interface Artist {
	id: number;
	libraryId: number;
	musicBrainzId?: string;
	name: string;
	sortName?: string;
	overview?: string;
	imagePath?: string;
	path: string;
}

export interface Album {
	id: number;
	artistId: number;
	musicBrainzId?: string;
	title: string;
	year: number;
	overview?: string;
	coverPath?: string;
	path: string;
}

export interface Track {
	id: number;
	albumId: number;
	musicBrainzId?: string;
	title: string;
	trackNumber: number;
	discNumber: number;
	duration: number;
	path: string;
	size: number;
}

export interface ArtistDetail extends Artist {
	albums: Album[];
}

export interface AlbumDetail extends Album {
	artist: Artist;
	tracks: Track[];
}

// Book types

export interface Book {
	id: number;
	libraryId: number;
	title: string;
	author?: string;
	isbn?: string;
	publisher?: string;
	year: number;
	description?: string;
	coverPath?: string;
	format: string;
	path: string;
	size: number;
	addedAt: string;
}

// Movie API functions

export async function getMovies(): Promise<Movie[]> {
	const response = await apiFetch(`${API_BASE}/movies`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getMovie(id: number): Promise<Movie> {
	const response = await apiFetch(`${API_BASE}/movies/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function refreshMovieMetadata(id: number): Promise<Movie> {
	const response = await apiFetch(`${API_BASE}/movies/${id}/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function matchMovie(id: number, tmdbId: number): Promise<Movie> {
	const response = await apiFetch(`${API_BASE}/movies/${id}/match`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ tmdbId })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Show API functions

export async function getShows(): Promise<Show[]> {
	const response = await apiFetch(`${API_BASE}/shows`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getShow(id: number): Promise<ShowDetail> {
	const response = await apiFetch(`${API_BASE}/shows/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function refreshShowMetadata(id: number): Promise<ShowDetail> {
	const response = await apiFetch(`${API_BASE}/shows/${id}/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function matchShow(id: number, tmdbId: number): Promise<ShowDetail> {
	const response = await apiFetch(`${API_BASE}/shows/${id}/match`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ tmdbId })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteEpisode(episodeId: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/episodes/${episodeId}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

// Music API functions

export async function getArtists(): Promise<Artist[]> {
	const response = await apiFetch(`${API_BASE}/artists`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getArtist(id: number): Promise<ArtistDetail> {
	const response = await apiFetch(`${API_BASE}/artists/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getAlbums(): Promise<Album[]> {
	const response = await apiFetch(`${API_BASE}/albums`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getAlbum(id: number): Promise<AlbumDetail> {
	const response = await apiFetch(`${API_BASE}/albums/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTrack(id: number): Promise<Track> {
	const response = await apiFetch(`${API_BASE}/tracks/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Book API functions

export async function getBooks(): Promise<Book[]> {
	const response = await apiFetch(`${API_BASE}/books`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getBook(id: number): Promise<Book> {
	const response = await apiFetch(`${API_BASE}/books/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}
