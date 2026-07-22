//go:build windows

package localrecovery

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func windowsAvailableBytes(directoryPath string) (uint64, error) {
	pointer, err := windows.UTF16PtrFromString(directoryPath)
	if err != nil {
		return 0, fmt.Errorf("encode local snapshot destination path: %w", err)
	}
	var available uint64
	var total uint64
	var free uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &available, &total, &free); err != nil {
		return 0, fmt.Errorf("inspect local snapshot destination space: %w", err)
	}
	return available, nil
}
