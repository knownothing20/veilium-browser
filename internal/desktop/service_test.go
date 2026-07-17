package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestCreateUpdateCloneLifecycle(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.UserDataDir == "" {
		t.Fatalf("expected generated identity and data directory: %#v", created)
	}
	created.Group = "Work"
	created.Tags = []string{"Commerce", "commerce", " US "}
	updated, err := service.UpdateProfile(created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Group != "Work" || len(updated.Tags) != 2 {
		t.Fatalf("unexpected update result: %#v", updated)
	}
	cloned, err := service.CloneProfile(updated.ID, "Store B")
	if err != nil {
		t.Fatal(err)
	}
	if cloned.ID == updated.ID || cloned.UserDataDir == updated.UserDataDir {
		t.Fatalf("clone did not receive isolated identity: %#v", cloned)
	}
	if len(service.ListProfiles()) != 2 {
		t.Fatalf("expected two profiles")
	}
}

func TestCapabilitiesRejectUnknownProvider(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, _ := NewService(store, root)
	if _, err := service.Capabilities("unknown", "148.0.0"); err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestKernelRegistryProtectsProfilesAndLaunchPlans(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, _ := NewService(store, root)
	record, err := service.ImportKernel(kernel.ImportRequest{Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Kernel.Executable != record.Executable || created.Kernel.Provider != record.Provider {
		t.Fatalf("kernel was not resolved: %#v", created.Kernel)
	}
	if err := service.DeleteKernel(record.ID); err == nil || !strings.Contains(err.Error(), "used by profile") {
		t.Fatalf("expected in-use protection, got %v", err)
	}
	if err := os.WriteFile(record.Executable, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: created.ID}); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity failure, got %v", err)
	}
}

func validProfile() domain.Profile {
	return domain.Profile{
		Name: "Store A",
		Kernel: domain.KernelRef{Provider: fingerprint.ProviderPatched, Version: "148.0.0", Executable: `C:\Browsers\Chromium\chrome.exe`},
		Fingerprint: domain.FingerprintConfig{Platform: "windows", Brand: "Chrome", Language: "en-US", Timezone: "America/Los_Angeles", ScreenWidth: 1920, ScreenHeight: 1080, HardwareConcurrency: 8, WebRTCPolicy: "proxy-only", CanvasMode: "seeded", AudioMode: "seeded", FontMode: "seeded", ClientRectsMode: "seeded", GPUProfile: "auto"},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
}
