package downloadclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
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
	log.Printf("DEBUG qBit AddTorrent: starting, URL length=%d, category=%s", len(torrentURL), category)

	if err := q.login(); err != nil {
		log.Printf("DEBUG qBit AddTorrent: login failed: %v", err)
		return err
	}
	log.Printf("DEBUG qBit AddTorrent: login successful")

	// Check if this is a magnet link or HTTP URL
	if strings.HasPrefix(torrentURL, "magnet:") {
		// Magnet links can be sent directly via URL
		log.Printf("DEBUG qBit AddTorrent: detected magnet link, sending directly")
		return q.addTorrentByURL(torrentURL, category)
	}

	// For HTTP URLs (like Prowlarr download links), we need to resolve them first
	// They may redirect to magnet links or return .torrent files
	log.Printf("DEBUG qBit AddTorrent: resolving download URL")
	resolvedURL, torrentData, err := q.resolveDownloadURL(torrentURL)
	if err != nil {
		log.Printf("DEBUG qBit AddTorrent: failed to resolve URL: %v", err)
		return err
	}

	// If we got a magnet link, send it directly
	if strings.HasPrefix(resolvedURL, "magnet:") {
		log.Printf("DEBUG qBit AddTorrent: URL resolved to magnet link, sending directly")
		return q.addTorrentByURL(resolvedURL, category)
	}

	// If we got torrent data, upload it
	if len(torrentData) > 0 {
		log.Printf("DEBUG qBit AddTorrent: got %d bytes of torrent data, uploading", len(torrentData))
		return q.addTorrentByFile(torrentData, category)
	}

	// Shouldn't get here, but fall back to URL method
	log.Printf("DEBUG qBit AddTorrent: falling back to URL method")
	return q.addTorrentByURL(torrentURL, category)
}

// resolveDownloadURL follows redirects and returns either a magnet link or torrent file data
func (q *QBittorrent) resolveDownloadURL(downloadURL string) (string, []byte, error) {
	// Create a client that doesn't auto-follow redirects so we can catch magnet redirects
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Check if redirecting to a magnet link
			if strings.HasPrefix(req.URL.String(), "magnet:") {
				return http.ErrUseLastResponse
			}
			// Allow up to 10 redirects for HTTP URLs
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got redirected to a magnet link
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect {
		location := resp.Header.Get("Location")
		if strings.HasPrefix(location, "magnet:") {
			log.Printf("DEBUG qBit resolveDownloadURL: got magnet redirect: %s", location[:min(100, len(location))])
			return location, nil, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the response is a magnet link (some APIs return magnet as plain text)
	dataStr := strings.TrimSpace(string(data))
	if strings.HasPrefix(dataStr, "magnet:") {
		log.Printf("DEBUG qBit resolveDownloadURL: response body is magnet link")
		return dataStr, nil, nil
	}

	// Otherwise it should be torrent file data
	log.Printf("DEBUG qBit resolveDownloadURL: got %d bytes of torrent data", len(data))
	return "", data, nil
}

func (q *QBittorrent) addTorrentByURL(torrentURL string, category string) error {
	data := url.Values{
		"urls": {torrentURL},
	}
	if category != "" {
		data.Set("category", category)
	} else if q.config.Category != "" {
		data.Set("category", q.config.Category)
	}

	log.Printf("DEBUG qBit addTorrentByURL: sending to %s/api/v2/torrents/add", q.baseURL)
	resp, err := q.client.PostForm(q.baseURL+"/api/v2/torrents/add", data)
	if err != nil {
		log.Printf("DEBUG qBit addTorrentByURL: request failed: %v", err)
		return fmt.Errorf("failed to add torrent: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("DEBUG qBit addTorrentByURL: response status=%d, body=%q", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add torrent: %s", string(body))
	}

	return nil
}

func (q *QBittorrent) addTorrentByFile(torrentData []byte, category string) error {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the torrent file
	part, err := writer.CreateFormFile("torrents", "download.torrent")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(torrentData); err != nil {
		return fmt.Errorf("failed to write torrent data: %w", err)
	}

	// Add category
	if category != "" {
		writer.WriteField("category", category)
	} else if q.config.Category != "" {
		writer.WriteField("category", q.config.Category)
	}

	writer.Close()

	req, err := http.NewRequest("POST", q.baseURL+"/api/v2/torrents/add", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	log.Printf("DEBUG qBit addTorrentByFile: uploading torrent file to %s/api/v2/torrents/add", q.baseURL)
	resp, err := q.client.Do(req)
	if err != nil {
		log.Printf("DEBUG qBit addTorrentByFile: request failed: %v", err)
		return fmt.Errorf("failed to add torrent: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("DEBUG qBit addTorrentByFile: response status=%d, body=%q", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
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
	log.Printf("DEBUG qBit DeleteDownload: hash=%s, deleteFiles=%v", id, deleteFiles)

	if err := q.login(); err != nil {
		log.Printf("DEBUG qBit DeleteDownload: login failed: %v", err)
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
		log.Printf("DEBUG qBit DeleteDownload: request failed: %v", err)
		return fmt.Errorf("failed to delete torrent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("DEBUG qBit DeleteDownload: unexpected status %d", resp.StatusCode)
		return fmt.Errorf("delete returned status %d", resp.StatusCode)
	}

	log.Printf("DEBUG qBit DeleteDownload: success for hash=%s", id)
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
