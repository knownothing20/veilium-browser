package desktop

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestShutdownPreservesInterruptedLifecycleStateForStartupRecovery(t *testing.T) {
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
	operation := lifecycle.NewOperation("shutdown-interrupted-operation", lifecycle.OperationStorageReconcile, []string{created.ID}, time.Now().UTC())
	operation.ApplicationVersion = AppVersion
	operation.Platform = runtime.GOOS
	operation.SafeCancellationStage = "between-items"
	running, reused, err := service.lifecycleCoordinator.Begin(operation)
	if err != nil {
		t.Fatal(err)
	}
	if reused || running.Status != lifecycle.OperationRunning {
		t.Fatalf("unexpected operation start: %#v, reused=%v", running, reused)
	}

	if err := service.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	persisted, err := service.lifecycleJournal.Get(operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != lifecycle.OperationRunning || persisted.CompletedAt != nil {
		t.Fatalf("shutdown silently changed interrupted operation: %#v", persisted)
	}

	reopenedStore, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := newService(reopenedStore, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	reconciled, err := restarted.lifecycleJournal.Get(operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != lifecycle.OperationRecoveryRequired || reconciled.CompletedAt == nil {
		t.Fatalf("startup did not preserve truthful recovery state: %#v", reconciled)
	}
	record, err := restarted.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil {
		t.Fatalf("startup left a stale lock after journal reconciliation: %#v", record)
	}
}
