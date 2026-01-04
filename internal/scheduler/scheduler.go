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
	"github.com/outpost/outpost/internal/storage"
)

type Scheduler struct {
	db        *database.Database
	indexers  *indexer.Manager
	downloads *downloadclient.Manager

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

func New(db *database.Database, indexers *indexer.Manager, downloads *downloadclient.Manager) *Scheduler {
	s := &Scheduler{
		db:             db,
		indexers:       indexers,
		downloads:      downloads,
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
	}

	finishedAt := time.Now()
	durationMs := finishedAt.Sub(startedAt).Milliseconds()

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
	// Check if downloads should be paused due to low storage
	if s.shouldPauseDownloads() {
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

	results, err := s.indexers.Search(params)
	if err != nil {
		log.Printf("Scheduler: search failed for %s: %v", item.Title, err)
		return
	}

	// Update last searched
	s.db.UpdateWantedLastSearched(item.ID)

	if len(results) == 0 {
		log.Printf("Scheduler: no results for %s", item.Title)
		return
	}

	// Score results
	scoredResults := s.scoreResults(results, item.QualityProfileID)

	// Check auto-grab setting
	autoGrab, _ := s.db.GetSetting("scheduler_auto_grab")
	if autoGrab != "true" {
		log.Printf("Scheduler: found %d results for %s (auto-grab disabled)", len(scoredResults), item.Title)
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

	// Find best non-rejected, non-blocklisted result
	// Results are sorted by score descending, so pick first acceptable one
	var bestResult *indexer.ScoredSearchResult
	for i := range scoredResults {
		if scoredResults[i].Rejected || scoredResults[i].TotalScore <= 0 {
			continue
		}

		// Check blocklist
		blocked, _ := s.db.IsReleaseBlocklisted(scoredResults[i].Title)
		if blocked {
			log.Printf("Scheduler: release blocklisted, trying next: %s", scoredResults[i].Title)
			continue
		}

		// Check if indexer is excluded for this library
		if libraryID > 0 {
			excluded, _ := s.db.IsIndexerExcludedForLibrary(scoredResults[i].IndexerID, libraryID)
			if excluded {
				log.Printf("Scheduler: indexer excluded for library, trying next: %s", scoredResults[i].Title)
				continue
			}
		}

		// Check release filters
		if !s.passesReleaseFilters(scoredResults[i].Title, item.QualityProfileID) {
			continue
		}

		bestResult = &scoredResults[i]
		break
	}

	if bestResult == nil {
		log.Printf("Scheduler: no acceptable releases for %s", item.Title)
		return
	}

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
	err = s.grabRelease(bestResult)
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

func (s *Scheduler) grabRelease(result *indexer.ScoredSearchResult) error {
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

	if isTorrent {
		return s.downloads.AddTorrent(targetClient.ID, downloadURL, result.Category)
	}
	return s.downloads.AddNZB(targetClient.ID, downloadURL, result.Category)
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

		err := s.grabRelease(&scored[0])
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
