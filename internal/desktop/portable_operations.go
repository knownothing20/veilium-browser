package desktop

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

func (s *Service) ExportPortableProfile(request PortableExportRequest) (PortableExportResult, error) {
	profileID := strings.TrimSpace(request.ProfileID)
	path := portableDestination(request.Destination)
	preflight, err := s.store.Get(profileID)
	if err != nil {
		return PortableExportResult{}, err
	}
	key := portableOperationKey(request.IdempotencyKey, profileID, preflight.UpdatedAt.Format(time.RFC3339Nano), path, string(request.IdentityMode))
	started, reused, err := s.beginPortableOperation(lifecycle.OperationExportDefinition, []string{profileID}, key, "portable-export-preflight")
	if err != nil {
		return PortableExportResult{}, err
	}
	if reused {
		return s.reusedPortableExport(request, path, started)
	}

	artifact, profile, err := s.buildPortableArtifactForOperation(profileID, request.IdentityMode, started.ID)
	if err != nil {
		return PortableExportResult{}, s.failPortableOperation(started, profileID, "portable-export-preflight-failed", err)
	}
	if !profile.UpdatedAt.Equal(preflight.UpdatedAt) {
		return PortableExportResult{}, s.failPortableOperation(started, profileID, "portable-export-source-changed", lifecycle.ErrConflict)
	}
	if err := portableprofile.Write(path, artifact); err != nil {
		return PortableExportResult{}, s.failPortableOperation(started, profileID, "portable-export-publish-failed", err)
	}
	if err := s.finishPortableOperation(started, profileID, artifact.PayloadSHA256); err != nil {
		return PortableExportResult{}, err
	}
	return portableExportResult(path, profile, artifact), nil
}

func (s *Service) ImportPortableProfile(request PortableImportRequest) (PortableImportResult, error) {
	if s.lifecycleJournal == nil {
		return PortableImportResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}
	artifact, err := portableprofile.Read(request.Path)
	if err != nil {
		return PortableImportResult{}, err
	}
	mode := request.IdentityMode
	if mode == "" {
		mode = portableprofile.IdentityNew
	}
	key := portableOperationKey(request.IdempotencyKey, artifact.PayloadSHA256, strings.TrimSpace(request.Name), string(mode), strings.TrimSpace(request.KernelID), strings.TrimSpace(request.AdapterID), strings.TrimSpace(request.CredentialID))
	operationID := localRecoveryID(string(lifecycle.OperationImportDefinition)+"-op", key)
	if existing, lookupErr := s.lifecycleJournal.Get(operationID); lookupErr == nil {
		return s.reusedPortableImport(existing, mode)
	} else if !errors.Is(lookupErr, lifecycle.ErrNotFound) {
		return PortableImportResult{}, lookupErr
	}

	profileInput, warnings, err := s.profileFromPortablePayload(artifact.Payload, request.Name, mode, request.KernelID, request.AdapterID, request.CredentialID)
	if err != nil {
		return PortableImportResult{}, err
	}
	created, err := s.CreateProfile(profileInput)
	if err != nil {
		return PortableImportResult{}, fmt.Errorf("create imported Profile: %w", err)
	}
	started, reused, err := s.beginPortableOperationWithID(operationID, lifecycle.OperationImportDefinition, []string{created.ID}, key, "portable-import-commit")
	if err != nil {
		return PortableImportResult{}, s.rollbackPortableProfile(created.ID, "begin import operation", err)
	}
	if reused {
		if rollbackErr := s.rollbackPortableProfile(created.ID, "discard duplicate imported Profile", nil); rollbackErr != nil {
			return PortableImportResult{}, rollbackErr
		}
		return s.reusedPortableImport(started, mode)
	}
	if err := s.finishPortableOperation(started, created.ID, created.ID); err != nil {
		return PortableImportResult{}, err
	}
	return PortableImportResult{Profile: created, IdentityMode: mode, Warnings: warnings}, nil
}

