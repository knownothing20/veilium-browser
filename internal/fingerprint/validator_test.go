package fingerprint

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestAcceptsGenericCustomProfileWithoutReviewedClaims(t *testing.T) {
	profile := genericProfile()
	warnings, err := Validate(profile)
	if err != nil {
		t.Fatalf("validate generic profile: %v", err)
	}
	if len(warnings) == 0 || !containsWarning(warnings, "custom Chromium") {
		t.Fatalf("expected custom-provider warning, got %v", warnings)
	}
}

func TestRejectsLegacyPatchedAdvancedClaims(t *testing.T) {
	profile := genericProfile()
	profile.Kernel.Provider = ProviderPatched
	profile.Fingerprint.Brand = "Chrome"
	profile.Fingerprint.Seed = "legacy-seed"
	profile.Fingerprint.CanvasMode = "seeded"
	_, err := Validate(profile)
	if err == nil || !strings.Contains(err.Error(), "capability is unverified") {
		t.Fatalf("expected legacy capability rejection, got %v", err)
	}
}

func TestRejectsCustomHardwareOverride(t *testing.T) {
	profile := genericProfile()
	profile.Fingerprint.HardwareConcurrency = 8
	_, err := Validate(profile)
	if err == nil || !strings.Contains(err.Error(), "hardware-concurrency override") {
		t.Fatalf("expected hardware capability error, got %v", err)
	}
}

func TestRejectsCustomGPUWithoutReviewedCapability(t *testing.T) {
	profile := genericProfile()
	profile.Fingerprint.GPUProfile = "custom"
	profile.Fingerprint.GPUVendor = "Example"
	profile.Fingerprint.GPURenderer = "Example GPU"
	_, err := Validate(profile)
	if err == nil || !strings.Contains(err.Error(), "custom GPU metadata") {
		t.Fatalf("expected custom GPU capability error, got %v", err)
	}
}

func TestRejectsSeededModesWithoutSeed(t *testing.T) {
	profile := genericProfile()
	profile.Kernel.Provider = ProviderPatched
	profile.Fingerprint.CanvasMode = "seeded"
	_, err := Validate(profile)
	if err == nil || !strings.Contains(err.Error(), "non-empty seed") {
		t.Fatalf("expected missing seed error, got %v", err)
	}
}

func genericProfile() domain.Profile {
	return domain.Profile{
		Name:        "Account A",
		UserDataDir: "/tmp/veilium/a",
		Kernel: domain.KernelRef{
			Provider:   ProviderCustom,
			Version:    "148.0.0",
			Executable: "/opt/chromium/chrome",
		},
		Fingerprint: domain.FingerprintConfig{
			Platform:        "linux",
			Brand:           "Chromium",
			Language:        "en-US",
			Timezone:        "America/Los_Angeles",
			ScreenWidth:     1920,
			ScreenHeight:    1080,
			WebRTCPolicy:    "proxy-only",
			CanvasMode:      "native",
			AudioMode:       "native",
			FontMode:        "native",
			ClientRectsMode: "native",
			GPUProfile:      "auto",
		},
	}
}

func containsWarning(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}
	return false
}
