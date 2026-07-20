package localrecovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func (e *RestoreExecutor) Restore(ctx context.Context, request RestoreRequest) (RestoreResult, error) {
	if e == nil || e.profiles == nil || e.records == nil || e.journal == nil || e.coordinator == nil || e.catalog == nil {
		return RestoreResult{}, fmt.Errorf("local restore executor is unavailable")
	}
	if err := request.Validate(); err != nil {
		return RestoreResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.duration(request))
	defer cancel()

	destinationID := restoreDestinationID(request.OperationID, request.SnapshotID)
	managedRef := restoreManagedRef(destinationID)
	if existing, err := e.journal.Get(request.OperationID); err == nil {
		return e.resultForReusedRestore(request, destinationID, existing)
	} else if !errors.Is(err, lifecycle.ErrNotFound) {
		return RestoreResult{}, err
	}

	reserved, createdReservation, err := e.reserveRestoreLifecycle(destinationID, managedRef, request.SnapshotID)
	if err != nil {
		return RestoreResult{}, err
	}
	operation := lifecycle.NewOperation(request.OperationID, lifecycle.OperationRestore, []string{destinationID}, e.now())
	operation.IdempotencyKey = restoreIdempotencyKey(request)
	operation.ApplicationVersion = "local-restore"
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.StagingRef = restoreStagingRef(request.OperationID)
	operation.SafeCancellationStage = string(RestoreStagePreflight)
	started, reused, err := e.coordinator.Begin(operation)
	if err != nil {
		if createdReservation {
			_, _ = e.records.RemoveRecord(destinationID)
		}
		return RestoreResult{}, err
	}
	if reused {
		return e.resultForReusedRestore(request, destinationID, started)
	}
	if reserved.Lock == nil {
		reserved, err = e.records.Get(destinationID)
		if err != nil {
			return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, domain.Profile{}, RestoreDependencyResolution{}, err)
		}
	}

	if _, err := e.setRestoreStage(request.OperationID, RestoreStagePreflight); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, domain.Profile{}, RestoreDependencyResolution{}, err)
	}
	source, err := e.verifyRestoreSource(ctx, request)
	if err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, domain.Profile{}, RestoreDependencyResolution{}, err)
	}
	resolution, kernelRef, proxyRefs := e.resolveDependencies(source.Manifest, request.Dependencies)
	if err := resolution.Validate(); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, domain.Profile{}, resolution, err)
	}
	if err := e.updateRestoreLifecycle(destinationID, source.Manifest.SnapshotID, resolution.Limitations); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, domain.Profile{}, resolution, err)
	}
	seed := restoreFingerprintSeed(request.OperationID, request.SnapshotID, source.ManifestDigest)
	finalProfilePath := filepath.Join(e.profilesRoot, destinationID)
	restoredProfile := buildRestoredProfile(source.SourceProfile, request, destinationID, seed, finalProfilePath, kernelRef, proxyRefs)

	if _, err := e.setRestoreStage(request.OperationID, RestoreStageStaging); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, "", false, false, false, restoredProfile, resolution, err)
	}
	stagePath, err := e.createRestoreStage(request.OperationID)
	if err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, restoreStagePath(e.recoveryRoot, request.OperationID), false, false, false, restoredProfile, resolution, err)
	}
	if err := writeRestoredProfileMetadata(stagePath, restoredProfile); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, err)
	}
	if _, err := e.setRestoreStage(request.OperationID, RestoreStageCopying); err != nil {
		return e.abortRestore(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, err)
	}
	counters, err := e.copyRestoreFiles(ctx, request, source, stagePath)
	if err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if _, err := e.setRestoreStage(request.OperationID, RestoreStageVerifying); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if err := verifyRestoredProfileMetadata(filepath.Join(stagePath, "restored-profile.json"), restoredProfile); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if err := verifyRestoredBrowserData(filepath.Join(stagePath, browserDataDirectory), source.Manifest); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}

	if _, err := e.setRestoreStage(request.OperationID, RestoreStageActivating); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	if _, err := os.Lstat(finalProfilePath); err == nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, profile.ErrNotFound)
	} else if !errors.Is(err, os.ErrNotExist) {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, err)
	}
	browserStagePath := filepath.Join(stagePath, browserDataDirectory)
	if err := e.rename(browserStagePath, finalProfilePath); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, false, false, restoredProfile, resolution, counters, fmt.Errorf("activate restored browser data: %w", err))
	}
	syncDirectory(e.profilesRoot)
	if _, err := e.setRestoreStage(request.OperationID, RestoreStageMetadata); err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, true, false, restoredProfile, resolution, counters, err)
	}
	createdProfile, err := e.profiles.Create(restoredProfile)
	if err != nil {
		return e.abortRestoreWithCounters(request, destinationID, managedRef, started, stagePath, true, true, false, restoredProfile, resolution, counters, err)
	}

	cleanupErr := e.removeStage(e.recoveryRoot, stagePath)
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         destinationID,
		Status:         lifecycle.ItemSucceeded,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: string(RestoreStageFinished),
		FilesProcessed: counters.files,
		BytesProcessed: counters.bytes,
		OutputID:       destinationID,
	}
	status := lifecycle.OperationCompleted
	recoveryActions := []string(nil)
	limitations := append([]string(nil), resolution.Limitations...)
	if cleanupErr != nil {
		status = lifecycle.OperationPartial
		recoveryActions = []string{"cleanup-restore-staging"}
		limitations = append(limitations, "restore-staging-cleanup-required")
		sort.Strings(limitations)
		limitations = uniqueStrings(limitations)
	}
	finished, finishErr := e.coordinator.Finish(request.OperationID, status, []lifecycle.OperationItemResult{item}, limitations, recoveryActions)
	result := RestoreResult{
		Operation:    finished,
		Profile:      createdProfile,
		Dependencies: resolution,
		ManagedRef:   managedRef,
	}
	result.Lifecycle, _ = e.records.Get(destinationID)
	if finishErr != nil {
		return result, fmt.Errorf("%w: restored Profile activated but operation finalization failed: %v", ErrRecoveryRequired, finishErr)
	}
	if cleanupErr != nil {
		return result, fmt.Errorf("%w: restored Profile activated but staging cleanup failed: %v", ErrRecoveryRequired, cleanupErr)
	}
	e.reportProgress(RestoreProgress{
		Stage:          RestoreStageFinished,
		FilesProcessed: counters.files,
		FilesTotal:     counters.files,
		BytesProcessed: counters.bytes,
		BytesTotal:     counters.bytes,
	})
	return result, nil
}

