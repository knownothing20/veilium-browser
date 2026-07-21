package localrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

var ErrLifecycleStorageRecoveryRequired = errors.New("local lifecycle storage operation requires recovery")

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

type ArchiveExecutor struct {
	dataRoot    string
	records     *lifecycle.RecordStore
	journal     *lifecycle.Journal
	coordinator *lifecycle.Coordinator
	now         func() time.Time
}

func OpenArchiveExecutor(dataRoot string, records *lifecycle.RecordStore, journal *lifecycle.Journal, coordinator *lifecycle.Coordinator) (*ArchiveExecutor, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return nil, fmt.Errorf("archive data root is required")
	}
	if records == nil || journal == nil || coordinator == nil {
		return nil, fmt.Errorf("archive operations require lifecycle records, journal, and coordinator")
	}
	return &ArchiveExecutor{
		dataRoot:    dataRoot,
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

func archivedOrigin(record lifecycle.Record) (lifecycle.State, error) {
	hasAvailable := containsString(record.LimitationCodes, "archive-origin-available")
	hasDraft := containsString(record.LimitationCodes, "archive-origin-draft")
	if hasAvailable == hasDraft {
		return "", fmt.Errorf("%w: archived Profile origin is missing or contradictory", ErrLifecycleStorageRecoveryRequired)
	}
	if hasDraft {
		return lifecycle.StateDraft, nil
	}
	return lifecycle.StateAvailable, nil
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
