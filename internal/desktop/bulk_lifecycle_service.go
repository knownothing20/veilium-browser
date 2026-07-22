package desktop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/localrecovery"
)

type BulkLifecycleAction string

const (
	BulkLifecycleArchive   BulkLifecycleAction = "archive"
	BulkLifecycleUnarchive BulkLifecycleAction = "unarchive"
	BulkLifecycleTrash     BulkLifecycleAction = "trash"
)

func (a BulkLifecycleAction) Valid() bool {
	switch a {
	case BulkLifecycleArchive, BulkLifecycleUnarchive, BulkLifecycleTrash:
		return true
	default:
		return false
	}
}

type BulkLifecycleRequest struct {
	ProfileIDs     []string            `json:"profileIds"`
	Action         BulkLifecycleAction `json:"action"`
	RetentionDays  int                 `json:"retentionDays,omitempty"`
	Confirmation   string              `json:"confirmation,omitempty"`
	IdempotencyKey string              `json:"idempotencyKey"`
}

type BulkLifecycleItemResult struct {
	ProfileID      string               `json:"profileId"`
	Status         lifecycle.ItemStatus `json:"status"`
	OperationID    string               `json:"operationId,omitempty"`
	LifecycleState lifecycle.State      `json:"lifecycleState,omitempty"`
	ReasonCode     string               `json:"reasonCode,omitempty"`
	Limitations    []string             `json:"limitations,omitempty"`
}

type BulkLifecycleResult struct {
	RequestID   string                    `json:"requestId"`
	Action      BulkLifecycleAction       `json:"action"`
	Status      lifecycle.OperationStatus `json:"status"`
	Items       []BulkLifecycleItemResult `json:"items"`
	CompletedAt time.Time                 `json:"completedAt"`
	Limitations []string                  `json:"limitations,omitempty"`
}

