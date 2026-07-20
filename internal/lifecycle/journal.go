package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"
)

type operationEnvelope struct {
	SchemaVersion int         `json:"schemaVersion"`
	Operations    []Operation `json:"operations"`
}

type Journal struct {
	mu          sync.RWMutex
	path        string
	operations  map[string]Operation
	idempotency map[string]string
	now         func() time.Time
	write       writeFileFunc
}

func OpenJournal(path string) (*Journal, error) {
	journal := &Journal{
		path:        path,
		operations:  make(map[string]Operation),
		idempotency: make(map[string]string),
		now:         func() time.Time { return time.Now().UTC() },
		write:       atomicWrite,
	}
	if err := journal.load(); err != nil {
		return nil, err
	}
	return journal, nil
}

func (j *Journal) List() []Operation {
	j.mu.RLock()
	defer j.mu.RUnlock()
	result := make([]Operation, 0, len(j.operations))
	for _, operation := range j.operations {
		result = append(result, cloneOperation(operation))
	}
	sort.Slice(result, func(i, k int) bool {
		if result[i].StartedAt.Equal(result[k].StartedAt) {
			return result[i].ID < result[k].ID
		}
		return result[i].StartedAt.Before(result[k].StartedAt)
	})
	return result
}

func (j *Journal) Get(id string) (Operation, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	operation, ok := j.operations[id]
	if !ok {
		return Operation{}, ErrNotFound
	}
	return cloneOperation(operation), nil
}

func (j *Journal) Create(input Operation) (operation Operation, reused bool, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if existing, exists := j.operations[input.ID]; exists {
		if sameOperationRequest(existing, input, true) {
			return cloneOperation(existing), true, nil
		}
		return Operation{}, false, ErrConflict
	}
	if input.IdempotencyKey != "" {
		if existingID, exists := j.idempotency[input.IdempotencyKey]; exists {
			existing := j.operations[existingID]
			if sameOperationRequest(existing, input, false) {
				return cloneOperation(existing), true, nil
			}
			return Operation{}, false, ErrConflict
		}
	}

	now := j.now().UTC()
	input.SchemaVersion = OperationSchemaVersion
	input.ProfileIDs = normalizeIdentifiers(input.ProfileIDs)
	input.StartedAt = now
	input.UpdatedAt = now
	input.CompletedAt = nil
	input.Revision = 1
	if input.Status == "" {
		input.Status = OperationPending
	}
	if input.Stage == "" {
		input.Stage = "accepted"
	}
	if err := input.Validate(); err != nil {
		return Operation{}, false, err
	}

	next := cloneOperationMap(j.operations)
	next[input.ID] = cloneOperation(input)
	if err := j.persist(next); err != nil {
		return Operation{}, false, err
	}
	j.operations = next
	if input.IdempotencyKey != "" {
		j.idempotency[input.IdempotencyKey] = input.ID
	}
	return cloneOperation(input), false, nil
}

func (j *Journal) Update(input Operation) (Operation, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	current, exists := j.operations[input.ID]
	if !exists {
		return Operation{}, ErrNotFound
	}
	if input.Revision != current.Revision {
		return Operation{}, ErrConflict
	}
	if input.Type != current.Type || !reflect.DeepEqual(input.ProfileIDs, current.ProfileIDs) ||
		input.IdempotencyKey != current.IdempotencyKey || input.PredecessorID != current.PredecessorID ||
		!input.StartedAt.Equal(current.StartedAt) {
		return Operation{}, fmt.Errorf("%w: immutable operation request changed", ErrConflict)
	}
	input.SchemaVersion = OperationSchemaVersion
	input.UpdatedAt = j.now().UTC()
	input.Revision = current.Revision + 1
	if err := input.Validate(); err != nil {
		return Operation{}, err
	}
	next := cloneOperationMap(j.operations)
	next[input.ID] = cloneOperation(input)
	if err := j.persist(next); err != nil {
		return Operation{}, err
	}
	j.operations = next
	return cloneOperation(input), nil
}

