package importer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/parser"
)

// Manager handles file imports and organization
type Manager struct {
	db       *database.Database
	scanDir  string
}

// NewManager creates a new import manager
func NewManager(db *database.Database) *Manager {
	return &Manager{
		db: db,
	}
}

// ProcessImport processes a completed download
func (m *Manager) ProcessImport(download *database.Download, sourcePath string) error {
	// Update status
	download.Status = "importing"
	if err := m.db.UpdateDownload(download); err != nil {
		return err
	}

	// Find video files
	files, err := m.findVideoFiles(sourcePath)
	if err != nil {
		return m.failImport(download, fmt.Errorf("failed to find video files: %w", err))
	}

	if len(files) == 0 {
		return m.failImport(download, fmt.Errorf("no video files found"))
	}

	// Select main file (largest video file)
	mainFile := m.selectMainFile(files)

	// Parse the release name
	parsed := parser.ParseReleaseName(filepath.Base(mainFile))

	// Try to match to library item
	if download.MediaID != nil && download.MediaType != nil {
		destPath, err := m.generateDestPath(*download.MediaID, *download.MediaType, parsed)
		if err != nil {
			return m.failImport(download, fmt.Errorf("failed to generate destination: %w", err))
		}

		// Create directory structure
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return m.failImport(download, fmt.Errorf("failed to create directory: %w", err))
		}

		// Move main file
		if err := m.moveFile(mainFile, destPath); err != nil {
			return m.failImport(download, fmt.Errorf("failed to move file: %w", err))
		}

		// Handle extras
		extras := m.findExtras(files, mainFile)
		if len(extras) > 0 {
			extrasDir := filepath.Join(filepath.Dir(destPath), "Extras")
			os.MkdirAll(extrasDir, 0755)
			for _, extra := range extras {
				m.moveFile(extra, filepath.Join(extrasDir, filepath.Base(extra)))
			}
		}

		// Handle subtitles
		subs := m.findSubtitles(sourcePath)
		for _, sub := range subs {
			subDest := m.generateSubtitlePath(destPath, sub)
			m.moveFile(sub, subDest)
		}

		// Update download status
		download.Status = "imported"
		importedPath := destPath
		download.ImportedPath = &importedPath
		if err := m.db.UpdateDownload(download); err != nil {
			return err
		}

		// Log import
		m.db.CreateImportHistory(&database.ImportHistory{
			DownloadID: &download.ID,
			SourcePath: sourcePath,
			DestPath:   destPath,
			MediaID:    download.MediaID,
			MediaType:  download.MediaType,
			Success:    true,
		})

		// Clean up source
		m.cleanupSource(sourcePath)

		return nil
	}

	// No match - move to unmatched folder
	return m.handleUnmatched(download, files)
}

// findVideoFiles finds all video files in a directory
func (m *Manager) findVideoFiles(dir string) ([]string, error) {
	var files []string
	videoExts := []string{".mkv", ".mp4", ".avi", ".m4v", ".wmv", ".mov", ".ts", ".m2ts"}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, videoExt := range videoExts {
			if ext == videoExt {
				// Skip sample files
				if strings.Contains(strings.ToLower(filepath.Base(path)), "sample") {
					continue
				}
				files = append(files, path)
				break
			}
		}
		return nil
	})

	return files, err
}

// selectMainFile selects the main video file (largest)
func (m *Manager) selectMainFile(files []string) string {
	var largest string
	var maxSize int64

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.Size() > maxSize {
			maxSize = info.Size()
			largest = f
		}
	}

	return largest
}

// findExtras identifies extra/bonus content files
func (m *Manager) findExtras(files []string, mainFile string) []string {
	var extras []string
	extrasPatterns := []string{
		`(?i)extras?`,
		`(?i)featurettes?`,
		`(?i)bonus`,
		`(?i)deleted.?scenes?`,
		`(?i)behind.?the.?scenes?`,
		`(?i)making.?of`,
		`(?i)interview`,
		`(?i)trailer`,
		`(?i)gag.?reel`,
		`(?i)bloopers?`,
	}

	for _, f := range files {
		if f == mainFile {
			continue
		}

		name := filepath.Base(f)
		for _, pattern := range extrasPatterns {
			if matched, _ := regexp.MatchString(pattern, name); matched {
				extras = append(extras, f)
				break
			}
		}
	}

	return extras
}

