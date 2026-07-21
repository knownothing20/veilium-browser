package localrecovery

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"
)

type trashEnvelope struct {
	SchemaVersion int           `json:"schemaVersion"`
	Records       []TrashRecord `json:"records"`
}

type TrashStore struct {
	mu      sync.RWMutex
	path    string
	records map[string]TrashRecord
	now     func() time.Time
	write   writeFileFunc
}

func OpenTrashStore(filePath string) (*TrashStore, error) {
	store := &TrashStore{
		path:    filePath,
		records: make(map[string]TrashRecord),
		now:     func() time.Time { return time.Now().UTC() },
	}
	store.write = func(path string, data []byte) error {
		return atomicWrite(path, data, MaxTrashCatalogSize, true)
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TrashStore) List() []TrashRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]TrashRecord, 0, len(s.records))
	for _, record := range s.records {
		result = append(result, cloneTrashRecord(record))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TrashedAt.Equal(result[j].TrashedAt) {
			return result[i].TrashID < result[j].TrashID
		}
		return result[i].TrashedAt.Before(result[j].TrashedAt)
	})
	return result
}

func (s *TrashStore) Get(trashID string) (TrashRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.records[trashID]
	if !exists {
		return TrashRecord{}, ErrNotFound
	}
	return cloneTrashRecord(record), nil
}

func (s *TrashStore) Create(input TrashRecord) (TrashRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[input.TrashID]; exists {
		return TrashRecord{}, ErrAlreadyExists
	}
	if len(s.records) >= MaxTrashRecords {
		return TrashRecord{}, fmt.Errorf("%w: trash record limit reached", ErrInvalidRecord)
	}
	now := s.now().UTC()
	input.SchemaVersion = TrashSchemaVersion
	input.Status = TrashPending
	if input.TrashedAt.IsZero() {
		input.TrashedAt = now
	}
	input.UpdatedAt = now
	input.Revision = 1
	input.OriginalLimitationCodes = sortedUnique(input.OriginalLimitationCodes)
	input.Limitations = sortedUnique(input.Limitations)
	if err := input.Validate(); err != nil {
		return TrashRecord{}, err
	}
	for _, record := range s.records {
		if record.ProfileID == input.ProfileID {
			return TrashRecord{}, fmt.Errorf("%w: Profile %q already has trash record %q", ErrConflict, input.ProfileID, record.TrashID)
		}
		if record.TrashRef == input.TrashRef {
			return TrashRecord{}, fmt.Errorf("%w: trash reference already belongs to %q", ErrConflict, record.TrashID)
		}
	}
	next := cloneTrashMap(s.records)
	next[input.TrashID] = cloneTrashRecord(input)
	if err := s.persist(next); err != nil {
		return TrashRecord{}, err
	}
	s.records = next
	return cloneTrashRecord(input), nil
}

func (s *TrashStore) Update(input TrashRecord) (TrashRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[input.TrashID]
	if !exists {
		return TrashRecord{}, ErrNotFound
	}
	if input.Revision != current.Revision {
		return TrashRecord{}, ErrConflict
	}
	if !sameTrashIdentity(current, input) {
		return TrashRecord{}, fmt.Errorf("%w: immutable trash identity changed", ErrConflict)
	}
	if !validTrashTransition(current.Status, input.Status) {
		return TrashRecord{}, fmt.Errorf("%w: unsupported trash transition %q to %q", ErrConflict, current.Status, input.Status)
	}
	input.SchemaVersion = TrashSchemaVersion
	input.UpdatedAt = s.now().UTC()
	input.Revision = current.Revision + 1
	input.OriginalLimitationCodes = sortedUnique(input.OriginalLimitationCodes)
	input.Limitations = sortedUnique(input.Limitations)
	if err := input.Validate(); err != nil {
		return TrashRecord{}, err
	}
	next := cloneTrashMap(s.records)
	next[input.TrashID] = cloneTrashRecord(input)
	if err := s.persist(next); err != nil {
		return TrashRecord{}, err
	}
	s.records = next
	return cloneTrashRecord(input), nil
}

func (s *TrashStore) Remove(trashID string, revision uint64) (TrashRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[trashID]
	if !exists {
		return TrashRecord{}, ErrNotFound
	}
	if revision != current.Revision {
		return TrashRecord{}, ErrConflict
	}
	next := cloneTrashMap(s.records)
	delete(next, trashID)
	if err := s.persist(next); err != nil {
		return TrashRecord{}, err
	}
	s.records = next
	return cloneTrashRecord(current), nil
}

