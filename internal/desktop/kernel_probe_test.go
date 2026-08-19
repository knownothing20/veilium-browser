package desktop

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"testing"
)

const testKernelProbeEnvironment = "VEILIUM_TEST_KERNEL_PROBE"

func TestMain(main *testing.M) {
	if os.Getenv(testKernelProbeEnvironment) == "1" {
		runDesktopTestKernelProbe()
	}
	os.Exit(main.Run())
}

func runDesktopTestKernelProbe() {
	profileDir := ""
	for _, argument := range os.Args {
		if strings.HasPrefix(argument, "--user-data-dir=") {
			profileDir = strings.TrimPrefix(argument, "--user-data-dir=")
		}
	}
	if profileDir == "" {
		_, _ = fmt.Fprint(os.Stderr, "missing test probe user data directory")
		os.Exit(2)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	path := "/devtools/browser/test-desktop-kernel-probe"
	payload := []byte(strconv.Itoa(port) + "\n" + path + "\n")
	if err := os.WriteFile(profileDir+string(os.PathSeparator)+"DevToolsActivePort", payload, 0o600); err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]string{
			"Browser":              "Test Chromium/148.0.0",
			"webSocketDebuggerUrl": "ws://127.0.0.1:" + strconv.Itoa(port) + path,
		})
	})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		_ = listener.Close()
	}()
	if err := http.Serve(listener, mux); err != nil && err != net.ErrClosed {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func testKernelExecutable(t *testing.T) string {
	t.Helper()
	t.Setenv(testKernelProbeEnvironment, "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return executable
}
