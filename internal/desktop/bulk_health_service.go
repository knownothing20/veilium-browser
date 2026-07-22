package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

type HealthCheckStatus string

const (
	HealthCheckPass    HealthCheckStatus = "pass"
	HealthCheckWarning HealthCheckStatus = "warning"
	HealthCheckFail    HealthCheckStatus = "fail"
)

type ProfileHealthStatus string

const (
	ProfileHealthReady   ProfileHealthStatus = "ready"
	ProfileHealthLimited ProfileHealthStatus = "limited"
	ProfileHealthBlocked ProfileHealthStatus = "blocked"
)

type BulkHealthRefreshRequest struct {
	ProfileIDs     []string `json:"profileIds"`
	IdempotencyKey string   `json:"idempotencyKey,omitempty"`
}

type ProfileHealthCheck struct {
	ID      string            `json:"id"`
	Status  HealthCheckStatus `json:"status"`
	Message string            `json:"message"`
}

type ProfileHealthReport struct {
	ProfileID      string               `json:"profileId"`
	ProfileName    string               `json:"profileName"`
	LifecycleState lifecycle.State      `json:"lifecycleState"`
	Status         ProfileHealthStatus  `json:"status"`
	Checks         []ProfileHealthCheck `json:"checks"`
	RefreshedAt    time.Time            `json:"refreshedAt"`
}

type BulkHealthRefreshResult struct {
	Operation lifecycle.Operation   `json:"operation"`
	Reports   []ProfileHealthReport `json:"reports"`
}

type bulkHealthPreflight struct {
	updatedAt time.Time
	revision  uint64
}

var bulkHealthCheckIDs = []string{
	"lifecycle",
	"kernel",
	"route",
	"fingerprint",
	"consistency",
	"managed-data",
}

func (s *Service) BulkRefreshProfileHealth(request BulkHealthRefreshRequest) (BulkHealthRefreshResult, error) {
	profileIDs, err := normalizeBulkProfileIDs(request.ProfileIDs)
	if err != nil {
		return BulkHealthRefreshResult{}, err
	}
	if s.lifecycleCoordinator == nil || s.lifecycleJournal == nil || s.lifecycleRecords == nil {
		return BulkHealthRefreshResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}

	preflight := make(map[string]bulkHealthPreflight, len(profileIDs))
	keyParts := []string{strings.TrimSpace(request.IdempotencyKey)}
	for _, profileID := range profileIDs {
		item, getErr := s.store.Get(profileID)
		if getErr != nil {
			return BulkHealthRefreshResult{}, getErr
		}
		record, recordErr := s.requireLifecycleMutable(profileID)
		if recordErr != nil {
			return BulkHealthRefreshResult{}, recordErr
		}
		if record.State != lifecycle.StateAvailable && record.State != lifecycle.StateDraft {
			return BulkHealthRefreshResult{}, fmt.Errorf("profile %q cannot receive a health refresh while lifecycle state is %q", profileID, record.State)
		}
		preflight[profileID] = bulkHealthPreflight{updatedAt: item.UpdatedAt.UTC(), revision: record.Revision}
		keyParts = append(keyParts, profileID)
		if strings.TrimSpace(request.IdempotencyKey) == "" {
			keyParts = append(keyParts, item.UpdatedAt.UTC().Format(time.RFC3339Nano), fmt.Sprintf("lifecycle-revision=%d", record.Revision))
		}
	}

	key := localRecoveryID("bulk-health-request", keyParts...)
	operationID := localRecoveryID(string(lifecycle.OperationBulkHealthRefresh)+"-op", key)
	if existing, lookupErr := s.lifecycleJournal.Get(operationID); lookupErr == nil {
		return s.reusedBulkHealthResult(existing)
	} else if !errors.Is(lookupErr, lifecycle.ErrNotFound) {
		return BulkHealthRefreshResult{}, lookupErr
	}

	operation := lifecycle.NewOperation(operationID, lifecycle.OperationBulkHealthRefresh, profileIDs, time.Now().UTC())
	operation.IdempotencyKey = localRecoveryID("bulk-health-idempotency", key)
	operation.ApplicationVersion = AppVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.SafeCancellationStage = "between-profiles"
	started, reused, err := s.lifecycleCoordinator.Begin(operation)
	if err != nil {
		return BulkHealthRefreshResult{}, err
	}
	if reused {
		return s.reusedBulkHealthResult(started)
	}

	items := make([]lifecycle.OperationItemResult, 0, len(profileIDs))
	reports := make([]ProfileHealthReport, 0, len(profileIDs))
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
				CompletedStage: "health-preflight", ReasonCode: "profile-read-failed",
			})
			continue
		}
		original := preflight[profileID]
		if !current.UpdatedAt.UTC().Equal(original.updatedAt) {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemSkipped, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "health-preflight", ReasonCode: "profile-changed-after-preflight",
			})
			continue
		}
		record, recordErr := s.lifecycleRecords.Get(profileID)
		if recordErr != nil || record.Revision != original.revision+1 || record.Lock == nil || record.Lock.OperationID != started.ID {
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemSkipped, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "health-preflight", ReasonCode: "operation-lock-lost",
			})
			continue
		}

		completed := time.Now().UTC()
		report := s.evaluateProfileHealth(current, record.State, completed)
		reports = append(reports, report)
		items = append(items, lifecycle.OperationItemResult{
			ItemID: profileID, Status: lifecycle.ItemSucceeded, StartedAt: &itemStarted, CompletedAt: &completed,
			CompletedStage: "health-refreshed", ReasonCode: "profile-health-" + string(report.Status),
			OutputID: profileID, Limitations: encodeHealthChecks(report.Checks),
		})
	}

	status := bulkOperationStatus(items)
	limitations := []string{"bulk-health-read-only"}
	if status == lifecycle.OperationPartial {
		limitations = append(limitations, "bulk-health-partial-result")
	}
	finished, finishErr := s.lifecycleCoordinator.Finish(started.ID, status, items, limitations, nil)
	if finishErr != nil {
		return BulkHealthRefreshResult{}, finishErr
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].ProfileID < reports[j].ProfileID })
	return BulkHealthRefreshResult{Operation: finished, Reports: reports}, nil
}

