//go:build windows

package localrecovery

func availableBytes(directoryPath string) (uint64, error) {
	return windowsAvailableBytes(directoryPath)
}