func (j *Journal) RequestCancellation(id string) (Operation, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	current, exists := j.operations[id]
	if !exists {
		return Operation{}, false, ErrNotFound
	}
	if current.Status.Terminal() || current.CancellationRequested {
		return cloneOperation(current), false, nil
	}
	current.CancellationRequested = true
	current.UpdatedAt = j.now().UTC()
	current.Revision++
	if err := current.Validate(); err != nil {
		return Operation{}, false, err
	}
	next := cloneOperationMap(j.operations)
	next[id] = cloneOperation(current)
	if err := j.persist(next); err != nil {
		return Operation{}, false, err
	}
	j.operations = next
	return cloneOperation(current), true, nil
}

func (j *Journal) load() error {
	var envelope operationEnvelope
	if err := decodeStrictFile(j.path, &envelope); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open lifecycle journal: %w", err)
	}
	if envelope.SchemaVersion != OperationSchemaVersion {
		return fmt.Errorf("%w: operation envelope version %d", ErrUnsupportedVersion, envelope.SchemaVersion)
	}
	if len(envelope.Operations) > MaxOperations {
		return fmt.Errorf("%w: too many lifecycle operations", ErrInvalidRecord)
	}
	operations := make(map[string]Operation, len(envelope.Operations))
	idempotency := make(map[string]string)
	for _, operation := range envelope.Operations {
		if err := operation.Validate(); err != nil {
			return fmt.Errorf("load lifecycle operation %q: %w", operation.ID, err)
		}
		if _, exists := operations[operation.ID]; exists {
			return fmt.Errorf("%w: duplicate lifecycle operation id %q", ErrInvalidRecord, operation.ID)
		}
		if operation.IdempotencyKey != "" {
			if _, exists := idempotency[operation.IdempotencyKey]; exists {
				return fmt.Errorf("%w: duplicate lifecycle idempotency key", ErrInvalidRecord)
			}
			idempotency[operation.IdempotencyKey] = operation.ID
		}
		operations[operation.ID] = cloneOperation(operation)
	}
	j.operations = operations
	j.idempotency = idempotency
	return nil
}

func (j *Journal) persist(operations map[string]Operation) error {
	items := make([]Operation, 0, len(operations))
	for _, operation := range operations {
		items = append(items, cloneOperation(operation))
	}
	sort.Slice(items, func(i, k int) bool { return items[i].ID < items[k].ID })
	data, err := encodeIndented(operationEnvelope{SchemaVersion: OperationSchemaVersion, Operations: items})
	if err != nil {
		return err
	}
	if err := j.write(j.path, data); err != nil {
		return fmt.Errorf("persist lifecycle journal: %w", err)
	}
	return nil
}

func sameOperationRequest(left, right Operation, requireID bool) bool {
	if requireID && left.ID != right.ID {
		return false
	}
	return left.Type == right.Type &&
		reflect.DeepEqual(left.ProfileIDs, normalizeIdentifiers(right.ProfileIDs)) &&
		left.IdempotencyKey == right.IdempotencyKey && left.PredecessorID == right.PredecessorID
}

func cloneOperationMap(source map[string]Operation) map[string]Operation {
	result := make(map[string]Operation, len(source))
	for key, operation := range source {
		result[key] = cloneOperation(operation)
	}
	return result
}

func cloneOperation(operation Operation) Operation {
	operation.ProfileIDs = append([]string(nil), operation.ProfileIDs...)
	operation.Limitations = append([]string(nil), operation.Limitations...)
	operation.RecoveryActions = append([]string(nil), operation.RecoveryActions...)
	operation.Items = append([]OperationItemResult(nil), operation.Items...)
	for index := range operation.Items {
		operation.Items[index].Limitations = append([]string(nil), operation.Items[index].Limitations...)
	}
	return operation
}
