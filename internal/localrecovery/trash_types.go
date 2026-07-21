package localrecovery

import (
	"fmt"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

const (
	TrashSchemaVersion   = 1
	MaxTrashRecords      = 4096
	MaxTrashCatalogSize  = 8 << 20
	DefaultRetentionDays = 30
	MaxRetentionDays     = 365
)

type TrashStatus string

const (
	TrashPending          TrashStatus = "pending"
	TrashStored           TrashStatus = "stored"
	TrashRestoring        TrashStatus = "restoring"
	TrashCleanupPending   TrashStatus = "cleanup-pending"
	TrashDeleted          TrashStatus = "deleted"
	TrashRecoveryRequired TrashStatus = "recovery-required"
)

func (s TrashStatus) Valid() bool {
	switch s {
	case TrashPending, TrashStored, TrashRestoring, TrashCleanupPending, TrashDeleted, TrashRecoveryRequired:
		return true
	default:
		return false
	}
}

type TrashRecord struct {
	SchemaVersion           int             `json:"schemaVersion"`
	TrashID                 string          `json:"trashId"`
	ProfileID               string          `json:"profileId"`
	OperatingSystem         string          `json:"operatingSystem"`
	Architecture            string          `json:"architecture"`
	OriginalState           lifecycle.State `json:"originalState"`
	OriginalManagedDir      string          `json:"originalManagedDir"`
	OriginalArchivedAt      *time.Time      `json:"originalArchivedAt,omitempty"`
	OriginalSourceID        string          `json:"originalSourceId,omitempty"`
	OriginalRecoveryCodes   []string        `json:"originalRecoveryCodes,omitempty"`
	OriginalLimitationCodes []string        `json:"originalLimitationCodes,omitempty"`
	TrashRef                string          `json:"trashRef"`
	DataPresent             bool            `json:"dataPresent"`
	ProfileDefinitionDigest string          `json:"profileDefinitionDigest"`
	TreeDigest              string          `json:"treeDigest"`
	FileCount               int64           `json:"fileCount"`
	TotalBytes              int64           `json:"totalBytes"`
	Status                  TrashStatus     `json:"status"`
	TrashedAt               time.Time       `json:"trashedAt"`
	RetentionDeadline       time.Time       `json:"retentionDeadline"`
	DeletedAt               *time.Time      `json:"deletedAt,omitempty"`
	UpdatedAt               time.Time       `json:"updatedAt"`
	Limitations             []string        `json:"limitations,omitempty"`
	Revision                uint64          `json:"revision"`
}

func (r TrashRecord) Validate() error {
	if r.SchemaVersion != TrashSchemaVersion {
		return fmt.Errorf("%w: trash record version %d", ErrUnsupportedVersion, r.SchemaVersion)
	}
	for label, value := range map[string]string{
		"trash id":   r.TrashID,
		"Profile id": r.ProfileID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidRecord); err != nil {
			return err
		}
	}
	if !validOS(r.OperatingSystem) || !validArch(r.Architecture) {
		return fmt.Errorf("%w: unsupported trash platform %q/%q", ErrInvalidRecord, r.OperatingSystem, r.Architecture)
	}
	if r.OperatingSystem != runtime.GOOS || r.Architecture != runtime.GOARCH {
		return fmt.Errorf("%w: trash record does not apply to this machine", ErrInvalidRecord)
	}
	switch r.OriginalState {
	case lifecycle.StateAvailable, lifecycle.StateDraft, lifecycle.StateArchived:
	default:
		return fmt.Errorf("%w: unsupported original lifecycle state %q", ErrInvalidRecord, r.OriginalState)
	}
	expectedOriginal := path.Join("profiles", r.ProfileID)
	if r.OriginalManagedDir != expectedOriginal {
		return fmt.Errorf("%w: original managed directory is not Profile-owned", ErrInvalidRecord)
	}
	if r.OriginalState == lifecycle.StateArchived {
		if r.OriginalArchivedAt == nil {
			return fmt.Errorf("%w: archived trash origin requires archived timestamp", ErrInvalidRecord)
		}
	} else if r.OriginalArchivedAt != nil {
		return fmt.Errorf("%w: non-archived trash origin has archived timestamp", ErrInvalidRecord)
	}
	if r.OriginalArchivedAt != nil {
		if _, offset := r.OriginalArchivedAt.Zone(); offset != 0 {
			return fmt.Errorf("%w: original archived timestamp must use UTC", ErrInvalidRecord)
		}
	}
	if err := validateText("original source id", r.OriginalSourceID, false, ErrInvalidRecord); err != nil {
		return err
	}
	if err := validateCodes("original recovery codes", r.OriginalRecoveryCodes, MaxCodes, ErrInvalidRecord); err != nil {
		return err
	}
	if err := validateCodes("original limitation codes", r.OriginalLimitationCodes, MaxCodes, ErrInvalidRecord); err != nil {
		return err
	}
	expectedTrashRef := path.Join("local-recovery", "trash", r.TrashID, browserDataDirectory)
	if r.TrashRef != expectedTrashRef {
		return fmt.Errorf("%w: trash reference is not canonical", ErrInvalidRecord)
	}
	if err := validateDigest("Profile definition digest", r.ProfileDefinitionDigest, ErrInvalidRecord); err != nil {
		return err
	}
	if err := validateDigest("trash tree digest", r.TreeDigest, ErrInvalidRecord); err != nil {
		return err
	}
	if r.FileCount < 0 || r.FileCount > MaxFiles || r.TotalBytes < 0 || r.TotalBytes > MaxTotalBytes {
		return fmt.Errorf("%w: trash file summary is outside bounds", ErrInvalidRecord)
	}
	if !r.DataPresent && r.Status != TrashDeleted && r.Status != TrashRecoveryRequired && (r.FileCount != 0 || r.TotalBytes != 0) {
		return fmt.Errorf("%w: metadata-only trash record contains file totals", ErrInvalidRecord)
	}
	if !r.Status.Valid() {
		return fmt.Errorf("%w: unsupported trash status %q", ErrInvalidRecord, r.Status)
	}
	if r.TrashedAt.IsZero() || r.RetentionDeadline.IsZero() || r.RetentionDeadline.Before(r.TrashedAt) {
		return fmt.Errorf("%w: invalid trash retention timestamps", ErrInvalidRecord)
	}
	if _, offset := r.TrashedAt.Zone(); offset != 0 {
		return fmt.Errorf("%w: trash timestamp must use UTC", ErrInvalidRecord)
	}
	if _, offset := r.RetentionDeadline.Zone(); offset != 0 {
		return fmt.Errorf("%w: retention deadline must use UTC", ErrInvalidRecord)
	}
	if r.DeletedAt != nil {
		if _, offset := r.DeletedAt.Zone(); offset != 0 {
			return fmt.Errorf("%w: deletion timestamp must use UTC", ErrInvalidRecord)
		}
	}
	if r.Status == TrashDeleted {
		if r.DeletedAt == nil || r.DataPresent {
			return fmt.Errorf("%w: deleted trash record requires a deletion timestamp and no live data", ErrInvalidRecord)
		}
	} else if r.DeletedAt != nil && r.Status != TrashRecoveryRequired {
		return fmt.Errorf("%w: non-deleted trash record has a deletion timestamp", ErrInvalidRecord)
	}
	if r.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: trash update timestamp is required", ErrInvalidRecord)
	}
	if _, offset := r.UpdatedAt.Zone(); offset != 0 {
		return fmt.Errorf("%w: trash update timestamp must use UTC", ErrInvalidRecord)
	}
	if err := validateCodes("trash limitations", r.Limitations, MaxCodes, ErrInvalidRecord); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: trash revision must be positive", ErrInvalidRecord)
	}
	return nil
}

