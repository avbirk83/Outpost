package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
	"github.com/outpost/outpost/internal/storage"
)

// Status represents the health status of a check
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusWarning   Status = "warning"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a single health check result
type Check struct {
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	Message   string    `json:"message"`
	Latency   *int64    `json:"latency,omitempty"` // milliseconds
	LastCheck time.Time `json:"lastCheck"`
	Error     *string   `json:"error,omitempty"`
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Overall       Status    `json:"overall"`
	Checks        []Check   `json:"checks"`
	LastFullCheck time.Time `json:"lastFullCheck"`
}

// Checker provides health check functionality
type Checker struct {
	db        *database.Database
	downloads *downloadclient.Manager
	indexers  *indexer.Manager
	tmdbKey   string
	mu        sync.RWMutex
	cache     *HealthStatus
	cacheTime time.Time
}

// NewChecker creates a new health checker
func NewChecker(db *database.Database, downloads *downloadclient.Manager, indexers *indexer.Manager) *Checker {
	return &Checker{
		db:        db,
		downloads: downloads,
		indexers:  indexers,
	}
}

// SetTMDBKey sets the TMDB API key for health checks
func (c *Checker) SetTMDBKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tmdbKey = key
}

// GetFullStatus returns the complete health status
func (c *Checker) GetFullStatus() *HealthStatus {
	checks := []Check{}
	now := time.Now()

	// Run all checks concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	addCheck := func(check Check) {
		mu.Lock()
		checks = append(checks, check)
		mu.Unlock()
	}

	// Database check
	wg.Add(1)
	go func() {
		defer wg.Done()
		addCheck(c.checkDatabase())
	}()

	// Download clients check
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientChecks := c.checkDownloadClients()
		for _, check := range clientChecks {
			addCheck(check)
		}
	}()

	// Prowlarr check
	wg.Add(1)
	go func() {
		defer wg.Done()
		if check := c.checkProwlarr(); check != nil {
			addCheck(*check)
		}
	}()

	// Indexers check
	wg.Add(1)
	go func() {
		defer wg.Done()
		indexerChecks := c.checkIndexers()
		for _, check := range indexerChecks {
			addCheck(check)
		}
	}()

	// Disk space check
	wg.Add(1)
	go func() {
		defer wg.Done()
		diskChecks := c.checkDiskSpace()
		for _, check := range diskChecks {
			addCheck(check)
		}
	}()

	// TMDB check
	wg.Add(1)
	go func() {
		defer wg.Done()
		addCheck(c.checkTMDB())
	}()

	wg.Wait()

	// Calculate overall status
	overall := StatusHealthy
	for _, check := range checks {
		if check.Status == StatusUnhealthy {
			overall = StatusUnhealthy
			break
		}
		if check.Status == StatusWarning && overall != StatusUnhealthy {
			overall = StatusWarning
		}
	}

	status := &HealthStatus{
		Overall:       overall,
		Checks:        checks,
		LastFullCheck: now,
	}

	// Cache the result
	c.mu.Lock()
	c.cache = status
	c.cacheTime = now
	c.mu.Unlock()

	return status
}

// RunSingleCheck runs a single health check by name
func (c *Checker) RunSingleCheck(name string) *Check {
	switch name {
	case "database":
		check := c.checkDatabase()
		return &check
	case "tmdb":
		check := c.checkTMDB()
		return &check
	case "prowlarr":
		return c.checkProwlarr()
	default:
		// Check if it's a download client
		if len(name) > 17 && name[:17] == "download_client_" {
			// Parse client name
			clientName := name[17:]
			clients, _ := c.db.GetEnabledDownloadClients()
			for _, client := range clients {
				if client.Name == clientName {
					check := c.checkSingleDownloadClient(&client)
					return &check
				}
			}
		}
		// Check if it's an indexer
		if len(name) > 8 && name[:8] == "indexer_" {
			indexerName := name[8:]
			indexers, _ := c.db.GetEnabledIndexers()
			for _, idx := range indexers {
				if idx.Name == indexerName {
					check := c.checkSingleIndexer(&idx)
					return &check
				}
			}
		}
	}
	return nil
}

