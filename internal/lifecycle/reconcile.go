package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"
)

type ReconciliationAction struct {
	Kind         string `json:"kind"`
	ProfileID    string `json:"profileId,omitempty"`
	OperationID  string `json:"operationId,omitempty"`
	RelativePath string `json:"relativePath,omitempty"`
	ReasonCode   string `json:"reasonCode"`
}

type ReconciliationReport struct {
	GeneratedAt          time.Time              `json:"generatedAt"`
	CompatibilityCreated []string               `json:"compatibilityCreated,omitempty"`
	Actions              []ReconciliationAction `json:"actions,omitempty"`
	Inventory            StorageInventory       `json:"inventory"`
	Limitations          []string               `json:"limitations,omitempty"`
}

type Reconciler struct {
	records *RecordStore
	journal *Journal
	scanner *InventoryScanner
	now     func() time.Time
}

func NewReconciler(records *RecordStore, journal *Journal, scanner *InventoryScanner) (*Reconciler, error) {
	if records == nil || journal == nil || scanner == nil {
		return nil, fmt.Errorf("lifecycle records, journal, and inventory scanner are required")
	}
	return &Reconciler{records: records, journal: journal, scanner: scanner, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, compatibility []CompatibilityInput) (ReconciliationReport, error) {
	now := r.now().UTC()
	report := ReconciliationReport{GeneratedAt: now}
	_, created, err := r.records.EnsureCompatibility(compatibility)
	if err != nil {
		return report, fmt.Errorf("ensure lifecycle compatibility: %w", err)
	}
	report.CompatibilityCreated = created
	for _, id := range created {
		report.Actions = append(report.Actions, ReconciliationAction{Kind: "compatibility-created", ProfileID: id, ReasonCode: "existing-profile-mapped"})
	}

	operations := r.journal.List()
	operationByID := make(map[string]Operation, len(operations))
	for _, operation := range operations {
		operationByID[operation.ID] = operation
		if operation.Status.Terminal() {
			continue
		}
		items := make([]OperationItemResult, 0, len(operation.ProfileIDs))
		for _, profileID := range operation.ProfileIDs {
			completedAt := now
			items = append(items, OperationItemResult{
				ItemID:         profileID,
				Status:         ItemRecoveryRequired,
				CompletedAt:    &completedAt,
				CompletedStage: operation.Stage,
				ReasonCode:     "operation-interrupted",
				RecoveryID:     firstNonEmpty(operation.StagingRef, operation.QuarantineRef),
			})
		}
		operation.Status = OperationRecoveryRequired
		operation.Stage = "startup-reconciliation"
		operation.CompletedAt = &now
		operation.Items = items
		operation.RecoveryActions = appendUnique(operation.RecoveryActions, "inspect-and-retry-operation")
		updated, updateErr := r.journal.Update(operation)
		if updateErr != nil {
			return report, fmt.Errorf("mark operation %q recovery-required: %w", operation.ID, updateErr)
		}
		operationByID[operation.ID] = updated
		report.Actions = append(report.Actions, ReconciliationAction{Kind: "operation-recovery-required", OperationID: operation.ID, ReasonCode: "operation-interrupted"})
	}

	for _, record := range r.records.List() {
		if record.Lock == nil {
			continue
		}
		operation, exists := operationByID[record.Lock.OperationID]
		reason := "stale-lock-operation-missing"
		if exists {
			if !operation.Status.Terminal() {
				return report, fmt.Errorf("%w: operation %q remained non-terminal after reconciliation", ErrConflict, operation.ID)
			}
			reason = "stale-lock-terminal-operation"
		}
		if _, changed, clearErr := r.records.ReconcileLock(record.ProfileID, record.Lock.OperationID, reason); clearErr != nil {
			return report, fmt.Errorf("reconcile lock for profile %q: %w", record.ProfileID, clearErr)
		} else if changed {
			report.Actions = append(report.Actions, ReconciliationAction{Kind: "stale-lock-cleared", ProfileID: record.ProfileID, OperationID: record.Lock.OperationID, ReasonCode: reason})
		}
	}

	for _, operation := range operationByID {
		for kind, relative := range map[string]string{"staging": operation.StagingRef, "quarantine": operation.QuarantineRef} {
			if relative == "" {
				continue
			}
			absolute, resolveErr := r.scanner.resolve(relative)
			if resolveErr != nil {
				report.Actions = append(report.Actions, ReconciliationAction{Kind: kind + "-unsafe", OperationID: operation.ID, RelativePath: relative, ReasonCode: "managed-reference-unsafe"})
				continue
			}
			info, statErr := os.Lstat(absolute)
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			if statErr != nil {
				report.Actions = append(report.Actions, ReconciliationAction{Kind: kind + "-uninspectable", OperationID: operation.ID, RelativePath: relative, ReasonCode: "lstat-failed"})
				continue
			}
			unsafe, reparseErr := pathHasReparsePoint(absolute)
			if reparseErr != nil || info.Mode()&os.ModeSymlink != 0 || unsafe {
				report.Actions = append(report.Actions, ReconciliationAction{Kind: kind + "-unsafe", OperationID: operation.ID, RelativePath: relative, ReasonCode: "unsafe-link-or-reparse"})
				continue
			}
			report.Actions = append(report.Actions, ReconciliationAction{Kind: kind + "-present", OperationID: operation.ID, RelativePath: relative, ReasonCode: "manual-recovery-available"})
		}
	}

	inventory, err := r.scanner.Scan(ctx, r.records.List())
	if err != nil {
		return report, fmt.Errorf("scan lifecycle storage: %w", err)
	}
	report.Inventory = inventory
	if inventory.Incomplete {
		report.Limitations = appendUnique(report.Limitations, "inventory-incomplete")
	}
	sort.Strings(report.CompatibilityCreated)
	sort.Slice(report.Actions, func(i, j int) bool {
		left := report.Actions[i].Kind + "\x00" + report.Actions[i].ProfileID + "\x00" + report.Actions[i].OperationID + "\x00" + report.Actions[i].RelativePath
		right := report.Actions[j].Kind + "\x00" + report.Actions[j].ProfileID + "\x00" + report.Actions[j].OperationID + "\x00" + report.Actions[j].RelativePath
		return left < right
	})
	return report, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
