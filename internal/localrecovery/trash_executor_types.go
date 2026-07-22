package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type trashRecordStore interface {
	Get(string) (lifecycle.Record, error)
	Update(lifecycle.Record) (lifecycle.Record, error)
	AddRecoveryCode(string, string) (lifecycle.Record, bool, error)
}

type trashJournal interface {
	Get(string) (lifecycle.Operation, error)
	Update(lifecycle.Operation) (lifecycle.Operation, error)
	List() []lifecycle.Operation
}

type trashCoordinator interface {
	Begin(lifecycle.Operation) (lifecycle.Operation, bool, error)
	Finish(string, lifecycle.OperationStatus, []lifecycle.OperationItemResult, []string, []string) (lifecycle.Operation, error)
}

type trashCatalog interface {
	List() []TrashRecord
	Get(string) (TrashRecord, error)
	Create(TrashRecord) (TrashRecord, error)
	Update(TrashRecord) (TrashRecord, error)
	Remove(string, uint64) (TrashRecord, error)
}

type trashProfileStore interface {
	Get(string) (domain.Profile, error)
	Delete(string) error
}

type TrashExecutor struct {
	dataRoot     string
	recoveryRoot string
	records      trashRecordStore
	journal      trashJournal
	coordinator  trashCoordinator
	profiles     trashProfileStore
	trash        trashCatalog
	now          func() time.Time
	rename       func(string, string) error
	removeTree   func(string, string) error
}

func OpenTrashExecutor(dataRoot string, records *lifecycle.RecordStore, journal *lifecycle.Journal, coordinator *lifecycle.Coordinator, profiles *profile.Store, trash *TrashStore) (*TrashExecutor, error) {
	if records == nil || journal == nil || coordinator == nil || profiles == nil || trash == nil {
		return nil, fmt.Errorf("trash operations require lifecycle, Profile, journal, coordinator, and trash stores")
	}
	absolute, recoveryRoot, err := prepareRecoveryRoots(dataRoot)
	if err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectoryTree(filepath.Join(recoveryRoot, trashFinalDirectory)); err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectoryTree(filepath.Join(recoveryRoot, trashStagingDirectory)); err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectoryTree(filepath.Join(recoveryRoot, deleteStagingDirectory)); err != nil {
		return nil, err
	}
	return &TrashExecutor{
		dataRoot:     absolute,
		recoveryRoot: recoveryRoot,
		records:      records,
		journal:      journal,
		coordinator:  coordinator,
		profiles:     profiles,
		trash:        trash,
		now:          func() time.Time { return time.Now().UTC() },
		rename:       renamePath,
		removeTree:   removeOwnedTrashTree,
	}, nil
}

const (
	trashFinalDirectory    = "trash"
	trashStagingDirectory  = ".trash-staging"
	deleteStagingDirectory = ".delete-staging"
)

func trashIDFor(request TrashRequest) string {
	key := request.IdempotencyKey
	if key == "" {
		key = request.OperationID
	}
	digest := sha256.Sum256([]byte(request.ProfileID + "\x00" + key))
	return "trash-" + hex.EncodeToString(digest[:16])
}

func trashIdempotencyKey(operationType lifecycle.OperationType, profileID, trashID, key string) string {
	if key == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(string(operationType) + "\x00" + profileID + "\x00" + trashID + "\x00" + key))
	return "trash-op-" + hex.EncodeToString(digest[:])
}

func newTrashOperation(operationType lifecycle.OperationType, operationID, profileID, trashID, idempotencyKey, applicationVersion string, now time.Time) lifecycle.Operation {
	operation := lifecycle.NewOperation(operationID, operationType, []string{profileID}, now)
	operation.IdempotencyKey = trashIdempotencyKey(operationType, profileID, trashID, idempotencyKey)
	operation.ApplicationVersion = applicationVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.SafeCancellationStage = string(operationType) + "-preflight"
	return operation
}

func trashRootPath(recoveryRoot, trashID string) string {
	return filepath.Join(recoveryRoot, trashFinalDirectory, trashID)
}

func trashStagePath(recoveryRoot, operationID string) string {
	return filepath.Join(recoveryRoot, trashStagingDirectory, operationID)
}

func deleteStagePath(recoveryRoot, operationID string) string {
	return filepath.Join(recoveryRoot, deleteStagingDirectory, operationID)
}

func trashStageRef(operationID string) string {
	return path.Join("local-recovery", trashStagingDirectory, operationID)
}

func deleteStageRef(operationID string) string {
	return path.Join("local-recovery", deleteStagingDirectory, operationID)
}

func createPrivateOperationStage(root, operationID string) (string, error) {
	if err := ensurePrivateDirectoryTree(root); err != nil {
		return "", err
	}
	stage := filepath.Join(root, operationID)
	if err := os.Mkdir(stage, 0o700); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%w: operation staging already exists", ErrLifecycleStorageRecoveryRequired)
		}
		return "", err
	}
	return stage, nil
}

