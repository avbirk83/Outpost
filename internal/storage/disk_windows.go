//go:build windows

package storage

import (
	"golang.org/x/sys/windows"
)

// GetDiskUsage returns disk usage for a given path
func GetDiskUsage(path string) (*DiskUsage, error) {
	var freeBytesAvailable, totalBytes, freeBytes uint64

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailable, &totalBytes, &freeBytes)
	if err != nil {
		return nil, err
	}

	used := totalBytes - freeBytes

	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(used) / float64(totalBytes) * 100
	}

	return &DiskUsage{
		Total:       totalBytes,
		Free:        freeBytes,
		Used:        used,
		UsedPercent: usedPercent,
	}, nil
}
