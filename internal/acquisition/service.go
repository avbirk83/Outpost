package acquisition

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/download"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
	importpkg "github.com/outpost/outpost/internal/import"
	"github.com/outpost/outpost/internal/parser"
	"github.com/outpost/outpost/internal/quality"
	"github.com/outpost/outpost/internal/request"
)

// Service orchestrates the download lifecycle using TrackedDownload
type Service struct {
	db         *database.Database
	rawDB      *sql.DB
	clients    *downloadclient.Manager
	indexers   *indexer.Manager
	monitoring *download.MonitoringService
	requests   *request.LifecycleManager
	decisions  *importpkg.DecisionMaker
	upgrades   *importpkg.UpgradeChecker

	seedingConfig     download.SeedingConfig
	autoBlockAfter    int
	deleteOnFail      bool
	searchAlternative bool

	stopCh  chan struct{}
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// Config holds configuration for the acquisition service
type Config struct {
	PollInterval      time.Duration
	StalledThreshold  time.Duration
	SeedingConfig     download.SeedingConfig
	AutoBlockAfter    int
	DeleteOnFail      bool
	SearchAlternative bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		PollInterval:      5 * time.Second,
		StalledThreshold:  6 * time.Hour,
		SeedingConfig:     download.DefaultSeedingConfig(),
		AutoBlockAfter:    3,
		DeleteOnFail:      true,
		SearchAlternative: true,
	}
}

// NewService creates a new acquisition service
func NewService(db *database.Database, rawDB *sql.DB, clients *downloadclient.Manager, indexers *indexer.Manager, cfg *Config) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create monitoring service
	monConfig := download.MonitoringConfig{
		PollInterval:     cfg.PollInterval,
		StalledThreshold: cfg.StalledThreshold,
		SeedingConfig:    cfg.SeedingConfig,
	}
	monitoring := download.NewMonitoringService(rawDB, clients, monConfig)

	svc := &Service{
		db:                db,
		rawDB:             rawDB,
		clients:           clients,
		indexers:          indexers,
		monitoring:        monitoring,
		requests:          request.NewLifecycleManager(rawDB),
		decisions:         importpkg.NewDecisionMaker(),
		upgrades:          importpkg.NewUpgradeChecker(),
		seedingConfig:     cfg.SeedingConfig,
		autoBlockAfter:    cfg.AutoBlockAfter,
		deleteOnFail:      cfg.DeleteOnFail,
		searchAlternative: cfg.SearchAlternative,
		stopCh:            make(chan struct{}),
	}

	// Wire up callbacks
	monitoring.OnReadyForImport = svc.handleReadyForImport
	monitoring.OnReadyToRemove = svc.handleReadyToRemove

	return svc
}

// Start begins the service
func (s *Service) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.monitoring.Start()
	log.Println("Acquisition service started (using TrackedDownload)")
}

// Stop stops the service
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.monitoring.Stop()
	log.Println("Acquisition service stopped")
}

// handleReadyForImport is called when a download completes and is ready for import
func (s *Service) handleReadyForImport(td *download.TrackedDownload) {
	log.Printf("Processing import for: %s", td.Title)

	// Mark as importing
	if err := s.monitoring.MarkImporting(td); err != nil {
		log.Printf("Error marking as importing: %v", err)
		return
	}

	// Update linked request status
	if td.RequestID != nil {
		s.requests.MarkProcessing(*td.RequestID)
	}

	// Run import
	importPath, err := s.runImport(td)
	if err != nil {
		log.Printf("Import failed for %s: %v", td.Title, err)
		s.handleImportFailure(td, err)
		return
	}

	// Mark as imported
	if err := s.monitoring.MarkImported(td, importPath); err != nil {
		log.Printf("Error marking as imported: %v", err)
	}

	// Update linked request status
	if td.RequestID != nil {
		s.requests.MarkAvailable(*td.RequestID)
	}

	log.Printf("Successfully imported: %s -> %s", td.Title, importPath)
}

