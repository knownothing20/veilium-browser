package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestTrashAndRestoreRoundTrip(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	result, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation.Status != lifecycle.OperationCompleted || result.Record.State != lifecycle.StateTrashed || result.Trash.Status != TrashStored {
		t.Fatalf("unexpected trash result: %#v", result)
	}
	if _, err := os.Lstat(harness.profileRoot); !os.IsNotExist(err) {
		t.Fatalf("source Profile directory still exists after trash: %v", err)
	}
	trashRoot := trashRootPath(filepath.Join(harness.dataRoot, "local-recovery"), result.Trash.TrashID)
	if data, err := os.ReadFile(filepath.Join(trashRoot, browserDataDirectory, "sentinel.txt")); err != nil || string(data) != "keep" {
		t.Fatalf("trashed data mismatch: %q, %v", string(data), err)
	}
	if _, err := harness.profiles.Get(harness.request.ProfileID); err != nil {
		t.Fatalf("Profile metadata was removed by recoverable trash: %v", err)
	}
	if !containsCode(result.Record.LimitationCodes, "profile-trashed") || !containsCode(result.Record.LimitationCodes, "trash-origin-available") {
		t.Fatalf("trash lifecycle markers missing: %#v", result.Record.LimitationCodes)
	}
	if result.Record.RetentionDeadline == nil || result.Record.TrashedAt == nil {
		t.Fatal("trash timestamps were not persisted")
	}

	reused, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Trash.TrashID != result.Trash.TrashID || reused.Operation.ID != result.Operation.ID {
		t.Fatalf("trash retry was not idempotent: %#v", reused)
	}

	restoreRequest := TrashActionRequest{
		OperationID:        "restore-trash-operation-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            result.Trash.TrashID,
		IdempotencyKey:     "restore-trash-request-a",
		ApplicationVersion: "0.15.0-dev",
	}
	restored, err := harness.executor.RestoreTrash(context.Background(), restoreRequest)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Operation.Status != lifecycle.OperationCompleted || restored.Record.State != lifecycle.StateAvailable {
		t.Fatalf("unexpected restore result: %#v", restored)
	}
	if data, err := os.ReadFile(filepath.Join(harness.profileRoot, "sentinel.txt")); err != nil || string(data) != "keep" {
		t.Fatalf("restored data mismatch: %q, %v", string(data), err)
	}
	if _, err := harness.trash.Get(result.Trash.TrashID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trash catalog record remained after restore: %v", err)
	}
	if !containsCode(restored.Record.RecoveryCodes, "existing-recovery") || !containsCode(restored.Record.LimitationCodes, "existing-limit") {
		t.Fatalf("original lifecycle metadata was not restored: %#v", restored.Record)
	}
	if restored.Record.TrashedAt != nil || restored.Record.RetentionDeadline != nil {
		t.Fatalf("trash timestamps remained after restore: %#v", restored.Record)
	}

	reusedRestore, err := harness.executor.RestoreTrash(context.Background(), restoreRequest)
	if err != nil {
		t.Fatal(err)
	}
	if reusedRestore.Operation.ID != restored.Operation.ID || reusedRestore.Record.State != lifecycle.StateAvailable {
		t.Fatalf("restore retry was not idempotent: %#v", reusedRestore)
	}
}

func TestTrashRestoresArchivedStateExactly(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateArchived)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if trashed.Trash.OriginalState != lifecycle.StateArchived || trashed.Trash.OriginalArchivedAt == nil {
		t.Fatalf("archived origin was not captured: %#v", trashed.Trash)
	}
	restored, err := harness.executor.RestoreTrash(context.Background(), TrashActionRequest{
		OperationID:        "restore-trash-archive-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "restore-trash-archive-request-a",
		ApplicationVersion: "0.15.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Record.State != lifecycle.StateArchived || restored.Record.ArchivedAt == nil {
		t.Fatalf("archived lifecycle state was not restored: %#v", restored.Record)
	}
	if !containsCode(restored.Record.LimitationCodes, "profile-archived") || !containsCode(restored.Record.LimitationCodes, "archive-origin-available") {
		t.Fatalf("archived lifecycle limitations were not restored: %#v", restored.Record.LimitationCodes)
	}
}

func TestPermanentDeleteRetainsAuditTombstones(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateDraft)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	request := TrashActionRequest{
		OperationID:        "permanent-delete-operation-a",
		ProfileID:          harness.request.ProfileID,
		TrashID:            trashed.Trash.TrashID,
		IdempotencyKey:     "permanent-delete-request-a",
		ApplicationVersion: "0.15.0-dev",
		Confirmation:       harness.request.ProfileID,
	}
	deleted, err := harness.executor.PermanentDelete(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Operation.Status != lifecycle.OperationCompleted || deleted.Record.State != lifecycle.StateInvalid {
		t.Fatalf("unexpected permanent delete result: %#v", deleted)
	}
	if deleted.Trash.Status != TrashDeleted || deleted.Trash.DataPresent || deleted.Trash.DeletedAt == nil {
		t.Fatalf("trash audit tombstone is incomplete: %#v", deleted.Trash)
	}
	if _, err := harness.profiles.Get(harness.request.ProfileID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("Profile metadata remained after permanent deletion: %v", err)
	}
	if _, err := os.Lstat(trashRootPath(filepath.Join(harness.dataRoot, "local-recovery"), deleted.Trash.TrashID)); !os.IsNotExist(err) {
		t.Fatalf("trash payload remained after permanent deletion: %v", err)
	}
	if !containsCode(deleted.Record.LimitationCodes, "permanent-delete-complete") || !containsCode(deleted.Record.LimitationCodes, "lifecycle-audit-tombstone") {
		t.Fatalf("lifecycle audit tombstone markers missing: %#v", deleted.Record.LimitationCodes)
	}

	reused, err := harness.executor.PermanentDelete(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Trash.Status != TrashDeleted || reused.Operation.ID != deleted.Operation.ID {
		t.Fatalf("permanent-delete retry was not idempotent: %#v", reused)
	}
}
