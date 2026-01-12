package database

// GetShowsByLibrary retrieves all shows for a library
func (d *Database) GetShowsByLibrary(libraryID int64) ([]Show, error) {
	rows, err := d.db.Query(`SELECT id, library_id, title, path FROM shows WHERE library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var s Show
		if err := rows.Scan(&s.ID, &s.LibraryID, &s.Title, &s.Path); err != nil {
			continue
		}
		shows = append(shows, s)
	}
	return shows, nil
}
