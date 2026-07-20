package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type recordEnvelope struct {
	SchemaVersion int      `json:"schemaVersion"`
	Records       []Record `json:"records"`
}

type RecordStore struct {
	mu      sync.RWMutex
	path    string
	records map[string]Record
	now     func() time.Time
	write   writeFileFunc
}

func OpenRecordStore(path string) (*RecordStore, error) {
	store := &RecordStore{
		path:    path,
		records: make(map[string]Record),
		now:     func() time.Time { return time.Now().UTC() },
		write:   atomicWrite,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *RecordStore) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		result = append(result, cloneRecord(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ProfileID < result[j].ProfileID })
	return result
}

func (s *RecordStore) Get(profileID string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[profileID]
	if !ok {
		return Record{}, ErrNotFound
	}
	return cloneRecord(record), nil
}

func (s *RecordStore) Create(input Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[input.ProfileID]; exists {
		return Record{}, ErrAlreadyExists
	}
	now := s.now().UTC()
	input.SchemaVersion = LifecycleSchemaVersion
	input.CreatedAt = now
	input.UpdatedAt = now
	input.Revision = 1
	if err := input.Validate(); err != nil {
		return Record{}, err
	}
	next := cloneRecordMap(s.records)
	next[input.ProfileID] = cloneRecord(input)
	if err := s.persist(next); err != nil {
		return Record{}, err
	}
	s.records = next
	return cloneRecord(input), nil
}

func (s *RecordStore) Update(input Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[input.ProfileID]
	if !exists {
		return Record{}, ErrNotFound
	}
	if input.Revision != current.Revision {
		return Record{}, ErrConflict
	}
	input.SchemaVersion = LifecycleSchemaVersion
	input.CreatedAt = current.CreatedAt
	input.UpdatedAt = s.now().UTC()
	input.Revision = current.Revision + 1
	if err := input.Validate(); err != nil {
		return Record{}, err
	}
	next := cloneRecordMap(s.records)
	next[input.ProfileID] = cloneRecord(input)
	if err := s.persist(next); err != nil {
		return Record{}, err
	}
	s.records = next
	return cloneRecord(input), nil
}

func (s *RecordStore) AcquireLocks(operationID string, profileIDs []string, acquiredAt time.Time) ([]Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateIdentifier("operation id", operationID); err != nil {
		return nil, false, err
	}
	ids := normalizeIdentifiers(profileIDs)
	if len(ids) == 0 || len(ids) > MaxProfilesPerOp {
		return nil, false, fmt.Errorf("%w: invalid lock profile selection", ErrInvalidRecord)
	}
	allReused := true
	for _, id := range ids {
		record, exists := s.records[id]
		if !exists {
			return nil, false, fmt.Errorf("%w: profile %q", ErrNotFound, id)
		}
		if record.Lock == nil {
			allReused = false
			continue
		}
		if record.Lock.OperationID != operationID {
			return nil, false, fmt.Errorf("%w: profile %q is locked by operation %q", ErrConflict, id, record.Lock.OperationID)
		}
	}
	if allReused {
		result := make([]Record, 0, len(ids))
		for _, id := range ids {
			result = append(result, cloneRecord(s.records[id]))
		}
		return result, true, nil
	}

	next := cloneRecordMap(s.records)
	for _, id := range ids {
		record := next[id]
		if record.Lock == nil {
			record.Lock = &OperationLock{OperationID: operationID, AcquiredAt: acquiredAt.UTC()}
			record.UpdatedAt = s.now().UTC()
			record.Revision++
			if err := record.Validate(); err != nil {
				return nil, false, err
			}
			next[id] = record
		}
	}
	if err := s.persist(next); err != nil {
		return nil, false, err
	}
	s.records = next
	result := make([]Record, 0, len(ids))
	for _, id := range ids {
		result = append(result, cloneRecord(next[id]))
	}
	return result, false, nil
}

