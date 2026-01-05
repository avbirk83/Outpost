package acquisition

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
	"github.com/outpost/outpost/internal/parser"
	"github.com/outpost/outpost/internal/quality"
)

// Service handles download tracking, import, and failed download handling
type Service struct {
	db        *database.Database
	downloads *downloadclient.Manager
	indexers  *indexer.Manager

	pollInterval       time.Duration
	stalledThreshold   time.Duration
	autoBlockAfter     int
	deleteOnFail       bool
	searchAlternative  bool

	stopCh   chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// Config holds configuration for the acquisition service
type Config struct {
	PollInterval       time.Duration
	StalledThreshold   time.Duration
	AutoBlockAfter     int  // Auto-block group after N failures
	DeleteOnFail       bool // Delete files on failed download
	SearchAlternative  bool // Search for alternative on failure
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		PollInterval:      30 * time.Second,
		StalledThreshold:  6 * time.Hour,
		AutoBlockAfter:    3,
		DeleteOnFail:      true,
		SearchAlternative: true,
	}
}

// NewService creates a new acquisition service
func NewService(db *database.Database, downloads *downloadclient.Manager, indexers *indexer.Manager, cfg *Config) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Service{
		db:                db,
		downloads:         downloads,
		indexers:          indexers,
		pollInterval:      cfg.PollInterval,
		stalledThreshold:  cfg.StalledThreshold,
		autoBlockAfter:    cfg.AutoBlockAfter,
		deleteOnFail:      cfg.DeleteOnFail,
		searchAlternative: cfg.SearchAlternative,
		stopCh:            make(chan struct{}),
	}
}

// Start begins the polling loop
func (s *Service) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.pollLoop()

	log.Println("Acquisition service started")
}

// Stop stops the polling loop
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()

	log.Println("Acquisition service stopped")
}

// pollLoop runs the main polling loop
func (s *Service) pollLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.checkDownloads()
	s.processDelayedGrabs()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkDownloads()
			s.processDelayedGrabs()
		}
	}
}

// checkDownloads polls all download clients and processes downloads
func (s *Service) checkDownloads() {
	downloads, err := s.downloads.GetAllDownloads()
	if err != nil {
		log.Printf("Error getting downloads: %v", err)
		return
	}

	for _, dl := range downloads {
		s.processDownload(dl)
	}

	// Check for stalled downloads
	s.checkStalledDownloads()
}

// processDownload handles a single download from a client
func (s *Service) processDownload(dl downloadclient.Download) {
	// Check if we're tracking this download
	existing, err := s.db.GetDownloadByExternalID(dl.ClientID, dl.ID)
	if err != nil {
		log.Printf("Error checking download: %v", err)
		return
	}

	if existing == nil {
		// New download - try to match to wanted item
		s.handleNewDownload(dl)
		return
	}

	// Update progress
	existing.Progress = dl.Progress
	existing.Status = mapStatus(dl.Status)
	if err := s.db.UpdateDownload(existing); err != nil {
		log.Printf("Error updating download: %v", err)
	}

	// Check if completed
	if dl.Status == "completed" && existing.Status != "imported" {
		s.handleCompletedDownload(existing, dl.SavePath)
	}

	// Check if failed
	if dl.Status == "error" && existing.Status != "failed" {
		s.handleFailedDownload(existing, "Download client reported error")
	}
}

// handleNewDownload processes a new download found in a client
func (s *Service) handleNewDownload(dl downloadclient.Download) {
	// Parse the release name
	parsed := parser.Parse(dl.Name)

	// Try to match to a wanted item
	wanted, err := s.matchToWanted(parsed)
	if err != nil {
		log.Printf("Error matching download to wanted: %v", err)
	}

	// Create download record
	download := &database.Download{
		DownloadClientID: &dl.ClientID,
		ExternalID:       dl.ID,
		Title:            dl.Name,
		Size:             dl.Size,
		Status:           mapStatus(dl.Status),
		Progress:         dl.Progress,
		DownloadPath:     &dl.SavePath,
	}

	if wanted != nil {
		download.MediaID = &wanted.TmdbID
		download.MediaType = &wanted.Type
	}

	if err := s.db.CreateDownload(download); err != nil {
		log.Printf("Error creating download record: %v", err)
	}
}

