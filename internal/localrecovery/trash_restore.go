package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (e *TrashExecutor) RestoreTrash(ctx context.Context, request TrashActionRequest) (TrashResult, error) {
	if e == nil || e.records == nil || e.journal == nil || e.coordinator == nil || e.trash == nil {
		return TrashResult{}, fmt.Errorf("trash executor is unavailable")
	}
	if err := request.Validate(false); err != nil {
		return TrashResult{}, err
	}
	if err := checkTrashContext(ctx); err != nil {
		return TrashResult{}, err
	}
	operation := newTrashOperation(lifecycle.OperationRestoreTrash, request.OperationID, request.ProfileID, request.TrashID, request.IdempotencyKey, request.ApplicationVersion, e.now())
	started, reused, err := e.coordinator.Begin(operation)
	if err != nil {
		return TrashResult{}, err
	}
	if reused {
		return e.resultForReusedRestore(request, started)
	}
	if _, err := e.setStage(request.OperationID, "restore-trash-preflight", "", filepath.ToSlash(filepath.Dir(filepath.Join("local-recovery", trashFinalDirectory, request.TrashID)))); err != nil {
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
	sourceRoot := filepath.Join(e.dataRoot, filepath.FromSlash(trashRecord.OriginalManagedDir))
	if !pathContainedBy(sourceRoot, e.dataRoot) {
		return e.abortTrash(request.ProfileID, request.TrashID, started, lifecycle.ErrConflict)
	}
	if err := ensureNoExistingPath(sourceRoot); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	if err := inspectDirectoryTree(e.dataRoot, filepath.Dir(sourceRoot)); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}

	trashRecord.Status = TrashRestoring
	restoring, err := e.trash.Update(trashRecord)
	if err != nil {
		return e.abortTrash(request.ProfileID, request.TrashID, started, err)
	}
	stageRoot, err := prepareEmptyStagePath(filepath.Join(e.recoveryRoot, trashStagingDirectory), request.OperationID)
	if err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, restoring, TrashStored, "", err)
	}
	if _, err := e.setStage(request.OperationID, "restore-trash-ready-to-move", trashStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(restoring.TrashRef))); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, restoring, TrashStored, "", err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, restoring, TrashStored, "", err)
	}
	if err := e.rename(finalRoot, stageRoot); err != nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, restoring, TrashStored, "", fmt.Errorf("move trash payload to restore staging: %w", err))
	}
	if err := verifyStoredTrash(restoring, stageRoot); err != nil {
		return e.rollbackRestoreBeforeActivation(request, started, restoring, stageRoot, finalRoot, err)
	}
	if _, err := e.setStage(request.OperationID, "restore-trash-activating", trashStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(restoring.TrashRef))); err != nil {
		return e.rollbackRestoreBeforeActivation(request, started, restoring, stageRoot, finalRoot, err)
	}
	stagedBrowser := filepath.Join(stageRoot, browserDataDirectory)
	if err := e.rename(stagedBrowser, sourceRoot); err != nil {
		return e.rollbackRestoreBeforeActivation(request, started, restoring, stageRoot, finalRoot, fmt.Errorf("activate restored Profile data: %w", err))
	}
	syncDirectory(filepath.Dir(sourceRoot))

	restoredLifecycle, err := e.commitRestoredLifecycle(request, restoring)
	if err != nil {
		return e.rollbackRestoreAfterActivation(request, started, record, restoring, sourceRoot, stageRoot, finalRoot, err)
	}
	if _, err := e.trash.Remove(restoring.TrashID, restoring.Revision); err != nil {
		return e.rollbackRestoreAfterActivation(request, started, record, restoring, sourceRoot, stageRoot, finalRoot, err)
	}

	cleanupErr := e.removeOwnedStage(stageRoot, filepath.Join(e.recoveryRoot, trashStagingDirectory))
	status := lifecycle.OperationCompleted
	itemStatus := lifecycle.ItemSucceeded
	limitations := []string(nil)
	recoveryActions := []string(nil)
	if cleanupErr != nil {
		status = lifecycle.OperationPartial
		itemStatus = lifecycle.ItemRecoveryRequired
		limitations = []string{"restore-trash-orphan-staging"}
		recoveryActions = []string{"remove-owned-restore-trash-staging"}
		e.markRecovery(request.ProfileID, "restore-trash-orphan-staging")
	}
	completedAt := e.now().UTC()
	itemResult := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         itemStatus,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: "restore-trash-finished",
		FilesProcessed: restoring.FileCount,
		BytesProcessed: restoring.TotalBytes,
		OutputID:       request.ProfileID,
		Limitations:    limitations,
	}
	finished, finishErr := e.coordinator.Finish(request.OperationID, status, []lifecycle.OperationItemResult{itemResult}, limitations, recoveryActions)
	if finishErr != nil {
		e.markRecovery(request.ProfileID, "restore-trash-operation-finalization-required")
		return TrashResult{Operation: started, Record: restoredLifecycle}, fmt.Errorf("%w: restore committed but operation finalization failed: %v", ErrLifecycleStorageRecoveryRequired, finishErr)
	}
	current, readErr := e.records.Get(request.ProfileID)
	if readErr != nil || current.Lock != nil {
		return TrashResult{Operation: finished, Record: restoredLifecycle}, fmt.Errorf("%w: restored lifecycle state cannot be read unlocked", ErrLifecycleStorageRecoveryRequired)
	}
	if cleanupErr != nil {
		return TrashResult{Operation: finished, Record: current}, fmt.Errorf("%w: Profile restored but owned staging cleanup failed: %v", ErrLifecycleStorageRecoveryRequired, cleanupErr)
	}
	return TrashResult{Operation: finished, Record: current}, nil
}

