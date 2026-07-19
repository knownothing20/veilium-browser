package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/consistency"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/networkevidence"
)

type networkEvidenceServiceState struct {
	once    sync.Once
	manager *networkevidence.Manager
	probes  *networkevidence.ProbeStore
	err     error
}

var networkEvidenceServiceRegistry sync.Map

func networkEvidenceStateFor(service *Service) (*networkEvidenceServiceState, error) {
	if service == nil {
		return nil, fmt.Errorf("desktop service is required")
	}
	candidate := &networkEvidenceServiceState{}
	loaded, _ := networkEvidenceServiceRegistry.LoadOrStore(service, candidate)
	state := loaded.(*networkEvidenceServiceState)
	state.once.Do(func() {
		store, err := networkevidence.OpenStore(filepath.Join(service.dataRoot, "network-evidence"), networkevidence.StoreOptions{})
		if err != nil {
			state.err = err
			return
		}
		state.probes, err = networkevidence.OpenProbeStore(filepath.Join(service.dataRoot, "network-probes.json"))
		if err != nil {
			state.err = err
			return
		}
		browserExecutor, err := networkevidence.NewBrowserExecutor(networkevidence.BrowserExecutorOptions{
			CollectorFactory: func(set networkevidence.ProbeSet) (networkevidence.BrowserCollector, error) {
				return networkevidence.StartCollector(set)
			},
		})
		if err != nil {
			state.err = err
			return
		}
		state.manager, state.err = networkevidence.NewManager(store, networkevidence.ReconcilingExecutor{Inner: browserExecutor}, networkevidence.ManagerOptions{})
	})
	return state, state.err
}

func (service *Service) NetworkProbeSet() (networkevidence.ProbeSet, bool, error) {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return networkevidence.ProbeSet{}, false, err
	}
	return state.probes.Get()
}

func (service *Service) SaveNetworkProbeSet(set networkevidence.ProbeSet) (networkevidence.ProbeSet, error) {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return networkevidence.ProbeSet{}, err
	}
	if state.manager != nil {
		for _, item := range service.store.List() {
			if state.manager.IsActive(item.ID) {
				return networkevidence.ProbeSet{}, fmt.Errorf("network probe configuration cannot change while evidence is running")
			}
		}
	}
	return state.probes.Save(set)
}

func (service *Service) DeleteNetworkProbeSet() error {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return err
	}
	for _, item := range service.store.List() {
		if state.manager.IsActive(item.ID) {
			return fmt.Errorf("network probe configuration cannot be deleted while evidence is running")
		}
	}
	return state.probes.Delete()
}

func (service *Service) RunNetworkEvidence(ctx context.Context, profileID string) (networkevidence.Run, error) {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return networkevidence.Run{}, err
	}
	set, exists, err := state.probes.Get()
	if err != nil {
		return networkevidence.Run{}, err
	}
	if !exists {
		return networkevidence.Run{}, fmt.Errorf("configure an explicit replaceable or self-hostable network ProbeSet first")
	}
	profileID = strings.TrimSpace(profileID)
	item, err := service.store.Get(profileID)
	if err != nil {
		return networkevidence.Run{}, err
	}
	if strings.TrimSpace(item.Kernel.ID) == "" {
		return networkevidence.Run{}, fmt.Errorf("profile %q must use a managed kernel before network evidence can run", item.Name)
	}
	if _, err := service.kernels.Verify(item.Kernel.ID); err != nil {
		return networkevidence.Run{}, err
	}
	if err := service.resolveKernel(&item); err != nil {
		return networkevidence.Run{}, err
	}
	session, ok := readyEvidenceSession(service.supervisor.List(), item.ID)
	if !ok || strings.TrimSpace(session.WebSocketDebuggerURL) == "" {
		return networkevidence.Run{}, fmt.Errorf("profile %q requires a ready managed browser session", item.Name)
	}
	base, ok, err := latestApplicableBrowserEvidence(service, item)
	if err != nil {
		return networkevidence.Run{}, err
	}
	if !ok {
		return networkevidence.Run{}, fmt.Errorf("profile %q requires a current passed or partial real-browser Evidence run", item.Name)
	}
	return state.manager.Run(ctx, networkevidence.RunRequest{
		Profile:      item,
		BaseEvidence: base,
		Session:      session,
		ProbeSet:     set,
		SessionReady: func() bool {
			current, exists := readyEvidenceSession(service.supervisor.List(), item.ID)
			return exists && sameEvidenceSession(session, current)
		},
	})
}

func latestApplicableBrowserEvidence(service *Service, item domain.Profile) (evidence.Run, bool, error) {
	runs, err := service.ListEvidence(item.ID)
	if err != nil {
		return evidence.Run{}, false, err
	}
	now := time.Now().UTC()
	for _, run := range runs {
		if run.CompletedAt == nil || !run.ExpiresAt.After(now) {
			continue
		}
		if run.Status != evidence.RunPassed && run.Status != evidence.RunPartial {
			continue
		}
		if run.ProfileID != item.ID || run.ProviderID != item.Kernel.Provider || run.BrowserVersion != item.Kernel.Version {
			continue
		}
		if strings.TrimSpace(run.ConsistencyInputDigest) == "" {
			continue
		}
		return run, true, nil
	}
	return evidence.Run{}, false, nil
}