func (s *RecordStore) ReleaseLocks(operationID string, profileIDs []string) ([]Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateIdentifier("operation id", operationID); err != nil {
		return nil, false, err
	}
	ids := normalizeIdentifiers(profileIDs)
	if len(ids) == 0 || len(ids) > MaxProfilesPerOp {
		return nil, false, fmt.Errorf("%w: invalid unlock profile selection", ErrInvalidRecord)
	}
	changed := false
	for _, id := range ids {
		record, exists := s.records[id]
		if !exists {
			return nil, false, fmt.Errorf("%w: profile %q", ErrNotFound, id)
		}
		if record.Lock != nil && record.Lock.OperationID != operationID {
			return nil, false, fmt.Errorf("%w: profile %q lock ownership changed", ErrConflict, id)
		}
		changed = changed || record.Lock != nil
	}
	if !changed {
		result := make([]Record, 0, len(ids))
		for _, id := range ids {
			result = append(result, cloneRecord(s.records[id]))
		}
		return result, false, nil
	}

	next := cloneRecordMap(s.records)
	for _, id := range ids {
		record := next[id]
		if record.Lock != nil {
			record.Lock = nil
			record.UpdatedAt = s.now().UTC()
			record.Revision++
			if err := record.Validate(); err != nil {
				return nil, false, err
			}
			next[id] = record
		}
	}
	if err := s.persist(next); err != nil {
		return nil, false, err
	}
	s.records = next
	result := make([]Record, 0, len(ids))
	for _, id := range ids {
		result = append(result, cloneRecord(next[id]))
	}
	return result, true, nil
}

func (s *RecordStore) AddRecoveryCode(profileID, code string) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateText("recovery code", code, true); err != nil {
		return Record{}, false, err
	}
	record, exists := s.records[profileID]
	if !exists {
		return Record{}, false, ErrNotFound
	}
	for _, existing := range record.RecoveryCodes {
		if existing == code {
			return cloneRecord(record), false, nil
		}
	}
	record.RecoveryCodes = append(record.RecoveryCodes, code)
	sort.Strings(record.RecoveryCodes)
	record.UpdatedAt = s.now().UTC()
	record.Revision++
	if err := record.Validate(); err != nil {
		return Record{}, false, err
	}
	next := cloneRecordMap(s.records)
	next[profileID] = cloneRecord(record)
	if err := s.persist(next); err != nil {
		return Record{}, false, err
	}
	s.records = next
	return cloneRecord(record), true, nil
}

type CompatibilityInput struct {
	ProfileID       string
	ManagedDir      string
	State           State
	RecoveryCodes   []string
	LimitationCodes []string
}

func (s *RecordStore) EnsureCompatibility(inputs []CompatibilityInput) ([]Record, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(inputs) > MaxRecords {
		return nil, nil, fmt.Errorf("%w: too many compatibility records", ErrInvalidRecord)
	}
	next := cloneRecordMap(s.records)
	created := make([]string, 0)
	seenProfiles := make(map[string]struct{}, len(inputs))
	seenDirs := make(map[string]string, len(inputs))
	now := s.now().UTC()
	for _, input := range inputs {
		if err := validateIdentifier("profile id", input.ProfileID); err != nil {
			return nil, nil, err
		}
		if _, exists := seenProfiles[input.ProfileID]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate compatibility profile %q", ErrInvalidRecord, input.ProfileID)
		}
		seenProfiles[input.ProfileID] = struct{}{}
		managedDir := filepath.ToSlash(filepath.Clean(input.ManagedDir))
		if err := validateManagedRelativePath(managedDir); err != nil {
			return nil, nil, err
		}
		if other, exists := seenDirs[managedDir]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate managed directory %q for profiles %q and %q", ErrInvalidRecord, managedDir, other, input.ProfileID)
		}
		seenDirs[managedDir] = input.ProfileID
		if existing, exists := next[input.ProfileID]; exists {
			if existing.ManagedDir != managedDir {
				return nil, nil, fmt.Errorf("%w: profile %q lifecycle managed directory changed", ErrConflict, input.ProfileID)
			}
			continue
		}
		state := input.State
		if state == "" {
			state = StateAvailable
		}
		record := Record{
			SchemaVersion:   LifecycleSchemaVersion,
			ProfileID:       input.ProfileID,
			State:           state,
			ManagedDir:      managedDir,
			CreatedAt:       now,
			UpdatedAt:       now,
			RecoveryCodes:   append([]string(nil), input.RecoveryCodes...),
			LimitationCodes: append([]string(nil), input.LimitationCodes...),
			Revision:        1,
		}
		sort.Strings(record.RecoveryCodes)
		sort.Strings(record.LimitationCodes)
		if err := record.Validate(); err != nil {
			return nil, nil, err
		}
		next[input.ProfileID] = record
		created = append(created, input.ProfileID)
	}
	if len(created) > 0 {
		if err := s.persist(next); err != nil {
			return nil, nil, err
		}
		s.records = next
	}
	sort.Strings(created)
	result := make([]Record, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, cloneRecord(next[input.ProfileID]))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ProfileID < result[j].ProfileID })
	return result, created, nil
}

