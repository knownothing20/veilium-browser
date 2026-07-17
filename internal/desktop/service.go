package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/proxy"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const AppVersion = "0.6.0-dev"

type RuntimeSupervisor interface {
	Start(context.Context, string, string, supervisor.PlanBuilder) (supervisor.Session, error)
	Stop(context.Context, string) (supervisor.Session, error)
	Shutdown(context.Context) error
	List() []supervisor.Session
	IsActive(string) bool
}

type Service struct {
	store       *profile.Store
	kernels     *kernel.Store
	credentials *credential.Manager
	planner     launch.Planner
	supervisor  RuntimeSupervisor
	dataRoot    string
	profilesDir string
}

type Bootstrap struct {
	Version            string               `json:"version"`
	Profiles           []domain.Profile     `json:"profiles"`
	Providers          []ProviderDescriptor `json:"providers"`
	Kernels            []kernel.Record      `json:"kernels"`
	Sessions           []supervisor.Session `json:"sessions"`
	Credentials        []credential.Record  `json:"credentials"`
	CredentialProvider string               `json:"credentialProvider"`
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
	runtimeSupervisor, err := supervisor.New(filepath.Join(dataRoot, "runtime-logs"))
	if err != nil {
		return nil, err
	}
	return newService(store, dataRoot, runtimeSupervisor)
}

func newService(store *profile.Store, dataRoot string, runtimeSupervisor RuntimeSupervisor) (*Service, error) {
	credentials, err := credential.Open(filepath.Join(dataRoot, "credentials.json"))
	if err != nil {
		return nil, err
	}
	return newServiceWithCredentials(store, dataRoot, runtimeSupervisor, credentials)
}

func newServiceWithCredentials(store *profile.Store, dataRoot string, runtimeSupervisor RuntimeSupervisor, credentials *credential.Manager) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("profile store is required")
	}
	if strings.TrimSpace(dataRoot) == "" {
		return nil, fmt.Errorf("data root is required")
	}
	if runtimeSupervisor == nil {
		return nil, fmt.Errorf("runtime supervisor is required")
	}
	if credentials == nil {
		return nil, fmt.Errorf("credential manager is required")
	}
	kernels, err := kernel.Open(filepath.Join(dataRoot, "kernels.json"), filepath.Join(dataRoot, "kernels"))
	if err != nil {
		return nil, err
	}
	service := &Service{
		store:       store,
		kernels:     kernels,
		credentials: credentials,
		planner:     launch.Planner{},
		supervisor:  runtimeSupervisor,
		dataRoot:    dataRoot,
		profilesDir: filepath.Join(dataRoot, "profiles"),
	}
	_ = registryFor(service)
	return service, nil
}

func (s *Service) Bootstrap() Bootstrap {
	return Bootstrap{
		Version:            AppVersion,
		Profiles:           s.store.List(),
		Providers:          providerCatalog(),
		Kernels:            s.kernels.List(),
		Sessions:           s.supervisor.List(),
		Credentials:        s.credentials.List(),
		CredentialProvider: credential.ProviderName(),
	}
}

func (s *Service) ListProfiles() []domain.Profile        { return s.store.List() }
func (s *Service) ListKernels() []kernel.Record          { return s.kernels.List() }
func (s *Service) ListSessions() []supervisor.Session    { return s.supervisor.List() }
func (s *Service) ListCredentials() []credential.Record  { return s.credentials.List() }
func (s *Service) Shutdown(ctx context.Context) error    { return shutdownRuntimeAndBridges(s, ctx) }
func (s *Service) IsProfileActive(profileID string) bool { return s.supervisor.IsActive(profileID) }
func (s *Service) Capabilities(provider, version string) (fingerprint.Capabilities, error) {
	return fingerprint.For(provider, version)
}

func (s *Service) SaveCredential(request credential.SaveRequest) (credential.Record, error) {
	return s.credentials.Save(request)
}

func (s *Service) DeleteCredential(id string) error {
	for _, item := range s.store.List() {
		if strings.TrimSpace(item.Proxy.CredentialRef) == strings.TrimSpace(id) {
			return fmt.Errorf("credential is used by profile %q", item.Name)
		}
	}
	return s.credentials.Delete(id)
}

func (s *Service) ImportKernel(request kernel.ImportRequest) (kernel.Record, error) {
	return s.kernels.Import(request)
}

