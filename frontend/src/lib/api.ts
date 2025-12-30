const API_BASE = '/api';

export interface HealthResponse {
	status: string;
}

export interface Library {
	id: number;
	name: string;
	path: string;
	type: 'movies' | 'tv' | 'anime' | 'music' | 'books';
	scanInterval: number;
}

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
	// Watch state
	watchState?: 'unwatched' | 'partial' | 'watched';
	progress?: number;
}

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
	network?: string;
	posterPath?: string;
	backdropPath?: string;
	focalX?: number;
	focalY?: number;
	path: string;
	// Watch state
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

export async function getHealth(): Promise<HealthResponse> {
	const response = await fetch(`${API_BASE}/health`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Libraries

export async function getLibraries(): Promise<Library[]> {
	const response = await fetch(`${API_BASE}/libraries`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createLibrary(library: Omit<Library, 'id'>): Promise<Library> {
	const response = await fetch(`${API_BASE}/libraries`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(library)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteLibrary(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/libraries/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function scanLibrary(id: number): Promise<{ status: string; message: string }> {
	const response = await fetch(`${API_BASE}/libraries/${id}/scan`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Movies

export async function getMovies(): Promise<Movie[]> {
	const response = await fetch(`${API_BASE}/movies`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getMovie(id: number): Promise<Movie> {
	const response = await fetch(`${API_BASE}/movies/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function refreshMovieMetadata(id: number): Promise<Movie> {
	const response = await fetch(`${API_BASE}/movies/${id}/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function matchMovie(id: number, tmdbId: number): Promise<Movie> {
	const response = await fetch(`${API_BASE}/movies/${id}/match`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ tmdbId })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Shows

export async function getShows(): Promise<Show[]> {
	const response = await fetch(`${API_BASE}/shows`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getShow(id: number): Promise<ShowDetail> {
	const response = await fetch(`${API_BASE}/shows/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function refreshShowMetadata(id: number): Promise<ShowDetail> {
	const response = await fetch(`${API_BASE}/shows/${id}/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function matchShow(id: number, tmdbId: number): Promise<ShowDetail> {
	const response = await fetch(`${API_BASE}/shows/${id}/match`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ tmdbId })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Streaming

export function getStreamUrl(type: 'movie' | 'episode', id: number): string {
	return `${API_BASE}/stream/${type}/${id}`;
}

export interface VideoStream {
	index: number;
	codec: string;
	profile?: string;
	width: number;
	height: number;
	aspectRatio?: string;
	frameRate?: string;
	bitRate?: number;
	pixelFormat?: string;
	default: boolean;
}

export interface AudioStream {
	index: number;
	codec: string;
	channels: number;
	channelLayout?: string;
	sampleRate?: number;
	bitRate?: number;
	language?: string;
	title?: string;
	default: boolean;
}

export interface MediaInfo {
	duration: number;
	fileSize?: number;
	bitRate?: number;
	videoStreams: VideoStream[];
	audioStreams: AudioStream[];
	subtitleTracks: SubtitleTrack[];
}

export async function getMediaInfo(type: 'movie' | 'episode', id: number): Promise<MediaInfo> {
	const response = await fetch(`${API_BASE}/media-info/${type}/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Progress

export interface Progress {
	mediaType: string;
	mediaId: number;
	position: number;
	duration: number;
}

export async function getProgress(type: string, id: number): Promise<Progress> {
	const response = await fetch(`${API_BASE}/progress/${type}/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveProgress(progress: Progress): Promise<void> {
	const response = await fetch(`${API_BASE}/progress`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(progress)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Continue Watching

export interface ContinueWatchingItem {
	mediaType: 'movie' | 'episode';
	mediaId: number;
	title: string;
	subtitle?: string;
	showTitle?: string;
	season?: number;
	episode?: number;
	posterPath?: string;
	backdropPath?: string;
	position: number;
	duration: number;
	progressPercent: number;
	updatedAt: string;
}

export async function getContinueWatching(): Promise<ContinueWatchingItem[]> {
	const response = await fetch(`${API_BASE}/continue-watching`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function removeContinueWatching(mediaType: string, mediaId: number): Promise<void> {
	const response = await fetch(`${API_BASE}/continue-watching/${mediaType}/${mediaId}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Settings

export async function getSettings(): Promise<Record<string, string>> {
	const response = await fetch(`${API_BASE}/settings`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveSettings(settings: Record<string, string>): Promise<void> {
	const response = await fetch(`${API_BASE}/settings`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(settings)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function refreshAllMetadata(): Promise<{ refreshed: number; errors: number; total: number }> {
	const response = await fetch(`${API_BASE}/metadata/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// TMDB Search

export async function searchTmdbMovies(query: string, year?: number): Promise<TmdbMovieResult[]> {
	const params = new URLSearchParams({ q: query });
	if (year) params.set('year', year.toString());
	const response = await fetch(`${API_BASE}/tmdb/search/movie?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function searchTmdbTV(query: string, year?: number): Promise<TmdbTVResult[]> {
	const params = new URLSearchParams({ q: query });
	if (year) params.set('year', year.toString());
	const response = await fetch(`${API_BASE}/tmdb/search/tv?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Image URL helper - for library items (served through our API)
export function getImageUrl(path: string | undefined): string | undefined {
	if (!path) return undefined;
	// If path already starts with /images/, use it directly
	if (path.startsWith('/images/')) {
		return path;
	}
	// Otherwise prepend /images/
	const cleanPath = path.startsWith('/') ? path : `/${path}`;
	return `/images${cleanPath}`;
}

// Auth

export interface User {
	id: number;
	username: string;
	role: 'admin' | 'user' | 'kid';
}

export interface LoginResponse {
	token: string;
	user: User;
}

export interface SetupStatus {
	setupRequired: boolean;
}

export async function checkSetup(): Promise<SetupStatus> {
	const response = await fetch(`${API_BASE}/auth/setup`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function setup(username: string, password: string): Promise<User> {
	const response = await fetch(`${API_BASE}/auth/setup`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function login(username: string, password: string): Promise<LoginResponse> {
	const response = await fetch(`${API_BASE}/auth/login`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password })
	});
	if (!response.ok) {
		throw new Error(`Invalid credentials`);
	}
	return response.json();
}

export async function logout(): Promise<void> {
	const response = await fetch(`${API_BASE}/auth/logout`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function getCurrentUser(): Promise<User | null> {
	try {
		const response = await fetch(`${API_BASE}/auth/me`);
		if (!response.ok) {
			return null;
		}
		return response.json();
	} catch {
		return null;
	}
}

// Users

export async function getUsers(): Promise<User[]> {
	const response = await fetch(`${API_BASE}/users`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createUser(username: string, password: string, role: string): Promise<User> {
	const response = await fetch(`${API_BASE}/users`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password, role })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateUser(id: number, data: { username?: string; password?: string; role?: string }): Promise<User> {
	const response = await fetch(`${API_BASE}/users/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteUser(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/users/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// Download Clients

export interface DownloadClient {
	id: number;
	name: string;
	type: 'qbittorrent' | 'transmission' | 'sabnzbd' | 'nzbget';
	host: string;
	port: number;
	username?: string;
	password?: string;
	apiKey?: string;
	useTls: boolean;
	category?: string;
	priority: number;
	enabled: boolean;
}

export interface Download {
	id: string;
	name: string;
	size: number;
	downloaded: number;
	progress: number;
	speed: number;
	eta: number;
	status: 'downloading' | 'paused' | 'completed' | 'error' | 'queued';
	savePath: string;
	category: string;
	clientId: number;
	clientName: string;
	clientType: string;
}

export interface TestConnectionResult {
	success: boolean;
	message?: string;
	error?: string;
}

export async function getDownloadClients(): Promise<DownloadClient[]> {
	const response = await fetch(`${API_BASE}/download-clients`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createDownloadClient(client: Omit<DownloadClient, 'id'>): Promise<DownloadClient> {
	const response = await fetch(`${API_BASE}/download-clients`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(client)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateDownloadClient(id: number, client: Partial<DownloadClient>): Promise<DownloadClient> {
	const response = await fetch(`${API_BASE}/download-clients/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(client)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteDownloadClient(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/download-clients/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function testDownloadClient(id: number): Promise<TestConnectionResult> {
	const response = await fetch(`${API_BASE}/download-clients/${id}/test`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getDownloads(): Promise<Download[]> {
	const response = await fetch(`${API_BASE}/downloads`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Indexers

export interface Indexer {
	id: number;
	name: string;
	type: 'torznab' | 'newznab' | 'prowlarr';
	url: string;
	apiKey?: string;
	categories?: string;
	priority: number;
	enabled: boolean;
}

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

export async function getIndexers(): Promise<Indexer[]> {
	const response = await fetch(`${API_BASE}/indexers`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createIndexer(indexer: Omit<Indexer, 'id'>): Promise<Indexer> {
	const response = await fetch(`${API_BASE}/indexers`, {
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
	const response = await fetch(`${API_BASE}/indexers/${id}`, {
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
	const response = await fetch(`${API_BASE}/indexers/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function testIndexer(id: number): Promise<TestConnectionResult> {
	const response = await fetch(`${API_BASE}/indexers/${id}/test`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getIndexerCapabilities(id: number): Promise<IndexerCapabilities> {
	const response = await fetch(`${API_BASE}/indexers/${id}/capabilities`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
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

	const response = await fetch(`${API_BASE}/search?${searchParams}`);
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

	const response = await fetch(`${API_BASE}/search/scored?${searchParams}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export interface GrabParams {
	link: string;
	magnetLink?: string;
	indexerType: string;
	category?: string;
}

export async function grabRelease(params: GrabParams): Promise<{ success: boolean; message: string; client?: string }> {
	const response = await fetch(`${API_BASE}/grab`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(params)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Quality Profiles

export interface QualityProfile {
	id: number;
	name: string;
	upgradeAllowed: boolean;
	upgradeUntilScore: number;
	minFormatScore: number;
	cutoffFormatScore: number;
	qualities: string; // JSON array
	customFormats: string; // JSON object of format_id -> score
}

export interface CustomFormat {
	id: number;
	name: string;
	conditions: string; // JSON array
}

export interface ParsedRelease {
	title: string;
	year?: number;
	resolution?: string;
	source?: string;
	codec?: string;
	audioCodec?: string;
	audioFeature?: string;
	hdr?: string[];
	releaseGroup?: string;
	proper?: boolean;
	repack?: boolean;
	edition?: string;
	season?: number;
	episode?: number;
	quality?: string;
}

export async function getQualityProfiles(): Promise<QualityProfile[]> {
	const response = await fetch(`${API_BASE}/quality-profiles`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createQualityProfile(profile: Omit<QualityProfile, 'id'>): Promise<QualityProfile> {
	const response = await fetch(`${API_BASE}/quality-profiles`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(profile)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateQualityProfile(id: number, profile: Partial<QualityProfile>): Promise<QualityProfile> {
	const response = await fetch(`${API_BASE}/quality-profiles/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(profile)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteQualityProfile(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/quality-profiles/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function getCustomFormats(): Promise<CustomFormat[]> {
	const response = await fetch(`${API_BASE}/custom-formats`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createCustomFormat(format: Omit<CustomFormat, 'id'>): Promise<CustomFormat> {
	const response = await fetch(`${API_BASE}/custom-formats`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(format)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateCustomFormat(id: number, format: Partial<CustomFormat>): Promise<CustomFormat> {
	const response = await fetch(`${API_BASE}/custom-formats/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(format)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteCustomFormat(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/custom-formats/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function parseReleaseName(name: string): Promise<ParsedRelease> {
	const response = await fetch(`${API_BASE}/releases/parse`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
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
	seasons?: string; // JSON array of season numbers
	lastSearched?: string;
	addedAt: string;
}

export async function getWantedItems(): Promise<WantedItem[]> {
	const response = await fetch(`${API_BASE}/wanted`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getWantedItem(id: number): Promise<WantedItem> {
	const response = await fetch(`${API_BASE}/wanted/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createWantedItem(item: Omit<WantedItem, 'id' | 'addedAt'>): Promise<WantedItem> {
	const response = await fetch(`${API_BASE}/wanted`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(item)
	});
	if (!response.ok) {
		if (response.status === 409) {
			throw new Error('Item already in wanted list');
		}
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateWantedItem(id: number, updates: Partial<WantedItem>): Promise<WantedItem> {
	const response = await fetch(`${API_BASE}/wanted/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(updates)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteWantedItem(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/wanted/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function searchWantedItem(id: number): Promise<ScoredSearchResult[]> {
	const response = await fetch(`${API_BASE}/wanted/search/${id}`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Discover types and functions

export interface DiscoverItem {
	id: number;
	type: 'movie' | 'show';
	title: string;
	overview: string;
	releaseDate: string;
	posterPath: string;
	backdropPath: string;
	rating: number;
	popularity: number;
	focalX?: number;
	focalY?: number;
	// Status fields - from enriched API
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

export async function getTrendingMovies(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/movies/trending?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getPopularMovies(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/movies/popular?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getUpcomingMovies(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/movies/upcoming?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getTopRatedMovies(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/movies/top-rated?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getTrendingShows(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/shows/trending?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getPopularShows(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/shows/popular?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getTopRatedShows(page = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/shows/top-rated?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Request types and functions

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
	status: 'requested' | 'approved' | 'denied' | 'available';
	statusReason?: string;
	requestedAt: string;
	updatedAt: string;
}

export async function getRequests(status?: string): Promise<Request[]> {
	const url = status
		? `${API_BASE}/requests?status=${status}`
		: `${API_BASE}/requests`;
	const response = await fetch(url);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createRequest(request: {
	type: 'movie' | 'show';
	tmdbId: number;
	title: string;
	year?: number;
	overview?: string;
	posterPath?: string;
}): Promise<Request> {
	const response = await fetch(`${API_BASE}/requests`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
	if (!response.ok) {
		if (response.status === 409) {
			throw new Error('Already requested');
		}
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateRequest(id: number, status: string, statusReason?: string): Promise<Request> {
	const response = await fetch(`${API_BASE}/requests/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ status, statusReason })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteRequest(id: number): Promise<void> {
	const response = await fetch(`${API_BASE}/requests/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

// TMDB image URL helper
export function getTmdbImageUrl(path: string | undefined, size = 'w500'): string {
	if (!path) return '';
	return `https://image.tmdb.org/t/p/${size}${path}`;
}

// Discover detail types

export interface CastMember {
	name: string;
	character: string;
	photo: string;
}

export interface DiscoverMovieDetail {
	id: number;
	title: string;
	overview: string;
	tagline: string;
	releaseDate: string;
	runtime: number;
	rating: number;
	posterPath: string;
	backdropPath: string;
	genres: string[];
	cast: CastMember[];
	director: string;
}

export interface DiscoverShowDetail {
	id: number;
	title: string;
	overview: string;
	firstAirDate: string;
	status: string;
	rating: number;
	posterPath: string;
	backdropPath: string;
	genres: string[];
	networks: string[];
	seasons: number;
	cast: CastMember[];
}

export async function getDiscoverMovieDetail(id: number): Promise<DiscoverMovieDetail> {
	const response = await fetch(`${API_BASE}/discover/movie/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getDiscoverShowDetail(id: number): Promise<DiscoverShowDetail> {
	const response = await fetch(`${API_BASE}/discover/show/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Movie recommendations (raw TMDB format)
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

export async function getMovieRecommendations(tmdbId: number): Promise<TMDBMovieSearchResult> {
	const response = await fetch(`${API_BASE}/movie/recommendations/${tmdbId}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Movie suggestions - genre-based, excluding library items
export async function getMovieSuggestions(movieId: number): Promise<{ results: TMDBMovieResult[] }> {
	const response = await fetch(`${API_BASE}/movies/suggestions/${movieId}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Genre types
export interface Genre {
	id: number;
	name: string;
}

export async function getMovieGenres(): Promise<{ genres: Genre[] }> {
	const response = await fetch(`${API_BASE}/discover/genres/movie`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getTVGenres(): Promise<{ genres: Genre[] }> {
	const response = await fetch(`${API_BASE}/discover/genres/tv`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getMoviesByGenre(genreId: number, page: number = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/movies/genre/${genreId}?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getTVByGenre(genreId: number, page: number = 1): Promise<DiscoverResult> {
	const response = await fetch(`${API_BASE}/discover/shows/genre/${genreId}?page=${page}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
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

// Music API functions

export async function getArtists(): Promise<Artist[]> {
	const response = await fetch(`${API_BASE}/artists`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getArtist(id: number): Promise<ArtistDetail> {
	const response = await fetch(`${API_BASE}/artists/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getAlbums(): Promise<Album[]> {
	const response = await fetch(`${API_BASE}/albums`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getAlbum(id: number): Promise<AlbumDetail> {
	const response = await fetch(`${API_BASE}/albums/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTrack(id: number): Promise<Track> {
	const response = await fetch(`${API_BASE}/tracks/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Book API functions

export async function getBooks(): Promise<Book[]> {
	const response = await fetch(`${API_BASE}/books`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getBook(id: number): Promise<Book> {
	const response = await fetch(`${API_BASE}/books/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Watch state API functions

export interface WatchStatus {
	watched: boolean;
	progress: number;
}

export async function getWatchStatus(mediaType: 'movie' | 'episode', mediaId: number): Promise<WatchStatus> {
	const response = await fetch(`${API_BASE}/watched/${mediaType}/${mediaId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function markAsWatched(mediaType: 'movie' | 'episode', mediaId: number, duration?: number): Promise<void> {
	const response = await fetch(`${API_BASE}/watched/${mediaType}/${mediaId}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ duration: duration || 3600 })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function markAsUnwatched(mediaType: 'movie' | 'episode', mediaId: number): Promise<void> {
	const response = await fetch(`${API_BASE}/watched/${mediaType}/${mediaId}`, {
		method: 'DELETE'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

// Subtitle API functions

export interface SubtitleTrack {
	index: number;
	language: string;
	title: string;
	codec: string;
	default: boolean;
	forced: boolean;
	external: boolean;
	filePath?: string;
}

export async function getSubtitleTracks(mediaType: 'movie' | 'episode', mediaId: number): Promise<SubtitleTrack[]> {
	const response = await fetch(`${API_BASE}/subtitles/${mediaType}/${mediaId}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export function getSubtitleTrackUrl(mediaType: 'movie' | 'episode', mediaId: number, trackIndex: number): string {
	return `${API_BASE}/subtitles/${mediaType}/${mediaId}/track/${trackIndex}`;
}
