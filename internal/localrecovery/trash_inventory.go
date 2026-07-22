package localrecovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type trashPlan struct {
	SourceRoot string
	Files      []plannedSnapshotFile
	TotalBytes int64
}

func planTrashTree(ctx context.Context, sourceRoot string, check func(context.Context) error) (trashPlan, error) {
	if err := check(ctx); err != nil {
		return trashPlan{}, err
	}
	if err := inspectRealDirectory(sourceRoot); err != nil {
		return trashPlan{}, err
	}
	plan := trashPlan{SourceRoot: sourceRoot}
	err := filepath.WalkDir(sourceRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := check(ctx); err != nil {
			return err
		}
		if current == sourceRoot {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, current)
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
			return fmt.Errorf("%w: Profile data contains a symbolic link at %q", ErrInvalidManifest, relative)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return fmt.Errorf("inspect Profile data %q: %w", relative, err)
		}
		if unsafe {
			return fmt.Errorf("%w: Profile data contains a reparse point at %q", ErrInvalidManifest, relative)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: Profile data contains an unsupported special entry at %q", ErrInvalidManifest, relative)
		}
		if len(plan.Files) >= MaxFiles {
			return fmt.Errorf("%w: Profile data file count exceeds %d", ErrInvalidManifest, MaxFiles)
		}
		token, err := inspectRegularPath(current)
		if err != nil {
			return fmt.Errorf("inspect Profile data file %q: %w", relative, err)
		}
		if token.Size < 0 || token.Size > MaxFileBytes {
			return fmt.Errorf("%w: Profile data file %q exceeds the file-size bound", ErrInvalidManifest, relative)
		}
		if token.Size > MaxTotalBytes-plan.TotalBytes {
			return fmt.Errorf("%w: Profile data total bytes exceed the bound", ErrInvalidManifest)
		}
		plan.TotalBytes += token.Size
		plan.Files = append(plan.Files, plannedSnapshotFile{
			RelativePath: relative,
			SourcePath:   current,
			Size:         token.Size,
			Token:        token,
		})
		return nil
	})
	if err != nil {
		return trashPlan{}, err
	}
	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].RelativePath < plan.Files[j].RelativePath })
	return plan, nil
}

func verifyMovedTrashTree(ctx context.Context, movedRoot string, plan trashPlan, check func(context.Context) error) ([]FileEntry, string, error) {
	if err := inspectRealDirectory(movedRoot); err != nil {
		return nil, "", err
	}
	expected := make(map[string]plannedSnapshotFile, len(plan.Files))
	for _, item := range plan.Files {
		expected[item.RelativePath] = item
	}
	entries := make([]FileEntry, 0, len(plan.Files))
	seen := make(map[string]struct{}, len(plan.Files))
	err := filepath.WalkDir(movedRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := check(ctx); err != nil {
			return err
		}
		if current == movedRoot {
			return nil
		}
		relative, err := filepath.Rel(movedRoot, current)
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
			return fmt.Errorf("%w: moved trash data contains a symbolic link", ErrInvalidManifest)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: moved trash data contains a reparse point", ErrInvalidManifest)
		}
		if info.IsDir() {
			return nil
		}
		planned, exists := expected[relative]
		if !exists || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: moved trash data contains unexpected entry %q", ErrInvalidManifest, relative)
		}
		currentToken, err := inspectRegularPath(current)
		if err != nil {
			return err
		}
		if !currentToken.Equal(planned.Token) {
			return fmt.Errorf("%w: moved Profile data changed at %q", ErrSourceChanged, relative)
		}
		digest, size, err := hashStableRegularFile(current)
		if err != nil {
			return err
		}
		if size != planned.Size {
			return fmt.Errorf("%w: moved Profile data size changed at %q", ErrSourceChanged, relative)
		}
		entries = append(entries, FileEntry{Path: relative, Size: size, SHA256: digest})
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(seen) != len(expected) {
		return nil, "", fmt.Errorf("%w: moved Profile data is missing files", ErrInvalidManifest)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	treeDigest, err := ComputeTreeDigest(runtime.GOOS, entries)
	if err != nil {
		return nil, "", err
	}
	return entries, treeDigest, nil
}

