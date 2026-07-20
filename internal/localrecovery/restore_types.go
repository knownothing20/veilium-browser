package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

const (
	DefaultRestoreDuration = 30 * time.Minute
	MaxRestoreDuration     = 2 * time.Hour
)

var (
	ErrRestoreCancelled    = errors.New("local restore cancelled")
	ErrDependencyMismatch  = errors.New("local restore dependency mismatch")
	ErrSnapshotUnavailable = errors.New("verified local snapshot is unavailable")
)

type RestoreStage string

const (
	RestoreStagePreflight  RestoreStage = "restore-preflight"
	RestoreStageStaging    RestoreStage = "restore-staging"
	RestoreStageCopying    RestoreStage = "restore-copying"
	RestoreStageVerifying  RestoreStage = "restore-verifying"
	RestoreStageActivating RestoreStage = "restore-activating"
	RestoreStageMetadata   RestoreStage = "restore-metadata"
	RestoreStageFinished   RestoreStage = "restore-finished"
)

type DependencyResolutionStatus string

const (
	DependencyResolved           DependencyResolutionStatus = "resolved"
	DependencyMissing            DependencyResolutionStatus = "missing"
	DependencyIncompatible       DependencyResolutionStatus = "incompatible"
	DependencyUserActionRequired DependencyResolutionStatus = "user-action-required"
	DependencyUnsupported        DependencyResolutionStatus = "unsupported"
)

type RestoreDependencySelection struct {
	KernelID     string
	AdapterID    string
	CredentialID string
}

type ResolvedDependency struct {
	Kind       string
	Status     DependencyResolutionStatus
	RecordID   string
	ReasonCode string
}

type RestoreDependencyResolution struct {
	Kernel      ResolvedDependency
	Adapter     *ResolvedDependency
	Credential  *ResolvedDependency
	Limitations []string
}

func (r RestoreDependencyResolution) FullyResolved() bool {
	if r.Kernel.Status != DependencyResolved {
		return false
	}
	if r.Adapter != nil && r.Adapter.Status != DependencyResolved {
		return false
	}
	if r.Credential != nil && r.Credential.Status != DependencyResolved {
		return false
	}
	return true
}

type RestoreRequest struct {
	OperationID        string
	SnapshotID         string
	IdempotencyKey     string
	ApplicationVersion string
	Name               string
	Dependencies       RestoreDependencySelection
	MaxDuration        time.Duration
}

func (r RestoreRequest) Validate() error {
	for label, value := range map[string]string{
		"operation id": r.OperationID,
		"snapshot id":  r.SnapshotID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidManifest); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) != r.IdempotencyKey || len(r.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid restore idempotency key", ErrInvalidManifest)
	}
	if err := validateText("application version", r.ApplicationVersion, true, ErrInvalidManifest); err != nil {
		return err
	}
	if err := validateText("restored Profile name", r.Name, false, ErrInvalidManifest); err != nil {
		return err
	}
	for label, value := range map[string]string{
		"selected Kernel id":     r.Dependencies.KernelID,
		"selected adapter id":    r.Dependencies.AdapterID,
		"selected credential id": r.Dependencies.CredentialID,
	} {
		if value != "" {
			if err := validateIdentifier(label, value, ErrInvalidManifest); err != nil {
				return err
			}
		}
	}
	if r.MaxDuration < 0 || r.MaxDuration > MaxRestoreDuration {
		return fmt.Errorf("%w: restore duration is outside bounds", ErrInvalidManifest)
	}
	return nil
}

type RestoreProgress struct {
	Stage          RestoreStage
	FilesProcessed int64
	FilesTotal     int64
	BytesProcessed int64
	BytesTotal     int64
}

type RestoreResult struct {
	Operation    lifecycle.Operation
	Profile      domain.Profile
	Lifecycle    lifecycle.Record
	Dependencies RestoreDependencyResolution
	ManagedRef   string
}

type RestoreProgressFunc func(RestoreProgress)

