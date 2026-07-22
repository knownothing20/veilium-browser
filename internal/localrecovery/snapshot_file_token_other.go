//go:build !windows

package localrecovery

import (
	"fmt"
	"os"
	"syscall"
)

func tokenFromOpenFile(file *os.File) (fileToken, error) {
	info, err := file.Stat()
	if err != nil {
		return fileToken{}, err
	}
	if !info.Mode().IsRegular() {
		return fileToken{}, fmt.Errorf("%w: opened snapshot source is not regular", ErrInvalidManifest)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileToken{}, fmt.Errorf("%w: snapshot source identity is unavailable", ErrInvalidManifest)
	}
	if stat.Nlink != 1 {
		return fileToken{}, fmt.Errorf("%w: snapshot source has ambiguous hard links", ErrInvalidManifest)
	}
	return fileToken{
		Identity: fmt.Sprintf("%v:%v", stat.Dev, stat.Ino),
		Size:     info.Size(),
		Modified: info.ModTime().UnixNano(),
	}, nil
}