func (e *TrashExecutor) commitRestoredLifecycle(request TrashActionRequest, trashRecord TrashRecord) (lifecycle.Record, error) {
	for attempt := 0; attempt < 3; attempt++ {
		current, err := e.records.Get(request.ProfileID)
		if err != nil {
			return lifecycle.Record{}, err
		}
		if current.Lock == nil || current.Lock.OperationID != request.OperationID || current.State != lifecycle.StateTrashed {
			return lifecycle.Record{}, lifecycle.ErrConflict
		}
		current.State = trashRecord.OriginalState
		current.ManagedDir = trashRecord.OriginalManagedDir
		current.ArchivedAt = cloneTime(trashRecord.OriginalArchivedAt)
		current.TrashedAt = nil
		current.RetentionDeadline = nil
		current.SourceID = trashRecord.OriginalSourceID
		current.RecoveryCodes = append([]string(nil), trashRecord.OriginalRecoveryCodes...)
		current.LimitationCodes = append([]string(nil), trashRecord.OriginalLimitationCodes...)
		updated, updateErr := e.records.Update(current)
		if errors.Is(updateErr, lifecycle.ErrConflict) {
			continue
		}
		return updated, updateErr
	}
	return lifecycle.Record{}, lifecycle.ErrConflict
}

func (e *TrashExecutor) rollbackRestoreBeforeActivation(request TrashActionRequest, started lifecycle.Operation, trashRecord TrashRecord, stageRoot, finalRoot string, cause error) (TrashResult, error) {
	rollbackErr := e.rename(stageRoot, finalRoot)
	if rollbackErr == nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashStored, "", cause)
	}
	e.markRecovery(request.ProfileID, "restore-trash-payload-rollback-required")
	return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashRecoveryRequired, "restore-trash-payload-rollback-required", fmt.Errorf("%w: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, rollbackErr, cause))
}