func profileDefinitionForTrash(item domain.Profile) ([]byte, string, error) {
	data, err := json.Marshal(item)
	if err != nil {
		return nil, "", fmt.Errorf("encode Profile definition: %w", err)
	}
	if len(data) == 0 || len(data) > MaxProfileDefinitionBytes {
		return nil, "", fmt.Errorf("%w: Profile definition size is outside bounds", ErrInvalidManifest)
	}
	if err := validateProfileDefinitionExclusions(data); err != nil {
		return nil, "", err
	}
	digest, err := DigestProfileDefinition(data)
	if err != nil {
		return nil, "", err
	}
	return data, digest, nil
}
func (e *TrashExecutor) verifyRetainedProfile(profileID, expectedDigest string) error {
	item, err := e.profiles.Get(profileID)
	if err != nil {
		return fmt.Errorf("%w: retained Profile metadata is unavailable: %v", ErrLifecycleStorageRecoveryRequired, err)
	}
	_, digest, err := profileDefinitionForTrash(item)
	if err != nil {
		return fmt.Errorf("%w: retained Profile metadata cannot be verified: %v", ErrLifecycleStorageRecoveryRequired, err)
	}
	if digest != expectedDigest {
		return fmt.Errorf("%w: retained Profile metadata changed while in lifecycle storage", ErrLifecycleStorageRecoveryRequired)
	}
	return nil
}

func checkTrashContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrLifecycleStorageCancelled
		}
		return ctx.Err()
	default:
		return nil
	}
}

func (e *TrashExecutor) checkCancellation(ctx context.Context, operationID string) error {
	if err := checkTrashContext(ctx); err != nil {
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

func (e *TrashExecutor) setStage(operationID, stage, stagingRef, quarantineRef string) (lifecycle.Operation, error) {
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
		operation.StagingRef = stagingRef
		operation.QuarantineRef = quarantineRef
		updated, err := e.journal.Update(operation)
		if errors.Is(err, lifecycle.ErrConflict) {
			continue
		}
		return updated, err
	}
	return lifecycle.Operation{}, lifecycle.ErrConflict
}

func validateTrashLifecycleSource(record lifecycle.Record) error {
	if record.Lock == nil {
		return lifecycle.ErrConflict
	}
	if record.ManagedDir != path.Join("profiles", record.ProfileID) {
		return fmt.Errorf("%w: lifecycle managed directory is not Profile-owned", lifecycle.ErrConflict)
	}
	if record.TrashedAt != nil || record.RetentionDeadline != nil {
		return fmt.Errorf("%w: lifecycle record already contains trash timestamps", ErrLifecycleStorageRecoveryRequired)
	}
	switch record.State {
	case lifecycle.StateAvailable, lifecycle.StateDraft:
		if record.ArchivedAt != nil {
			return fmt.Errorf("%w: non-archived Profile has archived timestamp", ErrLifecycleStorageRecoveryRequired)
		}
	case lifecycle.StateArchived:
		if record.ArchivedAt == nil {
			return fmt.Errorf("%w: archived Profile is missing its timestamp", ErrLifecycleStorageRecoveryRequired)
		}
	default:
		return fmt.Errorf("%w: lifecycle state %q cannot be trashed", lifecycle.ErrConflict, record.State)
	}
	return nil
}

func trashCurrentLimitations(original []string, state lifecycle.State, trashID string) []string {
	base := removeLifecycleCodes(original, "profile-archived", "archive-origin-available", "archive-origin-draft")
	return addTrashCodes(base, state, trashID)
}

func removeTrashCodes(values []string, trashID string) []string {
	return removeLifecycleCodes(values,
		"profile-trashed",
		"trash-origin-available",
		"trash-origin-draft",
		"trash-origin-archived",
		"trash-id-"+trashID,
	)
}

func (e *TrashExecutor) markRecovery(profileID, code string) {
	_, _, _ = e.records.AddRecoveryCode(profileID, code)
}

func trashReasonCode(err error) string {
	switch {
	case errors.Is(err, ErrLifecycleStorageCancelled), errors.Is(err, context.Canceled):
		return "trash-cancelled"
	case errors.Is(err, ErrLifecycleStorageRecoveryRequired):
		return "trash-recovery-required"
	case errors.Is(err, ErrSourceChanged), errors.Is(err, ErrInvalidManifest):
		return "trash-verification-failed"
	case errors.Is(err, lifecycle.ErrConflict), errors.Is(err, ErrConflict):
		return "trash-conflict"
	case errors.Is(err, context.DeadlineExceeded):
		return "trash-deadline-exceeded"
	default:
		return "trash-failed"
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func validateOperationIdentity(operation lifecycle.Operation, operationType lifecycle.OperationType, operationID, profileID, trashID, idempotencyKey string) error {
	if operation.ID != operationID || operation.Type != operationType || len(operation.ProfileIDs) != 1 || operation.ProfileIDs[0] != profileID {
		return lifecycle.ErrConflict
	}
	if operation.IdempotencyKey != trashIdempotencyKey(operationType, profileID, trashID, idempotencyKey) {
		return lifecycle.ErrConflict
	}
	return nil
}

func ensureNoExistingPath(candidate string) error {
	_, err := os.Lstat(candidate)
	if err == nil {
		return fmt.Errorf("%w: destination already exists", lifecycle.ErrConflict)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ensureOwnedStage(stage, boundary string) error {
	if !pathContainedBy(stage, boundary) || filepath.Clean(stage) == filepath.Clean(boundary) {
		return fmt.Errorf("%w: operation staging is outside its owned boundary", ErrLifecycleStorageRecoveryRequired)
	}
	return nil
}

func profileAlreadyDeleted(err error) bool {
	return errors.Is(err, profile.ErrNotFound)
}

func prepareEmptyStagePath(root, operationID string) (string, error) {
	if err := ensurePrivateDirectoryTree(root); err != nil {
		return "", err
	}
	stage := filepath.Join(root, operationID)
	if err := ensureNoExistingPath(stage); err != nil {
		return "", err
	}
	return stage, nil
}