// matchToWanted tries to match a parsed release to a wanted item
func (s *Service) matchToWanted(parsed *parser.ParsedRelease) (*database.WantedItem, error) {
	wanted, err := s.db.GetMonitoredItems()
	if err != nil {
		return nil, err
	}

	for _, w := range wanted {
		// Compare title (simplified matching)
		if strings.EqualFold(normalizeTitle(parsed.Title), normalizeTitle(w.Title)) {
			// Check year if available
			if parsed.Year > 0 && w.Year > 0 && parsed.Year == w.Year {
				return &w, nil
			}
			if parsed.Year == 0 || w.Year == 0 {
				return &w, nil
			}
		}
	}

	return nil, nil
}

// handleCompletedDownload processes a completed download
func (s *Service) handleCompletedDownload(download *database.Download, sourcePath string) {
	log.Printf("Processing completed download: %s", download.Title)

	download.Status = "importing"
	if err := s.db.UpdateDownload(download); err != nil {
		log.Printf("Error updating download status: %v", err)
	}

	// Find video files
	files, err := findVideoFiles(sourcePath)
	if err != nil {
		s.failImport(download, err)
		return
	}

	if len(files) == 0 {
		s.failImport(download, &ImportError{Message: "No video files found"})
		return
	}

	// Get destination library
	library, err := s.getDestinationLibrary(download)
	if err != nil {
		s.failImport(download, err)
		return
	}

	// Parse release for quality info
	parsed := parser.Parse(download.Title)

	// Select main file (largest video file)
	mainFile := selectMainFile(files)

	// Generate destination path
	destPath, err := s.generateDestPath(download, library, parsed, mainFile)
	if err != nil {
		s.failImport(download, err)
		return
	}

	// Create folder structure
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		s.failImport(download, err)
		return
	}

	// Move main file
	if err := moveFile(mainFile, destPath); err != nil {
		s.failImport(download, err)
		return
	}

	// Handle extras
	extras := findExtras(files, mainFile)
	if len(extras) > 0 {
		extrasDir := filepath.Join(destDir, "Extras")
		os.MkdirAll(extrasDir, 0755)
		for _, extra := range extras {
			moveFile(extra, filepath.Join(extrasDir, filepath.Base(extra)))
		}
	}

	// Handle subtitles
	subs := findSubtitles(sourcePath)
	for _, sub := range subs {
		subDest := generateSubtitlePath(destPath, sub)
		moveFile(sub, subDest)
	}

	// Update download record
	download.Status = "imported"
	download.ImportedPath = &destPath
	if err := s.db.UpdateDownload(download); err != nil {
		log.Printf("Error updating download: %v", err)
	}

	// Update media quality status
	if download.MediaID != nil && download.MediaType != nil {
		s.updateQualityStatus(*download.MediaID, *download.MediaType, parsed)
	}

	// Create import history
	s.db.CreateImportHistory(&database.ImportHistory{
		DownloadID: &download.ID,
		SourcePath: sourcePath,
		DestPath:   destPath,
		MediaID:    download.MediaID,
		MediaType:  download.MediaType,
		Success:    true,
	})

	// Clean up source directory
	cleanupSource(sourcePath)

	log.Printf("Successfully imported: %s -> %s", download.Title, destPath)
}

// failImport handles import failure
func (s *Service) failImport(download *database.Download, err error) {
	log.Printf("Import failed for %s: %v", download.Title, err)

	errMsg := err.Error()
	download.Status = "failed"
	download.LastError = &errMsg
	now := time.Now()
	download.FailedAt = &now

	if updateErr := s.db.UpdateDownload(download); updateErr != nil {
		log.Printf("Error updating download: %v", updateErr)
	}

	// Create import history
	s.db.CreateImportHistory(&database.ImportHistory{
		DownloadID: &download.ID,
		SourcePath: *download.DownloadPath,
		DestPath:   "",
		MediaID:    download.MediaID,
		MediaType:  download.MediaType,
		Success:    false,
		Error:      &errMsg,
	})
}

