package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (e *TrashExecutor) PermanentDelete(ctx context.Context, request TrashActionRequest) (TrashResult, error) {
	if e == nil || e.records == nil || e.journal == nil || e.coordinator == nil || e.profiles == nil || e.trash == nil {
		return TrashResult{}, fmt.Errorf("trash executor is unavailable")
	}
	if err := request.Validate(true); err != nil {
		return TrashResult{}, err
	}
	if err := checkTrashContext(ctx); err != nil {
		return TrashResult{}, err
	}
	operation := newTrashOperation(lifecycle.OperationPermanentDelete, request.OperationID, request.ProfileID, request.TrashID, request.IdempotencyKey, request.ApplicationVersion, e.now())
	started, reused, err := e.coordinator.Begin(operation)
	if err != nil {
		return TrashResult{}, err
	}
	if reused {
		return e.resultForReusedDelete(request, started)
	}
	if _, err := e.setStage(request.OperationID, "permanent-delete-preflight", "", filepath.ToSlash(filepath.Dir(filepath.Join("local-recovery", trashFinalDirectory, request.TrashID)))); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}

	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	if record.Lock == nil || record.Lock.OperationID != request.OperationID || record.State != lifecycle.StateTrashed {
		return e.abortTrash(request.ProfileID, request.TrashID, started, lifecycle.ErrConflict)
	}
	trashRecord, err := e.trash.Get(request.TrashID)
	if err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	if trashRecord.ProfileID != request.ProfileID || trashRecord.Status != TrashStored {
		return e.abortTrash(request.ProfileID, request.TrashID, started, lifecycle.ErrConflict)
	}
	if err := e.verifyRetainedProfile(request.ProfileID, trashRecord.ProfileDefinitionDigest); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	finalRoot := trashRootPath(e.recoveryRoot, request.TrashID)
	if err := verifyStoredTrash(trashRecord, finalRoot); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}

	trashRecord.Status = TrashCleanupPending
	cleanupPending, err := e.trash.Update(trashRecord)
	if err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	stageRoot, err := prepareEmptyStagePath(filepath.Join(e.recoveryRoot, deleteStagingDirectory), request.OperationID)
	if err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, cleanupPending, TrashStored, "", err)
	}
	if _, err := e.setStage(request.OperationID, "permanent-delete-ready", deleteStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(cleanupPending.TrashRef))); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, cleanupPending, TrashStored, "", err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, cleanupPending, TrashStored, "", err)
	}
	if err := e.rename(finalRoot, stageRoot); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, cleanupPending, TrashStored, "", fmt.Errorf("move trash payload to delete staging: %w", err))
	}
	if err := verifyStoredTrash(cleanupPending, stageRoot); err != nil {
		return e.rollbackDeleteBeforeIrreversible(request, started, cleanupPending, stageRoot, finalRoot, err)
	}
	if _, err := e.setStage(request.OperationID, "permanent-delete-irreversible", deleteStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(cleanupPending.TrashRef))); err != nil {
		return e.rollbackDeleteBeforeIrreversible(request, started, cleanupPending, stageRoot, finalRoot, err)
	}

	browserRoot := filepath.Join(stageRoot, browserDataDirectory)
	if err := e.removeTree(browserRoot, stageRoot); err != nil {
		return e.failIrreversibleDelete(request, started, cleanupPending, false, fmt.Errorf("remove trashed browser data: %w", err))
	}
	if err := e.profiles.Delete(request.ProfileID); err != nil && !profileAlreadyDeleted(err) {
		return e.failIrreversibleDelete(request, started, cleanupPending, true, fmt.Errorf("remove Profile metadata: %w", err))
	}
	if err := e.removeOwnedStage(stageRoot, filepath.Join(e.recoveryRoot, deleteStagingDirectory)); err != nil {
		return e.failIrreversibleDelete(request, started, cleanupPending, true, fmt.Errorf("remove permanent-delete staging: %w", err))
	}
	deletedLifecycle, err := e.commitDeletedLifecycle(request, record, cleanupPending)
	if err != nil {
		return e.failIrreversibleDelete(request, started, cleanupPending, true, err)
	}
	deletedAt := e.now().UTC()
	cleanupPending.Status = TrashDeleted
	cleanupPending.DataPresent = false
	cleanupPending.DeletedAt = &deletedAt
	cleanupPending.Limitations = sortedUnique(append(cleanupPending.Limitations, "irreversible-delete-complete", "audit-tombstone-retained"))
	deletedTrash, err := e.trash.Update(cleanupPending)
	if err != nil {
		return e.failIrreversibleDelete(request, started, cleanupPending, true, err)
	}

	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         lifecycle.ItemSucceeded,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: "permanent-delete-finished",
		FilesProcessed: deletedTrash.FileCount,
		BytesProcessed: deletedTrash.TotalBytes,
		OutputID:       deletedTrash.TrashID,
		Limitations:    []string{"audit-tombstone-retained"},
	}
	finished, finishErr := e.coordinator.Finish(request.OperationID, lifecycle.OperationCompleted, []lifecycle.OperationItemResult{item}, []string{"audit-tombstone-retained"}, nil)
	if finishErr != nil {
		e.markRecovery(request.ProfileID, "permanent-delete-operation-finalization-required")
		return TrashResult{Operation: started, Record: deletedLifecycle, Trash: deletedTrash}, fmt.Errorf("%w: permanent deletion committed but operation finalization failed: %v", ErrLifecycleStorageRecoveryRequired, finishErr)
	}
	current, readErr := e.records.Get(request.ProfileID)
	if readErr != nil || current.Lock != nil {
		return TrashResult{Operation: finished, Record: deletedLifecycle, Trash: deletedTrash}, fmt.Errorf("%w: permanent-delete lifecycle tombstone cannot be read unlocked", ErrLifecycleStorageRecoveryRequired)
	}
	return TrashResult{Operation: finished, Record: current, Trash: deletedTrash}, nil
}

