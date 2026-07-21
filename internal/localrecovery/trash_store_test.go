package localrecovery

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func validTrashRecordForTest() TrashRecord {
	now := time.Now().UTC().Truncate(time.Second)
	return TrashRecord{
		TrashID:                 "trash-a",
		ProfileID:               "profile-a",
		OperatingSystem:         runtime.GOOS,
		Architecture:            runtime.GOARCH,
		OriginalState:           lifecycle.StateAvailable,
		OriginalManagedDir:      "profiles/profile-a",
		OriginalRecoveryCodes:   []string{"existing-recovery"},
		OriginalLimitationCodes: []string{"existing-limit"},
		TrashRef:                "local-recovery/trash/trash-a/browser-data",
		DataPresent:             true,
		ProfileDefinitionDigest: strings.Repeat("a", 64),
		TreeDigest:              strings.Repeat("b", 64),
		FileCount:               1,
		TotalBytes:              4,
		TrashedAt:               now,
		RetentionDeadline:       now.Add(30 * 24 * time.Hour),
		Limitations:             []string{"local-only"},
	}
}

func TestTrashStoreRoundTripAndTransitions(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "trash.json")
	store, err := OpenTrashStore(path)
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(validTrashRecordForTest())
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != TrashPending || created.Revision != 1 {
		t.Fatalf("unexpected created record: %#v", created)
	}
	created.Status = TrashStored
	stored, err := store.Update(created)
	if err != nil {
		t.Fatal(err)
	}
	stored.Status = TrashRestoring
	restoring, err := store.Update(stored)
	if err != nil {
		t.Fatal(err)
	}
	restoring.Status = TrashStored
	stored, err = store.Update(restoring)
	if err != nil {
		t.Fatal(err)
	}
	stored.Status = TrashCleanupPending
	cleanup, err := store.Update(stored)
	if err != nil {
		t.Fatal(err)
	}
	deletedAt := time.Now().UTC()
	cleanup.Status = TrashDeleted
	cleanup.DataPresent = false
	cleanup.DeletedAt = &deletedAt
	deleted, err := store.Update(cleanup)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Status != TrashDeleted || deleted.Revision != 6 {
		t.Fatalf("unexpected deleted tombstone: %#v", deleted)
	}
	reopened, err := OpenTrashStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get(deleted.TrashID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != TrashDeleted || loaded.DeletedAt == nil || loaded.DataPresent {
		t.Fatalf("deleted tombstone did not persist: %#v", loaded)
	}
}

func TestTrashStorePersistenceFailureRollsBackMemory(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := OpenTrashStore(filepath.Join(root, "trash.json"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(validTrashRecordForTest())
	if err != nil {
		t.Fatal(err)
	}
	store.write = func(string, []byte) error { return errors.New("simulated persistence failure") }
	created.Status = TrashStored
	if _, err := store.Update(created); err == nil {
		t.Fatal("expected persistence failure")
	}
	current, err := store.Get(created.TrashID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != TrashPending || current.Revision != created.Revision {
		t.Fatalf("failed persistence changed in-memory record: %#v", current)
	}
}

func TestTrashStoreRejectsSecondRecordForProfile(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := OpenTrashStore(filepath.Join(root, "trash.json"))
	if err != nil {
		t.Fatal(err)
	}
	first := validTrashRecordForTest()
	if _, err := store.Create(first); err != nil {
		t.Fatal(err)
	}
	second := validTrashRecordForTest()
	second.TrashID = "trash-b"
	second.TrashRef = "local-recovery/trash/trash-b/browser-data"
	if _, err := store.Create(second); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected duplicate Profile conflict, got %v", err)
	}
}
