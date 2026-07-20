package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestRestoreCancelsBetweenFiles(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{
		"a.txt": strings.Repeat("a", 64),
		"b.txt": strings.Repeat("b", 64),
	})
	requested := false
	harness.executor.SetProgressCallback(func(progress RestoreProgress) {
		if !requested && progress.Stage == RestoreStageCopying && progress.FilesProcessed == 1 {
			requested = true
			_, _, _ = harness.snapshot.coordinator.RequestCancellation(harness.request.OperationID)
		}
	})

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if !errors.Is(err, ErrRestoreCancelled) {
		t.Fatalf("restore cancellation was not returned: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationCancelled {
		t.Fatalf("cancelled restore has wrong operation state: %#v", result.Operation)
	}
	assertRestoreRolledBack(t, harness)
}

func TestRestoreRejectsTamperedVerifiedSnapshot(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "original"})
	snapshotFile := filepath.Join(
		snapshotFinalPath(harness.executor.recoveryRoot, harness.request.SnapshotID),
		browserDataDirectory,
		"file.txt",
	)
	if err := os.WriteFile(snapshotFile, []byte("modified"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if !errors.Is(err, ErrSnapshotUnavailable) {
		t.Fatalf("tampered snapshot was not rejected: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("tampered snapshot produced wrong operation state: %#v", result.Operation)
	}
	assertRestoreRolledBack(t, harness)
}
