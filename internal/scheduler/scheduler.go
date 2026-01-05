package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/outpost/outpost/internal/scanner"
	"github.com/outpost/outpost/internal/storage"
)

type Scheduler struct {
	db        *database.Database
	indexers  *indexer.Manager
	downloads *downloadclient.Manager
	scanner   *scanner.Scanner

	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    bool
	mu         sync.Mutex

	// Configurable intervals (in minutes)
	searchInterval int
	rssInterval    int

	// Task tracking
	taskRunning map[string]bool
	taskMu      sync.RWMutex
}

func New(db *database.Database, indexers *indexer.Manager, downloads *downloadclient.Manager, scan *scanner.Scanner) *Scheduler {
	s := &Scheduler{
		db:             db,
		indexers:       indexers,
		downloads:      downloads,
		scanner:        scan,
		stopChan:       make(chan struct{}),
		searchInterval: 60, // Default: search every 60 minutes
		rssInterval:    15, // Default: check RSS every 15 minutes
		taskRunning:    make(map[string]bool),
	}
	s.initDefaultTasks()
	return s
}

// initDefaultTasks creates default task entries in the database
func (s *Scheduler) initDefaultTasks() {
	defaultTasks := []database.ScheduledTask{
		{
			Name:            "Search Monitored",
			Description:     "Search indexers for monitored movies and shows",
			TaskType:        "search",
			Enabled:         true,
			IntervalMinutes: 60,
		},
		{
			Name:            "RSS Sync",
			Description:     "Check RSS feeds for new releases",
			TaskType:        "rss",
			Enabled:         true,
			IntervalMinutes: 15,
		},
		{
			Name:            "Import Downloads",
			Description:     "Check download clients and import completed items",
			TaskType:        "import",
			Enabled:         true,
			IntervalMinutes: 1,
		},
		{
			Name:            "Cleanup",
			Description:     "Clean up old history, logs, and temporary files",
			TaskType:        "cleanup",
			Enabled:         true,
			IntervalMinutes: 1440, // 24 hours
		},
		{
			Name:            "Refresh Metadata",
			Description:     "Refresh metadata for items missing info",
			TaskType:        "metadata_refresh",
			Enabled:         true,
			IntervalMinutes: 360, // 6 hours
		},
		{
			Name:            "Library Scan",
			Description:     "Scan library folders for new and changed files",
			TaskType:        "library_scan",
			Enabled:         true,
			IntervalMinutes: 60, // 1 hour
		},
	}

	for _, task := range defaultTasks {
		if err := s.db.UpsertTask(&task); err != nil {
			log.Printf("Failed to create task %s: %v", task.Name, err)
		} else {
			log.Printf("Created task: %s (ID: %d)", task.Name, task.ID)
		}
	}
}

func (s *Scheduler) SetSearchInterval(minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if minutes > 0 {
		s.searchInterval = minutes
	}
}

