package localrecovery

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestSnapshotCreatorRollsBackPublicationFailure(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	harness.creator.rename = func(string, string) error { return errors.New("simulated rename failure") }

	result, err := harness.creator.Create(context.Background(), harness.request)
	if err == nil || errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("publication failure should roll back cleanly: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed || result.Catalog.Status != SnapshotInvalid {
		t.Fatalf("publication rollback was not recorded: %#v %#v", result.Operation, result.Catalog)
	}
	if _, err := os.Stat(snapshotStagePath(harness.creator.recoveryRoot, harness.request.OperationID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rolled-back staging remains: %v", err)
	}
	assertProfileUnlocked(t, harness)
}

func TestSnapshotCreatorPreservesRecoveryStateWhenCleanupFails(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	harness.creator.rename = func(string, string) error { return errors.New("simulated rename failure") }
	harness.creator.removeStage = func(string, string) error { return errors.New("simulated cleanup failure") }

	result, err := harness.creator.Create(context.Background(), harness.request)
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("cleanup failure did not require recovery: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationRecoveryRequired || result.Catalog.Status != SnapshotRecoveryRequired {
		t.Fatalf("recovery state was not recorded: %#v %#v", result.Operation, result.Catalog)
	}
	if _, err := os.Stat(snapshotStagePath(harness.creator.recoveryRoot, harness.request.OperationID)); err != nil {
		t.Fatalf("recovery staging was not preserved: %v", err)
	}
	if len(result.Operation.Items) != 1 || result.Operation.Items[0].RecoveryID != snapshotStagingRef(harness.request.OperationID) {
		t.Fatalf("recovery identity was not recorded: %#v", result.Operation.Items)
	}
	assertProfileUnlocked(t, harness)
}
