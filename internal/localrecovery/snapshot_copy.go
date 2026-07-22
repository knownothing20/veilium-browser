package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type snapshotCounters struct {
	files int64
	bytes int64
}

func (c *SnapshotCreator) createStage(operationID string) (string, error) {
	stagingRoot := filepath.Join(c.recoveryRoot, snapshotStagingDirectory)
	if err := ensurePrivateDirectoryTree(stagingRoot); err != nil {
		return "", err
	}
	stagePath := snapshotStagePath(c.recoveryRoot, operationID)
	if err := os.Mkdir(stagePath, 0o700); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%w: operation staging already exists", ErrRecoveryRequired)
		}
		return "", fmt.Errorf("create snapshot staging: %w", err)
	}
	if err := os.Mkdir(filepath.Join(stagePath, browserDataDirectory), 0o700); err != nil {
		return "", fmt.Errorf("create snapshot browser-data staging: %w", err)
	}
	return stagePath, nil
}

func (c *SnapshotCreator) copySnapshotFiles(ctx context.Context, request SnapshotRequest, plan snapshotPlan, stagePath string) ([]FileEntry, snapshotCounters, error) {
	if err := writeExclusiveFile(filepath.Join(stagePath, profileDefinitionName), request.ProfileDefinition); err != nil {
		return nil, snapshotCounters{}, err
	}
	entries := make([]FileEntry, 0, len(plan.Files))
	counters := snapshotCounters{}
	c.reportProgress(SnapshotProgress{
		Stage:      SnapshotStageCopying,
		FilesTotal: int64(len(plan.Files)),
		BytesTotal: plan.TotalBytes,
	})
	for _, planned := range plan.Files {
		if err := c.checkCancellation(ctx, request.OperationID); err != nil {
			return nil, counters, err
		}
		if err := inspectDirectoryTree(plan.SourceRoot, filepath.Dir(planned.SourcePath)); err != nil {
			return nil, counters, err
		}
		current, err := inspectRegularPath(planned.SourcePath)
		if err != nil {
			return nil, counters, err
		}
		if !current.Equal(planned.Token) {
			return nil, counters, fmt.Errorf("%w: %q changed after preflight", ErrSourceChanged, planned.RelativePath)
		}
		destination := filepath.Join(stagePath, browserDataDirectory, filepath.FromSlash(planned.RelativePath))
		digest, copied, err := copyStableRegularFile(planned.SourcePath, destination, planned.Token)
		if err != nil {
			return nil, counters, fmt.Errorf("copy snapshot file %q: %w", planned.RelativePath, err)
		}
		counters.files++
		counters.bytes += copied
		entries = append(entries, FileEntry{Path: planned.RelativePath, Size: copied, SHA256: digest})
		c.reportProgress(SnapshotProgress{
			Stage:          SnapshotStageCopying,
			FilesProcessed: counters.files,
			FilesTotal:     int64(len(plan.Files)),
			BytesProcessed: counters.bytes,
			BytesTotal:     plan.TotalBytes,
		})
	}
	return entries, counters, nil
}

func copyStableRegularFile(sourcePath, destinationPath string, expected fileToken) (string, int64, error) {
	if err := ensurePrivateDirectoryTree(filepath.Dir(destinationPath)); err != nil {
		return "", 0, err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", 0, err
	}
	defer source.Close()
	openedBefore, err := tokenFromOpenFile(source)
	if err != nil {
		return "", 0, err
	}
	if !openedBefore.Equal(expected) {
		return "", 0, ErrSourceChanged
	}

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, err
	}
	keepDestination := false
	defer func() {
		_ = destination.Close()
		if !keepDestination {
			_ = os.Remove(destinationPath)
		}
	}()

	digest := sha256.New()
	buffer := make([]byte, SnapshotCopyBufferBytes)
	written, err := io.CopyBuffer(io.MultiWriter(destination, digest), io.LimitReader(source, MaxFileBytes+1), buffer)
	if err != nil {
		return "", written, err
	}
	if written != expected.Size || written > MaxFileBytes {
		return "", written, ErrSourceChanged
	}
	if err := destination.Sync(); err != nil {
		return "", written, fmt.Errorf("sync staged snapshot file: %w", err)
	}
	if err := destination.Close(); err != nil {
		return "", written, fmt.Errorf("close staged snapshot file: %w", err)
	}
	openedAfter, err := tokenFromOpenFile(source)
	if err != nil {
		return "", written, err
	}
	if !openedAfter.Equal(openedBefore) {
		return "", written, ErrSourceChanged
	}
	pathAfter, err := inspectRegularPath(sourcePath)
	if err != nil {
		return "", written, err
	}
	if !pathAfter.Equal(openedBefore) {
		return "", written, ErrSourceChanged
	}
	keepDestination = true
	return hex.EncodeToString(digest.Sum(nil)), written, nil
}

func writeExclusiveFile(filePath string, data []byte) error {
	if len(data) == 0 || len(data) > MaxProfileDefinitionBytes {
		return fmt.Errorf("%w: staged metadata size is outside bounds", ErrInvalidManifest)
	}
	if err := ensurePrivateDirectoryTree(filepath.Dir(filePath)); err != nil {
		return err
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(filePath)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	keep = true
	return nil
}

func (c *SnapshotCreator) reportProgress(progress SnapshotProgress) {
	if c.progress != nil {
		c.progress(progress)
	}
}
