//go:build !windows

package localrecovery

import (
	"fmt"
	"math"
	"syscall"
)

func availableBytes(directoryPath string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(directoryPath, &stat); err != nil {
		return 0, fmt.Errorf("inspect local snapshot destination space: %w", err)
	}
	blockSize := uint64(stat.Bsize)
	availableBlocks := uint64(stat.Bavail)
	if blockSize != 0 && availableBlocks > math.MaxUint64/blockSize {
		return math.MaxUint64, nil
	}
	return availableBlocks * blockSize, nil
}
