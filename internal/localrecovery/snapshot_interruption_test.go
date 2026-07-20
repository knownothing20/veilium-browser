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

func TestSnapshotCreatorCancelsBetweenFiles(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{
		"a.txt": strings.Repeat("a", 64),
		"b.txt": strings.Repeat("b", 64),
	})
	requested := false
	harness.creator.SetProgressCallback(func(progress SnapshotProgress) {
		if !requested && progress.Stage == SnapshotStageCopying && progress.FilesProcessed == 1 {
			requested = true
			_, _, _ = harness.coordinator.RequestCancellation(harness.request.OperationID)
		}
	})

	result, err := harness.creator.Create(context.Background(), harness.request)
	if !errors.Is(err, ErrSnapshotCancelled) {
		t.Fatalf("snapshot cancellation was not returned: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationCancelled {
		t.Fatalf("cancelled snapshot has wrong operation state: %#v", result.Operation)
	}
	if _, err := os.Stat(snapshotFinalPath(harness.creator.recoveryRoot, harness.request.SnapshotID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled snapshot was published: %v", err)
	}
	assertProfileUnlocked(t, harness)
}

func TestSnapshotCreatorDetectsSourceChange(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{
		"a.txt": "first",
		"b.txt": "second",
	})
	changed := false
	harness.creator.SetProgressCallback(func(progress SnapshotProgress) {
		if !changed && progress.Stage == SnapshotStageCopying && progress.FilesProcessed == 1 {
			changed = true
			_ = os.WriteFile(filepath.Join(harness.profileRoot, "b.txt"), []byte("changed"), 0o600)
		}
	})

	result, err := harness.creator.Create(context.Background(), harness.request)
	if !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("source change was not detected: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed || len(result.Operation.Items) != 1 || result.Operation.Items[0].ReasonCode != "snapshot-source-changed" {
		t.Fatalf("source change produced the wrong result: %#v", result.Operation)
	}
	assertProfileUnlocked(t, harness)
}

func TestSnapshotCreatorChecksDestinationSpace(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	harness.creator.space = func(string) (uint64, error) { return 0, nil }

	result, err := harness.creator.Create(context.Background(), harness.request)
	if !errors.Is(err, ErrInsufficientSpace) {
		t.Fatalf("insufficient destination space was not rejected: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("space failure produced the wrong operation state: %#v", result.Operation)
	}
	assertProfileUnlocked(t, harness)
}