type restoreProfileStore interface {
	Get(string) (domain.Profile, error)
	Create(domain.Profile) (domain.Profile, error)
	Delete(string) error
}

type restoreKernelStore interface {
	Verify(string) (kernel.Record, error)
}

type restoreAdapterStore interface {
	Verify(string) (adapter.Record, error)
}

type restoreCredentialStore interface {
	Get(string) (credential.Record, error)
}

type RestoreExecutor struct {
	dataRoot      string
	recoveryRoot  string
	profilesRoot  string
	profiles      restoreProfileStore
	records       *lifecycle.RecordStore
	journal       lifecycleJournal
	coordinator   lifecycleCoordinator
	catalog       *CatalogStore
	kernels       restoreKernelStore
	adapters      restoreAdapterStore
	credentials   restoreCredentialStore
	now           func() time.Time
	rename        func(string, string) error
	removeStage   func(string, string) error
	removeProfile func(string) error
	space         func(string) (uint64, error)
	progress      RestoreProgressFunc
}

func OpenRestoreExecutor(dataRoot string, profiles *profile.Store, records *lifecycle.RecordStore, journal *lifecycle.Journal, coordinator *lifecycle.Coordinator, kernels *kernel.Store, adapters *adapter.Store, credentials *credential.Manager) (*RestoreExecutor, error) {
	if profiles == nil || records == nil || journal == nil || coordinator == nil || kernels == nil || adapters == nil || credentials == nil {
		return nil, fmt.Errorf("local restore requires Profile, lifecycle, dependency, and credential stores")
	}
	root, recoveryRoot, err := prepareRecoveryRoots(dataRoot)
	if err != nil {
		return nil, err
	}
	profilesRoot := filepath.Join(root, "profiles")
	if err := ensurePrivateDirectoryTree(profilesRoot); err != nil {
		return nil, err
	}
	catalog, err := OpenCatalogStore(filepath.Join(recoveryRoot, "catalog.json"))
	if err != nil {
		return nil, err
	}
	executor := &RestoreExecutor{
		dataRoot:     root,
		recoveryRoot: recoveryRoot,
		profilesRoot: profilesRoot,
		profiles:     profiles,
		records:      records,
		journal:      journal,
		coordinator:  coordinator,
		catalog:      catalog,
		kernels:      kernels,
		adapters:     adapters,
		credentials:  credentials,
		now:          func() time.Time { return time.Now().UTC() },
		rename:       renamePath,
		space:        availableBytes,
	}
	executor.removeStage = executor.removeOwnedRestoreStage
	executor.removeProfile = executor.removeOwnedRestoredProfile
	return executor, nil
}

func (e *RestoreExecutor) SetProgressCallback(callback RestoreProgressFunc) {
	if e != nil {
		e.progress = callback
	}
}

func (e *RestoreExecutor) duration(request RestoreRequest) time.Duration {
	if request.MaxDuration > 0 {
		return request.MaxDuration
	}
	return DefaultRestoreDuration
}

func restoreDestinationID(idempotencyKey, snapshotID string) string {
	digest := sha256.Sum256([]byte("restore-profile\x00" + idempotencyKey + "\x00" + snapshotID))
	return hex.EncodeToString(digest[:16])
}

func restoreFingerprintSeed(idempotencyKey, snapshotID, manifestDigest string) string {
	digest := sha256.Sum256([]byte("restore-fingerprint\x00" + idempotencyKey + "\x00" + snapshotID + "\x00" + manifestDigest))
	return hex.EncodeToString(digest[:])
}

func restoreIdempotencyKey(request RestoreRequest) string {
	digest := sha256.Sum256([]byte(request.SnapshotID + "\x00" + request.IdempotencyKey))
	return "restore-" + hex.EncodeToString(digest[:])
}

func (e *RestoreExecutor) reportProgress(progress RestoreProgress) {
	if e.progress != nil {
		e.progress(progress)
	}
}

func checkRestoreContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrRestoreCancelled
		}
		return ctx.Err()
	default:
		return nil
	}
}
