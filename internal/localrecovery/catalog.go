package localrecovery

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type catalogEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Records       []CatalogRecord `json:"records"`
}

type CatalogStore struct {
	mu      sync.RWMutex
	path    string
	records map[string]CatalogRecord
	now     func() time.Time
	write   writeFileFunc
}

func OpenCatalogStore(filePath string) (*CatalogStore, error) {
	store := &CatalogStore{
		path:    filePath,
		records: make(map[string]CatalogRecord),
		now:     func() time.Time { return time.Now().UTC() },
	}
	store.write = func(path string, data []byte) error {
		return atomicWrite(path, data, MaxCatalogBytes, true)
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *CatalogStore) List() []CatalogRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]CatalogRecord, 0, len(s.records))
	for _, record := range s.records {
		result = append(result, cloneCatalogRecord(record))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].SnapshotID < result[j].SnapshotID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (s *CatalogStore) Get(snapshotID string) (CatalogRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.records[snapshotID]
	if !exists {
		return CatalogRecord{}, ErrNotFound
	}
	return cloneCatalogRecord(record), nil
}

func (s *CatalogStore) Create(input CatalogRecord) (CatalogRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[input.SnapshotID]; exists {
		return CatalogRecord{}, ErrAlreadyExists
	}
	if len(s.records) >= MaxSnapshots {
		return CatalogRecord{}, fmt.Errorf("%w: catalog record limit reached", ErrInvalidRecord)
	}
	now := s.now().UTC()
	input.SchemaVersion = CatalogSchemaVersion
	if input.Status == "" {
		input.Status = SnapshotPending
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	input.Revision = 1
	if input.Status != SnapshotVerified {
		input.VerifiedAt = nil
	}
	if err := input.Validate(); err != nil {
		return CatalogRecord{}, err
	}
	for _, record := range s.records {
		if record.ManifestRef == input.ManifestRef {
			return CatalogRecord{}, fmt.Errorf("%w: manifest reference already belongs to %q", ErrConflict, record.SnapshotID)
		}
	}
	next := cloneCatalogMap(s.records)
	next[input.SnapshotID] = cloneCatalogRecord(input)
	if err := s.persist(next); err != nil {
		return CatalogRecord{}, err
	}
	s.records = next
	return cloneCatalogRecord(input), nil
}

func (s *CatalogStore) Update(input CatalogRecord) (CatalogRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[input.SnapshotID]
	if !exists {
		return CatalogRecord{}, ErrNotFound
	}
	if input.Revision != current.Revision {
		return CatalogRecord{}, ErrConflict
	}
	if input.SourceProfileID != current.SourceProfileID || input.ManifestRef != current.ManifestRef || input.ManifestDigest != current.ManifestDigest || input.TreeDigest != current.TreeDigest || input.FileCount != current.FileCount || input.TotalBytes != current.TotalBytes || !input.CreatedAt.Equal(current.CreatedAt) {
		return CatalogRecord{}, fmt.Errorf("%w: immutable catalog identity changed", ErrConflict)
	}
	input.SchemaVersion = CatalogSchemaVersion
	input.UpdatedAt = s.now().UTC()
	input.Revision = current.Revision + 1
	if err := input.Validate(); err != nil {
		return CatalogRecord{}, err
	}
	next := cloneCatalogMap(s.records)
	next[input.SnapshotID] = cloneCatalogRecord(input)
	if err := s.persist(next); err != nil {
		return CatalogRecord{}, err
	}
	s.records = next
	return cloneCatalogRecord(input), nil
}

func (s *CatalogStore) load() error {
	var envelope catalogEnvelope
	if err := decodeStrictFile(s.path, MaxCatalogBytes, ErrInvalidRecord, &envelope); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open local recovery catalog: %w", err)
	}
	if envelope.SchemaVersion != CatalogSchemaVersion {
		return fmt.Errorf("%w: catalog envelope version %d", ErrUnsupportedVersion, envelope.SchemaVersion)
	}
	if len(envelope.Records) > MaxSnapshots {
		return fmt.Errorf("%w: too many catalog records", ErrInvalidRecord)
	}
	records := make(map[string]CatalogRecord, len(envelope.Records))
	manifestRefs := make(map[string]string, len(envelope.Records))
	for _, record := range envelope.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("load local recovery catalog record %q: %w", record.SnapshotID, err)
		}
		if _, exists := records[record.SnapshotID]; exists {
			return fmt.Errorf("%w: duplicate snapshot id %q", ErrInvalidRecord, record.SnapshotID)
		}
		if existing, exists := manifestRefs[record.ManifestRef]; exists {
			return fmt.Errorf("%w: manifest reference shared by %q and %q", ErrInvalidRecord, existing, record.SnapshotID)
		}
		records[record.SnapshotID] = cloneCatalogRecord(record)
		manifestRefs[record.ManifestRef] = record.SnapshotID
	}
	s.records = records
	return nil
}

func (s *CatalogStore) persist(records map[string]CatalogRecord) error {
	items := make([]CatalogRecord, 0, len(records))
	for _, record := range records {
		items = append(items, cloneCatalogRecord(record))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SnapshotID < items[j].SnapshotID })
	data, err := encodeBounded(catalogEnvelope{SchemaVersion: CatalogSchemaVersion, Records: items}, MaxCatalogBytes, ErrInvalidRecord)
	if err != nil {
		return err
	}
	if err := s.write(s.path, data); err != nil {
		return fmt.Errorf("persist local recovery catalog: %w", err)
	}
	return nil
}

func cloneCatalogMap(source map[string]CatalogRecord) map[string]CatalogRecord {
	result := make(map[string]CatalogRecord, len(source))
	for key, record := range source {
		result[key] = cloneCatalogRecord(record)
	}
	return result
}

func cloneCatalogRecord(record CatalogRecord) CatalogRecord {
	record.Limitations = append([]string(nil), record.Limitations...)
	if record.VerifiedAt != nil {
		value := *record.VerifiedAt
		record.VerifiedAt = &value
	}
	return record
}
