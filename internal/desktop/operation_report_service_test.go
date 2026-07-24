package desktop

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestLifecycleOperationReportRedactsLocalExecutionReferencesAndDetectsTamper(t *testing.T) {
	now := time.Date(2026, 7, 23, 1, 2, 3, 0, time.UTC)
	completed := now.Add(time.Second)
	operation := lifecycle.NewOperation("operation-report-test", lifecycle.OperationBulkHealthRefresh, []string{"profile-b", "profile-a"}, now)
	operation.IdempotencyKey = "private-idempotency-key"
	operation.Status = lifecycle.OperationCompleted
	operation.Stage = "health-refresh-completed"
	operation.CompletedAt = &completed
	operation.UpdatedAt = completed
	operation.StagingRef = "staging/private-operation"
	operation.QuarantineRef = "quarantine/private-operation"
	operation.ApplicationVersion = "0.15.0-dev"
	operation.Platform = "windows/amd64"
	operation.Items = []lifecycle.OperationItemResult{
		{
			ItemID:         "profile-a",
			Status:         lifecycle.ItemSucceeded,
			StartedAt:      &now,
			CompletedAt:    &completed,
			CompletedStage: "health-refreshed",
			FilesProcessed: 4,
			BytesProcessed: 128,
			OutputID:       `C:\Users\tester\AppData\Local\Veilium\report.json`,
		},
		{
			ItemID:         "profile-b",
			Status:         lifecycle.ItemSucceeded,
			StartedAt:      &now,
			CompletedAt:    &completed,
			CompletedStage: "health-refreshed",
			FilesProcessed: 2,
			BytesProcessed: 64,
			RecoveryID:     "recovery-record-1",
		},
	}

	artifact, err := buildLifecycleOperationReport(operation, completed)
	if err != nil {
		t.Fatal(err)
	}
	data, err := encodeLifecycleOperationReport(artifact)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"private-idempotency-key",
		"staging/private-operation",
		"quarantine/private-operation",
		`C:\Users\tester`,
	} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("operation report leaked %q", forbidden)
		}
	}
	if !bytes.Contains(data, []byte("redacted-local-reference")) {
		t.Fatal("expected absolute local output path to be redacted")
	}
	decoded, err := decodeLifecycleOperationReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Payload.Summary.ItemCount != 2 || decoded.Payload.Summary.Succeeded != 2 {
		t.Fatalf("unexpected report summary: %+v", decoded.Payload.Summary)
	}
	if decoded.Payload.Summary.FilesProcessed != 6 || decoded.Payload.Summary.BytesProcessed != 192 {
		t.Fatalf("unexpected report progress totals: %+v", decoded.Payload.Summary)
	}

	var tampered lifecycleOperationReportArtifact
	if err := json.Unmarshal(data, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.Payload.Operation.Stage = "tampered-stage"
	tamperedData, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeLifecycleOperationReport(tamperedData); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected payload digest mismatch, got %v", err)
	}
}

func TestExportLifecycleOperationReportWritesOnceFromJournal(t *testing.T) {
	root := t.TempDir()
	journal, err := lifecycle.OpenJournal(filepath.Join(root, "lifecycle-operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 23, 2, 3, 4, 0, time.UTC)
	operation := lifecycle.NewOperation("operation-report-write", lifecycle.OperationExportDefinition, []string{"profile-a"}, now)
	operation.IdempotencyKey = "journal-private-key"
	operation.ApplicationVersion = "0.15.0-dev"
	operation.Platform = "windows/amd64"
	created, reused, err := journal.Create(operation)
	if err != nil {
		t.Fatal(err)
	}
	if reused {
		t.Fatal("new journal operation was unexpectedly reused")
	}

	service := &Service{lifecycleJournal: journal}
	destination := filepath.Join(root, "operation-report.veilium-operation-report.json")
	result, err := service.ExportLifecycleOperationReport(OperationReportExportRequest{
		OperationID:     created.ID,
		DestinationPath: destination,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OperationID != created.ID || result.Path != destination || result.PayloadSHA256 == "" {
		t.Fatalf("unexpected export result: %+v", result)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := decodeLifecycleOperationReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Payload.Operation.ID != created.ID || artifact.Payload.Operation.IdempotencyKey != "" {
		t.Fatalf("unexpected exported operation: %+v", artifact.Payload.Operation)
	}
	if _, err := service.ExportLifecycleOperationReport(OperationReportExportRequest{
		OperationID:     created.ID,
		DestinationPath: destination,
	}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected no-overwrite failure, got %v", err)
	}
}
