package lifecycle

import "fmt"

// RemoveRecord removes lifecycle metadata only. It is used for cross-store
// rollback and never changes Profile definitions or managed browser files.
func (s *RecordStore) RemoveRecord(profileID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.records[profileID]
	if !exists {
		return Record{}, ErrNotFound
	}
	if current.Lock != nil {
		return Record{}, fmt.Errorf("%w: profile %q is locked by operation %q", ErrConflict, profileID, current.Lock.OperationID)
	}

	next := cloneRecordMap(s.records)
	delete(next, profileID)
	if err := s.persist(next); err != nil {
		return Record{}, err
	}
	s.records = next
	return cloneRecord(current), nil
}
