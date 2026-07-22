package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/localrecovery"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestDesktopLocalRecoverySnapshotArchiveAndTrash(t *testing.T) {
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
	payload := filepath.Join(created.UserDataDir, "opaque-browser-state.bin")
	if err := os.WriteFile(payload, []byte("opaque-state"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	preflight, err := service.LocalRecoveryPreflight(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !preflight.SnapshotAllowed || !preflight.ArchiveAllowed || !preflight.TrashAllowed {
		t.Fatalf("unexpected available Profile preflight: %#v", preflight)
	}

	if _, err := service.ArchiveProfile(ctx, ArchiveProfileRequest{ProfileID: created.ID, IdempotencyKey: "archive-request"}); err != nil {
		t.Fatal(err)
	}
	record, err := service.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateArchived {
		t.Fatalf("expected archived lifecycle state, got %#v", record)
	}
	if _, err := service.UnarchiveProfile(ctx, ArchiveProfileRequest{ProfileID: created.ID, IdempotencyKey: "unarchive-request"}); err != nil {
		t.Fatal(err)
	}
	record, err = service.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateAvailable {
		t.Fatalf("expected exact available origin state after unarchive, got %#v", record)
	}

	snapshot, err := service.CreateLocalSnapshot(ctx, CreateLocalSnapshotRequest{ProfileID: created.ID, IdempotencyKey: "snapshot-request"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Catalog.Status != localrecovery.SnapshotVerified || snapshot.Catalog.FileCount != 1 {
		t.Fatalf("unexpected snapshot result: %#v", snapshot)
	}
	state, err := service.LocalRecoveryState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Snapshots) != 1 || state.Snapshots[0].SnapshotID != snapshot.Catalog.SnapshotID {
		t.Fatalf("snapshot catalog was not exposed: %#v", state.Snapshots)
	}

	trashed, err := service.TrashProfile(ctx, TrashProfileRequest{ProfileID: created.ID, RetentionDays: 30, IdempotencyKey: "trash-request"})
	if err != nil {
		t.Fatal(err)
	}
	if trashed.Trash.Status != localrecovery.TrashStored || !trashed.Trash.DataPresent {
		t.Fatalf("unexpected trash result: %#v", trashed)
	}
	if _, err := os.Stat(created.UserDataDir); !os.IsNotExist(err) {
		t.Fatalf("managed directory remained after verified trash: %v", err)
	}
	preflight, err = service.LocalRecoveryPreflight(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !preflight.RestoreTrashAllowed || !preflight.PermanentDeleteAllowed {
		t.Fatalf("unexpected trashed Profile preflight: %#v", preflight)
	}

	if _, err := service.RestoreTrashedProfile(ctx, TrashProfileActionRequest{ProfileID: created.ID, TrashID: trashed.Trash.TrashID, IdempotencyKey: "restore-trash-request"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(payload)
	if err != nil || string(data) != "opaque-state" {
		t.Fatalf("restored browser data changed: %q, %v", data, err)
	}
	record, err = service.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateAvailable {
		t.Fatalf("restore-trash did not recover exact origin lifecycle state: %#v", record)
	}
}

func TestDesktopLocalRecoveryRejectsUnconfirmedPermanentDelete(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	trashed, err := service.TrashProfile(ctx, TrashProfileRequest{ProfileID: created.ID, RetentionDays: 30, IdempotencyKey: "empty-trash-request"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PermanentlyDeleteTrashedProfile(ctx, TrashProfileActionRequest{ProfileID: created.ID, TrashID: trashed.Trash.TrashID, Confirmation: "wrong", IdempotencyKey: "delete-request"}); err == nil {
		t.Fatal("expected exact Profile confirmation requirement")
	}
	state, err := service.LocalRecoveryState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Trash) != 1 || state.Trash[0].Status != localrecovery.TrashStored || !state.Trash[0].DataPresent {
		t.Fatalf("failed confirmation changed recoverable trash: %#v", state.Trash)
	}
}
