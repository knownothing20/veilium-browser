package desktop

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

func (s *Service) beginPortableOperation(operationType lifecycle.OperationType, profileIDs []string, key, safeStage string) (lifecycle.Operation, bool, error) {
	return s.beginPortableOperationWithID(localRecoveryID(string(operationType)+"-op", key), operationType, profileIDs, key, safeStage)
}

func (s *Service) beginPortableOperationWithID(operationID string, operationType lifecycle.OperationType, profileIDs []string, key, safeStage string) (lifecycle.Operation, bool, error) {
	if s.lifecycleCoordinator == nil || s.lifecycleJournal == nil {
		return lifecycle.Operation{}, false, fmt.Errorf("lifecycle operation service is unavailable")
	}
	operation := lifecycle.NewOperation(operationID, operationType, profileIDs, time.Now().UTC())
	operation.IdempotencyKey = localRecoveryID(string(operationType)+"-idempotency", key)
	operation.ApplicationVersion = AppVersion
	operation.Platform = runtime.GOOS + "/" + runtime.GOARCH
	operation.SafeCancellationStage = safeStage
	return s.lifecycleCoordinator.Begin(operation)
}

func (s *Service) finishPortableOperation(operation lifecycle.Operation, itemID, outputID string) error {
	now := time.Now().UTC()
	startedAt := operation.StartedAt.UTC()
	_, err := s.lifecycleCoordinator.Finish(operation.ID, lifecycle.OperationCompleted, []lifecycle.OperationItemResult{{
		ItemID: itemID, Status: lifecycle.ItemSucceeded, StartedAt: &startedAt, CompletedAt: &now,
		CompletedStage: "committed", OutputID: outputID,
	}}, nil, nil)
	return err
}

func (s *Service) failPortableOperation(operation lifecycle.Operation, itemID, reasonCode string, cause error) error {
	now := time.Now().UTC()
	startedAt := operation.StartedAt.UTC()
	_, finishErr := s.lifecycleCoordinator.Finish(operation.ID, lifecycle.OperationFailed, []lifecycle.OperationItemResult{{
		ItemID: itemID, Status: lifecycle.ItemFailed, StartedAt: &startedAt, CompletedAt: &now,
		CompletedStage: "failed", ReasonCode: reasonCode,
	}}, nil, nil)
	if finishErr != nil {
		return fmt.Errorf("%v; finalize portable operation: %w", cause, finishErr)
	}
	return cause
}

func (s *Service) reusedPortableExport(request PortableExportRequest, path string, operation lifecycle.Operation) (PortableExportResult, error) {
	if operation.Status != lifecycle.OperationCompleted || len(operation.Items) != 1 || operation.Items[0].Status != lifecycle.ItemSucceeded {
		return PortableExportResult{}, fmt.Errorf("portable export operation %q is %q", operation.ID, operation.Status)
	}
	artifact, err := portableprofile.Read(path)
	if err != nil {
		return PortableExportResult{}, fmt.Errorf("read idempotent portable export: %w", err)
	}
	profile, err := s.store.Get(strings.TrimSpace(request.ProfileID))
	if err != nil {
		return PortableExportResult{}, err
	}
	if !strings.EqualFold(artifact.PayloadSHA256, operation.Items[0].OutputID) {
		return PortableExportResult{}, fmt.Errorf("idempotent portable export identity changed")
	}
	return portableExportResult(path, profile, artifact), nil
}

func (s *Service) reusedPortableTemplate(operation lifecycle.Operation) (portableprofile.Template, error) {
	if operation.Status != lifecycle.OperationCompleted || len(operation.Items) != 1 || operation.Items[0].Status != lifecycle.ItemSucceeded {
		return portableprofile.Template{}, fmt.Errorf("portable template operation %q is %q", operation.ID, operation.Status)
	}
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return portableprofile.Template{}, err
	}
	for _, item := range catalog.Templates {
		if item.ID == operation.Items[0].OutputID {
			return item, nil
		}
	}
	return portableprofile.Template{}, fmt.Errorf("idempotent portable template result is missing")
}

func (s *Service) reusedPortableImport(operation lifecycle.Operation, mode portableprofile.IdentityMode) (PortableImportResult, error) {
	if operation.Status != lifecycle.OperationCompleted || len(operation.Items) != 1 || operation.Items[0].Status != lifecycle.ItemSucceeded {
		return PortableImportResult{}, fmt.Errorf("portable Profile operation %q is %q", operation.ID, operation.Status)
	}
	profile, err := s.store.Get(operation.Items[0].OutputID)
	if err != nil {
		return PortableImportResult{}, fmt.Errorf("resolve idempotent portable Profile: %w", err)
	}
	return PortableImportResult{
		Profile:      profile,
		IdentityMode: mode,
		Warnings:     []string{"This request reused the previously committed portable Profile result."},
	}, nil
}

func (s *Service) rollbackPortableProfile(profileID, action string, cause error) error {
	if profileErr := s.store.Delete(profileID); profileErr != nil {
		return fmt.Errorf("%s: %v; rollback profile metadata: %w", action, cause, profileErr)
	}
	if lifecycleErr := s.rollbackLifecycleRecord(profileID); lifecycleErr != nil {
		return fmt.Errorf("%s: %v; profile metadata rolled back but lifecycle cleanup requires reconciliation: %w", action, cause, lifecycleErr)
	}
	return cause
}

func portableExportResult(path string, profile domain.Profile, artifact portableprofile.Artifact) PortableExportResult {
	return PortableExportResult{
		Path:          path,
		ProfileID:     profile.ID,
		ProfileName:   profile.Name,
		IdentityMode:  artifact.Payload.IdentityMode,
		PayloadSHA256: artifact.PayloadSHA256,
		ExportedAt:    artifact.ExportedAt,
		Exclusions:    append([]string(nil), artifact.Exclusions...),
		Limitations:   append([]string(nil), artifact.Limitations...),
	}
}

func portableDestination(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path != "." && path != "" && !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	return path
}

func portableOperationKey(explicit string, values ...string) string {
	values = append([]string{strings.TrimSpace(explicit)}, values...)
	return localRecoveryID("portable-request", values...)
}