// runImport performs the actual import
func (s *Service) runImport(td *download.TrackedDownload) (string, error) {
	sourcePath := td.DownloadPath
	if sourcePath == "" {
		return "", &importpkg.ImportError{Message: "No download path set"}
	}

	// Evaluate files
	decisions, err := s.decisions.EvaluateFiles(sourcePath, td)
	if err != nil {
		return "", err
	}

	// Get main file
	mainFile := s.decisions.GetMainFile(decisions)
	if mainFile == nil {
		return "", &importpkg.ImportError{Message: "No valid video files found (all rejected as samples)"}
	}

	// Get destination library
	library, err := s.getDestinationLibrary(td)
	if err != nil {
		return "", err
	}

	// Generate destination path
	destPath, err := s.generateDestPath(td, library, mainFile)
	if err != nil {
		return "", err
	}

	// Create directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Check for upgrade - if we already have this media, handle the old file
	if td.MediaID != nil {
		s.handleUpgrade(td, destDir)
	}

	// Move main file
	if err := moveFile(mainFile.FilePath, destPath); err != nil {
		return "", err
	}

	// Handle extras
	extras := s.decisions.GetExtras(decisions)
	if len(extras) > 0 {
		extrasDir := filepath.Join(destDir, "Extras")
		os.MkdirAll(extrasDir, 0755)
		for _, extra := range extras {
			moveFile(extra.FilePath, filepath.Join(extrasDir, filepath.Base(extra.FilePath)))
		}
	}

	// Handle subtitles
	subs := findSubtitles(sourcePath)
	for _, sub := range subs {
		subDest := generateSubtitlePath(destPath, sub)
		moveFile(sub, subDest)
	}

	// Record import history
	s.db.CreateImportHistory(&database.ImportHistory{
		SourcePath: sourcePath,
		DestPath:   destPath,
		MediaID:    td.MediaID,
		MediaType:  &td.MediaType,
		Success:    true,
	})

	// Update media quality status
	if td.MediaID != nil {
		s.updateQualityStatus(*td.MediaID, td.MediaType, td.ParsedInfo)
	}

	// Clean up source
	s.cleanupSource(sourcePath)

	return destPath, nil
}

// handleUpgrade checks for and handles file upgrades
func (s *Service) handleUpgrade(td *download.TrackedDownload, destDir string) {
	// Get current quality status
	status, err := s.db.GetMediaQualityStatus(*td.MediaID, td.MediaType)
	if err != nil || status == nil {
		return
	}

	// Build current parsed info from status
	current := &parser.ParsedRelease{
		Resolution:  deref(status.CurrentResolution),
		Source:      deref(status.CurrentSource),
		HDR:         deref(status.CurrentHDR),
		AudioFormat: deref(status.CurrentAudio),
	}

	// Check if this is an upgrade
	result := s.upgrades.ShouldUpgrade(current, td.ParsedInfo)
	if !result.ShouldUpgrade {
		return
	}

	log.Printf("Upgrade detected: %s -> %s (%s)", result.CurrentTier, result.NewTier, result.Reason)

	// Find and remove old files in the destination
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".mkv" || ext == ".mp4" || ext == ".avi" {
			oldPath := filepath.Join(destDir, entry.Name())
			s.upgrades.HandleOldFile(oldPath)
		}
	}
}

// handleImportFailure handles when import fails
func (s *Service) handleImportFailure(td *download.TrackedDownload, err error) {
	s.monitoring.MarkFailed(td, err.Error())

	// Update linked request
	if td.RequestID != nil {
		s.requests.MarkFailed(*td.RequestID, err.Error())
	}

	// Record in blocklist if we have parsed info
	if td.ParsedInfo != nil {
		s.db.AddToBlocklist(&database.BlocklistEntry{
			MediaID:      td.MediaID,
			MediaType:    &td.MediaType,
			ReleaseTitle: td.Title,
			ReleaseGroup: &td.ParsedInfo.ReleaseGroup,
			Reason:       "Import failed",
			ErrorMessage: strPtr(err.Error()),
		})
	}

	// Track group failures
	if td.ParsedInfo != nil && td.ParsedInfo.ReleaseGroup != "" {
		s.db.IncrementGroupFailures(td.ParsedInfo.ReleaseGroup)
	}

	// Delete from client if configured
	if s.deleteOnFail {
		s.removeFromClient(td, true)
	}

	// Search for alternative
	if s.searchAlternative && td.MediaID != nil {
		go s.searchAlternative_(*td.MediaID, td.MediaType)
	}
}

// handleReadyToRemove is called when a download has met seeding requirements
func (s *Service) handleReadyToRemove(td *download.TrackedDownload) {
	log.Printf("Download ready for removal (ratio: %.2f, time: %v): %s",
		td.Ratio, td.SeedingTime, td.Title)
	s.removeFromClient(td, false)
}

