package indexer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ProwlarrClient implements the Prowlarr API
type ProwlarrClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewProwlarrClient creates a new Prowlarr client
func NewProwlarrClient(baseURL, apiKey string) *ProwlarrClient {
	return &ProwlarrClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Prowlarr can take longer as it queries multiple indexers
		},
	}
}

func (c *ProwlarrClient) GetType() string {
	return "prowlarr"
}

// prowlarrSearchResult represents a search result from Prowlarr
type prowlarrSearchResult struct {
	GUID            string   `json:"guid"`
	Title           string   `json:"title"`
	SortTitle       string   `json:"sortTitle"`
	Size            int64    `json:"size"`
	IndexerID       int      `json:"indexerId"`
	Indexer         string   `json:"indexer"`
	PublishDate     string   `json:"publishDate"`
	DownloadURL     string   `json:"downloadUrl"`
	InfoURL         string   `json:"infoUrl"`
	MagnetURL       string   `json:"magnetUrl"`
	Seeders         int      `json:"seeders"`
	Leechers        int      `json:"leechers"`
	Protocol        string   `json:"protocol"` // torrent or usenet
	Categories      []prowlarrCategory `json:"categories"`
	ImdbID          int      `json:"imdbId"`
	TmdbID          int      `json:"tmdbId"`
	TvdbID          int      `json:"tvdbId"`
}

type prowlarrCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// prowlarrIndexer represents an indexer in Prowlarr
type prowlarrIndexer struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Enable      bool   `json:"enable"`
	SupportsRss bool   `json:"supportsRss"`
	SupportsSearch bool `json:"supportsSearch"`
}

// Search performs a search via Prowlarr
func (c *ProwlarrClient) Search(params SearchParams) ([]SearchResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/search")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()

	if params.Query != "" {
		q.Set("query", params.Query)
	}

	if len(params.Categories) > 0 {
		cats := make([]string, len(params.Categories))
		for i, cat := range params.Categories {
			cats[i] = strconv.Itoa(cat)
		}
		q.Set("categories", strings.Join(cats, ","))
	}

	// Set search type
	searchType := "search"
	if params.Type != "" {
		switch params.Type {
		case "movie":
			searchType = "movie"
		case "tvsearch":
			searchType = "tvsearch"
		case "music":
			searchType = "music"
		case "book":
			searchType = "book"
		}
	} else if params.ImdbID != "" || params.TmdbID != "" {
		searchType = "movie"
	} else if params.TvdbID != "" || params.Season > 0 {
		searchType = "tvsearch"
	}
	q.Set("type", searchType)

	if params.ImdbID != "" {
		// Prowlarr expects numeric IMDB ID
		imdb := strings.TrimPrefix(params.ImdbID, "tt")
		q.Set("imdbId", imdb)
	}

	if params.TmdbID != "" {
		q.Set("tmdbId", params.TmdbID)
	}

	if params.TvdbID != "" {
		q.Set("tvdbId", params.TvdbID)
	}

	if params.Season > 0 {
		q.Set("season", strconv.Itoa(params.Season))
	}

	if params.Episode > 0 {
		q.Set("episode", strconv.Itoa(params.Episode))
	}

	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}

	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResults []prowlarrSearchResult
	if err := json.Unmarshal(body, &searchResults); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResults(searchResults), nil
}

func (c *ProwlarrClient) convertResults(items []prowlarrSearchResult) []SearchResult {
	results := make([]SearchResult, 0, len(items))

	for _, item := range items {
		result := SearchResult{
			Title:       item.Title,
			GUID:        item.GUID,
			Link:        item.DownloadURL,
			MagnetLink:  item.MagnetURL,
			Size:        item.Size,
			Seeders:     item.Seeders,
			Leechers:    item.Leechers,
			PublishDate: item.PublishDate,
			InfoURL:     item.InfoURL,
			IndexerName: item.Indexer,
		}

		// Set indexer type based on protocol
		if item.Protocol == "torrent" {
			result.IndexerType = "torznab"
		} else {
			result.IndexerType = "newznab"
		}

		// Set category
		if len(item.Categories) > 0 {
			result.Category = item.Categories[0].Name
			result.CategoryID = strconv.Itoa(item.Categories[0].ID)
		}

		// Set IDs
		if item.ImdbID > 0 {
			result.ImdbID = fmt.Sprintf("tt%07d", item.ImdbID)
		}
		if item.TvdbID > 0 {
			result.TvdbID = strconv.Itoa(item.TvdbID)
		}

		results = append(results, result)
	}

	return results
}

