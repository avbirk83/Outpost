package downloadclient

import (
	"fmt"

	"github.com/outpost/outpost/internal/database"
)

// Download represents an active or completed download
type Download struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	Downloaded    int64   `json:"downloaded"`
	Progress      float64 `json:"progress"` // 0-100
	Speed         int64   `json:"speed"`    // bytes/sec
	ETA           int64   `json:"eta"`      // seconds
	Status        string  `json:"status"`   // downloading, paused, completed, error, queued
	SavePath      string  `json:"savePath"`
	Category      string  `json:"category"`
	ClientID      int64   `json:"clientId"`
	ClientName    string  `json:"clientName"`
	ClientType    string  `json:"clientType"`
}

// Client interface that all download clients must implement
type Client interface {
	// TestConnection tests if we can connect to the download client
	TestConnection() error

	// GetDownloads returns a list of all active/queued downloads
	GetDownloads() ([]Download, error)

	// AddTorrent adds a torrent by URL or magnet link (torrent clients only)
	AddTorrent(url string, category string) error

	// AddNZB adds an NZB by URL (usenet clients only)
	AddNZB(url string, category string) error

	// PauseDownload pauses a specific download
	PauseDownload(id string) error

	// ResumeDownload resumes a specific download
	ResumeDownload(id string) error

	// DeleteDownload removes a download (optionally with files)
	DeleteDownload(id string, deleteFiles bool) error

	// GetCategories returns available categories/labels
	GetCategories() ([]string, error)

	// GetClientType returns the type of client (torrent/usenet)
	GetClientType() string
}

// New creates a new download client based on the config
func New(config *database.DownloadClient) (Client, error) {
	switch config.Type {
	case "qbittorrent":
		return NewQBittorrent(config), nil
	case "transmission":
		return NewTransmission(config), nil
	case "sabnzbd":
		return NewSABnzbd(config), nil
	case "nzbget":
		return NewNZBGet(config), nil
	default:
		return nil, fmt.Errorf("unknown client type: %s", config.Type)
	}
}

// Manager manages multiple download clients
type Manager struct {
	db *database.Database
}

// NewManager creates a new download client manager
func NewManager(db *database.Database) *Manager {
	return &Manager{db: db}
}

// GetAllDownloads returns downloads from all enabled clients
func (m *Manager) GetAllDownloads() ([]Download, error) {
	clients, err := m.db.GetEnabledDownloadClients()
	if err != nil {
		return nil, err
	}

	var allDownloads []Download
	for _, clientConfig := range clients {
		client, err := New(&clientConfig)
		if err != nil {
			continue // Skip clients we can't initialize
		}

		downloads, err := client.GetDownloads()
		if err != nil {
			continue // Skip clients we can't connect to
		}

		// Add client info to each download
		for i := range downloads {
			downloads[i].ClientID = clientConfig.ID
			downloads[i].ClientName = clientConfig.Name
			downloads[i].ClientType = clientConfig.Type
		}

		allDownloads = append(allDownloads, downloads...)
	}

	return allDownloads, nil
}

// TestClient tests connection to a specific client
func (m *Manager) TestClient(id int64) error {
	clientConfig, err := m.db.GetDownloadClient(id)
	if err != nil {
		return err
	}

	client, err := New(clientConfig)
	if err != nil {
		return err
	}

	return client.TestConnection()
}

// AddDownload adds a download to the first available client of the appropriate type
func (m *Manager) AddDownload(url string, category string, isTorrent bool) error {
	clients, err := m.db.GetEnabledDownloadClients()
	if err != nil {
		return err
	}

	for _, clientConfig := range clients {
		client, err := New(&clientConfig)
		if err != nil {
			continue
		}

		clientType := client.GetClientType()

		// Match torrent URLs to torrent clients, NZB URLs to usenet clients
		if isTorrent && clientType == "torrent" {
			return client.AddTorrent(url, category)
		} else if !isTorrent && clientType == "usenet" {
			return client.AddNZB(url, category)
		}
	}

	return fmt.Errorf("no suitable download client available")
}

// AddTorrent adds a torrent to a specific client
func (m *Manager) AddTorrent(clientID int64, url string, category string) error {
	clientConfig, err := m.db.GetDownloadClient(clientID)
	if err != nil {
		return err
	}

	client, err := New(clientConfig)
	if err != nil {
		return err
	}

	return client.AddTorrent(url, category)
}

// AddNZB adds an NZB to a specific client
func (m *Manager) AddNZB(clientID int64, url string, category string) error {
	clientConfig, err := m.db.GetDownloadClient(clientID)
	if err != nil {
		return err
	}

	client, err := New(clientConfig)
	if err != nil {
		return err
	}

	return client.AddNZB(url, category)
}
