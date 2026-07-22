package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (e *ArchiveExecutor) Archive(ctx context.Context, request ArchiveRequest) (ArchiveResult, error) {
	return e.executeArchiveTransition(ctx, lifecycle.OperationArchive, request)
}

func (e *ArchiveExecutor) Unarchive(ctx context.Context, request ArchiveRequest) (ArchiveResult, error) {
	return e.executeArchiveTransition(ctx, lifecycle.OperationUnarchive, request)
}

func (e *ArchiveExecutor) executeArchiveTransition(ctx context.Context, operationType lifecycle.OperationType, request ArchiveRequest) (ArchiveResult, error) {
	if e == nil || e.records == nil || e.journal == nil || e.coordinator == nil {
		return ArchiveResult{}, fmt.Errorf("archive executor is unavailable")
	}
	if operationType != lifecycle.OperationArchive && operationType != lifecycle.OperationUnarchive {
		return ArchiveResult{}, fmt.Errorf("%w: unsupported archive transition %q", ErrInvalidRecord, operationType)
	}
	if err := request.Validate(); err != nil {
		return ArchiveResult{}, err
	}
	if err := checkArchiveContext(ctx); err != nil {
		return ArchiveResult{}, err
	}

	operation := newArchiveOperation(operationType, request, e.now())
	started, reused, err := e.coordinator.Begin(operation)
	if err != nil {
		return ArchiveResult{}, err
	}
	if reused {
		return e.resultForReusedArchive(request, operationType, started)
	}

	preflightStage := archiveStage(operationType, "preflight")
	if _, err := e.setArchiveStage(request.OperationID, preflightStage); err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	if record.Lock == nil || record.Lock.OperationID != request.OperationID {
		return e.abortArchive(request, operationType, started, lifecycle.ErrConflict)
	}
	if err := validateArchiveManagedLocation(e.dataRoot, record); err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	if err := e.checkArchiveCancellation(ctx, request.OperationID); err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	if _, err := e.setArchiveStage(request.OperationID, archiveStage(operationType, "committing")); err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	if err := e.checkArchiveCancellation(ctx, request.OperationID); err != nil {
		return e.abortArchive(request, operationType, started, err)
	}

	updated, originCode, err := e.commitArchiveTransition(request, operationType)
	if err != nil {
		return e.abortArchive(request, operationType, started, err)
	}
	return e.finishArchiveTransition(request, operationType, started, updated, originCode)
}

func (e *ArchiveExecutor) commitArchiveTransition(request ArchiveRequest, operationType lifecycle.OperationType) (lifecycle.Record, string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		record, err := e.records.Get(request.ProfileID)
		if err != nil {
			return lifecycle.Record{}, "", err
		}
		if record.Lock == nil || record.Lock.OperationID != request.OperationID {
			return lifecycle.Record{}, "", lifecycle.ErrConflict
		}
		if err := validateArchiveManagedLocation(e.dataRoot, record); err != nil {
			return lifecycle.Record{}, "", err
		}

		now := e.now().UTC()
		var originCode string
		switch operationType {
		case lifecycle.OperationArchive:
			originCode, err = prepareArchiveRecord(&record, now)
		case lifecycle.OperationUnarchive:
			originCode, err = prepareUnarchiveRecord(&record)
		default:
			err = fmt.Errorf("%w: unsupported archive transition", ErrInvalidRecord)
		}
		if err != nil {
			return lifecycle.Record{}, "", err
		}

		updated, updateErr := e.records.Update(record)
		if errors.Is(updateErr, lifecycle.ErrConflict) {
			continue
		}
		return updated, originCode, updateErr
	}
	return lifecycle.Record{}, "", lifecycle.ErrConflict
}

