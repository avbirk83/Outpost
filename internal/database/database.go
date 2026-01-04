package database

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

type Library struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	Type         string `json:"type"` // movies, tv, anime, music, books
	ScanInterval int    `json:"scanInterval"`
}

type Movie struct {
	ID                 int64     `json:"id"`
	LibraryID          int64     `json:"libraryId"`
	TmdbID             *int64    `json:"tmdbId,omitempty"`
	ImdbID             *string   `json:"imdbId,omitempty"`
	Title              string    `json:"title"`
	OriginalTitle      *string   `json:"originalTitle,omitempty"`
	Year               int       `json:"year"`
	Overview           *string   `json:"overview,omitempty"`
	Tagline            *string   `json:"tagline,omitempty"`
	Runtime            *int      `json:"runtime,omitempty"`
	Rating             *float64  `json:"rating,omitempty"`
	ContentRating      *string   `json:"contentRating,omitempty"`
	Genres             *string   `json:"genres,omitempty"`
	Cast               *string   `json:"cast,omitempty"`
	Crew               *string   `json:"crew,omitempty"`
	Director           *string   `json:"director,omitempty"`
	Writer             *string   `json:"writer,omitempty"`
	Editor             *string   `json:"editor,omitempty"`
	Producers          *string   `json:"producers,omitempty"`
	Status             *string   `json:"status,omitempty"`
	Budget             *int64    `json:"budget,omitempty"`
	Revenue            *int64    `json:"revenue,omitempty"`
	Country            *string   `json:"country,omitempty"`
	OriginalLanguage   *string   `json:"originalLanguage,omitempty"`
	TheatricalRelease  *string   `json:"theatricalRelease,omitempty"`
	DigitalRelease     *string   `json:"digitalRelease,omitempty"`
	Studios            *string   `json:"studios,omitempty"`
	Trailers           *string   `json:"trailers,omitempty"`
	PosterPath         *string   `json:"posterPath,omitempty"`
	BackdropPath       *string   `json:"backdropPath,omitempty"`
	FocalX             *float64  `json:"focalX,omitempty"`
	FocalY             *float64  `json:"focalY,omitempty"`
	Path               string    `json:"path"`
	Size               int64     `json:"size"`
	AddedAt            time.Time `json:"addedAt"`
	LastWatchedAt      *string   `json:"lastWatchedAt,omitempty"`
	PlayCount          int       `json:"playCount"`
}

type Show struct {
	ID            int64    `json:"id"`
	LibraryID     int64    `json:"libraryId"`
	TmdbID        *int64   `json:"tmdbId,omitempty"`
	TvdbID        *int64   `json:"tvdbId,omitempty"`
	ImdbID        *string  `json:"imdbId,omitempty"`
	Title         string   `json:"title"`
	OriginalTitle *string  `json:"originalTitle,omitempty"`
	Year          int      `json:"year"`
	Overview      *string  `json:"overview,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Rating        *float64 `json:"rating,omitempty"`
	ContentRating *string  `json:"contentRating,omitempty"`
	Genres        *string  `json:"genres,omitempty"`
	Cast          *string  `json:"cast,omitempty"`
	Crew          *string  `json:"crew,omitempty"`
	Network       *string  `json:"network,omitempty"`
	PosterPath    *string  `json:"posterPath,omitempty"`
	BackdropPath  *string     `json:"backdropPath,omitempty"`
	FocalX        *float64   `json:"focalX,omitempty"`
	FocalY        *float64   `json:"focalY,omitempty"`
	Path          string     `json:"path"`
	AddedAt       *time.Time `json:"addedAt,omitempty"`
}

type Season struct {
	ID           int64   `json:"id"`
	ShowID       int64   `json:"showId"`
	SeasonNumber int     `json:"seasonNumber"`
	Name         *string `json:"name,omitempty"`
	Overview     *string `json:"overview,omitempty"`
	PosterPath   *string `json:"posterPath,omitempty"`
	AirDate      *string `json:"airDate,omitempty"`
}

type Episode struct {
	ID            int64   `json:"id"`
	SeasonID      int64   `json:"seasonId"`
	EpisodeNumber int     `json:"episodeNumber"`
	Title         string  `json:"title"`
	Overview      *string `json:"overview,omitempty"`
	AirDate       *string `json:"airDate,omitempty"`
	Runtime       *int    `json:"runtime,omitempty"`
	StillPath     *string `json:"stillPath,omitempty"`
	Path          string  `json:"path"`
	Size          int64   `json:"size"`
}

type Progress struct {
	ID        int64     `json:"id"`
	MediaType string    `json:"mediaType"` // movie, episode
	MediaID   int64     `json:"mediaId"`
	Position  float64   `json:"position"`  // seconds
	Duration  float64   `json:"duration"`  // seconds
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	Role         string    `json:"role"` // admin, user, kid
	CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type DownloadClient struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // qbittorrent, transmission, sabnzbd, nzbget
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	APIKey   string `json:"apiKey,omitempty"` // For SABnzbd/NZBGet
	UseTLS   bool   `json:"useTls"`
	Category string `json:"category,omitempty"` // Download category/label
	Priority int    `json:"priority"`           // Client priority (for selecting which to use)
	Enabled  bool   `json:"enabled"`
}

type Indexer struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // torznab, newznab, prowlarr
	URL        string `json:"url"`
	APIKey     string `json:"apiKey,omitempty"`
	Categories string `json:"categories,omitempty"` // Comma-separated category IDs
	Priority   int    `json:"priority"`
	Enabled    bool   `json:"enabled"`
}

type QualityProfile struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	UpgradeAllowed     bool   `json:"upgradeAllowed"`
	UpgradeUntilScore  int    `json:"upgradeUntilScore"`
	MinFormatScore     int    `json:"minFormatScore"`
	CutoffFormatScore  int    `json:"cutoffFormatScore"`
	Qualities          string `json:"qualities"`     // JSON array of enabled qualities
	CustomFormatScores string `json:"customFormats"` // JSON object of format_id -> score
}

type CustomFormat struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Conditions string `json:"conditions"` // JSON array of conditions
}

type WantedItem struct {
	ID               int64     `json:"id"`
	Type             string    `json:"type"`             // movie, show
	TmdbID           int64     `json:"tmdbId"`
	Title            string    `json:"title"`
	Year             int       `json:"year,omitempty"`
	PosterPath       *string   `json:"posterPath,omitempty"`
	QualityProfileID int64     `json:"qualityProfileId"`
	Monitored        bool      `json:"monitored"`
	Seasons          string    `json:"seasons,omitempty"`       // JSON array of season numbers, empty = all
	SearchNow        bool      `json:"searchNow,omitempty"`     // For triggering immediate search
	LastSearched     *time.Time `json:"lastSearched,omitempty"`
	AddedAt          time.Time `json:"addedAt"`
}

type Request struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"userId"`
	Username     string    `json:"username,omitempty"` // Populated from join
	Type         string    `json:"type"`               // movie, show
	TmdbID       int64     `json:"tmdbId"`
	Title        string    `json:"title"`
	Year         int       `json:"year,omitempty"`
	Overview     *string   `json:"overview,omitempty"`
	PosterPath   *string   `json:"posterPath,omitempty"`
	Status       string    `json:"status"` // requested, approved, denied, available
	StatusReason *string   `json:"statusReason,omitempty"`
	RequestedAt  time.Time `json:"requestedAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Music types

type Artist struct {
	ID           int64   `json:"id"`
	LibraryID    int64   `json:"libraryId"`
	MusicBrainzID *string `json:"musicBrainzId,omitempty"`
	Name         string  `json:"name"`
	SortName     *string `json:"sortName,omitempty"`
	Overview     *string `json:"overview,omitempty"`
	ImagePath    *string `json:"imagePath,omitempty"`
	Path         string  `json:"path"`
}

type Album struct {
	ID            int64   `json:"id"`
	ArtistID      int64   `json:"artistId"`
	MusicBrainzID *string `json:"musicBrainzId,omitempty"`
	Title         string  `json:"title"`
	Year          int     `json:"year,omitempty"`
	Overview      *string `json:"overview,omitempty"`
	CoverPath     *string `json:"coverPath,omitempty"`
	Path          string  `json:"path"`
}

type Track struct {
	ID            int64   `json:"id"`
	AlbumID       int64   `json:"albumId"`
	MusicBrainzID *string `json:"musicBrainzId,omitempty"`
	Title         string  `json:"title"`
	TrackNumber   int     `json:"trackNumber"`
	DiscNumber    int     `json:"discNumber"`
	Duration      int     `json:"duration"` // seconds
	Path          string  `json:"path"`
	Size          int64   `json:"size"`
}

// Book types

type Book struct {
	ID          int64     `json:"id"`
	LibraryID   int64     `json:"libraryId"`
	Title       string    `json:"title"`
	Author      *string   `json:"author,omitempty"`
	ISBN        *string   `json:"isbn,omitempty"`
	Publisher   *string   `json:"publisher,omitempty"`
	Year        int       `json:"year,omitempty"`
	Description *string   `json:"description,omitempty"`
	CoverPath   *string   `json:"coverPath,omitempty"`
	Format      string    `json:"format"` // epub, pdf, mobi, cbz, cbr
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	AddedAt     time.Time `json:"addedAt"`
}

// Watchlist types

type WatchlistItem struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	TmdbID    int64     `json:"tmdbId"`
	MediaType string    `json:"mediaType"` // "movie" or "tv"
	AddedAt   time.Time `json:"addedAt"`
}

// Quality preset types

type QualityPreset struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	IsDefault        bool      `json:"isDefault"`
	IsBuiltIn        bool      `json:"isBuiltIn"`
	Resolution       string    `json:"resolution"`       // "4k", "1080p", "720p", "480p"
	Source           string    `json:"source"`           // "remux", "bluray", "web", "any"
	HDRFormats       []string  `json:"hdrFormats"`       // Array of HDR formats
	Codec            string    `json:"codec"`            // "any", "hevc", "av1"
	AudioFormats     []string  `json:"audioFormats"`     // Array of audio formats
	PreferredEdition string    `json:"preferredEdition"` // "any", "theatrical", "directors", etc
	MinSeeders       int       `json:"minSeeders"`
	PreferSeasonPacks bool     `json:"preferSeasonPacks"`
	AutoUpgrade      bool      `json:"autoUpgrade"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type MediaQualityOverride struct {
	ID        int64     `json:"id"`
	MediaID   int64     `json:"mediaId"`
	MediaType string    `json:"mediaType"`
	PresetID  *int64    `json:"presetId"`
	Monitored bool      `json:"monitored"`
	CreatedAt time.Time `json:"createdAt"`
}

type MediaQualityStatus struct {
	ID                int64      `json:"id"`
	MediaID           int64      `json:"mediaId"`
	MediaType         string     `json:"mediaType"`
	CurrentResolution *string    `json:"currentResolution"`
	CurrentSource     *string    `json:"currentSource"`
	CurrentHDR        *string    `json:"currentHdr"`
	CurrentAudio      *string    `json:"currentAudio"`
	CurrentEdition    *string    `json:"currentEdition"`
	TargetMet         bool       `json:"targetMet"`
	UpgradeAvailable  bool       `json:"upgradeAvailable"`
	LastSearch        *time.Time `json:"lastSearch"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// Download tracking types

type Download struct {
	ID               int64      `json:"id"`
	DownloadClientID *int64     `json:"downloadClientId"`
	ExternalID       string     `json:"externalId"`
	MediaID          *int64     `json:"mediaId"`
	MediaType        *string    `json:"mediaType"`
	Title            string     `json:"title"`
	Size             int64      `json:"size"`
	Status           string     `json:"status"` // downloading, completed, importing, imported, failed, unmatched
	Progress         float64    `json:"progress"`
	DownloadPath     *string    `json:"downloadPath"`
	ImportedPath     *string    `json:"importedPath"`
	Error            *string    `json:"error"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type NamingTemplate struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"` // movie, tv, daily
	FolderTemplate string `json:"folderTemplate"`
	FileTemplate   string `json:"fileTemplate"`
	IsDefault      bool   `json:"isDefault"`
}

type ImportHistory struct {
	ID         int64     `json:"id"`
	DownloadID *int64    `json:"downloadId"`
	SourcePath string    `json:"sourcePath"`
	DestPath   string    `json:"destPath"`
	MediaID    *int64    `json:"mediaId"`
	MediaType  *string   `json:"mediaType"`
	Success    bool      `json:"success"`
	Error      *string   `json:"error"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Grab history tracks all release grabs
type GrabHistory struct {
	ID               int64      `json:"id"`
	MediaID          int64      `json:"mediaId"`
	MediaType        string     `json:"mediaType"`
	ReleaseTitle     string     `json:"releaseTitle"`
	IndexerID        *int64     `json:"indexerId"`
	IndexerName      *string    `json:"indexerName"`
	QualityResolution *string   `json:"qualityResolution"`
	QualitySource    *string    `json:"qualitySource"`
	QualityCodec     *string    `json:"qualityCodec"`
	QualityAudio     *string    `json:"qualityAudio"`
	QualityHDR       *string    `json:"qualityHdr"`
	ReleaseGroup     *string    `json:"releaseGroup"`
	Size             int64      `json:"size"`
	DownloadClientID *int64     `json:"downloadClientId"`
	DownloadID       *string    `json:"downloadId"`
	Status           string     `json:"status"` // grabbed, imported, failed
	ErrorMessage     *string    `json:"errorMessage"`
	GrabbedAt        time.Time  `json:"grabbedAt"`
	ImportedAt       *time.Time `json:"importedAt"`
}

// Blocklist tracks releases that should not be grabbed again
type BlocklistEntry struct {
	ID           int64      `json:"id"`
	MediaID      *int64     `json:"mediaId"`
	MediaType    *string    `json:"mediaType"`
	ReleaseTitle string     `json:"releaseTitle"`
	ReleaseGroup *string    `json:"releaseGroup"`
	IndexerID    *int64     `json:"indexerId"`
	Reason       string     `json:"reason"`
	ErrorMessage *string    `json:"errorMessage"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// BlockedGroup tracks blocked release groups
type BlockedGroup struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Reason       *string   `json:"reason"`
	AutoBlocked  bool      `json:"autoBlocked"`
	FailureCount int       `json:"failureCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

// TrustedGroup tracks trusted release groups
type TrustedGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"` // movies, tv, anime
	CreatedAt time.Time `json:"createdAt"`
}

// ReleaseFilter for must/must not contain filters
type ReleaseFilter struct {
	ID         int64     `json:"id"`
	PresetID   int64     `json:"presetId"`
	FilterType string    `json:"filterType"` // must_contain, must_not_contain
	Value      string    `json:"value"`
	IsRegex    bool      `json:"isRegex"`
	CreatedAt  time.Time `json:"createdAt"`
}

// DelayProfile for waiting for better quality
type DelayProfile struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Enabled           bool    `json:"enabled"`
	DelayMinutes      int     `json:"delayMinutes"`
	BypassIfResolution *string `json:"bypassIfResolution"`
	BypassIfSource    *string `json:"bypassIfSource"`
	BypassIfScoreAbove *int   `json:"bypassIfScoreAbove"`
	LibraryID         *int64  `json:"libraryId"`
	CreatedAt         time.Time `json:"createdAt"`
}

