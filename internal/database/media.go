package database

import (
	"database/sql"
	"time"
)

// OwnedEpisode represents an owned episode with season/episode numbers
type OwnedEpisode struct {
	SeasonNumber  int
	EpisodeNumber int
}

// Movie operations

func (d *Database) CreateMovie(movie *Movie) error {
	result, err := d.db.Exec(
		"INSERT INTO movies (library_id, title, year, path, size) VALUES (?, ?, ?, ?, ?)",
		movie.LibraryID, movie.Title, movie.Year, movie.Path, movie.Size,
	)
	if err != nil {
		return err
	}
	movie.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateMovieMetadata(movie *Movie) error {
	_, err := d.db.Exec(`
		UPDATE movies SET
			tmdb_id = ?, imdb_id = ?, original_title = ?, overview = ?, tagline = ?,
			runtime = ?, rating = ?, content_rating = ?, genres = ?, "cast" = ?, crew = ?,
			director = ?, writer = ?, editor = ?, producers = ?, status = ?, budget = ?, revenue = ?,
			country = ?, original_language = ?, theatrical_release = ?, digital_release = ?, studios = ?, trailers = ?,
			poster_path = ?, backdrop_path = ?, focal_x = ?, focal_y = ?
		WHERE id = ?`,
		movie.TmdbID, movie.ImdbID, movie.OriginalTitle, movie.Overview, movie.Tagline,
		movie.Runtime, movie.Rating, movie.ContentRating, movie.Genres, movie.Cast, movie.Crew,
		movie.Director, movie.Writer, movie.Editor, movie.Producers, movie.Status, movie.Budget, movie.Revenue,
		movie.Country, movie.OriginalLanguage, movie.TheatricalRelease, movie.DigitalRelease, movie.Studios, movie.Trailers,
		movie.PosterPath, movie.BackdropPath, movie.FocalX, movie.FocalY, movie.ID,
	)
	return err
}

func (d *Database) UpdateMoviePath(id int64, newPath string) error {
	_, err := d.db.Exec(`UPDATE movies SET path = ? WHERE id = ?`, newPath, id)
	return err
}

func (d *Database) GetMovies() ([]Movie, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
			&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
			&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
			&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
			&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount); err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// GetMovieTMDBIDs returns a set of all TMDB IDs in the movie library
func (d *Database) GetMovieTMDBIDs() (map[int64]bool, error) {
	rows, err := d.db.Query(`SELECT tmdb_id FROM movies WHERE tmdb_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

// GetShowTMDBIDs returns a set of all TMDB IDs in the TV show library
func (d *Database) GetShowTMDBIDs() (map[int64]bool, error) {
	rows, err := d.db.Query(`SELECT tmdb_id FROM shows WHERE tmdb_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

func (d *Database) GetMovieByPath(path string) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE path = ?`, path,
	).Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
		&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
		&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
		&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
		&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Show operations

func (d *Database) CreateShow(show *Show) error {
	result, err := d.db.Exec(
		"INSERT INTO shows (library_id, title, year, path) VALUES (?, ?, ?, ?)",
		show.LibraryID, show.Title, show.Year, show.Path,
	)
	if err != nil {
		return err
	}
	show.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateShowMetadata(show *Show) error {
	_, err := d.db.Exec(`
		UPDATE shows SET
			tmdb_id = ?, tvdb_id = ?, imdb_id = ?, original_title = ?, year = ?, overview = ?,
			status = ?, rating = ?, content_rating = ?, genres = ?, "cast" = ?, crew = ?,
			network = ?, poster_path = ?, backdrop_path = ?, focal_x = ?, focal_y = ?
		WHERE id = ?`,
		show.TmdbID, show.TvdbID, show.ImdbID, show.OriginalTitle, show.Year, show.Overview,
		show.Status, show.Rating, show.ContentRating, show.Genres, show.Cast, show.Crew,
		show.Network, show.PosterPath, show.BackdropPath, show.FocalX, show.FocalY, show.ID,
	)
	return err
}

func (d *Database) GetShows() ([]Show, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var s Show
		var addedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
			&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
			&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt); err != nil {
			return nil, err
		}
		if addedAt.Valid {
			s.AddedAt = &addedAt.Time
		}
		shows = append(shows, s)
	}
	return shows, nil
}

func (d *Database) GetShowByPath(path string) (*Show, error) {
	var s Show
	var addedAt sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows WHERE path = ?`, path,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
		&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt)
	if err != nil {
		return nil, err
	}
	if addedAt.Valid {
		s.AddedAt = &addedAt.Time
	}
	return &s, nil
}

func (d *Database) GetShow(id int64) (*Show, error) {
	var s Show
	var addedAt sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year,
			overview, status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path, focal_x, focal_y, path, added_at
		FROM shows WHERE id = ?`, id,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew,
		&s.Network, &s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &addedAt)
	if err != nil {
		return nil, err
	}
	if addedAt.Valid {
		s.AddedAt = &addedAt.Time
	}
	return &s, nil
}

// Season operations

func (d *Database) CreateSeason(season *Season) error {
	result, err := d.db.Exec(
		"INSERT INTO seasons (show_id, season_number) VALUES (?, ?)",
		season.ShowID, season.SeasonNumber,
	)
	if err != nil {
		return err
	}
	season.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateSeasonMetadata(season *Season) error {
	_, err := d.db.Exec(`
		UPDATE seasons SET name = ?, overview = ?, poster_path = ?, air_date = ?
		WHERE id = ?`,
		season.Name, season.Overview, season.PosterPath, season.AirDate, season.ID,
	)
	return err
}

func (d *Database) GetSeasonsByShow(showID int64) ([]Season, error) {
	rows, err := d.db.Query(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE show_id = ? ORDER BY season_number`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []Season
	for rows.Next() {
		var s Season
		if err := rows.Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

func (d *Database) GetSeason(showID int64, seasonNumber int) (*Season, error) {
	var s Season
	err := d.db.QueryRow(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE show_id = ? AND season_number = ?`,
		showID, seasonNumber,
	).Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) GetSeasonByID(id int64) (*Season, error) {
	var s Season
	err := d.db.QueryRow(`
		SELECT id, show_id, season_number, name, overview, poster_path, air_date
		FROM seasons WHERE id = ?`, id,
	).Scan(&s.ID, &s.ShowID, &s.SeasonNumber, &s.Name, &s.Overview, &s.PosterPath, &s.AirDate)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Episode operations

func (d *Database) CreateEpisode(ep *Episode) error {
	result, err := d.db.Exec(
		"INSERT INTO episodes (season_id, episode_number, title, path, size) VALUES (?, ?, ?, ?, ?)",
		ep.SeasonID, ep.EpisodeNumber, ep.Title, ep.Path, ep.Size,
	)
	if err != nil {
		return err
	}
	ep.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) UpdateEpisodeMetadata(ep *Episode) error {
	_, err := d.db.Exec(`
		UPDATE episodes SET title = ?, overview = ?, air_date = ?, runtime = ?, still_path = ?
		WHERE id = ?`,
		ep.Title, ep.Overview, ep.AirDate, ep.Runtime, ep.StillPath, ep.ID,
	)
	return err
}

func (d *Database) GetEpisodesBySeason(seasonID int64) ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE season_id = ? ORDER BY episode_number`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size); err != nil {
			return nil, err
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

func (d *Database) GetEpisodeByPath(path string) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE path = ?`, path,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetOwnedEpisodesByShow returns all owned episodes for a show as season/episode number pairs
func (d *Database) GetOwnedEpisodesByShow(showID int64) ([]OwnedEpisode, error) {
	rows, err := d.db.Query(`
		SELECT s.season_number, e.episode_number
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE s.show_id = ?
		ORDER BY s.season_number, e.episode_number`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []OwnedEpisode
	for rows.Next() {
		var ep OwnedEpisode
		if err := rows.Scan(&ep.SeasonNumber, &ep.EpisodeNumber); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}
	return episodes, nil
}

func (d *Database) GetEpisode(id int64) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE id = ?`, id,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) GetShowIDForEpisode(episodeID int64) (int64, error) {
	var showID int64
	err := d.db.QueryRow(`
		SELECT s.show_id
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE e.id = ?`, episodeID,
	).Scan(&showID)
	return showID, err
}

// GetEpisodeByShowSeasonEpisode finds an episode by show ID, season number, and episode number
func (d *Database) GetEpisodeByShowSeasonEpisode(showID int64, seasonNum, episodeNum int) (*Episode, error) {
	var e Episode
	err := d.db.QueryRow(`
		SELECT e.id, e.season_id, e.episode_number, e.title, e.overview, e.air_date, e.runtime, e.still_path, e.path, e.size
		FROM episodes e
		JOIN seasons s ON e.season_id = s.id
		WHERE s.show_id = ? AND s.season_number = ? AND e.episode_number = ?`,
		showID, seasonNum, episodeNum,
	).Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) DeleteEpisode(id int64) error {
	_, err := d.db.Exec("DELETE FROM episodes WHERE id = ?", id)
	return err
}

// GetEpisodesByLibrary retrieves all episodes for a library (for cleanup)
func (d *Database) GetEpisodesByLibrary(libraryID int64) ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT e.id, e.episode_number, e.path, e.missing_since
		FROM episodes e
		JOIN seasons sea ON e.season_id = sea.id
		JOIN shows s ON sea.show_id = s.id
		WHERE s.library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.EpisodeNumber, &e.Path, &e.MissingSince); err != nil {
			continue
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

// GetAllEpisodes retrieves all episodes (for quality detection)
func (d *Database) GetAllEpisodes() ([]Episode, error) {
	rows, err := d.db.Query(`SELECT id, path FROM episodes WHERE path IS NOT NULL AND path != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.Path); err != nil {
			continue
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

func (d *Database) UpdateEpisodeSize(id int64, size int64) error {
	_, err := d.db.Exec("UPDATE episodes SET size = ? WHERE id = ?", size, id)
	return err
}

func (d *Database) GetEpisodesWithMissingSize() ([]Episode, error) {
	rows, err := d.db.Query(`
		SELECT id, season_id, episode_number, title, overview, air_date, runtime, still_path, path, size
		FROM episodes WHERE size = 0 OR size IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath, &e.Path, &e.Size); err != nil {
			return nil, err
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

func (d *Database) GetMovie(id int64) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE id = ?`, id,
	).Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
		&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
		&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
		&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
		&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteMovie removes a movie from the database
func (d *Database) DeleteMovie(id int64) error {
	_, err := d.db.Exec("DELETE FROM movies WHERE id = ?", id)
	return err
}

// GetMoviesByLibrary retrieves all movies for a library (for cleanup)
func (d *Database) GetMoviesByLibrary(libraryID int64) ([]Movie, error) {
	rows, err := d.db.Query(`SELECT id, title, path, missing_since FROM movies WHERE library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Path, &m.MissingSince); err != nil {
			continue
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// GetMovieByTmdb retrieves a movie by its TMDB ID
func (d *Database) GetMovieByTmdb(tmdbID int64) (*Movie, error) {
	var m Movie
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, imdb_id, title, original_title, year, overview, tagline,
			runtime, rating, content_rating, genres, "cast", crew, director, writer, editor, producers, status, budget, revenue,
			country, original_language, theatrical_release, digital_release, studios, trailers, poster_path, backdrop_path, focal_x, focal_y, path, size, added_at, last_watched_at, play_count
		FROM movies WHERE tmdb_id = ?`, tmdbID,
	).Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.ImdbID, &m.Title, &m.OriginalTitle, &m.Year,
		&m.Overview, &m.Tagline, &m.Runtime, &m.Rating, &m.ContentRating, &m.Genres, &m.Cast, &m.Crew,
		&m.Director, &m.Writer, &m.Editor, &m.Producers, &m.Status, &m.Budget, &m.Revenue,
		&m.Country, &m.OriginalLanguage, &m.TheatricalRelease, &m.DigitalRelease, &m.Studios, &m.Trailers,
		&m.PosterPath, &m.BackdropPath, &m.FocalX, &m.FocalY, &m.Path, &m.Size, &m.AddedAt, &m.LastWatchedAt, &m.PlayCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetShowByTmdb retrieves a show by its TMDB ID
func (d *Database) GetShowByTmdb(tmdbID int64) (*Show, error) {
	var s Show
	err := d.db.QueryRow(`
		SELECT id, library_id, tmdb_id, tvdb_id, imdb_id, title, original_title, year, overview,
			status, rating, content_rating, genres, "cast", crew, network, poster_path, backdrop_path,
			focal_x, focal_y, path, added_at
		FROM shows WHERE tmdb_id = ?`, tmdbID,
	).Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.TvdbID, &s.ImdbID, &s.Title, &s.OriginalTitle, &s.Year,
		&s.Overview, &s.Status, &s.Rating, &s.ContentRating, &s.Genres, &s.Cast, &s.Crew, &s.Network,
		&s.PosterPath, &s.BackdropPath, &s.FocalX, &s.FocalY, &s.Path, &s.AddedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateMoviePlayCount increments the play count and updates last watched time
func (d *Database) UpdateMoviePlayCount(id int64) error {
	now := time.Now().Format(time.RFC3339)
	_, err := d.db.Exec(`
		UPDATE movies SET
			play_count = play_count + 1,
			last_watched_at = ?
		WHERE id = ?`, now, id)
	return err
}

// Missing tracking methods

// MarkMovieMissing marks a movie as missing (file not found)
func (d *Database) MarkMovieMissing(id int64) error {
	_, err := d.db.Exec(`UPDATE movies SET missing_since = ? WHERE id = ? AND missing_since IS NULL`,
		time.Now(), id)
	return err
}

// ClearMovieMissing clears the missing status when file reappears
func (d *Database) ClearMovieMissing(id int64) error {
	_, err := d.db.Exec(`UPDATE movies SET missing_since = NULL WHERE id = ?`, id)
	return err
}

// MarkEpisodeMissing marks an episode as missing
func (d *Database) MarkEpisodeMissing(id int64) error {
	_, err := d.db.Exec(`UPDATE episodes SET missing_since = ? WHERE id = ? AND missing_since IS NULL`,
		time.Now(), id)
	return err
}

// ClearEpisodeMissing clears the missing status when file reappears
func (d *Database) ClearEpisodeMissing(id int64) error {
	_, err := d.db.Exec(`UPDATE episodes SET missing_since = NULL WHERE id = ?`, id)
	return err
}

// GetMissingMovies returns movies that have been missing for longer than the given duration
func (d *Database) GetMissingMovies(olderThan time.Duration) ([]Movie, error) {
	cutoff := time.Now().Add(-olderThan)
	rows, err := d.db.Query(`
		SELECT id, library_id, title, path, missing_since
		FROM movies
		WHERE missing_since IS NOT NULL AND missing_since < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.LibraryID, &m.Title, &m.Path, &m.MissingSince); err != nil {
			continue
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// GetMissingEpisodes returns episodes that have been missing for longer than the given duration
func (d *Database) GetMissingEpisodes(olderThan time.Duration) ([]Episode, error) {
	cutoff := time.Now().Add(-olderThan)
	rows, err := d.db.Query(`
		SELECT id, season_id, episode_number, title, path, missing_since
		FROM episodes
		WHERE missing_since IS NOT NULL AND missing_since < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.Title, &e.Path, &e.MissingSince); err != nil {
			continue
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

// DeleteMissingMovies deletes movies that have been missing for longer than the given duration
func (d *Database) DeleteMissingMovies(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := d.db.Exec(`DELETE FROM movies WHERE missing_since IS NOT NULL AND missing_since < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// DeleteMissingEpisodes deletes episodes that have been missing for longer than the given duration
func (d *Database) DeleteMissingEpisodes(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := d.db.Exec(`DELETE FROM episodes WHERE missing_since IS NOT NULL AND missing_since < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// Match review methods

// GetMoviesNeedingReview returns movies flagged for manual review (low confidence matches)
func (d *Database) GetMoviesNeedingReview() ([]Movie, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, title, year, path, match_confidence
		FROM movies
		WHERE needs_match_review = 1
		ORDER BY match_confidence ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.LibraryID, &m.TmdbID, &m.Title, &m.Year, &m.Path, &m.MatchConfidence); err != nil {
			continue
		}
		m.NeedsMatchReview = true
		movies = append(movies, m)
	}
	return movies, nil
}

// GetShowsNeedingReview returns shows flagged for manual review
func (d *Database) GetShowsNeedingReview() ([]Show, error) {
	rows, err := d.db.Query(`
		SELECT id, library_id, tmdb_id, title, year, path, match_confidence
		FROM shows
		WHERE needs_match_review = 1
		ORDER BY match_confidence ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var s Show
		if err := rows.Scan(&s.ID, &s.LibraryID, &s.TmdbID, &s.Title, &s.Year, &s.Path, &s.MatchConfidence); err != nil {
			continue
		}
		s.NeedsMatchReview = true
		shows = append(shows, s)
	}
	return shows, nil
}

// UpdateMovieTmdbMatch updates the TMDB ID for a movie and clears the review flag
func (d *Database) UpdateMovieTmdbMatch(id, tmdbID int64) error {
	_, err := d.db.Exec(`
		UPDATE movies SET
			tmdb_id = ?,
			needs_match_review = 0,
			match_confidence = 1.0
		WHERE id = ?`, tmdbID, id)
	return err
}

// UpdateShowTmdbMatch updates the TMDB ID for a show and clears the review flag
func (d *Database) UpdateShowTmdbMatch(id, tmdbID int64) error {
	_, err := d.db.Exec(`
		UPDATE shows SET
			tmdb_id = ?,
			needs_match_review = 0,
			match_confidence = 1.0
		WHERE id = ?`, tmdbID, id)
	return err
}

// CreateEpisodeWithExtras creates an episode with multi-episode and absolute number support
func (d *Database) CreateEpisodeWithExtras(ep *Episode) error {
	result, err := d.db.Exec(
		`INSERT INTO episodes (season_id, episode_number, episode_end, absolute_number, title, path, size, match_confidence)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ep.SeasonID, ep.EpisodeNumber, ep.EpisodeEnd, ep.AbsoluteNumber, ep.Title, ep.Path, ep.Size, ep.MatchConfidence,
	)
	if err != nil {
		return err
	}
	ep.ID, _ = result.LastInsertId()
	return nil
}

// SetMovieMatchConfidence updates the match confidence and review flag for a movie
func (d *Database) SetMovieMatchConfidence(id int64, confidence float64, needsReview bool) error {
	reviewInt := 0
	if needsReview {
		reviewInt = 1
	}
	_, err := d.db.Exec(`UPDATE movies SET match_confidence = ?, needs_match_review = ? WHERE id = ?`,
		confidence, reviewInt, id)
	return err
}

// SetShowMatchConfidence updates the match confidence and review flag for a show
func (d *Database) SetShowMatchConfidence(id int64, confidence float64, needsReview bool) error {
	reviewInt := 0
	if needsReview {
		reviewInt = 1
	}
	_, err := d.db.Exec(`UPDATE shows SET match_confidence = ?, needs_match_review = ? WHERE id = ?`,
		confidence, reviewInt, id)
	return err
}
