package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestSnapshotCreatorPublishesVerifiedSnapshot(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{
		"Default/Preferences": `{"theme":"dark"}`,
		"Local State":         `{"browser":"state"}`,
	})
	var last SnapshotProgress
	harness.creator.SetProgressCallback(func(progress SnapshotProgress) { last = progress })

	result, err := harness.creator.Create(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation.Status != lifecycle.OperationCompleted || result.Catalog.Status != SnapshotVerified {
		t.Fatalf("snapshot was not completed and verified: %#v %#v", result.Operation, result.Catalog)
	}
	if result.Manifest.FileCount != 2 || result.Manifest.TotalBytes == 0 || result.PublishedRef != "local-recovery/snapshots/snapshot-a" {
		t.Fatalf("unexpected snapshot result: %#v", result)
	}
	if last.Stage != SnapshotStageFinished || last.FilesProcessed != 2 {
		t.Fatalf("unexpected final progress: %#v", last)
	}
	published := snapshotFinalPath(harness.creator.recoveryRoot, harness.request.SnapshotID)
	for relative, expected := range map[string]string{
		"Default/Preferences": `{"theme":"dark"}`,
		"Local State":         `{"browser":"state"}`,
	} {
		data, err := os.ReadFile(filepath.Join(published, browserDataDirectory, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != expected {
			t.Fatalf("published file %q changed", relative)
		}
	}
	if _, err := os.Stat(snapshotStagePath(harness.creator.recoveryRoot, harness.request.OperationID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging remained after successful publication: %v", err)
	}
	assertProfileUnlocked(t, harness)

	reused, err := harness.creator.Create(context.Background(), harness.request)
	if err != nil {
		t.Fatalf("idempotent snapshot retry failed: %v", err)
	}
	if reused.Operation.ID != result.Operation.ID || reused.Catalog.Revision != result.Catalog.Revision {
		t.Fatalf("idempotent retry did not return the existing result: %#v", reused)
	}
}

func TestSnapshotOperationIDCannotBeReusedForAnotherSnapshot(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	if _, err := harness.creator.Create(context.Background(), harness.request); err != nil {
		t.Fatal(err)
	}
	conflict := harness.request
	conflict.SnapshotID = "snapshot-b"
	if _, err := harness.creator.Create(context.Background(), conflict); !errors.Is(err, lifecycle.ErrConflict) {
		t.Fatalf("operation id was reused for a different snapshot: %v", err)
	}
}