// PendingGrab for releases waiting for delay
type PendingGrab struct {
	ID           int64     `json:"id"`
	MediaID      int64     `json:"mediaId"`
	MediaType    string    `json:"mediaType"`
	ReleaseTitle string    `json:"releaseTitle"`
	ReleaseData  *string   `json:"releaseData"` // JSON encoded release info
	Score        int       `json:"score"`
	IndexerID    *int64    `json:"indexerId"`
	AvailableAt  time.Time `json:"availableAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Exclusion represents an excluded media item or indexer
type Exclusion struct {
	ID            int64     `json:"id"`
	ExclusionType string    `json:"exclusionType"` // "movie", "show", "indexer"
	MediaID       *int64    `json:"mediaId,omitempty"`
	MediaType     *string   `json:"mediaType,omitempty"`
	IndexerID     *int64    `json:"indexerId,omitempty"`
	LibraryID     *int64    `json:"libraryId,omitempty"`
	Reason        *string   `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	d := &Database{db: db}
	if err := d.migrate(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS libraries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		type TEXT NOT NULL,
		scan_interval INTEGER DEFAULT 3600
	);

	CREATE TABLE IF NOT EXISTS movies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER NOT NULL,
		tmdb_id INTEGER,
		imdb_id TEXT,
		title TEXT NOT NULL,
		original_title TEXT,
		year INTEGER,
		overview TEXT,
		tagline TEXT,
		runtime INTEGER,
		rating REAL,
		content_rating TEXT,
		genres TEXT,
		cast TEXT,
		director TEXT,
		writer TEXT,
		status TEXT,
		budget INTEGER,
		revenue INTEGER,
		country TEXT,
		original_language TEXT,
		theatrical_release TEXT,
		digital_release TEXT,
		studios TEXT,
		trailers TEXT,
		poster_path TEXT,
		backdrop_path TEXT,
		focal_x REAL,
		focal_y REAL,
		path TEXT NOT NULL UNIQUE,
		size INTEGER,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_watched_at TEXT,
		play_count INTEGER DEFAULT 0,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS shows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER NOT NULL,
		tmdb_id INTEGER,
		tvdb_id INTEGER,
		imdb_id TEXT,
		title TEXT NOT NULL,
		original_title TEXT,
		year INTEGER,
		overview TEXT,
		status TEXT,
		rating REAL,
		content_rating TEXT,
		genres TEXT,
		cast TEXT,
		crew TEXT,
		network TEXT,
		poster_path TEXT,
		backdrop_path TEXT,
		focal_x REAL,
		focal_y REAL,
		path TEXT NOT NULL UNIQUE,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS seasons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		show_id INTEGER NOT NULL,
		season_number INTEGER NOT NULL,
		name TEXT,
		overview TEXT,
		poster_path TEXT,
		air_date TEXT,
		FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE,
		UNIQUE(show_id, season_number)
	);

	CREATE TABLE IF NOT EXISTS episodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		season_id INTEGER NOT NULL,
		episode_number INTEGER NOT NULL,
		title TEXT,
		overview TEXT,
		air_date TEXT,
		runtime INTEGER,
		still_path TEXT,
		path TEXT NOT NULL UNIQUE,
		size INTEGER,
		FOREIGN KEY (season_id) REFERENCES seasons(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_type TEXT NOT NULL,
		media_id INTEGER NOT NULL,
		position REAL NOT NULL DEFAULT 0,
		duration REAL NOT NULL DEFAULT 0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(media_type, media_id)
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS user_watchlist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		tmdb_id INTEGER NOT NULL,
		media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, tmdb_id, media_type)
	);
	CREATE INDEX IF NOT EXISTS idx_watchlist_user ON user_watchlist(user_id);

	CREATE TABLE IF NOT EXISTS download_clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT,
		password TEXT,
		api_key TEXT,
		use_tls INTEGER DEFAULT 0,
		category TEXT,
		priority INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS indexers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		url TEXT NOT NULL,
		api_key TEXT,
		categories TEXT,
		priority INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS quality_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		upgrade_allowed INTEGER DEFAULT 1,
		upgrade_until_score INTEGER DEFAULT 0,
		min_format_score INTEGER DEFAULT 0,
		cutoff_format_score INTEGER DEFAULT 0,
		qualities TEXT DEFAULT '[]',
		custom_format_scores TEXT DEFAULT '{}'
	);

	CREATE TABLE IF NOT EXISTS custom_formats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		conditions TEXT DEFAULT '[]'
	);

	CREATE TABLE IF NOT EXISTS wanted (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		tmdb_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		year INTEGER,
		poster_path TEXT,
		quality_profile_id INTEGER NOT NULL,
		monitored INTEGER DEFAULT 1,
		seasons TEXT DEFAULT '[]',
		last_searched DATETIME,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(type, tmdb_id)
	);

	CREATE TABLE IF NOT EXISTS requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		tmdb_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		year INTEGER,
		overview TEXT,
		poster_path TEXT,
		status TEXT NOT NULL DEFAULT 'requested',
		status_reason TEXT,
		requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, type, tmdb_id)
	);

	CREATE TABLE IF NOT EXISTS artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER NOT NULL,
		musicbrainz_id TEXT,
		name TEXT NOT NULL,
		sort_name TEXT,
		overview TEXT,
		image_path TEXT,
		path TEXT NOT NULL UNIQUE,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		artist_id INTEGER NOT NULL,
		musicbrainz_id TEXT,
		title TEXT NOT NULL,
		year INTEGER,
		overview TEXT,
		cover_path TEXT,
		path TEXT NOT NULL UNIQUE,
		FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		album_id INTEGER NOT NULL,
		musicbrainz_id TEXT,
		title TEXT NOT NULL,
		track_number INTEGER DEFAULT 1,
		disc_number INTEGER DEFAULT 1,
		duration INTEGER DEFAULT 0,
		path TEXT NOT NULL UNIQUE,
		size INTEGER,
		FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		author TEXT,
		isbn TEXT,
		publisher TEXT,
		year INTEGER,
		description TEXT,
		cover_path TEXT,
		format TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		size INTEGER,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
	);

	-- Quality presets (built-in + custom)
	CREATE TABLE IF NOT EXISTS quality_presets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		is_default INTEGER DEFAULT 0,
		is_built_in INTEGER DEFAULT 0,
		resolution TEXT NOT NULL,
		source TEXT NOT NULL,
		hdr_formats TEXT,
		codec TEXT DEFAULT 'any',
		audio_formats TEXT,
		preferred_edition TEXT DEFAULT 'any',
		min_seeders INTEGER DEFAULT 3,
		prefer_season_packs INTEGER DEFAULT 1,
		auto_upgrade INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Per-item quality override
	CREATE TABLE IF NOT EXISTS media_quality_override (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		preset_id INTEGER REFERENCES quality_presets(id),
		monitored INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Track current vs target quality
	CREATE TABLE IF NOT EXISTS media_quality_status (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		current_resolution TEXT,
		current_source TEXT,
		current_hdr TEXT,
		current_audio TEXT,
		current_edition TEXT,
		target_met INTEGER DEFAULT 0,
		upgrade_available INTEGER DEFAULT 0,
		last_search DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(media_id, media_type)
	);

	-- Download tracking
	CREATE TABLE IF NOT EXISTS downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_client_id INTEGER REFERENCES download_clients(id),
		external_id TEXT NOT NULL,
		media_id INTEGER,
		media_type TEXT,
		title TEXT NOT NULL,
		size INTEGER,
		status TEXT NOT NULL DEFAULT 'downloading',
		progress REAL DEFAULT 0,
		download_path TEXT,
		imported_path TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Naming templates
	CREATE TABLE IF NOT EXISTS naming_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		folder_template TEXT NOT NULL,
		file_template TEXT NOT NULL,
		is_default INTEGER DEFAULT 0
	);

	-- Import history
	CREATE TABLE IF NOT EXISTS import_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id INTEGER REFERENCES downloads(id),
		source_path TEXT NOT NULL,
		dest_path TEXT NOT NULL,
		media_id INTEGER,
		media_type TEXT,
		success INTEGER,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Release filters (must/must not contain)
	CREATE TABLE IF NOT EXISTS release_filters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		preset_id INTEGER REFERENCES quality_presets(id) ON DELETE CASCADE,
		filter_type TEXT NOT NULL,
		value TEXT NOT NULL,
		is_regex INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Exclusions (media and indexer)
	CREATE TABLE IF NOT EXISTS exclusions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		exclusion_type TEXT NOT NULL,
		media_id INTEGER,
		media_type TEXT,
		indexer_id INTEGER,
		library_id INTEGER,
		reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Grab history
	CREATE TABLE IF NOT EXISTS grab_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		release_title TEXT NOT NULL,
		indexer_id INTEGER,
		indexer_name TEXT,
		quality_resolution TEXT,
		quality_source TEXT,
		quality_codec TEXT,
		quality_audio TEXT,
		quality_hdr TEXT,
		release_group TEXT,
		size INTEGER,
		download_client_id INTEGER,
		download_id TEXT,
		status TEXT NOT NULL DEFAULT 'grabbed',
		error_message TEXT,
		grabbed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		imported_at DATETIME
	);

	-- Blocklist
	CREATE TABLE IF NOT EXISTS blocklist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER,
		media_type TEXT,
		release_title TEXT NOT NULL,
		release_group TEXT,
		indexer_id INTEGER,
		reason TEXT NOT NULL,
		error_message TEXT,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Delay profiles
	CREATE TABLE IF NOT EXISTS delay_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		delay_minutes INTEGER DEFAULT 0,
		bypass_if_resolution TEXT,
		bypass_if_source TEXT,
		bypass_if_score_above INTEGER,
		library_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Pending grabs (waiting for delay)
	CREATE TABLE IF NOT EXISTS pending_grabs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		release_title TEXT NOT NULL,
		release_data TEXT,
		score INTEGER,
		indexer_id INTEGER,
		available_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Blocked groups (user-configurable)
	CREATE TABLE IF NOT EXISTS blocked_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		reason TEXT,
		auto_blocked INTEGER DEFAULT 0,
		failure_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Trusted groups (user-configurable)
	CREATE TABLE IF NOT EXISTS trusted_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		category TEXT DEFAULT 'movies',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := d.db.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations for existing databases
	migrations := []string{
		"ALTER TABLE movies ADD COLUMN focal_x REAL",
		"ALTER TABLE movies ADD COLUMN focal_y REAL",
		"ALTER TABLE shows ADD COLUMN focal_x REAL",
		"ALTER TABLE shows ADD COLUMN focal_y REAL",
		"ALTER TABLE movies ADD COLUMN writer TEXT",
		"ALTER TABLE movies ADD COLUMN status TEXT",
		"ALTER TABLE movies ADD COLUMN budget INTEGER",
		"ALTER TABLE movies ADD COLUMN revenue INTEGER",
		"ALTER TABLE movies ADD COLUMN country TEXT",
		"ALTER TABLE movies ADD COLUMN original_language TEXT",
		"ALTER TABLE movies ADD COLUMN trailers TEXT",
		"ALTER TABLE movies ADD COLUMN editor TEXT",
		"ALTER TABLE movies ADD COLUMN producers TEXT",
		"ALTER TABLE movies ADD COLUMN crew TEXT",
		"ALTER TABLE shows ADD COLUMN crew TEXT",
		"ALTER TABLE movies ADD COLUMN theatrical_release TEXT",
		"ALTER TABLE movies ADD COLUMN digital_release TEXT",
		"ALTER TABLE movies ADD COLUMN studios TEXT",
		"ALTER TABLE movies ADD COLUMN last_watched_at TEXT",
		"ALTER TABLE movies ADD COLUMN play_count INTEGER DEFAULT 0",
		"ALTER TABLE shows ADD COLUMN added_at DATETIME DEFAULT CURRENT_TIMESTAMP",
		// Quality preset migrations
		"ALTER TABLE quality_presets ADD COLUMN min_resolution TEXT DEFAULT '720p'",
		"ALTER TABLE quality_presets ADD COLUMN cutoff_resolution TEXT",
		"ALTER TABLE quality_presets ADD COLUMN cutoff_source TEXT",
		"ALTER TABLE quality_presets ADD COLUMN cutoff_met_behavior TEXT DEFAULT 'stop'",
		"ALTER TABLE quality_presets ADD COLUMN sources TEXT",
		// Download tracking migrations
		"ALTER TABLE downloads ADD COLUMN retry_count INTEGER DEFAULT 0",
		"ALTER TABLE downloads ADD COLUMN last_error TEXT",
		"ALTER TABLE downloads ADD COLUMN failed_at DATETIME",
		"ALTER TABLE downloads ADD COLUMN stalled_notified INTEGER DEFAULT 0",
		// Library preset assignment
		"ALTER TABLE libraries ADD COLUMN quality_preset_id INTEGER",
	}
	for _, m := range migrations {
		// Ignore errors (column may already exist)
		d.db.Exec(m)
	}

	// Seed built-in quality presets if none exist
	var presetCount int
	d.db.QueryRow("SELECT COUNT(*) FROM quality_presets WHERE is_built_in = 1").Scan(&presetCount)
	if presetCount == 0 {
		presets := []string{
			`INSERT INTO quality_presets (name, is_built_in, is_default, resolution, source, hdr_formats, audio_formats, min_seeders, auto_upgrade) VALUES ('Best', 1, 1, '4k', 'remux', '["dv", "hdr10+", "hdr10"]', '["atmos", "truehd", "dtshd"]', 3, 1)`,
			`INSERT INTO quality_presets (name, is_built_in, resolution, source, hdr_formats, min_seeders, auto_upgrade) VALUES ('High', 1, '4k', 'web', '["dv", "hdr10+", "hdr10"]', 3, 1)`,
			`INSERT INTO quality_presets (name, is_built_in, resolution, source, min_seeders, auto_upgrade) VALUES ('Balanced', 1, '1080p', 'web', 3, 1)`,
			`INSERT INTO quality_presets (name, is_built_in, resolution, source, codec, min_seeders, auto_upgrade) VALUES ('Storage Saver', 1, '1080p', 'web', 'hevc', 3, 0)`,
		}
		for _, p := range presets {
			d.db.Exec(p)
		}
	}

	// Seed default naming templates if none exist
	var templateCount int
	d.db.QueryRow("SELECT COUNT(*) FROM naming_templates").Scan(&templateCount)
	if templateCount == 0 {
		templates := []string{
			`INSERT INTO naming_templates (type, folder_template, file_template, is_default) VALUES ('movie', '{Title} ({Year})', '{Title} ({Year})', 1)`,
			`INSERT INTO naming_templates (type, folder_template, file_template, is_default) VALUES ('tv', '{Title} ({Year})/Season {Season:00}', '{Title} - S{Season:00}E{Episode:00} - {EpisodeTitle}', 1)`,
			`INSERT INTO naming_templates (type, folder_template, file_template, is_default) VALUES ('daily', '{Title} ({Year})/Season {Year}', '{Title} - {Air-Date} - {EpisodeTitle}', 1)`,
		}
		for _, t := range templates {
			d.db.Exec(t)
		}
	}

	return nil
}

