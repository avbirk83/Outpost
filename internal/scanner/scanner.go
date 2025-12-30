package scanner

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/metadata"
)

var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
	".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
}

var audioExtensions = map[string]bool{
	".mp3": true, ".flac": true, ".m4a": true, ".aac": true,
	".ogg": true, ".wav": true, ".wma": true, ".opus": true,
}

var bookExtensions = map[string]bool{
	".epub": true, ".pdf": true, ".mobi": true,
	".cbz": true, ".cbr": true, ".azw": true, ".azw3": true,
}

// Movie filename patterns
// Examples: "Movie Name (2020).mkv", "Movie.Name.2020.1080p.BluRay.mkv"
var movieYearPattern = regexp.MustCompile(`^(.+?)[\.\s\-_]*\(?(\d{4})\)?`)

// TV show patterns
// Examples: "Show Name S01E02.mkv", "Show.Name.S01E02.Episode.Title.mkv"
var tvPattern = regexp.MustCompile(`(?i)^(.+?)[\.\s\-_]*S(\d{1,2})E(\d{1,2})`)

// Alternative TV pattern: "Show Name - 1x02" or "Show Name 1x02"
var tvAltPattern = regexp.MustCompile(`(?i)^(.+?)[\.\s\-_]*(\d{1,2})x(\d{1,2})`)

type Scanner struct {
	db   *database.Database
	meta *metadata.Service
}

func New(db *database.Database, meta *metadata.Service) *Scanner {
	return &Scanner{db: db, meta: meta}
}

func (s *Scanner) ScanLibrary(lib *database.Library) error {
	log.Printf("Scanning library: %s (%s)", lib.Name, lib.Path)

	switch lib.Type {
	case "movies":
		return s.scanMovies(lib)
	case "tv":
		return s.scanTV(lib)
	case "music":
		return s.scanMusic(lib)
	case "books":
		return s.scanBooks(lib)
	default:
		log.Printf("Unknown library type: %s", lib.Type)
		return nil
	}
}

func (s *Scanner) scanMovies(lib *database.Library) error {
	return filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] {
			return nil
		}

		// Check if already in database
		if _, err := s.db.GetMovieByPath(path); err == nil {
			return nil // Already exists
		}

		// Parse filename
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		title, year := parseMovieFilename(filename)

		movie := &database.Movie{
			LibraryID: lib.ID,
			Title:     title,
			Year:      year,
			Path:      path,
			Size:      info.Size(),
		}

		if err := s.db.CreateMovie(movie); err != nil {
			log.Printf("Failed to add movie %s: %v", path, err)
		} else {
			log.Printf("Added movie: %s (%d)", title, year)
			// Fetch metadata from TMDB
			if s.meta != nil {
				if err := s.meta.FetchMovieMetadata(movie); err != nil {
					log.Printf("Failed to fetch metadata for %s: %v", title, err)
				}
			}
		}

		return nil
	})
}

