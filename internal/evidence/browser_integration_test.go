package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
	"github.com/knownothing20/veilium-browser/internal/windowcontrol"
)

func TestRealChromiumEvidenceCollector(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("VEILIUM_CHROMIUM_BINARY"))
	diagnosticStage := "validate Chromium binary"
	var logs bytes.Buffer
	defer func() {
		if t.Failed() {
			writeRealBrowserCIDiagnostic(binary, diagnosticStage, logs.String())
		}
	}()
	if binary == "" {
		t.Skip("VEILIUM_CHROMIUM_BINARY is not configured")
	}
	info, err := os.Stat(binary)
	if err != nil || info.IsDir() {
		t.Fatalf("invalid Chromium binary %q: %v", binary, err)
	}

	userDataDir := filepath.Join(t.TempDir(), "chromium-profile")
	if err := os.Mkdir(userDataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	discovery := supervisor.DevToolsActivePortDiscovery{Interval: 50 * time.Millisecond}
	if err := discovery.Prepare(userDataDir); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--disable-background-networking",
		"--disable-component-update",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-sync",
		"--metrics-recording-only",
		"--no-default-browser-check",
		"--no-first-run",
		"--no-proxy-server",
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"--user-data-dir=" + userDataDir,
		"--window-size=1024,768",
		"about:blank",
	}
	if runtime.GOOS == "linux" {
		args = append([]string{"--no-sandbox"}, args...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	command.Stdout = &logs
	command.Stderr = &logs
	diagnosticStage = "start Chromium"
	if err := command.Start(); err != nil {
		t.Fatalf("start Chromium: %v", err)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- command.Wait() }()
	defer func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(5 * time.Second):
			t.Log("Chromium did not exit within cleanup timeout")
		}
	}()

	diagnosticStage = "discover Chromium CDP port"
	portContext, portCancel := context.WithTimeout(ctx, 30*time.Second)
	port, err := discovery.Wait(portContext, userDataDir)
	portCancel()
	if err != nil {
		t.Fatalf("discover Chromium CDP port: %v\n%s", err, boundedLog(logs.String()))
	}

	diagnosticStage = "read Chromium Browser WebSocket"
	prober := supervisor.HTTPProber{
		Client:   &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}},
		Interval: 50 * time.Millisecond,
	}
	version, err := prober.Wait(ctx, port)
	if err != nil {
		t.Fatalf("read Chromium Browser WebSocket: %v\n%s", err, boundedLog(logs.String()))
	}

	diagnosticStage = "start controlled evidence collector"
	collector, err := StartCollector(CollectorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	collectorClosed := false
	defer func() {
		if !collectorClosed {
			closeContext, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = collector.Close(closeContext)
			closeCancel()
		}
	}()

	diagnosticStage = "open controlled Chromium target"
	targetClient := NewTargetClient()
	target, err := targetClient.Open(ctx, port, collector.URL())
	if err != nil {
		t.Fatalf("open controlled Chromium target: %v\n%s", err, boundedLog(logs.String()))
	}
	defer func() {
		closeContext, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = targetClient.Close(closeContext, port, target.ID)
		closeCancel()
	}()

	diagnosticStage = "apply real Chromium managed window"
	windowPlan := domain.WindowPlan{Width: 1024, Height: 768, DeviceScaleFactor: 1, Source: domain.WindowSourceExplicit}
	windowState, err := windowcontrol.New().Apply(ctx, port, version.WebSocketDebuggerURL, windowPlan)
	if err != nil {
		t.Fatalf("apply real Chromium managed window: %v\n%s", err, boundedLog(logs.String()))
	}
	if !windowState.Applied || !withinWindowTolerance(windowState.Width, windowPlan.Width) || !withinWindowTolerance(windowState.Height, windowPlan.Height) {
		t.Fatalf("unexpected real Chromium window state: %#v", windowState)
	}

	diagnosticStage = "wait for real-browser evidence"
	submissionContext, submissionCancel := context.WithTimeout(ctx, 15*time.Second)
	submission, err := collector.Wait(submissionContext)
	submissionCancel()
	if err != nil {
		t.Fatalf("wait for real-browser evidence: %v\n%s", err, boundedLog(logs.String()))
	}
	diagnosticStage = "validate real-browser evidence"
	if err := submission.Validate(); err != nil {
		t.Fatalf("validate real-browser evidence: %v", err)
	}

	diagnosticStage = "validate required evidence contexts"
	contexts := make(map[BrowserContext]BrowserSnapshot, len(submission.Contexts))
	for _, snapshot := range submission.Contexts {
		contexts[snapshot.Context] = snapshot
	}
	for _, required := range []BrowserContext{ContextTopLevel, ContextIframe, ContextWorker} {
		snapshot, ok := contexts[required]
		if !ok {
			t.Fatalf("missing %s evidence context; limitations=%v", required, submission.Limitations)
		}
		if strings.TrimSpace(snapshot.UserAgent) == "" || strings.TrimSpace(snapshot.Language) == "" || strings.TrimSpace(snapshot.Timezone) == "" {
			t.Fatalf("incomplete %s identity snapshot: %#v", required, snapshot)
		}
	}

	diagnosticStage = "close evidence collector"
	closeContext, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := collector.Close(closeContext); err != nil {
		closeCancel()
		t.Fatalf("close evidence collector: %v", err)
	}
	closeCancel()
	collectorClosed = true
}

func writeRealBrowserCIDiagnostic(binary, stage, browserLog string) {
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
		BrowserLog string `json:"browserLog"`
	}{Binary: binary, Stage: stage, BrowserLog: boundedLog(browserLog)}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(runnerTemp, "reviewed-chromium-evidence-packet.json"), data, 0o600)
}

func withinWindowTolerance(observed, expected int) bool {
	difference := observed - expected
	if difference < 0 {
		difference = -difference
	}
	return difference <= 2
}

func boundedLog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 4096 {
		return fmt.Sprintf("%s...", value[:4096])
	}
	return value
}
