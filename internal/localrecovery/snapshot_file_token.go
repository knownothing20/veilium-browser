package localrecovery

import (
	"fmt"
	"os"
)

type fileToken struct {
	Identity string
	Size     int64
	Modified int64
}

func (t fileToken) Equal(other fileToken) bool {
	return t.Identity == other.Identity && t.Size == other.Size && t.Modified == other.Modified
}

func inspectRegularPath(filePath string) (fileToken, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return fileToken{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fileToken{}, fmt.Errorf("%w: snapshot source entry is not a regular file", ErrInvalidManifest)
	}
	unsafe, err := pathHasReparsePoint(filePath)
	if err != nil {
		return fileToken{}, fmt.Errorf("inspect snapshot source entry: %w", err)
	}
	if unsafe {
		return fileToken{}, fmt.Errorf("%w: snapshot source entry is a reparse point", ErrInvalidManifest)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fileToken{}, err
	}
	defer file.Close()
	token, err := tokenFromOpenFile(file)
	if err != nil {
		return fileToken{}, err
	}
	if token.Size != info.Size() || token.Modified != info.ModTime().UnixNano() {
		return fileToken{}, ErrSourceChanged
	}
	return token, nil
}
