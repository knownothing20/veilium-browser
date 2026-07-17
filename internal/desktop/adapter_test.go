package desktop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestManagedAdapterBindingAndInUseProtection(t *testing.T) {
	service, _, credentialRecord := adapterTestService(t)
	record := importTestAdapter(t, service, adapter.KindXray)
	item := validProfile()
	item.Proxy = domain.ProxyConfig{URL: "vless://proxy.example:443", CredentialRef: credentialRecord.ID, AdapterRef: record.ID}
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
	item.Proxy = domain.ProxyConfig{URL: "vless://proxy.example:443", CredentialRef: credentialRecord.ID, AdapterRef: record.ID}
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

func TestAdvancedRuntimeStaysBlockedUntilProviderExists(t *testing.T) {
	service, runtime, credentialRecord := adapterTestService(t)
	record := importTestAdapter(t, service, adapter.KindXray)
	kernelSource := filepath.Join(service.dataRoot, "chromium-test")
	if err := os.WriteFile(kernelSource, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	kernelRecord, err := service.ImportKernel(kernel.ImportRequest{Name: "Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: kernelSource})
	if err != nil {
		t.Fatal(err)
	}
	item := validProfile()
	item.Kernel = domain.KernelRef{ID: kernelRecord.ID}
	item.Proxy = domain.ProxyConfig{URL: "vless://proxy.example:443", CredentialRef: credentialRecord.ID, AdapterRef: record.ID}
	created, err := service.CreateProfile(item)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartProfile(context.Background(), created.ID); err == nil || !strings.Contains(err.Error(), "configuration provider is unavailable") {
		t.Fatalf("expected explicit provider boundary, got %v", err)
	}
	if runtime.starts != 0 {
		t.Fatalf("browser started before adapter provider existed: %d", runtime.starts)
	}
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
	credentialRecord, err := manager.Save(credential.SaveRequest{Name: "Advanced route", Username: "identity", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &adapterRuntime{}
	service, err := newServiceWithCredentials(store, root, runtime, manager)
	if err != nil {
		t.Fatal(err)
	}
	return service, runtime, credentialRecord
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

type adapterRuntime struct{ starts int }

func (r *adapterRuntime) Start(context.Context, string, string, supervisor.PlanBuilder) (supervisor.Session, error) {
	r.starts++
	return supervisor.Session{}, errors.New("unexpected browser start")
}
func (r *adapterRuntime) Stop(context.Context, string) (supervisor.Session, error) {
	return supervisor.Session{}, nil
}
func (r *adapterRuntime) Shutdown(context.Context) error { return nil }
func (r *adapterRuntime) List() []supervisor.Session     { return nil }
func (r *adapterRuntime) IsActive(string) bool           { return false }

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