func prepareArchiveRecord(record *lifecycle.Record, archivedAt time.Time) (string, error) {
	if record == nil {
		return "", fmt.Errorf("%w: lifecycle record is required", ErrInvalidRecord)
	}
	if record.ArchivedAt != nil || record.TrashedAt != nil || record.RetentionDeadline != nil {
		return "", fmt.Errorf("%w: lifecycle timestamps contradict archive source state", ErrLifecycleStorageRecoveryRequired)
	}
	if hasLifecycleCode(record.LimitationCodes, "profile-archived") ||
		hasLifecycleCode(record.LimitationCodes, "archive-origin-available") ||
		hasLifecycleCode(record.LimitationCodes, "archive-origin-draft") {
		return "", fmt.Errorf("%w: archive origin metadata already exists", ErrLifecycleStorageRecoveryRequired)
	}
	originCode, err := archiveOriginCode(record.State)
	if err != nil {
		return "", err
	}
	record.State = lifecycle.StateArchived
	record.ArchivedAt = timePointer(archivedAt)
	record.LimitationCodes = addLifecycleCodes(record.LimitationCodes, "profile-archived", originCode)
	return originCode, nil
}

func prepareUnarchiveRecord(record *lifecycle.Record) (string, error) {
	if record == nil {
		return "", fmt.Errorf("%w: lifecycle record is required", ErrInvalidRecord)
	}
	if record.State != lifecycle.StateArchived {
		return "", fmt.Errorf("%w: lifecycle state %q cannot be unarchived", lifecycle.ErrConflict, record.State)
	}
	if record.ArchivedAt == nil || record.TrashedAt != nil || record.RetentionDeadline != nil {
		return "", fmt.Errorf("%w: archived lifecycle timestamps are contradictory", ErrLifecycleStorageRecoveryRequired)
	}
	if !hasLifecycleCode(record.LimitationCodes, "profile-archived") {
		return "", fmt.Errorf("%w: archived limitation marker is missing", ErrLifecycleStorageRecoveryRequired)
	}
	origin, originCode, err := archivedOrigin(*record)
	if err != nil {
		return "", err
	}
	record.State = origin
	record.ArchivedAt = nil
	record.LimitationCodes = removeLifecycleCodes(
		record.LimitationCodes,
		"profile-archived",
		"archive-origin-available",
		"archive-origin-draft",
	)
	return originCode, nil
}

func (e *ArchiveExecutor) finishArchiveTransition(request ArchiveRequest, operationType lifecycle.OperationType, started lifecycle.Operation, updated lifecycle.Record, originCode string) (ArchiveResult, error) {
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         lifecycle.ItemSucceeded,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: archiveStage(operationType, "finished"),
		OutputID:       request.ProfileID,
		Limitations:    []string{originCode},
	}
	finished, err := e.coordinator.Finish(request.OperationID, lifecycle.OperationCompleted, []lifecycle.OperationItemResult{item}, nil, nil)
	if err != nil {
		code := archiveRecoveryCode(operationType, "operation-finalization-required")
		e.markArchiveRecovery(request.ProfileID, code)
		current, _ := e.records.Get(request.ProfileID)
		return ArchiveResult{Operation: started, Record: current}, fmt.Errorf("%w: lifecycle state changed but operation finalization failed: %v", ErrLifecycleStorageRecoveryRequired, err)
	}
	current, err := e.records.Get(request.ProfileID)
	if err != nil {
		return ArchiveResult{Operation: finished, Record: updated}, fmt.Errorf("%w: finalized lifecycle state cannot be read: %v", ErrLifecycleStorageRecoveryRequired, err)
	}
	if current.Lock != nil {
		return ArchiveResult{Operation: finished, Record: current}, fmt.Errorf("%w: finalized lifecycle state remains locked", ErrLifecycleStorageRecoveryRequired)
	}
	return ArchiveResult{Operation: finished, Record: current}, nil
}

