package indexer

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NewznabClient implements the Newznab API for usenet indexers
type NewznabClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewNewznabClient creates a new Newznab client
func NewNewznabClient(baseURL, apiKey string) *NewznabClient {
	return &NewznabClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *NewznabClient) GetType() string {
	return "newznab"
}

// newznabRSS represents the RSS response from a Newznab API
type newznabRSS struct {
	XMLName xml.Name       `xml:"rss"`
	Channel newznabChannel `xml:"channel"`
}

type newznabChannel struct {
	Items []newznabItem `xml:"item"`
}

type newznabItem struct {
	Title       string           `xml:"title"`
	GUID        newznabGUID      `xml:"guid"`
	Link        string           `xml:"link"`
	Comments    string           `xml:"comments"`
	PubDate     string           `xml:"pubDate"`
	Category    string           `xml:"category"`
	Enclosure   newznabEnclosure `xml:"enclosure"`
	Attributes  []newznabAttr    `xml:"attr"`
}

type newznabGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type newznabEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type newznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// newznabCaps represents the capabilities response
type newznabCaps struct {
	XMLName    xml.Name          `xml:"caps"`
	Searching  newznabSearching  `xml:"searching"`
	Categories newznabCategories `xml:"categories"`
}

type newznabSearching struct {
	Search      newznabSearchType `xml:"search"`
	TVSearch    newznabSearchType `xml:"tv-search"`
	MovieSearch newznabSearchType `xml:"movie-search"`
	MusicSearch newznabSearchType `xml:"music-search"`
	BookSearch  newznabSearchType `xml:"book-search"`
}

type newznabSearchType struct {
	Available       string `xml:"available,attr"`
	SupportedParams string `xml:"supportedParams,attr"`
}

type newznabCategories struct {
	Categories []newznabCategory `xml:"category"`
}

type newznabCategory struct {
	ID   string            `xml:"id,attr"`
	Name string            `xml:"name,attr"`
	Subs []newznabCategory `xml:"subcat"`
}

// Search performs a search on the Newznab indexer
func (c *NewznabClient) Search(params SearchParams) ([]SearchResult, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	q.Set("t", c.getSearchType(params))
	q.Set("o", "json") // Request JSON format when available, fallback to XML

	if params.Query != "" {
		q.Set("q", params.Query)
	}

	if len(params.Categories) > 0 {
		cats := make([]string, len(params.Categories))
		for i, cat := range params.Categories {
			cats[i] = strconv.Itoa(cat)
		}
		q.Set("cat", strings.Join(cats, ","))
	}

	if params.ImdbID != "" {
		imdb := strings.TrimPrefix(params.ImdbID, "tt")
		q.Set("imdbid", imdb)
	}

	if params.TvdbID != "" {
		q.Set("tvdbid", params.TvdbID)
	}

	if params.TmdbID != "" {
		q.Set("tmdbid", params.TmdbID)
	}

	if params.Season > 0 {
		q.Set("season", strconv.Itoa(params.Season))
	}

	if params.Episode > 0 {
		q.Set("ep", strconv.Itoa(params.Episode))
	}

	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}

	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
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

	// Parse as XML (most Newznab indexers return XML)
	var rss newznabRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResults(rss.Channel.Items), nil
}

func (c *NewznabClient) getSearchType(params SearchParams) string {
	if params.Type != "" {
		switch params.Type {
		case "movie":
			return "movie"
		case "tvsearch":
			return "tvsearch"
		case "music":
			return "music"
		case "book":
			return "book"
		}
	}

	if params.ImdbID != "" {
		return "movie"
	}
	if params.TvdbID != "" || params.Season > 0 {
		return "tvsearch"
	}

	return "search"
}

func (c *NewznabClient) convertResults(items []newznabItem) []SearchResult {
	results := make([]SearchResult, 0, len(items))

	for _, item := range items {
		result := SearchResult{
			Title:       item.Title,
			GUID:        item.GUID.Value,
			Link:        item.Link,
			PublishDate: item.PubDate,
			Category:    item.Category,
			IndexerType: "newznab",
		}

		// Get size from enclosure
		if item.Enclosure.Length > 0 {
			result.Size = item.Enclosure.Length
		}

		// Use enclosure URL if link is empty
		if result.Link == "" && item.Enclosure.URL != "" {
			result.Link = item.Enclosure.URL
		}

		// Parse newznab attributes
		for _, attr := range item.Attributes {
			switch attr.Name {
			case "size":
				if result.Size == 0 {
					result.Size, _ = strconv.ParseInt(attr.Value, 10, 64)
				}
			case "imdb", "imdbid":
				result.ImdbID = attr.Value
			case "tvdbid":
				result.TvdbID = attr.Value
			case "category":
				result.CategoryID = attr.Value
			case "grabs":
				// Number of times grabbed (similar to seeders for sorting)
				grabs, _ := strconv.Atoi(attr.Value)
				result.Seeders = grabs // Use seeders field for sorting
			}
		}

		// Use comments as info URL if available
		if item.Comments != "" {
			result.InfoURL = item.Comments
		}

		results = append(results, result)
	}

	return results
}

// GetCapabilities fetches the indexer capabilities
func (c *NewznabClient) GetCapabilities() (*Capabilities, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	q.Set("t", "caps")
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
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

	var caps newznabCaps
	if err := xml.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertCapabilities(&caps), nil
}

func (c *NewznabClient) convertCapabilities(caps *newznabCaps) *Capabilities {
	result := &Capabilities{
		SearchAvailable:      caps.Searching.Search.Available == "yes",
		MovieSearchAvailable: caps.Searching.MovieSearch.Available == "yes",
		TVSearchAvailable:    caps.Searching.TVSearch.Available == "yes",
		MusicSearchAvailable: caps.Searching.MusicSearch.Available == "yes",
		BookSearchAvailable:  caps.Searching.BookSearch.Available == "yes",
	}

	// Check supported params for ID-based searches
	movieParams := caps.Searching.MovieSearch.SupportedParams
	result.SupportsImdbSearch = strings.Contains(movieParams, "imdbid")
	result.SupportsTmdbSearch = strings.Contains(movieParams, "tmdbid")

	tvParams := caps.Searching.TVSearch.SupportedParams
	result.SupportsTvdbSearch = strings.Contains(tvParams, "tvdbid")

	// Convert categories
	for _, cat := range caps.Categories.Categories {
		id, _ := strconv.Atoi(cat.ID)
		result.Categories = append(result.Categories, Category{
			ID:   id,
			Name: cat.Name,
		})

		for _, sub := range cat.Subs {
			subID, _ := strconv.Atoi(sub.ID)
			result.Categories = append(result.Categories, Category{
				ID:   subID,
				Name: fmt.Sprintf("%s > %s", cat.Name, sub.Name),
			})
		}
	}

	return result
}

// TestConnection tests the connection to the indexer
func (c *NewznabClient) TestConnection() error {
	_, err := c.GetCapabilities()
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

// FetchRSS fetches the RSS feed (latest releases) from the indexer
func (c *NewznabClient) FetchRSS() ([]SearchResult, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	q.Set("t", "search") // Empty search returns latest
	q.Set("limit", "100") // Get last 100 items
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
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

	var rss newznabRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResults(rss.Channel.Items), nil
}
