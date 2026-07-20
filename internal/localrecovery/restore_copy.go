package localrecovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (e *RestoreExecutor) copyRestoreFiles(ctx context.Context, request RestoreRequest, source verifiedRestoreSource, stagePath string) (snapshotCounters, error) {
	browserDestination := filepath.Join(stagePath, browserDataDirectory)
	counters := snapshotCounters{}
	e.reportProgress(RestoreProgress{
		Stage:      RestoreStageCopying,
		FilesTotal: source.Manifest.FileCount,
		BytesTotal: source.Manifest.TotalBytes,
	})
	for index, planned := range source.Plan.Files {
		if err := e.checkCancellation(ctx, request.OperationID); err != nil {
			return counters, err
		}
		expected := source.Manifest.Files[index]
		if planned.RelativePath != expected.Path || planned.Size != expected.Size {
			return counters, fmt.Errorf("%w: restore plan no longer matches the manifest", ErrSnapshotUnavailable)
		}
		current, err := inspectRegularPath(planned.SourcePath)
		if err != nil {
			return counters, err
		}
		if !current.Equal(planned.Token) {
			return counters, fmt.Errorf("%w: snapshot source changed for %q", ErrSourceChanged, planned.RelativePath)
		}
		destination := filepath.Join(browserDestination, filepath.FromSlash(planned.RelativePath))
		digest, copied, err := copyStableRegularFile(planned.SourcePath, destination, planned.Token)
		if err != nil {
			return counters, fmt.Errorf("copy restored file %q: %w", planned.RelativePath, err)
		}
		if digest != expected.SHA256 || copied != expected.Size {
			return counters, fmt.Errorf("%w: restored copy differs for %q", ErrSourceChanged, planned.RelativePath)
		}
		counters.files++
		counters.bytes += copied
		e.reportProgress(RestoreProgress{
			Stage:          RestoreStageCopying,
			FilesProcessed: counters.files,
			FilesTotal:     source.Manifest.FileCount,
			BytesProcessed: counters.bytes,
			BytesTotal:     source.Manifest.TotalBytes,
		})
	}
	return counters, nil
}

func verifyRestoredBrowserData(browserRoot string, manifest LocalSnapshotManifest) error {
	if err := inspectRealDirectory(browserRoot); err != nil {
		return err
	}
	expected := make(map[string]FileEntry, len(manifest.Files))
	for _, entry := range manifest.Files {
		expected[entry.Path] = entry
	}
	seen := make(map[string]struct{}, len(expected))
	actual := make([]FileEntry, 0, len(expected))
	err := filepath.WalkDir(browserRoot, func(current string, entry fs.DirEntry, walkErr error) error {
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
		if err := ValidateRelativePath(relative, manifest.SourceOS); err != nil {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: restored staging contains a symbolic link", ErrInvalidManifest)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: restored staging contains a reparse point", ErrInvalidManifest)
		}
		if info.IsDir() {
			return nil
		}
		expectedEntry, exists := expected[relative]
		if !exists || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: restored staging contains unexpected entry %q", ErrInvalidManifest, relative)
		}
		digest, size, err := hashStableRegularFile(current)
		if err != nil {
			return err
		}
		if size != expectedEntry.Size || digest != expectedEntry.SHA256 {
			return fmt.Errorf("%w: restored file verification failed for %q", ErrInvalidManifest, relative)
		}
		actual = append(actual, FileEntry{Path: relative, Size: size, SHA256: digest})
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("%w: restored staging is missing required files", ErrInvalidManifest)
	}
	treeDigest, err := ComputeTreeDigest(manifest.SourceOS, actual)
	if err != nil {
		return err
	}
	if treeDigest != manifest.TreeDigest {
		return fmt.Errorf("%w: restored staging tree digest changed", ErrInvalidManifest)
	}
	return nil
}