func (s *Service) BulkApplyProfileLifecycle(ctx context.Context, request BulkLifecycleRequest) (BulkLifecycleResult, error) {
	profileIDs, err := normalizeBulkProfileIDs(request.ProfileIDs)
	if err != nil {
		return BulkLifecycleResult{}, err
	}
	if !request.Action.Valid() {
		return BulkLifecycleResult{}, fmt.Errorf("unsupported bulk lifecycle action %q", request.Action)
	}
	if err := validateLocalRecoveryKey(request.IdempotencyKey); err != nil {
		return BulkLifecycleResult{}, err
	}
	if request.RetentionDays < 0 || request.RetentionDays > localrecovery.MaxRetentionDays {
		return BulkLifecycleResult{}, fmt.Errorf("retention days must be between 0 and %d", localrecovery.MaxRetentionDays)
	}
	if request.Action != BulkLifecycleTrash && (request.RetentionDays != 0 || strings.TrimSpace(request.Confirmation) != "") {
		return BulkLifecycleResult{}, fmt.Errorf("retention and confirmation apply only to recoverable trash")
	}
	if request.Action == BulkLifecycleTrash {
		expected := bulkTrashConfirmation(len(profileIDs))
		if request.Confirmation != expected {
			return BulkLifecycleResult{}, fmt.Errorf("type %q to confirm recoverable trash for the fixed selection", expected)
		}
	}
	if s.lifecycleRecords == nil || s.lifecycleJournal == nil || s.lifecycleCoordinator == nil {
		return BulkLifecycleResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}

	requestID := localRecoveryID(
		"bulk-lifecycle-request",
		string(request.Action),
		request.IdempotencyKey,
		strconv.Itoa(request.RetentionDays),
		strings.Join(profileIDs, ","),
	)
	result := BulkLifecycleResult{
		RequestID: requestID,
		Action:    request.Action,
		Items:     make([]BulkLifecycleItemResult, 0, len(profileIDs)),
		Limitations: []string{
			"Each selected Profile uses its own authoritative M5.1/M5.2 lifecycle operation and journal result.",
			"Bulk trash is recoverable only; permanent deletion is never a bulk action.",
		},
	}
	operationItems := make([]lifecycle.OperationItemResult, 0, len(profileIDs))
	cancelRemaining := false

	for _, profileID := range profileIDs {
		startedAt := time.Now().UTC()
		if cancelRemaining || ctx.Err() != nil {
			cancelRemaining = true
			completedAt := time.Now().UTC()
			item := BulkLifecycleItemResult{
				ProfileID:  profileID,
				Status:     lifecycle.ItemCancelled,
				ReasonCode: "bulk-cancellation-requested",
			}
			result.Items = append(result.Items, item)
			operationItems = append(operationItems, lifecycle.OperationItemResult{
				ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
				CompletedStage: "not-started", ReasonCode: item.ReasonCode,
			})
			continue
		}

		childKey := localRecoveryID("bulk-lifecycle-item", requestID, profileID)
		operationID := bulkLifecycleOperationID(request.Action, profileID, childKey)
		existing, existingErr := s.lifecycleJournal.Get(operationID)
		if existingErr != nil && !errors.Is(existingErr, lifecycle.ErrNotFound) {
			return BulkLifecycleResult{}, existingErr
		}
		if existingErr == nil {
			status, reasonCode, limitations := bulkLifecycleItemOutcome(existing, nil)
			currentState := lifecycle.State("")
			if record, recordErr := s.lifecycleRecords.Get(profileID); recordErr == nil {
				currentState = record.State
			}
			item := BulkLifecycleItemResult{
				ProfileID: profileID, Status: status, OperationID: existing.ID,
				LifecycleState: currentState, ReasonCode: reasonCode, Limitations: limitations,
			}
			result.Items = append(result.Items, item)
			completedAt := time.Now().UTC()
			operationItem := lifecycle.OperationItemResult{
				ItemID: profileID, Status: status, StartedAt: &startedAt, CompletedAt: &completedAt,
				CompletedStage: "bulk-lifecycle-reused", ReasonCode: reasonCode, Limitations: limitations,
			}
			if status == lifecycle.ItemSucceeded {
				operationItem.OutputID = profileID
			}
			operationItems = append(operationItems, operationItem)
			if status == lifecycle.ItemCancelled {
				cancelRemaining = true
			}
			continue
		}
		if errors.Is(existingErr, lifecycle.ErrNotFound) {
			if _, getErr := s.store.Get(profileID); getErr != nil {
				completedAt := time.Now().UTC()
				item := BulkLifecycleItemResult{ProfileID: profileID, Status: lifecycle.ItemFailed, ReasonCode: "profile-read-failed"}
				result.Items = append(result.Items, item)
				operationItems = append(operationItems, lifecycle.OperationItemResult{
					ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
					CompletedStage: "bulk-lifecycle-preflight", ReasonCode: item.ReasonCode,
				})
				continue
			}
			record, recordErr := s.lifecycleRecords.Get(profileID)
			if recordErr != nil {
				completedAt := time.Now().UTC()
				item := BulkLifecycleItemResult{ProfileID: profileID, Status: lifecycle.ItemFailed, ReasonCode: "lifecycle-record-read-failed"}
				result.Items = append(result.Items, item)
				operationItems = append(operationItems, lifecycle.OperationItemResult{
					ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
					CompletedStage: "bulk-lifecycle-preflight", ReasonCode: item.ReasonCode,
				})
				continue
			}
			if record.Lock != nil {
				completedAt := time.Now().UTC()
				item := BulkLifecycleItemResult{ProfileID: profileID, Status: lifecycle.ItemSkipped, LifecycleState: record.State, ReasonCode: "lifecycle-operation-active"}
				result.Items = append(result.Items, item)
				operationItems = append(operationItems, lifecycle.OperationItemResult{
					ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
					CompletedStage: "bulk-lifecycle-preflight", ReasonCode: item.ReasonCode,
				})
				continue
			}
			if s.supervisor.IsActive(profileID) {
				completedAt := time.Now().UTC()
				item := BulkLifecycleItemResult{ProfileID: profileID, Status: lifecycle.ItemSkipped, LifecycleState: record.State, ReasonCode: "browser-session-active"}
				result.Items = append(result.Items, item)
				operationItems = append(operationItems, lifecycle.OperationItemResult{
					ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
					CompletedStage: "bulk-lifecycle-preflight", ReasonCode: item.ReasonCode,
				})
				continue
			}
			if !bulkLifecycleStateAllowed(request.Action, record.State) {
				completedAt := time.Now().UTC()
				item := BulkLifecycleItemResult{ProfileID: profileID, Status: lifecycle.ItemSkipped, LifecycleState: record.State, ReasonCode: "lifecycle-state-not-eligible"}
				result.Items = append(result.Items, item)
				operationItems = append(operationItems, lifecycle.OperationItemResult{
					ItemID: profileID, Status: item.Status, StartedAt: &startedAt, CompletedAt: &completedAt,
					CompletedStage: "bulk-lifecycle-preflight", ReasonCode: item.ReasonCode,
				})
				continue
			}
		}

		operation, actionErr := s.executeBulkLifecycleItem(ctx, request.Action, profileID, request.RetentionDays, childKey)
		status, reasonCode, limitations := bulkLifecycleItemOutcome(operation, actionErr)
		currentState := lifecycle.State("")
		if record, recordErr := s.lifecycleRecords.Get(profileID); recordErr == nil {
			currentState = record.State
		}
		item := BulkLifecycleItemResult{
			ProfileID: profileID, Status: status, OperationID: operation.ID,
			LifecycleState: currentState, ReasonCode: reasonCode, Limitations: limitations,
		}
		result.Items = append(result.Items, item)
		completedAt := time.Now().UTC()
		operationItem := lifecycle.OperationItemResult{
			ItemID: profileID, Status: status, StartedAt: &startedAt, CompletedAt: &completedAt,
			CompletedStage: "bulk-lifecycle-finished", ReasonCode: reasonCode, Limitations: limitations,
		}
		if status == lifecycle.ItemSucceeded {
			operationItem.OutputID = profileID
		}
		operationItems = append(operationItems, operationItem)
		if status == lifecycle.ItemCancelled || ctx.Err() != nil {
			cancelRemaining = true
		}
	}

	result.Status = bulkOperationStatus(operationItems)
	result.CompletedAt = time.Now().UTC()
	if result.Status == lifecycle.OperationPartial {
		result.Limitations = append(result.Limitations, "bulk-lifecycle-partial-result")
	}
	return result, nil
}

