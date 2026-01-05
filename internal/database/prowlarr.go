package database

import (
	"database/sql"
	"strings"
	"time"
)

// Prowlarr Config operations

func (d *Database) GetProwlarrConfig() (*ProwlarrConfig, error) {
	var config ProwlarrConfig
	var lastSync sql.NullString
	err := d.db.QueryRow(`
		SELECT id, url, api_key, auto_sync, sync_interval_hours, last_sync, created_at
		FROM prowlarr_config LIMIT 1`).Scan(
		&config.ID, &config.URL, &config.APIKey, &config.AutoSync,
		&config.SyncIntervalHours, &lastSync, &config.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastSync.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", lastSync.String); err == nil {
			config.LastSync = &parsed
		}
	}
	return &config, nil
}

func (d *Database) SaveProwlarrConfig(config *ProwlarrConfig) error {
	// Check if config exists
	existing, _ := d.GetProwlarrConfig()
	if existing != nil {
		_, err := d.db.Exec(`
			UPDATE prowlarr_config SET url = ?, api_key = ?, auto_sync = ?, sync_interval_hours = ?
			WHERE id = ?`,
			config.URL, config.APIKey, config.AutoSync, config.SyncIntervalHours, existing.ID)
		return err
	}
	_, err := d.db.Exec(`
		INSERT INTO prowlarr_config (url, api_key, auto_sync, sync_interval_hours)
		VALUES (?, ?, ?, ?)`,
		config.URL, config.APIKey, config.AutoSync, config.SyncIntervalHours)
	return err
}

func (d *Database) UpdateProwlarrLastSync() error {
	_, err := d.db.Exec("UPDATE prowlarr_config SET last_sync = CURRENT_TIMESTAMP")
	return err
}

// Indexer Tag operations

