package desktop

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

type PortableExportRequest struct {
	ProfileID    string                       `json:"profileId"`
	Destination  string                       `json:"destination"`
	IdentityMode portableprofile.IdentityMode `json:"identityMode"`
}

type PortableExportResult struct {
	Path          string                       `json:"path"`
	ProfileID     string                       `json:"profileId"`
	ProfileName   string                       `json:"profileName"`
	IdentityMode  portableprofile.IdentityMode `json:"identityMode"`
	PayloadSHA256 string                       `json:"payloadSha256"`
	ExportedAt    time.Time                    `json:"exportedAt"`
	Exclusions    []string                     `json:"exclusions"`
	Limitations   []string                     `json:"limitations"`
}

type PortableDependencyOption struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type PortableImportPreview struct {
	Path               string                          `json:"path"`
	Artifact           portableprofile.Artifact        `json:"artifact"`
	KernelMatches      []PortableDependencyOption      `json:"kernelMatches"`
	AdapterMatches     []PortableDependencyOption      `json:"adapterMatches"`
	CredentialRequired bool                            `json:"credentialRequired"`
	Warnings           []string                        `json:"warnings"`
	Ready              bool                            `json:"ready"`
}

type PortableImportRequest struct {
	Path         string                       `json:"path"`
	Name         string                       `json:"name"`
	IdentityMode portableprofile.IdentityMode `json:"identityMode"`
	KernelID     string                       `json:"kernelId"`
	AdapterID    string                       `json:"adapterId"`
	CredentialID string                       `json:"credentialId"`
}

type PortableImportResult struct {
	Profile      domain.Profile                `json:"profile"`
	IdentityMode portableprofile.IdentityMode `json:"identityMode"`
	Warnings     []string                      `json:"warnings"`
}

type PortableTemplateCreateRequest struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
}

type PortableTemplateApplyRequest struct {
	TemplateID  string `json:"templateId"`
	Name        string `json:"name"`
	KernelID    string `json:"kernelId"`
	AdapterID   string `json:"adapterId"`
	CredentialID string `json:"credentialId"`
}

func (s *Service) ExportPortableProfile(request PortableExportRequest) (PortableExportResult, error) {
	artifact, profile, err := s.buildPortableArtifact(request.ProfileID, request.IdentityMode)
	if err != nil {
		return PortableExportResult{}, err
	}
	if err := portableprofile.Write(request.Destination, artifact); err != nil {
		return PortableExportResult{}, err
	}
	path := strings.TrimSpace(request.Destination)
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	return PortableExportResult{
		Path:          path,
		ProfileID:     profile.ID,
		ProfileName:   profile.Name,
		IdentityMode:  artifact.Payload.IdentityMode,
		PayloadSHA256: artifact.PayloadSHA256,
		ExportedAt:    artifact.ExportedAt,
		Exclusions:    append([]string(nil), artifact.Exclusions...),
		Limitations:   append([]string(nil), artifact.Limitations...),
	}, nil
}

func (s *Service) PreviewPortableImport(path string) (PortableImportPreview, error) {
	artifact, err := portableprofile.Read(path)
	if err != nil {
		return PortableImportPreview{}, err
	}
	preview := PortableImportPreview{
		Path:               strings.TrimSpace(path),
		Artifact:           artifact,
		CredentialRequired: artifact.Payload.CredentialRequired,
		Warnings:           append([]string(nil), artifact.Limitations...),
	}
	preview.KernelMatches = s.matchingKernels(artifact.Payload.Kernel)
	if artifact.Payload.Adapter != nil {
		preview.AdapterMatches = s.matchingAdapters(*artifact.Payload.Adapter)
	}
	preview.Ready = len(preview.KernelMatches) > 0 && (artifact.Payload.Adapter == nil || len(preview.AdapterMatches) > 0)
	if len(preview.KernelMatches) == 0 {
		preview.Warnings = append(preview.Warnings, "No currently verified local Kernel matches the exported identity.")
	}
	if artifact.Payload.Adapter != nil && len(preview.AdapterMatches) == 0 {
		preview.Warnings = append(preview.Warnings, "No currently verified local proxy adapter matches the exported identity.")
	}
	if preview.CredentialRequired && len(s.credentials.List()) == 0 {
		preview.Warnings = append(preview.Warnings, "This route requires a credential selected from the local operating-system vault.")
	}
	if artifact.Payload.IdentityMode == portableprofile.IdentityPreserve {
		preview.Warnings = append(preview.Warnings, "Preserved identity material must not be used simultaneously on another device or Profile.")
	}
	return preview, nil
}

func (s *Service) ImportPortableProfile(request PortableImportRequest) (PortableImportResult, error) {
	artifact, err := portableprofile.Read(request.Path)
	if err != nil {
		return PortableImportResult{}, err
	}
	mode := request.IdentityMode
	if mode == "" {
		mode = portableprofile.IdentityNew
	}
	profileInput, warnings, err := s.profileFromPortablePayload(artifact.Payload, request.Name, mode, request.KernelID, request.AdapterID, request.CredentialID)
	if err != nil {
		return PortableImportResult{}, err
	}
	created, err := s.CreateProfile(profileInput)
	if err != nil {
		return PortableImportResult{}, fmt.Errorf("create imported Profile: %w", err)
	}
	return PortableImportResult{Profile: created, IdentityMode: mode, Warnings: warnings}, nil
}

