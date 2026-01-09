// Core utilities
export { API_BASE, apiFetch, getHealth, getImageUrl, getTmdbImageUrl } from './core';
export type { HealthResponse } from './core';

// Libraries
export {
	getLibraries,
	createLibrary,
	deleteLibrary,
	scanLibrary,
	getScanProgress
} from './libraries';
export type { Library, ScanProgress } from './libraries';

// Media (Movies, Shows, Episodes, Music, Books)
export {
	// Movies
	getMovies,
	getMovie,
	refreshMovieMetadata,
	matchMovie,
	deleteMovie,
	// Shows
	getShows,
	getShow,
	refreshShowMetadata,
	matchShow,
	deleteEpisode,
	// Music
	getArtists,
	getArtist,
	getAlbums,
	getAlbum,
	getTrack,
	// Books
	getBooks,
	getBook,
	// Quality Override
	getMovieQuality,
	setMovieQuality,
	deleteMovieQuality,
	getShowQuality,
	setShowQuality
} from './media';
export type {
	Movie,
	Show,
	Season,
	Episode,
	ShowDetail,
	Artist,
	Album,
	Track,
	ArtistDetail,
	AlbumDetail,
	Book,
	MediaQualityStatus,
	MediaQualityOverride,
	QualityInfo
} from './media';

// Streaming
export {
	getStreamUrl,
	getMediaInfo,
	getSubtitleTracks,
	getSubtitleTrackUrl
} from './streaming';
export type {
	VideoStream,
	AudioStream,
	SubtitleTrack,
	MediaInfo
} from './streaming';

// Progress and Watch State
export {
	getProgress,
	saveProgress,
	getContinueWatching,
	removeContinueWatching,
	getWatchStatus,
	markAsWatched,
	markAsUnwatched
} from './progress';
export type {
	Progress,
	ContinueWatchingItem,
	WatchStatus
} from './progress';

// Auth and Users
export {
	checkSetup,
	setup,
	login,
	logout,
	getCurrentUser,
	getUsers,
	createUser,
	updateUser,
	deleteUser
} from './auth';
export type {
	User,
	LoginResponse,
	SetupStatus
} from './auth';

// Settings
export {
	getSettings,
	saveSettings,
	refreshAllMetadata,
	clearLibraryData,
	getNamingTemplates,
	updateNamingTemplate,
	getFormatSettings,
	saveFormatSettings
} from './settings';
export type { NamingTemplate, FormatSettings } from './settings';

// Downloads
export {
	getDownloadClients,
	createDownloadClient,
	updateDownloadClient,
	deleteDownloadClient,
	testDownloadClient,
	getDownloads,
	getDownloadItems,
	deleteDownloadItem,
	getImportHistory,
	getGrabHistory,
	getBlocklist,
	addToBlocklist,
	removeFromBlocklist
} from './downloads';
export type {
	DownloadClient,
	Download,
	TestConnectionResult,
	DownloadItem,
	DeleteDownloadOptions,
	ImportHistoryItem,
	GrabHistoryItem,
	BlocklistEntry
} from './downloads';

// Indexers and Search
export {
	getIndexers,
	createIndexer,
	updateIndexer,
	deleteIndexer,
	testIndexer,
	getIndexerCapabilities,
	getProwlarrConfig,
	saveProwlarrConfig,
	testProwlarrConnection,
	syncProwlarr,
	getIndexerTags,
	searchIndexers,
	searchIndexersScored,
	grabRelease
} from './indexers';
export type {
	Indexer,
	ProwlarrConfig,
	IndexerTag,
	IndexerCapabilities,
	SearchResult,
	CustomFormatHit,
	ScoredSearchResult,
	SearchParams,
	GrabParams,
	ProwlarrTestResult,
	ProwlarrSyncResult
} from './indexers';

// Quality
export {
	getQualityProfiles,
	createQualityProfile,
	updateQualityProfile,
	deleteQualityProfile,
	getCustomFormats,
	createCustomFormat,
	updateCustomFormat,
	deleteCustomFormat,
	parseReleaseName,
	getQualityPresets,
	getQualityPreset,
	createQualityPreset,
	updateQualityPreset,
	deleteQualityPreset,
	toggleQualityPresetEnabled,
	updateQualityPresetPriority,
	setDefaultQualityPreset
} from './quality';
export type {
	QualityProfile,
	CustomFormat,
	ParsedRelease,
	QualityPreset
} from './quality';

// Discover, TMDB, Requests, Wanted, Person, Watchlist
export {
	// TMDB Search
	searchTmdbMovies,
	searchTmdbTV,
	// Discover
	getTrendingMovies,
	getPopularMovies,
	getUpcomingMovies,
	getTheatricalReleases,
	getTopRatedMovies,
	getTrendingShows,
	getPopularShows,
	getUpcomingShows,
	getTopRatedShows,
	getDiscoverMovieDetail,
	getDiscoverShowDetail,
	getDiscoverMovieDetailWithStatus,
	getDiscoverShowDetailWithStatus,
	// Trailers
	getTrailers,
	// Recommendations
	getMovieRecommendations,
	getMovieSuggestions,
	getShowSuggestions,
	// Genres
	getMovieGenres,
	getTVGenres,
	getMoviesByGenre,
	getTVByGenre,
	// Requests
	getRequests,
	createRequest,
	updateRequest,
	deleteRequest,
	// Wanted
	getWantedItems,
	getWantedItem,
	createWantedItem,
	updateWantedItem,
	deleteWantedItem,
	searchWantedItem,
	// Person
	getPersonDetail,
	// Watchlist
	getWatchlist,
	addToWatchlist,
	removeFromWatchlist,
	isInWatchlist
} from './discover';
export type {
	TmdbMovieResult,
	TmdbTVResult,
	DiscoverItem,
	DiscoverResult,
	CastMember,
	CrewMember,
	RecommendedItem,
	SeasonSummary,
	DiscoverMovieDetail,
	DiscoverShowDetail,
	DiscoverMovieDetailWithStatus,
	DiscoverShowDetailWithStatus,
	TrailerInfo,
	TMDBMovieResult,
	TMDBMovieSearchResult,
	TMDBShowResult,
	Genre,
	Request,
	WantedItem,
	PersonCredit,
	LibraryAppearance,
	PersonDetail,
	WatchlistItem
} from './discover';

// System (Storage, Status, Tasks)
export {
	getStorageStatus,
	getSystemStatus,
	getTasks,
	getTask,
	updateTask,
	triggerTask,
	getTaskHistory
} from './system';
export type {
	DiskUsage,
	StorageStatus,
	SystemStatus,
	ScheduledTask,
	TaskHistory
} from './system';
