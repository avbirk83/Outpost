package scanner

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

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
	db       *database.Database
	meta     *metadata.Service
	cacheDir string

	// Progress tracking
	scanning     bool
	scanLibrary  string
	scanTotal    int
	scanCurrent  int
	scanPhase    string // "counting", "scanning", "extracting"
	mu           sync.RWMutex

	// Result tracking (persists after scan completes)
	lastLibrary string
	lastAdded   int
	lastSkipped int
	lastErrors  int
	lastScanAt  time.Time
}

type ScanProgress struct {
	Scanning    bool   `json:"scanning"`
	Library     string `json:"library"`
	Phase       string `json:"phase"`
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	Percent     int    `json:"percent"`
	// Result of last scan
	LastLibrary string `json:"lastLibrary,omitempty"`
	LastAdded   int    `json:"lastAdded"`
	LastSkipped int    `json:"lastSkipped"`
	LastErrors  int    `json:"lastErrors"`
	LastScanAt  string `json:"lastScanAt,omitempty"`
}

func New(db *database.Database, meta *metadata.Service, cacheDir string) *Scanner {
	// Create subtitle cache directory
	subtitleDir := filepath.Join(cacheDir, "subtitles")
	os.MkdirAll(subtitleDir, 0755)

	s := &Scanner{db: db, meta: meta, cacheDir: cacheDir}

	// Fix any episodes/movies with missing sizes
	go s.FixMissingSizes()

	return s
}

// FixMissingSizes updates file sizes for any episodes that have size=0
func (s *Scanner) FixMissingSizes() {
	episodes, err := s.db.GetEpisodesWithMissingSize()
	if err != nil {
		log.Printf("Failed to get episodes with missing sizes: %v", err)
		return
	}

	if len(episodes) == 0 {
		return
	}

	log.Printf("Fixing file sizes for %d episodes...", len(episodes))
	fixed := 0
	for _, ep := range episodes {
		info, err := os.Stat(ep.Path)
		if err != nil {
			continue
		}
		if err := s.db.UpdateEpisodeSize(ep.ID, info.Size()); err == nil {
			fixed++
		}
	}
	if fixed > 0 {
		log.Printf("Fixed file sizes for %d episodes", fixed)
	}
}

// cleanupOrphanedMovies removes movies from database whose files no longer exist
func (s *Scanner) cleanupOrphanedMovies(libraryID int64) {
	movies, err := s.db.GetMoviesByLibrary(libraryID)
	if err != nil {
		log.Printf("Failed to get movies for cleanup: %v", err)
		return
	}

	removed := 0
	for _, movie := range movies {
		if movie.Path == "" {
			continue
		}
		if _, err := os.Stat(movie.Path); os.IsNotExist(err) {
			if err := s.db.DeleteMovie(movie.ID); err == nil {
				removed++
				log.Printf("Removed orphaned movie: %s (file no longer exists)", movie.Title)
			}
		}
	}
	if removed > 0 {
		log.Printf("Cleaned up %d orphaned movies", removed)
	}
}

// cleanupOrphanedEpisodes removes episodes from database whose files no longer exist
func (s *Scanner) cleanupOrphanedEpisodes(libraryID int64) {
	episodes, err := s.db.GetEpisodesByLibrary(libraryID)
	if err != nil {
		log.Printf("Failed to get episodes for cleanup: %v", err)
		return
	}

	removed := 0
	for _, ep := range episodes {
		if ep.Path == "" {
			continue
		}
		if _, err := os.Stat(ep.Path); os.IsNotExist(err) {
			if err := s.db.DeleteEpisode(ep.ID); err == nil {
				removed++
				log.Printf("Removed orphaned episode: E%02d (file no longer exists)", ep.EpisodeNumber)
			}
		}
	}
	if removed > 0 {
		log.Printf("Cleaned up %d orphaned episodes", removed)
	}
}

