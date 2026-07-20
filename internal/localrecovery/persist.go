package localrecovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type writeFileFunc func(string, []byte) error

func WriteManifest(filePath string, manifest LocalSnapshotManifest) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	data, err := encodeBounded(manifest, MaxManifestBytes, ErrInvalidManifest)
	if err != nil {
		return "", err
	}
	if err := atomicWrite(filePath, data, MaxManifestBytes, false); err != nil {
		return "", fmt.Errorf("publish local recovery manifest: %w", err)
	}
	digest, err := ComputeManifestDigest(manifest)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func ReadManifest(filePath string) (LocalSnapshotManifest, error) {
	var manifest LocalSnapshotManifest
	if err := decodeStrictFile(filePath, MaxManifestBytes, ErrInvalidManifest, &manifest); err != nil {
		return LocalSnapshotManifest{}, fmt.Errorf("read local recovery manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return LocalSnapshotManifest{}, err
	}
	return manifest, nil
}

func encodeBounded(value any, maximum int64, base error) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode local recovery record: %w", err)
	}
	if len(data) == 0 || int64(len(data)) > maximum {
		return nil, fmt.Errorf("%w: encoded record exceeds %d bytes", base, maximum)
	}
	return data, nil
}

func decodeStrictFile(filePath string, maximum int64, base error, value any) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: authoritative file must be regular", base)
	}
	if unsafe, inspectErr := pathHasReparsePoint(filePath); inspectErr != nil {
		return fmt.Errorf("inspect authoritative file: %w", inspectErr)
	} else if unsafe {
		return fmt.Errorf("%w: authoritative file is a reparse point", base)
	}
	if info.Size() <= 0 || info.Size() > maximum {
		return fmt.Errorf("%w: authoritative file size is outside bounds", base)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: authoritative file permissions are not private", base)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, maximum+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode local recovery record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%w: authoritative file contains trailing data", base)
		}
		return fmt.Errorf("decode trailing local recovery data: %w", err)
	}
	return nil
}

func atomicWrite(filePath string, data []byte, maximum int64, replace bool) error {
	if len(data) == 0 || int64(len(data)) > maximum {
		return fmt.Errorf("%w: persistence payload is outside bounds", ErrInvalidRecord)
	}
	parent := filepath.Dir(filePath)
	if err := ensurePrivateDirectoryTree(parent); err != nil {
		return err
	}
	if info, err := os.Lstat(filePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: persistence target is unsafe", ErrInvalidRecord)
		}
		if unsafe, inspectErr := pathHasReparsePoint(filePath); inspectErr != nil {
			return fmt.Errorf("inspect persistence target: %w", inspectErr)
		} else if unsafe {
			return fmt.Errorf("%w: persistence target is a reparse point", ErrInvalidRecord)
		}
		if !replace {
			return ErrAlreadyExists
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	temp, err := os.CreateTemp(parent, ".local-recovery-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary local recovery file: %w", err)
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("secure temporary local recovery file: %w", err)
	}
	if _, err := io.Copy(temp, bytes.NewReader(data)); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary local recovery file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary local recovery file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary local recovery file: %w", err)
	}

	if replace {
		if err := os.Rename(tempName, filePath); err != nil {
			return fmt.Errorf("replace local recovery file: %w", err)
		}
	} else {
		if err := os.Link(tempName, filePath); err != nil {
			if errors.Is(err, os.ErrExist) {
				return ErrAlreadyExists
			}
			return fmt.Errorf("publish immutable local recovery file: %w", err)
		}
		if err := os.Remove(tempName); err != nil {
			return fmt.Errorf("remove temporary local recovery link: %w", err)
		}
	}
	removeTemp = false
	if err := os.Chmod(filePath, 0o600); err != nil {
		return fmt.Errorf("secure local recovery file: %w", err)
	}
	if runtime.GOOS != "windows" {
		if directory, err := os.Open(parent); err == nil {
			_ = directory.Sync()
			_ = directory.Close()
		}
	}
	return nil
}

func ensurePrivateDirectoryTree(directoryPath string) error {
	absolute, err := filepath.Abs(directoryPath)
	if err != nil {
		return fmt.Errorf("resolve local recovery directory: %w", err)
	}
	absolute = filepath.Clean(absolute)
	volume := filepath.VolumeName(absolute)
	remainder := strings.TrimPrefix(absolute, volume)
	remainder = strings.TrimPrefix(remainder, string(filepath.Separator))
	current := volume + string(filepath.Separator)
	if volume == "" {
		current = string(filepath.Separator)
	}
	if remainder == "" {
		return nil
	}
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("create local recovery directory %q: %w", current, err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return fmt.Errorf("inspect local recovery directory %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%w: local recovery directory path is unsafe", ErrInvalidRecord)
		}
		if unsafe, inspectErr := pathHasReparsePoint(current); inspectErr != nil {
			return fmt.Errorf("inspect local recovery directory %q: %w", current, inspectErr)
		} else if unsafe {
			return fmt.Errorf("%w: local recovery directory path contains a reparse point", ErrInvalidRecord)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("%w: local recovery directory permissions are not private", ErrInvalidRecord)
		}
	}
	return nil
}
