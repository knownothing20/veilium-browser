package desktop

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestServiceMigratesLegacyLifecycleAndExposesBootstrap(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := store.Create(domain.Profile{
		ID:          "legacy-profile",
		Name:        "Legacy",
		UserDataDir: filepath.Join(root, "profiles", "legacy-profile"),
	})
	if err != nil {
		t.Fatal(err)
	}

	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	records := service.ListLifecycleRecords()
	if len(records) != 1 || records[0].ProfileID != legacy.ID || records[0].State != lifecycle.StateAvailable {
		t.Fatalf("unexpected compatibility records: %#v", records)
	}
	if records[0].ManagedDir != "profiles/legacy-profile" {
		t.Fatalf("unexpected managed directory: %q", records[0].ManagedDir)
	}
	report := service.LifecycleReconciliation()
	if len(report.CompatibilityCreated) != 1 || report.CompatibilityCreated[0] != legacy.ID {
		t.Fatalf("unexpected reconciliation report: %#v", report)
	}
	bootstrap := service.Bootstrap()
	if len(bootstrap.LifecycleRecords) != 1 || len(bootstrap.LifecycleOperations) != 0 {
		t.Fatalf("bootstrap omitted lifecycle state: %#v", bootstrap)
	}
}

func TestServiceMarksUnmanagedLegacyProfileInvalid(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := store.Create(domain.Profile{
		ID:          "unmanaged-profile",
		Name:        "Unmanaged",
		UserDataDir: filepath.Join(root, "outside"),
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.lifecycleRecords.Get(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateInvalid || !containsString(record.LimitationCodes, "legacy-user-data-unmanaged") {
		t.Fatalf("unsafe legacy profile was not limited: %#v", record)
	}
	if _, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: legacy.ID}); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid lifecycle rejection, got %v", err)
	}
}

func TestCreateProfileCreatesLifecycleRecordAndDeleteFailsClosed(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.lifecycleRecords.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != lifecycle.StateAvailable || record.ManagedDir != filepath.ToSlash(filepath.Join("profiles", created.ID)) {
		t.Fatalf("unexpected lifecycle record: %#v", record)
	}
	if err := service.DeleteProfile(created.ID); err == nil || !strings.Contains(err.Error(), "trash transactions") {
		t.Fatalf("expected bounded delete rejection, got %v", err)
	}
	if _, err := store.Get(created.ID); err != nil {
		t.Fatalf("failed delete changed Profile metadata: %v", err)
	}
}

func TestLifecycleStateAndLockBlockProfileActions(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	record, _ := service.lifecycleRecords.Get(created.ID)
	record.State = lifecycle.StateArchived
	if _, err := service.lifecycleRecords.Update(record); err != nil {
		t.Fatal(err)
	}
	created.Notes = "blocked archive edit"
	if _, err := service.UpdateProfile(created); err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("expected archived edit rejection, got %v", err)
	}
	if _, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: created.ID}); err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("expected archived launch rejection, got %v", err)
	}

	record, _ = service.lifecycleRecords.Get(created.ID)
	record.State = lifecycle.StateAvailable
	if _, err := service.lifecycleRecords.Update(record); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.lifecycleRecords.AcquireLocks("operation-lock", []string{created.ID}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateProfile(created); err == nil || !strings.Contains(err.Error(), "operation-lock") {
		t.Fatalf("expected lifecycle lock edit rejection, got %v", err)
	}
	if _, err := service.CloneProfile(created.ID, "Clone"); err == nil || !strings.Contains(err.Error(), "operation-lock") {
		t.Fatalf("expected lifecycle lock clone rejection, got %v", err)
	}
}

func TestServiceStartupReconcilesInterruptedOperation(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	item, err := store.Create(domain.Profile{
		ID:          "recovery-profile",
		Name:        "Recovery",
		UserDataDir: filepath.Join(root, "profiles", "recovery-profile"),
	})
	if err != nil {
		t.Fatal(err)
	}
	records, err := lifecycle.OpenRecordStore(filepath.Join(root, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := records.EnsureCompatibility([]lifecycle.CompatibilityInput{{
		ProfileID:  item.ID,
		ManagedDir: filepath.ToSlash(filepath.Join("profiles", item.ID)),
		State:      lifecycle.StateAvailable,
	}}); err != nil {
		t.Fatal(err)
	}
	journal, err := lifecycle.OpenJournal(filepath.Join(root, "lifecycle-operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := lifecycle.NewCoordinator(records, journal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	operation := lifecycle.NewOperation("interrupted-operation", lifecycle.OperationStorageReconcile, []string{item.ID}, time.Now().UTC())
	operation.ApplicationVersion = AppVersion
	operation.Platform = runtime.GOOS
	if _, _, err := coordinator.Begin(operation); err != nil {
		t.Fatal(err)
	}

	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	reconciled, err := service.lifecycleJournal.Get(operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != lifecycle.OperationRecoveryRequired || reconciled.CompletedAt == nil {
		t.Fatalf("interrupted operation was not reconciled: %#v", reconciled)
	}
	record, _ := service.lifecycleRecords.Get(item.ID)
	if record.Lock != nil || !containsString(record.RecoveryCodes, "stale-lock-terminal-operation") {
		t.Fatalf("stale lifecycle lock was not reconciled: %#v", record)
	}
	if len(service.LifecycleReconciliation().Actions) == 0 {
		t.Fatal("startup reconciliation actions were not exposed")
	}
}

func TestServiceFailsClosedOnMalformedLifecycleState(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	if err := os.WriteFile(filepath.Join(root, "lifecycle.json"), []byte(`{"schemaVersion":1,"records":[],"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newService(store, root, newFakeRuntime()); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("expected lifecycle startup rejection, got %v", err)
	}
}

func TestScanLifecycleStorageIsReadOnlyAndBounded(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(created.UserDataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(created.UserDataDir, "opaque-browser-data")
	if err := os.WriteFile(payload, []byte("opaque"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory, err := service.ScanLifecycleStorage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Profiles) != 1 || inventory.Profiles[0].Summary.Files != 1 {
		t.Fatalf("unexpected inventory: %#v", inventory)
	}
	data, err := os.ReadFile(payload)
	if err != nil || string(data) != "opaque" {
		t.Fatalf("inventory changed browser data: %q, %v", data, err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
