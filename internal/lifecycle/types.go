package lifecycle

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	LifecycleSchemaVersion = 1
	OperationSchemaVersion = 1

	MaxRecords          = 4096
	MaxOperations       = 4096
	MaxProfilesPerOp    = 256
	MaxItemsPerOp       = 256
	MaxCodesPerRecord   = 32
	MaxTextLength       = 512
	MaxIdentifierLength = 256
	MaxStoreBytes       = 8 << 20
)

var (
	ErrNotFound           = errors.New("lifecycle record not found")
	ErrAlreadyExists      = errors.New("lifecycle record already exists")
	ErrConflict           = errors.New("lifecycle revision or idempotency conflict")
	ErrUnsupportedVersion = errors.New("unsupported lifecycle schema version")
	ErrInvalidRecord      = errors.New("invalid lifecycle record")
)

type State string

const (
	StateAvailable State = "available"
	StateDraft     State = "draft"
	StateArchived  State = "archived"
	StateTrashed   State = "trashed"
	StateInvalid   State = "invalid"
)

func (s State) Valid() bool {
	switch s {
	case StateAvailable, StateDraft, StateArchived, StateTrashed, StateInvalid:
		return true
	default:
		return false
	}
}

type OperationStatus string

const (
	OperationPending          OperationStatus = "pending"
	OperationRunning          OperationStatus = "running"
	OperationCompleted        OperationStatus = "completed"
	OperationPartial          OperationStatus = "partial"
	OperationCancelled        OperationStatus = "cancelled"
	OperationFailed           OperationStatus = "failed"
	OperationRecoveryRequired OperationStatus = "recovery-required"
	OperationRecovered        OperationStatus = "recovered"
)

func (s OperationStatus) Valid() bool {
	switch s {
	case OperationPending, OperationRunning, OperationCompleted, OperationPartial,
		OperationCancelled, OperationFailed, OperationRecoveryRequired, OperationRecovered:
		return true
	default:
		return false
	}
}

func (s OperationStatus) Terminal() bool {
	switch s {
	case OperationCompleted, OperationPartial, OperationCancelled, OperationFailed,
		OperationRecoveryRequired, OperationRecovered:
		return true
	default:
		return false
	}
}

type ItemStatus string

const (
	ItemSucceeded        ItemStatus = "succeeded"
	ItemSkipped          ItemStatus = "skipped"
	ItemCancelled        ItemStatus = "cancelled"
	ItemFailed           ItemStatus = "failed"
	ItemRolledBack       ItemStatus = "rolled-back"
	ItemRecoveryRequired ItemStatus = "recovery-required"
)

func (s ItemStatus) Valid() bool {
	switch s {
	case ItemSucceeded, ItemSkipped, ItemCancelled, ItemFailed, ItemRolledBack, ItemRecoveryRequired:
		return true
	default:
		return false
	}
}

type OperationType string

const (
	OperationSnapshot           OperationType = "snapshot"
	OperationRestore            OperationType = "restore"
	OperationArchive            OperationType = "archive"
	OperationUnarchive          OperationType = "unarchive"
	OperationTrash              OperationType = "trash"
	OperationRestoreTrash       OperationType = "restore-trash"
	OperationPermanentDelete    OperationType = "permanent-delete"
	OperationExportDefinition   OperationType = "export-definition"
	OperationImportDefinition   OperationType = "import-definition"
	OperationCreateTemplate     OperationType = "create-template"
	OperationApplyTemplate      OperationType = "apply-template"
	OperationBulkMetadataUpdate OperationType = "bulk-metadata-update"
	OperationBulkHealthRefresh  OperationType = "bulk-health-refresh"
	OperationStorageReconcile   OperationType = "storage-reconcile"
)

