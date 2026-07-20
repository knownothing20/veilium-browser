package localrecovery

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type plannedSnapshotFile struct {
	RelativePath string
	SourcePath   string
	Size         int64
	Token        fileToken
}

type snapshotPlan struct {
	SourceRoot string
	Files      []plannedSnapshotFile
	TotalBytes int64
}

func (c *SnapshotCreator) preflight(ctx context.Context, record lifecycleRecordView, request SnapshotRequest) (snapshotPlan, error) {
	if err := c.checkCancellation(ctx, request.OperationID); err != nil {
		return snapshotPlan{}, err
	}
	if record.State != "available" {
		return snapshotPlan{}, fmt.Errorf("%w: Profile lifecycle state is not available", ErrInvalidRecord)
	}
	sourceRoot, err := managedSourcePath(c.dataRoot, record.ManagedDir)
	if err != nil {
		return snapshotPlan{}, err
	}
	if pathsOverlap(sourceRoot, c.recoveryRoot) {
		return snapshotPlan{}, fmt.Errorf("%w: snapshot source and destination overlap", ErrInvalidRecord)
	}
	if err := inspectRealDirectory(sourceRoot); err != nil {
		return snapshotPlan{}, err
	}

	plan := snapshotPlan{SourceRoot: sourceRoot}
	err = filepath.WalkDir(sourceRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := c.checkCancellation(ctx, request.OperationID); err != nil {
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
			return fmt.Errorf("%w: snapshot source contains a symbolic link at %q", ErrInvalidManifest, relative)
		}
		unsafe, err := pathHasReparsePoint(current)
		if err != nil {
			return fmt.Errorf("inspect snapshot source %q: %w", relative, err)
		}
		if unsafe {
			return fmt.Errorf("%w: snapshot source contains a reparse point at %q", ErrInvalidManifest, relative)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: snapshot source contains an unsupported special entry at %q", ErrInvalidManifest, relative)
		}
		if len(plan.Files) >= MaxFiles {
			return fmt.Errorf("%w: snapshot file count exceeds %d", ErrInvalidManifest, MaxFiles)
		}
		token, err := inspectRegularPath(current)
		if err != nil {
			return fmt.Errorf("inspect snapshot file %q: %w", relative, err)
		}
		if token.Size < 0 || token.Size > MaxFileBytes {
			return fmt.Errorf("%w: snapshot file %q exceeds the file-size bound", ErrInvalidManifest, relative)
		}
		if token.Size > MaxTotalBytes-plan.TotalBytes {
			return fmt.Errorf("%w: snapshot total bytes exceed the bound", ErrInvalidManifest)
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
		return snapshotPlan{}, err
	}
	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].RelativePath < plan.Files[j].RelativePath })

	required, err := requiredSnapshotSpace(plan.TotalBytes, len(request.ProfileDefinition))
	if err != nil {
		return snapshotPlan{}, err
	}
	available, err := c.space(c.recoveryRoot)
	if err != nil {
		return snapshotPlan{}, err
	}
	if available < required {
		return snapshotPlan{}, fmt.Errorf("%w: need %d bytes but only %d are available", ErrInsufficientSpace, required, available)
	}
	return plan, nil
}

func requiredSnapshotSpace(fileBytes int64, profileBytes int) (uint64, error) {
	if fileBytes < 0 || fileBytes > MaxTotalBytes || profileBytes < 0 || profileBytes > MaxProfileDefinitionBytes {
		return 0, fmt.Errorf("%w: snapshot resource estimate is outside bounds", ErrInvalidManifest)
	}
	base := uint64(fileBytes) + uint64(profileBytes) + uint64(MaxManifestBytes) + uint64(SnapshotSpaceReserve)
	if base < uint64(fileBytes) {
		return 0, fmt.Errorf("%w: snapshot resource estimate overflow", ErrInvalidManifest)
	}
	margin := base / 20
	if base > math.MaxUint64-margin {
		return math.MaxUint64, nil
	}
	return base + margin, nil
}
