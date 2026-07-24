package desktop

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

const (
	maxBulkMetadataTags = 64
	maxBulkTagLength    = 64
	maxBulkGroupLength  = 128
)

type BulkMetadataUpdateRequest struct {
	ProfileIDs     []string `json:"profileIds"`
	SetGroup       bool     `json:"setGroup"`
	Group          string   `json:"group,omitempty"`
	AddTags        []string `json:"addTags,omitempty"`
	RemoveTags     []string `json:"removeTags,omitempty"`
	IdempotencyKey string   `json:"idempotencyKey,omitempty"`
}

type BulkMetadataUpdateResult struct {
	Operation lifecycle.Operation `json:"operation"`
	Profiles  []domain.Profile    `json:"profiles"`
}

type StorageManagementState struct {
	Inventory      lifecycle.StorageInventory `json:"inventory"`
	SnapshotCount  int                        `json:"snapshotCount"`
	SnapshotBytes  int64                      `json:"snapshotBytes"`
	TrashCount     int                        `json:"trashCount"`
	TrashBytes     int64                      `json:"trashBytes"`
	OperationCount int                        `json:"operationCount"`
	GeneratedAt    time.Time                  `json:"generatedAt"`
	Limitations    []string                   `json:"limitations,omitempty"`
}

type bulkMetadataMutation struct {
	setGroup   bool
	group      string
	addTags    []string
	removeTags []string
}

func (s *Service) BulkUpdateProfileMetadata(request BulkMetadataUpdateRequest) (BulkMetadataUpdateResult, error) {
	profileIDs, err := normalizeBulkProfileIDs(request.ProfileIDs)
	if err != nil {
		return BulkMetadataUpdateResult{}, err
	}
	mutation, err := normalizeBulkMetadataMutation(request)
	if err != nil {
		return BulkMetadataUpdateResult{}, err
	}
	if s.lifecycleCoordinator == nil || s.lifecycleJournal == nil || s.lifecycleRecords == nil {
		return BulkMetadataUpdateResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}

	preflight := make(map[string]time.Time, len(profileIDs))
	prepared := make(map[string]domain.Profile, len(profileIDs))
	keyParts := []string{strings.TrimSpace(request.IdempotencyKey), fmt.Sprintf("set-group=%t", mutation.setGroup), mutation.group, strings.Join(mutation.addTags, ","), strings.Join(mutation.removeTags, ",")}
	for _, profileID := range profileIDs {
		item, getErr := s.store.Get(profileID)
		if getErr != nil {
			return BulkMetadataUpdateResult{}, getErr
		}
		record, recordErr := s.requireLifecycleMutable(profileID)
		if recordErr != nil {
			return BulkMetadataUpdateResult{}, recordErr
		}
		if record.State != lifecycle.StateAvailable && record.State != lifecycle.StateDraft {
			return BulkMetadataUpdateResult{}, fmt.Errorf("profile %q cannot receive a bulk metadata update while lifecycle state is %q", profileID, record.State)
		}
		next, mutationErr := checkedBulkMetadataMutation(item, mutation)
		if mutationErr != nil {
			return BulkMetadataUpdateResult{}, fmt.Errorf("profile %q bulk metadata result is invalid: %w", profileID, mutationErr)
		}
		preflight[profileID] = item.UpdatedAt.UTC()
		prepared[profileID] = next
		keyParts = append(keyParts, profileID, item.UpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	key := localRecoveryID("bulk-metadata-request", keyParts...)
	operationID := localRecoveryID(string(lifecycle.OperationBulkMetadataUpdate)+"-op", key)
	if existing, lookupErr := s.lifecycleJournal.Get(operationID); lookupErr == nil {
		return s.reusedBulkMetadataResult(existing)
	} else if !errors.Is(lookupErr, lifecycle.ErrNotFound) {
		return BulkMetadataUpdateResult{}, lookupErr
	}

	operation := lifecycle.NewOperation(operationID, lifecycle.OperationBulkMetadataUpdate, profileIDs, time.Now().UTC())
	operation.IdempotencyKey = localRecoveryID("bulk-metadata-idempotency", key)
	operation.ApplicationVersion = AppVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.SafeCancellationStage = "between-profiles"
	started, reused, err := s.lifecycleCoordinator.Begin(operation)
	if err != nil {
		return BulkMetadataUpdateResult{}, err
	}
	if reused {
		return s.reusedBulkMetadataResult(started)
	}

	items := make([]lifecycle.OperationItemResult, 0, len(profileIDs))
	updatedProfiles := make([]domain.Profile, 0, len(profileIDs))
	cancelRemaining := false
	for _, profileID := range profileIDs {
		now := time.Now().UTC()
		if cancelRemaining || s.bulkCancellationRequested(started.ID) {
			cancelRemaining = true
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemCancelled, StartedAt: &now, CompletedAt: &now,
				CompletedStage: "not-started", ReasonCode: "bulk-cancellation-requested",
			})
			continue
		}

		itemStarted := now
		current, getErr := s.store.Get(profileID)
		if getErr != nil {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemFailed, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "metadata-preflight", ReasonCode: "profile-read-failed",
			})
			continue
		}
		if !current.UpdatedAt.UTC().Equal(preflight[profileID]) {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemSkipped, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "metadata-preflight", ReasonCode: "profile-changed-after-preflight",
			})
			continue
		}
		record, recordErr := s.lifecycleRecords.Get(profileID)
		if recordErr != nil || record.Lock == nil || record.Lock.OperationID != started.ID {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemSkipped, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "metadata-preflight", ReasonCode: "operation-lock-lost",
			})
			continue
		}

		next, preparedOK := prepared[profileID]
		if !preparedOK {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemFailed, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "metadata-preflight", ReasonCode: "metadata-result-unavailable",
			})
			continue
		}
		updated, updateErr := s.store.Update(next)
		completed := time.Now().UTC()
		if updateErr != nil {
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemFailed, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "metadata-persist", ReasonCode: "profile-update-failed",
			})
			continue
		}
		updatedProfiles = append(updatedProfiles, updated)
		items = append(items, lifecycle.OperationItemResult{
			ItemID: profileID, Status: lifecycle.ItemSucceeded, StartedAt: &itemStarted, CompletedAt: &completed,
			CompletedStage: "metadata-committed", OutputID: profileID,
		})
	}

	status := bulkOperationStatus(items)
	limitations := []string(nil)
	if status == lifecycle.OperationPartial {
		limitations = append(limitations, "bulk-metadata-partial-result")
	}
	finished, finishErr := s.lifecycleCoordinator.Finish(started.ID, status, items, limitations, nil)
	if finishErr != nil {
		return BulkMetadataUpdateResult{}, finishErr
	}
	sort.Slice(updatedProfiles, func(i, j int) bool { return updatedProfiles[i].ID < updatedProfiles[j].ID })
	return BulkMetadataUpdateResult{Operation: finished, Profiles: updatedProfiles}, nil
}

