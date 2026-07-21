package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/localrecovery"
)

const localRecoveryRefreshTimeout = 10 * time.Second

type LocalRecoveryBootstrap struct {
	Snapshots           []localrecovery.CatalogRecord           `json:"snapshots"`
	Trash               []localrecovery.TrashRecord             `json:"trash"`
	Progress            []LocalRecoveryProgress                 `json:"progress"`
	TrashReconciliation localrecovery.TrashReconciliationReport `json:"trashReconciliation"`
}

type LocalRecoveryProgress struct {
	OperationID           string    `json:"operationId"`
	OperationType         string    `json:"operationType"`
	ProfileID             string    `json:"profileId"`
	Status                string    `json:"status"`
	Stage                 string    `json:"stage"`
	FilesProcessed        int64     `json:"filesProcessed"`
	FilesTotal            int64     `json:"filesTotal"`
	BytesProcessed        int64     `json:"bytesProcessed"`
	BytesTotal            int64     `json:"bytesTotal"`
	CancellationAvailable bool      `json:"cancellationAvailable"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type LocalRecoveryPreflight struct {
	ProfileID              string          `json:"profileId"`
	LifecycleState         lifecycle.State `json:"lifecycleState"`
	StorageStatus          string          `json:"storageStatus"`
	Active                 bool            `json:"active"`
	Locked                 bool            `json:"locked"`
	SnapshotAllowed        bool            `json:"snapshotAllowed"`
	ArchiveAllowed         bool            `json:"archiveAllowed"`
	UnarchiveAllowed       bool            `json:"unarchiveAllowed"`
	TrashAllowed           bool            `json:"trashAllowed"`
	RestoreTrashAllowed    bool            `json:"restoreTrashAllowed"`
	PermanentDeleteAllowed bool            `json:"permanentDeleteAllowed"`
	TrashID                string          `json:"trashId,omitempty"`
	RetentionDeadline      *time.Time      `json:"retentionDeadline,omitempty"`
	Reasons                []string        `json:"reasons,omitempty"`
}

type LocalSnapshotDetail struct {
	Record   localrecovery.CatalogRecord         `json:"record"`
	Manifest localrecovery.LocalSnapshotManifest `json:"manifest"`
}

type CreateLocalSnapshotRequest struct {
	ProfileID      string `json:"profileId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type RestoreLocalSnapshotRequest struct {
	SnapshotID     string `json:"snapshotId"`
	Name           string `json:"name,omitempty"`
	KernelID       string `json:"kernelId,omitempty"`
	AdapterID      string `json:"adapterId,omitempty"`
	CredentialID   string `json:"credentialId,omitempty"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type ArchiveProfileRequest struct {
	ProfileID      string `json:"profileId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type TrashProfileRequest struct {
	ProfileID      string `json:"profileId"`
	RetentionDays  int    `json:"retentionDays"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type TrashProfileActionRequest struct {
	ProfileID      string `json:"profileId"`
	TrashID        string `json:"trashId"`
	Confirmation   string `json:"confirmation,omitempty"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type localRecoveryService struct {
	mu             sync.RWMutex
	executionMu    sync.Mutex
	catalog        *localrecovery.CatalogStore
	trash          *localrecovery.TrashStore
	reconciler     *localrecovery.TrashReconciler
	reconciliation localrecovery.TrashReconciliationReport
	progress       map[string]LocalRecoveryProgress
}

var localRecoveryRegistry sync.Map
var localRecoveryRegistryMu sync.Mutex

func (s *Service) localRecoveryRuntime() (*localRecoveryService, error) {
	if s == nil {
		return nil, fmt.Errorf("desktop service is unavailable")
	}
	if value, ok := localRecoveryRegistry.Load(s); ok {
		return value.(*localRecoveryService), nil
	}
	localRecoveryRegistryMu.Lock()
	defer localRecoveryRegistryMu.Unlock()
	if value, ok := localRecoveryRegistry.Load(s); ok {
		return value.(*localRecoveryService), nil
	}
	catalog, err := localrecovery.OpenCatalogStore(filepath.Join(s.dataRoot, "local-recovery", "catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("open local recovery catalog: %w", err)
	}
	trash, err := localrecovery.OpenTrashStore(filepath.Join(s.dataRoot, "local-recovery", "trash-catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("open local recovery trash catalog: %w", err)
	}
	reconciler, err := localrecovery.OpenTrashReconciler(s.dataRoot, s.lifecycleRecords, s.lifecycleJournal, s.store, trash)
	if err != nil {
		return nil, fmt.Errorf("open local recovery reconciler: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), localRecoveryRefreshTimeout)
	report, reconcileErr := reconciler.Reconcile(ctx)
	cancel()
	if reconcileErr != nil {
		return nil, fmt.Errorf("reconcile local recovery trash: %w", reconcileErr)
	}
	runtime := &localRecoveryService{
		catalog: catalog, trash: trash, reconciler: reconciler,
		reconciliation: report, progress: make(map[string]LocalRecoveryProgress),
	}
	localRecoveryRegistry.Store(s, runtime)
	return runtime, nil
}

func (s *Service) LocalRecoveryState() (LocalRecoveryBootstrap, error) {
	runtime, err := s.localRecoveryRuntime()
	if err != nil {
		return LocalRecoveryBootstrap{}, err
	}
	runtime.mu.RLock()
	catalog := runtime.catalog
	trash := runtime.trash
	reconciliation := cloneTrashReconciliation(runtime.reconciliation)
	progress := make([]LocalRecoveryProgress, 0, len(runtime.progress))
	for _, item := range runtime.progress {
		progress = append(progress, item)
	}
	runtime.mu.RUnlock()

	snapshots := []localrecovery.CatalogRecord{}
	if catalog != nil {
		snapshots = catalog.List()
	}
	trashRecords := []localrecovery.TrashRecord{}
	if trash != nil {
		trashRecords = trash.List()
	}
	sort.Slice(progress, func(i, j int) bool {
		if progress[i].UpdatedAt.Equal(progress[j].UpdatedAt) {
			return progress[i].OperationID < progress[j].OperationID
		}
		return progress[i].UpdatedAt.After(progress[j].UpdatedAt)
	})
	return LocalRecoveryBootstrap{
		Snapshots: snapshots, Trash: trashRecords, Progress: progress,
		TrashReconciliation: reconciliation,
	}, nil
}

func (s *Service) ListLocalSnapshots() ([]localrecovery.CatalogRecord, error) {
	state, err := s.LocalRecoveryState()
	return state.Snapshots, err
}

func (s *Service) ListLocalTrash() ([]localrecovery.TrashRecord, error) {
	state, err := s.LocalRecoveryState()
	return state.Trash, err
}

func (s *Service) GetLocalSnapshot(snapshotID string) (LocalSnapshotDetail, error) {
	runtime, err := s.localRecoveryRuntime()
	if err != nil {
		return LocalSnapshotDetail{}, err
	}
	runtime.mu.RLock()
	catalog := runtime.catalog
	runtime.mu.RUnlock()
	record, err := catalog.Get(strings.TrimSpace(snapshotID))
	if err != nil {
		return LocalSnapshotDetail{}, err
	}
	manifest, err := localrecovery.ReadManifest(filepath.Join(s.dataRoot, "local-recovery", filepath.FromSlash(record.ManifestRef)))
	if err != nil {
		return LocalSnapshotDetail{Record: record}, err
	}
	return LocalSnapshotDetail{Record: record, Manifest: manifest}, nil
}

func (s *Service) LocalRecoveryPreflight(ctx context.Context, profileID string) (LocalRecoveryPreflight, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return LocalRecoveryPreflight{}, fmt.Errorf("profile id is required")
	}
	runtime, err := s.localRecoveryRuntime()
	if err != nil {
		return LocalRecoveryPreflight{}, err
	}
	record, err := s.lifecycleRecords.Get(profileID)
	if err != nil {
		return LocalRecoveryPreflight{}, err
	}
	result := LocalRecoveryPreflight{
		ProfileID: profileID, LifecycleState: record.State,
		Active: s.supervisor.IsActive(profileID), Locked: record.Lock != nil, StorageStatus: "unknown",
	}
	inventory, scanErr := s.ScanLifecycleStorage(ctx)
	if scanErr != nil {
		result.Reasons = append(result.Reasons, "storage-preflight-failed")
	} else {
		for _, item := range inventory.Profiles {
			if item.ProfileID != profileID {
				continue
			}
			result.StorageStatus = string(item.Status)
			if item.ReasonCode != "" {
				result.Reasons = append(result.Reasons, item.ReasonCode)
			}
			break
		}
	}
	var trashRecord *localrecovery.TrashRecord
	for _, item := range runtime.trash.List() {
		if item.ProfileID == profileID {
			copy := item
			trashRecord = &copy
			result.TrashID = item.TrashID
			deadline := item.RetentionDeadline.UTC()
			result.RetentionDeadline = &deadline
			break
		}
	}
	blocked := result.Active || result.Locked
	if result.Active {
		result.Reasons = append(result.Reasons, "browser-session-active")
	}
	if result.Locked {
		result.Reasons = append(result.Reasons, "lifecycle-operation-active")
	}
	present := result.StorageStatus == string(lifecycle.InventoryPresent)
	missing := result.StorageStatus == string(lifecycle.InventoryMissing)
	result.SnapshotAllowed = !blocked && record.State == lifecycle.StateAvailable && present
	result.ArchiveAllowed = !blocked && (record.State == lifecycle.StateAvailable || record.State == lifecycle.StateDraft)
	result.UnarchiveAllowed = !blocked && record.State == lifecycle.StateArchived
	result.TrashAllowed = !blocked && (record.State == lifecycle.StateAvailable || record.State == lifecycle.StateDraft || record.State == lifecycle.StateArchived) && (present || missing)
	result.RestoreTrashAllowed = !blocked && record.State == lifecycle.StateTrashed && trashRecord != nil && trashRecord.Status == localrecovery.TrashStored && trashRecord.DataPresent
	result.PermanentDeleteAllowed = result.RestoreTrashAllowed
	result.Reasons = sortedStrings(result.Reasons)
	return result, nil
}

func (s *Service) RefreshLocalRecovery(ctx context.Context) (LocalRecoveryBootstrap, error) {
	runtime, err := s.localRecoveryRuntime()
	if err != nil {
		return LocalRecoveryBootstrap{}, err
	}
	if err := reloadLocalSnapshotCatalog(s.dataRoot, runtime); err != nil {
		return LocalRecoveryBootstrap{}, err
	}
	report, err := runtime.reconciler.Reconcile(ctx)
	if err != nil {
		return LocalRecoveryBootstrap{}, err
	}
	runtime.mu.Lock()
	runtime.reconciliation = report
	runtime.mu.Unlock()
	return s.LocalRecoveryState()
}

func (s *Service) CreateLocalSnapshot(ctx context.Context, input CreateLocalSnapshotRequest) (localrecovery.SnapshotResult, error) {
	runtimeState, err := s.localRecoveryRuntime()
	if err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	if err := validateLocalRecoveryKey(input.IdempotencyKey); err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	item, err := s.store.Get(strings.TrimSpace(input.ProfileID))
	if err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	definition, err := snapshotProfileDefinition(item)
	if err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	dependencies, err := s.snapshotDependencyRequirements(item)
	if err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	operationID := localRecoveryID("snapshot-op", item.ID, input.IdempotencyKey)
	request := localrecovery.SnapshotRequest{
		OperationID: operationID, SnapshotID: localRecoveryID("snapshot", item.ID, input.IdempotencyKey),
		ProfileID: item.ID, IdempotencyKey: input.IdempotencyKey, ProfileName: item.Name,
		ProfileSchemaVersion: 1, ApplicationVersion: AppVersion, ProfileDefinition: definition, Dependencies: dependencies,
	}
	runtimeState.executionMu.Lock()
	defer runtimeState.executionMu.Unlock()
	creator, err := localrecovery.OpenSnapshotCreator(s.dataRoot, s.lifecycleRecords, s.lifecycleJournal, s.lifecycleCoordinator)
	if err != nil {
		return localrecovery.SnapshotResult{}, err
	}
	setRecoveryProgress(runtimeState, LocalRecoveryProgress{OperationID: operationID, OperationType: string(lifecycle.OperationSnapshot), ProfileID: item.ID, Status: string(lifecycle.OperationPending), Stage: "preflight", CancellationAvailable: true, UpdatedAt: time.Now().UTC()})
	creator.SetProgressCallback(func(progress localrecovery.SnapshotProgress) {
		setRecoveryProgress(runtimeState, LocalRecoveryProgress{
			OperationID: operationID, OperationType: string(lifecycle.OperationSnapshot), ProfileID: item.ID,
			Status: string(lifecycle.OperationRunning), Stage: string(progress.Stage),
			FilesProcessed: progress.FilesProcessed, FilesTotal: progress.FilesTotal,
			BytesProcessed: progress.BytesProcessed, BytesTotal: progress.BytesTotal,
			CancellationAvailable: progress.Stage != localrecovery.SnapshotStagePublishing && progress.Stage != localrecovery.SnapshotStageFinished,
			UpdatedAt:             time.Now().UTC(),
		})
	})
	result, createErr := creator.Create(ctx, request)
	_ = reloadLocalSnapshotCatalog(s.dataRoot, runtimeState)
	syncRecoveryProgress(runtimeState, s.lifecycleJournal, operationID)
	return result, createErr
}

func (s *Service) RestoreLocalSnapshot(ctx context.Context, input RestoreLocalSnapshotRequest) (localrecovery.RestoreResult, error) {
	runtimeState, err := s.localRecoveryRuntime()
	if err != nil {
		return localrecovery.RestoreResult{}, err
	}
	if err := validateLocalRecoveryKey(input.IdempotencyKey); err != nil {
		return localrecovery.RestoreResult{}, err
	}
	input.SnapshotID = strings.TrimSpace(input.SnapshotID)
	if input.SnapshotID == "" {
		return localrecovery.RestoreResult{}, fmt.Errorf("snapshot id is required")
	}
	operationID := localRecoveryID("restore-op", input.SnapshotID, input.IdempotencyKey)
	request := localrecovery.RestoreRequest{
		OperationID: operationID, SnapshotID: input.SnapshotID, IdempotencyKey: input.IdempotencyKey,
		ApplicationVersion: AppVersion, Name: strings.TrimSpace(input.Name),
		Dependencies: localrecovery.RestoreDependencySelection{
			KernelID: strings.TrimSpace(input.KernelID), AdapterID: strings.TrimSpace(input.AdapterID), CredentialID: strings.TrimSpace(input.CredentialID),
		},
	}
	runtimeState.executionMu.Lock()
	defer runtimeState.executionMu.Unlock()
	executor, err := localrecovery.OpenRestoreExecutor(s.dataRoot, s.store, s.lifecycleRecords, s.lifecycleJournal, s.lifecycleCoordinator, s.kernels, s.adapters, s.credentials)
	if err != nil {
		return localrecovery.RestoreResult{}, err
	}
	setRecoveryProgress(runtimeState, LocalRecoveryProgress{OperationID: operationID, OperationType: string(lifecycle.OperationRestore), Status: string(lifecycle.OperationPending), Stage: string(localrecovery.RestoreStagePreflight), CancellationAvailable: true, UpdatedAt: time.Now().UTC()})
	executor.SetProgressCallback(func(progress localrecovery.RestoreProgress) {
		setRecoveryProgress(runtimeState, LocalRecoveryProgress{
			OperationID: operationID, OperationType: string(lifecycle.OperationRestore), Status: string(lifecycle.OperationRunning),
			Stage: string(progress.Stage), FilesProcessed: progress.FilesProcessed, FilesTotal: progress.FilesTotal,
			BytesProcessed: progress.BytesProcessed, BytesTotal: progress.BytesTotal,
			CancellationAvailable: progress.Stage != localrecovery.RestoreStageActivating && progress.Stage != localrecovery.RestoreStageMetadata && progress.Stage != localrecovery.RestoreStageFinished,
			UpdatedAt:             time.Now().UTC(),
		})
	})
	result, restoreErr := executor.Restore(ctx, request)
	syncRecoveryProgress(runtimeState, s.lifecycleJournal, operationID)
	return result, restoreErr
}

func (s *Service) ArchiveProfile(ctx context.Context, input ArchiveProfileRequest) (localrecovery.ArchiveResult, error) {
	return s.executeArchive(ctx, lifecycle.OperationArchive, input)
}

func (s *Service) UnarchiveProfile(ctx context.Context, input ArchiveProfileRequest) (localrecovery.ArchiveResult, error) {
	return s.executeArchive(ctx, lifecycle.OperationUnarchive, input)
}

func (s *Service) executeArchive(ctx context.Context, operationType lifecycle.OperationType, input ArchiveProfileRequest) (localrecovery.ArchiveResult, error) {
	runtimeState, err := s.localRecoveryRuntime()
	if err != nil {
		return localrecovery.ArchiveResult{}, err
	}
	if err := validateLocalRecoveryKey(input.IdempotencyKey); err != nil {
		return localrecovery.ArchiveResult{}, err
	}
	profileID := strings.TrimSpace(input.ProfileID)
	operationID := localRecoveryID(string(operationType)+"-op", profileID, input.IdempotencyKey)
	request := localrecovery.ArchiveRequest{OperationID: operationID, ProfileID: profileID, IdempotencyKey: input.IdempotencyKey, ApplicationVersion: AppVersion}
	runtimeState.executionMu.Lock()
	defer runtimeState.executionMu.Unlock()
	executor, err := localrecovery.OpenArchiveExecutor(s.dataRoot, s.lifecycleRecords, s.lifecycleJournal, s.lifecycleCoordinator)
	if err != nil {
		return localrecovery.ArchiveResult{}, err
	}
	setRecoveryProgress(runtimeState, LocalRecoveryProgress{OperationID: operationID, OperationType: string(operationType), ProfileID: profileID, Status: string(lifecycle.OperationPending), Stage: "lifecycle-preflight", CancellationAvailable: true, UpdatedAt: time.Now().UTC()})
	var result localrecovery.ArchiveResult
	if operationType == lifecycle.OperationArchive {
		result, err = executor.Archive(ctx, request)
	} else {
		result, err = executor.Unarchive(ctx, request)
	}
	syncRecoveryProgress(runtimeState, s.lifecycleJournal, operationID)
	return result, err
}

func (s *Service) TrashProfile(ctx context.Context, input TrashProfileRequest) (localrecovery.TrashResult, error) {
	runtimeState, err := s.localRecoveryRuntime()
	if err != nil {
		return localrecovery.TrashResult{}, err
	}
	if err := validateLocalRecoveryKey(input.IdempotencyKey); err != nil {
		return localrecovery.TrashResult{}, err
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if err := s.prepareEmptyManagedDirectoryForTrash(profileID); err != nil {
		return localrecovery.TrashResult{}, err
	}
	operationID := localRecoveryID("trash-op", profileID, input.IdempotencyKey)
	request := localrecovery.TrashRequest{OperationID: operationID, ProfileID: profileID, IdempotencyKey: input.IdempotencyKey, ApplicationVersion: AppVersion, RetentionDays: input.RetentionDays}
	runtimeState.executionMu.Lock()
	defer runtimeState.executionMu.Unlock()
	executor, err := localrecovery.OpenTrashExecutor(s.dataRoot, s.lifecycleRecords, s.lifecycleJournal, s.lifecycleCoordinator, s.store, runtimeState.trash)
	if err != nil {
		return localrecovery.TrashResult{}, err
	}
	setRecoveryProgress(runtimeState, LocalRecoveryProgress{OperationID: operationID, OperationType: string(lifecycle.OperationTrash), ProfileID: profileID, Status: string(lifecycle.OperationPending), Stage: "trash-preflight", CancellationAvailable: true, UpdatedAt: time.Now().UTC()})
	result, trashErr := executor.Trash(ctx, request)
	refreshTrashReconciliation(runtimeState)
	syncRecoveryProgress(runtimeState, s.lifecycleJournal, operationID)
	return result, trashErr
}

func (s *Service) RestoreTrashedProfile(ctx context.Context, input TrashProfileActionRequest) (localrecovery.TrashResult, error) {
	return s.executeTrashAction(ctx, lifecycle.OperationRestoreTrash, input)
}

func (s *Service) PermanentlyDeleteTrashedProfile(ctx context.Context, input TrashProfileActionRequest) (localrecovery.TrashResult, error) {
	return s.executeTrashAction(ctx, lifecycle.OperationPermanentDelete, input)
}

func (s *Service) executeTrashAction(ctx context.Context, operationType lifecycle.OperationType, input TrashProfileActionRequest) (localrecovery.TrashResult, error) {
	runtimeState, err := s.localRecoveryRuntime()
	if err != nil {
		return localrecovery.TrashResult{}, err
	}
	if err := validateLocalRecoveryKey(input.IdempotencyKey); err != nil {
		return localrecovery.TrashResult{}, err
	}
	profileID := strings.TrimSpace(input.ProfileID)
	trashID := strings.TrimSpace(input.TrashID)
	operationID := localRecoveryID(string(operationType)+"-op", profileID, trashID, input.IdempotencyKey)
	request := localrecovery.TrashActionRequest{OperationID: operationID, ProfileID: profileID, TrashID: trashID, IdempotencyKey: input.IdempotencyKey, ApplicationVersion: AppVersion, Confirmation: input.Confirmation}
	runtimeState.executionMu.Lock()
	defer runtimeState.executionMu.Unlock()
	executor, err := localrecovery.OpenTrashExecutor(s.dataRoot, s.lifecycleRecords, s.lifecycleJournal, s.lifecycleCoordinator, s.store, runtimeState.trash)
	if err != nil {
		return localrecovery.TrashResult{}, err
	}
	setRecoveryProgress(runtimeState, LocalRecoveryProgress{OperationID: operationID, OperationType: string(operationType), ProfileID: profileID, Status: string(lifecycle.OperationPending), Stage: string(operationType) + "-preflight", CancellationAvailable: operationType == lifecycle.OperationRestoreTrash, UpdatedAt: time.Now().UTC()})
	var result localrecovery.TrashResult
	if operationType == lifecycle.OperationRestoreTrash {
		result, err = executor.RestoreTrash(ctx, request)
	} else {
		result, err = executor.PermanentDelete(ctx, request)
	}
	refreshTrashReconciliation(runtimeState)
	syncRecoveryProgress(runtimeState, s.lifecycleJournal, operationID)
	return result, err
}

func (s *Service) CancelLocalRecoveryOperation(operationID string) (lifecycle.Operation, error) {
	if s.lifecycleCoordinator == nil {
		return lifecycle.Operation{}, fmt.Errorf("lifecycle coordinator is unavailable")
	}
	operation, changed, err := s.lifecycleCoordinator.RequestCancellation(strings.TrimSpace(operationID))
	if err != nil {
		return lifecycle.Operation{}, err
	}
	if !changed && operation.Status.Terminal() {
		return operation, fmt.Errorf("operation %q is already terminal", operation.ID)
	}
	if runtimeState, runtimeErr := s.localRecoveryRuntime(); runtimeErr == nil {
		syncRecoveryProgress(runtimeState, s.lifecycleJournal, operation.ID)
	}
	return operation, nil
}

func (s *Service) prepareEmptyManagedDirectoryForTrash(profileID string) error {
	if s.supervisor.IsActive(profileID) {
		return fmt.Errorf("profile cannot be moved to trash while its browser is running")
	}
	record, err := s.lifecycleRecords.Get(profileID)
	if err != nil {
		return err
	}
	if record.Lock != nil {
		return fmt.Errorf("profile cannot be moved to trash while lifecycle operation %q is active", record.Lock.OperationID)
	}
	switch record.State {
	case lifecycle.StateAvailable, lifecycle.StateDraft, lifecycle.StateArchived:
	default:
		return fmt.Errorf("profile cannot be moved to trash while lifecycle state is %q", record.State)
	}
	if record.ManagedDir != filepath.ToSlash(filepath.Join("profiles", profileID)) {
		return fmt.Errorf("profile managed directory is not Profile-owned")
	}
	expected := filepath.Join(s.profilesDir, profileID)
	if info, statErr := os.Lstat(expected); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("profile managed directory is unsafe")
		}
		return nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	return prepareManagedProfileDir(s.profilesDir, expected)
}

func (s *Service) snapshotDependencyRequirements(item domain.Profile) (localrecovery.DependencyRequirements, error) {
	providerID := strings.TrimSpace(item.Kernel.Provider)
	browserVersion := strings.TrimSpace(item.Kernel.Version)
	if providerID == "" || browserVersion == "" {
		return localrecovery.DependencyRequirements{}, fmt.Errorf("profile requires a kernel provider and browser version before snapshot")
	}
	kernelRequirement := localrecovery.KernelRequirement{
		ProviderID: providerID, BrowserVersion: browserVersion, OperatingSystem: runtime.GOOS,
		Architecture: runtime.GOARCH, TrustRequirement: "custom",
	}
	if strings.TrimSpace(item.Kernel.ID) != "" {
		record, err := s.kernels.Verify(item.Kernel.ID)
		if err != nil {
			return localrecovery.DependencyRequirements{}, err
		}
		identity, err := kernel.BinaryIdentity(record)
		if err != nil {
			return localrecovery.DependencyRequirements{}, err
		}
		trust := string(identity.ProviderTrust)
		if trust != "reviewed" && trust != "custom" && trust != "legacy" {
			return localrecovery.DependencyRequirements{}, fmt.Errorf("kernel trust state %q cannot be represented by local recovery", trust)
		}
		kernelRequirement.ProviderID = identity.ProviderID
		kernelRequirement.ProviderRevision = identity.ProviderRevision
		kernelRequirement.BrowserVersion = identity.BrowserVersion
		kernelRequirement.TrustRequirement = trust
		kernelRequirement.ExecutableSHA256 = identity.ExecutableSHA256
		kernelRequirement.PackageTreeSHA256 = identity.PackageTreeSHA256
		kernelRequirement.Limitations = sortedStrings(identity.Limitations)
	} else {
		capabilities, err := fingerprint.For(providerID, browserVersion)
		if err != nil {
			return localrecovery.DependencyRequirements{}, err
		}
		trust := string(capabilities.TrustStatus)
		if trust != "reviewed" && trust != "custom" && trust != "legacy" {
			return localrecovery.DependencyRequirements{}, fmt.Errorf("kernel trust state %q cannot be represented by local recovery", trust)
		}
		kernelRequirement.ProviderRevision = capabilities.Revision
		kernelRequirement.TrustRequirement = trust
		kernelRequirement.Limitations = sortedStrings(capabilities.Limitations)
	}
	result := localrecovery.DependencyRequirements{Kernel: kernelRequirement}
	if adapterID := strings.TrimSpace(item.Proxy.AdapterRef); adapterID != "" {
		record, err := s.adapters.Verify(adapterID)
		if err != nil {
			return localrecovery.DependencyRequirements{}, err
		}
		parsed, err := url.Parse(item.Proxy.URL)
		if err != nil || strings.TrimSpace(parsed.Scheme) == "" {
			return localrecovery.DependencyRequirements{}, fmt.Errorf("profile proxy scheme is required for adapter recovery")
		}
		result.Adapter = &localrecovery.AdapterRequirement{
			Kind: record.Kind, Version: record.Version, Official: record.Official,
			ExecutableSHA256: record.SHA256, Scheme: strings.ToLower(parsed.Scheme),
			OperatingSystem: runtime.GOOS, Architecture: runtime.GOARCH,
		}
	}
	if credentialID := strings.TrimSpace(item.Proxy.CredentialRef); credentialID != "" {
		record, err := s.credentials.Get(credentialID)
		if err != nil {
			return localrecovery.DependencyRequirements{}, err
		}
		parsed, err := url.Parse(item.Proxy.URL)
		if err != nil || strings.TrimSpace(parsed.Scheme) == "" {
			return localrecovery.DependencyRequirements{}, fmt.Errorf("profile proxy authentication scheme is required for credential recovery")
		}
		result.Credential = &localrecovery.CredentialRequirement{
			PlaceholderID: "proxy-credential", Authentication: strings.ToLower(parsed.Scheme), Label: record.Name,
			RequiresUsername: strings.TrimSpace(record.Username) != "", RequiresSecret: true,
		}
	}
	return result, result.Validate()
}

func snapshotProfileDefinition(item domain.Profile) ([]byte, error) {
	copy := item
	copy.UserDataDir = ""
	copy.Kernel.ID = ""
	copy.Kernel.Executable = ""
	copy.Proxy.AdapterRef = ""
	copy.Proxy.CredentialRef = ""
	copy.Fingerprint.Seed = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return nil, fmt.Errorf("encode local snapshot Profile definition: %w", err)
	}
	return data, nil
}

func reloadLocalSnapshotCatalog(dataRoot string, runtimeState *localRecoveryService) error {
	catalog, err := localrecovery.OpenCatalogStore(filepath.Join(dataRoot, "local-recovery", "catalog.json"))
	if err != nil {
		return fmt.Errorf("reload local recovery catalog: %w", err)
	}
	runtimeState.mu.Lock()
	runtimeState.catalog = catalog
	runtimeState.mu.Unlock()
	return nil
}

func refreshTrashReconciliation(runtimeState *localRecoveryService) {
	ctx, cancel := context.WithTimeout(context.Background(), localRecoveryRefreshTimeout)
	report, err := runtimeState.reconciler.Reconcile(ctx)
	cancel()
	if err != nil {
		return
	}
	runtimeState.mu.Lock()
	runtimeState.reconciliation = report
	runtimeState.mu.Unlock()
}

func setRecoveryProgress(runtimeState *localRecoveryService, progress LocalRecoveryProgress) {
	progress.UpdatedAt = progress.UpdatedAt.UTC()
	runtimeState.mu.Lock()
	runtimeState.progress[progress.OperationID] = progress
	runtimeState.mu.Unlock()
}

func syncRecoveryProgress(runtimeState *localRecoveryService, journal *lifecycle.Journal, operationID string) {
	if journal == nil {
		return
	}
	operation, err := journal.Get(operationID)
	if err != nil {
		return
	}
	runtimeState.mu.Lock()
	progress := runtimeState.progress[operationID]
	progress.OperationID = operation.ID
	progress.OperationType = string(operation.Type)
	if progress.ProfileID == "" && len(operation.ProfileIDs) > 0 {
		progress.ProfileID = operation.ProfileIDs[0]
	}
	progress.Status = string(operation.Status)
	progress.Stage = operation.Stage
	progress.CancellationAvailable = !operation.Status.Terminal() && !operation.CancellationRequested && operation.SafeCancellationStage != ""
	progress.UpdatedAt = operation.UpdatedAt.UTC()
	if len(operation.Items) > 0 {
		progress.FilesProcessed = operation.Items[0].FilesProcessed
		progress.BytesProcessed = operation.Items[0].BytesProcessed
	}
	runtimeState.progress[operationID] = progress
	runtimeState.mu.Unlock()
}

func validateLocalRecoveryKey(value string) error {
	if value == "" || strings.TrimSpace(value) != value || len(value) > localrecovery.MaxIdentifierLength {
		return fmt.Errorf("local recovery idempotency key is required and must be bounded")
	}
	return nil
}

func localRecoveryID(prefix string, values ...string) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(prefix))
	for _, value := range values {
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(value))
	}
	return prefix + "-" + hex.EncodeToString(hasher.Sum(nil)[:16])
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	unique := result[:0]
	for _, value := range result {
		if value == "" || (len(unique) > 0 && unique[len(unique)-1] == value) {
			continue
		}
		unique = append(unique, value)
	}
	return unique
}

func cloneTrashReconciliation(input localrecovery.TrashReconciliationReport) localrecovery.TrashReconciliationReport {
	result := input
	result.Findings = append([]localrecovery.TrashReconciliationFinding(nil), input.Findings...)
	result.Limitations = append([]string(nil), input.Limitations...)
	return result
}
