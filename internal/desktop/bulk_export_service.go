package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

type BulkPortableExportRequest struct {
	ProfileIDs           []string                     `json:"profileIds"`
	DestinationDirectory string                       `json:"destinationDirectory"`
	IdentityMode         portableprofile.IdentityMode `json:"identityMode"`
	IdempotencyKey       string                       `json:"idempotencyKey,omitempty"`
}

type BulkPortableExportResult struct {
	Operation            lifecycle.Operation    `json:"operation"`
	DestinationDirectory string                 `json:"destinationDirectory"`
	Exports              []PortableExportResult `json:"exports"`
}

type bulkPortableExportPreflight struct {
	profile   domain.Profile
	path      string
	updatedAt time.Time
}

func (s *Service) BulkExportPortableProfiles(request BulkPortableExportRequest) (BulkPortableExportResult, error) {
	profileIDs, err := normalizeBulkProfileIDs(request.ProfileIDs)
	if err != nil {
		return BulkPortableExportResult{}, err
	}
	directory, err := inspectBulkExportDirectory(request.DestinationDirectory)
	if err != nil {
		return BulkPortableExportResult{}, err
	}
	mode, err := normalizeBulkPortableIdentityMode(request.IdentityMode)
	if err != nil {
		return BulkPortableExportResult{}, err
	}
	if s.lifecycleCoordinator == nil || s.lifecycleJournal == nil || s.lifecycleRecords == nil {
		return BulkPortableExportResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}

	preflight := make(map[string]bulkPortableExportPreflight, len(profileIDs))
	keyParts := []string{directory, string(mode)}
	for _, profileID := range profileIDs {
		if err := s.requirePortableSource(profileID, ""); err != nil {
			return BulkPortableExportResult{}, err
		}
		item, getErr := s.store.Get(profileID)
		if getErr != nil {
			return BulkPortableExportResult{}, getErr
		}
		path, pathErr := bulkPortableExportPath(directory, item.Name, profileID)
		if pathErr != nil {
			return BulkPortableExportResult{}, pathErr
		}
		preflight[profileID] = bulkPortableExportPreflight{
			profile: item, path: path, updatedAt: item.UpdatedAt.UTC(),
		}
		keyParts = append(keyParts, profileID, item.UpdatedAt.UTC().Format(time.RFC3339Nano))
	}

	key := portableOperationKey(request.IdempotencyKey, keyParts...)
	operationID := localRecoveryID(string(lifecycle.OperationExportDefinition)+"-op", key)
	if existing, lookupErr := s.lifecycleJournal.Get(operationID); lookupErr == nil {
		return s.reusedBulkPortableExport(profileIDs, directory, mode, preflight, existing)
	} else if !errors.Is(lookupErr, lifecycle.ErrNotFound) {
		return BulkPortableExportResult{}, lookupErr
	}
	for _, profileID := range profileIDs {
		if err := ensureBulkPortableTargetAvailable(preflight[profileID].path); err != nil {
			return BulkPortableExportResult{}, err
		}
	}

	started, reused, err := s.beginPortableOperationWithID(
		operationID,
		lifecycle.OperationExportDefinition,
		profileIDs,
		key,
		"between-profiles",
	)
	if err != nil {
		return BulkPortableExportResult{}, err
	}
	if reused {
		return s.reusedBulkPortableExport(profileIDs, directory, mode, preflight, started)
	}

	items := make([]lifecycle.OperationItemResult, 0, len(profileIDs))
	exports := make([]PortableExportResult, 0, len(profileIDs))
	cancelRemaining := false
	for _, profileID := range profileIDs {
		itemStarted := time.Now().UTC()
		if cancelRemaining || s.bulkCancellationRequested(started.ID) {
			cancelRemaining = true
			completed := time.Now().UTC()
			items = append(items, lifecycle.OperationItemResult{
				ItemID: profileID, Status: lifecycle.ItemCancelled, StartedAt: &itemStarted, CompletedAt: &completed,
				CompletedStage: "not-started", ReasonCode: "bulk-cancellation-requested",
			})
			continue
		}

		current, getErr := s.store.Get(profileID)
		if getErr != nil {
			items = append(items, failedBulkExportItem(profileID, itemStarted, "profile-read-failed"))
			continue
		}
		if !current.UpdatedAt.UTC().Equal(preflight[profileID].updatedAt) {
			items = append(items, skippedBulkExportItem(profileID, itemStarted, "profile-changed-after-preflight"))
			continue
		}
		record, recordErr := s.lifecycleRecords.Get(profileID)
		if recordErr != nil || record.Lock == nil || record.Lock.OperationID != started.ID {
			items = append(items, skippedBulkExportItem(profileID, itemStarted, "operation-lock-lost"))
			continue
		}
		artifact, profile, buildErr := s.buildPortableArtifactForOperation(profileID, mode, started.ID)
		if buildErr != nil {
			items = append(items, failedBulkExportItem(profileID, itemStarted, "portable-export-preflight-failed"))
			continue
		}
		if !profile.UpdatedAt.UTC().Equal(preflight[profileID].updatedAt) {
			items = append(items, skippedBulkExportItem(profileID, itemStarted, "profile-changed-after-preflight"))
			continue
		}
		path := preflight[profileID].path
		if writeErr := portableprofile.Write(path, artifact); writeErr != nil {
			items = append(items, failedBulkExportItem(profileID, itemStarted, "portable-export-publish-failed"))
			continue
		}
		completed := time.Now().UTC()
		items = append(items, lifecycle.OperationItemResult{
			ItemID: profileID, Status: lifecycle.ItemSucceeded, StartedAt: &itemStarted, CompletedAt: &completed,
			CompletedStage: "portable-export-published", OutputID: artifact.PayloadSHA256,
		})
		exports = append(exports, portableExportResult(path, profile, artifact))
	}

	status := bulkOperationStatus(items)
	limitations := []string(nil)
	if status == lifecycle.OperationPartial {
		limitations = append(limitations, "bulk-portable-export-partial-result")
	}
	if mode == portableprofile.IdentityPreserve {
		limitations = append(limitations, "preserved-identity-simultaneous-use-warning")
	}
	finished, finishErr := s.lifecycleCoordinator.Finish(started.ID, status, items, limitations, nil)
	if finishErr != nil {
		return BulkPortableExportResult{}, finishErr
	}
	sort.Slice(exports, func(i, j int) bool { return exports[i].ProfileID < exports[j].ProfileID })
	return BulkPortableExportResult{
		Operation: finished, DestinationDirectory: directory, Exports: exports,
	}, nil
}

