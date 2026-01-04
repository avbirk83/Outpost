package storage

import (
	"os"
	"path/filepath"
)

// StorageAlert represents a low disk space warning
type StorageAlert struct {
	LibraryID   int64   `json:"libraryId"`
	LibraryPath string  `json:"libraryPath"`
	LibraryName string  `json:"libraryName"`
	FreeGB      int64   `json:"freeGb"`
	TotalGB     int64   `json:"totalGb"`
	UsedGB      int64   `json:"usedGb"`
	UsedPercent float64 `json:"usedPercent"`
	ThresholdGB int64   `json:"thresholdGb"`
}

// DiskUsage represents disk usage statistics
type DiskUsage struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// Manager handles storage monitoring and management
type Manager struct {
	ThresholdGB      int64
	PauseEnabled     bool
	UpgradeDeleteOld bool
}

// NewManager creates a new storage manager
func NewManager(thresholdGB int64, pauseEnabled, upgradeDeleteOld bool) *Manager {
	return &Manager{
		ThresholdGB:      thresholdGB,
		PauseEnabled:     pauseEnabled,
		UpgradeDeleteOld: upgradeDeleteOld,
	}
}

// CheckPath checks disk space for a given path
func (m *Manager) CheckPath(path, name string, libraryID int64) *StorageAlert {
	usage, err := GetDiskUsage(path)
	if err != nil {
		return nil
	}

	freeGB := int64(usage.Free / (1024 * 1024 * 1024))
	totalGB := int64(usage.Total / (1024 * 1024 * 1024))
	usedGB := int64(usage.Used / (1024 * 1024 * 1024))

	if freeGB < m.ThresholdGB {
		return &StorageAlert{
			LibraryID:   libraryID,
			LibraryPath: path,
			LibraryName: name,
			FreeGB:      freeGB,
			TotalGB:     totalGB,
			UsedGB:      usedGB,
			UsedPercent: usage.UsedPercent,
			ThresholdGB: m.ThresholdGB,
		}
	}

	return nil
}

// ShouldPauseDownloads checks if downloads should be paused due to low disk space
func (m *Manager) ShouldPauseDownloads(paths []string) bool {
	if !m.PauseEnabled {
		return false
	}

	for _, path := range paths {
		usage, err := GetDiskUsage(path)
		if err != nil {
			continue
		}

		freeGB := int64(usage.Free / (1024 * 1024 * 1024))
		if freeGB < m.ThresholdGB {
			return true
		}
	}

	return false
}

// GetStorageStatus returns storage status for all provided paths
func (m *Manager) GetStorageStatus(libraries []struct {
	ID   int64
	Name string
	Path string
}) []StorageAlert {
	var alerts []StorageAlert

	for _, lib := range libraries {
		usage, err := GetDiskUsage(lib.Path)
		if err != nil {
			continue
		}

		freeGB := int64(usage.Free / (1024 * 1024 * 1024))
		totalGB := int64(usage.Total / (1024 * 1024 * 1024))
		usedGB := int64(usage.Used / (1024 * 1024 * 1024))

		alert := StorageAlert{
			LibraryID:   lib.ID,
			LibraryPath: lib.Path,
			LibraryName: lib.Name,
			FreeGB:      freeGB,
			TotalGB:     totalGB,
			UsedGB:      usedGB,
			UsedPercent: usage.UsedPercent,
			ThresholdGB: m.ThresholdGB,
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// HandleUpgrade handles file upgrade, optionally deleting the old file
func (m *Manager) HandleUpgrade(newFile, existingFile string) error {
	if m.UpgradeDeleteOld && existingFile != "" {
		// Delete the old file
		if err := os.Remove(existingFile); err != nil {
			// Log but don't fail - the new file is already imported
			return nil
		}

		// Try to clean up empty parent directories
		dir := filepath.Dir(existingFile)
		m.cleanEmptyDirs(dir)
	}
	return nil
}

// cleanEmptyDirs removes empty parent directories up to a reasonable level
func (m *Manager) cleanEmptyDirs(dir string) {
	for i := 0; i < 3; i++ { // Only go up 3 levels max
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}

		if err := os.Remove(dir); err != nil {
			return
		}

		dir = filepath.Dir(dir)
	}
}

// BytesToGB converts bytes to gigabytes
func BytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// GBToBytes converts gigabytes to bytes
func GBToBytes(gb int64) uint64 {
	return uint64(gb) * 1024 * 1024 * 1024
}