func verifyStoredTrash(record TrashRecord, trashRoot string) error {
	if record.Status == TrashDeleted {
		return fmt.Errorf("%w: deleted trash data cannot be verified as stored", ErrInvalidRecord)
	}
	if !record.DataPresent {
		return fmt.Errorf("%w: recoverable trash record has no data", ErrInvalidRecord)
	}
	if err := inspectRealDirectory(trashRoot); err != nil {
		return err
	}
	profilePath := filepath.Join(trashRoot, profileDefinitionName)
	profileData, err := readBoundedFile(profilePath, MaxProfileDefinitionBytes)
	if err != nil {
		return fmt.Errorf("read trashed Profile definition: %w", err)
	}
	if err := validateProfileDefinitionExclusions(profileData); err != nil {
		return err
	}
	profileDigest, err := DigestProfileDefinition(profileData)
	if err != nil {
		return err
	}
	if profileDigest != record.ProfileDefinitionDigest {
		return fmt.Errorf("%w: trashed Profile definition digest changed", ErrInvalidManifest)
	}
	browserRoot := filepath.Join(trashRoot, browserDataDirectory)
	plan, err := planTrashTree(context.Background(), browserRoot, func(context.Context) error { return nil })
	if err != nil {
		return err
	}
	entries := make([]FileEntry, 0, len(plan.Files))
	for _, item := range plan.Files {
		digest, size, err := hashStableRegularFile(item.SourcePath)
		if err != nil {
			return err
		}
		entries = append(entries, FileEntry{Path: item.RelativePath, Size: size, SHA256: digest})
	}
	treeDigest, err := ComputeTreeDigest(runtime.GOOS, entries)
	if err != nil {
		return err
	}
	if treeDigest != record.TreeDigest || int64(len(entries)) != record.FileCount || plan.TotalBytes != record.TotalBytes {
		return fmt.Errorf("%w: trashed browser data summary changed", ErrInvalidManifest)
	}
	return nil
}

func removeOwnedTrashTree(target, boundary string) error {
	if !pathContainedBy(target, boundary) || filepath.Clean(target) == filepath.Clean(boundary) {
		return fmt.Errorf("%w: trash cleanup target is outside the owned boundary", ErrLifecycleStorageRecoveryRequired)
	}
	if _, err := os.Lstat(target); os.IsNotExist(err) {
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
			return fmt.Errorf("%w: owned trash tree contains a symbolic link", ErrLifecycleStorageRecoveryRequired)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return err
		}
		if unsafe {
			return fmt.Errorf("%w: owned trash tree contains a reparse point", ErrLifecycleStorageRecoveryRequired)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("%w: owned trash tree contains an unsupported special entry", ErrLifecycleStorageRecoveryRequired)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(target)
}

func hashTrashPlan(plan trashPlan) ([]FileEntry, string, error) {
	entries := make([]FileEntry, 0, len(plan.Files))
	for _, item := range plan.Files {
		current, err := inspectRegularPath(item.SourcePath)
		if err != nil {
			return nil, "", err
		}
		if !current.Equal(item.Token) {
			return nil, "", fmt.Errorf("%w: Profile data changed at %q", ErrSourceChanged, item.RelativePath)
		}
		digest, size, err := hashStableRegularFile(item.SourcePath)
		if err != nil {
			return nil, "", err
		}
		if size != item.Size {
			return nil, "", fmt.Errorf("%w: Profile data size changed at %q", ErrSourceChanged, item.RelativePath)
		}
		after, err := inspectRegularPath(item.SourcePath)
		if err != nil {
			return nil, "", err
		}
		if !after.Equal(item.Token) {
			return nil, "", fmt.Errorf("%w: Profile data changed while hashing %q", ErrSourceChanged, item.RelativePath)
		}
		entries = append(entries, FileEntry{Path: item.RelativePath, Size: size, SHA256: digest})
	}
	treeDigest, err := ComputeTreeDigest(runtime.GOOS, entries)
	if err != nil {
		return nil, "", err
	}
	return entries, treeDigest, nil
}
