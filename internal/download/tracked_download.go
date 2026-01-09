package download

import (
	"fmt"
	"time"

	"github.com/outpost/outpost/internal/parser"
)

// DownloadState represents the current state of a tracked download
type DownloadState string

const (
	StateQueued        DownloadState = "queued"
	StateDownloading   DownloadState = "downloading"
	StatePaused        DownloadState = "paused"
	StateStalled       DownloadState = "stalled"
	StateCompleted     DownloadState = "completed"
	StateImportPending DownloadState = "import_pending"
	StateImporting     DownloadState = "importing"
	StateImported      DownloadState = "imported"
	StateImportBlocked DownloadState = "import_blocked"
	StateFailed        DownloadState = "failed"
	StateIgnored       DownloadState = "ignored"
)

// ValidTransitions defines allowed state transitions
var ValidTransitions = map[DownloadState][]DownloadState{
	StateQueued:        {StateDownloading, StateFailed},
	StateDownloading:   {StateCompleted, StatePaused, StateStalled, StateFailed},
	StatePaused:        {StateDownloading, StateFailed},
	StateStalled:       {StateDownloading, StateFailed, StateIgnored},
	StateCompleted:     {StateImportPending},
	StateImportPending: {StateImporting, StateImportBlocked},
	StateImporting:     {StateImported, StateImportBlocked, StateFailed},
	StateImportBlocked: {StateImporting, StateIgnored},
	StateImported:      {}, // Terminal state
	StateFailed:        {StateQueued}, // Can retry
	StateIgnored:       {}, // Terminal state
}

// TrackedDownload represents a download being monitored through its lifecycle
type TrackedDownload struct {
	ID               int64         `json:"id"`
	DownloadClientID int64         `json:"downloadClientId"`
	ExternalID       string        `json:"externalId"`
	RequestID        *int64        `json:"requestId,omitempty"`
	MediaID          *int64        `json:"mediaId,omitempty"`
	MediaType        string        `json:"mediaType"`

	State          DownloadState `json:"state"`
	PreviousState  DownloadState `json:"previousState,omitempty"`
	StateChangedAt time.Time     `json:"stateChangedAt"`

	Title      string                `json:"title"`
	ParsedInfo *parser.ParsedRelease `json:"parsedInfo,omitempty"`

	// Progress tracking
	Size       int64         `json:"size"`
	Downloaded int64         `json:"downloaded"`
	Progress   float64       `json:"progress"`
	Speed      int64         `json:"speed"`
	ETA        time.Duration `json:"eta"`
	Seeders    int           `json:"seeders"`

	// Paths
	DownloadPath string `json:"downloadPath,omitempty"`
	ImportPath   string `json:"importPath,omitempty"`

	// Quality info
	Quality           string `json:"quality,omitempty"`
	CustomFormatScore int    `json:"customFormatScore"`

	// Timestamps
	GrabbedAt   time.Time  `json:"grabbedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	ImportedAt  *time.Time `json:"importedAt,omitempty"`

	// Issues
	Warnings          []string `json:"warnings,omitempty"`
	Errors            []string `json:"errors,omitempty"`
	ImportBlockReason string   `json:"importBlockReason,omitempty"`

	// Seeding
	Ratio       float64       `json:"ratio"`
	SeedingTime time.Duration `json:"seedingTime"`
	CanRemove   bool          `json:"canRemove"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DownloadEvent records a state change in the download lifecycle
type DownloadEvent struct {
	ID         int64         `json:"id"`
	DownloadID int64         `json:"downloadId"`
	FromState  DownloadState `json:"fromState,omitempty"`
	ToState    DownloadState `json:"toState"`
	Reason     string        `json:"reason,omitempty"`
	Details    string        `json:"details,omitempty"`
	CreatedAt  time.Time     `json:"createdAt"`
}

// SeedingConfig defines when a torrent can be removed
type SeedingConfig struct {
	MinRatio    float64       `json:"minRatio"`    // e.g., 1.0
	MinSeedTime time.Duration `json:"minSeedTime"` // e.g., 24h
	MaxSeedTime time.Duration `json:"maxSeedTime"` // e.g., 7d
}

// DefaultSeedingConfig returns sensible defaults
func DefaultSeedingConfig() SeedingConfig {
	return SeedingConfig{
		MinRatio:    1.0,
		MinSeedTime: 24 * time.Hour,
		MaxSeedTime: 7 * 24 * time.Hour,
	}
}

// CanTransitionTo checks if a state transition is valid
func (t *TrackedDownload) CanTransitionTo(newState DownloadState) bool {
	allowed, ok := ValidTransitions[t.State]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == newState {
			return true
		}
	}
	return false
}

// TransitionTo attempts to transition to a new state
func (t *TrackedDownload) TransitionTo(newState DownloadState, reason string) error {
	if !t.CanTransitionTo(newState) {
		return fmt.Errorf("invalid transition from %s to %s", t.State, newState)
	}

	t.PreviousState = t.State
	t.State = newState
	t.StateChangedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Set timestamps for specific transitions
	switch newState {
	case StateCompleted:
		now := time.Now()
		t.CompletedAt = &now
	case StateImported:
		now := time.Now()
		t.ImportedAt = &now
	}

	return nil
}

// CanRemoveFromClient checks if the download meets removal criteria
func (t *TrackedDownload) CanRemoveFromClient(config SeedingConfig) bool {
	if t.State != StateImported {
		return false
	}

	ratioMet := t.Ratio >= config.MinRatio
	timeMet := t.SeedingTime >= config.MinSeedTime
	maxTimeHit := t.SeedingTime >= config.MaxSeedTime

	return maxTimeHit || (ratioMet && timeMet)
}

// IsActive returns true if the download is still in progress
func (t *TrackedDownload) IsActive() bool {
	switch t.State {
	case StateQueued, StateDownloading, StatePaused, StateStalled:
		return true
	}
	return false
}

// IsPending returns true if the download is waiting for import
func (t *TrackedDownload) IsPending() bool {
	switch t.State {
	case StateCompleted, StateImportPending, StateImportBlocked:
		return true
	}
	return false
}

// IsTerminal returns true if the download is in a final state
func (t *TrackedDownload) IsTerminal() bool {
	switch t.State {
	case StateImported, StateIgnored:
		return true
	}
	return false
}

// HasError returns true if there are any errors
func (t *TrackedDownload) HasError() bool {
	return len(t.Errors) > 0 || t.State == StateFailed
}

// AddWarning adds a warning message
func (t *TrackedDownload) AddWarning(msg string) {
	t.Warnings = append(t.Warnings, msg)
}

// AddError adds an error message
func (t *TrackedDownload) AddError(msg string) {
	t.Errors = append(t.Errors, msg)
}

// SetImportBlocked marks the download as blocked with a reason
func (t *TrackedDownload) SetImportBlocked(reason string) error {
	if err := t.TransitionTo(StateImportBlocked, reason); err != nil {
		return err
	}
	t.ImportBlockReason = reason
	return nil
}
