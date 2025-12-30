package downloadclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/outpost/outpost/internal/database"
)

// Transmission implements the Client interface for Transmission
type Transmission struct {
	config    *database.DownloadClient
	client    *http.Client
	baseURL   string
	sessionID string
}

// NewTransmission creates a new Transmission client
func NewTransmission(config *database.DownloadClient) *Transmission {
	scheme := "http"
	if config.UseTLS {
		scheme = "https"
	}

	return &Transmission{
		config:  config,
		baseURL: fmt.Sprintf("%s://%s:%d/transmission/rpc", scheme, config.Host, config.Port),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type transmissionRequest struct {
	Method    string      `json:"method"`
	Arguments interface{} `json:"arguments,omitempty"`
}

type transmissionResponse struct {
	Result    string          `json:"result"`
	Arguments json.RawMessage `json:"arguments"`
}

type transmissionTorrent struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	TotalSize     int64   `json:"totalSize"`
	DownloadedEver int64  `json:"downloadedEver"`
	PercentDone   float64 `json:"percentDone"`
	RateDownload  int64   `json:"rateDownload"`
	Eta           int64   `json:"eta"`
	Status        int     `json:"status"`
	DownloadDir   string  `json:"downloadDir"`
	Labels        []string `json:"labels"`
	HashString    string  `json:"hashString"`
}

func (t *Transmission) doRequest(req *transmissionRequest) (*transmissionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", t.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if t.config.Username != "" {
		httpReq.SetBasicAuth(t.config.Username, t.config.Password)
	}
	if t.sessionID != "" {
		httpReq.Header.Set("X-Transmission-Session-Id", t.sessionID)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// Handle CSRF token
	if resp.StatusCode == http.StatusConflict {
		t.sessionID = resp.Header.Get("X-Transmission-Session-Id")
		return t.doRequest(req)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s", string(body))
	}

	var result transmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Result != "success" {
		return nil, fmt.Errorf("request failed: %s", result.Result)
	}

	return &result, nil
}

func (t *Transmission) TestConnection() error {
	req := &transmissionRequest{
		Method: "session-get",
	}
	_, err := t.doRequest(req)
	return err
}

func (t *Transmission) GetDownloads() ([]Download, error) {
	req := &transmissionRequest{
		Method: "torrent-get",
		Arguments: map[string]interface{}{
			"fields": []string{
				"id", "name", "totalSize", "downloadedEver", "percentDone",
				"rateDownload", "eta", "status", "downloadDir", "labels", "hashString",
			},
		},
	}

	resp, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	var args struct {
		Torrents []transmissionTorrent `json:"torrents"`
	}
	if err := json.Unmarshal(resp.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse torrents: %w", err)
	}

	downloads := make([]Download, len(args.Torrents))
	for i, torrent := range args.Torrents {
		category := ""
		if len(torrent.Labels) > 0 {
			category = torrent.Labels[0]
		}

		downloads[i] = Download{
			ID:         torrent.HashString,
			Name:       torrent.Name,
			Size:       torrent.TotalSize,
			Downloaded: torrent.DownloadedEver,
			Progress:   torrent.PercentDone * 100,
			Speed:      torrent.RateDownload,
			ETA:        torrent.Eta,
			Status:     t.mapStatus(torrent.Status),
			SavePath:   torrent.DownloadDir,
			Category:   category,
		}
	}

	return downloads, nil
}

func (t *Transmission) mapStatus(status int) string {
	switch status {
	case 0: // Stopped
		return "paused"
	case 1, 2: // Queued to verify, verifying
		return "queued"
	case 3: // Queued to download
		return "queued"
	case 4: // Downloading
		return "downloading"
	case 5: // Queued to seed
		return "completed"
	case 6: // Seeding
		return "completed"
	default:
		return "unknown"
	}
}

func (t *Transmission) AddTorrent(torrentURL string, category string) error {
	args := map[string]interface{}{
		"filename": torrentURL,
	}

	if category != "" {
		args["labels"] = []string{category}
	} else if t.config.Category != "" {
		args["labels"] = []string{t.config.Category}
	}

	req := &transmissionRequest{
		Method:    "torrent-add",
		Arguments: args,
	}

	_, err := t.doRequest(req)
	return err
}

func (t *Transmission) AddNZB(url string, category string) error {
	return fmt.Errorf("Transmission does not support NZB files")
}

func (t *Transmission) PauseDownload(id string) error {
	req := &transmissionRequest{
		Method: "torrent-stop",
		Arguments: map[string]interface{}{
			"ids": []string{id},
		},
	}
	_, err := t.doRequest(req)
	return err
}

func (t *Transmission) ResumeDownload(id string) error {
	req := &transmissionRequest{
		Method: "torrent-start",
		Arguments: map[string]interface{}{
			"ids": []string{id},
		},
	}
	_, err := t.doRequest(req)
	return err
}

func (t *Transmission) DeleteDownload(id string, deleteFiles bool) error {
	req := &transmissionRequest{
		Method: "torrent-remove",
		Arguments: map[string]interface{}{
			"ids":               []string{id},
			"delete-local-data": deleteFiles,
		},
	}
	_, err := t.doRequest(req)
	return err
}

func (t *Transmission) GetCategories() ([]string, error) {
	// Transmission uses labels, get all unique labels from torrents
	downloads, err := t.GetDownloads()
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[string]bool)
	for _, d := range downloads {
		if d.Category != "" {
			categoryMap[d.Category] = true
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	return categories, nil
}

func (t *Transmission) GetClientType() string {
	return "torrent"
}
