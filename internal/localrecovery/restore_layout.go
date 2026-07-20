package localrecovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

const restoreStagingDirectory = ".restore-staging"

func restoreStagingRef(operationID string) string {
	return path.Join("local-recovery", "restore-staging", operationID)
}

func restoreStagePath(recoveryRoot, operationID string) string {
	return filepath.Join(recoveryRoot, restoreStagingDirectory, operationID)
}

func restoreManagedRef(profileID string) string {
	return path.Join("profiles", profileID)
}

func (e *RestoreExecutor) createRestoreStage(operationID string) (string, error) {
	stagingRoot := filepath.Join(e.recoveryRoot, restoreStagingDirectory)
	if err := ensurePrivateDirectoryTree(stagingRoot); err != nil {
		return "", err
	}
	stagePath := restoreStagePath(e.recoveryRoot, operationID)
	if err := os.Mkdir(stagePath, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%w: restore staging already exists", ErrRecoveryRequired)
		}
		return "", fmt.Errorf("create restore staging: %w", err)
	}
	if err := os.Mkdir(filepath.Join(stagePath, browserDataDirectory), 0o700); err != nil {
		return "", fmt.Errorf("create restored browser-data staging: %w", err)
	}
	return stagePath, nil
}

func (e *RestoreExecutor) removeOwnedRestoreStage(recoveryRoot, stagePath string) error {
	stagingRoot := filepath.Join(recoveryRoot, restoreStagingDirectory)
	return removeOwnedDirectory(stagePath, stagingRoot, "restore staging")
}

func (e *RestoreExecutor) removeOwnedRestoredProfile(profilePath string) error {
	return removeOwnedDirectory(profilePath, e.profilesRoot, "restored Profile")
}

func removeOwnedDirectory(target, root, label string) error {
	if !pathContainedBy(target, root) || filepath.Clean(target) == filepath.Clean(root) {
		return fmt.Errorf("%w: %s path is outside the owned boundary", ErrRecoveryRequired, label)
	}
	if _, err := os.Lstat(target); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s contains a symbolic link", ErrRecoveryRequired, label)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: %s contains a reparse point", ErrRecoveryRequired, label)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(target)
}
