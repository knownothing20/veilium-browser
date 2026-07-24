package desktop

import (
	"context"
	"sort"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/localrecovery"
)

type StorageRepairPlan struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	ProfileID    string `json:"profileId,omitempty"`
	RelativePath string `json:"relativePath,omitempty"`
	ReasonCode   string `json:"reasonCode"`
	Risk         string `json:"risk"`
	Description  string `json:"description"`
	Automatic    bool   `json:"automatic"`
}

type StorageManagementReview struct {
	State       StorageManagementState `json:"state"`
	RepairPlans []StorageRepairPlan    `json:"repairPlans"`
}

func (s *Service) ReviewStorageManagement(ctx context.Context) (StorageManagementReview, error) {
	state, err := s.RefreshStorageManagement(ctx)
	if err != nil {
		return StorageManagementReview{}, err
	}
	snapshotProfiles := map[string]bool{}
	if recovery, recoveryErr := s.LocalRecoveryState(); recoveryErr == nil {
		for _, snapshot := range recovery.Snapshots {
			if snapshot.Status == localrecovery.SnapshotVerified {
				snapshotProfiles[snapshot.SourceProfileID] = true
			}
		}
	} else {
		state.Limitations = append(state.Limitations, "Snapshot-aware repair suggestions are unavailable: "+recoveryErr.Error())
	}
	return StorageManagementReview{
		State: state, RepairPlans: buildStorageRepairPlans(state.Inventory, snapshotProfiles),
	}, nil
}

func buildStorageRepairPlans(inventory lifecycle.StorageInventory, snapshotProfiles map[string]bool) []StorageRepairPlan {
	plans := make([]StorageRepairPlan, 0)
	seen := map[string]struct{}{}
	add := func(plan StorageRepairPlan) {
		plan.Automatic = false
		plan.ID = localRecoveryID("storage-repair-plan", plan.Kind, plan.ProfileID, plan.RelativePath, plan.ReasonCode)
		if _, exists := seen[plan.ID]; exists {
			return
		}
		seen[plan.ID] = struct{}{}
		plans = append(plans, plan)
	}

	profilePaths := make(map[string]struct{}, len(inventory.Profiles))
	for _, profile := range inventory.Profiles {
		profilePaths[profile.ManagedDir] = struct{}{}
		switch profile.Status {
		case lifecycle.InventoryMissing:
			if snapshotProfiles[profile.ProfileID] {
				add(StorageRepairPlan{
					Kind: "review-snapshot-restore", ProfileID: profile.ProfileID, RelativePath: profile.ManagedDir,
					ReasonCode: "managed-directory-missing-with-verified-snapshot", Risk: "warning",
					Description: "Review a verified local snapshot and restore it to a new Profile identity. The missing directory is not recreated automatically.",
				})
			} else {
				add(StorageRepairPlan{
					Kind: "inspect-missing-profile-data", ProfileID: profile.ProfileID, RelativePath: profile.ManagedDir,
					ReasonCode: "managed-directory-missing-without-snapshot", Risk: "danger",
					Description: "Inspect the missing managed directory and lifecycle history. No verified local snapshot is currently available.",
				})
			}
		case lifecycle.InventoryUnsafe:
			add(StorageRepairPlan{
				Kind: "manual-security-review", ProfileID: profile.ProfileID, RelativePath: profile.ManagedDir,
				ReasonCode: nonEmptyReason(profile.ReasonCode, "managed-entry-unsafe"), Risk: "danger",
				Description: "Stop using this Profile until the linked, reparse, special, or escaped storage entry is reviewed manually.",
			})
		case lifecycle.InventoryIncomplete:
			add(StorageRepairPlan{
				Kind: "rerun-bounded-inventory", ProfileID: profile.ProfileID, RelativePath: profile.ManagedDir,
				ReasonCode: nonEmptyReason(profile.ReasonCode, "inventory-incomplete"), Risk: "warning",
				Description: "Rerun the bounded inventory after resolving access, size, duration, or transient filesystem limitations.",
			})
		}
	}
	for _, orphan := range inventory.Orphans {
		add(StorageRepairPlan{
			Kind: "review-orphan-ownership", RelativePath: orphan.RelativePath,
			ReasonCode: nonEmptyReason(orphan.ReasonCode, "unregistered-profile-directory"), Risk: "warning",
			Description: "Review ownership and lifecycle history before importing, quarantining, or deleting this unregistered directory.",
		})
	}
	for _, unsafe := range inventory.Unsafe {
		if _, belongsToProfile := profilePaths[unsafe.RelativePath]; belongsToProfile {
			continue
		}
		add(StorageRepairPlan{
			Kind: "manual-security-review", RelativePath: unsafe.RelativePath,
			ReasonCode: nonEmptyReason(unsafe.ReasonCode, "unsafe-storage-entry"), Risk: "danger",
			Description: "Inspect this unsafe storage entry manually. Veilium will not follow, repair, move, or delete it automatically.",
		})
	}
	if inventory.Incomplete {
		add(StorageRepairPlan{
			Kind: "rerun-bounded-inventory", ReasonCode: "inventory-scan-incomplete", Risk: "warning",
			Description: "The inventory is incomplete. Resolve the reported limitations and run the read-only scan again.",
		})
	}

	sort.Slice(plans, func(i, j int) bool {
		left, right := storageRiskRank(plans[i].Risk), storageRiskRank(plans[j].Risk)
		if left != right {
			return left < right
		}
		return plans[i].ID < plans[j].ID
	})
	return plans
}

func storageRiskRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "danger":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}

func nonEmptyReason(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
