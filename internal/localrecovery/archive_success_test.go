package localrecovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestArchiveAndUnarchiveAvailableProfile(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)

	archived, err := harness.executor.Archive(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Operation.Status != lifecycle.OperationCompleted {
		t.Fatalf("archive did not complete: %#v", archived.Operation)
	}
	if archived.Record.State != lifecycle.StateArchived || archived.Record.ArchivedAt == nil || archived.Record.Lock != nil {
		t.Fatalf("archive produced the wrong lifecycle state: %#v", archived.Record)
	}
	if !hasLifecycleCode(archived.Record.LimitationCodes, "profile-archived") || !hasLifecycleCode(archived.Record.LimitationCodes, "archive-origin-available") {
		t.Fatalf("archive origin was not recorded: %#v", archived.Record.LimitationCodes)
	}
	data, err := os.ReadFile(filepath.Join(harness.profileRoot, "sentinel.txt"))
	if err != nil || string(data) != "keep" {
		t.Fatalf("archive changed managed browser data: %q %v", data, err)
	}

	unarchiveRequest := harness.request
	unarchiveRequest.OperationID = "unarchive-operation-a"
	unarchiveRequest.IdempotencyKey = "unarchive-request-a"
	unarchived, err := harness.executor.Unarchive(context.Background(), unarchiveRequest)
	if err != nil {
		t.Fatal(err)
	}
	if unarchived.Operation.Status != lifecycle.OperationCompleted || unarchived.Record.State != lifecycle.StateAvailable || unarchived.Record.ArchivedAt != nil || unarchived.Record.Lock != nil {
		t.Fatalf("unarchive did not restore the available state: %#v %#v", unarchived.Operation, unarchived.Record)
	}
	for _, code := range []string{"profile-archived", "archive-origin-available", "archive-origin-draft"} {
		if hasLifecycleCode(unarchived.Record.LimitationCodes, code) {
			t.Fatalf("unarchive retained archive code %q: %#v", code, unarchived.Record.LimitationCodes)
		}
	}
}

func TestArchiveAndUnarchiveDraftPreservesLimitations(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateDraft, []string{"restore-current-validation-required", "restore-lifecycle-draft"})

	archived, err := harness.executor.Archive(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Record.State != lifecycle.StateArchived || !hasLifecycleCode(archived.Record.LimitationCodes, "archive-origin-draft") {
		t.Fatalf("draft archive did not retain its origin: %#v", archived.Record)
	}
	for _, code := range []string{"restore-current-validation-required", "restore-lifecycle-draft"} {
		if !hasLifecycleCode(archived.Record.LimitationCodes, code) {
			t.Fatalf("archive dropped existing draft limitation %q", code)
		}
	}

	request := harness.request
	request.OperationID = "unarchive-operation-draft"
	request.IdempotencyKey = "unarchive-request-draft"
	unarchived, err := harness.executor.Unarchive(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if unarchived.Record.State != lifecycle.StateDraft {
		t.Fatalf("draft unarchive changed the origin state: %#v", unarchived.Record)
	}
	for _, code := range []string{"restore-current-validation-required", "restore-lifecycle-draft"} {
		if !hasLifecycleCode(unarchived.Record.LimitationCodes, code) {
			t.Fatalf("unarchive dropped existing draft limitation %q", code)
		}
	}
}

func TestArchiveIdempotentRetryReturnsCommittedResult(t *testing.T) {
	harness := newArchiveHarness(t, lifecycle.StateAvailable, nil)
	first, err := harness.executor.Archive(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	retry := harness.request
	retry.OperationID = "archive-operation-b"
	second, err := harness.executor.Archive(context.Background(), retry)
	if err != nil {
		t.Fatal(err)
	}
	if second.Operation.ID != first.Operation.ID || second.Record.Revision != first.Record.Revision || second.Record.State != lifecycle.StateArchived {
		t.Fatalf("idempotent archive returned a different result: first=%#v second=%#v", first, second)
	}
	if operations := harness.journal.List(); len(operations) != 1 {
		t.Fatalf("idempotent archive created another operation: %#v", operations)
	}
}