func (service *Service) CancelNetworkEvidence(profileID string) error {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return err
	}
	return state.manager.Cancel(profileID)
}

func (service *Service) ListNetworkEvidence(profileID string) ([]networkevidence.Run, error) {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return nil, err
	}
	return state.manager.List(strings.TrimSpace(profileID))
}

func (service *Service) GetNetworkEvidence(id string) (networkevidence.Run, error) {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return networkevidence.Run{}, err
	}
	return state.manager.Get(strings.TrimSpace(id))
}

func (service *Service) DeleteNetworkEvidence(id string) error {
	state, err := networkEvidenceStateFor(service)
	if err != nil {
		return err
	}
	return state.manager.Delete(strings.TrimSpace(id))
}

func (service *Service) NetworkEvidenceActive(profileID string) bool {
	state, err := networkEvidenceStateFor(service)
	return err == nil && state.manager.IsActive(profileID)
}

func (service *Service) ProfileHealth(profileID string) (consistency.Result, error) {
	result, err := service.ProfileConsistency(profileID)
	if err != nil {
		return consistency.Result{}, err
	}
	item, err := service.store.Get(strings.TrimSpace(profileID))
	if err != nil {
		return consistency.Result{}, err
	}
	return service.applyNetworkHealth(item, result)
}

func (service *Service) applyNetworkHealth(item domain.Profile, result consistency.Result) (consistency.Result, error) {
	set, configured, err := service.NetworkProbeSet()
	if err != nil {
		return consistency.Result{}, err
	}
	if !configured {
		appendNetworkCheck(&result, consistency.Check{ID: "network.probes", Status: consistency.CheckWarning, ReasonCode: "probe-set-not-configured", Detail: "No explicit replaceable or self-hostable network ProbeSet is configured."})
		return normalizeNetworkHealth(result), nil
	}
	route, err := networkevidence.RouteForProfile(item)
	if err != nil {
		return consistency.Result{}, err
	}
	runs, err := service.ListNetworkEvidence(item.ID)
	if err != nil {
		return consistency.Result{}, err
	}
	if len(runs) == 0 {
		appendNetworkCheck(&result, consistency.Check{ID: "network.evidence", Status: consistency.CheckWarning, ReasonCode: "network-evidence-missing", Detail: "Run browser network evidence with ProbeSet " + set.ID + "."})
		return normalizeNetworkHealth(result), nil
	}
	run := runs[0]
	if run.ProviderID != item.Kernel.Provider || run.BrowserVersion != item.Kernel.Version || run.ConsistencyInputDigest != result.InputDigest || run.Route.Digest != route.Digest || run.ProbeSetID != set.ID || run.ProbeSetRevision != set.Revision {
		appendNetworkCheck(&result, consistency.Check{ID: "network.freshness", Status: consistency.CheckWarning, ReasonCode: "network-evidence-stale", Detail: "The latest network evidence does not match the current Profile, route, browser identity, consistency input, or ProbeSet."})
		return normalizeNetworkHealth(result), nil
	}
	if !run.ExpiresAt.After(time.Now().UTC()) {
		appendNetworkCheck(&result, consistency.Check{ID: "network.freshness", Status: consistency.CheckWarning, ReasonCode: "network-evidence-expired", Detail: "The latest network evidence has expired."})
		return normalizeNetworkHealth(result), nil
	}
	for _, observation := range run.Observations {
		status := consistency.CheckPassed
		switch observation.Status {
		case networkevidence.ObservationFailed:
			status = consistency.CheckFailed
		case networkevidence.ObservationPartial, networkevidence.ObservationUnavailable, networkevidence.ObservationSkipped:
			status = consistency.CheckWarning
		}
		appendNetworkCheck(&result, consistency.Check{
			ID:         "network." + string(observation.ProbeKind),
			Status:     status,
			Expected:   observation.Expected,
			Observed:   strings.Join(observation.Values, ", "),
			ReasonCode: observation.ReasonCode,
			Detail:     observation.Detail,
		})
	}
	for _, limitation := range run.Limitations {
		result.DegradedReasons = append(result.DegradedReasons, "network limitation: "+limitation)
	}
	if run.Status == networkevidence.RunFailed && len(run.Observations) == 0 {
		appendNetworkCheck(&result, consistency.Check{ID: "network.run", Status: consistency.CheckWarning, ReasonCode: run.FailureCode, Detail: run.FailureDetail})
	}
	return normalizeNetworkHealth(result), nil
}

func appendNetworkCheck(result *consistency.Result, check consistency.Check) {
	result.Checks = append(result.Checks, check)
	if check.Status == consistency.CheckFailed {
		result.BlockingReasons = append(result.BlockingReasons, check.ID+": "+networkCheckDetail(check))
	}
	if check.Status == consistency.CheckWarning || check.Status == consistency.CheckUnknown {
		result.DegradedReasons = append(result.DegradedReasons, check.ID+": "+networkCheckDetail(check))
	}
}