func (s *Scheduler) SetRSSInterval(minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if minutes > 0 {
		s.rssInterval = minutes
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	// Load intervals from settings
	s.loadIntervals()

	// Start the search job
	s.wg.Add(1)
	go s.runSearchJob()

	// Start the RSS job
	s.wg.Add(1)
	go s.runRSSJob()

	log.Printf("Scheduler started (search: %dm, rss: %dm)", s.searchInterval, s.rssInterval)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) loadIntervals() {
	if val, err := s.db.GetSetting("scheduler_search_interval"); err == nil && val != "" {
		if mins, err := strconv.Atoi(val); err == nil && mins > 0 {
			s.searchInterval = mins
		}
	}
	if val, err := s.db.GetSetting("scheduler_rss_interval"); err == nil && val != "" {
		if mins, err := strconv.Atoi(val); err == nil && mins > 0 {
			s.rssInterval = mins
		}
	}
}

// GetStatus returns all tasks with their current running status
func (s *Scheduler) GetStatus() []database.ScheduledTask {
	tasks, _ := s.db.GetAllTasks()

	s.taskMu.RLock()
	defer s.taskMu.RUnlock()

	for i := range tasks {
		if running, ok := s.taskRunning[tasks[i].Name]; ok {
			tasks[i].IsRunning = running
		}
	}

	return tasks
}

// TriggerTask manually triggers a task by ID
func (s *Scheduler) TriggerTask(taskID int64) error {
	task, err := s.db.GetTask(taskID)
	if err != nil {
		return err
	}

	go s.executeTask(task)
	return nil
}

// UpdateTask updates task settings and restarts if needed
func (s *Scheduler) UpdateTask(taskID int64, enabled bool, intervalMinutes int) error {
	task, err := s.db.GetTask(taskID)
	if err != nil {
		return err
	}

	task.Enabled = enabled
	task.IntervalMinutes = intervalMinutes

	return s.db.UpdateTask(task)
}

// executeTask runs a task and records the result
func (s *Scheduler) executeTask(task *database.ScheduledTask) {
	// Check if already running
	s.taskMu.Lock()
	if s.taskRunning[task.Name] {
		s.taskMu.Unlock()
		return
	}
	s.taskRunning[task.Name] = true
	s.taskMu.Unlock()

	defer func() {
		s.taskMu.Lock()
		s.taskRunning[task.Name] = false
		s.taskMu.Unlock()
	}()

	startedAt := time.Now()
	var itemsProcessed, itemsFound int
	var taskError error

	log.Printf("Task started: %s (ID: %d, Type: %s)", task.Name, task.ID, task.TaskType)

	// Execute based on task type
	switch task.TaskType {
	case "search":
		itemsProcessed, itemsFound = s.runSearchTask()
	case "rss":
		itemsProcessed, itemsFound = s.runRSSTask()
	case "import":
		itemsProcessed = s.runImportTask()
	case "cleanup":
		itemsProcessed = s.runCleanupTask()
	case "metadata_refresh":
		itemsProcessed = s.runMetadataRefreshTask()
	case "library_scan":
		itemsProcessed = s.runLibraryScanTask()
	}

	finishedAt := time.Now()
	durationMs := finishedAt.Sub(startedAt).Milliseconds()

	log.Printf("Task completed: %s - processed: %d, found: %d, duration: %dms", task.Name, itemsProcessed, itemsFound, durationMs)

	status := "success"
	var errorMsg *string
	if taskError != nil {
		status = "failed"
		errStr := taskError.Error()
		errorMsg = &errStr
	}

	// Record run in history
	s.db.RecordTaskRun(task.ID, startedAt, finishedAt, status, itemsProcessed, itemsFound, errorMsg, nil)

	// Update task stats
	s.db.UpdateTaskStats(task.ID, status, durationMs, errorMsg)
}

// runSearchTask executes the search monitored items task
func (s *Scheduler) runSearchTask() (processed, found int) {
	// Check if auto-search is enabled
	autoSearch, _ := s.db.GetSetting("scheduler_auto_search")
	if autoSearch != "true" {
		return 0, 0
	}

	items, err := s.db.GetMonitoredItems()
	if err != nil {
		return 0, 0
	}

	for _, item := range items {
		if item.LastSearched != nil {
			hoursSinceLast := time.Since(*item.LastSearched).Hours()
			if hoursSinceLast < float64(s.searchInterval)/60.0 {
				continue
			}
		}

		s.searchAndGrab(&item)
		processed++
		time.Sleep(5 * time.Second)
	}

	return processed, found
}

// runRSSTask executes the RSS sync task
func (s *Scheduler) runRSSTask() (processed, found int) {
	rssEnabled, _ := s.db.GetSetting("scheduler_rss_enabled")
	if rssEnabled != "true" {
		return 0, 0
	}

	indexers, err := s.db.GetEnabledIndexers()
	if err != nil {
		return 0, 0
	}

	for _, idx := range indexers {
		results, err := s.indexers.FetchRSS(idx.ID)
		if err != nil {
			continue
		}
		processed++
		found += len(results)
	}

	return processed, found
}

// runImportTask checks for completed downloads
func (s *Scheduler) runImportTask() int {
	// This would check download clients for completed items
	// For now, just return 0 as actual import is done elsewhere
	return 0
}

// runCleanupTask cleans up old data
func (s *Scheduler) runCleanupTask() int {
	processed := 0

	// Cleanup task history older than 30 days
	if err := s.db.CleanupTaskHistory(30); err == nil {
		processed++
	}

	return processed
}

// runMetadataRefreshTask refreshes missing metadata
func (s *Scheduler) runMetadataRefreshTask() int {
	// This would refresh metadata for items missing info
	// Actual implementation depends on TMDB client availability
	return 0
}

// runLibraryScanTask scans all libraries for new files
func (s *Scheduler) runLibraryScanTask() int {
	if s.scanner == nil {
		log.Printf("Scheduler: scanner not available for library scan")
		return 0
	}

	libraries, err := s.db.GetLibraries()
	if err != nil {
		log.Printf("Scheduler: failed to get libraries: %v", err)
		return 0
	}

	scanned := 0
	for _, lib := range libraries {
		log.Printf("Scheduler: scanning library %s (%s)", lib.Name, lib.Path)
		if err := s.scanner.ScanLibrary(&lib); err != nil {
			log.Printf("Scheduler: failed to scan library %s: %v", lib.Name, err)
			continue
		}
		scanned++
	}

	return scanned
}
// SearchWantedItem searches for a specific wanted item by TMDB ID and type
func (s *Scheduler) SearchWantedItem(tmdbID int64, mediaType string) error {
	item, err := s.db.GetWantedByTmdb(mediaType, tmdbID)
	if err != nil {
		return fmt.Errorf("failed to find wanted item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("wanted item not found")
	}

	// Run search in background goroutine with task tracking
	go func() {
		taskName := "Request Search"
		s.taskMu.Lock()
		s.taskRunning[taskName] = true
		s.taskMu.Unlock()
		defer func() {
			s.taskMu.Lock()
			s.taskRunning[taskName] = false
			s.taskMu.Unlock()
		}()

		log.Printf("Scheduler: searching for requested item: %s (%s)", item.Title, mediaType)
		s.searchAndGrab(item)
	}()

	return nil
}


func (s *Scheduler) runSearchJob() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.searchInterval) * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	s.executeTaskByName("Search Monitored")

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.executeTaskByName("Search Monitored")
		}
	}
}