func (s *Scanner) scanTV(lib *database.Library) error {
	return filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] {
			return nil
		}

		// Check if already in database
		if _, err := s.db.GetEpisodeByPath(path); err == nil {
			return nil
		}

		// Parse filename
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		showTitle, seasonNum, episodeNum := parseTVFilename(filename)

		if showTitle == "" || seasonNum == 0 {
			log.Printf("Could not parse TV filename: %s", filename)
			return nil
		}

		// Get or create show
		showPath := filepath.Dir(path)
		// Try to go up one more level if we're in a season folder
		if strings.Contains(strings.ToLower(filepath.Base(showPath)), "season") {
			showPath = filepath.Dir(showPath)
		}

		show, err := s.db.GetShowByPath(showPath)
		isNewShow := false
		if err == sql.ErrNoRows {
			showYear := extractYearFromPath(showPath)
			show = &database.Show{
				LibraryID: lib.ID,
				Title:     showTitle,
				Year:      showYear,
				Path:      showPath,
			}
			if err := s.db.CreateShow(show); err != nil {
				log.Printf("Failed to create show %s: %v", showTitle, err)
				return nil
			}
			log.Printf("Added show: %s", showTitle)
			isNewShow = true
		} else if err != nil {
			return nil
		}

		// Get or create season
		season, err := s.db.GetSeason(show.ID, seasonNum)
		if err == sql.ErrNoRows {
			season = &database.Season{
				ShowID:       show.ID,
				SeasonNumber: seasonNum,
			}
			if err := s.db.CreateSeason(season); err != nil {
				log.Printf("Failed to create season %d: %v", seasonNum, err)
				return nil
			}
		} else if err != nil {
			return nil
		}

		// Create episode
		episode := &database.Episode{
			SeasonID:      season.ID,
			EpisodeNumber: episodeNum,
			Title:         "", // Will be fetched from metadata later
			Path:          path,
			Size:          info.Size(),
		}

		if err := s.db.CreateEpisode(episode); err != nil {
			log.Printf("Failed to add episode: %v", err)
		} else {
			log.Printf("Added episode: %s S%02dE%02d", showTitle, seasonNum, episodeNum)
		}

		// Fetch show metadata if this is a new show
		if isNewShow && s.meta != nil {
			if err := s.meta.FetchShowMetadata(show); err != nil {
				log.Printf("Failed to fetch metadata for %s: %v", showTitle, err)
			}
		}

		return nil
	})
}

func (s *Scanner) scanMusic(lib *database.Library) error {
	// Music structure: Artist/Album/Track.mp3
	return filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !audioExtensions[ext] {
			return nil
		}

		// Check if already in database
		if _, err := s.db.GetTrackByPath(path); err == nil {
			return nil
		}

		// Parse path structure: Artist/Album/Track.ext
		relPath, _ := filepath.Rel(lib.Path, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		var artistName, albumName string
		if len(parts) >= 3 {
			artistName = parts[0]
			albumName = parts[1]
		} else if len(parts) == 2 {
			artistName = parts[0]
			albumName = "Unknown Album"
		} else {
			artistName = "Unknown Artist"
			albumName = "Unknown Album"
		}

		// Get or create artist
		artistPath := filepath.Join(lib.Path, artistName)
		artist, err := s.db.GetArtistByPath(artistPath)
		if err != nil {
			artist = &database.Artist{
				LibraryID: lib.ID,
				Name:      artistName,
				Path:      artistPath,
			}
			if err := s.db.CreateArtist(artist); err != nil {
				log.Printf("Failed to create artist %s: %v", artistName, err)
				return nil
			}
			log.Printf("Added artist: %s", artistName)
		}

		// Get or create album
		albumPath := filepath.Join(artistPath, albumName)
		album, err := s.db.GetAlbumByPath(albumPath)
		if err != nil {
			albumYear := extractYearFromPath(albumPath)
			album = &database.Album{
				ArtistID: artist.ID,
				Title:    albumName,
				Year:     albumYear,
				Path:     albumPath,
			}
			if err := s.db.CreateAlbum(album); err != nil {
				log.Printf("Failed to create album %s: %v", albumName, err)
				return nil
			}
			log.Printf("Added album: %s by %s", albumName, artistName)
		}

		// Parse track info from filename
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		trackNum, title := parseTrackFilename(filename)

		track := &database.Track{
			AlbumID:     album.ID,
			Title:       title,
			TrackNumber: trackNum,
			DiscNumber:  1,
			Duration:    0, // Would need audio parsing library to get actual duration
			Path:        path,
			Size:        info.Size(),
		}

		if err := s.db.CreateTrack(track); err != nil {
			log.Printf("Failed to add track: %v", err)
		} else {
			log.Printf("Added track: %s", title)
		}

		return nil
	})
}

func (s *Scanner) scanBooks(lib *database.Library) error {
	return filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !bookExtensions[ext] {
			return nil
		}

		// Check if already in database
		if _, err := s.db.GetBookByPath(path); err == nil {
			return nil
		}

		// Parse filename for title and author
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		title, author := parseBookFilename(filename)

		// Determine format from extension
		format := strings.TrimPrefix(ext, ".")

		book := &database.Book{
			LibraryID: lib.ID,
			Title:     title,
			Author:    &author,
			Format:    format,
			Path:      path,
			Size:      info.Size(),
		}

		if err := s.db.CreateBook(book); err != nil {
			log.Printf("Failed to add book: %v", err)
		} else {
			log.Printf("Added book: %s by %s", title, author)
		}

		return nil
	})
}

