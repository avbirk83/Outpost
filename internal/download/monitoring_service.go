package download

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/parser"
)

// MonitoringService polls download clients and manages download lifecycle
type MonitoringService struct {
	repo      *Repository
	clients   *downloadclient.Manager
	db        *sql.DB

	pollInterval     time.Duration
	stalledThreshold time.Duration
	seedingConfig    SeedingConfig

	// Import callback - will be set by the application
	OnReadyForImport func(td *TrackedDownload)
	OnReadyToRemove  func(td *TrackedDownload)

	stopCh  chan struct{}
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	PollInterval     time.Duration
	StalledThreshold time.Duration
	SeedingConfig    SeedingConfig
}

// DefaultMonitoringConfig returns sensible defaults
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		PollInterval:     5 * time.Second,
		StalledThreshold: 6 * time.Hour,
		SeedingConfig:    DefaultSeedingConfig(),
	}
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(db *sql.DB, clients *downloadclient.Manager, config MonitoringConfig) *MonitoringService {
	return &MonitoringService{
		repo:             NewRepository(db),
		clients:          clients,
		db:               db,
		pollInterval:     config.PollInterval,
		stalledThreshold: config.StalledThreshold,
		seedingConfig:    config.SeedingConfig,
		stopCh:           make(chan struct{}),
	}
}

// Start begins the monitoring loop
func (m *MonitoringService) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.pollLoop()

	log.Println("Download monitoring service started")
}

// Stop stops the monitoring loop
func (m *MonitoringService) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()

	log.Println("Download monitoring service stopped")
}

// pollLoop runs the main monitoring loop
func (m *MonitoringService) pollLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	// Run immediately on start
	m.poll()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.poll()
		}
	}
}

// poll checks all download clients and updates tracked downloads
func (m *MonitoringService) poll() {
	// Get all downloads from clients
	clientDownloads, err := m.clients.GetAllDownloads()
	if err != nil {
		log.Printf("Error getting downloads from clients: %v", err)
		return
	}

	// Map by client+external ID for quick lookup
	clientMap := make(map[string]downloadclient.Download)
	for _, dl := range clientDownloads {
		key := m.makeKey(dl.ClientID, dl.ID)
		clientMap[key] = dl
	}

	// Get active tracked downloads
	tracked, err := m.repo.GetActive()
	if err != nil {
		log.Printf("Error getting tracked downloads: %v", err)
		return
	}

	// Update tracked downloads from client state
	for _, td := range tracked {
		key := m.makeKey(td.DownloadClientID, td.ExternalID)
		if clientDL, ok := clientMap[key]; ok {
			m.updateFromClient(td, clientDL)
			delete(clientMap, key) // Remove from map so we know what's new
		} else {
			// Download no longer in client
			m.handleMissingFromClient(td)
		}
	}

	// Handle new downloads in clients that we're not tracking
	for _, dl := range clientMap {
		m.handleNewDownload(dl)
	}

	// Check for stalled downloads
	m.checkStalled()

	// Check for downloads ready for removal
	m.checkReadyForRemoval()
}

// makeKey creates a unique key for client+external ID
func (m *MonitoringService) makeKey(clientID int64, externalID string) string {
	return string(rune(clientID)) + ":" + externalID
}

// updateFromClient updates a tracked download from client state
func (m *MonitoringService) updateFromClient(td *TrackedDownload, dl downloadclient.Download) {
	// Update progress metrics
	td.Size = dl.Size
	td.Downloaded = int64(float64(dl.Size) * dl.Progress / 100)
	td.Progress = dl.Progress
	td.Ratio = dl.Ratio
	if td.CompletedAt != nil {
		td.SeedingTime = time.Since(*td.CompletedAt)
	}

	// Map client status to our state
	newState := m.mapClientStatus(dl.Status, td)

	// Handle state transitions
	if newState != td.State && td.CanTransitionTo(newState) {
		reason := "Client status: " + dl.Status
		if err := m.repo.UpdateState(td, newState, reason); err != nil {
			log.Printf("Error updating download state: %v", err)
			return
		}

		// Trigger callbacks on specific transitions
		if newState == StateCompleted {
			td.DownloadPath = dl.SavePath
			m.repo.Update(td)
			// Automatically transition to import_pending
			if td.CanTransitionTo(StateImportPending) {
				m.repo.UpdateState(td, StateImportPending, "Download completed")
				if m.OnReadyForImport != nil {
					m.OnReadyForImport(td)
				}
			}
		}
	} else {
		// Just update the record with new progress
		if err := m.repo.Update(td); err != nil {
			log.Printf("Error updating download: %v", err)
		}
	}
}