func (e *RestoreExecutor) reserveRestoreLifecycle(profileID, managedDir, snapshotID string) (lifecycle.Record, bool, error) {
	record, err := e.records.Get(profileID)
	if err == nil {
		if record.State != lifecycle.StateDraft || record.ManagedDir != managedDir || record.SourceID != snapshotID {
			return lifecycle.Record{}, false, lifecycle.ErrConflict
		}
		return record, false, nil
	}
	if !errors.Is(err, lifecycle.ErrNotFound) {
		return lifecycle.Record{}, false, err
	}
	created, err := e.records.Create(lifecycle.Record{
		ProfileID:       profileID,
		State:           lifecycle.StateDraft,
		ManagedDir:      managedDir,
		SourceID:         snapshotID,
		LimitationCodes: []string{"restore-in-progress", "restore-lifecycle-draft"},
	})
	return created, err == nil, err
}

func (e *RestoreExecutor) updateRestoreLifecycle(profileID, snapshotID string, limitations []string) error {
	for attempt := 0; attempt < 3; attempt++ {
		record, err := e.records.Get(profileID)
		if err != nil {
			return err
		}
		if record.State != lifecycle.StateDraft || record.SourceID != snapshotID {
			return lifecycle.ErrConflict
		}
		record.LimitationCodes = append([]string(nil), limitations...)
		sort.Strings(record.LimitationCodes)
		record.LimitationCodes = uniqueStrings(record.LimitationCodes)
		_, err = e.records.Update(record)
		if errors.Is(err, lifecycle.ErrConflict) {
			continue
		}
		return err
	}
	return lifecycle.ErrConflict
}

func writeRestoredProfileMetadata(stagePath string, restored domain.Profile) error {
	data, err := json.MarshalIndent(restored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode restored Profile metadata: %w", err)
	}
	if err := validateProfileDefinitionExclusions(data); err != nil {
		return err
	}
	return writeExclusiveFile(filepath.Join(stagePath, "restored-profile.json"), data)
}

