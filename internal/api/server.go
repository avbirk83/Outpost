package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/outpost/outpost/internal/auth"
	"github.com/outpost/outpost/internal/config"
	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/download"
	"github.com/outpost/outpost/internal/health"
	"github.com/outpost/outpost/internal/logging"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
	"github.com/outpost/outpost/internal/metadata"
	"github.com/outpost/outpost/internal/prowlarr"
	"github.com/outpost/outpost/internal/parser"
	"github.com/outpost/outpost/internal/quality"
	"github.com/outpost/outpost/internal/scanner"
	"github.com/outpost/outpost/internal/storage"
	"github.com/outpost/outpost/internal/subtitles"
	"github.com/outpost/outpost/internal/tmdb"
	"github.com/outpost/outpost/internal/trakt"
)

type contextKey string

const userContextKey contextKey = "user"
const sessionContextKey contextKey = "session"

type Server struct {
	config        *config.Config
	db            *database.Database
	scanner       *scanner.Scanner
	metadata      *metadata.Service
	auth          *auth.Service
	downloads     *downloadclient.Manager
	indexers      *indexer.Manager
	scheduler     Scheduler
	acquisition   AcquisitionService
	notifications NotificationService
	healthChecker *health.Checker
	mux           *http.ServeMux
	subtitleCache map[string][]byte
	subtitleMu    sync.RWMutex
}

// Scheduler interface for task management
type Scheduler interface {
	GetStatus() []database.ScheduledTask
	TriggerTask(taskID int64) error
	UpdateTask(taskID int64, enabled bool, intervalMinutes int) error
	SearchWantedItem(tmdbID int64, mediaType string) error
	GetActiveSearch() string
	GetRunningTaskNames() []string
}

// AcquisitionService interface for download tracking
type AcquisitionService interface {
	GetActiveDownloads() ([]*download.TrackedDownload, error)
	GetTrackedDownload(id int64) (*download.TrackedDownload, error)
	DeleteTrackedDownload(id int64, deleteFromClient bool, deleteFiles bool) error
}

// NotificationService interface for in-app notifications
type NotificationService interface {
	Create(userID int64, notifType, title, message string, imageURL, link *string) error
	CreateForAdmins(notifType, title, message string, imageURL, link *string) error
	GetForUser(userID int64, unreadOnly bool, limit int) ([]database.Notification, error)
	GetUnreadCount(userID int64) (int, error)
	MarkRead(notificationID int64) error
	MarkAllRead(userID int64) error
	Delete(notificationID int64) error
	NotifyNewContent(userID int64, title, mediaType string, mediaID int64, posterPath *string) error
	NotifyRequestApproved(userID int64, title string, tmdbID int64, mediaType string, posterPath *string) error
	NotifyRequestDenied(userID int64, title string, reason string, posterPath *string) error
	NotifyDownloadComplete(title string, mediaType string, mediaID int64, posterPath *string) error
	NotifyDownloadFailed(title string, errorMsg string, posterPath *string) error
}

func NewServer(cfg *config.Config, db *database.Database, scan *scanner.Scanner, meta *metadata.Service, authSvc *auth.Service, downloads *downloadclient.Manager, indexers *indexer.Manager, sched Scheduler, acq AcquisitionService, notif NotificationService) *Server {
	s := &Server{
		config:        cfg,
		db:            db,
		scanner:       scan,
		metadata:      meta,
		auth:          authSvc,
		downloads:     downloads,
		indexers:      indexers,
		scheduler:     sched,
		acquisition:   acq,
		notifications: notif,
		healthChecker: health.NewChecker(db, downloads, indexers),
		mux:           http.NewServeMux(),
		subtitleCache: make(map[string][]byte),
	}
	s.setupRoutes()
	s.loadIndexers()
	return s
}

func (s *Server) loadIndexers() {
	indexers, err := s.db.GetEnabledIndexers()
	if err != nil {
		log.Printf("Error loading indexers from database: %v", err)
		return
	}
	log.Printf("Found %d enabled indexers in database", len(indexers))
	for _, idx := range indexers {
		config := &indexer.IndexerConfig{
			ID:         idx.ID,
			Name:       idx.Name,
			Type:       idx.Type,
			URL:        idx.URL,
			APIKey:     idx.APIKey,
			Categories: idx.Categories,
			Priority:   idx.Priority,
			Enabled:    idx.Enabled,
		}
		if err := s.indexers.AddIndexer(config); err != nil {
			log.Printf("Failed to add indexer %s: %v", idx.Name, err)
		} else {
			log.Printf("Added indexer: %s (type=%s, url=%s)", idx.Name, idx.Type, idx.URL)
		}
	}
}

// reloadIndexers clears and reloads all indexers from the database
func (s *Server) reloadIndexers() {
	s.indexers.Clear()
	s.loadIndexers()
	log.Printf("Reloaded %d indexers into manager", s.indexers.Count())
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Auth routes (public)
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/auth/me", s.handleMe)
	s.mux.HandleFunc("/api/auth/setup", s.handleSetup)
	s.mux.HandleFunc("/api/auth/verify-pin", s.requireAuth(s.handleVerifyPin))

	// Setup wizard routes (admin only after initial setup)
	s.mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("/api/setup/complete", s.requireAdmin(s.handleSetupComplete))

	// User management routes (admin only)
	s.mux.HandleFunc("/api/users", s.requireAdmin(s.handleUsers))
	s.mux.HandleFunc("/api/users/", s.requireAdmin(s.handleUser))

	// Profile routes (authenticated)
	s.mux.HandleFunc("/api/profiles", s.requireAuth(s.handleProfiles))
	s.mux.HandleFunc("/api/profiles/", s.requireAuth(s.handleProfile))

	// Library routes (admin only)
	s.mux.HandleFunc("/api/libraries", s.requireAdmin(s.handleLibraries))
	s.mux.HandleFunc("/api/libraries/", s.requireAdmin(s.handleLibrary))
	s.mux.HandleFunc("/api/scan/progress", s.requireAuth(s.handleScanProgress))

	// Media routes (authenticated)
	s.mux.HandleFunc("/api/movies", s.requireAuth(s.handleMovies))
	s.mux.HandleFunc("/api/movies/", s.requireAuth(s.handleMovie))
	s.mux.HandleFunc("/api/shows", s.requireAuth(s.handleShows))
	s.mux.HandleFunc("/api/shows/", s.requireAuth(s.handleShow))
	s.mux.HandleFunc("/api/episodes/", s.requireAdmin(s.handleEpisode))

	// Music routes (authenticated)
	s.mux.HandleFunc("/api/artists", s.requireAuth(s.handleArtists))
	s.mux.HandleFunc("/api/artists/", s.requireAuth(s.handleArtist))
	s.mux.HandleFunc("/api/albums", s.requireAuth(s.handleAlbums))
	s.mux.HandleFunc("/api/albums/", s.requireAuth(s.handleAlbum))
	s.mux.HandleFunc("/api/tracks/", s.requireAuth(s.handleTrack))

	// Book routes (authenticated)
	s.mux.HandleFunc("/api/books", s.requireAuth(s.handleBooks))
	s.mux.HandleFunc("/api/books/", s.requireAuth(s.handleBook))

	// Streaming routes (authenticated)
	s.mux.HandleFunc("/api/stream/", s.requireAuth(s.handleStream))
	s.mux.HandleFunc("/api/media-info/", s.requireAuth(s.handleMediaInfo))

	// Subtitle routes (authenticated)
	s.mux.HandleFunc("/api/subtitles/", s.requireAuth(s.handleSubtitles))

	// OpenSubtitles routes (admin for search/download)
	s.mux.HandleFunc("/api/opensubtitles/search", s.requireAdmin(s.handleOpenSubtitlesSearch))
	s.mux.HandleFunc("/api/opensubtitles/download", s.requireAdmin(s.handleOpenSubtitlesDownload))
	s.mux.HandleFunc("/api/opensubtitles/languages", s.requireAuth(s.handleOpenSubtitlesLanguages))
	s.mux.HandleFunc("/api/opensubtitles/test", s.requireAdmin(s.handleOpenSubtitlesTest))

	// Trakt routes (user-specific)
	s.mux.HandleFunc("/api/trakt/auth-url", s.requireAuth(s.handleTraktAuthURL))
	s.mux.HandleFunc("/api/trakt/callback", s.requireAuth(s.handleTraktCallback))
	s.mux.HandleFunc("/api/trakt/config", s.requireAuth(s.handleTraktConfig))
	s.mux.HandleFunc("/api/trakt/disconnect", s.requireAuth(s.handleTraktDisconnect))
	s.mux.HandleFunc("/api/trakt/sync", s.requireAuth(s.handleTraktSync))
	s.mux.HandleFunc("/api/trakt/test", s.requireAuth(s.handleTraktTest))

	// Progress routes (authenticated)
	s.mux.HandleFunc("/api/progress", s.requireAuth(s.handleProgress))
	s.mux.HandleFunc("/api/progress/", s.requireAuth(s.handleProgressGet))
	s.mux.HandleFunc("/api/continue-watching", s.requireAuth(s.handleContinueWatching))

	// Chapter routes (authenticated)
	s.mux.HandleFunc("/api/chapters/", s.requireAuth(s.handleChapters))

	// Skip segments routes (authenticated)
	s.mux.HandleFunc("/api/skip-segments/", s.requireAuth(s.handleSkipSegments))

	// Watch state routes (authenticated)
	s.mux.HandleFunc("/api/watched/", s.requireAuth(s.handleWatched))

	// Settings routes (admin only)
	s.mux.HandleFunc("/api/settings", s.requireAdmin(s.handleSettings))
	s.mux.HandleFunc("/api/settings/", s.requireAdmin(s.handleSetting))
	s.mux.HandleFunc("/api/settings/formats", s.requireAdmin(s.handleFormatSettings))

	// TMDB search routes (admin only)
	s.mux.HandleFunc("/api/tmdb/search/movie", s.requireAdmin(s.handleTmdbSearchMovie))
	s.mux.HandleFunc("/api/tmdb/search/tv", s.requireAdmin(s.handleTmdbSearchTV))

	// Person details route
	s.mux.HandleFunc("/api/person/", s.requireAuth(s.handlePerson))

	// Metadata refresh route (admin only)
	s.mux.HandleFunc("/api/metadata/refresh", s.requireAdmin(s.handleMetadataRefresh))
	s.mux.HandleFunc("/api/library/clear", s.requireAdmin(s.handleLibraryClear))

	// Download client routes (admin only)
	s.mux.HandleFunc("/api/download-clients", s.requireAdmin(s.handleDownloadClients))
	s.mux.HandleFunc("/api/download-clients/", s.requireAdmin(s.handleDownloadClient))
	s.mux.HandleFunc("/api/downloads", s.requireAdmin(s.handleDownloads))

	// Indexer routes (admin only)
	s.mux.HandleFunc("/api/indexers", s.requireAdmin(s.handleIndexers))
	s.mux.HandleFunc("/api/indexers/", s.requireAdmin(s.handleIndexer))
	s.mux.HandleFunc("/api/search", s.requireAdmin(s.handleSearch))
	s.mux.HandleFunc("/api/search/scored", s.requireAdmin(s.handleSearchScored))
	s.mux.HandleFunc("/api/grab", s.requireAdmin(s.handleGrab))

	// Prowlarr sync routes (admin only)
	s.mux.HandleFunc("/api/prowlarr/config", s.requireAdmin(s.handleProwlarrConfig))
	s.mux.HandleFunc("/api/prowlarr/test", s.requireAdmin(s.handleProwlarrTest))
	s.mux.HandleFunc("/api/prowlarr/sync", s.requireAdmin(s.handleProwlarrSync))
	s.mux.HandleFunc("/api/indexer-tags", s.requireAdmin(s.handleIndexerTags))

	// Quality profile routes (GET is auth only, modifications are admin only)
	s.mux.HandleFunc("/api/quality-profiles", s.requireAuth(s.handleQualityProfiles))
	s.mux.HandleFunc("/api/quality-profiles/", s.requireAdmin(s.handleQualityProfile))
	s.mux.HandleFunc("/api/custom-formats", s.requireAdmin(s.handleCustomFormats))
	s.mux.HandleFunc("/api/custom-formats/", s.requireAdmin(s.handleCustomFormat))
	s.mux.HandleFunc("/api/releases/parse", s.requireAdmin(s.handleParseRelease))

	// Quality preset routes (GET is auth only, modifications are admin only)
	s.mux.HandleFunc("/api/quality/presets", s.requireAuth(s.handleQualityPresets))
	s.mux.HandleFunc("/api/quality/presets/", s.requireAdmin(s.handleQualityPreset))

	// Collection routes (GET is auth only, modifications are admin only)
	s.mux.HandleFunc("/api/collections", s.requireAuth(s.handleCollections))
	s.mux.HandleFunc("/api/collections/", s.requireAuth(s.handleCollection))

	// Smart playlist routes (authenticated)
	s.mux.HandleFunc("/api/smart-playlists", s.requireAuth(s.handleSmartPlaylists))
	s.mux.HandleFunc("/api/smart-playlists/preview", s.requireAuth(s.handleSmartPlaylistPreview))
	s.mux.HandleFunc("/api/smart-playlists/", s.requireAuth(s.handleSmartPlaylist))

	// Upgrade search routes (admin only)
	s.mux.HandleFunc("/api/upgrades", s.requireAdmin(s.handleUpgrades))
	s.mux.HandleFunc("/api/upgrades/search", s.requireAdmin(s.handleUpgradeSearch))
	s.mux.HandleFunc("/api/upgrades/search-all", s.requireAdmin(s.handleUpgradeSearchAll))

	// Download tracking routes (admin only)
	s.mux.HandleFunc("/api/download-items", s.requireAdmin(s.handleDownloadItems))
	s.mux.HandleFunc("/api/download-items/", s.requireAdmin(s.handleDownloadItem))

	// Import and naming routes (admin only)
	s.mux.HandleFunc("/api/imports/history", s.requireAdmin(s.handleImportHistory))
	s.mux.HandleFunc("/api/settings/naming", s.requireAdmin(s.handleNamingTemplates))
	s.mux.HandleFunc("/api/storage/status", s.requireAdmin(s.handleStorageStatus))
	s.mux.HandleFunc("/api/storage/analytics", s.requireAdmin(s.handleStorageAnalytics))

	// Wanted/Monitoring routes (admin only)
	s.mux.HandleFunc("/api/wanted", s.requireAdmin(s.handleWantedItems))
	s.mux.HandleFunc("/api/wanted/", s.requireAdmin(s.handleWantedItem))
	s.mux.HandleFunc("/api/wanted/search/", s.requireAdmin(s.handleWantedSearch))

	// Public route for login background (no auth required)
	s.mux.HandleFunc("/api/public/trending-posters", s.handlePublicTrendingPosters)

	// Discover routes (authenticated)
	s.mux.HandleFunc("/api/discover/movies/trending", s.requireAuth(s.handleDiscoverTrendingMovies))
	s.mux.HandleFunc("/api/discover/movies/popular", s.requireAuth(s.handleDiscoverPopularMovies))
	s.mux.HandleFunc("/api/discover/movies/upcoming", s.requireAuth(s.handleDiscoverUpcomingMovies))
	s.mux.HandleFunc("/api/discover/movies/theatrical", s.requireAuth(s.handleDiscoverTheatricalReleases))
	s.mux.HandleFunc("/api/discover/movies/top-rated", s.requireAuth(s.handleDiscoverTopRatedMovies))
	s.mux.HandleFunc("/api/discover/movies/genre/", s.requireAuth(s.handleDiscoverMoviesByGenre))
	s.mux.HandleFunc("/api/discover/shows/trending", s.requireAuth(s.handleDiscoverTrendingTV))
	s.mux.HandleFunc("/api/discover/shows/popular", s.requireAuth(s.handleDiscoverPopularTV))
	s.mux.HandleFunc("/api/discover/shows/top-rated", s.requireAuth(s.handleDiscoverTopRatedTV))
	s.mux.HandleFunc("/api/discover/shows/upcoming", s.requireAuth(s.handleDiscoverUpcomingTV))
	s.mux.HandleFunc("/api/discover/shows/genre/", s.requireAuth(s.handleDiscoverTVByGenre))
	s.mux.HandleFunc("/api/discover/genres/movie", s.requireAuth(s.handleMovieGenres))
	s.mux.HandleFunc("/api/discover/genres/tv", s.requireAuth(s.handleTVGenres))
	s.mux.HandleFunc("/api/discover/movie/", s.requireAuth(s.handleDiscoverMovieDetail))
	s.mux.HandleFunc("/api/discover/show/", s.requireAuth(s.handleDiscoverShowDetail))
	s.mux.HandleFunc("/api/trailers/movie/", s.requireAuth(s.handleMovieTrailers))
	s.mux.HandleFunc("/api/trailers/tv/", s.requireAuth(s.handleTVTrailers))
	s.mux.HandleFunc("/api/movie/recommendations/", s.requireAuth(s.handleMovieRecommendations))
	s.mux.HandleFunc("/api/movies/suggestions/", s.requireAuth(s.handleMovieSuggestions))
	s.mux.HandleFunc("/api/shows/suggestions/", s.requireAuth(s.handleShowSuggestions))

	// Calendar route
	s.mux.HandleFunc("/api/calendar", s.requireAuth(s.handleCalendar))

	// Notification routes
	s.mux.HandleFunc("/api/notifications", s.requireAuth(s.handleNotifications))
	s.mux.HandleFunc("/api/notifications/unread-count", s.requireAuth(s.handleNotificationUnreadCount))
	s.mux.HandleFunc("/api/notifications/read-all", s.requireAuth(s.handleNotificationReadAll))
	s.mux.HandleFunc("/api/notifications/", s.requireAuth(s.handleNotification))

	// Request routes
	s.mux.HandleFunc("/api/requests", s.requireAuth(s.handleRequests))
	s.mux.HandleFunc("/api/requests/clear-denied", s.requireAdmin(s.handleClearDeniedRequests))
	s.mux.HandleFunc("/api/requests/", s.requireAuth(s.handleRequest))

	// Watchlist routes
	s.mux.HandleFunc("/api/watchlist", s.requireAuth(s.handleWatchlist))
	s.mux.HandleFunc("/api/watchlist/", s.requireAuth(s.handleWatchlistItem))

	// Blocklist routes (admin only)
	s.mux.HandleFunc("/api/blocklist", s.requireAdmin(s.handleBlocklist))
	s.mux.HandleFunc("/api/blocklist/", s.requireAdmin(s.handleBlocklistItem))

	// Grab history routes (admin only)
	s.mux.HandleFunc("/api/grab-history", s.requireAdmin(s.handleGrabHistory))

	// Blocked groups routes (admin only)
	s.mux.HandleFunc("/api/blocked-groups", s.requireAdmin(s.handleBlockedGroups))
	s.mux.HandleFunc("/api/blocked-groups/", s.requireAdmin(s.handleBlockedGroup))

	// Release filters routes (admin only)
	s.mux.HandleFunc("/api/release-filters", s.requireAdmin(s.handleReleaseFilters))
	s.mux.HandleFunc("/api/release-filters/", s.requireAdmin(s.handleReleaseFilter))

	// Delay profiles routes (admin only)
	s.mux.HandleFunc("/api/delay-profiles", s.requireAdmin(s.handleDelayProfiles))
	s.mux.HandleFunc("/api/delay-profiles/", s.requireAdmin(s.handleDelayProfile))

	// Exclusions routes (admin only)
	s.mux.HandleFunc("/api/exclusions", s.requireAdmin(s.handleExclusions))
	s.mux.HandleFunc("/api/exclusions/", s.requireAdmin(s.handleExclusion))

	// Movie quality status routes (admin only)
	s.mux.HandleFunc("/api/movies/quality/", s.requireAdmin(s.handleMovieQuality))

	// Show quality status routes (admin only)
	s.mux.HandleFunc("/api/shows/quality/", s.requireAdmin(s.handleShowQuality))

	// Task/Scheduler routes (admin only)
	s.mux.HandleFunc("/api/tasks", s.requireAdmin(s.handleTasks))
	s.mux.HandleFunc("/api/tasks/history", s.requireAdmin(s.handleTaskHistory))
	s.mux.HandleFunc("/api/tasks/", s.requireAdmin(s.handleTask))

	// System status route (authenticated)
	s.mux.HandleFunc("/api/system/status", s.requireAuth(s.handleSystemStatus))

	// Logs routes (admin only)
	s.mux.HandleFunc("/api/logs", s.requireAdmin(s.handleLogs))
	s.mux.HandleFunc("/api/logs/download", s.requireAdmin(s.handleLogsDownload))

	// Health check routes (admin only)
	s.mux.HandleFunc("/api/health/full", s.requireAdmin(s.handleHealthFull))
	s.mux.HandleFunc("/api/health/check/", s.requireAdmin(s.handleHealthCheck))

	// Backup/Restore routes (admin only)
	s.mux.HandleFunc("/api/backup", s.requireAdmin(s.handleBackup))
	s.mux.HandleFunc("/api/backup/restore", s.requireAdmin(s.handleRestore))

	// Filesystem browse route (admin only)
	s.mux.HandleFunc("/api/filesystem/browse", s.requireAdmin(s.handleFilesystemBrowse))

	// Image cache (public for posters)
	s.mux.HandleFunc("/images/", s.handleImages)

	// Static file serving for frontend (catch-all)
	s.mux.HandleFunc("/", s.handleStatic)
}

// Middleware

func (s *Server) getSessionToken(r *http.Request) string {
	// Check Authorization header first
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}
	// Check cookie
	if cookie, err := r.Cookie("session"); err == nil {
		return cookie.Value
	}
	return ""
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := s.getSessionToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.auth.ValidateSession(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get session to access active profile
		session, _ := s.db.GetSessionByToken(token)

		// Add user and session to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		if session != nil {
			ctx = context.WithValue(ctx, sessionContextKey, session)
		}
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := s.getSessionToken(r)
		if token == "" {
			log.Printf("Auth failed: no token for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.auth.ValidateSession(token)
		if err != nil {
			log.Printf("Auth failed: invalid token for %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user.Role != "admin" {
			log.Printf("Auth failed: not admin for %s %s (user: %s)", r.Method, r.URL.Path, user.Username)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) getCurrentUser(r *http.Request) *database.User {
	if user, ok := r.Context().Value(userContextKey).(*database.User); ok {
		return user
	}
	return nil
}

func (s *Server) getActiveProfileID(r *http.Request) *int64 {
	if session, ok := r.Context().Value(sessionContextKey).(*database.Session); ok {
		return session.ActiveProfileID
	}
	return nil
}

func (s *Server) Start() error {
	// Wrap mux with static file fallback for SPA
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try the mux first
		// Create a response recorder to check if mux handled it
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/images/") {
			s.mux.ServeHTTP(w, r)
			return
		}
		// For all other paths, serve static files
		s.handleStatic(w, r)
	})
	return http.ListenAndServe(":"+s.config.Port, handler)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Library handlers

func (s *Server) handleLibraries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		libraries, err := s.db.GetLibraries()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if libraries == nil {
			libraries = []database.Library{}
		}
		json.NewEncoder(w).Encode(libraries)

	case http.MethodPost:
		var lib database.Library
		if err := json.NewDecoder(r.Body).Decode(&lib); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if lib.ScanInterval == 0 {
			lib.ScanInterval = 3600
		}
		if err := s.db.CreateLibrary(&lib); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(lib)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/libraries/{id} or /api/libraries/{id}/scan
	path := strings.TrimPrefix(r.URL.Path, "/api/libraries/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Library ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid library ID", http.StatusBadRequest)
		return
	}

	// Handle scan endpoint
	if len(parts) == 2 && parts[1] == "scan" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleScan(w, r, id)
		return
	}

	// Handle single library
	switch r.Method {
	case http.MethodGet:
		lib, err := s.db.GetLibrary(id)
		if err != nil {
			http.Error(w, "Library not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(lib)

	case http.MethodDelete:
		if err := s.db.DeleteLibrary(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request, libraryID int64) {
	lib, err := s.db.GetLibrary(libraryID)
	if err != nil {
		http.Error(w, "Library not found", http.StatusNotFound)
		return
	}

	// Run scan in goroutine so we don't block the response
	go func() {
		if err := s.scanner.ScanLibrary(lib); err != nil {
			// Log error (can't send to client since response already sent)
			println("Scan error:", err.Error())
		}
	}()

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "scanning",
		"message": "Library scan started",
	})
}

func (s *Server) handleScanProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	progress := s.scanner.GetProgress()
	json.NewEncoder(w).Encode(progress)
}

// Content filtering for parental controls

// isContentAllowed checks if content is allowed for a user based on their content rating limit
func (s *Server) isContentAllowed(user *database.User, contentRating *string, r *http.Request) bool {
	// No user or no limit means all content allowed
	if user == nil || user.ContentRatingLimit == nil {
		return true
	}

	userLimit := database.ContentRatingLevel(*user.ContentRatingLimit)

	// If user requires PIN and is elevated, allow all content
	if user.RequirePin {
		elevationToken := s.getElevationToken(r)
		if elevationToken != "" {
			elevation, err := s.db.GetPinElevationByToken(elevationToken)
			if err == nil && elevation.UserID == user.ID {
				return true
			}
		}
	}

	// Unknown/unrated content is restricted for users with limits
	if contentRating == nil || *contentRating == "" {
		return false
	}

	// Normalize the content rating
	normalizedRating := database.NormalizeContentRating(*contentRating, "")
	contentLevel := database.ContentRatingLevel(normalizedRating)

	// If content level is 0 (unknown), it's restricted
	if contentLevel == 0 {
		return false
	}

	return contentLevel <= userLimit
}

