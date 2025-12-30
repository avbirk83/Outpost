package downloadclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/outpost/outpost/internal/database"
)

// QBittorrent implements the Client interface for qBittorrent
type QBittorrent struct {
	config *database.DownloadClient
	client *http.Client
	baseURL string
}

// NewQBittorrent creates a new qBittorrent client
func NewQBittorrent(config *database.DownloadClient) *QBittorrent {
	scheme := "http"
	if config.UseTLS {
		scheme = "https"
	}

	jar, _ := cookiejar.New(nil)
	return &QBittorrent{
		config:  config,
		baseURL: fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.Port),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

// qbTorrent represents a torrent in qBittorrent's API response
type qbTorrent struct {
	Hash           string  `json:"hash"`
	Name           string  `json:"name"`
	Size           int64   `json:"size"`
	Downloaded     int64   `json:"downloaded"`
	Progress       float64 `json:"progress"`
	DlSpeed        int64   `json:"dlspeed"`
	ETA            int64   `json:"eta"`
	State          string  `json:"state"`
	SavePath       string  `json:"save_path"`
	Category       string  `json:"category"`
}

func (q *QBittorrent) login() error {
	data := url.Values{
		"username": {q.config.Username},
		"password": {q.config.Password},
	}

	resp, err := q.client.PostForm(q.baseURL+"/api/v2/auth/login", data)
	if err != nil {
		return fmt.Errorf("failed to connect to qBittorrent: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", string(body))
	}

	if string(body) != "Ok." {
		return fmt.Errorf("login failed: %s", string(body))
	}

	return nil
}

func (q *QBittorrent) TestConnection() error {
	if err := q.login(); err != nil {
		return err
	}

	resp, err := q.client.Get(q.baseURL + "/api/v2/app/version")
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get version: status %d", resp.StatusCode)
	}

	return nil
}

func (q *QBittorrent) GetDownloads() ([]Download, error) {
	if err := q.login(); err != nil {
		return nil, err
	}

	resp, err := q.client.Get(q.baseURL + "/api/v2/torrents/info")
	if err != nil {
		return nil, fmt.Errorf("failed to get torrents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get torrents: status %d", resp.StatusCode)
	}

	var torrents []qbTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	downloads := make([]Download, len(torrents))
	for i, t := range torrents {
		downloads[i] = Download{
			ID:         t.Hash,
			Name:       t.Name,
			Size:       t.Size,
			Downloaded: t.Downloaded,
			Progress:   t.Progress * 100,
			Speed:      t.DlSpeed,
			ETA:        t.ETA,
			Status:     q.mapState(t.State),
			SavePath:   t.SavePath,
			Category:   t.Category,
		}
	}

	return downloads, nil
}

func (q *QBittorrent) mapState(state string) string {
	switch state {
	case "downloading", "forcedDL", "metaDL", "stalledDL":
		return "downloading"
	case "uploading", "forcedUP", "stalledUP":
		return "completed"
	case "pausedDL", "pausedUP":
		return "paused"
	case "queuedDL", "queuedUP", "checkingDL", "checkingUP", "allocating":
		return "queued"
	case "error", "missingFiles":
		return "error"
	default:
		return "unknown"
	}
}

func (q *QBittorrent) AddTorrent(torrentURL string, category string) error {
	if err := q.login(); err != nil {
		return err
	}

	data := url.Values{
		"urls": {torrentURL},
	}
	if category != "" {
		data.Set("category", category)
	} else if q.config.Category != "" {
		data.Set("category", q.config.Category)
	}

	resp, err := q.client.PostForm(q.baseURL+"/api/v2/torrents/add", data)
	if err != nil {
		return fmt.Errorf("failed to add torrent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add torrent: %s", string(body))
	}

	return nil
}

func (q *QBittorrent) AddNZB(url string, category string) error {
	return fmt.Errorf("qBittorrent does not support NZB files")
}

func (q *QBittorrent) PauseDownload(id string) error {
	if err := q.login(); err != nil {
		return err
	}

	data := url.Values{
		"hashes": {id},
	}

	resp, err := q.client.PostForm(q.baseURL+"/api/v2/torrents/pause", data)
	if err != nil {
		return fmt.Errorf("failed to pause torrent: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (q *QBittorrent) ResumeDownload(id string) error {
	if err := q.login(); err != nil {
		return err
	}

	data := url.Values{
		"hashes": {id},
	}

	resp, err := q.client.PostForm(q.baseURL+"/api/v2/torrents/resume", data)
	if err != nil {
		return fmt.Errorf("failed to resume torrent: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (q *QBittorrent) DeleteDownload(id string, deleteFiles bool) error {
	if err := q.login(); err != nil {
		return err
	}

	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}

	data := url.Values{
		"hashes":      {id},
		"deleteFiles": {deleteFilesStr},
	}

	resp, err := q.client.PostForm(q.baseURL+"/api/v2/torrents/delete", data)
	if err != nil {
		return fmt.Errorf("failed to delete torrent: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (q *QBittorrent) GetCategories() ([]string, error) {
	if err := q.login(); err != nil {
		return nil, err
	}

	resp, err := q.client.Get(q.baseURL + "/api/v2/torrents/categories")
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get categories: status %d", resp.StatusCode)
	}

	var categories map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]string, 0, len(categories))
	for name := range categories {
		result = append(result, name)
	}

	return result, nil
}

func (q *QBittorrent) GetClientType() string {
	return "torrent"
}

// Helper to check if a string contains any of the given substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