func (s *Scanner) GetProgress() ScanProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	percent := 0
	if s.scanTotal > 0 {
		percent = (s.scanCurrent * 100) / s.scanTotal
	}

	lastScanAt := ""
	if !s.lastScanAt.IsZero() {
		lastScanAt = s.lastScanAt.Format(time.RFC3339)
	}

	return ScanProgress{
		Scanning:    s.scanning,
		Library:     s.scanLibrary,
		Phase:       s.scanPhase,
		Current:     s.scanCurrent,
		Total:       s.scanTotal,
		Percent:     percent,
		LastLibrary: s.lastLibrary,
		LastAdded:   s.lastAdded,
		LastSkipped: s.lastSkipped,
		LastErrors:  s.lastErrors,
		LastScanAt:  lastScanAt,
	}
}

func (s *Scanner) setProgress(library, phase string, current, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scanning = true
	s.scanLibrary = library
	s.scanPhase = phase
	s.scanCurrent = current
	s.scanTotal = total
}

func (s *Scanner) clearProgress() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scanning = false
	s.scanLibrary = ""
	s.scanPhase = ""
	s.scanCurrent = 0
	s.scanTotal = 0
}

func (s *Scanner) setResult(library string, added, skipped, errors int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastLibrary = library
	s.lastAdded = added
	s.lastSkipped = skipped
	s.lastErrors = errors
	s.lastScanAt = time.Now()
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
	defer s.clearProgress()

	var added, skipped, errors int

	// Phase 0: Clean up orphaned entries (files that no longer exist)
	s.cleanupOrphanedMovies(lib.ID)

	// Phase 1: Count video files
	s.setProgress(lib.Name, "counting", 0, 0)
	var videoFiles []string
	filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			videoFiles = append(videoFiles, path)
		}
		return nil
	})

	total := len(videoFiles)
	log.Printf("Found %d video files in %s", total, lib.Name)

	// Phase 2: Process each file
	for i, path := range videoFiles {
		s.setProgress(lib.Name, "scanning", i+1, total)

		info, err := os.Stat(path)
		if err != nil {
			errors++
			continue
		}

		// Check if already in database
		if _, err := s.db.GetMovieByPath(path); err == nil {
			skipped++
			continue // Already exists
		}

		// Parse filename
		ext := filepath.Ext(path)
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
			errors++
		} else {
			added++
			log.Printf("Added movie: %s (%d)", title, year)
			// Fetch metadata from TMDB
			if s.meta != nil {
				if err := s.meta.FetchMovieMetadata(movie); err != nil {
					log.Printf("Failed to fetch metadata for %s: %v", title, err)
				}
			}
			// Organize folder and extract subtitles in background
			go s.OrganizeAndExtractSubtitles(movie, lib.Path)
		}
	}

	s.setResult(lib.Name, added, skipped, errors)
	return nil
}

func (s *Scanner) scanTV(lib *database.Library) error {
	defer s.clearProgress()

	var added, skipped, errors int

	// Phase 0: Clean up orphaned entries (files that no longer exist)
	s.cleanupOrphanedEpisodes(lib.ID)

	// Phase 1: Count video files
	s.setProgress(lib.Name, "counting", 0, 0)
	var videoFiles []string
	filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			videoFiles = append(videoFiles, path)
		}
		return nil
	})

	total := len(videoFiles)
	log.Printf("Found %d video files in %s", total, lib.Name)

	// Phase 2: Process each file
	for i, path := range videoFiles {
		s.setProgress(lib.Name, "scanning", i+1, total)

		info, err := os.Stat(path)
		if err != nil {
			errors++
			continue
		}

		// Check if already in database
		if _, err := s.db.GetEpisodeByPath(path); err == nil {
			skipped++
			continue
		}

		// Parse filename
		ext := filepath.Ext(path)
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		showTitle, seasonNum, episodeNum := parseTVFilename(filename)

		if showTitle == "" || seasonNum == 0 {
			log.Printf("Could not parse TV filename: %s", filename)
			errors++
			continue
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
				errors++
				continue
			}
			log.Printf("Added show: %s", showTitle)
			isNewShow = true
		} else if err != nil {
			errors++
			continue
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
				errors++
				continue
			}
		} else if err != nil {
			errors++
			continue
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
			errors++
		} else {
			added++
			log.Printf("Added episode: %s S%02dE%02d", showTitle, seasonNum, episodeNum)
			// Extract subtitles in background
			go s.ExtractSubtitles(path)
		}

		// Fetch show metadata if this is a new show
		if isNewShow && s.meta != nil {
			if err := s.meta.FetchShowMetadata(show); err != nil {
				log.Printf("Failed to fetch metadata for %s: %v", showTitle, err)
			}
		}
	}

	s.setResult(lib.Name, added, skipped, errors)
	return nil
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