func (s *Service) VerifyKernel(id string) (kernel.Record, error) { return s.kernels.Verify(id) }

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
		input.UserDataDir = filepath.Join(s.profilesDir, input.ID)
	}
	if err := s.resolveKernel(&input); err != nil {
		return domain.Profile{}, err
	}
	if err := s.validateProxy(input); err != nil {
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
	if s.supervisor.IsActive(input.ID) {
		return domain.Profile{}, fmt.Errorf("profile %q cannot be edited while its browser is running", input.Name)
	}
	if strings.TrimSpace(input.UserDataDir) == "" {
		input.UserDataDir = filepath.Join(s.profilesDir, input.ID)
	}
	if err := s.resolveKernel(&input); err != nil {
		return domain.Profile{}, err
	}
	if err := s.validateProxy(input); err != nil {
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
	source.UserDataDir = filepath.Join(s.profilesDir, newID)
	source.Fingerprint.Seed = ""
	source.CreatedAt = source.CreatedAt.UTC()
	source.UpdatedAt = source.UpdatedAt.UTC()
	return s.CreateProfile(source)
}

func (s *Service) DeleteProfile(id string) error {
	if s.supervisor.IsActive(id) {
		return fmt.Errorf("profile cannot be deleted while its browser is running")
	}
	return s.store.Delete(id)
}

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
	if err := s.validateProxy(item); err != nil {
		return domain.LaunchPlan{}, err
	}
	return s.planner.Build(item, request.RemoteDebuggingPort)
}

func (s *Service) StartProfile(ctx context.Context, profileID string) (supervisor.Session, error) {
	item, err := s.store.Get(profileID)
	if err != nil {
		return supervisor.Session{}, err
	}
	if strings.TrimSpace(item.Kernel.ID) == "" {
		return supervisor.Session{}, fmt.Errorf("profile %q must use a registered kernel before it can start", item.Name)
	}
	if err := s.resolveKernel(&item); err != nil {
		return supervisor.Session{}, err
	}
	if err := s.validateProxy(item); err != nil {
		return supervisor.Session{}, err
	}
	managedDir := filepath.Join(s.profilesDir, item.ID)
	if !sameCleanPath(item.UserDataDir, managedDir) {
		return supervisor.Session{}, fmt.Errorf("profile %q uses an unmanaged user data directory", item.Name)
	}
	if err := prepareManagedProfileDir(s.profilesDir, managedDir); err != nil {
		return supervisor.Session{}, err
	}
	route, err := proxy.Resolve(item.Proxy.URL, item.Proxy.CredentialRef)
	if err != nil {
		return supervisor.Session{}, err
	}
	if !route.RequiresBridge {
		return s.supervisor.Start(ctx, item.ID, item.Name, func(port int) (domain.LaunchPlan, error) {
			return s.planner.Build(item, port)
		})
	}
	if route.BridgeKind != "local-auth-bridge" {
		return supervisor.Session{}, fmt.Errorf("proxy bridge %q is not available yet", route.BridgeKind)
	}
	material, err := s.credentials.Resolve(route.CredentialRef)
	if err != nil {
		return supervisor.Session{}, fmt.Errorf("resolve proxy credential: %w", err)
	}
	bridge, err := proxyBridgeFactory(s).Start(ctx, route.DisplayURL, material)
	if err != nil {
		return supervisor.Session{}, fmt.Errorf("start authenticated proxy bridge: %w", err)
	}
	bridgedProfile := item
	bridgedProfile.Proxy.URL = bridge.URL()
	bridgedProfile.Proxy.CredentialRef = ""
	session, err := s.supervisor.Start(ctx, item.ID, item.Name, func(port int) (domain.LaunchPlan, error) {
		plan, buildErr := s.planner.Build(bridgedProfile, port)
		if buildErr != nil {
			return domain.LaunchPlan{}, buildErr
		}
		plan.ProxyDisplay = route.DisplayURL
		plan.Warnings = append(plan.Warnings, "authenticated upstream proxy is routed through a loopback-only Veilium bridge")
		return plan, nil
	})
	if err != nil {
		_ = bridge.Close()
		return session, err
	}
	token := registerProxyBridge(s, item.ID, bridge)
	go watchProxyBridge(s, item.ID, token)
	return session, nil
}

func (s *Service) StopProfile(ctx context.Context, profileID string) (supervisor.Session, error) {
	session, runtimeErr := s.supervisor.Stop(ctx, profileID)
	bridgeErr := closeProfileProxyBridge(s, profileID)
	if runtimeErr != nil && bridgeErr != nil {
		return session, fmt.Errorf("stop browser: %v; stop proxy bridge: %w", runtimeErr, bridgeErr)
	}
	if runtimeErr != nil {
		return session, runtimeErr
	}
	return session, bridgeErr
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

func (s *Service) validateProxy(item domain.Profile) error {
	route, err := proxy.Resolve(item.Proxy.URL, item.Proxy.CredentialRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(route.CredentialRef) == "" {
		return nil
	}
	if _, err := s.credentials.Get(route.CredentialRef); err != nil {
		if errors.Is(err, credential.ErrNotFound) {
			return fmt.Errorf("credential reference %q is not registered", route.CredentialRef)
		}
		return err
	}
	return nil
}

func prepareManagedProfileDir(root, directory string) error {
	if !isPathWithin(root, directory) {
		return fmt.Errorf("refusing profile directory outside the managed root")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create managed profile root: %w", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect managed profile root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("managed profile root must be a real directory")
	}
	if info, err := os.Lstat(directory); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("profile user data path must be a real directory")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect profile user data directory: %w", err)
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return fmt.Errorf("create profile user data directory: %w", err)
	}
	return nil
}

func sameCleanPath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func isPathWithin(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
