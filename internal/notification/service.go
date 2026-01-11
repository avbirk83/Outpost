package notification

import (
	"log"

	"github.com/outpost/outpost/internal/database"
)

// NotificationType constants
const (
	TypeNewContent        = "new_content"
	TypeRequestApproved   = "request_approved"
	TypeRequestDenied     = "request_denied"
	TypeDownloadComplete  = "download_complete"
	TypeDownloadFailed    = "download_failed"
)

// Service handles in-app notifications
type Service struct {
	db *database.Database
}

// New creates a new notification service
func New(db *database.Database) *Service {
	return &Service{db: db}
}

// Create creates a notification for a specific user
func (s *Service) Create(userID int64, notifType, title, message string, imageURL, link *string) error {
	err := s.db.CreateNotification(userID, notifType, title, message, imageURL, link)
	if err != nil {
		log.Printf("Failed to create notification for user %d: %v", userID, err)
	}
	return err
}

// CreateForAdmins creates a notification for all admin users
func (s *Service) CreateForAdmins(notifType, title, message string, imageURL, link *string) error {
	adminIDs, err := s.db.GetAdminUserIDs()
	if err != nil {
		log.Printf("Failed to get admin IDs for notification: %v", err)
		return err
	}

	for _, adminID := range adminIDs {
		if err := s.db.CreateNotification(adminID, notifType, title, message, imageURL, link); err != nil {
			log.Printf("Failed to create notification for admin %d: %v", adminID, err)
		}
	}
	return nil
}

// GetForUser returns notifications for a user
func (s *Service) GetForUser(userID int64, unreadOnly bool, limit int) ([]database.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.db.GetNotifications(userID, unreadOnly, limit)
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *Service) GetUnreadCount(userID int64) (int, error) {
	return s.db.GetUnreadNotificationCount(userID)
}

// MarkRead marks a notification as read
func (s *Service) MarkRead(notificationID int64) error {
	return s.db.MarkNotificationRead(notificationID)
}

// MarkAllRead marks all notifications as read for a user
func (s *Service) MarkAllRead(userID int64) error {
	return s.db.MarkAllNotificationsRead(userID)
}

// Delete deletes a notification
func (s *Service) Delete(notificationID int64) error {
	return s.db.DeleteNotification(notificationID)
}

// Cleanup removes old read notifications
func (s *Service) Cleanup(olderThanDays int) error {
	if olderThanDays <= 0 {
		olderThanDays = 30
	}
	return s.db.CleanupOldNotifications(olderThanDays)
}

// Helper methods for creating specific notification types

// NotifyNewContent notifies a user that their requested content is now available
func (s *Service) NotifyNewContent(userID int64, title, mediaType string, mediaID int64, posterPath *string) error {
	message := title + " is now available in your library"
	var link string
	if mediaType == "movie" {
		link = "/movies/" + itoa(mediaID)
	} else {
		link = "/tv/" + itoa(mediaID)
	}
	return s.Create(userID, TypeNewContent, "New Content Available", message, posterPath, &link)
}

// NotifyRequestApproved notifies a user that their request was approved
func (s *Service) NotifyRequestApproved(userID int64, title string, tmdbID int64, mediaType string, posterPath *string) error {
	message := "Your request for \"" + title + "\" has been approved"
	var link string
	if mediaType == "movie" {
		link = "/explore/movie/" + itoa(tmdbID)
	} else {
		link = "/explore/show/" + itoa(tmdbID)
	}
	return s.Create(userID, TypeRequestApproved, "Request Approved", message, posterPath, &link)
}

// NotifyRequestDenied notifies a user that their request was denied
func (s *Service) NotifyRequestDenied(userID int64, title string, reason string, posterPath *string) error {
	message := "Your request for \"" + title + "\" was denied"
	if reason != "" {
		message += ": " + reason
	}
	return s.Create(userID, TypeRequestDenied, "Request Denied", message, posterPath, nil)
}

// NotifyDownloadComplete notifies admins that a download completed
func (s *Service) NotifyDownloadComplete(title string, mediaType string, mediaID int64, posterPath *string) error {
	message := title + " has finished downloading"
	var link string
	if mediaType == "movie" {
		link = "/movies/" + itoa(mediaID)
	} else {
		link = "/tv/" + itoa(mediaID)
	}
	return s.CreateForAdmins(TypeDownloadComplete, "Download Complete", message, posterPath, &link)
}

// NotifyDownloadFailed notifies admins that a download failed
func (s *Service) NotifyDownloadFailed(title string, errorMsg string, posterPath *string) error {
	message := "Download failed for \"" + title + "\""
	if errorMsg != "" {
		message += ": " + errorMsg
	}
	link := "/activity"
	return s.CreateForAdmins(TypeDownloadFailed, "Download Failed", message, posterPath, &link)
}

// Simple int64 to string conversion without importing strconv
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}
