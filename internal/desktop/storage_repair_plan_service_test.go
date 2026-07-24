package desktop

import (
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestBuildStorageRepairPlansAreManualAndSnapshotAware(t *testing.T) {
	inventory := lifecycle.StorageInventory{
		Incomplete: true,
		Profiles: []lifecycle.ProfileStorage{
			{ProfileID: "with-snapshot", ManagedDir: "profiles/with-snapshot", Status: lifecycle.InventoryMissing},
			{ProfileID: "without-snapshot", ManagedDir: "profiles/without-snapshot", Status: lifecycle.InventoryMissing},
			{ProfileID: "unsafe", ManagedDir: "profiles/unsafe", Status: lifecycle.InventoryUnsafe, ReasonCode: "unsafe-link-or-reparse"},
		},
		Orphans: []lifecycle.InventoryFinding{
			{RelativePath: "profiles/orphan", ReasonCode: "unregistered-profile-directory"},
		},
	}
	plans := buildStorageRepairPlans(inventory, map[string]bool{"with-snapshot": true})
	if len(plans) < 5 {
		t.Fatalf("repair plans = %d, want at least 5", len(plans))
	}
	foundRestore := false
	foundMissing := false
	for _, plan := range plans {
		if plan.Automatic {
			t.Fatalf("repair plan %q was marked automatic", plan.ID)
		}
		if plan.ProfileID == "with-snapshot" && plan.Kind == "review-snapshot-restore" {
			foundRestore = true
		}
		if plan.ProfileID == "without-snapshot" && plan.Kind == "inspect-missing-profile-data" {
			foundMissing = true
		}
	}
	if !foundRestore || !foundMissing {
		t.Fatalf("snapshot-aware plans missing: restore=%t missing=%t", foundRestore, foundMissing)
	}
}

func TestBuildStorageRepairPlansReturnsNoneForHealthyInventory(t *testing.T) {
	inventory := lifecycle.StorageInventory{
		Profiles: []lifecycle.ProfileStorage{
			{ProfileID: "healthy", ManagedDir: "profiles/healthy", Status: lifecycle.InventoryPresent},
		},
	}
	if plans := buildStorageRepairPlans(inventory, nil); len(plans) != 0 {
		t.Fatalf("healthy inventory produced %#v", plans)
	}
}