// handleFailedDownload processes a failed download
func (s *Service) handleFailedDownload(download *database.Download, reason string) {
	log.Printf("Handling failed download: %s - %s", download.Title, reason)

	// Update status
	download.Status = "failed"
	download.LastError = &reason
	now := time.Now()
	download.FailedAt = &now
	retryCount := 0
	if download.RetryCount != nil {
		retryCount = *download.RetryCount + 1
	}
	download.RetryCount = &retryCount

	if err := s.db.UpdateDownload(download); err != nil {
		log.Printf("Error updating download: %v", err)
	}

	// Parse release for group info
	parsed := parser.Parse(download.Title)

	// Add to blocklist
	s.db.AddToBlocklist(&database.BlocklistEntry{
		MediaID:      download.MediaID,
		MediaType:    download.MediaType,
		ReleaseTitle: download.Title,
		ReleaseGroup: &parsed.ReleaseGroup,
		Reason:       "failed_download",
		ErrorMessage: &reason,
	})

	// Track group failures
	if parsed.ReleaseGroup != "" {
		if err := s.db.IncrementGroupFailures(parsed.ReleaseGroup); err != nil {
			log.Printf("Error incrementing group failures: %v", err)
		}

		// Check if we should auto-block the group
		groups, _ := s.db.GetBlockedGroups()
		for _, g := range groups {
			if strings.EqualFold(g.Name, parsed.ReleaseGroup) && g.FailureCount >= s.autoBlockAfter {
				log.Printf("Auto-blocking group %s after %d failures", parsed.ReleaseGroup, g.FailureCount)
			}
		}
	}

	// Delete from download client if configured
	if s.deleteOnFail {
		s.deleteFromClient(download)
	}

	// Search for alternative if configured
	if s.searchAlternative && download.MediaID != nil && download.MediaType != nil {
		go s.searchAlternative_(*download.MediaID, *download.MediaType)
	}
}

// deleteFromClient removes a download from the client
func (s *Service) deleteFromClient(download *database.Download) {
	clientConfig, err := s.db.GetDownloadClient(download.DownloadClientID)
	if err != nil {
		return
	}

	client, err := downloadclient.New(clientConfig)
	if err != nil {
		return
	}

	if err := client.DeleteDownload(download.ExternalID, s.deleteOnFail); err != nil {
		log.Printf("Error deleting download from client: %v", err)
	}
}