// checkDatabase checks database connectivity
func (c *Checker) checkDatabase() Check {
	start := time.Now()
	now := time.Now()

	// Simple ping by running a query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.db.DB().PingContext(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errStr := err.Error()
		return Check{
			Name:      "Database",
			Status:    StatusUnhealthy,
			Message:   "Connection failed",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}

	return Check{
		Name:      "Database",
		Status:    StatusHealthy,
		Message:   "Connected",
		Latency:   &latency,
		LastCheck: now,
	}
}

// checkDownloadClients checks all configured download clients
func (c *Checker) checkDownloadClients() []Check {
	clients, err := c.db.GetEnabledDownloadClients()
	if err != nil {
		return []Check{}
	}

	var checks []Check
	for _, client := range clients {
		checks = append(checks, c.checkSingleDownloadClient(&client))
	}

	return checks
}

// checkSingleDownloadClient checks a single download client
func (c *Checker) checkSingleDownloadClient(clientConfig *database.DownloadClient) Check {
	start := time.Now()
	now := time.Now()

	client, err := downloadclient.New(clientConfig)
	if err != nil {
		errStr := err.Error()
		return Check{
			Name:      fmt.Sprintf("Download Client: %s", clientConfig.Name),
			Status:    StatusUnhealthy,
			Message:   "Failed to initialize",
			LastCheck: now,
			Error:     &errStr,
		}
	}

	err = client.TestConnection()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errStr := err.Error()
		return Check{
			Name:      fmt.Sprintf("Download Client: %s", clientConfig.Name),
			Status:    StatusUnhealthy,
			Message:   "Connection failed",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}

	// Try to get active downloads count
	downloads, err := client.GetDownloads()
	msg := "Connected"
	if err == nil {
		msg = fmt.Sprintf("Connected, %d active downloads", len(downloads))
	}

	return Check{
		Name:      fmt.Sprintf("Download Client: %s", clientConfig.Name),
		Status:    StatusHealthy,
		Message:   msg,
		Latency:   &latency,
		LastCheck: now,
	}
}

