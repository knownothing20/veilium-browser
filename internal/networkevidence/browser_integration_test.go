package networkevidence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestRealChromiumNetworkEvidence(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("VEILIUM_CHROMIUM_BINARY"))
	if binary == "" {
		t.Skip("VEILIUM_CHROMIUM_BINARY is not configured")
	}
	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ip":"203.0.113.8"}`))
	}))
	defer probe.Close()
	set := ProbeSet{SchemaVersion: 1, ID: "controlled-exit", Revision: 1, Definitions: []ProbeDefinition{{
		SchemaVersion: 1, ID: "exit", Revision: 1, Kind: ProbeExitIP, HTTPSURL: probe.URL,
		TimeoutSeconds: 10, MaxResponseBytes: 4096, SelfHostable: true,
		PrivacyNote: "Returns only a synthetic public IP for the controlled CI request.",
	}}}
	profileDir := filepath.Join(t.TempDir(), "profile")
	if err := os.Mkdir(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	discovery := supervisor.DevToolsActivePortDiscovery{Interval: 50 * time.Millisecond}
	if err := discovery.Prepare(profileDir); err != nil {
		t.Fatal(err)
	}
	args := []string{"--headless=new", "--disable-gpu", "--disable-background-networking", "--no-first-run", "--no-proxy-server", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--user-data-dir=" + profileDir, "about:blank"}
	if runtime.GOOS == "linux" {
		args = append([]string{"--no-sandbox", "--disable-dev-shm-usage"}, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer command.Process.Kill()
	port, err := discovery.Wait(ctx, profileDir)
	if err != nil {
		t.Fatal(err)
	}
	version, err := (supervisor.HTTPProber{Client: &http.Client{Timeout: 2 * time.Second}, Interval: 50 * time.Millisecond}).Wait(ctx, port)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{CollectorFactory: func(set ProbeSet) (BrowserCollector, error) { return StartCollector(set) }})
	if err != nil {
		t.Fatal(err)
	}
	result, err := (ReconcilingExecutor{Inner: executor}).Execute(ctx, ExecutionRequest{
		ProfileID: "controlled-network-profile",
		Session: supervisor.Session{ProfileID: "controlled-network-profile", State: supervisor.StateReady, CDPPort: port, WebSocketDebuggerURL: version.WebSocketDebuggerURL, StartedAt: time.Now().UTC()},
		Route: RouteIdentity{Kind: RouteDirect, Scheme: "direct", Digest: strings.Repeat("a", 64)}, ProbeSet: set,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 || result.Observations[0].Status != ObservationPassed || result.Observations[0].Values[0] != "203.0.113.8" {
		t.Fatalf("unexpected network evidence: %#v", result)
	}
}
