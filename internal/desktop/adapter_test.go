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

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const adapterTestUUID = "5783a3e7-e373-51cd-8642-c83782b807c5"

func TestManagedAdapterBindingAndInUseProtection(t *testing.T) {
	service, _, credentialRecord := adapterTestService(t)
	record := importTestAdapter(t, service, adapter.KindXray)
	item := validProfile()
	item.Proxy = domain.ProxyConfig{
		URL:           "vless://proxy.example:443?security=tls&sni=proxy.example&encryption=none",
		CredentialRef: credentialRecord.ID,
		AdapterRef:    record.ID,
	}
	created, err := service.CreateProfile(item)
	if err != nil {
		t.Fatal(err)
	}
	if created.Proxy.AdapterRef != record.ID {
		t.Fatalf("adapter reference was not persisted: %#v", created.Proxy)
	}
	if err := service.DeleteAdapter(record.ID); err == nil || !strings.Contains(err.Error(), "used by profile") {
		t.Fatalf("expected in-use protection, got %v", err)
	}
	plan, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: created.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.RequiresBridge || plan.BridgeKind != adapter.KindXray {
		t.Fatalf("unexpected advanced route plan: %#v", plan)
	}
}

func TestManagedAdapterKindMismatchAndTamperAreRejected(t *testing.T) {
	service, _, credentialRecord := adapterTestService(t)
	record := importTestAdapter(t, service, adapter.KindSingBox)
	item := validProfile()
	item.Proxy = domain.ProxyConfig{
		URL:           "vless://proxy.example:443?security=tls&sni=proxy.example&encryption=none",
		CredentialRef: credentialRecord.ID,
		AdapterRef:    record.ID,
	}
	if _, err := service.CreateProfile(item); err == nil || !strings.Contains(err.Error(), "does not support") {
		t.Fatalf("expected adapter mismatch, got %v", err)
	}

	xray := importTestAdapter(t, service, adapter.KindXray)
	item.Proxy.AdapterRef = xray.ID
	created, err := service.CreateProfile(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xray.Executable, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: created.ID}); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
}

func TestXrayProviderRoutesBrowserThroughManagedRuntime(t *testing.T) {
	service, browser, credentialRecord := adapterTestService(t)
	instance := newFakeAdapterInstance("socks5://127.0.0.1:45231", adapter.KindXray)
	factory := &fakeAdapterFactory{instance: instance}
	service.adapterRuntime = factory
	record := importTestAdapter(t, service, adapter.KindXray)
	created := createAdvancedProfile(t, service, credentialRecord.ID, record.ID)

	session, err := service.StartProfile(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.State != supervisor.StateReady || factory.starts != 1 || browser.starts != 1 {
		t.Fatalf("unexpected start: session=%#v adapter=%d browser=%d", session, factory.starts, browser.starts)
	}
	if factory.request.CredentialUsername != "identity" || factory.request.CredentialSecret != adapterTestUUID {
		t.Fatal("credential material was not passed only to the adapter runtime")
	}
	joined := strings.Join(browser.plan.Args, " ")
	if !strings.Contains(joined, "--proxy-server=socks5://127.0.0.1:45231") {
		t.Fatalf("browser did not receive the adapter loopback endpoint: %s", joined)
	}
	for _, forbidden := range []string{"proxy.example", adapterTestUUID, credentialRecord.ID} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("advanced proxy material leaked into browser arguments: %s", joined)
		}
	}
	if browser.plan.ProxyDisplay != created.Proxy.URL || browser.plan.RequiresBridge {
		t.Fatalf("unexpected browser plan: %#v", browser.plan)
	}
	if _, err := service.StopProfile(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if instance.closeCount() != 1 {
		t.Fatalf("adapter runtime was not closed exactly once: %d", instance.closeCount())
	}
}

func TestAdapterRuntimeStartupFailureDoesNotStartBrowser(t *testing.T) {
	service, browser, credentialRecord := adapterTestService(t)
	service.adapterRuntime = &fakeAdapterFactory{err: errors.New("xray unavailable")}
	record := importTestAdapter(t, service, adapter.KindXray)
	created := createAdvancedProfile(t, service, credentialRecord.ID, record.ID)
	if _, err := service.StartProfile(context.Background(), created.ID); err == nil || !strings.Contains(err.Error(), "start xray adapter runtime") {
		t.Fatalf("expected adapter startup failure, got %v", err)
	}
	if browser.starts != 0 {
		t.Fatalf("browser started despite adapter failure: %d", browser.starts)
	}
}

