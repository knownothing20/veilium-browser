package supervisor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHTTPProberReadsLocalVersionEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Browser":"Chrome/148","webSocketDebuggerUrl":"ws://127.0.0.1:%d/devtools/browser/test"}`, port)
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	prober := HTTPProber{Client: &http.Client{Timeout: time.Second}, Interval: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	version, err := prober.Wait(ctx, port)
	if err != nil {
		t.Fatal(err)
	}
	if version.Browser != "Chrome/148" {
		t.Fatalf("unexpected version: %#v", version)
	}
}
