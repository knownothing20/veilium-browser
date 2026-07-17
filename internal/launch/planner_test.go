package launch

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestBuildProducesStableSafeLaunchPlan(t *testing.T) {
	item := domain.Profile{
		ID:          "stable-profile-id",
		Name:        "Stable profile",
		UserDataDir: "/tmp/veilium/profile",
		Kernel: domain.KernelRef{
			Provider:   fingerprint.ProviderPatched,
			Version:    "144.0.7559.132",
			Executable: "/opt/chrome",
		},
		Fingerprint: domain.FingerprintConfig{
			Platform: "linux", Brand: "Chrome", Language: "en-US",
			Timezone: "UTC", ScreenWidth: 1366, ScreenHeight: 768,
			HardwareConcurrency: 4, WebRTCPolicy: "proxy-only",
			CanvasMode: "seeded", AudioMode: "seeded", FontMode: "seeded",
			ClientRectsMode: "seeded", GPUProfile: "auto",
		},
		Proxy: domain.ProxyConfig{URL: "http://127.0.0.1:8080"},
	}
	plan, err := (Planner{}).Build(item, 9222)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(plan.Args, " ")
	for _, required := range []string{
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=9222",
		"--fingerprint=",
		"--proxy-server=http://127.0.0.1:8080",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %s", required, joined)
		}
	}
}
