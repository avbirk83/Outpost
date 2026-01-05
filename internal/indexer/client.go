package indexer

import (
	"fmt"
	"sort"
	"sync"
)

// SearchResult represents a search result from an indexer
type SearchResult struct {
	IndexerID   int64  `json:"indexerId"`
	IndexerName string `json:"indexerName"`
	IndexerType string `json:"indexerType"` // torznab, newznab
	Title       string `json:"title"`
	GUID        string `json:"guid"`
	Link        string `json:"link"`        // Download link
	MagnetLink  string `json:"magnetLink"`  // Magnet link for torrents
	Size        int64  `json:"size"`        // Size in bytes
	Seeders     int    `json:"seeders"`     // For torrents
	Leechers    int    `json:"leechers"`    // For torrents
	PublishDate string `json:"publishDate"` // ISO 8601 date
	Category    string `json:"category"`
	CategoryID  string `json:"categoryId"`
	ImdbID      string `json:"imdbId,omitempty"`
	TvdbID      string `json:"tvdbId,omitempty"`
	InfoURL     string `json:"infoUrl,omitempty"`
}

// ScoredSearchResult extends SearchResult with quality scoring info
type ScoredSearchResult struct {
	SearchResult
	Quality          string            `json:"quality,omitempty"`
	Resolution       string            `json:"resolution,omitempty"`
	Source           string            `json:"source,omitempty"`
	Codec            string            `json:"codec,omitempty"`
	AudioCodec       string            `json:"audioCodec,omitempty"`
	AudioFeature     string            `json:"audioFeature,omitempty"`
	HDR              []string          `json:"hdr,omitempty"`
	ReleaseGroup     string            `json:"releaseGroup,omitempty"`
	Proper           bool              `json:"proper,omitempty"`
	Repack           bool              `json:"repack,omitempty"`
	BaseScore        int               `json:"baseScore"`
	CustomFormatHits []CustomFormatHit `json:"customFormatHits,omitempty"`
	TotalScore       int               `json:"totalScore"`
	Rejected         bool              `json:"rejected"`
	RejectionReason  string            `json:"rejectionReason,omitempty"`
}

// CustomFormatHit represents a matched custom format
type CustomFormatHit struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// SearchParams contains the search parameters
type SearchParams struct {
	Query      string // Free-text search query
	Type       string // movie, tvsearch, search, music, book
	Categories []int  // Category IDs to search
	ImdbID     string // IMDB ID for movie/tv searches
	TvdbID     string // TVDB ID for tv searches
	TmdbID     string // TMDB ID for searches
	Season     int    // Season number for tv searches
	Episode    int    // Episode number for tv searches
	Limit      int    // Max results per indexer
	Offset     int    // Offset for pagination
}

// Client interface for indexers
type Client interface {
	Search(params SearchParams) ([]SearchResult, error)
	FetchRSS() ([]SearchResult, error)
	GetCapabilities() (*Capabilities, error)
	TestConnection() error
	GetType() string
}

// Capabilities represents what an indexer can do
type Capabilities struct {
	SearchAvailable       bool     `json:"searchAvailable"`
	MovieSearchAvailable  bool     `json:"movieSearchAvailable"`
	TVSearchAvailable     bool     `json:"tvSearchAvailable"`
	MusicSearchAvailable  bool     `json:"musicSearchAvailable"`
	BookSearchAvailable   bool     `json:"bookSearchAvailable"`
	Categories            []Category `json:"categories"`
	SupportsImdbSearch    bool     `json:"supportsImdbSearch"`
	SupportsTvdbSearch    bool     `json:"supportsTvdbSearch"`
	SupportsTmdbSearch    bool     `json:"supportsTmdbSearch"`
}

// Category represents an indexer category
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IndexerConfig contains the configuration for an indexer
type IndexerConfig struct {
	ID         int64
	Name       string
	Type       string // torznab, newznab, prowlarr
	URL        string
	APIKey     string
	Categories string
	Priority   int
	Enabled    bool
}

// Manager handles multiple indexers
type Manager struct {
	indexers map[int64]Client
	configs  map[int64]*IndexerConfig
	mu       sync.RWMutex
}

// NewManager creates a new indexer manager
func NewManager() *Manager {
	return &Manager{
		indexers: make(map[int64]Client),
		configs:  make(map[int64]*IndexerConfig),
	}
}

// AddIndexer adds an indexer to the manager
func (m *Manager) AddIndexer(config *IndexerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var client Client
	var err error

	switch config.Type {
	case "torznab":
		client = NewTorznabClient(config.URL, config.APIKey)
	case "newznab":
		client = NewNewznabClient(config.URL, config.APIKey)
	case "prowlarr":
		client = NewProwlarrClient(config.URL, config.APIKey)
	default:
		return fmt.Errorf("unknown indexer type: %s", config.Type)
	}

	// Test connection
	if err = client.TestConnection(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	m.indexers[config.ID] = client
	m.configs[config.ID] = config
	return nil
}

// RemoveIndexer removes an indexer from the manager
func (m *Manager) RemoveIndexer(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.indexers, id)
	delete(m.configs, id)
}