// findSubtitles finds subtitle files
func (m *Manager) findSubtitles(dir string) []string {
	var subs []string
	subExts := []string{".srt", ".sub", ".idx", ".ass", ".ssa", ".vtt"}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, subExt := range subExts {
			if ext == subExt {
				subs = append(subs, path)
				break
			}
		}
		return nil
	})

	return subs
}

// generateDestPath generates the destination path for a file
func (m *Manager) generateDestPath(mediaID int64, mediaType string, parsed *parser.ParsedRelease) (string, error) {
	// Get library path based on media type
	libraries, err := m.db.GetLibraries()
	if err != nil {
		return "", err
	}

	var libraryPath string
	for _, lib := range libraries {
		if (mediaType == "movie" && lib.Type == "movies") ||
			(mediaType == "episode" && lib.Type == "tv") {
			libraryPath = lib.Path
			break
		}
	}

	if libraryPath == "" {
		return "", fmt.Errorf("no library found for media type: %s", mediaType)
	}

	// Get naming template
	templates, err := m.db.GetNamingTemplates()
	if err != nil {
		return "", err
	}

	var template *database.NamingTemplate
	for _, t := range templates {
		if (mediaType == "movie" && t.Type == "movie") ||
			(mediaType == "episode" && t.Type == "tv") {
			template = &t
			break
		}
	}

	if template == nil {
		// Default template
		if mediaType == "movie" {
			return filepath.Join(libraryPath, sanitizeFilename(parsed.Title)+" ("+strconv.Itoa(parsed.Year)+")", sanitizeFilename(parsed.Title)+" ("+strconv.Itoa(parsed.Year)+").mkv"), nil
		}
		return filepath.Join(libraryPath, sanitizeFilename(parsed.Title), fmt.Sprintf("Season %02d", parsed.Season), fmt.Sprintf("%s - S%02dE%02d.mkv", sanitizeFilename(parsed.Title), parsed.Season, parsed.Episode)), nil
	}

	// Apply template
	folder := m.applyTemplate(template.FolderTemplate, parsed, mediaType)
	file := m.applyTemplate(template.FileTemplate, parsed, mediaType)

	ext := ".mkv" // Default extension
	return filepath.Join(libraryPath, folder, file+ext), nil
}

// applyTemplate applies naming template placeholders
func (m *Manager) applyTemplate(template string, parsed *parser.ParsedRelease, mediaType string) string {
	result := template

	// Common replacements
	result = strings.ReplaceAll(result, "{Title}", sanitizeFilename(parsed.Title))
	result = strings.ReplaceAll(result, "{Year}", strconv.Itoa(parsed.Year))

	// TV-specific
	result = strings.ReplaceAll(result, "{Season:00}", fmt.Sprintf("%02d", parsed.Season))
	result = strings.ReplaceAll(result, "{Episode:00}", fmt.Sprintf("%02d", parsed.Episode))
	result = strings.ReplaceAll(result, "{EpisodeTitle}", sanitizeFilename(parsed.EpisodeTitle))

	// Quality info
	result = strings.ReplaceAll(result, "{Resolution}", parsed.Resolution)
	result = strings.ReplaceAll(result, "{Source}", parsed.Source)
	result = strings.ReplaceAll(result, "{Codec}", parsed.Codec)

	// Daily shows
	if parsed.IsDailyShow && parsed.AirDate != "" {
		result = strings.ReplaceAll(result, "{Air-Date}", parsed.AirDate)
	}

	return result
}

// generateSubtitlePath generates path for a subtitle file
func (m *Manager) generateSubtitlePath(videoPath, subPath string) string {
	videoBase := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	subExt := filepath.Ext(subPath)

	// Try to detect language from subtitle filename
	subName := strings.ToLower(filepath.Base(subPath))
	langCode := ""

	langPatterns := map[string]string{
		"english": ".en",
		"eng":     ".en",
		"spanish": ".es",
		"spa":     ".es",
		"french":  ".fr",
		"fra":     ".fr",
		"german":  ".de",
		"deu":     ".de",
		"ger":     ".de",
		"italian": ".it",
		"ita":     ".it",
	}

	for pattern, code := range langPatterns {
		if strings.Contains(subName, pattern) {
			langCode = code
			break
		}
	}

	return videoBase + langCode + subExt
}

