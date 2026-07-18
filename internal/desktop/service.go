package desktop

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterinstaller"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
	"github.com/knownothing20/veilium-browser/internal/adaptervalidation"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/launch"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/proxy"
	"github.com/knownothing20/veilium-browser/internal/singboxprovider"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
	"github.com/knownothing20/veilium-browser/internal/xrayprovider"
)

const AppVersion = "0.13.0-dev"

type RuntimeSupervisor interface {
	Start(context.Context, string, string, supervisor.PlanBuilder) (supervisor.Session, error)
	Stop(context.Context, string) (supervisor.Session, error)
	Shutdown(context.Context) error
	List() []supervisor.Session
	IsActive(string) bool
}

type AdapterValidator interface {
	Validate(context.Context, adapter.Record) (adaptervalidation.Report, error)
}

type AdapterInstaller interface {
	Install(context.Context, adapterinstaller.Request) (adapter.Record, error)
}

type Service struct {
	store            *profile.Store
	kernels          *kernel.Store
	adapters         *adapter.Store
	adapterRuntime   adapterruntime.Factory
	adapterValidator AdapterValidator
	adapterInstaller AdapterInstaller
	credentials      *credential.Manager
	planner          launch.Planner
	supervisor       RuntimeSupervisor
	dataRoot         string
	profilesDir      string
}

type Bootstrap struct {
	Version            string               `json:"version"`
	Profiles           []domain.Profile     `json:"profiles"`
	Providers          []ProviderDescriptor `json:"providers"`
	Kernels            []kernel.Record      `json:"kernels"`
	Adapters           []adapter.Record     `json:"adapters"`
	Sessions           []supervisor.Session `json:"sessions"`
	Credentials        []credential.Record  `json:"credentials"`
	CredentialProvider string               `json:"credentialProvider"`
	AdapterPins        []adapterrelease.Pin `json:"adapterPins"`
	RuntimePlatform    string               `json:"runtimePlatform"`
	RuntimeArch        string               `json:"runtimeArch"`
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
	adapters, err := adapter.Open(filepath.Join(dataRoot, "adapters.json"), filepath.Join(dataRoot, "adapters"))
	if err != nil {
		return nil, err
	}
	providers := adapterruntime.NewRegistry()
	if err := providers.Register(xrayprovider.New()); err != nil {
		return nil, fmt.Errorf("register Xray adapter provider: %w", err)
	}
	if err := providers.Register(singboxprovider.New()); err != nil {
		return nil, fmt.Errorf("register sing-box adapter provider: %w", err)
	}
	adapterManager, err := adapterruntime.NewManager(filepath.Join(dataRoot, "adapter-runtime"), providers)
	if err != nil {
		return nil, err
	}
	officialInstaller, err := adapterinstaller.New(adapters, filepath.Join(dataRoot, "adapter-installer"))
	if err != nil {
		return nil, err
	}
	service := &Service{
		store:            store,
		kernels:          kernels,
		adapters:         adapters,
		adapterRuntime:   adapterManager,
		adapterValidator: adaptervalidation.New(),
		adapterInstaller: officialInstaller,
		credentials:      credentials,
		planner:          launch.Planner{},
		supervisor:       runtimeSupervisor,
		dataRoot:         dataRoot,
		profilesDir:      filepath.Join(dataRoot, "profiles"),
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
		Adapters:           s.adapters.List(),
		Sessions:           s.supervisor.List(),
		Credentials:        s.credentials.List(),
		CredentialProvider: credential.ProviderName(),
		AdapterPins:        officialAdapterPins(),
		RuntimePlatform:    runtime.GOOS,
		RuntimeArch:        runtime.GOARCH,
	}
}

func (s *Service) ListProfiles() []domain.Profile        { return s.store.List() }
func (s *Service) ListKernels() []kernel.Record          { return s.kernels.List() }
func (s *Service) ListAdapters() []adapter.Record        { return s.adapters.List() }
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

func (s *Service) ImportAdapter(request adapter.ImportRequest) (adapter.Record, error) {
	return s.adapters.Import(request)
}

func (s *Service) VerifyAdapter(id string) (adapter.Record, error) {
	return s.adapters.Verify(id)
}

func (s *Service) ValidateAdapter(ctx context.Context, id string) (adaptervalidation.Report, error) {
	record, err := s.adapters.Verify(id)
	if err != nil {
		return adaptervalidation.Report{}, err
	}
	if s.adapterValidator == nil {
		return adaptervalidation.Report{}, fmt.Errorf("adapter validator is unavailable")
	}
	return s.adapterValidator.Validate(ctx, record)
}

func (s *Service) InstallOfficialAdapter(ctx context.Context, request adapterinstaller.Request) (adapter.Record, error) {
	if s.adapterInstaller == nil {
		return adapter.Record{}, fmt.Errorf("official adapter installer is unavailable")
	}
	return s.adapterInstaller.Install(ctx, request)
}

