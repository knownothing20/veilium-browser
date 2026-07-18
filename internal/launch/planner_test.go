package launch

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestBuildProducesStableSafeLaunchPlan(t *testing.T) {
	item := testProfile()
	plan, err := (Planner{}).Build(item, 9222)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(plan.Args, " ")
	for _, required := range []string{
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=9222",
		"--proxy-server=http://127.0.0.1:8080",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %s", required, joined)
		}
	}
	for _, blocked := range []string{"--fingerprint=", "--fingerprint-brand=", "--fingerprint-hardware-concurrency="} {
		if strings.Contains(joined, blocked) {
			t.Fatalf("generic provider emitted unverified flag %q in %s", blocked, joined)
		}
	}
}

func TestBuildSupportsChromiumAssignedDebuggingPort(t *testing.T) {
	plan, err := (Planner{}).Build(testProfile(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(plan.Args, " "), "--remote-debugging-port=0") {
		t.Fatalf("dynamic CDP port was not requested: %#v", plan.Args)
	}
}

func testProfile() domain.Profile {
	return domain.Profile{
		ID:          "stable-profile-id",
		Name:        "Stable profile",
		UserDataDir: "/tmp/veilium/profile",
		Kernel: domain.KernelRef{
			Provider:   fingerprint.ProviderCustom,
			Version:    "148.0.0",
			Executable: "/opt/chrome",
		},
		Fingerprint: domain.FingerprintConfig{
			Platform: "linux", Brand: "Chromium", Language: "en-US",
			Timezone: "UTC", ScreenWidth: 1366, ScreenHeight: 768,
			WebRTCPolicy: "proxy-only",
			CanvasMode: "native", AudioMode: "native", FontMode: "native",
			ClientRectsMode: "native", GPUProfile: "auto",
		},
		Proxy: domain.ProxyConfig{URL: "http://127.0.0.1:8080"},
	}
}
