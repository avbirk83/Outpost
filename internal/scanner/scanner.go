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
	"github.com/outpost/outpost/internal/parser"
	"github.com/outpost/outpost/internal/quality"
	"github.com/outpost/outpost/internal/subtitles"
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

// Multi-episode patterns
// Examples: "Show.S01E01E02.mkv", "Show.S01E01-E02.mkv", "Show.S01E01-02.mkv"
var multiEpisodePattern = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})[-E]?E?(\d{1,3})`)
var multiEpisodeAltPattern = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})-(\d{1,3})`)

// Anime absolute episode patterns
// Examples: "[SubGroup] Show Name - 01 [1080p].mkv", "Show Name - 01v2.mkv"
var animeAbsolutePattern = regexp.MustCompile(`(?i)^(?:\[.+?\]\s*)?(.+?)\s*-\s*(\d{2,4})(?:\s*v\d+)?`)
var animeAbsoluteAltPattern = regexp.MustCompile(`(?i)^(.+?)\s+(\d{2,4})(?:\s*v\d+)?(?:\s*[\[\(])`)

// Folder structure patterns
// Examples: "Show Name (2020)", "Show Name"
var showFolderPattern = regexp.MustCompile(`(?i)^(.+?)\s*\((\d{4})\)$`)
var seasonFolderPattern = regexp.MustCompile(`(?i)^season\s*(\d+)$`)

// ParseResult contains parsed information from filename/path
type ParseResult struct {
	Title       string
	Year        int
	Season      int
	Episode     int
	EpisodeEnd  int     // For multi-episode files (S01E01-E03)
	Absolute    int     // For anime absolute numbering
	Confidence  float64 // 0.0 - 1.0
	Source      string  // "folder", "filename", "guess"
}

// ShowParseResult contains parsed show info from folder name
type ShowParseResult struct {
	Title      string
	Year       int
	Confidence float64
}

// Low confidence threshold - below this requires manual review
const lowConfidenceThreshold = 0.6

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

// detectQualityFromFile uses ffprobe to detect video resolution and estimates source from file size
func (s *Scanner) detectQualityFromFile(filePath string) (resolution string, source string) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "json", filePath)

	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil || len(result.Streams) == 0 {
		return "", ""
	}

	height := result.Streams[0].Height
	switch {
	case height >= 2000:
		resolution = "2160p"
	case height >= 1000:
		resolution = "1080p"
	case height >= 700:
		resolution = "720p"
	case height >= 400:
		resolution = "480p"
	default:
		return "", ""
	}

	// Estimate source from file size
	if info, err := os.Stat(filePath); err == nil {
		sizeGB := float64(info.Size()) / (1024 * 1024 * 1024)
		if resolution == "2160p" {
			if sizeGB > 40 {
				source = "remux"
			} else if sizeGB > 15 {
				source = "bluray"
			} else {
				source = "webdl"
			}
		} else if resolution == "1080p" {
			if sizeGB > 20 {
				source = "remux"
			} else if sizeGB > 8 {
				source = "bluray"
			} else {
				source = "webdl"
			}
		} else {
			source = "webdl"
		}
	}

	return resolution, source
}