func (s *Service) executeBulkLifecycleItem(ctx context.Context, action BulkLifecycleAction, profileID string, retentionDays int, childKey string) (lifecycle.Operation, error) {
	switch action {
	case BulkLifecycleArchive:
		result, err := s.ArchiveProfile(ctx, ArchiveProfileRequest{ProfileID: profileID, IdempotencyKey: childKey})
		return result.Operation, err
	case BulkLifecycleUnarchive:
		result, err := s.UnarchiveProfile(ctx, ArchiveProfileRequest{ProfileID: profileID, IdempotencyKey: childKey})
		return result.Operation, err
	case BulkLifecycleTrash:
		result, err := s.TrashProfile(ctx, TrashProfileRequest{ProfileID: profileID, RetentionDays: retentionDays, IdempotencyKey: childKey})
		return result.Operation, err
	default:
		return lifecycle.Operation{}, fmt.Errorf("unsupported bulk lifecycle action %q", action)
	}
}

func bulkLifecycleOperationID(action BulkLifecycleAction, profileID, childKey string) string {
	prefix := string(action) + "-op"
	if action == BulkLifecycleTrash {
		prefix = "trash-op"
	}
	return localRecoveryID(prefix, profileID, childKey)
}

func bulkLifecycleStateAllowed(action BulkLifecycleAction, state lifecycle.State) bool {
	switch action {
	case BulkLifecycleArchive:
		return state == lifecycle.StateAvailable || state == lifecycle.StateDraft
	case BulkLifecycleUnarchive:
		return state == lifecycle.StateArchived
	case BulkLifecycleTrash:
		return state == lifecycle.StateAvailable || state == lifecycle.StateDraft || state == lifecycle.StateArchived
	default:
		return false
	}
}

func bulkLifecycleItemOutcome(operation lifecycle.Operation, actionErr error) (lifecycle.ItemStatus, string, []string) {
	if operation.ID != "" && !operation.Status.Terminal() {
		return lifecycle.ItemSkipped, "lifecycle-operation-running", nil
	}
	if len(operation.Items) == 1 {
		item := operation.Items[0]
		return item.Status, item.ReasonCode, append([]string(nil), item.Limitations...)
	}
	if actionErr == nil && operation.Status == lifecycle.OperationCompleted {
		return lifecycle.ItemSucceeded, "", nil
	}
	if errors.Is(actionErr, context.Canceled) || errors.Is(actionErr, localrecovery.ErrLifecycleStorageCancelled) || operation.Status == lifecycle.OperationCancelled {
		return lifecycle.ItemCancelled, "bulk-cancellation-requested", nil
	}
	if errors.Is(actionErr, localrecovery.ErrLifecycleStorageRecoveryRequired) || operation.Status == lifecycle.OperationRecoveryRequired {
		return lifecycle.ItemRecoveryRequired, "lifecycle-recovery-required", []string{"manual-lifecycle-recovery-review"}
	}
	if errors.Is(actionErr, lifecycle.ErrConflict) {
		return lifecycle.ItemSkipped, "lifecycle-conflict", nil
	}
	return lifecycle.ItemFailed, "lifecycle-action-failed", nil
}

func bulkTrashConfirmation(profileCount int) string {
	return fmt.Sprintf("TRASH %d PROFILES", profileCount)
}
