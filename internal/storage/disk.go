//go:build !windows

package storage

import (
	"syscall"
)

// GetDiskUsage returns disk usage for a given path
func GetDiskUsage(path string) (*DiskUsage, error) {
	var stat syscall.Statfs_t

	err := syscall.Statfs(path, &stat)
	if err != nil {
		return nil, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100
	}

	return &DiskUsage{
		Total:       total,
		Free:        free,
		Used:        used,
		UsedPercent: usedPercent,
	}, nil
}
