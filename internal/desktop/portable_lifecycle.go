package desktop

import (
	"fmt"
	"sort"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func (s *Service) markPortableProfileDraft(profileID, operationID, sourceID string, codes ...string) error {
	if s.lifecycleRecords == nil {
		return fmt.Errorf("lifecycle service is unavailable")
	}
	record, err := s.lifecycleRecords.Get(strings.TrimSpace(profileID))
	if err != nil {
		return err
	}
	next, err := preparePortableDraftRecord(record, operationID, sourceID, codes...)
	if err != nil {
		return err
	}
	_, err = s.lifecycleRecords.Update(next)
	return err
}

func preparePortableDraftRecord(record lifecycle.Record, operationID, sourceID string, codes ...string) (lifecycle.Record, error) {
	operationID = strings.TrimSpace(operationID)
	sourceID = strings.TrimSpace(sourceID)
	if record.Lock == nil || record.Lock.OperationID != operationID {
		return lifecycle.Record{}, fmt.Errorf("%w: portable operation does not own the imported Profile lock", lifecycle.ErrConflict)
	}
	if record.State != lifecycle.StateAvailable && record.State != lifecycle.StateDraft {
		return lifecycle.Record{}, fmt.Errorf("%w: imported Profile lifecycle state is %q", lifecycle.ErrConflict, record.State)
	}
	if sourceID == "" || len(sourceID) > lifecycle.MaxIdentifierLength {
		return lifecycle.Record{}, fmt.Errorf("%w: portable source identity is invalid", lifecycle.ErrInvalidRecord)
	}
	record.State = lifecycle.StateDraft
	record.SourceID = sourceID
	seen := make(map[string]struct{}, len(record.LimitationCodes)+len(codes))
	limitations := make([]string, 0, len(record.LimitationCodes)+len(codes))
	for _, value := range append(append([]string(nil), record.LimitationCodes...), codes...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		limitations = append(limitations, value)
	}
	sort.Strings(limitations)
	record.LimitationCodes = limitations
	return record, nil
}