// GetCapabilities returns the capabilities of Prowlarr
func (c *ProwlarrClient) GetCapabilities() (*Capabilities, error) {
	// Get indexers to determine capabilities
	indexers, err := c.getIndexers()
	if err != nil {
		return nil, err
	}

	caps := &Capabilities{
		SearchAvailable:       len(indexers) > 0,
		MovieSearchAvailable:  true,
		TVSearchAvailable:     true,
		MusicSearchAvailable:  true,
		BookSearchAvailable:   true,
		SupportsImdbSearch:    true,
		SupportsTvdbSearch:    true,
		SupportsTmdbSearch:    true,
	}

	// Get categories from Prowlarr
	categories, err := c.getCategories()
	if err == nil {
		caps.Categories = categories
	}

	return caps, nil
}

func (c *ProwlarrClient) getIndexers() ([]prowlarrIndexer, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/indexer", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var indexers []prowlarrIndexer
	if err := json.Unmarshal(body, &indexers); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return indexers, nil
}

func (c *ProwlarrClient) getCategories() ([]Category, error) {
	// Prowlarr standard categories (Newznab categories)
	return []Category{
		{ID: 1000, Name: "Console"},
		{ID: 2000, Name: "Movies"},
		{ID: 2010, Name: "Movies/Foreign"},
		{ID: 2020, Name: "Movies/Other"},
		{ID: 2030, Name: "Movies/SD"},
		{ID: 2040, Name: "Movies/HD"},
		{ID: 2045, Name: "Movies/UHD"},
		{ID: 2050, Name: "Movies/BluRay"},
		{ID: 2060, Name: "Movies/3D"},
		{ID: 3000, Name: "Audio"},
		{ID: 4000, Name: "PC"},
		{ID: 5000, Name: "TV"},
		{ID: 5010, Name: "TV/Foreign"},
		{ID: 5020, Name: "TV/SD"},
		{ID: 5030, Name: "TV/HD"},
		{ID: 5040, Name: "TV/UHD"},
		{ID: 5045, Name: "TV/Other"},
		{ID: 5050, Name: "TV/Sport"},
		{ID: 5060, Name: "TV/Anime"},
		{ID: 5070, Name: "TV/Documentary"},
		{ID: 6000, Name: "XXX"},
		{ID: 7000, Name: "Books"},
		{ID: 8000, Name: "Other"},
	}, nil
}

// TestConnection tests the connection to Prowlarr
func (c *ProwlarrClient) TestConnection() error {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error %d", resp.StatusCode)
	}

	return nil
}

// GetIndexerList returns the list of configured indexers in Prowlarr
func (c *ProwlarrClient) GetIndexerList() ([]map[string]interface{}, error) {
	indexers, err := c.getIndexers()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(indexers))
	for _, idx := range indexers {
		result = append(result, map[string]interface{}{
			"id":       idx.ID,
			"name":     idx.Name,
			"protocol": idx.Protocol,
			"enabled":  idx.Enable,
		})
	}

	return result, nil
}

// FetchRSS fetches the RSS feed (latest releases) from Prowlarr
func (c *ProwlarrClient) FetchRSS() ([]SearchResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/search")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("type", "search")
	q.Set("limit", "100")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResults []prowlarrSearchResult
	if err := json.Unmarshal(body, &searchResults); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResults(searchResults), nil
}
