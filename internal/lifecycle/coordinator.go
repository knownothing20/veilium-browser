package lifecycle

import (
	"fmt"
	"time"
)

type ProfileBlocker func(profileID string) (reason string, blocked bool)

type Coordinator struct {
	records          *RecordStore
	journal          *Journal
	activeBlocker    ProfileBlocker
	dependentBlocker ProfileBlocker
	now              func() time.Time
}

func NewCoordinator(records *RecordStore, journal *Journal, activeBlocker, dependentBlocker ProfileBlocker) (*Coordinator, error) {
	if records == nil {
		return nil, fmt.Errorf("lifecycle record store is required")
	}
	if journal == nil {
		return nil, fmt.Errorf("lifecycle journal is required")
	}
	return &Coordinator{
		records:          records,
		journal:          journal,
		activeBlocker:    activeBlocker,
		dependentBlocker: dependentBlocker,
		now:              func() time.Time { return time.Now().UTC() },
	}, nil
}

func (c *Coordinator) Begin(input Operation) (Operation, bool, error) {
	for _, profileID := range normalizeIdentifiers(input.ProfileIDs) {
		if _, err := c.records.Get(profileID); err != nil {
			return Operation{}, false, err
		}
		if reason, blocked := checkBlocker(c.activeBlocker, profileID); blocked {
			return Operation{}, false, fmt.Errorf("%w: profile %q has an active browser session: %s", ErrConflict, profileID, reason)
		}
		if reason, blocked := checkBlocker(c.dependentBlocker, profileID); blocked {
			return Operation{}, false, fmt.Errorf("%w: profile %q has a protected dependent operation: %s", ErrConflict, profileID, reason)
		}
	}

	operation, reused, err := c.journal.Create(input)
	if err != nil {
		return Operation{}, false, err
	}
	if operation.Status.Terminal() {
		return operation, reused, nil
	}
	if _, _, err := c.records.AcquireLocks(operation.ID, operation.ProfileIDs, c.now()); err != nil {
		return Operation{}, false, err
	}
	if operation.Status == OperationPending {
		operation.Status = OperationRunning
		operation.Stage = "locked"
		updated, updateErr := c.journal.Update(operation)
		if updateErr != nil {
			_, _, _ = c.records.ReleaseLocks(operation.ID, operation.ProfileIDs)
			return Operation{}, false, updateErr
		}
		operation = updated
	}
	return operation, reused, nil
}

func (c *Coordinator) RequestCancellation(operationID string) (Operation, bool, error) {
	return c.journal.RequestCancellation(operationID)
}

func (c *Coordinator) Finish(operationID string, status OperationStatus, items []OperationItemResult, limitations, recoveryActions []string) (Operation, error) {
	operation, err := c.journal.Get(operationID)
	if err != nil {
		return Operation{}, err
	}
	if !status.Terminal() {
		return Operation{}, fmt.Errorf("%w: finish requires terminal status", ErrInvalidRecord)
	}
	now := c.now().UTC()
	operation.Status = status
	operation.Stage = "finished"
	operation.CompletedAt = &now
	operation.Items = items
	operation.Limitations = append([]string(nil), limitations...)
	operation.RecoveryActions = append([]string(nil), recoveryActions...)
	updated, err := c.journal.Update(operation)
	if err != nil {
		return Operation{}, err
	}
	if _, _, err := c.records.ReleaseLocks(updated.ID, updated.ProfileIDs); err != nil {
		return updated, fmt.Errorf("release lifecycle locks: %w", err)
	}
	return updated, nil
}

func checkBlocker(blocker ProfileBlocker, profileID string) (string, bool) {
	if blocker == nil {
		return "", false
	}
	return blocker(profileID)
}