// detectAndStoreQuality parses a filename to detect quality and stores it in media_quality_status
func (s *Scanner) detectAndStoreQuality(mediaID int64, mediaType string, filename string, filePath string) {
	// Parse the filename to extract quality information
	parsed := parser.Parse(filename)
	if parsed == nil {
		parsed = &parser.ParsedRelease{}
	}

	// Compute quality tier and base score
	qualityTier := quality.ComputeQualityTier(parsed)
	baseScore := quality.BaseQualityScores[qualityTier]
	if baseScore == 0 {
		baseScore = 1000 // Unknown quality
	}

	// If filename parsing returned Unknown or missing resolution/source, try ffprobe
	if filePath != "" && (qualityTier == "Unknown" || parsed.Resolution == "" || parsed.Source == "") {
		if detectedRes, detectedSource := s.detectQualityFromFile(filePath); detectedRes != "" {
			if parsed.Resolution == "" {
				parsed.Resolution = detectedRes
			}
			if parsed.Source == "" && detectedSource != "" {
				parsed.Source = detectedSource
			}
			// Recompute quality tier with detected values
			qualityTier = quality.ComputeQualityTier(parsed)
			if newScore, ok := quality.BaseQualityScores[qualityTier]; ok && newScore > 0 {
				baseScore = newScore
			}
			log.Printf("Detected quality from file %s: %s %s (score: %d)", filename, parsed.Resolution, parsed.Source, baseScore)
		}
	}

	// Default cutoff to current quality (no upgrades needed if no preset configured)
	cutoffScore := baseScore

	// Try to get the default preset's cutoff
	preset, err := s.db.GetDefaultQualityPreset()
	if err == nil && preset != nil {
		// Use cutoff settings if available, otherwise fall back to target settings
		cutoffRes := preset.CutoffResolution
		cutoffSrc := preset.CutoffSource
		if cutoffRes == "" {
			cutoffRes = preset.Resolution
		}
		if cutoffSrc == "" {
			cutoffSrc = preset.Source
		}

		// Map preset format to parser format
		presetResolution := mapPresetResolution(cutoffRes)
		presetSource := mapPresetSource(cutoffSrc)

		// Calculate cutoff score from preset's cutoff resolution and source
		cutoffTier := quality.ComputeQualityTier(&parser.ParsedRelease{
			Resolution: presetResolution,
			Source:     presetSource,
		})
		if score, ok := quality.BaseQualityScores[cutoffTier]; ok && score > 0 {
			cutoffScore = score
		}
	}

	// Determine if target is met (current quality meets or exceeds cutoff)
	targetMet := baseScore >= cutoffScore

	// Prepare quality status record
	resolution := parsed.Resolution
	source := parsed.Source
	hdr := parsed.HDR
	audio := parsed.AudioFormat
	edition := parsed.Edition

	status := &database.MediaQualityStatus{
		MediaID:           mediaID,
		MediaType:         mediaType,
		CurrentResolution: &resolution,
		CurrentSource:     &source,
		CurrentHDR:        &hdr,
		CurrentAudio:      &audio,
		CurrentEdition:    &edition,
		CurrentScore:      baseScore,
		CutoffScore:       cutoffScore,
		TargetMet:         targetMet,
		UpgradeAvailable:  false,
	}

	if err := s.db.UpsertMediaQualityStatus(status); err != nil {
		log.Printf("Failed to store quality status for %s %d: %v", mediaType, mediaID, err)
	}
}

// RescanQualityStatus re-scans all media to update quality status
// This is useful after changing quality presets or fixing detection logic
func (s *Scanner) RescanQualityStatus() (int, int, error) {
	var moviesUpdated, episodesUpdated int

	// Get all movies
	movies, err := s.db.GetMovies()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get movies: %w", err)
	}

	for _, movie := range movies {
		if movie.Path != "" {
			s.detectAndStoreQuality(movie.ID, "movie", filepath.Base(movie.Path), movie.Path)
			moviesUpdated++
		}
	}

	// Get all episodes
	episodes, err := s.db.GetAllEpisodes()
	if err != nil {
		return moviesUpdated, 0, fmt.Errorf("failed to get episodes: %w", err)
	}

	for _, ep := range episodes {
		if ep.Path != "" {
			s.detectAndStoreQuality(ep.ID, "episode", filepath.Base(ep.Path), ep.Path)
			episodesUpdated++
		}
	}

	log.Printf("Rescanned quality status: %d movies, %d episodes", moviesUpdated, episodesUpdated)
	return moviesUpdated, episodesUpdated, nil
}