func verifyRestoredProfileMetadata(filePath string, expected domain.Profile) error {
	data, err := readBoundedFile(filePath, MaxProfileDefinitionBytes)
	if err != nil {
		return err
	}
	if err := validateProfileDefinitionExclusions(data); err != nil {
		return err
	}
	var actual domain.Profile
	if err := json.Unmarshal(data, &actual); err != nil {
		return fmt.Errorf("decode restored Profile metadata: %w", err)
	}
	expectedData, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	actualData, err := json.Marshal(actual)
	if err != nil {
		return err
	}
	if string(actualData) != string(expectedData) {
		return fmt.Errorf("%w: restored Profile metadata changed in staging", ErrSourceChanged)
	}
	return nil
}

func (e *RestoreExecutor) setRestoreStage(operationID string, stage RestoreStage) (lifecycle.Operation, error) {
	for attempt := 0; attempt < 3; attempt++ {
		operation, err := e.journal.Get(operationID)
		if err != nil {
			return lifecycle.Operation{}, err
		}
		if operation.Status.Terminal() {
			return lifecycle.Operation{}, lifecycle.ErrConflict
		}
		operation.Stage = string(stage)
		operation.StagingRef = restoreStagingRef(operationID)
		operation.SafeCancellationStage = string(stage)
		updated, err := e.journal.Update(operation)
		if errors.Is(err, lifecycle.ErrConflict) {
			continue
		}
		return updated, err
	}
	return lifecycle.Operation{}, lifecycle.ErrConflict
}

func (e *RestoreExecutor) checkCancellation(ctx context.Context, operationID string) error {
	if err := checkRestoreContext(ctx); err != nil {
		return err
	}
	operation, err := e.journal.Get(operationID)
	if err != nil {
		return err
	}
	if operation.CancellationRequested {
		return ErrRestoreCancelled
	}
	return nil
}

func (e *RestoreExecutor) resultForReusedRestore(request RestoreRequest, destinationID string, operation lifecycle.Operation) (RestoreResult, error) {
	if operation.Type != lifecycle.OperationRestore || len(operation.ProfileIDs) != 1 || operation.ProfileIDs[0] != destinationID || operation.IdempotencyKey != restoreIdempotencyKey(request) {
		return RestoreResult{Operation: operation}, lifecycle.ErrConflict
	}
	if !operation.Status.Terminal() {
		return RestoreResult{Operation: operation}, fmt.Errorf("%w: restore operation is already running", lifecycle.ErrConflict)
	}
	if operation.Status != lifecycle.OperationCompleted && operation.Status != lifecycle.OperationPartial {
		return RestoreResult{Operation: operation}, fmt.Errorf("%w: reused restore operation ended with %s", ErrRecoveryRequired, operation.Status)
	}
	profileRecord, err := e.profiles.Get(destinationID)
	if err != nil {
		return RestoreResult{Operation: operation}, fmt.Errorf("%w: completed restore has no Profile metadata", ErrRecoveryRequired)
	}
	lifecycleRecord, err := e.records.Get(destinationID)
	if err != nil {
		return RestoreResult{Operation: operation, Profile: profileRecord}, fmt.Errorf("%w: completed restore has no lifecycle record", ErrRecoveryRequired)
	}
	return RestoreResult{
		Operation:  operation,
		Profile:    profileRecord,
		Lifecycle:  lifecycleRecord,
		ManagedRef: restoreManagedRef(destinationID),
	}, nil
}

func (e *RestoreExecutor) abortRestore(request RestoreRequest, destinationID, managedRef string, started lifecycle.Operation, ownedPath string, stageExists, finalExists, profileCreated bool, restoredProfile domain.Profile, resolution RestoreDependencyResolution, cause error) (RestoreResult, error) {
	return e.abortRestoreWithCounters(request, destinationID, managedRef, started, ownedPath, stageExists, finalExists, profileCreated, restoredProfile, resolution, snapshotCounters{}, cause)
}