func (s *TrashStore) load() error {
	var envelope trashEnvelope
	if err := decodeStrictFile(s.path, MaxTrashCatalogSize, ErrInvalidRecord, &envelope); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open trash catalog: %w", err)
	}
	if envelope.SchemaVersion != TrashSchemaVersion {
		return fmt.Errorf("%w: trash envelope version %d", ErrUnsupportedVersion, envelope.SchemaVersion)
	}
	if len(envelope.Records) > MaxTrashRecords {
		return fmt.Errorf("%w: too many trash records", ErrInvalidRecord)
	}
	records := make(map[string]TrashRecord, len(envelope.Records))
	profiles := make(map[string]string, len(envelope.Records))
	refs := make(map[string]string, len(envelope.Records))
	for _, record := range envelope.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("load trash record %q: %w", record.TrashID, err)
		}
		if _, exists := records[record.TrashID]; exists {
			return fmt.Errorf("%w: duplicate trash id %q", ErrInvalidRecord, record.TrashID)
		}
		if existing, exists := profiles[record.ProfileID]; exists {
			return fmt.Errorf("%w: Profile %q has trash records %q and %q", ErrInvalidRecord, record.ProfileID, existing, record.TrashID)
		}
		if existing, exists := refs[record.TrashRef]; exists {
			return fmt.Errorf("%w: trash reference shared by %q and %q", ErrInvalidRecord, existing, record.TrashID)
		}
		records[record.TrashID] = cloneTrashRecord(record)
		profiles[record.ProfileID] = record.TrashID
		refs[record.TrashRef] = record.TrashID
	}
	s.records = records
	return nil
}

func (s *TrashStore) persist(records map[string]TrashRecord) error {
	items := make([]TrashRecord, 0, len(records))
	for _, record := range records {
		items = append(items, cloneTrashRecord(record))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TrashID < items[j].TrashID })
	data, err := encodeBounded(trashEnvelope{SchemaVersion: TrashSchemaVersion, Records: items}, MaxTrashCatalogSize, ErrInvalidRecord)
	if err != nil {
		return err
	}
	if err := s.write(s.path, data); err != nil {
		return fmt.Errorf("persist trash catalog: %w", err)
	}
	return nil
}

func sameTrashIdentity(left, right TrashRecord) bool {
	return left.ProfileID == right.ProfileID &&
		left.OperatingSystem == right.OperatingSystem &&
		left.Architecture == right.Architecture &&
		left.OriginalState == right.OriginalState &&
		left.OriginalManagedDir == right.OriginalManagedDir &&
		timePointersEqual(left.OriginalArchivedAt, right.OriginalArchivedAt) &&
		left.OriginalSourceID == right.OriginalSourceID &&
		reflect.DeepEqual(left.OriginalLimitationCodes, right.OriginalLimitationCodes) &&
		left.TrashRef == right.TrashRef &&
		left.DataPresent == right.DataPresent &&
		left.ProfileDefinitionDigest == right.ProfileDefinitionDigest &&
		left.TreeDigest == right.TreeDigest &&
		left.FileCount == right.FileCount &&
		left.TotalBytes == right.TotalBytes &&
		left.TrashedAt.Equal(right.TrashedAt) &&
		left.RetentionDeadline.Equal(right.RetentionDeadline)
}

func validTrashTransition(from, to TrashStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case TrashPending:
		return to == TrashStored || to == TrashRecoveryRequired
	case TrashStored:
		return to == TrashRestoring || to == TrashCleanupPending || to == TrashRecoveryRequired
	case TrashRestoring:
		return to == TrashStored || to == TrashRecoveryRequired
	case TrashCleanupPending:
		return to == TrashStored || to == TrashRecoveryRequired
	case TrashRecoveryRequired:
		return to == TrashStored
	default:
		return false
	}
}

func cloneTrashMap(source map[string]TrashRecord) map[string]TrashRecord {
	result := make(map[string]TrashRecord, len(source))
	for key, record := range source {
		result[key] = cloneTrashRecord(record)
	}
	return result
}

func cloneTrashRecord(record TrashRecord) TrashRecord {
	record.OriginalLimitationCodes = append([]string(nil), record.OriginalLimitationCodes...)
	record.Limitations = append([]string(nil), record.Limitations...)
	if record.OriginalArchivedAt != nil {
		value := *record.OriginalArchivedAt
		record.OriginalArchivedAt = &value
	}
	return record
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return uniqueStrings(result)
}

func timePointersEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