func (s *Scheduler) runRSSJob() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.rssInterval) * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	s.executeTaskByName("RSS Sync")

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.executeTaskByName("RSS Sync")
		}
	}
}

// executeTaskByName runs a task by name
func (s *Scheduler) executeTaskByName(name string) {
	task, err := s.db.GetTaskByName(name)
	if err != nil {
		log.Printf("Scheduler: task not found: %s", name)
		return
	}
	if !task.Enabled {
		return
	}
	s.executeTask(task)
}

func (s *Scheduler) searchMonitoredItems() {
	// Check if auto-search is enabled
	autoSearch, _ := s.db.GetSetting("scheduler_auto_search")
	if autoSearch != "true" {
		return
	}

	items, err := s.db.GetMonitoredItems()
	if err != nil {
		log.Printf("Scheduler: failed to get monitored items: %v", err)
		return
	}

	if len(items) == 0 {
		return
	}

	log.Printf("Scheduler: searching %d monitored items", len(items))

	for _, item := range items {
		// Check if we should search (based on last search time)
		if item.LastSearched != nil {
			hoursSinceLast := time.Since(*item.LastSearched).Hours()
			if hoursSinceLast < float64(s.searchInterval)/60.0 {
				continue
			}
		}

		s.searchAndGrab(&item)

		// Small delay between searches to avoid hammering indexers
		time.Sleep(5 * time.Second)
	}
}

