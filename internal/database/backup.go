package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// BackupVersion is the current backup format version
const BackupVersion = "1.0"

// Backup represents a complete backup of the application settings and configuration
type Backup struct {
	Version    string    `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	AppVersion string    `json:"appVersion"`

	Settings          map[string]string      `json:"settings"`
	Users             []BackupUser           `json:"users"`
	Libraries         []Library              `json:"libraries"`
	DownloadClients   []DownloadClient       `json:"downloadClients"`
	ProwlarrConfig    *ProwlarrConfig        `json:"prowlarrConfig,omitempty"`
	Indexers          []Indexer              `json:"indexers"`
	IndexerTags       []IndexerTag           `json:"indexerTags"`
	QualityProfiles   []QualityProfile       `json:"qualityProfiles"`
	QualityPresets    []QualityPreset        `json:"qualityPresets"`
	CustomFormats     []CustomFormat         `json:"customFormats"`
	Collections       []Collection           `json:"collections"`
	CollectionItems   []CollectionItem       `json:"collectionItems"`
	SkipSegments      []BackupSkipSegment    `json:"skipSegments"`
	NamingTemplates   []NamingTemplate       `json:"namingTemplates"`
	BlockedGroups     []BlockedGroup         `json:"blockedGroups"`
	TrustedGroups     []TrustedGroup         `json:"trustedGroups"`
	DelayProfiles     []DelayProfile         `json:"delayProfiles"`
	ReleaseFilters    []ReleaseFilter        `json:"releaseFilters"`
	ScheduledTasks    []ScheduledTask        `json:"scheduledTasks"`
}

// BackupUser is a User without password hash for backup
type BackupUser struct {
	ID                 int64   `json:"id"`
	Username           string  `json:"username"`
	Role               string  `json:"role"`
	ContentRatingLimit *string `json:"contentRatingLimit,omitempty"`
	RequirePin         bool    `json:"requirePin"`
}

// BackupSkipSegment stores skip segments with their show association
type BackupSkipSegment struct {
	ShowID      int64        `json:"showId"`
	ShowTmdbID  *int64       `json:"showTmdbId,omitempty"`
	ShowTitle   string       `json:"showTitle"`
	IntroStart  *float64     `json:"introStart,omitempty"`
	IntroEnd    *float64     `json:"introEnd,omitempty"`
	CredStart   *float64     `json:"creditsStart,omitempty"`
	CredEnd     *float64     `json:"creditsEnd,omitempty"`
}

// RestoreResult contains the result of a restore operation
type RestoreResult struct {
	Success  bool              `json:"success"`
	Restored map[string]int    `json:"restored"`
	Warnings []string          `json:"warnings"`
	Errors   []string          `json:"errors,omitempty"`
}

// CreateBackup exports all settings and configuration to a Backup structure
func (d *Database) CreateBackup(appVersion string) (*Backup, error) {
	backup := &Backup{
		Version:    BackupVersion,
		CreatedAt:  time.Now(),
		AppVersion: appVersion,
	}

	var err error

	// Export settings
	backup.Settings, err = d.GetAllSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to export settings: %w", err)
	}

	// Export users (without password hashes)
	users, err := d.getAllUsersForBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to export users: %w", err)
	}
	backup.Users = users

	// Export libraries
	backup.Libraries, err = d.GetLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to export libraries: %w", err)
	}

	// Export download clients
	backup.DownloadClients, err = d.GetDownloadClients()
	if err != nil {
		return nil, fmt.Errorf("failed to export download clients: %w", err)
	}

	// Export Prowlarr config
	backup.ProwlarrConfig, err = d.GetProwlarrConfig()
	if err != nil {
		// Not a fatal error - config might not exist
		backup.ProwlarrConfig = nil
	}

	// Export indexers (only manual ones, not synced from Prowlarr)
	backup.Indexers, err = d.getManualIndexers()
	if err != nil {
		return nil, fmt.Errorf("failed to export indexers: %w", err)
	}

	// Export indexer tags
	backup.IndexerTags, err = d.getIndexerTagsForBackup()
	if err != nil {
		// Not fatal
		backup.IndexerTags = []IndexerTag{}
	}

	// Export quality profiles
	backup.QualityProfiles, err = d.getQualityProfilesForBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to export quality profiles: %w", err)
	}

	// Export quality presets
	backup.QualityPresets, err = d.GetQualityPresets()
	if err != nil {
		return nil, fmt.Errorf("failed to export quality presets: %w", err)
	}

	// Export custom formats
	backup.CustomFormats, err = d.GetCustomFormats()
	if err != nil {
		return nil, fmt.Errorf("failed to export custom formats: %w", err)
	}

	// Export collections
	backup.Collections, err = d.getCollectionsForBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to export collections: %w", err)
	}

	// Export collection items
	backup.CollectionItems, err = d.getCollectionItemsForBackup()
	if err != nil {
		return nil, fmt.Errorf("failed to export collection items: %w", err)
	}

	// Export skip segments
	backup.SkipSegments, err = d.getSkipSegmentsForBackup()
	if err != nil {
		// Not fatal
		backup.SkipSegments = []BackupSkipSegment{}
	}

	// Export naming templates
	backup.NamingTemplates, err = d.getNamingTemplatesForBackup()
	if err != nil {
		backup.NamingTemplates = []NamingTemplate{}
	}

	// Export release filters
	backup.ReleaseFilters, err = d.getReleaseFiltersForBackup()
	if err != nil {
		backup.ReleaseFilters = []ReleaseFilter{}
	}

	// Export delay profiles
	backup.DelayProfiles, err = d.getDelayProfilesForBackup()
	if err != nil {
		backup.DelayProfiles = []DelayProfile{}
	}

	// Export blocked groups
	backup.BlockedGroups, err = d.getBlockedGroupsForBackup()
	if err != nil {
		backup.BlockedGroups = []BlockedGroup{}
	}

	// Export trusted groups
	backup.TrustedGroups, err = d.getTrustedGroupsForBackup()
	if err != nil {
		backup.TrustedGroups = []TrustedGroup{}
	}

	// Export scheduled tasks (just the configuration, not history)
	backup.ScheduledTasks, err = d.getScheduledTasksForBackup()
	if err != nil {
		backup.ScheduledTasks = []ScheduledTask{}
	}

	return backup, nil
}

// Helper functions for backup export

func (d *Database) getAllUsersForBackup() ([]BackupUser, error) {
	rows, err := d.db.Query(`SELECT id, username, role, content_rating_limit, require_pin FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []BackupUser
	for rows.Next() {
		var u BackupUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.ContentRatingLimit, &u.RequirePin); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) getManualIndexers() ([]Indexer, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, url, api_key, categories, priority, enabled,
			   prowlarr_id, synced_from_prowlarr, protocol,
			   supports_movies, supports_tv, supports_music, supports_books, supports_anime,
			   supports_imdb, supports_tmdb, supports_tvdb
		FROM indexers
		WHERE synced_from_prowlarr = 0 OR synced_from_prowlarr IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexers []Indexer
	for rows.Next() {
		var idx Indexer
		var apiKey, categories, protocol sql.NullString
		var prowlarrID sql.NullInt64
		if err := rows.Scan(
			&idx.ID, &idx.Name, &idx.Type, &idx.URL, &apiKey, &categories, &idx.Priority, &idx.Enabled,
			&prowlarrID, &idx.SyncedFromProwlarr, &protocol,
			&idx.SupportsMovies, &idx.SupportsTV, &idx.SupportsMusic, &idx.SupportsBooks, &idx.SupportsAnime,
			&idx.SupportsIMDB, &idx.SupportsTMDB, &idx.SupportsTVDB,
		); err != nil {
			return nil, err
		}
		if apiKey.Valid {
			idx.APIKey = apiKey.String
		}
		if categories.Valid {
			idx.Categories = categories.String
		}
		if protocol.Valid {
			idx.Protocol = protocol.String
		}
		if prowlarrID.Valid {
			idx.ProwlarrID = &prowlarrID.Int64
		}
		indexers = append(indexers, idx)
	}
	return indexers, nil
}