// searchAlternative_ searches for an alternative release after a download failure
func (s *Service) searchAlternative_(mediaID int64, mediaType string) {
	log.Printf("Searching for alternative release for %s %d", mediaType, mediaID)

	if s.indexers == nil {
		log.Printf("Acquisition: no indexers configured")
		return
	}

	// Get the wanted item info
	wantedType := mediaType
	if mediaType == "episode" {
		wantedType = "show"
	}
	wanted, err := s.db.GetWantedByTmdb(wantedType, mediaID)
	if err != nil || wanted == nil {
		log.Printf("Acquisition: failed to get wanted item for TMDB %d: %v", mediaID, err)
		return
	}

	// Build search params
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

	// Search indexers
	results, err := s.indexers.Search(params)
	if err != nil {
		log.Printf("Acquisition: search failed for %s: %v", wanted.Title, err)
		return
	}

	if len(results) == 0 {
		log.Printf("Acquisition: no results for %s", wanted.Title)
		return
	}

	// Score results and filter out blocklisted
	var bestResult *indexer.ScoredSearchResult
	var bestScore int

	for _, result := range results {
		// Check blocklist
		blocked, _ := s.db.IsReleaseBlocklisted(result.Title)
		if blocked {
			log.Printf("Acquisition: skipping blocklisted release: %s", result.Title)
			continue
		}

		// Parse and score
		parsed := parser.Parse(result.Title)
		qualityTier := quality.ComputeQualityTier(parsed)
		baseScore := quality.BaseQualityScores[qualityTier]

		if baseScore > bestScore {
			bestScore = baseScore
			// Convert HDR string to slice (indexer uses []string)
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
				AudioFeature: parsed.AudioChannels,
				HDR:          hdrSlice,
				ReleaseGroup: parsed.ReleaseGroup,
				Proper:       parsed.IsProper,
				Repack:       parsed.IsRepack,
				BaseScore:    baseScore,
				TotalScore:   baseScore,
			}
			bestResult = scored
		}
	}

	if bestResult == nil {
		log.Printf("Acquisition: no acceptable alternative found for %s", wanted.Title)
		return
	}

	// Grab the best result
	err = s.grabRelease(bestResult)
	if err != nil {
		log.Printf("Acquisition: failed to grab alternative for %s: %v", wanted.Title, err)
		return
	}

	log.Printf("Acquisition: grabbed alternative release: %s (score: %d)", bestResult.Title, bestResult.TotalScore)
}

// grabRelease sends a release to the download client
func (s *Service) grabRelease(result *indexer.ScoredSearchResult) error {
	var downloadURL string
	if result.MagnetLink != "" {
		downloadURL = result.MagnetLink
	} else {
		downloadURL = result.Link
	}

	isTorrent := result.IndexerType == "torznab" || result.MagnetLink != ""

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
		return fmt.Errorf("no suitable download client for release")
	}

	if isTorrent {
		return s.downloads.AddTorrent(targetClient.ID, downloadURL, result.Category)
	}
	return s.downloads.AddNZB(targetClient.ID, downloadURL, result.Category)
}

// checkStalledDownloads checks for downloads with no progress
func (s *Service) checkStalledDownloads() {
	downloads, err := s.db.GetDownloads()
	if err != nil {
		return
	}

	for _, dl := range downloads {
		if dl.Status != "downloading" {
			continue
		}

		// Check if stalled (no progress for threshold time)
		if dl.UpdatedAt != nil {
			timeSinceUpdate := time.Since(*dl.UpdatedAt)
			if timeSinceUpdate > s.stalledThreshold {
				if dl.StalledNotified == nil || !*dl.StalledNotified {
					log.Printf("Download stalled: %s (no progress for %v)", dl.Title, timeSinceUpdate)
					notified := true
					dl.StalledNotified = &notified
					s.db.UpdateDownload(&dl)
				}
			}
		}
	}
}

// processDelayedGrabs processes pending grabs whose delay has expired
func (s *Service) processDelayedGrabs() {
	pending, err := s.db.GetReadyPendingGrabs()
	if err != nil {
		log.Printf("Error getting pending grabs: %v", err)
		return
	}

	for _, pg := range pending {
		log.Printf("Processing delayed grab: %s", pg.ReleaseTitle)

		// Parse the stored release data
		var releaseData map[string]interface{}
		if pg.ReleaseData != nil {
			json.Unmarshal([]byte(*pg.ReleaseData), &releaseData)
		}

		// Remove from pending
		s.db.RemovePendingGrab(pg.ID)

		// The actual grab would be triggered here
		// This integrates with the search/grab flow
	}
}

// getDestinationLibrary finds the appropriate library for a download
func (s *Service) getDestinationLibrary(download *database.Download) (*database.Library, error) {
	libraries, err := s.db.GetLibraries()
	if err != nil {
		return nil, err
	}

	// Find library matching media type
	mediaType := "movies"
	if download.MediaType != nil {
		mediaType = *download.MediaType
	}

	for _, lib := range libraries {
		if lib.Type == mediaType || (mediaType == "show" && lib.Type == "tv") {
			return &lib, nil
		}
	}

	// Fall back to first library
	if len(libraries) > 0 {
		return &libraries[0], nil
	}

	return nil, &ImportError{Message: "No library found"}
}