func (t OperationType) Valid() bool {
	switch t {
	case OperationSnapshot, OperationRestore, OperationArchive, OperationUnarchive,
		OperationTrash, OperationRestoreTrash, OperationPermanentDelete,
		OperationExportDefinition, OperationImportDefinition, OperationCreateTemplate,
		OperationApplyTemplate, OperationBulkMetadataUpdate, OperationBulkHealthRefresh,
		OperationStorageReconcile:
		return true
	default:
		return false
	}
}

type OperationLock struct {
	OperationID string    `json:"operationId"`
	AcquiredAt  time.Time `json:"acquiredAt"`
}

type Record struct {
	SchemaVersion     int            `json:"schemaVersion"`
	ProfileID         string         `json:"profileId"`
	State             State          `json:"state"`
	ManagedDir        string         `json:"managedDir"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	ArchivedAt        *time.Time     `json:"archivedAt,omitempty"`
	TrashedAt         *time.Time     `json:"trashedAt,omitempty"`
	RetentionDeadline *time.Time     `json:"retentionDeadline,omitempty"`
	SourceID          string         `json:"sourceId,omitempty"`
	Lock              *OperationLock `json:"lock,omitempty"`
	RecoveryCodes     []string       `json:"recoveryCodes,omitempty"`
	LimitationCodes   []string       `json:"limitationCodes,omitempty"`
	Revision          uint64         `json:"revision"`
}

func NewCompatibilityRecord(profileID, managedDir string, now time.Time) Record {
	now = now.UTC()
	return Record{
		SchemaVersion: LifecycleSchemaVersion,
		ProfileID:     strings.TrimSpace(profileID),
		State:         StateAvailable,
		ManagedDir:    filepath.ToSlash(filepath.Clean(strings.TrimSpace(managedDir))),
		CreatedAt:     now,
		UpdatedAt:     now,
		Revision:      1,
	}
}

func (r Record) Validate() error {
	if r.SchemaVersion != LifecycleSchemaVersion {
		return fmt.Errorf("%w: lifecycle record version %d", ErrUnsupportedVersion, r.SchemaVersion)
	}
	if err := validateIdentifier("profile id", r.ProfileID); err != nil {
		return err
	}
	if !r.State.Valid() {
		return fmt.Errorf("%w: unsupported state %q", ErrInvalidRecord, r.State)
	}
	if err := validateManagedRelativePath(r.ManagedDir); err != nil {
		return err
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: invalid record timestamps", ErrInvalidRecord)
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidRecord)
	}
	if r.Lock != nil {
		if err := validateIdentifier("operation lock id", r.Lock.OperationID); err != nil {
			return err
		}
		if r.Lock.AcquiredAt.IsZero() {
			return fmt.Errorf("%w: operation lock timestamp is required", ErrInvalidRecord)
		}
	}
	if err := validateCodes("recovery codes", r.RecoveryCodes); err != nil {
		return err
	}
	return validateCodes("limitation codes", r.LimitationCodes)
}

type OperationItemResult struct {
	ItemID         string     `json:"itemId"`
	Status         ItemStatus `json:"status"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CompletedStage string     `json:"completedStage,omitempty"`
	FilesProcessed int64      `json:"filesProcessed,omitempty"`
	BytesProcessed int64      `json:"bytesProcessed,omitempty"`
	ReasonCode     string     `json:"reasonCode,omitempty"`
	OutputID       string     `json:"outputId,omitempty"`
	RecoveryID     string     `json:"recoveryId,omitempty"`
	Limitations    []string   `json:"limitations,omitempty"`
}

