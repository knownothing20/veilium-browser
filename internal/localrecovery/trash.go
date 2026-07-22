package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (e *TrashExecutor) Trash(ctx context.Context, request TrashRequest) (TrashResult, error) {
	if e == nil || e.records == nil || e.journal == nil || e.coordinator == nil || e.profiles == nil || e.trash == nil {
		return TrashResult{}, fmt.Errorf("trash executor is unavailable")
	}
	if err := request.Validate(); err != nil {
		return TrashResult{}, err
	}
	if err := checkTrashContext(ctx); err != nil {
		return TrashResult{}, err
	}
	trashID := trashIDFor(request)
	operation := newTrashOperation(lifecycle.OperationTrash, request.OperationID, request.ProfileID, trashID, request.IdempotencyKey, request.ApplicationVersion, e.now())
	started, reused, err := e.coordinator.Begin(operation)
	if err != nil {
		return TrashResult{}, err
	}
	if reused {
		return e.resultForReusedTrash(request, trashID, started)
	}
	if _, err := e.setStage(request.OperationID, "trash-preflight", "", ""); err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}

	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	if record.Lock == nil || record.Lock.OperationID != request.OperationID {
		return e.abortTrash(request.ProfileID, trashID, started, lifecycle.ErrConflict)
	}
	if err := validateTrashLifecycleSource(record); err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	item, err := e.profiles.Get(request.ProfileID)
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	definition, profileDigest, err := profileDefinitionForTrash(item)
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	sourceRoot, err := managedSourcePath(e.dataRoot, record.ManagedDir)
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	expectedSource := filepath.Join(e.dataRoot, "profiles", request.ProfileID)
	if filepath.Clean(sourceRoot) != filepath.Clean(expectedSource) {
		return e.abortTrash(request.ProfileID, trashID, started, lifecycle.ErrConflict)
	}
	plan, err := planTrashTree(ctx, sourceRoot, func(ctx context.Context) error {
		return e.checkCancellation(ctx, request.OperationID)
	})
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	entries, treeDigest, err := hashTrashPlan(plan)
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}
	if int64(len(entries)) != int64(len(plan.Files)) {
		return e.abortTrash(request.ProfileID, trashID, started, ErrSourceChanged)
	}

	now := e.now().UTC()
	trashRecord, err := e.trash.Create(TrashRecord{
		TrashID:                 trashID,
		ProfileID:               request.ProfileID,
		OperatingSystem:         runtime.GOOS,
		Architecture:            runtime.GOARCH,
		OriginalState:           record.State,
		OriginalManagedDir:      record.ManagedDir,
		OriginalArchivedAt:      cloneTime(record.ArchivedAt),
		OriginalSourceID:        record.SourceID,
		OriginalRecoveryCodes:   append([]string(nil), record.RecoveryCodes...),
		OriginalLimitationCodes: append([]string(nil), record.LimitationCodes...),
		TrashRef:                filepath.ToSlash(filepath.Join("local-recovery", trashFinalDirectory, trashID, browserDataDirectory)),
		DataPresent:             true,
		ProfileDefinitionDigest: profileDigest,
		TreeDigest:              treeDigest,
		FileCount:               int64(len(entries)),
		TotalBytes:              plan.TotalBytes,
		TrashedAt:               now,
		RetentionDeadline:       now.Add(request.retentionDuration()),
		Limitations:             []string{"local-only", "retention-metadata-only"},
	})
	if err != nil {
		return e.abortTrash(request.ProfileID, trashID, started, err)
	}

	stageRoot, err := createPrivateOperationStage(filepath.Join(e.recoveryRoot, trashStagingDirectory), request.OperationID)
	if err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, "", err)
	}
	if err := writeExclusiveFile(filepath.Join(stageRoot, profileDefinitionName), definition); err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, stageRoot, err)
	}
	if _, err := e.setStage(request.OperationID, "trash-ready-to-move", trashStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(trashRecord.TrashRef))); err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, stageRoot, err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, stageRoot, err)
	}
	if err := e.verifyRetainedProfile(request.ProfileID, trashRecord.ProfileDefinitionDigest); err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, stageRoot, err)
	}

	stagedBrowser := filepath.Join(stageRoot, browserDataDirectory)
	if err := e.rename(sourceRoot, stagedBrowser); err != nil {
		return e.abortTrashBeforeMove(request, started, trashRecord, stageRoot, fmt.Errorf("move Profile data to trash staging: %w", err))
	}
	syncDirectory(filepath.Dir(sourceRoot))
	if _, err := e.setStage(request.OperationID, "trash-verifying", trashStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(trashRecord.TrashRef))); err != nil {
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, "", err)
	}
	movedEntries, movedDigest, err := verifyMovedTrashTree(ctx, stagedBrowser, plan, func(context.Context) error { return nil })
	if err != nil || movedDigest != trashRecord.TreeDigest || int64(len(movedEntries)) != trashRecord.FileCount {
		if err == nil {
			err = fmt.Errorf("%w: staged trash tree identity changed", ErrInvalidManifest)
		}
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, "", err)
	}

	finalRoot := trashRootPath(e.recoveryRoot, trashID)
	if err := ensureNoExistingPath(finalRoot); err != nil {
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, "", err)
	}
	if _, err := e.setStage(request.OperationID, "trash-publishing", trashStageRef(request.OperationID), filepath.ToSlash(filepath.Dir(trashRecord.TrashRef))); err != nil {
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, "", err)
	}
	if err := e.rename(stageRoot, finalRoot); err != nil {
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, "", fmt.Errorf("publish trash payload: %w", err))
	}
	syncDirectory(filepath.Dir(finalRoot))
	trashRecord.Status = TrashStored
	stored, err := e.trash.Update(trashRecord)
	if err != nil {
		return e.rollbackMovedTrash(request, started, trashRecord, sourceRoot, stageRoot, finalRoot, err)
	}
	updated, err := e.commitTrashedLifecycle(request, record, stored)
	if err != nil {
		return e.rollbackCommittedTrash(request, started, stored, sourceRoot, stageRoot, finalRoot, err)
	}
	return e.finishTrash(request, started, updated, stored)
}

