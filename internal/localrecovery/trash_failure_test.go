package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestTrashRejectsActiveAndDependentWork(t *testing.T) {
	for name, configure := range map[string]func(*trashBlockers){
		"active":    func(blockers *trashBlockers) { blockers.active = true },
		"dependent": func(blockers *trashBlockers) { blockers.dependent = true },
	} {
		t.Run(name, func(t *testing.T) {
			harness := newTrashHarness(t, lifecycle.StateAvailable)
			configure(harness.blockers)
			if _, err := harness.executor.Trash(context.Background(), harness.request); !errors.Is(err, lifecycle.ErrConflict) {
				t.Fatalf("expected conflict, got %v", err)
			}
			if data, err := os.ReadFile(filepath.Join(harness.profileRoot, "sentinel.txt")); err != nil || string(data) != "keep" {
				t.Fatalf("blocked trash changed source data: %q, %v", string(data), err)
			}
			if len(harness.trash.List()) != 0 {
				t.Fatalf("blocked trash created catalog records: %#v", harness.trash.List())
			}
		})
	}
}

func TestTrashCancellationLeavesSourceUntouched(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	harness.executor.journal = &cancellingTrashJournal{delegate: harness.journal}
	if _, err := harness.executor.Trash(context.Background(), harness.request); !errors.Is(err, ErrLifecycleStorageCancelled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	record := assertTrashUnlocked(t, harness)
	if record.State != lifecycle.StateAvailable {
		t.Fatalf("cancelled trash changed lifecycle state: %#v", record)
	}
	if _, err := os.Stat(harness.profileRoot); err != nil {
		t.Fatalf("cancelled trash removed source directory: %v", err)
	}
	if len(harness.trash.List()) != 0 {
		t.Fatalf("cancelled trash retained catalog records: %#v", harness.trash.List())
	}
}

func TestTrashRenameFailureRollsBackCatalogAndLock(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	harness.executor.rename = func(string, string) error { return errors.New("simulated rename failure") }
	if _, err := harness.executor.Trash(context.Background(), harness.request); err == nil {
		t.Fatal("expected rename failure")
	}
	record := assertTrashUnlocked(t, harness)
	if record.State != lifecycle.StateAvailable {
		t.Fatalf("rename failure changed lifecycle state: %#v", record)
	}
	if _, err := os.Stat(harness.profileRoot); err != nil {
		t.Fatalf("rename failure removed source directory: %v", err)
	}
	if len(harness.trash.List()) != 0 {
		t.Fatalf("rename failure retained catalog records: %#v", harness.trash.List())
	}
}

func TestTrashLifecyclePersistenceFailureRestoresSource(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	harness.executor.records = &failingTrashRecordStore{delegate: harness.records, failUpdate: true}
	if _, err := harness.executor.Trash(context.Background(), harness.request); err == nil {
		t.Fatal("expected lifecycle persistence failure")
	}
	if _, err := os.Stat(harness.profileRoot); err != nil {
		t.Fatalf("lifecycle failure did not restore source data: %v", err)
	}
	if len(harness.trash.List()) != 0 {
		t.Fatalf("lifecycle failure retained trash catalog: %#v", harness.trash.List())
	}
	actual, err := harness.records.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.State != lifecycle.StateAvailable {
		t.Fatalf("lifecycle failure changed persisted state: %#v", actual)
	}
}

func TestRestoreTrashRejectsOccupiedOriginalPath(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(harness.profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harness.profileRoot, "foreign.txt"), []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = harness.executor.RestoreTrash(context.Background(), TrashActionRequest{
		OperationID:        "restore-trash-conflict-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "restore-trash-conflict-request-a",
		ApplicationVersion: "0.15.0-dev",
	})
	if !errors.Is(err, lifecycle.ErrConflict) {
		t.Fatalf("expected restore conflict, got %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(harness.profileRoot, "foreign.txt")); err != nil || string(data) != "foreign" {
		t.Fatalf("restore conflict changed occupying data: %q, %v", string(data), err)
	}
	stored, err := harness.trash.Get(trashed.Trash.TrashID)
	if err != nil || stored.Status != TrashStored {
		t.Fatalf("restore conflict changed trash catalog: %#v, %v", stored, err)
	}
}

func TestPermanentDeleteRequiresExactConfirmation(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	_, err = harness.executor.PermanentDelete(context.Background(), TrashActionRequest{
		OperationID:        "permanent-delete-confirmation-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "permanent-delete-confirmation-request-a",
		ApplicationVersion: "0.15.0-dev",
		Confirmation:       "wrong-profile",
	})
	if !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("expected invalid confirmation, got %v", err)
	}
	stored, err := harness.trash.Get(trashed.Trash.TrashID)
	if err != nil || stored.Status != TrashStored {
		t.Fatalf("invalid confirmation changed trash state: %#v, %v", stored, err)
	}
}

func TestPermanentDeleteFailureBecomesRecoveryRequired(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	harness.executor.removeTree = func(target, boundary string) error {
		if filepath.Base(target) == browserDataDirectory {
			return errors.New("simulated irreversible cleanup failure")
		}
		return removeOwnedTrashTree(target, boundary)
	}
	_, err = harness.executor.PermanentDelete(context.Background(), TrashActionRequest{
		OperationID:        "permanent-delete-failure-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "permanent-delete-failure-request-a",
		ApplicationVersion: "0.15.0-dev",
		Confirmation:       harness.request.ProfileID,
	})
	if !errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("expected recovery-required failure, got %v", err)
	}
	current, getErr := harness.trash.Get(trashed.Trash.TrashID)
	if getErr != nil || current.Status != TrashRecoveryRequired {
		t.Fatalf("cleanup failure was not preserved for recovery: %#v, %v", current, getErr)
	}
	record := assertTrashUnlocked(t, harness)
	if !containsCode(record.RecoveryCodes, "permanent-delete-recovery-required") {
		t.Fatalf("cleanup failure recovery code missing: %#v", record.RecoveryCodes)
	}
}

