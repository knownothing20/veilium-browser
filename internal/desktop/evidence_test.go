package desktop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestRunEvidenceRequiresReadyManagedSession(t *testing.T) {
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
	if err := os.WriteFile(source, []byte("evidence-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	record, err := service.ImportKernel(kernel.ImportRequest{
		Name:       "Evidence Chromium",
		Provider:   fingerprint.ProviderCustom,
		Version:    "148.0.0",
		SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := genericEvidenceProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.RunEvidence(context.Background(), created.ID)
	if err == nil || !strings.Contains(err.Error(), "ready managed browser session") {
		t.Fatalf("expected ready-session rejection, got %v", err)
	}
}

func TestListEvidenceStartsEmpty(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	runs, err := service.ListEvidence("profile-with-no-runs")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected no evidence runs, got %#v", runs)
	}
}

func genericEvidenceProfile() domain.Profile {
	return domain.Profile{
		Name: "Evidence Profile",
		Fingerprint: domain.FingerprintConfig{
			Platform:            "windows",
			Brand:               "Chromium",
			Language:            "en-US",
			Timezone:            "America/Los_Angeles",
			ScreenWidth:         1920,
			ScreenHeight:        1080,
			HardwareConcurrency: 0,
			WebRTCPolicy:        "proxy-only",
			CanvasMode:          "native",
			AudioMode:           "native",
			FontMode:            "native",
			ClientRectsMode:     "native",
			GPUProfile:          "native",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
}