func (s *Scheduler) searchAndGrab(item *database.WantedItem) {
	log.Printf("Scheduler: searchAndGrab started for %s", item.Title)

	// Check if downloads should be paused due to low storage
	if s.shouldPauseDownloads() {
		log.Printf("Scheduler: downloads paused due to low storage")
		return
	}

	// Check if this media is excluded
	excluded, _ := s.db.IsMediaExcluded(item.TmdbID, item.Type)
	if excluded {
		log.Printf("Scheduler: skipping excluded media: %s", item.Title)
		return
	}

	searchType := "movie"
	if item.Type == "show" {
		searchType = "tvsearch"
	}

	params := indexer.SearchParams{
		Query: item.Title,
		Type:  searchType,
		Limit: 50,
	}

	if item.TmdbID > 0 {
		params.TmdbID = strconv.FormatInt(item.TmdbID, 10)
	}

	log.Printf("Scheduler: searching with params: query=%s, type=%s, tmdbId=%s", params.Query, params.Type, params.TmdbID)

	// Get indexers for this media type based on library tags
	indexerIDs := s.getIndexerIDsForMediaType(item.Type)
	log.Printf("Scheduler: using %d indexer IDs for search", len(indexerIDs))

	var results []indexer.SearchResult
	var err error
	if len(indexerIDs) > 0 {
		results, err = s.indexers.SearchWithIndexerIDs(params, indexerIDs)
	} else {
		results, err = s.indexers.Search(params)
	}
	if err != nil {
		log.Printf("Scheduler: search failed for %s: %v", item.Title, err)
		return
	}

	log.Printf("Scheduler: found %d raw results for %s", len(results), item.Title)

	// Update last searched
	s.db.UpdateWantedLastSearched(item.ID)

	if len(results) == 0 {
		log.Printf("Scheduler: no results for %s", item.Title)
		return
	}

	// Check auto-grab setting
	autoGrab, _ := s.db.GetSetting("scheduler_auto_grab")
	if autoGrab != "true" {
		log.Printf("Scheduler: found %d results for %s (auto-grab disabled)", len(results), item.Title)
		return
	}

	// Get library ID for indexer exclusion check
	var libraryID int64
	libraries, _ := s.db.GetLibraries()
	libType := "movies"
	if item.Type == "show" {
		libType = "tv"
	}
	for _, lib := range libraries {
		if lib.Type == libType {
			libraryID = lib.ID
			break
		}
	}

	// Get all enabled quality presets for fallback
	allPresets, _ := s.db.GetQualityPresets()
	var presetsToTry []*int64

	// First try the item's assigned preset
	if item.QualityPresetID != nil && *item.QualityPresetID > 0 {
		presetsToTry = append(presetsToTry, item.QualityPresetID)
	}

	// Then add other enabled presets as fallbacks (sorted by ID for consistency)
	for i := range allPresets {
		if !allPresets[i].Enabled {
			continue
		}
		// Skip if already added as primary
		if item.QualityPresetID != nil && allPresets[i].ID == *item.QualityPresetID {
			continue
		}
		id := allPresets[i].ID
		presetsToTry = append(presetsToTry, &id)
	}

	// Try each preset until we find an acceptable result
	var bestResult *indexer.ScoredSearchResult
	var usedPresetID *int64
	for _, presetID := range presetsToTry {
		scoredResults := s.scoreResultsWithPreset(results, presetID)

		// Find best non-rejected, non-blocklisted result for this preset
		for i := range scoredResults {
			if scoredResults[i].Rejected || scoredResults[i].TotalScore <= 0 {
				continue
			}

			// Verify the release actually matches the wanted item (title + year)
			matches, reason := s.verifyReleaseMatch(scoredResults[i].Title, item)
			if !matches {
				log.Printf("Scheduler: rejecting %s - %s", scoredResults[i].Title, reason)
				continue
			}

			// Check blocklist
			blocked, _ := s.db.IsReleaseBlocklisted(scoredResults[i].Title)
			if blocked {
				continue
			}

			// Check if indexer is excluded for this library
			if libraryID > 0 {
				excluded, _ := s.db.IsIndexerExcludedForLibrary(scoredResults[i].IndexerID, libraryID)
				if excluded {
					continue
				}
			}

			// Check release filters (legacy profile support)
			if item.QualityProfileID > 0 && !s.passesReleaseFilters(scoredResults[i].Title, item.QualityProfileID) {
				continue
			}

			bestResult = &scoredResults[i]
			usedPresetID = presetID
			break
		}

		if bestResult != nil {
			// Found a result with this preset
			if presetID != item.QualityPresetID {
				presetName := "unknown"
				for _, p := range allPresets {
					if p.ID == *presetID {
						presetName = p.Name
						break
					}
				}
				log.Printf("Scheduler: using fallback preset '%s' for %s", presetName, item.Title)
			}
			break
		}
	}

	if bestResult == nil {
		log.Printf("Scheduler: no acceptable releases for %s after trying %d presets", item.Title, len(presetsToTry))
		return
	}
	_ = usedPresetID // Mark as used

	// Check if delay profile applies
	shouldDelay, availableAt := s.shouldDelayGrab(bestResult, libraryID)
	if shouldDelay {
		// Add to pending grabs instead of grabbing immediately
		releaseData := fmt.Sprintf(`{"indexerId":%d,"link":"%s","magnetLink":"%s","category":"%s"}`,
			bestResult.IndexerID, bestResult.Link, bestResult.MagnetLink, bestResult.Category)
		s.db.AddPendingGrab(&database.PendingGrab{
			MediaID:      item.TmdbID,
			MediaType:    item.Type,
			ReleaseTitle: bestResult.Title,
			ReleaseData:  &releaseData,
			Score:        bestResult.TotalScore,
			IndexerID:    &bestResult.IndexerID,
			AvailableAt:  availableAt,
		})
		log.Printf("Scheduler: delayed grab until %s: %s for %s", availableAt.Format(time.RFC3339), bestResult.Title, item.Title)
		return
	}

	// Grab the best result
	err = s.grabRelease(bestResult, item.Type)
	if err != nil {
		log.Printf("Scheduler: grab failed for %s: %v", item.Title, err)
		return
	}

	log.Printf("Scheduler: grabbed %s for %s (score: %d)", bestResult.Title, item.Title, bestResult.TotalScore)
}

func (s *Scheduler) scoreResults(results []indexer.SearchResult, profileID int64) []indexer.ScoredSearchResult {
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

	// Sort by score (descending)
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].TotalScore > scoredResults[i].TotalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	return scoredResults
}