// GetIndexer returns a specific indexer client
func (m *Manager) GetIndexer(id int64) (Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.indexers[id]
	return client, ok
}

// Search searches all enabled indexers and aggregates results
func (m *Manager) Search(params SearchParams) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.indexers) == 0 {
		return nil, nil
	}

	var wg sync.WaitGroup
	resultsChan := make(chan []SearchResult, len(m.indexers))
	errorsChan := make(chan error, len(m.indexers))

	for id, client := range m.indexers {
		config := m.configs[id]
		if !config.Enabled {
			continue
		}

		wg.Add(1)
		go func(id int64, c Client, cfg *IndexerConfig) {
			defer wg.Done()

			results, err := c.Search(params)
			if err != nil {
				errorsChan <- fmt.Errorf("indexer %s: %w", cfg.Name, err)
				return
			}

			// Add indexer info to results
			for i := range results {
				results[i].IndexerID = id
				results[i].IndexerName = cfg.Name
				results[i].IndexerType = cfg.Type
			}

			resultsChan <- results
		}(id, client, config)
	}

	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	var allResults []SearchResult
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	// Sort by seeders (descending) for torrents, or by date for usenet
	sort.Slice(allResults, func(i, j int) bool {
		// Torrents: sort by seeders
		if allResults[i].Seeders != allResults[j].Seeders {
			return allResults[i].Seeders > allResults[j].Seeders
		}
		// Fallback: sort by size
		return allResults[i].Size > allResults[j].Size
	})

	return allResults, nil
}



// SearchWithIndexerIDs searches only the specified indexers
func (m *Manager) SearchWithIndexerIDs(params SearchParams, indexerIDs []int64) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.indexers) == 0 || len(indexerIDs) == 0 {
		return nil, nil
	}

	// Create a set of allowed IDs for quick lookup
	allowedIDs := make(map[int64]bool)
	for _, id := range indexerIDs {
		allowedIDs[id] = true
	}

	var wg sync.WaitGroup
	resultsChan := make(chan []SearchResult, len(indexerIDs))
	errorsChan := make(chan error, len(indexerIDs))

	for id, client := range m.indexers {
		// Skip if not in allowed list
		if !allowedIDs[id] {
			continue
		}

		config := m.configs[id]
		if !config.Enabled {
			continue
		}

		wg.Add(1)
		go func(id int64, c Client, cfg *IndexerConfig) {
			defer wg.Done()

			results, err := c.Search(params)
			if err != nil {
				errorsChan <- fmt.Errorf("indexer %s: %w", cfg.Name, err)
				return
			}

			// Add indexer info to results
			for i := range results {
				results[i].IndexerID = id
				results[i].IndexerName = cfg.Name
				results[i].IndexerType = cfg.Type
			}

			resultsChan <- results
		}(id, client, config)
	}

	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	var allResults []SearchResult
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	// Sort by seeders (descending)
	sort.Slice(allResults, func(i, j int) bool {
		if allResults[i].Seeders != allResults[j].Seeders {
			return allResults[i].Seeders > allResults[j].Seeders
		}
		return allResults[i].Size > allResults[j].Size
	})

	return allResults, nil
}

// TestIndexer tests a specific indexer connection
func (m *Manager) TestIndexer(id int64) error {
	m.mu.RLock()
	client, ok := m.indexers[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("indexer not found")
	}

	return client.TestConnection()
}

// GetCapabilities returns the capabilities of a specific indexer
func (m *Manager) GetCapabilities(id int64) (*Capabilities, error) {
	m.mu.RLock()
	client, ok := m.indexers[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("indexer not found")
	}

	return client.GetCapabilities()
}

// Clear removes all indexers
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.indexers = make(map[int64]Client)
	m.configs = make(map[int64]*IndexerConfig)
}

// Count returns the number of loaded indexers
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.indexers)
}

// FetchRSS fetches RSS feed from a specific indexer
func (m *Manager) FetchRSS(id int64) ([]SearchResult, error) {
	m.mu.RLock()
	client, ok := m.indexers[id]
	config := m.configs[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("indexer not found")
	}

	results, err := client.FetchRSS()
	if err != nil {
		return nil, err
	}

	// Add indexer info to results
	for i := range results {
		results[i].IndexerID = id
		results[i].IndexerName = config.Name
		results[i].IndexerType = config.Type
	}

	return results, nil
}