func (s *Service) CreatePortableTemplate(request PortableTemplateCreateRequest) (portableprofile.Template, error) {
	profileID := strings.TrimSpace(request.ProfileID)
	preflight, err := s.store.Get(profileID)
	if err != nil {
		return portableprofile.Template{}, err
	}
	key := portableOperationKey(request.IdempotencyKey, profileID, preflight.UpdatedAt.Format(time.RFC3339Nano), strings.ToLower(strings.TrimSpace(request.Name)))
	started, reused, err := s.beginPortableOperation(lifecycle.OperationCreateTemplate, []string{profileID}, key, "portable-template-preflight")
	if err != nil {
		return portableprofile.Template{}, err
	}
	if reused {
		return s.reusedPortableTemplate(started)
	}
	artifact, profile, err := s.buildPortableArtifactForOperation(profileID, portableprofile.IdentityNew, started.ID)
	if err != nil {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-preflight-failed", err)
	}
	if !profile.UpdatedAt.Equal(preflight.UpdatedAt) {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-source-changed", lifecycle.ErrConflict)
	}
	template, err := portableprofile.NewTemplate(request.Name, artifact.Payload, time.Now().UTC())
	if err != nil {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-invalid", err)
	}
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-catalog-read-failed", err)
	}
	if len(catalog.Templates) >= portableprofile.MaxTemplates {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-catalog-full", fmt.Errorf("template catalog is full"))
	}
	for _, existing := range catalog.Templates {
		if strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(template.Name)) {
			return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-name-conflict", fmt.Errorf("template name %q already exists", template.Name))
		}
	}
	catalog.Templates = append(catalog.Templates, template)
	if err := portableprofile.SaveTemplates(s.portableTemplatePath(), catalog); err != nil {
		return portableprofile.Template{}, s.failPortableOperation(started, profileID, "portable-template-persist-failed", err)
	}
	if err := s.finishPortableOperation(started, profileID, template.ID); err != nil {
		return portableprofile.Template{}, err
	}
	return template, nil
}

func (s *Service) ApplyPortableTemplate(request PortableTemplateApplyRequest) (PortableImportResult, error) {
	if s.lifecycleJournal == nil {
		return PortableImportResult{}, fmt.Errorf("lifecycle operation service is unavailable")
	}
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return PortableImportResult{}, err
	}
	var selected *portableprofile.Template
	for index := range catalog.Templates {
		if catalog.Templates[index].ID == strings.TrimSpace(request.TemplateID) {
			copy := catalog.Templates[index]
			selected = &copy
			break
		}
	}
	if selected == nil {
		return PortableImportResult{}, fmt.Errorf("portable template %q was not found", request.TemplateID)
	}
	key := portableOperationKey(request.IdempotencyKey, selected.ID, selected.UpdatedAt.Format(time.RFC3339Nano), strings.TrimSpace(request.Name), strings.TrimSpace(request.KernelID), strings.TrimSpace(request.AdapterID), strings.TrimSpace(request.CredentialID))
	operationID := localRecoveryID(string(lifecycle.OperationApplyTemplate)+"-op", key)
	if existing, lookupErr := s.lifecycleJournal.Get(operationID); lookupErr == nil {
		return s.reusedPortableImport(existing, portableprofile.IdentityNew)
	} else if !errors.Is(lookupErr, lifecycle.ErrNotFound) {
		return PortableImportResult{}, lookupErr
	}
	profileInput, warnings, err := s.profileFromPortablePayload(selected.Payload, request.Name, portableprofile.IdentityNew, request.KernelID, request.AdapterID, request.CredentialID)
	if err != nil {
		return PortableImportResult{}, err
	}
	created, err := s.CreateProfile(profileInput)
	if err != nil {
		return PortableImportResult{}, fmt.Errorf("create Profile from template: %w", err)
	}
	started, reused, err := s.beginPortableOperationWithID(operationID, lifecycle.OperationApplyTemplate, []string{created.ID}, key, "portable-template-apply-commit")
	if err != nil {
		return PortableImportResult{}, s.rollbackPortableProfile(created.ID, "begin template operation", err)
	}
	if reused {
		if rollbackErr := s.rollbackPortableProfile(created.ID, "discard duplicate template Profile", nil); rollbackErr != nil {
			return PortableImportResult{}, rollbackErr
		}
		return s.reusedPortableImport(started, portableprofile.IdentityNew)
	}
	if err := s.finishPortableOperation(started, created.ID, created.ID); err != nil {
		return PortableImportResult{}, err
	}
	warnings = append(warnings, "Template application created a new Profile ID, managed directory, and fingerprint seed.")
	return PortableImportResult{Profile: created, IdentityMode: portableprofile.IdentityNew, Warnings: warnings}, nil
}