// generateDestPath generates the destination path for an import
func (s *Service) generateDestPath(download *database.Download, library *database.Library, parsed *parser.ParsedRelease, sourceFile string) (string, error) {
	// Get naming template
	templateType := "movie"
	if download.MediaType != nil && *download.MediaType == "episode" {
		if parsed.IsDailyShow {
			templateType = "daily"
		} else {
			templateType = "tv"
		}
	}

	template, err := s.db.GetNamingTemplate(templateType)
	if err != nil || template == nil {
		// Use default template
		template = &database.NamingTemplate{
			Type:           templateType,
			FolderTemplate: "{Title} ({Year})",
			FileTemplate:   "{Title} ({Year})",
		}
	}

	// Replace placeholders
	folder := replacePlaceholders(template.FolderTemplate, parsed)
	file := replacePlaceholders(template.FileTemplate, parsed)

	// Get extension from source file
	ext := filepath.Ext(sourceFile)

	return filepath.Join(library.Path, folder, file+ext), nil
}

// updateQualityStatus updates the quality status for imported media
func (s *Service) updateQualityStatus(mediaID int64, mediaType string, parsed *parser.ParsedRelease) {
	status := &database.MediaQualityStatus{
		MediaID:           mediaID,
		MediaType:         mediaType,
		CurrentResolution: &parsed.Resolution,
		CurrentSource:     &parsed.Source,
		CurrentHDR:        &parsed.HDR,
		CurrentAudio:      &parsed.AudioFormat,
	}

	// Check if target is met
	preset, err := s.db.GetDefaultQualityPreset()
	if err == nil && preset != nil {
		targetMet := quality.MeetsCutoff(parsed, &quality.Preset{
			CutoffResolution: preset.CutoffResolution,
			CutoffSource:     preset.CutoffSource,
		})
		status.TargetMet = &targetMet
	}

	now := time.Now()
	status.UpdatedAt = &now

	if err := s.db.UpsertMediaQualityStatus(status); err != nil {
		log.Printf("Error updating quality status: %v", err)
	}
}

// ShouldPauseDownloads checks if downloads should be paused due to low storage
func (s *Service) ShouldPauseDownloads() bool {
	settings, err := s.db.GetAllSettings()
	if err != nil {
		return false
	}

	pauseEnabled := settings["storage_pause_enabled"] == "true"
	if !pauseEnabled {
		return false
	}

	// Check storage on all libraries
	libraries, err := s.db.GetLibraries()
	if err != nil {
		return false
	}

	thresholdGB := int64(100)
	if val, ok := settings["storage_threshold_gb"]; ok {
		var parsed int64
		if _, err := json.Unmarshal([]byte(val), &parsed); err == nil {
			thresholdGB = parsed
		}
	}

	for _, lib := range libraries {
		freeGB := getFreeSpaceGB(lib.Path)
		if freeGB < thresholdGB {
			log.Printf("Low disk space on %s: %d GB free (threshold: %d GB)", lib.Path, freeGB, thresholdGB)
			return true
		}
	}

	return false
}

// Helper types and functions

type ImportError struct {
	Message string
}

func (e *ImportError) Error() string {
	return e.Message
}

func mapStatus(clientStatus string) string {
	switch clientStatus {
	case "downloading":
		return "downloading"
	case "paused":
		return "paused"
	case "completed":
		return "completed"
	case "error":
		return "failed"
	case "queued":
		return "queued"
	default:
		return clientStatus
	}
}

func normalizeTitle(title string) string {
	// Remove special characters and normalize spaces
	re := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	normalized := re.ReplaceAllString(title, "")
	return strings.ToLower(strings.TrimSpace(normalized))
}

func findVideoFiles(path string) ([]string, error) {
	var files []string
	videoExts := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".m4v": true,
		".mov": true, ".wmv": true, ".ts": true, ".m2ts": true,
	}

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(p))
			if videoExts[ext] {
				files = append(files, p)
			}
		}
		return nil
	})

	return files, err
}