func (e *ArchiveExecutor) abortArchive(request ArchiveRequest, operationType lifecycle.OperationType, started lifecycle.Operation, cause error) (ArchiveResult, error) {
	status := lifecycle.OperationFailed
	itemStatus := lifecycle.ItemFailed
	reasonCode := archiveReasonCode(cause)
	recoveryActions := []string(nil)
	if errors.Is(cause, ErrLifecycleStorageCancelled) || errors.Is(cause, context.Canceled) {
		status = lifecycle.OperationCancelled
		itemStatus = lifecycle.ItemCancelled
	}
	if errors.Is(cause, ErrLifecycleStorageRecoveryRequired) {
		status = lifecycle.OperationRecoveryRequired
		itemStatus = lifecycle.ItemRecoveryRequired
		recoveryActions = []string{"inspect-lifecycle-storage-state"}
	}
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         itemStatus,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: archiveStage(operationType, "finished"),
		ReasonCode:     reasonCode,
	}
	finished, finishErr := e.coordinator.Finish(request.OperationID, status, []lifecycle.OperationItemResult{item}, nil, recoveryActions)
	if finishErr != nil {
		code := archiveRecoveryCode(operationType, "failure-finalization-required")
		e.markArchiveRecovery(request.ProfileID, code)
		current, _ := e.records.Get(request.ProfileID)
		return ArchiveResult{Operation: started, Record: current}, fmt.Errorf("%w: lifecycle failure could not be finalized: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, finishErr, cause)
	}
	current, readErr := e.records.Get(request.ProfileID)
	if readErr != nil {
		return ArchiveResult{Operation: finished}, fmt.Errorf("%w: lifecycle failure finalized but state cannot be read: %v", ErrLifecycleStorageRecoveryRequired, readErr)
	}
	result := ArchiveResult{Operation: finished, Record: current}
	if status == lifecycle.OperationRecoveryRequired {
		e.markArchiveRecovery(request.ProfileID, reasonCode)
		result.Record, _ = e.records.Get(request.ProfileID)
		return result, fmt.Errorf("%w: %v", ErrLifecycleStorageRecoveryRequired, cause)
	}
	if status == lifecycle.OperationCancelled {
		return result, ErrLifecycleStorageCancelled
	}
	return result, cause
}

func (e *ArchiveExecutor) resultForReusedArchive(request ArchiveRequest, operationType lifecycle.OperationType, operation lifecycle.Operation) (ArchiveResult, error) {
	if operation.Type != operationType || len(operation.ProfileIDs) != 1 || operation.ProfileIDs[0] != request.ProfileID || operation.IdempotencyKey != archiveIdempotencyKey(operationType, request) {
		return ArchiveResult{Operation: operation}, lifecycle.ErrConflict
	}
	if !operation.Status.Terminal() {
		return ArchiveResult{Operation: operation}, fmt.Errorf("%w: lifecycle storage operation is already running", lifecycle.ErrConflict)
	}
	if operation.Status != lifecycle.OperationCompleted {
		if operation.Status == lifecycle.OperationRecoveryRequired {
			return ArchiveResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
		}
		return ArchiveResult{Operation: operation}, fmt.Errorf("%w: reused lifecycle storage operation ended with %s", lifecycle.ErrConflict, operation.Status)
	}
	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return ArchiveResult{Operation: operation}, fmt.Errorf("%w: completed lifecycle storage operation has no record", ErrLifecycleStorageRecoveryRequired)
	}
	if record.Lock != nil {
		return ArchiveResult{Operation: operation, Record: record}, fmt.Errorf("%w: completed lifecycle storage operation remains locked", ErrLifecycleStorageRecoveryRequired)
	}
	originCode, err := operationArchiveOrigin(operation, request.ProfileID)
	if err != nil {
		return ArchiveResult{Operation: operation, Record: record}, err
	}
	switch operationType {
	case lifecycle.OperationArchive:
		if record.State != lifecycle.StateArchived || record.ArchivedAt == nil || !hasLifecycleCode(record.LimitationCodes, "profile-archived") || !hasLifecycleCode(record.LimitationCodes, originCode) {
			return ArchiveResult{Operation: operation, Record: record}, fmt.Errorf("%w: completed archive state is contradictory", ErrLifecycleStorageRecoveryRequired)
		}
	case lifecycle.OperationUnarchive:
		expectedState := lifecycle.StateAvailable
		if originCode == "archive-origin-draft" {
			expectedState = lifecycle.StateDraft
		}
		if record.State != expectedState || record.ArchivedAt != nil || hasLifecycleCode(record.LimitationCodes, "profile-archived") || hasLifecycleCode(record.LimitationCodes, "archive-origin-available") || hasLifecycleCode(record.LimitationCodes, "archive-origin-draft") {
			return ArchiveResult{Operation: operation, Record: record}, fmt.Errorf("%w: completed unarchive state is contradictory", ErrLifecycleStorageRecoveryRequired)
		}
	}
	return ArchiveResult{Operation: operation, Record: record}, nil
}

