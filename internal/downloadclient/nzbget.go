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

// NZBGet implements the Client interface for NZBGet
type NZBGet struct {
	config  *database.DownloadClient
	client  *http.Client
	baseURL string
}

// NewNZBGet creates a new NZBGet client
func NewNZBGet(config *database.DownloadClient) *NZBGet {
	scheme := "http"
	if config.UseTLS {
		scheme = "https"
	}

	username := config.Username
	password := config.Password
	if username == "" {
		username = "nzbget"
	}
	if password == "" {
		password = config.APIKey
	}

	return &NZBGet{
		config:  config,
		baseURL: fmt.Sprintf("%s://%s:%s@%s:%d/jsonrpc", scheme, username, password, config.Host, config.Port),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type nzbgetRequest struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
	ID      int           `json:"id"`
}

type nzbgetResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type nzbgetGroup struct {
	NZBID           int    `json:"NZBID"`
	NZBName         string `json:"NZBName"`
	FileSizeMB      int64  `json:"FileSizeMB"`
	DownloadedSizeMB int64 `json:"DownloadedSizeMB"`
	Status          string `json:"Status"`
	Category        string `json:"Category"`
	DestDir         string `json:"DestDir"`
	DownloadRate    int64  `json:"DownloadRate"`
	RemainingSizeMB int64  `json:"RemainingSizeMB"`
}

type nzbgetHistory struct {
	NZBID      int    `json:"NZBID"`
	Name       string `json:"Name"`
	FileSizeMB int64  `json:"FileSizeMB"`
	Status     string `json:"Status"`
	Category   string `json:"Category"`
	DestDir    string `json:"DestDir"`
}

func (n *NZBGet) doRequest(method string, params ...interface{}) (*nzbgetResponse, error) {
	req := &nzbgetRequest{
		Method: method,
		Params: params,
		ID:     1,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", n.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NZBGet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s", string(body))
	}

	var result nzbgetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("NZBGet error: %s", result.Error.Message)
	}

	return &result, nil
}

func (n *NZBGet) TestConnection() error {
	resp, err := n.doRequest("version")
	if err != nil {
		return err
	}

	var version string
	if err := json.Unmarshal(resp.Result, &version); err != nil {
		return fmt.Errorf("failed to parse version: %w", err)
	}

	return nil
}

func (n *NZBGet) GetDownloads() ([]Download, error) {
	// Get active queue
	queueResp, err := n.doRequest("listgroups")
	if err != nil {
		return nil, err
	}

	var groups []nzbgetGroup
	if err := json.Unmarshal(queueResp.Result, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse queue: %w", err)
	}

	var downloads []Download
	for _, g := range groups {
		sizeMB := g.FileSizeMB
		downloadedMB := g.DownloadedSizeMB
		progress := float64(0)
		if sizeMB > 0 {
			progress = float64(downloadedMB) / float64(sizeMB) * 100
		}

		var eta int64
		if g.DownloadRate > 0 {
			eta = (g.RemainingSizeMB * 1024 * 1024) / g.DownloadRate
		}

		downloads = append(downloads, Download{
			ID:         fmt.Sprintf("%d", g.NZBID),
			Name:       g.NZBName,
			Size:       sizeMB * 1024 * 1024,
			Downloaded: downloadedMB * 1024 * 1024,
			Progress:   progress,
			Speed:      g.DownloadRate,
			ETA:        eta,
			Status:     n.mapQueueStatus(g.Status),
			Category:   g.Category,
			SavePath:   g.DestDir,
		})
	}

	// Get history
	historyResp, err := n.doRequest("history", false)
	if err != nil {
		return downloads, nil // Return queue even if history fails
	}

	var history []nzbgetHistory
	if err := json.Unmarshal(historyResp.Result, &history); err != nil {
		return downloads, nil
	}

	for _, h := range history {
		downloads = append(downloads, Download{
			ID:         fmt.Sprintf("%d", h.NZBID),
			Name:       h.Name,
			Size:       h.FileSizeMB * 1024 * 1024,
			Downloaded: h.FileSizeMB * 1024 * 1024,
			Progress:   100,
			Status:     n.mapHistoryStatus(h.Status),
			Category:   h.Category,
			SavePath:   h.DestDir,
		})
	}

	return downloads, nil
}

func (n *NZBGet) mapQueueStatus(status string) string {
	switch status {
	case "DOWNLOADING":
		return "downloading"
	case "PAUSED":
		return "paused"
	case "QUEUED":
		return "queued"
	default:
		return "downloading"
	}
}

func (n *NZBGet) mapHistoryStatus(status string) string {
	switch status {
	case "SUCCESS":
		return "completed"
	case "FAILURE", "DELETED":
		return "error"
	default:
		return "completed"
	}
}

func (n *NZBGet) AddTorrent(url string, category string) error {
	return fmt.Errorf("NZBGet does not support torrent files")
}

func (n *NZBGet) AddNZB(nzbURL string, category string) error {
	cat := category
	if cat == "" {
		cat = n.config.Category
	}

	// NZBGet append method: Name, URL, Category, Priority, AddToTop, AddPaused, DupeKey, DupeScore, DupeMode
	_, err := n.doRequest("append", "", nzbURL, cat, 0, false, false, "", 0, "SCORE")
	return err
}

func (n *NZBGet) PauseDownload(id string) error {
	_, err := n.doRequest("editqueue", "GroupPause", "", []string{id})
	return err
}

func (n *NZBGet) ResumeDownload(id string) error {
	_, err := n.doRequest("editqueue", "GroupResume", "", []string{id})
	return err
}

func (n *NZBGet) DeleteDownload(id string, deleteFiles bool) error {
	action := "GroupDelete"
	if deleteFiles {
		action = "GroupFinalDelete"
	}
	_, err := n.doRequest("editqueue", action, "", []string{id})
	return err
}

func (n *NZBGet) GetCategories() ([]string, error) {
	resp, err := n.doRequest("config")
	if err != nil {
		return nil, err
	}

	var configs []struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}
	if err := json.Unmarshal(resp.Result, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var categories []string
	for _, c := range configs {
		if len(c.Name) > 8 && c.Name[:8] == "Category" && c.Name[len(c.Name)-5:] == ".Name" {
			if c.Value != "" {
				categories = append(categories, c.Value)
			}
		}
	}

	return categories, nil
}

func (n *NZBGet) GetClientType() string {
	return "usenet"
}
