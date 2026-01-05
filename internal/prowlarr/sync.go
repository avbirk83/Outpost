package prowlarr

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/outpost/outpost/internal/database"
)

type SyncService struct {
	db     *database.Database
	client *http.Client
}

// Prowlarr API response types
type ProwlarrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type ProwlarrIndexer struct {
	ID           int64               `json:"id"`
	Name         string              `json:"name"`
	Protocol     string              `json:"protocol"` // "torrent" or "usenet"
	Enable       bool                `json:"enable"`
	Priority     int                 `json:"priority"`
	Tags         []int               `json:"tags"`
	Capabilities IndexerCapabilities `json:"capabilities"`
	Fields       []IndexerField      `json:"fields"`
}

type IndexerCapabilities struct {
	SupportsRawSearch bool       `json:"supportsRawSearch"`
	SearchParams      []string   `json:"searchParams"`
	TvSearchParams    []string   `json:"tvSearchParams"`
	MovieSearchParams []string   `json:"movieSearchParams"`
	MusicSearchParams []string   `json:"musicSearchParams"`
	BookSearchParams  []string   `json:"bookSearchParams"`
	Categories        []Category `json:"categories"`
}

type Category struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	SubCategories []Category `json:"subCategories"`
}

type IndexerField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func NewSyncService(db *database.Database) *SyncService {
	return &SyncService{
		db:     db,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SyncService) TestConnection(url, apiKey string) error {
	req, err := http.NewRequest("GET", strings.TrimSuffix(url, "/")+"/api/v1/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (s *SyncService) FetchTags(url, apiKey string) ([]ProwlarrTag, error) {
	req, err := http.NewRequest("GET", strings.TrimSuffix(url, "/")+"/api/v1/tag", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tags: %d", resp.StatusCode)
	}

	var tags []ProwlarrTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *SyncService) FetchIndexers(url, apiKey string) ([]ProwlarrIndexer, error) {
	req, err := http.NewRequest("GET", strings.TrimSuffix(url, "/")+"/api/v1/indexer", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch indexers: %d", resp.StatusCode)
	}

	var indexers []ProwlarrIndexer
	if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
		return nil, err
	}
	return indexers, nil
}

func (s *SyncService) SyncAll() (int, error) {
	// Get Prowlarr config
	config, err := s.db.GetProwlarrConfig()
	if err != nil || config == nil {
		return 0, fmt.Errorf("prowlarr not configured")
	}

	// 1. Sync tags
	tags, err := s.FetchTags(config.URL, config.APIKey)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch tags: %w", err)
	}

	tagMap := make(map[int]int64) // prowlarr_id -> outpost_id
	for _, t := range tags {
		outpostID, err := s.db.UpsertIndexerTag(t.ID, t.Label)
		if err != nil {
			log.Printf("Failed to upsert tag %s: %v", t.Label, err)
			continue
		}
		tagMap[t.ID] = outpostID
	}

	log.Printf("Synced %d tags from Prowlarr", len(tags))

	// 2. Sync indexers
	indexers, err := s.FetchIndexers(config.URL, config.APIKey)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch indexers: %w", err)
	}

	syncedCount := 0
	for _, pi := range indexers {
		// Parse capabilities
		caps := parseCapabilities(pi)

		// For synced indexers, use Prowlarr as proxy (type "prowlarr")
		// Prowlarr handles all the actual indexer connections
		indexer := &database.Indexer{
			Name:               pi.Name,
			Type:               "prowlarr", // Use prowlarr client to proxy through Prowlarr
			URL:                config.URL,  // Prowlarr's URL
			APIKey:             config.APIKey, // Prowlarr's API key
			Enabled:            pi.Enable,
			Priority:           pi.Priority,
			ProwlarrID:         &pi.ID,
			SyncedFromProwlarr: true,
			Protocol:           pi.Protocol,
			SupportsMovies:     caps.Movies,
			SupportsTV:         caps.TV,
			SupportsMusic:      caps.Music,
			SupportsBooks:      caps.Books,
			SupportsAnime:      caps.Anime,
			SupportsIMDB:       caps.IMDB,
			SupportsTMDB:       caps.TMDB,
			SupportsTVDB:       caps.TVDB,
		}

		// Upsert indexer
		indexerID, err := s.db.UpsertSyncedIndexer(indexer)
		if err != nil {
			log.Printf("Failed to upsert indexer %s: %v", pi.Name, err)
			continue
		}

		// Sync tag associations
		s.db.ClearIndexerTags(indexerID)
		for _, prowlarrTagID := range pi.Tags {
			if outpostTagID, ok := tagMap[prowlarrTagID]; ok {
				s.db.AddIndexerTag(indexerID, outpostTagID)
			}
		}

		// Sync category IDs for filtering
		categoryIDs := collectCategoryIDs(pi.Capabilities.Categories)
		if err := s.db.SetIndexerCategories(indexerID, categoryIDs); err != nil {
			log.Printf("Failed to set categories for indexer %s: %v", pi.Name, err)
		}

		syncedCount++
	}

	log.Printf("Synced %d indexers from Prowlarr", syncedCount)

	// 3. Mark stale indexers (in Outpost but not in Prowlarr)
	s.markStaleIndexers(indexers)

	// 4. Update last sync time
	s.db.UpdateProwlarrLastSync()

	return syncedCount, nil
}