func (e *RestoreExecutor) abortRestoreWithCounters(request RestoreRequest, destinationID, managedRef string, started lifecycle.Operation, ownedPath string, stageExists, finalExists, profileCreated bool, restoredProfile domain.Profile, resolution RestoreDependencyResolution, counters snapshotCounters, cause error) (RestoreResult, error) {
	recoveryRequired := errors.Is(cause, ErrRecoveryRequired)
	if profileCreated {
		if err := e.profiles.Delete(destinationID); err != nil {
			recoveryRequired = true
			cause = fmt.Errorf("%v; rollback Profile metadata: %w", cause, err)
		} else {
			profileCreated = false
		}
	}
	if finalExists {
		if err := e.removeProfile(filepath.Join(e.profilesRoot, destinationID)); err != nil {
			recoveryRequired = true
			cause = fmt.Errorf("%v; rollback restored browser data: %w", cause, err)
		} else {
			finalExists = false
		}
	}
	if stageExists {
		if err := e.removeStage(e.recoveryRoot, ownedPath); err != nil {
			recoveryRequired = true
			cause = fmt.Errorf("%v; cleanup restore staging: %w", cause, err)
		}
	}
	status := lifecycle.OperationFailed
	itemStatus := lifecycle.ItemFailed
	reason := restoreReasonCode(cause)
	recoveryActions := []string(nil)
	recoveryID := ""
	if errors.Is(cause, ErrRestoreCancelled) && !recoveryRequired {
		status = lifecycle.OperationCancelled
		itemStatus = lifecycle.ItemCancelled
	}
	if recoveryRequired {
		status = lifecycle.OperationRecoveryRequired
		itemStatus = lifecycle.ItemRecoveryRequired
		recoveryActions = []string{"inspect-local-restore-state"}
		if finalExists || profileCreated {
			recoveryID = managedRef
		} else {
			recoveryID = restoreStagingRef(request.OperationID)
		}
		reason = "restore-recovery-required"
	}
	completedAt := e.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         destinationID,
		Status:         itemStatus,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: string(RestoreStageFinished),
		FilesProcessed: counters.files,
		BytesProcessed: counters.bytes,
		ReasonCode:     reason,
		RecoveryID:     recoveryID,
	}
	finished, finishErr := e.coordinator.Finish(request.OperationID, status, []lifecycle.OperationItemResult{item}, resolution.Limitations, recoveryActions)
	result := RestoreResult{Operation: finished, Profile: restoredProfile, Dependencies: resolution, ManagedRef: managedRef}
	if finishErr != nil {
		return result, fmt.Errorf("%w: restore failure could not be finalized: %v; original error: %v", ErrRecoveryRequired, finishErr, cause)
	}
	if !recoveryRequired {
		if _, err := e.records.RemoveRecord(destinationID); err != nil {
			return result, fmt.Errorf("%w: restore rolled back but lifecycle reservation could not be removed: %v", ErrRecoveryRequired, err)
		}
	} else {
		e.markRestoreRecovery(destinationID, reason)
	}
	if recoveryRequired {
		return result, fmt.Errorf("%w: %v", ErrRecoveryRequired, cause)
	}
	if errors.Is(cause, ErrRestoreCancelled) {
		return result, ErrRestoreCancelled
	}
	return result, cause
}

func (e *RestoreExecutor) markRestoreRecovery(profileID, code string) {
	for attempt := 0; attempt < 3; attempt++ {
		record, err := e.records.Get(profileID)
		if err != nil {
			return
		}
		record.RecoveryCodes = append(record.RecoveryCodes, code)
		sort.Strings(record.RecoveryCodes)
		record.RecoveryCodes = uniqueStrings(record.RecoveryCodes)
		if _, err := e.records.Update(record); !errors.Is(err, lifecycle.ErrConflict) {
			return
		}
	}
}

func restoreReasonCode(err error) string {
	switch {
	case errors.Is(err, ErrRestoreCancelled):
		return "restore-cancelled"
	case errors.Is(err, ErrInsufficientSpace):
		return "restore-insufficient-space"
	case errors.Is(err, ErrSnapshotUnavailable):
		return "restore-snapshot-unavailable"
	case errors.Is(err, ErrSourceChanged):
		return "restore-source-changed"
	case errors.Is(err, context.DeadlineExceeded):
		return "restore-duration-exceeded"
	default:
		return "restore-failed"
	}
}