// removeFromClient removes a download from the client
func (s *Service) removeFromClient(td *download.TrackedDownload, deleteFiles bool) {
	clientConfig, err := s.db.GetDownloadClient(td.DownloadClientID)
	if err != nil {
		log.Printf("Error getting download client: %v", err)
		return
	}

	client, err := downloadclient.New(clientConfig)
	if err != nil {
		log.Printf("Error creating download client: %v", err)
		return
	}

	if err := client.DeleteDownload(td.ExternalID, deleteFiles); err != nil {
		log.Printf("Error removing from client: %v", err)
	} else {
		log.Printf("Removed from client: %s", td.Title)
	}
}

// GrabRelease sends a release to the download client and tracks it
func (s *Service) GrabRelease(result *indexer.ScoredSearchResult, mediaID int64, mediaType string, requestID *int64) error {
	var downloadURL string
	if result.MagnetLink != "" {
		downloadURL = result.MagnetLink
	} else {
		downloadURL = result.Link
	}

	isTorrent := result.IndexerType == "torznab" || result.MagnetLink != ""

	// Find appropriate client
	clients, err := s.db.GetEnabledDownloadClients()
	if err != nil {
		return err
	}

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
		return &importpkg.ImportError{Message: "No suitable download client configured"}
	}

	// Add to client
	client, err := downloadclient.New(targetClient)
	if err != nil {
		return err
	}

	category := targetClient.Category
	if isTorrent {
		err = client.AddTorrent(downloadURL, category)
	} else {
		err = client.AddNZB(downloadURL, category)
	}

	if err != nil {
		return err
	}

	// Record the grab
	s.db.AddGrabHistory(&database.GrabHistory{
		MediaID:           mediaID,
		MediaType:         mediaType,
		ReleaseTitle:      result.Title,
		IndexerID:         &result.IndexerID,
		IndexerName:       &result.IndexerName,
		Size:              result.Size,
		DownloadClientID:  &targetClient.ID,
		Status:            "grabbed",
		QualityResolution: &result.Resolution,
		QualitySource:     &result.Source,
		QualityCodec:      &result.Codec,
		QualityAudio:      &result.AudioCodec,
		ReleaseGroup:      &result.ReleaseGroup,
	})

	// Update request status if provided
	if requestID != nil {
		s.requests.MarkProcessing(*requestID)
	}

	log.Printf("Grabbed: %s (client: %s)", result.Title, targetClient.Name)
	return nil
}

// searchAlternative_ searches for an alternative release after failure
func (s *Service) searchAlternative_(mediaID int64, mediaType string) {
	log.Printf("Searching for alternative release for %s %d", mediaType, mediaID)

	if s.indexers == nil {
		return
	}

	wantedType := mediaType
	if mediaType == "episode" {
		wantedType = "show"
	}

	wanted, err := s.db.GetWantedByTmdb(wantedType, mediaID)
	if err != nil || wanted == nil {
		return
	}

	searchType := "movie"
	if wantedType == "show" {
		searchType = "tvsearch"
	}

	params := indexer.SearchParams{
		Query: wanted.Title,
		Type:  searchType,
		Limit: 50,
	}

	if wanted.TmdbID > 0 {
		params.TmdbID = strconv.FormatInt(wanted.TmdbID, 10)
	}

	results, err := s.indexers.Search(params)
	if err != nil || len(results) == 0 {
		return
	}

	var bestResult *indexer.ScoredSearchResult
	var bestScore int

	for _, result := range results {
		blocked, _ := s.db.IsReleaseBlocklisted(result.Title)
		if blocked {
			continue
		}

		parsed := parser.Parse(result.Title)
		qualityTier := quality.ComputeQualityTier(parsed)
		baseScore := quality.BaseQualityScores[qualityTier]

		if baseScore > bestScore {
			bestScore = baseScore
			var hdrSlice []string
			if parsed.HDR != "" {
				hdrSlice = []string{parsed.HDR}
			}
			scored := &indexer.ScoredSearchResult{
				SearchResult: result,
				Quality:      qualityTier,
				Resolution:   parsed.Resolution,
				Source:       parsed.Source,
				Codec:        parsed.Codec,
				AudioCodec:   parsed.AudioFormat,
				HDR:          hdrSlice,
				ReleaseGroup: parsed.ReleaseGroup,
				BaseScore:    baseScore,
				TotalScore:   baseScore,
			}
			bestResult = scored
		}
	}

	if bestResult == nil {
		return
	}

	if err := s.GrabRelease(bestResult, mediaID, mediaType, nil); err != nil {
		log.Printf("Failed to grab alternative: %v", err)
	}
}

