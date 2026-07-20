package lifecycle

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newCoordinatorFixture(t *testing.T, profileIDs ...string) (*RecordStore, *Journal, *Coordinator, *time.Time) {
	t.Helper()
	root := t.TempDir()
	records, err := OpenRecordStore(filepath.Join(root, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenJournal(filepath.Join(root, "operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	records.now = func() time.Time { return now }
	journal.now = func() time.Time { return now }
	for _, id := range profileIDs {
		if _, err := records.Create(NewCompatibilityRecord(id, "profiles/"+id, now)); err != nil {
			t.Fatal(err)
		}
	}
	coordinator, err := NewCoordinator(records, journal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.now = func() time.Time { return now }
	return records, journal, coordinator, &now
}

func operationForCoordinator(id string, profiles ...string) Operation {
	op := NewOperation(id, OperationStorageReconcile, profiles, time.Time{})
	op.ApplicationVersion = "0.15.0-dev"
	op.Platform = "linux/amd64"
	return op
}

func TestCoordinatorBlocksActiveAndDependentProfilesBeforeJournaling(t *testing.T) {
	records, journal, _, now := newCoordinatorFixture(t, "profile-a")
	coordinator, err := NewCoordinator(records, journal,
		func(profileID string) (string, bool) { return "session-1", profileID == "profile-a" },
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.now = func() time.Time { return *now }
	if _, _, err := coordinator.Begin(operationForCoordinator("op-a", "profile-a")); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "active browser") {
		t.Fatalf("expected active-session conflict, got %v", err)
	}
	if len(journal.List()) != 0 {
		t.Fatalf("blocked request should not be journaled: %+v", journal.List())
	}
	record, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil {
		t.Fatalf("blocked request acquired a lock: %+v", record.Lock)
	}

	coordinator, err = NewCoordinator(records, journal, nil,
		func(profileID string) (string, bool) { return "evidence-collection", profileID == "profile-a" },
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.now = func() time.Time { return *now }
	if _, _, err := coordinator.Begin(operationForCoordinator("op-b", "profile-a")); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "dependent") {
		t.Fatalf("expected dependent-operation conflict, got %v", err)
	}
	if len(journal.List()) != 0 {
		t.Fatalf("blocked dependent request should not be journaled")
	}
}

func TestCoordinatorBeginFinishAndIdempotentRetry(t *testing.T) {
	records, journal, coordinator, now := newCoordinatorFixture(t, "profile-a", "profile-b")
	input := operationForCoordinator("op-a", "profile-b", "profile-a")
	input.IdempotencyKey = "request-a"
	running, reused, err := coordinator.Begin(input)
	if err != nil {
		t.Fatal(err)
	}
	if reused || running.Status != OperationRunning || running.Stage != "locked" || running.Revision != 2 {
		t.Fatalf("unexpected running operation: reused=%v op=%+v", reused, running)
	}
	for _, id := range []string{"profile-a", "profile-b"} {
		record, err := records.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if record.Lock == nil || record.Lock.OperationID != "op-a" {
			t.Fatalf("profile %s is not locked by operation: %+v", id, record.Lock)
		}
	}

	retry := input
	retry.ID = "op-retry"
	existing, reused, err := coordinator.Begin(retry)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || existing.ID != "op-a" || existing.Status != OperationRunning {
		t.Fatalf("unexpected idempotent retry: reused=%v op=%+v", reused, existing)
	}

	*now = now.Add(time.Minute)
	completedAt := *now
	items := []OperationItemResult{
		{ItemID: "profile-a", Status: ItemSucceeded, CompletedAt: &completedAt},
		{ItemID: "profile-b", Status: ItemSucceeded, CompletedAt: &completedAt},
	}
	finished, err := coordinator.Finish("op-a", OperationCompleted, items, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != OperationCompleted || finished.CompletedAt == nil {
		t.Fatalf("unexpected finished operation: %+v", finished)
	}
	for _, id := range []string{"profile-a", "profile-b"} {
		record, err := records.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if record.Lock != nil {
			t.Fatalf("profile %s lock was not released: %+v", id, record.Lock)
		}
	}
	loaded, err := journal.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != OperationCompleted {
		t.Fatalf("journal did not persist completion: %+v", loaded)
	}
}

func TestRecordStoreLockConflictIsAtomic(t *testing.T) {
	records, _, _, now := newCoordinatorFixture(t, "profile-a", "profile-b")
	if _, _, err := records.AcquireLocks("op-owner", []string{"profile-b"}, *now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := records.AcquireLocks("op-other", []string{"profile-a", "profile-b"}, *now); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected lock conflict, got %v", err)
	}
	a, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := records.Get("profile-b")
	if err != nil {
		t.Fatal(err)
	}
	if a.Lock != nil {
		t.Fatalf("atomic conflict partially locked profile-a: %+v", a.Lock)
	}
	if b.Lock == nil || b.Lock.OperationID != "op-owner" {
		t.Fatalf("existing lock changed: %+v", b.Lock)
	}
}

func TestCoordinatorJournalFailureReleasesLocks(t *testing.T) {
	records, journal, coordinator, _ := newCoordinatorFixture(t, "profile-a")
	originalWrite := journal.write
	writes := 0
	journal.write = func(path string, data []byte) error {
		writes++
		if writes == 2 {
			return errors.New("journal unavailable")
		}
		return originalWrite(path, data)
	}
	if _, _, err := coordinator.Begin(operationForCoordinator("op-a", "profile-a")); err == nil || !strings.Contains(err.Error(), "journal unavailable") {
		t.Fatalf("expected journal transition failure, got %v", err)
	}
	record, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil {
		t.Fatalf("lock remained after journal transition failure: %+v", record.Lock)
	}
	operation, err := journal.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != OperationPending {
		t.Fatalf("failed transition must remain pending for reconciliation: %+v", operation)
	}
}

func TestRequestCancellationDoesNotImplyCompletion(t *testing.T) {
	_, journal, coordinator, _ := newCoordinatorFixture(t, "profile-a")
	running, _, err := coordinator.Begin(operationForCoordinator("op-a", "profile-a"))
	if err != nil {
		t.Fatal(err)
	}
	cancelled, changed, err := coordinator.RequestCancellation(running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !cancelled.CancellationRequested || cancelled.Status != OperationRunning || cancelled.CompletedAt != nil {
		t.Fatalf("cancellation request implied terminal success: changed=%v op=%+v", changed, cancelled)
	}
	persisted, err := journal.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != OperationRunning || persisted.CompletedAt != nil {
		t.Fatalf("unexpected persisted cancellation: %+v", persisted)
	}
}
