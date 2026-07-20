package lifecycle

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

func encodeIndented(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode lifecycle store: %w", err)
	}
	if len(data) > MaxStoreBytes {
		return nil, fmt.Errorf("%w: encoded store exceeds %d bytes", ErrInvalidRecord, MaxStoreBytes)
	}
	return data, nil
}

func decodeStrictFile(path string, value any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: lifecycle store must be a regular file", ErrInvalidRecord)
	}
	if info.Size() <= 0 || info.Size() > MaxStoreBytes {
		return fmt.Errorf("%w: lifecycle store size is outside bounds", ErrInvalidRecord)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: lifecycle store permissions are not private", ErrInvalidRecord)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	limited := io.LimitReader(file, MaxStoreBytes+1)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode lifecycle store: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%w: lifecycle store contains trailing data", ErrInvalidRecord)
		}
		return fmt.Errorf("decode lifecycle store trailing data: %w", err)
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	if len(data) == 0 || len(data) > MaxStoreBytes {
		return fmt.Errorf("%w: lifecycle store payload is outside bounds", ErrInvalidRecord)
	}
	parent := filepath.Dir(path)
	if err := ensureDirectoryTree(parent); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: lifecycle store target is not a regular file", ErrInvalidRecord)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	temp, err := os.CreateTemp(parent, ".lifecycle-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary lifecycle store: %w", err)
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
		return fmt.Errorf("secure temporary lifecycle store: %w", err)
	}
	if _, err := io.Copy(temp, bytes.NewReader(data)); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary lifecycle store: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary lifecycle store: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary lifecycle store: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace lifecycle store: %w", err)
	}
	removeTemp = false
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure lifecycle store: %w", err)
	}
	if runtime.GOOS != "windows" {
		if directory, err := os.Open(parent); err == nil {
			_ = directory.Sync()
			_ = directory.Close()
		}
	}
	return nil
}

func ensureDirectoryTree(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve lifecycle directory: %w", err)
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
				return fmt.Errorf("create lifecycle directory %q: %w", current, err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return fmt.Errorf("inspect lifecycle directory %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%w: lifecycle directory path is unsafe", ErrInvalidRecord)
		}
	}
	return nil
}
