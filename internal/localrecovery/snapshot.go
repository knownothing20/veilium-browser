package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (c *SnapshotCreator) Create(ctx context.Context, request SnapshotRequest) (SnapshotResult, error) {
	if c == nil || c.records == nil || c.coordinator == nil || c.journal == nil || c.catalog == nil {
		return SnapshotResult{}, fmt.Errorf("local snapshot creator is unavailable")
	}
	if err := request.Validate(); err != nil {
		return SnapshotResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.duration(request))
	defer cancel()

	stagingRef := snapshotStagingRef(request.OperationID)
	operation := lifecycle.NewOperation(request.OperationID, lifecycle.OperationSnapshot, []string{request.ProfileID}, c.now())
	operation.IdempotencyKey = snapshotIdempotencyKey(request)
	operation.ApplicationVersion = request.ApplicationVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.StagingRef = stagingRef
	operation.SafeCancellationStage = string(SnapshotStagePreflight)
	started, reused, err := c.coordinator.Begin(operation)
	if err != nil {
		return SnapshotResult{}, err
	}
	if reused {
		return c.resultForReusedOperation(request, started)
	}

	record, err := c.records.Get(request.ProfileID)
	if err != nil {
		return c.abortSnapshot(request, started, "", false, false, CatalogRecord{}, snapshotCounters{}, err)
	}
	if record.State != lifecycle.StateAvailable || record.Lock == nil || record.Lock.OperationID != request.OperationID {
		return c.abortSnapshot(request, started, "", false, false, CatalogRecord{}, snapshotCounters{}, fmt.Errorf("%w: Profile lifecycle state or lock is not valid for snapshot", ErrInvalidRecord))
	}
	if _, err := c.setOperationStage(request.OperationID, SnapshotStagePreflight, stagingRef); err != nil {
		return c.abortSnapshot(request, started, "", false, false, CatalogRecord{}, snapshotCounters{}, err)
	}

	plan, err := c.preflight(ctx, record, request)
	if err != nil {
		return c.abortSnapshot(request, started, "", false, false, CatalogRecord{}, snapshotCounters{}, err)
	}
	if _, err := c.setOperationStage(request.OperationID, SnapshotStageStaging, stagingRef); err != nil {
		return c.abortSnapshot(request, started, "", false, false, CatalogRecord{}, snapshotCounters{}, err)
	}
	stagePath, err := c.createStage(request.OperationID)
	if err != nil {
		return c.abortSnapshot(request, started, snapshotStagePath(c.recoveryRoot, request.OperationID), false, false, CatalogRecord{}, snapshotCounters{}, err)
	}

	if _, err := c.setOperationStage(request.OperationID, SnapshotStageCopying, stagingRef); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, snapshotCounters{}, err)
	}
	entries, counters, err := c.copySnapshotFiles(ctx, request, plan, stagePath)
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if err := c.checkCancellation(ctx, request.OperationID); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if _, err := c.setOperationStage(request.OperationID, SnapshotStageVerifying, stagingRef); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if err := c.verifySourcePlan(ctx, request.OperationID, plan); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}

	profileDigest, err := DigestProfileDefinition(request.ProfileDefinition)
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	treeDigest, err := ComputeTreeDigest(runtime.GOOS, entries)
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	manifest := LocalSnapshotManifest{
		SchemaVersion:              ManifestSchemaVersion,
		SnapshotID:                 request.SnapshotID,
		Scope:                      ScopeLocalFullSnapshot,
		SourceProfileID:            request.ProfileID,
		SourceProfileName:          request.ProfileName,
		SourceProfileSchemaVersion: request.ProfileSchemaVersion,
		SourceApplicationVersion:   request.ApplicationVersion,
		SourceOS:                   runtime.GOOS,
		SourceArch:                 runtime.GOARCH,
		CreatedAt:                  c.now().UTC(),
		ProfileDefinitionDigest:    profileDigest,
		IncludedRoots:              []string{"browser-data", "profile-definition"},
		TreeDigest:                 treeDigest,
		FileCount:                  int64(len(entries)),
		TotalBytes:                 counters.bytes,
		Files:                      entries,
		Dependencies:               request.Dependencies,
		ExcludedData: []string{
			"adapter-binaries",
			"browser-evidence",
			"credential-secrets",
			"kernel-binaries",
			"runtime-logs",
			"runtime-state",
		},
		Portability: PortabilitySameUserSameMachine,
	}
	if err := manifest.Validate(); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if _, err := WriteManifest(filepath.Join(stagePath, manifestFileName), manifest); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	verifiedManifest, err := verifyStagedSnapshot(stagePath, manifest)
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if err := c.checkCancellation(ctx, request.OperationID); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}

	manifestDigest, err := ComputeManifestDigest(verifiedManifest)
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	catalog, err := c.catalog.Create(CatalogRecord{
		SnapshotID:      request.SnapshotID,
		SourceProfileID: request.ProfileID,
		ManifestRef:     ExpectedManifestRef(request.SnapshotID),
		Status:          SnapshotStaged,
		ManifestDigest:  manifestDigest,
		TreeDigest:      verifiedManifest.TreeDigest,
		FileCount:       verifiedManifest.FileCount,
		TotalBytes:      verifiedManifest.TotalBytes,
	})
	if err != nil {
		return c.abortSnapshot(request, started, stagePath, true, false, CatalogRecord{}, counters, err)
	}
	if err := c.checkCancellation(ctx, request.OperationID); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, err)
	}
	if _, err := c.setOperationStage(request.OperationID, SnapshotStagePublishing, stagingRef); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, err)
	}

	finalParent := filepath.Join(c.recoveryRoot, snapshotFinalDirectory)
	if err := ensurePrivateDirectoryTree(finalParent); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, err)
	}
	finalPath := snapshotFinalPath(c.recoveryRoot, request.SnapshotID)
	if _, err := os.Lstat(finalPath); err == nil {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, ErrAlreadyExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, err)
	}
	if err := c.rename(stagePath, finalPath); err != nil {
		return c.abortSnapshot(request, started, stagePath, true, true, catalog, counters, fmt.Errorf("publish verified snapshot: %w", err))
	}
	syncDirectory(finalParent)
	verifiedAt := c.now().UTC()
	catalog.Status = SnapshotVerified
	catalog.VerifiedAt = &verifiedAt
	catalog, err = c.catalog.Update(catalog)
	if err != nil {
		return c.abortSnapshot(request, started, finalPath, false, true, catalog, counters, fmt.Errorf("%w: published snapshot catalog update failed: %v", ErrRecoveryRequired, err))
	}

	completedAt := c.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         lifecycle.ItemSucceeded,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: string(SnapshotStageFinished),
		FilesProcessed: counters.files,
		BytesProcessed: counters.bytes,
		OutputID:       request.SnapshotID,
	}
	finished, err := c.coordinator.Finish(request.OperationID, lifecycle.OperationCompleted, []lifecycle.OperationItemResult{item}, nil, nil)
	if err != nil {
		return SnapshotResult{
			Operation:    started,
			Catalog:      catalog,
			Manifest:     verifiedManifest,
			PublishedRef: snapshotPublishedRef(request.SnapshotID),
		}, fmt.Errorf("%w: snapshot published but operation finalization failed: %v", ErrRecoveryRequired, err)
	}
	c.reportProgress(SnapshotProgress{
		Stage:          SnapshotStageFinished,
		FilesProcessed: counters.files,
		FilesTotal:     counters.files,
		BytesProcessed: counters.bytes,
		BytesTotal:     counters.bytes,
	})
	return SnapshotResult{
		Operation:    finished,
		Catalog:      catalog,
		Manifest:     verifiedManifest,
		PublishedRef: snapshotPublishedRef(request.SnapshotID),
	}, nil
}