func (s *Service) reusedBulkHealthResult(operation lifecycle.Operation) (BulkHealthRefreshResult, error) {
	if !operation.Status.Terminal() {
		return BulkHealthRefreshResult{}, fmt.Errorf("bulk health operation %q is still %q", operation.ID, operation.Status)
	}
	reports := make([]ProfileHealthReport, 0, len(operation.Items))
	for _, item := range operation.Items {
		if item.Status != lifecycle.ItemSucceeded || strings.TrimSpace(item.OutputID) == "" {
			continue
		}
		profileItem, err := s.store.Get(item.OutputID)
		if err != nil {
			return BulkHealthRefreshResult{}, fmt.Errorf("resolve idempotent bulk health result: %w", err)
		}
		record, err := s.lifecycleRecords.Get(item.OutputID)
		if err != nil {
			return BulkHealthRefreshResult{}, fmt.Errorf("resolve idempotent bulk health lifecycle: %w", err)
		}
		refreshedAt := operation.UpdatedAt.UTC()
		if item.CompletedAt != nil {
			refreshedAt = item.CompletedAt.UTC()
		}
		checks := decodeHealthChecks(item.Limitations)
		if len(checks) == 0 {
			checks = s.evaluateProfileHealth(profileItem, record.State, refreshedAt).Checks
		}
		reports = append(reports, ProfileHealthReport{
			ProfileID: profileItem.ID, ProfileName: profileItem.Name, LifecycleState: record.State,
			Status: healthStatusFromReason(item.ReasonCode, checks), Checks: checks, RefreshedAt: refreshedAt,
		})
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].ProfileID < reports[j].ProfileID })
	return BulkHealthRefreshResult{Operation: operation, Reports: reports}, nil
}

func (s *Service) evaluateProfileHealth(item domain.Profile, state lifecycle.State, refreshedAt time.Time) ProfileHealthReport {
	checks := make([]ProfileHealthCheck, 0, len(bulkHealthCheckIDs))
	if state == lifecycle.StateAvailable {
		checks = append(checks, newHealthCheck("lifecycle", HealthCheckPass))
	} else if state == lifecycle.StateDraft {
		checks = append(checks, newHealthCheck("lifecycle", HealthCheckWarning))
	} else {
		checks = append(checks, newHealthCheck("lifecycle", HealthCheckFail))
	}

	resolved := item
	if strings.TrimSpace(item.Kernel.ID) == "" {
		checks = append(checks, newHealthCheck("kernel", HealthCheckFail))
	} else if err := s.resolveKernel(&resolved); err != nil {
		checks = append(checks, newHealthCheck("kernel", HealthCheckFail))
	} else {
		checks = append(checks, newHealthCheck("kernel", HealthCheckPass))
	}

	if err := s.validateProxy(resolved); err != nil {
		checks = append(checks, newHealthCheck("route", HealthCheckFail))
	} else {
		checks = append(checks, newHealthCheck("route", HealthCheckPass))
	}
	if _, err := fingerprint.Validate(withValidationSeed(resolved)); err != nil {
		checks = append(checks, newHealthCheck("fingerprint", HealthCheckFail))
	} else {
		checks = append(checks, newHealthCheck("fingerprint", HealthCheckPass))
	}
	if err := s.validateProfileConsistency(resolved); err != nil {
		checks = append(checks, newHealthCheck("consistency", HealthCheckFail))
	} else {
		checks = append(checks, newHealthCheck("consistency", HealthCheckPass))
	}
	checks = append(checks, s.managedDataHealthCheck(resolved))

	return ProfileHealthReport{
		ProfileID: item.ID, ProfileName: item.Name, LifecycleState: state,
		Status: deriveProfileHealthStatus(checks), Checks: checks, RefreshedAt: refreshedAt.UTC(),
	}
}