func (e *TrashExecutor) commitTrashedLifecycle(request TrashRequest, original lifecycle.Record, trashRecord TrashRecord) (lifecycle.Record, error) {
	for attempt := 0; attempt < 3; attempt++ {
		current, err := e.records.Get(request.ProfileID)
		if err != nil {
			return lifecycle.Record{}, err
		}
		if current.Lock == nil || current.Lock.OperationID != request.OperationID {
			return lifecycle.Record{}, lifecycle.ErrConflict
		}
		if current.State != original.State || current.ManagedDir != original.ManagedDir || !timePointersEqual(current.ArchivedAt, original.ArchivedAt) {
			return lifecycle.Record{}, lifecycle.ErrConflict
		}
		current.State = lifecycle.StateTrashed
		current.ArchivedAt = nil
		current.TrashedAt = cloneTime(&trashRecord.TrashedAt)
		current.RetentionDeadline = cloneTime(&trashRecord.RetentionDeadline)
		current.LimitationCodes = trashCurrentLimitations(original.LimitationCodes, original.State, trashRecord.TrashID)
		updated, updateErr := e.records.Update(current)
		if errors.Is(updateErr, lifecycle.ErrConflict) {
			continue
		}
		return updated, updateErr
	}
	return lifecycle.Record{}, lifecycle.ErrConflict
}

func (e *TrashExecutor) finishTrash(request TrashRequest, started lifecycle.Operation, record lifecycle.Record, trashRecord TrashRecord) (TrashResult, error) {
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         lifecycle.ItemSucceeded,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: "trash-finished",
		FilesProcessed: trashRecord.FileCount,
		BytesProcessed: trashRecord.TotalBytes,
		OutputID:       trashRecord.TrashID,
		Limitations:    []string{"retention-metadata-only"},
	}
	finished, err := e.coordinator.Finish(request.OperationID, lifecycle.OperationCompleted, []lifecycle.OperationItemResult{item}, []string{"retention-metadata-only"}, nil)
	if err != nil {
		e.markRecovery(request.ProfileID, "trash-operation-finalization-required")
		return TrashResult{Operation: started, Record: record, Trash: trashRecord}, fmt.Errorf("%w: trash committed but operation finalization failed: %v", ErrLifecycleStorageRecoveryRequired, err)
	}
	current, readErr := e.records.Get(request.ProfileID)
	if readErr != nil || current.Lock != nil {
		return TrashResult{Operation: finished, Record: record, Trash: trashRecord}, fmt.Errorf("%w: committed trash state cannot be read unlocked", ErrLifecycleStorageRecoveryRequired)
	}
	return TrashResult{Operation: finished, Record: current, Trash: trashRecord}, nil
}

