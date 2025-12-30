package downloadclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/outpost/outpost/internal/database"
)

// SABnzbd implements the Client interface for SABnzbd
type SABnzbd struct {
	config  *database.DownloadClient
	client  *http.Client
	baseURL string
}

// NewSABnzbd creates a new SABnzbd client
func NewSABnzbd(config *database.DownloadClient) *SABnzbd {
	scheme := "http"
	if config.UseTLS {
		scheme = "https"
	}

	return &SABnzbd{
		config:  config,
		baseURL: fmt.Sprintf("%s://%s:%d/api", scheme, config.Host, config.Port),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type sabQueue struct {
	Slots []sabSlot `json:"slots"`
}

type sabSlot struct {
	NzoID      string `json:"nzo_id"`
	Filename   string `json:"filename"`
	Size       string `json:"size"`
	SizeLeft   string `json:"sizeleft"`
	Percentage string `json:"percentage"`
	Status     string `json:"status"`
	Category   string `json:"cat"`
	TimeLeft   string `json:"timeleft"`
}

type sabHistory struct {
	Slots []sabHistorySlot `json:"slots"`
}

type sabHistorySlot struct {
	NzoID    string `json:"nzo_id"`
	Name     string `json:"name"`
	Bytes    int64  `json:"bytes"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Storage  string `json:"storage"`
}

func (s *SABnzbd) doRequest(mode string, params url.Values) (*http.Response, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("output", "json")
	params.Set("apikey", s.config.APIKey)
	params.Set("mode", mode)

	resp, err := s.client.Get(s.baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SABnzbd: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	return resp, nil
}

func (s *SABnzbd) TestConnection() error {
	resp, err := s.doRequest("version", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if _, ok := result["version"]; !ok {
		return fmt.Errorf("invalid response from SABnzbd")
	}

	return nil
}

func (s *SABnzbd) GetDownloads() ([]Download, error) {
	// Get queue (active downloads)
	queueResp, err := s.doRequest("queue", nil)
	if err != nil {
		return nil, err
	}
	defer queueResp.Body.Close()

	var queueResult struct {
		Queue sabQueue `json:"queue"`
	}
	if err := json.NewDecoder(queueResp.Body).Decode(&queueResult); err != nil {
		return nil, fmt.Errorf("failed to decode queue: %w", err)
	}

	var downloads []Download
	for _, slot := range queueResult.Queue.Slots {
		var progress float64
		fmt.Sscanf(slot.Percentage, "%f", &progress)

		downloads = append(downloads, Download{
			ID:       slot.NzoID,
			Name:     slot.Filename,
			Progress: progress,
			Status:   s.mapQueueStatus(slot.Status),
			Category: slot.Category,
		})
	}

	// Get history (completed downloads)
	historyResp, err := s.doRequest("history", url.Values{"limit": {"50"}})
	if err != nil {
		return downloads, nil // Return queue even if history fails
	}
	defer historyResp.Body.Close()

	var historyResult struct {
		History sabHistory `json:"history"`
	}
	if err := json.NewDecoder(historyResp.Body).Decode(&historyResult); err != nil {
		return downloads, nil
	}

	for _, slot := range historyResult.History.Slots {
		downloads = append(downloads, Download{
			ID:         slot.NzoID,
			Name:       slot.Name,
			Size:       slot.Bytes,
			Downloaded: slot.Bytes,
			Progress:   100,
			Status:     s.mapHistoryStatus(slot.Status),
			Category:   slot.Category,
			SavePath:   slot.Storage,
		})
	}

	return downloads, nil
}

func (s *SABnzbd) mapQueueStatus(status string) string {
	switch status {
	case "Downloading":
		return "downloading"
	case "Paused":
		return "paused"
	case "Queued":
		return "queued"
	default:
		return "downloading"
	}
}

func (s *SABnzbd) mapHistoryStatus(status string) string {
	switch status {
	case "Completed":
		return "completed"
	case "Failed":
		return "error"
	default:
		return "completed"
	}
}

func (s *SABnzbd) AddTorrent(url string, category string) error {
	return fmt.Errorf("SABnzbd does not support torrent files")
}

func (s *SABnzbd) AddNZB(nzbURL string, category string) error {
	params := url.Values{
		"name": {nzbURL},
	}

	if category != "" {
		params.Set("cat", category)
	} else if s.config.Category != "" {
		params.Set("cat", s.config.Category)
	}

	resp, err := s.doRequest("addurl", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return fmt.Errorf("failed to add NZB: %s", result.Error)
	}

	return nil
}

func (s *SABnzbd) PauseDownload(id string) error {
	params := url.Values{"value": {id}}
	resp, err := s.doRequest("queue", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *SABnzbd) ResumeDownload(id string) error {
	params := url.Values{"value": {id}}
	resp, err := s.doRequest("resume", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *SABnzbd) DeleteDownload(id string, deleteFiles bool) error {
	// Try deleting from queue first
	params := url.Values{"value": {id}}
	resp, err := s.doRequest("queue", url.Values{"name": {"delete"}, "value": {id}})
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Also try deleting from history
	if deleteFiles {
		params.Set("del_files", "1")
	}
	resp, err = s.doRequest("history", url.Values{"name": {"delete"}, "value": {id}})
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (s *SABnzbd) GetCategories() ([]string, error) {
	resp, err := s.doRequest("get_cats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Categories []string `json:"categories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %w", err)
	}

	return result.Categories, nil
}

func (s *SABnzbd) GetClientType() string {
	return "usenet"
}
