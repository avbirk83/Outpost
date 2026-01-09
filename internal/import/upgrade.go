package importpkg

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/outpost/outpost/internal/parser"
	"github.com/outpost/outpost/internal/quality"
)

// UpgradeChecker determines if a new release is an upgrade over existing
type UpgradeChecker struct {
	recycleBinPath string // If set, old files are moved here instead of deleted
	keepOldFiles   bool   // If true, don't delete old files at all
}

// NewUpgradeChecker creates a new upgrade checker
func NewUpgradeChecker() *UpgradeChecker {
	return &UpgradeChecker{}
}

// SetRecycleBin sets the path where old files are moved instead of deleted
func (u *UpgradeChecker) SetRecycleBin(path string) {
	u.recycleBinPath = path
}

// SetKeepOldFiles configures whether to keep old files after upgrade
func (u *UpgradeChecker) SetKeepOldFiles(keep bool) {
	u.keepOldFiles = keep
}

// UpgradeResult contains the decision about an upgrade
type UpgradeResult struct {
	ShouldUpgrade bool
	Reason        string
	CurrentTier   string
	NewTier       string
	CurrentScore  int
	NewScore      int
}

// ShouldUpgrade checks if the new release is better than existing
func (u *UpgradeChecker) ShouldUpgrade(existing, new *parser.ParsedRelease) UpgradeResult {
	currentTier := quality.ComputeQualityTier(existing)
	newTier := quality.ComputeQualityTier(new)

	result := UpgradeResult{
		CurrentTier:  currentTier,
		NewTier:      newTier,
		CurrentScore: 0, // Could add custom format scoring later
		NewScore:     0,
	}

	// Compare tiers numerically based on quality order
	tierOrder := map[string]int{
		"unknown": 0, "sd": 1, "480p": 2, "720p": 3, "1080p": 4, "2160p": 5,
	}
	currentOrder := tierOrder[currentTier]
	newOrder := tierOrder[newTier]

	// Higher tier = better quality
	if newOrder > currentOrder {
		result.ShouldUpgrade = true
		result.Reason = "Higher quality tier"
		return result
	}

	// Same tier - check for proper/repack
	if newOrder == currentOrder {
		// Proper/repack is always an upgrade
		if new.IsProper && !existing.IsProper {
			result.ShouldUpgrade = true
			result.Reason = "PROPER release"
			return result
		}
		if new.IsRepack && !existing.IsRepack {
			result.ShouldUpgrade = true
			result.Reason = "REPACK release"
			return result
		}

		// Better audio
		if quality.GetAudioScore(new.AudioFormat) > quality.GetAudioScore(existing.AudioFormat) {
			result.ShouldUpgrade = true
			result.Reason = "Better audio codec"
			return result
		}
	}

	result.ShouldUpgrade = false
	result.Reason = "Not an upgrade"
	return result
}

// HandleOldFile removes or recycles the old file after successful upgrade
func (u *UpgradeChecker) HandleOldFile(oldPath string) error {
	if u.keepOldFiles {
		log.Printf("Keeping old file (keepOldFiles=true): %s", oldPath)
		return nil
	}

	if u.recycleBinPath != "" {
		return u.moveToRecycleBin(oldPath)
	}

	return u.deleteFile(oldPath)
}

// moveToRecycleBin moves a file to the recycle bin
func (u *UpgradeChecker) moveToRecycleBin(oldPath string) error {
	if err := os.MkdirAll(u.recycleBinPath, 0755); err != nil {
		return err
	}

	// Create timestamped name to avoid conflicts
	base := filepath.Base(oldPath)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	newName := timestamp + "_" + base

	dest := filepath.Join(u.recycleBinPath, newName)

	log.Printf("Moving to recycle bin: %s -> %s", oldPath, dest)
	return os.Rename(oldPath, dest)
}

// deleteFile permanently deletes a file or directory
func (u *UpgradeChecker) deleteFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return err
	}

	if info.IsDir() {
		log.Printf("Deleting old directory: %s", path)
		return os.RemoveAll(path)
	}

	log.Printf("Deleting old file: %s", path)
	return os.Remove(path)
}

// CleanRecycleBin removes files older than maxAge from the recycle bin
func (u *UpgradeChecker) CleanRecycleBin(maxAge time.Duration) error {
	if u.recycleBinPath == "" {
		return nil
	}

	cutoff := time.Now().Add(-maxAge)

	entries, err := os.ReadDir(u.recycleBinPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(u.recycleBinPath, entry.Name())
			log.Printf("Cleaning old recycle bin item: %s", path)
			if entry.IsDir() {
				os.RemoveAll(path)
			} else {
				os.Remove(path)
			}
		}
	}

	return nil
}
