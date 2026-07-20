package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func (c *SnapshotCreator) verifySourcePlan(ctx context.Context, operationID string, plan snapshotPlan) error {
	expected := make(map[string]plannedSnapshotFile, len(plan.Files))
	for _, item := range plan.Files {
		expected[item.RelativePath] = item
	}
	seen := make(map[string]struct{}, len(plan.Files))
	err := filepath.WalkDir(plan.SourceRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := c.checkCancellation(ctx, operationID); err != nil {
			return err
		}
		if current == plan.SourceRoot {
			return nil
		}
		relative, err := filepath.Rel(plan.SourceRoot, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := ValidateRelativePath(relative, runtime.GOOS); err != nil {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrSourceChanged
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil || unsafe {
			return ErrSourceChanged
		}
		if info.IsDir() {
			return nil
		}
		planned, exists := expected[relative]
		if !exists || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: source entry set changed at %q", ErrSourceChanged, relative)
		}
		token, err := inspectRegularPath(current)
		if err != nil {
			return err
		}
		if !token.Equal(planned.Token) {
			return fmt.Errorf("%w: source file changed at %q", ErrSourceChanged, relative)
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("%w: source file set changed", ErrSourceChanged)
	}
	return nil
}

func verifyStagedSnapshot(stagePath string, expected LocalSnapshotManifest) (LocalSnapshotManifest, error) {
	profilePath := filepath.Join(stagePath, profileDefinitionName)
	if _, err := inspectRegularPath(profilePath); err != nil {
		return LocalSnapshotManifest{}, fmt.Errorf("inspect staged Profile definition: %w", err)
	}
	profileData, err := readBoundedFile(profilePath, MaxProfileDefinitionBytes)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	if err := validateProfileDefinitionExclusions(profileData); err != nil {
		return LocalSnapshotManifest{}, err
	}
	profileDigest, err := DigestProfileDefinition(profileData)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	if profileDigest != expected.ProfileDefinitionDigest {
		return LocalSnapshotManifest{}, fmt.Errorf("%w: staged Profile definition digest changed", ErrInvalidManifest)
	}

	manifestPath := filepath.Join(stagePath, manifestFileName)
	if _, err := inspectRegularPath(manifestPath); err != nil {
		return LocalSnapshotManifest{}, fmt.Errorf("inspect staged manifest: %w", err)
	}
	loaded, err := ReadManifest(manifestPath)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	expectedDigest, err := ComputeManifestDigest(expected)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	loadedDigest, err := ComputeManifestDigest(loaded)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	if loadedDigest != expectedDigest {
		return LocalSnapshotManifest{}, fmt.Errorf("%w: staged manifest differs from the generated manifest", ErrInvalidManifest)
	}

	browserRoot := filepath.Join(stagePath, browserDataDirectory)
	if err := inspectRealDirectory(browserRoot); err != nil {
		return LocalSnapshotManifest{}, err
	}
	expectedFiles := make(map[string]FileEntry, len(loaded.Files))
	for _, entry := range loaded.Files {
		expectedFiles[entry.Path] = entry
	}
	actual := make([]FileEntry, 0, len(loaded.Files))
	seen := make(map[string]struct{}, len(loaded.Files))
	err = filepath.WalkDir(browserRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == browserRoot {
			return nil
		}
		relative, err := filepath.Rel(browserRoot, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := ValidateRelativePath(relative, loaded.SourceOS); err != nil {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: staged snapshot contains a symbolic link", ErrInvalidManifest)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: staged snapshot contains a reparse point", ErrInvalidManifest)
		}
		if info.IsDir() {
			return nil
		}
		expectedEntry, exists := expectedFiles[relative]
		if !exists || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: staged snapshot contains unexpected entry %q", ErrInvalidManifest, relative)
		}
		digest, size, err := hashStableRegularFile(current)
		if err != nil {
			return err
		}
		if size != expectedEntry.Size || digest != expectedEntry.SHA256 {
			return fmt.Errorf("%w: staged file verification failed for %q", ErrInvalidManifest, relative)
		}
		actual = append(actual, FileEntry{Path: relative, Size: size, SHA256: digest})
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	if len(seen) != len(expectedFiles) {
		return LocalSnapshotManifest{}, fmt.Errorf("%w: staged snapshot is missing required files", ErrInvalidManifest)
	}
	treeDigest, err := ComputeTreeDigest(loaded.SourceOS, actual)
	if err != nil {
		return LocalSnapshotManifest{}, err
	}
	if treeDigest != loaded.TreeDigest {
		return LocalSnapshotManifest{}, fmt.Errorf("%w: staged snapshot tree digest changed", ErrInvalidManifest)
	}
	return loaded, nil
}

func hashStableRegularFile(filePath string) (string, int64, error) {
	before, err := inspectRegularPath(filePath)
	if err != nil {
		return "", 0, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	openedBefore, err := tokenFromOpenFile(file)
	if err != nil {
		return "", 0, err
	}
	if !openedBefore.Equal(before) {
		return "", 0, ErrSourceChanged
	}
	digest := sha256.New()
	buffer := make([]byte, SnapshotCopyBufferBytes)
	read, err := io.CopyBuffer(digest, io.LimitReader(file, MaxFileBytes+1), buffer)
	if err != nil {
		return "", read, err
	}
	if read != before.Size || read > MaxFileBytes {
		return "", read, ErrSourceChanged
	}
	openedAfter, err := tokenFromOpenFile(file)
	if err != nil {
		return "", read, err
	}
	if !openedAfter.Equal(openedBefore) {
		return "", read, ErrSourceChanged
	}
	return hex.EncodeToString(digest.Sum(nil)), read, nil
}

func readBoundedFile(filePath string, maximum int) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maximum {
		return nil, fmt.Errorf("%w: staged metadata size is outside bounds", ErrInvalidManifest)
	}
	return data, nil
}