// mapPresetResolution maps preset resolution format ("4k") to parser format ("2160p")
func mapPresetResolution(presetRes string) string {
	switch strings.ToLower(presetRes) {
	case "4k", "2160p", "uhd":
		return "2160p"
	case "1080p", "fhd":
		return "1080p"
	case "720p", "hd":
		return "720p"
	case "480p", "sd":
		return "480p"
	default:
		return "1080p" // Default
	}
}

// mapPresetSource maps preset source format ("web") to parser format ("webdl")
func mapPresetSource(presetSource string) string {
	switch strings.ToLower(presetSource) {
	case "remux":
		return "remux"
	case "bluray", "blu-ray":
		return "bluray"
	case "web", "webdl", "web-dl":
		return "webdl"
	case "webrip", "web-rip":
		return "webrip"
	case "hdtv":
		return "hdtv"
	case "dvd":
		return "dvd"
	default:
		return "webdl" // Default
	}
}

// DetectQualityForExistingMedia scans all existing movies and episodes to detect and store their quality
// This is useful for media that was added before quality detection was implemented
func (s *Scanner) DetectQualityForExistingMedia() {
	log.Println("Starting quality detection for existing media...")

	// Process all movies
	movies, err := s.db.GetMovies()
	if err != nil {
		log.Printf("Failed to get movies for quality detection: %v", err)
	} else {
		detected := 0
		for _, movie := range movies {
			// Check if quality status already exists
			status, _ := s.db.GetMediaQualityStatus(movie.ID, "movie")
			if status != nil && status.CurrentScore > 0 {
				continue // Already has quality info
			}

			if movie.Path != "" {
				s.detectAndStoreQuality(movie.ID, "movie", filepath.Base(movie.Path), movie.Path)
				detected++
			}
		}
		if detected > 0 {
			log.Printf("Detected quality for %d movies", detected)
		}
	}

	// Process all episodes
	episodes, err := s.db.GetAllEpisodes()
	if err != nil {
		log.Printf("Failed to get episodes for quality detection: %v", err)
	} else {
		detected := 0
		for _, ep := range episodes {
			// Check if quality status already exists
			status, _ := s.db.GetMediaQualityStatus(ep.ID, "episode")
			if status != nil && status.CurrentScore > 0 {
				continue // Already has quality info
			}

			if ep.Path != "" {
				s.detectAndStoreQuality(ep.ID, "episode", filepath.Base(ep.Path), ep.Path)
				detected++
			}
		}
		if detected > 0 {
			log.Printf("Detected quality for %d episodes", detected)
		}
	}

	log.Println("Quality detection for existing media complete")
}

// RedetectAllQuality forces re-detection of quality for ALL media using ffprobe
func (s *Scanner) RedetectAllQuality() {
	log.Println("Starting full quality re-detection using ffprobe...")

	// Process all movies
	movies, err := s.db.GetMovies()
	if err != nil {
		log.Printf("Failed to get movies for quality re-detection: %v", err)
	} else {
		for _, movie := range movies {
			if movie.Path != "" {
				s.detectAndStoreQuality(movie.ID, "movie", filepath.Base(movie.Path), movie.Path)
			}
		}
		log.Printf("Re-detected quality for %d movies", len(movies))
	}

	// Process all episodes
	episodes, err := s.db.GetAllEpisodes()
	if err != nil {
		log.Printf("Failed to get episodes for quality re-detection: %v", err)
	} else {
		for _, ep := range episodes {
			if ep.Path != "" {
				s.detectAndStoreQuality(ep.ID, "episode", filepath.Base(ep.Path), ep.Path)
			}
		}
		log.Printf("Re-detected quality for %d episodes", len(episodes))
	}

	log.Println("Quality re-detection complete")
}

// missingGracePeriod is how long a file can be missing before being deleted
const missingGracePeriod = 24 * time.Hour

