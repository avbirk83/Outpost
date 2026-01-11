package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

// DB returns the underlying sql.DB connection
func (d *Database) DB() *sql.DB {
	return d.db
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
	ProfileID int64     `json:"profileId"`
	MediaType string    `json:"mediaType"` // movie, episode
	MediaID   int64     `json:"mediaId"`
	Position  float64   `json:"position"`  // seconds
	Duration  float64   `json:"duration"`  // seconds
	UpdatedAt time.Time `json:"updatedAt"`
}

type Chapter struct {
	ID           int64   `json:"id"`
	MediaType    string  `json:"mediaType"` // movie, episode
	MediaID      int64   `json:"mediaId"`
	ChapterIndex int     `json:"index"`
	Title        string  `json:"title"`
	StartTime    float64 `json:"startTime"` // seconds
	EndTime      float64 `json:"endTime"`   // seconds
}

type SkipSegment struct {
	StartTime float64 `json:"startTime"` // seconds
	EndTime   float64 `json:"endTime"`   // seconds
}

type SkipSegments struct {
	Intro   *SkipSegment `json:"intro,omitempty"`
	Credits *SkipSegment `json:"credits,omitempty"`
}

type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"` // Never expose in JSON
	Role               string    `json:"role"` // admin, user, kid
	ContentRatingLimit *string   `json:"contentRatingLimit,omitempty"` // G, PG, PG-13, R, NC-17, or nil (no limit)
	PinHash            *string   `json:"-"`                            // PIN hash, never expose
	RequirePin         bool      `json:"requirePin"`                   // Require PIN for elevated content
	CreatedAt          time.Time `json:"createdAt"`
}

// Profile represents a viewing profile within a user account (Netflix-style)
type Profile struct {
	ID                 int64     `json:"id"`
	UserID             int64     `json:"userId"`
	Name               string    `json:"name"`
	AvatarURL          *string   `json:"avatarUrl,omitempty"`
	IsDefault          bool      `json:"isDefault"`
	IsKid              bool      `json:"isKid"`
	ContentRatingLimit *string   `json:"contentRatingLimit,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ContentRatingLevel returns the numeric level for a content rating (for comparison)
func ContentRatingLevel(rating string) int {
	switch rating {
	case "G":
		return 1
	case "PG":
		return 2
	case "PG-13":
		return 3
	case "R":
		return 4
	case "NC-17":
		return 5
	default:
		return 0 // Unknown or unrated
	}
}

// NormalizeContentRating converts various content rating formats to US MPAA ratings
func NormalizeContentRating(rating string, country string) string {
	if rating == "" {
		return ""
	}

	// Already normalized US ratings
	switch rating {
	case "G", "PG", "PG-13", "R", "NC-17":
		return rating
	}

	// US TV ratings
	switch rating {
	case "TV-Y", "TV-Y7", "TV-G":
		return "G"
	case "TV-PG":
		return "PG"
	case "TV-14":
		return "PG-13"
	case "TV-MA":
		return "R"
	}

	// UK ratings (BBFC)
	switch rating {
	case "U", "Uc":
		return "G"
	case "PG": // UK PG
		return "PG"
	case "12", "12A":
		return "PG-13"
	case "15":
		return "R"
	case "18", "R18":
		return "NC-17"
	}

	// Australia ratings
	switch rating {
	case "G", "E":
		return "G"
	case "M", "PG": // Australia
		return "PG"
	case "MA", "MA15+", "M15+":
		return "PG-13"
	case "R", "R18+":
		return "R"
	case "X", "X18+":
		return "NC-17"
	}

	// Germany ratings (FSK)
	switch rating {
	case "FSK 0":
		return "G"
	case "FSK 6":
		return "PG"
	case "FSK 12":
		return "PG-13"
	case "FSK 16":
		return "R"
	case "FSK 18":
		return "NC-17"
	}

	// Canada ratings
	switch rating {
	case "G", "E":
		return "G"
	case "PG":
		return "PG"
	case "14A", "14+":
		return "PG-13"
	case "18A", "18+", "R":
		return "R"
	case "A":
		return "NC-17"
	}

	// Default: try to match common patterns
	ratingUpper := rating
	if len(rating) > 0 {
		// Handle numeric ratings
		switch rating {
		case "0", "6":
			return "G"
		case "7", "10":
			return "PG"
		case "12", "13":
			return "PG-13"
		case "16", "17":
			return "R"
		case "18", "21":
			return "NC-17"
		}
	}

	// If we can't determine, return as-is (will show as "Unrated" in UI)
	return ratingUpper
}

type Session struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"userId"`
	Token           string    `json:"token"`
	ExpiresAt       time.Time `json:"expiresAt"`
	ActiveProfileID *int64    `json:"activeProfileId,omitempty"`
}

// PinElevation represents a temporary elevated access session after PIN verification
type PinElevation struct {
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
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"` // torznab, newznab, prowlarr
	URL                string `json:"url"`
	APIKey             string `json:"apiKey,omitempty"`
	Categories         string `json:"categories,omitempty"` // Comma-separated category IDs
	Priority           int    `json:"priority"`
	Enabled            bool   `json:"enabled"`
	ProwlarrID         *int64 `json:"prowlarrId,omitempty"`
	SyncedFromProwlarr bool   `json:"syncedFromProwlarr"`
	Protocol           string `json:"protocol,omitempty"` // torrent, usenet
	SupportsMovies     bool   `json:"supportsMovies"`
	SupportsTV         bool   `json:"supportsTV"`
	SupportsMusic      bool   `json:"supportsMusic"`
	SupportsBooks      bool   `json:"supportsBooks"`
	SupportsAnime      bool   `json:"supportsAnime"`
	SupportsIMDB       bool   `json:"supportsImdb"`
	SupportsTMDB       bool   `json:"supportsTmdb"`
	SupportsTVDB       bool   `json:"supportsTvdb"`
}