func selectMainFile(files []string) string {
	var largest string
	var largestSize int64

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.Size() > largestSize {
			largestSize = info.Size()
			largest = f
		}
	}

	return largest
}

var extrasPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)extras?`),
	regexp.MustCompile(`(?i)featurettes?`),
	regexp.MustCompile(`(?i)bonus`),
	regexp.MustCompile(`(?i)deleted.?scenes?`),
	regexp.MustCompile(`(?i)behind.?the.?scenes?`),
	regexp.MustCompile(`(?i)making.?of`),
	regexp.MustCompile(`(?i)interview`),
	regexp.MustCompile(`(?i)trailer`),
	regexp.MustCompile(`(?i)gag.?reel`),
	regexp.MustCompile(`(?i)bloopers?`),
}

func findExtras(files []string, mainFile string) []string {
	var extras []string

	for _, f := range files {
		if f == mainFile {
			continue
		}

		for _, pattern := range extrasPatterns {
			if pattern.MatchString(f) {
				extras = append(extras, f)
				break
			}
		}
	}

	return extras
}

func findSubtitles(path string) []string {
	var subs []string
	subExts := map[string]bool{
		".srt": true, ".sub": true, ".ass": true, ".ssa": true, ".vtt": true,
	}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(p))
			if subExts[ext] {
				subs = append(subs, p)
			}
		}
		return nil
	})

	return subs
}

func generateSubtitlePath(videoPath, subPath string) string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	ext := filepath.Ext(subPath)

	// Try to detect language from subtitle filename
	subBase := strings.ToLower(filepath.Base(subPath))
	lang := ""
	if strings.Contains(subBase, "eng") || strings.Contains(subBase, "english") {
		lang = ".en"
	} else if strings.Contains(subBase, "spa") || strings.Contains(subBase, "spanish") {
		lang = ".es"
	} else if strings.Contains(subBase, "fre") || strings.Contains(subBase, "french") {
		lang = ".fr"
	}

	return filepath.Join(dir, base+lang+ext)
}

func moveFile(src, dst string) error {
	// Try rename first (same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to copy + delete
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return err
	}

	return os.Remove(src)
}

func cleanupSource(path string) {
	// Remove empty directories
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			os.Remove(p) // Will only succeed if empty
		}
		return nil
	})
}

func replacePlaceholders(template string, parsed *parser.ParsedRelease) string {
	result := template

	result = strings.ReplaceAll(result, "{Title}", sanitizeFilename(parsed.Title))
	result = strings.ReplaceAll(result, "{Year}", string(rune(parsed.Year+'0')))

	if parsed.Year > 0 {
		result = strings.ReplaceAll(result, "{Year}", fmt.Sprintf("%d", parsed.Year))
	} else {
		result = strings.ReplaceAll(result, " ({Year})", "")
		result = strings.ReplaceAll(result, "{Year}", "")
	}

	if parsed.Season > 0 {
		result = strings.ReplaceAll(result, "{Season:00}", fmt.Sprintf("%02d", parsed.Season))
		result = strings.ReplaceAll(result, "{Season}", fmt.Sprintf("%d", parsed.Season))
	}

	if parsed.Episode > 0 {
		result = strings.ReplaceAll(result, "{Episode:00}", fmt.Sprintf("%02d", parsed.Episode))
		result = strings.ReplaceAll(result, "{Episode}", fmt.Sprintf("%d", parsed.Episode))
	}

	return result
}

func sanitizeFilename(s string) string {
	// Remove invalid characters: / \ : * ? " < > |
	invalid := regexp.MustCompile(`[/\\:*?"<>|]`)
	return invalid.ReplaceAllString(s, "")
}

func getFreeSpaceGB(path string) int64 {
	// This is a placeholder - actual implementation would use syscall
	// For now, return a large number to not block
	return 1000
}