// OrganizeAndExtractSubtitles renames folder to proper format and extracts subtitles
func (s *Scanner) OrganizeAndExtractSubtitles(movie *database.Movie, libraryPath string) {
	videoPath := movie.Path
	videoDir := filepath.Dir(videoPath)
	videoFile := filepath.Base(videoPath)
	ext := filepath.Ext(videoFile)

	// Build expected folder name: "Title (Year)"
	expectedFolder := movie.Title
	if movie.Year > 0 {
		expectedFolder = fmt.Sprintf("%s (%d)", movie.Title, movie.Year)
	}
	// Clean folder name of invalid characters
	expectedFolder = cleanFolderName(expectedFolder)

	expectedPath := filepath.Join(libraryPath, expectedFolder)
	currentFolder := filepath.Base(videoDir)

	// Case 1: Video is directly in library root - create folder and move it
	if videoDir == libraryPath {
		// Check if expected folder already exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			log.Printf("Creating folder for movie at library root: %s", expectedFolder)
			if err := os.MkdirAll(expectedPath, 0755); err != nil {
				log.Printf("Failed to create folder: %v", err)
			} else {
				// Move video file into the new folder
				expectedVideoName := expectedFolder + ext
				newVideoPath := filepath.Join(expectedPath, expectedVideoName)
				if err := os.Rename(videoPath, newVideoPath); err != nil {
					log.Printf("Failed to move video file: %v", err)
				} else {
					log.Printf("Moved video to: %s", newVideoPath)
					movie.Path = newVideoPath
					videoPath = newVideoPath
					videoDir = expectedPath

					// Update path in database
					if err := s.db.UpdateMoviePath(movie.ID, newVideoPath); err != nil {
						log.Printf("Failed to update movie path in database: %v", err)
					}
				}
			}
		}
	} else if currentFolder != expectedFolder {
		// Case 2: Video is in a folder but folder name doesn't match - rename folder
		// Check if expected folder already exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			log.Printf("Renaming folder: %s -> %s", currentFolder, expectedFolder)
			if err := os.Rename(videoDir, expectedPath); err != nil {
				log.Printf("Failed to rename folder: %v", err)
			} else {
				// Update video path
				newVideoPath := filepath.Join(expectedPath, videoFile)
				movie.Path = newVideoPath
				videoPath = newVideoPath
				videoDir = expectedPath

				// Update path in database
				if err := s.db.UpdateMoviePath(movie.ID, newVideoPath); err != nil {
					log.Printf("Failed to update movie path in database: %v", err)
				}

				// Rename video file to match folder
				expectedVideoName := expectedFolder + ext
				if videoFile != expectedVideoName {
					finalVideoPath := filepath.Join(expectedPath, expectedVideoName)
					if err := os.Rename(newVideoPath, finalVideoPath); err != nil {
						log.Printf("Failed to rename video file: %v", err)
					} else {
						movie.Path = finalVideoPath
						videoPath = finalVideoPath
						if err := s.db.UpdateMoviePath(movie.ID, finalVideoPath); err != nil {
							log.Printf("Failed to update movie path in database: %v", err)
						}
					}
				}
			}
		}
	} else {
		// Case 3: Folder name matches but video file might not - rename video file only
		expectedVideoName := expectedFolder + ext
		if videoFile != expectedVideoName {
			finalVideoPath := filepath.Join(videoDir, expectedVideoName)
			// Check if target doesn't already exist
			if _, err := os.Stat(finalVideoPath); os.IsNotExist(err) {
				log.Printf("Renaming video file: %s -> %s", videoFile, expectedVideoName)
				if err := os.Rename(videoPath, finalVideoPath); err != nil {
					log.Printf("Failed to rename video file: %v", err)
				} else {
					movie.Path = finalVideoPath
					videoPath = finalVideoPath
					if err := s.db.UpdateMoviePath(movie.ID, finalVideoPath); err != nil {
						log.Printf("Failed to update movie path in database: %v", err)
					}
				}
			}
		}
	}

	// Create subtitles subfolder
	subtitleDir := filepath.Join(filepath.Dir(videoPath), "subtitles")
	if err := os.MkdirAll(subtitleDir, 0755); err != nil {
		log.Printf("Failed to create subtitles directory: %v", err)
		return
	}

	// Extract subtitles to the subfolder
	s.extractSubtitlesToDir(videoPath, subtitleDir)
}

