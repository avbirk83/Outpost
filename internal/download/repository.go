package download

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Repository handles database operations for tracked downloads
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new download repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new tracked download
func (r *Repository) Create(td *TrackedDownload) error {
	parsedInfoJSON, _ := json.Marshal(td.ParsedInfo)
	warningsJSON, _ := json.Marshal(td.Warnings)
	errorsJSON, _ := json.Marshal(td.Errors)

	result, err := r.db.Exec(`
		INSERT INTO tracked_downloads (
			download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		td.DownloadClientID, td.ExternalID, td.RequestID, td.MediaID, td.MediaType,
		td.State, td.PreviousState, td.StateChangedAt, td.Title, string(parsedInfoJSON),
		td.Size, td.Downloaded, td.Progress, td.Speed, int64(td.ETA.Seconds()), td.Seeders,
		td.DownloadPath, td.ImportPath, td.Quality, td.CustomFormatScore,
		td.GrabbedAt, td.CompletedAt, td.ImportedAt,
		string(warningsJSON), string(errorsJSON), td.ImportBlockReason,
		td.Ratio, int64(td.SeedingTime.Seconds()), td.CanRemove, time.Now(), time.Now(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	td.ID = id
	return nil
}

// GetByID retrieves a tracked download by ID
func (r *Repository) GetByID(id int64) (*TrackedDownload, error) {
	row := r.db.QueryRow(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads WHERE id = ?`, id)
	return r.scanRow(row)
}

