package localrecovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	snapshotStagingDirectory = ".staging"
	snapshotFinalDirectory   = "snapshots"
	profileDefinitionName    = "profile-definition.json"
	browserDataDirectory     = "browser-data"
	manifestFileName         = "manifest.json"
)

func prepareRecoveryRoots(dataRoot string) (string, string, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return "", "", fmt.Errorf("local recovery data root is required")
	}
	absolute, err := filepath.Abs(dataRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve local recovery data root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if err := inspectRealDirectory(absolute); err != nil {
		return "", "", err
	}
	recoveryRoot := filepath.Join(absolute, "local-recovery")
	if err := ensurePrivateDirectoryTree(recoveryRoot); err != nil {
		return "", "", err
	}
	return absolute, recoveryRoot, nil
}

func inspectRealDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect local recovery directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: local recovery directory must be real", ErrInvalidRecord)
	}
	unsafe, err := pathHasReparsePoint(directory)
	if err != nil {
		return fmt.Errorf("inspect local recovery directory attributes: %w", err)
	}
	if unsafe {
		return fmt.Errorf("%w: local recovery directory is a reparse point", ErrInvalidRecord)
	}
	return nil
}

func snapshotStagingRef(operationID string) string {
	return path.Join("local-recovery", "staging", operationID)
}

func snapshotPublishedRef(snapshotID string) string {
	return path.Join("local-recovery", snapshotFinalDirectory, snapshotID)
}

func snapshotStagePath(recoveryRoot, operationID string) string {
	return filepath.Join(recoveryRoot, snapshotStagingDirectory, operationID)
}

func snapshotFinalPath(recoveryRoot, snapshotID string) string {
	return filepath.Join(recoveryRoot, snapshotFinalDirectory, snapshotID)
}

func managedSourcePath(dataRoot, managedDir string) (string, error) {
	if err := ValidateRelativePath(managedDir, runtime.GOOS); err != nil {
		return "", err
	}
	candidate := filepath.Join(dataRoot, filepath.FromSlash(managedDir))
	if !pathContainedBy(candidate, dataRoot) {
		return "", fmt.Errorf("%w: managed source escapes the application root", ErrInvalidRecord)
	}
	if err := inspectDirectoryTree(dataRoot, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func inspectDirectoryTree(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if !pathContainedBy(target, root) {
		return fmt.Errorf("%w: directory path escapes its root", ErrInvalidRecord)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	current := root
	if err := inspectRealDirectory(current); err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if err := inspectRealDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	return pathContainedBy(left, right) || pathContainedBy(right, left)
}

func pathContainedBy(candidate, root string) bool {
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	if runtime.GOOS == "windows" && strings.EqualFold(candidate, root) {
		return true
	}
	if candidate == root {
		return true
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || filepath.IsAbs(relative) {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (c *SnapshotCreator) removeOwnedStage(recoveryRoot, stagePath string) error {
	stagingRoot := filepath.Join(recoveryRoot, snapshotStagingDirectory)
	if !pathContainedBy(stagePath, stagingRoot) || filepath.Clean(stagePath) == filepath.Clean(stagingRoot) {
		return fmt.Errorf("%w: staging cleanup path is outside the owned boundary", ErrRecoveryRequired)
	}
	if _, err := os.Lstat(stagePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := filepath.WalkDir(stagePath, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: owned staging contains a symbolic link", ErrRecoveryRequired)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: owned staging contains a reparse point", ErrRecoveryRequired)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(stagePath)
}

func renamePath(source, destination string) error {
	return os.Rename(source, destination)
}

func syncDirectory(directoryPath string) {
	if runtime.GOOS == "windows" {
		return
	}
	directory, err := os.Open(directoryPath)
	if err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
}
