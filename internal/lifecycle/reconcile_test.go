package lifecycle

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestReconcilerCreatesCompatibilityAndRecoversInterruptedOperation(t *testing.T) {
	root := t.TempDir()
	records, err := OpenRecordStore(filepath.Join(root, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenJournal(filepath.Join(root, "operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	scanner, err := NewInventoryScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	records.now = func() time.Time { return now }
	journal.now = func() time.Time { return now }
	scanner.Now = func() time.Time { return now }
	if _, _, err := records.EnsureCompatibility([]CompatibilityInput{{ProfileID: "profile-a", ManagedDir: "profiles/profile-a", State: StateAvailable}}); err != nil {
		t.Fatal(err)
	}
	op := operationForCoordinator("op-a", "profile-a")
	created, _, err := journal.Create(op)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := records.AcquireLocks(created.ID, created.ProfileIDs, now); err != nil {
		t.Fatal(err)
	}
	created.Status = OperationRunning
	created.Stage = "inventory"
	created, err = journal.Update(created)
	if err != nil {
		t.Fatal(err)
	}

	reconciler, err := NewReconciler(records, journal, scanner)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	reconciler.now = func() time.Time { return now }
	report, err := reconciler.Reconcile(context.Background(), []CompatibilityInput{
		{ProfileID: "profile-a", ManagedDir: "profiles/profile-a", State: StateAvailable},
		{ProfileID: "profile-b", ManagedDir: "profiles/profile-b", State: StateAvailable},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.CompatibilityCreated) != 1 || report.CompatibilityCreated[0] != "profile-b" {
		t.Fatalf("unexpected compatibility report: %+v", report.CompatibilityCreated)
	}
	operation, err := journal.Get("op-a")
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != OperationRecoveryRequired || operation.CompletedAt == nil || len(operation.Items) != 1 || operation.Items[0].Status != ItemRecoveryRequired {
		t.Fatalf("interrupted operation was not made recovery-required: %+v", operation)
	}
	record, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil || len(record.RecoveryCodes) == 0 {
		t.Fatalf("stale lock was not reconciled truthfully: %+v", record)
	}
}

func TestReconcilerClearsLockForMissingOperation(t *testing.T) {
	root := t.TempDir()
	records, _ := OpenRecordStore(filepath.Join(root, "lifecycle.json"))
	journal, _ := OpenJournal(filepath.Join(root, "operations.json"))
	scanner, _ := NewInventoryScanner(root)
	now := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	records.now = func() time.Time { return now }
	if _, _, err := records.EnsureCompatibility([]CompatibilityInput{{ProfileID: "profile-a", ManagedDir: "profiles/profile-a", State: StateAvailable}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := records.AcquireLocks("missing-op", []string{"profile-a"}, now); err != nil {
		t.Fatal(err)
	}
	reconciler, _ := NewReconciler(records, journal, scanner)
	reconciler.now = func() time.Time { return now.Add(time.Minute) }
	if _, err := reconciler.Reconcile(context.Background(), []CompatibilityInput{{ProfileID: "profile-a", ManagedDir: "profiles/profile-a", State: StateAvailable}}); err != nil {
		t.Fatal(err)
	}
	record, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil || len(record.RecoveryCodes) != 1 || record.RecoveryCodes[0] != "stale-lock-operation-missing" {
		t.Fatalf("missing-operation lock not reconciled: %+v", record)
	}
}

func TestEnsureCompatibilityIsAtomicAndRejectsPathChange(t *testing.T) {
	root := t.TempDir()
	records, _ := OpenRecordStore(filepath.Join(root, "lifecycle.json"))
	now := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	records.now = func() time.Time { return now }
	_, created, err := records.EnsureCompatibility([]CompatibilityInput{
		{ProfileID: "profile-a", ManagedDir: "profiles/profile-a", State: StateAvailable},
		{ProfileID: "profile-b", ManagedDir: "profiles/profile-b", State: StateInvalid, RecoveryCodes: []string{"unmanaged-profile-path"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("unexpected created records: %+v", created)
	}
	if _, _, err := records.EnsureCompatibility([]CompatibilityInput{{ProfileID: "profile-a", ManagedDir: "profiles/other", State: StateAvailable}}); err == nil {
		t.Fatal("expected managed path conflict")
	}
	loaded, err := records.Get("profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ManagedDir != "profiles/profile-a" {
		t.Fatalf("conflicting migration changed record: %+v", loaded)
	}
}
