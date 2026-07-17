package desktop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/proxybridge"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestCredentialBackedProfileUsesLoopbackBridgeAndClosesIt(t *testing.T) {
	service, runtime, factory, record := bridgeTestService(t)
	created := createBridgeProfile(t, service, record.ID)
	session, err := service.StartProfile(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.State != supervisor.StateReady || factory.started != 1 {
		t.Fatalf("unexpected start result: %#v starts=%d", session, factory.started)
	}
	joined := strings.Join(runtime.plan.Args, " ")
	if !strings.Contains(joined, "--proxy-server=http://127.0.0.1:43123") {
		t.Fatalf("browser did not receive the loopback bridge: %s", joined)
	}
	for _, forbidden := range []string{"proxy.example", "alice", "top-secret"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("sensitive upstream material leaked into launch args: %s", joined)
		}
	}
	if runtime.plan.RequiresBridge {
		t.Fatal("prepared runtime plan still requires a bridge")
	}
	if _, err := service.StopProfile(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if factory.instance.closeCount() != 1 {
		t.Fatalf("bridge was not closed exactly once: %d", factory.instance.closeCount())
	}
}

func TestBridgeClosesAfterNaturalBrowserExit(t *testing.T) {
	service, runtime, factory, record := bridgeTestService(t)
	created := createBridgeProfile(t, service, record.ID)
	if _, err := service.StartProfile(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	runtime.setActive(created.ID, false)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if factory.instance.closeCount() == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("bridge remained open after browser exit")
}

func TestBridgeStartupFailureDoesNotStartBrowser(t *testing.T) {
	service, runtime, _, record := bridgeTestService(t)
	setProxyBridgeFactory(service, failingBridgeFactory{})
	created := createBridgeProfile(t, service, record.ID)
	if _, err := service.StartProfile(context.Background(), created.ID); err == nil || !strings.Contains(err.Error(), "start authenticated proxy bridge") {
		t.Fatalf("expected bridge startup failure, got %v", err)
	}
	if runtime.starts != 0 {
		t.Fatalf("browser started despite bridge failure: %d", runtime.starts)
	}
}

func bridgeTestService(t *testing.T) (*Service, *bridgeRuntime, *fakeBridgeFactory, credential.Record) {
	t.Helper()
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &bridgeCredentialBackend{items: make(map[string]string)}
	manager, err := credential.OpenWithBackend(filepath.Join(root, "credentials.json"), backend)
	if err != nil {
		t.Fatal(err)
	}
	record, err := manager.Save(credential.SaveRequest{Name: "Authenticated proxy", Username: "alice", Secret: "top-secret"})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &bridgeRuntime{active: make(map[string]bool)}
	service, err := newServiceWithCredentials(store, root, runtime, manager)
	if err != nil {
		t.Fatal(err)
	}
	factory := &fakeBridgeFactory{instance: &fakeBridge{url: "http://127.0.0.1:43123", kind: "http-auth"}}
	setProxyBridgeFactory(service, factory)
	return service, runtime, factory, record
}

func createBridgeProfile(t *testing.T, service *Service, credentialID string) domain.Profile {
	t.Helper()
	root := service.dataRoot
	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	kernelRecord, err := service.ImportKernel(kernel.ImportRequest{Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	item := validProfile()
	item.Kernel = domain.KernelRef{ID: kernelRecord.ID}
	item.Proxy = domain.ProxyConfig{URL: "http://proxy.example:3128", CredentialRef: credentialID}
	created, err := service.CreateProfile(item)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

type bridgeRuntime struct {
	mu     sync.Mutex
	active map[string]bool
	plan   domain.LaunchPlan
	starts int
}

func (r *bridgeRuntime) Start(_ context.Context, id, name string, build supervisor.PlanBuilder) (supervisor.Session, error) {
	plan, err := build(0)
	if err != nil {
		return supervisor.Session{}, err
	}
	r.mu.Lock()
	r.plan = plan
	r.starts++
	r.active[id] = true
	r.mu.Unlock()
	return supervisor.Session{ProfileID: id, ProfileName: name, State: supervisor.StateReady, PID: 99, StartedAt: time.Now().UTC()}, nil
}
func (r *bridgeRuntime) Stop(_ context.Context, id string) (supervisor.Session, error) {
	r.setActive(id, false)
	return supervisor.Session{ProfileID: id, State: supervisor.StateExited}, nil
}
func (r *bridgeRuntime) Shutdown(context.Context) error {
	r.mu.Lock()
	for id := range r.active {
		r.active[id] = false
	}
	r.mu.Unlock()
	return nil
}
func (r *bridgeRuntime) List() []supervisor.Session { return nil }
func (r *bridgeRuntime) IsActive(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[id]
}
func (r *bridgeRuntime) setActive(id string, active bool) {
	r.mu.Lock()
	r.active[id] = active
	r.mu.Unlock()
}

type fakeBridgeFactory struct {
	instance *fakeBridge
	started  int
}

func (f *fakeBridgeFactory) Start(_ context.Context, upstream string, material credential.Material) (proxybridge.Instance, error) {
	if upstream != "http://proxy.example:3128" || material.Username != "alice" || material.Secret != "top-secret" {
		return nil, errors.New("unexpected bridge material")
	}
	f.started++
	return f.instance, nil
}

type failingBridgeFactory struct{}

func (failingBridgeFactory) Start(context.Context, string, credential.Material) (proxybridge.Instance, error) {
	return nil, errors.New("bridge unavailable")
}

type fakeBridge struct {
	mu     sync.Mutex
	url    string
	kind   string
	closed int
}

func (b *fakeBridge) URL() string                  { return b.url }
func (b *fakeBridge) Kind() string                 { return b.kind }
func (b *fakeBridge) Health(context.Context) error { return nil }
func (b *fakeBridge) Close() error                 { b.mu.Lock(); defer b.mu.Unlock(); b.closed++; return nil }
func (b *fakeBridge) closeCount() int              { b.mu.Lock(); defer b.mu.Unlock(); return b.closed }

type bridgeCredentialBackend struct {
	mu    sync.Mutex
	items map[string]string
}

func (b *bridgeCredentialBackend) Set(service, account, secret string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items[service+"\x00"+account] = secret
	return nil
}
func (b *bridgeCredentialBackend) Get(service, account string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	value, ok := b.items[service+"\x00"+account]
	if !ok {
		return "", credential.ErrSecretNotFound
	}
	return value, nil
}
func (b *bridgeCredentialBackend) Delete(service, account string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.items, service+"\x00"+account)
	return nil
}
