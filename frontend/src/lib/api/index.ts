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
	// Episodes
	getEpisode,
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
	setShowQuality,
	// Missing Episodes
	getMissingEpisodes,
	requestMissingEpisodes
} from './media';
export type {
	Movie,
	Show,
	Season,
	Episode,
	EpisodeDetail,
	ShowDetail,
	Artist,
	Album,
	Track,
	ArtistDetail,
	AlbumDetail,
	Book,
	MediaQualityStatus,
	MediaQualityOverride,
	QualityInfo,
	MissingEpisode,
	SeasonMissingSummary,
	MissingEpisodesResult
} from './media';

// Streaming
export {
	getStreamUrl,
	getMediaInfo,
	getSubtitleTracks,
	getSubtitleTrackUrl,
	getChapters,
	getSkipSegments,
	saveSkipSegment,
	deleteSkipSegment
} from './streaming';
export type {
	VideoStream,
	AudioStream,
	SubtitleTrack,
	MediaInfo,
	Chapter,
	SkipSegment,
	SkipSegments
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
	deleteUser,
	getSetupWizardStatus,
	completeSetupWizard,
	verifyPin
} from './auth';
export type {
	User,
	LoginResponse,
	SetupStatus,
	SetupWizardStatus,
	SetupWizardSteps,
	ContentRating,
	PinVerifyResponse,
	CreateUserData,
	UpdateUserData
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
	saveFormatSettings,
	downloadBackup,
	restoreBackup
} from './settings';
export type { NamingTemplate, FormatSettings, RestoreResult } from './settings';

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

// System (Storage, Status, Tasks, Logs, Analytics, Health)
export {
	getStorageStatus,
	getSystemStatus,
	getTasks,
	getTask,
	updateTask,
	triggerTask,
	getTaskHistory,
	getLogs,
	downloadLogs,
	getStorageAnalytics,
	getHealthFull,
	recheckHealth
} from './system';
export type {
	DiskUsage,
	StorageStatus,
	SystemStatus,
	ScheduledTask,
	TaskHistory,
	LogEntry,
	LogsResponse,
	LogsQuery,
	StorageAnalytics,
	LibrarySize,
	QualitySize,
	YearSize,
	LargestItem,
	DuplicateCopy,
	DuplicateItem,
	HealthCheckStatus,
	HealthCheck,
	FullHealthResponse
} from './system';

// Calendar
export { getCalendarItems } from './calendar';
export type { CalendarItem, CalendarFilter } from './calendar';

// Notifications
export {
	getNotifications,
	getUnreadCount,
	markRead,
	markAllRead,
	deleteNotification
} from './notifications';
export type { Notification } from './notifications';

// Collections
export {
	getCollections,
	getCollection,
	createCollection,
	updateCollection,
	deleteCollection,
	addCollectionItem,
	removeCollectionItem,
	reorderCollectionItems,
	getMediaCollections
} from './collections';
export type { Collection, CollectionItem, CollectionDetail } from './collections';

// Profiles
export {
	getProfiles,
	getProfile,
	getActiveProfile,
	createProfile,
	updateProfile,
	deleteProfile,
	selectProfile,
	AVATARS
} from './profiles';
export type { Profile, CreateProfileData, UpdateProfileData } from './profiles';

// Smart Playlists
export {
	getSmartPlaylists,
	getSmartPlaylist,
	createSmartPlaylist,
	updateSmartPlaylist,
	deleteSmartPlaylist,
	refreshSmartPlaylist,
	previewSmartPlaylist,
	parseRules,
	stringifyRules,
	RULE_FIELDS,
	OPERATORS
} from './smartPlaylists';
export type {
	SmartPlaylist,
	SmartPlaylistDetail,
	SmartPlaylistItem,
	PlaylistRules,
	RuleCondition
} from './smartPlaylists';

// Upgrades
export {
	getUpgrades,
	searchUpgrade,
	searchAllUpgrades,
	resetUpgradeSearch,
	formatQualityComparison,
	getScoreDifference
} from './upgrades';
export type { UpgradeableItem, UpgradesSummary } from './upgrades';

// Subtitles (OpenSubtitles)
export {
	searchSubtitles,
	downloadSubtitle,
	getSubtitleLanguages,
	testOpenSubtitlesConnection,
	COMMON_LANGUAGES,
	formatSubtitleDate,
	formatDownloads
} from './subtitles';
export type {
	SubtitleResult,
	SubtitleLanguage,
	SubtitleSearchParams,
	SubtitleDownloadParams,
	SubtitleDownloadResult
} from './subtitles';

// Trakt.tv
export {
	getTraktAuthURL,
	exchangeTraktCode,
	getTraktConfig,
	updateTraktConfig,
	disconnectTrakt,
	syncTrakt,
	testTraktConnection,
	formatLastSynced
} from './trakt';
export type {
	TraktConfig,
	TraktSyncResult,
	TraktTestResult
} from './trakt';
