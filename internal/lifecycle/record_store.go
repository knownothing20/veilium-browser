package lifecycle

import (
	"errors"
	"fmt"
	"os"
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