func (d *Database) getIndexerTagsForBackup() ([]IndexerTag, error) {
	rows, err := d.db.Query(`SELECT id, prowlarr_id, name FROM indexer_tags`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []IndexerTag
	for rows.Next() {
		var t IndexerTag
		if err := rows.Scan(&t.ID, &t.ProwlarrID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (d *Database) getQualityProfilesForBackup() ([]QualityProfile, error) {
	rows, err := d.db.Query(`
		SELECT id, name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores
		FROM quality_profiles
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []QualityProfile
	for rows.Next() {
		var p QualityProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.UpgradeAllowed, &p.UpgradeUntilScore, &p.MinFormatScore, &p.CutoffFormatScore, &p.Qualities, &p.CustomFormatScores); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) getCollectionsForBackup() ([]Collection, error) {
	rows, err := d.db.Query(`
		SELECT id, name, description, tmdb_collection_id, poster_path, backdrop_path, is_auto, sort_order, created_at, updated_at
		FROM collections
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.TmdbCollectionID, &c.PosterPath, &c.BackdropPath, &c.IsAuto, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, nil
}

func (d *Database) getCollectionItemsForBackup() ([]CollectionItem, error) {
	rows, err := d.db.Query(`
		SELECT id, collection_id, media_type, media_id, tmdb_id, title, year, poster_path, sort_order, added_at
		FROM collection_items
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CollectionItem
	for rows.Next() {
		var item CollectionItem
		if err := rows.Scan(&item.ID, &item.CollectionID, &item.MediaType, &item.MediaID, &item.TmdbID, &item.Title, &item.Year, &item.PosterPath, &item.SortOrder, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) getSkipSegmentsForBackup() ([]BackupSkipSegment, error) {
	rows, err := d.db.Query(`
		SELECT s.show_id, sh.tmdb_id, sh.title, s.intro_start, s.intro_end, s.credits_start, s.credits_end
		FROM skip_segments s
		LEFT JOIN shows sh ON s.show_id = sh.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []BackupSkipSegment
	for rows.Next() {
		var seg BackupSkipSegment
		if err := rows.Scan(&seg.ShowID, &seg.ShowTmdbID, &seg.ShowTitle, &seg.IntroStart, &seg.IntroEnd, &seg.CredStart, &seg.CredEnd); err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}
	return segments, nil
}

func (d *Database) getNamingTemplatesForBackup() ([]NamingTemplate, error) {
	rows, err := d.db.Query(`SELECT id, type, folder_template, file_template, is_default FROM naming_templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []NamingTemplate
	for rows.Next() {
		var t NamingTemplate
		if err := rows.Scan(&t.ID, &t.Type, &t.FolderTemplate, &t.FileTemplate, &t.IsDefault); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (d *Database) getReleaseFiltersForBackup() ([]ReleaseFilter, error) {
	rows, err := d.db.Query(`SELECT id, preset_id, filter_type, value, is_regex, created_at FROM release_filters`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []ReleaseFilter
	for rows.Next() {
		var f ReleaseFilter
		if err := rows.Scan(&f.ID, &f.PresetID, &f.FilterType, &f.Value, &f.IsRegex, &f.CreatedAt); err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (d *Database) getDelayProfilesForBackup() ([]DelayProfile, error) {
	rows, err := d.db.Query(`SELECT id, name, enabled, delay_minutes, bypass_if_resolution, bypass_if_source, bypass_if_score_above, library_id, created_at FROM delay_profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []DelayProfile
	for rows.Next() {
		var p DelayProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, &p.DelayMinutes, &p.BypassIfResolution, &p.BypassIfSource, &p.BypassIfScoreAbove, &p.LibraryID, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) getBlockedGroupsForBackup() ([]BlockedGroup, error) {
	rows, err := d.db.Query(`SELECT id, name, reason, auto_blocked, failure_count, created_at FROM blocked_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []BlockedGroup
	for rows.Next() {
		var g BlockedGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Reason, &g.AutoBlocked, &g.FailureCount, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *Database) getTrustedGroupsForBackup() ([]TrustedGroup, error) {
	rows, err := d.db.Query(`SELECT id, name, category, created_at FROM trusted_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []TrustedGroup
	for rows.Next() {
		var g TrustedGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Category, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (d *Database) getScheduledTasksForBackup() ([]ScheduledTask, error) {
	rows, err := d.db.Query(`SELECT id, name, description, task_type, enabled, interval_minutes FROM scheduled_tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		var t ScheduledTask
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.TaskType, &t.Enabled, &t.IntervalMinutes); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ValidateBackup checks if a backup is valid and compatible
func ValidateBackup(data []byte) (*Backup, error) {
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("invalid backup format: %w", err)
	}

	if backup.Version == "" {
		return nil, fmt.Errorf("missing backup version")
	}

	// Check version compatibility
	if backup.Version != BackupVersion {
		return nil, fmt.Errorf("backup version %s is not compatible with current version %s", backup.Version, BackupVersion)
	}

	return &backup, nil
}

// RestoreBackup restores settings and configuration from a backup
// mode can be "replace" (clear existing data) or "merge" (keep existing, add new)
func (d *Database) RestoreBackup(backup *Backup, mode string) (*RestoreResult, error) {
	result := &RestoreResult{
		Success:  true,
		Restored: make(map[string]int),
		Warnings: []string{},
	}

	tx, err := d.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// If replace mode, clear existing data
	if mode == "replace" {
		if err := d.clearDataForRestore(tx); err != nil {
			return nil, fmt.Errorf("failed to clear existing data: %w", err)
		}
	}

	// Restore settings
	count, err := d.restoreSettings(tx, backup.Settings, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Settings: %v", err))
	} else {
		result.Restored["settings"] = count
	}

	// Restore users
	count, warns, err := d.restoreUsers(tx, backup.Users, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Users: %v", err))
	} else {
		result.Restored["users"] = count
		result.Warnings = append(result.Warnings, warns...)
	}

	// Restore libraries
	count, err = d.restoreLibraries(tx, backup.Libraries, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Libraries: %v", err))
	} else {
		result.Restored["libraries"] = count
	}

	// Restore download clients
	count, err = d.restoreDownloadClients(tx, backup.DownloadClients, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Download clients: %v", err))
	} else {
		result.Restored["downloadClients"] = count
	}

	// Restore Prowlarr config
	if backup.ProwlarrConfig != nil {
		if err := d.restoreProwlarrConfig(tx, backup.ProwlarrConfig, mode); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Prowlarr config: %v", err))
		} else {
			result.Restored["prowlarrConfig"] = 1
		}
	}

	// Restore indexers
	count, err = d.restoreIndexers(tx, backup.Indexers, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Indexers: %v", err))
	} else {
		result.Restored["indexers"] = count
	}

	// Restore quality profiles
	count, err = d.restoreQualityProfiles(tx, backup.QualityProfiles, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Quality profiles: %v", err))
	} else {
		result.Restored["qualityProfiles"] = count
	}

	// Restore quality presets
	count, err = d.restoreQualityPresets(tx, backup.QualityPresets, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Quality presets: %v", err))
	} else {
		result.Restored["qualityPresets"] = count
	}

	// Restore custom formats
	count, err = d.restoreCustomFormats(tx, backup.CustomFormats, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Custom formats: %v", err))
	} else {
		result.Restored["customFormats"] = count
	}

	// Restore collections
	count, err = d.restoreCollections(tx, backup.Collections, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Collections: %v", err))
	} else {
		result.Restored["collections"] = count
	}

	// Restore collection items
	count, err = d.restoreCollectionItems(tx, backup.CollectionItems, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Collection items: %v", err))
	} else {
		result.Restored["collectionItems"] = count
	}

	// Restore naming templates
	count, err = d.restoreNamingTemplates(tx, backup.NamingTemplates, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Naming templates: %v", err))
	} else {
		result.Restored["namingTemplates"] = count
	}

	// Restore release filters
	count, err = d.restoreReleaseFilters(tx, backup.ReleaseFilters, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Release filters: %v", err))
	} else {
		result.Restored["releaseFilters"] = count
	}

	// Restore delay profiles
	count, err = d.restoreDelayProfiles(tx, backup.DelayProfiles, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Delay profiles: %v", err))
	} else {
		result.Restored["delayProfiles"] = count
	}

	// Restore blocked groups
	count, err = d.restoreBlockedGroups(tx, backup.BlockedGroups, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Blocked groups: %v", err))
	} else {
		result.Restored["blockedGroups"] = count
	}

	// Restore trusted groups
	count, err = d.restoreTrustedGroups(tx, backup.TrustedGroups, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Trusted groups: %v", err))
	} else {
		result.Restored["trustedGroups"] = count
	}

	// Restore scheduled task settings
	count, err = d.restoreScheduledTasks(tx, backup.ScheduledTasks, mode)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Scheduled tasks: %v", err))
	} else {
		result.Restored["scheduledTasks"] = count
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Add warning about password reset
	if result.Restored["users"] > 0 {
		result.Warnings = append(result.Warnings, "Users were restored without passwords. Users must reset their passwords.")
	}

	return result, nil
}

// clearDataForRestore clears tables that will be restored
func (d *Database) clearDataForRestore(tx *sql.Tx) error {
	tables := []string{
		"settings",
		"download_clients",
		"prowlarr_config",
		"indexers",
		"indexer_tags",
		"quality_profiles",
		"quality_presets",
		"custom_formats",
		"collections",
		"collection_items",
		"naming_templates",
		"release_filters",
		"delay_profiles",
		"blocked_groups",
		"trusted_groups",
	}

	for _, table := range tables {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			// Ignore errors for tables that might not exist
			continue
		}
	}

	// Don't clear users or libraries in replace mode - too dangerous
	// Just update existing ones

	return nil
}

// Restore helper functions

func (d *Database) restoreSettings(tx *sql.Tx, settings map[string]string, mode string) (int, error) {
	count := 0
	for key, value := range settings {
		_, err := tx.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreUsers(tx *sql.Tx, users []BackupUser, mode string) (int, []string, error) {
	var warnings []string
	count := 0

	for _, u := range users {
		// Check if user exists
		var existingID int64
		err := tx.QueryRow(`SELECT id FROM users WHERE username = ?`, u.Username).Scan(&existingID)

		if err == sql.ErrNoRows {
			// Insert new user with placeholder password
			_, err = tx.Exec(`
				INSERT INTO users (username, password_hash, role, content_rating_limit, require_pin, created_at)
				VALUES (?, '', ?, ?, ?, CURRENT_TIMESTAMP)
			`, u.Username, u.Role, u.ContentRatingLimit, u.RequirePin)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to restore user %s: %v", u.Username, err))
				continue
			}
			count++
		} else if err == nil && mode == "replace" {
			// Update existing user (but not password)
			_, err = tx.Exec(`
				UPDATE users SET role = ?, content_rating_limit = ?, require_pin = ?
				WHERE username = ?
			`, u.Role, u.ContentRatingLimit, u.RequirePin, u.Username)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to update user %s: %v", u.Username, err))
				continue
			}
			count++
		}
		// In merge mode, skip existing users
	}

	return count, warnings, nil
}

func (d *Database) restoreLibraries(tx *sql.Tx, libraries []Library, mode string) (int, error) {
	count := 0
	for _, lib := range libraries {
		if mode == "replace" {
			_, err := tx.Exec(`
				INSERT OR REPLACE INTO libraries (name, path, type, scan_interval)
				VALUES (?, ?, ?, ?)
			`, lib.Name, lib.Path, lib.Type, lib.ScanInterval)
			if err != nil {
				return count, err
			}
			count++
		} else {
			// Merge mode: only insert if path doesn't exist
			var existingID int64
			err := tx.QueryRow(`SELECT id FROM libraries WHERE path = ?`, lib.Path).Scan(&existingID)
			if err == sql.ErrNoRows {
				_, err = tx.Exec(`
					INSERT INTO libraries (name, path, type, scan_interval)
					VALUES (?, ?, ?, ?)
				`, lib.Name, lib.Path, lib.Type, lib.ScanInterval)
				if err != nil {
					return count, err
				}
				count++
			}
		}
	}
	return count, nil
}

func (d *Database) restoreDownloadClients(tx *sql.Tx, clients []DownloadClient, mode string) (int, error) {
	count := 0
	for _, c := range clients {
		if mode == "replace" {
			_, err := tx.Exec(`
				INSERT OR REPLACE INTO download_clients (name, type, host, port, username, password, api_key, use_tls, category, priority, enabled)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, c.Name, c.Type, c.Host, c.Port, c.Username, c.Password, c.APIKey, c.UseTLS, c.Category, c.Priority, c.Enabled)
			if err != nil {
				return count, err
			}
			count++
		} else {
			// Merge mode: only insert if name doesn't exist
			var existingID int64
			err := tx.QueryRow(`SELECT id FROM download_clients WHERE name = ?`, c.Name).Scan(&existingID)
			if err == sql.ErrNoRows {
				_, err = tx.Exec(`
					INSERT INTO download_clients (name, type, host, port, username, password, api_key, use_tls, category, priority, enabled)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, c.Name, c.Type, c.Host, c.Port, c.Username, c.Password, c.APIKey, c.UseTLS, c.Category, c.Priority, c.Enabled)
				if err != nil {
					return count, err
				}
				count++
			}
		}
	}
	return count, nil
}

func (d *Database) restoreProwlarrConfig(tx *sql.Tx, config *ProwlarrConfig, mode string) error {
	if mode == "replace" {
		_, err := tx.Exec(`DELETE FROM prowlarr_config`)
		if err != nil {
			return err
		}
	}

	_, err := tx.Exec(`
		INSERT OR REPLACE INTO prowlarr_config (url, api_key, auto_sync, sync_interval_hours, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, config.URL, config.APIKey, config.AutoSync, config.SyncIntervalHours)
	return err
}

func (d *Database) restoreIndexers(tx *sql.Tx, indexers []Indexer, mode string) (int, error) {
	count := 0
	for _, idx := range indexers {
		if mode == "replace" {
			_, err := tx.Exec(`
				INSERT OR REPLACE INTO indexers (name, type, url, api_key, categories, priority, enabled, synced_from_prowlarr, protocol, supports_movies, supports_tv, supports_music, supports_books, supports_anime, supports_imdb, supports_tmdb, supports_tvdb)
				VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, idx.Name, idx.Type, idx.URL, idx.APIKey, idx.Categories, idx.Priority, idx.Enabled, idx.Protocol, idx.SupportsMovies, idx.SupportsTV, idx.SupportsMusic, idx.SupportsBooks, idx.SupportsAnime, idx.SupportsIMDB, idx.SupportsTMDB, idx.SupportsTVDB)
			if err != nil {
				return count, err
			}
			count++
		} else {
			var existingID int64
			err := tx.QueryRow(`SELECT id FROM indexers WHERE name = ?`, idx.Name).Scan(&existingID)
			if err == sql.ErrNoRows {
				_, err = tx.Exec(`
					INSERT INTO indexers (name, type, url, api_key, categories, priority, enabled, synced_from_prowlarr, protocol, supports_movies, supports_tv, supports_music, supports_books, supports_anime, supports_imdb, supports_tmdb, supports_tvdb)
					VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, idx.Name, idx.Type, idx.URL, idx.APIKey, idx.Categories, idx.Priority, idx.Enabled, idx.Protocol, idx.SupportsMovies, idx.SupportsTV, idx.SupportsMusic, idx.SupportsBooks, idx.SupportsAnime, idx.SupportsIMDB, idx.SupportsTMDB, idx.SupportsTVDB)
				if err != nil {
					return count, err
				}
				count++
			}
		}
	}
	return count, nil
}

func (d *Database) restoreQualityProfiles(tx *sql.Tx, profiles []QualityProfile, mode string) (int, error) {
	count := 0
	for _, p := range profiles {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO quality_profiles (name, upgrade_allowed, upgrade_until_score, min_format_score, cutoff_format_score, qualities, custom_format_scores)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.Name, p.UpgradeAllowed, p.UpgradeUntilScore, p.MinFormatScore, p.CutoffFormatScore, p.Qualities, p.CustomFormatScores)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreQualityPresets(tx *sql.Tx, presets []QualityPreset, mode string) (int, error) {
	count := 0
	for _, p := range presets {
		hdrJSON, _ := json.Marshal(p.HDRFormats)
		audioJSON, _ := json.Marshal(p.AudioFormats)

		_, err := tx.Exec(`
			INSERT OR REPLACE INTO quality_presets (name, media_type, is_default, is_built_in, enabled, priority, resolution, source, hdr_formats, codec, audio_formats, preferred_edition, min_seeders, prefer_season_packs, auto_upgrade, prefer_dual_audio, prefer_dubbed, preferred_language, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, p.Name, p.MediaType, p.IsDefault, p.IsBuiltIn, p.Enabled, p.Priority, p.Resolution, p.Source, string(hdrJSON), p.Codec, string(audioJSON), p.PreferredEdition, p.MinSeeders, p.PreferSeasonPacks, p.AutoUpgrade, p.PreferDualAudio, p.PreferDubbed, p.PreferredLanguage)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreCustomFormats(tx *sql.Tx, formats []CustomFormat, mode string) (int, error) {
	count := 0
	for _, f := range formats {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO custom_formats (name, conditions)
			VALUES (?, ?)
		`, f.Name, f.Conditions)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreCollections(tx *sql.Tx, collections []Collection, mode string) (int, error) {
	count := 0
	for _, c := range collections {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO collections (name, description, tmdb_collection_id, poster_path, backdrop_path, is_auto, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, c.Name, c.Description, c.TmdbCollectionID, c.PosterPath, c.BackdropPath, c.IsAuto, c.SortOrder, c.CreatedAt, c.UpdatedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreCollectionItems(tx *sql.Tx, items []CollectionItem, mode string) (int, error) {
	count := 0
	for _, item := range items {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO collection_items (collection_id, media_type, tmdb_id, title, year, poster_path, sort_order, added_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, item.CollectionID, item.MediaType, item.TmdbID, item.Title, item.Year, item.PosterPath, item.SortOrder, item.AddedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreNamingTemplates(tx *sql.Tx, templates []NamingTemplate, mode string) (int, error) {
	count := 0
	for _, t := range templates {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO naming_templates (type, folder_template, file_template, is_default)
			VALUES (?, ?, ?, ?)
		`, t.Type, t.FolderTemplate, t.FileTemplate, t.IsDefault)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreReleaseFilters(tx *sql.Tx, filters []ReleaseFilter, mode string) (int, error) {
	count := 0
	for _, f := range filters {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO release_filters (preset_id, filter_type, value, is_regex, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, f.PresetID, f.FilterType, f.Value, f.IsRegex, f.CreatedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreDelayProfiles(tx *sql.Tx, profiles []DelayProfile, mode string) (int, error) {
	count := 0
	for _, p := range profiles {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO delay_profiles (name, enabled, delay_minutes, bypass_if_resolution, bypass_if_source, bypass_if_score_above, library_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, p.Name, p.Enabled, p.DelayMinutes, p.BypassIfResolution, p.BypassIfSource, p.BypassIfScoreAbove, p.LibraryID, p.CreatedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreBlockedGroups(tx *sql.Tx, groups []BlockedGroup, mode string) (int, error) {
	count := 0
	for _, g := range groups {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO blocked_groups (name, reason, auto_blocked, failure_count, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, g.Name, g.Reason, g.AutoBlocked, g.FailureCount, g.CreatedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreTrustedGroups(tx *sql.Tx, groups []TrustedGroup, mode string) (int, error) {
	count := 0
	for _, g := range groups {
		_, err := tx.Exec(`
			INSERT OR REPLACE INTO trusted_groups (name, category, created_at)
			VALUES (?, ?, ?)
		`, g.Name, g.Category, g.CreatedAt)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (d *Database) restoreScheduledTasks(tx *sql.Tx, tasks []ScheduledTask, mode string) (int, error) {
	count := 0
	for _, t := range tasks {
		// Only update enabled and interval for existing tasks
		_, err := tx.Exec(`
			UPDATE scheduled_tasks SET enabled = ?, interval_minutes = ?
			WHERE task_type = ?
		`, t.Enabled, t.IntervalMinutes, t.TaskType)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