type parsedCaps struct {
	Movies, TV, Music, Books, Anime bool
	IMDB, TMDB, TVDB                bool
}

func parseCapabilities(pi ProwlarrIndexer) parsedCaps {
	caps := pi.Capabilities
	cats := caps.Categories

	return parsedCaps{
		Movies: len(caps.MovieSearchParams) > 0 || hasCategory(cats, 2000),
		TV:     len(caps.TvSearchParams) > 0 || hasCategory(cats, 5000),
		Music:  len(caps.MusicSearchParams) > 0 || hasCategory(cats, 3000),
		Books:  len(caps.BookSearchParams) > 0 || hasCategory(cats, 7000),
		Anime:  hasCategory(cats, 5070) || hasCategory(cats, 127720),
		IMDB:   containsParam(caps.MovieSearchParams, "imdbId") || containsParam(caps.TvSearchParams, "imdbId"),
		TMDB:   containsParam(caps.MovieSearchParams, "tmdbId"),
		TVDB:   containsParam(caps.TvSearchParams, "tvdbId"),
	}
}

func hasCategory(categories []Category, targetID int) bool {
	for _, cat := range categories {
		if cat.ID == targetID || cat.ID/1000 == targetID/1000 {
			return true
		}
		if hasCategory(cat.SubCategories, targetID) {
			return true
		}
	}
	return false
}

// collectCategoryIDs extracts all category IDs from the Prowlarr category tree
func collectCategoryIDs(categories []Category) []int {
	var ids []int
	for _, cat := range categories {
		ids = append(ids, cat.ID)
		// Also collect subcategories
		ids = append(ids, collectCategoryIDs(cat.SubCategories)...)
	}
	return ids
}

func containsParam(params []string, target string) bool {
	for _, p := range params {
		if strings.EqualFold(p, target) {
			return true
		}
	}
	return false
}

func extractCredentials(fields []IndexerField) (url, apiKey string) {
	for _, f := range fields {
		switch f.Name {
		case "baseUrl", "baseSettings.baseUrl":
			if v, ok := f.Value.(string); ok {
				url = v
			}
		case "apiKey", "baseSettings.apiKey":
			if v, ok := f.Value.(string); ok {
				apiKey = v
			}
		}
	}
	return
}

func (s *SyncService) markStaleIndexers(currentIndexers []ProwlarrIndexer) {
	prowlarrIDs := make(map[int64]bool)
	for _, pi := range currentIndexers {
		prowlarrIDs[pi.ID] = true
	}

	syncedIndexers, err := s.db.GetSyncedIndexers()
	if err != nil {
		log.Printf("Failed to get synced indexers: %v", err)
		return
	}

	for _, idx := range syncedIndexers {
		if idx.ProwlarrID != nil && !prowlarrIDs[*idx.ProwlarrID] {
			// Disable rather than delete to preserve history
			log.Printf("Disabling stale indexer: %s", idx.Name)
			s.db.DisableIndexer(idx.ID)
		}
	}
}
