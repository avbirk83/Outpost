package subtitles

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	OpenSubtitlesAPIBase = "https://api.opensubtitles.com/api/v1"
	UserAgent            = "Outpost v1.0"
)

// Client handles OpenSubtitles API requests
type Client struct {
	APIKey     string
	httpClient *http.Client
}

// NewClient creates a new OpenSubtitles client
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Subtitle represents a subtitle result
type Subtitle struct {
	ID           string            `json:"id"`
	LanguageCode string            `json:"language"`
	LanguageName string            `json:"language_name"`
	DownloadURL  string            `json:"download_url"`
	FileName     string            `json:"file_name"`
	Release      string            `json:"release"`
	UploadDate   string            `json:"upload_date"`
	Downloads    int               `json:"downloads"`
	FPS          float64           `json:"fps"`
	Attributes   SubtitleAttributes `json:"attributes,omitempty"`
}

// SubtitleAttributes contains additional subtitle info
type SubtitleAttributes struct {
	SubtitleID string `json:"subtitle_id"`
	Language   string `json:"language"`
	Release    string `json:"release"`
	Files      []struct {
		FileID   int    `json:"file_id"`
		FileName string `json:"file_name"`
	} `json:"files"`
}

// SearchRequest contains search parameters
type SearchRequest struct {
	Query      string
	IMDbID     string
	TMDbID     int
	Year       int
	Season     int
	Episode    int
	Languages  []string
	MovieHash  string
}

// SearchResponse represents the API search response
type SearchResponse struct {
	TotalPages int `json:"total_pages"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Data       []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			SubtitleID     string  `json:"subtitle_id"`
			Language       string  `json:"language"`
			DownloadCount  int     `json:"download_count"`
			NewDownloadCount int   `json:"new_download_count"`
			HearingImpaired bool   `json:"hearing_impaired"`
			HD              bool   `json:"hd"`
			FPS             float64 `json:"fps"`
			Votes           int     `json:"votes"`
			Points          int     `json:"points"`
			Ratings         float64 `json:"ratings"`
			FromTrusted     bool    `json:"from_trusted"`
			ForeignPartsOnly bool  `json:"foreign_parts_only"`
			UploadDate      string  `json:"upload_date"`
			AITranslated    bool    `json:"ai_translated"`
			MachineTranslated bool `json:"machine_translated"`
			Release         string  `json:"release"`
			URL             string  `json:"url"`
			FeatureDetails  struct {
				FeatureID   int    `json:"feature_id"`
				FeatureType string `json:"feature_type"`
				Year        int    `json:"year"`
				Title       string `json:"title"`
				MovieName   string `json:"movie_name"`
				IMDbID      int    `json:"imdb_id"`
				TMDbID      int    `json:"tmdb_id"`
			} `json:"feature_details"`
			Files []struct {
				FileID   int    `json:"file_id"`
				FileName string `json:"file_name"`
			} `json:"files"`
		} `json:"attributes"`
	} `json:"data"`
}

// DownloadResponse represents the download link response
type DownloadResponse struct {
	Link         string `json:"link"`
	FileName     string `json:"file_name"`
	Requests     int    `json:"requests"`
	Remaining    int    `json:"remaining"`
	Message      string `json:"message"`
	ResetTime    string `json:"reset_time"`
	ResetTimeUTC string `json:"reset_time_utc"`
}

// Search searches for subtitles
func (c *Client) Search(req SearchRequest) ([]Subtitle, error) {
	params := url.Values{}

	if req.Query != "" {
		params.Set("query", req.Query)
	}
	if req.IMDbID != "" {
		params.Set("imdb_id", req.IMDbID)
	}
	if req.TMDbID > 0 {
		params.Set("tmdb_id", fmt.Sprintf("%d", req.TMDbID))
	}
	if req.Year > 0 {
		params.Set("year", fmt.Sprintf("%d", req.Year))
	}
	if req.Season > 0 {
		params.Set("season_number", fmt.Sprintf("%d", req.Season))
	}
	if req.Episode > 0 {
		params.Set("episode_number", fmt.Sprintf("%d", req.Episode))
	}
	if len(req.Languages) > 0 {
		params.Set("languages", strings.Join(req.Languages, ","))
	}
	if req.MovieHash != "" {
		params.Set("moviehash", req.MovieHash)
	}

	endpoint := fmt.Sprintf("%s/subtitles?%s", OpenSubtitlesAPIBase, params.Encode())

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Api-Key", c.APIKey)
	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	var subtitles []Subtitle
	for _, item := range searchResp.Data {
		fileName := ""
		if len(item.Attributes.Files) > 0 {
			fileName = item.Attributes.Files[0].FileName
		}

		subtitles = append(subtitles, Subtitle{
			ID:           item.ID,
			LanguageCode: item.Attributes.Language,
			Release:      item.Attributes.Release,
			FileName:     fileName,
			UploadDate:   item.Attributes.UploadDate,
			Downloads:    item.Attributes.DownloadCount,
			FPS:          item.Attributes.FPS,
		})
	}

	return subtitles, nil
}

// GetDownloadLink gets a download link for a subtitle
func (c *Client) GetDownloadLink(fileID int) (*DownloadResponse, error) {
	endpoint := fmt.Sprintf("%s/download", OpenSubtitlesAPIBase)

	body := fmt.Sprintf(`{"file_id": %d}`, fileID)

	httpReq, err := http.NewRequest("POST", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Api-Key", c.APIKey)
	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var dlResp DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&dlResp); err != nil {
		return nil, err
	}

	return &dlResp, nil
}

// Download downloads a subtitle file to the specified path
func (c *Client) Download(downloadURL, destPath string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// ComputeMovieHash computes the OpenSubtitles hash for a video file
func ComputeMovieHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := stat.Size()
	if size < 65536*2 {
		return "", fmt.Errorf("file too small for hash")
	}

	// Read first 64KB
	head := make([]byte, 65536)
	if _, err := file.Read(head); err != nil {
		return "", err
	}

	// Read last 64KB
	tail := make([]byte, 65536)
	if _, err := file.Seek(-65536, io.SeekEnd); err != nil {
		return "", err
	}
	if _, err := file.Read(tail); err != nil {
		return "", err
	}

	// Compute hash
	hash := md5.New()
	hash.Write(head)
	hash.Write(tail)
	hash.Write([]byte(fmt.Sprintf("%d", size)))

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SearchAndDownload searches for subtitles and downloads the best match
func (c *Client) SearchAndDownload(videoPath string, title string, year int, language string) (string, error) {
	// Try hash search first
	hash, _ := ComputeMovieHash(videoPath)

	req := SearchRequest{
		Query:     title,
		Year:      year,
		MovieHash: hash,
		Languages: []string{language},
	}

	subtitles, err := c.Search(req)
	if err != nil {
		return "", err
	}

	if len(subtitles) == 0 {
		return "", fmt.Errorf("no subtitles found")
	}

	// Get first result (best match)
	sub := subtitles[0]

	// Generate subtitle path
	videoBase := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	subPath := videoBase + "." + language + ".srt"

	// For now, we can't download without the file_id from the files array
	// This is a simplified implementation - full implementation would parse file_id
	return subPath, fmt.Errorf("subtitle found but download requires file_id (found: %s)", sub.FileName)
}

// Test tests the API connection
func (c *Client) Test() error {
	endpoint := fmt.Sprintf("%s/infos/user", OpenSubtitlesAPIBase)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return nil
}