type TrashRequest struct {
	OperationID        string
	ProfileID          string
	IdempotencyKey     string
	ApplicationVersion string
	RetentionDays      int
}

func (r TrashRequest) Validate() error {
	for label, value := range map[string]string{
		"operation id": r.OperationID,
		"Profile id":   r.ProfileID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidRecord); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) != r.IdempotencyKey || len(r.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid trash idempotency key", ErrInvalidRecord)
	}
	if err := validateText("application version", r.ApplicationVersion, true, ErrInvalidRecord); err != nil {
		return err
	}
	if r.RetentionDays < 0 || r.RetentionDays > MaxRetentionDays {
		return fmt.Errorf("%w: retention days must be between 0 and %d", ErrInvalidRecord, MaxRetentionDays)
	}
	return nil
}

func (r TrashRequest) retentionDuration() time.Duration {
	days := r.RetentionDays
	if days == 0 {
		days = DefaultRetentionDays
	}
	return time.Duration(days) * 24 * time.Hour
}

type TrashActionRequest struct {
	OperationID        string
	ProfileID          string
	TrashID            string
	IdempotencyKey     string
	ApplicationVersion string
	Confirmation       string
}

func (r TrashActionRequest) Validate(requireConfirmation bool) error {
	for label, value := range map[string]string{
		"operation id": r.OperationID,
		"Profile id":   r.ProfileID,
		"trash id":     r.TrashID,
	} {
		if err := validateIdentifier(label, value, ErrInvalidRecord); err != nil {
			return err
		}
	}
	if strings.TrimSpace(r.IdempotencyKey) != r.IdempotencyKey || len(r.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid trash action idempotency key", ErrInvalidRecord)
	}
	if err := validateText("application version", r.ApplicationVersion, true, ErrInvalidRecord); err != nil {
		return err
	}
	if requireConfirmation && r.Confirmation != r.ProfileID {
		return fmt.Errorf("%w: permanent cleanup confirmation must equal the Profile id", ErrInvalidRecord)
	}
	return nil
}

type TrashResult struct {
	Operation lifecycle.Operation
	Record    lifecycle.Record
	Trash     TrashRecord
}

func trashOriginCode(state lifecycle.State) string {
	return "trash-origin-" + string(state)
}

func addTrashCodes(values []string, state lifecycle.State, trashID string) []string {
	result := append([]string(nil), values...)
	result = append(result, "profile-trashed", trashOriginCode(state), "trash-id-"+trashID)
	sort.Strings(result)
	return uniqueStrings(result)
}