// mapClientStatus maps download client status to our state
func (m *MonitoringService) mapClientStatus(status string, td *TrackedDownload) DownloadState {
	switch strings.ToLower(status) {
	case "downloading", "active":
		return StateDownloading
	case "completed", "seeding":
		// If already past completed, don't go back
		if td.State == StateImportPending || td.State == StateImporting ||
			td.State == StateImported || td.State == StateImportBlocked {
			return td.State
		}
		return StateCompleted
	case "paused":
		return StatePaused
	case "stalled":
		return StateStalled
	case "error", "failed":
		return StateFailed
	case "queued", "waiting":
		return StateQueued
	default:
		return td.State // Keep current state if unknown
	}
}

// handleMissingFromClient handles when a download disappears from the client
func (m *MonitoringService) handleMissingFromClient(td *TrackedDownload) {
	// If it was imported, this is expected (we removed it)
	if td.State == StateImported {
		return
	}

	// If it's still being tracked, something went wrong
	if td.IsActive() {
		log.Printf("Download %s missing from client, marking as failed", td.Title)
		td.AddError("Download removed from client unexpectedly")
		m.repo.UpdateState(td, StateFailed, "Download missing from client")
	}
}

// handleNewDownload processes a download found in client that we're not tracking
func (m *MonitoringService) handleNewDownload(dl downloadclient.Download) {
	// Check if we already have this download
	existing, err := m.repo.GetByExternalID(dl.ClientID, dl.ID)
	if err != nil {
		log.Printf("Error checking for existing download: %v", err)
		return
	}
	if existing != nil {
		return // Already tracking
	}

	// Parse the release name
	parsed := parser.Parse(dl.Name)

	// Create new tracked download
	td := &TrackedDownload{
		DownloadClientID: dl.ClientID,
		ExternalID:       dl.ID,
		Title:            dl.Name,
		ParsedInfo:       parsed,
		State:            StateQueued,
		StateChangedAt:   time.Now(),
		Size:             dl.Size,
		Progress:         dl.Progress,
		DownloadPath:     dl.SavePath,
		GrabbedAt:        time.Now(),
	}

	// Determine media type from parsed info
	if parsed.Season > 0 || parsed.Episode > 0 {
		td.MediaType = "show"
	} else {
		td.MediaType = "movie"
	}

	// Set initial state based on client status
	td.State = m.mapClientStatus(dl.Status, td)

	if err := m.repo.Create(td); err != nil {
		log.Printf("Error creating tracked download: %v", err)
		return
	}

	log.Printf("Now tracking download: %s (state: %s)", td.Title, td.State)

	// If already completed, trigger import
	if td.State == StateCompleted {
		if td.CanTransitionTo(StateImportPending) {
			m.repo.UpdateState(td, StateImportPending, "Download already completed")
			if m.OnReadyForImport != nil {
				m.OnReadyForImport(td)
			}
		}
	}
}

// checkStalled checks for downloads that have stalled
func (m *MonitoringService) checkStalled() {
	// Get downloading downloads
	downloading, err := m.repo.GetByState(StateDownloading)
	if err != nil {
		return
	}

	now := time.Now()
	for _, td := range downloading {
		// Check if no progress for threshold duration
		if now.Sub(td.StateChangedAt) > m.stalledThreshold && td.Progress < 100 {
			// Check if progress has changed (would need to track last progress)
			// For now, mark as stalled if it's been downloading too long
			if td.CanTransitionTo(StateStalled) {
				td.AddWarning("Download appears stalled - no progress")
				m.repo.UpdateState(td, StateStalled, "No progress detected")
				log.Printf("Download %s marked as stalled", td.Title)
			}
		}
	}
}