func (e *TrashExecutor) commitDeletedLifecycle(request TrashActionRequest, original lifecycle.Record, trashRecord TrashRecord) (lifecycle.Record, error) {
	for attempt := 0; attempt < 3; attempt++ {
		current, err := e.records.Get(request.ProfileID)
		if err != nil {
			return lifecycle.Record{}, err
		}
		if current.Lock == nil || current.Lock.OperationID != request.OperationID || current.State != lifecycle.StateTrashed {
			return lifecycle.Record{}, lifecycle.ErrConflict
		}
		current.State = lifecycle.StateInvalid
		current.ArchivedAt = nil
		current.TrashedAt = nil
		current.RetentionDeadline = nil
		current.LimitationCodes = addLifecycleCodes(removeTrashCodes(current.LimitationCodes, trashRecord.TrashID), "permanent-delete-complete", "lifecycle-audit-tombstone")
		updated, updateErr := e.records.Update(current)
		if errors.Is(updateErr, lifecycle.ErrConflict) {
			continue
		}
		return updated, updateErr
	}
	return lifecycle.Record{}, lifecycle.ErrConflict
}

func (e *TrashExecutor) rollbackDeleteBeforeIrreversible(request TrashActionRequest, started lifecycle.Operation, trashRecord TrashRecord, stageRoot, finalRoot string, cause error) (TrashResult, error) {
	if err := e.rename(stageRoot, finalRoot); err != nil {
		e.markRecovery(request.ProfileID, "permanent-delete-payload-rollback-required")
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashRecoveryRequired, "permanent-delete-payload-rollback-required", fmt.Errorf("%w: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, err, cause))
	}
	return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashStored, "", cause)
}

func (e *TrashExecutor) failIrreversibleDelete(request TrashActionRequest, started lifecycle.Operation, trashRecord TrashRecord, browserDataAbsent bool, cause error) (TrashResult, error) {
	current, err := e.trash.Get(trashRecord.TrashID)
	if err == nil {
		current.Status = TrashRecoveryRequired
		if browserDataAbsent {
			current.DataPresent = false
		}
		current.Limitations = sortedUnique(append(current.Limitations, "permanent-delete-recovery-required"))
		_, _ = e.trash.Update(current)
	}
	e.markRecovery(request.ProfileID, "permanent-delete-recovery-required")
	return e.abortTrash(request.ProfileID, request.TrashID, started, fmt.Errorf("%w: %v", ErrLifecycleStorageRecoveryRequired, cause))
}

func (e *TrashExecutor) resultForReusedDelete(request TrashActionRequest, operation lifecycle.Operation) (TrashResult, error) {
	if err := validateOperationIdentity(operation, lifecycle.OperationPermanentDelete, request.OperationID, request.ProfileID, request.TrashID, request.IdempotencyKey); err != nil {
		return TrashResult{Operation: operation}, err
	}
	if !operation.Status.Terminal() {
		return TrashResult{Operation: operation}, lifecycle.ErrConflict
	}
	if operation.Status != lifecycle.OperationCompleted {
		if operation.Status == lifecycle.OperationRecoveryRequired {
			return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
		}
		return TrashResult{Operation: operation}, lifecycle.ErrConflict
	}
	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
	}
	trashRecord, err := e.trash.Get(request.TrashID)
	if err != nil {
		return TrashResult{Operation: operation, Record: record}, ErrLifecycleStorageRecoveryRequired
	}
	if record.State != lifecycle.StateInvalid || record.Lock != nil || trashRecord.Status != TrashDeleted || trashRecord.DataPresent {
		return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, ErrLifecycleStorageRecoveryRequired
	}
	if _, err := e.profiles.Get(request.ProfileID); !profileAlreadyDeleted(err) {
		return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, ErrLifecycleStorageRecoveryRequired
	}
	for _, candidate := range []string{trashRootPath(e.recoveryRoot, request.TrashID), deleteStagePath(e.recoveryRoot, request.OperationID)} {
		if _, err := os.Lstat(candidate); err == nil || !os.IsNotExist(err) {
			return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, ErrLifecycleStorageRecoveryRequired
		}
	}
	return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, nil
}
