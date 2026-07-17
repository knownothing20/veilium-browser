package desktop

import (
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
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

func validProfile() domain.Profile {
	return domain.Profile{
		Name: "Store A",
		Kernel: domain.KernelRef{
			Provider:   fingerprint.ProviderPatched,
			Version:    "148.0.0",
			Executable: `C:\Browsers\Chromium\chrome.exe`,
		},
		Fingerprint: domain.FingerprintConfig{
			Platform:            "windows",
			Brand:               "Chrome",
			Language:            "en-US",
			Timezone:            "America/Los_Angeles",
			ScreenWidth:         1920,
			ScreenHeight:        1080,
			HardwareConcurrency: 8,
			WebRTCPolicy:        "proxy-only",
			CanvasMode:          "seeded",
			AudioMode:           "seeded",
			FontMode:            "seeded",
			ClientRectsMode:     "seeded",
			GPUProfile:          "auto",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
}