// isKidFriendly is a legacy function for basic kid filtering (kept for backwards compatibility)
func (s *Server) isKidFriendly(contentRating *string) bool {
	if contentRating == nil || *contentRating == "" {
		return false
	}
	normalizedRating := database.NormalizeContentRating(*contentRating, "")
	level := database.ContentRatingLevel(normalizedRating)
	// G and PG are kid-friendly (levels 1 and 2)
	return level > 0 && level <= 2
}

// Media handlers

// MovieWithWatchState extends Movie with watch state
type MovieWithWatchState struct {
	database.Movie
	WatchState string  `json:"watchState,omitempty"`
	Progress   float64 `json:"progress,omitempty"`
}

func (s *Server) handleMovies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	movies, err := s.db.GetMovies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if movies == nil {
		movies = []database.Movie{}
	}

	// Filter based on user's content rating limit
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil {
		var filtered []database.Movie
		for _, m := range movies {
			if s.isContentAllowed(user, m.ContentRating, r) {
				filtered = append(filtered, m)
			}
		}
		movies = filtered
		if movies == nil {
			movies = []database.Movie{}
		}
	}

	// Get watch states
	watchStates, _ := s.db.GetAllMovieWatchStates()

	// Build response with watch states
	result := make([]MovieWithWatchState, len(movies))
	for i, m := range movies {
		result[i] = MovieWithWatchState{Movie: m}
		if state, ok := watchStates[m.ID]; ok {
			result[i].WatchState = state.WatchState
			result[i].Progress = state.Progress
		}
	}

	json.NewEncoder(w).Encode(result)
}

// ShowWithWatchState extends Show with watch state and episode progress
type ShowWithWatchState struct {
	database.Show
	WatchState      string `json:"watchState,omitempty"`
	WatchedEpisodes int    `json:"watchedEpisodes,omitempty"`
	TotalEpisodes   int    `json:"totalEpisodes,omitempty"`
}

func (s *Server) handleShows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shows, err := s.db.GetShows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if shows == nil {
		shows = []database.Show{}
	}

	// Filter based on user's content rating limit
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil {
		var filtered []database.Show
		for _, sh := range shows {
			if s.isContentAllowed(user, sh.ContentRating, r) {
				filtered = append(filtered, sh)
			}
		}
		shows = filtered
		if shows == nil {
			shows = []database.Show{}
		}
	}

	// Get watch states
	watchStates, _ := s.db.GetAllShowWatchStates()

	// Build response with watch states
	result := make([]ShowWithWatchState, len(shows))
	for i, sh := range shows {
		result[i] = ShowWithWatchState{Show: sh}
		if state, ok := watchStates[sh.ID]; ok {
			result[i].WatchState = state.WatchState
			result[i].WatchedEpisodes = state.WatchedEpisodes
			result[i].TotalEpisodes = state.TotalEpisodes
		}
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/shows/{id} or /api/shows/{id}/refresh or /api/shows/{id}/match
	path := strings.TrimPrefix(r.URL.Path, "/api/shows/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Show ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	show, err := s.db.GetShow(id)
	if err != nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}

	// Check content rating restriction
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil && !s.isContentAllowed(user, show.ContentRating, r) {
		// Content is restricted - check if PIN is required
		if user.RequirePin {
			http.Error(w, "Content restricted - PIN required", http.StatusForbidden)
		} else {
			http.Error(w, "Content not available", http.StatusForbidden)
		}
		return
	}

	// Handle refresh endpoint
	if len(parts) == 2 && parts[1] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchShowMetadata(show); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			show, _ = s.db.GetShow(id)
		}
		s.sendShowDetail(w, show)
		return
	}

	// Handle match endpoint
	if len(parts) == 2 && parts[1] == "match" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TmdbID int64 `json:"tmdbId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchShowMetadataByTmdbID(show, req.TmdbID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			show, _ = s.db.GetShow(id)
		}
		s.sendShowDetail(w, show)
		return
	}

	// Handle missing episodes endpoint
	if len(parts) == 2 && parts[1] == "missing" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleMissingEpisodes(w, r, show)
		return
	}

	// Handle request-missing endpoint
	if len(parts) == 2 && parts[1] == "request-missing" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleRequestMissingEpisodes(w, r, show)
		return
	}

	// Default: GET show
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.sendShowDetail(w, show)
}

func (s *Server) sendShowDetail(w http.ResponseWriter, show *database.Show) {
	seasons, err := s.db.GetSeasonsByShow(show.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type SeasonWithEpisodes struct {
		database.Season
		Episodes []database.Episode `json:"episodes"`
	}

	type ShowDetail struct {
		database.Show
		Seasons []SeasonWithEpisodes `json:"seasons"`
	}

	detail := ShowDetail{Show: *show}
	for _, season := range seasons {
		episodes, _ := s.db.GetEpisodesBySeason(season.ID)
		if episodes == nil {
			episodes = []database.Episode{}
		}
		detail.Seasons = append(detail.Seasons, SeasonWithEpisodes{
			Season:   season,
			Episodes: episodes,
		})
	}

	json.NewEncoder(w).Encode(detail)
}

func (s *Server) handleEpisode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/episodes/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/episodes/")
	if path == "" {
		http.Error(w, "Episode ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid episode ID", http.StatusBadRequest)
		return
	}

	episode, err := s.db.GetEpisode(id)
	if err != nil {
		http.Error(w, "Episode not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get showId for the episode
		showID, err := s.db.GetShowIDForEpisode(id)
		if err != nil {
			log.Printf("Failed to get show ID for episode %d: %v", id, err)
		}
		response := struct {
			*database.Episode
			ShowID int64 `json:"showId"`
		}{
			Episode: episode,
			ShowID:  showID,
		}
		json.NewEncoder(w).Encode(response)
	case http.MethodDelete:
		// Delete the file if it exists
		if episode.Path != "" {
			if err := os.Remove(episode.Path); err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete episode file: %v", err)
			}
		}
		// Delete from database
		if err := s.db.DeleteEpisode(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Don't serve API routes here
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	staticDir := s.config.StaticDir
	// Get path from URL (use URL.Path which includes leading /)
	urlPath := r.URL.Path
	if urlPath == "" {
		urlPath = "/"
	}

	path := filepath.Join(staticDir, urlPath)
	log.Printf("Static request: URL=%s, StaticDir=%s, Path=%s", r.URL.Path, staticDir, path)

	// Check if file exists and is not a directory
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		// SPA fallback: serve index.html for all non-file routes
		path = filepath.Join(staticDir, "index.html")
		log.Printf("Falling back to index.html: %s", path)
	}

	http.ServeFile(w, r, path)
}

// Single movie handler
func (s *Server) handleMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/movies/{id} or /api/movies/{id}/refresh or /api/movies/{id}/match
	path := strings.TrimPrefix(r.URL.Path, "/api/movies/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Movie ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := s.db.GetMovie(id)
	if err != nil {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	// Check content rating restriction
	user := s.getCurrentUser(r)
	if user != nil && user.ContentRatingLimit != nil && !s.isContentAllowed(user, movie.ContentRating, r) {
		// Content is restricted - check if PIN is required
		if user.RequirePin {
			http.Error(w, "Content restricted - PIN required", http.StatusForbidden)
		} else {
			http.Error(w, "Content not available", http.StatusForbidden)
		}
		return
	}

	// Handle refresh endpoint
	if len(parts) == 2 && parts[1] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchMovieMetadata(movie); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Reload movie to get updated data
			movie, _ = s.db.GetMovie(id)
		}
		json.NewEncoder(w).Encode(movie)
		return
	}

	// Handle match endpoint
	if len(parts) == 2 && parts[1] == "match" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TmdbID int64 `json:"tmdbId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.metadata != nil {
			if err := s.metadata.FetchMovieMetadataByTmdbID(movie, req.TmdbID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			movie, _ = s.db.GetMovie(id)
		}
		json.NewEncoder(w).Encode(movie)
		return
	}

	// Handle DELETE
	if r.Method == http.MethodDelete {
		// Delete the file if it exists
		if movie.Path != "" {
			if err := os.Remove(movie.Path); err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete movie file: %v", err)
			}
		}
		// Delete from database
		if err := s.db.DeleteMovie(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Default: GET movie
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(movie)
}

// Streaming handler with transcoding support
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/stream/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid stream path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var filePath string

	switch mediaType {
	case "movie":
		movie, err := s.db.GetMovie(id)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		filePath = movie.Path
	case "episode":
		episode, err := s.db.GetEpisode(id)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		filePath = episode.Path
	case "track":
		track, err := s.db.GetTrack(id)
		if err != nil {
			http.Error(w, "Track not found", http.StatusNotFound)
			return
		}
		filePath = track.Path
		// Serve audio files directly
		s.serveFileDirectly(w, r, filePath)
		return
	case "book":
		book, err := s.db.GetBook(id)
		if err != nil {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		filePath = book.Path
		// Serve books directly
		s.serveFileDirectly(w, r, filePath)
		return
	default:
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	// Check file exists
	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Check if file is browser-compatible (direct play)
	ext := strings.ToLower(filepath.Ext(filePath))
	canDirectPlay := ext == ".mp4" || ext == ".webm" || ext == ".m4v"

	// Direct play for compatible files (browser handles seeking via Range requests)
	if canDirectPlay {
		s.serveFileDirectly(w, r, filePath)
		return
	}

	// Transcode for non-compatible files (MKV, AVI, etc.)
	s.serveTranscodedVideo(w, r, filePath)
}

// serveFileDirectly serves a file without transcoding
func (s *Server) serveFileDirectly(w http.ResponseWriter, r *http.Request, filePath string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Cannot open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".mp4":
		contentType = "video/mp4"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".webm":
		contentType = "video/webm"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".flac":
		contentType = "audio/flac"
	case ".m4a", ".aac":
		contentType = "audio/mp4"
	case ".ogg":
		contentType = "audio/ogg"
	case ".pdf":
		contentType = "application/pdf"
	case ".epub":
		contentType = "application/epub+zip"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, filepath.Base(filePath), fileInfo.ModTime(), file)
}

