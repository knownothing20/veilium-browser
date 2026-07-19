package networkevidence

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

type fakeExecutor struct {
	mu      sync.Mutex
	result  ExecutionResult
	err     error
	started chan struct{}
	release chan struct{}
}

func (executor *fakeExecutor) Execute(ctx context.Context, _ ExecutionRequest) (ExecutionResult, error) {
	if executor.started != nil {
		select {
		case executor.started <- struct{}{}:
		default:
		}
	}
	if executor.release != nil {
		select {
		case <-executor.release:
		case <-ctx.Done():
			return ExecutionResult{}, ctx.Err()
		}
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.result, executor.err
}

func TestManagerPersistsSuccessfulNetworkEvidence(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "network"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: ExecutionResult{Observations: validRun(now).Observations}}
	manager, err := NewManager(store, executor, ManagerOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	run, err := manager.Run(context.Background(), validRunRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunPassed || run.EvidenceRunID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected network evidence run: %#v", run)
	}
	stored, err := store.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Route.Kind != RouteDirect || !validSHA256(stored.BinaryIdentityDigest) {
		t.Fatalf("unexpected stored route or binary identity: %#v", stored)
	}
}

func TestManagerRejectsConcurrentRunAndPersistsCancellation(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "network"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{started: make(chan struct{}, 1), release: make(chan struct{})}
	manager, err := NewManager(store, executor, ManagerOptions{Now: func() time.Time { return now }, Timeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	request := validRunRequest(now)
	type response struct {
		run Run
		err error
	}
	responses := make(chan response, 1)
	go func() {
		run, err := manager.Run(context.Background(), request)
		responses <- response{run: run, err: err}
	}()
	<-executor.started
	if _, err := manager.Run(context.Background(), request); !errors.Is(err, ErrRunActive) {
		t.Fatalf("expected concurrent-run rejection, got %v", err)
	}
	if err := manager.Cancel(request.Profile.ID); err != nil {
		t.Fatal(err)
	}
	result := <-responses
	if !errors.Is(result.err, context.Canceled) || result.run.Status != RunCancelled {
		t.Fatalf("expected persisted cancellation, got %#v, %v", result.run, result.err)
	}
	items, err := store.List(request.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != RunCancelled {
		t.Fatalf("unexpected stored cancellation: %#v", items)
	}
}

func TestManagerPersistsBrowserExitAsIncomplete(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "network"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{release: make(chan struct{})}
	manager, err := NewManager(store, executor, ManagerOptions{Now: func() time.Time { return now }, Timeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	request := validRunRequest(now)
	request.SessionReady = func() bool { return false }
	run, err := manager.Run(context.Background(), request)
	if !errors.Is(err, ErrSessionExited) || run.Status != RunIncomplete || run.FailureCode != "browser-exited" {
		t.Fatalf("expected incomplete browser-exit report, got %#v, %v", run, err)
	}
}

func TestManagerRejectsExpiredBaseEvidence(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "network"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(store, &fakeExecutor{}, ManagerOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	request := validRunRequest(now)
	request.BaseEvidence.ExpiresAt = now.Add(-time.Second)
	if _, err := manager.Run(context.Background(), request); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired base Evidence rejection, got %v", err)
	}
}

func TestManagerFailsClosedOnObservationOutsideProbeSet(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "network"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	observation := validRun(now).Observations[0]
	observation.ProbeID = "unknown"
	executor := &fakeExecutor{result: ExecutionResult{Observations: []Observation{observation}}}
	manager, err := NewManager(store, executor, ManagerOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	run, err := manager.Run(context.Background(), validRunRequest(now))
	if err == nil || run.Status != RunFailed || run.FailureCode != "invalid-probe-result" {
		t.Fatalf("expected invalid result failure, got %#v, %v", run, err)
	}
}

func validRunRequest(now time.Time) RunRequest {
	profile := domain.Profile{
		ID: "profile-a", Name: "Profile A",
		Kernel: domain.KernelRef{ID: "kernel-a", Provider: fingerprint.ProviderCustom, Version: "148.0.0", Executable: "/managed/chrome"},
		Fingerprint: domain.FingerprintConfig{
			Platform: "windows", Brand: "Chromium", Language: "en-US", Timezone: "UTC",
			ScreenWidth: 1920, ScreenHeight: 1080, WindowWidth: 1280, WindowHeight: 800, DeviceScaleFactor: 1,
			WebRTCPolicy: "proxy-only", CanvasMode: "native", AudioMode: "native",
			FontMode: "native", ClientRectsMode: "native", GPUProfile: "native",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
	completed := now.Add(-time.Second)
	base := evidence.Run{
		SchemaVersion: evidence.SchemaVersion,
		ID:            "0123456789abcdef0123456789abcdef",
		ProfileID:     profile.ID, ProfileName: profile.Name,
		ProviderID: profile.Kernel.Provider, ProviderRevision: 1, ProviderTrust: fingerprint.TrustCustom,
		BinaryIdentity: kernel.ProviderBinaryIdentity{
			SchemaVersion: kernel.BinaryIdentitySchemaVersion,
			ProviderID:    profile.Kernel.Provider, ProviderRevision: 1, ProviderTrust: fingerprint.TrustCustom,
			BrowserVersion: profile.Kernel.Version, OperatingSystem: "windows", Architecture: "amd64",
			ExecutablePath: profile.Kernel.Executable, ExecutableSize: 10, ExecutableSHA256: strings.Repeat("a", 64),
			IntegrityStatus: kernel.StatusVerified, Provenance: "test", Reviewed: false,
		},
		BrowserVersion: profile.Kernel.Version, OperatingSystem: "windows", Architecture: "amd64",
		HarnessRevision:        evidence.HarnessRevision,
		ConsistencyInputDigest: strings.Repeat("b", 64), ConsistencyRulesRevision: "m4.3-v1",
		Status: evidence.RunPartial, StartedAt: now.Add(-2 * time.Second), CompletedAt: &completed, ExpiresAt: now.Add(time.Hour),
		Limitations: []string{"custom Provider remains unreviewed"},
	}
	return RunRequest{
		Profile:      profile,
		BaseEvidence: base,
		Session: supervisor.Session{
			ProfileID: profile.ID, ProfileName: profile.Name, State: supervisor.StateReady,
			PID: 42, CDPPort: 9222, WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test", StartedAt: now,
		},
		ProbeSet:     validProbeSet(),
		SessionReady: func() bool { return true },
	}
}