// moveFile moves a file to a new location
func (m *Manager) moveFile(src, dst string) error {
	// First try rename (same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to copy + delete
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return err
	}

	return os.Remove(src)
}

// cleanupSource removes empty directories after import
func (m *Manager) cleanupSource(dir string) {
	// Walk backwards, removing empty directories
	for i := 0; i < 5; i++ { // Max 5 levels
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}

		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// handleUnmatched moves files to unmatched folder
func (m *Manager) handleUnmatched(download *database.Download, files []string) error {
	// Get a library path for unmatched files
	libraries, err := m.db.GetLibraries()
	if err != nil || len(libraries) == 0 {
		download.Status = "unmatched"
		download.Error = stringPtr("No library configured")
		return m.db.UpdateDownload(download)
	}

	unmatchedDir := filepath.Join(libraries[0].Path, "_Unmatched")
	os.MkdirAll(unmatchedDir, 0755)

	destDir := filepath.Join(unmatchedDir, sanitizeFilename(download.Title))
	os.MkdirAll(destDir, 0755)

	for _, f := range files {
		dest := filepath.Join(destDir, filepath.Base(f))
		m.moveFile(f, dest)
	}

	download.Status = "unmatched"
	importedPath := destDir
	download.ImportedPath = &importedPath
	return m.db.UpdateDownload(download)
}

// failImport marks an import as failed
func (m *Manager) failImport(download *database.Download, err error) error {
	download.Status = "failed"
	errMsg := err.Error()
	download.Error = &errMsg
	m.db.UpdateDownload(download)

	m.db.CreateImportHistory(&database.ImportHistory{
		DownloadID: &download.ID,
		SourcePath: "",
		DestPath:   "",
		MediaID:    download.MediaID,
		MediaType:  download.MediaType,
		Success:    false,
		Error:      &errMsg,
	})

	log.Printf("Import failed for %s: %s", download.Title, err.Error())
	return err
}

// sanitizeFilename removes invalid characters from a filename
func sanitizeFilename(s string) string {
	invalid := regexp.MustCompile(`[/\\:*?"<>|]`)
	return invalid.ReplaceAllString(s, "")
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// ParseMultiEpisode parses multi-episode filenames
func ParseMultiEpisode(filename string) (start, end int, isMulti bool) {
	patterns := []string{
		`S(\d+)E(\d+)E(\d+)`,
		`S(\d+)E(\d+)-E?(\d+)`,
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if matches := re.FindStringSubmatch(filename); matches != nil {
			start, _ = strconv.Atoi(matches[2])
			end, _ = strconv.Atoi(matches[3])
			return start, end, true
		}
	}

	return 0, 0, false
}

// DownloadTracker monitors download clients for completed downloads
type DownloadTracker struct {
	db       *database.Database
	importer *Manager
	interval time.Duration
	stop     chan struct{}
}

// NewDownloadTracker creates a new download tracker
func NewDownloadTracker(db *database.Database, importer *Manager, interval time.Duration) *DownloadTracker {
	return &DownloadTracker{
		db:       db,
		importer: importer,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start begins the download tracking loop
func (t *DownloadTracker) Start() {
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.checkDownloads()
			case <-t.stop:
				return
			}
		}
	}()
}

// Stop stops the download tracker
func (t *DownloadTracker) Stop() {
	close(t.stop)
}

// checkDownloads checks for completed downloads
func (t *DownloadTracker) checkDownloads() {
	// Get downloads that are completed but not imported
	downloads, err := t.db.GetDownloads()
	if err != nil {
		log.Printf("Failed to get downloads: %v", err)
		return
	}

	for _, dl := range downloads {
		if dl.Status == "completed" && dl.DownloadPath != nil {
			go t.importer.ProcessImport(&dl, *dl.DownloadPath)
		}
	}
}