// handleMediaInfo returns media information including duration
// VideoStream represents a video stream in a media file
type VideoStream struct {
	Index       int    `json:"index"`
	Codec       string `json:"codec"`
	Profile     string `json:"profile,omitempty"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	AspectRatio string `json:"aspectRatio,omitempty"`
	FrameRate   string `json:"frameRate,omitempty"`
	BitRate     int64  `json:"bitRate,omitempty"`
	PixelFormat string `json:"pixelFormat,omitempty"`
	Default     bool   `json:"default"`
}

// AudioStream represents an audio stream in a media file
type AudioStream struct {
	Index        int    `json:"index"`
	Codec        string `json:"codec"`
	Channels     int    `json:"channels"`
	ChannelLayout string `json:"channelLayout,omitempty"`
	SampleRate   int    `json:"sampleRate,omitempty"`
	BitRate      int64  `json:"bitRate,omitempty"`
	Language     string `json:"language,omitempty"`
	Title        string `json:"title,omitempty"`
	Default      bool   `json:"default"`
}

func (s *Server) handleMediaInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/media-info/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/media-info/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var filePath string

	switch mediaType {
	case "movie":
		movie, err := s.db.GetMovie(id)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		filePath = movie.Path
	case "episode":
		episode, err := s.db.GetEpisode(id)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		filePath = episode.Path
	default:
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	// Get full media info using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to get media info", http.StatusInternalServerError)
		return
	}

	// Parse ffprobe output
	var probeResult struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			Size       string `json:"size"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			Index          int               `json:"index"`
			CodecType      string            `json:"codec_type"`
			CodecName      string            `json:"codec_name"`
			Profile        string            `json:"profile"`
			Width          int               `json:"width"`
			Height         int               `json:"height"`
			DisplayAspect  string            `json:"display_aspect_ratio"`
			PixelFormat    string            `json:"pix_fmt"`
			FrameRate      string            `json:"r_frame_rate"`
			AvgFrameRate   string            `json:"avg_frame_rate"`
			BitRate        string            `json:"bit_rate"`
			Channels       int               `json:"channels"`
			ChannelLayout  string            `json:"channel_layout"`
			SampleRate     string            `json:"sample_rate"`
			Tags           map[string]string `json:"tags"`
			Disposition    struct {
				Default int `json:"default"`
				Forced  int `json:"forced"`
			} `json:"disposition"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		http.Error(w, "Failed to parse media info", http.StatusInternalServerError)
		return
	}

	// Parse duration
	duration, _ := strconv.ParseFloat(probeResult.Format.Duration, 64)
	fileSize, _ := strconv.ParseInt(probeResult.Format.Size, 10, 64)
	containerBitRate, _ := strconv.ParseInt(probeResult.Format.BitRate, 10, 64)

	// Build response
	videoStreams := []VideoStream{}
	audioStreams := []AudioStream{}
	subtitleTracks := []SubtitleTrack{}
	subtitleIndex := 0

	for _, stream := range probeResult.Streams {
		switch stream.CodecType {
		case "video":
			bitRate, _ := strconv.ParseInt(stream.BitRate, 10, 64)
			// Calculate frame rate from fraction if available
			frameRate := stream.AvgFrameRate
			if frameRate == "" || frameRate == "0/0" {
				frameRate = stream.FrameRate
			}
			// Simplify frame rate (e.g., "24000/1001" -> "23.976")
			if parts := strings.Split(frameRate, "/"); len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den > 0 {
					frameRate = fmt.Sprintf("%.3f", num/den)
				}
			}
			videoStreams = append(videoStreams, VideoStream{
				Index:       stream.Index,
				Codec:       stream.CodecName,
				Profile:     stream.Profile,
				Width:       stream.Width,
				Height:      stream.Height,
				AspectRatio: stream.DisplayAspect,
				FrameRate:   frameRate,
				BitRate:     bitRate,
				PixelFormat: stream.PixelFormat,
				Default:     stream.Disposition.Default == 1,
			})
		case "audio":
			bitRate, _ := strconv.ParseInt(stream.BitRate, 10, 64)
			sampleRate, _ := strconv.Atoi(stream.SampleRate)
			audioStreams = append(audioStreams, AudioStream{
				Index:         stream.Index,
				Codec:         stream.CodecName,
				Channels:      stream.Channels,
				ChannelLayout: stream.ChannelLayout,
				SampleRate:    sampleRate,
				BitRate:       bitRate,
				Language:      stream.Tags["language"],
				Title:         stream.Tags["title"],
				Default:       stream.Disposition.Default == 1,
			})
		case "subtitle":
			subtitleTracks = append(subtitleTracks, SubtitleTrack{
				Index:    subtitleIndex,
				Language: stream.Tags["language"],
				Title:    stream.Tags["title"],
				Codec:    stream.CodecName,
				Default:  stream.Disposition.Default == 1,
				Forced:   stream.Disposition.Forced == 1,
				External: false,
			})
			subtitleIndex++
		}
	}

	// Also get external subtitles
	externalTracks := s.findExternalSubtitles(filePath, subtitleIndex)
	subtitleTracks = append(subtitleTracks, externalTracks...)

	// Get container format (e.g., "matroska,webm" -> "MKV")
	container := probeResult.Format.FormatName
	containerMap := map[string]string{
		"matroska,webm": "MKV",
		"mov,mp4,m4a,3gp,3g2,mj2": "MP4",
		"avi": "AVI",
		"mpegts": "TS",
		"webm": "WEBM",
	}
	if mapped, ok := containerMap[container]; ok {
		container = mapped
	} else {
		container = strings.ToUpper(strings.Split(container, ",")[0])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"duration":       duration,
		"fileSize":       fileSize,
		"bitRate":        containerBitRate,
		"container":      container,
		"videoStreams":   videoStreams,
		"audioStreams":   audioStreams,
		"subtitleTracks": subtitleTracks,
	})
}

// serveTranscodedVideo transcodes video on-the-fly using FFmpeg
func (s *Server) serveTranscodedVideo(w http.ResponseWriter, r *http.Request, filePath string) {
	// Check for seek position (in seconds)
	startTime := r.URL.Query().Get("t")

	// Build FFmpeg arguments
	args := []string{}

	// Add seek position before input for fast initial seek
	if startTime != "" {
		args = append(args, "-ss", startTime)
	}

	args = append(args,
		"-i", filePath,
		"-c:v", "libx264",        // Re-encode video to ensure proper sync after seek
		"-preset", "ultrafast",   // Fast encoding
		"-crf", "23",             // Quality level
		"-c:a", "aac",            // Transcode audio to AAC
		"-b:a", "192k",           // Audio bitrate
		"-ac", "2",               // Stereo audio
		"-movflags", "frag_keyframe+empty_moov+faststart",
		"-f", "mp4",              // Output format
		"-",                      // Output to stdout
	)

	cmd := exec.Command("ffmpeg", args...)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to create pipe", http.StatusInternalServerError)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		http.Error(w, "Failed to start transcoding", http.StatusInternalServerError)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")

	// Stream the output
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				cmd.Process.Kill()
				break
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
}

// Progress handlers
func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get active profile ID
	profileID := s.getActiveProfileID(r)
	if profileID == nil {
		http.Error(w, "No profile selected", http.StatusBadRequest)
		return
	}

	var p database.Progress
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Override profile ID from session for security
	p.ProfileID = *profileID

	if err := s.db.SaveProgress(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func (s *Server) handleProgressGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get active profile ID
	profileID := s.getActiveProfileID(r)
	if profileID == nil {
		// No profile selected, return empty progress
		json.NewEncoder(w).Encode(database.Progress{
			Position: 0,
			Duration: 0,
		})
		return
	}

	// Parse path: /api/progress/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid progress path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	progress, err := s.db.GetProgress(*profileID, mediaType, id)
	if err != nil {
		// No progress yet, return zeros
		json.NewEncoder(w).Encode(database.Progress{
			ProfileID: *profileID,
			MediaType: mediaType,
			MediaID:   id,
			Position:  0,
			Duration:  0,
		})
		return
	}

	json.NewEncoder(w).Encode(progress)
}

func (s *Server) handleChapters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/chapters/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/chapters/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid chapters path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	chapters, err := s.db.GetChapters(mediaType, id)
	if err != nil {
		// No chapters, return empty array
		json.NewEncoder(w).Encode([]database.Chapter{})
		return
	}

	json.NewEncoder(w).Encode(chapters)
}

func (s *Server) handleSkipSegments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/skip-segments/{showId} or /api/skip-segments/{showId}/{type}
	path := strings.TrimPrefix(r.URL.Path, "/api/skip-segments/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid skip-segments path", http.StatusBadRequest)
		return
	}

	showID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		segments, err := s.db.GetSkipSegments(showID)
		if err != nil {
			http.Error(w, "Failed to get skip segments", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(segments)

	case http.MethodPost:
		var req struct {
			Type      string  `json:"type"`
			StartTime float64 `json:"startTime"`
			EndTime   float64 `json:"endTime"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Type != "intro" && req.Type != "credits" {
			http.Error(w, "Type must be 'intro' or 'credits'", http.StatusBadRequest)
			return
		}

		if req.StartTime >= req.EndTime {
			http.Error(w, "Start time must be less than end time", http.StatusBadRequest)
			return
		}

		if err := s.db.SaveSkipSegment(showID, req.Type, req.StartTime, req.EndTime); err != nil {
			http.Error(w, "Failed to save skip segment", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		if len(parts) < 2 {
			http.Error(w, "Segment type required for delete", http.StatusBadRequest)
			return
		}
		segmentType := parts[1]
		if segmentType != "intro" && segmentType != "credits" {
			http.Error(w, "Type must be 'intro' or 'credits'", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteSkipSegment(showID, segmentType); err != nil {
			http.Error(w, "Failed to delete skip segment", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleContinueWatching(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodDelete {
		// Remove item from continue watching
		path := strings.TrimPrefix(r.URL.Path, "/api/continue-watching/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		mediaType := parts[0]
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		if err := s.db.DeleteProgress(mediaType, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := s.db.GetContinueWatching(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if items == nil {
		items = []database.ContinueWatchingItem{}
	}

	json.NewEncoder(w).Encode(items)
}

// handleWatched handles marking items as watched or unwatched
// POST /api/watched/{type}/{id} - mark as watched
// DELETE /api/watched/{type}/{id} - mark as unwatched
// GET /api/watched/{type}/{id} - get watch status
func (s *Server) handleWatched(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/watched/{type}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/watched/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid path - expected /api/watched/{type}/{id}", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	if mediaType != "movie" && mediaType != "episode" {
		http.Error(w, "Invalid media type - must be 'movie' or 'episode'", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Mark as watched - we need to know the duration
		var req struct {
			Duration float64 `json:"duration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// If no body, use a default duration
			req.Duration = 3600 // Default 1 hour
		}
		if req.Duration <= 0 {
			req.Duration = 3600
		}

		if err := s.db.MarkAsWatched(mediaType, id, req.Duration); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "watched", "mediaType": mediaType, "mediaId": id})

	case http.MethodDelete:
		// Mark as unwatched
		if err := s.db.MarkAsUnwatched(mediaType, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "unwatched", "mediaType": mediaType, "mediaId": id})

	case http.MethodGet:
		// Get watch status
		watched, progress, err := s.db.GetWatchedStatus(mediaType, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"watched":  watched,
			"progress": progress,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Settings handlers
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		settings, err := s.db.GetAllSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if settings == nil {
			settings = make(map[string]string)
		}
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var data map[string]string
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for key, value := range data {
			if err := s.db.SetSetting(key, value); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Update metadata service if TMDB API key changed
			if key == "tmdb_api_key" && s.metadata != nil {
				s.metadata.UpdateAPIKey(value)
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFormatSettings manages acceptable file format settings
func (s *Server) handleFormatSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		settings, err := s.db.GetFormatSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(settings)

	case http.MethodPut, http.MethodPost:
		var settings database.FormatSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.db.SaveFormatSettings(&settings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(settings)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMetadataRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	// Get all movies and refresh their metadata
	movies, err := s.db.GetMovies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	refreshed := 0
	errors := 0
	for _, movie := range movies {
		if err := s.metadata.FetchMovieMetadata(&movie); err != nil {
			log.Printf("Failed to refresh metadata for movie %s: %v", movie.Title, err)
			errors++
		} else {
			refreshed++
		}
	}

	// Get all shows and refresh their metadata
	shows, err := s.db.GetShows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, show := range shows {
		if err := s.metadata.FetchShowMetadata(&show); err != nil {
			log.Printf("Failed to refresh metadata for show %s: %v", show.Title, err)
			errors++
		} else {
			refreshed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"refreshed": refreshed,
		"errors":    errors,
		"total":     len(movies) + len(shows),
	})
}

func (s *Server) handleLibraryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.db.ClearAllLibraryData(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "All library data cleared",
	})
}

func (s *Server) handleSetting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	key := strings.TrimPrefix(r.URL.Path, "/api/settings/")
	if key == "" {
		http.Error(w, "Setting key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		value, err := s.db.GetSetting(key)
		if err != nil {
			http.Error(w, "Setting not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"key": key, "value": value})

	case http.MethodPut:
		var data struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.db.SetSetting(key, data.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Image cache handler - serves cached TMDB images
func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve from data/images directory
	imagePath := strings.TrimPrefix(r.URL.Path, "/images/")
	fullPath := filepath.Join(filepath.Dir(s.config.DBPath), "images", imagePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// TMDB search handlers
func (s *Server) handleTmdbSearchMovie(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' required", http.StatusBadRequest)
		return
	}

	year := 0
	if y := r.URL.Query().Get("year"); y != "" {
		year, _ = strconv.Atoi(y)
	}

	if s.metadata == nil {
		http.Error(w, "Metadata service not configured", http.StatusServiceUnavailable)
		return
	}

	results, err := s.metadata.SearchMovies(query, year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleTmdbSearchTV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' required", http.StatusBadRequest)
		return
	}

	year := 0
	if y := r.URL.Query().Get("year"); y != "" {
		year, _ = strconv.Atoi(y)
	}

	if s.metadata == nil {
		http.Error(w, "Metadata service not configured", http.StatusServiceUnavailable)
		return
	}

	results, err := s.metadata.SearchTV(query, year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

// Auth handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, user, err := s.auth.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.ExpiresAt,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": session.Token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := s.getSessionToken(r)
	if token != "" {
		s.auth.Logout(token)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := s.getSessionToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.auth.ValidateSession(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has PIN elevation
	isElevated := false
	elevationToken := s.getElevationToken(r)
	if elevationToken != "" {
		elevation, err := s.db.GetPinElevationByToken(elevationToken)
		if err == nil && elevation.UserID == user.ID {
			isElevated = true
		}
	}

	response := map[string]interface{}{
		"id":                 user.ID,
		"username":           user.Username,
		"role":               user.Role,
		"contentRatingLimit": user.ContentRatingLimit,
		"requirePin":         user.RequirePin,
		"isElevated":         isElevated,
		"hasPin":             user.PinHash != nil && *user.PinHash != "",
	}

	json.NewEncoder(w).Encode(response)
}

// getElevationToken extracts the elevation token from request headers or cookies
func (s *Server) getElevationToken(r *http.Request) string {
	// Check header first
	token := r.Header.Get("X-Elevation-Token")
	if token != "" {
		return token
	}

	// Check cookie
	cookie, err := r.Cookie("elevation_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

func (s *Server) handleVerifyPin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Pin == "" || len(req.Pin) != 4 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "PIN must be 4 digits",
		})
		return
	}

	// Check if user has a PIN set
	if user.PinHash == nil || *user.PinHash == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "No PIN set for this user",
		})
		return
	}

	// Verify PIN
	if !auth.CheckPassword(req.Pin, *user.PinHash) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": "Incorrect PIN",
		})
		return
	}

	// Create elevation token (valid for 1 hour)
	elevationToken, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	elevation := &database.PinElevation{
		UserID:    user.ID,
		Token:     elevationToken,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.db.CreatePinElevation(elevation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set elevation token as cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "elevation_token",
		Value:    elevationToken,
		Path:     "/",
		Expires:  elevation.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"token": elevationToken,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if any users exist
	count, err := s.db.CountUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		// Return setup status
		json.NewEncoder(w).Encode(map[string]interface{}{
			"setupRequired": count == 0,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only allow setup if no users exist
	if count > 0 {
		http.Error(w, "Setup already completed", http.StatusForbidden)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	// Create admin user
	user, err := s.auth.CreateUser(req.Username, req.Password, "admin")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Setup wizard handlers

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check setup_completed setting first
	setupCompleted, _ := s.db.GetSetting("setup_completed")
	if setupCompleted == "true" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"needsSetup":     false,
			"setupCompleted": true,
		})
		return
	}

	// Check each step
	steps := map[string]bool{
		"adminCreated":             false,
		"libraryAdded":             false,
		"downloadClientConfigured": false,
		"indexerConfigured":        false,
		"qualityProfileSet":        false,
	}

	// 1. Check if admin exists
	userCount, _ := s.db.CountUsers()
	steps["adminCreated"] = userCount > 0

	// 2. Check if any library exists
	libraries, _ := s.db.GetLibraries()
	steps["libraryAdded"] = len(libraries) > 0

	// 3. Check if download client is configured
	downloadClients, _ := s.db.GetDownloadClients()
	for _, dc := range downloadClients {
		if dc.Enabled {
			steps["downloadClientConfigured"] = true
			break
		}
	}

	// 4. Check if indexers are configured (direct or via Prowlarr)
	enabledIndexers, _ := s.db.GetEnabledIndexers()
	prowlarrConfig, _ := s.db.GetProwlarrConfig()
	steps["indexerConfigured"] = len(enabledIndexers) > 0 || (prowlarrConfig != nil && prowlarrConfig.URL != "")

	// 5. Check if quality presets exist
	presets, _ := s.db.GetQualityPresets()
	for _, p := range presets {
		if p.Enabled {
			steps["qualityProfileSet"] = true
			break
		}
	}

	// Determine if setup is needed (admin must exist, but other steps can be skipped)
	needsSetup := !steps["adminCreated"] || !steps["libraryAdded"]

	// All steps except admin can be skipped
	canSkip := steps["adminCreated"]

	json.NewEncoder(w).Encode(map[string]interface{}{
		"needsSetup":     needsSetup,
		"setupCompleted": false,
		"steps":          steps,
		"canSkip":        canSkip,
	})
}

func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Set setup_completed flag
	if err := s.db.SetSetting("setup_completed", "true"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// User management handlers

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		users, err := s.db.GetUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if users == nil {
			users = []database.User{}
		}
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		var req struct {
			Username           string  `json:"username"`
			Password           string  `json:"password"`
			Role               string  `json:"role"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
			RequirePin         bool    `json:"requirePin"`
			Pin                string  `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "Username and password required", http.StatusBadRequest)
			return
		}

		if req.Role == "" {
			req.Role = "user"
		}

		// For kid role, default to PG if no limit set
		if req.Role == "kid" && req.ContentRatingLimit == nil {
			pg := "PG"
			req.ContentRatingLimit = &pg
		}

		user, err := s.auth.CreateUser(req.Username, req.Password, req.Role)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set parental controls
		user.ContentRatingLimit = req.ContentRatingLimit
		user.RequirePin = req.RequirePin
		if err := s.db.UpdateUser(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set PIN if provided
		if req.Pin != "" && len(req.Pin) == 4 {
			pinHash, err := auth.HashPassword(req.Pin)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPin(user.ID, &pinHash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/users/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := s.db.GetUserByID(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		var req struct {
			Username           string  `json:"username"`
			Password           string  `json:"password"`
			Role               string  `json:"role"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
			RequirePin         *bool   `json:"requirePin"`
			Pin                string  `json:"pin"`
			ClearPin           bool    `json:"clearPin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := s.db.GetUserByID(id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if req.Username != "" {
			user.Username = req.Username
		}
		if req.Role != "" {
			user.Role = req.Role
			// For kid role, default to PG if no limit set
			if req.Role == "kid" && user.ContentRatingLimit == nil && req.ContentRatingLimit == nil {
				pg := "PG"
				user.ContentRatingLimit = &pg
			}
		}

		// Handle content rating limit - allow setting to nil to remove limit
		// The request sends the field, so we update it
		user.ContentRatingLimit = req.ContentRatingLimit

		if req.RequirePin != nil {
			user.RequirePin = *req.RequirePin
		}

		if err := s.db.UpdateUser(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update password if provided
		if req.Password != "" {
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPassword(id, hash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Update or clear PIN
		if req.ClearPin {
			if err := s.db.UpdateUserPin(id, nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if req.Pin != "" && len(req.Pin) == 4 {
			pinHash, err := auth.HashPassword(req.Pin)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := s.db.UpdateUserPin(user.ID, &pinHash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		json.NewEncoder(w).Encode(user)

	case http.MethodDelete:
		// Don't allow deleting yourself
		currentUser := s.getCurrentUser(r)
		if currentUser != nil && currentUser.ID == id {
			http.Error(w, "Cannot delete yourself", http.StatusBadRequest)
			return
		}

		// Delete user sessions first
		s.db.DeleteUserSessions(id)

		if err := s.db.DeleteUser(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Profile handlers

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		profiles, err := s.db.GetProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if profiles == nil {
			profiles = []database.Profile{}
		}
		json.NewEncoder(w).Encode(profiles)

	case http.MethodPost:
		var req struct {
			Name               string  `json:"name"`
			AvatarURL          *string `json:"avatarUrl"`
			IsKid              bool    `json:"isKid"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Profile name is required", http.StatusBadRequest)
			return
		}

		// Check profile limit (max 5 per user)
		count, err := s.db.CountProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count >= 5 {
			http.Error(w, "Maximum 5 profiles per user", http.StatusBadRequest)
			return
		}

		profile := &database.Profile{
			UserID:             user.ID,
			Name:               req.Name,
			AvatarURL:          req.AvatarURL,
			IsKid:              req.IsKid,
			ContentRatingLimit: req.ContentRatingLimit,
		}

		if err := s.db.CreateProfile(profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(profile)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := s.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse path: /api/profiles/{id} or /api/profiles/{id}/select
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Profile ID required", http.StatusBadRequest)
		return
	}

	// Check for special "active" endpoint
	if parts[0] == "active" {
		s.handleActiveProfile(w, r, user)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Check if this is a select action
	if len(parts) >= 2 && parts[1] == "select" {
		s.handleProfileSelect(w, r, user, id)
		return
	}

	// Get profile and verify ownership
	profile, err := s.db.GetProfile(id)
	if err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	if profile.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(profile)

	case http.MethodPut:
		var req struct {
			Name               string  `json:"name"`
			AvatarURL          *string `json:"avatarUrl"`
			IsKid              *bool   `json:"isKid"`
			ContentRatingLimit *string `json:"contentRatingLimit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name != "" {
			profile.Name = req.Name
		}
		if req.AvatarURL != nil {
			profile.AvatarURL = req.AvatarURL
		}
		if req.IsKid != nil {
			profile.IsKid = *req.IsKid
		}
		profile.ContentRatingLimit = req.ContentRatingLimit

		if err := s.db.UpdateProfile(profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodDelete:
		// Don't allow deleting the only profile
		count, err := s.db.CountProfilesByUser(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count <= 1 {
			http.Error(w, "Cannot delete the only profile", http.StatusBadRequest)
			return
		}

		// Don't allow deleting default profile
		if profile.IsDefault {
			http.Error(w, "Cannot delete default profile", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteProfile(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfileSelect(w http.ResponseWriter, r *http.Request, user *database.User, profileID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify profile belongs to user
	profile, err := s.db.GetProfile(profileID)
	if err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	if profile.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get session token and update active profile
	token := s.getSessionToken(r)
	if token == "" {
		http.Error(w, "No session", http.StatusBadRequest)
		return
	}

	if err := s.db.SetActiveProfile(token, profileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

func (s *Server) handleActiveProfile(w http.ResponseWriter, r *http.Request, user *database.User) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profileID := s.getActiveProfileID(r)
	if profileID == nil {
		// No active profile, return null
		w.Write([]byte("null"))
		return
	}

	profile, err := s.db.GetProfile(*profileID)
	if err != nil {
		w.Write([]byte("null"))
		return
	}

	// Verify profile still belongs to user
	if profile.UserID != user.ID {
		w.Write([]byte("null"))
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// Download client handlers

func (s *Server) handleDownloadClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		clients, err := s.db.GetDownloadClients()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if clients == nil {
			clients = []database.DownloadClient{}
		}
		// Don't expose passwords in responses
		for i := range clients {
			clients[i].Password = ""
		}
		json.NewEncoder(w).Encode(clients)

	case http.MethodPost:
		var client database.DownloadClient
		if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if client.Name == "" || client.Type == "" || client.Host == "" || client.Port == 0 {
			http.Error(w, "Name, type, host, and port are required", http.StatusBadRequest)
			return
		}

		// Validate client type
		validTypes := map[string]bool{"qbittorrent": true, "transmission": true, "sabnzbd": true, "nzbget": true}
		if !validTypes[client.Type] {
			http.Error(w, "Invalid client type", http.StatusBadRequest)
			return
		}

		client.Enabled = true
		if err := s.db.CreateDownloadClient(&client); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client.Password = "" // Don't expose password
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDownloadClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/download-clients/{id} or /api/download-clients/{id}/test
	path := strings.TrimPrefix(r.URL.Path, "/api/download-clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	// Handle test endpoint
	if len(parts) == 2 && parts[1] == "test" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.downloads.TestClient(id); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Connection successful",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		client, err := s.db.GetDownloadClient(id)
		if err != nil {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}
		client.Password = ""
		json.NewEncoder(w).Encode(client)

	case http.MethodPut:
		var req database.DownloadClient
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		client, err := s.db.GetDownloadClient(id)
		if err != nil {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}

		// Update fields
		if req.Name != "" {
			client.Name = req.Name
		}
		if req.Type != "" {
			client.Type = req.Type
		}
		if req.Host != "" {
			client.Host = req.Host
		}
		if req.Port != 0 {
			client.Port = req.Port
		}
		if req.Username != "" {
			client.Username = req.Username
		}
		if req.Password != "" {
			client.Password = req.Password
		}
		if req.APIKey != "" {
			client.APIKey = req.APIKey
		}
		client.UseTLS = req.UseTLS
		client.Category = req.Category
		client.Priority = req.Priority
		client.Enabled = req.Enabled

		if err := s.db.UpdateDownloadClient(client); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client.Password = ""
		json.NewEncoder(w).Encode(client)

	case http.MethodDelete:
		if err := s.db.DeleteDownloadClient(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	downloads, err := s.downloads.GetAllDownloads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if downloads == nil {
		downloads = []downloadclient.Download{}
	}

	json.NewEncoder(w).Encode(downloads)
}

// Indexer handlers

func (s *Server) handleIndexers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		indexers, err := s.db.GetIndexers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if indexers == nil {
			indexers = []database.Indexer{}
		}
		// Don't expose API keys in responses
		for i := range indexers {
			indexers[i].APIKey = ""
		}
		json.NewEncoder(w).Encode(indexers)

	case http.MethodPost:
		var idx database.Indexer
		if err := json.NewDecoder(r.Body).Decode(&idx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if idx.Name == "" || idx.Type == "" || idx.URL == "" {
			http.Error(w, "Name, type, and URL are required", http.StatusBadRequest)
			return
		}

		// Validate indexer type
		validTypes := map[string]bool{"torznab": true, "newznab": true, "prowlarr": true}
		if !validTypes[idx.Type] {
			http.Error(w, "Invalid indexer type", http.StatusBadRequest)
			return
		}

		idx.Enabled = true
		if err := s.db.CreateIndexer(&idx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add to manager
		config := &indexer.IndexerConfig{
			ID:         idx.ID,
			Name:       idx.Name,
			Type:       idx.Type,
			URL:        idx.URL,
			APIKey:     idx.APIKey,
			Categories: idx.Categories,
			Priority:   idx.Priority,
			Enabled:    idx.Enabled,
		}
		if err := s.indexers.AddIndexer(config); err != nil {
			// Indexer created but connection failed - still return success
			idx.APIKey = ""
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(idx)
			return
		}

		idx.APIKey = ""
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(idx)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIndexer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/indexers/{id} or /api/indexers/{id}/test
	path := strings.TrimPrefix(r.URL.Path, "/api/indexers/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Indexer ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid indexer ID", http.StatusBadRequest)
		return
	}

	// Handle test endpoint
	if len(parts) == 2 && parts[1] == "test" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.indexers.TestIndexer(id); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Connection successful",
		})
		return
	}

	// Handle capabilities endpoint
	if len(parts) == 2 && parts[1] == "capabilities" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		caps, err := s.indexers.GetCapabilities(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(caps)
		return
	}

	switch r.Method {
	case http.MethodGet:
		idx, err := s.db.GetIndexer(id)
		if err != nil {
			http.Error(w, "Indexer not found", http.StatusNotFound)
			return
		}
		idx.APIKey = ""
		json.NewEncoder(w).Encode(idx)

	case http.MethodPut:
		var req database.Indexer
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		idx, err := s.db.GetIndexer(id)
		if err != nil {
			http.Error(w, "Indexer not found", http.StatusNotFound)
			return
		}

		// Update fields
		if req.Name != "" {
			idx.Name = req.Name
		}
		if req.Type != "" {
			idx.Type = req.Type
		}
		if req.URL != "" {
			idx.URL = req.URL
		}
		if req.APIKey != "" {
			idx.APIKey = req.APIKey
		}
		idx.Categories = req.Categories
		idx.Priority = req.Priority
		idx.Enabled = req.Enabled

		if err := s.db.UpdateIndexer(idx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update manager
		s.indexers.RemoveIndexer(id)
		if idx.Enabled {
			config := &indexer.IndexerConfig{
				ID:         idx.ID,
				Name:       idx.Name,
				Type:       idx.Type,
				URL:        idx.URL,
				APIKey:     idx.APIKey,
				Categories: idx.Categories,
				Priority:   idx.Priority,
				Enabled:    idx.Enabled,
			}
			s.indexers.AddIndexer(config)
		}

		idx.APIKey = ""
		json.NewEncoder(w).Encode(idx)

	case http.MethodDelete:
		s.indexers.RemoveIndexer(id)
		if err := s.db.DeleteIndexer(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}



// Prowlarr handlers

func (s *Server) handleProwlarrConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		config, err := s.db.GetProwlarrConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if config == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
		// Mask API key for response
		config.APIKey = ""
		json.NewEncoder(w).Encode(config)

	case http.MethodPost:
		var config database.ProwlarrConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if config.URL == "" || config.APIKey == "" {
			http.Error(w, "URL and API key are required", http.StatusBadRequest)
			return
		}

		if err := s.db.SaveProwlarrConfig(&config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config.APIKey = ""
		json.NewEncoder(w).Encode(config)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProwlarrTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	syncSvc := prowlarr.NewSyncService(s.db)
	if err := syncSvc.TestConnection(req.URL, req.APIKey); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Also return indexer count for preview
	indexers, _ := syncSvc.FetchIndexers(req.URL, req.APIKey)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"indexerCount": len(indexers),
	})
}

func (s *Server) handleProwlarrSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	syncSvc := prowlarr.NewSyncService(s.db)
	synced, err := syncSvc.SyncAll()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Reload indexers into the manager after sync
	s.reloadIndexers()

	// Return updated indexer list
	indexers, _ := s.db.GetSyncedIndexers()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"synced":   synced,
		"indexers": indexers,
	})
}

func (s *Server) handleIndexerTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tags, err := s.db.GetIndexerTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tags == nil {
		tags = []database.IndexerTag{}
	}
	json.NewEncoder(w).Encode(tags)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	params := indexer.SearchParams{
		Query:  query.Get("q"),
		Type:   query.Get("type"),
		ImdbID: query.Get("imdbId"),
		TvdbID: query.Get("tvdbId"),
		TmdbID: query.Get("tmdbId"),
	}

	if season := query.Get("season"); season != "" {
		params.Season, _ = strconv.Atoi(season)
	}
	if episode := query.Get("episode"); episode != "" {
		params.Episode, _ = strconv.Atoi(episode)
	}
	if limit := query.Get("limit"); limit != "" {
		params.Limit, _ = strconv.Atoi(limit)
	}
	if cats := query.Get("categories"); cats != "" {
		for _, cat := range strings.Split(cats, ",") {
			if c, err := strconv.Atoi(strings.TrimSpace(cat)); err == nil {
				params.Categories = append(params.Categories, c)
			}
		}
	}

	results, err := s.indexers.Search(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []indexer.SearchResult{}
	}

	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleSearchScored(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	params := indexer.SearchParams{
		Query:  query.Get("q"),
		Type:   query.Get("type"),
		ImdbID: query.Get("imdbId"),
		TvdbID: query.Get("tvdbId"),
		TmdbID: query.Get("tmdbId"),
	}

	if season := query.Get("season"); season != "" {
		params.Season, _ = strconv.Atoi(season)
	}
	if episode := query.Get("episode"); episode != "" {
		params.Episode, _ = strconv.Atoi(episode)
	}
	if limit := query.Get("limit"); limit != "" {
		params.Limit, _ = strconv.Atoi(limit)
	}
	if cats := query.Get("categories"); cats != "" {
		for _, cat := range strings.Split(cats, ",") {
			if c, err := strconv.Atoi(strings.TrimSpace(cat)); err == nil {
				params.Categories = append(params.Categories, c)
			}
		}
	}

	// Get profile ID for scoring
	var profileID int64
	if pid := query.Get("profileId"); pid != "" {
		profileID, _ = strconv.ParseInt(pid, 10, 64)
	}

	// Search indexers
	results, err := s.indexers.Search(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []indexer.SearchResult{}
	}

	// Get profile and custom formats for scoring
	var profile *quality.Profile
	var customFormats []quality.CustomFormatDef

	if profileID > 0 {
		dbProfile, err := s.db.GetQualityProfile(profileID)
		if err == nil {
			qualities, _ := quality.ParseQualities(dbProfile.Qualities)
			scores, _ := quality.ParseCustomFormatScores(dbProfile.CustomFormatScores)
			profile = &quality.Profile{
				ID:                 dbProfile.ID,
				Name:               dbProfile.Name,
				UpgradeAllowed:     dbProfile.UpgradeAllowed,
				UpgradeUntilScore:  dbProfile.UpgradeUntilScore,
				MinFormatScore:     dbProfile.MinFormatScore,
				CutoffFormatScore:  dbProfile.CutoffFormatScore,
				Qualities:          qualities,
				CustomFormatScores: scores,
			}
		}

		// Get custom formats
		dbFormats, err := s.db.GetCustomFormats()
		if err == nil {
			for _, f := range dbFormats {
				conditions, _ := quality.ParseConditions(f.Conditions)
				customFormats = append(customFormats, quality.CustomFormatDef{
					ID:         f.ID,
					Name:       f.Name,
					Conditions: conditions,
				})
			}
		}
	}

	// Score each result
	scoredResults := make([]indexer.ScoredSearchResult, 0, len(results))
	for _, result := range results {
		parsed := parser.Parse(result.Title)
		qualityTier := quality.ComputeQualityTier(parsed)

		// Convert HDR string to slice (indexer uses []string)
		var hdrSlice []string
		if parsed.HDR != "" {
			hdrSlice = []string{parsed.HDR}
		}

		scored := indexer.ScoredSearchResult{
			SearchResult: result,
			Quality:      qualityTier,
			Resolution:   parsed.Resolution,
			Source:       parsed.Source,
			Codec:        parsed.Codec,
			AudioCodec:   parsed.AudioFormat,
			AudioFeature: parsed.AudioChannels,
			HDR:          hdrSlice,
			ReleaseGroup: parsed.ReleaseGroup,
			Proper:       parsed.IsProper,
			Repack:       parsed.IsRepack,
		}

		// Apply scoring if profile is available
		if profile != nil {
			scoredRelease := quality.ScoreRelease(parsed, profile, customFormats)
			scored.BaseScore = scoredRelease.BaseScore
			scored.TotalScore = scoredRelease.TotalScore
			scored.Rejected = scoredRelease.Rejected
			scored.RejectionReason = scoredRelease.RejectionReason

			// Convert custom format hits
			for _, hit := range scoredRelease.CustomFormatHits {
				scored.CustomFormatHits = append(scored.CustomFormatHits, indexer.CustomFormatHit{
					Name:  hit.Name,
					Score: hit.Score,
				})
			}
		} else {
			// No profile, just use base score for quality tier
			scored.BaseScore = quality.BaseQualityScores[qualityTier]
			scored.TotalScore = scored.BaseScore
		}

		scoredResults = append(scoredResults, scored)
	}

	// Sort by total score (descending)
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].TotalScore > scoredResults[i].TotalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	json.NewEncoder(w).Encode(scoredResults)
}

func (s *Server) handleGrab(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Link       string `json:"link"`
		MagnetLink string `json:"magnetLink"`
		IndexerType string `json:"indexerType"`
		Category   string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Link == "" && req.MagnetLink == "" {
		http.Error(w, "Either link or magnetLink is required", http.StatusBadRequest)
		return
	}

	// Get appropriate download client
	var downloadURL string
	if req.MagnetLink != "" {
		downloadURL = req.MagnetLink
	} else {
		downloadURL = req.Link
	}

	// Determine if this is a torrent or NZB based on indexer type
	isTorrent := req.IndexerType == "torznab" || req.MagnetLink != ""

	// Get enabled download clients
	clients, err := s.db.GetEnabledDownloadClients()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find appropriate client
	var targetClient *database.DownloadClient
	for _, client := range clients {
		if isTorrent && (client.Type == "qbittorrent" || client.Type == "transmission") {
			targetClient = &client
			break
		}
		if !isTorrent && (client.Type == "sabnzbd" || client.Type == "nzbget") {
			targetClient = &client
			break
		}
	}

	if targetClient == nil {
		http.Error(w, "No suitable download client found", http.StatusBadRequest)
		return
	}

	// Add to download client
	if isTorrent {
		err = s.downloads.AddTorrent(targetClient.ID, downloadURL, req.Category)
	} else {
		err = s.downloads.AddNZB(targetClient.ID, downloadURL, req.Category)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Added to download client",
		"client":  targetClient.Name,
	})
}

// Quality Profile handlers

func (s *Server) handleQualityProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profiles, err := s.db.GetQualityProfiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if profiles == nil {
			profiles = []database.QualityProfile{}
		}
		json.NewEncoder(w).Encode(profiles)

	case http.MethodPost:
		// POST requires admin
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var profile database.QualityProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if profile.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		// Set defaults
		if profile.Qualities == "" {
			profile.Qualities = "[]"
		}
		if profile.CustomFormatScores == "" {
			profile.CustomFormatScores = "{}"
		}

		if err := s.db.CreateQualityProfile(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(profile)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleQualityProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/quality-profiles/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := s.db.GetQualityProfile(id)
		if err != nil {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodPut:
		var req database.QualityProfile
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		profile, err := s.db.GetQualityProfile(id)
		if err != nil {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		}

		// Update fields
		if req.Name != "" {
			profile.Name = req.Name
		}
		profile.UpgradeAllowed = req.UpgradeAllowed
		profile.UpgradeUntilScore = req.UpgradeUntilScore
		profile.MinFormatScore = req.MinFormatScore
		profile.CutoffFormatScore = req.CutoffFormatScore
		if req.Qualities != "" {
			profile.Qualities = req.Qualities
		}
		if req.CustomFormatScores != "" {
			profile.CustomFormatScores = req.CustomFormatScores
		}

		if err := s.db.UpdateQualityProfile(profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(profile)

	case http.MethodDelete:
		if err := s.db.DeleteQualityProfile(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Custom Format handlers

func (s *Server) handleCustomFormats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		formats, err := s.db.GetCustomFormats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if formats == nil {
			formats = []database.CustomFormat{}
		}
		json.NewEncoder(w).Encode(formats)

	case http.MethodPost:
		var format database.CustomFormat
		if err := json.NewDecoder(r.Body).Decode(&format); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if format.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		if format.Conditions == "" {
			format.Conditions = "[]"
		}

		if err := s.db.CreateCustomFormat(&format); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(format)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCustomFormat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/custom-formats/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid format ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		format, err := s.db.GetCustomFormat(id)
		if err != nil {
			http.Error(w, "Format not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(format)

	case http.MethodPut:
		var req database.CustomFormat
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		format, err := s.db.GetCustomFormat(id)
		if err != nil {
			http.Error(w, "Format not found", http.StatusNotFound)
			return
		}

		if req.Name != "" {
			format.Name = req.Name
		}
		if req.Conditions != "" {
			format.Conditions = req.Conditions
		}

		if err := s.db.UpdateCustomFormat(format); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(format)

	case http.MethodDelete:
		if err := s.db.DeleteCustomFormat(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Release parsing handler

func (s *Server) handleParseRelease(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	parsed := parser.Parse(req.Name)
	// Include computed quality tier in response
	response := struct {
		*parser.ParsedRelease
		Quality string `json:"quality"`
	}{
		ParsedRelease: parsed,
		Quality:       quality.ComputeQualityTier(parsed),
	}
	json.NewEncoder(w).Encode(response)
}

// Wanted/Monitoring handlers

func (s *Server) handleWantedItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		items, err := s.db.GetWantedItems()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if items == nil {
			items = []database.WantedItem{}
		}
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var item database.WantedItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if item.Title == "" || item.TmdbID == 0 {
			http.Error(w, "Title and tmdbId are required", http.StatusBadRequest)
			return
		}

		if item.Type == "" {
			item.Type = "movie"
		}

		if item.Seasons == "" {
			item.Seasons = "[]"
		}

		// Check if already exists
		existing, _ := s.db.GetWantedByTmdb(item.Type, item.TmdbID)
		if existing != nil {
			http.Error(w, "Item already in wanted list", http.StatusConflict)
			return
		}

		if err := s.db.CreateWantedItem(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWantedItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/wanted/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/wanted/")
	// Handle /api/wanted/search/{id} separately
	if strings.HasPrefix(path, "search/") {
		return // Let handleWantedSearch handle it
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := s.db.GetWantedItem(id)
		if err != nil {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(item)

	case http.MethodPut:
		item, err := s.db.GetWantedItem(id)
		if err != nil {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}

		var update struct {
			QualityProfileID *int64 `json:"qualityProfileId"`
			Monitored        *bool  `json:"monitored"`
			Seasons          string `json:"seasons"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if update.QualityProfileID != nil {
			item.QualityProfileID = *update.QualityProfileID
		}
		if update.Monitored != nil {
			item.Monitored = *update.Monitored
		}
		if update.Seasons != "" {
			item.Seasons = update.Seasons
		}

		if err := s.db.UpdateWantedItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(item)

	case http.MethodDelete:
		if err := s.db.DeleteWantedItem(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWantedSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /api/wanted/search/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/wanted/search/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	item, err := s.db.GetWantedItem(id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Search for the item
	searchType := "movie"
	if item.Type == "show" {
		searchType = "tvsearch"
	}

	params := indexer.SearchParams{
		Query: item.Title,
		Type:  searchType,
		Limit: 50,
	}

	// Add TMDB ID to search if we have it
	if item.TmdbID > 0 {
		params.TmdbID = strconv.FormatInt(item.TmdbID, 10)
	}

	results, err := s.indexers.Search(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update last searched timestamp
	s.db.UpdateWantedLastSearched(id)

	// Get profile and score results
	var profile *quality.Profile
	var customFormats []quality.CustomFormatDef

	if item.QualityProfileID > 0 {
		dbProfile, err := s.db.GetQualityProfile(item.QualityProfileID)
		if err == nil {
			qualities, _ := quality.ParseQualities(dbProfile.Qualities)
			scores, _ := quality.ParseCustomFormatScores(dbProfile.CustomFormatScores)
			profile = &quality.Profile{
				ID:                 dbProfile.ID,
				Name:               dbProfile.Name,
				UpgradeAllowed:     dbProfile.UpgradeAllowed,
				UpgradeUntilScore:  dbProfile.UpgradeUntilScore,
				MinFormatScore:     dbProfile.MinFormatScore,
				CutoffFormatScore:  dbProfile.CutoffFormatScore,
				Qualities:          qualities,
				CustomFormatScores: scores,
			}
		}

		dbFormats, err := s.db.GetCustomFormats()
		if err == nil {
			for _, f := range dbFormats {
				conditions, _ := quality.ParseConditions(f.Conditions)
				customFormats = append(customFormats, quality.CustomFormatDef{
					ID:         f.ID,
					Name:       f.Name,
					Conditions: conditions,
				})
			}
		}
	}

	// Score results
	scoredResults := make([]indexer.ScoredSearchResult, 0, len(results))
	for _, result := range results {
		parsed := parser.Parse(result.Title)
		qualityTier := quality.ComputeQualityTier(parsed)

		// Convert HDR string to slice (indexer uses []string)
		var hdrSlice []string
		if parsed.HDR != "" {
			hdrSlice = []string{parsed.HDR}
		}

		scored := indexer.ScoredSearchResult{
			SearchResult: result,
			Quality:      qualityTier,
			Resolution:   parsed.Resolution,
			Source:       parsed.Source,
			Codec:        parsed.Codec,
			AudioCodec:   parsed.AudioFormat,
			AudioFeature: parsed.AudioChannels,
			HDR:          hdrSlice,
			ReleaseGroup: parsed.ReleaseGroup,
			Proper:       parsed.IsProper,
			Repack:       parsed.IsRepack,
		}

		if profile != nil {
			scoredRelease := quality.ScoreRelease(parsed, profile, customFormats)
			scored.BaseScore = scoredRelease.BaseScore
			scored.TotalScore = scoredRelease.TotalScore
			scored.Rejected = scoredRelease.Rejected
			scored.RejectionReason = scoredRelease.RejectionReason

			for _, hit := range scoredRelease.CustomFormatHits {
				scored.CustomFormatHits = append(scored.CustomFormatHits, indexer.CustomFormatHit{
					Name:  hit.Name,
					Score: hit.Score,
				})
			}
		} else {
			scored.BaseScore = quality.BaseQualityScores[qualityTier]
			scored.TotalScore = scored.BaseScore
		}

		scoredResults = append(scoredResults, scored)
	}

	// Sort by score
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].TotalScore > scoredResults[i].TotalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	json.NewEncoder(w).Encode(scoredResults)
}

// Discover handlers

// DiscoverItemWithStatus wraps a discover item with library/request status
type DiscoverItemWithStatus struct {
	ID            int64    `json:"id"`
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Overview      string   `json:"overview"`
	ReleaseDate   string   `json:"releaseDate"`
	PosterPath    string   `json:"posterPath"`
	BackdropPath  string   `json:"backdropPath"`
	Rating        float64  `json:"rating"`
	Popularity    float64  `json:"popularity"`
	FocalX        *float64 `json:"focalX,omitempty"`
	FocalY        *float64 `json:"focalY,omitempty"`
	InLibrary     bool     `json:"inLibrary"`
	LibraryID     *int64   `json:"libraryId,omitempty"`
	Requested     bool     `json:"requested"`
	RequestID     *int64   `json:"requestId,omitempty"`
	RequestStatus *string  `json:"requestStatus,omitempty"`
}

// DiscoverResultWithStatus is discover result with status fields
type DiscoverResultWithStatus struct {
	Page         int                       `json:"page"`
	TotalPages   int                       `json:"totalPages"`
	TotalResults int                       `json:"totalResults"`
	Results      []DiscoverItemWithStatus  `json:"results"`
}

// enrichMovieResults adds library/request status to movie results
func (s *Server) enrichMovieResults(result *metadata.DiscoverResult) *DiscoverResultWithStatus {
	// Collect all TMDB IDs
	tmdbIDs := make([]int64, len(result.Results))
	for i, item := range result.Results {
		tmdbIDs[i] = item.ID
	}

	// Get bulk status
	statuses, _ := s.db.GetBulkMovieStatus(tmdbIDs)

	// Build enriched results
	enriched := &DiscoverResultWithStatus{
		Page:         result.Page,
		TotalPages:   result.TotalPages,
		TotalResults: result.TotalResults,
		Results:      make([]DiscoverItemWithStatus, len(result.Results)),
	}

	for i, item := range result.Results {
		enriched.Results[i] = DiscoverItemWithStatus{
			ID:           item.ID,
			Type:         item.Type,
			Title:        item.Title,
			Overview:     item.Overview,
			ReleaseDate:  item.ReleaseDate,
			PosterPath:   item.PosterPath,
			BackdropPath: item.BackdropPath,
			Rating:       item.Rating,
			Popularity:   item.Popularity,
			FocalX:       item.FocalX,
			FocalY:       item.FocalY,
		}
		if status, ok := statuses[item.ID]; ok {
			enriched.Results[i].InLibrary = status.InLibrary
			enriched.Results[i].LibraryID = status.LibraryID
			enriched.Results[i].Requested = status.Requested
			enriched.Results[i].RequestID = status.RequestID
			enriched.Results[i].RequestStatus = status.RequestStatus
		}
	}

	return enriched
}

// enrichTVResults adds library/request status to TV results
func (s *Server) enrichTVResults(result *metadata.DiscoverResult) *DiscoverResultWithStatus {
	// Collect all TMDB IDs
	tmdbIDs := make([]int64, len(result.Results))
	for i, item := range result.Results {
		tmdbIDs[i] = item.ID
	}

	// Get bulk status
	statuses, _ := s.db.GetBulkShowStatus(tmdbIDs)

	// Build enriched results
	enriched := &DiscoverResultWithStatus{
		Page:         result.Page,
		TotalPages:   result.TotalPages,
		TotalResults: result.TotalResults,
		Results:      make([]DiscoverItemWithStatus, len(result.Results)),
	}

	for i, item := range result.Results {
		enriched.Results[i] = DiscoverItemWithStatus{
			ID:           item.ID,
			Type:         item.Type,
			Title:        item.Title,
			Overview:     item.Overview,
			ReleaseDate:  item.ReleaseDate,
			PosterPath:   item.PosterPath,
			BackdropPath: item.BackdropPath,
			Rating:       item.Rating,
			Popularity:   item.Popularity,
			FocalX:       item.FocalX,
			FocalY:       item.FocalY,
		}
		if status, ok := statuses[item.ID]; ok {
			enriched.Results[i].InLibrary = status.InLibrary
			enriched.Results[i].LibraryID = status.LibraryID
			enriched.Results[i].Requested = status.Requested
			enriched.Results[i].RequestID = status.RequestID
			enriched.Results[i].RequestStatus = status.RequestStatus
		}
	}

	return enriched
}

// Public endpoint for login page background - returns poster URLs only
func (s *Server) handlePublicTrendingPosters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var posters []string

	// Get trending movies
	movies, err := s.metadata.GetTrendingMovies(1)
	if err == nil {
		for _, m := range movies.Results {
			if m.PosterPath != "" {
				posters = append(posters, m.PosterPath)
			}
		}
	}

	// Get trending TV shows
	shows, err := s.metadata.GetTrendingTV(1)
	if err == nil {
		for _, s := range shows.Results {
			if s.PosterPath != "" {
				posters = append(posters, s.PosterPath)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"posters": posters})
}

func (s *Server) handleDiscoverTrendingMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetTrendingMovies(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverPopularMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetPopularMovies(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverUpcomingMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetUpcomingMovies(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverTheatricalReleases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	region := r.URL.Query().Get("region") // Optional: US, GB, CA, AU, etc.

	result, err := s.metadata.GetTheatricalReleases(region, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverUpcomingTV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetUpcomingTV(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichTVResults(result))
}

func (s *Server) handleDiscoverTopRatedMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetTopRatedMovies(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverTrendingTV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetTrendingTV(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichTVResults(result))
}

func (s *Server) handleDiscoverPopularTV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetPopularTV(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichTVResults(result))
}

func (s *Server) handleDiscoverTopRatedTV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	result, err := s.metadata.GetTopRatedTV(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichTVResults(result))
}

func (s *Server) handleMovieGenres(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	genres, err := s.metadata.GetMovieGenres()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"genres": genres})
}

func (s *Server) handleTVGenres(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	genres, err := s.metadata.GetTVGenres()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"genres": genres})
}

func (s *Server) handleDiscoverMoviesByGenre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract genre ID from path: /api/discover/movies/genre/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/discover/movies/genre/")
	genreID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid genre ID", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := s.metadata.GetMoviesByGenre(genreID, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichMovieResults(result))
}

func (s *Server) handleDiscoverTVByGenre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract genre ID from path: /api/discover/shows/genre/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/discover/shows/genre/")
	genreID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid genre ID", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		page, _ = strconv.Atoi(p)
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := s.metadata.GetTVByGenre(genreID, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.enrichTVResults(result))
}

func (s *Server) handleDiscoverMovieDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/discover/movie/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	log.Printf("Discover movie detail request for ID: %d", id)

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := s.metadata.GetMovieDetail(id)
	if err != nil {
		log.Printf("Error getting movie detail: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Movie %s has %d recommendations", result.Title, len(result.Recommendations))

	// Enrich with library/request status
	response := struct {
		*metadata.DiscoverMovieDetail
		InLibrary     bool    `json:"inLibrary"`
		LibraryID     *int64  `json:"libraryId,omitempty"`
		Requested     bool    `json:"requested"`
		RequestID     *int64  `json:"requestId,omitempty"`
		RequestStatus *string `json:"requestStatus,omitempty"`
	}{
		DiscoverMovieDetail: result,
	}

	status, err := s.db.GetMovieStatusByTmdbID(id)
	if err == nil && status != nil {
		response.InLibrary = status.InLibrary
		response.LibraryID = status.LibraryID
		response.Requested = status.Requested
		response.RequestID = status.RequestID
		response.RequestStatus = status.RequestStatus
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleDiscoverShowDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/discover/show/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if s.metadata == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := s.metadata.GetShowDetail(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with library/request status
	response := struct {
		*metadata.DiscoverShowDetail
		InLibrary     bool    `json:"inLibrary"`
		LibraryID     *int64  `json:"libraryId,omitempty"`
		Requested     bool    `json:"requested"`
		RequestID     *int64  `json:"requestId,omitempty"`
		RequestStatus *string `json:"requestStatus,omitempty"`
	}{
		DiscoverShowDetail: result,
	}

	status, err := s.db.GetShowStatusByTmdbID(id)
	if err == nil && status != nil {
		response.InLibrary = status.InLibrary
		response.LibraryID = status.LibraryID
		response.Requested = status.Requested
		response.RequestID = status.RequestID
		response.RequestStatus = status.RequestStatus
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMovieTrailers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/trailers/movie/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	tmdbClient := s.metadata.GetTMDBClient()
	if tmdbClient == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	details, err := tmdbClient.GetMovieDetails(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	trailers := make([]map[string]interface{}, 0)
	for _, v := range details.Videos.Results {
		trailers = append(trailers, map[string]interface{}{
			"key":      v.Key,
			"name":     v.Name,
			"type":     v.Type,
			"site":     v.Site,
			"official": v.Official,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trailers)
}

func (s *Server) handleTVTrailers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/trailers/tv/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	tmdbClient := s.metadata.GetTMDBClient()
	if tmdbClient == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	details, err := tmdbClient.GetTVDetails(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	trailers := make([]map[string]interface{}, 0)
	for _, v := range details.Videos.Results {
		trailers = append(trailers, map[string]interface{}{
			"key":      v.Key,
			"name":     v.Name,
			"type":     v.Type,
			"site":     v.Site,
			"official": v.Official,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trailers)
}

func (s *Server) handleMovieRecommendations(w http.ResponseWriter, r *http.Request) {
	// Extract TMDB ID from path: /api/movie/recommendations/{tmdbId}
	path := strings.TrimPrefix(r.URL.Path, "/api/movie/recommendations/")
	idStr := strings.TrimSuffix(path, "/")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid TMDB ID", http.StatusBadRequest)
		return
	}

	if s.metadata == nil || s.metadata.GetTMDBClient() == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := s.metadata.GetTMDBClient().GetMovieRecommendations(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleMovieSuggestions returns movie suggestions based on genres, excluding library items
func (s *Server) handleMovieSuggestions(w http.ResponseWriter, r *http.Request) {
	// Extract movie ID from path: /api/movies/suggestions/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/movies/suggestions/")
	idStr := strings.TrimSuffix(path, "/")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get the movie from database
	movie, err := s.db.GetMovie(id)
	if err != nil {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	if s.metadata == nil || s.metadata.GetTMDBClient() == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	tmdbClient := s.metadata.GetTMDBClient()

	// Get all TMDB IDs in library for filtering
	libraryIDs, err := s.db.GetMovieTMDBIDs()
	if err != nil {
		libraryIDs = make(map[int64]bool) // Continue without filtering if error
	}

	// Parse genres from movie
	var genres []string
	if movie.Genres != nil && *movie.Genres != "" {
		if err := json.Unmarshal([]byte(*movie.Genres), &genres); err != nil {
			genres = []string{}
		}
	}

	// Response type matching TMDB format
	type SuggestionResult struct {
		Results []struct {
			ID           int64   `json:"id"`
			Title        string  `json:"title"`
			Overview     string  `json:"overview"`
			PosterPath   string  `json:"poster_path"`
			BackdropPath string  `json:"backdrop_path"`
			ReleaseDate  string  `json:"release_date"`
			VoteAverage  float64 `json:"vote_average"`
			Popularity   float64 `json:"popularity"`
		} `json:"results"`
	}

	var suggestions SuggestionResult

	// Try genre-based discovery first
	if len(genres) > 0 {
		// Get genre name to ID mapping
		genreMap, err := tmdbClient.GetGenreNameToIDMap()
		if err == nil {
			// Convert genre names to IDs
			var genreIDs []int
			for _, name := range genres {
				if id, ok := genreMap[name]; ok {
					genreIDs = append(genreIDs, id)
				}
			}

			if len(genreIDs) > 0 {
				// Discover movies by genres
				discovered, err := tmdbClient.DiscoverMoviesByGenres(genreIDs, 1)
				if err == nil {
					for _, m := range discovered.Results {
						// Skip if already in library or is the same movie
						if libraryIDs[m.ID] || (movie.TmdbID != nil && m.ID == *movie.TmdbID) {
							continue
						}
						suggestions.Results = append(suggestions.Results, struct {
							ID           int64   `json:"id"`
							Title        string  `json:"title"`
							Overview     string  `json:"overview"`
							PosterPath   string  `json:"poster_path"`
							BackdropPath string  `json:"backdrop_path"`
							ReleaseDate  string  `json:"release_date"`
							VoteAverage  float64 `json:"vote_average"`
							Popularity   float64 `json:"popularity"`
						}{
							ID:           m.ID,
							Title:        m.Title,
							Overview:     m.Overview,
							PosterPath:   m.PosterPath,
							BackdropPath: m.BackdropPath,
							ReleaseDate:  m.ReleaseDate,
							VoteAverage:  m.VoteAverage,
							Popularity:   m.Popularity,
						})
						if len(suggestions.Results) >= 25 {
							break
						}
					}
				}
			}
		}
	}

	// If not enough results, fall back to TMDB recommendations
	if len(suggestions.Results) < 10 && movie.TmdbID != nil {
		recs, err := tmdbClient.GetMovieRecommendations(*movie.TmdbID)
		if err == nil {
			for _, m := range recs.Results {
				// Skip if already in library, in suggestions, or is the same movie
				if libraryIDs[m.ID] || m.ID == *movie.TmdbID {
					continue
				}
				// Check if already in suggestions
				exists := false
				for _, s := range suggestions.Results {
					if s.ID == m.ID {
						exists = true
						break
					}
				}
				if exists {
					continue
				}

				suggestions.Results = append(suggestions.Results, struct {
					ID           int64   `json:"id"`
					Title        string  `json:"title"`
					Overview     string  `json:"overview"`
					PosterPath   string  `json:"poster_path"`
					BackdropPath string  `json:"backdrop_path"`
					ReleaseDate  string  `json:"release_date"`
					VoteAverage  float64 `json:"vote_average"`
					Popularity   float64 `json:"popularity"`
				}{
					ID:           m.ID,
					Title:        m.Title,
					Overview:     m.Overview,
					PosterPath:   m.PosterPath,
					BackdropPath: m.BackdropPath,
					ReleaseDate:  m.ReleaseDate,
					VoteAverage:  m.VoteAverage,
					Popularity:   m.Popularity,
				})
				if len(suggestions.Results) >= 25 {
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// handleShowSuggestions returns TV show suggestions based on genres, excluding library items
func (s *Server) handleShowSuggestions(w http.ResponseWriter, r *http.Request) {
	// Extract show ID from path: /api/shows/suggestions/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/shows/suggestions/")
	idStr := strings.TrimSuffix(path, "/")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	// Get the show from database
	show, err := s.db.GetShow(id)
	if err != nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}

	if s.metadata == nil || s.metadata.GetTMDBClient() == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	tmdbClient := s.metadata.GetTMDBClient()

	// Get all TMDB IDs in library for filtering
	libraryIDs, err := s.db.GetShowTMDBIDs()
	if err != nil {
		libraryIDs = make(map[int64]bool) // Continue without filtering if error
	}

	// Parse genres from show
	var genres []string
	if show.Genres != nil && *show.Genres != "" {
		if err := json.Unmarshal([]byte(*show.Genres), &genres); err != nil {
			genres = []string{}
		}
	}

	// Response type matching TMDB format
	type SuggestionResult struct {
		Results []struct {
			ID           int64   `json:"id"`
			Name         string  `json:"name"`
			Overview     string  `json:"overview"`
			PosterPath   string  `json:"poster_path"`
			BackdropPath string  `json:"backdrop_path"`
			FirstAirDate string  `json:"first_air_date"`
			VoteAverage  float64 `json:"vote_average"`
			Popularity   float64 `json:"popularity"`
		} `json:"results"`
	}

	var suggestions SuggestionResult

	// Try genre-based discovery first
	if len(genres) > 0 {
		// Get genre name to ID mapping for TV
		genreMap, err := tmdbClient.GetTVGenreNameToIDMap()
		if err == nil {
			// Convert genre names to IDs
			var genreIDs []int
			for _, name := range genres {
				if id, ok := genreMap[name]; ok {
					genreIDs = append(genreIDs, id)
				}
			}

			if len(genreIDs) > 0 {
				// Discover shows by genres
				discovered, err := tmdbClient.DiscoverTVByGenres(genreIDs, 1)
				if err == nil {
					for _, s := range discovered.Results {
						// Skip if already in library or is the same show
						if libraryIDs[s.ID] || (show.TmdbID != nil && s.ID == *show.TmdbID) {
							continue
						}
						suggestions.Results = append(suggestions.Results, struct {
							ID           int64   `json:"id"`
							Name         string  `json:"name"`
							Overview     string  `json:"overview"`
							PosterPath   string  `json:"poster_path"`
							BackdropPath string  `json:"backdrop_path"`
							FirstAirDate string  `json:"first_air_date"`
							VoteAverage  float64 `json:"vote_average"`
							Popularity   float64 `json:"popularity"`
						}{
							ID:           s.ID,
							Name:         s.Name,
							Overview:     s.Overview,
							PosterPath:   s.PosterPath,
							BackdropPath: s.BackdropPath,
							FirstAirDate: s.FirstAirDate,
							VoteAverage:  s.VoteAverage,
							Popularity:   s.Popularity,
						})
						if len(suggestions.Results) >= 25 {
							break
						}
					}
				}
			}
		}
	}

	// If not enough results, fall back to TMDB recommendations
	if len(suggestions.Results) < 10 && show.TmdbID != nil {
		recs, err := tmdbClient.GetTVRecommendations(*show.TmdbID)
		if err == nil {
			for _, s := range recs.Results {
				// Skip if already in library or is the same show
				if libraryIDs[s.ID] || s.ID == *show.TmdbID {
					continue
				}
				// Check if already in suggestions
				exists := false
				for _, existing := range suggestions.Results {
					if existing.ID == s.ID {
						exists = true
						break
					}
				}
				if exists {
					continue
				}

				suggestions.Results = append(suggestions.Results, struct {
					ID           int64   `json:"id"`
					Name         string  `json:"name"`
					Overview     string  `json:"overview"`
					PosterPath   string  `json:"poster_path"`
					BackdropPath string  `json:"backdrop_path"`
					FirstAirDate string  `json:"first_air_date"`
					VoteAverage  float64 `json:"vote_average"`
					Popularity   float64 `json:"popularity"`
				}{
					ID:           s.ID,
					Name:         s.Name,
					Overview:     s.Overview,
					PosterPath:   s.PosterPath,
					BackdropPath: s.BackdropPath,
					FirstAirDate: s.FirstAirDate,
					VoteAverage:  s.VoteAverage,
					Popularity:   s.Popularity,
				})
				if len(suggestions.Results) >= 25 {
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// Request handlers


// handlePerson returns detailed information about a person (actor/crew member)
func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract person ID from path: /api/person/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/person/")
	idStr := strings.TrimSuffix(path, "/")

	personID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	if s.metadata == nil || s.metadata.GetTMDBClient() == nil {
		http.Error(w, "TMDB not configured", http.StatusServiceUnavailable)
		return
	}

	tmdbClient := s.metadata.GetTMDBClient()

	// Fetch person details and credits in parallel
	type detailsResult struct {
		details interface{}
		err     error
	}
	type creditsResult struct {
		credits interface{}
		err     error
	}

	detailsChan := make(chan detailsResult)
	creditsChan := make(chan creditsResult)

	go func() {
		details, err := tmdbClient.GetPersonDetails(personID)
		detailsChan <- detailsResult{details: details, err: err}
	}()

	go func() {
		credits, err := tmdbClient.GetPersonCombinedCredits(personID)
		creditsChan <- creditsResult{credits: credits, err: err}
	}()

	detailsRes := <-detailsChan
	creditsRes := <-creditsChan

	if detailsRes.err != nil {
		http.Error(w, "Failed to fetch person details", http.StatusInternalServerError)
		return
	}

	// Find "Also in your library" - search cast JSON in movies and shows
	alsoInLibrary := s.findPersonInLibrary(personID)

	// Get known for credits sorted by popularity
	knownFor := s.getKnownForCredits(creditsRes.credits, 20)

	// Marshal details to JSON and unmarshal into a map for easy manipulation
	detailsJSON, _ := json.Marshal(detailsRes.details)
	var detailsMap map[string]interface{}
	json.Unmarshal(detailsJSON, &detailsMap)

	// Build response
	response := map[string]interface{}{
		"id":            personID,
		"name":          detailsMap["name"],
		"biography":     detailsMap["biography"],
		"birthday":      detailsMap["birthday"],
		"deathday":      detailsMap["deathday"],
		"placeOfBirth":  detailsMap["place_of_birth"],
		"profilePath":   detailsMap["profile_path"],
		"knownFor":      detailsMap["known_for_department"],
		"credits":       knownFor,
		"alsoInLibrary": alsoInLibrary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// findPersonInLibrary searches cast/crew JSON in movies and shows for a person
func (s *Server) findPersonInLibrary(personID int64) []map[string]interface{} {
	var results []map[string]interface{}

	// Search movies
	movies, err := s.db.GetMovies()
	if err == nil {
		for _, movie := range movies {
			if movie.Cast != nil && containsPersonID(*movie.Cast, personID) {
				var posterPath string
				if movie.PosterPath != nil {
					posterPath = *movie.PosterPath
				}
				results = append(results, map[string]interface{}{
					"id":         movie.ID,
					"type":       "movie",
					"title":      movie.Title,
					"year":       movie.Year,
					"posterPath": posterPath,
				})
			}
		}
	}

	// Search shows
	shows, err := s.db.GetShows()
	if err == nil {
		for _, show := range shows {
			if show.Cast != nil && containsPersonID(*show.Cast, personID) {
				var posterPath string
				if show.PosterPath != nil {
					posterPath = *show.PosterPath
				}
				results = append(results, map[string]interface{}{
					"id":         show.ID,
					"type":       "show",
					"title":      show.Title,
					"year":       show.Year,
					"posterPath": posterPath,
				})
			}
		}
	}

	return results
}

// containsPersonID checks if a cast/crew JSON string contains a person with the given ID
func containsPersonID(castJSON string, personID int64) bool {
	var cast []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(castJSON), &cast); err != nil {
		return false
	}
	for _, c := range cast {
		if c.ID == personID {
			return true
		}
	}
	return false
}

// getKnownForCredits returns sorted credits by popularity, filtered and deduplicated
func (s *Server) getKnownForCredits(creditsInterface interface{}, limit int) []map[string]interface{} {
	if creditsInterface == nil {
		return nil
	}

	// Try to marshal and unmarshal to work with the data
	data, err := json.Marshal(creditsInterface)
	if err != nil {
		return nil
	}

	var credits struct {
		Cast []struct {
			ID           int64   `json:"id"`
			MediaType    string  `json:"media_type"`
			Title        string  `json:"title"`
			Name         string  `json:"name"`
			Character    string  `json:"character"`
			PosterPath   string  `json:"poster_path"`
			ReleaseDate  string  `json:"release_date"`
			FirstAirDate string  `json:"first_air_date"`
			VoteAverage  float64 `json:"vote_average"`
			Popularity   float64 `json:"popularity"`
			GenreIDs     []int   `json:"genre_ids"`
		} `json:"cast"`
	}

	if err := json.Unmarshal(data, &credits); err != nil {
		return nil
	}

	// Filter out talk shows, news, and reality TV (genre IDs: 10767=Talk, 10763=News, 10764=Reality)
	// Also filter out items where the character is "Self" or "Themselves" (talk show appearances)
	excludeGenres := map[int]bool{10767: true, 10763: true, 10764: true}
	var filtered []struct {
		ID           int64
		MediaType    string
		Title        string
		Name         string
		Character    string
		PosterPath   string
		ReleaseDate  string
		FirstAirDate string
		VoteAverage  float64
		Popularity   float64
	}

	for _, c := range credits.Cast {
		// Skip if character is "Self", "Themselves", or similar
		charLower := strings.ToLower(c.Character)
		if charLower == "self" || charLower == "themselves" || charLower == "himself" || charLower == "herself" ||
			strings.HasPrefix(charLower, "self ") || strings.HasPrefix(charLower, "himself ") ||
			strings.HasPrefix(charLower, "herself ") || strings.Contains(charLower, "(uncredited)") {
			continue
		}

		// Skip excluded genres
		skip := false
		for _, gid := range c.GenreIDs {
			if excludeGenres[gid] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		filtered = append(filtered, struct {
			ID           int64
			MediaType    string
			Title        string
			Name         string
			Character    string
			PosterPath   string
			ReleaseDate  string
			FirstAirDate string
			VoteAverage  float64
			Popularity   float64
		}{
			ID:           c.ID,
			MediaType:    c.MediaType,
			Title:        c.Title,
			Name:         c.Name,
			Character:    c.Character,
			PosterPath:   c.PosterPath,
			ReleaseDate:  c.ReleaseDate,
			FirstAirDate: c.FirstAirDate,
			VoteAverage:  c.VoteAverage,
			Popularity:   c.Popularity,
		})
	}

	// Sort by popularity descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Popularity > filtered[j].Popularity
	})

	// Deduplicate by ID (same movie/show appearing multiple times)
	seen := make(map[int64]bool)
	var results []map[string]interface{}

	for _, c := range filtered {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true

		title := c.Title
		if c.MediaType == "tv" {
			title = c.Name
		}
		results = append(results, map[string]interface{}{
			"id":          c.ID,
			"mediaType":   c.MediaType,
			"title":       title,
			"character":   c.Character,
			"posterPath":  c.PosterPath,
			"releaseDate": c.ReleaseDate,
			"voteAverage": c.VoteAverage,
		})

		if len(results) >= limit {
			break
		}
	}

	return results
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*database.User)

	switch r.Method {
	case http.MethodGet:
		// Admin can see all requests, users see their own
		var requests []database.Request
		var err error

		if user.Role == "admin" {
			// Check for status filter
			if status := r.URL.Query().Get("status"); status != "" {
				requests, err = s.db.GetRequestsByStatus(status)
			} else {
				requests, err = s.db.GetRequests()
			}
		} else {
			requests, err = s.db.GetRequestsByUser(user.ID)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if requests == nil {
			requests = []database.Request{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requests)

	case http.MethodPost:
		var req struct {
			Type             string  `json:"type"`
			TmdbID           int64   `json:"tmdbId"`
			Title            string  `json:"title"`
			Year             int     `json:"year"`
			Overview         *string `json:"overview"`
			PosterPath       *string `json:"posterPath"`
			BackdropPath     *string `json:"backdropPath"`
			QualityProfileID *int64  `json:"qualityProfileId"`
			QualityPresetID  *int64  `json:"qualityPresetId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Type == "" || req.TmdbID == 0 || req.Title == "" {
			http.Error(w, "type, tmdbId, and title are required", http.StatusBadRequest)
			return
		}

		// Check if already requested
		existing, _ := s.db.GetRequestByTmdb(user.ID, req.Type, req.TmdbID)
		if existing != nil {
			http.Error(w, "Already requested", http.StatusConflict)
			return
		}

		request := &database.Request{
			UserID:           user.ID,
			Type:             req.Type,
			TmdbID:           req.TmdbID,
			Title:            req.Title,
			Year:             req.Year,
			Overview:         req.Overview,
			PosterPath:       req.PosterPath,
			BackdropPath:     req.BackdropPath,
			QualityProfileID: req.QualityProfileID,
			QualityPresetID:  req.QualityPresetID,
		}

		if err := s.db.CreateRequest(request); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(request)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*database.User)

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/requests/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	request, err := s.db.GetRequest(id)
	if err != nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Users can only see their own requests
		if user.Role != "admin" && request.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(request)

	case http.MethodPut:
		// Only admin can update status
		if user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var updates struct {
			Status          string  `json:"status"`
			StatusReason    *string `json:"statusReason"`
			QualityPresetID *int64  `json:"qualityPresetId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if updates.Status == "" {
			http.Error(w, "status is required", http.StatusBadRequest)
			return
		}

		// Validate status
		validStatuses := map[string]bool{
			"requested": true,
			"approved":  true,
			"denied":    true,
			"available": true,
		}
		if !validStatuses[updates.Status] {
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}

		if err := s.db.UpdateRequestStatus(id, updates.Status, updates.StatusReason); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// If approved, optionally add to wanted list
		if updates.Status == "approved" {
			log.Printf("Request approved: %s (tmdb=%d, type=%s)", request.Title, request.TmdbID, request.Type)
			// Check if not already in wanted
			existing, _ := s.db.GetWantedByTmdb(request.Type, request.TmdbID)
			if existing == nil {
				log.Printf("Adding to wanted list: %s", request.Title)
				// Use quality preset from request, or provided in update, or get default
				var presetID *int64
				if request.QualityPresetID != nil && *request.QualityPresetID > 0 {
					presetID = request.QualityPresetID
				} else if updates.QualityPresetID != nil && *updates.QualityPresetID > 0 {
					presetID = updates.QualityPresetID
				} else {
					// Get default preset
					presets, _ := s.db.GetQualityPresets()
					for _, p := range presets {
						if p.IsDefault && p.Enabled {
							presetID = &p.ID
							break
						}
					}
					// If no default, use first enabled
					if presetID == nil {
						for _, p := range presets {
							if p.Enabled {
								presetID = &p.ID
								break
							}
						}
					}
				}

				wanted := &database.WantedItem{
					Type:            request.Type,
					TmdbID:          request.TmdbID,
					Title:           request.Title,
					Year:            request.Year,
					PosterPath:      request.PosterPath,
					QualityPresetID: presetID,
					Monitored:       true,
				}
				if err := s.db.CreateWantedItem(wanted); err != nil {
					log.Printf("Failed to create wanted item: %v", err)
				}

				// Trigger immediate search for the item
				if s.scheduler != nil {
					log.Printf("Triggering search for: %s", request.Title)
					go s.scheduler.SearchWantedItem(request.TmdbID, request.Type)
				} else {
					log.Printf("Scheduler is nil, cannot trigger search")
				}
			} else {
				log.Printf("Already in wanted list: %s", request.Title)
			}

			// Notify the requesting user that their request was approved
			if s.notifications != nil {
				go s.notifications.NotifyRequestApproved(request.UserID, request.Title, request.TmdbID, request.Type, request.PosterPath)
			}
		} else if updates.Status == "denied" {
			// Notify the requesting user that their request was denied
			if s.notifications != nil {
				reason := ""
				if updates.StatusReason != nil {
					reason = *updates.StatusReason
				}
				go s.notifications.NotifyRequestDenied(request.UserID, request.Title, reason, request.PosterPath)
			}
		}

		// Fetch updated request
		request, _ = s.db.GetRequest(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(request)

	case http.MethodDelete:
		// Users can only delete their own pending requests
		if user.Role != "admin" {
			if request.UserID != user.ID {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if request.Status != "requested" {
				http.Error(w, "Cannot delete processed request", http.StatusForbidden)
				return
			}
		}

		if err := s.db.DeleteRequest(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleClearDeniedRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count, err := s.db.DeleteDeniedRequests()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"deleted": count})
}

// Watchlist handlers

func (s *Server) handleWatchlist(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*database.User)
	profileID := s.getActiveProfileID(r)

	switch r.Method {
	case http.MethodGet:
		items, err := s.db.GetWatchlist(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if items == nil {
			items = []database.WatchlistItem{}
		}

		// Enrich with metadata
		type EnrichedWatchlistItem struct {
			database.WatchlistItem
			Title        string   `json:"title"`
			PosterPath   *string  `json:"posterPath"`
			BackdropPath *string  `json:"backdropPath"`
			Year         int      `json:"year"`
			InLibrary    bool     `json:"inLibrary"`
			LibraryID    *int64   `json:"libraryId,omitempty"`
			Progress     *float64 `json:"progress,omitempty"`
		}

		enriched := make([]EnrichedWatchlistItem, 0, len(items))
		for _, item := range items {
			e := EnrichedWatchlistItem{
				WatchlistItem: item,
			}

			// Check if in library and get metadata
			if item.MediaType == "movie" {
				status, err := s.db.GetMovieStatusByTmdbID(item.TmdbID)
				if err == nil && status.InLibrary && status.LibraryID != nil {
					e.InLibrary = true
					e.LibraryID = status.LibraryID
					// Get movie details
					movie, err := s.db.GetMovie(*status.LibraryID)
					if err == nil {
						e.Title = movie.Title
						e.PosterPath = movie.PosterPath
						e.BackdropPath = movie.BackdropPath
						e.Year = movie.Year
						// Get progress (only if profile selected)
						if profileID != nil {
							progress, err := s.db.GetProgress(*profileID, "movie", *status.LibraryID)
							if err == nil && progress != nil && progress.Duration > 0 {
								pct := (progress.Position / progress.Duration) * 100
								e.Progress = &pct
							}
						}
					}
				} else if s.metadata != nil && s.metadata.GetTMDBClient() != nil {
					// Fetch from TMDB
					details, err := s.metadata.GetTMDBClient().GetMovieDetails(item.TmdbID)
					if err == nil {
						e.Title = details.Title
						if details.PosterPath != "" {
							e.PosterPath = &details.PosterPath
						}
						if details.BackdropPath != "" {
							e.BackdropPath = &details.BackdropPath
						}
						if details.ReleaseDate != "" && len(details.ReleaseDate) >= 4 {
							year, _ := strconv.Atoi(details.ReleaseDate[:4])
							e.Year = year
						}
					}
				}
			} else {
				status, err := s.db.GetShowStatusByTmdbID(item.TmdbID)
				if err == nil && status.InLibrary && status.LibraryID != nil {
					e.InLibrary = true
					e.LibraryID = status.LibraryID
					// Get show details
					show, err := s.db.GetShow(*status.LibraryID)
					if err == nil {
						e.Title = show.Title
						e.PosterPath = show.PosterPath
						e.BackdropPath = show.BackdropPath
						e.Year = show.Year
					}
				} else if s.metadata != nil && s.metadata.GetTMDBClient() != nil {
					// Fetch from TMDB
					details, err := s.metadata.GetTMDBClient().GetTVDetails(item.TmdbID)
					if err == nil {
						e.Title = details.Name
						if details.PosterPath != "" {
							e.PosterPath = &details.PosterPath
						}
						if details.BackdropPath != "" {
							e.BackdropPath = &details.BackdropPath
						}
						if details.FirstAirDate != "" && len(details.FirstAirDate) >= 4 {
							year, _ := strconv.Atoi(details.FirstAirDate[:4])
							e.Year = year
						}
					}
				}
			}

			enriched = append(enriched, e)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(enriched)

	case http.MethodPost:
		var req struct {
			TmdbID    int64  `json:"tmdbId"`
			MediaType string `json:"mediaType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.TmdbID == 0 || (req.MediaType != "movie" && req.MediaType != "tv") {
			http.Error(w, "tmdbId and valid mediaType (movie/tv) are required", http.StatusBadRequest)
			return
		}

		item := &database.WatchlistItem{
			UserID:    user.ID,
			TmdbID:    req.TmdbID,
			MediaType: req.MediaType,
		}

		if err := s.db.AddToWatchlist(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWatchlistItem(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*database.User)

	// Path format: /api/watchlist/{tmdbId}/{mediaType}
	path := strings.TrimPrefix(r.URL.Path, "/api/watchlist/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid path, expected /api/watchlist/{tmdbId}/{mediaType}", http.StatusBadRequest)
		return
	}

	tmdbID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid TMDB ID", http.StatusBadRequest)
		return
	}

	mediaType := parts[1]
	if mediaType != "movie" && mediaType != "tv" {
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Check if in watchlist
		inWatchlist, err := s.db.IsInWatchlist(user.ID, tmdbID, mediaType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"inWatchlist": inWatchlist})

	case http.MethodDelete:
		if err := s.db.RemoveFromWatchlist(user.ID, tmdbID, mediaType); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Music handlers

func (s *Server) handleArtists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists, err := s.db.GetArtists()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if artists == nil {
		artists = []database.Artist{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artists)
}

func (s *Server) handleArtist(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/artists/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artist, err := s.db.GetArtist(id)
	if err != nil {
		http.Error(w, "Artist not found", http.StatusNotFound)
		return
	}

	// Get albums for this artist
	albums, _ := s.db.GetAlbumsByArtist(id)
	if albums == nil {
		albums = []database.Album{}
	}

	response := struct {
		*database.Artist
		Albums []database.Album `json:"albums"`
	}{
		Artist: artist,
		Albums: albums,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAlbums(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	albums, err := s.db.GetAlbums()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if albums == nil {
		albums = []database.Album{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albums)
}

func (s *Server) handleAlbum(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/albums/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	album, err := s.db.GetAlbum(id)
	if err != nil {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	// Get artist info
	artist, _ := s.db.GetArtist(album.ArtistID)

	// Get tracks for this album
	tracks, _ := s.db.GetTracksByAlbum(id)
	if tracks == nil {
		tracks = []database.Track{}
	}

	response := struct {
		*database.Album
		Artist *database.Artist `json:"artist"`
		Tracks []database.Track `json:"tracks"`
	}{
		Album:  album,
		Artist: artist,
		Tracks: tracks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTrack(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/tracks/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	track, err := s.db.GetTrack(id)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

// Book handlers

func (s *Server) handleBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	books, err := s.db.GetBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if books == nil {
		books = []database.Book{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/books/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	book, err := s.db.GetBook(id)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// SubtitleTrack represents a subtitle stream in a media file or external file
type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Codec    string `json:"codec"`
	Default  bool   `json:"default"`
	Forced   bool   `json:"forced"`
	External bool   `json:"external"`
	FilePath string `json:"filePath,omitempty"` // Only set for external subtitles
}

// handleSubtitles handles subtitle listing and extraction
// Routes:
//   GET /api/subtitles/{type}/{id} - List subtitle tracks
//   GET /api/subtitles/{type}/{id}/track/{index} - Get subtitle as WebVTT
func (s *Server) handleSubtitles(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleSubtitles: %s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/subtitles/{type}/{id} or /api/subtitles/{type}/{id}/track/{index}
	path := strings.TrimPrefix(r.URL.Path, "/api/subtitles/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid subtitle path", http.StatusBadRequest)
		return
	}

	mediaType := parts[0]
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Get file path based on media type
	var filePath string
	switch mediaType {
	case "movie":
		movie, err := s.db.GetMovie(id)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		filePath = movie.Path
	case "episode":
		episode, err := s.db.GetEpisode(id)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		filePath = episode.Path
	default:
		http.Error(w, "Invalid media type", http.StatusBadRequest)
		return
	}

	// Check if requesting track extraction or track list
	if len(parts) >= 4 && parts[2] == "track" {
		// Extract specific subtitle track
		trackIndex, err := strconv.Atoi(parts[3])
		if err != nil {
			http.Error(w, "Invalid track index", http.StatusBadRequest)
			return
		}
		s.serveSubtitleTrack(w, r, filePath, trackIndex)
		return
	}

	// List available subtitle tracks
	s.listSubtitleTracks(w, filePath)
}

// listSubtitleTracks uses ffprobe to list all subtitle streams in a file
// and scans for external subtitle files
func (s *Server) listSubtitleTracks(w http.ResponseWriter, filePath string) {
	tracks := []SubtitleTrack{}

	// 1. Get embedded subtitle streams using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		filePath,
	)

	output, err := cmd.Output()
	if err == nil {
		// Parse ffprobe JSON output
		var probeResult struct {
			Streams []struct {
				Index       int               `json:"index"`
				CodecName   string            `json:"codec_name"`
				CodecType   string            `json:"codec_type"`
				Tags        map[string]string `json:"tags"`
				Disposition struct {
					Default int `json:"default"`
					Forced  int `json:"forced"`
				} `json:"disposition"`
			} `json:"streams"`
		}

		if err := json.Unmarshal(output, &probeResult); err == nil {
			// Convert to our subtitle track format
			for i, stream := range probeResult.Streams {
				track := SubtitleTrack{
					Index:    i, // Use sequential index for subtitle streams
					Codec:    stream.CodecName,
					Default:  stream.Disposition.Default == 1,
					Forced:   stream.Disposition.Forced == 1,
					External: false,
				}

				// Get language from tags
				if lang, ok := stream.Tags["language"]; ok {
					track.Language = lang
				}
				// Get title from tags
				if title, ok := stream.Tags["title"]; ok {
					track.Title = title
				}

				tracks = append(tracks, track)
			}
		}
	}

	// 2. Scan for external subtitle files
	externalTracks := s.findExternalSubtitles(filePath, len(tracks))
	tracks = append(tracks, externalTracks...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// findExternalSubtitles scans for subtitle files adjacent to the media file
func (s *Server) findExternalSubtitles(mediaPath string, startIndex int) []SubtitleTrack {
	tracks := []SubtitleTrack{}

	// Get the directory and base name of the media file
	dir := filepath.Dir(mediaPath)
	baseName := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))

	// Common subtitle extensions
	subtitleExts := []string{".srt", ".ass", ".ssa", ".sub", ".vtt"}

	// Language code patterns in filenames
	langPatterns := map[string]string{
		"en": "eng", "eng": "eng", "english": "eng",
		"es": "spa", "spa": "spa", "spanish": "spa",
		"fr": "fre", "fra": "fre", "french": "fre",
		"de": "ger", "deu": "ger", "german": "ger",
		"it": "ita", "italian": "ita",
		"pt": "por", "portuguese": "por",
		"ru": "rus", "russian": "rus",
		"ja": "jpn", "jpn": "jpn", "japanese": "jpn",
		"ko": "kor", "korean": "kor",
		"zh": "chi", "chi": "chi", "chinese": "chi",
		"ar": "ara", "arabic": "ara",
		"hi": "hin", "hindi": "hin",
		"nl": "dut", "dutch": "dut",
		"pl": "pol", "polish": "pol",
		"sv": "swe", "swedish": "swe",
		"da": "dan", "danish": "dan",
		"fi": "fin", "finnish": "fin",
		"no": "nor", "norwegian": "nor",
		"cs": "cze", "czech": "cze",
		"hu": "hun", "hungarian": "hun",
		"el": "gre", "greek": "gre",
		"he": "heb", "hebrew": "heb",
		"th": "tha", "thai": "tha",
		"tr": "tur", "turkish": "tur",
		"vi": "vie", "vietnamese": "vie",
		"id": "ind", "indonesian": "ind",
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return tracks
	}

	externalIndex := startIndex
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Check if it's a subtitle file
		isSubtitle := false
		for _, subExt := range subtitleExts {
			if ext == subExt {
				isSubtitle = true
				break
			}
		}
		if !isSubtitle {
			continue
		}

		// Check if the subtitle file matches the media file
		nameWithoutExt := strings.TrimSuffix(name, ext)

		// Must start with the media file's base name
		if !strings.HasPrefix(strings.ToLower(nameWithoutExt), strings.ToLower(baseName)) {
			continue
		}

		// Parse language from filename
		// Common patterns: movie.en.srt, movie.eng.srt, movie.english.srt
		lang := ""
		title := ""
		suffix := nameWithoutExt[len(baseName):]
		if suffix != "" {
			// Remove leading dot or other separator
			suffix = strings.TrimPrefix(suffix, ".")
			suffix = strings.TrimPrefix(suffix, "_")
			suffix = strings.TrimPrefix(suffix, "-")

			// Check for language code
			parts := strings.Split(strings.ToLower(suffix), ".")
			for _, part := range parts {
				if mapped, ok := langPatterns[part]; ok {
					lang = mapped
				} else if part == "forced" {
					title = "Forced"
				} else if part == "sdh" || part == "cc" {
					title = "SDH"
				} else if part == "hi" && lang == "" {
					// Could be "hearing impaired" or Hindi
					title = "SDH"
				}
			}

			// If no language found, use suffix as title
			if lang == "" && suffix != "" && title == "" {
				title = suffix
			}
		}

		// Determine codec from extension
		codec := ext[1:] // Remove the dot
		if codec == "ass" || codec == "ssa" {
			codec = "ass"
		}

		track := SubtitleTrack{
			Index:    externalIndex,
			Language: lang,
			Title:    title,
			Codec:    codec,
			Default:  false,
			Forced:   title == "Forced",
			External: true,
			FilePath: filepath.Join(dir, name),
		}
		tracks = append(tracks, track)
		externalIndex++
	}

	return tracks
}

// serveSubtitleTrack extracts and serves a subtitle track as WebVTT
func (s *Server) serveSubtitleTrack(w http.ResponseWriter, r *http.Request, filePath string, trackIndex int) {
	log.Printf("serveSubtitleTrack: file=%s, trackIndex=%d", filePath, trackIndex)

	// First, get count of embedded subtitles to determine if this is external
	embeddedCount := s.countEmbeddedSubtitles(filePath)
	log.Printf("serveSubtitleTrack: embeddedCount=%d", embeddedCount)

	if trackIndex >= embeddedCount {
		// This is an external subtitle file
		log.Printf("serveSubtitleTrack: looking for external subtitle")
		externalTracks := s.findExternalSubtitles(filePath, embeddedCount)
		for _, track := range externalTracks {
			if track.Index == trackIndex {
				s.serveExternalSubtitle(w, track.FilePath)
				return
			}
		}
		http.Error(w, "Subtitle track not found", http.StatusNotFound)
		return
	}

	// Check for pre-extracted subtitles in the "subtitles" subfolder next to the video
	videoDir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	baseNameNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	subtitlesDir := filepath.Join(videoDir, "subtitles")

	// Look for VTT files matching pattern: {name}.{index}.{lang}.vtt
	if files, err := os.ReadDir(subtitlesDir); err == nil {
		pattern := fmt.Sprintf("%s.%d.", baseNameNoExt, trackIndex)
		for _, f := range files {
			if strings.HasPrefix(f.Name(), pattern) && strings.HasSuffix(f.Name(), ".vtt") {
				vttPath := filepath.Join(subtitlesDir, f.Name())
				if cached, err := os.ReadFile(vttPath); err == nil {
					log.Printf("serveSubtitleTrack: found pre-extracted subtitle: %s", f.Name())
					w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
					w.Header().Set("Cache-Control", "max-age=86400")
					w.Write(cached)
					return
				}
			}
		}
	}

	// Fallback: check central cache directory
	cacheDir := filepath.Join(filepath.Dir(s.config.DBPath), "subtitles")
	os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s.track%d.vtt", baseName, trackIndex))

	// Check disk cache
	if cached, err := os.ReadFile(cacheFile); err == nil {
		log.Printf("serveSubtitleTrack: disk cache hit, returning %d bytes", len(cached))
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(cached)
		return
	}

	// Check memory cache
	cacheKey := fmt.Sprintf("%s:%d", filePath, trackIndex)
	s.subtitleMu.RLock()
	cached, found := s.subtitleCache[cacheKey]
	s.subtitleMu.RUnlock()

	if found {
		log.Printf("serveSubtitleTrack: memory cache hit, returning %d bytes", len(cached))
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(cached)
		return
	}

	// Embedded subtitle - extract and convert to WebVTT
	log.Printf("serveSubtitleTrack: cache miss, extracting from: %s (this may take 1-2 minutes for large files)", filePath)

	var output []byte
	var err error

	// Use mkvextract for MKV files, otherwise use ffmpeg
	if strings.HasSuffix(strings.ToLower(filePath), ".mkv") {
		output, err = s.extractSubtitleMKV(filePath, trackIndex)
	} else {
		output, err = s.extractSubtitleFFmpeg(filePath, trackIndex)
	}

	if err != nil {
		log.Printf("serveSubtitleTrack: extraction error: %v", err)
		http.Error(w, "Failed to extract subtitle", http.StatusInternalServerError)
		return
	}
	log.Printf("serveSubtitleTrack: extracted %d bytes, caching to disk", len(output))

	// Save to disk cache (persistent)
	if err := os.WriteFile(cacheFile, output, 0644); err != nil {
		log.Printf("serveSubtitleTrack: failed to save to disk cache: %v", err)
	}

	// Also cache in memory for current session
	s.subtitleMu.Lock()
	s.subtitleCache[cacheKey] = output
	s.subtitleMu.Unlock()

	// Set headers for WebVTT
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=86400") // Cache for 24 hours
	w.Write(output)
}

// extractSubtitleMKV uses mkvextract for fast MKV subtitle extraction
func (s *Server) extractSubtitleMKV(filePath string, trackIndex int) ([]byte, error) {
	// First get the actual track ID from ffprobe (subtitle stream index -> MKV track ID)
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		filePath,
	)
	probeOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeResult struct {
		Streams []struct {
			Index int `json:"index"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(probeOutput, &probeResult); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if trackIndex >= len(probeResult.Streams) {
		return nil, fmt.Errorf("track index out of range")
	}

	// Get the actual stream index in the file
	actualIndex := probeResult.Streams[trackIndex].Index

	// Create temp file for extracted subtitle
	tmpFile, err := os.CreateTemp("", "subtitle-*.srt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Extract with mkvextract
	cmd = exec.Command("mkvextract", "tracks", filePath, fmt.Sprintf("%d:%s", actualIndex, tmpPath))
	if err := cmd.Run(); err != nil {
		// Fall back to ffmpeg if mkvextract fails
		log.Printf("mkvextract failed, falling back to ffmpeg: %v", err)
		return s.extractSubtitleFFmpeg(filePath, trackIndex)
	}

	// Read extracted subtitle
	srtData, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted subtitle: %w", err)
	}

	// Convert SRT to VTT
	return s.srtToVtt(srtData), nil
}

// srtToVtt converts SRT subtitle format to WebVTT
func (s *Server) srtToVtt(srt []byte) []byte {
	content := string(srt)

	// Replace SRT timestamp format (00:00:00,000) with VTT format (00:00:00.000)
	content = strings.ReplaceAll(content, ",", ".")

	// Add VTT header
	vtt := "WEBVTT\n\n" + content

	return []byte(vtt)
}

// extractSubtitleFFmpeg uses ffmpeg for subtitle extraction (fallback)
func (s *Server) extractSubtitleFFmpeg(filePath string, trackIndex int) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "subtitle-*.vtt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("ffmpeg",
		"-y",
		"-v", "error",
		"-i", filePath,
		"-map", fmt.Sprintf("0:s:%d", trackIndex),
		"-f", "webvtt",
		tmpPath,
	)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffmpeg error: %v, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffmpeg error: %w", err)
	}

	return os.ReadFile(tmpPath)
}

// countEmbeddedSubtitles returns the number of embedded subtitle streams in a file
func (s *Server) countEmbeddedSubtitles(filePath string) int {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	var probeResult struct {
		Streams []struct{} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return 0
	}

	return len(probeResult.Streams)
}

// serveExternalSubtitle reads and serves an external subtitle file, converting to WebVTT if needed
func (s *Server) serveExternalSubtitle(w http.ResponseWriter, subPath string) {
	ext := strings.ToLower(filepath.Ext(subPath))

	// If it's already VTT, serve directly
	if ext == ".vtt" {
		content, err := os.ReadFile(subPath)
		if err != nil {
			http.Error(w, "Failed to read subtitle file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Write(content)
		return
	}

	// Convert SRT, ASS, SSA to WebVTT using FFmpeg
	cmd := exec.Command("ffmpeg",
		"-i", subPath,
		"-f", "webvtt",
		"-",
	)

	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to convert subtitle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Write(output)
}

// Quality Preset handlers

func (s *Server) handleQualityPresets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		presets, err := s.db.GetQualityPresets()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(presets)

	case http.MethodPost:
		// POST requires admin
		user := r.Context().Value(userContextKey).(*database.User)
		if user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var preset database.QualityPreset
		if err := json.NewDecoder(r.Body).Decode(&preset); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.CreateQualityPreset(&preset); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(preset)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleQualityPreset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/quality/presets/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Preset ID required", http.StatusBadRequest)
		return
	}

	idStr := parts[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid preset ID", http.StatusBadRequest)
		return
	}

	// Check for /api/quality/presets/:id/default
	if len(parts) > 1 && parts[1] == "default" && r.Method == http.MethodPost {
		if err := s.db.SetDefaultQualityPreset(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// Check for /api/quality/presets/:id/toggle - toggle enabled status
	if len(parts) > 1 && parts[1] == "toggle" && r.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.ToggleQualityPresetEnabled(id, req.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "enabled": req.Enabled})
		return
	}

	// Check for /api/quality/presets/:id/priority - update priority
	if len(parts) > 1 && parts[1] == "priority" && r.Method == http.MethodPost {
		var req struct {
			Priority int `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateQualityPresetPriority(id, req.Priority); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "priority": req.Priority})
		return
	}

	// Check for /api/quality/presets/:id/anime-preferences - update anime-specific preferences
	if len(parts) > 1 && parts[1] == "anime-preferences" && r.Method == http.MethodPatch {
		var req struct {
			PreferDualAudio   *bool   `json:"preferDualAudio,omitempty"`
			PreferDubbed      *bool   `json:"preferDubbed,omitempty"`
			PreferredLanguage *string `json:"preferredLanguage,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateQualityPresetAnimePreferences(id, req.PreferDualAudio, req.PreferDubbed, req.PreferredLanguage); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		preset, err := s.db.GetQualityPreset(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(preset)

	case http.MethodPut:
		var preset database.QualityPreset
		if err := json.NewDecoder(r.Body).Decode(&preset); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		preset.ID = id
		if err := s.db.UpdateQualityPreset(&preset); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(preset)

	case http.MethodDelete:
		if err := s.db.DeleteQualityPreset(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Import History handler

func (s *Server) handleImportHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := s.db.GetImportHistory(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(history)
}

// Download Items handlers (database-tracked downloads for import)

func (s *Server) handleDownloadItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	downloads, err := s.acquisition.GetActiveDownloads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if downloads == nil {
		downloads = []*download.TrackedDownload{}
	}

	// Enrich downloads with poster info from wanted items
	for _, dl := range downloads {
		if dl.MediaID != nil {
			wanted, _ := s.db.GetWantedByTmdb(dl.MediaType, *dl.MediaID)
			if wanted != nil {
				dl.TmdbID = wanted.TmdbID
				if wanted.PosterPath != nil {
					dl.PosterPath = *wanted.PosterPath
				}
				dl.Year = wanted.Year
			}
		}
	}

	json.NewEncoder(w).Encode(downloads)
}

func (s *Server) handleDownloadItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL
	idStr := strings.TrimPrefix(r.URL.Path, "/api/download-items/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid download ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		// Parse query parameters for delete options
		deleteFromClient := r.URL.Query().Get("deleteFromClient") != "false"
		deleteFiles := r.URL.Query().Get("deleteFiles") == "true"

		if err := s.acquisition.DeleteTrackedDownload(id, deleteFromClient, deleteFiles); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Naming Templates handler

func (s *Server) handleNamingTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		templates, err := s.db.GetNamingTemplates()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(templates)

	case http.MethodPut:
		var template database.NamingTemplate
		if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateNamingTemplate(&template); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(template)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Storage Status handler

// calculateDirSize walks a directory and sums all file sizes
func calculateDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func (s *Server) handleStorageStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get storage settings
	settings, err := s.db.GetAllSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thresholdGB := int64(100)
	if val, ok := settings["storage_threshold_gb"]; ok {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			thresholdGB = parsed
		}
	}

	pauseEnabled := true
	if val, ok := settings["storage_pause_enabled"]; ok {
		pauseEnabled = val == "true"
	}

	upgradeDeleteOld := true
	if val, ok := settings["upgrade_delete_old"]; ok {
		upgradeDeleteOld = val == "true"
	}

	// Get libraries and calculate sizes by scanning folders
	libraries, err := s.db.GetLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var moviesSize, tvSize, musicSize, booksSize int64
	for _, lib := range libraries {
		size := calculateDirSize(lib.Path)
		switch lib.Type {
		case "movies":
			moviesSize += size
		case "tv", "anime":
			tvSize += size
		case "music":
			musicSize += size
		case "books":
			booksSize += size
		}
	}

	// Get overall disk usage - try common mount points
	var diskUsage *storage.DiskUsage
	for _, checkPath := range []string{"/media", "/app/data", "/"} {
		usage, err := storage.GetDiskUsage(checkPath)
		if err == nil {
			diskUsage = usage
			break
		}
	}

	// Return simplified response
	response := struct {
		ThresholdGB      int64              `json:"thresholdGb"`
		PauseEnabled     bool               `json:"pauseEnabled"`
		UpgradeDeleteOld bool               `json:"upgradeDeleteOld"`
		MoviesSize       int64              `json:"moviesSize"`
		TvSize           int64              `json:"tvSize"`
		MusicSize        int64              `json:"musicSize"`
		BooksSize        int64              `json:"booksSize"`
		DiskUsage        *storage.DiskUsage `json:"diskUsage,omitempty"`
	}{
		ThresholdGB:      thresholdGB,
		PauseEnabled:     pauseEnabled,
		UpgradeDeleteOld: upgradeDeleteOld,
		MoviesSize:       moviesSize,
		TvSize:           tvSize,
		MusicSize:        musicSize,
		BooksSize:        booksSize,
		DiskUsage:        diskUsage,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStorageAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get disk usage
	var diskTotal, diskUsed, diskFree int64
	for _, checkPath := range []string{"/media", "/app/data", "/"} {
		usage, err := storage.GetDiskUsage(checkPath)
		if err == nil {
			diskTotal = int64(usage.Total)
			diskUsed = int64(usage.Used)
			diskFree = int64(usage.Free)
			break
		}
	}

	// Get storage by library
	byLibrary, err := s.db.GetStorageByLibrary()
	if err != nil {
		log.Printf("Error getting storage by library: %v", err)
		byLibrary = []database.LibrarySize{}
	}

	// Get storage by quality
	byQuality, err := s.db.GetStorageByQuality()
	if err != nil {
		log.Printf("Error getting storage by quality: %v", err)
		byQuality = []database.QualitySize{}
	}

	// Get storage by year
	byYear, err := s.db.GetStorageByYear()
	if err != nil {
		log.Printf("Error getting storage by year: %v", err)
		byYear = []database.YearSize{}
	}

	// Get largest items
	largest, err := s.db.GetLargestItems(20)
	if err != nil {
		log.Printf("Error getting largest items: %v", err)
		largest = []database.LargestItem{}
	}

	// Get duplicates
	duplicates, err := s.db.GetMovieDuplicates()
	if err != nil {
		log.Printf("Error getting duplicates: %v", err)
		duplicates = []database.DuplicateItem{}
	}

	response := database.StorageAnalytics{
		Total:      diskTotal,
		Used:       diskUsed,
		Free:       diskFree,
		ByLibrary:  byLibrary,
		ByQuality:  byQuality,
		ByYear:     byYear,
		Largest:    largest,
		Duplicates: duplicates,
	}

	json.NewEncoder(w).Encode(response)
}

// Blocklist handlers

func (s *Server) handleBlocklist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		entries, err := s.db.GetBlocklist()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(entries)

	case http.MethodPost:
		var entry database.BlocklistEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.AddToBlocklist(&entry); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(entry)

	case http.MethodDelete:
		// Clear all or expired
		if r.URL.Query().Get("expired") == "true" {
			if err := s.db.ClearExpiredBlocklist(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBlocklistItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/blocklist/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.db.RemoveFromBlocklist(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Grab history handler

func (s *Server) handleGrabHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := s.db.GetGrabHistory(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(history)
}

// Blocked groups handlers

func (s *Server) handleBlockedGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		groups, err := s.db.GetBlockedGroups()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(groups)

	case http.MethodPost:
		var req struct {
			Name   string `json:"name"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.AddBlockedGroup(req.Name, req.Reason, false); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBlockedGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/blocked-groups/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.db.RemoveBlockedGroup(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Release filters handlers

func (s *Server) handleReleaseFilters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	presetIDStr := r.URL.Query().Get("preset_id")
	if presetIDStr == "" {
		http.Error(w, "preset_id is required", http.StatusBadRequest)
		return
	}
	presetID, err := strconv.ParseInt(presetIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid preset_id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		filters, err := s.db.GetReleaseFilters(presetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(filters)

	case http.MethodPost:
		var filter database.ReleaseFilter
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		filter.PresetID = presetID
		if err := s.db.AddReleaseFilter(&filter); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(filter)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleReleaseFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/release-filters/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.db.RemoveReleaseFilter(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Delay profiles handlers

func (s *Server) handleDelayProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profiles, err := s.db.GetDelayProfiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profiles)

	case http.MethodPost:
		var profile database.DelayProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.CreateDelayProfile(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(profile)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDelayProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/delay-profiles/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var profile database.DelayProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		profile.ID = id
		if err := s.db.UpdateDelayProfile(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(profile)

	case http.MethodDelete:
		if err := s.db.DeleteDelayProfile(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Exclusions handlers

func (s *Server) handleExclusions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		exclusionType := r.URL.Query().Get("type")
		var exclusions []database.Exclusion
		var err error
		if exclusionType != "" {
			exclusions, err = s.db.GetExclusionsByType(exclusionType)
		} else {
			exclusions, err = s.db.GetExclusions()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(exclusions)

	case http.MethodPost:
		var exclusion database.Exclusion
		if err := json.NewDecoder(r.Body).Decode(&exclusion); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.db.AddExclusion(&exclusion); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(exclusion)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleExclusion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/exclusions/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.db.RemoveExclusion(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Movie quality status handlers

func (s *Server) handleMovieQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/movies/quality/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get quality status
		status, err := s.db.GetMediaQualityStatus(id, "movie")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Get quality override
		override, err := s.db.GetMediaQualityOverride(id, "movie")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := struct {
			Status   *database.MediaQualityStatus   `json:"status"`
			Override *database.MediaQualityOverride `json:"override"`
		}{
			Status:   status,
			Override: override,
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		var override database.MediaQualityOverride
		if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		override.MediaID = id
		override.MediaType = "movie"
		if err := s.db.SetMediaQualityOverride(&override); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(override)

	case http.MethodDelete:
		if err := s.db.DeleteMediaQualityOverride(id, "movie"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Show quality status handlers

func (s *Server) handleShowQuality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/shows/quality/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get quality status
		status, err := s.db.GetMediaQualityStatus(id, "show")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Get quality override
		override, err := s.db.GetMediaQualityOverride(id, "show")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := struct {
			Status   *database.MediaQualityStatus   `json:"status"`
			Override *database.MediaQualityOverride `json:"override"`
		}{
			Status:   status,
			Override: override,
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		var override database.MediaQualityOverride
		if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		override.MediaID = id
		override.MediaType = "show"
		if err := s.db.SetMediaQualityOverride(&override); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(override)

	case http.MethodDelete:
		if err := s.db.DeleteMediaQualityOverride(id, "show"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ==================== Task/Scheduler Handlers ====================

// handleTasks returns all scheduled tasks
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	tasks := s.scheduler.GetStatus()
	json.NewEncoder(w).Encode(tasks)
}

// handleTaskHistory returns recent task execution history
func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	w.Header().Set("Content-Type", "application/json")
	history, err := s.db.GetAllTaskHistory(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(history)
}

// handleTask handles individual task operations
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	w.Header().Set("Content-Type", "application/json")

	// Check for /trigger suffix
	if strings.HasSuffix(path, "/trigger") {
		idStr := strings.TrimSuffix(path, "/trigger")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid task ID", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPost {
			if err := s.scheduler.TriggerTask(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "triggered"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		task, err := s.db.GetTask(id)
		if err != nil {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		history, _ := s.db.GetTaskHistory(id, 10)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"task":    task,
			"history": history,
		})

	case http.MethodPut:
		var req struct {
			Enabled         bool `json:"enabled"`
			IntervalMinutes int  `json:"intervalMinutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.scheduler.UpdateTask(id, req.Enabled, req.IntervalMinutes); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		task, _ := s.db.GetTask(id)
		json.NewEncoder(w).Encode(task)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSystemStatus returns current system status for UI indicators
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pending requests count (status is "requested" not "pending")
	requests, _ := s.db.GetRequestsByStatus("requested")
	pendingRequests := len(requests)

	// Get active downloads
	downloads, _ := s.db.GetDownloads()
	activeDownloads := 0
	for _, d := range downloads {
		if d.State == "downloading" || d.State == "pending" {
			activeDownloads++
		}
	}

	// Get running tasks (includes dynamic names like "Searching: Movie Title")
	tasks := s.scheduler.GetStatus()
	runningTasks := []string{}
	for _, t := range tasks {
		if t.IsRunning {
			runningTasks = append(runningTasks, t.Name)
		}
	}
	// Add dynamic task names (e.g., "Searching: Movie Title")
	dynamicTasks := s.scheduler.GetRunningTaskNames()
	for _, name := range dynamicTasks {
		// Avoid duplicates from scheduled tasks
		found := false
		for _, existing := range runningTasks {
			if existing == name {
				found = true
				break
			}
		}
		if !found {
			runningTasks = append(runningTasks, name)
		}
	}

	// Get active search title
	activeSearch := s.scheduler.GetActiveSearch()

	// Get disk usage
	var diskUsed, diskTotal int64
	for _, checkPath := range []string{"/media", "/app/data", "/"} {
		usage, err := storage.GetDiskUsage(checkPath)
		if err == nil {
			diskUsed = int64(usage.Used)
			diskTotal = int64(usage.Total)
			break
		}
	}

	response := struct {
		PendingRequests int      `json:"pendingRequests"`
		ActiveDownloads int      `json:"activeDownloads"`
		RunningTasks    []string `json:"runningTasks"`
		ActiveSearch    string   `json:"activeSearch"`
		DiskUsed        int64    `json:"diskUsed"`
		DiskTotal       int64    `json:"diskTotal"`
	}{
		PendingRequests: pendingRequests,
		ActiveDownloads: activeDownloads,
		RunningTasks:    runningTasks,
		ActiveSearch:    activeSearch,
		DiskUsed:        diskUsed,
		DiskTotal:       diskTotal,
	}

	json.NewEncoder(w).Encode(response)
}

// handleFilesystemBrowse returns directory contents for the file browser
func (s *Server) handleFilesystemBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		// Default to root or common media paths
		path = "/"
	}

	// Clean and validate the path
	path = filepath.Clean(path)

	// Check if path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Path does not exist", http.StatusNotFound)
			return
		}
		http.Error(w, "Cannot access path", http.StatusForbidden)
		return
	}
	if !info.IsDir() {
		http.Error(w, "Path is not a directory", http.StatusBadRequest)
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, "Cannot read directory", http.StatusForbidden)
		return
	}

	type DirEntry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	}

	var dirs []DirEntry
	for _, entry := range entries {
		// Only include directories, skip hidden files
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, DirEntry{
				Name:  entry.Name(),
				Path:  filepath.Join(path, entry.Name()),
				IsDir: true,
			})
		}
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})

	// Get parent directory
	parent := filepath.Dir(path)
	if parent == path {
		parent = "" // We're at root
	}

	fsResponse := struct {
		Current string     `json:"current"`
		Parent  string     `json:"parent"`
		Dirs    []DirEntry `json:"dirs"`
	}{
		Current: path,
		Parent:  parent,
		Dirs:    dirs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsResponse)
}

// Calendar types and handler

type CalendarItem struct {
	Date       string  `json:"date"`       // YYYY-MM-DD
	Type       string  `json:"type"`       // episode, movie
	Title      string  `json:"title"`      // Show name or movie title
	Subtitle   string  `json:"subtitle"`   // "S02E05 - Episode Title" or "Theatrical Release"
	TmdbID     int64   `json:"tmdbId"`
	MediaID    *int64  `json:"mediaId"`    // Library ID if in library, null otherwise
	PosterPath *string `json:"posterPath"`
	InLibrary  bool    `json:"inLibrary"`
	IsWanted   bool    `json:"isWanted"`
	AirTime    string  `json:"airTime,omitempty"` // Optional time if known
}

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	filter := r.URL.Query().Get("filter") // all, movies, tv, library, wanted

	if filter == "" {
		filter = "all"
	}

	// Parse dates, default to current month if not provided
	now := time.Now()
	var startDate, endDate time.Time
	var err error

	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			http.Error(w, "Invalid start date format", http.StatusBadRequest)
			return
		}
	} else {
		// Default to first day of current month
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if endStr != "" {
		endDate, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			http.Error(w, "Invalid end date format", http.StatusBadRequest)
			return
		}
	} else {
		// Default to last day of current month
		endDate = startDate.AddDate(0, 1, -1)
	}

	var items []CalendarItem

	// Get TMDB client
	tmdbClient := s.metadata.GetTMDBClient()

	// Get library shows and their upcoming episodes
	if filter == "all" || filter == "tv" || filter == "library" {
		shows, err := s.db.GetShows()
		if err == nil {
			for _, show := range shows {
				if show.TmdbID == nil {
					continue
				}

				// Get TV details to find current/upcoming seasons
				tvDetails, err := tmdbClient.GetTVDetails(*show.TmdbID)
				if err != nil {
					continue
				}

				// Check each season for episodes in date range
				for _, seasonInfo := range tvDetails.Seasons {
					if seasonInfo.SeasonNumber == 0 {
						continue // Skip specials
					}

					seasonDetails, err := tmdbClient.GetSeasonDetails(*show.TmdbID, seasonInfo.SeasonNumber)
					if err != nil {
						continue
					}

					for _, ep := range seasonDetails.Episodes {
						if ep.AirDate == "" {
							continue
						}

						epDate, err := time.Parse("2006-01-02", ep.AirDate)
						if err != nil {
							continue
						}

						// Check if episode is within date range
						if epDate.Before(startDate) || epDate.After(endDate) {
							continue
						}

						showID := show.ID
						items = append(items, CalendarItem{
							Date:       ep.AirDate,
							Type:       "episode",
							Title:      show.Title,
							Subtitle:   fmt.Sprintf("S%02dE%02d - %s", seasonInfo.SeasonNumber, ep.EpisodeNumber, ep.Name),
							TmdbID:     *show.TmdbID,
							MediaID:    &showID,
							PosterPath: show.PosterPath,
							InLibrary:  true,
							IsWanted:   false,
						})
					}
				}
			}
		}
	}

	// Get wanted items with release dates
	if filter == "all" || filter == "movies" || filter == "wanted" {
		wantedItems, err := s.db.GetWantedItems()
		if err == nil {
			for _, item := range wantedItems {
				if item.Type == "movie" {
					// Get movie details for release dates
					movieDetails, err := tmdbClient.GetMovieDetails(item.TmdbID)
					if err != nil {
						continue
					}

					// Get US release dates
					theatrical, digital := tmdb.GetUSReleaseDates(movieDetails.ReleaseDates)

					// Add theatrical release if in range
					if theatrical != "" {
						// Parse the theatrical date (it comes with time)
						theatricalDate, err := time.Parse("2006-01-02T15:04:05.000Z", theatrical)
						if err != nil {
							theatricalDate, err = time.Parse("2006-01-02", theatrical[:10])
						}
						if err == nil && !theatricalDate.Before(startDate) && !theatricalDate.After(endDate) {
							items = append(items, CalendarItem{
								Date:       theatricalDate.Format("2006-01-02"),
								Type:       "movie",
								Title:      item.Title,
								Subtitle:   "Theatrical Release",
								TmdbID:     item.TmdbID,
								MediaID:    nil,
								PosterPath: item.PosterPath,
								InLibrary:  false,
								IsWanted:   true,
							})
						}
					}

					// Add digital release if in range
					if digital != "" {
						digitalDate, err := time.Parse("2006-01-02T15:04:05.000Z", digital)
						if err != nil {
							digitalDate, err = time.Parse("2006-01-02", digital[:10])
						}
						if err == nil && !digitalDate.Before(startDate) && !digitalDate.After(endDate) {
							items = append(items, CalendarItem{
								Date:       digitalDate.Format("2006-01-02"),
								Type:       "movie",
								Title:      item.Title,
								Subtitle:   "Digital Release",
								TmdbID:     item.TmdbID,
								MediaID:    nil,
								PosterPath: item.PosterPath,
								InLibrary:  false,
								IsWanted:   true,
							})
						}
					}

					// If no US dates, use general release date
					if theatrical == "" && digital == "" && movieDetails.ReleaseDate != "" {
						releaseDate, err := time.Parse("2006-01-02", movieDetails.ReleaseDate)
						if err == nil && !releaseDate.Before(startDate) && !releaseDate.After(endDate) {
							items = append(items, CalendarItem{
								Date:       movieDetails.ReleaseDate,
								Type:       "movie",
								Title:      item.Title,
								Subtitle:   "Release",
								TmdbID:     item.TmdbID,
								MediaID:    nil,
								PosterPath: item.PosterPath,
								InLibrary:  false,
								IsWanted:   true,
							})
						}
					}
				} else if item.Type == "show" {
					// Get upcoming episodes for wanted shows
					tvDetails, err := tmdbClient.GetTVDetails(item.TmdbID)
					if err != nil {
						continue
					}

					for _, seasonInfo := range tvDetails.Seasons {
						if seasonInfo.SeasonNumber == 0 {
							continue
						}

						seasonDetails, err := tmdbClient.GetSeasonDetails(item.TmdbID, seasonInfo.SeasonNumber)
						if err != nil {
							continue
						}

						for _, ep := range seasonDetails.Episodes {
							if ep.AirDate == "" {
								continue
							}

							epDate, err := time.Parse("2006-01-02", ep.AirDate)
							if err != nil {
								continue
							}

							if epDate.Before(startDate) || epDate.After(endDate) {
								continue
							}

							items = append(items, CalendarItem{
								Date:       ep.AirDate,
								Type:       "episode",
								Title:      item.Title,
								Subtitle:   fmt.Sprintf("S%02dE%02d - %s", seasonInfo.SeasonNumber, ep.EpisodeNumber, ep.Name),
								TmdbID:     item.TmdbID,
								MediaID:    nil,
								PosterPath: item.PosterPath,
								InLibrary:  false,
								IsWanted:   true,
							})
						}
					}
				}
			}
		}
	}

	// Sort items by date
	sort.Slice(items, func(i, j int) bool {
		return items[i].Date < items[j].Date
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// Notification handlers

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	// Parse query params
	unreadOnly := r.URL.Query().Get("unread") == "true"
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	notifications, err := s.notifications.GetForUser(user.ID, unreadOnly, limit)
	if err != nil {
		http.Error(w, "Failed to get notifications", http.StatusInternalServerError)
		return
	}

	if notifications == nil {
		notifications = []database.Notification{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (s *Server) handleNotificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	count, err := s.notifications.GetUnreadCount(user.ID)
	if err != nil {
		http.Error(w, "Failed to get unread count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func (s *Server) handleNotificationReadAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	if err := s.notifications.MarkAllRead(user.ID); err != nil {
		http.Error(w, "Failed to mark all as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNotification(w http.ResponseWriter, r *http.Request) {
	// Extract notification ID from path: /api/notifications/{id} or /api/notifications/{id}/read
	path := strings.TrimPrefix(r.URL.Path, "/api/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Notification ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	// Check if this is a /read sub-route
	if len(parts) > 1 && parts[1] == "read" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := s.notifications.MarkRead(id); err != nil {
			http.Error(w, "Failed to mark as read", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Handle DELETE
	if r.Method == http.MethodDelete {
		if err := s.notifications.Delete(id); err != nil {
			http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Missing episodes types and handlers

type MissingEpisode struct {
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	AirDate       string `json:"airDate"`
	Overview      string `json:"overview"`
	StillPath     string `json:"stillPath"`
}

type SeasonMissingSummary struct {
	Season  int `json:"season"`
	Missing int `json:"missing"`
	Total   int `json:"total"`
}

type MissingEpisodesResult struct {
	TotalEpisodes   int                    `json:"totalEpisodes"`
	OwnedEpisodes   int                    `json:"ownedEpisodes"`
	Missing         []MissingEpisode       `json:"missing"`
	MissingBySeason []SeasonMissingSummary `json:"missingBySeason"`
}

func (s *Server) handleMissingEpisodes(w http.ResponseWriter, r *http.Request, show *database.Show) {
	// Show must have a TMDB ID to check for missing episodes
	if show.TmdbID == nil {
		json.NewEncoder(w).Encode(MissingEpisodesResult{
			Missing:         []MissingEpisode{},
			MissingBySeason: []SeasonMissingSummary{},
		})
		return
	}

	// Get TMDB API key
	apiKey, _ := s.db.GetSetting("tmdb_api_key")
	if apiKey == "" {
		http.Error(w, "TMDB API key not configured", http.StatusInternalServerError)
		return
	}

	tmdbClient := tmdb.NewClient(apiKey, "")

	// Get show details from TMDB including seasons
	tvDetails, err := tmdbClient.GetTVDetails(*show.TmdbID)
	if err != nil {
		http.Error(w, "Failed to fetch show from TMDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get owned episodes from database
	ownedEpisodes, err := s.db.GetOwnedEpisodesByShow(show.ID)
	if err != nil {
		http.Error(w, "Failed to get owned episodes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a set of owned episodes for quick lookup
	ownedSet := make(map[string]bool)
	for _, ep := range ownedEpisodes {
		key := fmt.Sprintf("%d-%d", ep.SeasonNumber, ep.EpisodeNumber)
		ownedSet[key] = true
	}

	var missing []MissingEpisode
	var missingBySeason []SeasonMissingSummary
	totalEpisodes := 0

	// Iterate through each season (skip season 0 - specials)
	for _, seasonInfo := range tvDetails.Seasons {
		if seasonInfo.SeasonNumber == 0 {
			continue // Skip specials
		}

		// Fetch detailed episode info for this season
		seasonDetails, err := tmdbClient.GetSeasonDetails(*show.TmdbID, seasonInfo.SeasonNumber)
		if err != nil {
			log.Printf("Failed to fetch season %d details: %v", seasonInfo.SeasonNumber, err)
			continue
		}

		seasonMissing := 0
		seasonTotal := len(seasonDetails.Episodes)
		totalEpisodes += seasonTotal

		for _, ep := range seasonDetails.Episodes {
			key := fmt.Sprintf("%d-%d", seasonInfo.SeasonNumber, ep.EpisodeNumber)
			if !ownedSet[key] {
				seasonMissing++
				missing = append(missing, MissingEpisode{
					SeasonNumber:  seasonInfo.SeasonNumber,
					EpisodeNumber: ep.EpisodeNumber,
					Title:         ep.Name,
					AirDate:       ep.AirDate,
					Overview:      ep.Overview,
					StillPath:     ep.StillPath,
				})
			}
		}

		missingBySeason = append(missingBySeason, SeasonMissingSummary{
			Season:  seasonInfo.SeasonNumber,
			Missing: seasonMissing,
			Total:   seasonTotal,
		})
	}

	if missing == nil {
		missing = []MissingEpisode{}
	}
	if missingBySeason == nil {
		missingBySeason = []SeasonMissingSummary{}
	}

	result := MissingEpisodesResult{
		TotalEpisodes:   totalEpisodes,
		OwnedEpisodes:   len(ownedEpisodes),
		Missing:         missing,
		MissingBySeason: missingBySeason,
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleRequestMissingEpisodes(w http.ResponseWriter, r *http.Request, show *database.Show) {
	// Parse request body for optional season filter
	var req struct {
		SeasonNumber *int `json:"seasonNumber"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Show must have a TMDB ID
	if show.TmdbID == nil {
		http.Error(w, "Show has no TMDB ID", http.StatusBadRequest)
		return
	}

	// Get TMDB API key
	apiKey, _ := s.db.GetSetting("tmdb_api_key")
	if apiKey == "" {
		http.Error(w, "TMDB API key not configured", http.StatusInternalServerError)
		return
	}

	tmdbClient := tmdb.NewClient(apiKey, "")

	// Get show details from TMDB
	tvDetails, err := tmdbClient.GetTVDetails(*show.TmdbID)
	if err != nil {
		http.Error(w, "Failed to fetch show from TMDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get owned episodes
	ownedEpisodes, err := s.db.GetOwnedEpisodesByShow(show.ID)
	if err != nil {
		http.Error(w, "Failed to get owned episodes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ownedSet := make(map[string]bool)
	for _, ep := range ownedEpisodes {
		key := fmt.Sprintf("%d-%d", ep.SeasonNumber, ep.EpisodeNumber)
		ownedSet[key] = true
	}

	// Check if show already has a wanted item
	existingWanted, _ := s.db.GetWantedByTmdb("show", *show.TmdbID)

	addedCount := 0

	// Find missing episodes and track which seasons have missing episodes
	missingSeasonsMap := make(map[int]bool)
	for _, seasonInfo := range tvDetails.Seasons {
		if seasonInfo.SeasonNumber == 0 {
			continue
		}

		// If season filter is specified, skip non-matching seasons
		if req.SeasonNumber != nil && seasonInfo.SeasonNumber != *req.SeasonNumber {
			continue
		}

		seasonDetails, err := tmdbClient.GetSeasonDetails(*show.TmdbID, seasonInfo.SeasonNumber)
		if err != nil {
			continue
		}

		for _, ep := range seasonDetails.Episodes {
			key := fmt.Sprintf("%d-%d", seasonInfo.SeasonNumber, ep.EpisodeNumber)
			if !ownedSet[key] {
				addedCount++
				missingSeasonsMap[seasonInfo.SeasonNumber] = true
			}
		}
	}

	// Build sorted list of missing seasons
	missingSeasons := make([]int, 0, len(missingSeasonsMap))
	for s := range missingSeasonsMap {
		missingSeasons = append(missingSeasons, s)
	}
	sort.Ints(missingSeasons)

	// Convert to JSON array
	seasonsJSON, _ := json.Marshal(missingSeasons)

	// If there are missing episodes, create or update wanted item
	if addedCount > 0 {
		// Get default quality preset
		var presetID *int64
		presets, _ := s.db.GetQualityPresets()
		for _, p := range presets {
			if p.MediaType == "tv" && p.IsDefault && p.Enabled {
				presetID = &p.ID
				break
			}
		}
		if presetID == nil {
			for _, p := range presets {
				if p.MediaType == "tv" && p.Enabled {
					presetID = &p.ID
					break
				}
			}
		}

		if existingWanted != nil {
			// Update existing wanted item with new seasons and trigger search
			existingSeasons := []int{}
			if existingWanted.Seasons != "" {
				json.Unmarshal([]byte(existingWanted.Seasons), &existingSeasons)
			}
			// Merge missing seasons with existing
			seasonSet := make(map[int]bool)
			for _, s := range existingSeasons {
				seasonSet[s] = true
			}
			for _, s := range missingSeasons {
				seasonSet[s] = true
			}
			mergedSeasons := make([]int, 0, len(seasonSet))
			for s := range seasonSet {
				mergedSeasons = append(mergedSeasons, s)
			}
			sort.Ints(mergedSeasons)
			mergedJSON, _ := json.Marshal(mergedSeasons)

			existingWanted.Seasons = string(mergedJSON)
			existingWanted.Monitored = true
			existingWanted.SearchNow = true
			if err := s.db.UpdateWantedItem(existingWanted); err != nil {
				log.Printf("Failed to update wanted item: %v", err)
			}
		} else {
			// Create new wanted item
			wanted := &database.WantedItem{
				Type:            "show",
				TmdbID:          *show.TmdbID,
				Title:           show.Title,
				Year:            show.Year,
				PosterPath:      show.PosterPath,
				QualityPresetID: presetID,
				Monitored:       true,
				Seasons:         string(seasonsJSON),
				SearchNow:       true,
			}
			if err := s.db.CreateWantedItem(wanted); err != nil {
				log.Printf("Failed to create wanted item: %v", err)
			}
		}
	}

	// Return result
	json.NewEncoder(w).Encode(map[string]int{"addedCount": addedCount})
}

// ============================================================================
// Collection handlers
// ============================================================================

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Check for media-specific query parameters
		tmdbIdStr := r.URL.Query().Get("tmdbId")
		mediaType := r.URL.Query().Get("mediaType")

		var collections []database.Collection
		var err error

		if tmdbIdStr != "" && mediaType != "" {
			tmdbId, parseErr := strconv.ParseInt(tmdbIdStr, 10, 64)
			if parseErr != nil {
				http.Error(w, "Invalid tmdbId", http.StatusBadRequest)
				return
			}
			collections, err = s.db.GetCollectionsForMedia(tmdbId, mediaType)
		} else {
			collections, err = s.db.GetCollections()
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if collections == nil {
			collections = []database.Collection{}
		}
		json.NewEncoder(w).Encode(collections)

	case http.MethodPost:
		// Admin only for creation
		user := s.getCurrentUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var input struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if input.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		coll := &database.Collection{
			Name:        input.Name,
			Description: input.Description,
			IsAuto:      false,
			SortOrder:   "custom",
		}

		if err := s.db.CreateCollection(coll); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(coll)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse the path: /api/collections/{id} or /api/collections/{id}/items or /api/collections/{id}/reorder
	path := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	// Handle sub-routes
	if len(parts) > 1 {
		switch parts[1] {
		case "items":
			s.handleCollectionItems(w, r, id)
			return
		case "reorder":
			s.handleCollectionReorder(w, r, id)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		// Get collection with items
		coll, err := s.db.GetCollection(id)
		if err != nil {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		items, err := s.db.GetCollectionItems(id)
		if err != nil {
			items = []database.CollectionItem{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":               coll.ID,
			"name":             coll.Name,
			"description":      coll.Description,
			"tmdbCollectionId": coll.TmdbCollectionID,
			"posterPath":       coll.PosterPath,
			"backdropPath":     coll.BackdropPath,
			"isAuto":           coll.IsAuto,
			"sortOrder":        coll.SortOrder,
			"itemCount":        coll.ItemCount,
			"ownedCount":       coll.OwnedCount,
			"createdAt":        coll.CreatedAt,
			"updatedAt":        coll.UpdatedAt,
			"items":            items,
		})

	case http.MethodPut:
		// Admin only
		user := s.getCurrentUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		coll, err := s.db.GetCollection(id)
		if err != nil {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		var input struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			SortOrder   *string `json:"sortOrder"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if input.Name != nil {
			coll.Name = *input.Name
		}
		if input.Description != nil {
			coll.Description = input.Description
		}
		if input.SortOrder != nil {
			coll.SortOrder = *input.SortOrder
		}

		if err := s.db.UpdateCollection(coll); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(coll)

	case http.MethodDelete:
		// Admin only
		user := s.getCurrentUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if err := s.db.DeleteCollection(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCollectionItems(w http.ResponseWriter, r *http.Request, collectionID int64) {
	// Admin only for modifications
	user := s.getCurrentUser(r)
	if user == nil || user.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var input struct {
			MediaType  string  `json:"mediaType"`
			TmdbID     int64   `json:"tmdbId"`
			Title      string  `json:"title"`
			Year       int     `json:"year"`
			PosterPath *string `json:"posterPath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if input.MediaType == "" || input.TmdbID == 0 {
			http.Error(w, "mediaType and tmdbId are required", http.StatusBadRequest)
			return
		}

		item := &database.CollectionItem{
			CollectionID: collectionID,
			MediaType:    input.MediaType,
			TmdbID:       input.TmdbID,
			Title:        input.Title,
			Year:         input.Year,
			PosterPath:   input.PosterPath,
		}

		// Check if this item is already in the library
		if input.MediaType == "movie" {
			if movie, err := s.db.GetMovieByTmdb(input.TmdbID); err == nil && movie != nil {
				item.MediaID = &movie.ID
			}
		} else if input.MediaType == "show" {
			if show, err := s.db.GetShowByTmdb(input.TmdbID); err == nil && show != nil {
				item.MediaID = &show.ID
			}
		}

		if err := s.db.AddCollectionItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)

	case http.MethodDelete:
		// Get tmdbId and mediaType from query params
		tmdbIDStr := r.URL.Query().Get("tmdbId")
		mediaType := r.URL.Query().Get("mediaType")

		if tmdbIDStr == "" || mediaType == "" {
			http.Error(w, "tmdbId and mediaType query params required", http.StatusBadRequest)
			return
		}

		tmdbID, err := strconv.ParseInt(tmdbIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid tmdbId", http.StatusBadRequest)
			return
		}

		if err := s.db.RemoveCollectionItem(collectionID, tmdbID, mediaType); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCollectionReorder(w http.ResponseWriter, r *http.Request, collectionID int64) {
	// Admin only
	user := s.getCurrentUser(r)
	if user == nil || user.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		ItemIDs []int64 `json:"itemIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateCollectionItemOrder(collectionID, input.ItemIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Also update sort order to custom
	coll, err := s.db.GetCollection(collectionID)
	if err == nil && coll != nil {
		coll.SortOrder = "custom"
		s.db.UpdateCollection(coll)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleLogs handles GET /api/logs
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := logging.LogQuery{
		Level:  r.URL.Query().Get("level"),
		Source: r.URL.Query().Get("source"),
		Search: r.URL.Query().Get("search"),
		Limit:  500, // Default limit
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
			if query.Limit > 1000 {
				query.Limit = 1000 // Max limit
			}
		}
	}

	response := logging.Query(query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLogsDownload handles GET /api/logs/download
func (s *Server) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content := logging.ExportAll()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"outpost-logs.txt\"")
	w.Write([]byte(content))
}

// handleHealthFull handles GET /api/health/full
func (s *Server) handleHealthFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set TMDB key from settings if needed
	if key, err := s.db.GetSetting("tmdb_api_key"); err == nil && key != "" {
		s.healthChecker.SetTMDBKey(key)
	}

	status := s.healthChecker.GetFullStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleHealthCheck handles POST /api/health/check/{name}
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract check name from path: /api/health/check/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/health/check/")
	if path == "" {
		http.Error(w, "Check name required", http.StatusBadRequest)
		return
	}

	// Set TMDB key from settings if needed
	if key, err := s.db.GetSetting("tmdb_api_key"); err == nil && key != "" {
		s.healthChecker.SetTMDBKey(key)
	}

	check := s.healthChecker.RunSingleCheck(path)
	if check == nil {
		http.Error(w, "Unknown health check", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}

// handleBackup handles POST /api/backup - creates and downloads a backup
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create backup
	appVersion := "0.1.0" // TODO: Get from config or build info
	backup, err := s.db.CreateBackup(appVersion)
	if err != nil {
		log.Printf("Failed to create backup: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
		return
	}

	// Convert to JSON
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal backup: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("outpost-backup-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// handleRestore handles POST /api/backup/restore - restores from a backup file
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get restore mode from query param (replace or merge)
	mode := r.URL.Query().Get("mode")
	if mode != "replace" && mode != "merge" {
		mode = "merge" // Default to merge for safety
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	var data []byte
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Validate backup
	backup, err := database.ValidateBackup(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Perform restore
	result, err := s.db.RestoreBackup(backup, mode)
	if err != nil {
		log.Printf("Failed to restore backup: %v", err)
		http.Error(w, fmt.Sprintf("Failed to restore backup: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Smart playlist handlers
func (s *Server) handleSmartPlaylists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Get user ID to filter playlists (nil shows all system playlists)
		user := s.getCurrentUser(r)
		var userID *int64
		if user != nil {
			userID = &user.ID
		}
		playlists, err := s.db.GetSmartPlaylists(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if playlists == nil {
			playlists = []database.SmartPlaylist{}
		}
		json.NewEncoder(w).Encode(playlists)

	case http.MethodPost:
		user := s.getCurrentUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var input struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
			Rules       string  `json:"rules"`
			SortBy      string  `json:"sortBy"`
			SortOrder   string  `json:"sortOrder"`
			LimitCount  *int    `json:"limitCount"`
			MediaType   string  `json:"mediaType"`
			AutoRefresh bool    `json:"autoRefresh"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if input.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}
		if input.Rules == "" {
			http.Error(w, "Rules are required", http.StatusBadRequest)
			return
		}

		// Validate rules JSON
		var rules database.PlaylistRules
		if err := json.Unmarshal([]byte(input.Rules), &rules); err != nil {
			http.Error(w, "Invalid rules format", http.StatusBadRequest)
			return
		}

		// Set defaults
		if input.SortBy == "" {
			input.SortBy = "added"
		}
		if input.SortOrder == "" {
			input.SortOrder = "desc"
		}
		if input.MediaType == "" {
			input.MediaType = "both"
		}

		playlist := &database.SmartPlaylist{
			UserID:      &user.ID,
			Name:        input.Name,
			Description: input.Description,
			Rules:       input.Rules,
			SortBy:      input.SortBy,
			SortOrder:   input.SortOrder,
			LimitCount:  input.LimitCount,
			MediaType:   input.MediaType,
			AutoRefresh: input.AutoRefresh,
			IsSystem:    false,
		}

		if err := s.db.CreateSmartPlaylist(playlist); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(playlist)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse the path: /api/smart-playlists/{id} or /api/smart-playlists/{id}/refresh
	path := strings.TrimPrefix(r.URL.Path, "/api/smart-playlists/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Playlist ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	// Handle refresh sub-route
	if len(parts) > 1 && parts[1] == "refresh" {
		s.handleSmartPlaylistRefresh(w, r, id)
		return
	}

	// Handle items sub-route
	if len(parts) > 1 && parts[1] == "items" {
		s.handleSmartPlaylistItems(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		playlist, err := s.db.GetSmartPlaylist(id)
		if err != nil {
			http.Error(w, "Playlist not found", http.StatusNotFound)
			return
		}

		// Get current profile for watched status
		profileID := s.getActiveProfileID(r)

		// Get items
		items, err := s.db.GetSmartPlaylistItems(playlist, profileID)
		if err != nil {
			items = []database.SmartPlaylistItem{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            playlist.ID,
			"userId":        playlist.UserID,
			"name":          playlist.Name,
			"description":   playlist.Description,
			"rules":         playlist.Rules,
			"sortBy":        playlist.SortBy,
			"sortOrder":     playlist.SortOrder,
			"limitCount":    playlist.LimitCount,
			"mediaType":     playlist.MediaType,
			"autoRefresh":   playlist.AutoRefresh,
			"isSystem":      playlist.IsSystem,
			"lastRefreshed": playlist.LastRefreshed,
			"createdAt":     playlist.CreatedAt,
			"itemCount":     len(items),
			"items":         items,
		})

	case http.MethodPut:
		user := s.getCurrentUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		playlist, err := s.db.GetSmartPlaylist(id)
		if err != nil {
			http.Error(w, "Playlist not found", http.StatusNotFound)
			return
		}

		// Only owner or admin can update
		if playlist.UserID != nil && *playlist.UserID != user.ID && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// System playlists can only be updated by admin
		if playlist.IsSystem && user.Role != "admin" {
			http.Error(w, "Cannot modify system playlist", http.StatusForbidden)
			return
		}

		var input struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Rules       *string `json:"rules"`
			SortBy      *string `json:"sortBy"`
			SortOrder   *string `json:"sortOrder"`
			LimitCount  *int    `json:"limitCount"`
			MediaType   *string `json:"mediaType"`
			AutoRefresh *bool   `json:"autoRefresh"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if input.Name != nil {
			playlist.Name = *input.Name
		}
		if input.Description != nil {
			playlist.Description = input.Description
		}
		if input.Rules != nil {
			// Validate rules JSON
			var rules database.PlaylistRules
			if err := json.Unmarshal([]byte(*input.Rules), &rules); err != nil {
				http.Error(w, "Invalid rules format", http.StatusBadRequest)
				return
			}
			playlist.Rules = *input.Rules
		}
		if input.SortBy != nil {
			playlist.SortBy = *input.SortBy
		}
		if input.SortOrder != nil {
			playlist.SortOrder = *input.SortOrder
		}
		if input.LimitCount != nil {
			playlist.LimitCount = input.LimitCount
		}
		if input.MediaType != nil {
			playlist.MediaType = *input.MediaType
		}
		if input.AutoRefresh != nil {
			playlist.AutoRefresh = *input.AutoRefresh
		}

		if err := s.db.UpdateSmartPlaylist(playlist); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(playlist)

	case http.MethodDelete:
		user := s.getCurrentUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		playlist, err := s.db.GetSmartPlaylist(id)
		if err != nil {
			http.Error(w, "Playlist not found", http.StatusNotFound)
			return
		}

		// System playlists cannot be deleted
		if playlist.IsSystem {
			http.Error(w, "Cannot delete system playlist", http.StatusForbidden)
			return
		}

		// Only owner or admin can delete
		if playlist.UserID != nil && *playlist.UserID != user.ID && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if err := s.db.DeleteSmartPlaylist(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSmartPlaylistRefresh(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playlist, err := s.db.GetSmartPlaylist(id)
	if err != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	profileID := s.getActiveProfileID(r)

	// Get fresh items
	items, err := s.db.GetSmartPlaylistItems(playlist, profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update last refreshed
	s.db.UpdateSmartPlaylistRefreshed(id)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":     items,
		"itemCount": len(items),
	})
}

func (s *Server) handleSmartPlaylistItems(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playlist, err := s.db.GetSmartPlaylist(id)
	if err != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	profileID := s.getActiveProfileID(r)

	items, err := s.db.GetSmartPlaylistItems(playlist, profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleSmartPlaylistPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var input struct {
		Rules      string `json:"rules"`
		SortBy     string `json:"sortBy"`
		SortOrder  string `json:"sortOrder"`
		LimitCount *int   `json:"limitCount"`
		MediaType  string `json:"mediaType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate rules
	var rules database.PlaylistRules
	if err := json.Unmarshal([]byte(input.Rules), &rules); err != nil {
		http.Error(w, "Invalid rules format", http.StatusBadRequest)
		return
	}

	// Set defaults
	if input.SortBy == "" {
		input.SortBy = "added"
	}
	if input.SortOrder == "" {
		input.SortOrder = "desc"
	}
	if input.MediaType == "" {
		input.MediaType = "both"
	}

	// Create temporary playlist for preview
	playlist := &database.SmartPlaylist{
		Rules:      input.Rules,
		SortBy:     input.SortBy,
		SortOrder:  input.SortOrder,
		LimitCount: input.LimitCount,
		MediaType:  input.MediaType,
	}

	profileID := s.getActiveProfileID(r)

	items, err := s.db.GetSmartPlaylistItems(playlist, profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":     items,
		"itemCount": len(items),
	})
}

// Upgrade search handlers

func (s *Server) handleUpgrades(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := s.db.GetUpgradesSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleUpgradeSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MediaType string `json:"mediaType"`
		MediaID   int64  `json:"mediaId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MediaType == "" || req.MediaID == 0 {
		http.Error(w, "mediaType and mediaId are required", http.StatusBadRequest)
		return
	}

	// Get the media item to search for upgrades
	var tmdbID int64
	var title string
	var year int
	var imdbID string
	var qualityPresetID int64

	if req.MediaType == "movie" {
		movie, err := s.db.GetMovie(req.MediaID)
		if err != nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		if movie.TmdbID != nil {
			tmdbID = *movie.TmdbID
		}
		title = movie.Title
		year = movie.Year
		if movie.ImdbID != nil {
			imdbID = *movie.ImdbID
		}
		// Get quality preset from override or default
		override, _ := s.db.GetMediaQualityOverride(req.MediaID, "movie")
		if override != nil && override.PresetID != nil {
			qualityPresetID = *override.PresetID
		} else {
			// Get default preset
			presets, _ := s.db.GetQualityPresets()
			for _, p := range presets {
				if p.IsDefault && p.MediaType == "movie" {
					qualityPresetID = p.ID
					break
				}
			}
		}
	} else if req.MediaType == "episode" {
		// For episodes, we need to get the season and show info
		episode, err := s.db.GetEpisode(req.MediaID)
		if err != nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		season, err := s.db.GetSeasonByID(episode.SeasonID)
		if err != nil {
			http.Error(w, "Season not found", http.StatusNotFound)
			return
		}
		show, err := s.db.GetShow(season.ShowID)
		if err != nil {
			http.Error(w, "Show not found", http.StatusNotFound)
			return
		}
		if show.TmdbID != nil {
			tmdbID = *show.TmdbID
		}
		title = fmt.Sprintf("%s S%02dE%02d", show.Title, season.SeasonNumber, episode.EpisodeNumber)
		year = show.Year
		if show.ImdbID != nil {
			imdbID = *show.ImdbID
		}
		// Get quality preset from override or default
		override, _ := s.db.GetMediaQualityOverride(show.ID, "show")
		if override != nil && override.PresetID != nil {
			qualityPresetID = *override.PresetID
		} else {
			// Get default preset
			presets, _ := s.db.GetQualityPresets()
			for _, p := range presets {
				if p.IsDefault && p.MediaType == "tv" {
					qualityPresetID = p.ID
					break
				}
			}
		}
	} else {
		http.Error(w, "Invalid mediaType, must be 'movie' or 'episode'", http.StatusBadRequest)
		return
	}

	// Create a wanted item for the upgrade
	if tmdbID > 0 && qualityPresetID > 0 {
		err := s.db.CreateUpgradeWantedItem(req.MediaType, tmdbID, imdbID, title, year, "", qualityPresetID, req.MediaID)
		if err != nil {
			log.Printf("Failed to create upgrade wanted item: %v", err)
			http.Error(w, "Failed to queue upgrade search", http.StatusInternalServerError)
			return
		}

		// Trigger the search via scheduler
		if s.scheduler != nil {
			mediaType := req.MediaType
			if mediaType == "episode" {
				mediaType = "show"
			}
			go s.scheduler.SearchWantedItem(tmdbID, mediaType)
		}
	}

	// Update the search timestamp
	s.db.UpdateUpgradeSearched(req.MediaID, req.MediaType, false)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Upgrade search queued",
	})
}

func (s *Server) handleUpgradeSearchAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Limit     int    `json:"limit"`
		MediaType string `json:"mediaType"` // "movie", "episode", or "" for both
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10 // Default limit
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	var queuedCount int

	// Get upgradeable movies if type is empty or "movie"
	if req.MediaType == "" || req.MediaType == "movie" {
		movies, err := s.db.GetUpgradeableMovies(req.Limit)
		if err != nil {
			log.Printf("Failed to get upgradeable movies: %v", err)
		} else {
			for _, item := range movies {
				// Get full movie details
				movie, err := s.db.GetMovie(item.ID)
				if err != nil {
					continue
				}

				var imdbID string
				if movie.ImdbID != nil {
					imdbID = *movie.ImdbID
				}

				// Get quality preset from override or default
				var qualityPresetID int64
				override, _ := s.db.GetMediaQualityOverride(item.ID, "movie")
				if override != nil && override.PresetID != nil {
					qualityPresetID = *override.PresetID
				} else {
					presets, _ := s.db.GetQualityPresets()
					for _, p := range presets {
						if p.IsDefault && p.MediaType == "movie" {
							qualityPresetID = p.ID
							break
						}
					}
				}

				if movie.TmdbID != nil && qualityPresetID > 0 {
					err := s.db.CreateUpgradeWantedItem("movie", *movie.TmdbID, imdbID, movie.Title, movie.Year, "", qualityPresetID, item.ID)
					if err == nil {
						queuedCount++
						s.db.UpdateUpgradeSearched(item.ID, "movie", false)
						if s.scheduler != nil {
							go s.scheduler.SearchWantedItem(*movie.TmdbID, "movie")
						}
					}
				}
			}
		}
	}

	// Get upgradeable episodes if type is empty or "episode"
	if req.MediaType == "" || req.MediaType == "episode" {
		episodes, err := s.db.GetUpgradeableEpisodes(req.Limit)
		if err != nil {
			log.Printf("Failed to get upgradeable episodes: %v", err)
		} else {
			for _, item := range episodes {
				// Get episode, season and show details
				episode, err := s.db.GetEpisode(item.ID)
				if err != nil {
					continue
				}
				season, err := s.db.GetSeasonByID(episode.SeasonID)
				if err != nil {
					continue
				}
				show, err := s.db.GetShow(season.ShowID)
				if err != nil {
					continue
				}

				var imdbID string
				if show.ImdbID != nil {
					imdbID = *show.ImdbID
				}

				// Get quality preset from override or default
				var qualityPresetID int64
				override, _ := s.db.GetMediaQualityOverride(show.ID, "show")
				if override != nil && override.PresetID != nil {
					qualityPresetID = *override.PresetID
				} else {
					presets, _ := s.db.GetQualityPresets()
					for _, p := range presets {
						if p.IsDefault && p.MediaType == "tv" {
							qualityPresetID = p.ID
							break
						}
					}
				}

				if show.TmdbID != nil && qualityPresetID > 0 {
					title := fmt.Sprintf("%s S%02dE%02d", show.Title, season.SeasonNumber, episode.EpisodeNumber)
					err := s.db.CreateUpgradeWantedItem("episode", *show.TmdbID, imdbID, title, show.Year, "", qualityPresetID, item.ID)
					if err == nil {
						queuedCount++
						s.db.UpdateUpgradeSearched(item.ID, "episode", false)
						if s.scheduler != nil {
							go s.scheduler.SearchWantedItem(*show.TmdbID, "show")
						}
					}
				}
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"queued":  queuedCount,
		"message": fmt.Sprintf("Queued %d upgrade searches", queuedCount),
	})
}

// handleOpenSubtitlesSearch searches for subtitles on OpenSubtitles
func (s *Server) handleOpenSubtitlesSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query           string   `json:"query"`
		TMDbID          int      `json:"tmdbId"`
		IMDbID          string   `json:"imdbId"`
		Year            int      `json:"year"`
		Season          int      `json:"season"`
		Episode         int      `json:"episode"`
		Languages       []string `json:"languages"`
		HearingImpaired *bool    `json:"hearingImpaired"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiKey, _ := s.db.GetSetting("opensubtitles_api_key")
	if apiKey == "" {
		http.Error(w, "OpenSubtitles API key not configured", http.StatusBadRequest)
		return
	}

	client := subtitles.NewClient(apiKey)

	// Use configured languages if none provided
	if len(req.Languages) == 0 {
		langSetting, _ := s.db.GetSetting("opensubtitles_languages")
		if langSetting != "" {
			req.Languages = strings.Split(langSetting, ",")
		} else {
			req.Languages = []string{"en"}
		}
	}

	searchReq := subtitles.SearchRequest{
		Query:           req.Query,
		TMDbID:          req.TMDbID,
		IMDbID:          req.IMDbID,
		Year:            req.Year,
		Season:          req.Season,
		Episode:         req.Episode,
		Languages:       req.Languages,
		HearingImpaired: req.HearingImpaired,
	}

	results, err := client.Search(searchReq)
	if err != nil {
		log.Printf("OpenSubtitles search error: %v", err)
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleOpenSubtitlesDownload downloads a subtitle file
func (s *Server) handleOpenSubtitlesDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileID     int    `json:"fileId"`
		MediaType  string `json:"mediaType"`
		MediaID    int64  `json:"mediaId"`
		Language   string `json:"language"`
		EpisodeID  *int64 `json:"episodeId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileID == 0 {
		http.Error(w, "fileId is required", http.StatusBadRequest)
		return
	}

	apiKey, _ := s.db.GetSetting("opensubtitles_api_key")
	if apiKey == "" {
		http.Error(w, "OpenSubtitles API key not configured", http.StatusBadRequest)
		return
	}

	// Get the video file path
	var videoPath string
	if req.MediaType == "movie" {
		movie, err := s.db.GetMovie(req.MediaID)
		if err != nil || movie == nil {
			http.Error(w, "Movie not found", http.StatusNotFound)
			return
		}
		videoPath = movie.Path
	} else if req.MediaType == "episode" && req.EpisodeID != nil {
		episode, err := s.db.GetEpisode(*req.EpisodeID)
		if err != nil || episode == nil {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		videoPath = episode.Path
	} else {
		http.Error(w, "Invalid media type or missing episodeId", http.StatusBadRequest)
		return
	}

	if videoPath == "" {
		http.Error(w, "No video file found", http.StatusNotFound)
		return
	}

	client := subtitles.NewClient(apiKey)

	// Get download link
	dlResp, err := client.GetDownloadLink(req.FileID)
	if err != nil {
		log.Printf("OpenSubtitles download link error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get download link: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate subtitle path
	videoBase := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	subPath := videoBase + "." + req.Language + ".srt"

	// Download the subtitle
	if err := client.Download(dlResp.Link, subPath); err != nil {
		log.Printf("OpenSubtitles download error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to download subtitle: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Downloaded subtitle to: %s", subPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"path":     subPath,
		"message":  "Subtitle downloaded successfully",
		"remaining": dlResp.Remaining,
	})
}

// handleOpenSubtitlesLanguages returns available subtitle languages
func (s *Server) handleOpenSubtitlesLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey, _ := s.db.GetSetting("opensubtitles_api_key")
	if apiKey == "" {
		// Return common languages without API key
		languages := []subtitles.Language{
			{Code: "en", Name: "English"},
			{Code: "es", Name: "Spanish"},
			{Code: "fr", Name: "French"},
			{Code: "de", Name: "German"},
			{Code: "it", Name: "Italian"},
			{Code: "pt", Name: "Portuguese"},
			{Code: "nl", Name: "Dutch"},
			{Code: "pl", Name: "Polish"},
			{Code: "ru", Name: "Russian"},
			{Code: "ja", Name: "Japanese"},
			{Code: "ko", Name: "Korean"},
			{Code: "zh", Name: "Chinese"},
			{Code: "ar", Name: "Arabic"},
			{Code: "hi", Name: "Hindi"},
			{Code: "sv", Name: "Swedish"},
			{Code: "da", Name: "Danish"},
			{Code: "fi", Name: "Finnish"},
			{Code: "no", Name: "Norwegian"},
			{Code: "tr", Name: "Turkish"},
			{Code: "el", Name: "Greek"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
		return
	}

	client := subtitles.NewClient(apiKey)
	languages, err := client.GetLanguages()
	if err != nil {
		log.Printf("OpenSubtitles languages error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get languages: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

// handleOpenSubtitlesTest tests the OpenSubtitles API connection
func (s *Server) handleOpenSubtitlesTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey string `json:"apiKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiKey := req.APIKey
	if apiKey == "" {
		apiKey, _ = s.db.GetSetting("opensubtitles_api_key")
	}

	if apiKey == "" {
		http.Error(w, "No API key provided", http.StatusBadRequest)
		return
	}

	client := subtitles.NewClient(apiKey)
	if err := client.Test(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "API key is valid",
	})
}

// handleTraktAuthURL returns the OAuth authorization URL for Trakt
func (s *Server) handleTraktAuthURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID, _ := s.db.GetSetting("trakt_client_id")
	if clientID == "" {
		http.Error(w, "Trakt client ID not configured", http.StatusBadRequest)
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		http.Error(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}

	client := trakt.NewClient(clientID, "")
	authURL := client.GetAuthURL(redirectURI)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": authURL,
	})
}

// handleTraktCallback handles the OAuth callback from Trakt
func (s *Server) handleTraktCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	var req struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirectUri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	clientID, _ := s.db.GetSetting("trakt_client_id")
	clientSecret, _ := s.db.GetSetting("trakt_client_secret")

	if clientID == "" || clientSecret == "" {
		http.Error(w, "Trakt not configured", http.StatusBadRequest)
		return
	}

	client := trakt.NewClient(clientID, clientSecret)
	tokenResp, err := client.ExchangeCode(req.Code, req.RedirectURI)
	if err != nil {
		log.Printf("Trakt code exchange error: %v", err)
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get user info from Trakt
	settings, err := client.GetUserSettings()
	if err != nil {
		log.Printf("Trakt get user settings error: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Save config to database
	expiresAt := time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	config := &database.TraktConfig{
		UserID:        user.ID,
		AccessToken:   tokenResp.AccessToken,
		RefreshToken:  tokenResp.RefreshToken,
		ExpiresAt:     &expiresAt,
		Username:      &settings.User.Username,
		SyncEnabled:   true,
		SyncWatched:   true,
		SyncRatings:   true,
		SyncWatchlist: true,
	}

	if err := s.db.SaveTraktConfig(config); err != nil {
		log.Printf("Failed to save Trakt config: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": settings.User.Username,
	})
}

// handleTraktConfig gets or updates Trakt configuration for the current user
func (s *Server) handleTraktConfig(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*database.User)

	switch r.Method {
	case http.MethodGet:
		config, err := s.db.GetTraktConfig(user.ID)
		if err != nil {
			http.Error(w, "Failed to get config", http.StatusInternalServerError)
			return
		}

		// Return config without sensitive tokens
		response := map[string]interface{}{
			"connected": config != nil && config.AccessToken != "",
		}

		if config != nil {
			response["username"] = config.Username
			response["syncEnabled"] = config.SyncEnabled
			response["syncWatched"] = config.SyncWatched
			response["syncRatings"] = config.SyncRatings
			response["syncWatchlist"] = config.SyncWatchlist
			response["lastSyncedAt"] = config.LastSyncedAt
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		var req struct {
			SyncEnabled   bool `json:"syncEnabled"`
			SyncWatched   bool `json:"syncWatched"`
			SyncRatings   bool `json:"syncRatings"`
			SyncWatchlist bool `json:"syncWatchlist"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		config, err := s.db.GetTraktConfig(user.ID)
		if err != nil || config == nil {
			http.Error(w, "Trakt not connected", http.StatusBadRequest)
			return
		}

		config.SyncEnabled = req.SyncEnabled
		config.SyncWatched = req.SyncWatched
		config.SyncRatings = req.SyncRatings
		config.SyncWatchlist = req.SyncWatchlist

		if err := s.db.SaveTraktConfig(config); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTraktDisconnect disconnects Trakt for the current user
func (s *Server) handleTraktDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	if err := s.db.DeleteTraktConfig(user.ID); err != nil {
		http.Error(w, "Failed to disconnect", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleTraktSync triggers a manual sync with Trakt
func (s *Server) handleTraktSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	// Get the active profile or default profile
	profileID := s.getActiveProfileID(r)
	if profileID == nil {
		profile, err := s.db.GetDefaultProfile(user.ID)
		if err != nil || profile == nil {
			http.Error(w, "No profile found", http.StatusBadRequest)
			return
		}
		profileID = &profile.ID
	}

	config, err := s.db.GetTraktConfig(user.ID)
	if err != nil || config == nil || config.AccessToken == "" {
		http.Error(w, "Trakt not connected", http.StatusBadRequest)
		return
	}

	clientID, _ := s.db.GetSetting("trakt_client_id")
	clientSecret, _ := s.db.GetSetting("trakt_client_secret")

	client := trakt.NewClient(clientID, clientSecret)
	client.SetTokens(config.AccessToken, config.RefreshToken, *config.ExpiresAt)

	// Refresh token if needed
	if client.NeedsRefresh() {
		tokenResp, err := client.RefreshAccessToken()
		if err != nil {
			log.Printf("Trakt token refresh error: %v", err)
			http.Error(w, "Failed to refresh token", http.StatusInternalServerError)
			return
		}
		expiresAt := time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		config.AccessToken = tokenResp.AccessToken
		config.RefreshToken = tokenResp.RefreshToken
		config.ExpiresAt = &expiresAt
		s.db.SaveTraktConfig(config)
	}

	// Perform sync
	syncResult := s.performTraktSync(*profileID, client, config)

	// Update last synced time
	s.db.UpdateTraktSyncTime(user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncResult)
}

// performTraktSync performs the actual sync with Trakt
func (s *Server) performTraktSync(profileID int64, client *trakt.Client, config *database.TraktConfig) map[string]interface{} {
	result := map[string]interface{}{
		"success": true,
		"pulled":  map[string]int{},
		"pushed":  map[string]int{},
		"errors":  []string{},
	}

	errors := []string{}
	pulled := map[string]int{"movies": 0, "shows": 0}
	pushed := map[string]int{"movies": 0, "episodes": 0}

	// Helper to check if watched (>90% progress)
	isWatched := func(mediaType string, mediaID int64) bool {
		progress, err := s.db.GetProgress(profileID, mediaType, mediaID)
		if err != nil || progress == nil {
			return false
		}
		if progress.Duration > 0 && progress.Position/progress.Duration >= 0.9 {
			return true
		}
		return false
	}

	// Helper to mark as watched
	markWatched := func(mediaType string, mediaID int64) {
		progress := &database.Progress{
			ProfileID: profileID,
			MediaType: mediaType,
			MediaID:   mediaID,
			Position:  100, // 100% position
			Duration:  100, // 100% duration
		}
		s.db.SaveProgress(progress)
	}

	// Pull watched movies from Trakt
	if config.SyncWatched {
		watchedMovies, err := client.GetWatchedMovies()
		if err != nil {
			errors = append(errors, "Failed to get watched movies: "+err.Error())
		} else {
			for _, wm := range watchedMovies {
				// Find movie in library by TMDB ID
				movie, err := s.db.GetMovieByTmdb(int64(wm.Movie.IDs.TMDB))
				if err != nil || movie == nil {
					continue
				}
				// Mark as watched if not already
				if !isWatched("movie", movie.ID) {
					markWatched("movie", movie.ID)
					pulled["movies"]++
				}
			}
		}

		// Pull watched shows from Trakt
		watchedShows, err := client.GetWatchedShows()
		if err != nil {
			errors = append(errors, "Failed to get watched shows: "+err.Error())
		} else {
			for _, ws := range watchedShows {
				// Find show in library
				show, err := s.db.GetShowByTmdb(int64(ws.Show.IDs.TMDB))
				if err != nil || show == nil {
					continue
				}
				// Mark episodes as watched
				for _, season := range ws.Seasons {
					for _, ep := range season.Episodes {
						episode, err := s.db.GetEpisodeByShowSeasonEpisode(show.ID, season.Number, ep.Number)
						if err != nil || episode == nil {
							continue
						}
						if !isWatched("episode", episode.ID) {
							markWatched("episode", episode.ID)
							pulled["shows"]++
						}
					}
				}
			}
		}

		// Push unsynced watch history to Trakt
		unsyncedHistory, err := s.db.GetUnsyncedWatchHistory(profileID)
		if err == nil && len(unsyncedHistory) > 0 {
			var movieItems []trakt.HistoryItem
			var episodeItems []trakt.HistoryItem
			var syncedIDs []int64

			for _, item := range unsyncedHistory {
				if item.MediaType == "movie" {
					movie, err := s.db.GetMovie(item.MediaID)
					if err != nil || movie == nil || movie.TmdbID == nil {
						continue
					}
					movieItems = append(movieItems, trakt.HistoryItem{
						WatchedAt: item.WatchedAt,
						Movie: &trakt.Movie{
							IDs: trakt.IDs{TMDB: int(*movie.TmdbID)},
						},
					})
					syncedIDs = append(syncedIDs, item.ID)
				} else if item.MediaType == "episode" {
					episode, err := s.db.GetEpisode(item.MediaID)
					if err != nil || episode == nil {
						continue
					}
					season, err := s.db.GetSeasonByID(episode.SeasonID)
					if err != nil || season == nil {
						continue
					}
					show, err := s.db.GetShow(season.ShowID)
					if err != nil || show == nil || show.TmdbID == nil {
						continue
					}
					episodeItems = append(episodeItems, trakt.HistoryItem{
						WatchedAt: item.WatchedAt,
						Show: &trakt.Show{
							IDs: trakt.IDs{TMDB: int(*show.TmdbID)},
						},
						Episode: &trakt.Episode{
							Season: season.SeasonNumber,
							Number: episode.EpisodeNumber,
						},
					})
					syncedIDs = append(syncedIDs, item.ID)
				}
			}

			if len(movieItems) > 0 || len(episodeItems) > 0 {
				histReq := &trakt.HistoryRequest{
					Movies:   movieItems,
					Episodes: episodeItems,
				}
				resp, err := client.AddToHistory(histReq)
				if err != nil {
					errors = append(errors, "Failed to push history: "+err.Error())
				} else {
					pushed["movies"] = resp.Added.Movies
					pushed["episodes"] = resp.Added.Episodes
					s.db.MarkWatchHistorySynced(syncedIDs)
				}
			}
		}
	}

	result["pulled"] = pulled
	result["pushed"] = pushed
	result["errors"] = errors
	if len(errors) > 0 {
		result["success"] = false
	}

	return result
}

// handleTraktTest tests the Trakt API connection for the current user
func (s *Server) handleTraktTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.Context().Value(userContextKey).(*database.User)

	config, err := s.db.GetTraktConfig(user.ID)
	if err != nil || config == nil || config.AccessToken == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"connected": false,
			"error":     "Not connected to Trakt",
		})
		return
	}

	clientID, _ := s.db.GetSetting("trakt_client_id")
	clientSecret, _ := s.db.GetSetting("trakt_client_secret")

	client := trakt.NewClient(clientID, clientSecret)
	client.SetTokens(config.AccessToken, config.RefreshToken, *config.ExpiresAt)

	if err := client.Test(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"connected": true,
			"error":     err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"connected": true,
		"username":  config.Username,
	})
}
