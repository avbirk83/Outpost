package importpkg

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/outpost/outpost/internal/download"
	"github.com/outpost/outpost/internal/parser"
)

// RejectionType indicates if a rejection is permanent or can be retried
type RejectionType int

const (
	RejectionPermanent RejectionType = iota
	RejectionTemporary
)

// Rejection represents why a file was rejected for import
type Rejection struct {
	Reason string
	Type   RejectionType
}

// FileDecision represents the import decision for a single file
type FileDecision struct {
	FilePath   string
	FileSize   int64
	Approved   bool
	Rejections []Rejection
	ParsedInfo *parser.ParsedRelease
	IsSample   bool
	IsExtra    bool
}

// DecisionMaker evaluates files for import eligibility
type DecisionMaker struct {
	sampleSizeThreshold int64 // Files smaller than this are samples (default 100MB)
}

// NewDecisionMaker creates a new decision maker
func NewDecisionMaker() *DecisionMaker {
	return &DecisionMaker{
		sampleSizeThreshold: 100 * 1024 * 1024, // 100MB
	}
}

// SetSampleThreshold sets the minimum size for non-sample files
func (d *DecisionMaker) SetSampleThreshold(bytes int64) {
	d.sampleSizeThreshold = bytes
}

// EvaluateFiles examines all files from a download and returns decisions
func (d *DecisionMaker) EvaluateFiles(sourcePath string, td *download.TrackedDownload) ([]FileDecision, error) {
	// Find all video files
	files, err := findVideoFiles(sourcePath)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, &ImportError{Message: "No video files found in download"}
	}

	var decisions []FileDecision
	for _, file := range files {
		decision := d.evaluateFile(file, td)
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

// evaluateFile runs all checks on a single file
func (d *DecisionMaker) evaluateFile(filePath string, td *download.TrackedDownload) FileDecision {
	info, err := os.Stat(filePath)
	if err != nil {
		return FileDecision{
			FilePath:   filePath,
			Approved:   false,
			Rejections: []Rejection{{Reason: "Cannot read file: " + err.Error(), Type: RejectionPermanent}},
		}
	}

	decision := FileDecision{
		FilePath:   filePath,
		FileSize:   info.Size(),
		ParsedInfo: parser.Parse(filepath.Base(filePath)),
		Approved:   true,
	}

	// Run checks in order
	d.checkSample(&decision)
	d.checkExtra(&decision)
	d.checkValidMedia(&decision)

	return decision
}

// checkSample detects sample/trailer files
func (d *DecisionMaker) checkSample(decision *FileDecision) {
	name := strings.ToLower(filepath.Base(decision.FilePath))

	// Name-based detection
	samplePatterns := []string{"sample", "trailer", "preview", "teaser"}
	for _, pattern := range samplePatterns {
		if strings.Contains(name, pattern) {
			decision.IsSample = true
			decision.Approved = false
			decision.Rejections = append(decision.Rejections, Rejection{
				Reason: "File appears to be a sample/trailer",
				Type:   RejectionPermanent,
			})
			return
		}
	}

	// Size-based detection
	if decision.FileSize < d.sampleSizeThreshold {
		decision.IsSample = true
		decision.Approved = false
		decision.Rejections = append(decision.Rejections, Rejection{
			Reason: "File too small - likely a sample",
			Type:   RejectionPermanent,
		})
	}
}

// checkExtra detects extra/bonus content
func (d *DecisionMaker) checkExtra(decision *FileDecision) {
	name := strings.ToLower(filepath.Base(decision.FilePath))
	dir := strings.ToLower(filepath.Dir(decision.FilePath))

	extraPatterns := []string{"extras", "bonus", "featurette", "behind the scenes", "deleted scene", "interview"}
	for _, pattern := range extraPatterns {
		if strings.Contains(name, pattern) || strings.Contains(dir, pattern) {
			decision.IsExtra = true
			// Extras are approved but flagged - they go to an Extras folder
			return
		}
	}
}

// checkValidMedia verifies the file is a valid video
func (d *DecisionMaker) checkValidMedia(decision *FileDecision) {
	ext := strings.ToLower(filepath.Ext(decision.FilePath))

	validExtensions := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
		".wmv": true, ".m4v": true, ".webm": true, ".ts": true,
		".m2ts": true, ".flv": true,
	}

	if !validExtensions[ext] {
		decision.Approved = false
		decision.Rejections = append(decision.Rejections, Rejection{
			Reason: "Not a recognized video format",
			Type:   RejectionPermanent,
		})
	}
}

// GetMainFile returns the primary video file (largest approved file)
func (d *DecisionMaker) GetMainFile(decisions []FileDecision) *FileDecision {
	var main *FileDecision
	for i := range decisions {
		dec := &decisions[i]
		if !dec.Approved || dec.IsExtra {
			continue
		}
		if main == nil || dec.FileSize > main.FileSize {
			main = dec
		}
	}
	return main
}

// GetExtras returns all approved extra files
func (d *DecisionMaker) GetExtras(decisions []FileDecision) []FileDecision {
	var extras []FileDecision
	for _, dec := range decisions {
		if dec.Approved && dec.IsExtra {
			extras = append(extras, dec)
		}
	}
	return extras
}

// ImportError represents an import failure
type ImportError struct {
	Message string
}

func (e *ImportError) Error() string {
	return e.Message
}

// findVideoFiles recursively finds all video files in a path
func findVideoFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	// If it's a file, check if it's a video
	if !info.IsDir() {
		if isVideoFile(root) {
			return []string{root}, nil
		}
		return nil, nil
	}

	// Walk directory
	var files []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		if !info.IsDir() && isVideoFile(path) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// isVideoFile checks if a file has a video extension
func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := []string{".mkv", ".mp4", ".avi", ".mov", ".wmv", ".m4v", ".webm", ".ts", ".m2ts", ".flv"}
	for _, ve := range videoExts {
		if ext == ve {
			return true
		}
	}
	return false
}
