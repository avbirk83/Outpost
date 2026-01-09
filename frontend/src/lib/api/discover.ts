import { API_BASE, apiFetch } from './core';
import type { ScoredSearchResult } from './indexers';

// TMDB Search types

export interface TmdbMovieResult {
	id: number;
	title: string;
	original_title: string;
	overview: string;
	release_date: string;
	poster_path: string;
	vote_average: number;
}

export interface TmdbTVResult {
	id: number;
	name: string;
	original_name: string;
	overview: string;
	first_air_date: string;
	poster_path: string;
	vote_average: number;
}

export async function searchTmdbMovies(query: string, year?: number): Promise<TmdbMovieResult[]> {
	const params = new URLSearchParams({ q: query });
	if (year) params.set('year', year.toString());
	const response = await apiFetch(`${API_BASE}/tmdb/search/movie?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function searchTmdbTV(query: string, year?: number): Promise<TmdbTVResult[]> {
	const params = new URLSearchParams({ q: query });
	if (year) params.set('year', year.toString());
	const response = await apiFetch(`${API_BASE}/tmdb/search/tv?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Discover types

export interface DiscoverItem {
	id: number;
	type: 'movie' | 'show';
	mediaType?: 'movie' | 'tv';
	title: string;
	name?: string;
	overview: string;
	releaseDate: string;
	firstAirDate?: string;
	posterPath: string;
	backdropPath: string;
	rating: number;
	voteAverage?: number;
	popularity: number;
	focalX?: number;
	focalY?: number;
	inLibrary?: boolean;
	libraryId?: number;
	requested?: boolean;
	requestId?: number;
	requestStatus?: string;
}

export interface DiscoverResult {
	page: number;
	totalPages: number;
	totalResults: number;
	results: DiscoverItem[];
}

export interface CastMember {
	id: number;
	name: string;
	character: string;
	photo: string;
}

export interface CrewMember {
	id: number;
	name: string;
	job: string;
	photo: string | null;
}

export interface RecommendedItem {
	id: number;
	title: string;
	posterPath: string | null;
	releaseDate: string;
	rating: number;
	mediaType: 'movie' | 'tv';
	runtime?: number;
	contentRating?: string;
}

export interface SeasonSummary {
	season_number: number;
	name: string;
	overview?: string;
	poster_path?: string;
	air_date?: string;
	episode_count: number;
}

export interface DiscoverMovieDetail {
	id: number;
	title: string;
	overview: string;
	tagline: string;
	releaseDate: string;
	runtime: number;
	rating: number;
	contentRating?: string;
	posterPath: string;
	backdropPath: string;
	genres: string[];
	cast: CastMember[];
	crew: CrewMember[];
	director: string;
	imdbId?: string;
	status: string;
	budget?: number;
	revenue?: number;
	originalLanguage?: string;
	productionCountries?: string[];
	productionCompanies?: string[];
	trailerKey?: string;
	recommendations?: RecommendedItem[];
}

export interface DiscoverShowDetail {
	id: number;
	title: string;
	overview: string;
	firstAirDate: string;
	status: string;
	rating: number;
	contentRating?: string;
	posterPath: string;
	backdropPath: string;
	genres: string[];
	networks: string[];
	seasons: number;
	episodes: number;
	cast: CastMember[];
	crew: CrewMember[];
	imdbId?: string;
	originalLanguage?: string;
	productionCountries?: string[];
	trailerKey?: string;
	recommendations?: RecommendedItem[];
	seasonDetails?: SeasonSummary[];
}

export interface DiscoverMovieDetailWithStatus extends DiscoverMovieDetail {
	inLibrary: boolean;
	libraryId?: number;
	requested: boolean;
	requestId?: number;
	requestStatus?: string;
}

export interface DiscoverShowDetailWithStatus extends DiscoverShowDetail {
	inLibrary: boolean;
	libraryId?: number;
	requested: boolean;
	requestId?: number;
	requestStatus?: string;
}

// Discover API functions

export async function getTrendingMovies(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/movies/trending?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getPopularMovies(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/movies/popular?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getUpcomingMovies(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/movies/upcoming?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTheatricalReleases(region = '', page = 1): Promise<DiscoverResult> {
	const params = new URLSearchParams({ page: page.toString() });
	if (region) params.append('region', region);
	const response = await apiFetch(`${API_BASE}/discover/movies/theatrical?${params}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTopRatedMovies(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/movies/top-rated?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTrendingShows(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/shows/trending?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getPopularShows(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/shows/popular?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getUpcomingShows(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/shows/upcoming?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTopRatedShows(page = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/shows/top-rated?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getDiscoverMovieDetail(id: number): Promise<DiscoverMovieDetail> {
	const response = await apiFetch(`${API_BASE}/discover/movie/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getDiscoverShowDetail(id: number): Promise<DiscoverShowDetail> {
	const response = await apiFetch(`${API_BASE}/discover/show/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getDiscoverMovieDetailWithStatus(id: number): Promise<DiscoverMovieDetailWithStatus> {
	const response = await apiFetch(`${API_BASE}/discover/movie/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getDiscoverShowDetailWithStatus(id: number): Promise<DiscoverShowDetailWithStatus> {
	const response = await apiFetch(`${API_BASE}/discover/show/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Trailer types

export interface TrailerInfo {
	key: string;
	name: string;
	type: string;
	site?: string;
	official?: boolean;
}

export async function getTrailers(tmdbId: number, mediaType: 'movie' | 'tv'): Promise<TrailerInfo[]> {
	const response = await apiFetch(`${API_BASE}/trailers/${mediaType}/${tmdbId}`);
	if (!response.ok) throw new Error('Failed to fetch trailers');
	return response.json();
}

// Recommendations

export interface TMDBMovieResult {
	id: number;
	title: string;
	overview: string;
	release_date: string;
	poster_path: string | null;
	backdrop_path: string | null;
	vote_average: number;
}

export interface TMDBMovieSearchResult {
	page: number;
	results: TMDBMovieResult[];
	total_pages: number;
	total_results: number;
}

export interface TMDBShowResult {
	id: number;
	name: string;
	overview: string;
	first_air_date: string;
	poster_path: string | null;
	backdrop_path: string | null;
	vote_average: number;
}

export async function getMovieRecommendations(tmdbId: number): Promise<TMDBMovieSearchResult> {
	const response = await apiFetch(`${API_BASE}/movie/recommendations/${tmdbId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getMovieSuggestions(movieId: number): Promise<{ results: TMDBMovieResult[] }> {
	const response = await apiFetch(`${API_BASE}/movies/suggestions/${movieId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getShowSuggestions(showId: number): Promise<{ results: TMDBShowResult[] }> {
	const response = await apiFetch(`${API_BASE}/shows/suggestions/${showId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Genres

export interface Genre {
	id: number;
	name: string;
}

export async function getMovieGenres(): Promise<{ genres: Genre[] }> {
	const response = await apiFetch(`${API_BASE}/discover/genres/movie`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTVGenres(): Promise<{ genres: Genre[] }> {
	const response = await apiFetch(`${API_BASE}/discover/genres/tv`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getMoviesByGenre(genreId: number, page: number = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/movies/genre/${genreId}?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTVByGenre(genreId: number, page: number = 1): Promise<DiscoverResult> {
	const response = await apiFetch(`${API_BASE}/discover/shows/genre/${genreId}?page=${page}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Requests

export interface Request {
	id: number;
	userId: number;
	username?: string;
	type: 'movie' | 'show';
	tmdbId: number;
	title: string;
	year?: number;
	overview?: string;
	posterPath?: string;
	backdropPath?: string;
	qualityProfileId?: number;
	qualityPresetId?: number;
	status: 'requested' | 'approved' | 'denied' | 'available';
	statusReason?: string;
	requestedAt: string;
	updatedAt: string;
}

export async function getRequests(status?: string): Promise<Request[]> {
	const url = status
		? `${API_BASE}/requests?status=${status}`
		: `${API_BASE}/requests`;
	const response = await apiFetch(url);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function createRequest(request: {
	type: 'movie' | 'show';
	tmdbId: number;
	title: string;
	year?: number;
	overview?: string;
	posterPath?: string;
	backdropPath?: string;
	qualityProfileId?: number;
	qualityPresetId?: number;
}): Promise<Request> {
	const response = await apiFetch(`${API_BASE}/requests`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
	if (!response.ok) {
		if (response.status === 409) throw new Error('Already requested');
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateRequest(id: number, status: string, statusReason?: string, qualityProfileId?: number): Promise<Request> {
	const response = await apiFetch(`${API_BASE}/requests/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ status, statusReason, qualityProfileId })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteRequest(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/requests/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

// Wanted/Monitoring

export interface WantedItem {
	id: number;
	type: 'movie' | 'show';
	tmdbId: number;
	title: string;
	year?: number;
	posterPath?: string;
	qualityProfileId: number;
	monitored: boolean;
	seasons?: string;
	lastSearched?: string;
	addedAt: string;
}

export async function getWantedItems(): Promise<WantedItem[]> {
	const response = await apiFetch(`${API_BASE}/wanted`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getWantedItem(id: number): Promise<WantedItem> {
	const response = await apiFetch(`${API_BASE}/wanted/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function createWantedItem(item: Omit<WantedItem, 'id' | 'addedAt'>): Promise<WantedItem> {
	const response = await apiFetch(`${API_BASE}/wanted`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(item)
	});
	if (!response.ok) {
		if (response.status === 409) throw new Error('Item already in wanted list');
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateWantedItem(id: number, updates: Partial<WantedItem>): Promise<WantedItem> {
	const response = await apiFetch(`${API_BASE}/wanted/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(updates)
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function deleteWantedItem(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/wanted/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function searchWantedItem(id: number): Promise<ScoredSearchResult[]> {
	const response = await apiFetch(`${API_BASE}/wanted/search/${id}`, {
		method: 'POST'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Person

export interface PersonCredit {
	id: number;
	mediaType: 'movie' | 'tv';
	title: string;
	character?: string;
	job?: string;
	posterPath: string | null;
	releaseDate: string | null;
	voteAverage: number;
}

export interface LibraryAppearance {
	id: number;
	type: 'movie' | 'show';
	title: string;
	year: number;
	posterPath: string | null;
}

export interface PersonDetail {
	id: number;
	name: string;
	biography: string;
	birthday: string | null;
	deathday: string | null;
	placeOfBirth: string | null;
	profilePath: string | null;
	knownFor: string;
	credits: PersonCredit[];
	alsoInLibrary: LibraryAppearance[];
}

export async function getPersonDetail(personId: number): Promise<PersonDetail> {
	const response = await apiFetch(`${API_BASE}/person/${personId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Watchlist

export interface WatchlistItem {
	id: number;
	userId: number;
	tmdbId: number;
	mediaType: 'movie' | 'tv';
	addedAt: string;
	title: string;
	posterPath: string | null;
	backdropPath: string | null;
	year: number;
	inLibrary: boolean;
	libraryId?: number;
	progress?: number;
}

export async function getWatchlist(): Promise<WatchlistItem[]> {
	const response = await apiFetch(`${API_BASE}/watchlist`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function addToWatchlist(tmdbId: number, mediaType: 'movie' | 'tv'): Promise<void> {
	const response = await apiFetch(`${API_BASE}/watchlist`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ tmdbId, mediaType })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function removeFromWatchlist(tmdbId: number, mediaType: 'movie' | 'tv'): Promise<void> {
	const response = await apiFetch(`${API_BASE}/watchlist/${tmdbId}/${mediaType}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function isInWatchlist(tmdbId: number, mediaType: 'movie' | 'tv'): Promise<boolean> {
	const response = await apiFetch(`${API_BASE}/watchlist/${tmdbId}/${mediaType}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	const result = await response.json();
	return result.inWatchlist;
}