func (s *Service) RefreshStorageManagement(ctx context.Context) (StorageManagementState, error) {
	inventory, err := s.ScanLifecycleStorage(ctx)
	if err != nil {
		return StorageManagementState{}, err
	}
	state := StorageManagementState{
		Inventory:      inventory,
		OperationCount: len(s.ListLifecycleOperations()),
		GeneratedAt:    time.Now().UTC(),
		Limitations: []string{
			"Storage inventory is read-only and never deletes or repairs orphaned data automatically.",
			"Browser data is counted as opaque files and is not inspected.",
		},
	}
	if recovery, recoveryErr := s.LocalRecoveryState(); recoveryErr == nil {
		for _, item := range recovery.Snapshots {
			if item.Status == "verified" {
				state.SnapshotCount++
				state.SnapshotBytes += item.TotalBytes
			}
		}
		for _, item := range recovery.Trash {
			if item.Status != "deleted" {
				state.TrashCount++
				state.TrashBytes += item.TotalBytes
			}
		}
	} else {
		state.Limitations = append(state.Limitations, "Local recovery catalog totals are unavailable: "+recoveryErr.Error())
	}
	return state, nil
}

func (s *Service) reusedBulkMetadataResult(operation lifecycle.Operation) (BulkMetadataUpdateResult, error) {
	if !operation.Status.Terminal() {
		return BulkMetadataUpdateResult{}, fmt.Errorf("bulk metadata operation %q is still %q", operation.ID, operation.Status)
	}
	profiles := make([]domain.Profile, 0, len(operation.Items))
	for _, result := range operation.Items {
		if result.Status != lifecycle.ItemSucceeded || strings.TrimSpace(result.OutputID) == "" {
			continue
		}
		item, err := s.store.Get(result.OutputID)
		if err != nil {
			return BulkMetadataUpdateResult{}, fmt.Errorf("resolve idempotent bulk metadata result: %w", err)
		}
		profiles = append(profiles, item)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ID < profiles[j].ID })
	return BulkMetadataUpdateResult{Operation: operation, Profiles: profiles}, nil
}

