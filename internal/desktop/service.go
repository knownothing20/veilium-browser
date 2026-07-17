package desktop

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

const AppVersion = "0.3.0-dev"

type Service struct {
	store    *profile.Store
	kernels  *kernel.Store
	planner  launch.Planner
	dataRoot string
}

type Bootstrap struct {
	Version   string               `json:"version"`
	Profiles  []domain.Profile     `json:"profiles"`
	Providers []ProviderDescriptor `json:"providers"`
	Kernels   []kernel.Record      `json:"kernels"`
}

type ProviderDescriptor struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Versions    []string                   `json:"versions"`
	Samples     []fingerprint.Capabilities `json:"samples"`
}

type LaunchPlanRequest struct {
	ProfileID           string `json:"profileId"`
	RemoteDebuggingPort int    `json:"remoteDebuggingPort"`
}

func NewService(store *profile.Store, dataRoot string) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("profile store is required")
	}
	if strings.TrimSpace(dataRoot) == "" {
		return nil, fmt.Errorf("data root is required")
	}
	kernels, err := kernel.Open(filepath.Join(dataRoot, "kernels.json"), filepath.Join(dataRoot, "kernels"))
	if err != nil {
		return nil, err
	}
	return &Service{store: store, kernels: kernels, planner: launch.Planner{}, dataRoot: dataRoot}, nil
}

func (s *Service) Bootstrap() Bootstrap {
	return Bootstrap{Version: AppVersion, Profiles: s.store.List(), Providers: providerCatalog(), Kernels: s.kernels.List()}
}

func (s *Service) ListProfiles() []domain.Profile { return s.store.List() }
func (s *Service) ListKernels() []kernel.Record   { return s.kernels.List() }

func (s *Service) Capabilities(provider, version string) (fingerprint.Capabilities, error) {
	return fingerprint.For(provider, version)
}

func (s *Service) ImportKernel(request kernel.ImportRequest) (kernel.Record, error) {
	return s.kernels.Import(request)
}

func (s *Service) VerifyKernel(id string) (kernel.Record, error) {
	return s.kernels.Verify(id)
}

func (s *Service) DeleteKernel(id string) error {
	for _, item := range s.store.List() {
		if item.Kernel.ID == id {
			return fmt.Errorf("kernel is used by profile %q", item.Name)
		}
	}
	_, err := s.kernels.Delete(id)
	return err
}

func (s *Service) CreateProfile(input domain.Profile) (domain.Profile, error) {
	if strings.TrimSpace(input.ID) == "" {
		id, err := profile.NewID()
		if err != nil {
			return domain.Profile{}, err
		}
		input.ID = id
	}
	if strings.TrimSpace(input.UserDataDir) == "" {
		input.UserDataDir = filepath.Join(s.dataRoot, "profiles", input.ID)
	}
	if err := s.resolveKernel(&input); err != nil {
		return domain.Profile{}, err
	}
	if _, err := fingerprint.Validate(withValidationSeed(input)); err != nil {
		return domain.Profile{}, err
	}
	return s.store.Create(input)
}

func (s *Service) UpdateProfile(input domain.Profile) (domain.Profile, error) {
	if strings.TrimSpace(input.ID) == "" {
		return domain.Profile{}, fmt.Errorf("profile id is required")
	}
	if strings.TrimSpace(input.UserDataDir) == "" {
		input.UserDataDir = filepath.Join(s.dataRoot, "profiles", input.ID)
	}
	if err := s.resolveKernel(&input); err != nil {
		return domain.Profile{}, err
	}
	if _, err := fingerprint.Validate(withValidationSeed(input)); err != nil {
		return domain.Profile{}, err
	}
	return s.store.Update(input)
}

func (s *Service) CloneProfile(id, name string) (domain.Profile, error) {
	source, err := s.store.Get(id)
	if err != nil {
		return domain.Profile{}, err
	}
	newID, err := profile.NewID()
	if err != nil {
		return domain.Profile{}, err
	}
	originalName := source.Name
	source.ID = newID
	source.Name = strings.TrimSpace(name)
	if source.Name == "" {
		source.Name = originalName + " Copy"
	}
	source.UserDataDir = filepath.Join(s.dataRoot, "profiles", newID)
	source.Fingerprint.Seed = ""
	source.CreatedAt = source.CreatedAt.UTC()
	source.UpdatedAt = source.UpdatedAt.UTC()
	return s.CreateProfile(source)
}

func (s *Service) DeleteProfile(id string) error { return s.store.Delete(id) }

func (s *Service) BuildLaunchPlan(request LaunchPlanRequest) (domain.LaunchPlan, error) {
	item, err := s.store.Get(request.ProfileID)
	if err != nil {
		if errors.Is(err, profile.ErrNotFound) {
			return domain.LaunchPlan{}, profile.ErrNotFound
		}
		return domain.LaunchPlan{}, err
	}
	if err := s.resolveKernel(&item); err != nil {
		return domain.LaunchPlan{}, err
	}
	return s.planner.Build(item, request.RemoteDebuggingPort)
}

func (s *Service) resolveKernel(item *domain.Profile) error {
	if strings.TrimSpace(item.Kernel.ID) == "" {
		return nil
	}
	record, err := s.kernels.Verify(item.Kernel.ID)
	if err != nil {
		return err
	}
	if record.Status != kernel.StatusVerified {
		return fmt.Errorf("kernel %q failed integrity verification: %s", record.Name, record.Status)
	}
	item.Kernel.Provider = record.Provider
	item.Kernel.Version = record.Version
	item.Kernel.Executable = record.Executable
	return nil
}

func withValidationSeed(item domain.Profile) domain.Profile {
	if item.Fingerprint.Seed == "" {
		item.Fingerprint.Seed = "profile-default"
	}
	return item
}

func providerCatalog() []ProviderDescriptor {
	definitions := []ProviderDescriptor{
		{ID: fingerprint.ProviderPatched, Name: "Patched Chromium", Description: "Version-aware fingerprint provider with verified command-line contracts.", Versions: []string{"148.0.0", "144.0.0", "142.0.0"}},
		{ID: fingerprint.ProviderNative, Name: "Native Chromium", Description: "Standard Chromium isolation without synthetic fingerprint surfaces.", Versions: []string{"148.0.0", "144.0.0"}},
	}
	for index := range definitions {
		for _, version := range definitions[index].Versions {
			capabilities, err := fingerprint.For(definitions[index].ID, version)
			if err == nil {
				definitions[index].Samples = append(definitions[index].Samples, capabilities)
			}
		}
	}
	return definitions
}