func TestAdapterRuntimeExitStopsActiveBrowser(t *testing.T) {
	service, browser, credentialRecord := adapterTestService(t)
	instance := newFakeAdapterInstance("socks5://127.0.0.1:45232", adapter.KindXray)
	service.adapterRuntime = &fakeAdapterFactory{instance: instance}
	record := importTestAdapter(t, service, adapter.KindXray)
	created := createAdvancedProfile(t, service, credentialRecord.ID, record.ID)
	if _, err := service.StartProfile(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	instance.exit()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !browser.IsActive(created.ID) && browser.stops == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("browser stayed active after adapter exit: active=%v stops=%d", browser.IsActive(created.ID), browser.stops)
}

func adapterTestService(t *testing.T) (*Service, *adapterRuntime, credential.Record) {
	t.Helper()
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &adapterCredentialBackend{items: make(map[string]string)}
	manager, err := credential.OpenWithBackend(filepath.Join(root, "credentials.json"), backend)
	if err != nil {
		t.Fatal(err)
	}
	credentialRecord, err := manager.Save(credential.SaveRequest{
		Name: "Advanced route", Username: "identity", Secret: adapterTestUUID,
	})
	if err != nil {
		t.Fatal(err)
	}
	browser := &adapterRuntime{active: make(map[string]bool)}
	service, err := newServiceWithCredentials(store, root, browser, manager)
	if err != nil {
		t.Fatal(err)
	}
	return service, browser, credentialRecord
}

func createAdvancedProfile(t *testing.T, service *Service, credentialID, adapterID string) domain.Profile {
	t.Helper()
	kernelSource := filepath.Join(service.dataRoot, "chromium-test")
	if err := os.WriteFile(kernelSource, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	kernelRecord, err := service.ImportKernel(kernel.ImportRequest{
		Name: "Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: kernelSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	item := validProfile()
	item.Kernel = domain.KernelRef{ID: kernelRecord.ID}
	item.Proxy = domain.ProxyConfig{
		URL:           "vless://proxy.example:443?security=tls&sni=proxy.example&encryption=none",
		CredentialRef: credentialID,
		AdapterRef:    adapterID,
	}
	created, err := service.CreateProfile(item)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func importTestAdapter(t *testing.T, service *Service, kind string) adapter.Record {
	t.Helper()
	source := filepath.Join(service.dataRoot, kind+"-test")
	if err := os.WriteFile(source, []byte("adapter-"+kind), 0o700); err != nil {
		t.Fatal(err)
	}
	license := "MPL-2.0"
	if kind == adapter.KindSingBox {
		license = "GPL-3.0-or-later"
	}
	record, err := service.ImportAdapter(adapter.ImportRequest{
		Name: kind, Kind: kind, Version: "test-1", SourcePath: source,
		LicenseSPDX: license, SourceURL: "https://example.test/" + kind,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

type adapterRuntime struct {
	mu     sync.Mutex
	active map[string]bool
	plan   domain.LaunchPlan
	starts int
	stops  int
}

func (r *adapterRuntime) Start(_ context.Context, id, name string, build supervisor.PlanBuilder) (supervisor.Session, error) {
	plan, err := build(0)
	if err != nil {
		return supervisor.Session{}, err
	}
	r.mu.Lock()
	r.plan = plan
	r.starts++
	r.active[id] = true
	r.mu.Unlock()
	return supervisor.Session{ProfileID: id, ProfileName: name, State: supervisor.StateReady, PID: 88, StartedAt: time.Now().UTC()}, nil
}
func (r *adapterRuntime) Stop(_ context.Context, id string) (supervisor.Session, error) {
	r.mu.Lock()
	r.stops++
	r.active[id] = false
	r.mu.Unlock()
	return supervisor.Session{ProfileID: id, State: supervisor.StateExited}, nil
}
func (r *adapterRuntime) Shutdown(context.Context) error {
	r.mu.Lock()
	for id := range r.active {
		r.active[id] = false
	}
	r.mu.Unlock()
	return nil
}
func (r *adapterRuntime) List() []supervisor.Session { return nil }
func (r *adapterRuntime) IsActive(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[id]
}

type fakeAdapterFactory struct {
	instance *fakeAdapterInstance
	err      error
	request  adapterruntime.Request
	starts   int
}

func (f *fakeAdapterFactory) Start(_ context.Context, request adapterruntime.Request) (adapterruntime.Instance, error) {
	f.starts++
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	return f.instance, nil
}

type fakeAdapterInstance struct {
	mu     sync.Mutex
	url    string
	kind   string
	done   chan struct{}
	closed int
	once   sync.Once
}

func newFakeAdapterInstance(url, kind string) *fakeAdapterInstance {
	return &fakeAdapterInstance{url: url, kind: kind, done: make(chan struct{})}
}

func (i *fakeAdapterInstance) URL() string                  { return i.url }
func (i *fakeAdapterInstance) Kind() string                 { return i.kind }
func (i *fakeAdapterInstance) Health(context.Context) error { return nil }
func (i *fakeAdapterInstance) Done() <-chan struct{}        { return i.done }
func (i *fakeAdapterInstance) Close() error {
	i.mu.Lock()
	i.closed++
	i.mu.Unlock()
	i.once.Do(func() { close(i.done) })
	return nil
}
func (i *fakeAdapterInstance) exit() { i.once.Do(func() { close(i.done) }) }
func (i *fakeAdapterInstance) closeCount() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.closed
}

type adapterCredentialBackend struct {
	mu    sync.Mutex
	items map[string]string
}

func (b *adapterCredentialBackend) Set(service, account, secret string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items[service+"\x00"+account] = secret
	return nil
}
func (b *adapterCredentialBackend) Get(service, account string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	value, ok := b.items[service+"\x00"+account]
	if !ok {
		return "", credential.ErrSecretNotFound
	}
	return value, nil
}
func (b *adapterCredentialBackend) Delete(service, account string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.items, service+"\x00"+account)
	return nil
}