func (s *Service) bulkCancellationRequested(operationID string) bool {
	operation, err := s.lifecycleJournal.Get(operationID)
	return err == nil && operation.CancellationRequested
}

func normalizeBulkProfileIDs(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > lifecycle.MaxIdentifierLength {
			return nil, fmt.Errorf("profile id is too long")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 || len(result) > lifecycle.MaxProfilesPerOp {
		return nil, fmt.Errorf("select between 1 and %d Profiles", lifecycle.MaxProfilesPerOp)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeBulkMetadataMutation(request BulkMetadataUpdateRequest) (bulkMetadataMutation, error) {
	group := strings.TrimSpace(request.Group)
	if len(group) > maxBulkGroupLength {
		return bulkMetadataMutation{}, fmt.Errorf("group must be at most %d characters", maxBulkGroupLength)
	}
	addTags, err := normalizeBulkTags(request.AddTags)
	if err != nil {
		return bulkMetadataMutation{}, err
	}
	removeTags, err := normalizeBulkTags(request.RemoveTags)
	if err != nil {
		return bulkMetadataMutation{}, err
	}
	removeSet := make(map[string]struct{}, len(removeTags))
	for _, tag := range removeTags {
		removeSet[strings.ToLower(tag)] = struct{}{}
	}
	for _, tag := range addTags {
		if _, conflict := removeSet[strings.ToLower(tag)]; conflict {
			return bulkMetadataMutation{}, fmt.Errorf("tag %q cannot be added and removed in the same request", tag)
		}
	}
	if !request.SetGroup && len(addTags) == 0 && len(removeTags) == 0 {
		return bulkMetadataMutation{}, fmt.Errorf("bulk metadata request does not contain a change")
	}
	return bulkMetadataMutation{setGroup: request.SetGroup, group: group, addTags: addTags, removeTags: removeTags}, nil
}

func normalizeBulkTags(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > maxBulkTagLength {
			return nil, fmt.Errorf("tag %q exceeds %d characters", value, maxBulkTagLength)
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	if len(result) > maxBulkMetadataTags {
		return nil, fmt.Errorf("too many tags; maximum is %d", maxBulkMetadataTags)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i]) < strings.ToLower(result[j]) })
	return result, nil
}

func applyBulkMetadataMutation(input domain.Profile, mutation bulkMetadataMutation) domain.Profile {
	result, err := checkedBulkMetadataMutation(input, mutation)
	if err != nil {
		return input
	}
	return result
}

func checkedBulkMetadataMutation(input domain.Profile, mutation bulkMetadataMutation) (domain.Profile, error) {
	result := input
	if mutation.setGroup {
		result.Group = mutation.group
	}
	removed := make(map[string]struct{}, len(mutation.removeTags))
	for _, tag := range mutation.removeTags {
		removed[strings.ToLower(tag)] = struct{}{}
	}
	tags := make([]string, 0, len(result.Tags)+len(mutation.addTags))
	for _, tag := range result.Tags {
		if _, remove := removed[strings.ToLower(strings.TrimSpace(tag))]; !remove {
			tags = append(tags, tag)
		}
	}
	tags = append(tags, mutation.addTags...)
	normalized, err := normalizeBulkTags(tags)
	if err != nil {
		return domain.Profile{}, err
	}
	result.Tags = normalized
	return result, nil
}

func bulkOperationStatus(items []lifecycle.OperationItemResult) lifecycle.OperationStatus {
	succeeded := 0
	cancelled := 0
	failedOrSkipped := 0
	for _, item := range items {
		switch item.Status {
		case lifecycle.ItemSucceeded:
			succeeded++
		case lifecycle.ItemCancelled:
			cancelled++
		default:
			failedOrSkipped++
		}
	}
	switch {
	case succeeded == len(items):
		return lifecycle.OperationCompleted
	case succeeded > 0:
		return lifecycle.OperationPartial
	case cancelled == len(items):
		return lifecycle.OperationCancelled
	case cancelled > 0 && failedOrSkipped == 0:
		return lifecycle.OperationCancelled
	default:
		return lifecycle.OperationFailed
	}
}