// cleanupOrphanedMovies marks movies as missing and deletes after grace period
func (s *Scanner) cleanupOrphanedMovies(libraryID int64) {
	movies, err := s.db.GetMoviesByLibrary(libraryID)
	if err != nil {
		log.Printf("Failed to get movies for cleanup: %v", err)
		return
	}

	marked, cleared := 0, 0
	for _, movie := range movies {
		if movie.Path == "" {
			continue
		}
		_, statErr := os.Stat(movie.Path)
		fileExists := statErr == nil

		if !fileExists && os.IsNotExist(statErr) {
			// File is missing - mark it (if not already marked)
			if err := s.db.MarkMovieMissing(movie.ID); err == nil {
				marked++
				log.Printf("Marked movie as missing: %s", movie.Title)
			}
		} else if fileExists && movie.MissingSince != nil {
			// File reappeared - clear missing status
			if err := s.db.ClearMovieMissing(movie.ID); err == nil {
				cleared++
				log.Printf("Movie file reappeared: %s", movie.Title)
			}
		}
	}

	// Delete movies that have been missing for longer than grace period
	deleted, err := s.db.DeleteMissingMovies(missingGracePeriod)
	if err != nil {
		log.Printf("Failed to delete missing movies: %v", err)
	}

	if marked > 0 || cleared > 0 || deleted > 0 {
		log.Printf("Movie cleanup: %d marked missing, %d reappeared, %d deleted", marked, cleared, deleted)
	}
}