func (s *Service) DeleteAdapter(id string) error {
	id = strings.TrimSpace(id)
	for _, item := range s.store.List() {
		if strings.TrimSpace(item.Proxy.AdapterRef) == id {
			return fmt.Errorf("proxy adapter is used by profile %q", item.Name)
		}
	}
	_, err := s.adapters.Delete(id)
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
		record, recordErr := s.resolveAdapter(item)
		if recordErr != nil {
			return supervisor.Session{}, recordErr
		}
		material, materialErr := s.credentials.Resolve(route.CredentialRef)
		if materialErr != nil {
			return supervisor.Session{}, fmt.Errorf("resolve advanced proxy credential: %w", materialErr)
		}
		parsed, _ := url.Parse(item.Proxy.URL)
		instance, startErr := s.adapterRuntime.Start(ctx, adapterruntime.Request{
			Adapter: record, Scheme: strings.ToLower(parsed.Scheme), ProxyURL: item.Proxy.URL,
			CredentialRef: route.CredentialRef, CredentialUsername: material.Username, CredentialSecret: material.Secret,
			ProfileID: item.ID,
		})
		if startErr != nil {
			return supervisor.Session{}, fmt.Errorf("start %s adapter runtime: %w", record.Kind, startErr)
		}
		bridgedProfile := item
		bridgedProfile.Proxy.URL = instance.URL()
		bridgedProfile.Proxy.CredentialRef = ""
		bridgedProfile.Proxy.AdapterRef = ""
		session, browserErr := s.supervisor.Start(ctx, item.ID, item.Name, func(port int) (domain.LaunchPlan, error) {
			plan, buildErr := s.planner.Build(bridgedProfile, port)
			if buildErr != nil {
				return domain.LaunchPlan{}, buildErr
			}
			plan.ProxyDisplay = route.DisplayURL
			plan.Warnings = append(plan.Warnings, record.Kind+" routes are exposed to Chromium through a private loopback SOCKS5 runtime")
			return plan, nil
		})
		if browserErr != nil {
			_ = instance.Close()
			return session, browserErr
		}
		token := registerNetworkRuntime(s, item.ID, instance)
		go watchNetworkRuntime(s, item.ID, token, instance)
		return session, nil
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
	bridgedProfile.Proxy.AdapterRef = ""
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
	token := registerNetworkRuntime(s, item.ID, bridge)
	go watchNetworkRuntime(s, item.ID, token, bridge)
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

func (s *Service) resolveAdapter(item domain.Profile) (adapter.Record, error) {
	adapterID := strings.TrimSpace(item.Proxy.AdapterRef)
	if adapterID == "" {
		return adapter.Record{}, fmt.Errorf("advanced proxy route requires a managed adapter")
	}
	record, err := s.adapters.Verify(adapterID)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			return adapter.Record{}, fmt.Errorf("proxy adapter reference %q is not registered", adapterID)
		}
		return adapter.Record{}, err
	}
	if record.Status != adapter.StatusVerified {
		return adapter.Record{}, fmt.Errorf("proxy adapter %q failed integrity verification: %s", record.Name, record.Status)
	}
	parsed, err := url.Parse(item.Proxy.URL)
	if err != nil {
		return adapter.Record{}, fmt.Errorf("parse proxy scheme: %w", err)
	}
	if !adapter.SupportsScheme(record.Kind, parsed.Scheme) {
		return adapter.Record{}, fmt.Errorf("proxy adapter %q (%s) does not support scheme %q", record.Name, record.Kind, parsed.Scheme)
	}
	return record, nil
}

func (s *Service) validateProxy(item domain.Profile) error {
	route, err := proxy.Resolve(item.Proxy.URL, item.Proxy.CredentialRef)
	if err != nil {
		return err
	}
	adapterRef := strings.TrimSpace(item.Proxy.AdapterRef)
	if strings.TrimSpace(route.CredentialRef) == "" {
		if adapterRef != "" {
			return fmt.Errorf("direct and native proxy routes cannot use a managed adapter")
		}
		return nil
	}
	if _, err := s.credentials.Get(route.CredentialRef); err != nil {
		if errors.Is(err, credential.ErrNotFound) {
			return fmt.Errorf("credential reference %q is not registered", route.CredentialRef)
		}
		return err
	}
	if route.BridgeKind == "local-auth-bridge" {
		if adapterRef != "" {
			return fmt.Errorf("HTTP and SOCKS5 authentication routes do not use an Xray or sing-box adapter")
		}
		return nil
	}
	_, err = s.resolveAdapter(item)
	return err
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

func officialAdapterPins() []adapterrelease.Pin {
	pins, err := adapterrelease.Pins()
	if err != nil {
		return nil
	}
	return pins
}

func providerCatalog() []ProviderDescriptor {
	contracts := fingerprint.Definitions()
	definitions := make([]ProviderDescriptor, 0, len(contracts))
	for _, contract := range contracts {
		descriptor := ProviderDescriptor{
			ID:          contract.ID,
			Name:        contract.Name,
			Description: contract.Description,
			Versions:    append([]string(nil), contract.Versions...),
		}
		for _, version := range descriptor.Versions {
			capabilities, err := fingerprint.For(descriptor.ID, version)
			if err == nil {
				descriptor.Samples = append(descriptor.Samples, capabilities)
			}
		}
		definitions = append(definitions, descriptor)
	}
	return definitions
}