func (s *RecordStore) ReconcileLock(profileID, operationID, recoveryCode string) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateIdentifier("profile id", profileID); err != nil {
		return Record{}, false, err
	}
	if err := validateIdentifier("operation id", operationID); err != nil {
		return Record{}, false, err
	}
	if err := validateText("recovery code", recoveryCode, true); err != nil {
		return Record{}, false, err
	}
	record, exists := s.records[profileID]
	if !exists {
		return Record{}, false, ErrNotFound
	}
	if record.Lock == nil {
		return cloneRecord(record), false, nil
	}
	if record.Lock.OperationID != operationID {
		return Record{}, false, fmt.Errorf("%w: profile %q lock ownership changed", ErrConflict, profileID)
	}
	record.Lock = nil
	record.RecoveryCodes = appendUnique(record.RecoveryCodes, recoveryCode)
	sort.Strings(record.RecoveryCodes)
	record.UpdatedAt = s.now().UTC()
	record.Revision++
	if err := record.Validate(); err != nil {
		return Record{}, false, err
	}
	next := cloneRecordMap(s.records)
	next[profileID] = cloneRecord(record)
	if err := s.persist(next); err != nil {
		return Record{}, false, err
	}
	s.records = next
	return cloneRecord(record), true, nil
}

func (s *RecordStore) load() error {
	var envelope recordEnvelope
	if err := decodeStrictFile(s.path, &envelope); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open lifecycle records: %w", err)
	}
	if envelope.SchemaVersion != LifecycleSchemaVersion {
		return fmt.Errorf("%w: lifecycle envelope version %d", ErrUnsupportedVersion, envelope.SchemaVersion)
	}
	if len(envelope.Records) > MaxRecords {
		return fmt.Errorf("%w: too many lifecycle records", ErrInvalidRecord)
	}
	records := make(map[string]Record, len(envelope.Records))
	for _, record := range envelope.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("load lifecycle record %q: %w", record.ProfileID, err)
		}
		if _, exists := records[record.ProfileID]; exists {
			return fmt.Errorf("%w: duplicate lifecycle profile id %q", ErrInvalidRecord, record.ProfileID)
		}
		records[record.ProfileID] = cloneRecord(record)
	}
	s.records = records
	return nil
}

func (s *RecordStore) persist(records map[string]Record) error {
	items := make([]Record, 0, len(records))
	for _, record := range records {
		items = append(items, cloneRecord(record))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProfileID < items[j].ProfileID })
	data, err := encodeIndented(recordEnvelope{SchemaVersion: LifecycleSchemaVersion, Records: items})
	if err != nil {
		return err
	}
	if err := s.write(s.path, data); err != nil {
		return fmt.Errorf("persist lifecycle records: %w", err)
	}
	return nil
}

func cloneRecordMap(source map[string]Record) map[string]Record {
	result := make(map[string]Record, len(source))
	for key, record := range source {
		result[key] = cloneRecord(record)
	}
	return result
}

func cloneRecord(record Record) Record {
	record.RecoveryCodes = append([]string(nil), record.RecoveryCodes...)
	record.LimitationCodes = append([]string(nil), record.LimitationCodes...)
	if record.Lock != nil {
		lock := *record.Lock
		record.Lock = &lock
	}
	return record
}