type Operation struct {
	SchemaVersion         int                   `json:"schemaVersion"`
	ID                    string                `json:"id"`
	Type                  OperationType         `json:"type"`
	ProfileIDs            []string              `json:"profileIds"`
	IdempotencyKey        string                `json:"idempotencyKey,omitempty"`
	PredecessorID         string                `json:"predecessorId,omitempty"`
	Status                OperationStatus       `json:"status"`
	Stage                 string                `json:"stage"`
	StartedAt             time.Time             `json:"startedAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
	CompletedAt           *time.Time            `json:"completedAt,omitempty"`
	CancellationRequested bool                  `json:"cancellationRequested"`
	SafeCancellationStage string                `json:"safeCancellationStage,omitempty"`
	Items                 []OperationItemResult `json:"items,omitempty"`
	Limitations           []string              `json:"limitations,omitempty"`
	RecoveryActions       []string              `json:"recoveryActions,omitempty"`
	StagingRef            string                `json:"stagingRef,omitempty"`
	QuarantineRef         string                `json:"quarantineRef,omitempty"`
	ApplicationVersion    string                `json:"applicationVersion"`
	Platform              string                `json:"platform"`
	Revision              uint64                `json:"revision"`
}

func NewOperation(id string, operationType OperationType, profileIDs []string, now time.Time) Operation {
	now = now.UTC()
	return Operation{
		SchemaVersion: OperationSchemaVersion,
		ID:            strings.TrimSpace(id),
		Type:          operationType,
		ProfileIDs:    normalizeIdentifiers(profileIDs),
		Status:        OperationPending,
		Stage:         "accepted",
		StartedAt:     now,
		UpdatedAt:     now,
		Revision:      1,
	}
}

func (o Operation) Validate() error {
	if o.SchemaVersion != OperationSchemaVersion {
		return fmt.Errorf("%w: operation version %d", ErrUnsupportedVersion, o.SchemaVersion)
	}
	if err := validateIdentifier("operation id", o.ID); err != nil {
		return err
	}
	if !o.Type.Valid() {
		return fmt.Errorf("%w: unsupported operation type %q", ErrInvalidRecord, o.Type)
	}
	if len(o.ProfileIDs) == 0 || len(o.ProfileIDs) > MaxProfilesPerOp {
		return fmt.Errorf("%w: operation must select between 1 and %d profiles", ErrInvalidRecord, MaxProfilesPerOp)
	}
	if !sort.StringsAreSorted(o.ProfileIDs) {
		return fmt.Errorf("%w: profile ids must be sorted", ErrInvalidRecord)
	}
	for i, id := range o.ProfileIDs {
		if err := validateIdentifier("profile id", id); err != nil {
			return err
		}
		if i > 0 && id == o.ProfileIDs[i-1] {
			return fmt.Errorf("%w: duplicate profile id %q", ErrInvalidRecord, id)
		}
	}
	if strings.TrimSpace(o.IdempotencyKey) != o.IdempotencyKey || len(o.IdempotencyKey) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid idempotency key", ErrInvalidRecord)
	}
	if o.PredecessorID != "" {
		if err := validateIdentifier("predecessor id", o.PredecessorID); err != nil {
			return err
		}
	}
	if !o.Status.Valid() {
		return fmt.Errorf("%w: unsupported operation status %q", ErrInvalidRecord, o.Status)
	}
	if err := validateText("operation stage", o.Stage, true); err != nil {
		return err
	}
	if o.StartedAt.IsZero() || o.UpdatedAt.IsZero() || o.UpdatedAt.Before(o.StartedAt) {
		return fmt.Errorf("%w: invalid operation timestamps", ErrInvalidRecord)
	}
	if o.Status.Terminal() != (o.CompletedAt != nil) {
		return fmt.Errorf("%w: terminal status and completion timestamp disagree", ErrInvalidRecord)
	}
	if o.Status.Terminal() && len(o.Items) == 0 {
		return fmt.Errorf("%w: terminal operation requires item results", ErrInvalidRecord)
	}
	if o.CompletedAt != nil && o.CompletedAt.Before(o.StartedAt) {
		return fmt.Errorf("%w: operation completion precedes start", ErrInvalidRecord)
	}
	if o.Revision == 0 {
		return fmt.Errorf("%w: operation revision must be positive", ErrInvalidRecord)
	}
	if len(o.Items) > MaxItemsPerOp {
		return fmt.Errorf("%w: too many operation item results", ErrInvalidRecord)
	}
	seenItems := make(map[string]struct{}, len(o.Items))
	for _, item := range o.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if _, exists := seenItems[item.ItemID]; exists {
			return fmt.Errorf("%w: duplicate operation item %q", ErrInvalidRecord, item.ItemID)
		}
		seenItems[item.ItemID] = struct{}{}
	}
	if o.Status == OperationCompleted {
		for _, item := range o.Items {
			if item.Status != ItemSucceeded {
				return fmt.Errorf("%w: completed operation contains non-success item", ErrInvalidRecord)
			}
		}
	}
	if err := validateCodes("operation limitations", o.Limitations); err != nil {
		return err
	}
	if err := validateCodes("recovery actions", o.RecoveryActions); err != nil {
		return err
	}
	if err := validateManagedReference("staging reference", o.StagingRef); err != nil {
		return err
	}
	if err := validateManagedReference("quarantine reference", o.QuarantineRef); err != nil {
		return err
	}
	if err := validateText("application version", o.ApplicationVersion, true); err != nil {
		return err
	}
	return validateText("platform", o.Platform, true)
}

func (r OperationItemResult) Validate() error {
	if err := validateIdentifier("operation item id", r.ItemID); err != nil {
		return err
	}
	if !r.Status.Valid() {
		return fmt.Errorf("%w: unsupported item status %q", ErrInvalidRecord, r.Status)
	}
	if r.CompletedAt == nil {
		return fmt.Errorf("%w: item completion timestamp is required", ErrInvalidRecord)
	}
	if r.CompletedAt != nil && r.StartedAt != nil && r.CompletedAt.Before(*r.StartedAt) {
		return fmt.Errorf("%w: item completion precedes start", ErrInvalidRecord)
	}
	if r.FilesProcessed < 0 || r.BytesProcessed < 0 {
		return fmt.Errorf("%w: item progress cannot be negative", ErrInvalidRecord)
	}
	for label, value := range map[string]string{
		"completed stage": r.CompletedStage,
		"reason code":     r.ReasonCode,
		"output id":       r.OutputID,
		"recovery id":     r.RecoveryID,
	} {
		if err := validateText(label, value, false); err != nil {
			return err
		}
	}
	return validateCodes("item limitations", r.Limitations)
}

func normalizeIdentifiers(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validateIdentifier(label, value string) error {
	if strings.TrimSpace(value) != value || value == "" || len(value) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid %s", ErrInvalidRecord, label)
	}
	return nil
}

func validateText(label, value string, required bool) error {
	if strings.TrimSpace(value) != value || len(value) > MaxTextLength || (required && value == "") {
		return fmt.Errorf("%w: invalid %s", ErrInvalidRecord, label)
	}
	return nil
}

func validateCodes(label string, values []string) error {
	if len(values) > MaxCodesPerRecord {
		return fmt.Errorf("%w: too many %s", ErrInvalidRecord, label)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateText(label, value, true); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate %s value %q", ErrInvalidRecord, label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateManagedRelativePath(value string) error {
	if err := validateText("managed directory", value, true); err != nil {
		return err
	}
	if filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return fmt.Errorf("%w: managed directory must be relative", ErrInvalidRecord)
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != filepath.ToSlash(value) {
		return fmt.Errorf("%w: managed directory is not canonical", ErrInvalidRecord)
	}
	return nil
}

func validateManagedReference(label, value string) error {
	if value == "" {
		return nil
	}
	if err := validateText(label, value, true); err != nil {
		return err
	}
	if filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return fmt.Errorf("%w: %s must be relative", ErrInvalidRecord, label)
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != filepath.ToSlash(value) {
		return fmt.Errorf("%w: %s is not canonical", ErrInvalidRecord, label)
	}
	return nil
}