func (s *Scheduler) grabRelease(result *indexer.ScoredSearchResult, mediaType string) error {
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
		return nil // No suitable client, silently skip
	}

	// Generate category based on media type
	category := getCategoryForMediaType(mediaType)

	if isTorrent {
		return s.downloads.AddTorrent(targetClient.ID, downloadURL, category)
	}
	return s.downloads.AddNZB(targetClient.ID, downloadURL, category)
}

// getCategoryForMediaType returns the download client category for a given media type
func getCategoryForMediaType(mediaType string) string {
	switch mediaType {
	case "movie":
		return "movies-outpost"
	case "show", "tv":
		return "tv-outpost"
	case "anime":
		return "anime-outpost"
	case "music":
		return "music-outpost"
	case "book":
		return "books-outpost"
	default:
		return "outpost"
	}
}

// passesReleaseFilters checks if a release passes the configured filters for a quality profile
func (s *Scheduler) passesReleaseFilters(releaseName string, presetID int64) bool {
	filters, err := s.db.GetReleaseFilters(presetID)
	if err != nil || len(filters) == 0 {
		return true // No filters means all releases pass
	}

	releaseNameLower := strings.ToLower(releaseName)

	for _, filter := range filters {
		var matches bool
		valueLower := strings.ToLower(filter.Value)

		if filter.IsRegex {
			// Use regex matching
			re, err := regexp.Compile("(?i)" + filter.Value)
			if err != nil {
				log.Printf("Scheduler: invalid release filter regex: %s", filter.Value)
				continue
			}
			matches = re.MatchString(releaseName)
		} else {
			// Simple string contains matching
			matches = strings.Contains(releaseNameLower, valueLower)
		}

		switch filter.FilterType {
		case "must_contain":
			if !matches {
				log.Printf("Scheduler: release rejected (must_contain: %s): %s", filter.Value, releaseName)
				return false
			}
		case "must_not_contain":
			if matches {
				log.Printf("Scheduler: release rejected (must_not_contain: %s): %s", filter.Value, releaseName)
				return false
			}
		}
	}

	return true
}

// matchesPreset checks if a parsed release matches the quality preset criteria
func (s *Scheduler) matchesPreset(parsed *parser.ParsedRelease, preset *database.QualityPreset) (bool, string) {
	if preset == nil {
		return true, "" // No preset means accept all
	}

	// Check resolution
	if preset.Resolution != "" && preset.Resolution != "any" {
		releaseRes := strings.ToLower(parsed.Resolution)
		presetRes := strings.ToLower(preset.Resolution)

		// Normalize resolution names
		resMap := map[string]string{
			"4k": "2160p", "uhd": "2160p", "2160p": "2160p",
			"1080p": "1080p", "1080i": "1080p",
			"720p": "720p",
			"480p": "480p", "sd": "480p",
		}

		normalizedRelease := resMap[releaseRes]
		normalizedPreset := resMap[presetRes]

		if normalizedRelease == "" {
			normalizedRelease = releaseRes
		}
		if normalizedPreset == "" {
			normalizedPreset = presetRes
		}

		if normalizedRelease != normalizedPreset {
			return false, fmt.Sprintf("resolution mismatch: want %s, got %s", preset.Resolution, parsed.Resolution)
		}
	}

	// Check source
	if preset.Source != "" && preset.Source != "any" {
		releaseSource := strings.ToLower(parsed.Source)
		presetSource := strings.ToLower(preset.Source)

		// Normalize source names
		sourceMap := map[string]string{
			"remux": "remux",
			"bluray": "bluray", "bdrip": "bluray", "brrip": "bluray",
			"web": "web", "webdl": "web", "webrip": "web",
			"hdtv": "hdtv",
			"dvd": "dvd", "dvdrip": "dvd",
		}

		normalizedRelease := sourceMap[releaseSource]
		normalizedPreset := sourceMap[presetSource]

		if normalizedRelease == "" {
			normalizedRelease = releaseSource
		}
		if normalizedPreset == "" {
			normalizedPreset = presetSource
		}

		if normalizedRelease != normalizedPreset {
			return false, fmt.Sprintf("source mismatch: want %s, got %s", preset.Source, parsed.Source)
		}
	}

	// Check codec
	if preset.Codec != "" && preset.Codec != "any" {
		releaseCodec := strings.ToLower(parsed.Codec)
		presetCodec := strings.ToLower(preset.Codec)

		// Normalize codec names
		codecMap := map[string]string{
			"hevc": "hevc", "x265": "hevc", "h265": "hevc",
			"avc": "avc", "x264": "avc", "h264": "avc",
			"av1": "av1",
		}

		normalizedRelease := codecMap[releaseCodec]
		normalizedPreset := codecMap[presetCodec]

		if normalizedRelease == "" {
			normalizedRelease = releaseCodec
		}
		if normalizedPreset == "" {
			normalizedPreset = presetCodec
		}

		if normalizedRelease != normalizedPreset {
			return false, fmt.Sprintf("codec mismatch: want %s, got %s", preset.Codec, parsed.Codec)
		}
	}

	// Check HDR formats (if preset has specific requirements)
	if len(preset.HDRFormats) > 0 {
		releaseHDR := strings.ToLower(parsed.HDR)
		hasMatchingHDR := false
		for _, hdr := range preset.HDRFormats {
			if strings.EqualFold(hdr, releaseHDR) || strings.EqualFold(hdr, "any") {
				hasMatchingHDR = true
				break
			}
		}
		if !hasMatchingHDR && releaseHDR != "" {
			// Only reject if we have HDR requirements and release has different HDR
			// If release has no HDR, that's okay (SDR fallback)
		}
	}

	// Check minimum seeders
	if preset.MinSeeders > 0 && parsed.Seeders > 0 && parsed.Seeders < preset.MinSeeders {
		return false, fmt.Sprintf("insufficient seeders: want %d+, got %d", preset.MinSeeders, parsed.Seeders)
	}

	return true, ""
}