func snapshotIdempotencyKey(request SnapshotRequest) string {
	digest := sha256.Sum256([]byte(request.SnapshotID + "\x00" + request.IdempotencyKey))
	return "snapshot-" + hex.EncodeToString(digest[:])
}

func (c *SnapshotCreator) resultForReusedOperation(request SnapshotRequest, operation lifecycle.Operation) (SnapshotResult, error) {
	if !operation.Status.Terminal() {
		return SnapshotResult{Operation: operation}, fmt.Errorf("%w: snapshot operation is already running", lifecycle.ErrConflict)
	}
	if operation.Status != lifecycle.OperationCompleted {
		return SnapshotResult{Operation: operation}, fmt.Errorf("%w: reused snapshot operation ended with %s", ErrInvalidRecord, operation.Status)
	}
	matchedOutput := false
	for _, item := range operation.Items {
		if item.ItemID == request.ProfileID && item.OutputID == request.SnapshotID && item.Status == lifecycle.ItemSucceeded {
			matchedOutput = true
			break
		}
	}
	if !matchedOutput {
		return SnapshotResult{Operation: operation}, fmt.Errorf("%w: reused operation output does not match the requested snapshot", lifecycle.ErrConflict)
	}
	catalog, err := c.catalog.Get(request.SnapshotID)
	if err != nil {
		return SnapshotResult{Operation: operation}, fmt.Errorf("%w: completed operation has no catalog record", ErrRecoveryRequired)
	}
	if catalog.SourceProfileID != request.ProfileID || catalog.Status != SnapshotVerified {
		return SnapshotResult{Operation: operation, Catalog: catalog}, fmt.Errorf("%w: completed operation catalog is contradictory", ErrRecoveryRequired)
	}
	manifest, err := ReadManifest(filepath.Join(c.recoveryRoot, filepath.FromSlash(catalog.ManifestRef)))
	if err != nil {
		return SnapshotResult{Operation: operation, Catalog: catalog}, fmt.Errorf("%w: completed operation artifact cannot be read: %v", ErrRecoveryRequired, err)
	}
	return SnapshotResult{
		Operation:    operation,
		Catalog:      catalog,
		Manifest:     manifest,
		PublishedRef: snapshotPublishedRef(request.SnapshotID),
	}, nil
}

