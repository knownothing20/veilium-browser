package localrecovery

import (
	"context"
	"testing"
)

func TestRestoreIdempotentRetryReturnsSameIdentity(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "content"})
	first, err := harness.executor.Restore(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	retry := harness.request
	retry.OperationID = "restore-operation-b"
	second, err := harness.executor.Restore(context.Background(), retry)
	if err != nil {
		t.Fatal(err)
	}
	if second.Profile.ID != first.Profile.ID || second.Profile.Fingerprint.Seed != first.Profile.Fingerprint.Seed || second.Operation.ID != first.Operation.ID {
		t.Fatalf("idempotent restore created a different result: first=%#v second=%#v", first, second)
	}
	operations := harness.snapshot.journal.List()
	if len(operations) != 2 {
		t.Fatalf("idempotent restore created another journal operation: %#v", operations)
	}
}