// scoreResultsWithPreset scores results based on preset criteria
func (s *Scheduler) scoreResultsWithPreset(results []indexer.SearchResult, presetID *int64) []indexer.ScoredSearchResult {
	var preset *database.QualityPreset
	if presetID != nil {
		p, err := s.db.GetQualityPreset(*presetID)
		if err == nil {
			preset = p
		}
	}

	var scoredResults []indexer.ScoredSearchResult
	for _, result := range results {
		parsed := parser.Parse(result.Title)
		parsed.Size = result.Size
		parsed.Seeders = result.Seeders
		parsed.Indexer = result.IndexerName

		// Convert HDR string to slice
		var hdrFormats []string
		if parsed.HDR != "" {
			hdrFormats = []string{parsed.HDR}
		}

		scored := indexer.ScoredSearchResult{
			SearchResult: result,
			Resolution:   parsed.Resolution,
			Source:       parsed.Source,
			Codec:        parsed.Codec,
			HDR:          hdrFormats,
			AudioCodec:   parsed.AudioFormat,
			ReleaseGroup: parsed.ReleaseGroup,
			Proper:       parsed.IsProper,
			Repack:       parsed.IsRepack,
		}

		// Check if release matches preset
		matches, reason := s.matchesPreset(parsed, preset)
		if !matches {
			scored.Rejected = true
			scored.RejectionReason = reason
		}

		// Check for blocked releases
		if parsed.ShouldBlock() {
			scored.Rejected = true
			scored.RejectionReason = parsed.BlockReason()
		}

		// Calculate base score based on quality tier
		qualityTier := quality.ComputeQualityTier(parsed)
		scored.BaseScore = quality.BaseQualityScores[qualityTier]
		scored.TotalScore = scored.BaseScore

		// Bonus for matching preset exactly
		if preset != nil && matches {
			scored.TotalScore += 100 // Boost for preset match
		}

		// Bonus for proper/repack
		if parsed.IsProper {
			scored.TotalScore += 5
		}
		if parsed.IsRepack {
			scored.TotalScore += 5
		}

		// Bonus for trusted groups
		mediaCategory := "movies"
		if parsed.Season > 0 || parsed.IsSeasonPack {
			mediaCategory = "tv"
		}
		if parsed.IsAnime {
			mediaCategory = "anime"
		}
		if parser.IsTrustedGroup(parsed.ReleaseGroup, mediaCategory) {
			scored.TotalScore += 10
		}

		scoredResults = append(scoredResults, scored)
	}

	// Sort by score descending
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].TotalScore > scoredResults[i].TotalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	return scoredResults
}