func (s *Service) ListPortableTemplates() ([]portableprofile.Template, error) {
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return nil, err
	}
	return append([]portableprofile.Template(nil), catalog.Templates...), nil
}

func (s *Service) CreatePortableTemplate(request PortableTemplateCreateRequest) (portableprofile.Template, error) {
	artifact, _, err := s.buildPortableArtifact(request.ProfileID, portableprofile.IdentityNew)
	if err != nil {
		return portableprofile.Template{}, err
	}
	template, err := portableprofile.NewTemplate(request.Name, artifact.Payload, time.Now().UTC())
	if err != nil {
		return portableprofile.Template{}, err
	}
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return portableprofile.Template{}, err
	}
	if len(catalog.Templates) >= portableprofile.MaxTemplates {
		return portableprofile.Template{}, fmt.Errorf("template catalog is full")
	}
	for _, existing := range catalog.Templates {
		if strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(template.Name)) {
			return portableprofile.Template{}, fmt.Errorf("template name %q already exists", template.Name)
		}
	}
	catalog.Templates = append(catalog.Templates, template)
	if err := portableprofile.SaveTemplates(s.portableTemplatePath(), catalog); err != nil {
		return portableprofile.Template{}, err
	}
	return template, nil
}

func (s *Service) DeletePortableTemplate(templateID string) error {
	templateID = strings.TrimSpace(templateID)
	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return err
	}
	next := make([]portableprofile.Template, 0, len(catalog.Templates))
	found := false
	for _, item := range catalog.Templates {
		if item.ID == templateID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return fmt.Errorf("portable template %q was not found", templateID)
	}
	catalog.Templates = next
	return portableprofile.SaveTemplates(s.portableTemplatePath(), catalog)
}

func (s *Service) ApplyPortableTemplate(request PortableTemplateApplyRequest) (PortableImportResult, error) {
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
	profileInput, warnings, err := s.profileFromPortablePayload(selected.Payload, request.Name, portableprofile.IdentityNew, request.KernelID, request.AdapterID, request.CredentialID)
	if err != nil {
		return PortableImportResult{}, err
	}
	created, err := s.CreateProfile(profileInput)
	if err != nil {
		return PortableImportResult{}, fmt.Errorf("create Profile from template: %w", err)
	}
	warnings = append(warnings, "Template application created a new Profile ID, managed directory, and fingerprint seed.")
	return PortableImportResult{Profile: created, IdentityMode: portableprofile.IdentityNew, Warnings: warnings}, nil
}

func (s *Service) buildPortableArtifact(profileID string, mode portableprofile.IdentityMode) (portableprofile.Artifact, domain.Profile, error) {
	profileID = strings.TrimSpace(profileID)
	if s.supervisor.IsActive(profileID) {
		return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("Profile cannot be exported while its browser is running")
	}
	if _, err := s.requireLifecycleAvailable(profileID, "be exported"); err != nil {
		return portableprofile.Artifact{}, domain.Profile{}, err
	}
	item, err := s.store.Get(profileID)
	if err != nil {
		return portableprofile.Artifact{}, domain.Profile{}, err
	}
	if err := s.validateProxy(item); err != nil {
		return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("validate portable route: %w", err)
	}
	kernelRecord, err := s.kernels.Verify(strings.TrimSpace(item.Kernel.ID))
	if err != nil {
		return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("verify export Kernel: %w", err)
	}
	if kernelRecord.Status != kernel.StatusVerified {
		return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("export Kernel is not verified")
	}
	var adapterRequirement *portableprofile.AdapterRequirement
	if strings.TrimSpace(item.Proxy.AdapterRef) != "" {
		adapterRecord, verifyErr := s.adapters.Verify(item.Proxy.AdapterRef)
		if verifyErr != nil {
			return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("verify export adapter: %w", verifyErr)
		}
		if adapterRecord.Status != adapter.StatusVerified {
			return portableprofile.Artifact{}, domain.Profile{}, fmt.Errorf("export adapter is not verified")
		}
		adapterRequirement = &portableprofile.AdapterRequirement{Kind: adapterRecord.Kind, Version: adapterRecord.Version, SHA256: adapterRecord.SHA256, SizeBytes: adapterRecord.SizeBytes}
	}
	artifact, err := portableprofile.Build(portableprofile.BuildInput{
		ApplicationVersion: AppVersion,
		Profile:            item,
		Kernel: portableprofile.KernelRequirement{
			Provider: kernelRecord.Provider, Version: kernelRecord.Version, SHA256: kernelRecord.SHA256, SizeBytes: kernelRecord.SizeBytes,
		},
		Adapter:            adapterRequirement,
		CredentialRequired: strings.TrimSpace(item.Proxy.CredentialRef) != "",
		IdentityMode:       mode,
		ExportedAt:         time.Now().UTC(),
	})
	if err != nil {
		return portableprofile.Artifact{}, domain.Profile{}, err
	}
	return artifact, item, nil
}

