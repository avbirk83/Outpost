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
	"github.com/outpost/outpost/internal/trakt"
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

	// Active search tracking for UI
	activeSearch string
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
		{
			Name:            "Upgrade Search",
			Description:     "Search for better quality versions of owned media",
			TaskType:        "upgrade_search",
			Enabled:         false, // Disabled by default
			IntervalMinutes: 720,   // 12 hours
		},
		{
			Name:            "Trakt Sync",
			Description:     "Sync watch history and ratings with Trakt.tv",
			TaskType:        "trakt_sync",
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

// GetActiveSearch returns the title of the item currently being searched
func (s *Scheduler) GetActiveSearch() string {
	s.taskMu.RLock()
	defer s.taskMu.RUnlock()
	return s.activeSearch
}

// GetRunningTaskNames returns a list of currently running task names
func (s *Scheduler) GetRunningTaskNames() []string {
	s.taskMu.RLock()
	defer s.taskMu.RUnlock()

	var names []string
	for name, running := range s.taskRunning {
		if running {
			names = append(names, name)
		}
	}
	return names
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
	case "upgrade_search":
		itemsProcessed, itemsFound = s.runUpgradeSearchTask()
	case "trakt_sync":
		itemsProcessed = s.runTraktSyncTask()
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

// runUpgradeSearchTask searches for better quality versions of owned media
func (s *Scheduler) runUpgradeSearchTask() (processed, found int) {
	// Get upgrade search settings
	settings, err := s.db.GetAllSettings()
	if err != nil {
		log.Printf("Scheduler: failed to get settings: %v", err)
		return 0, 0
	}

	// Check if upgrade search is enabled
	if val, ok := settings["upgrade_search_enabled"]; ok && val != "true" {
		log.Printf("Scheduler: upgrade search is disabled")
		return 0, 0
	}

	// Get limit from settings (default 10)
	limit := 10
	if val, ok := settings["upgrade_search_limit"]; ok {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	log.Printf("Scheduler: searching for %d upgradeable items", limit)

	// Get upgradeable movies
	movies, err := s.db.GetUpgradeableMovies(limit / 2)
	if err != nil {
		log.Printf("Scheduler: failed to get upgradeable movies: %v", err)
	} else {
		for _, item := range movies {
			movie, err := s.db.GetMovie(item.ID)
			if err != nil {
				continue
			}

			var imdbID string
			if movie.ImdbID != nil {
				imdbID = *movie.ImdbID
			}

			// Get quality preset
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
					processed++
					s.db.UpdateUpgradeSearched(item.ID, "movie", false)
					// Search for the item
					s.SearchWantedItem(*movie.TmdbID, "movie")
				}
			}
		}
	}

	// Get upgradeable episodes
	episodes, err := s.db.GetUpgradeableEpisodes(limit / 2)
	if err != nil {
		log.Printf("Scheduler: failed to get upgradeable episodes: %v", err)
	} else {
		for _, item := range episodes {
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

			// Get quality preset
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
					processed++
					s.db.UpdateUpgradeSearched(item.ID, "episode", false)
					// Search for the item
					s.SearchWantedItem(*show.TmdbID, "show")
				}
			}
		}
	}

	log.Printf("Scheduler: upgrade search complete - processed %d items", processed)
	return processed, found
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
		taskName := fmt.Sprintf("Searching: %s", item.Title)
		s.taskMu.Lock()
		s.taskRunning[taskName] = true
		s.activeSearch = item.Title
		s.taskMu.Unlock()
		defer func() {
			s.taskMu.Lock()
			s.taskRunning[taskName] = false
			s.activeSearch = ""
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
	mediaTypeForCategories := "movie"
	if item.Type == "show" {
		searchType = "tvsearch"
		mediaTypeForCategories = "tv"
	} else if item.Type == "anime" {
		searchType = "tvsearch" // Anime uses TV search type
		mediaTypeForCategories = "anime"
	}

	params := indexer.SearchParams{
		Query:      item.Title,
		Type:       searchType,
		Limit:      50,
		Categories: database.GetCategoriesForMediaType(mediaTypeForCategories),
	}

	// Add TMDB ID if available
	if item.TmdbID > 0 {
		params.TmdbID = strconv.FormatInt(item.TmdbID, 10)
	}

	// Look up IMDB ID from the movie/show record for more accurate searches
	imdbID := s.lookupImdbID(item.Type, item.TmdbID)
	if imdbID != "" {
		params.ImdbID = imdbID
		log.Printf("Scheduler: found IMDB ID %s for %s (TMDB: %d)", imdbID, item.Title, item.TmdbID)
	} else {
		log.Printf("Scheduler: no IMDB ID found for %s (TMDB: %d) - using title search", item.Title, item.TmdbID)
	}

	// For TV shows/anime, also look up TVDB ID
	if item.Type == "show" || item.Type == "anime" {
		tvdbID := s.lookupTvdbID(item.TmdbID)
		if tvdbID != "" {
			params.TvdbID = tvdbID
			log.Printf("Scheduler: found TVDB ID %s for %s", tvdbID, item.Title)
		}
	}

	// Log search parameters for debugging
	log.Printf("Scheduler: SEARCH PARAMS for '%s' (%s):", item.Title, item.Type)
	log.Printf("  - Query: %s", params.Query)
	log.Printf("  - Type: %s", params.Type)
	log.Printf("  - IMDB ID: %s (enables exact matching)", params.ImdbID)
	log.Printf("  - TVDB ID: %s", params.TvdbID)
	log.Printf("  - TMDB ID: %s", params.TmdbID)
	log.Printf("  - Categories: %v", params.Categories)

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

	// Filter out adult content (category 6000-6999)
	results = filterAdultContent(results)
	log.Printf("Scheduler: %d results after adult content filtering", len(results))

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

	// Sort presets by quality priority (highest quality first)
	// Resolution priority: 2160p > 1080p > 720p > 480p > any
	sortedPresets := sortPresetsByQuality(allPresets)

	// Then add other enabled presets as fallbacks (sorted by quality priority)
	for i := range sortedPresets {
		if !sortedPresets[i].Enabled {
			continue
		}
		// Skip if already added as primary
		if item.QualityPresetID != nil && sortedPresets[i].ID == *item.QualityPresetID {
			continue
		}
		id := sortedPresets[i].ID
		presetsToTry = append(presetsToTry, &id)
	}

	// If no presets to try (item has no preset and no presets exist), use nil to score purely by quality
	if len(presetsToTry) == 0 {
		presetsToTry = append(presetsToTry, nil)
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

			// Format validation now happens in scoreResultsWithPreset, so format-rejected
			// releases will already have scored.Rejected = true and won't reach here

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

	// Try to grab with failover - if first choice fails, try alternatives
	// Collect all acceptable results for failover
	var acceptableResults []*indexer.ScoredSearchResult
	for _, presetID := range presetsToTry {
		scoredResults := s.scoreResultsWithPreset(results, presetID)
		for i := range scoredResults {
			if scoredResults[i].Rejected || scoredResults[i].TotalScore <= 0 {
				continue
			}
			matches, _ := s.verifyReleaseMatch(scoredResults[i].Title, item)
			if !matches {
				continue
			}
			blocked, _ := s.db.IsReleaseBlocklisted(scoredResults[i].Title)
			if blocked {
				continue
			}
			if libraryID > 0 {
				excluded, _ := s.db.IsIndexerExcludedForLibrary(scoredResults[i].IndexerID, libraryID)
				if excluded {
					continue
				}
			}
			if item.QualityProfileID > 0 && !s.passesReleaseFilters(scoredResults[i].Title, item.QualityProfileID) {
				continue
			}
			acceptableResults = append(acceptableResults, &scoredResults[i])
		}
	}

	// Try each acceptable result until one succeeds
	var grabbed bool
	for _, result := range acceptableResults {
		err = s.grabRelease(result, item.Type, item.TmdbID)
		if err == nil {
			log.Printf("Scheduler: grabbed %s for %s (score: %d)", result.Title, item.Title, result.TotalScore)
			grabbed = true
			break
		}
		log.Printf("Scheduler: grab failed for %s, trying next: %v", result.Title, err)
	}

	if !grabbed {
		log.Printf("Scheduler: all grab attempts failed for %s", item.Title)
	}
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

func (s *Scheduler) grabRelease(result *indexer.ScoredSearchResult, mediaType string, mediaID int64) error {
	var downloadURL string
	if result.MagnetLink != "" {
		downloadURL = result.MagnetLink
	} else {
		downloadURL = result.Link
	}

	// Prowlarr uses "prowlarr" as type, treat as torrent unless explicitly newznab
	isTorrent := result.IndexerType == "torznab" || result.IndexerType == "prowlarr" || result.MagnetLink != ""
	if result.IndexerType == "newznab" && result.MagnetLink == "" {
		isTorrent = false
	}

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
		return fmt.Errorf("no suitable download client found for %s", result.Title)
	}

	// Generate category based on media type
	category := getCategoryForMediaType(mediaType)

	// Attempt to add to download client
	var grabErr error
	if isTorrent {
		grabErr = s.downloads.AddTorrent(targetClient.ID, downloadURL, category)
	} else {
		grabErr = s.downloads.AddNZB(targetClient.ID, downloadURL, category)
	}

	// Record grab in history
	indexerName := result.IndexerName
	hdrStr := strings.Join(result.HDR, ",")
	grabHistory := &database.GrabHistory{
		MediaID:           mediaID,
		MediaType:         mediaType,
		ReleaseTitle:      result.Title,
		IndexerID:         &result.IndexerID,
		IndexerName:       &indexerName,
		QualityResolution: strPtr(result.Resolution),
		QualitySource:     strPtr(result.Source),
		QualityCodec:      strPtr(result.Codec),
		QualityAudio:      strPtr(result.AudioCodec),
		QualityHDR:        strPtr(hdrStr),
		ReleaseGroup:      strPtr(result.ReleaseGroup),
		Size:              result.Size,
		DownloadClientID:  &targetClient.ID,
		Status:            "grabbed",
	}

	if grabErr != nil {
		grabHistory.Status = "failed"
		errMsg := grabErr.Error()
		grabHistory.ErrorMessage = &errMsg
	}

	if dbErr := s.db.AddGrabHistory(grabHistory); dbErr != nil {
		log.Printf("Scheduler: failed to record grab history: %v", dbErr)
	}

	return grabErr
}

// strPtr returns a pointer to a string, or nil if empty
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

	// Check resolution - use minimum resolution instead of exact match
	// This allows 4K presets to grab 1080p as fallback when 4K isn't available
	// Resolution is used for scoring (higher = better), not strict filtering
	minResolution := "720p" // Default minimum acceptable resolution

	// Resolution priority order (higher = better quality)
	resOrder := map[string]int{
		"2160p": 4, "4k": 4, "uhd": 4,
		"1080p": 3, "1080i": 3,
		"720p": 2,
		"480p": 1, "sd": 1,
	}

	releaseRes := strings.ToLower(parsed.Resolution)
	releaseOrder := resOrder[releaseRes]
	if releaseOrder == 0 {
		// Unknown resolution - accept but treat as low quality
		releaseOrder = 1
	}

	minOrder := resOrder[minResolution]
	if releaseOrder < minOrder {
		return false, fmt.Sprintf("resolution below minimum: %s (minimum: %s)", parsed.Resolution, minResolution)
	}

	// Source and codec are now preference-based, not rejection-based
	// Scoring handles the preference (remux > bluray > web > hdtv)
	// We only reject truly bad sources like CAM, TS, WORKPRINT
	releaseSource := strings.ToLower(parsed.Source)
	badSources := map[string]bool{
		"cam": true, "ts": true, "tc": true, "screener": true,
		"dvdscr": true, "r5": true, "workprint": true,
	}
	if badSources[releaseSource] {
		return false, fmt.Sprintf("unacceptable source: %s", parsed.Source)
	}

	// Codec check is also preference-based now
	// All modern codecs (hevc, av1, avc) are acceptable, scoring handles preference
	releaseCodec := strings.ToLower(parsed.Codec)
	badCodecs := map[string]bool{
		"xvid": true, "divx": true, "mpeg2": true,
	}
	if badCodecs[releaseCodec] {
		return false, fmt.Sprintf("unacceptable codec: %s", parsed.Codec)
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

		// Check format settings (container/disc/archive filtering)
		if !scored.Rejected {
			formatSettings, _ := s.db.GetFormatSettings()
			if rejection := quality.ValidateFormat(parsed, formatSettings); rejection != nil {
				scored.Rejected = true
				scored.RejectionReason = rejection.Reason
				// Auto-blocklist if configured
				if formatSettings != nil && formatSettings.AutoBlocklist && rejection.Permanent {
					s.db.AddToBlocklist(&database.BlocklistEntry{
						ReleaseTitle: result.Title,
						Reason:       rejection.Reason,
					})
				}
			}
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
// based on library tag assignments, indexer capabilities, and category filtering
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
		// Use category-based filtering:
		// - For anime: Only use indexers with anime categories
		// - For movie/tv: Exclude anime-only indexers (like Nyaa)
		if mediaType == "anime" {
			indexers, err = s.db.GetIndexersWithCategories("anime")
		} else {
			indexers, err = s.db.GetIndexersExcludingAnimeOnly(mediaType)
		}
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
	for _, item := range items {
		// Try multiple matching strategies in order of reliability

		// 1. IMDB ID match (most reliable if available)
		if result.ImdbID != "" && item.ImdbID != nil {
			// Normalize IMDB IDs (some may have "tt" prefix, some may not)
			resultImdb := strings.TrimPrefix(result.ImdbID, "tt")
			itemImdb := strings.TrimPrefix(*item.ImdbID, "tt")
			if resultImdb == itemImdb {
				// IMDB match - proceed with this item
				log.Printf("Scheduler: RSS IMDB match for %s (tt%s)", item.Title, itemImdb)
				s.processRSSMatch(result, item)
				continue
			}
		}

		// 2. TVDB ID match for TV shows
		if result.TvdbID != "" && (item.Type == "show" || item.Type == "anime") {
			// Try to get TVDB ID for this show
			itemTvdbID := s.lookupTvdbID(item.TmdbID)
			if itemTvdbID != "" && result.TvdbID == itemTvdbID {
				log.Printf("Scheduler: RSS TVDB match for %s (tvdb:%s)", item.Title, itemTvdbID)
				s.processRSSMatch(result, item)
				continue
			}
		}

		// 3. Fallback to title + year matching using verifyReleaseMatch
		matches, reason := s.verifyReleaseMatch(result.Title, &item)
		if !matches {
			if reason != "" {
				log.Printf("DEBUG: RSS title mismatch for %s: %s - %s", item.Title, result.Title, reason)
			}
			continue
		}

		s.processRSSMatch(result, item)
	}
}

// processRSSMatch handles scoring and grabbing a matched RSS result
func (s *Scheduler) processRSSMatch(result indexer.SearchResult, item database.WantedItem) {
	// Score the result
	scored := s.scoreResults([]indexer.SearchResult{result}, item.QualityProfileID)
	if len(scored) == 0 || scored[0].Rejected {
		return
	}

	// Check auto-grab
	autoGrab, _ := s.db.GetSetting("scheduler_auto_grab")
	if autoGrab != "true" {
		log.Printf("Scheduler: RSS match for %s: %s (auto-grab disabled)", item.Title, result.Title)
		return
	}

	// Check minimum score threshold
	minScore := 0
	if minScoreStr, _ := s.db.GetSetting("scheduler_min_score"); minScoreStr != "" {
		minScore, _ = strconv.Atoi(minScoreStr)
	}

	if scored[0].TotalScore < minScore {
		return
	}

	err := s.grabRelease(&scored[0], item.Type, item.TmdbID)
	if err != nil {
		log.Printf("Scheduler: RSS grab failed for %s: %v", item.Title, err)
		return
	}

	log.Printf("Scheduler: RSS grabbed %s for %s (score: %d)", result.Title, item.Title, scored[0].TotalScore)
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

// lookupImdbID retrieves the IMDB ID for a movie or show from the database
func (s *Scheduler) lookupImdbID(mediaType string, tmdbID int64) string {
	if mediaType == "movie" {
		movie, err := s.db.GetMovieByTmdb(tmdbID)
		if err == nil && movie != nil && movie.ImdbID != nil {
			return *movie.ImdbID
		}
	} else if mediaType == "show" || mediaType == "anime" {
		show, err := s.db.GetShowByTmdb(tmdbID)
		if err == nil && show != nil && show.ImdbID != nil {
			return *show.ImdbID
		}
	}
	return ""
}

// lookupTvdbID retrieves the TVDB ID for a show from the database
func (s *Scheduler) lookupTvdbID(tmdbID int64) string {
	show, err := s.db.GetShowByTmdb(tmdbID)
	if err == nil && show != nil && show.TvdbID != nil {
		return strconv.FormatInt(*show.TvdbID, 10)
	}
	return ""
}

// filterAdultContent removes results with adult category IDs (6000-6999)
func filterAdultContent(results []indexer.SearchResult) []indexer.SearchResult {
	filtered := make([]indexer.SearchResult, 0, len(results))
	for _, r := range results {
		if isAdultCategory(r.CategoryID) {
			log.Printf("Scheduler: filtering adult content: %s (category: %s)", r.Title, r.CategoryID)
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// isAdultCategory checks if a category ID is in the adult range (6000-6999)
func isAdultCategory(categoryID string) bool {
	if categoryID == "" {
		return false
	}

	// Category can be comma-separated list
	for _, catStr := range strings.Split(categoryID, ",") {
		catStr = strings.TrimSpace(catStr)
		cat, err := strconv.Atoi(catStr)
		if err != nil {
			continue
		}
		// Adult categories are 6000-6999
		if cat >= 6000 && cat < 7000 {
			return true
		}
	}
	return false
}

// Common words to remove from titles for comparison
var commonWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true,
	"of": true, "in": true, "to": true, "for": true, "is": true,
}

func normalizeTitle(title string) string {
	// Lowercase and remove special chars
	var result strings.Builder
	for _, r := range title {
		if r >= 'a' && r <= 'z' {
			result.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else if r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else if r == ' ' || r == '.' || r == '-' || r == '_' {
			result.WriteRune(' ')
		}
	}

	// Remove common words
	words := strings.Fields(result.String())
	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if !commonWords[word] {
			filtered = append(filtered, word)
		}
	}
	return strings.Join(filtered, " ")
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	return matrix[len(s1)][len(s2)]
}

// similarityScore returns a 0-100 score based on Levenshtein distance
func similarityScore(s1, s2 string) int {
	if s1 == s2 {
		return 100
	}
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 100
	}
	distance := levenshteinDistance(s1, s2)
	similarity := float64(maxLen-distance) / float64(maxLen) * 100
	if similarity < 0 {
		return 0
	}
	return int(similarity)
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

// detectMediaType detects the media type from parsed release data
// Returns "movie", "tv", or "anime"
func detectMediaType(parsed *parser.ParsedRelease) string {
	// Anime detection: has anime flags or absolute episode numbering
	if parsed.IsAnime || parsed.IsAbsoluteEpisode || parsed.IsOVA || parsed.IsONA || parsed.IsOAD {
		return "anime"
	}

	// TV detection: has season/episode pattern
	if parsed.Season > 0 || parsed.Episode > 0 || parsed.IsSeasonPack || parsed.IsDailyShow {
		return "tv"
	}

	// Movie: no season/episode, typically has year
	return "movie"
}

// ReleaseMatchScore holds the individual scores for a release match
type ReleaseMatchScore struct {
	TitleScore    int    // 0-100 based on Levenshtein similarity
	YearScore     int    // 0, 80, or 100
	TypeScore     int    // 0 or 100
	CombinedScore int    // Weighted average
	Reason        string // Rejection reason if any
}

// verifyReleaseMatch ensures the release title actually matches the wanted item
// Returns match status, rejection reason, and detailed scores for debugging
func (s *Scheduler) verifyReleaseMatch(releaseTitle string, item *database.WantedItem) (bool, string) {
	score := s.calculateMatchScore(releaseTitle, item)

	// Log rejected results at debug level
	if score.CombinedScore < 60 {
		log.Printf("DEBUG: Rejected '%s' for '%s' - title:%d%%, year:%d%%, type:%d%%, combined:%d%% - %s",
			releaseTitle, item.Title, score.TitleScore, score.YearScore, score.TypeScore, score.CombinedScore, score.Reason)
		return false, score.Reason
	}

	return true, ""
}

// Adult content keywords that should cause rejection
var adultKeywords = []string{
	"xxx", "adult", "porn", "18+", "explicit", "erotic",
	"brazzers", "naughty", "bangbros", "realitykings",
}

// containsAdultKeyword checks if a release title contains adult content keywords
func containsAdultKeyword(title string) bool {
	titleLower := strings.ToLower(title)
	for _, keyword := range adultKeywords {
		// Check for keyword as separate word or surrounded by dots/dashes
		patterns := []string{
			" " + keyword + " ",
			"." + keyword + ".",
			"-" + keyword + "-",
			" " + keyword + ".",
			"." + keyword + " ",
		}
		for _, pattern := range patterns {
			if strings.Contains(titleLower, pattern) {
				return true
			}
		}
		// Also check if title starts or ends with keyword
		if strings.HasPrefix(titleLower, keyword+" ") || strings.HasPrefix(titleLower, keyword+".") ||
			strings.HasSuffix(titleLower, " "+keyword) || strings.HasSuffix(titleLower, "."+keyword) {
			return true
		}
	}
	return false
}

// calculateMatchScore computes detailed match scores between release and wanted item
func (s *Scheduler) calculateMatchScore(releaseTitle string, item *database.WantedItem) ReleaseMatchScore {
	parsed := parser.Parse(releaseTitle)
	score := ReleaseMatchScore{
		YearScore: 100, // Default if no year comparison needed
		TypeScore: 100, // Default if type matches
	}

	// 0. Check for adult content keywords (immediate rejection)
	if containsAdultKeyword(releaseTitle) {
		score.TitleScore = 0
		score.YearScore = 0
		score.TypeScore = 0
		score.CombinedScore = 0
		score.Reason = "contains adult content keywords"
		log.Printf("DEBUG: Adult content detected in '%s'", releaseTitle)
		return score
	}

	// 1. Title similarity score (0-100)
	releaseTitleNorm := normalizeTitle(parsed.Title)
	wantedTitleNorm := normalizeTitle(item.Title)

	// First try word-based matching for better accuracy
	score.TitleScore = calculateTitleScore(releaseTitleNorm, wantedTitleNorm)

	if score.TitleScore < 80 {
		score.Reason = fmt.Sprintf("title similarity too low: %d%% (wanted '%s', got '%s')",
			score.TitleScore, item.Title, parsed.Title)
	}

	// 2. Year validation (0, 80, or 100)
	// STRICT: Year mismatch >1 year should be an immediate disqualifier
	if item.Year > 0 && parsed.Year > 0 {
		yearDiff := item.Year - parsed.Year
		if yearDiff < 0 {
			yearDiff = -yearDiff
		}
		switch yearDiff {
		case 0:
			score.YearScore = 100
		case 1:
			score.YearScore = 80 // Allow ±1 year tolerance
		default:
			// Year mismatch >1 year is a hard reject
			score.YearScore = 0
			score.Reason = fmt.Sprintf("year mismatch: wanted %d, got %d (diff: %d years)", item.Year, parsed.Year, yearDiff)
			log.Printf("DEBUG: Year mismatch for '%s' - wanted %d, got %d", releaseTitle, item.Year, parsed.Year)
		}
	}
	// If release doesn't have year, don't penalize (keep default 100)

	// 3. Media type validation (0 or 100)
	detectedType := detectMediaType(parsed)
	expectedType := item.Type
	if expectedType == "show" {
		expectedType = "tv" // Normalize
	}

	// Type matching logic:
	// - Anime can match anime
	// - TV can match TV or anime (anime is often categorized with TV)
	// - Movie must match movie
	typeMatch := false
	switch expectedType {
	case "anime":
		typeMatch = detectedType == "anime" || detectedType == "tv"
	case "tv":
		typeMatch = detectedType == "tv" || detectedType == "anime"
	case "movie":
		typeMatch = detectedType == "movie"
	default:
		typeMatch = true // Unknown type, don't reject
	}

	if !typeMatch {
		score.TypeScore = 0
		if score.Reason == "" {
			score.Reason = fmt.Sprintf("type mismatch: expected %s, detected %s", expectedType, detectedType)
		}
	}

	// 4. Calculate combined score (weighted average)
	// Title: 50%, Year: 25%, Type: 25%
	score.CombinedScore = (score.TitleScore*50 + score.YearScore*25 + score.TypeScore*25) / 100

	if score.Reason == "" && score.CombinedScore < 60 {
		score.Reason = fmt.Sprintf("combined score too low: %d%%", score.CombinedScore)
	}

	return score
}

// calculateTitleScore computes title similarity using multiple methods
func calculateTitleScore(releaseTitle, wantedTitle string) int {
	// Exact match
	if releaseTitle == wantedTitle {
		return 100
	}

	// Word-based scoring for better accuracy with title variations
	releaseWords := strings.Fields(releaseTitle)
	wantedWords := strings.Fields(wantedTitle)

	if len(wantedWords) == 0 {
		return 0
	}

	// Count matching words (order-aware for prefix matching)
	matchCount := 0
	for i, wanted := range wantedWords {
		if i < len(releaseWords) {
			// Exact word match at same position
			if releaseWords[i] == wanted {
				matchCount++
			} else if similarityScore(releaseWords[i], wanted) >= 80 {
				// Fuzzy word match (typos, slight variations)
				matchCount++
			}
		}
	}

	wordScore := (matchCount * 100) / len(wantedWords)

	// Also calculate Levenshtein-based score for the full title
	levenScore := similarityScore(releaseTitle, wantedTitle)

	// Return the higher of the two scores
	if wordScore > levenScore {
		return wordScore
	}
	return levenScore
}

// sortPresetsByQuality sorts presets by quality priority (highest quality first)
func sortPresetsByQuality(presets []database.QualityPreset) []database.QualityPreset {
	sorted := make([]database.QualityPreset, len(presets))
	copy(sorted, presets)

	// Quality priority map (higher = better)
	resPriority := map[string]int{
		"2160p": 100, "4k": 100, "uhd": 100,
		"1080p": 80, "1080i": 75,
		"720p": 60,
		"480p": 40, "sd": 40,
		"any": 50, "": 50, // "any" or empty goes in the middle
	}

	sourcePriority := map[string]int{
		"remux": 100,
		"bluray": 90, "bdrip": 85,
		"webdl": 70, "web": 70,
		"webrip": 60,
		"hdtv": 50,
		"dvd": 30, "dvdrip": 30,
		"any": 50, "": 50,
	}

	// Bubble sort by quality priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			// Calculate priority score for each preset
			scoreI := resPriority[strings.ToLower(sorted[i].Resolution)] * 10
			scoreI += sourcePriority[strings.ToLower(sorted[i].Source)]

			scoreJ := resPriority[strings.ToLower(sorted[j].Resolution)] * 10
			scoreJ += sourcePriority[strings.ToLower(sorted[j].Source)]

			// Higher score = higher quality, should come first
			if scoreJ > scoreI {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// runTraktSyncTask syncs watch history with Trakt.tv for all enabled users
func (s *Scheduler) runTraktSyncTask() int {
	configs, err := s.db.GetAllTraktConfigs()
	if err != nil {
		log.Printf("Failed to get Trakt configs: %v", err)
		return 0
	}

	if len(configs) == 0 {
		return 0
	}

	clientID, _ := s.db.GetSetting("trakt_client_id")
	clientSecret, _ := s.db.GetSetting("trakt_client_secret")

	if clientID == "" || clientSecret == "" {
		log.Printf("Trakt client ID or secret not configured")
		return 0
	}

	processed := 0
	for _, config := range configs {
		if !config.SyncEnabled || config.AccessToken == "" {
			continue
		}

		// Get user's default profile
		profile, err := s.db.GetDefaultProfile(config.UserID)
		if err != nil || profile == nil {
			log.Printf("No default profile for user %d, skipping Trakt sync", config.UserID)
			continue
		}

		client := trakt.NewClient(clientID, clientSecret)
		client.SetTokens(config.AccessToken, config.RefreshToken, *config.ExpiresAt)

		// Refresh token if needed
		if client.NeedsRefresh() {
			tokenResp, err := client.RefreshAccessToken()
			if err != nil {
				log.Printf("Trakt token refresh failed for user %d: %v", config.UserID, err)
				continue
			}
			expiresAt := time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			config.AccessToken = tokenResp.AccessToken
			config.RefreshToken = tokenResp.RefreshToken
			config.ExpiresAt = &expiresAt
			s.db.SaveTraktConfig(&config)
		}

		// Sync watched status
		if config.SyncWatched {
			processed += s.syncTraktWatched(profile.ID, client)
		}

		// Update last synced time
		s.db.UpdateTraktSyncTime(config.UserID)
	}

	return processed
}

// syncTraktWatched syncs watched status between Trakt and local database
func (s *Scheduler) syncTraktWatched(profileID int64, client *trakt.Client) int {
	synced := 0

	// Pull watched movies from Trakt
	watchedMovies, err := client.GetWatchedMovies()
	if err == nil {
		for _, wm := range watchedMovies {
			movie, err := s.db.GetMovieByTmdb(int64(wm.Movie.IDs.TMDB))
			if err != nil || movie == nil {
				continue
			}
			// Check if already watched
			progress, _ := s.db.GetProgress(profileID, "movie", movie.ID)
			if progress == nil || (progress.Duration > 0 && progress.Position/progress.Duration < 0.9) {
				// Mark as watched
				s.db.SaveProgress(&database.Progress{
					ProfileID: profileID,
					MediaType: "movie",
					MediaID:   movie.ID,
					Position:  100,
					Duration:  100,
				})
				synced++
			}
		}
	}

	// Pull watched shows from Trakt
	watchedShows, err := client.GetWatchedShows()
	if err == nil {
		for _, ws := range watchedShows {
			show, err := s.db.GetShowByTmdb(int64(ws.Show.IDs.TMDB))
			if err != nil || show == nil {
				continue
			}
			for _, season := range ws.Seasons {
				for _, ep := range season.Episodes {
					episode, err := s.db.GetEpisodeByShowSeasonEpisode(show.ID, season.Number, ep.Number)
					if err != nil || episode == nil {
						continue
					}
					progress, _ := s.db.GetProgress(profileID, "episode", episode.ID)
					if progress == nil || (progress.Duration > 0 && progress.Position/progress.Duration < 0.9) {
						s.db.SaveProgress(&database.Progress{
							ProfileID: profileID,
							MediaType: "episode",
							MediaID:   episode.ID,
							Position:  100,
							Duration:  100,
						})
						synced++
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
			if err == nil {
				synced += resp.Added.Movies + resp.Added.Episodes
				s.db.MarkWatchHistorySynced(syncedIDs)
			}
		}
	}

	return synced
}