func parseTrackFilename(filename string) (trackNum int, title string) {
	// Common patterns: "01 - Track Title", "01. Track Title", "01 Track Title"
	trackPattern := regexp.MustCompile(`^(\d{1,3})[\s\.\-_]+(.+)$`)
	matches := trackPattern.FindStringSubmatch(filename)
	if len(matches) >= 3 {
		trackNum, _ = strconv.Atoi(matches[1])
		title = cleanTitle(matches[2])
		return
	}
	return 1, cleanTitle(filename)
}

func parseBookFilename(filename string) (title, author string) {
	// Common patterns: "Author - Title", "Title - Author", "Title (Author)"
	dashPattern := regexp.MustCompile(`^(.+?)\s*-\s*(.+)$`)
	parenPattern := regexp.MustCompile(`^(.+?)\s*\((.+?)\)$`)

	// Try "Author - Title" pattern
	matches := dashPattern.FindStringSubmatch(filename)
	if len(matches) >= 3 {
		// Heuristic: if first part looks like a name, it's probably the author
		author = cleanTitle(matches[1])
		title = cleanTitle(matches[2])
		return
	}

	// Try "Title (Author)" pattern
	matches = parenPattern.FindStringSubmatch(filename)
	if len(matches) >= 3 {
		title = cleanTitle(matches[1])
		author = cleanTitle(matches[2])
		return
	}

	// No pattern matched, just use filename as title
	return cleanTitle(filename), ""
}

func parseMovieFilename(filename string) (title string, year int) {
	// Clean up common release info
	cleaned := cleanFilename(filename)

	matches := movieYearPattern.FindStringSubmatch(cleaned)
	if len(matches) >= 3 {
		title = cleanTitle(matches[1])
		year, _ = strconv.Atoi(matches[2])
		return
	}

	// No year found, just use the cleaned filename as title
	return cleanTitle(cleaned), 0
}

func parseTVFilename(filename string) (showTitle string, season, episode int) {
	cleaned := cleanFilename(filename)

	// Try S01E02 pattern first
	matches := tvPattern.FindStringSubmatch(cleaned)
	if len(matches) >= 4 {
		showTitle = cleanTitle(matches[1])
		season, _ = strconv.Atoi(matches[2])
		episode, _ = strconv.Atoi(matches[3])
		return
	}

	// Try 1x02 pattern
	matches = tvAltPattern.FindStringSubmatch(cleaned)
	if len(matches) >= 4 {
		showTitle = cleanTitle(matches[1])
		season, _ = strconv.Atoi(matches[2])
		episode, _ = strconv.Atoi(matches[3])
		return
	}

	return "", 0, 0
}

func cleanFilename(filename string) string {
	// Remove common release tags
	patterns := []string{
		`(?i)\.?(1080p|720p|480p|2160p|4k)`,
		`(?i)\.?(bluray|bdrip|brrip|webrip|web-dl|hdtv|dvdrip)`,
		`(?i)\.?(x264|x265|hevc|h\.?264|h\.?265)`,
		`(?i)\.?(aac|ac3|dts|atmos)`,
		`(?i)\.?(proper|repack|internal)`,
		`(?i)\[.*?\]`,
	}

	result := filename
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return strings.TrimSpace(result)
}

func cleanTitle(title string) string {
	// Replace dots and underscores with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Collapse multiple spaces
	re := regexp.MustCompile(`\s+`)
	title = re.ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}

func extractYearFromPath(path string) int {
	yearPattern := regexp.MustCompile(`\((\d{4})\)`)
	matches := yearPattern.FindStringSubmatch(filepath.Base(path))
	if len(matches) >= 2 {
		year, _ := strconv.Atoi(matches[1])
		return year
	}
	return 0
}