// GetActiveDownloads returns all active tracked downloads
func (s *Service) GetActiveDownloads() ([]*download.TrackedDownload, error) {
	return s.monitoring.GetActiveDownloads()
}

// GetTrackedDownload returns a specific tracked download
func (s *Service) GetTrackedDownload(id int64) (*download.TrackedDownload, error) {
	return s.monitoring.GetTrackedDownload(id)
}

// DeleteTrackedDownload removes a tracked download, optionally deleting from client
func (s *Service) DeleteTrackedDownload(id int64, deleteFromClient bool, deleteFiles bool) error {
	td, err := s.monitoring.GetTrackedDownload(id)
	if err != nil {
		return err
	}
	if td == nil {
		return nil // Already deleted
	}

	// Remove from download client if requested
	if deleteFromClient {
		s.removeFromClient(td, deleteFiles)
	}

	// Delete from database
	return s.monitoring.DeleteTrackedDownload(id)
}

// --- Helper functions ---

func (s *Service) getDestinationLibrary(td *download.TrackedDownload) (*database.Library, error) {
	libraries, err := s.db.GetLibraries()
	if err != nil {
		return nil, err
	}

	targetType := "movies"
	if td.MediaType == "show" || td.MediaType == "episode" {
		targetType = "tv"
	}

	for _, lib := range libraries {
		if lib.Type == targetType {
			return &lib, nil
		}
	}

	if len(libraries) > 0 {
		return &libraries[0], nil
	}

	return nil, &importpkg.ImportError{Message: "No library configured"}
}

func (s *Service) generateDestPath(td *download.TrackedDownload, library *database.Library, file *importpkg.FileDecision) (string, error) {
	parsed := td.ParsedInfo
	if parsed == nil {
		parsed = parser.Parse(td.Title)
	}

	ext := filepath.Ext(file.FilePath)
	year := ""
	if parsed.Year > 0 {
		year = strconv.Itoa(parsed.Year)
	}

	if td.MediaType == "movie" {
		folderName := parsed.Title
		if year != "" {
			folderName = parsed.Title + " (" + year + ")"
		}
		fileName := folderName + ext
		return filepath.Join(library.Path, folderName, fileName), nil
	}

	// TV show
	showFolder := parsed.Title
	if year != "" {
		showFolder = parsed.Title + " (" + year + ")"
	}

	seasonFolder := "Season " + strconv.Itoa(parsed.Season)
	if parsed.Season == 0 {
		seasonFolder = "Season 1"
	}

	episodeFile := parsed.Title
	if parsed.Season > 0 && parsed.Episode > 0 {
		episodeFile = parsed.Title + " - S" + padZero(parsed.Season) + "E" + padZero(parsed.Episode)
	}
	episodeFile += ext

	return filepath.Join(library.Path, showFolder, seasonFolder, episodeFile), nil
}

func (s *Service) updateQualityStatus(mediaID int64, mediaType string, parsed *parser.ParsedRelease) {
	if parsed == nil {
		return
	}

	s.db.UpsertMediaQualityStatus(&database.MediaQualityStatus{
		MediaID:           mediaID,
		MediaType:         mediaType,
		CurrentResolution: &parsed.Resolution,
		CurrentSource:     &parsed.Source,
		CurrentHDR:        &parsed.HDR,
		CurrentAudio:      &parsed.AudioFormat,
		TargetMet:         true,
	})
}

func (s *Service) cleanupSource(sourcePath string) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return
	}

	if info.IsDir() {
		os.RemoveAll(sourcePath)
	} else {
		os.Remove(sourcePath)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string {
	return &s
}

func padZero(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func findSubtitles(dir string) []string {
	var subs []string
	subExts := []string{".srt", ".sub", ".ass", ".ssa", ".vtt"}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		for _, subExt := range subExts {
			if ext == subExt {
				subs = append(subs, path)
				break
			}
		}
		return nil
	})

	return subs
}

func generateSubtitlePath(videoPath, subPath string) string {
	videoBase := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	subExt := filepath.Ext(subPath)
	subName := strings.TrimSuffix(filepath.Base(subPath), subExt)

	// Try to extract language code
	lang := ""
	parts := strings.Split(subName, ".")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) == 2 || len(lastPart) == 3 {
			lang = "." + lastPart
		}
	}

	return videoBase + lang + subExt
}
