//go:build windows

package localrecovery

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func tokenFromOpenFile(file *os.File) (fileToken, error) {
	info, err := file.Stat()
	if err != nil {
		return fileToken{}, err
	}
	if !info.Mode().IsRegular() {
		return fileToken{}, fmt.Errorf("%w: opened snapshot source is not regular", ErrInvalidManifest)
	}
	var identity windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &identity); err != nil {
		return fileToken{}, fmt.Errorf("inspect Windows snapshot source identity: %w", err)
	}
	if identity.NumberOfLinks != 1 {
		return fileToken{}, fmt.Errorf("%w: snapshot source has ambiguous hard links", ErrInvalidManifest)
	}
	size := int64(uint64(identity.FileSizeHigh)<<32 | uint64(identity.FileSizeLow))
	return fileToken{
		Identity: fmt.Sprintf("%08x:%08x:%08x", identity.VolumeSerialNumber, identity.FileIndexHigh, identity.FileIndexLow),
		Size:     size,
		Modified: identity.LastWriteTime.Nanoseconds(),
	}, nil
}
