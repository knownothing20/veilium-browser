package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRecordStorePersistsAndRejectsStaleRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.json")
	store, err := OpenRecordStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	created, err := store.Create(NewCompatibilityRecord("profile-a", "profiles/profile-a", now))
	if err != nil {
		t.Fatal(err)
	}
	if created.State != StateAvailable || created.Revision != 1 {
		t.Fatalf("unexpected created record: %+v", created)
	}

	stale := created
	now = now.Add(time.Minute)
	created.State = StateDraft
	updated, err := store.Update(created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || !updated.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected updated record: %+v", updated)
	}
	if _, err := store.Update(stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}

	reopened, err := OpenRecordStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateDraft || loaded.Revision != 2 {
		t.Fatalf("unexpected reopened record: %+v", loaded)
	}
}

func TestRecordStoreRollsBackMemoryWhenPersistenceFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.json")
	store, err := OpenRecordStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	created, err := store.Create(NewCompatibilityRecord("profile-a", "profiles/profile-a", now))
	if err != nil {
		t.Fatal(err)
	}

	store.write = func(string, []byte) error { return errors.New("disk failure") }
	created.State = StateDraft
	if _, err := store.Update(created); err == nil || !strings.Contains(err.Error(), "disk failure") {
		t.Fatalf("expected persistence failure, got %v", err)
	}
	loaded, err := store.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateAvailable || loaded.Revision != 1 {
		t.Fatalf("memory changed after failed persistence: %+v", loaded)
	}

	reopened, err := OpenRecordStore(path)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := reopened.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if disk.State != StateAvailable || disk.Revision != 1 {
		t.Fatalf("disk changed after failed persistence: %+v", disk)
	}
}

func TestRecordStoreRejectsDuplicateAndFutureRecords(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	record := NewCompatibilityRecord("profile-a", "profiles/profile-a", now)

	t.Run("duplicate profile id", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lifecycle.json")
		writePrivateJSON(t, path, recordEnvelope{
			SchemaVersion: LifecycleSchemaVersion,
			Records:       []Record{record, record},
		})
		if _, err := OpenRecordStore(path); err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate rejection, got %v", err)
		}
	})

	t.Run("future record version", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lifecycle.json")
		future := record
		future.SchemaVersion = LifecycleSchemaVersion + 1
		writePrivateJSON(t, path, recordEnvelope{
			SchemaVersion: LifecycleSchemaVersion,
			Records:       []Record{future},
		})
		if _, err := OpenRecordStore(path); !errors.Is(err, ErrUnsupportedVersion) {
			t.Fatalf("expected unsupported version, got %v", err)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lifecycle.json")
		data := `{"schemaVersion":1,"records":[],"unknown":true}`
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenRecordStore(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("expected strict decoding failure, got %v", err)
		}
	})
}

func TestRecordStoreRejectsSymlinkTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated Windows permissions")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte(`{"schemaVersion":1,"records":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "lifecycle.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRecordStore(link); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestJournalIdempotencyCancellationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operations.json")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	journal.now = func() time.Time { return now }

	input := NewOperation("op-a", OperationStorageReconcile, []string{"profile-b", "profile-a"}, now)
	input.IdempotencyKey = "request-a"
	input.ApplicationVersion = "0.15.0-dev"
	input.Platform = "linux/amd64"
	created, reused, err := journal.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	if reused || created.Revision != 1 || created.ProfileIDs[0] != "profile-a" {
		t.Fatalf("unexpected created operation: reused=%v operation=%+v", reused, created)
	}

	retry := input
	retry.ID = "op-retry"
	existing, reused, err := journal.Create(retry)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || existing.ID != "op-a" {
		t.Fatalf("idempotent retry did not return existing operation: %+v", existing)
	}

	conflicting := retry
	conflicting.ProfileIDs = []string{"profile-c"}
	if _, _, err := journal.Create(conflicting); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}

	now = now.Add(time.Minute)
	cancelled, changed, err := journal.RequestCancellation("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !cancelled.CancellationRequested || cancelled.Revision != 2 {
		t.Fatalf("unexpected cancellation result: changed=%v operation=%+v", changed, cancelled)
	}
	second, changed, err := journal.RequestCancellation("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if changed || second.Revision != 2 {
		t.Fatalf("duplicate cancellation should be stable: changed=%v operation=%+v", changed, second)
	}

	reopened, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.CancellationRequested || loaded.Revision != 2 {
		t.Fatalf("unexpected reopened operation: %+v", loaded)
	}
}

func TestOperationTerminalStatusRequiresTruthfulResults(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	operation := NewOperation("op-a", OperationStorageReconcile, []string{"profile-a"}, now)
	operation.ApplicationVersion = "0.15.0-dev"
	operation.Platform = "windows/amd64"
	operation.Status = OperationCompleted
	operation.CompletedAt = pointerTime(now.Add(time.Minute))
	if err := operation.Validate(); err == nil || !strings.Contains(err.Error(), "item results") {
		t.Fatalf("expected terminal item requirement, got %v", err)
	}

	operation.Items = []OperationItemResult{{
		ItemID:      "profile-a",
		Status:      ItemFailed,
		CompletedAt: pointerTime(now.Add(time.Minute)),
	}}
	if err := operation.Validate(); err == nil || !strings.Contains(err.Error(), "non-success") {
		t.Fatalf("expected completed aggregate rejection, got %v", err)
	}

	operation.Status = OperationPartial
	if err := operation.Validate(); err != nil {
		t.Fatalf("partial operation with failed item should be valid: %v", err)
	}
}

func TestJournalRollsBackMemoryWhenPersistenceFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operations.json")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	journal.now = func() time.Time { return now }
	operation := NewOperation("op-a", OperationStorageReconcile, []string{"profile-a"}, now)
	operation.ApplicationVersion = "0.15.0-dev"
	operation.Platform = "linux/amd64"
	created, _, err := journal.Create(operation)
	if err != nil {
		t.Fatal(err)
	}

	journal.write = func(string, []byte) error { return errors.New("disk failure") }
	created.Status = OperationRunning
	created.Stage = "inventory"
	if _, err := journal.Update(created); err == nil || !strings.Contains(err.Error(), "disk failure") {
		t.Fatalf("expected update failure, got %v", err)
	}
	loaded, err := journal.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != OperationPending || loaded.Revision != 1 {
		t.Fatalf("memory changed after failed journal write: %+v", loaded)
	}
}

func writePrivateJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

func pointerTime(value time.Time) *time.Time { return &value }
