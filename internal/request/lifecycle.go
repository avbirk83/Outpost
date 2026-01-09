package request

import (
	"database/sql"
	"fmt"
	"time"
)

// Status represents the state of a request
type Status string

const (
	StatusPending    Status = "requested"  // Awaiting approval
	StatusApproved   Status = "approved"   // Approved, searching
	StatusDeclined   Status = "denied"     // Declined by admin
	StatusProcessing Status = "processing" // Download in progress
	StatusFailed     Status = "failed"     // Download/import failed
	StatusAvailable  Status = "available"  // Media is ready
)

// ValidTransitions defines allowed status transitions
var ValidTransitions = map[Status][]Status{
	StatusPending:    {StatusApproved, StatusDeclined},
	StatusApproved:   {StatusProcessing, StatusFailed, StatusDeclined},
	StatusDeclined:   {},                                               // Terminal
	StatusProcessing: {StatusAvailable, StatusFailed},
	StatusFailed:     {StatusApproved}, // Can retry
	StatusAvailable:  {},               // Terminal
}

// LifecycleManager handles request status transitions
type LifecycleManager struct {
	db *sql.DB
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(db *sql.DB) *LifecycleManager {
	return &LifecycleManager{db: db}
}

// CanTransition checks if a transition is valid
func (l *LifecycleManager) CanTransition(from, to Status) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// UpdateStatus updates a request's status
func (l *LifecycleManager) UpdateStatus(requestID int64, newStatus Status, reason string) error {
	// Get current status
	var current string
	err := l.db.QueryRow("SELECT status FROM requests WHERE id = ?", requestID).Scan(&current)
	if err != nil {
		return err
	}

	// Check if transition is valid
	if !l.CanTransition(Status(current), newStatus) {
		return fmt.Errorf("invalid transition from %s to %s", current, newStatus)
	}

	// Update status
	_, err = l.db.Exec(`
		UPDATE requests SET status = ?, status_reason = ?, updated_at = ? WHERE id = ?`,
		newStatus, reason, time.Now(), requestID)

	return err
}

// Approve approves a pending request
func (l *LifecycleManager) Approve(requestID int64) error {
	return l.UpdateStatus(requestID, StatusApproved, "")
}

// Decline declines a pending request with a reason
func (l *LifecycleManager) Decline(requestID int64, reason string) error {
	return l.UpdateStatus(requestID, StatusDeclined, reason)
}

// MarkProcessing marks a request as processing (download started)
func (l *LifecycleManager) MarkProcessing(requestID int64) error {
	return l.UpdateStatus(requestID, StatusProcessing, "Download in progress")
}

// MarkFailed marks a request as failed with an error message
func (l *LifecycleManager) MarkFailed(requestID int64, errorMsg string) error {
	return l.UpdateStatus(requestID, StatusFailed, errorMsg)
}

// MarkAvailable marks a request as available (media ready)
func (l *LifecycleManager) MarkAvailable(requestID int64) error {
	return l.UpdateStatus(requestID, StatusAvailable, "Media is now available")
}

// RetryRequest retries a failed request
func (l *LifecycleManager) RetryRequest(requestID int64) error {
	return l.UpdateStatus(requestID, StatusApproved, "Retrying")
}

// LinkDownload links a request to a tracked download
func (l *LifecycleManager) LinkDownload(requestID, downloadID int64) error {
	_, err := l.db.Exec(`
		INSERT OR IGNORE INTO request_downloads (request_id, download_id, created_at)
		VALUES (?, ?, ?)`, requestID, downloadID, time.Now())
	return err
}

// GetLinkedDownloads returns all download IDs linked to a request
func (l *LifecycleManager) GetLinkedDownloads(requestID int64) ([]int64, error) {
	rows, err := l.db.Query(`
		SELECT download_id FROM request_downloads WHERE request_id = ?`, requestID)
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

// GetRequestsForDownload returns all request IDs linked to a download
func (l *LifecycleManager) GetRequestsForDownload(downloadID int64) ([]int64, error) {
	rows, err := l.db.Query(`
		SELECT request_id FROM request_downloads WHERE download_id = ?`, downloadID)
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

// GetPendingRequests returns all requests awaiting approval
func (l *LifecycleManager) GetPendingRequests() ([]int64, error) {
	return l.getRequestsByStatus(StatusPending)
}

// GetApprovedRequests returns all approved requests (ready to search)
func (l *LifecycleManager) GetApprovedRequests() ([]int64, error) {
	return l.getRequestsByStatus(StatusApproved)
}

// GetProcessingRequests returns all requests with active downloads
func (l *LifecycleManager) GetProcessingRequests() ([]int64, error) {
	return l.getRequestsByStatus(StatusProcessing)
}

func (l *LifecycleManager) getRequestsByStatus(status Status) ([]int64, error) {
	rows, err := l.db.Query(`SELECT id FROM requests WHERE status = ?`, status)
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

// AutoApproveSettings defines auto-approval rules
type AutoApproveSettings struct {
	AutoApproveMovies  bool
	AutoApproveTVShows bool
	AutoApproveUsers   []int64 // Users always auto-approved
}

// ShouldAutoApprove checks if a request should be auto-approved
func (l *LifecycleManager) ShouldAutoApprove(userID int64, mediaType string, settings AutoApproveSettings) bool {
	// Check if user is in auto-approve list
	for _, id := range settings.AutoApproveUsers {
		if id == userID {
			return true
		}
	}

	// Check media type rules
	switch mediaType {
	case "movie":
		return settings.AutoApproveMovies
	case "show", "tv":
		return settings.AutoApproveTVShows
	}

	return false
}