// checkReadyForRemoval checks for imported downloads ready to be removed
func (m *MonitoringService) checkReadyForRemoval() {
	ready, err := m.repo.GetReadyForRemoval(m.seedingConfig)
	if err != nil {
		return
	}

	for _, td := range ready {
		if !td.CanRemove {
			td.CanRemove = true
			m.repo.Update(td)
			if m.OnReadyToRemove != nil {
				m.OnReadyToRemove(td)
			}
		}
	}
}

// TrackDownload manually adds a download to tracking (called after grab)
func (m *MonitoringService) TrackDownload(clientID int64, externalID string, title string, mediaID *int64, mediaType string, requestID *int64) (*TrackedDownload, error) {
	// Check if already tracking
	existing, err := m.repo.GetByExternalID(clientID, externalID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update existing with media info if provided
		if mediaID != nil {
			existing.MediaID = mediaID
		}
		if mediaType != "" {
			existing.MediaType = mediaType
		}
		if requestID != nil {
			existing.RequestID = requestID
		}
		m.repo.Update(existing)
		return existing, nil
	}

	// Parse the release
	parsed := parser.Parse(title)

	td := &TrackedDownload{
		DownloadClientID: clientID,
		ExternalID:       externalID,
		RequestID:        requestID,
		MediaID:          mediaID,
		MediaType:        mediaType,
		Title:            title,
		ParsedInfo:       parsed,
		State:            StateQueued,
		StateChangedAt:   time.Now(),
		GrabbedAt:        time.Now(),
	}

	if err := m.repo.Create(td); err != nil {
		return nil, err
	}

	log.Printf("Tracking new download: %s", title)
	return td, nil
}

// GetTrackedDownload retrieves a tracked download by ID
func (m *MonitoringService) GetTrackedDownload(id int64) (*TrackedDownload, error) {
	return m.repo.GetByID(id)
}

// GetActiveDownloads returns all non-terminal downloads
func (m *MonitoringService) GetActiveDownloads() ([]*TrackedDownload, error) {
	return m.repo.GetActive()
}

// GetPendingImports returns downloads ready for import
func (m *MonitoringService) GetPendingImports() ([]*TrackedDownload, error) {
	return m.repo.GetPendingImport()
}

// MarkImporting marks a download as currently importing
func (m *MonitoringService) MarkImporting(td *TrackedDownload) error {
	if !td.CanTransitionTo(StateImporting) {
		return nil // Already past this state
	}
	return m.repo.UpdateState(td, StateImporting, "Import started")
}

// MarkImported marks a download as successfully imported
func (m *MonitoringService) MarkImported(td *TrackedDownload, importPath string) error {
	td.ImportPath = importPath
	m.repo.Update(td)
	return m.repo.UpdateState(td, StateImported, "Import completed")
}

// MarkImportBlocked marks a download as blocked with a reason
func (m *MonitoringService) MarkImportBlocked(td *TrackedDownload, reason string) error {
	td.ImportBlockReason = reason
	m.repo.Update(td)
	return m.repo.UpdateState(td, StateImportBlocked, reason)
}

// MarkFailed marks a download as failed
func (m *MonitoringService) MarkFailed(td *TrackedDownload, errorMsg string) error {
	td.AddError(errorMsg)
	m.repo.Update(td)
	return m.repo.UpdateState(td, StateFailed, errorMsg)
}

// RetryDownload attempts to retry a failed download
func (m *MonitoringService) RetryDownload(td *TrackedDownload) error {
	if td.State != StateFailed {
		return nil
	}
	return m.repo.UpdateState(td, StateQueued, "Retry requested")
}

// IgnoreDownload marks a download as ignored
func (m *MonitoringService) IgnoreDownload(td *TrackedDownload) error {
	return m.repo.UpdateState(td, StateIgnored, "Manually ignored")
}

// DeleteTrackedDownload removes a tracked download from the database
func (m *MonitoringService) DeleteTrackedDownload(id int64) error {
	return m.repo.Delete(id)
}
