package localrecovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestTrashReconcilerMarksContradictoryStoredState(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(harness.profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harness.profileRoot, "unexpected.txt"), []byte("duplicate"), 0o600); err != nil {
		t.Fatal(err)
	}
	reconciler, err := OpenTrashReconciler(harness.dataRoot, harness.records, harness.journal, harness.profiles, harness.trash)
	if err != nil {
		t.Fatal(err)
	}
	report, err := reconciler.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].ReasonCode != "trash-stored-state-contradictory" {
		t.Fatalf("unexpected reconciliation report: %#v", report)
	}
	current, err := harness.trash.Get(trashed.Trash.TrashID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != TrashRecoveryRequired {
		t.Fatalf("contradictory trash was not marked recovery-required: %#v", current)
	}
	record, err := harness.records.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(record.RecoveryCodes, "trash-stored-state-contradictory") {
		t.Fatalf("lifecycle recovery code missing: %#v", record.RecoveryCodes)
	}
}

func TestTrashReconcilerLeavesHealthyStoredStateUntouched(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	reconciler, err := OpenTrashReconciler(harness.dataRoot, harness.records, harness.journal, harness.profiles, harness.trash)
	if err != nil {
		t.Fatal(err)
	}
	report, err := reconciler.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("healthy stored trash produced findings: %#v", report)
	}
	current, err := harness.trash.Get(trashed.Trash.TrashID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != TrashStored {
		t.Fatalf("healthy stored trash state changed: %#v", current)
	}
}

func TestTrashReconcilerMarksMissingProfileMetadata(t *testing.T) {
	harness := newTrashHarness(t, lifecycle.StateAvailable)
	trashed, err := harness.executor.Trash(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if err := harness.profiles.Delete(harness.request.ProfileID); err != nil {
		t.Fatal(err)
	}
	reconciler, err := OpenTrashReconciler(harness.dataRoot, harness.records, harness.journal, harness.profiles, harness.trash)
	if err != nil {
		t.Fatal(err)
	}
	report, err := reconciler.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].ProfileState != "absent" || report.Findings[0].ReasonCode != "trash-stored-state-contradictory" {
		t.Fatalf("missing Profile metadata was not reported: %#v", report)
	}
	current, err := harness.trash.Get(trashed.Trash.TrashID)
	if err != nil || current.Status != TrashRecoveryRequired {
		t.Fatalf("missing Profile metadata was not preserved for recovery: %#v, %v", current, err)
	}
}