func TestRestoreTrashRejectsChangedRetainedProfileMetadata(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	item, err := harness.profiles.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	item.Notes = "changed while trashed"
	if _, err := harness.profiles.Update(item); err != nil {
		t.Fatal(err)
	}
	_, err = harness.executor.RestoreTrash(context.Background(), TrashActionRequest{
		OperationID:        "restore-trash-metadata-changed-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "restore-trash-metadata-changed-request-a",
		ApplicationVersion: "0.15.0-dev",
	})
	if !errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("expected changed Profile metadata to fail closed, got %v", err)
	}
	stored, getErr := harness.trash.Get(trashed.Trash.TrashID)
	if getErr != nil || stored.Status != TrashStored {
		t.Fatalf("changed metadata disturbed recoverable trash: %#v, %v", stored, getErr)
	}
	if _, statErr := os.Stat(trashRootPath(filepath.Join(harness.dataRoot, "local-recovery"), trashed.Trash.TrashID)); statErr != nil {
		t.Fatalf("changed metadata removed recoverable trash: %v", statErr)
	}
}

func TestTrashCatalogRemovalFailurePreservesRecoveryRecord(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	harness.executor.trash = &failingTrashCatalog{delegate: harness.trash, failRemove: true}
	harness.executor.rename = func(string, string) error { return errors.New("simulated rename failure") }
	_, err := harness.executor.Trash(context.Background(), harness.request)
	if !errors.Is(err, ErrLifecycleStorageRecoveryRequired) {
		t.Fatalf("expected catalog cleanup recovery state, got %v", err)
	}
	items := harness.trash.List()
	if len(items) != 1 || items[0].Status != TrashRecoveryRequired {
		t.Fatalf("failed catalog cleanup lost recovery metadata: %#v", items)
	}
	if _, statErr := os.Stat(harness.profileRoot); statErr != nil {
		t.Fatalf("failed catalog cleanup changed source Profile: %v", statErr)
	}
}
