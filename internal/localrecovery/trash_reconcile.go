package localrecovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type TrashReconciliationFinding struct {
	TrashID      string `json:"trashId"`
	ProfileID    string `json:"profileId"`
	Status       string `json:"status"`
	SourceState  string `json:"sourceState"`
	TrashState   string `json:"trashState"`
	ProfileState string `json:"profileState"`
	StagingState string `json:"stagingState,omitempty"`
	ReasonCode   string `json:"reasonCode"`
}

type TrashReconciliationReport struct {
	GeneratedAt time.Time                    `json:"generatedAt"`
	Findings    []TrashReconciliationFinding `json:"findings,omitempty"`
	Limitations []string                     `json:"limitations,omitempty"`
}

type TrashReconciler struct {
	dataRoot     string
	recoveryRoot string
	records      trashRecordStore
	journal      trashJournal
	profiles     trashProfileStore
	trash        trashCatalog
	now          func() time.Time
}

func OpenTrashReconciler(dataRoot string, records *lifecycle.RecordStore, journal *lifecycle.Journal, profiles *profile.Store, trash *TrashStore) (*TrashReconciler, error) {
	if records == nil || journal == nil || profiles == nil || trash == nil {
		return nil, fmt.Errorf("trash reconciliation requires lifecycle records, Profile metadata, journal, and trash catalog")
	}
	absolute, recoveryRoot, err := prepareRecoveryRoots(dataRoot)
	if err != nil {
		return nil, err
	}
	return &TrashReconciler{
		dataRoot:     absolute,
		recoveryRoot: recoveryRoot,
		records:      records,
		journal:      journal,
		profiles:     profiles,
		trash:        trash,
		now:          func() time.Time { return time.Now().UTC() },
	}, nil
}

func (r *TrashReconciler) Reconcile(ctx context.Context) (TrashReconciliationReport, error) {
	if r == nil || r.records == nil || r.journal == nil || r.profiles == nil || r.trash == nil {
		return TrashReconciliationReport{}, fmt.Errorf("trash reconciler is unavailable")
	}
	report := TrashReconciliationReport{GeneratedAt: r.now().UTC()}
	stagingByProfile := r.operationStagingByProfile()
	for _, trashRecord := range r.trash.List() {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}
		lifecycleRecord, lifecycleErr := r.records.Get(trashRecord.ProfileID)
		sourcePath := filepath.Join(r.dataRoot, filepath.FromSlash(trashRecord.OriginalManagedDir))
		trashPath := trashRootPath(r.recoveryRoot, trashRecord.TrashID)
		sourceState := inspectReconciliationPath(sourcePath)
		trashState := inspectReconciliationPath(trashPath)
		profileState := r.inspectRetainedProfile(trashRecord)
		stagingState := "absent"
		for _, relative := range stagingByProfile[trashRecord.ProfileID] {
			candidate := filepath.Join(r.dataRoot, filepath.FromSlash(relative))
			state := inspectReconciliationPath(candidate)
			if state != "absent" {
				stagingState = state
				break
			}
		}
		reason := trashReconciliationReason(trashRecord, lifecycleRecord, lifecycleErr, sourceState, trashState, profileState, stagingState)
		if reason == "" {
			continue
		}
		finding := TrashReconciliationFinding{
			TrashID:      trashRecord.TrashID,
			ProfileID:    trashRecord.ProfileID,
			Status:       string(trashRecord.Status),
			SourceState:  sourceState,
			TrashState:   trashState,
			ProfileState: profileState,
			StagingState: stagingState,
			ReasonCode:   reason,
		}
		report.Findings = append(report.Findings, finding)
		if trashRecord.Status != TrashRecoveryRequired {
			trashRecord.Status = TrashRecoveryRequired
			trashRecord.Limitations = sortedUnique(append(trashRecord.Limitations, reason))
			if _, err := r.trash.Update(trashRecord); err != nil {
				return report, fmt.Errorf("mark trash record %q recovery-required: %w", trashRecord.TrashID, err)
			}
		}
		if _, _, err := r.records.AddRecoveryCode(trashRecord.ProfileID, reason); err != nil && lifecycleErr == nil {
			return report, fmt.Errorf("persist trash recovery code for Profile %q: %w", trashRecord.ProfileID, err)
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		left := report.Findings[i]
		right := report.Findings[j]
		return left.ProfileID+"\x00"+left.TrashID+"\x00"+left.ReasonCode < right.ProfileID+"\x00"+right.TrashID+"\x00"+right.ReasonCode
	})
	if len(report.Findings) > 0 {
		report.Limitations = []string{"manual-recovery-required", "no-authority-guessed"}
	}
	return report, nil
}