func (e *TrashExecutor) rollbackRestoreAfterActivation(request TrashActionRequest, started lifecycle.Operation, originalLifecycle lifecycle.Record, trashRecord TrashRecord, sourceRoot, stageRoot, finalRoot string, cause error) (TrashResult, error) {
	lifecycleErr := e.rollbackLifecycleToTrashed(request, originalLifecycle, trashRecord)
	payloadErr := error(nil)
	stagedBrowser := filepath.Join(stageRoot, browserDataDirectory)
	if err := ensureNoExistingPath(stagedBrowser); err != nil {
		payloadErr = err
	} else if err := e.rename(sourceRoot, stagedBrowser); err != nil {
		payloadErr = err
	} else if err := e.rename(stageRoot, finalRoot); err != nil {
		payloadErr = err
	}
	if lifecycleErr == nil && payloadErr == nil {
		return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashStored, "", cause)
	}
	e.markRecovery(request.ProfileID, "restore-trash-activation-rollback-required")
	return e.abortAfterCatalogReset(request.ProfileID, request.TrashID, started, trashRecord, TrashRecoveryRequired, "restore-trash-activation-rollback-required", fmt.Errorf("%w: lifecycle rollback: %v; payload rollback: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, lifecycleErr, payloadErr, cause))
}

func (e *TrashExecutor) rollbackLifecycleToTrashed(request TrashActionRequest, original lifecycle.Record, trashRecord TrashRecord) error {
	for attempt := 0; attempt < 3; attempt++ {
		current, err := e.records.Get(request.ProfileID)
		if err != nil {
			return err
		}
		if current.Lock == nil || current.Lock.OperationID != request.OperationID {
			return lifecycle.ErrConflict
		}
		current.State = lifecycle.StateTrashed
		current.ManagedDir = original.ManagedDir
		current.ArchivedAt = nil
		current.TrashedAt = cloneTime(&trashRecord.TrashedAt)
		current.RetentionDeadline = cloneTime(&trashRecord.RetentionDeadline)
		current.SourceID = original.SourceID
		current.RecoveryCodes = append([]string(nil), original.RecoveryCodes...)
		current.LimitationCodes = trashCurrentLimitations(trashRecord.OriginalLimitationCodes, trashRecord.OriginalState, trashRecord.TrashID)
		_, updateErr := e.records.Update(current)
		if errors.Is(updateErr, lifecycle.ErrConflict) {
			continue
		}
		return updateErr
	}
	return lifecycle.ErrConflict
}

func (e *TrashExecutor) restoreTrashCatalogStatus(record TrashRecord, status TrashStatus, limitation string) error {
	current, err := e.trash.Get(record.TrashID)
	if err != nil {
		return err
	}
	current.Status = status
	if limitation != "" {
		current.Limitations = sortedUnique(append(current.Limitations, limitation))
	}
	_, err = e.trash.Update(current)
	return err
}

func (e *TrashExecutor) abortAfterCatalogReset(profileID, trashID string, started lifecycle.Operation, record TrashRecord, status TrashStatus, limitation string, cause error) (TrashResult, error) {
	if resetErr := e.restoreTrashCatalogStatus(record, status, limitation); resetErr != nil {
		e.markRecovery(profileID, "trash-catalog-rollback-required")
		return e.abortTrash(profileID, trashID, started, fmt.Errorf("%w: trash catalog rollback failed: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, resetErr, cause))
	}
	return e.abortTrash(profileID, trashID, started, cause)
}

func (e *TrashExecutor) resultForReusedRestore(request TrashActionRequest, operation lifecycle.Operation) (TrashResult, error) {
	if err := validateOperationIdentity(operation, lifecycle.OperationRestoreTrash, request.OperationID, request.ProfileID, request.TrashID, request.IdempotencyKey); err != nil {
		return TrashResult{Operation: operation}, err
	}
	if !operation.Status.Terminal() {
		return TrashResult{Operation: operation}, lifecycle.ErrConflict
	}
	if operation.Status != lifecycle.OperationCompleted && operation.Status != lifecycle.OperationPartial {
		if operation.Status == lifecycle.OperationRecoveryRequired {
			return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
		}
		return TrashResult{Operation: operation}, lifecycle.ErrConflict
	}
	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
	}
	if record.State == lifecycle.StateTrashed || record.Lock != nil {
		return TrashResult{Operation: operation, Record: record}, ErrLifecycleStorageRecoveryRequired
	}
	if _, err := e.trash.Get(request.TrashID); !errors.Is(err, ErrNotFound) {
		return TrashResult{Operation: operation, Record: record}, ErrLifecycleStorageRecoveryRequired
	}
	sourceRoot := filepath.Join(e.dataRoot, filepath.FromSlash(record.ManagedDir))
	if _, err := os.Lstat(sourceRoot); err != nil {
		return TrashResult{Operation: operation, Record: record}, ErrLifecycleStorageRecoveryRequired
	}
	return TrashResult{Operation: operation, Record: record}, nil
}
