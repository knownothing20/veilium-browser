package fingerprint

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestRejectsRemovedCustomGPUContract(t *testing.T) {
	profile := validProfile()
	profile.Kernel.Version = "144.0.7559.132"
	profile.Fingerprint.GPUProfile = "custom"
	profile.Fingerprint.GPUVendor = "Example"
	profile.Fingerprint.GPURenderer = "Example GPU"
	_, err := Validate(profile)
	if err == nil || !strings.Contains(err.Error(), "does not support custom GPU") {
		t.Fatalf("expected custom GPU capability error, got %v", err)
	}
}

func TestAcceptsConsistentPatchedProfile(t *testing.T) {
	profile := validProfile()
	warnings, err := Validate(profile)
	if err != nil {
		t.Fatalf("validate profile: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
}

func validProfile() domain.Profile {
	return domain.Profile{
		Name:        "Account A",
		UserDataDir: "/tmp/veilium/a",
		Kernel: domain.KernelRef{
			Provider:   fingerprintProvider(),
			Version:    "144.0.7559.132",
			Executable: "/opt/chromium/chrome",
		},
		Fingerprint: domain.FingerprintConfig{
			Seed:                "123456",
			Platform:            "linux",
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
	}
}

func fingerprintProvider() string { return ProviderPatched }
