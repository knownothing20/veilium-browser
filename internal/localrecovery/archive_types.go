package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

var (
	ErrLifecycleStorageCancelled        = errors.New("local lifecycle storage operation cancelled")
	ErrLifecycleStorageRecoveryRequired = errors.New("local lifecycle storage operation requires recovery")
)

type ArchiveRequest struct {
	OperationID        string
	ProfileID          string
	IdempotencyKey     string
	ApplicationVersion string
}

func (r ArchiveRequest) Validate() error {
	for label, value := range map[string]string{
		"operation id": r.OperationID,
		"Profile id":   r.ProfileID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidRecord); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) != r.IdempotencyKey || len(r.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid lifecycle storage idempotency key", ErrInvalidRecord)
	}
	return validateText("application version", r.ApplicationVersion, true, ErrInvalidRecord)
}

type ArchiveResult struct {
	Operation lifecycle.Operation
	Record    lifecycle.Record
}

type archiveRecordStore interface {
	Get(string) (lifecycle.Record, error)
	Update(lifecycle.Record) (lifecycle.Record, error)
	AddRecoveryCode(string, string) (lifecycle.Record, bool, error)
}

type archiveJournal interface {
	Get(string) (lifecycle.Operation, error)
	Update(lifecycle.Operation) (lifecycle.Operation, error)
}

type archiveCoordinator interface {
	Begin(lifecycle.Operation) (lifecycle.Operation, bool, error)
	Finish(string, lifecycle.OperationStatus, []lifecycle.OperationItemResult, []string, []string) (lifecycle.Operation, error)
}

type ArchiveExecutor struct {
	dataRoot    string
	records     archiveRecordStore
	journal     archiveJournal
	coordinator archiveCoordinator
	now         func() time.Time
}

func OpenArchiveExecutor(dataRoot string, records *lifecycle.RecordStore, journal *lifecycle.Journal, coordinator *lifecycle.Coordinator) (*ArchiveExecutor, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return nil, fmt.Errorf("archive data root is required")
	}
	if records == nil || journal == nil || coordinator == nil {
		return nil, fmt.Errorf("archive operations require lifecycle records, journal, and coordinator")
	}
	absolute, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve archive data root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if err := inspectRealDirectory(absolute); err != nil {
		return nil, err
	}
	return &ArchiveExecutor{
		dataRoot:    absolute,
		records:     records,
		journal:     journal,
		coordinator: coordinator,
		now:         func() time.Time { return time.Now().UTC() },
	}, nil
}

func archiveIdempotencyKey(operationType lifecycle.OperationType, request ArchiveRequest) string {
	if request.IdempotencyKey == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(string(operationType) + "\x00" + request.ProfileID + "\x00" + request.IdempotencyKey))
	return "lifecycle-" + hex.EncodeToString(digest[:])
}

func newArchiveOperation(operationType lifecycle.OperationType, request ArchiveRequest, now time.Time) lifecycle.Operation {
	operation := lifecycle.NewOperation(request.OperationID, operationType, []string{request.ProfileID}, now)
	operation.IdempotencyKey = archiveIdempotencyKey(operationType, request)
	operation.ApplicationVersion = request.ApplicationVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.SafeCancellationStage = "lifecycle-preflight"
	return operation
}

func checkArchiveContext(ctx context.Context) error {
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

func archiveOriginCode(state lifecycle.State) (string, error) {
	switch state {
	case lifecycle.StateAvailable:
		return "archive-origin-available", nil
	case lifecycle.StateDraft:
		return "archive-origin-draft", nil
	default:
		return "", fmt.Errorf("%w: lifecycle state %q cannot be archived", lifecycle.ErrConflict, state)
	}
}

func archivedOrigin(record lifecycle.Record) (lifecycle.State, string, error) {
	hasAvailable := hasLifecycleCode(record.LimitationCodes, "archive-origin-available")
	hasDraft := hasLifecycleCode(record.LimitationCodes, "archive-origin-draft")
	if hasAvailable == hasDraft {
		return "", "", fmt.Errorf("%w: archived Profile origin is missing or contradictory", ErrLifecycleStorageRecoveryRequired)
	}
	if hasDraft {
		return lifecycle.StateDraft, "archive-origin-draft", nil
	}
	return lifecycle.StateAvailable, "archive-origin-available", nil
}

func hasLifecycleCode(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func addLifecycleCodes(values []string, additions ...string) []string {
	result := append([]string(nil), values...)
	result = append(result, additions...)
	sort.Strings(result)
	return uniqueStrings(result)
}

func removeLifecycleCodes(values []string, removals ...string) []string {
	removed := make(map[string]struct{}, len(removals))
	for _, value := range removals {
		removed[value] = struct{}{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, drop := removed[value]; !drop {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return uniqueStrings(result)
}