func (d *Database) UpsertIndexerTag(prowlarrID int, name string) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO indexer_tags (prowlarr_id, name) VALUES (?, ?)
		ON CONFLICT(prowlarr_id) DO UPDATE SET name = excluded.name`,
		prowlarrID, name)
	if err != nil {
		return 0, err
	}
	// Get the ID (either new or existing)
	var id int64
	err = d.db.QueryRow("SELECT id FROM indexer_tags WHERE prowlarr_id = ?", prowlarrID).Scan(&id)
	if err != nil {
		return result.LastInsertId()
	}
	return id, nil
}

func (d *Database) GetIndexerTags() ([]IndexerTag, error) {
	rows, err := d.db.Query(`
		SELECT t.id, t.prowlarr_id, t.name,
			(SELECT COUNT(*) FROM indexer_tag_map WHERE tag_id = t.id) as indexer_count
		FROM indexer_tags t ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []IndexerTag
	for rows.Next() {
		var t IndexerTag
		if err := rows.Scan(&t.ID, &t.ProwlarrID, &t.Name, &t.IndexerCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (d *Database) ClearIndexerTags(indexerID int64) error {
	_, err := d.db.Exec("DELETE FROM indexer_tag_map WHERE indexer_id = ?", indexerID)
	return err
}

func (d *Database) AddIndexerTag(indexerID, tagID int64) error {
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO indexer_tag_map (indexer_id, tag_id) VALUES (?, ?)`,
		indexerID, tagID)
	return err
}

func (d *Database) GetIndexerTagIDs(indexerID int64) ([]int64, error) {
	rows, err := d.db.Query("SELECT tag_id FROM indexer_tag_map WHERE indexer_id = ?", indexerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		tagIDs = append(tagIDs, id)
	}
	return tagIDs, nil
}

// Synced Indexer operations

func (d *Database) UpsertSyncedIndexer(indexer *Indexer) (int64, error) {
	// Check if exists by prowlarr_id
	var existingID int64
	err := d.db.QueryRow("SELECT id FROM indexers WHERE prowlarr_id = ?", indexer.ProwlarrID).Scan(&existingID)

	if err == nil {
		// Update existing
		_, err := d.db.Exec(`
			UPDATE indexers SET
				name = ?, type = ?, url = ?, api_key = ?, priority = ?, enabled = ?,
				synced_from_prowlarr = 1, protocol = ?,
				supports_movies = ?, supports_tv = ?, supports_music = ?,
				supports_books = ?, supports_anime = ?, supports_imdb = ?,
				supports_tmdb = ?, supports_tvdb = ?
			WHERE id = ?`,
			indexer.Name, indexer.Type, indexer.URL, indexer.APIKey, indexer.Priority, indexer.Enabled,
			indexer.Protocol, indexer.SupportsMovies, indexer.SupportsTV, indexer.SupportsMusic,
			indexer.SupportsBooks, indexer.SupportsAnime, indexer.SupportsIMDB,
			indexer.SupportsTMDB, indexer.SupportsTVDB, existingID)
		return existingID, err
	}

	// Insert new
	result, err := d.db.Exec(`
		INSERT INTO indexers (name, type, url, api_key, priority, enabled, prowlarr_id, synced_from_prowlarr, protocol,
			supports_movies, supports_tv, supports_music, supports_books, supports_anime,
			supports_imdb, supports_tmdb, supports_tvdb)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		indexer.Name, indexer.Type, indexer.URL, indexer.APIKey, indexer.Priority, indexer.Enabled,
		indexer.ProwlarrID, indexer.Protocol, indexer.SupportsMovies, indexer.SupportsTV,
		indexer.SupportsMusic, indexer.SupportsBooks, indexer.SupportsAnime,
		indexer.SupportsIMDB, indexer.SupportsTMDB, indexer.SupportsTVDB)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) GetSyncedIndexers() ([]Indexer, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, url, api_key, COALESCE(categories, ''), priority, enabled,
			COALESCE(prowlarr_id, 0), synced_from_prowlarr, COALESCE(protocol, ''),
			COALESCE(supports_movies, 1), COALESCE(supports_tv, 1), COALESCE(supports_music, 0),
			COALESCE(supports_books, 0), COALESCE(supports_anime, 0), COALESCE(supports_imdb, 0),
			COALESCE(supports_tmdb, 0), COALESCE(supports_tvdb, 0)
		FROM indexers WHERE synced_from_prowlarr = 1 ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}

func (d *Database) DisableIndexer(id int64) error {
	_, err := d.db.Exec("UPDATE indexers SET enabled = 0 WHERE id = ?", id)
	return err
}

func (d *Database) GetIndexersByMediaType(mediaType string) ([]Indexer, error) {
	var query string
	switch mediaType {
	case "movie":
		query = "supports_movies = 1"
	case "tv", "show":
		query = "supports_tv = 1"
	case "music":
		query = "supports_music = 1"
	case "book":
		query = "supports_books = 1"
	case "anime":
		query = "supports_anime = 1"
	default:
		query = "1=1"
	}

	rows, err := d.db.Query(`
		SELECT id, name, type, url, api_key, categories, priority, enabled,
			COALESCE(prowlarr_id, 0), COALESCE(synced_from_prowlarr, 0), COALESCE(protocol, ''),
			COALESCE(supports_movies, 1), COALESCE(supports_tv, 1), COALESCE(supports_music, 0),
			COALESCE(supports_books, 0), COALESCE(supports_anime, 0), COALESCE(supports_imdb, 0),
			COALESCE(supports_tmdb, 0), COALESCE(supports_tvdb, 0)
		FROM indexers WHERE enabled = 1 AND ` + query + ` ORDER BY priority DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}

func (d *Database) GetIndexersByTags(tagIDs []int64, mediaType string) ([]Indexer, error) {
	if len(tagIDs) == 0 {
		return d.GetIndexersByMediaType(mediaType)
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tagIDs))
	args := make([]interface{}, len(tagIDs))
	for i, id := range tagIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	var mediaFilter string
	switch mediaType {
	case "movie":
		mediaFilter = "AND i.supports_movies = 1"
	case "tv", "show":
		mediaFilter = "AND i.supports_tv = 1"
	case "music":
		mediaFilter = "AND i.supports_music = 1"
	case "book":
		mediaFilter = "AND i.supports_books = 1"
	case "anime":
		mediaFilter = "AND i.supports_anime = 1"
	}

	query := `
		SELECT DISTINCT i.id, i.name, i.type, i.url, i.api_key, i.categories, i.priority, i.enabled,
			COALESCE(i.prowlarr_id, 0), COALESCE(i.synced_from_prowlarr, 0), COALESCE(i.protocol, ''),
			COALESCE(i.supports_movies, 1), COALESCE(i.supports_tv, 1), COALESCE(i.supports_music, 0),
			COALESCE(i.supports_books, 0), COALESCE(i.supports_anime, 0), COALESCE(i.supports_imdb, 0),
			COALESCE(i.supports_tmdb, 0), COALESCE(i.supports_tvdb, 0)
		FROM indexers i
		INNER JOIN indexer_tag_map tm ON i.id = tm.indexer_id
		WHERE i.enabled = 1 AND tm.tag_id IN (` + strings.Join(placeholders, ",") + `) ` + mediaFilter + `
		ORDER BY i.priority DESC, i.name`

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}

// Library Indexer Tag operations

func (d *Database) GetLibraryIndexerTags(libraryID int64) ([]int64, error) {
	rows, err := d.db.Query("SELECT tag_id FROM library_indexer_tags WHERE library_id = ?", libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		tagIDs = append(tagIDs, id)
	}
	return tagIDs, nil
}

func (d *Database) SetLibraryIndexerTags(libraryID int64, tagIDs []int64) error {
	// Start transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing tags
	_, err = tx.Exec("DELETE FROM library_indexer_tags WHERE library_id = ?", libraryID)
	if err != nil {
		return err
	}

	// Insert new tags
	for _, tagID := range tagIDs {
		_, err = tx.Exec("INSERT INTO library_indexer_tags (library_id, tag_id) VALUES (?, ?)", libraryID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Indexer Category operations

// ClearIndexerCategories removes all category associations for an indexer
func (d *Database) ClearIndexerCategories(indexerID int64) error {
	_, err := d.db.Exec("DELETE FROM indexer_categories WHERE indexer_id = ?", indexerID)
	return err
}

// AddIndexerCategory associates a category ID with an indexer
func (d *Database) AddIndexerCategory(indexerID int64, categoryID int) error {
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO indexer_categories (indexer_id, category_id) VALUES (?, ?)`,
		indexerID, categoryID)
	return err
}

// SetIndexerCategories replaces all categories for an indexer
func (d *Database) SetIndexerCategories(indexerID int64, categoryIDs []int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing
	_, err = tx.Exec("DELETE FROM indexer_categories WHERE indexer_id = ?", indexerID)
	if err != nil {
		return err
	}

	// Insert new
	for _, catID := range categoryIDs {
		_, err = tx.Exec("INSERT INTO indexer_categories (indexer_id, category_id) VALUES (?, ?)", indexerID, catID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetIndexerCategories returns all category IDs for an indexer
func (d *Database) GetIndexerCategories(indexerID int64) ([]int, error) {
	rows, err := d.db.Query("SELECT category_id FROM indexer_categories WHERE indexer_id = ?", indexerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []int
	for rows.Next() {
		var catID int
		if err := rows.Scan(&catID); err != nil {
			return nil, err
		}
		cats = append(cats, catID)
	}
	return cats, nil
}

// CategoryToMediaType maps a Newznab category ID to a media type
// Categories: 2000-2999=movie, 5000-5069=tv, 5070-5079/100000+=anime, 3000-3999=music, 7000-7999=books
func CategoryToMediaType(categoryID int) string {
	switch {
	case categoryID >= 100000:
		return "anime" // Custom anime categories (e.g., Nyaa uses 127720)
	case categoryID >= 5070 && categoryID < 5080:
		return "anime" // Standard anime category range
	case categoryID >= 5000 && categoryID < 5070:
		return "tv"
	case categoryID >= 5080 && categoryID < 6000:
		return "tv" // Other TV subcategories
	case categoryID >= 2000 && categoryID < 3000:
		return "movie"
	case categoryID >= 3000 && categoryID < 4000:
		return "music"
	case categoryID >= 7000 && categoryID < 8000:
		return "book"
	default:
		return ""
	}
}

// GetCategoriesForMediaType returns the standard category IDs for a media type
func GetCategoriesForMediaType(mediaType string) []int {
	switch mediaType {
	case "movie":
		return []int{2000} // Movies (will match 2000-2999)
	case "tv", "show":
		return []int{5000} // TV (will match 5000-5069)
	case "anime":
		return []int{5070, 127720} // Anime categories
	case "music":
		return []int{3000}
	case "book":
		return []int{7000}
	default:
		return nil
	}
}

// IsAnimeOnlyIndexer checks if an indexer only has anime categories
func (d *Database) IsAnimeOnlyIndexer(indexerID int64) (bool, error) {
	// Check if indexer has any non-anime categories
	row := d.db.QueryRow(`
		SELECT COUNT(*) FROM indexer_categories
		WHERE indexer_id = ? AND category_id < 5070 AND category_id >= 2000`, indexerID)
	var nonAnimeCount int
	if err := row.Scan(&nonAnimeCount); err != nil {
		return false, err
	}
	if nonAnimeCount > 0 {
		return false, nil
	}

	// Check if it has any anime categories
	row = d.db.QueryRow(`
		SELECT COUNT(*) FROM indexer_categories
		WHERE indexer_id = ? AND (category_id >= 5070 OR category_id >= 100000)`, indexerID)
	var animeCount int
	if err := row.Scan(&animeCount); err != nil {
		return false, err
	}

	return animeCount > 0, nil
}

// GetIndexersWithCategories returns indexers that have specific category types
func (d *Database) GetIndexersWithCategories(mediaType string) ([]Indexer, error) {
	var categoryFilter string
	switch mediaType {
	case "anime":
		// Anime: 5070-5079 or 100000+
		categoryFilter = "(c.category_id >= 5070 AND c.category_id < 5080) OR c.category_id >= 100000"
	case "tv", "show":
		// TV but NOT anime-only: 5000-5069 or 5080+
		categoryFilter = "(c.category_id >= 5000 AND c.category_id < 5070) OR (c.category_id >= 5080 AND c.category_id < 6000)"
	case "movie":
		categoryFilter = "c.category_id >= 2000 AND c.category_id < 3000"
	case "music":
		categoryFilter = "c.category_id >= 3000 AND c.category_id < 4000"
	case "book":
		categoryFilter = "c.category_id >= 7000 AND c.category_id < 8000"
	default:
		return d.GetEnabledIndexers()
	}

	query := `
		SELECT DISTINCT i.id, i.name, i.type, i.url, i.api_key, COALESCE(i.categories, ''), i.priority, i.enabled,
			COALESCE(i.prowlarr_id, 0), COALESCE(i.synced_from_prowlarr, 0), COALESCE(i.protocol, ''),
			COALESCE(i.supports_movies, 1), COALESCE(i.supports_tv, 1), COALESCE(i.supports_music, 0),
			COALESCE(i.supports_books, 0), COALESCE(i.supports_anime, 0), COALESCE(i.supports_imdb, 0),
			COALESCE(i.supports_tmdb, 0), COALESCE(i.supports_tvdb, 0)
		FROM indexers i
		INNER JOIN indexer_categories c ON i.id = c.indexer_id
		WHERE i.enabled = 1 AND (` + categoryFilter + `)
		ORDER BY i.priority DESC, i.name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}

// GetIndexersExcludingAnimeOnly returns indexers for non-anime searches,
// excluding indexers that ONLY have anime categories (like Nyaa)
func (d *Database) GetIndexersExcludingAnimeOnly(mediaType string) ([]Indexer, error) {
	var supportFilter string
	switch mediaType {
	case "movie":
		supportFilter = "i.supports_movies = 1"
	case "tv", "show":
		supportFilter = "i.supports_tv = 1"
	default:
		supportFilter = "1=1"
	}

	// Get indexers that support the media type AND either:
	// 1. Have no categories stored (legacy/manual indexers - use supports_* flags)
	// 2. Have at least one non-anime category
	query := `
		SELECT DISTINCT i.id, i.name, i.type, i.url, i.api_key, COALESCE(i.categories, ''), i.priority, i.enabled,
			COALESCE(i.prowlarr_id, 0), COALESCE(i.synced_from_prowlarr, 0), COALESCE(i.protocol, ''),
			COALESCE(i.supports_movies, 1), COALESCE(i.supports_tv, 1), COALESCE(i.supports_music, 0),
			COALESCE(i.supports_books, 0), COALESCE(i.supports_anime, 0), COALESCE(i.supports_imdb, 0),
			COALESCE(i.supports_tmdb, 0), COALESCE(i.supports_tvdb, 0)
		FROM indexers i
		WHERE i.enabled = 1 AND ` + supportFilter + `
		AND (
			-- No categories stored (use supports_* flags)
			NOT EXISTS (SELECT 1 FROM indexer_categories WHERE indexer_id = i.id)
			OR
			-- Has at least one non-anime category in the right range
			EXISTS (
				SELECT 1 FROM indexer_categories c
				WHERE c.indexer_id = i.id
				AND ((c.category_id >= 2000 AND c.category_id < 5070) OR (c.category_id >= 5080 AND c.category_id < 100000))
			)
		)
		ORDER BY i.priority DESC, i.name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var i Indexer
		var prowlarrID int64
		var syncedFromProwlarr int
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey,
			&i.Categories, &i.Priority, &i.Enabled,
			&prowlarrID, &syncedFromProwlarr, &i.Protocol,
			&i.SupportsMovies, &i.SupportsTV, &i.SupportsMusic,
			&i.SupportsBooks, &i.SupportsAnime, &i.SupportsIMDB,
			&i.SupportsTMDB, &i.SupportsTVDB); err != nil {
			return nil, err
		}
		if prowlarrID > 0 {
			i.ProwlarrID = &prowlarrID
		}
		i.SyncedFromProwlarr = syncedFromProwlarr == 1
		indexers = append(indexers, i)
	}
	return indexers, nil
}
