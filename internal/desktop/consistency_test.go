package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/consistency"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestProfileConsistencyDerivesDegradedCustomHealth(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("consistency-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	record, err := service.ImportKernel(kernel.ImportRequest{
		Name: "Consistency Chromium", Provider: fingerprint.ProviderCustom,
		Version: "148.0.0", SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(domain.Profile{
		Name: "Consistency Profile",
		Kernel: domain.KernelRef{ID: record.ID},
		Fingerprint: domain.FingerprintConfig{
			Platform: runtime.GOOS, Brand: "Chromium", Language: "en-US", Timezone: "UTC",
			ScreenWidth: 1920, ScreenHeight: 1080, WindowWidth: 1280, WindowHeight: 800, DeviceScaleFactor: 1,
			WebRTCPolicy: "proxy-only", CanvasMode: "native", AudioMode: "native",
			FontMode: "native", ClientRectsMode: "native", GPUProfile: "native",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ProfileConsistency(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != consistency.HealthDegraded {
		t.Fatalf("custom provider without reviewed evidence must be degraded: %#v", result)
	}
	if result.Window.Width != 1280 || result.Window.Height != 800 || result.Window.Source != consistency.WindowExplicit {
		t.Fatalf("unexpected effective window: %#v", result.Window)
	}
}

func TestProfileConsistencyRequiresManagedKernel(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(domain.Profile{
		ID: "manual-profile", Name: "Manual profile",
		Kernel: domain.KernelRef{Provider: fingerprint.ProviderCustom, Version: "148.0.0", Executable: "/manual/chrome"},
		Fingerprint: domain.FingerprintConfig{
			Platform: runtime.GOOS, Brand: "Chromium", Language: "en-US", Timezone: "UTC",
			ScreenWidth: 1920, ScreenHeight: 1080,
			WebRTCPolicy: "proxy-only", CanvasMode: "native", AudioMode: "native",
			FontMode: "native", ClientRectsMode: "native", GPUProfile: "native",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProfileConsistency(created.ID); err == nil {
		t.Fatal("expected managed-kernel requirement")
	}
}