type ProwlarrConfig struct {
	ID                int64      `json:"id"`
	URL               string     `json:"url"`
	APIKey            string     `json:"apiKey,omitempty"`
	AutoSync          bool       `json:"autoSync"`
	SyncIntervalHours int        `json:"syncIntervalHours"`
	LastSync          *time.Time `json:"lastSync,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
}

type IndexerTag struct {
	ID           int64  `json:"id"`
	ProwlarrID   int    `json:"prowlarrId"`
	Name         string `json:"name"`
	IndexerCount int    `json:"indexerCount,omitempty"`
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
	ID               int64      `json:"id"`
	Type             string     `json:"type"`             // movie, show
	TmdbID           int64      `json:"tmdbId"`
	ImdbID           *string    `json:"imdbId,omitempty"` // IMDB ID for more accurate searches
	Title            string     `json:"title"`
	Year             int        `json:"year,omitempty"`
	PosterPath       *string    `json:"posterPath,omitempty"`
	QualityProfileID int64      `json:"qualityProfileId"`  // Deprecated, kept for compatibility
	QualityPresetID  *int64     `json:"qualityPresetId,omitempty"` // New: which preset to use for filtering
	Monitored        bool       `json:"monitored"`
	Seasons          string     `json:"seasons,omitempty"`       // JSON array of season numbers, empty = all
	SearchNow        bool       `json:"searchNow,omitempty"`     // For triggering immediate search
	LastSearched     *time.Time `json:"lastSearched,omitempty"`
	AddedAt          time.Time  `json:"addedAt"`
}

type Request struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"userId"`
	Username         string    `json:"username,omitempty"` // Populated from join
	Type             string    `json:"type"`               // movie, show
	TmdbID           int64     `json:"tmdbId"`
	Title            string    `json:"title"`
	Year             int       `json:"year,omitempty"`
	Overview         *string   `json:"overview,omitempty"`
	PosterPath       *string   `json:"posterPath,omitempty"`
	BackdropPath     *string   `json:"backdropPath,omitempty"`
	QualityProfileID *int64    `json:"qualityProfileId,omitempty"` // Deprecated, use QualityPresetID
	QualityPresetID  *int64    `json:"qualityPresetId,omitempty"`
	Status           string    `json:"status"` // requested, approved, denied, available
	StatusReason     *string   `json:"statusReason,omitempty"`
	RequestedAt      time.Time `json:"requestedAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
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
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	MediaType         string    `json:"mediaType"`         // "movie", "tv", "anime"
	IsDefault         bool      `json:"isDefault"`
	IsBuiltIn         bool      `json:"isBuiltIn"`
	Enabled           bool      `json:"enabled"`           // Whether this preset is shown in request modal
	Priority          int       `json:"priority"`          // Order for fallback (lower = higher priority)
	Resolution        string    `json:"resolution"`        // "4k", "1080p", "720p", "480p", "sd", "any"
	Source            string    `json:"source"`            // "remux", "bluray", "web", "hdtv", "dvd", "any"
	HDRFormats        []string  `json:"hdrFormats"`        // Array of HDR formats
	Codec             string    `json:"codec"`             // "any", "hevc", "av1", "x264"
	AudioFormats      []string  `json:"audioFormats"`      // Array of audio formats
	PreferredEdition  string    `json:"preferredEdition"`  // "any", "theatrical", "directors", etc
	MinSeeders        int       `json:"minSeeders"`
	PreferSeasonPacks bool      `json:"preferSeasonPacks"`
	AutoUpgrade       bool      `json:"autoUpgrade"`
	// Anime-specific preferences
	PreferDualAudio   bool   `json:"preferDualAudio"`
	PreferDubbed      bool   `json:"preferDubbed"`
	PreferredLanguage string `json:"preferredLanguage"` // "english", "japanese", "any"
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type MediaQualityOverride struct {
	ID                    int64     `json:"id"`
	MediaID               int64     `json:"mediaId"`
	MediaType             string    `json:"mediaType"`
	PresetID              *int64    `json:"presetId"`
	Monitored             bool      `json:"monitored"`
	MonitoredSeasons      string    `json:"monitoredSeasons,omitempty"`      // JSON array of season numbers, empty = all
	PreferredAudioLang    string    `json:"preferredAudioLang,omitempty"`    // ISO 639-1 language code (e.g., "en", "ja")
	PreferredSubtitleLang string    `json:"preferredSubtitleLang,omitempty"` // ISO 639-1 language code or "off"
	CreatedAt             time.Time `json:"createdAt"`
}

type MediaQualityStatus struct {
	ID                 int64      `json:"id"`
	MediaID            int64      `json:"mediaId"`
	MediaType          string     `json:"mediaType"`
	CurrentResolution  *string    `json:"currentResolution"`
	CurrentSource      *string    `json:"currentSource"`
	CurrentHDR         *string    `json:"currentHdr"`
	CurrentAudio       *string    `json:"currentAudio"`
	CurrentEdition     *string    `json:"currentEdition"`
	TargetMet          bool       `json:"targetMet"`
	UpgradeAvailable   bool       `json:"upgradeAvailable"`
	LastSearch         *time.Time `json:"lastSearch"`
	UpgradeSearchedAt  *time.Time `json:"upgradeSearchedAt"`
	CurrentScore       int        `json:"currentScore"`
	CutoffScore        int        `json:"cutoffScore"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// UpgradeableItem represents a media item that can be upgraded
type UpgradeableItem struct {
	ID             int64   `json:"id"`
	Type           string  `json:"type"` // movie or episode
	Title          string  `json:"title"`
	Year           int     `json:"year,omitempty"`
	SeasonNumber   int     `json:"seasonNumber,omitempty"`
	EpisodeNumber  int     `json:"episodeNumber,omitempty"`
	ShowTitle      string  `json:"showTitle,omitempty"`
	CurrentQuality string  `json:"currentQuality"`
	CurrentScore   int     `json:"currentScore"`
	CutoffQuality  string  `json:"cutoffQuality"`
	CutoffScore    int     `json:"cutoffScore"`
	PosterPath     *string `json:"posterPath,omitempty"`
	Size           int64   `json:"size"`
	LastSearched   *string `json:"lastSearched,omitempty"`
}

// UpgradesSummary contains the list of upgradeable items
type UpgradesSummary struct {
	Movies     []UpgradeableItem `json:"movies"`
	Episodes   []UpgradeableItem `json:"episodes"`
	TotalCount int               `json:"totalCount"`
	TotalSize  int64             `json:"totalSize"`
}

// Download tracking types

type Download struct {
	ID               int64      `json:"id"`
	DownloadClientID *int64     `json:"downloadClientId"`
	ExternalID       string     `json:"externalId"`
	MediaID          *int64     `json:"mediaId"`
	MediaType        *string    `json:"mediaType"`
	Title            string     `json:"title"`
	TmdbID           int64      `json:"tmdbId,omitempty"`
	PosterPath       string     `json:"posterPath,omitempty"`
	Year             int        `json:"year,omitempty"`
	Size             int64      `json:"size"`
	State            string     `json:"state"` // downloading, completed, importing, imported, failed, unmatched
	Progress         float64    `json:"progress"`
	DownloadPath     *string    `json:"downloadPath"`
	ImportedPath     *string    `json:"importedPath"`
	Error            *string    `json:"error"`
	LastError        *string    `json:"lastError"`
	FailedAt         *time.Time `json:"failedAt"`
	RetryCount       *int       `json:"retryCount"`
	StalledNotified  bool       `json:"stalledNotified"`
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

// ScheduledTask represents a background task
type ScheduledTask struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	TaskType        string     `json:"taskType"`
	Enabled         bool       `json:"enabled"`
	IntervalMinutes int        `json:"intervalMinutes"`
	LastRun         *time.Time `json:"lastRun"`
	NextRun         *time.Time `json:"nextRun"`
	LastDurationMs  *int64     `json:"lastDurationMs"`
	LastStatus      string     `json:"lastStatus"`
	LastError       *string    `json:"lastError"`
	RunCount        int        `json:"runCount"`
	FailCount       int        `json:"failCount"`
	IsRunning       bool       `json:"isRunning"` // Computed at runtime
}

// TaskHistory represents a task execution record
type TaskHistory struct {
	ID             int64      `json:"id"`
	TaskID         int64      `json:"taskId"`
	TaskName       string     `json:"taskName,omitempty"` // Populated from join
	StartedAt      time.Time  `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
	DurationMs     *int64     `json:"durationMs"`
	Status         string     `json:"status"`
	ItemsProcessed int        `json:"itemsProcessed"`
	ItemsFound     int        `json:"itemsFound"`
	Error          *string    `json:"error"`
	Details        *string    `json:"details"`
}

// Notification represents an in-app notification
type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Type      string    `json:"type"`    // new_content, request_approved, request_denied, download_complete, download_failed
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ImageURL  *string   `json:"imageUrl,omitempty"`
	Link      *string   `json:"link,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

// Collection represents a collection of movies/shows (franchise, custom list)
type Collection struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	TmdbCollectionID *int64    `json:"tmdbCollectionId,omitempty"`
	PosterPath       *string   `json:"posterPath,omitempty"`
	BackdropPath     *string   `json:"backdropPath,omitempty"`
	IsAuto           bool      `json:"isAuto"`
	SortOrder        string    `json:"sortOrder"` // release, added, title, custom
	ItemCount        int       `json:"itemCount,omitempty"`
	OwnedCount       int       `json:"ownedCount,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// CollectionItem represents a movie or show in a collection
type CollectionItem struct {
	ID           int64     `json:"id"`
	CollectionID int64     `json:"collectionId"`
	MediaType    string    `json:"mediaType"` // movie, show
	MediaID      *int64    `json:"mediaId,omitempty"`
	TmdbID       int64     `json:"tmdbId"`
	Title        string    `json:"title"`
	Year         int       `json:"year,omitempty"`
	PosterPath   *string   `json:"posterPath,omitempty"`
	SortOrder    int       `json:"sortOrder"`
	InLibrary    bool      `json:"inLibrary"`
	AddedAt      time.Time `json:"addedAt"`
}

// SmartPlaylist represents a dynamic collection based on rules
type SmartPlaylist struct {
	ID            int64      `json:"id"`
	UserID        *int64     `json:"userId,omitempty"` // nil = system/global
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	Rules         string     `json:"rules"` // JSON rules
	SortBy        string     `json:"sortBy"`
	SortOrder     string     `json:"sortOrder"`
	LimitCount    *int       `json:"limitCount,omitempty"`
	MediaType     string     `json:"mediaType"` // movie, show, both
	AutoRefresh   bool       `json:"autoRefresh"`
	IsSystem      bool       `json:"isSystem"`
	ItemCount     int        `json:"itemCount,omitempty"`
	LastRefreshed *time.Time `json:"lastRefreshed,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// PlaylistRules defines the structure for smart playlist rules
type PlaylistRules struct {
	Match      string              `json:"match"` // all, any
	Conditions []PlaylistCondition `json:"conditions"`
}

// PlaylistCondition represents a single rule condition
type PlaylistCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// SmartPlaylistItem represents a media item in a smart playlist result
type SmartPlaylistItem struct {
	ID         int64   `json:"id"`
	MediaType  string  `json:"mediaType"`
	Title      string  `json:"title"`
	Year       int     `json:"year,omitempty"`
	PosterPath *string `json:"posterPath,omitempty"`
	Rating     float64 `json:"rating,omitempty"`
	Runtime    int     `json:"runtime,omitempty"`
	AddedAt    string  `json:"addedAt,omitempty"`
}

// TraktConfig represents a user's Trakt.tv configuration
type TraktConfig struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"userId"`
	AccessToken   string     `json:"-"`
	RefreshToken  string     `json:"-"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Username      *string    `json:"username,omitempty"`
	SyncEnabled   bool       `json:"syncEnabled"`
	SyncWatched   bool       `json:"syncWatched"`
	SyncRatings   bool       `json:"syncRatings"`
	SyncWatchlist bool       `json:"syncWatchlist"`
	LastSyncedAt  *time.Time `json:"lastSyncedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// WatchHistoryItem represents an item in watch history
type WatchHistoryItem struct {
	ID            int64     `json:"id"`
	ProfileID     int64     `json:"profileId"`
	MediaType     string    `json:"mediaType"`
	MediaID       int64     `json:"mediaId"`
	TmdbID        *int64    `json:"tmdbId,omitempty"`
	WatchedAt     time.Time `json:"watchedAt"`
	SyncedToTrakt bool      `json:"syncedToTrakt"`
	TraktID       *int64    `json:"traktId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TraktSyncQueueItem represents a queued Trakt sync action
type TraktSyncQueueItem struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"userId"`
	Action      string     `json:"action"` // watched, rating, watchlist
	MediaType   string     `json:"mediaType"`
	TmdbID      int64      `json:"tmdbId"`
	Data        *string    `json:"data,omitempty"` // JSON data (e.g., rating value)
	Status      string     `json:"status"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
}

func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	// Set busy timeout to 5 seconds
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
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

	-- PIN elevation sessions for parental controls
	CREATE TABLE IF NOT EXISTS pin_elevations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_pin_elevations_token ON pin_elevations(token);
	CREATE INDEX IF NOT EXISTS idx_pin_elevations_user ON pin_elevations(user_id);

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

	-- Prowlarr connection config
	CREATE TABLE IF NOT EXISTS prowlarr_config (
		id INTEGER PRIMARY KEY,
		url TEXT NOT NULL,
		api_key TEXT NOT NULL,
		auto_sync INTEGER DEFAULT 1,
		sync_interval_hours INTEGER DEFAULT 24,
		last_sync DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Tags synced from Prowlarr
	CREATE TABLE IF NOT EXISTS indexer_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		prowlarr_id INTEGER NOT NULL UNIQUE,
		name TEXT NOT NULL
	);

	-- Indexer <-> Tag mapping
	CREATE TABLE IF NOT EXISTS indexer_tag_map (
		indexer_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (indexer_id, tag_id),
		FOREIGN KEY (indexer_id) REFERENCES indexers(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES indexer_tags(id) ON DELETE CASCADE
	);

	-- Library <-> Tag assignment
	CREATE TABLE IF NOT EXISTS library_indexer_tags (
		library_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (library_id, tag_id),
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES indexer_tags(id) ON DELETE CASCADE
	);

	-- Indexer category IDs (from Prowlarr)
	CREATE TABLE IF NOT EXISTS indexer_categories (
		indexer_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (indexer_id, category_id),
		FOREIGN KEY (indexer_id) REFERENCES indexers(id) ON DELETE CASCADE
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
		imdb_id TEXT,
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
		backdrop_path TEXT,
		quality_profile_id INTEGER,
		status TEXT NOT NULL DEFAULT 'requested',
		status_reason TEXT,
		requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (quality_profile_id) REFERENCES quality_profiles(id),
		UNIQUE(user_id, type, tmdb_id)
	);

	-- Link requests to downloads
	CREATE TABLE IF NOT EXISTS request_downloads (
		request_id INTEGER NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
		download_id INTEGER NOT NULL REFERENCES tracked_downloads(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (request_id, download_id)
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
		monitored_seasons TEXT DEFAULT '',
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

	-- Tracked downloads (state machine with full lifecycle tracking)
	CREATE TABLE IF NOT EXISTS tracked_downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_client_id INTEGER NOT NULL REFERENCES download_clients(id),
		external_id TEXT NOT NULL,
		request_id INTEGER REFERENCES requests(id),
		media_id INTEGER,
		media_type TEXT,

		state TEXT NOT NULL DEFAULT 'queued',
		previous_state TEXT,
		state_changed_at DATETIME,

		title TEXT NOT NULL,
		parsed_info TEXT,

		size INTEGER DEFAULT 0,
		downloaded INTEGER DEFAULT 0,
		progress REAL DEFAULT 0,
		speed INTEGER DEFAULT 0,
		eta INTEGER DEFAULT 0,
		seeders INTEGER DEFAULT 0,

		download_path TEXT,
		import_path TEXT,

		quality TEXT,
		custom_format_score INTEGER DEFAULT 0,

		grabbed_at DATETIME,
		completed_at DATETIME,
		imported_at DATETIME,

		warnings TEXT,
		errors TEXT,
		import_block_reason TEXT,

		ratio REAL DEFAULT 0,
		seeding_time INTEGER DEFAULT 0,
		can_remove INTEGER DEFAULT 0,

		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

		UNIQUE(download_client_id, external_id)
	);

	-- Download events (state change history)
	CREATE TABLE IF NOT EXISTS download_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id INTEGER NOT NULL REFERENCES tracked_downloads(id) ON DELETE CASCADE,
		from_state TEXT,
		to_state TEXT NOT NULL,
		reason TEXT,
		details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

	-- Scheduled tasks definition
	CREATE TABLE IF NOT EXISTS scheduled_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		task_type TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		interval_minutes INTEGER NOT NULL,
		last_run DATETIME,
		next_run DATETIME,
		last_duration_ms INTEGER,
		last_status TEXT,
		last_error TEXT,
		run_count INTEGER DEFAULT 0,
		fail_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Task execution history
	CREATE TABLE IF NOT EXISTS task_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		started_at DATETIME NOT NULL,
		finished_at DATETIME,
		duration_ms INTEGER,
		status TEXT NOT NULL,
		items_processed INTEGER DEFAULT 0,
		items_found INTEGER DEFAULT 0,
		error TEXT,
		details TEXT,
		FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);
	CREATE INDEX IF NOT EXISTS idx_task_history_started_at ON task_history(started_at);

	-- Performance indexes for frequently queried columns
	CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_movies_library_id ON movies(library_id);
	CREATE INDEX IF NOT EXISTS idx_shows_tmdb_id ON shows(tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_shows_library_id ON shows(library_id);
	CREATE INDEX IF NOT EXISTS idx_episodes_season_id ON episodes(season_id);
	CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
	CREATE INDEX IF NOT EXISTS idx_downloads_media ON downloads(media_type, media_id);
	CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
	CREATE INDEX IF NOT EXISTS idx_requests_user_id ON requests(user_id);
	CREATE INDEX IF NOT EXISTS idx_requests_tmdb_id ON requests(type, tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_wanted_tmdb_id ON wanted(type, tmdb_id);

	-- Video chapters
	CREATE TABLE IF NOT EXISTS chapters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_type TEXT NOT NULL,
		media_id INTEGER NOT NULL,
		chapter_index INTEGER NOT NULL,
		title TEXT,
		start_time REAL NOT NULL,
		end_time REAL NOT NULL,
		UNIQUE(media_type, media_id, chapter_index)
	);
	CREATE INDEX IF NOT EXISTS idx_chapters_media ON chapters(media_type, media_id);

	-- Skip segments (intro/credits) for shows
	CREATE TABLE IF NOT EXISTS skip_segments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		show_id INTEGER NOT NULL,
		segment_type TEXT NOT NULL,
		start_time REAL NOT NULL,
		end_time REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(show_id, segment_type),
		FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_skip_segments_show ON skip_segments(show_id);

	-- In-app notifications
	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		image_url TEXT,
		link TEXT,
		read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, read);
	CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);

	-- Collections (franchises, custom lists)
	CREATE TABLE IF NOT EXISTS collections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		tmdb_collection_id INTEGER UNIQUE,
		poster_path TEXT,
		backdrop_path TEXT,
		is_auto INTEGER DEFAULT 0,
		sort_order TEXT DEFAULT 'release',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_collections_tmdb ON collections(tmdb_collection_id);

	CREATE TABLE IF NOT EXISTS collection_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collection_id INTEGER NOT NULL,
		media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'show')),
		media_id INTEGER,
		tmdb_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		year INTEGER,
		poster_path TEXT,
		sort_order INTEGER DEFAULT 0,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
		UNIQUE(collection_id, tmdb_id, media_type)
	);
	CREATE INDEX IF NOT EXISTS idx_collection_items_collection ON collection_items(collection_id);
	CREATE INDEX IF NOT EXISTS idx_collection_items_tmdb ON collection_items(tmdb_id);

	-- Viewing profiles (Netflix-style)
	CREATE TABLE IF NOT EXISTS profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		avatar_url TEXT,
		is_default INTEGER DEFAULT 0,
		is_kid INTEGER DEFAULT 0,
		content_rating_limit TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_profiles_user ON profiles(user_id);

	-- Smart playlists (dynamic collections based on rules)
	CREATE TABLE IF NOT EXISTS smart_playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		name TEXT NOT NULL,
		description TEXT,
		rules TEXT NOT NULL,
		sort_by TEXT DEFAULT 'added',
		sort_order TEXT DEFAULT 'desc',
		limit_count INTEGER,
		media_type TEXT DEFAULT 'both',
		auto_refresh INTEGER DEFAULT 1,
		is_system INTEGER DEFAULT 0,
		last_refreshed DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_smart_playlists_user ON smart_playlists(user_id);

	-- Trakt.tv integration
	CREATE TABLE IF NOT EXISTS trakt_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
		access_token TEXT,
		refresh_token TEXT,
		expires_at DATETIME,
		username TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_watched INTEGER DEFAULT 1,
		sync_ratings INTEGER DEFAULT 1,
		sync_watchlist INTEGER DEFAULT 1,
		last_synced_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_trakt_config_user ON trakt_config(user_id);

	-- Watch history for Trakt sync
	CREATE TABLE IF NOT EXISTS watch_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id INTEGER REFERENCES profiles(id) ON DELETE CASCADE,
		media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'episode')),
		media_id INTEGER NOT NULL,
		tmdb_id INTEGER,
		watched_at DATETIME NOT NULL,
		synced_to_trakt INTEGER DEFAULT 0,
		trakt_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_watch_history_profile ON watch_history(profile_id);
	CREATE INDEX IF NOT EXISTS idx_watch_history_tmdb ON watch_history(tmdb_id);
	CREATE INDEX IF NOT EXISTS idx_watch_history_synced ON watch_history(synced_to_trakt);

	-- Trakt sync queue for async processing
	CREATE TABLE IF NOT EXISTS trakt_sync_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		action TEXT NOT NULL,
		media_type TEXT NOT NULL,
		tmdb_id INTEGER NOT NULL,
		data TEXT,
		status TEXT DEFAULT 'pending',
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		processed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_trakt_queue_status ON trakt_sync_queue(status);
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
		// Prowlarr sync migrations
		"ALTER TABLE indexers ADD COLUMN prowlarr_id INTEGER",
		"ALTER TABLE indexers ADD COLUMN synced_from_prowlarr INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN protocol TEXT",
		"ALTER TABLE indexers ADD COLUMN supports_movies INTEGER DEFAULT 1",
		"ALTER TABLE indexers ADD COLUMN supports_tv INTEGER DEFAULT 1",
		"ALTER TABLE indexers ADD COLUMN supports_music INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN supports_books INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN supports_anime INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN supports_imdb INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN supports_tmdb INTEGER DEFAULT 0",
		"ALTER TABLE indexers ADD COLUMN supports_tvdb INTEGER DEFAULT 0",
		// Request quality profile migrations
		"ALTER TABLE requests ADD COLUMN backdrop_path TEXT",
		"ALTER TABLE requests ADD COLUMN quality_profile_id INTEGER",
		"ALTER TABLE requests ADD COLUMN quality_preset_id INTEGER",
		// Quality preset improvements
		"ALTER TABLE quality_presets ADD COLUMN enabled INTEGER DEFAULT 1",
		"ALTER TABLE quality_presets ADD COLUMN priority INTEGER DEFAULT 100",
		// Wanted table preset support
		"ALTER TABLE wanted ADD COLUMN quality_preset_id INTEGER",
		// Wanted table IMDB ID support for better search matching
		"ALTER TABLE wanted ADD COLUMN imdb_id TEXT",
		// Media type for presets (movie, tv, anime)
		"ALTER TABLE quality_presets ADD COLUMN media_type TEXT DEFAULT 'movie'",
		// Anime preferences
		"ALTER TABLE quality_presets ADD COLUMN prefer_dual_audio INTEGER DEFAULT 0",
		"ALTER TABLE quality_presets ADD COLUMN prefer_dubbed INTEGER DEFAULT 0",
		"ALTER TABLE quality_presets ADD COLUMN preferred_language TEXT DEFAULT 'any'",
		// Per-season monitoring
		"ALTER TABLE media_quality_override ADD COLUMN monitored_seasons TEXT DEFAULT ''",
		// Audio/subtitle preferences for shows
		"ALTER TABLE media_quality_override ADD COLUMN preferred_audio_lang TEXT DEFAULT ''",
		"ALTER TABLE media_quality_override ADD COLUMN preferred_subtitle_lang TEXT DEFAULT ''",
		// Parental controls
		"ALTER TABLE users ADD COLUMN content_rating_limit TEXT",
		"ALTER TABLE users ADD COLUMN pin_hash TEXT",
		"ALTER TABLE users ADD COLUMN require_pin INTEGER DEFAULT 0",
		// Profile support for progress tracking
		"ALTER TABLE progress ADD COLUMN profile_id INTEGER",
		// Session active profile tracking
		"ALTER TABLE sessions ADD COLUMN active_profile_id INTEGER",
		// Upgrade tracking for wanted items
		"ALTER TABLE wanted ADD COLUMN is_upgrade INTEGER DEFAULT 0",
		"ALTER TABLE wanted ADD COLUMN existing_media_id INTEGER",
		// Upgrade search tracking for quality status
		"ALTER TABLE media_quality_status ADD COLUMN upgrade_searched_at DATETIME",
		"ALTER TABLE media_quality_status ADD COLUMN current_score INTEGER DEFAULT 0",
		"ALTER TABLE media_quality_status ADD COLUMN cutoff_score INTEGER DEFAULT 0",
	}
	for _, m := range migrations {
		// Ignore errors (column may already exist)
		d.db.Exec(m)
	}

	// Drop old unique index and create new one with media_type
	d.db.Exec(`DROP INDEX IF EXISTS idx_quality_presets_name`)
	d.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_quality_presets_name_type ON quality_presets(name, media_type)`)

	// Delete old presets without media_type to re-seed properly
	d.db.Exec(`DELETE FROM quality_presets WHERE media_type IS NULL OR media_type = ''`)

	// Delete old built-in presets to reseed with new structure
	d.db.Exec(`DELETE FROM quality_presets WHERE is_built_in = 1`)

	// Seed built-in quality presets by media type
	builtInPresets := []struct {
		name       string
		mediaType  string
		priority   int
		resolution string
		source     string
		hdrFormats string
		audioFmts  string
		codec      string
		minSeeders int
		isDefault  bool
	}{
		// Movie presets (8 total - quality matters most)
		// 4K Tier (3 presets)
		{"4K Remux", "movie", 10, "4k", "remux", `["dv", "hdr10plus", "hdr10"]`, `["atmos", "truehd", "dtshd", "dtsx"]`, "", 3, true},
		{"4K HDR", "movie", 20, "4k", "bluray", `["dv", "hdr10plus", "hdr10"]`, `["atmos", "truehd", "dtshd", "ddplus"]`, "", 3, false},
		{"4K", "movie", 30, "4k", "web", "", "", "", 3, false},
		// 1080p Tier (3 presets)
		{"1080p Remux", "movie", 40, "1080p", "remux", "", `["atmos", "truehd", "dtshd", "dtsx"]`, "", 3, false},
		{"1080p BluRay", "movie", 50, "1080p", "bluray", "", `["truehd", "dtshd", "ddplus", "dts"]`, "", 3, false},
		{"1080p", "movie", 60, "1080p", "web", "", "", "", 3, false},
		// 720p Tier (1 preset)
		{"720p", "movie", 70, "720p", "web", "", "", "", 2, false},
		// 480p Tier (1 preset)
		{"480p", "movie", 80, "480p", "web", "", "", "", 1, false},

		// TV presets (6 total)
		// 4K Tier (2 presets)
		{"4K HDR", "tv", 10, "4k", "web", `["dv", "hdr10plus", "hdr10"]`, "", "", 3, true},
		{"4K", "tv", 20, "4k", "web", "", "", "", 3, false},
		// 1080p Tier (2 presets)
		{"1080p", "tv", 30, "1080p", "web", "", "", "", 3, false},
		{"1080p HDTV", "tv", 40, "1080p", "hdtv", "", "", "", 2, false},
		// 720p Tier (1 preset)
		{"720p", "tv", 50, "720p", "web", "", "", "", 2, false},
		// Any/480p Tier (1 preset)
		{"Any", "tv", 60, "any", "any", "", "", "", 1, false},

		// Anime presets (4 total - simple with editable preferences)
		{"4K", "anime", 10, "4k", "bluray", `["dv", "hdr10plus", "hdr10"]`, `["flac", "aac", "opus"]`, "", 2, true},
		{"1080p", "anime", 20, "1080p", "bluray", "", `["flac", "aac", "opus"]`, "", 2, false},
		{"720p", "anime", 30, "720p", "web", "", `["aac", "opus"]`, "", 2, false},
		{"480p", "anime", 40, "480p", "web", "", `["aac", "opus"]`, "", 1, false},
	}

	for _, p := range builtInPresets {
		defaultVal := 0
		if p.isDefault {
			defaultVal = 1
		}
		hdrFormats := p.hdrFormats
		if hdrFormats == "" {
			hdrFormats = "[]"
		}
		audioFormats := p.audioFmts
		if audioFormats == "" {
			audioFormats = "[]"
		}
		d.db.Exec(`INSERT OR IGNORE INTO quality_presets (name, media_type, is_built_in, is_default, enabled, priority, resolution, source, hdr_formats, audio_formats, codec, min_seeders, auto_upgrade, prefer_dual_audio, prefer_dubbed, preferred_language)
			VALUES (?, ?, 1, ?, 1, ?, ?, ?, ?, ?, ?, ?, 1, 0, 0, 'any')`,
			p.name, p.mediaType, defaultVal, p.priority, p.resolution, p.source, hdrFormats, audioFormats, p.codec, p.minSeeders)
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

	// Seed default quality profiles if none exist
	var profileCount int
	d.db.QueryRow("SELECT COUNT(*) FROM quality_profiles").Scan(&profileCount)
	if profileCount == 0 {
		profiles := []string{
			`INSERT INTO quality_profiles (name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores) VALUES ('Any', 1, 10000, 0, 0, '["2160p","1080p","720p","480p"]', '{}')`,
			`INSERT INTO quality_profiles (name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores) VALUES ('HD-1080p', 1, 10000, 0, 0, '["1080p","720p"]', '{}')`,
			`INSERT INTO quality_profiles (name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores) VALUES ('Ultra-HD', 1, 10000, 0, 0, '["2160p","1080p"]', '{}')`,
		}
		for _, p := range profiles {
			d.db.Exec(p)
		}
	}

	// Seed default scheduler settings (use INSERT OR IGNORE to avoid duplicates)
	defaultSettings := map[string]string{
		"scheduler_auto_search":          "true",
		"scheduler_auto_grab":            "true",
		"scheduler_rss_enabled":          "true",
		"scheduler_min_score":            "0",
		"storage_pause_enabled":          "false",
		"storage_threshold_gb":           "50",
		"upgrade_search_enabled":         "false",
		"upgrade_search_limit":           "10",
		"upgrade_search_interval":        "720",
		"upgrade_delete_old":             "true",
		"opensubtitles_api_key":          "",
		"opensubtitles_languages":        "en",
		"opensubtitles_auto_download":    "false",
		"opensubtitles_hearing_impaired": "include",
	}
	for key, value := range defaultSettings {
		d.db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, key, value)
	}

	// Create default profiles for users without any profiles
	d.db.Exec(`
		INSERT INTO profiles (user_id, name, is_default, created_at)
		SELECT id, username, 1, CURRENT_TIMESTAMP
		FROM users
		WHERE id NOT IN (SELECT DISTINCT user_id FROM profiles)
	`)

	// Migrate progress records to use the default profile for each user
	// First, get the first user's default profile (for single-user progress migration)
	var defaultProfileID int64
	d.db.QueryRow(`SELECT id FROM profiles WHERE is_default = 1 LIMIT 1`).Scan(&defaultProfileID)
	if defaultProfileID > 0 {
		d.db.Exec(`UPDATE progress SET profile_id = ? WHERE profile_id IS NULL`, defaultProfileID)
	}

	// Create built-in smart playlists if they don't exist
	d.CreateBuiltInSmartPlaylists()

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

// OwnedEpisode represents an owned episode with season/episode numbers
type OwnedEpisode struct {
	SeasonNumber  int
	EpisodeNumber int
}

// GetOwnedEpisodesByShow returns all owned episodes for a show as season/episode number pairs
func (d *Database) GetOwnedEpisodesByShow(showID int64) ([]OwnedEpisode, error) {
	rows, err := d.db.Query(`
		SELECT s.season_number, e.episode_number
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE s.show_id = ?
		ORDER BY s.season_number, e.episode_number`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []OwnedEpisode
	for rows.Next() {
		var ep OwnedEpisode
		if err := rows.Scan(&ep.SeasonNumber, &ep.EpisodeNumber); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}
	return episodes, nil
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

func (d *Database) GetShowIDForEpisode(episodeID int64) (int64, error) {
	var showID int64
	err := d.db.QueryRow(`
		SELECT s.show_id
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE e.id = ?`, episodeID,
	).Scan(&showID)
	return showID, err
}

// GetEpisodeByShowSeasonEpisode finds an episode by show ID, season number, and episode number
func (d *Database) GetEpisodeByShowSeasonEpisode(showID int64, seasonNum, episodeNum int) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT e.id, e.season_id, e.episode_number, e.title, e.overview, e.air_date, e.runtime, e.still_path, e.path, e.size
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE s.show_id = ? AND s.season_number = ? AND e.episode_number = ?`,
		showID, seasonNum, episodeNum,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) DeleteEpisode(id int64) error {
	_, err := d.db.Exec("DELETE FROM episodes WHERE id = ?", id)
	return err
}