func networkCheckDetail(check consistency.Check) string {
	if strings.TrimSpace(check.Detail) != "" {
		return strings.TrimSpace(check.Detail)
	}
	if strings.TrimSpace(check.ReasonCode) != "" {
		return strings.TrimSpace(check.ReasonCode)
	}
	return "network evidence check did not pass"
}

func normalizeNetworkHealth(result consistency.Result) consistency.Result {
	result.BlockingReasons = uniqueHealthReasons(result.BlockingReasons)
	result.DegradedReasons = uniqueHealthReasons(result.DegradedReasons)
	if len(result.BlockingReasons) > 0 {
		result.Status = consistency.HealthBlocked
	} else if len(result.DegradedReasons) > 0 && result.Status == consistency.HealthHealthy {
		result.Status = consistency.HealthDegraded
	}
	return result
}

func uniqueHealthReasons(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (service *Service) NetworkCompatibilityMatrix() (networkevidence.CompatibilityMatrix, error) {
	runs, err := service.ListNetworkEvidence("")
	if err != nil {
		return networkevidence.CompatibilityMatrix{}, err
	}
	entries := make([]networkevidence.CompatibilityEntry, 0, len(runs)*3)
	seen := make(map[string]struct{}, len(runs)*3)
	for _, run := range runs {
		capabilities, err := fingerprint.For(run.ProviderID, run.BrowserVersion)
		if err != nil {
			continue
		}
		for _, capabilityID := range []string{"network.route", "network.webrtc", "network.dns"} {
			entry := compatibilityEntryForRun(run, capabilities.TrustStatus, capabilityID)
			key := fmt.Sprintf("%s|%d|%s|%s|%s|%s|%s", entry.ProviderID, entry.ProviderRevision, entry.BrowserVersion, entry.OperatingSystem, entry.Architecture, entry.BinaryIdentityDigest, entry.CapabilityID)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			entries = append(entries, entry)
		}
	}
	return networkevidence.GenerateMatrix(time.Now().UTC(), entries)
}

func compatibilityEntryForRun(run networkevidence.Run, trust fingerprint.TrustStatus, capabilityID string) networkevidence.CompatibilityEntry {
	status := networkevidence.CompatibilityUntested
	limitations := append([]string(nil), run.Limitations...)
	evidenceIDs := []string(nil)
	var reviewedAt *time.Time
	var expiresAt *time.Time
	if trust == fingerprint.TrustReviewed && run.CompletedAt != nil {
		status = networkevidence.CompatibilityPartial
		observationStatus, found := networkCapabilityStatus(run, capabilityID)
		if found {
			switch observationStatus {
			case networkevidence.ObservationPassed:
				status = networkevidence.CompatibilityVerified
			case networkevidence.ObservationFailed:
				status = networkevidence.CompatibilityFailed
			default:
				status = networkevidence.CompatibilityPartial
			}
		}
		evidenceIDs = []string{run.ID}
		reviewed := run.CompletedAt.UTC()
		reviewedAt = &reviewed
		expires := run.ExpiresAt.UTC()
		expiresAt = &expires
	} else {
		limitations = append(limitations, "Provider is custom or legacy and cannot receive reviewed compatibility status")
	}
	return networkevidence.CompatibilityEntry{
		SchemaVersion:        networkevidence.MatrixSchemaVersion,
		ProviderID:           run.ProviderID,
		ProviderRevision:     run.ProviderRevision,
		ProviderTrust:        trust,
		BrowserVersion:       run.BrowserVersion,
		OperatingSystem:      run.OperatingSystem,
		Architecture:         run.Architecture,
		BinaryIdentityDigest: run.BinaryIdentityDigest,
		CapabilityID:         capabilityID,
		Status:               status,
		ProbeSetID:           run.ProbeSetID,
		ProbeSetRevision:     run.ProbeSetRevision,
		NetworkEvidenceIDs:   evidenceIDs,
		ReviewedAt:           reviewedAt,
		EvidenceExpiresAt:    expiresAt,
		Limitations:          limitations,
	}
}

func networkCapabilityStatus(run networkevidence.Run, capabilityID string) (networkevidence.ObservationStatus, bool) {
	kind := networkevidence.ProbeExitIP
	switch capabilityID {
	case "network.webrtc":
		kind = networkevidence.ProbeWebRTCSTUN
	case "network.dns":
		kind = networkevidence.ProbeDelegatedDNS
	}
	for _, observation := range run.Observations {
		if observation.ProbeKind == kind {
			return observation.Status, true
		}
	}
	return networkevidence.ObservationUnavailable, false
}

func shutdownNetworkEvidenceRuntimes(service *Service) {
	loaded, ok := networkEvidenceServiceRegistry.Load(service)
	if !ok {
		return
	}
	state := loaded.(*networkEvidenceServiceState)
	if state.manager != nil {
		state.manager.Shutdown()
	}
	networkEvidenceServiceRegistry.Delete(service)
}