// shouldDelayGrab checks if a release should be delayed based on delay profiles
func (s *Scheduler) shouldDelayGrab(result *indexer.ScoredSearchResult, libraryID int64) (bool, time.Time) {
	profiles, err := s.db.GetDelayProfiles()
	if err != nil || len(profiles) == 0 {
		return false, time.Time{}
	}

	for _, profile := range profiles {
		if !profile.Enabled {
			continue
		}

		// Check if profile applies to this library
		if profile.LibraryID != nil && *profile.LibraryID != libraryID && libraryID > 0 {
			continue
		}

		// Check bypass conditions
		if profile.BypassIfResolution != nil && *profile.BypassIfResolution != "" {
			if strings.EqualFold(result.Resolution, *profile.BypassIfResolution) {
				continue // Bypass delay for this resolution
			}
		}

		if profile.BypassIfSource != nil && *profile.BypassIfSource != "" {
			if strings.EqualFold(result.Source, *profile.BypassIfSource) {
				continue // Bypass delay for this source
			}
		}

		if profile.BypassIfScoreAbove != nil && result.TotalScore > *profile.BypassIfScoreAbove {
			continue // Bypass delay for high scores
		}

		// Apply delay
		availableAt := time.Now().Add(time.Duration(profile.DelayMinutes) * time.Minute)
		log.Printf("Scheduler: delaying grab for %d minutes: %s", profile.DelayMinutes, result.Title)
		return true, availableAt
	}

	return false, time.Time{}
}

// shouldPauseDownloads checks if downloads should be paused due to low storage
// getIndexerIDsForMediaType returns indexer IDs suitable for the given media type
// based on library tag assignments and indexer capabilities
func (s *Scheduler) getIndexerIDsForMediaType(mediaType string) []int64 {
	// Find the library for this media type
	libraries, err := s.db.GetLibraries()
	if err != nil {
		return nil
	}

	var libraryID int64
	libType := "movies"
	if mediaType == "show" {
		libType = "tv"
	} else if mediaType == "anime" {
		libType = "anime"
	}

	for _, lib := range libraries {
		if lib.Type == libType {
			libraryID = lib.ID
			break
		}
	}

	if libraryID == 0 {
		return nil
	}

	// Get tags assigned to this library
	tagIDs, err := s.db.GetLibraryIndexerTags(libraryID)
	if err != nil {
		return nil
	}

	// Get indexers matching these tags (or by media type if no tags)
	var indexers []database.Indexer
	if len(tagIDs) > 0 {
		indexers, err = s.db.GetIndexersByTags(tagIDs, mediaType)
	} else {
		indexers, err = s.db.GetIndexersByMediaType(mediaType)
	}
	if err != nil {
		return nil
	}

	// Extract IDs
	ids := make([]int64, len(indexers))
	for i, idx := range indexers {
		ids[i] = idx.ID
	}
	return ids
}

func (s *Scheduler) shouldPauseDownloads() bool {
	settings, err := s.db.GetAllSettings()
	if err != nil {
		return false
	}

	pauseEnabled := settings["storage_pause_enabled"] == "true"
	if !pauseEnabled {
		return false
	}

	// Get threshold from settings
	thresholdGB := int64(100)
	if val, ok := settings["storage_threshold_gb"]; ok {
		var parsed int64
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			thresholdGB = parsed
		}
	}

	// Check storage on all libraries
	libraries, err := s.db.GetLibraries()
	if err != nil {
		return false
	}

	for _, lib := range libraries {
		usage, err := storage.GetDiskUsage(lib.Path)
		if err != nil {
			continue
		}

		freeGB := int64(usage.Free / (1024 * 1024 * 1024))
		if freeGB < thresholdGB {
			log.Printf("Scheduler: pausing downloads - low disk space on %s: %d GB free (threshold: %d GB)", lib.Path, freeGB, thresholdGB)
			return true
		}
	}

	return false
}

func (s *Scheduler) checkRSSFeeds() {
	// Check if RSS checking is enabled
	rssEnabled, _ := s.db.GetSetting("scheduler_rss_enabled")
	if rssEnabled != "true" {
		return
	}

	// Get monitored items for RSS matching
	items, err := s.db.GetMonitoredItems()
	if err != nil || len(items) == 0 {
		return
	}

	// Get enabled indexers
	indexers, err := s.db.GetEnabledIndexers()
	if err != nil || len(indexers) == 0 {
		return
	}

	log.Printf("Scheduler: checking RSS feeds from %d indexers", len(indexers))

	for _, idx := range indexers {
		// Fetch RSS feed
		results, err := s.indexers.FetchRSS(idx.ID)
		if err != nil {
			log.Printf("Scheduler: RSS fetch failed for %s: %v", idx.Name, err)
			continue
		}

		// Match results against wanted items
		for _, result := range results {
			s.matchRSSResult(result, items)
		}

		time.Sleep(2 * time.Second) // Delay between indexers
	}
}

