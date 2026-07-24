package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestBulkApplyProfileLifecycleArchivesUnarchivesAndReuses(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	secondInput := validProfile()
	secondInput.Name = "Second Profile"
	second, err := service.CreateProfile(secondInput)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{first.UserDataDir, second.UserDataDir} {
		if err := os.MkdirAll(item, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	request := BulkLifecycleRequest{
		ProfileIDs: []string{second.ID, first.ID}, Action: BulkLifecycleArchive, IdempotencyKey: "bulk-archive-1",
	}
	archived, err := service.BulkApplyProfileLifecycle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != lifecycle.OperationCompleted || len(archived.Items) != 2 {
		t.Fatalf("unexpected archive result: %#v", archived)
	}
	for _, profileID := range []string{first.ID, second.ID} {
		record, getErr := service.lifecycleRecords.Get(profileID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if record.State != lifecycle.StateArchived {
			t.Fatalf("profile %q state = %q, want archived", profileID, record.State)
		}
	}
	if operations := service.ListLifecycleOperations(); len(operations) != 2 {
		t.Fatalf("archive created %d operations, want 2", len(operations))
	}

	reused, err := service.BulkApplyProfileLifecycle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if reused.RequestID != archived.RequestID || reused.Status != lifecycle.OperationCompleted {
		t.Fatalf("bulk archive retry did not reuse deterministic children: %#v", reused)
	}
	if operations := service.ListLifecycleOperations(); len(operations) != 2 {
		t.Fatalf("idempotent archive retry created %d operations, want 2", len(operations))
	}

	unarchived, err := service.BulkApplyProfileLifecycle(context.Background(), BulkLifecycleRequest{
		ProfileIDs: []string{first.ID, second.ID}, Action: BulkLifecycleUnarchive, IdempotencyKey: "bulk-unarchive-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if unarchived.Status != lifecycle.OperationCompleted {
		t.Fatalf("unexpected unarchive result: %#v", unarchived)
	}
	for _, profileID := range []string{first.ID, second.ID} {
		record, getErr := service.lifecycleRecords.Get(profileID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if record.State != lifecycle.StateAvailable {
			t.Fatalf("profile %q state = %q, want available", profileID, record.State)
		}
	}
}

func TestBulkApplyProfileLifecycleRequiresConfirmationAndUsesRecoverableTrash(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(created.UserDataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(created.UserDataDir, "marker.txt"), []byte("recoverable"), 0o600); err != nil {
		t.Fatal(err)
	}

	request := BulkLifecycleRequest{
		ProfileIDs: []string{created.ID}, Action: BulkLifecycleTrash, RetentionDays: 30,
		IdempotencyKey: "bulk-trash-1",
	}
	if _, err := service.BulkApplyProfileLifecycle(context.Background(), request); err == nil {
		t.Fatal("bulk trash accepted a request without exact confirmation")
	}
	request.Confirmation = "TRASH 1 PROFILES"
	result, err := service.BulkApplyProfileLifecycle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != lifecycle.OperationCompleted || len(result.Items) != 1 || result.Items[0].Status != lifecycle.ItemSucceeded {
		t.Fatalf("unexpected bulk trash result: %#v", result)
	}
	record, err := service.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateTrashed {
		t.Fatalf("lifecycle state = %q, want trashed", record.State)
	}
	recovery, err := service.LocalRecoveryState()
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.Trash) != 1 || recovery.Trash[0].ProfileID != created.ID || !recovery.Trash[0].DataPresent {
		t.Fatalf("bulk trash did not retain recoverable data: %#v", recovery.Trash)
	}
}
