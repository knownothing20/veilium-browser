package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

const (
	DefaultSnapshotDuration = 30 * time.Minute
	MaxSnapshotDuration     = 2 * time.Hour
	SnapshotCopyBufferBytes = 1 << 20
	SnapshotSpaceReserve    = int64(64 << 20)
)

var (
	ErrSnapshotCancelled = errors.New("local snapshot cancelled")
	ErrInsufficientSpace = errors.New("insufficient space for local snapshot")
	ErrSourceChanged     = errors.New("snapshot source changed during operation")
	ErrRecoveryRequired  = errors.New("local snapshot requires recovery")
)

type SnapshotStage string

const (
	SnapshotStagePreflight  SnapshotStage = "preflight"
	SnapshotStageStaging    SnapshotStage = "staging"
	SnapshotStageCopying    SnapshotStage = "copying"
	SnapshotStageVerifying  SnapshotStage = "verifying"
	SnapshotStagePublishing SnapshotStage = "publishing"
	SnapshotStageFinished   SnapshotStage = "finished"
)

type SnapshotRequest struct {
	OperationID          string
	SnapshotID           string
	ProfileID            string
	IdempotencyKey       string
	ProfileName          string
	ProfileSchemaVersion int
	ApplicationVersion   string
	ProfileDefinition    []byte
	Dependencies         DependencyRequirements
	MaxDuration          time.Duration
}

func (r SnapshotRequest) Validate() error {
	for label, value := range map[string]string{
		"operation id": r.OperationID,
		"snapshot id":  r.SnapshotID,
		"profile id":   r.ProfileID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidManifest); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) != r.IdempotencyKey || len(r.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid idempotency key", ErrInvalidManifest)
	}
	if err := validateText("Profile name", r.ProfileName, true, ErrInvalidManifest); err != nil {
		return err
	}
	if r.ProfileSchemaVersion <= 0 {
		return fmt.Errorf("%w: Profile schema version must be positive", ErrInvalidManifest)
	}
	if err := validateText("application version", r.ApplicationVersion, true, ErrInvalidManifest); err != nil {
		return err
	}
	if _, err := DigestProfileDefinition(r.ProfileDefinition); err != nil {
		return err
	}
	if err := validateProfileDefinitionExclusions(r.ProfileDefinition); err != nil {
		return err
	}
	if err := r.Dependencies.Validate(); err != nil {
		return err
	}
	if r.Dependencies.Kernel.OperatingSystem != runtime.GOOS || r.Dependencies.Kernel.Architecture != runtime.GOARCH {
		return fmt.Errorf("%w: kernel requirement does not match the current machine", ErrInvalidManifest)
	}
	if adapter := r.Dependencies.Adapter; adapter != nil {
		if adapter.OperatingSystem != runtime.GOOS || adapter.Architecture != runtime.GOARCH {
			return fmt.Errorf("%w: adapter requirement does not match the current machine", ErrInvalidManifest)
		}
	}
	if r.MaxDuration < 0 || r.MaxDuration > MaxSnapshotDuration {
		return fmt.Errorf("%w: snapshot duration is outside bounds", ErrInvalidManifest)
	}
	return nil
}

type SnapshotProgress struct {
	Stage          SnapshotStage
	FilesProcessed int64
	FilesTotal     int64
	BytesProcessed int64
	BytesTotal     int64
}

type SnapshotResult struct {
	Operation    lifecycle.Operation
	Catalog      CatalogRecord
	Manifest     LocalSnapshotManifest
	PublishedRef string
}

type SnapshotProgressFunc func(SnapshotProgress)

type lifecycleCoordinator interface {
	Begin(lifecycle.Operation) (lifecycle.Operation, bool, error)
	Finish(string, lifecycle.OperationStatus, []lifecycle.OperationItemResult, []string, []string) (lifecycle.Operation, error)
}

type lifecycleJournal interface {
	Get(string) (lifecycle.Operation, error)
	Update(lifecycle.Operation) (lifecycle.Operation, error)
}

type SnapshotCreator struct {
	dataRoot     string
	recoveryRoot string
	records      *lifecycle.RecordStore
	coordinator  lifecycleCoordinator
	journal      lifecycleJournal
	catalog      *CatalogStore
	now          func() time.Time
	rename       func(string, string) error
	removeStage  func(string, string) error
	space        func(string) (uint64, error)
	progress     SnapshotProgressFunc
}

func OpenSnapshotCreator(dataRoot string, records *lifecycle.RecordStore, journal *lifecycle.Journal, coordinator *lifecycle.Coordinator) (*SnapshotCreator, error) {
	if records == nil || journal == nil || coordinator == nil {
		return nil, fmt.Errorf("local snapshot requires lifecycle records, journal, and coordinator")
	}
	root, recoveryRoot, err := prepareRecoveryRoots(dataRoot)
	if err != nil {
		return nil, err
	}
	catalog, err := OpenCatalogStore(filepath.Join(recoveryRoot, "catalog.json"))
	if err != nil {
		return nil, err
	}
	creator := &SnapshotCreator{
		dataRoot:     root,
		recoveryRoot: recoveryRoot,
		records:      records,
		coordinator:  coordinator,
		journal:      journal,
		catalog:      catalog,
		now:          func() time.Time { return time.Now().UTC() },
		rename:       renamePath,
		space:        availableBytes,
	}
	creator.removeStage = creator.removeOwnedStage
	return creator, nil
}

func (c *SnapshotCreator) SetProgressCallback(callback SnapshotProgressFunc) {
	if c != nil {
		c.progress = callback
	}
}

func (c *SnapshotCreator) duration(request SnapshotRequest) time.Duration {
	if request.MaxDuration > 0 {
		return request.MaxDuration
	}
	return DefaultSnapshotDuration
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrSnapshotCancelled
		}
		return ctx.Err()
	default:
		return nil
	}
}