// cleanFolderName removes characters that are invalid in folder names
func cleanFolderName(name string) string {
	// Remove characters invalid on Windows/Linux
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "")
	}
	return strings.TrimSpace(result)
}

// extractSubtitlesToDir extracts all subtitle tracks to a specific directory
func (s *Scanner) extractSubtitlesToDir(videoPath, subtitleDir string) {
	baseName := filepath.Base(videoPath)
	baseNameNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Get subtitle track info using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to probe subtitles for %s: %v", baseName, err)
		return
	}

	var probeResult struct {
		Streams []struct {
			Index int               `json:"index"`
			Tags  map[string]string `json:"tags"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &probeResult); err != nil {
		log.Printf("Failed to parse ffprobe output for %s: %v", baseName, err)
		return
	}

	if len(probeResult.Streams) == 0 {
		log.Printf("No subtitles found in %s", baseName)
		return
	}

	log.Printf("Extracting %d subtitle tracks from %s", len(probeResult.Streams), baseName)

	// Extract each subtitle track
	for i, stream := range probeResult.Streams {
		// Get language tag if available
		lang := "und"
		if l, ok := stream.Tags["language"]; ok && l != "" {
			lang = l
		}

		subtitleFile := filepath.Join(subtitleDir, fmt.Sprintf("%s.%d.%s.vtt", baseNameNoExt, i, lang))

		// Skip if already extracted
		if _, err := os.Stat(subtitleFile); err == nil {
			continue
		}

		// Extract subtitle using ffmpeg
		cmd := exec.Command("ffmpeg",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:s:%d", i),
			"-an", "-vn",
			"-c:s", "webvtt",
			"-f", "webvtt",
			subtitleFile,
			"-y",
		)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to extract subtitle track %d from %s: %v", i, baseName, err)
			continue
		}
		log.Printf("Extracted subtitle: %s", filepath.Base(subtitleFile))
	}

	log.Printf("Finished extracting subtitles from %s", baseName)
}

// ExtractSubtitles extracts all subtitle tracks from a video file to the cache directory
func (s *Scanner) ExtractSubtitles(videoPath string) {
	if s.cacheDir == "" {
		return
	}

	subtitleDir := filepath.Join(s.cacheDir, "subtitles")
	baseName := filepath.Base(videoPath)

	// Get subtitle track count using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to probe subtitles for %s: %v", baseName, err)
		return
	}

	var probeResult struct {
		Streams []struct {
			Index int `json:"index"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &probeResult); err != nil {
		log.Printf("Failed to parse ffprobe output for %s: %v", baseName, err)
		return
	}

	if len(probeResult.Streams) == 0 {
		log.Printf("No subtitles found in %s", baseName)
		return
	}

	log.Printf("Extracting %d subtitle tracks from %s in background", len(probeResult.Streams), baseName)

	// Extract each subtitle track
	for i := range probeResult.Streams {
		cacheFile := filepath.Join(subtitleDir, fmt.Sprintf("%s.track%d.vtt", baseName, i))

		// Skip if already extracted
		if _, err := os.Stat(cacheFile); err == nil {
			continue
		}

		// Extract subtitle using ffmpeg
		cmd := exec.Command("ffmpeg",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:s:%d", i),
			"-an", "-vn",
			"-c:s", "webvtt",
			"-f", "webvtt",
			cacheFile,
			"-y",
		)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to extract subtitle track %d from %s: %v", i, baseName, err)
			continue
		}
		log.Printf("Extracted subtitle track %d from %s", i, baseName)
	}

	log.Printf("Finished extracting subtitles from %s", baseName)
}
