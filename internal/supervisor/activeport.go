package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	devToolsActivePortFilename = "DevToolsActivePort"
	maxActivePortBytes         = 4096
)

type EndpointDiscovery interface {
	Prepare(userDataDir string) error
	Wait(context.Context, string) (int, error)
}

type DevToolsActivePortDiscovery struct {
	Interval time.Duration
}

func (d DevToolsActivePortDiscovery) Prepare(userDataDir string) error {
	path, err := activePortPath(userDataDir)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect stale DevToolsActivePort: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("stale DevToolsActivePort must be a regular file")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale DevToolsActivePort: %w", err)
	}
	return nil
}

func (d DevToolsActivePortDiscovery) Wait(ctx context.Context, userDataDir string) (int, error) {
	path, err := activePortPath(userDataDir)
	if err != nil {
		return 0, err
	}
	interval := d.Interval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	var lastErr error
	for {
		port, readErr := readDevToolsActivePort(path)
		if readErr == nil {
			return port, nil
		}
		lastErr = readErr
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if lastErr != nil {
				return 0, fmt.Errorf("%v: %w", lastErr, ctx.Err())
			}
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
}

func activePortPath(userDataDir string) (string, error) {
	userDataDir = strings.TrimSpace(userDataDir)
	if userDataDir == "" || strings.ContainsRune(userDataDir, '\x00') {
		return "", fmt.Errorf("valid user data directory is required for CDP discovery")
	}
	info, err := os.Lstat(userDataDir)
	if err != nil {
		return "", fmt.Errorf("inspect user data directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("user data directory must be a real directory")
	}
	return filepath.Join(userDataDir, devToolsActivePortFilename), nil
}

func readDevToolsActivePort(path string) (int, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return 0, fmt.Errorf("DevToolsActivePort must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if !os.SameFile(before, after) || !after.Mode().IsRegular() {
		return 0, fmt.Errorf("DevToolsActivePort changed while opening")
	}
	if after.Size() <= 0 || after.Size() > maxActivePortBytes {
		return 0, fmt.Errorf("DevToolsActivePort has an invalid size")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxActivePortBytes+1))
	if err != nil {
		return 0, err
	}
	if len(data) > maxActivePortBytes {
		return 0, fmt.Errorf("DevToolsActivePort exceeds the size limit")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("DevToolsActivePort is incomplete")
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("DevToolsActivePort contains an invalid port")
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[1]), "/devtools/browser/") {
		return 0, fmt.Errorf("DevToolsActivePort contains an invalid browser path")
	}
	return port, nil
}