func (c *SnapshotCreator) setOperationStage(operationID string, stage SnapshotStage, stagingRef string) (lifecycle.Operation, error) {
	for attempt := 0; attempt < 3; attempt++ {
		operation, err := c.journal.Get(operationID)
		if err != nil {
			return lifecycle.Operation{}, err
		}
		if operation.Status.Terminal() {
			return lifecycle.Operation{}, fmt.Errorf("%w: snapshot operation became terminal during execution", lifecycle.ErrConflict)
		}
		operation.Stage = string(stage)
		operation.StagingRef = stagingRef
		operation.SafeCancellationStage = string(stage)
		updated, err := c.journal.Update(operation)
		if errors.Is(err, lifecycle.ErrConflict) {
			continue
		}
		return updated, err
	}
	return lifecycle.Operation{}, lifecycle.ErrConflict
}

func (c *SnapshotCreator) checkCancellation(ctx context.Context, operationID string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	operation, err := c.journal.Get(operationID)
	if err != nil {
		return err
	}
	if operation.CancellationRequested {
		return ErrSnapshotCancelled
	}
	return nil
}

func (c *SnapshotCreator) abortSnapshot(request SnapshotRequest, started lifecycle.Operation, ownedPath string, stageExists, catalogExists bool, catalog CatalogRecord, counters snapshotCounters, cause error) (SnapshotResult, error) {
	recoveryRequired := errors.Is(cause, ErrRecoveryRequired)
	finalPublished := ownedPath != "" && pathContainedBy(ownedPath, filepath.Join(c.recoveryRoot, snapshotFinalDirectory))
	if stageExists && !finalPublished {
		if cleanupErr := c.removeStage(c.recoveryRoot, ownedPath); cleanupErr != nil {
			recoveryRequired = true
			cause = fmt.Errorf("%v; staging cleanup failed: %w", cause, cleanupErr)
		}
	}
	if finalPublished {
		recoveryRequired = true
	}
	if catalogExists {
		catalog.VerifiedAt = nil
		if recoveryRequired {
			catalog.Status = SnapshotRecoveryRequired
		} else {
			catalog.Status = SnapshotInvalid
		}
		updated, updateErr := c.catalog.Update(catalog)
		if updateErr == nil {
			catalog = updated
		} else {
			recoveryRequired = true
			cause = fmt.Errorf("%v; catalog recovery update failed: %w", cause, updateErr)
		}
	}

	status := lifecycle.OperationFailed
	itemStatus := lifecycle.ItemFailed
	reason := snapshotReasonCode(cause)
	recoveryActions := []string(nil)
	recoveryID := ""
	if errors.Is(cause, ErrSnapshotCancelled) && !recoveryRequired {
		status = lifecycle.OperationCancelled
		itemStatus = lifecycle.ItemCancelled
	}
	if recoveryRequired {
		status = lifecycle.OperationRecoveryRequired
		itemStatus = lifecycle.ItemRecoveryRequired
		if finalPublished {
			recoveryActions = []string{"inspect-local-recovery-published"}
			recoveryID = snapshotPublishedRef(request.SnapshotID)
		} else {
			recoveryActions = []string{"inspect-local-recovery-staging"}
			recoveryID = snapshotStagingRef(request.OperationID)
		}
		reason = "snapshot-recovery-required"
	}
	sort.Strings(recoveryActions)
	completedAt := c.now().UTC()
	item := lifecycle.OperationItemResult{
		ItemID:         request.ProfileID,
		Status:         itemStatus,
		StartedAt:      &started.StartedAt,
		CompletedAt:    &completedAt,
		CompletedStage: string(SnapshotStageFinished),
		FilesProcessed: counters.files,
		BytesProcessed: counters.bytes,
		ReasonCode:     reason,
		RecoveryID:     recoveryID,
	}
	finished, finishErr := c.coordinator.Finish(request.OperationID, status, []lifecycle.OperationItemResult{item}, nil, recoveryActions)
	result := SnapshotResult{Operation: finished, Catalog: catalog}
	if finishErr != nil {
		return result, fmt.Errorf("%w: snapshot failure could not be finalized: %v; original error: %v", ErrRecoveryRequired, finishErr, cause)
	}
	if recoveryRequired {
		return result, fmt.Errorf("%w: %v", ErrRecoveryRequired, cause)
	}
	if errors.Is(cause, ErrSnapshotCancelled) {
		return result, ErrSnapshotCancelled
	}
	return result, cause
}

func snapshotReasonCode(err error) string {
	switch {
	case errors.Is(err, ErrSnapshotCancelled):
		return "snapshot-cancelled"
	case errors.Is(err, ErrInsufficientSpace):
		return "snapshot-insufficient-space"
	case errors.Is(err, ErrSourceChanged):
		return "snapshot-source-changed"
	case errors.Is(err, context.DeadlineExceeded):
		return "snapshot-duration-exceeded"
	default:
		return "snapshot-failed"
	}
}