func (r *TrashReconciler) inspectRetainedProfile(record TrashRecord) string {
	item, err := r.profiles.Get(record.ProfileID)
	if profileAlreadyDeleted(err) {
		return "absent"
	}
	if err != nil {
		return "uninspectable"
	}
	_, digest, err := profileDefinitionForTrash(item)
	if err != nil {
		return "invalid"
	}
	if digest != record.ProfileDefinitionDigest {
		return "changed"
	}
	return "present-matching"
}

func (r *TrashReconciler) operationStagingByProfile() map[string][]string {
	result := make(map[string][]string)
	for _, operation := range r.journal.List() {
		if operation.Status.Terminal() && operation.Status != lifecycle.OperationRecoveryRequired {
			continue
		}
		switch operation.Type {
		case lifecycle.OperationTrash, lifecycle.OperationRestoreTrash, lifecycle.OperationPermanentDelete:
		default:
			continue
		}
		for _, profileID := range operation.ProfileIDs {
			if operation.StagingRef != "" {
				result[profileID] = append(result[profileID], operation.StagingRef)
			}
			if operation.QuarantineRef != "" {
				result[profileID] = append(result[profileID], operation.QuarantineRef)
			}
		}
	}
	for profileID := range result {
		sort.Strings(result[profileID])
		result[profileID] = uniqueStrings(result[profileID])
	}
	return result
}

func inspectReconciliationPath(candidate string) string {
	info, err := os.Lstat(candidate)
	if errors.Is(err, os.ErrNotExist) {
		return "absent"
	}
	if err != nil {
		return "uninspectable"
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "unsafe-link"
	}
	unsafe, err := pathHasReparsePoint(candidate)
	if err != nil {
		return "uninspectable"
	}
	if unsafe {
		return "unsafe-reparse"
	}
	if !info.IsDir() {
		return "unsafe-nondirectory"
	}
	return "present"
}

func trashReconciliationReason(trashRecord TrashRecord, lifecycleRecord lifecycle.Record, lifecycleErr error, sourceState, trashState, profileState, stagingState string) string {
	if lifecycleErr != nil {
		return "trash-lifecycle-record-missing"
	}
	if lifecycleRecord.Lock != nil {
		return "trash-stale-operation-lock"
	}
	if stagingState != "absent" {
		return "trash-operation-staging-present"
	}
	switch trashRecord.Status {
	case TrashStored:
		if lifecycleRecord.State != lifecycle.StateTrashed || sourceState != "absent" || trashState != "present" || profileState != "present-matching" {
			return "trash-stored-state-contradictory"
		}
	case TrashDeleted:
		if lifecycleRecord.State != lifecycle.StateInvalid || sourceState != "absent" || trashState != "absent" || profileState != "absent" || trashRecord.DataPresent {
			return "trash-deleted-state-contradictory"
		}
	case TrashPending, TrashRestoring, TrashCleanupPending:
		return "trash-operation-interrupted"
	case TrashRecoveryRequired:
		return "trash-recovery-still-required"
	default:
		return "trash-status-unsupported"
	}
	return ""
}
