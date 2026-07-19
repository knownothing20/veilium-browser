package networkevidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	diagnosticStage := "validate Chromium binary"
	diagnosticDetail := ""
	var browserLogs bytes.Buffer
	defer func() {
		if t.Failed() {
			writeNetworkEvidenceCIDiagnostic(binary, diagnosticStage, diagnosticDetail, browserLogs.String())
		}
	}()
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
	command.Stdout = &browserLogs
	command.Stderr = &browserLogs
	diagnosticStage = "start Chromium"
	if err := command.Start(); err != nil {
		diagnosticDetail = err.Error()
		t.Fatal(err)
	}
	defer command.Process.Kill()
	diagnosticStage = "discover Chromium CDP port"
	port, err := discovery.Wait(ctx, profileDir)
	if err != nil {
		diagnosticDetail = err.Error()
		t.Fatal(err)
	}
	diagnosticStage = "read Chromium Browser WebSocket"
	version, err := (supervisor.HTTPProber{Client: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}, Interval: 50 * time.Millisecond}).Wait(ctx, port)
	if err != nil {
		diagnosticDetail = err.Error()
		t.Fatal(err)
	}
	diagnosticStage = "create Network Evidence browser executor"
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{CollectorFactory: func(set ProbeSet) (BrowserCollector, error) { return StartCollector(set) }})
	if err != nil {
		diagnosticDetail = err.Error()
		t.Fatal(err)
	}
	diagnosticStage = "execute controlled Network Evidence"
	result, err := (ReconcilingExecutor{Inner: executor}).Execute(ctx, ExecutionRequest{
		ProfileID: "controlled-network-profile",
		Session:   supervisor.Session{ProfileID: "controlled-network-profile", State: supervisor.StateReady, CDPPort: port, WebSocketDebuggerURL: version.WebSocketDebuggerURL, StartedAt: time.Now().UTC()},
		Route:     RouteIdentity{Kind: RouteDirect, Scheme: "direct", Digest: strings.Repeat("a", 64)}, ProbeSet: set,
	})
	if err != nil {
		diagnosticDetail = err.Error()
		t.Fatal(err)
	}
	diagnosticStage = "validate controlled Network Evidence"
	if len(result.Observations) != 1 || result.Observations[0].Status != ObservationPassed || result.Observations[0].Values[0] != "203.0.113.8" {
		diagnosticDetail = fmt.Sprintf("%#v", result)
		t.Fatalf("unexpected network evidence: %#v", result)
	}
}

func writeNetworkEvidenceCIDiagnostic(binary, stage, detail, browserLog string) {
	if runtime.GOOS != "windows" {
		return
	}
	runnerTemp := strings.TrimSpace(os.Getenv("RUNNER_TEMP"))
	if runnerTemp == "" {
		return
	}
	payload := struct {
		Binary     string `json:"binary"`
		Stage      string `json:"stage"`
		Detail     string `json:"detail"`
		BrowserLog string `json:"browserLog"`
	}{Binary: binary, Stage: stage, Detail: detail, BrowserLog: boundedNetworkCILog(browserLog)}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(runnerTemp, "reviewed-chromium-evidence-packet.json"), data, 0o600)
}

func boundedNetworkCILog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 4096 {
		return value[:4096] + "..."
	}
	return value
}