// GetByExternalID retrieves a tracked download by client and external ID
func (r *Repository) GetByExternalID(clientID int64, externalID string) (*TrackedDownload, error) {
	row := r.db.QueryRow(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads WHERE download_client_id = ? AND external_id = ?`, clientID, externalID)
	return r.scanRow(row)
}

// GetActive retrieves all non-terminal downloads
func (r *Repository) GetActive() ([]*TrackedDownload, error) {
	rows, err := r.db.Query(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads
		WHERE state NOT IN ('imported', 'ignored')
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

// GetByState retrieves downloads in a specific state
func (r *Repository) GetByState(state DownloadState) ([]*TrackedDownload, error) {
	rows, err := r.db.Query(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads WHERE state = ?
		ORDER BY created_at DESC`, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

// GetPendingImport retrieves downloads ready for import
func (r *Repository) GetPendingImport() ([]*TrackedDownload, error) {
	rows, err := r.db.Query(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads
		WHERE state IN ('completed', 'import_pending')
		ORDER BY completed_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

// GetReadyForRemoval retrieves imported downloads that can be removed
func (r *Repository) GetReadyForRemoval(config SeedingConfig) ([]*TrackedDownload, error) {
	minSeedSeconds := int64(config.MinSeedTime.Seconds())
	maxSeedSeconds := int64(config.MaxSeedTime.Seconds())

	rows, err := r.db.Query(`
		SELECT id, download_client_id, external_id, request_id, media_id, media_type,
			state, previous_state, state_changed_at, title, parsed_info,
			size, downloaded, progress, speed, eta, seeders,
			download_path, import_path, quality, custom_format_score,
			grabbed_at, completed_at, imported_at,
			warnings, errors, import_block_reason,
			ratio, seeding_time, can_remove, created_at, updated_at
		FROM tracked_downloads
		WHERE state = 'imported'
		AND (seeding_time >= ? OR (ratio >= ? AND seeding_time >= ?))`,
		maxSeedSeconds, config.MinRatio, minSeedSeconds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

// Update updates a tracked download
func (r *Repository) Update(td *TrackedDownload) error {
	parsedInfoJSON, _ := json.Marshal(td.ParsedInfo)
	warningsJSON, _ := json.Marshal(td.Warnings)
	errorsJSON, _ := json.Marshal(td.Errors)

	_, err := r.db.Exec(`
		UPDATE tracked_downloads SET
			download_client_id = ?, external_id = ?, request_id = ?, media_id = ?, media_type = ?,
			state = ?, previous_state = ?, state_changed_at = ?, title = ?, parsed_info = ?,
			size = ?, downloaded = ?, progress = ?, speed = ?, eta = ?, seeders = ?,
			download_path = ?, import_path = ?, quality = ?, custom_format_score = ?,
			grabbed_at = ?, completed_at = ?, imported_at = ?,
			warnings = ?, errors = ?, import_block_reason = ?,
			ratio = ?, seeding_time = ?, can_remove = ?, updated_at = ?
		WHERE id = ?`,
		td.DownloadClientID, td.ExternalID, td.RequestID, td.MediaID, td.MediaType,
		td.State, td.PreviousState, td.StateChangedAt, td.Title, string(parsedInfoJSON),
		td.Size, td.Downloaded, td.Progress, td.Speed, int64(td.ETA.Seconds()), td.Seeders,
		td.DownloadPath, td.ImportPath, td.Quality, td.CustomFormatScore,
		td.GrabbedAt, td.CompletedAt, td.ImportedAt,
		string(warningsJSON), string(errorsJSON), td.ImportBlockReason,
		td.Ratio, int64(td.SeedingTime.Seconds()), td.CanRemove, time.Now(),
		td.ID,
	)
	return err
}

// UpdateState updates just the state with an event record
func (r *Repository) UpdateState(td *TrackedDownload, newState DownloadState, reason string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Record the event
	_, err = tx.Exec(`
		INSERT INTO download_events (download_id, from_state, to_state, reason, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		td.ID, td.State, newState, reason, time.Now())
	if err != nil {
		return err
	}

	// Update the download
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE tracked_downloads SET
			previous_state = ?, state = ?, state_changed_at = ?, updated_at = ?
		WHERE id = ?`,
		td.State, newState, now, now, td.ID)
	if err != nil {
		return err
	}

	// Update struct
	td.PreviousState = td.State
	td.State = newState
	td.StateChangedAt = now
	td.UpdatedAt = now

	return tx.Commit()
}

// Delete removes a tracked download
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM tracked_downloads WHERE id = ?`, id)
	return err
}

// GetEvents retrieves events for a download
func (r *Repository) GetEvents(downloadID int64) ([]*DownloadEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, download_id, from_state, to_state, reason, details, created_at
		FROM download_events WHERE download_id = ?
		ORDER BY created_at DESC`, downloadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*DownloadEvent
	for rows.Next() {
		e := &DownloadEvent{}
		var fromState sql.NullString
		err := rows.Scan(&e.ID, &e.DownloadID, &fromState, &e.ToState, &e.Reason, &e.Details, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		if fromState.Valid {
			e.FromState = DownloadState(fromState.String)
		}
		events = append(events, e)
	}
	return events, nil
}

// scanRow scans a single row into a TrackedDownload
func (r *Repository) scanRow(row *sql.Row) (*TrackedDownload, error) {
	td := &TrackedDownload{}
	var requestID, mediaID sql.NullInt64
	var mediaType, prevState, quality, downloadPath, importPath sql.NullString
	var stateChangedAt, grabbedAt, completedAt, importedAt sql.NullTime
	var parsedInfoJSON, warningsJSON, errorsJSON, importBlockReason sql.NullString
	var etaSeconds, seedingTimeSeconds int64
	var canRemove int

	err := row.Scan(
		&td.ID, &td.DownloadClientID, &td.ExternalID, &requestID, &mediaID, &mediaType,
		&td.State, &prevState, &stateChangedAt, &td.Title, &parsedInfoJSON,
		&td.Size, &td.Downloaded, &td.Progress, &td.Speed, &etaSeconds, &td.Seeders,
		&downloadPath, &importPath, &quality, &td.CustomFormatScore,
		&grabbedAt, &completedAt, &importedAt,
		&warningsJSON, &errorsJSON, &importBlockReason,
		&td.Ratio, &seedingTimeSeconds, &canRemove, &td.CreatedAt, &td.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if requestID.Valid {
		td.RequestID = &requestID.Int64
	}
	if mediaID.Valid {
		td.MediaID = &mediaID.Int64
	}
	if mediaType.Valid {
		td.MediaType = mediaType.String
	}
	if prevState.Valid {
		td.PreviousState = DownloadState(prevState.String)
	}
	if stateChangedAt.Valid {
		td.StateChangedAt = stateChangedAt.Time
	}
	if quality.Valid {
		td.Quality = quality.String
	}
	if downloadPath.Valid {
		td.DownloadPath = downloadPath.String
	}
	if importPath.Valid {
		td.ImportPath = importPath.String
	}
	if grabbedAt.Valid {
		td.GrabbedAt = grabbedAt.Time
	}
	if completedAt.Valid {
		td.CompletedAt = &completedAt.Time
	}
	if importedAt.Valid {
		td.ImportedAt = &importedAt.Time
	}
	if importBlockReason.Valid {
		td.ImportBlockReason = importBlockReason.String
	}

	td.ETA = time.Duration(etaSeconds) * time.Second
	td.SeedingTime = time.Duration(seedingTimeSeconds) * time.Second
	td.CanRemove = canRemove == 1

	// Parse JSON fields
	if parsedInfoJSON.Valid && parsedInfoJSON.String != "" && parsedInfoJSON.String != "null" {
		json.Unmarshal([]byte(parsedInfoJSON.String), &td.ParsedInfo)
	}
	if warningsJSON.Valid && warningsJSON.String != "" {
		json.Unmarshal([]byte(warningsJSON.String), &td.Warnings)
	}
	if errorsJSON.Valid && errorsJSON.String != "" {
		json.Unmarshal([]byte(errorsJSON.String), &td.Errors)
	}

	return td, nil
}

// scanRows scans multiple rows into TrackedDownloads
func (r *Repository) scanRows(rows *sql.Rows) ([]*TrackedDownload, error) {
	var downloads []*TrackedDownload
	for rows.Next() {
		td := &TrackedDownload{}
		var requestID, mediaID sql.NullInt64
		var mediaType, prevState, quality, downloadPath, importPath sql.NullString
		var stateChangedAt, grabbedAt, completedAt, importedAt sql.NullTime
		var parsedInfoJSON, warningsJSON, errorsJSON, importBlockReason sql.NullString
		var etaSeconds, seedingTimeSeconds int64
		var canRemove int

		err := rows.Scan(
			&td.ID, &td.DownloadClientID, &td.ExternalID, &requestID, &mediaID, &mediaType,
			&td.State, &prevState, &stateChangedAt, &td.Title, &parsedInfoJSON,
			&td.Size, &td.Downloaded, &td.Progress, &td.Speed, &etaSeconds, &td.Seeders,
			&downloadPath, &importPath, &quality, &td.CustomFormatScore,
			&grabbedAt, &completedAt, &importedAt,
			&warningsJSON, &errorsJSON, &importBlockReason,
			&td.Ratio, &seedingTimeSeconds, &canRemove, &td.CreatedAt, &td.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if requestID.Valid {
			td.RequestID = &requestID.Int64
		}
		if mediaID.Valid {
			td.MediaID = &mediaID.Int64
		}
		if mediaType.Valid {
			td.MediaType = mediaType.String
		}
		if prevState.Valid {
			td.PreviousState = DownloadState(prevState.String)
		}
		if stateChangedAt.Valid {
			td.StateChangedAt = stateChangedAt.Time
		}
		if quality.Valid {
			td.Quality = quality.String
		}
		if downloadPath.Valid {
			td.DownloadPath = downloadPath.String
		}
		if importPath.Valid {
			td.ImportPath = importPath.String
		}
		if grabbedAt.Valid {
			td.GrabbedAt = grabbedAt.Time
		}
		if completedAt.Valid {
			td.CompletedAt = &completedAt.Time
		}
		if importedAt.Valid {
			td.ImportedAt = &importedAt.Time
		}
		if importBlockReason.Valid {
			td.ImportBlockReason = importBlockReason.String
		}

		td.ETA = time.Duration(etaSeconds) * time.Second
		td.SeedingTime = time.Duration(seedingTimeSeconds) * time.Second
		td.CanRemove = canRemove == 1

		// Parse JSON fields
		if parsedInfoJSON.Valid && parsedInfoJSON.String != "" && parsedInfoJSON.String != "null" {
			json.Unmarshal([]byte(parsedInfoJSON.String), &td.ParsedInfo)
		}
		if warningsJSON.Valid && warningsJSON.String != "" {
			json.Unmarshal([]byte(warningsJSON.String), &td.Warnings)
		}
		if errorsJSON.Valid && errorsJSON.String != "" {
			json.Unmarshal([]byte(errorsJSON.String), &td.Errors)
		}

		downloads = append(downloads, td)
	}
	return downloads, nil
}