func (e *TrashExecutor) abortTrashBeforeMove(request TrashRequest, started lifecycle.Operation, trashRecord TrashRecord, stageRoot string, cause error) (TrashResult, error) {
	cleanupErr := error(nil)
	if stageRoot != "" {
		cleanupErr = e.removeOwnedStage(stageRoot, filepath.Join(e.recoveryRoot, trashStagingDirectory))
	}
	if cleanupErr == nil {
		cleanupErr = e.removeTrashRecord(trashRecord)
	}
	if cleanupErr != nil {
		e.markTrashRecordRecovery(trashRecord, "trash-precommit-cleanup-required")
		e.markRecovery(request.ProfileID, "trash-precommit-cleanup-required")
		return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, fmt.Errorf("%w: precommit cleanup failed: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, cleanupErr, cause))
	}
	return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, cause)
}

func (e *TrashExecutor) rollbackMovedTrash(request TrashRequest, started lifecycle.Operation, trashRecord TrashRecord, sourceRoot, stageRoot, finalRoot string, cause error) (TrashResult, error) {
	rollbackErr := e.restoreTrashPayloadToSource(sourceRoot, stageRoot, finalRoot)
	if rollbackErr != nil {
		e.markTrashRecordRecovery(trashRecord, "trash-payload-rollback-required")
		e.markRecovery(request.ProfileID, "trash-payload-rollback-required")
		return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, fmt.Errorf("%w: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, rollbackErr, cause))
	}
	if removeErr := e.removeTrashRecord(trashRecord); removeErr != nil {
		e.markTrashRecordRecovery(trashRecord, "trash-catalog-cleanup-required")
		e.markRecovery(request.ProfileID, "trash-catalog-cleanup-required")
		return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, fmt.Errorf("%w: trash payload rolled back but catalog cleanup failed: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, removeErr, cause))
	}
	return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, cause)
}

func (e *TrashExecutor) rollbackCommittedTrash(request TrashRequest, started lifecycle.Operation, trashRecord TrashRecord, sourceRoot, stageRoot, finalRoot string, cause error) (TrashResult, error) {
	rollbackErr := e.restoreTrashPayloadToSource(sourceRoot, stageRoot, finalRoot)
	if rollbackErr != nil {
		e.markTrashRecordRecovery(trashRecord, "trash-lifecycle-rollback-required")
		e.markRecovery(request.ProfileID, "trash-lifecycle-rollback-required")
		return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, fmt.Errorf("%w: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, rollbackErr, cause))
	}
	if removeErr := e.removeTrashRecord(trashRecord); removeErr != nil {
		e.markTrashRecordRecovery(trashRecord, "trash-catalog-cleanup-required")
		e.markRecovery(request.ProfileID, "trash-catalog-cleanup-required")
		return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, fmt.Errorf("%w: lifecycle rollback restored the Profile but catalog cleanup failed: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, removeErr, cause))
	}
	return e.abortTrash(request.ProfileID, trashRecord.TrashID, started, cause)
}

func (e *TrashExecutor) restoreTrashPayloadToSource(sourceRoot, stageRoot, finalRoot string) error {
	stagingBoundary := filepath.Join(e.recoveryRoot, trashStagingDirectory)
	if finalRoot != "" {
		if _, err := os.Lstat(finalRoot); err == nil {
			if err := ensureOwnedStage(stageRoot, stagingBoundary); err != nil {
				return err
			}
			if _, err := os.Lstat(stageRoot); err == nil {
				return fmt.Errorf("%w: rollback staging already exists", ErrLifecycleStorageRecoveryRequired)
			} else if !os.IsNotExist(err) {
				return err
			}
			if err := e.rename(finalRoot, stageRoot); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	stagedBrowser := filepath.Join(stageRoot, browserDataDirectory)
	if _, err := os.Lstat(stagedBrowser); err != nil {
		return err
	}
	if err := ensureNoExistingPath(sourceRoot); err != nil {
		return err
	}
	if err := e.rename(stagedBrowser, sourceRoot); err != nil {
		return err
	}
	syncDirectory(filepath.Dir(sourceRoot))
	return e.removeOwnedStage(stageRoot, stagingBoundary)
}

func (e *TrashExecutor) removeOwnedStage(stageRoot, boundary string) error {
	return e.removeTree(stageRoot, boundary)
}

func (e *TrashExecutor) removeTrashRecord(record TrashRecord) error {
	current, err := e.trash.Get(record.TrashID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = e.trash.Remove(current.TrashID, current.Revision)
	return err
}

func (e *TrashExecutor) markTrashRecordRecovery(record TrashRecord, code string) {
	current, err := e.trash.Get(record.TrashID)
	if err != nil {
		return
	}
	current.Status = TrashRecoveryRequired
	current.Limitations = sortedUnique(append(current.Limitations, code))
	_, _ = e.trash.Update(current)
}

func (e *TrashExecutor) abortTrash(profileID, trashID string, started lifecycle.Operation, cause error) (TrashResult, error) {
	status := lifecycle.OperationFailed
	itemStatus := lifecycle.ItemFailed
	recoveryActions := []string(nil)
	if errors.Is(cause, ErrLifecycleStorageCancelled) || errors.Is(cause, context.Canceled) {
		status = lifecycle.OperationCancelled
		itemStatus = lifecycle.ItemCancelled
	}
	if errors.Is(cause, ErrLifecycleStorageRecoveryRequired) {
		status = lifecycle.OperationRecoveryRequired
		itemStatus = lifecycle.ItemRecoveryRequired
		recoveryActions = []string{"inspect-trash-storage-state"}
	}
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         profileID,
		Status:         itemStatus,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: "trash-finished",
		ReasonCode:     trashReasonCode(cause),
		RecoveryID:     trashID,
	}
	finished, finishErr := e.coordinator.Finish(started.ID, status, []lifecycle.OperationItemResult{item}, nil, recoveryActions)
	if finishErr != nil {
		e.markRecovery(profileID, "trash-failure-finalization-required")
		return TrashResult{Operation: started}, fmt.Errorf("%w: trash failure could not be finalized: %v; original error: %v", ErrLifecycleStorageRecoveryRequired, finishErr, cause)
	}
	result := TrashResult{Operation: finished}
	result.Record, _ = e.records.Get(profileID)
	result.Trash, _ = e.trash.Get(trashID)
	if status == lifecycle.OperationRecoveryRequired {
		e.markRecovery(profileID, trashReasonCode(cause))
		return result, fmt.Errorf("%w: %v", ErrLifecycleStorageRecoveryRequired, cause)
	}
	if status == lifecycle.OperationCancelled {
		return result, ErrLifecycleStorageCancelled
	}
	return result, cause
}

func (e *TrashExecutor) resultForReusedTrash(request TrashRequest, trashID string, operation lifecycle.Operation) (TrashResult, error) {
	if err := validateOperationIdentity(operation, lifecycle.OperationTrash, request.OperationID, request.ProfileID, trashID, request.IdempotencyKey); err != nil {
		return TrashResult{Operation: operation}, err
	}
	if !operation.Status.Terminal() {
		return TrashResult{Operation: operation}, lifecycle.ErrConflict
	}
	if operation.Status != lifecycle.OperationCompleted {
		if operation.Status == lifecycle.OperationRecoveryRequired {
			return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
		}
		return TrashResult{Operation: operation}, fmt.Errorf("%w: reused trash operation ended with %s", lifecycle.ErrConflict, operation.Status)
	}
	record, err := e.records.Get(request.ProfileID)
	if err != nil {
		return TrashResult{Operation: operation}, ErrLifecycleStorageRecoveryRequired
	}
	trashRecord, err := e.trash.Get(trashID)
	if err != nil {
		return TrashResult{Operation: operation, Record: record}, ErrLifecycleStorageRecoveryRequired
	}
	if record.State != lifecycle.StateTrashed || record.Lock != nil || trashRecord.Status != TrashStored || trashRecord.ProfileID != request.ProfileID {
		return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, ErrLifecycleStorageRecoveryRequired
	}
	if err := verifyStoredTrash(trashRecord, trashRootPath(e.recoveryRoot, trashID)); err != nil {
		return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, err
	}
	return TrashResult{Operation: operation, Record: record, Trash: trashRecord}, nil
}

func (e *TrashExecutor) trashCatalogRecord(profileID string) (TrashRecord, error) {
	for _, record := range e.trash.List() {
		if record.ProfileID == profileID {
			return record, nil
		}
	}
	return TrashRecord{}, ErrNotFound
}

func trashCompletedAt(now time.Time) *time.Time {
	value := now.UTC()
	return &value
}