func (s *Service) reusedBulkPortableExport(
	profileIDs []string,
	directory string,
	mode portableprofile.IdentityMode,
	preflight map[string]bulkPortableExportPreflight,
	operation lifecycle.Operation,
) (BulkPortableExportResult, error) {
	if operation.Type != lifecycle.OperationExportDefinition || !operation.Status.Terminal() {
		return BulkPortableExportResult{}, fmt.Errorf("bulk portable export operation %q is %q", operation.ID, operation.Status)
	}
	if !equalStringSlices(operation.ProfileIDs, profileIDs) {
		return BulkPortableExportResult{}, lifecycle.ErrConflict
	}
	exports := make([]PortableExportResult, 0, len(operation.Items))
	for _, result := range operation.Items {
		if result.Status != lifecycle.ItemSucceeded {
			continue
		}
		entry, exists := preflight[result.ItemID]
		if !exists {
			return BulkPortableExportResult{}, fmt.Errorf("idempotent bulk portable export contains an unexpected Profile")
		}
		artifact, err := portableprofile.Read(entry.path)
		if err != nil {
			return BulkPortableExportResult{}, fmt.Errorf("read idempotent bulk portable export: %w", err)
		}
		if artifact.Payload.IdentityMode != mode || !strings.EqualFold(artifact.PayloadSHA256, result.OutputID) {
			return BulkPortableExportResult{}, fmt.Errorf("idempotent bulk portable export identity changed")
		}
		exports = append(exports, portableExportResult(entry.path, entry.profile, artifact))
	}
	sort.Slice(exports, func(i, j int) bool { return exports[i].ProfileID < exports[j].ProfileID })
	return BulkPortableExportResult{
		Operation: operation, DestinationDirectory: directory, Exports: exports,
	}, nil
}

func inspectBulkExportDirectory(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("bulk portable export directory is required")
	}
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", fmt.Errorf("resolve bulk portable export directory: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect bulk portable export directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("bulk portable export destination must be an existing regular directory")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve bulk portable export directory links: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve bulk portable export directory: %w", err)
	}
	if !sameBulkExportPath(absolute, resolved) {
		return "", fmt.Errorf("bulk portable export directory cannot pass through a link or reparse alias")
	}
	return filepath.Clean(absolute), nil
}

func bulkPortableExportPath(directory, profileName, profileID string) (string, error) {
	filename := safeBulkPortableFilename(profileName, profileID)
	target := filepath.Join(directory, filename)
	relative, err := filepath.Rel(directory, target)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("bulk portable export target escaped the selected directory")
	}
	return target, nil
}

func safeBulkPortableFilename(profileName, profileID string) string {
	name := strings.TrimSpace(profileName)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00', '\r', '\n', '\t':
			return '-'
		default:
			return r
		}
	}, name)
	name = strings.Trim(strings.TrimSpace(name), ".")
	if name == "" {
		name = "veilium-profile"
	}
	runes := []rune(name)
	if len(runes) > 72 {
		name = string(runes[:72])
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if isWindowsReservedFilename(base) {
		name = "_" + name
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(profileID)))
	suffix := hex.EncodeToString(sum[:6])
	return name + "-" + suffix + ".veilium-profile.json"
}

func isWindowsReservedFilename(value string) bool {
	switch value {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(value) == 4 && (strings.HasPrefix(value, "COM") || strings.HasPrefix(value, "LPT")) {
		return value[3] >= '1' && value[3] <= '9'
	}
	return false
}

func ensureBulkPortableTargetAvailable(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect bulk portable export target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("bulk portable export target must not be a link or special entry")
	}
	return fmt.Errorf("bulk portable export target already exists: %s", filepath.Base(path))
}

func normalizeBulkPortableIdentityMode(mode portableprofile.IdentityMode) (portableprofile.IdentityMode, error) {
	if mode == "" {
		return portableprofile.IdentityNew, nil
	}
	switch mode {
	case portableprofile.IdentityNew, portableprofile.IdentityPreserve:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported identity transfer mode %q", mode)
	}
}

func sameBulkExportPath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func failedBulkExportItem(profileID string, started time.Time, reason string) lifecycle.OperationItemResult {
	completed := time.Now().UTC()
	return lifecycle.OperationItemResult{
		ItemID: profileID, Status: lifecycle.ItemFailed, StartedAt: &started, CompletedAt: &completed,
		CompletedStage: "portable-export", ReasonCode: reason,
	}
}

func skippedBulkExportItem(profileID string, started time.Time, reason string) lifecycle.OperationItemResult {
	completed := time.Now().UTC()
	return lifecycle.OperationItemResult{
		ItemID: profileID, Status: lifecycle.ItemSkipped, StartedAt: &started, CompletedAt: &completed,
		CompletedStage: "portable-export-preflight", ReasonCode: reason,
	}
}