func (s *Service) profileFromPortablePayload(payload portableprofile.Payload, name string, mode portableprofile.IdentityMode, kernelID, adapterID, credentialID string) (domain.Profile, []string, error) {
	kernelRecord, err := s.kernels.Verify(strings.TrimSpace(kernelID))
	if err != nil {
		return domain.Profile{}, nil, fmt.Errorf("verify selected Kernel: %w", err)
	}
	if !kernelMatches(kernelRecord, payload.Kernel) {
		return domain.Profile{}, nil, fmt.Errorf("selected Kernel does not match the portable requirement")
	}
	adapterID = strings.TrimSpace(adapterID)
	credentialID = strings.TrimSpace(credentialID)
	if payload.Adapter != nil {
		adapterRecord, verifyErr := s.adapters.Verify(adapterID)
		if verifyErr != nil {
			return domain.Profile{}, nil, fmt.Errorf("verify selected adapter: %w", verifyErr)
		}
		if !adapterMatches(adapterRecord, *payload.Adapter) {
			return domain.Profile{}, nil, fmt.Errorf("selected adapter does not match the portable requirement")
		}
	} else if adapterID != "" {
		return domain.Profile{}, nil, fmt.Errorf("portable definition does not require an adapter")
	}
	if payload.CredentialRequired {
		if credentialID == "" {
			return domain.Profile{}, nil, fmt.Errorf("a local credential selection is required")
		}
		if _, err := s.credentials.Get(credentialID); err != nil {
			return domain.Profile{}, nil, fmt.Errorf("resolve selected credential: %w", err)
		}
	} else if credentialID != "" {
		return domain.Profile{}, nil, fmt.Errorf("portable definition does not require a credential")
	}
	fingerprint := payload.Fingerprint
	warnings := []string{"Imported metadata does not inherit health, compatibility, Provider trust, or Evidence."}
	switch mode {
	case "", portableprofile.IdentityNew:
		mode = portableprofile.IdentityNew
		seed, seedErr := portableprofile.NewSeed()
		if seedErr != nil {
			return domain.Profile{}, nil, seedErr
		}
		fingerprint.Seed = seed
	case portableprofile.IdentityPreserve:
		if strings.TrimSpace(payload.Fingerprint.Seed) == "" {
			return domain.Profile{}, nil, fmt.Errorf("portable definition does not contain identity material to preserve")
		}
		warnings = append(warnings, "Do not use this preserved identity simultaneously on multiple devices or Profiles.")
	default:
		return domain.Profile{}, nil, fmt.Errorf("unsupported identity transfer mode %q", mode)
	}
	profileName := strings.TrimSpace(name)
	if profileName == "" {
		profileName = payload.Name
	}
	item := domain.Profile{
		Name:        profileName,
		Group:       payload.Group,
		Notes:       payload.Notes,
		Tags:        append([]string(nil), payload.Tags...),
		Kernel:      domain.KernelRef{ID: kernelRecord.ID, Provider: kernelRecord.Provider, Version: kernelRecord.Version},
		Fingerprint: fingerprint,
		Proxy: domain.ProxyConfig{
			URL:           payload.ProxyURL,
			CredentialRef: credentialID,
			AdapterRef:    adapterID,
		},
	}
	return item, warnings, nil
}

func (s *Service) matchingKernels(requirement portableprofile.KernelRequirement) []PortableDependencyOption {
	result := []PortableDependencyOption{}
	for _, record := range s.kernels.List() {
		verified, err := s.kernels.Verify(record.ID)
		if err == nil && kernelMatches(verified, requirement) {
			result = append(result, PortableDependencyOption{ID: verified.ID, Name: verified.Name, Kind: verified.Provider, Version: verified.Version, SHA256: verified.SHA256})
		}
	}
	return result
}

func (s *Service) matchingAdapters(requirement portableprofile.AdapterRequirement) []PortableDependencyOption {
	result := []PortableDependencyOption{}
	for _, record := range s.adapters.List() {
		verified, err := s.adapters.Verify(record.ID)
		if err == nil && adapterMatches(verified, requirement) {
			result = append(result, PortableDependencyOption{ID: verified.ID, Name: verified.Name, Kind: verified.Kind, Version: verified.Version, SHA256: verified.SHA256})
		}
	}
	return result
}

func kernelMatches(record kernel.Record, requirement portableprofile.KernelRequirement) bool {
	return record.Status == kernel.StatusVerified && record.Provider == requirement.Provider && record.Version == requirement.Version && strings.EqualFold(record.SHA256, requirement.SHA256) && record.SizeBytes == requirement.SizeBytes
}

func adapterMatches(record adapter.Record, requirement portableprofile.AdapterRequirement) bool {
	return record.Status == adapter.StatusVerified && record.Kind == requirement.Kind && record.Version == requirement.Version && strings.EqualFold(record.SHA256, requirement.SHA256) && record.SizeBytes == requirement.SizeBytes
}

func (s *Service) portableTemplatePath() string {
	return filepath.Join(s.dataRoot, "portable-templates.json")
}