// GetEpisodesByLibrary retrieves all episodes for a library (for cleanup)
func (d *Database) GetEpisodesByLibrary(libraryID int64) ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT e.id, e.episode_number, e.path
		FROM episodes e
		JOIN seasons sea ON e.season_id = sea.id
		JOIN shows s ON sea.show_id = s.id
		WHERE s.library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.EpisodeNumber, &e.Path); err != nil {
			continue
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
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

// DeleteMovie removes a movie from the database
func (d *Database) DeleteMovie(id int64) error {
	_, err := d.db.Exec("DELETE FROM movies WHERE id = ?", id)
	return err
}

// GetMoviesByLibrary retrieves all movies for a library (for cleanup)
func (d *Database) GetMoviesByLibrary(libraryID int64) ([]Movie, error) {
	rows, err := d.db.Query(`SELECT id, title, path FROM movies WHERE library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Path); err != nil {
			continue
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// GetMovieByTmdb retrieves a movie by its TMDB ID
func (d *Database) GetMovieByTmdb(tmdbID int64) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE tmdb_id = ?`, tmdbID,
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

// GetShowByTmdb retrieves a show by its TMDB ID
func (d *Database) GetShowByTmdb(tmdbID int64) (*Show, error) {
	var s Show
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year, overview,
			status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path,
			focal_x, focal_y, path, added_at
		FROM shows WHERE tmdb_id = ?`, tmdbID,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew, &s.Network,
		&s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &s.AddedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
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

// FormatSettings controls which file formats are acceptable for download
type FormatSettings struct {
	AcceptedContainers []string `json:"acceptedContainers"` // e.g., ["mkv", "mp4", "avi"]
	RejectedKeywords   []string `json:"rejectedKeywords"`   // Keywords to reject (e.g., "bdmv", "rar", "cam")
	AutoBlocklist      bool     `json:"autoBlocklist"`      // Add rejected releases to blocklist
}

// DefaultFormatSettings returns sensible defaults
func DefaultFormatSettings() *FormatSettings {
	return &FormatSettings{
		AcceptedContainers: []string{"mkv", "mp4", "avi", "mov", "webm", "m4v", "ts", "m2ts", "wmv", "flv"},
		RejectedKeywords: []string{
			// Disc releases
			"bdmv", "video_ts", "iso", "full disc", "complete disc", "disc1", "disc2",
			// Archives
			"rar", "zip", "7z",
			// Low quality captures
			"cam", "camrip", "hdcam", "hdts", "telesync", "telecine", "ts-scr",
			"dvdscr", "dvdscreener", "screener", "scr", "r5", "workprint",
			// Samples
			"sample",
			// 3D (most people don't want)
			"3d", "hsbs", "hou",
		},
		AutoBlocklist: true,
	}
}

// GetFormatSettings retrieves format settings from database
func (d *Database) GetFormatSettings() (*FormatSettings, error) {
	value, err := d.GetSetting("format_settings")
	if err != nil {
		// Return defaults if not set
		return DefaultFormatSettings(), nil
	}

	var settings FormatSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultFormatSettings(), nil
	}
	return &settings, nil
}

// SaveFormatSettings stores format settings in database
func (d *Database) SaveFormatSettings(settings *FormatSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return d.SetSetting("format_settings", string(data))
}

// Progress operations

func (d *Database) GetProgress(profileID int64, mediaType string, mediaID int64) (*Progress, error) {
	var p Progress
	err := d.db.QueryRow(
		"SELECT id, COALESCE(profile_id, 0), media_type, media_id, position, duration, updated_at FROM progress WHERE media_type = ? AND media_id = ? AND (profile_id = ? OR profile_id IS NULL)",
		mediaType, mediaID, profileID,
	).Scan(&p.ID, &p.ProfileID, &p.MediaType, &p.MediaID, &p.Position, &p.Duration, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *Database) SaveProgress(p *Progress) error {
	_, err := d.db.Exec(`
		INSERT INTO progress (profile_id, media_type, media_id, position, duration, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(media_type, media_id) DO UPDATE SET
			profile_id = excluded.profile_id,
			position = excluded.position,
			duration = excluded.duration,
			updated_at = CURRENT_TIMESTAMP
	`, p.ProfileID, p.MediaType, p.MediaID, p.Position, p.Duration)
	return err
}

// Chapter operations

func (d *Database) GetChapters(mediaType string, mediaID int64) ([]Chapter, error) {
	rows, err := d.db.Query(
		"SELECT id, media_type, media_id, chapter_index, title, start_time, end_time FROM chapters WHERE media_type = ? AND media_id = ? ORDER BY chapter_index",
		mediaType, mediaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []Chapter
	for rows.Next() {
		var c Chapter
		var title sql.NullString
		if err := rows.Scan(&c.ID, &c.MediaType, &c.MediaID, &c.ChapterIndex, &title, &c.StartTime, &c.EndTime); err != nil {
			return nil, err
		}
		if title.Valid {
			c.Title = title.String
		}
		chapters = append(chapters, c)
	}
	return chapters, nil
}

func (d *Database) SaveChapters(mediaType string, mediaID int64, chapters []Chapter) error {
	// Delete existing chapters first
	if err := d.DeleteChapters(mediaType, mediaID); err != nil {
		return err
	}

	// Insert new chapters
	for _, c := range chapters {
		_, err := d.db.Exec(
			"INSERT INTO chapters (media_type, media_id, chapter_index, title, start_time, end_time) VALUES (?, ?, ?, ?, ?, ?)",
			mediaType, mediaID, c.ChapterIndex, c.Title, c.StartTime, c.EndTime,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) DeleteChapters(mediaType string, mediaID int64) error {
	_, err := d.db.Exec("DELETE FROM chapters WHERE media_type = ? AND media_id = ?", mediaType, mediaID)
	return err
}

// Skip segment operations

func (d *Database) GetSkipSegments(showID int64) (*SkipSegments, error) {
	rows, err := d.db.Query(
		"SELECT segment_type, start_time, end_time FROM skip_segments WHERE show_id = ?",
		showID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := &SkipSegments{}
	for rows.Next() {
		var segmentType string
		var startTime, endTime float64
		if err := rows.Scan(&segmentType, &startTime, &endTime); err != nil {
			return nil, err
		}
		segment := &SkipSegment{StartTime: startTime, EndTime: endTime}
		if segmentType == "intro" {
			segments.Intro = segment
		} else if segmentType == "credits" {
			segments.Credits = segment
		}
	}
	return segments, nil
}

func (d *Database) SaveSkipSegment(showID int64, segmentType string, startTime, endTime float64) error {
	_, err := d.db.Exec(
		`INSERT INTO skip_segments (show_id, segment_type, start_time, end_time)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(show_id, segment_type) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time`,
		showID, segmentType, startTime, endTime,
	)
	return err
}

func (d *Database) DeleteSkipSegment(showID int64, segmentType string) error {
	_, err := d.db.Exec("DELETE FROM skip_segments WHERE show_id = ? AND segment_type = ?", showID, segmentType)
	return err
}

// User operations

func (d *Database) CreateUser(user *User) error {
	result, err := d.db.Exec(
		"INSERT INTO users (username, password_hash, role, content_rating_limit, pin_hash, require_pin) VALUES (?, ?, ?, ?, ?, ?)",
		user.Username, user.PasswordHash, user.Role, user.ContentRatingLimit, user.PinHash, user.RequirePin,
	)
	if err != nil {
		return err
	}
	user.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	var u User
	var requirePin int
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.RequirePin = requirePin == 1
	return &u, nil
}

func (d *Database) GetUserByID(id int64) (*User, error) {
	var u User
	var requirePin int
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.RequirePin = requirePin == 1
	return &u, nil
}

func (d *Database) GetUsers() ([]User, error) {
	rows, err := d.db.Query("SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var requirePin int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.RequirePin = requirePin == 1
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) UpdateUser(user *User) error {
	_, err := d.db.Exec(
		"UPDATE users SET username = ?, role = ?, content_rating_limit = ?, require_pin = ? WHERE id = ?",
		user.Username, user.Role, user.ContentRatingLimit, user.RequirePin, user.ID,
	)
	return err
}

func (d *Database) UpdateUserPin(id int64, pinHash *string) error {
	_, err := d.db.Exec("UPDATE users SET pin_hash = ? WHERE id = ?", pinHash, id)
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

// Profile operations

func (d *Database) CreateProfile(profile *Profile) error {
	result, err := d.db.Exec(
		`INSERT INTO profiles (user_id, name, avatar_url, is_default, is_kid, content_rating_limit)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		profile.UserID, profile.Name, profile.AvatarURL, profile.IsDefault, profile.IsKid, profile.ContentRatingLimit,
	)
	if err != nil {
		return err
	}
	profile.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetProfile(id int64) (*Profile, error) {
	var p Profile
	var isDefault, isKid int
	err := d.db.QueryRow(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsKid = isKid == 1
	return &p, nil
}

func (d *Database) GetProfilesByUser(userID int64) ([]Profile, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE user_id = ? ORDER BY is_default DESC, created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var isDefault, isKid int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		p.IsKid = isKid == 1
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) GetDefaultProfile(userID int64) (*Profile, error) {
	var p Profile
	var isDefault, isKid int
	err := d.db.QueryRow(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE user_id = ? AND is_default = 1`, userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsKid = isKid == 1
	return &p, nil
}

func (d *Database) UpdateProfile(profile *Profile) error {
	_, err := d.db.Exec(
		`UPDATE profiles SET name = ?, avatar_url = ?, is_kid = ?, content_rating_limit = ?
		 WHERE id = ?`,
		profile.Name, profile.AvatarURL, profile.IsKid, profile.ContentRatingLimit, profile.ID,
	)
	return err
}

func (d *Database) DeleteProfile(id int64) error {
	_, err := d.db.Exec("DELETE FROM profiles WHERE id = ?", id)
	return err
}

func (d *Database) CountProfilesByUser(userID int64) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM profiles WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (d *Database) CreateDefaultProfileForUser(userID int64, username string) (*Profile, error) {
	profile := &Profile{
		UserID:    userID,
		Name:      username,
		IsDefault: true,
		IsKid:     false,
	}
	if err := d.CreateProfile(profile); err != nil {
		return nil, err
	}
	return profile, nil
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
		"SELECT id, user_id, token, expires_at, active_profile_id FROM sessions WHERE token = ?", token,
	).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.ActiveProfileID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) SetActiveProfile(token string, profileID int64) error {
	_, err := d.db.Exec("UPDATE sessions SET active_profile_id = ? WHERE token = ?", profileID, token)
	return err
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

// PIN elevation operations

func (d *Database) CreatePinElevation(elevation *PinElevation) error {
	result, err := d.db.Exec(
		"INSERT INTO pin_elevations (user_id, token, expires_at) VALUES (?, ?, ?)",
		elevation.UserID, elevation.Token, elevation.ExpiresAt,
	)
	if err != nil {
		return err
	}
	elevation.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetPinElevationByToken(token string) (*PinElevation, error) {
	var e PinElevation
	err := d.db.QueryRow(
		"SELECT id, user_id, token, expires_at FROM pin_elevations WHERE token = ? AND expires_at > CURRENT_TIMESTAMP", token,
	).Scan(&e.ID, &e.UserID, &e.Token, &e.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) DeletePinElevation(token string) error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE token = ?", token)
	return err
}

func (d *Database) DeleteExpiredPinElevations() error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (d *Database) DeleteUserPinElevations(userID int64) error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE user_id = ?", userID)
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
		SELECT id, name, type, url, COALESCE(api_key, ''), COALESCE(categories, ''), priority, enabled,
			COALESCE(prowlarr_id, 0), COALESCE(synced_from_prowlarr, 0), COALESCE(protocol, ''),
			COALESCE(supports_movies, 1), COALESCE(supports_tv, 1), COALESCE(supports_music, 0),
			COALESCE(supports_books, 0), COALESCE(supports_anime, 0), COALESCE(supports_imdb, 0),
			COALESCE(supports_tmdb, 0), COALESCE(supports_tvdb, 0)
		FROM indexers ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}

func (d *Database) GetIndexer(id int64) (*Indexer, error) {
	var i Indexer
	var prowlarrID int64
	var syncedFromProwlarr int
	err := d.db.QueryRow(`
		SELECT id, name, type, url, COALESCE(api_key, ''), COALESCE(categories, ''), priority, enabled,
			COALESCE(prowlarr_id, 0), COALESCE(synced_from_prowlarr, 0), COALESCE(protocol, ''),
			COALESCE(supports_movies, 1), COALESCE(supports_tv, 1), COALESCE(supports_music, 0),
			COALESCE(supports_books, 0), COALESCE(supports_anime, 0), COALESCE(supports_imdb, 0),
			COALESCE(supports_tmdb, 0), COALESCE(supports_tvdb, 0)
		FROM indexers WHERE id = ?`, id,
	).Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
		&i.Categories, &i.Priority, &i.Enabled,
		&prowlarrID, &syncedFromProwlarr, &i.Protocol,
		&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
		&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
		&i.SupportsTMDB, &i.SupportsTVDB)
	if err != nil {
		return nil, err
	}
	if prowlarrID > 0 {
		i.ProwlarrID = &prowlarrID
	}
	i.SyncedFromProwlarr = syncedFromProwlarr == 1
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
		SELECT id, name, type, url, COALESCE(api_key, ''), COALESCE(categories, ''), priority, enabled,
			COALESCE(prowlarr_id, 0), COALESCE(synced_from_prowlarr, 0), COALESCE(protocol, ''),
			COALESCE(supports_movies, 1), COALESCE(supports_tv, 1), COALESCE(supports_music, 0),
			COALESCE(supports_books, 0), COALESCE(supports_anime, 0), COALESCE(supports_imdb, 0),
			COALESCE(supports_tmdb, 0), COALESCE(supports_tvdb, 0)
		FROM indexers WHERE enabled = 1 ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
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
		INSERT INTO wanted (type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, quality_preset_id, monitored, seasons)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Type, item.TmdbID, item.ImdbID, item.Title, item.Year, item.PosterPath,
		item.QualityProfileID, item.QualityPresetID, item.Monitored, item.Seasons,
	)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetWantedItems() ([]WantedItem, error) {
	rows, err := d.db.Query(`
		SELECT id, type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, quality_preset_id, monitored, seasons, last_searched, added_at
		FROM wanted ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WantedItem
	for rows.Next() {
		var item WantedItem
		if err := rows.Scan(&item.ID, &item.Type, &item.TmdbID, &item.ImdbID, &item.Title, &item.Year,
			&item.PosterPath, &item.QualityProfileID, &item.QualityPresetID, &item.Monitored, &item.Seasons,
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
		SELECT id, type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, quality_preset_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE id = ?`, id,
	).Scan(&item.ID, &item.Type, &item.TmdbID, &item.ImdbID, &item.Title, &item.Year,
		&item.PosterPath, &item.QualityProfileID, &item.QualityPresetID, &item.Monitored, &item.Seasons,
		&item.LastSearched, &item.AddedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *Database) GetWantedByTmdb(itemType string, tmdbID int64) (*WantedItem, error) {
	var item WantedItem
	err := d.db.QueryRow(`
		SELECT id, type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, quality_preset_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE type = ? AND tmdb_id = ?`, itemType, tmdbID,
	).Scan(&item.ID, &item.Type, &item.TmdbID, &item.ImdbID, &item.Title, &item.Year,
		&item.PosterPath, &item.QualityProfileID, &item.QualityPresetID, &item.Monitored, &item.Seasons,
		&item.LastSearched, &item.AddedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *Database) GetMonitoredItems() ([]WantedItem, error) {
	rows, err := d.db.Query(`
		SELECT id, type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, quality_preset_id, monitored, seasons, last_searched, added_at
		FROM wanted WHERE monitored = 1 ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WantedItem
	for rows.Next() {
		var item WantedItem
		if err := rows.Scan(&item.ID, &item.Type, &item.TmdbID, &item.ImdbID, &item.Title, &item.Year,
			&item.PosterPath, &item.QualityProfileID, &item.QualityPresetID, &item.Monitored, &item.Seasons,
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
			quality_profile_id = ?, quality_preset_id = ?, monitored = ?, seasons = ?
		WHERE id = ?`,
		item.QualityProfileID, item.QualityPresetID, item.Monitored, item.Seasons, item.ID,
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
		INSERT INTO requests (user_id, type, tmdb_id, title, year, overview, poster_path, backdrop_path, quality_profile_id, quality_preset_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.UserID, req.Type, req.TmdbID, req.Title, req.Year, req.Overview, req.PosterPath, req.BackdropPath, req.QualityProfileID, req.QualityPresetID, "requested",
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
		       r.poster_path, r.backdrop_path, r.quality_profile_id, r.quality_preset_id, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.status != 'denied'
		ORDER BY r.requested_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID, &req.Title,
			&req.Year, &req.Overview, &req.PosterPath, &req.BackdropPath, &req.QualityProfileID, &req.QualityPresetID,
			&req.Status, &req.StatusReason, &req.RequestedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (d *Database) GetRequestsByUser(userID int64) ([]Request, error) {
	rows, err := d.db.Query(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.backdrop_path, r.quality_profile_id, r.quality_preset_id, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.user_id = ? AND r.status != 'denied'
		ORDER BY r.requested_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var req Request
		if err := rows.Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID, &req.Title,
			&req.Year, &req.Overview, &req.PosterPath, &req.BackdropPath, &req.QualityProfileID, &req.QualityPresetID,
			&req.Status, &req.StatusReason, &req.RequestedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (d *Database) GetRequestsByStatus(status string) ([]Request, error) {
	rows, err := d.db.Query(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.backdrop_path, r.quality_profile_id, r.quality_preset_id, r.status, r.status_reason, r.requested_at, r.updated_at
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
			&req.Year, &req.Overview, &req.PosterPath, &req.BackdropPath, &req.QualityProfileID, &req.QualityPresetID,
			&req.Status, &req.StatusReason, &req.RequestedAt, &req.UpdatedAt); err != nil {
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
		       r.poster_path, r.backdrop_path, r.quality_profile_id, r.quality_preset_id, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.id = ?`, id).Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID,
		&req.Title, &req.Year, &req.Overview, &req.PosterPath, &req.BackdropPath, &req.QualityProfileID, &req.QualityPresetID,
		&req.Status, &req.StatusReason, &req.RequestedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (d *Database) GetRequestByTmdb(userID int64, mediaType string, tmdbID int64) (*Request, error) {
	var req Request
	err := d.db.QueryRow(`
		SELECT r.id, r.user_id, u.username, r.type, r.tmdb_id, r.title, r.year, r.overview,
		       r.poster_path, r.backdrop_path, r.quality_profile_id, r.quality_preset_id, r.status, r.status_reason, r.requested_at, r.updated_at
		FROM requests r
		LEFT JOIN users u ON r.user_id = u.id
		WHERE r.user_id = ? AND r.type = ? AND r.tmdb_id = ?`,
		userID, mediaType, tmdbID).Scan(&req.ID, &req.UserID, &req.Username, &req.Type, &req.TmdbID,
		&req.Title, &req.Year, &req.Overview, &req.PosterPath, &req.BackdropPath, &req.QualityProfileID, &req.QualityPresetID,
		&req.Status, &req.StatusReason, &req.RequestedAt, &req.UpdatedAt)
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

func (d *Database) DeleteDeniedRequests() (int64, error) {
	result, err := d.db.Exec("DELETE FROM requests WHERE status = 'denied'")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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

	// Check if requested (exclude denied so users can re-request)
	var requestID int64
	var requestStatus string
	err = d.db.QueryRow("SELECT id, status FROM requests WHERE type = 'movie' AND status != 'denied' AND tmdb_id = ?", tmdbID).Scan(&requestID, &requestStatus)
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

	// Check if requested (exclude denied so users can re-request)
	var requestID int64
	var requestStatus string
	err = d.db.QueryRow("SELECT id, status FROM requests WHERE type = 'tv' AND status != 'denied' AND tmdb_id = ?", tmdbID).Scan(&requestID, &requestStatus)
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

	// Check request status (exclude denied requests so users can re-request)
	rows2, err := d.db.Query("SELECT id, tmdb_id, status FROM requests WHERE type = 'movie' AND status != 'denied' AND tmdb_id IN ("+placeholderStr+")", args...)
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

	// Check request status (exclude denied requests so users can re-request)
	rows2, err := d.db.Query("SELECT id, tmdb_id, status FROM requests WHERE type = 'tv' AND status != 'denied' AND tmdb_id IN ("+placeholderStr+")", args...)
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
		SELECT id, name, COALESCE(media_type, 'movie') as media_type, is_default, is_built_in, enabled, priority, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade,
		       COALESCE(prefer_dual_audio, 0) as prefer_dual_audio,
		       COALESCE(prefer_dubbed, 0) as prefer_dubbed,
		       COALESCE(preferred_language, 'any') as preferred_language,
		       created_at, updated_at
		FROM quality_presets
		ORDER BY media_type ASC, priority ASC, is_default DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []QualityPreset
	for rows.Next() {
		var p QualityPreset
		var isDefault, isBuiltIn, enabled, preferSeasonPacks, autoUpgrade, preferDualAudio, preferDubbed int
		var hdrFormatsJSON, audioFormatsJSON *string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.MediaType, &isDefault, &isBuiltIn, &enabled, &p.Priority, &p.Resolution, &p.Source,
			&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
			&p.MinSeeders, &preferSeasonPacks, &autoUpgrade,
			&preferDualAudio, &preferDubbed, &p.PreferredLanguage,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		p.IsBuiltIn = isBuiltIn == 1
		p.Enabled = enabled == 1
		p.PreferSeasonPacks = preferSeasonPacks == 1
		p.AutoUpgrade = autoUpgrade == 1
		p.PreferDualAudio = preferDualAudio == 1
		p.PreferDubbed = preferDubbed == 1
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
	var isDefault, isBuiltIn, enabled, preferSeasonPacks, autoUpgrade, preferDualAudio, preferDubbed int
	var hdrFormatsJSON, audioFormatsJSON *string
	err := d.db.QueryRow(`
		SELECT id, name, COALESCE(media_type, 'movie') as media_type, is_default, is_built_in, enabled, priority, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade,
		       COALESCE(prefer_dual_audio, 0) as prefer_dual_audio,
		       COALESCE(prefer_dubbed, 0) as prefer_dubbed,
		       COALESCE(preferred_language, 'any') as preferred_language,
		       created_at, updated_at
		FROM quality_presets WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &p.MediaType, &isDefault, &isBuiltIn, &enabled, &p.Priority, &p.Resolution, &p.Source,
		&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
		&p.MinSeeders, &preferSeasonPacks, &autoUpgrade,
		&preferDualAudio, &preferDubbed, &p.PreferredLanguage,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsBuiltIn = isBuiltIn == 1
	p.Enabled = enabled == 1
	p.PreferSeasonPacks = preferSeasonPacks == 1
	p.AutoUpgrade = autoUpgrade == 1
	p.PreferDualAudio = preferDualAudio == 1
	p.PreferDubbed = preferDubbed == 1
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
	var isDefault, isBuiltIn, enabled, preferSeasonPacks, autoUpgrade, preferDualAudio, preferDubbed int
	var hdrFormatsJSON, audioFormatsJSON *string
	err := d.db.QueryRow(`
		SELECT id, name, COALESCE(media_type, 'movie') as media_type, is_default, is_built_in, enabled, priority, resolution, source,
		       hdr_formats, codec, audio_formats, preferred_edition,
		       min_seeders, prefer_season_packs, auto_upgrade,
		       COALESCE(prefer_dual_audio, 0) as prefer_dual_audio,
		       COALESCE(prefer_dubbed, 0) as prefer_dubbed,
		       COALESCE(preferred_language, 'any') as preferred_language,
		       created_at, updated_at
		FROM quality_presets WHERE is_default = 1 LIMIT 1
	`).Scan(
		&p.ID, &p.Name, &p.MediaType, &isDefault, &isBuiltIn, &enabled, &p.Priority, &p.Resolution, &p.Source,
		&hdrFormatsJSON, &p.Codec, &audioFormatsJSON, &p.PreferredEdition,
		&p.MinSeeders, &preferSeasonPacks, &autoUpgrade,
		&preferDualAudio, &preferDubbed, &p.PreferredLanguage,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsBuiltIn = isBuiltIn == 1
	p.Enabled = enabled == 1
	p.PreferSeasonPacks = preferSeasonPacks == 1
	p.AutoUpgrade = autoUpgrade == 1
	p.PreferDualAudio = preferDualAudio == 1
	p.PreferDubbed = preferDubbed == 1
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
	// Default to enabled with priority 100 for new presets
	enabled := 1
	if !p.Enabled {
		enabled = 0
	}
	priority := p.Priority
	if priority == 0 {
		priority = 100
	}
	mediaType := p.MediaType
	if mediaType == "" {
		mediaType = "movie"
	}
	result, err := d.db.Exec(`
		INSERT INTO quality_presets (name, media_type, is_default, is_built_in, enabled, priority, resolution, source,
		                            hdr_formats, codec, audio_formats, preferred_edition,
		                            min_seeders, prefer_season_packs, auto_upgrade)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.Name, mediaType, p.IsDefault, p.IsBuiltIn, enabled, priority, p.Resolution, p.Source,
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
			name = ?, enabled = ?, priority = ?, resolution = ?, source = ?, hdr_formats = ?,
			codec = ?, audio_formats = ?, preferred_edition = ?,
			min_seeders = ?, prefer_season_packs = ?, auto_upgrade = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_built_in = 0
	`, p.Name, p.Enabled, p.Priority, p.Resolution, p.Source, string(hdrFormatsJSON),
		p.Codec, string(audioFormatsJSON), p.PreferredEdition,
		p.MinSeeders, p.PreferSeasonPacks, p.AutoUpgrade, p.ID)
	return err
}

// ToggleQualityPresetEnabled allows toggling enabled status for any preset (including built-in)
func (d *Database) ToggleQualityPresetEnabled(id int64, enabled bool) error {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	_, err := d.db.Exec(`
		UPDATE quality_presets SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, enabledVal, id)
	return err
}

// UpdateQualityPresetPriority allows changing priority for any preset (including built-in)
func (d *Database) UpdateQualityPresetPriority(id int64, priority int) error {
	_, err := d.db.Exec(`
		UPDATE quality_presets SET priority = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, priority, id)
	return err
}

// UpdateQualityPresetAnimePreferences updates anime-specific preferences for a preset
func (d *Database) UpdateQualityPresetAnimePreferences(id int64, preferDualAudio, preferDubbed *bool, preferredLanguage *string) error {
	// Build dynamic SQL based on which fields are provided
	query := "UPDATE quality_presets SET updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{}

	if preferDualAudio != nil {
		val := 0
		if *preferDualAudio {
			val = 1
		}
		query += ", prefer_dual_audio = ?"
		args = append(args, val)
	}
	if preferDubbed != nil {
		val := 0
		if *preferDubbed {
			val = 1
		}
		query += ", prefer_dubbed = ?"
		args = append(args, val)
	}
	if preferredLanguage != nil {
		query += ", preferred_language = ?"
		args = append(args, *preferredLanguage)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := d.db.Exec(query, args...)
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
		       retry_count, COALESCE(stalled_notified, 0), created_at, updated_at
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
		var stalledNotified int
		if err := rows.Scan(
			&dl.ID, &dl.DownloadClientID, &dl.ExternalID, &dl.MediaID, &dl.MediaType,
			&dl.Title, &dl.Size, &dl.State, &dl.Progress, &dl.DownloadPath,
			&dl.ImportedPath, &dl.Error, &dl.RetryCount, &stalledNotified, &dl.CreatedAt, &dl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		dl.StalledNotified = stalledNotified == 1
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
		&dl.Title, &dl.Size, &dl.State, &dl.Progress, &dl.DownloadPath,
		&dl.ImportedPath, &dl.Error, &dl.CreatedAt, &dl.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No existing download found - not an error
	}
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
		dl.Title, dl.Size, dl.State, dl.Progress, dl.DownloadPath)
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
	`, dl.State, dl.Progress, dl.DownloadPath, dl.ImportedPath, dl.Error, dl.ID)
	return err
}

func (d *Database) DeleteDownload(id int64) error {
	_, err := d.db.Exec("DELETE FROM downloads WHERE id = ?", id)
	return err
}

// MarkDownloadStalled marks a download as stalled (notified)
func (d *Database) MarkDownloadStalled(id int64) error {
	_, err := d.db.Exec("UPDATE downloads SET stalled_notified = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

// UpdateDownloadStatus updates the status and error message of a download
func (d *Database) UpdateDownloadStatus(id int64, status string, errorMsg string) error {
	_, err := d.db.Exec(`
		UPDATE downloads SET
			status = ?, error = ?, last_error = ?,
			failed_at = CASE WHEN ? = 'failed' THEN CURRENT_TIMESTAMP ELSE failed_at END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, errorMsg, errorMsg, status, id)
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
	if err == sql.ErrNoRows {
		return nil, nil // No status record exists yet
	}
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

// GetUpgradeableMovies returns movies that are below their quality cutoff
func (d *Database) GetUpgradeableMovies(limit int) ([]UpgradeableItem, error) {
	query := `
		SELECT m.id, m.title, m.year, m.poster_path, m.size,
		       COALESCE(mqs.current_resolution, 'Unknown') || ' ' || COALESCE(mqs.current_source, '') as current_quality,
		       COALESCE(mqs.current_score, 0),
		       COALESCE(qp.cutoff_resolution, '1080p') || ' ' || COALESCE(qp.cutoff_source, 'bluray') as cutoff_quality,
		       COALESCE(mqs.cutoff_score, 100),
		       mqs.upgrade_searched_at
		FROM movies m
		LEFT JOIN media_quality_status mqs ON mqs.media_id = m.id AND mqs.media_type = 'movie'
		LEFT JOIN media_quality_override mqo ON mqo.media_id = m.id AND mqo.media_type = 'movie'
		LEFT JOIN quality_presets qp ON qp.id = COALESCE(mqo.preset_id, (SELECT id FROM quality_presets WHERE is_default = 1 AND media_type = 'movie' LIMIT 1))
		WHERE COALESCE(mqs.target_met, 0) = 0
		ORDER BY mqs.upgrade_searched_at ASC NULLS FIRST, m.added_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UpgradeableItem
	for rows.Next() {
		var item UpgradeableItem
		var lastSearched *time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.Year, &item.PosterPath, &item.Size,
			&item.CurrentQuality, &item.CurrentScore, &item.CutoffQuality, &item.CutoffScore,
			&lastSearched); err != nil {
			return nil, err
		}
		item.Type = "movie"
		if lastSearched != nil {
			ls := lastSearched.Format(time.RFC3339)
			item.LastSearched = &ls
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUpgradeableEpisodes returns episodes that are below their quality cutoff
func (d *Database) GetUpgradeableEpisodes(limit int) ([]UpgradeableItem, error) {
	query := `
		SELECT e.id, e.title, s.title as show_title, se.season_number, e.episode_number,
		       sh.poster_path, e.size,
		       COALESCE(mqs.current_resolution, 'Unknown') || ' ' || COALESCE(mqs.current_source, '') as current_quality,
		       COALESCE(mqs.current_score, 0),
		       COALESCE(qp.cutoff_resolution, '1080p') || ' ' || COALESCE(qp.cutoff_source, 'web') as cutoff_quality,
		       COALESCE(mqs.cutoff_score, 100),
		       mqs.upgrade_searched_at
		FROM episodes e
		JOIN seasons se ON se.id = e.season_id
		JOIN shows s ON s.id = se.show_id
		JOIN shows sh ON sh.id = se.show_id
		LEFT JOIN media_quality_status mqs ON mqs.media_id = e.id AND mqs.media_type = 'episode'
		LEFT JOIN media_quality_override mqo ON mqo.media_id = s.id AND mqo.media_type = 'show'
		LEFT JOIN quality_presets qp ON qp.id = COALESCE(mqo.preset_id, (SELECT id FROM quality_presets WHERE is_default = 1 AND media_type = 'tv' LIMIT 1))
		WHERE COALESCE(mqs.target_met, 0) = 0
		ORDER BY mqs.upgrade_searched_at ASC NULLS FIRST
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UpgradeableItem
	for rows.Next() {
		var item UpgradeableItem
		var lastSearched *time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.ShowTitle, &item.SeasonNumber, &item.EpisodeNumber,
			&item.PosterPath, &item.Size, &item.CurrentQuality, &item.CurrentScore,
			&item.CutoffQuality, &item.CutoffScore, &lastSearched); err != nil {
			return nil, err
		}
		item.Type = "episode"
		if lastSearched != nil {
			ls := lastSearched.Format(time.RFC3339)
			item.LastSearched = &ls
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUpgradesSummary returns a summary of all upgradeable media
func (d *Database) GetUpgradesSummary() (*UpgradesSummary, error) {
	movies, err := d.GetUpgradeableMovies(0)
	if err != nil {
		return nil, err
	}

	episodes, err := d.GetUpgradeableEpisodes(0)
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for _, m := range movies {
		totalSize += m.Size
	}
	for _, e := range episodes {
		totalSize += e.Size
	}

	return &UpgradesSummary{
		Movies:     movies,
		Episodes:   episodes,
		TotalCount: len(movies) + len(episodes),
		TotalSize:  totalSize,
	}, nil
}

// UpdateUpgradeSearched marks an item as having been searched for upgrades
func (d *Database) UpdateUpgradeSearched(mediaID int64, mediaType string, upgradeAvailable bool) error {
	_, err := d.db.Exec(`
		UPDATE media_quality_status
		SET upgrade_searched_at = CURRENT_TIMESTAMP,
		    upgrade_available = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE media_id = ? AND media_type = ?
	`, upgradeAvailable, mediaID, mediaType)
	return err
}

// UpdateQualityScores updates the current and cutoff scores for a media item
func (d *Database) UpdateQualityScores(mediaID int64, mediaType string, currentScore, cutoffScore int) error {
	_, err := d.db.Exec(`
		UPDATE media_quality_status
		SET current_score = ?,
		    cutoff_score = ?,
		    target_met = CASE WHEN ? >= ? THEN 1 ELSE 0 END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE media_id = ? AND media_type = ?
	`, currentScore, cutoffScore, currentScore, cutoffScore, mediaID, mediaType)
	return err
}

// CreateUpgradeWantedItem creates a wanted item for an upgrade
func (d *Database) CreateUpgradeWantedItem(mediaType string, tmdbID int64, imdbID, title string, year int, posterPath string, qualityProfileID, existingMediaID int64) error {
	_, err := d.db.Exec(`
		INSERT INTO wanted (type, tmdb_id, imdb_id, title, year, poster_path, quality_profile_id, is_upgrade, existing_media_id, monitored)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, 1)
		ON CONFLICT(type, tmdb_id) DO UPDATE SET
		    is_upgrade = 1,
		    existing_media_id = excluded.existing_media_id,
		    quality_profile_id = excluded.quality_profile_id
	`, mediaType, tmdbID, imdbID, title, year, posterPath, qualityProfileID, existingMediaID)
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

// UpdateGrabHistoryByTitle updates grab history status for a release by its title
func (d *Database) UpdateGrabHistoryByTitle(releaseTitle string, status string, errorMsg *string) error {
	if status == "imported" {
		_, err := d.db.Exec(`
			UPDATE grab_history SET status = ?, error_message = ?, imported_at = CURRENT_TIMESTAMP
			WHERE id = (
				SELECT id FROM grab_history
				WHERE release_title = ? AND status = 'grabbed'
				ORDER BY grabbed_at DESC LIMIT 1
			)
		`, status, errorMsg, releaseTitle)
		return err
	}
	_, err := d.db.Exec(`
		UPDATE grab_history SET status = ?, error_message = ?
		WHERE id = (
			SELECT id FROM grab_history
			WHERE release_title = ? AND status = 'grabbed'
			ORDER BY grabbed_at DESC LIMIT 1
		)
	`, status, errorMsg, releaseTitle)
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
		SELECT id, media_id, media_type, preset_id, monitored, COALESCE(monitored_seasons, ''),
		       COALESCE(preferred_audio_lang, ''), COALESCE(preferred_subtitle_lang, ''), created_at
		FROM media_quality_override
		WHERE media_id = ? AND media_type = ?`, mediaID, mediaType).Scan(
		&override.ID, &override.MediaID, &override.MediaType,
		&override.PresetID, &override.Monitored, &override.MonitoredSeasons,
		&override.PreferredAudioLang, &override.PreferredSubtitleLang, &override.CreatedAt)
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
		INSERT OR REPLACE INTO media_quality_override (media_id, media_type, preset_id, monitored, monitored_seasons, preferred_audio_lang, preferred_subtitle_lang)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		override.MediaID, override.MediaType, override.PresetID, override.Monitored, override.MonitoredSeasons,
		override.PreferredAudioLang, override.PreferredSubtitleLang)
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

// ==================== Scheduled Tasks ====================

// GetAllTasks returns all scheduled tasks
func (d *Database) GetAllTasks() ([]ScheduledTask, error) {
	rows, err := d.db.Query(`
		SELECT id, name, COALESCE(description, ''), task_type, enabled, interval_minutes,
		       last_run, next_run, last_duration_ms, COALESCE(last_status, ''), last_error,
		       run_count, fail_count
		FROM scheduled_tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		var t ScheduledTask
		var lastRun, nextRun sql.NullString
		var lastDurationMs sql.NullInt64
		err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.TaskType, &t.Enabled,
			&t.IntervalMinutes, &lastRun, &nextRun, &lastDurationMs,
			&t.LastStatus, &t.LastError, &t.RunCount, &t.FailCount)
		if err != nil {
			return nil, err
		}
		if lastRun.Valid {
			if parsed, err := time.Parse("2006-01-02 15:04:05", lastRun.String); err == nil {
				t.LastRun = &parsed
			}
		}
		if nextRun.Valid {
			if parsed, err := time.Parse("2006-01-02 15:04:05", nextRun.String); err == nil {
				t.NextRun = &parsed
			}
		}
		if lastDurationMs.Valid {
			t.LastDurationMs = &lastDurationMs.Int64
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetTask returns a single task by ID
func (d *Database) GetTask(id int64) (*ScheduledTask, error) {
	var t ScheduledTask
	var lastRun, nextRun sql.NullString
	var lastDurationMs sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, name, COALESCE(description, ''), task_type, enabled, interval_minutes,
		       last_run, next_run, last_duration_ms, COALESCE(last_status, ''), last_error,
		       run_count, fail_count
		FROM scheduled_tasks WHERE id = ?`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.TaskType, &t.Enabled,
		&t.IntervalMinutes, &lastRun, &nextRun, &lastDurationMs,
		&t.LastStatus, &t.LastError, &t.RunCount, &t.FailCount)
	if err != nil {
		return nil, err
	}
	if lastRun.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", lastRun.String); err == nil {
			t.LastRun = &parsed
		}
	}
	if nextRun.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", nextRun.String); err == nil {
			t.NextRun = &parsed
		}
	}
	if lastDurationMs.Valid {
		t.LastDurationMs = &lastDurationMs.Int64
	}
	return &t, nil
}

// GetTaskByName returns a task by name
func (d *Database) GetTaskByName(name string) (*ScheduledTask, error) {
	var t ScheduledTask
	var lastRun, nextRun sql.NullString
	var lastDurationMs sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, name, COALESCE(description, ''), task_type, enabled, interval_minutes,
		       last_run, next_run, last_duration_ms, COALESCE(last_status, ''), last_error,
		       run_count, fail_count
		FROM scheduled_tasks WHERE name = ?`, name).Scan(
		&t.ID, &t.Name, &t.Description, &t.TaskType, &t.Enabled,
		&t.IntervalMinutes, &lastRun, &nextRun, &lastDurationMs,
		&t.LastStatus, &t.LastError, &t.RunCount, &t.FailCount)
	if err != nil {
		return nil, err
	}
	if lastRun.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", lastRun.String); err == nil {
			t.LastRun = &parsed
		}
	}
	if nextRun.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", nextRun.String); err == nil {
			t.NextRun = &parsed
		}
	}
	if lastDurationMs.Valid {
		t.LastDurationMs = &lastDurationMs.Int64
	}
	return &t, nil
}

// UpsertTask creates or updates a task
func (d *Database) UpsertTask(task *ScheduledTask) error {
	result, err := d.db.Exec(`
		INSERT INTO scheduled_tasks (name, description, task_type, enabled, interval_minutes, next_run)
		VALUES (?, ?, ?, ?, ?, datetime('now', '+' || ? || ' minutes'))
		ON CONFLICT(name) DO UPDATE SET
			description = excluded.description,
			task_type = excluded.task_type
		WHERE scheduled_tasks.name = excluded.name`,
		task.Name, task.Description, task.TaskType, task.Enabled, task.IntervalMinutes, task.IntervalMinutes)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	if id > 0 {
		task.ID = id
	}
	return nil
}

// UpdateTask updates task settings
func (d *Database) UpdateTask(task *ScheduledTask) error {
	_, err := d.db.Exec(`
		UPDATE scheduled_tasks SET
			enabled = ?,
			interval_minutes = ?,
			next_run = CASE WHEN enabled = 1 THEN datetime('now', '+' || ? || ' minutes') ELSE next_run END
		WHERE id = ?`,
		task.Enabled, task.IntervalMinutes, task.IntervalMinutes, task.ID)
	return err
}

// UpdateTaskStats updates task run statistics
func (d *Database) UpdateTaskStats(taskID int64, status string, durationMs int64, errorMsg *string) error {
	failIncrement := 0
	if status == "failed" {
		failIncrement = 1
	}
	_, err := d.db.Exec(`
		UPDATE scheduled_tasks SET
			last_run = datetime('now'),
			next_run = datetime('now', '+' || interval_minutes || ' minutes'),
			last_duration_ms = ?,
			last_status = ?,
			last_error = ?,
			run_count = run_count + 1,
			fail_count = fail_count + ?
		WHERE id = ?`,
		durationMs, status, errorMsg, failIncrement, taskID)
	return err
}

// RecordTaskRun records a task execution in history
func (d *Database) RecordTaskRun(taskID int64, startedAt, finishedAt time.Time, status string, itemsProcessed, itemsFound int, errorMsg *string, details *string) error {
	durationMs := finishedAt.Sub(startedAt).Milliseconds()
	_, err := d.db.Exec(`
		INSERT INTO task_history (task_id, started_at, finished_at, duration_ms, status, items_processed, items_found, error, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, startedAt, finishedAt, durationMs, status, itemsProcessed, itemsFound, errorMsg, details)
	return err
}

// GetTaskHistory returns recent history for a specific task
func (d *Database) GetTaskHistory(taskID int64, limit int) ([]TaskHistory, error) {
	rows, err := d.db.Query(`
		SELECT h.id, h.task_id, t.name, h.started_at, h.finished_at, h.duration_ms,
		       h.status, h.items_processed, h.items_found, h.error, h.details
		FROM task_history h
		JOIN scheduled_tasks t ON h.task_id = t.id
		WHERE h.task_id = ?
		ORDER BY h.started_at DESC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []TaskHistory
	for rows.Next() {
		var h TaskHistory
		err := rows.Scan(&h.ID, &h.TaskID, &h.TaskName, &h.StartedAt, &h.FinishedAt,
			&h.DurationMs, &h.Status, &h.ItemsProcessed, &h.ItemsFound, &h.Error, &h.Details)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// GetAllTaskHistory returns recent history across all tasks
func (d *Database) GetAllTaskHistory(limit int) ([]TaskHistory, error) {
	rows, err := d.db.Query(`
		SELECT h.id, h.task_id, t.name, h.started_at, h.finished_at, h.duration_ms,
		       h.status, h.items_processed, h.items_found, h.error, h.details
		FROM task_history h
		JOIN scheduled_tasks t ON h.task_id = t.id
		ORDER BY h.started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []TaskHistory
	for rows.Next() {
		var h TaskHistory
		err := rows.Scan(&h.ID, &h.TaskID, &h.TaskName, &h.StartedAt, &h.FinishedAt,
			&h.DurationMs, &h.Status, &h.ItemsProcessed, &h.ItemsFound, &h.Error, &h.Details)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// CleanupTaskHistory removes old task history entries
func (d *Database) CleanupTaskHistory(daysToKeep int) error {
	_, err := d.db.Exec(`
		DELETE FROM task_history
		WHERE started_at < datetime('now', '-' || ? || ' days')`, daysToKeep)
	return err
}

// Notification methods

// CreateNotification creates a new notification for a user
func (d *Database) CreateNotification(userID int64, notifType, title, message string, imageURL, link *string) error {
	_, err := d.db.Exec(`
		INSERT INTO notifications (user_id, type, title, message, image_url, link)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, title, message, imageURL, link)
	return err
}

// GetNotifications returns notifications for a user
func (d *Database) GetNotifications(userID int64, unreadOnly bool, limit int) ([]Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, image_url, link, read, created_at
		FROM notifications
		WHERE user_id = ?`
	if unreadOnly {
		query += " AND read = 0"
	}
	query += " ORDER BY created_at DESC LIMIT ?"

	rows, err := d.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var readInt int
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message,
			&n.ImageURL, &n.Link, &readInt, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		n.Read = readInt == 1
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// GetUnreadNotificationCount returns the count of unread notifications for a user
func (d *Database) GetUnreadNotificationCount(userID int64) (int, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE user_id = ? AND read = 0`, userID).Scan(&count)
	return count, err
}

// MarkNotificationRead marks a single notification as read
func (d *Database) MarkNotificationRead(notificationID int64) error {
	_, err := d.db.Exec(`UPDATE notifications SET read = 1 WHERE id = ?`, notificationID)
	return err
}

// MarkAllNotificationsRead marks all notifications as read for a user
func (d *Database) MarkAllNotificationsRead(userID int64) error {
	_, err := d.db.Exec(`UPDATE notifications SET read = 1 WHERE user_id = ?`, userID)
	return err
}

// DeleteNotification deletes a notification
func (d *Database) DeleteNotification(notificationID int64) error {
	_, err := d.db.Exec(`DELETE FROM notifications WHERE id = ?`, notificationID)
	return err
}

// CleanupOldNotifications removes read notifications older than specified days
func (d *Database) CleanupOldNotifications(daysToKeep int) error {
	_, err := d.db.Exec(`
		DELETE FROM notifications
		WHERE read = 1 AND created_at < datetime('now', '-' || ? || ' days')`, daysToKeep)
	return err
}

// GetAdminUserIDs returns all admin user IDs for sending admin notifications
func (d *Database) GetAdminUserIDs() ([]int64, error) {
	rows, err := d.db.Query(`SELECT id FROM users WHERE role = 'admin'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ============================================================================
// Collection methods
// ============================================================================

// GetCollections returns all collections with item and owned counts
func (d *Database) GetCollections() ([]Collection, error) {
	rows, err := d.db.Query(`
		SELECT c.id, c.name, c.description, c.tmdb_collection_id, c.poster_path, c.backdrop_path,
			   c.is_auto, c.sort_order, c.created_at, c.updated_at,
			   COUNT(ci.id) as item_count,
			   SUM(CASE WHEN ci.media_id IS NOT NULL THEN 1 ELSE 0 END) as owned_count
		FROM collections c
		LEFT JOIN collection_items ci ON c.id = ci.collection_id
		GROUP BY c.id
		ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		var description, posterPath, backdropPath sql.NullString
		var tmdbID sql.NullInt64
		var isAuto int

		if err := rows.Scan(&c.ID, &c.Name, &description, &tmdbID, &posterPath, &backdropPath,
			&isAuto, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.ItemCount, &c.OwnedCount); err != nil {
			return nil, err
		}

		if description.Valid {
			c.Description = &description.String
		}
		if tmdbID.Valid {
			c.TmdbCollectionID = &tmdbID.Int64
		}
		if posterPath.Valid {
			c.PosterPath = &posterPath.String
		}
		if backdropPath.Valid {
			c.BackdropPath = &backdropPath.String
		}
		c.IsAuto = isAuto == 1

		collections = append(collections, c)
	}
	return collections, nil
}

// GetCollection returns a single collection by ID
func (d *Database) GetCollection(id int64) (*Collection, error) {
	var c Collection
	var description, posterPath, backdropPath sql.NullString
	var tmdbID sql.NullInt64
	var isAuto int

	err := d.db.QueryRow(`
		SELECT c.id, c.name, c.description, c.tmdb_collection_id, c.poster_path, c.backdrop_path,
			   c.is_auto, c.sort_order, c.created_at, c.updated_at,
			   COUNT(ci.id) as item_count,
			   SUM(CASE WHEN ci.media_id IS NOT NULL THEN 1 ELSE 0 END) as owned_count
		FROM collections c
		LEFT JOIN collection_items ci ON c.id = ci.collection_id
		WHERE c.id = ?
		GROUP BY c.id`, id).Scan(&c.ID, &c.Name, &description, &tmdbID, &posterPath, &backdropPath,
		&isAuto, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.ItemCount, &c.OwnedCount)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		c.Description = &description.String
	}
	if tmdbID.Valid {
		c.TmdbCollectionID = &tmdbID.Int64
	}
	if posterPath.Valid {
		c.PosterPath = &posterPath.String
	}
	if backdropPath.Valid {
		c.BackdropPath = &backdropPath.String
	}
	c.IsAuto = isAuto == 1

	return &c, nil
}

// GetCollectionByTmdbID returns a collection by its TMDB collection ID
func (d *Database) GetCollectionByTmdbID(tmdbCollectionID int64) (*Collection, error) {
	var c Collection
	var description, posterPath, backdropPath sql.NullString
	var tmdbID sql.NullInt64
	var isAuto int

	err := d.db.QueryRow(`
		SELECT id, name, description, tmdb_collection_id, poster_path, backdrop_path,
			   is_auto, sort_order, created_at, updated_at
		FROM collections WHERE tmdb_collection_id = ?`, tmdbCollectionID).Scan(
		&c.ID, &c.Name, &description, &tmdbID, &posterPath, &backdropPath,
		&isAuto, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		c.Description = &description.String
	}
	if tmdbID.Valid {
		c.TmdbCollectionID = &tmdbID.Int64
	}
	if posterPath.Valid {
		c.PosterPath = &posterPath.String
	}
	if backdropPath.Valid {
		c.BackdropPath = &backdropPath.String
	}
	c.IsAuto = isAuto == 1

	return &c, nil
}

// CreateCollection creates a new collection
func (d *Database) CreateCollection(c *Collection) error {
	isAuto := 0
	if c.IsAuto {
		isAuto = 1
	}
	if c.SortOrder == "" {
		c.SortOrder = "release"
	}

	result, err := d.db.Exec(`
		INSERT INTO collections (name, description, tmdb_collection_id, poster_path, backdrop_path, is_auto, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.Description, c.TmdbCollectionID, c.PosterPath, c.BackdropPath, isAuto, c.SortOrder)
	if err != nil {
		return err
	}

	c.ID, _ = result.LastInsertId()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateCollection updates an existing collection
func (d *Database) UpdateCollection(c *Collection) error {
	isAuto := 0
	if c.IsAuto {
		isAuto = 1
	}

	_, err := d.db.Exec(`
		UPDATE collections SET
			name = ?, description = ?, poster_path = ?, backdrop_path = ?,
			is_auto = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		c.Name, c.Description, c.PosterPath, c.BackdropPath, isAuto, c.SortOrder, c.ID)
	return err
}

// DeleteCollection deletes a collection and all its items (cascade)
func (d *Database) DeleteCollection(id int64) error {
	_, err := d.db.Exec(`DELETE FROM collections WHERE id = ?`, id)
	return err
}

// GetCollectionItems returns all items in a collection with library status
func (d *Database) GetCollectionItems(collectionID int64) ([]CollectionItem, error) {
	rows, err := d.db.Query(`
		SELECT ci.id, ci.collection_id, ci.media_type, ci.media_id, ci.tmdb_id,
			   ci.title, ci.year, ci.poster_path, ci.sort_order, ci.added_at,
			   CASE WHEN ci.media_id IS NOT NULL THEN 1 ELSE 0 END as in_library
		FROM collection_items ci
		WHERE ci.collection_id = ?
		ORDER BY ci.sort_order, ci.year, ci.title`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CollectionItem
	for rows.Next() {
		var item CollectionItem
		var mediaID sql.NullInt64
		var posterPath sql.NullString
		var year sql.NullInt64
		var inLibrary int

		if err := rows.Scan(&item.ID, &item.CollectionID, &item.MediaType, &mediaID, &item.TmdbID,
			&item.Title, &year, &posterPath, &item.SortOrder, &item.AddedAt, &inLibrary); err != nil {
			return nil, err
		}

		if mediaID.Valid {
			item.MediaID = &mediaID.Int64
		}
		if posterPath.Valid {
			item.PosterPath = &posterPath.String
		}
		if year.Valid {
			item.Year = int(year.Int64)
		}
		item.InLibrary = inLibrary == 1

		items = append(items, item)
	}
	return items, nil
}

// AddCollectionItem adds an item to a collection
func (d *Database) AddCollectionItem(item *CollectionItem) error {
	// Get the max sort_order for this collection
	var maxOrder sql.NullInt64
	d.db.QueryRow(`SELECT MAX(sort_order) FROM collection_items WHERE collection_id = ?`, item.CollectionID).Scan(&maxOrder)
	if maxOrder.Valid {
		item.SortOrder = int(maxOrder.Int64) + 1
	}

	result, err := d.db.Exec(`
		INSERT INTO collection_items (collection_id, media_type, media_id, tmdb_id, title, year, poster_path, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(collection_id, tmdb_id, media_type) DO UPDATE SET
			media_id = COALESCE(excluded.media_id, collection_items.media_id),
			title = excluded.title,
			poster_path = COALESCE(excluded.poster_path, collection_items.poster_path)`,
		item.CollectionID, item.MediaType, item.MediaID, item.TmdbID, item.Title, item.Year, item.PosterPath, item.SortOrder)
	if err != nil {
		return err
	}

	item.ID, _ = result.LastInsertId()
	item.AddedAt = time.Now()
	return nil
}

// RemoveCollectionItem removes an item from a collection
func (d *Database) RemoveCollectionItem(collectionID, tmdbID int64, mediaType string) error {
	_, err := d.db.Exec(`DELETE FROM collection_items WHERE collection_id = ? AND tmdb_id = ? AND media_type = ?`,
		collectionID, tmdbID, mediaType)
	return err
}

// UpdateCollectionItemOrder updates the sort order of items in a collection
func (d *Database) UpdateCollectionItemOrder(collectionID int64, itemIDs []int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, itemID := range itemIDs {
		_, err := tx.Exec(`UPDATE collection_items SET sort_order = ? WHERE id = ? AND collection_id = ?`,
			i, itemID, collectionID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetCollectionsForMedia returns all collections that contain a specific media item
func (d *Database) GetCollectionsForMedia(tmdbID int64, mediaType string) ([]Collection, error) {
	rows, err := d.db.Query(`
		SELECT c.id, c.name, c.description, c.tmdb_collection_id, c.poster_path, c.backdrop_path,
			   c.is_auto, c.sort_order, c.created_at, c.updated_at
		FROM collections c
		INNER JOIN collection_items ci ON c.id = ci.collection_id
		WHERE ci.tmdb_id = ? AND ci.media_type = ?
		ORDER BY c.name`, tmdbID, mediaType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		var description, posterPath, backdropPath sql.NullString
		var tmdbCollID sql.NullInt64
		var isAuto int

		if err := rows.Scan(&c.ID, &c.Name, &description, &tmdbCollID, &posterPath, &backdropPath,
			&isAuto, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}

		if description.Valid {
			c.Description = &description.String
		}
		if tmdbCollID.Valid {
			c.TmdbCollectionID = &tmdbCollID.Int64
		}
		if posterPath.Valid {
			c.PosterPath = &posterPath.String
		}
		if backdropPath.Valid {
			c.BackdropPath = &backdropPath.String
		}
		c.IsAuto = isAuto == 1

		collections = append(collections, c)
	}
	return collections, nil
}

// UpdateCollectionItemMediaID updates the media_id of a collection item when the item is added to the library
func (d *Database) UpdateCollectionItemMediaID(tmdbID int64, mediaType string, mediaID int64) error {
	_, err := d.db.Exec(`
		UPDATE collection_items SET media_id = ? WHERE tmdb_id = ? AND media_type = ?`,
		mediaID, tmdbID, mediaType)
	return err
}

// Storage Analytics Types

// LibrarySize represents storage usage per library
type LibrarySize struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Size  int64  `json:"size"`
	Count int    `json:"count"`
}

// QualitySize represents storage usage per quality level
type QualitySize struct {
	Quality string `json:"quality"`
	Size    int64  `json:"size"`
	Count   int    `json:"count"`
}

// YearSize represents storage usage per release year
type YearSize struct {
	Year  int   `json:"year"`
	Size  int64 `json:"size"`
	Count int   `json:"count"`
}

// LargestItem represents a large media file
type LargestItem struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Year    int    `json:"year"`
	Size    int64  `json:"size"`
	Quality string `json:"quality"`
	Path    string `json:"path"`
}

// DuplicateCopy represents a single copy of a duplicate item
type DuplicateCopy struct {
	ID      int64  `json:"id"`
	Quality string `json:"quality"`
	Size    int64  `json:"size"`
	Path    string `json:"path"`
}

// DuplicateItem represents an item with multiple copies
type DuplicateItem struct {
	TmdbID int64           `json:"tmdbId"`
	Title  string          `json:"title"`
	Year   int             `json:"year"`
	Type   string          `json:"type"`
	Copies []DuplicateCopy `json:"copies"`
}

// StorageAnalytics contains all storage analytics data
type StorageAnalytics struct {
	Total      int64           `json:"total"`
	Used       int64           `json:"used"`
	Free       int64           `json:"free"`
	ByLibrary  []LibrarySize   `json:"byLibrary"`
	ByQuality  []QualitySize   `json:"byQuality"`
	ByYear     []YearSize      `json:"byYear"`
	Largest    []LargestItem   `json:"largest"`
	Duplicates []DuplicateItem `json:"duplicates"`
}

// GetStorageByLibrary returns storage usage grouped by library
func (d *Database) GetStorageByLibrary() ([]LibrarySize, error) {
	// Get movie storage by library
	movieQuery := `
		SELECT l.id, l.name, l.type, COALESCE(SUM(m.size), 0) as size, COUNT(m.id) as count
		FROM libraries l
		LEFT JOIN movies m ON m.library_id = l.id
		WHERE l.type = 'movies'
		GROUP BY l.id`

	// Get TV/anime storage by library (sum episode sizes)
	tvQuery := `
		SELECT l.id, l.name, l.type, COALESCE(SUM(e.size), 0) as size, COUNT(DISTINCT s.id) as count
		FROM libraries l
		LEFT JOIN shows s ON s.library_id = l.id
		LEFT JOIN seasons sea ON sea.show_id = s.id
		LEFT JOIN episodes e ON e.season_id = sea.id
		WHERE l.type IN ('tv', 'anime')
		GROUP BY l.id`

	var results []LibrarySize

	// Execute movie query
	rows, err := d.db.Query(movieQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ls LibrarySize
		if err := rows.Scan(&ls.ID, &ls.Name, &ls.Type, &ls.Size, &ls.Count); err != nil {
			return nil, err
		}
		results = append(results, ls)
	}

	// Execute TV query
	rows, err = d.db.Query(tvQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ls LibrarySize
		if err := rows.Scan(&ls.ID, &ls.Name, &ls.Type, &ls.Size, &ls.Count); err != nil {
			return nil, err
		}
		results = append(results, ls)
	}

	return results, nil
}

// GetStorageByYear returns storage usage grouped by release year
func (d *Database) GetStorageByYear() ([]YearSize, error) {
	query := `
		SELECT year, SUM(size) as total_size, COUNT(*) as count
		FROM (
			SELECT year, size FROM movies WHERE size > 0
			UNION ALL
			SELECT s.year, e.size
			FROM episodes e
			JOIN seasons sea ON e.season_id = sea.id
			JOIN shows s ON sea.show_id = s.id
			WHERE e.size > 0
		)
		WHERE year > 0
		GROUP BY year
		ORDER BY year DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []YearSize
	for rows.Next() {
		var ys YearSize
		if err := rows.Scan(&ys.Year, &ys.Size, &ys.Count); err != nil {
			return nil, err
		}
		results = append(results, ys)
	}

	return results, nil
}

// GetLargestItems returns the largest media files
func (d *Database) GetLargestItems(limit int) ([]LargestItem, error) {
	query := `
		SELECT id, 'movie' as type, title, year, size, path
		FROM movies
		WHERE size > 0
		UNION ALL
		SELECT e.id, 'episode' as type,
			s.title || ' S' || printf('%02d', sea.season_number) || 'E' || printf('%02d', e.episode_number) as title,
			s.year, e.size, e.path
		FROM episodes e
		JOIN seasons sea ON e.season_id = sea.id
		JOIN shows s ON sea.show_id = s.id
		WHERE e.size > 0
		ORDER BY size DESC
		LIMIT ?`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []LargestItem
	for rows.Next() {
		var item LargestItem
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Year, &item.Size, &item.Path); err != nil {
			return nil, err
		}
		// Extract quality from path
		item.Quality = extractQualityFromPath(item.Path)
		results = append(results, item)
	}

	return results, nil
}

// GetMovieDuplicates returns movies with the same TMDB ID (multiple copies)
func (d *Database) GetMovieDuplicates() ([]DuplicateItem, error) {
	// Find tmdb_ids with multiple movies
	query := `
		SELECT tmdb_id, title, year, id, size, path
		FROM movies
		WHERE tmdb_id IN (
			SELECT tmdb_id FROM movies WHERE tmdb_id IS NOT NULL GROUP BY tmdb_id HAVING COUNT(*) > 1
		)
		ORDER BY tmdb_id, size DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	duplicateMap := make(map[int64]*DuplicateItem)
	var order []int64

	for rows.Next() {
		var tmdbID int64
		var title string
		var year int
		var id int64
		var size int64
		var path string

		if err := rows.Scan(&tmdbID, &title, &year, &id, &size, &path); err != nil {
			return nil, err
		}

		if _, exists := duplicateMap[tmdbID]; !exists {
			duplicateMap[tmdbID] = &DuplicateItem{
				TmdbID: tmdbID,
				Title:  title,
				Year:   year,
				Type:   "movie",
				Copies: []DuplicateCopy{},
			}
			order = append(order, tmdbID)
		}

		duplicateMap[tmdbID].Copies = append(duplicateMap[tmdbID].Copies, DuplicateCopy{
			ID:      id,
			Quality: extractQualityFromPath(path),
			Size:    size,
			Path:    path,
		})
	}

	var results []DuplicateItem
	for _, tmdbID := range order {
		results = append(results, *duplicateMap[tmdbID])
	}

	return results, nil
}

// GetEpisodeDuplicates returns episodes with multiple files for the same episode
func (d *Database) GetEpisodeDuplicates() ([]DuplicateItem, error) {
	// This is complex - episodes are unique by season_id + episode_number
	// For now, we won't track episode duplicates as they should be unique
	return []DuplicateItem{}, nil
}

// extractQualityFromPath extracts quality information from a file path
func extractQualityFromPath(path string) string {
	pathLower := strings.ToLower(path)

	// Check for resolution
	if strings.Contains(pathLower, "2160p") || strings.Contains(pathLower, "4k") || strings.Contains(pathLower, "uhd") {
		quality := "2160p"
		if strings.Contains(pathLower, "remux") {
			quality += " Remux"
		} else if strings.Contains(pathLower, "bluray") || strings.Contains(pathLower, "blu-ray") {
			quality += " BluRay"
		} else if strings.Contains(pathLower, "web") {
			quality += " WEB"
		}
		return quality
	}

	if strings.Contains(pathLower, "1080p") {
		quality := "1080p"
		if strings.Contains(pathLower, "remux") {
			quality += " Remux"
		} else if strings.Contains(pathLower, "bluray") || strings.Contains(pathLower, "blu-ray") {
			quality += " BluRay"
		} else if strings.Contains(pathLower, "web") {
			quality += " WEB"
		}
		return quality
	}

	if strings.Contains(pathLower, "720p") {
		quality := "720p"
		if strings.Contains(pathLower, "bluray") || strings.Contains(pathLower, "blu-ray") {
			quality += " BluRay"
		} else if strings.Contains(pathLower, "web") {
			quality += " WEB"
		}
		return quality
	}

	if strings.Contains(pathLower, "480p") || strings.Contains(pathLower, "dvd") {
		return "480p"
	}

	return "Unknown"
}

// GetStorageByQuality returns storage usage grouped by quality
func (d *Database) GetStorageByQuality() ([]QualitySize, error) {
	// Get all movies with size
	movieQuery := `SELECT path, size FROM movies WHERE size > 0`
	episodeQuery := `SELECT path, size FROM episodes WHERE size > 0`

	qualityMap := make(map[string]QualitySize)

	// Process movies
	rows, err := d.db.Query(movieQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var size int64
		if err := rows.Scan(&path, &size); err != nil {
			return nil, err
		}
		quality := extractQualityFromPath(path)
		// Normalize to base quality for grouping
		baseQuality := normalizeQuality(quality)
		qs := qualityMap[baseQuality]
		qs.Quality = baseQuality
		qs.Size += size
		qs.Count++
		qualityMap[baseQuality] = qs
	}

	// Process episodes
	rows, err = d.db.Query(episodeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var size int64
		if err := rows.Scan(&path, &size); err != nil {
			return nil, err
		}
		quality := extractQualityFromPath(path)
		baseQuality := normalizeQuality(quality)
		qs := qualityMap[baseQuality]
		qs.Quality = baseQuality
		qs.Size += size
		qs.Count++
		qualityMap[baseQuality] = qs
	}

	// Convert map to sorted slice
	qualityOrder := []string{"2160p", "1080p", "720p", "480p", "Unknown"}
	var results []QualitySize
	for _, q := range qualityOrder {
		if qs, exists := qualityMap[q]; exists {
			results = append(results, qs)
		}
	}

	return results, nil
}

// normalizeQuality extracts just the resolution from a quality string
func normalizeQuality(quality string) string {
	if strings.HasPrefix(quality, "2160p") {
		return "2160p"
	}
	if strings.HasPrefix(quality, "1080p") {
		return "1080p"
	}
	if strings.HasPrefix(quality, "720p") {
		return "720p"
	}
	if strings.HasPrefix(quality, "480p") {
		return "480p"
	}
	return "Unknown"
}

// Smart Playlist methods

// GetSmartPlaylists returns all smart playlists for a user (including system playlists)
func (d *Database) GetSmartPlaylists(userID *int64) ([]SmartPlaylist, error) {
	query := `
		SELECT id, user_id, name, description, rules, sort_by, sort_order, limit_count,
		       media_type, auto_refresh, is_system, last_refreshed, created_at
		FROM smart_playlists
		WHERE user_id IS NULL OR user_id = ?
		ORDER BY is_system DESC, name ASC
	`
	var uid int64
	if userID != nil {
		uid = *userID
	}
	rows, err := d.db.Query(query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []SmartPlaylist
	for rows.Next() {
		var p SmartPlaylist
		err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Rules, &p.SortBy,
			&p.SortOrder, &p.LimitCount, &p.MediaType, &p.AutoRefresh, &p.IsSystem,
			&p.LastRefreshed, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

// GetSmartPlaylist returns a single smart playlist by ID
func (d *Database) GetSmartPlaylist(id int64) (*SmartPlaylist, error) {
	query := `
		SELECT id, user_id, name, description, rules, sort_by, sort_order, limit_count,
		       media_type, auto_refresh, is_system, last_refreshed, created_at
		FROM smart_playlists WHERE id = ?
	`
	var p SmartPlaylist
	err := d.db.QueryRow(query, id).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Rules,
		&p.SortBy, &p.SortOrder, &p.LimitCount, &p.MediaType, &p.AutoRefresh, &p.IsSystem,
		&p.LastRefreshed, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateSmartPlaylist creates a new smart playlist
func (d *Database) CreateSmartPlaylist(p *SmartPlaylist) error {
	query := `
		INSERT INTO smart_playlists (user_id, name, description, rules, sort_by, sort_order,
		                             limit_count, media_type, auto_refresh, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query, p.UserID, p.Name, p.Description, p.Rules, p.SortBy,
		p.SortOrder, p.LimitCount, p.MediaType, p.AutoRefresh, p.IsSystem)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

// UpdateSmartPlaylist updates an existing smart playlist
func (d *Database) UpdateSmartPlaylist(p *SmartPlaylist) error {
	query := `
		UPDATE smart_playlists
		SET name = ?, description = ?, rules = ?, sort_by = ?, sort_order = ?,
		    limit_count = ?, media_type = ?, auto_refresh = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(query, p.Name, p.Description, p.Rules, p.SortBy, p.SortOrder,
		p.LimitCount, p.MediaType, p.AutoRefresh, p.ID)
	return err
}

// DeleteSmartPlaylist deletes a smart playlist
func (d *Database) DeleteSmartPlaylist(id int64) error {
	_, err := d.db.Exec("DELETE FROM smart_playlists WHERE id = ? AND is_system = 0", id)
	return err
}

// UpdateSmartPlaylistRefreshed updates the last_refreshed timestamp
func (d *Database) UpdateSmartPlaylistRefreshed(id int64) error {
	_, err := d.db.Exec("UPDATE smart_playlists SET last_refreshed = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

// GetSmartPlaylistItems returns media items matching the playlist rules
func (d *Database) GetSmartPlaylistItems(p *SmartPlaylist, profileID *int64) ([]SmartPlaylistItem, error) {
	var rules PlaylistRules
	if err := json.Unmarshal([]byte(p.Rules), &rules); err != nil {
		return nil, err
	}

	// Build queries for movies and/or shows
	var items []SmartPlaylistItem

	if p.MediaType == "movie" || p.MediaType == "both" {
		movieItems, err := d.querySmartPlaylistMovies(&rules, p.SortBy, p.SortOrder, p.LimitCount, profileID)
		if err != nil {
			return nil, err
		}
		items = append(items, movieItems...)
	}

	if p.MediaType == "show" || p.MediaType == "both" {
		showItems, err := d.querySmartPlaylistShows(&rules, p.SortBy, p.SortOrder, p.LimitCount, profileID)
		if err != nil {
			return nil, err
		}
		items = append(items, showItems...)
	}

	// Sort combined results if both types
	if p.MediaType == "both" && len(items) > 0 {
		sortSmartPlaylistItems(items, p.SortBy, p.SortOrder)
		if p.LimitCount != nil && len(items) > *p.LimitCount {
			items = items[:*p.LimitCount]
		}
	}

	return items, nil
}

func (d *Database) querySmartPlaylistMovies(rules *PlaylistRules, sortBy, sortOrder string, limit *int, profileID *int64) ([]SmartPlaylistItem, error) {
	whereClause, args := buildMovieWhereClause(rules, profileID)

	orderBy := getMovieOrderBy(sortBy, sortOrder)

	query := fmt.Sprintf(`
		SELECT m.id, 'movie' as media_type, m.title, m.year, m.poster_path, m.tmdb_rating, m.runtime, m.added_at
		FROM movies m
		%s
		%s
	`, whereClause, orderBy)

	if limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SmartPlaylistItem
	for rows.Next() {
		var item SmartPlaylistItem
		var addedAt sql.NullString
		err := rows.Scan(&item.ID, &item.MediaType, &item.Title, &item.Year, &item.PosterPath, &item.Rating, &item.Runtime, &addedAt)
		if err != nil {
			return nil, err
		}
		if addedAt.Valid {
			item.AddedAt = addedAt.String
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) querySmartPlaylistShows(rules *PlaylistRules, sortBy, sortOrder string, limit *int, profileID *int64) ([]SmartPlaylistItem, error) {
	whereClause, args := buildShowWhereClause(rules, profileID)

	orderBy := getShowOrderBy(sortBy, sortOrder)

	query := fmt.Sprintf(`
		SELECT s.id, 'show' as media_type, s.title, s.year, s.poster_path, s.tmdb_rating, 0 as runtime, s.added_at
		FROM shows s
		%s
		%s
	`, whereClause, orderBy)

	if limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SmartPlaylistItem
	for rows.Next() {
		var item SmartPlaylistItem
		var addedAt sql.NullString
		err := rows.Scan(&item.ID, &item.MediaType, &item.Title, &item.Year, &item.PosterPath, &item.Rating, &item.Runtime, &addedAt)
		if err != nil {
			return nil, err
		}
		if addedAt.Valid {
			item.AddedAt = addedAt.String
		}
		items = append(items, item)
	}
	return items, nil
}

func buildMovieWhereClause(rules *PlaylistRules, profileID *int64) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	for _, cond := range rules.Conditions {
		clause, condArgs := buildMovieCondition(cond, profileID)
		if clause != "" {
			conditions = append(conditions, clause)
			args = append(args, condArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	joiner := " AND "
	if rules.Match == "any" {
		joiner = " OR "
	}

	return "WHERE " + strings.Join(conditions, joiner), args
}

func buildShowWhereClause(rules *PlaylistRules, profileID *int64) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	for _, cond := range rules.Conditions {
		clause, condArgs := buildShowCondition(cond, profileID)
		if clause != "" {
			conditions = append(conditions, clause)
			args = append(args, condArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	joiner := " AND "
	if rules.Match == "any" {
		joiner = " OR "
	}

	return "WHERE " + strings.Join(conditions, joiner), args
}

func buildMovieCondition(cond PlaylistCondition, profileID *int64) (string, []interface{}) {
	var args []interface{}

	switch cond.Field {
	case "genre":
		val := fmt.Sprintf("%v", cond.Value)
		if cond.Operator == "contains" {
			return "m.genres LIKE ?", []interface{}{"%" + val + "%"}
		} else if cond.Operator == "not_contains" {
			return "(m.genres IS NULL OR m.genres NOT LIKE ?)", []interface{}{"%" + val + "%"}
		}
	case "year":
		val := toInt(cond.Value)
		switch cond.Operator {
		case "eq":
			return "m.year = ?", []interface{}{val}
		case "gte":
			return "m.year >= ?", []interface{}{val}
		case "lte":
			return "m.year <= ?", []interface{}{val}
		}
	case "rating":
		val := toFloat(cond.Value)
		switch cond.Operator {
		case "eq":
			return "m.tmdb_rating = ?", []interface{}{val}
		case "gte":
			return "m.tmdb_rating >= ?", []interface{}{val}
		case "lte":
			return "m.tmdb_rating <= ?", []interface{}{val}
		}
	case "runtime":
		val := toInt(cond.Value)
		switch cond.Operator {
		case "eq":
			return "m.runtime = ?", []interface{}{val}
		case "gte":
			return "m.runtime >= ?", []interface{}{val}
		case "lte":
			return "m.runtime <= ?", []interface{}{val}
		}
	case "resolution":
		val := fmt.Sprintf("%v", cond.Value)
		if cond.Operator == "eq" {
			return "m.quality LIKE ?", []interface{}{val + "%"}
		} else if cond.Operator == "contains" {
			return "m.quality LIKE ?", []interface{}{"%" + val + "%"}
		}
	case "codec":
		val := fmt.Sprintf("%v", cond.Value)
		return "m.quality LIKE ?", []interface{}{"%" + val + "%"}
	case "added":
		val := fmt.Sprintf("%v", cond.Value)
		days := parseDuration(val)
		return "m.added_at >= datetime('now', ?)", []interface{}{fmt.Sprintf("-%d days", days)}
	case "watched":
		if profileID == nil {
			return "", nil
		}
		val := toBool(cond.Value)
		if val {
			return "EXISTS (SELECT 1 FROM progress p WHERE p.media_type = 'movie' AND p.media_id = m.id AND p.profile_id = ? AND p.position > 0)", []interface{}{*profileID}
		}
		return "NOT EXISTS (SELECT 1 FROM progress p WHERE p.media_type = 'movie' AND p.media_id = m.id AND p.profile_id = ? AND p.position > 0)", []interface{}{*profileID}
	case "library":
		val := toInt(cond.Value)
		return "m.library_id = ?", []interface{}{val}
	case "actor":
		val := fmt.Sprintf("%v", cond.Value)
		return "m.cast_list LIKE ?", []interface{}{"%" + val + "%"}
	case "director":
		val := fmt.Sprintf("%v", cond.Value)
		return "m.director LIKE ?", []interface{}{"%" + val + "%"}
	case "studio":
		val := fmt.Sprintf("%v", cond.Value)
		return "m.studios LIKE ?", []interface{}{"%" + val + "%"}
	}

	return "", args
}

func buildShowCondition(cond PlaylistCondition, profileID *int64) (string, []interface{}) {
	var args []interface{}

	switch cond.Field {
	case "genre":
		val := fmt.Sprintf("%v", cond.Value)
		if cond.Operator == "contains" {
			return "s.genres LIKE ?", []interface{}{"%" + val + "%"}
		} else if cond.Operator == "not_contains" {
			return "(s.genres IS NULL OR s.genres NOT LIKE ?)", []interface{}{"%" + val + "%"}
		}
	case "year":
		val := toInt(cond.Value)
		switch cond.Operator {
		case "eq":
			return "s.year = ?", []interface{}{val}
		case "gte":
			return "s.year >= ?", []interface{}{val}
		case "lte":
			return "s.year <= ?", []interface{}{val}
		}
	case "rating":
		val := toFloat(cond.Value)
		switch cond.Operator {
		case "eq":
			return "s.tmdb_rating = ?", []interface{}{val}
		case "gte":
			return "s.tmdb_rating >= ?", []interface{}{val}
		case "lte":
			return "s.tmdb_rating <= ?", []interface{}{val}
		}
	case "added":
		val := fmt.Sprintf("%v", cond.Value)
		days := parseDuration(val)
		return "s.added_at >= datetime('now', ?)", []interface{}{fmt.Sprintf("-%d days", days)}
	case "library":
		val := toInt(cond.Value)
		return "s.library_id = ?", []interface{}{val}
	case "status":
		val := fmt.Sprintf("%v", cond.Value)
		return "s.status = ?", []interface{}{val}
	case "actor":
		val := fmt.Sprintf("%v", cond.Value)
		return "s.cast_list LIKE ?", []interface{}{"%" + val + "%"}
	case "studio":
		val := fmt.Sprintf("%v", cond.Value)
		return "s.network LIKE ?", []interface{}{"%" + val + "%"}
	}

	return "", args
}

func getMovieOrderBy(sortBy, sortOrder string) string {
	order := "DESC"
	if sortOrder == "asc" {
		order = "ASC"
	}

	switch sortBy {
	case "title":
		return fmt.Sprintf("ORDER BY m.title %s", order)
	case "year":
		return fmt.Sprintf("ORDER BY m.year %s", order)
	case "rating":
		return fmt.Sprintf("ORDER BY m.tmdb_rating %s", order)
	case "runtime":
		return fmt.Sprintf("ORDER BY m.runtime %s", order)
	case "added":
		return fmt.Sprintf("ORDER BY m.added_at %s", order)
	default:
		return fmt.Sprintf("ORDER BY m.added_at %s", order)
	}
}

func getShowOrderBy(sortBy, sortOrder string) string {
	order := "DESC"
	if sortOrder == "asc" {
		order = "ASC"
	}

	switch sortBy {
	case "title":
		return fmt.Sprintf("ORDER BY s.title %s", order)
	case "year":
		return fmt.Sprintf("ORDER BY s.year %s", order)
	case "rating":
		return fmt.Sprintf("ORDER BY s.tmdb_rating %s", order)
	case "added":
		return fmt.Sprintf("ORDER BY s.added_at %s", order)
	default:
		return fmt.Sprintf("ORDER BY s.added_at %s", order)
	}
}

func sortSmartPlaylistItems(items []SmartPlaylistItem, sortBy, sortOrder string) {
	sort.Slice(items, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "title":
			less = items[i].Title < items[j].Title
		case "year":
			less = items[i].Year < items[j].Year
		case "rating":
			less = items[i].Rating < items[j].Rating
		case "runtime":
			less = items[i].Runtime < items[j].Runtime
		default: // added
			less = items[i].AddedAt < items[j].AddedAt
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func parseDuration(val string) int {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "d") {
		days, _ := strconv.Atoi(strings.TrimSuffix(val, "d"))
		return days
	}
	if strings.HasSuffix(val, "y") {
		years, _ := strconv.Atoi(strings.TrimSuffix(val, "y"))
		return years * 365
	}
	// Default to days if just a number
	days, _ := strconv.Atoi(val)
	return days
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	case int:
		return val != 0
	case float64:
		return val != 0
	}
	return false
}

// CreateBuiltInSmartPlaylists creates the default system playlists
func (d *Database) CreateBuiltInSmartPlaylists() error {
	// Check if system playlists already exist
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM smart_playlists WHERE is_system = 1").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Already created
	}

	builtIn := []SmartPlaylist{
		{
			Name:        "Recently Added",
			Description: strPtr("Movies and shows added in the last 7 days"),
			Rules:       `{"match":"all","conditions":[{"field":"added","operator":"within","value":"7d"}]}`,
			SortBy:      "added",
			SortOrder:   "desc",
			MediaType:   "both",
			AutoRefresh: true,
			IsSystem:    true,
		},
		{
			Name:        "Unwatched Movies",
			Description: strPtr("Movies you haven't watched yet"),
			Rules:       `{"match":"all","conditions":[{"field":"watched","operator":"eq","value":false}]}`,
			SortBy:      "added",
			SortOrder:   "desc",
			MediaType:   "movie",
			AutoRefresh: true,
			IsSystem:    true,
		},
		{
			Name:        "4K Collection",
			Description: strPtr("Movies and shows available in 4K resolution"),
			Rules:       `{"match":"all","conditions":[{"field":"resolution","operator":"eq","value":"2160p"}]}`,
			SortBy:      "title",
			SortOrder:   "asc",
			MediaType:   "both",
			AutoRefresh: true,
			IsSystem:    true,
		},
		{
			Name:        "Top Rated",
			Description: strPtr("Highly rated content (8.0+ on TMDB)"),
			Rules:       `{"match":"all","conditions":[{"field":"rating","operator":"gte","value":8.0}]}`,
			SortBy:      "rating",
			SortOrder:   "desc",
			MediaType:   "both",
			AutoRefresh: true,
			IsSystem:    true,
		},
		{
			Name:        "Short Films",
			Description: strPtr("Movies under 90 minutes"),
			Rules:       `{"match":"all","conditions":[{"field":"runtime","operator":"lte","value":90}]}`,
			SortBy:      "runtime",
			SortOrder:   "asc",
			MediaType:   "movie",
			AutoRefresh: true,
			IsSystem:    true,
		},
	}

	for _, p := range builtIn {
		if err := d.CreateSmartPlaylist(&p); err != nil {
			return err
		}
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

// GetTraktConfig retrieves Trakt configuration for a user
func (d *Database) GetTraktConfig(userID int64) (*TraktConfig, error) {
	row := d.db.QueryRow(`
		SELECT id, user_id, access_token, refresh_token, expires_at, username,
		       sync_enabled, sync_watched, sync_ratings, sync_watchlist, last_synced_at, created_at
		FROM trakt_config WHERE user_id = ?`, userID)

	var config TraktConfig
	var expiresAt, lastSyncedAt, username sql.NullString
	err := row.Scan(
		&config.ID, &config.UserID, &config.AccessToken, &config.RefreshToken,
		&expiresAt, &username, &config.SyncEnabled, &config.SyncWatched,
		&config.SyncRatings, &config.SyncWatchlist, &lastSyncedAt, &config.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		config.ExpiresAt = &t
	}
	if lastSyncedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastSyncedAt.String)
		config.LastSyncedAt = &t
	}
	if username.Valid {
		config.Username = &username.String
	}

	return &config, nil
}

// SaveTraktConfig creates or updates Trakt configuration for a user
func (d *Database) SaveTraktConfig(config *TraktConfig) error {
	var expiresAt, lastSyncedAt interface{}
	if config.ExpiresAt != nil {
		expiresAt = config.ExpiresAt.Format(time.RFC3339)
	}
	if config.LastSyncedAt != nil {
		lastSyncedAt = config.LastSyncedAt.Format(time.RFC3339)
	}

	_, err := d.db.Exec(`
		INSERT INTO trakt_config (user_id, access_token, refresh_token, expires_at, username,
		                          sync_enabled, sync_watched, sync_ratings, sync_watchlist, last_synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			username = excluded.username,
			sync_enabled = excluded.sync_enabled,
			sync_watched = excluded.sync_watched,
			sync_ratings = excluded.sync_ratings,
			sync_watchlist = excluded.sync_watchlist,
			last_synced_at = excluded.last_synced_at`,
		config.UserID, config.AccessToken, config.RefreshToken, expiresAt, config.Username,
		config.SyncEnabled, config.SyncWatched, config.SyncRatings, config.SyncWatchlist, lastSyncedAt,
	)
	return err
}

// UpdateTraktSyncTime updates the last synced timestamp
func (d *Database) UpdateTraktSyncTime(userID int64) error {
	_, err := d.db.Exec(`UPDATE trakt_config SET last_synced_at = ? WHERE user_id = ?`,
		time.Now().Format(time.RFC3339), userID)
	return err
}

// DeleteTraktConfig removes Trakt configuration for a user
func (d *Database) DeleteTraktConfig(userID int64) error {
	_, err := d.db.Exec(`DELETE FROM trakt_config WHERE user_id = ?`, userID)
	return err
}

// AddWatchHistoryItem adds a watch history record
func (d *Database) AddWatchHistoryItem(item *WatchHistoryItem) error {
	_, err := d.db.Exec(`
		INSERT INTO watch_history (profile_id, media_type, media_id, tmdb_id, watched_at, synced_to_trakt)
		VALUES (?, ?, ?, ?, ?, ?)`,
		item.ProfileID, item.MediaType, item.MediaID, item.TmdbID, item.WatchedAt.Format(time.RFC3339), item.SyncedToTrakt,
	)
	return err
}

// GetUnsyncedWatchHistory gets watch history items not yet synced to Trakt for a profile
func (d *Database) GetUnsyncedWatchHistory(profileID int64) ([]WatchHistoryItem, error) {
	rows, err := d.db.Query(`
		SELECT id, profile_id, media_type, media_id, tmdb_id, watched_at, synced_to_trakt, created_at
		FROM watch_history
		WHERE profile_id = ? AND synced_to_trakt = 0
		ORDER BY watched_at ASC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchHistoryItem
	for rows.Next() {
		var item WatchHistoryItem
		var watchedAt string
		err := rows.Scan(&item.ID, &item.ProfileID, &item.MediaType, &item.MediaID,
			&item.TmdbID, &watchedAt, &item.SyncedToTrakt, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		item.WatchedAt, _ = time.Parse(time.RFC3339, watchedAt)
		items = append(items, item)
	}
	return items, nil
}

// MarkWatchHistorySynced marks watch history items as synced to Trakt
func (d *Database) MarkWatchHistorySynced(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("UPDATE watch_history SET synced_to_trakt = 1 WHERE id IN (%s)",
		strings.Join(placeholders, ","))
	_, err := d.db.Exec(query, args...)
	return err
}

// AddToTraktSyncQueue adds an item to the Trakt sync queue
func (d *Database) AddToTraktSyncQueue(item *TraktSyncQueueItem) error {
	_, err := d.db.Exec(`
		INSERT INTO trakt_sync_queue (user_id, action, media_type, tmdb_id, data)
		VALUES (?, ?, ?, ?, ?)`,
		item.UserID, item.Action, item.MediaType, item.TmdbID, item.Data,
	)
	return err
}

// GetPendingTraktSyncItems gets pending sync queue items for a user
func (d *Database) GetPendingTraktSyncItems(userID int64) ([]TraktSyncQueueItem, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, action, media_type, tmdb_id, data, status, error, created_at
		FROM trakt_sync_queue
		WHERE user_id = ? AND status = 'pending'
		ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TraktSyncQueueItem
	for rows.Next() {
		var item TraktSyncQueueItem
		var errorMsg sql.NullString
		err := rows.Scan(&item.ID, &item.UserID, &item.Action, &item.MediaType,
			&item.TmdbID, &item.Data, &item.Status, &errorMsg, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		if errorMsg.Valid {
			item.Error = &errorMsg.String
		}
		items = append(items, item)
	}
	return items, nil
}

// MarkTraktSyncComplete marks sync queue items as completed
func (d *Database) MarkTraktSyncComplete(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("UPDATE trakt_sync_queue SET status = 'completed' WHERE id IN (%s)",
		strings.Join(placeholders, ","))
	_, err := d.db.Exec(query, args...)
	return err
}

// MarkTraktSyncFailed marks a sync queue item as failed
func (d *Database) MarkTraktSyncFailed(id int64, errMsg string) error {
	_, err := d.db.Exec(`
		UPDATE trakt_sync_queue
		SET status = 'failed', error = ?
		WHERE id = ?`, errMsg, id)
	return err
}

// CleanupTraktSyncQueue removes old completed/failed items
func (d *Database) CleanupTraktSyncQueue() error {
	_, err := d.db.Exec(`
		DELETE FROM trakt_sync_queue
		WHERE (status = 'completed' AND created_at < datetime('now', '-7 days'))
		   OR (status = 'failed' AND created_at < datetime('now', '-1 day'))`)
	return err
}

// GetAllTraktConfigs gets all enabled Trakt configurations for sync
func (d *Database) GetAllTraktConfigs() ([]TraktConfig, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, access_token, refresh_token, expires_at, username,
		       sync_enabled, sync_watched, sync_ratings, sync_watchlist, last_synced_at, created_at
		FROM trakt_config WHERE sync_enabled = 1 AND access_token IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []TraktConfig
	for rows.Next() {
		var config TraktConfig
		var expiresAt, lastSyncedAt, username sql.NullString
		err := rows.Scan(
			&config.ID, &config.UserID, &config.AccessToken, &config.RefreshToken,
			&expiresAt, &username, &config.SyncEnabled, &config.SyncWatched,
			&config.SyncRatings, &config.SyncWatchlist, &lastSyncedAt, &config.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			config.ExpiresAt = &t
		}
		if lastSyncedAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastSyncedAt.String)
			config.LastSyncedAt = &t
		}
		if username.Valid {
			config.Username = &username.String
		}
		configs = append(configs, config)
	}
	return configs, nil
}