func (s *Scheduler) matchRSSResult(result indexer.SearchResult, items []database.WantedItem) {
	// Simple title matching for now
	for _, item := range items {
		// TODO: Implement more sophisticated matching (IMDB/TMDB ID, release year, etc.)
		// For now, do basic title matching
		if !s.titleMatches(result.Title, item.Title, item.Year) {
			continue
		}

		// Score the result
		scored := s.scoreResults([]indexer.SearchResult{result}, item.QualityProfileID)
		if len(scored) == 0 || scored[0].Rejected {
			continue
		}

		// Check auto-grab
		autoGrab, _ := s.db.GetSetting("scheduler_auto_grab")
		if autoGrab != "true" {
			log.Printf("Scheduler: RSS match for %s: %s (auto-grab disabled)", item.Title, result.Title)
			continue
		}

		// Check minimum score threshold
		minScore := 0
		if minScoreStr, _ := s.db.GetSetting("scheduler_min_score"); minScoreStr != "" {
			minScore, _ = strconv.Atoi(minScoreStr)
		}

		if scored[0].TotalScore < minScore {
			continue
		}

		err := s.grabRelease(&scored[0], item.Type)
		if err != nil {
			log.Printf("Scheduler: RSS grab failed for %s: %v", item.Title, err)
			continue
		}

		log.Printf("Scheduler: RSS grabbed %s for %s (score: %d)", result.Title, item.Title, scored[0].TotalScore)
	}
}

func (s *Scheduler) titleMatches(releaseTitle, wantedTitle string, year int) bool {
	// Normalize titles for comparison
	releaseLower := normalizeTitle(releaseTitle)
	wantedLower := normalizeTitle(wantedTitle)

	// Check if wanted title is in release title
	if !containsTitle(releaseLower, wantedLower) {
		return false
	}

	// If we have a year, check it's in the release
	if year > 0 {
		yearStr := strconv.Itoa(year)
		if !containsTitle(releaseTitle, yearStr) {
			return false
		}
	}

	return true
}

func normalizeTitle(title string) string {
	// Simple normalization: lowercase and remove special chars
	result := ""
	for _, r := range title {
		if r >= 'a' && r <= 'z' {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r - 'A' + 'a')
		} else if r >= '0' && r <= '9' {
			result += string(r)
		} else if r == ' ' {
			result += " "
		}
	}
	return result
}

func containsTitle(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle ||
		 len(haystack) > len(needle) &&
		 (haystack[:len(needle)+1] == needle+" " ||
		  haystack[len(haystack)-len(needle)-1:] == " "+needle ||
		  containsSubstr(haystack, " "+needle+" ")))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// verifyReleaseMatch ensures the release title actually matches the wanted item
// This prevents grabbing wrong content that happens to match search terms
func (s *Scheduler) verifyReleaseMatch(releaseTitle string, item *database.WantedItem) (bool, string) {
	parsed := parser.Parse(releaseTitle)

	// Normalize both titles for comparison
	releaseTitleNorm := normalizeTitle(parsed.Title)
	wantedTitleNorm := normalizeTitle(item.Title)

	// Check if titles are similar enough
	if !titlesMatch(releaseTitleNorm, wantedTitleNorm) {
		return false, fmt.Sprintf("title mismatch: wanted '%s', got '%s'", item.Title, parsed.Title)
	}

	// Check year if we have both
	if item.Year > 0 && parsed.Year > 0 {
		// Allow 1 year tolerance for edge cases (release near year boundary)
		yearDiff := item.Year - parsed.Year
		if yearDiff < -1 || yearDiff > 1 {
			return false, fmt.Sprintf("year mismatch: wanted %d, got %d", item.Year, parsed.Year)
		}
	}

	return true, ""
}

// titlesMatch checks if two normalized titles are similar enough
func titlesMatch(releaseTitle, wantedTitle string) bool {
	// Exact match
	if releaseTitle == wantedTitle {
		return true
	}

	// Check if wanted title is at the start of release title
	if len(releaseTitle) > len(wantedTitle) &&
	   strings.HasPrefix(releaseTitle, wantedTitle) {
		// Make sure it's a word boundary (next char is space or end)
		nextChar := releaseTitle[len(wantedTitle)]
		if nextChar == ' ' || nextChar == '.' || nextChar == '-' {
			return true
		}
	}

	// Check if release title starts with wanted title words
	releaseWords := strings.Fields(releaseTitle)
	wantedWords := strings.Fields(wantedTitle)

	if len(releaseWords) < len(wantedWords) {
		return false
	}

	// All wanted words must match at the start
	matchCount := 0
	for i, word := range wantedWords {
		if i < len(releaseWords) && releaseWords[i] == word {
			matchCount++
		}
	}

	// Require at least 80% of words to match for longer titles
	minMatches := len(wantedWords)
	if len(wantedWords) > 3 {
		minMatches = int(float64(len(wantedWords)) * 0.8)
	}

	return matchCount >= minMatches
}

