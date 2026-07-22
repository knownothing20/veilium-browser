package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestArchiveBlockersPreventOperationCreation(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*archiveBlockers)
	}{
		{name: "active browser", configure: func(blockers *archiveBlockers) { blockers.active = true }},
		{name: "dependent operation", configure: func(blockers *archiveBlockers) { blockers.dependent = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
			test.configure(harness.blockers)
			if _, err := harness.executor.Archive(context.Background(), harness.request); !errors.Is(err, lifecycle.ErrConflict) {
				t.Fatalf("blocker did not reject archive: %v", err)
			}
			if operations := harness.journal.List(); len(operations) != 0 {
				t.Fatalf("blocked archive created an operation: %#v", operations)
			}
			record := assertArchiveUnlocked(t, harness)
			if record.State != lifecycle.StateAvailable {
				t.Fatalf("blocked archive changed lifecycle state: %#v", record)
			}
		})
	}
}

func TestArchiveCancellationBeforeCommitPreservesState(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	harness.executor.journal = &cancellingArchiveJournal{delegate: harness.journal}

	result, err := harness.executor.Archive(context.Background(), harness.request)
	if !errors.Is(err, ErrLifecycleStorageCancelled) {
		t.Fatalf("archive cancellation was not returned: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationCancelled {
		t.Fatalf("cancelled archive produced the wrong operation state: %#v", result.Operation)
	}
	record := assertArchiveUnlocked(t, harness)
	if record.State != lifecycle.StateAvailable || record.ArchivedAt != nil {
		t.Fatalf("cancelled archive changed lifecycle state: %#v", record)
	}
}

func TestArchivePersistenceFailureRollsBackState(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	harness.executor.records = &failingArchiveRecordStore{delegate: harness.records, failUpdate: true}

	result, err := harness.executor.Archive(context.Background(), harness.request)
	if err == nil || errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("persistence failure should fail cleanly: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("persistence failure produced the wrong operation state: %#v", result.Operation)
	}
	record := assertArchiveUnlocked(t, harness)
	if record.State != lifecycle.StateAvailable || record.ArchivedAt != nil {
		t.Fatalf("persistence failure changed lifecycle state: %#v", record)
	}
}

func TestArchiveFinalizationFailureRecordsRecoveryState(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	harness.executor.coordinator = &failingArchiveCoordinator{
		delegate: harness.coordinator,
		finish:   errors.New("simulated operation finalization failure"),
	}

	result, err := harness.executor.Archive(context.Background(), harness.request)
	if !errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("finalization failure did not require recovery: %v", err)
	}
	if result.Record.State != lifecycle.StateArchived || result.Record.ArchivedAt == nil {
		t.Fatalf("committed archive state was lost: %#v", result.Record)
	}
	if result.Record.Lock == nil || result.Record.Lock.OperationID != harness.request.OperationID {
		t.Fatalf("ambiguous operation lock was silently released: %#v", result.Record.Lock)
	}
	if !hasLifecycleCode(result.Record.RecoveryCodes, "archive-operation-finalization-required") {
		t.Fatalf("finalization recovery code was not recorded: %#v", result.Record.RecoveryCodes)
	}
}

func TestUnarchiveRejectsUnsafeManagedLocation(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	if _, err := harness.executor.Archive(context.Background(), harness.request); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(harness.profileRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness.profileRoot, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	request := harness.request
	request.OperationID = "unarchive-unsafe-location"
	request.IdempotencyKey = "unarchive-unsafe-location"
	result, err := harness.executor.Unarchive(context.Background(), request)
	if !errors.Is(err, lifecycle.ErrConflict) {
		t.Fatalf("unsafe managed location was not rejected: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("unsafe unarchive produced the wrong operation state: %#v", result.Operation)
	}
	record := assertArchiveUnlocked(t, harness)
	if record.State != lifecycle.StateArchived || record.ArchivedAt == nil {
		t.Fatalf("unsafe unarchive changed archived state: %#v", record)
	}
}

func TestUnarchiveContradictoryOriginRequiresRecovery(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	record, err := harness.records.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	record.State = lifecycle.StateArchived
	record.ArchivedAt = &now
	record.LimitationCodes = []string{"archive-origin-available", "archive-origin-draft", "profile-archived"}
	if _, err := harness.records.Update(record); err != nil {
		t.Fatal(err)
	}

	request := harness.request
	request.OperationID = "unarchive-contradictory-origin"
	request.IdempotencyKey = "unarchive-contradictory-origin"
	result, err := harness.executor.Unarchive(context.Background(), request)
	if !errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("contradictory archive origin did not require recovery: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationRecoveryRequired {
		t.Fatalf("contradictory archive origin produced the wrong operation state: %#v", result.Operation)
	}
	record = assertArchiveUnlocked(t, harness)
	if record.State != lifecycle.StateArchived || !hasLifecycleCode(record.RecoveryCodes, "lifecycle-storage-recovery-required") {
		t.Fatalf("contradictory archive state was not preserved for recovery: %#v", record)
	}
}

func TestArchiveAllowsProfileWithoutMaterializedBrowserDirectory(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	if err := os.RemoveAll(filepath.Join(harness.dataRoot, "profiles")); err != nil {
		t.Fatal(err)
	}
	result, err := harness.executor.Archive(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record.State != lifecycle.StateArchived {
		t.Fatalf("metadata-only Profile was not archived: %#v", result.Record)
	}
	if _, err := os.Stat(filepath.Join(harness.dataRoot, "profiles")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("archive materialized a missing browser directory: %v", err)
	}
}