func (s *Service) managedDataHealthCheck(item domain.Profile) ProfileHealthCheck {
	expected := filepath.Join(s.profilesDir, item.ID)
	if !sameCleanPath(item.UserDataDir, expected) {
		return newHealthCheck("managed-data", HealthCheckFail)
	}
	info, err := os.Lstat(expected)
	if errors.Is(err, os.ErrNotExist) {
		return newHealthCheck("managed-data", HealthCheckWarning)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return newHealthCheck("managed-data", HealthCheckFail)
	}
	return newHealthCheck("managed-data", HealthCheckPass)
}

func newHealthCheck(id string, status HealthCheckStatus) ProfileHealthCheck {
	return ProfileHealthCheck{ID: id, Status: status, Message: healthCheckMessage(id, status)}
}

func healthCheckMessage(id string, status HealthCheckStatus) string {
	messages := map[string]map[HealthCheckStatus]string{
		"lifecycle": {
			HealthCheckPass:    "Lifecycle state allows normal validation.",
			HealthCheckWarning: "Draft lifecycle state keeps this Profile limited until current validation is complete.",
			HealthCheckFail:    "Lifecycle state blocks launch and ordinary mutation.",
		},
		"kernel": {
			HealthCheckPass:    "Managed Kernel identity and integrity are verified locally.",
			HealthCheckWarning: "Managed Kernel verification requires attention.",
			HealthCheckFail:    "The selected managed Kernel is missing, modified, or incompatible.",
		},
		"route": {
			HealthCheckPass:    "Route, adapter, and credential references pass current local validation.",
			HealthCheckWarning: "Route validation requires attention.",
			HealthCheckFail:    "The selected route, adapter, or credential does not pass current local validation.",
		},
		"fingerprint": {
			HealthCheckPass:    "Fingerprint configuration passes the current Provider capability contract.",
			HealthCheckWarning: "Fingerprint configuration requires attention.",
			HealthCheckFail:    "Fingerprint configuration does not pass the current Provider capability contract.",
		},
		"consistency": {
			HealthCheckPass:    "Profile identity and window configuration are internally consistent.",
			HealthCheckWarning: "Identity consistency requires attention.",
			HealthCheckFail:    "Profile identity and window configuration are internally inconsistent.",
		},
		"managed-data": {
			HealthCheckPass:    "Managed browser data directory is contained and uses a real directory.",
			HealthCheckWarning: "Managed browser data has not been created yet.",
			HealthCheckFail:    "The Profile user-data path is outside the managed root or is not a real directory.",
		},
	}
	if values, ok := messages[id]; ok {
		if message, ok := values[status]; ok {
			return message
		}
	}
	return "Health check result is unavailable."
}

func deriveProfileHealthStatus(checks []ProfileHealthCheck) ProfileHealthStatus {
	hasWarning := false
	for _, check := range checks {
		if check.Status == HealthCheckFail {
			return ProfileHealthBlocked
		}
		if check.Status == HealthCheckWarning {
			hasWarning = true
		}
	}
	if hasWarning {
		return ProfileHealthLimited
	}
	return ProfileHealthReady
}

func encodeHealthChecks(checks []ProfileHealthCheck) []string {
	result := make([]string, 0, len(checks))
	for _, check := range checks {
		result = append(result, "health-check-"+check.ID+"-"+string(check.Status))
	}
	return result
}

func decodeHealthChecks(codes []string) []ProfileHealthCheck {
	statusByID := make(map[string]HealthCheckStatus, len(codes))
	for _, code := range codes {
		for _, id := range bulkHealthCheckIDs {
			prefix := "health-check-" + id + "-"
			if !strings.HasPrefix(code, prefix) {
				continue
			}
			status := HealthCheckStatus(strings.TrimPrefix(code, prefix))
			if status == HealthCheckPass || status == HealthCheckWarning || status == HealthCheckFail {
				statusByID[id] = status
			}
		}
	}
	checks := make([]ProfileHealthCheck, 0, len(statusByID))
	for _, id := range bulkHealthCheckIDs {
		if status, ok := statusByID[id]; ok {
			checks = append(checks, newHealthCheck(id, status))
		}
	}
	return checks
}

func healthStatusFromReason(reason string, checks []ProfileHealthCheck) ProfileHealthStatus {
	switch strings.TrimPrefix(reason, "profile-health-") {
	case string(ProfileHealthReady):
		return ProfileHealthReady
	case string(ProfileHealthLimited):
		return ProfileHealthLimited
	case string(ProfileHealthBlocked):
		return ProfileHealthBlocked
	default:
		return deriveProfileHealthStatus(checks)
	}
}