// cleanupOrphanedEpisodes marks episodes as missing and deletes after grace period
func (s *Scanner) cleanupOrphanedEpisodes(libraryID int64) {
	episodes, err := s.db.GetEpisodesByLibrary(libraryID)
	if err != nil {
		log.Printf("Failed to get episodes for cleanup: %v", err)
		return
	}

	marked, cleared := 0, 0
	for _, ep := range episodes {
		if ep.Path == "" {
			continue
		}
		_, statErr := os.Stat(ep.Path)
		fileExists := statErr == nil

		if !fileExists && os.IsNotExist(statErr) {
			// File is missing - mark it
			if err := s.db.MarkEpisodeMissing(ep.ID); err == nil {
				marked++
				log.Printf("Marked episode as missing: E%02d", ep.EpisodeNumber)
			}
		} else if fileExists && ep.MissingSince != nil {
			// File reappeared - clear missing status
			if err := s.db.ClearEpisodeMissing(ep.ID); err == nil {
				cleared++
				log.Printf("Episode file reappeared: E%02d", ep.EpisodeNumber)
			}
		}
	}

	// Delete episodes that have been missing for longer than grace period
	deleted, err := s.db.DeleteMissingEpisodes(missingGracePeriod)
	if err != nil {
		log.Printf("Failed to delete missing episodes: %v", err)
	}

	if marked > 0 || cleared > 0 || deleted > 0 {
		log.Printf("Episode cleanup: %d marked missing, %d reappeared, %d deleted", marked, cleared, deleted)
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
			// Detect and store quality from filename
			s.detectAndStoreQuality(movie.ID, "movie", filepath.Base(path), path)
			// Fetch metadata from TMDB
			if s.meta != nil {
				if err := s.meta.FetchMovieMetadata(movie); err != nil {
					log.Printf("Failed to fetch metadata for %s: %v", title, err)
				}
			}
			// Organize folder, extract subtitles, extract chapters, and auto-download subtitles in background
			go func(m *database.Movie, libPath string) {
				s.OrganizeAndExtractSubtitles(m, libPath)
				s.ExtractChapters("movie", m.ID, m.Path)
				s.AutoDownloadSubtitles("movie", m.Path, m.Title, m.Year, 0, 0)
			}(movie, lib.Path)
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

	// Phase 1: Group files by show folder
	s.setProgress(lib.Name, "counting", 0, 0)
	showFiles := make(map[string][]string) // showFolder -> list of video files

	filepath.Walk(lib.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] {
			return nil
		}

		showFolder, _, _ := s.findShowFolder(path, lib.Path)
		if showFolder != "" {
			showFiles[showFolder] = append(showFiles[showFolder], path)
		}
		return nil
	})

	// Count total files
	total := 0
	for _, files := range showFiles {
		total += len(files)
	}
	log.Printf("Found %d video files in %d shows in %s", total, len(showFiles), lib.Name)

	// Phase 2: Process each show folder
	current := 0
	for showFolder, files := range showFiles {
		folderName := filepath.Base(showFolder)
		folderInfo := parseShowFolder(folderName)

		// Get or create show
		show, err := s.db.GetShowByPath(showFolder)
		isNewShow := false
		if err == sql.ErrNoRows {
			// Use folder info for show title/year, with confidence scoring
			showYear, yearConfidence := extractYearFromPathEnhanced(showFolder, lib.Path)
			confidence := folderInfo.Confidence
			if yearConfidence > 0 {
				confidence = (confidence + yearConfidence) / 2
			}
			needsReview := confidence < lowConfidenceThreshold

			show = &database.Show{
				LibraryID:        lib.ID,
				Title:            folderInfo.Title,
				Year:             showYear,
				Path:             showFolder,
				MatchConfidence:  confidence,
				NeedsMatchReview: needsReview,
			}
			if err := s.db.CreateShow(show); err != nil {
				log.Printf("Failed to create show %s: %v", folderInfo.Title, err)
				errors++
				continue
			}
			if needsReview {
				log.Printf("Added show (needs review): %s (confidence: %.2f)", folderInfo.Title, confidence)
			} else {
				log.Printf("Added show: %s", folderInfo.Title)
			}
			isNewShow = true
		} else if err != nil {
			errors++
			continue
		}

		// Process each episode file in this show
		for _, path := range files {
			current++
			s.setProgress(lib.Name, "scanning", current, total)

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

			// Parse filename with enhanced parser
			ext := filepath.Ext(path)
			filename := strings.TrimSuffix(filepath.Base(path), ext)
			parseResult := parseTVFilenameEnhanced(filename)

			// Fall back to original parser if enhanced parser fails
			if parseResult.Season == 0 && parseResult.Episode == 0 {
				title, sNum, eNum := parseTVFilename(filename)
				if sNum > 0 || eNum > 0 {
					parseResult.Title = title
					parseResult.Season = sNum
					parseResult.Episode = eNum
					parseResult.Confidence = 0.8
				}
			}

			// Also try to get season from folder structure
			_, folderSeason, _ := s.findShowFolder(path, lib.Path)
			if parseResult.Season == 0 && folderSeason > 0 {
				parseResult.Season = folderSeason
			}

			if parseResult.Season == 0 {
				log.Printf("Could not parse TV filename: %s", filename)
				errors++
				continue
			}

			// Get or create season
			season, err := s.db.GetSeason(show.ID, parseResult.Season)
			if err == sql.ErrNoRows {
				season = &database.Season{
					ShowID:       show.ID,
					SeasonNumber: parseResult.Season,
				}
				if err := s.db.CreateSeason(season); err != nil {
					log.Printf("Failed to create season %d: %v", parseResult.Season, err)
					errors++
					continue
				}
			} else if err != nil {
				errors++
				continue
			}

			// Create episode with multi-episode support
			episode := &database.Episode{
				SeasonID:        season.ID,
				EpisodeNumber:   parseResult.Episode,
				Title:           "",
				Path:            path,
				Size:            info.Size(),
				MatchConfidence: parseResult.Confidence,
			}

			// Set multi-episode end if applicable
			if parseResult.EpisodeEnd > parseResult.Episode {
				episode.EpisodeEnd = &parseResult.EpisodeEnd
			}

			// Set absolute number for anime
			if parseResult.Absolute > 0 {
				episode.AbsoluteNumber = &parseResult.Absolute
			}

			// Use enhanced create that includes new fields
			if err := s.db.CreateEpisodeWithExtras(episode); err != nil {
				log.Printf("Failed to add episode: %v", err)
				errors++
			} else {
				added++
				if parseResult.EpisodeEnd > 0 {
					log.Printf("Added multi-episode: %s S%02dE%02d-E%02d", folderInfo.Title, parseResult.Season, parseResult.Episode, parseResult.EpisodeEnd)
				} else if parseResult.Absolute > 0 {
					log.Printf("Added anime episode: %s - %d (S%02dE%02d)", folderInfo.Title, parseResult.Absolute, parseResult.Season, parseResult.Episode)
				} else {
					log.Printf("Added episode: %s S%02dE%02d", folderInfo.Title, parseResult.Season, parseResult.Episode)
				}
				// Detect and store quality from filename
				s.detectAndStoreQuality(episode.ID, "episode", filepath.Base(path), path)
				// Extract subtitles, chapters, and auto-download subtitles in background
				go func(ep *database.Episode, p string, showName string, sNum, eNum int) {
					s.ExtractSubtitles(p)
					s.ExtractChapters("episode", ep.ID, p)
					s.AutoDownloadSubtitles("episode", p, showName, 0, sNum, eNum)
				}(episode, path, folderInfo.Title, parseResult.Season, parseResult.Episode)
			}
		}

		// Fetch show metadata if this is a new show
		if isNewShow && s.meta != nil {
			if err := s.meta.FetchShowMetadata(show); err != nil {
				log.Printf("Failed to fetch metadata for %s: %v", folderInfo.Title, err)
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

// findShowFolder finds the show folder for a given episode path
// Returns: show folder path, detected season number (0 if not from folder), error
func (s *Scanner) findShowFolder(filePath, libraryPath string) (string, int, error) {
	relPath := strings.TrimPrefix(filePath, libraryPath)
	relPath = strings.TrimPrefix(relPath, string(os.PathSeparator))
	parts := strings.Split(relPath, string(os.PathSeparator))

	// Expected structures:
	// /Library/Show Name (2020)/Season 01/S01E01.mkv -> show=parts[0], season=1
	// /Library/Show Name (2020)/S01E01.mkv -> show=parts[0], season from filename
	// /Library/Show Name/Season 1/S01E01.mkv -> show=parts[0], season=1

	if len(parts) >= 2 {
		showFolder := parts[0]

		// Check if second part is a season folder
		if len(parts) >= 3 {
			if match := seasonFolderPattern.FindStringSubmatch(parts[1]); match != nil {
				season, _ := strconv.Atoi(match[1])
				return filepath.Join(libraryPath, showFolder), season, nil
			}
		}

		return filepath.Join(libraryPath, showFolder), 0, nil
	}

	return "", 0, fmt.Errorf("cannot determine show folder")
}

// parseShowFolder extracts show name and year from folder name
func parseShowFolder(folderName string) ShowParseResult {
	result := ShowParseResult{Confidence: 0.5}

	// Try "Show Name (2020)" format
	if match := showFolderPattern.FindStringSubmatch(folderName); match != nil {
		result.Title = strings.TrimSpace(match[1])
		result.Year, _ = strconv.Atoi(match[2])
		result.Confidence = 1.0
		return result
	}

	// Fall back to cleaning folder name
	result.Title = cleanTitle(folderName)
	result.Confidence = 0.5
	return result
}

// extractYearFromPathEnhanced tries to extract year with confidence scoring
func extractYearFromPathEnhanced(path, libraryPath string) (int, float64) {
	relPath := strings.TrimPrefix(path, libraryPath)
	relPath = strings.TrimPrefix(relPath, string(os.PathSeparator))
	parts := strings.Split(filepath.ToSlash(relPath), "/")

	// Check each folder level from root down
	for _, part := range parts {
		if match := showFolderPattern.FindStringSubmatch(part); match != nil {
			if year, err := strconv.Atoi(match[2]); err == nil && year >= 1900 && year <= 2100 {
				return year, 1.0 // High confidence from folder
			}
		}
	}

	// Fallback: check filename for year
	base := filepath.Base(path)
	if match := movieYearPattern.FindStringSubmatch(base); match != nil {
		if year, err := strconv.Atoi(match[2]); err == nil && year >= 1900 && year <= 2100 {
			return year, 0.7 // Medium confidence from filename
		}
	}

	return 0, 0.0
}

// parseTVFilenameEnhanced parses TV filename with multi-episode and anime support
func parseTVFilenameEnhanced(filename string) ParseResult {
	result := ParseResult{Confidence: 0.0, Source: "filename"}
	cleaned := cleanFilename(filename)

	// Try multi-episode pattern first: S01E01E02 or S01E01-E02
	if match := multiEpisodePattern.FindStringSubmatch(cleaned); match != nil {
		result.Season, _ = strconv.Atoi(match[1])
		result.Episode, _ = strconv.Atoi(match[2])
		result.EpisodeEnd, _ = strconv.Atoi(match[3])
		result.Title = extractTitleBeforePattern(cleaned, match[0])
		result.Confidence = 0.9
		return result
	}

	// Standard S01E01 pattern
	if match := tvPattern.FindStringSubmatch(cleaned); match != nil {
		result.Season, _ = strconv.Atoi(match[2])
		result.Episode, _ = strconv.Atoi(match[3])
		result.Title = strings.TrimSpace(match[1])
		result.Confidence = 0.9
		return result
	}

	// Alternative 1x01 pattern
	if match := tvAltPattern.FindStringSubmatch(cleaned); match != nil {
		result.Season, _ = strconv.Atoi(match[2])
		result.Episode, _ = strconv.Atoi(match[3])
		result.Title = strings.TrimSpace(match[1])
		result.Confidence = 0.8
		return result
	}

	// Anime absolute pattern: [Group] Show - 01
	if match := animeAbsolutePattern.FindStringSubmatch(cleaned); match != nil {
		result.Title = strings.TrimSpace(match[1])
		result.Absolute, _ = strconv.Atoi(match[2])
		result.Season = 1 // Default to season 1 for absolute numbering
		result.Episode = result.Absolute
		result.Confidence = 0.7
		result.Source = "anime"
		return result
	}

	// Alternative anime pattern
	if match := animeAbsoluteAltPattern.FindStringSubmatch(cleaned); match != nil {
		result.Title = strings.TrimSpace(match[1])
		result.Absolute, _ = strconv.Atoi(match[2])
		result.Season = 1
		result.Episode = result.Absolute
		result.Confidence = 0.6
		result.Source = "anime"
		return result
	}

	return result
}

// extractTitleBeforePattern extracts title before a matched pattern
func extractTitleBeforePattern(text, pattern string) string {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(pattern))
	if idx <= 0 {
		return ""
	}
	return cleanTitle(text[:idx])
}

// calculateMatchConfidence calculates confidence based on folder and filename info
func calculateMatchConfidence(folderInfo ShowParseResult, filenameInfo ParseResult) float64 {
	score := 0.5 // Base score

	// Folder structure provides title/year?
	if folderInfo.Confidence > 0.8 {
		score += 0.3
	}

	// Filename parsing succeeded?
	if filenameInfo.Confidence > 0.7 {
		score += 0.2
	}

	// Folder and filename titles match?
	if strings.EqualFold(
		strings.TrimSpace(folderInfo.Title),
		strings.TrimSpace(filenameInfo.Title),
	) {
		score += 0.2
	}

	// Year matches between folder and filename?
	if folderInfo.Year > 0 && filenameInfo.Year > 0 && folderInfo.Year == filenameInfo.Year {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
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

// ExtractChapters extracts chapter information from a video file and saves to database
func (s *Scanner) ExtractChapters(mediaType string, mediaID int64, videoPath string) {
	baseName := filepath.Base(videoPath)

	// Get chapter info using ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_chapters",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		// Not an error - many files don't have chapters
		return
	}

	var probeResult struct {
		Chapters []struct {
			ID        int               `json:"id"`
			TimeBase  string            `json:"time_base"`
			Start     int64             `json:"start"`
			StartTime string            `json:"start_time"`
			End       int64             `json:"end"`
			EndTime   string            `json:"end_time"`
			Tags      map[string]string `json:"tags"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(output, &probeResult); err != nil {
		log.Printf("Failed to parse ffprobe chapters output for %s: %v", baseName, err)
		return
	}

	if len(probeResult.Chapters) == 0 {
		return
	}

	log.Printf("Found %d chapters in %s", len(probeResult.Chapters), baseName)

	var chapters []database.Chapter
	for i, ch := range probeResult.Chapters {
		// Parse start/end times from string (in seconds)
		startTime, _ := strconv.ParseFloat(ch.StartTime, 64)
		endTime, _ := strconv.ParseFloat(ch.EndTime, 64)

		title := ""
		if t, ok := ch.Tags["title"]; ok {
			title = t
		}

		chapters = append(chapters, database.Chapter{
			MediaType:    mediaType,
			MediaID:      mediaID,
			ChapterIndex: i,
			Title:        title,
			StartTime:    startTime,
			EndTime:      endTime,
		})
	}

	if err := s.db.SaveChapters(mediaType, mediaID, chapters); err != nil {
		log.Printf("Failed to save chapters for %s: %v", baseName, err)
	} else {
		log.Printf("Saved %d chapters for %s", len(chapters), baseName)
	}
}

// AutoDownloadSubtitles downloads subtitles from OpenSubtitles if auto-download is enabled
func (s *Scanner) AutoDownloadSubtitles(mediaType string, videoPath string, title string, year int, season, episode int) {
	// Check if auto-download is enabled
	autoDownload, err := s.db.GetSetting("opensubtitles_auto_download")
	if err != nil || autoDownload != "true" {
		return
	}

	// Get API key
	apiKey, err := s.db.GetSetting("opensubtitles_api_key")
	if err != nil || apiKey == "" {
		return
	}

	// Get preferred languages
	langSetting, _ := s.db.GetSetting("opensubtitles_languages")
	if langSetting == "" {
		langSetting = "en"
	}
	languages := strings.Split(langSetting, ",")

	// Get hearing impaired preference
	hiSetting, _ := s.db.GetSetting("opensubtitles_hearing_impaired")
	var hearingImpaired *bool
	if hiSetting == "only" {
		hi := true
		hearingImpaired = &hi
	} else if hiSetting == "exclude" {
		hi := false
		hearingImpaired = &hi
	}

	client := subtitles.NewClient(apiKey)

	// Try each language
	for _, lang := range languages {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			continue
		}

		// Check if subtitle already exists for this language
		videoBase := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
		subPath := videoBase + "." + lang + ".srt"
		if _, err := os.Stat(subPath); err == nil {
			log.Printf("Subtitle already exists for %s (%s), skipping", title, lang)
			continue
		}

		var downloadErr error
		if mediaType == "movie" {
			_, downloadErr = client.SearchAndDownload(videoPath, title, year, lang, hearingImpaired)
		} else if mediaType == "episode" {
			_, downloadErr = client.SearchAndDownloadEpisode(videoPath, title, season, episode, lang, hearingImpaired)
		}

		if downloadErr != nil {
			log.Printf("Failed to auto-download %s subtitles for %s: %v", lang, title, downloadErr)
		} else {
			log.Printf("Auto-downloaded %s subtitles for %s", lang, title)
		}
	}
}