// Library operations

func (d *Database) CreateLibrary(lib *Library) error {
	result, err := d.db.Exec(
		"INSERT INTO libraries (name, path, type, scan_interval) VALUES (?, ?, ?, ?)",
		lib.Name, lib.Path, lib.Type, lib.ScanInterval,
	)
	if err != nil {
		return err
	}
	lib.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetLibraries() ([]Library, error) {
	rows, err := d.db.Query("SELECT id, name, path, type, scan_interval FROM libraries")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []Library
	for rows.Next() {
		var lib Library
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Path, &lib.Type, &lib.ScanInterval); err != nil {
			return nil, err
		}
		libraries = append(libraries, lib)
	}
	return libraries, nil
}

func (d *Database) GetLibrary(id int64) (*Library, error) {
	var lib Library
	err := d.db.QueryRow(
		"SELECT id, name, path, type, scan_interval FROM libraries WHERE id = ?", id,
	).Scan(&lib.ID, &lib.Name, &lib.Path, &lib.Type, &lib.ScanInterval)
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (d *Database) DeleteLibrary(id int64) error {
	_, err := d.db.Exec("DELETE FROM libraries WHERE id = ?", id)
	return err
}

// ClearAllLibraryData removes all movies, shows, seasons, and episodes but keeps library definitions
func (d *Database) ClearAllLibraryData() error {
	// Delete in order to respect foreign key constraints
	if _, err := d.db.Exec("DELETE FROM episodes"); err != nil {
		return err
	}
	if _, err := d.db.Exec("DELETE FROM seasons"); err != nil {
		return err
	}
	if _, err := d.db.Exec("DELETE FROM shows"); err != nil {
		return err
	}
	if _, err := d.db.Exec("DELETE FROM movies"); err != nil {
		return err
	}
	// Also clear progress and continue watching
	if _, err := d.db.Exec("DELETE FROM progress"); err != nil {
		return err
	}
	return nil
}

// Movie operations

func (d *Database) CreateMovie(movie *Movie) error {
	result, err := d.db.Exec(
		"INSERT INTO movies (library_id, title, year, path, size) VALUES (?, ?, ?, ?, ?)",
		movie.LibraryID, movie.Title, movie.Year, movie.Path, movie.Size,
	)
	if err != nil {
		return err
	}
	movie.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateMovieMetadata(movie *Movie) error {
	_, err := d.db.Exec(`
		UPDATE movies SET
			tmdb_id = ?, imdb_id = ?, original_title = ?, overview = ?, tagline = ?,
			runtime = ?, rating = ?, content_rating = ?, genres = ?, "cast" = ?, crew = ?,
			director = ?, writer = ?, editor = ?, producers = ?, status = ?, budget = ?, revenue = ?,
			country = ?, original_language = ?, theatrical_release = ?, digital_release = ?, studios = ?, trailers = ?,
			poster_path = ?, backdrop_path = ?, focal_x = ?, focal_y = ?
		WHERE id = ?`,
		movie.TmdbID, movie.ImdbID, movie.OriginalTitle, movie.Overview, movie.Tagline,
		movie.Runtime, movie.Rating, movie.ContentRating, movie.Genres, movie.Cast, movie.Crew,
		movie.Director, movie.Writer, movie.Editor, movie.Producers, movie.Status, movie.Budget, movie.Revenue,
		movie.Country, movie.OriginalLanguage, movie.TheatricalRelease, movie.DigitalRelease, movie.Studios, movie.Trailers,
		movie.PosterPath, movie.BackdropPath, movie.FocalX, movie.FocalY, movie.ID,
	)
	return err
}

func (d *Database) UpdateMoviePath(id int64, newPath string) error {
	_, err := d.db.Exec(`UPDATE movies SET path = ? WHERE id = ?`, newPath, id)
	return err
}

func (d *Database) GetMovies() ([]Movie, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
			&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
			&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
			&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
			&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount); err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// GetMovieTMDBIDs returns a set of all TMDB IDs in the movie library
func (d *Database) GetMovieTMDBIDs() (map[int64]bool, error) {
	rows, err := d.db.Query(`SELECT tmdb_id FROM movies WHERE tmdb_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

// GetShowTMDBIDs returns a set of all TMDB IDs in the TV show library
func (d *Database) GetShowTMDBIDs() (map[int64]bool, error) {
	rows, err := d.db.Query(`SELECT tmdb_id FROM shows WHERE tmdb_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

func (d *Database) GetMovieByPath(path string) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE path = ?`, path,
	).Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
		&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
		&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
		&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
		&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Show operations

func (d *Database) CreateShow(show *Show) error {
	result, err := d.db.Exec(
		"INSERT INTO shows (library_id, title, year, path) VALUES (?, ?, ?, ?)",
		show.LibraryID, show.Title, show.Year, show.Path,
	)
	if err != nil {
		return err
	}
	show.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateShowMetadata(show *Show) error {
	_, err := d.db.Exec(`
		UPDATE shows SET
			tmdb_id = ?, tvdb_id = ?, imdb_id = ?, original_title = ?, year = ?, overview = ?,
			status = ?, rating = ?, content_rating = ?, genres = ?, "cast" = ?, crew = ?,
			network = ?, poster_path = ?, backdrop_path = ?, focal_x = ?, focal_y = ?
		WHERE id = ?`,
		show.TmdbID, show.TvdbID, show.ImdbID, show.OriginalTitle, show.Year, show.Overview,
		show.Status, show.Rating, show.ContentRating, show.Genres, show.Cast, show.Crew,
		show.Network, show.PosterPath, show.BackdropPath, show.FocalX, show.FocalY, show.ID,
	)
	return err
}

func (d *Database) GetShows() ([]Show, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var s Show
		var addedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
			&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
			&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt); err != nil {
			return nil, err
		}
		if addedAt.Valid {
			s.AddedAt = &addedAt.Time
		}
		shows = append(shows, s)
	}
	return shows, nil
}

func (d *Database) GetShowByPath(path string) (*Show, error) {
	var s Show
	var addedAt sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows WHERE path = ?`, path,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
		&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt)
	if err != nil {
		return nil, err
	}
	if addedAt.Valid {
		s.AddedAt = &addedAt.Time
	}
	return &s, nil
}

func (d *Database) GetShow(id int64) (*Show, error) {
	var s Show
	var addedAt sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows WHERE id = ?`, id,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
		&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt)
	if err != nil {
		return nil, err
	}
	if addedAt.Valid {
		s.AddedAt = &addedAt.Time
	}
	return &s, nil
}

// Season operations

func (d *Database) CreateSeason(season *Season) error {
	result, err := d.db.Exec(
		"INSERT INTO seasons (show_id, season_number) VALUES (?, ?)",
		season.ShowID, season.SeasonNumber,
	)
	if err != nil {
		return err
	}
	season.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateSeasonMetadata(season *Season) error {
	_, err := d.db.Exec(`
		UPDATE seasons SET name = ?, overview = ?, poster_path = ?, air_date = ?
		WHERE id = ?`,
		season.Name, season.Overview, season.PosterPath, season.AirDate, season.ID,
	)
	return err
}

func (d *Database) GetSeasonsByShow(showID int64) ([]Season, error) {
	rows, err := d.db.Query(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE show_id = ? ORDER BY season_number`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []Season
	for rows.Next() {
		var s Season
		if err := rows.Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

func (d *Database) GetSeason(showID int64, seasonNumber int) (*Season, error) {
	var s Season
	err := d.db.QueryRow(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE show_id = ? AND season_number = ?`,
		showID, seasonNumber,
	).Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) GetSeasonByID(id int64) (*Season, error) {
	var s Season
	err := d.db.QueryRow(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE id = ?`, id,
	).Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Episode operations

func (d *Database) CreateEpisode(ep *Episode) error {
	result, err := d.db.Exec(
		"INSERT INTO episodes (season_id, episode_number, title, path, size) VALUES (?, ?, ?, ?, ?)",
		ep.SeasonID, ep.EpisodeNumber, ep.Title, ep.Path, ep.Size,
	)
	if err != nil {
		return err
	}
	ep.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateEpisodeMetadata(ep *Episode) error {
	_, err := d.db.Exec(`
		UPDATE episodes SET title = ?, overview = ?, air_date = ?, runtime = ?, still_path = ?
		WHERE id = ?`,
		ep.Title, ep.Overview, ep.AirDate, ep.Runtime, ep.StillPath, ep.ID,
	)
	return err
}

func (d *Database) GetEpisodesBySeason(seasonID int64) ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE season_id = ? ORDER BY episode_number`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size); err != nil {
			return nil, err
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

func (d *Database) GetEpisodeByPath(path string) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE path = ?`, path,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) GetEpisode(id int64) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE id = ?`, id,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) DeleteEpisode(id int64) error {
	_, err := d.db.Exec("DELETE FROM episodes WHERE id = ?", id)
	return err
}

func (d *Database) UpdateEpisodeSize(id int64, size int64) error {
	_, err := d.db.Exec("UPDATE episodes SET size = ? WHERE id = ?", size, id)
	return err
}

func (d *Database) GetEpisodesWithMissingSize() ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE size = 0 OR size IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size); err != nil {
			return nil, err
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

func (d *Database) GetMovie(id int64) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE id = ?`, id,
	).Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
		&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
		&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
		&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
		&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMoviePlayCount increments the play count and updates last watched time
func (d *Database) UpdateMoviePlayCount(id int64) error {
	now := time.Now().Format(time.RFC3339)
	_, err := d.db.Exec(`
		UPDATE movies SET
			play_count = play_count + 1,
			last_watched_at = ?
		WHERE id = ?`, now, id)
	return err
}

// Settings operations

func (d *Database) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (d *Database) SetSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

func (d *Database) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

// Progress operations

func (d *Database) GetProgress(mediaType string, mediaID int64) (*Progress, error) {
	var p Progress
	err := d.db.QueryRow(
		"SELECT id, media_type, media_id, position, duration, updated_at FROM progress WHERE media_type = ? AND media_id = ?",
		mediaType, mediaID,
	).Scan(&p.ID, &p.MediaType, &p.MediaID, &p.Position, &p.Duration, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *Database) SaveProgress(p *Progress) error {
	_, err := d.db.Exec(`
		INSERT INTO progress (media_type, media_id, position, duration, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(media_type, media_id) DO UPDATE SET
			position = excluded.position,
			duration = excluded.duration,
			updated_at = CURRENT_TIMESTAMP
	`, p.MediaType, p.MediaID, p.Position, p.Duration)
	return err
}

// User operations

func (d *Database) CreateUser(user *User) error {
	result, err := d.db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		user.Username, user.PasswordHash, user.Role,
	)
	if err != nil {
		return err
	}
	user.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	var u User
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *Database) GetUserByID(id int64) (*User, error) {
	var u User
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *Database) GetUsers() ([]User, error) {
	rows, err := d.db.Query("SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) UpdateUser(user *User) error {
	_, err := d.db.Exec(
		"UPDATE users SET username = ?, role = ? WHERE id = ?",
		user.Username, user.Role, user.ID,
	)
	return err
}

func (d *Database) UpdateUserPassword(id int64, passwordHash string) error {
	_, err := d.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	return err
}

func (d *Database) DeleteUser(id int64) error {
	_, err := d.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (d *Database) CountUsers() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// Session operations

func (d *Database) CreateSession(session *Session) error {
	result, err := d.db.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		session.UserID, session.Token, session.ExpiresAt,
	)
	if err != nil {
		return err
	}
	session.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetSessionByToken(token string) (*Session, error) {
	var s Session
	err := d.db.QueryRow(
		"SELECT id, user_id, token, expires_at FROM sessions WHERE token = ?", token,
	).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) DeleteSession(token string) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (d *Database) DeleteExpiredSessions() error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (d *Database) DeleteUserSessions(userID int64) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// Download Client operations

func (d *Database) CreateDownloadClient(client *DownloadClient) error {
	result, err := d.db.Exec(`
		INSERT INTO download_clients (name, type, host, port, username, password, api_key, use_tls, category, priority, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		client.Name, client.Type, client.Host, client.Port, client.Username, client.Password,
		client.APIKey, client.UseTLS, client.Category, client.Priority, client.Enabled,
	)
	if err != nil {
		return err
	}
	client.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetDownloadClients() ([]DownloadClient, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, host, port, username, password, api_key, use_tls, category, priority, enabled
		FROM download_clients ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []DownloadClient
	for rows.Next() {
		var c DownloadClient
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Host, &c.Port, &c.Username, &c.Password,
			&c.APIKey, &c.UseTLS, &c.Category, &c.Priority, &c.Enabled); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (d *Database) GetDownloadClient(id int64) (*DownloadClient, error) {
	var c DownloadClient
	err := d.db.QueryRow(`
		SELECT id, name, type, host, port, username, password, api_key, use_tls, category, priority, enabled
		FROM download_clients WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Type, &c.Host, &c.Port, &c.Username, &c.Password,
		&c.APIKey, &c.UseTLS, &c.Category, &c.Priority, &c.Enabled)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *Database) UpdateDownloadClient(client *DownloadClient) error {
	_, err := d.db.Exec(`
		UPDATE download_clients SET
			name = ?, type = ?, host = ?, port = ?, username = ?, password = ?,
			api_key = ?, use_tls = ?, category = ?, priority = ?, enabled = ?
		WHERE id = ?`,
		client.Name, client.Type, client.Host, client.Port, client.Username, client.Password,
		client.APIKey, client.UseTLS, client.Category, client.Priority, client.Enabled, client.ID,
	)
	return err
}

func (d *Database) DeleteDownloadClient(id int64) error {
	_, err := d.db.Exec("DELETE FROM download_clients WHERE id = ?", id)
	return err
}

func (d *Database) GetEnabledDownloadClients() ([]DownloadClient, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, host, port, username, password, api_key, use_tls, category, priority, enabled
		FROM download_clients WHERE enabled = 1 ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []DownloadClient
	for rows.Next() {
		var c DownloadClient
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Host, &c.Port, &c.Username, &c.Password,
			&c.APIKey, &c.UseTLS, &c.Category, &c.Priority, &c.Enabled); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// Indexer operations

func (d *Database) CreateIndexer(indexer *Indexer) error {
	result, err := d.db.Exec(`
		INSERT INTO indexers (name, type, url, api_key, categories, priority, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		indexer.Name, indexer.Type, indexer.URL, indexer.APIKey,
		indexer.Categories, indexer.Priority, indexer.Enabled,
	)
	if err != nil {
		return err
	}
	indexer.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetIndexers() ([]Indexer, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, url, api_key, categories, priority, enabled
		FROM indexers ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled); err != nil {
			return nil, err
		}
		indexers = append(indexers, i)
	}
	return indexers, nil
}

func (d *Database) GetIndexer(id int64) (*Indexer, error) {
	var i Indexer
	err := d.db.QueryRow(`
		SELECT id, name, type, url, api_key, categories, priority, enabled
		FROM indexers WHERE id = ?`, id,
	).Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
		&i.Categories, &i.Priority, &i.Enabled)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (d *Database) UpdateIndexer(indexer *Indexer) error {
	_, err := d.db.Exec(`
		UPDATE indexers SET
			name = ?, type = ?, url = ?, api_key = ?, categories = ?, priority = ?, enabled = ?
		WHERE id = ?`,
		indexer.Name, indexer.Type, indexer.URL, indexer.APIKey,
		indexer.Categories, indexer.Priority, indexer.Enabled, indexer.ID,
	)
	return err
}

func (d *Database) DeleteIndexer(id int64) error {
	_, err := d.db.Exec("DELETE FROM indexers WHERE id = ?", id)
	return err
}

func (d *Database) GetEnabledIndexers() ([]Indexer, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, url, api_key, categories, priority, enabled
		FROM indexers WHERE enabled = 1 ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled); err != nil {
			return nil, err
		}
		indexers = append(indexers, i)
	}
	return indexers, nil
}

// Quality Profile operations

func (d *Database) CreateQualityProfile(profile *QualityProfile) error {
	result, err := d.db.Exec(`
		INSERT INTO quality_profiles (name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		profile.Name, profile.UpgradeAllowed, profile.UpgradeUntilScore,
		profile.MinFormatScore, profile.CutoffFormatScore, profile.Qualities, profile.CustomFormatScores,
	)
	if err != nil {
		return err
	}
	profile.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetQualityProfiles() ([]QualityProfile, error) {
	rows, err := d.db.Query(`
		SELECT id, name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores
		FROM quality_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []QualityProfile
	for rows.Next() {
		var p QualityProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.UpgradeAllowed, &p.UpgradeUntilScore,
			&p.MinFormatScore, &p.CutoffFormatScore, &p.Qualities, &p.CustomFormatScores); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) GetQualityProfile(id int64) (*QualityProfile, error) {
	var p QualityProfile
	err := d.db.QueryRow(`
		SELECT id, name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores
		FROM quality_profiles WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.UpgradeAllowed, &p.UpgradeUntilScore,
		&p.MinFormatScore, &p.CutoffFormatScore, &p.Qualities, &p.CustomFormatScores)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *Database) UpdateQualityProfile(profile *QualityProfile) error {
	_, err := d.db.Exec(`
		UPDATE quality_profiles SET
			name = ?, upgrade_allowed = ?, upgrade_until_score = ?, min_format_score = ?, cutoff_format_score = ?, qualities = ?, custom_format_scores = ?
		WHERE id = ?`,
		profile.Name, profile.UpgradeAllowed, profile.UpgradeUntilScore,
		profile.MinFormatScore, profile.CutoffFormatScore, profile.Qualities, profile.CustomFormatScores, profile.ID,
	)
	return err
}

func (d *Database) DeleteQualityProfile(id int64) error {
	_, err := d.db.Exec("DELETE FROM quality_profiles WHERE id = ?", id)
	return err
}

// Custom Format operations

func (d *Database) CreateCustomFormat(format *CustomFormat) error {
	result, err := d.db.Exec(`
		INSERT INTO custom_formats (name, conditions) VALUES (?, ?)`,
		format.Name, format.Conditions,
	)
	if err != nil {
		return err
	}
	format.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetCustomFormats() ([]CustomFormat, error) {
	rows, err := d.db.Query(`SELECT id, name, conditions FROM custom_formats ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var formats []CustomFormat
	for rows.Next() {
		var f CustomFormat
		if err := rows.Scan(&f.ID, &f.Name, &f.Conditions); err != nil {
			return nil, err
		}
		formats = append(formats, f)
	}
	return formats, nil
}

func (d *Database) GetCustomFormat(id int64) (*CustomFormat, error) {
	var f CustomFormat
	err := d.db.QueryRow(`SELECT id, name, conditions FROM custom_formats WHERE id = ?`, id).Scan(&f.ID, &f.Name, &f.Conditions)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *Database) UpdateCustomFormat(format *CustomFormat) error {
	_, err := d.db.Exec(`UPDATE custom_formats SET name = ?, conditions = ? WHERE id = ?`,
		format.Name, format.Conditions, format.ID)
	return err
}

func (d *Database) DeleteCustomFormat(id int64) error {
	_, err := d.db.Exec("DELETE FROM custom_formats WHERE id = ?", id)
	return err
}

// Wanted operations

func (d *Database) CreateWantedItem(item *WantedItem) error {
	result, err := d.db.Exec(`
		INSERT INTO wanted (type, tmdb_id, title, year, poster_path, quality_profile_id, monitored, seasons)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Type, item.TmdbID, item.Title, item.Year, item.PosterPath,
		item.QualityProfileID, item.Monitored, item.Seasons,
	)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetWantedItems() ([]WantedItem, error) {
	rows, err := d.db.Query(`
		SELECT id, type, tmdb_id, title, year, poster_path, quality_profile_id, monitored, seasons, last_searched, added_at
		FROM wanted ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WantedItem
	for rows.Next() {
		var item WantedItem
		if err := rows.Scan(&item.ID, &item.Type, &item.TmdbID, &item.Title, &item.Year,
			&item.PosterPath, &item.QualityProfileID, &item.Monitored, &item.Seasons,
			&item.LastSearched, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) GetWantedItem(id int64) (*WantedItem, error) {
	var item WantedItem
	err := d.db.QueryRow(`
		SELECT id, type, tmdb_id, title, year, poster_path, quality_profile_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE id = ?`, id,
	).Scan(&item.ID, &item.Type, &item.TmdbID, &item.Title, &item.Year,
		&item.PosterPath, &item.QualityProfileID, &item.Monitored, &item.Seasons,
		&item.LastSearched, &item.AddedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *Database) GetWantedByTmdb(itemType string, tmdbID int64) (*WantedItem, error) {
	var item WantedItem
	err := d.db.QueryRow(`
		SELECT id, type, tmdb_id, title, year, poster_path, quality_profile_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE type = ? AND tmdb_id = ?`, itemType, tmdbID,
	).Scan(&item.ID, &item.Type, &item.TmdbID, &item.Title, &item.Year,
		&item.PosterPath, &item.QualityProfileID, &item.Monitored, &item.Seasons,
		&item.LastSearched, &item.AddedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *Database) GetMonitoredItems() ([]WantedItem, error) {
	rows, err := d.db.Query(`
		SELECT id, type, tmdb_id, title, year, poster_path, quality_profile_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE monitored = 1 ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WantedItem
	for rows.Next() {
		var item WantedItem
		if err := rows.Scan(&item.ID, &item.Type, &item.TmdbID, &item.Title, &item.Year,
			&item.PosterPath, &item.QualityProfileID, &item.Monitored, &item.Seasons,
			&item.LastSearched, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) UpdateWantedItem(item *WantedItem) error {
	_, err := d.db.Exec(`
		UPDATE wanted SET
			quality_profile_id = ?, monitored = ?, seasons = ?
		WHERE id = ?`,
		item.QualityProfileID, item.Monitored, item.Seasons, item.ID,
	)
	return err
}

func (d *Database) UpdateWantedLastSearched(id int64) error {
	_, err := d.db.Exec("UPDATE wanted SET last_searched = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (d *Database) DeleteWantedItem(id int64) error {
	_, err := d.db.Exec("DELETE FROM wanted WHERE id = ?", id)
	return err
}

// Request operations

func (d *Database) CreateRequest(req *Request) error {
	result, err := d.db.Exec(`
		INSERT INTO requests (user_id, type, tmdb_id, title, year, overview, poster_path, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.UserID, req.Type, req.TmdbID, req.Title, req.Year, req.Overview, req.PosterPath, "requested",
	)
	if err != nil {
		return err
	}
	req.ID, _ = result.LastInsertId()
	req.Status = "requested"
	req.RequestedAt = time.Now()
	req.UpdatedAt = time.Now()
	return nil
}

func (d *Database) GetRequests() ([]Request, error) {
	rows, err := d.db.Query(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		ORDER BY r.requested_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID, &req.Title,
			&req.Year, &req.Overview, &req.PosterPath, &req.Status, &req.StatusReason,
			&req.RequestedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (d *Database) GetRequestsByUser(userID int64) ([]Request, error) {
	rows, err := d.db.Query(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.user_id = ?
		ORDER BY r.requested_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID, &req.Title,
			&req.Year, &req.Overview, &req.PosterPath, &req.Status, &req.StatusReason,
			&req.RequestedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (d *Database) GetRequestsByStatus(status string) ([]Request, error) {
	rows, err := d.db.Query(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.status = ?
		ORDER BY r.requested_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID, &req.Title,
			&req.Year, &req.Overview, &req.PosterPath, &req.Status, &req.StatusReason,
			&req.RequestedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (d *Database) GetRequest(id int64) (*Request, error) {
	var req Request
	err := d.db.QueryRow(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.id = ?`, id).Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID,
		&req.Title, &req.Year, &req.Overview, &req.PosterPath, &req.Status, &req.StatusReason,
		&req.RequestedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (d *Database) GetRequestByTmdb(userID int64, mediaType string, tmdbID int64) (*Request, error) {
	var req Request
	err := d.db.QueryRow(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.user_id = ? AND r.type = ? AND r.tmdb_id = ?`,
		userID, mediaType, tmdbID).Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID,
		&req.Title, &req.Year, &req.Overview, &req.PosterPath, &req.Status, &req.StatusReason,
		&req.RequestedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (d *Database) UpdateRequestStatus(id int64, status string, reason *string) error {
	_, err := d.db.Exec(`
		UPDATE requests
		SET status = ?, status_reason = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, status, reason, id)
	return err
}

func (d *Database) DeleteRequest(id int64) error {
	_, err := d.db.Exec("DELETE FROM requests WHERE id = ?", id)
	return err
}

// Artist operations

func (d *Database) CreateArtist(artist *Artist) error {
	result, err := d.db.Exec(
		"INSERT INTO artists (library_id, name, path) VALUES (?, ?, ?)",
		artist.LibraryID, artist.Name, artist.Path,
	)
	if err != nil {
		return err
	}
	artist.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateArtistMetadata(artist *Artist) error {
	_, err := d.db.Exec(`
		UPDATE artists SET musicbrainz_id = ?, sort_name = ?, overview = ?, image_path = ?
		WHERE id = ?`,
		artist.MusicBrainzID, artist.SortName, artist.Overview, artist.ImagePath, artist.ID,
	)
	return err
}

func (d *Database) GetArtists() ([]Artist, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, musicbrainz_id, name, sort_name, overview, image_path, path
		FROM artists ORDER BY COALESCE(sort_name, name)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []Artist
	for rows.Next() {
		var a Artist
		if err := rows.Scan(&a.ID, &a.LibraryID, &a.MusicBrainzID, &a.Name, &a.SortName, &a.Overview, &a.ImagePath, &a.Path); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, nil
}

func (d *Database) GetArtist(id int64) (*Artist, error) {
	var a Artist
	err := d.db.QueryRow(`
		SELECT id, library_id, musicbrainz_id, name, sort_name, overview, image_path, path
		FROM artists WHERE id = ?`, id,
	).Scan(&a.ID, &a.LibraryID, &a.MusicBrainzID, &a.Name, &a.SortName, &a.Overview, &a.ImagePath, &a.Path)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *Database) GetArtistByPath(path string) (*Artist, error) {
	var a Artist
	err := d.db.QueryRow(`
		SELECT id, library_id, musicbrainz_id, name, sort_name, overview, image_path, path
		FROM artists WHERE path = ?`, path,
	).Scan(&a.ID, &a.LibraryID, &a.MusicBrainzID, &a.Name, &a.SortName, &a.Overview, &a.ImagePath, &a.Path)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Album operations

func (d *Database) CreateAlbum(album *Album) error {
	result, err := d.db.Exec(
		"INSERT INTO albums (artist_id, title, year, path) VALUES (?, ?, ?, ?)",
		album.ArtistID, album.Title, album.Year, album.Path,
	)
	if err != nil {
		return err
	}
	album.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateAlbumMetadata(album *Album) error {
	_, err := d.db.Exec(`
		UPDATE albums SET musicbrainz_id = ?, overview = ?, cover_path = ?
		WHERE id = ?`,
		album.MusicBrainzID, album.Overview, album.CoverPath, album.ID,
	)
	return err
}

func (d *Database) GetAlbumsByArtist(artistID int64) ([]Album, error) {
	rows, err := d.db.Query(`
		SELECT id, artist_id, musicbrainz_id, title, year, overview, cover_path, path
		FROM albums WHERE artist_id = ? ORDER BY year, title`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []Album
	for rows.Next() {
		var a Album
		if err := rows.Scan(&a.ID, &a.ArtistID, &a.MusicBrainzID, &a.Title, &a.Year, &a.Overview, &a.CoverPath, &a.Path); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func (d *Database) GetAlbums() ([]Album, error) {
	rows, err := d.db.Query(`
		SELECT id, artist_id, musicbrainz_id, title, year, overview, cover_path, path
		FROM albums ORDER BY year DESC, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []Album
	for rows.Next() {
		var a Album
		if err := rows.Scan(&a.ID, &a.ArtistID, &a.MusicBrainzID, &a.Title, &a.Year, &a.Overview, &a.CoverPath, &a.Path); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func (d *Database) GetAlbum(id int64) (*Album, error) {
	var a Album
	err := d.db.QueryRow(`
		SELECT id, artist_id, musicbrainz_id, title, year, overview, cover_path, path
		FROM albums WHERE id = ?`, id,
	).Scan(&a.ID, &a.ArtistID, &a.MusicBrainzID, &a.Title, &a.Year, &a.Overview, &a.CoverPath, &a.Path)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *Database) GetAlbumByPath(path string) (*Album, error) {
	var a Album
	err := d.db.QueryRow(`
		SELECT id, artist_id, musicbrainz_id, title, year, overview, cover_path, path
		FROM albums WHERE path = ?`, path,
	).Scan(&a.ID, &a.ArtistID, &a.MusicBrainzID, &a.Title, &a.Year, &a.Overview, &a.CoverPath, &a.Path)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Track operations

func (d *Database) CreateTrack(track *Track) error {
	result, err := d.db.Exec(
		"INSERT INTO tracks (album_id, title, track_number, disc_number, duration, path, size) VALUES (?, ?, ?, ?, ?, ?, ?)",
		track.AlbumID, track.Title, track.TrackNumber, track.DiscNumber, track.Duration, track.Path, track.Size,
	)
	if err != nil {
		return err
	}
	track.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetTracksByAlbum(albumID int64) ([]Track, error) {
	rows, err := d.db.Query(`
		SELECT id, album_id, musicbrainz_id, title, track_number, disc_number, duration, path, size
		FROM tracks WHERE album_id = ? ORDER BY disc_number, track_number`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.MusicBrainzID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.Duration, &t.Path, &t.Size); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

func (d *Database) GetTrack(id int64) (*Track, error) {
	var t Track
	err := d.db.QueryRow(`
		SELECT id, album_id, musicbrainz_id, title, track_number, disc_number, duration, path, size
		FROM tracks WHERE id = ?`, id,
	).Scan(&t.ID, &t.AlbumID, &t.MusicBrainzID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.Duration, &t.Path, &t.Size)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *Database) GetTrackByPath(path string) (*Track, error) {
	var t Track
	err := d.db.QueryRow(`
		SELECT id, album_id, musicbrainz_id, title, track_number, disc_number, duration, path, size
		FROM tracks WHERE path = ?`, path,
	).Scan(&t.ID, &t.AlbumID, &t.MusicBrainzID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.Duration, &t.Path, &t.Size)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Book operations

func (d *Database) CreateBook(book *Book) error {
	result, err := d.db.Exec(
		"INSERT INTO books (library_id, title, author, format, path, size) VALUES (?, ?, ?, ?, ?, ?)",
		book.LibraryID, book.Title, book.Author, book.Format, book.Path, book.Size,
	)
	if err != nil {
		return err
	}
	book.ID, _ = result.LastInsertId()
	book.AddedAt = time.Now()
	return nil
}

func (d *Database) UpdateBookMetadata(book *Book) error {
	_, err := d.db.Exec(`
		UPDATE books SET isbn = ?, publisher = ?, year = ?, description = ?, cover_path = ?
		WHERE id = ?`,
		book.ISBN, book.Publisher, book.Year, book.Description, book.CoverPath, book.ID,
	)
	return err
}

func (d *Database) GetBooks() ([]Book, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, title, author, isbn, publisher, year, description, cover_path, format, path, size, added_at
		FROM books ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.LibraryID, &b.Title, &b.Author, &b.ISBN, &b.Publisher, &b.Year, &b.Description, &b.CoverPath, &b.Format, &b.Path, &b.Size, &b.AddedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, nil
}

func (d *Database) GetBook(id int64) (*Book, error) {
	var b Book
	err := d.db.QueryRow(`
		SELECT id, library_id, title, author, isbn, publisher, year, description, cover_path, format, path, size, added_at
		FROM books WHERE id = ?`, id,
	).Scan(&b.ID, &b.LibraryID, &b.Title, &b.Author, &b.ISBN, &b.Publisher, &b.Year, &b.Description, &b.CoverPath, &b.Format, &b.Path, &b.Size, &b.AddedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (d *Database) GetBookByPath(path string) (*Book, error) {
	var b Book
	err := d.db.QueryRow(`
		SELECT id, library_id, title, author, isbn, publisher, year, description, cover_path, format, path, size, added_at
		FROM books WHERE path = ?`, path,
	).Scan(&b.ID, &b.LibraryID, &b.Title, &b.Author, &b.ISBN, &b.Publisher, &b.Year, &b.Description, &b.CoverPath, &b.Format, &b.Path, &b.Size, &b.AddedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ContinueWatchingItem represents an in-progress media item
type ContinueWatchingItem struct {
	MediaType       string   `json:"mediaType"` // movie or episode
	MediaID         int64    `json:"mediaId"`
	Title           string   `json:"title"`
	Subtitle        *string  `json:"subtitle,omitempty"` // For episodes: "S1 E4 · Episode Name"
	ShowTitle       *string  `json:"showTitle,omitempty"`
	Season          *int     `json:"season,omitempty"`
	Episode         *int     `json:"episode,omitempty"`
	PosterPath      *string  `json:"posterPath,omitempty"`
	BackdropPath    *string  `json:"backdropPath,omitempty"`
	Position        float64  `json:"position"`        // seconds
	Duration        float64  `json:"duration"`        // seconds
	ProgressPercent float64  `json:"progressPercent"` // 0-100
	UpdatedAt       string   `json:"updatedAt"`
}

// GetContinueWatching returns in-progress items (position > 0 and not completed)
func (d *Database) GetContinueWatching(limit int) ([]ContinueWatchingItem, error) {
	if limit <= 0 {
		limit = 20
	}

	// Query for movies in progress
	movieRows, err := d.db.Query(`
		SELECT p.media_id, m.title, m.poster_path, m.backdrop_path, p.position, p.duration, p.updated_at
		FROM progress p
		JOIN movies m ON p.media_id = m.id
		WHERE p.media_type = 'movie'
		  AND p.position > 0
		  AND p.duration > 0
		  AND (p.position / p.duration) < 0.95
		ORDER BY p.updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer movieRows.Close()

	var items []ContinueWatchingItem
	for movieRows.Next() {
		var item ContinueWatchingItem
		var updatedAt time.Time
		if err := movieRows.Scan(&item.MediaID, &item.Title, &item.PosterPath, &item.BackdropPath,
			&item.Position, &item.Duration, &updatedAt); err != nil {
			return nil, err
		}
		item.MediaType = "movie"
		item.ProgressPercent = (item.Position / item.Duration) * 100
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}

	// Query for episodes in progress
	episodeRows, err := d.db.Query(`
		SELECT p.media_id, e.title, e.still_path, e.episode_number,
			   s.season_number, sh.title, sh.poster_path, sh.backdrop_path,
			   p.position, p.duration, p.updated_at
		FROM progress p
		JOIN episodes e ON p.media_id = e.id
		JOIN seasons s ON e.season_id = s.id
		JOIN shows sh ON s.show_id = sh.id
		WHERE p.media_type = 'episode'
		  AND p.position > 0
		  AND p.duration > 0
		  AND (p.position / p.duration) < 0.95
		ORDER BY p.updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer episodeRows.Close()

	for episodeRows.Next() {
		var item ContinueWatchingItem
		var episodeTitle, stillPath *string
		var episodeNum, seasonNum int
		var showTitle string
		var updatedAt time.Time
		if err := episodeRows.Scan(&item.MediaID, &episodeTitle, &stillPath, &episodeNum,
			&seasonNum, &showTitle, &item.PosterPath, &item.BackdropPath,
			&item.Position, &item.Duration, &updatedAt); err != nil {
			return nil, err
		}
		item.MediaType = "episode"
		item.Title = showTitle
		item.ShowTitle = &showTitle
		item.Season = &seasonNum
		item.Episode = &episodeNum
		// Create subtitle: "S1 E4 · Episode Name"
		epTitle := ""
		if episodeTitle != nil {
			epTitle = *episodeTitle
		}
		subtitle := "S" + strconv.Itoa(seasonNum) + " E" + strconv.Itoa(episodeNum)
		if epTitle != "" {
			subtitle += " · " + epTitle
		}
		item.Subtitle = &subtitle
		// Use still path as backdrop if available
		if stillPath != nil && *stillPath != "" {
			item.BackdropPath = stillPath
		}
		item.ProgressPercent = (item.Position / item.Duration) * 100
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}

	// Sort by UpdatedAt descending (merge the two lists)
	// Simple bubble sort for small lists
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].UpdatedAt > items[i].UpdatedAt {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Limit the final result
	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// DeleteProgress removes progress for a specific media item
func (d *Database) DeleteProgress(mediaType string, mediaID int64) error {
	_, err := d.db.Exec("DELETE FROM progress WHERE media_type = ? AND media_id = ?", mediaType, mediaID)
	return err
}

// MarkAsWatched sets progress to 100% complete
func (d *Database) MarkAsWatched(mediaType string, mediaID int64, duration float64) error {
	_, err := d.db.Exec(`
		INSERT INTO progress (media_type, media_id, position, duration, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(media_type, media_id) DO UPDATE SET
			position = excluded.position,
			duration = excluded.duration,
			updated_at = CURRENT_TIMESTAMP
	`, mediaType, mediaID, duration, duration)
	return err
}

// MarkAsUnwatched removes progress entry (resets to unwatched)
func (d *Database) MarkAsUnwatched(mediaType string, mediaID int64) error {
	_, err := d.db.Exec(`DELETE FROM progress WHERE media_type = ? AND media_id = ?`, mediaType, mediaID)
	return err
}

// GetWatchedStatus returns whether an item is watched and its progress
func (d *Database) GetWatchedStatus(mediaType string, mediaID int64) (bool, float64, error) {
	var position, duration float64
	err := d.db.QueryRow(`
		SELECT position, duration FROM progress
		WHERE media_type = ? AND media_id = ?
	`, mediaType, mediaID).Scan(&position, &duration)
	if err != nil {
		return false, 0, nil // Not watched
	}
	// Consider watched if > 90% complete
	if duration > 0 && position/duration >= 0.9 {
		return true, position / duration * 100, nil
	}
	return false, position / duration * 100, nil
}

// MovieWatchState represents watch state for a movie
type MovieWatchState struct {
	WatchState string  `json:"watchState"` // "unwatched", "partial", "watched"
	Progress   float64 `json:"progress"`   // 0-100
}

// ShowWatchState represents watch state for a show
type ShowWatchState struct {
	WatchState      string `json:"watchState"` // "unwatched", "partial", "watched"
	WatchedEpisodes int    `json:"watchedEpisodes"`
	TotalEpisodes   int    `json:"totalEpisodes"`
}

// GetAllMovieWatchStates returns watch states for all movies
func (d *Database) GetAllMovieWatchStates() (map[int64]MovieWatchState, error) {
	states := make(map[int64]MovieWatchState)

	rows, err := d.db.Query(`
		SELECT media_id, position, duration FROM progress
		WHERE media_type = 'movie'
	`)
	if err != nil {
		return states, err
	}
	defer rows.Close()

	for rows.Next() {
		var mediaID int64
		var position, duration float64
		if err := rows.Scan(&mediaID, &position, &duration); err != nil {
			continue
		}

		state := MovieWatchState{}
		if duration > 0 {
			progress := position / duration
			state.Progress = progress * 100
			if progress >= 0.9 {
				state.WatchState = "watched"
			} else if progress > 0.05 {
				state.WatchState = "partial"
			} else {
				state.WatchState = "unwatched"
			}
		}
		states[mediaID] = state
	}

	return states, nil
}

// GetAllShowWatchStates returns watch states for all shows based on episode progress
func (d *Database) GetAllShowWatchStates() (map[int64]ShowWatchState, error) {
	states := make(map[int64]ShowWatchState)

	// First, get total episode counts per show
	rows, err := d.db.Query(`
		SELECT s.id, COUNT(e.id) as total_episodes
		FROM shows s
		LEFT JOIN seasons sea ON sea.show_id = s.id
		LEFT JOIN episodes e ON e.season_id = sea.id
		GROUP BY s.id
	`)
	if err != nil {
		return states, err
	}
	defer rows.Close()

	for rows.Next() {
		var showID int64
		var totalEpisodes int
		if err := rows.Scan(&showID, &totalEpisodes); err != nil {
			continue
		}
		states[showID] = ShowWatchState{
			TotalEpisodes: totalEpisodes,
			WatchState:    "unwatched",
		}
	}

	// Now get watched episode counts (episodes with >= 90% progress)
	watchedRows, err := d.db.Query(`
		SELECT s.id, COUNT(DISTINCT p.media_id) as watched_episodes
		FROM shows s
		JOIN seasons sea ON sea.show_id = s.id
		JOIN episodes e ON e.season_id = sea.id
		JOIN progress p ON p.media_type = 'episode' AND p.media_id = e.id
		WHERE p.duration > 0 AND (p.position / p.duration) >= 0.9
		GROUP BY s.id
	`)
	if err != nil {
		return states, nil // Return what we have
	}
	defer watchedRows.Close()

	for watchedRows.Next() {
		var showID int64
		var watchedEpisodes int
		if err := watchedRows.Scan(&showID, &watchedEpisodes); err != nil {
			continue
		}

		if state, exists := states[showID]; exists {
			state.WatchedEpisodes = watchedEpisodes
			if watchedEpisodes >= state.TotalEpisodes && state.TotalEpisodes > 0 {
				state.WatchState = "watched"
			} else if watchedEpisodes > 0 {
				state.WatchState = "partial"
			}
			states[showID] = state
		}
	}

	return states, nil
}

// ItemStatus represents the library/request status of an item
type ItemStatus struct {
	InLibrary    bool    `json:"inLibrary"`
	LibraryID    *int64  `json:"libraryId,omitempty"`
	Requested    bool    `json:"requested"`
	RequestID    *int64  `json:"requestId,omitempty"`
	RequestStatus *string `json:"requestStatus,omitempty"`
}

// GetMovieStatusByTmdbID checks if a movie is in library or requested
func (d *Database) GetMovieStatusByTmdbID(tmdbID int64) (*ItemStatus, error) {
	status := &ItemStatus{}

	// Check if in library
	var libraryID int64
	err := d.db.QueryRow("SELECT id FROM movies WHERE tmdb_id = ?", tmdbID).Scan(&libraryID)
	if err == nil {
		status.InLibrary = true
		status.LibraryID = &libraryID
	}

	// Check if requested
	var requestID int64
	var requestStatus string
	err = d.db.QueryRow("SELECT id, status FROM requests WHERE type = 'movie' AND tmdb_id = ?", tmdbID).Scan(&requestID, &requestStatus)
	if err == nil {
		status.Requested = true
		status.RequestID = &requestID
		status.RequestStatus = &requestStatus
	}

	return status, nil
}

// GetShowStatusByTmdbID checks if a show is in library or requested
func (d *Database) GetShowStatusByTmdbID(tmdbID int64) (*ItemStatus, error) {
	status := &ItemStatus{}

	// Check if in library
	var libraryID int64
	err := d.db.QueryRow("SELECT id FROM shows WHERE tmdb_id = ?", tmdbID).Scan(&libraryID)
	if err == nil {
		status.InLibrary = true
		status.LibraryID = &libraryID
	}

	// Check if requested
	var requestID int64
	var requestStatus string
	err = d.db.QueryRow("SELECT id, status FROM requests WHERE type = 'tv' AND tmdb_id = ?", tmdbID).Scan(&requestID, &requestStatus)
	if err == nil {
		status.Requested = true
		status.RequestID = &requestID
		status.RequestStatus = &requestStatus
	}

	return status, nil
}

// GetBulkMovieStatus checks status for multiple TMDB IDs at once
func (d *Database) GetBulkMovieStatus(tmdbIDs []int64) (map[int64]*ItemStatus, error) {
	result := make(map[int64]*ItemStatus)
	for _, id := range tmdbIDs {
		result[id] = &ItemStatus{}
	}

	if len(tmdbIDs) == 0 {
		return result, nil
	}

	// Build placeholder string
	placeholders := make([]string, len(tmdbIDs))
	args := make([]interface{}, len(tmdbIDs))
	for i, id := range tmdbIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	// Check library status
	rows, err := d.db.Query("SELECT id, tmdb_id FROM movies WHERE tmdb_id IN ("+placeholderStr+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, tmdbID int64
		if err := rows.Scan(&id, &tmdbID); err != nil {
			return nil, err
		}
		if status, ok := result[tmdbID]; ok {
			status.InLibrary = true
			libID := id
			status.LibraryID = &libID
		}
	}

	// Check request status
	rows2, err := d.db.Query("SELECT id, tmdb_id, status FROM requests WHERE type = 'movie' AND tmdb_id IN ("+placeholderStr+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, tmdbID int64
		var reqStatus string
		if err := rows2.Scan(&id, &tmdbID, &reqStatus); err != nil {
			return nil, err
		}
		if status, ok := result[tmdbID]; ok {
			status.Requested = true
			reqID := id
			status.RequestID = &reqID
			status.RequestStatus = &reqStatus
		}
	}

	return result, nil
}

// GetBulkShowStatus checks status for multiple TMDB IDs at once
func (d *Database) GetBulkShowStatus(tmdbIDs []int64) (map[int64]*ItemStatus, error) {
	result := make(map[int64]*ItemStatus)
	for _, id := range tmdbIDs {
		result[id] = &ItemStatus{}
	}

	if len(tmdbIDs) == 0 {
		return result, nil
	}

	// Build placeholder string
	placeholders := make([]string, len(tmdbIDs))
	args := make([]interface{}, len(tmdbIDs))
	for i, id := range tmdbIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	// Check library status
	rows, err := d.db.Query("SELECT id, tmdb_id FROM shows WHERE tmdb_id IN ("+placeholderStr+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, tmdbID int64
		if err := rows.Scan(&id, &tmdbID); err != nil {
			return nil, err
		}
		if status, ok := result[tmdbID]; ok {
			status.InLibrary = true
			libID := id
			status.LibraryID = &libID
		}
	}

	// Check request status
	rows2, err := d.db.Query("SELECT id, tmdb_id, status FROM requests WHERE type = 'tv' AND tmdb_id IN ("+placeholderStr+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, tmdbID int64
		var reqStatus string
		if err := rows2.Scan(&id, &tmdbID, &reqStatus); err != nil {
			return nil, err
		}
		if status, ok := result[tmdbID]; ok {
			status.Requested = true
			reqID := id
			status.RequestID = &reqID
			status.RequestStatus = &reqStatus
		}
	}

	return result, nil
}

// Watchlist methods

func (d *Database) AddToWatchlist(item *WatchlistItem) error {
	_, err := d.db.Exec(`
		INSERT INTO user_watchlist (user_id, tmdb_id, media_type, added_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, tmdb_id, media_type) DO NOTHING
	`, item.UserID, item.TmdbID, item.MediaType)
	return err
}

func (d *Database) RemoveFromWatchlist(userID, tmdbID int64, mediaType string) error {
	_, err := d.db.Exec(`
		DELETE FROM user_watchlist
		WHERE user_id = ? AND tmdb_id = ? AND media_type = ?
	`, userID, tmdbID, mediaType)
	return err
}

func (d *Database) GetWatchlist(userID int64) ([]WatchlistItem, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, tmdb_id, media_type, added_at
		FROM user_watchlist
		WHERE user_id = ?
		ORDER BY added_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var item WatchlistItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.TmdbID, &item.MediaType, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) IsInWatchlist(userID, tmdbID int64, mediaType string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM user_watchlist
		WHERE user_id = ? AND tmdb_id = ? AND media_type = ?
	`, userID, tmdbID, mediaType).Scan(&count)
	return count > 0, err
}

// Quality Preset operations

func (d *Database) GetQualityPresets() ([]QualityPreset, error) {
	rows, err := d.db.Query(`
		SELECT id, name, is_default, is_built_in, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade, created_at, updated_at
		FROM quality_presets
		ORDER BY is_default DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []QualityPreset
	for rows.Next() {
		var p QualityPreset
		var isDefault, isBuiltIn, preferSeasonPacks, autoUpgrade int
		var hdrFormatsJSON, audioFormatsJSON *string
		if err := rows.Scan(
			&p.ID, &p.Name, &isDefault, &isBuiltIn, &p.Resolution, &p.Source,
			&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
			&p.MinSeeders, &preferSeasonPacks, &autoUpgrade, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		p.IsBuiltIn = isBuiltIn == 1
		p.PreferSeasonPacks = preferSeasonPacks == 1
		p.AutoUpgrade = autoUpgrade == 1
		// Parse JSON arrays
		if hdrFormatsJSON != nil && *hdrFormatsJSON != "" {
			json.Unmarshal([]byte(*hdrFormatsJSON), &p.HDRFormats)
		}
		if audioFormatsJSON != nil && *audioFormatsJSON != "" {
			json.Unmarshal([]byte(*audioFormatsJSON), &p.AudioFormats)
		}
		if p.HDRFormats == nil {
			p.HDRFormats = []string{}
		}
		if p.AudioFormats == nil {
			p.AudioFormats = []string{}
		}
		presets = append(presets, p)
	}
	return presets, nil
}

func (d *Database) GetQualityPreset(id int64) (*QualityPreset, error) {
	var p QualityPreset
	var isDefault, isBuiltIn, preferSeasonPacks, autoUpgrade int
	var hdrFormatsJSON, audioFormatsJSON *string
	err := d.db.QueryRow(`
		SELECT id, name, is_default, is_built_in, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade, created_at, updated_at
		FROM quality_presets WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &isDefault, &isBuiltIn, &p.Resolution, &p.Source,
		&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
		&p.MinSeeders, &preferSeasonPacks, &autoUpgrade, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsBuiltIn = isBuiltIn == 1
	p.PreferSeasonPacks = preferSeasonPacks == 1
	p.AutoUpgrade = autoUpgrade == 1
	if hdrFormatsJSON != nil && *hdrFormatsJSON != "" {
		json.Unmarshal([]byte(*hdrFormatsJSON), &p.HDRFormats)
	}
	if audioFormatsJSON != nil && *audioFormatsJSON != "" {
		json.Unmarshal([]byte(*audioFormatsJSON), &p.AudioFormats)
	}
	if p.HDRFormats == nil {
		p.HDRFormats = []string{}
	}
	if p.AudioFormats == nil {
		p.AudioFormats = []string{}
	}
	return &p, nil
}

func (d *Database) GetDefaultQualityPreset() (*QualityPreset, error) {
	var p QualityPreset
	var isDefault, isBuiltIn, preferSeasonPacks, autoUpgrade int
	var hdrFormatsJSON, audioFormatsJSON *string
	err := d.db.QueryRow(`
		SELECT id, name, is_default, is_built_in, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade, created_at, updated_at
		FROM quality_presets WHERE is_default = 1 LIMIT 1
	`).Scan(
		&p.ID, &p.Name, &isDefault, &isBuiltIn, &p.Resolution, &p.Source,
		&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
		&p.MinSeeders, &preferSeasonPacks, &autoUpgrade, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsBuiltIn = isBuiltIn == 1
	p.PreferSeasonPacks = preferSeasonPacks == 1
	p.AutoUpgrade = autoUpgrade == 1
	if hdrFormatsJSON != nil && *hdrFormatsJSON != "" {
		json.Unmarshal([]byte(*hdrFormatsJSON), &p.HDRFormats)
	}
	if audioFormatsJSON != nil && *audioFormatsJSON != "" {
		json.Unmarshal([]byte(*audioFormatsJSON), &p.AudioFormats)
	}
	if p.HDRFormats == nil {
		p.HDRFormats = []string{}
	}
	if p.AudioFormats == nil {
		p.AudioFormats = []string{}
	}
	return &p, nil
}

func (d *Database) CreateQualityPreset(p *QualityPreset) error {
	hdrFormatsJSON, _ := json.Marshal(p.HDRFormats)
	audioFormatsJSON, _ := json.Marshal(p.AudioFormats)
	result, err := d.db.Exec(`
		INSERT INTO quality_presets (name, is_default, is_built_in, resolution, source,
		                            hdr_formats, codec, audio_formats, preferred_edition,
		                            min_seeders, prefer_season_packs, auto_upgrade)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.Name, p.IsDefault, p.IsBuiltIn, p.Resolution, p.Source,
		string(hdrFormatsJSON), p.Codec, string(audioFormatsJSON), p.PreferredEdition,
		p.MinSeeders, p.PreferSeasonPacks, p.AutoUpgrade)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateQualityPreset(p *QualityPreset) error {
	hdrFormatsJSON, _ := json.Marshal(p.HDRFormats)
	audioFormatsJSON, _ := json.Marshal(p.AudioFormats)
	_, err := d.db.Exec(`
		UPDATE quality_presets SET
			name = ?, resolution = ?, source = ?, hdr_formats = ?,
			codec = ?, audio_formats = ?, preferred_edition = ?,
			min_seeders = ?, prefer_season_packs = ?, auto_upgrade = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_built_in = 0
	`, p.Name, p.Resolution, p.Source, string(hdrFormatsJSON),
		p.Codec, string(audioFormatsJSON), p.PreferredEdition,
		p.MinSeeders, p.PreferSeasonPacks, p.AutoUpgrade, p.ID)
	return err
}

func (d *Database) DeleteQualityPreset(id int64) error {
	_, err := d.db.Exec("DELETE FROM quality_presets WHERE id = ? AND is_built_in = 0", id)
	return err
}

func (d *Database) SetDefaultQualityPreset(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing default
	if _, err := tx.Exec("UPDATE quality_presets SET is_default = 0"); err != nil {
		return err
	}
	// Set new default
	if _, err := tx.Exec("UPDATE quality_presets SET is_default = 1 WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

// Download operations

func (d *Database) GetDownloads() ([]Download, error) {
	rows, err := d.db.Query(`
		SELECT id, download_client_id, external_id, media_id, media_type, title,
		       size, status, progress, download_path, imported_path, error,
		       created_at, updated_at
		FROM downloads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []Download
	for rows.Next() {
		var dl Download
		if err := rows.Scan(
			&dl.ID, &dl.DownloadClientID, &dl.ExternalID, &dl.MediaID, &dl.MediaType,
			&dl.Title, &dl.Size, &dl.Status, &dl.Progress, &dl.DownloadPath,
			&dl.ImportedPath, &dl.Error, &dl.CreatedAt, &dl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		downloads = append(downloads, dl)
	}
	return downloads, nil
}

func (d *Database) GetDownloadByExternalID(clientID int64, externalID string) (*Download, error) {
	var dl Download
	err := d.db.QueryRow(`
		SELECT id, download_client_id, external_id, media_id, media_type, title,
		       size, status, progress, download_path, imported_path, error,
		       created_at, updated_at
		FROM downloads WHERE download_client_id = ? AND external_id = ?
	`, clientID, externalID).Scan(
		&dl.ID, &dl.DownloadClientID, &dl.ExternalID, &dl.MediaID, &dl.MediaType,
		&dl.Title, &dl.Size, &dl.Status, &dl.Progress, &dl.DownloadPath,
		&dl.ImportedPath, &dl.Error, &dl.CreatedAt, &dl.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dl, nil
}

func (d *Database) CreateDownload(dl *Download) error {
	result, err := d.db.Exec(`
		INSERT INTO downloads (download_client_id, external_id, media_id, media_type,
		                       title, size, status, progress, download_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, dl.DownloadClientID, dl.ExternalID, dl.MediaID, dl.MediaType,
		dl.Title, dl.Size, dl.Status, dl.Progress, dl.DownloadPath)
	if err != nil {
		return err
	}
	dl.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateDownload(dl *Download) error {
	_, err := d.db.Exec(`
		UPDATE downloads SET
			status = ?, progress = ?, download_path = ?, imported_path = ?,
			error = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, dl.Status, dl.Progress, dl.DownloadPath, dl.ImportedPath, dl.Error, dl.ID)
	return err
}

func (d *Database) DeleteDownload(id int64) error {
	_, err := d.db.Exec("DELETE FROM downloads WHERE id = ?", id)
	return err
}

// Naming Template operations

func (d *Database) GetNamingTemplates() ([]NamingTemplate, error) {
	rows, err := d.db.Query(`
		SELECT id, type, folder_template, file_template, is_default
		FROM naming_templates ORDER BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []NamingTemplate
	for rows.Next() {
		var t NamingTemplate
		var isDefault int
		if err := rows.Scan(&t.ID, &t.Type, &t.FolderTemplate, &t.FileTemplate, &isDefault); err != nil {
			return nil, err
		}
		t.IsDefault = isDefault == 1
		templates = append(templates, t)
	}
	return templates, nil
}

func (d *Database) GetNamingTemplate(templateType string) (*NamingTemplate, error) {
	var t NamingTemplate
	var isDefault int
	err := d.db.QueryRow(`
		SELECT id, type, folder_template, file_template, is_default
		FROM naming_templates WHERE type = ? AND is_default = 1
	`, templateType).Scan(&t.ID, &t.Type, &t.FolderTemplate, &t.FileTemplate, &isDefault)
	if err != nil {
		return nil, err
	}
	t.IsDefault = isDefault == 1
	return &t, nil
}

func (d *Database) UpdateNamingTemplate(t *NamingTemplate) error {
	_, err := d.db.Exec(`
		UPDATE naming_templates SET folder_template = ?, file_template = ?
		WHERE id = ?
	`, t.FolderTemplate, t.FileTemplate, t.ID)
	return err
}

// Import History operations

func (d *Database) CreateImportHistory(ih *ImportHistory) error {
	result, err := d.db.Exec(`
		INSERT INTO import_history (download_id, source_path, dest_path, media_id, media_type, success, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ih.DownloadID, ih.SourcePath, ih.DestPath, ih.MediaID, ih.MediaType, ih.Success, ih.Error)
	if err != nil {
		return err
	}
	ih.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetImportHistory(limit int) ([]ImportHistory, error) {
	rows, err := d.db.Query(`
		SELECT id, download_id, source_path, dest_path, media_id, media_type, success, error, created_at
		FROM import_history ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ImportHistory
	for rows.Next() {
		var ih ImportHistory
		var success int
		if err := rows.Scan(&ih.ID, &ih.DownloadID, &ih.SourcePath, &ih.DestPath,
			&ih.MediaID, &ih.MediaType, &success, &ih.Error, &ih.CreatedAt); err != nil {
			return nil, err
		}
		ih.Success = success == 1
		history = append(history, ih)
	}
	return history, nil
}

// Media Quality Status operations

func (d *Database) GetMediaQualityStatus(mediaID int64, mediaType string) (*MediaQualityStatus, error) {
	var s MediaQualityStatus
	var targetMet, upgradeAvailable int
	err := d.db.QueryRow(`
		SELECT id, media_id, media_type, current_resolution, current_source,
		       current_hdr, current_audio, current_edition, target_met,
		       upgrade_available, last_search, created_at, updated_at
		FROM media_quality_status WHERE media_id = ? AND media_type = ?
	`, mediaID, mediaType).Scan(
		&s.ID, &s.MediaID, &s.MediaType, &s.CurrentResolution, &s.CurrentSource,
		&s.CurrentHDR, &s.CurrentAudio, &s.CurrentEdition, &targetMet,
		&upgradeAvailable, &s.LastSearch, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.TargetMet = targetMet == 1
	s.UpgradeAvailable = upgradeAvailable == 1
	return &s, nil
}

func (d *Database) UpsertMediaQualityStatus(s *MediaQualityStatus) error {
	_, err := d.db.Exec(`
		INSERT INTO media_quality_status (media_id, media_type, current_resolution, current_source,
		                                  current_hdr, current_audio, current_edition, target_met,
		                                  upgrade_available, last_search)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(media_id, media_type) DO UPDATE SET
			current_resolution = excluded.current_resolution,
			current_source = excluded.current_source,
			current_hdr = excluded.current_hdr,
			current_audio = excluded.current_audio,
			current_edition = excluded.current_edition,
			target_met = excluded.target_met,
			upgrade_available = excluded.upgrade_available,
			last_search = excluded.last_search,
			updated_at = CURRENT_TIMESTAMP
	`, s.MediaID, s.MediaType, s.CurrentResolution, s.CurrentSource,
		s.CurrentHDR, s.CurrentAudio, s.CurrentEdition, s.TargetMet,
		s.UpgradeAvailable, s.LastSearch)
	return err
}

// Storage size operations

func (d *Database) GetTotalMoviesSize() (int64, error) {
	var total int64
	err := d.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM movies").Scan(&total)
	return total, err
}

func (d *Database) GetTotalTVSize() (int64, error) {
	var total int64
	err := d.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM episodes").Scan(&total)
	return total, err
}

func (d *Database) GetTotalMusicSize() (int64, error) {
	var total int64
	err := d.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM tracks").Scan(&total)
	return total, err
}

func (d *Database) GetTotalBooksSize() (int64, error) {
	var total int64
	err := d.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM books").Scan(&total)
	return total, err
}

// =====================
// Blocklist Operations
// =====================

func (d *Database) AddToBlocklist(entry *BlocklistEntry) error {
	result, err := d.db.Exec(`
		INSERT INTO blocklist (media_id, media_type, release_title, release_group, indexer_id, reason, error_message, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.MediaID, entry.MediaType, entry.ReleaseTitle, entry.ReleaseGroup,
		entry.IndexerID, entry.Reason, entry.ErrorMessage, entry.ExpiresAt)
	if err != nil {
		return err
	}
	entry.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetBlocklist() ([]BlocklistEntry, error) {
	rows, err := d.db.Query(`
		SELECT id, media_id, media_type, release_title, release_group, indexer_id, reason, error_message, expires_at, created_at
		FROM blocklist
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []BlocklistEntry
	for rows.Next() {
		var e BlocklistEntry
		if err := rows.Scan(&e.ID, &e.MediaID, &e.MediaType, &e.ReleaseTitle, &e.ReleaseGroup,
			&e.IndexerID, &e.Reason, &e.ErrorMessage, &e.ExpiresAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *Database) IsReleaseBlocklisted(releaseTitle string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM blocklist
		WHERE release_title = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`, releaseTitle).Scan(&count)
	return count > 0, err
}

func (d *Database) RemoveFromBlocklist(id int64) error {
	_, err := d.db.Exec("DELETE FROM blocklist WHERE id = ?", id)
	return err
}

func (d *Database) ClearExpiredBlocklist() error {
	_, err := d.db.Exec("DELETE FROM blocklist WHERE expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP")
	return err
}

// =====================
// Grab History Operations
// =====================

func (d *Database) AddGrabHistory(h *GrabHistory) error {
	result, err := d.db.Exec(`
		INSERT INTO grab_history (media_id, media_type, release_title, indexer_id, indexer_name,
			quality_resolution, quality_source, quality_codec, quality_audio, quality_hdr,
			release_group, size, download_client_id, download_id, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.MediaID, h.MediaType, h.ReleaseTitle, h.IndexerID, h.IndexerName,
		h.QualityResolution, h.QualitySource, h.QualityCodec, h.QualityAudio, h.QualityHDR,
		h.ReleaseGroup, h.Size, h.DownloadClientID, h.DownloadID, h.Status, h.ErrorMessage)
	if err != nil {
		return err
	}
	h.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetGrabHistory(limit int) ([]GrabHistory, error) {
	rows, err := d.db.Query(`
		SELECT id, media_id, media_type, release_title, indexer_id, indexer_name,
			quality_resolution, quality_source, quality_codec, quality_audio, quality_hdr,
			release_group, size, download_client_id, download_id, status, error_message, grabbed_at, imported_at
		FROM grab_history
		ORDER BY grabbed_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []GrabHistory
	for rows.Next() {
		var h GrabHistory
		if err := rows.Scan(&h.ID, &h.MediaID, &h.MediaType, &h.ReleaseTitle, &h.IndexerID, &h.IndexerName,
			&h.QualityResolution, &h.QualitySource, &h.QualityCodec, &h.QualityAudio, &h.QualityHDR,
			&h.ReleaseGroup, &h.Size, &h.DownloadClientID, &h.DownloadID, &h.Status, &h.ErrorMessage,
			&h.GrabbedAt, &h.ImportedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

func (d *Database) GetGrabHistoryForMedia(mediaID int64, mediaType string) ([]GrabHistory, error) {
	rows, err := d.db.Query(`
		SELECT id, media_id, media_type, release_title, indexer_id, indexer_name,
			quality_resolution, quality_source, quality_codec, quality_audio, quality_hdr,
			release_group, size, download_client_id, download_id, status, error_message, grabbed_at, imported_at
		FROM grab_history
		WHERE media_id = ? AND media_type = ?
		ORDER BY grabbed_at DESC
	`, mediaID, mediaType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []GrabHistory
	for rows.Next() {
		var h GrabHistory
		if err := rows.Scan(&h.ID, &h.MediaID, &h.MediaType, &h.ReleaseTitle, &h.IndexerID, &h.IndexerName,
			&h.QualityResolution, &h.QualitySource, &h.QualityCodec, &h.QualityAudio, &h.QualityHDR,
			&h.ReleaseGroup, &h.Size, &h.DownloadClientID, &h.DownloadID, &h.Status, &h.ErrorMessage,
			&h.GrabbedAt, &h.ImportedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

func (d *Database) UpdateGrabHistoryStatus(id int64, status string, errorMsg *string) error {
	if status == "imported" {
		_, err := d.db.Exec(`
			UPDATE grab_history SET status = ?, error_message = ?, imported_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, errorMsg, id)
		return err
	}
	_, err := d.db.Exec("UPDATE grab_history SET status = ?, error_message = ? WHERE id = ?", status, errorMsg, id)
	return err
}

// =====================
// Blocked Groups Operations
// =====================

func (d *Database) GetBlockedGroups() ([]BlockedGroup, error) {
	rows, err := d.db.Query(`
		SELECT id, name, reason, auto_blocked, failure_count, created_at
		FROM blocked_groups
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []BlockedGroup
	for rows.Next() {
		var g BlockedGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Reason, &g.AutoBlocked, &g.FailureCount, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *Database) AddBlockedGroup(name, reason string, autoBlocked bool) error {
	_, err := d.db.Exec(`
		INSERT INTO blocked_groups (name, reason, auto_blocked, failure_count)
		VALUES (?, ?, ?, 0)
		ON CONFLICT(name) DO UPDATE SET reason = excluded.reason, auto_blocked = excluded.auto_blocked
	`, name, reason, autoBlocked)
	return err
}

func (d *Database) IncrementGroupFailures(name string) error {
	_, err := d.db.Exec(`
		INSERT INTO blocked_groups (name, auto_blocked, failure_count)
		VALUES (?, 1, 1)
		ON CONFLICT(name) DO UPDATE SET failure_count = failure_count + 1
	`, name)
	return err
}

func (d *Database) RemoveBlockedGroup(id int64) error {
	_, err := d.db.Exec("DELETE FROM blocked_groups WHERE id = ?", id)
	return err
}

func (d *Database) IsGroupBlocked(name string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM blocked_groups WHERE name = ?", name).Scan(&count)
	return count > 0, err
}

// =====================
// Trusted Groups Operations
// =====================

func (d *Database) GetTrustedGroups() ([]TrustedGroup, error) {
	rows, err := d.db.Query(`
		SELECT id, name, category, created_at
		FROM trusted_groups
		ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []TrustedGroup
	for rows.Next() {
		var g TrustedGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Category, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *Database) AddTrustedGroup(name, category string) error {
	_, err := d.db.Exec(`
		INSERT INTO trusted_groups (name, category) VALUES (?, ?)
		ON CONFLICT(name) DO UPDATE SET category = excluded.category
	`, name, category)
	return err
}

func (d *Database) RemoveTrustedGroup(id int64) error {
	_, err := d.db.Exec("DELETE FROM trusted_groups WHERE id = ?", id)
	return err
}

// =====================
// Release Filters Operations
// =====================

func (d *Database) GetReleaseFilters(presetID int64) ([]ReleaseFilter, error) {
	rows, err := d.db.Query(`
		SELECT id, preset_id, filter_type, value, is_regex, created_at
		FROM release_filters
		WHERE preset_id = ?
		ORDER BY filter_type, value
	`, presetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []ReleaseFilter
	for rows.Next() {
		var f ReleaseFilter
		if err := rows.Scan(&f.ID, &f.PresetID, &f.FilterType, &f.Value, &f.IsRegex, &f.CreatedAt); err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (d *Database) AddReleaseFilter(f *ReleaseFilter) error {
	result, err := d.db.Exec(`
		INSERT INTO release_filters (preset_id, filter_type, value, is_regex)
		VALUES (?, ?, ?, ?)
	`, f.PresetID, f.FilterType, f.Value, f.IsRegex)
	if err != nil {
		return err
	}
	f.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) RemoveReleaseFilter(id int64) error {
	_, err := d.db.Exec("DELETE FROM release_filters WHERE id = ?", id)
	return err
}

// =====================
// Delay Profiles Operations
// =====================

func (d *Database) GetDelayProfiles() ([]DelayProfile, error) {
	rows, err := d.db.Query(`
		SELECT id, name, enabled, delay_minutes, bypass_if_resolution, bypass_if_source, bypass_if_score_above, library_id, created_at
		FROM delay_profiles
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []DelayProfile
	for rows.Next() {
		var p DelayProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, &p.DelayMinutes, &p.BypassIfResolution,
			&p.BypassIfSource, &p.BypassIfScoreAbove, &p.LibraryID, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) CreateDelayProfile(p *DelayProfile) error {
	result, err := d.db.Exec(`
		INSERT INTO delay_profiles (name, enabled, delay_minutes, bypass_if_resolution, bypass_if_source, bypass_if_score_above, library_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.Name, p.Enabled, p.DelayMinutes, p.BypassIfResolution, p.BypassIfSource, p.BypassIfScoreAbove, p.LibraryID)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateDelayProfile(p *DelayProfile) error {
	_, err := d.db.Exec(`
		UPDATE delay_profiles SET name = ?, enabled = ?, delay_minutes = ?,
		bypass_if_resolution = ?, bypass_if_source = ?, bypass_if_score_above = ?, library_id = ?
		WHERE id = ?
	`, p.Name, p.Enabled, p.DelayMinutes, p.BypassIfResolution, p.BypassIfSource, p.BypassIfScoreAbove, p.LibraryID, p.ID)
	return err
}

func (d *Database) DeleteDelayProfile(id int64) error {
	_, err := d.db.Exec("DELETE FROM delay_profiles WHERE id = ?", id)
	return err
}

// =====================
// Pending Grabs Operations
// =====================

func (d *Database) AddPendingGrab(pg *PendingGrab) error {
	result, err := d.db.Exec(`
		INSERT INTO pending_grabs (media_id, media_type, release_title, release_data, score, indexer_id, available_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, pg.MediaID, pg.MediaType, pg.ReleaseTitle, pg.ReleaseData, pg.Score, pg.IndexerID, pg.AvailableAt)
	if err != nil {
		return err
	}
	pg.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetPendingGrabs() ([]PendingGrab, error) {
	rows, err := d.db.Query(`
		SELECT id, media_id, media_type, release_title, release_data, score, indexer_id, available_at, created_at
		FROM pending_grabs
		ORDER BY available_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pending []PendingGrab
	for rows.Next() {
		var pg PendingGrab
		if err := rows.Scan(&pg.ID, &pg.MediaID, &pg.MediaType, &pg.ReleaseTitle, &pg.ReleaseData,
			&pg.Score, &pg.IndexerID, &pg.AvailableAt, &pg.CreatedAt); err != nil {
			return nil, err
		}
		pending = append(pending, pg)
	}
	return pending, nil
}

func (d *Database) GetReadyPendingGrabs() ([]PendingGrab, error) {
	rows, err := d.db.Query(`
		SELECT id, media_id, media_type, release_title, release_data, score, indexer_id, available_at, created_at
		FROM pending_grabs
		WHERE available_at <= CURRENT_TIMESTAMP
		ORDER BY available_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pending []PendingGrab
	for rows.Next() {
		var pg PendingGrab
		if err := rows.Scan(&pg.ID, &pg.MediaID, &pg.MediaType, &pg.ReleaseTitle, &pg.ReleaseData,
			&pg.Score, &pg.IndexerID, &pg.AvailableAt, &pg.CreatedAt); err != nil {
			return nil, err
		}
		pending = append(pending, pg)
	}
	return pending, nil
}

func (d *Database) RemovePendingGrab(id int64) error {
	_, err := d.db.Exec("DELETE FROM pending_grabs WHERE id = ?", id)
	return err
}

func (d *Database) RemovePendingGrabsForMedia(mediaID int64, mediaType string) error {
	_, err := d.db.Exec("DELETE FROM pending_grabs WHERE media_id = ? AND media_type = ?", mediaID, mediaType)
	return err
}

// Exclusion CRUD methods

func (d *Database) GetExclusions() ([]Exclusion, error) {
	rows, err := d.db.Query(`
		SELECT id, exclusion_type, media_id, media_type, indexer_id, library_id, reason, created_at
		FROM exclusions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exclusions []Exclusion
	for rows.Next() {
		var e Exclusion
		if err := rows.Scan(&e.ID, &e.ExclusionType, &e.MediaID, &e.MediaType,
			&e.IndexerID, &e.LibraryID, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
		}
		exclusions = append(exclusions, e)
	}
	return exclusions, nil
}

func (d *Database) GetExclusionsByType(exclusionType string) ([]Exclusion, error) {
	rows, err := d.db.Query(`
		SELECT id, exclusion_type, media_id, media_type, indexer_id, library_id, reason, created_at
		FROM exclusions WHERE exclusion_type = ? ORDER BY created_at DESC`, exclusionType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exclusions []Exclusion
	for rows.Next() {
		var e Exclusion
		if err := rows.Scan(&e.ID, &e.ExclusionType, &e.MediaID, &e.MediaType,
			&e.IndexerID, &e.LibraryID, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
		}
		exclusions = append(exclusions, e)
	}
	return exclusions, nil
}

func (d *Database) AddExclusion(e *Exclusion) error {
	result, err := d.db.Exec(`
		INSERT INTO exclusions (exclusion_type, media_id, media_type, indexer_id, library_id, reason)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.ExclusionType, e.MediaID, e.MediaType, e.IndexerID, e.LibraryID, e.Reason)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

func (d *Database) RemoveExclusion(id int64) error {
	_, err := d.db.Exec("DELETE FROM exclusions WHERE id = ?", id)
	return err
}

func (d *Database) IsMediaExcluded(mediaID int64, mediaType string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM exclusions
		WHERE media_id = ? AND media_type = ?`, mediaID, mediaType).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *Database) IsIndexerExcludedForLibrary(indexerID, libraryID int64) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM exclusions
		WHERE exclusion_type = 'indexer' AND indexer_id = ? AND library_id = ?`,
		indexerID, libraryID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Media quality override methods

func (d *Database) GetMediaQualityOverride(mediaID int64, mediaType string) (*MediaQualityOverride, error) {
	var override MediaQualityOverride
	err := d.db.QueryRow(`
		SELECT id, media_id, media_type, preset_id, monitored, created_at
		FROM media_quality_override
		WHERE media_id = ? AND media_type = ?`, mediaID, mediaType).Scan(
		&override.ID, &override.MediaID, &override.MediaType,
		&override.PresetID, &override.Monitored, &override.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

func (d *Database) SetMediaQualityOverride(override *MediaQualityOverride) error {
	result, err := d.db.Exec(`
		INSERT OR REPLACE INTO media_quality_override (media_id, media_type, preset_id, monitored)
		VALUES (?, ?, ?, ?)`,
		override.MediaID, override.MediaType, override.PresetID, override.Monitored)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	override.ID = id
	return nil
}

func (d *Database) DeleteMediaQualityOverride(mediaID int64, mediaType string) error {
	_, err := d.db.Exec("DELETE FROM media_quality_override WHERE media_id = ? AND media_type = ?",
		mediaID, mediaType)
	return err
}
