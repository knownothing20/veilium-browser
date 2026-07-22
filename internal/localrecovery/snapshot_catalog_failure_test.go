package localrecovery

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestSnapshotCreatorRequiresRecoveryAfterPublishedCatalogFailure(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	originalWrite := harness.creator.catalog.write
	writes := 0
	harness.creator.catalog.write = func(filePath string, data []byte) error {
		writes++
		if writes == 2 {
			return errors.New("simulated verified catalog write failure")
		}
		return originalWrite(filePath, data)
	}

	result, err := harness.creator.Create(context.Background(), harness.request)
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("published catalog failure did not require recovery: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationRecoveryRequired || result.Catalog.Status != SnapshotRecoveryRequired {
		t.Fatalf("published recovery state was not recorded: %#v %#v", result.Operation, result.Catalog)
	}
	if _, err := os.Stat(snapshotFinalPath(harness.creator.recoveryRoot, harness.request.SnapshotID)); err != nil {
		t.Fatalf("verified published snapshot was not preserved: %v", err)
	}
	if len(result.Operation.Items) != 1 || result.Operation.Items[0].RecoveryID != snapshotPublishedRef(harness.request.SnapshotID) {
		t.Fatalf("published recovery identity was not recorded: %#v", result.Operation.Items)
	}
	assertProfileUnlocked(t, harness)
}
