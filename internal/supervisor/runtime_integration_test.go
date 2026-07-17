package supervisor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestRealRuntimeHandshakeAndStop(t *testing.T) {
	userDataDir := t.TempDir()
	logDir := t.TempDir()
	supervisor, err := New(logDir)
	if err != nil {
		t.Fatal(err)
	}
	build := func(port int) (domain.LaunchPlan, error) {
		return domain.LaunchPlan{
			Executable: os.Args[0],
			Args: []string{
				"-test.run=^TestRuntimeHelperProcess$",
				"--",
				"--user-data-dir=" + userDataDir,
				"--remote-debugging-address=127.0.0.1",
				fmt.Sprintf("--remote-debugging-port=%d", port),
			},
			Environment: map[string]string{
				"VEILIUM_RUNTIME_HELPER":    "1",
				"VEILIUM_RUNTIME_USER_DATA": userDataDir,
			},
		}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := supervisor.Start(ctx, "integration-profile", "Integration Profile", build)
	if err != nil {
		t.Fatal(err)
	}
	if session.State != StateReady || session.CDPPort < 1 || session.Browser != "VeiliumTestBrowser/1" {
		t.Fatalf("unexpected ready session: %#v", session)
	}
	stopContext, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	stopped, err := supervisor.Stop(stopContext, session.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != StateExited {
		t.Fatalf("unexpected stopped session: %#v", stopped)
	}
}

func TestRuntimeHelperProcess(t *testing.T) {
	if os.Getenv("VEILIUM_RUNTIME_HELPER") != "1" {
		return
	}
	userDataDir := os.Getenv("VEILIUM_RUNTIME_USER_DATA")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		os.Exit(31)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	activePort := fmt.Sprintf("%d\n/devtools/browser/veilium-test\n", port)
	if err := os.WriteFile(filepath.Join(userDataDir, devToolsActivePortFilename), []byte(activePort), 0o600); err != nil {
		_ = listener.Close()
		os.Exit(32)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"Browser":"VeiliumTestBrowser/1","webSocketDebuggerUrl":"ws://127.0.0.1:%d/devtools/browser/veilium-test"}`, port)
	})}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		os.Exit(33)
	}
}