func (e *ArchiveExecutor) setArchiveStage(operationID, stage string) (lifecycle.Operation, error) {
	for attempt := 0; attempt < 3; attempt++ {
		operation, err := e.journal.Get(operationID)
		if err != nil {
			return lifecycle.Operation{}, err
		}
		if operation.Status.Terminal() {
			return lifecycle.Operation{}, lifecycle.ErrConflict
		}
		operation.Stage = stage
		operation.SafeCancellationStage = stage
		updated, err := e.journal.Update(operation)
		if errors.Is(err, lifecycle.ErrConflict) {
			continue
		}
		return updated, err
	}
	return lifecycle.Operation{}, lifecycle.ErrConflict
}

func (e *ArchiveExecutor) checkArchiveCancellation(ctx context.Context, operationID string) error {
	if err := checkArchiveContext(ctx); err != nil {
		return err
	}
	operation, err := e.journal.Get(operationID)
	if err != nil {
		return err
	}
	if operation.Status.Terminal() {
		return lifecycle.ErrConflict
	}
	if operation.CancellationRequested {
		return ErrLifecycleStorageCancelled
	}
	return nil
}

func (e *ArchiveExecutor) markArchiveRecovery(profileID, code string) {
	_, _, _ = e.records.AddRecoveryCode(profileID, code)
}

func validateArchiveManagedLocation(dataRoot string, record lifecycle.Record) error {
	expected := filepath.ToSlash(filepath.Join("profiles", record.ProfileID))
	if record.ManagedDir != expected {
		return fmt.Errorf("%w: lifecycle managed directory is not the Profile-owned location", lifecycle.ErrConflict)
	}
	candidate := filepath.Join(dataRoot, filepath.FromSlash(record.ManagedDir))
	if !pathContainedBy(candidate, dataRoot) {
		return fmt.Errorf("%w: lifecycle managed directory escapes the application root", lifecycle.ErrConflict)
	}
	current := filepath.Clean(candidate)
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("%w: lifecycle managed location is not a real directory", lifecycle.ErrConflict)
			}
			return inspectDirectoryTree(dataRoot, current)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect lifecycle managed location: %w", err)
		}
		if filepath.Clean(current) == filepath.Clean(dataRoot) {
			return inspectRealDirectory(dataRoot)
		}
		parent := filepath.Dir(current)
		if parent == current || !pathContainedBy(parent, dataRoot) {
			return fmt.Errorf("%w: lifecycle managed location has no safe ancestor", lifecycle.ErrConflict)
		}
		current = parent
	}
}

func operationArchiveOrigin(operation lifecycle.Operation, profileID string) (string, error) {
	origins := make([]string, 0, 2)
	for _, item := range operation.Items {
		if item.ItemID != profileID {
			continue
		}
		for _, code := range item.Limitations {
			if code == "archive-origin-available" || code == "archive-origin-draft" {
				origins = append(origins, code)
			}
		}
	}
	sort.Strings(origins)
	origins = uniqueStrings(origins)
	if len(origins) != 1 {
		return "", fmt.Errorf("%w: completed archive operation origin is missing or contradictory", ErrLifecycleStorageRecoveryRequired)
	}
	return origins[0], nil
}

func archiveStage(operationType lifecycle.OperationType, suffix string) string {
	return string(operationType) + "-" + suffix
}

func archiveRecoveryCode(operationType lifecycle.OperationType, suffix string) string {
	return string(operationType) + "-" + suffix
}

func archiveReasonCode(err error) string {
	switch {
	case errors.Is(err, ErrLifecycleStorageCancelled), errors.Is(err, context.Canceled):
		return "lifecycle-storage-cancelled"
	case errors.Is(err, ErrLifecycleStorageRecoveryRequired):
		return "lifecycle-storage-recovery-required"
	case errors.Is(err, lifecycle.ErrConflict):
		return "lifecycle-storage-conflict"
	case errors.Is(err, context.DeadlineExceeded):
		return "lifecycle-storage-deadline-exceeded"
	default:
		return "lifecycle-storage-failed"
	}
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
