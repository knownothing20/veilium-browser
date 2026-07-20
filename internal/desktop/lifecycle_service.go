package desktop

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

const lifecycleStartupTimeout = 5 * time.Second

func (s *Service) initializeLifecycle() error {
	records, err := lifecycle.OpenRecordStore(filepath.Join(s.dataRoot, "lifecycle.json"))
	if err != nil {
		return fmt.Errorf("open lifecycle records: %w", err)
	}
	journal, err := lifecycle.OpenJournal(filepath.Join(s.dataRoot, "lifecycle-operations.json"))
	if err != nil {
		return fmt.Errorf("open lifecycle operation journal: %w", err)
	}
	scanner, err := lifecycle.NewInventoryScanner(s.dataRoot)
	if err != nil {
		return err
	}
	reconciler, err := lifecycle.NewReconciler(records, journal, scanner)
	if err != nil {
		return err
	}

	inputs := make([]lifecycle.CompatibilityInput, 0, len(s.store.List()))
	for _, item := range s.store.List() {
		inputs = append(inputs, s.compatibilityInput(item))
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleStartupTimeout)
	defer cancel()
	report, err := reconciler.Reconcile(ctx, inputs)
	if err != nil {
		return fmt.Errorf("reconcile lifecycle state: %w", err)
	}

	coordinator, err := lifecycle.NewCoordinator(records, journal, func(profileID string) (string, bool) {
		if s.supervisor.IsActive(profileID) {
			return "browser-session-active", true
		}
		return "", false
	}, nil)
	if err != nil {
		return err
	}

	s.lifecycleRecords = records
	s.lifecycleJournal = journal
	s.lifecycleCoordinator = coordinator
	s.lifecycleScanner = scanner
	s.lifecycleReconciliation = report
	return nil
}

func (s *Service) compatibilityInput(item domain.Profile) lifecycle.CompatibilityInput {
	relative := managedLifecycleDir(item.ID)
	input := lifecycle.CompatibilityInput{
		ProfileID:  strings.TrimSpace(item.ID),
		ManagedDir: relative,
		State:      lifecycle.StateAvailable,
	}
	expected := filepath.Join(s.profilesDir, item.ID)
	if strings.TrimSpace(item.Name) == "" {
		input.State = lifecycle.StateInvalid
		input.LimitationCodes = append(input.LimitationCodes, "legacy-profile-name-missing")
	}
	if strings.TrimSpace(item.UserDataDir) == "" || !sameCleanPath(item.UserDataDir, expected) {
		input.State = lifecycle.StateInvalid
		input.LimitationCodes = append(input.LimitationCodes, "legacy-user-data-unmanaged")
	}
	return input
}

func managedLifecycleDir(profileID string) string {
	return filepath.ToSlash(filepath.Join("profiles", strings.TrimSpace(profileID)))
}

func (s *Service) createLifecycleRecord(item domain.Profile) (lifecycle.Record, error) {
	if s.lifecycleRecords == nil {
		return lifecycle.Record{}, fmt.Errorf("lifecycle service is unavailable")
	}
	expected := filepath.Join(s.profilesDir, item.ID)
	if !sameCleanPath(item.UserDataDir, expected) {
		return lifecycle.Record{}, fmt.Errorf("profile %q does not use its managed lifecycle directory", item.ID)
	}
	return s.lifecycleRecords.Create(lifecycle.Record{
		ProfileID:  item.ID,
		State:      lifecycle.StateAvailable,
		ManagedDir: managedLifecycleDir(item.ID),
	})
}

func (s *Service) rollbackLifecycleRecord(profileID string) error {
	if s.lifecycleRecords == nil {
		return nil
	}
	_, err := s.lifecycleRecords.RemoveRecord(profileID)
	if errors.Is(err, lifecycle.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) requireLifecycleAvailable(profileID, action string) (lifecycle.Record, error) {
	if s.lifecycleRecords == nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot %s because lifecycle state is unavailable", profileID, action)
	}
	record, err := s.lifecycleRecords.Get(profileID)
	if err != nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot %s because lifecycle metadata is missing: %w", profileID, action, err)
	}
	if record.State != lifecycle.StateAvailable {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot %s while lifecycle state is %q", profileID, action, record.State)
	}
	if record.Lock != nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot %s while lifecycle operation %q is active", profileID, action, record.Lock.OperationID)
	}
	return record, nil
}

func (s *Service) requireLifecycleMutable(profileID string) (lifecycle.Record, error) {
	if s.lifecycleRecords == nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot be edited because lifecycle state is unavailable", profileID)
	}
	record, err := s.lifecycleRecords.Get(profileID)
	if err != nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot be edited because lifecycle metadata is missing: %w", profileID, err)
	}
	if record.Lock != nil {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot be edited while lifecycle operation %q is active", profileID, record.Lock.OperationID)
	}
	if record.State == lifecycle.StateArchived || record.State == lifecycle.StateTrashed {
		return lifecycle.Record{}, fmt.Errorf("profile %q cannot be edited while lifecycle state is %q", profileID, record.State)
	}
	return record, nil
}

func (s *Service) ListLifecycleRecords() []lifecycle.Record {
	if s.lifecycleRecords == nil {
		return nil
	}
	return s.lifecycleRecords.List()
}

func (s *Service) ListLifecycleOperations() []lifecycle.Operation {
	if s.lifecycleJournal == nil {
		return nil
	}
	return s.lifecycleJournal.List()
}

func (s *Service) LifecycleReconciliation() lifecycle.ReconciliationReport {
	return cloneLifecycleReport(s.lifecycleReconciliation)
}

func (s *Service) ScanLifecycleStorage(ctx context.Context) (lifecycle.StorageInventory, error) {
	if s.lifecycleScanner == nil || s.lifecycleRecords == nil {
		return lifecycle.StorageInventory{}, fmt.Errorf("lifecycle inventory is unavailable")
	}
	return s.lifecycleScanner.Scan(ctx, s.lifecycleRecords.List())
}

func cloneLifecycleReport(input lifecycle.ReconciliationReport) lifecycle.ReconciliationReport {
	result := input
	result.CompatibilityCreated = append([]string(nil), input.CompatibilityCreated...)
	result.Actions = append([]lifecycle.ReconciliationAction(nil), input.Actions...)
	result.Limitations = append([]string(nil), input.Limitations...)
	result.Inventory.Profiles = append([]lifecycle.ProfileStorage(nil), input.Inventory.Profiles...)
	for index := range result.Inventory.Profiles {
		result.Inventory.Profiles[index].Limitations = append([]string(nil), input.Inventory.Profiles[index].Limitations...)
	}
	result.Inventory.Orphans = append([]lifecycle.InventoryFinding(nil), input.Inventory.Orphans...)
	result.Inventory.Unsafe = append([]lifecycle.InventoryFinding(nil), input.Inventory.Unsafe...)
	result.Inventory.Limitations = append([]string(nil), input.Inventory.Limitations...)
	return result
}