// checkProwlarr checks Prowlarr connection if configured
func (c *Checker) checkProwlarr() *Check {
	config, err := c.db.GetProwlarrConfig()
	if err != nil || config == nil || config.URL == "" {
		return nil // Not configured
	}

	start := time.Now()
	now := time.Now()

	// Make a simple request to Prowlarr
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/health", config.URL), nil)
	if err != nil {
		errStr := err.Error()
		return &Check{
			Name:      "Prowlarr",
			Status:    StatusUnhealthy,
			Message:   "Request failed",
			LastCheck: now,
			Error:     &errStr,
		}
	}
	req.Header.Set("X-Api-Key", config.APIKey)

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errStr := err.Error()
		return &Check{
			Name:      "Prowlarr",
			Status:    StatusUnhealthy,
			Message:   "Connection failed",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errStr := fmt.Sprintf("HTTP %d", resp.StatusCode)
		return &Check{
			Name:      "Prowlarr",
			Status:    StatusUnhealthy,
			Message:   "API error",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}

	return &Check{
		Name:      "Prowlarr",
		Status:    StatusHealthy,
		Message:   "Connected",
		Latency:   &latency,
		LastCheck: now,
	}
}

// checkIndexers checks enabled indexers
func (c *Checker) checkIndexers() []Check {
	indexers, err := c.db.GetEnabledIndexers()
	if err != nil {
		return []Check{}
	}

	// Only check first 5 indexers to avoid too many checks
	limit := 5
	if len(indexers) < limit {
		limit = len(indexers)
	}

	var checks []Check
	for i := 0; i < limit; i++ {
		checks = append(checks, c.checkSingleIndexer(&indexers[i]))
	}

	return checks
}

// checkSingleIndexer checks a single indexer
func (c *Checker) checkSingleIndexer(idx *database.Indexer) Check {
	now := time.Now()

	// For Prowlarr-synced indexers, just report based on enabled status
	if idx.SyncedFromProwlarr {
		return Check{
			Name:      fmt.Sprintf("Indexer: %s", idx.Name),
			Status:    StatusHealthy,
			Message:   "Synced from Prowlarr",
			LastCheck: now,
		}
	}

	// For manual indexers, we could test but it's often rate-limited
	// Just report as healthy if enabled
	return Check{
		Name:      fmt.Sprintf("Indexer: %s", idx.Name),
		Status:    StatusHealthy,
		Message:   "Enabled",
		LastCheck: now,
	}
}

// checkDiskSpace checks disk space for library paths
func (c *Checker) checkDiskSpace() []Check {
	libraries, err := c.db.GetLibraries()
	if err != nil {
		return []Check{}
	}

	// Check unique paths
	checkedPaths := make(map[string]bool)
	var checks []Check

	for _, lib := range libraries {
		if checkedPaths[lib.Path] {
			continue
		}
		checkedPaths[lib.Path] = true

		now := time.Now()
		usage, err := storage.GetDiskUsage(lib.Path)

		if err != nil {
			errStr := err.Error()
			checks = append(checks, Check{
				Name:      fmt.Sprintf("Disk: %s", lib.Name),
				Status:    StatusWarning,
				Message:   "Unable to check",
				LastCheck: now,
				Error:     &errStr,
			})
			continue
		}

		usedPercent := usage.UsedPercent
		freeGB := usage.Free / (1024 * 1024 * 1024)

		status := StatusHealthy
		if usedPercent >= 95 {
			status = StatusUnhealthy
		} else if usedPercent >= 80 {
			status = StatusWarning
		}

		msg := fmt.Sprintf("%.0f%% used (%d GB free)", usedPercent, freeGB)

		checks = append(checks, Check{
			Name:      fmt.Sprintf("Disk: %s", lib.Name),
			Status:    status,
			Message:   msg,
			LastCheck: now,
		})
	}

	// Also check common paths
	for _, checkPath := range []string{"/media", "/app/data", "/"} {
		if checkedPaths[checkPath] {
			continue
		}

		now := time.Now()
		usage, err := storage.GetDiskUsage(checkPath)
		if err != nil {
			continue // Path doesn't exist
		}

		usedPercent := usage.UsedPercent
		freeGB := usage.Free / (1024 * 1024 * 1024)

		status := StatusHealthy
		if usedPercent >= 95 {
			status = StatusUnhealthy
		} else if usedPercent >= 80 {
			status = StatusWarning
		}

		msg := fmt.Sprintf("%.0f%% used (%d GB free)", usedPercent, freeGB)

		checks = append(checks, Check{
			Name:      fmt.Sprintf("Disk: %s", checkPath),
			Status:    status,
			Message:   msg,
			LastCheck: now,
		})

		break // Only report one system disk
	}

	return checks
}

// checkTMDB checks TMDB API connectivity
func (c *Checker) checkTMDB() Check {
	now := time.Now()

	c.mu.RLock()
	apiKey := c.tmdbKey
	c.mu.RUnlock()

	if apiKey == "" {
		// Try to get from settings
		key, _ := c.db.GetSetting("tmdb_api_key")
		if key != "" {
			c.SetTMDBKey(key)
			apiKey = key
		}
	}

	if apiKey == "" {
		return Check{
			Name:      "TMDB API",
			Status:    StatusWarning,
			Message:   "Not configured",
			LastCheck: now,
		}
	}

	start := time.Now()

	// Make a simple API call to TMDB
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.themoviedb.org/3/configuration?api_key=%s", apiKey)
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		errStr := err.Error()
		return Check{
			Name:      "TMDB API",
			Status:    StatusUnhealthy,
			Message:   "Connection failed",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		errStr := "Invalid API key"
		return Check{
			Name:      "TMDB API",
			Status:    StatusUnhealthy,
			Message:   "Authentication failed",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}

	if resp.StatusCode != http.StatusOK {
		errStr := fmt.Sprintf("HTTP %d", resp.StatusCode)
		return Check{
			Name:      "TMDB API",
			Status:    StatusUnhealthy,
			Message:   "API error",
			Latency:   &latency,
			LastCheck: now,
			Error:     &errStr,
		}
	}

	// Check latency
	status := StatusHealthy
	if latency > 5000 {
		status = StatusWarning
	}

	return Check{
		Name:      "TMDB API",
		Status:    status,
		Message:   "Reachable",
		Latency:   &latency,
		LastCheck: now,
	}
}
