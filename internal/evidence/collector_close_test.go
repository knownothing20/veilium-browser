package evidence

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestCollectorCloseForceClosesActiveLoopbackRequestAtDeadline(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(writer, "finished")
	})}
	collector := &Collector{listener: listener, server: server}
	go func() { _ = server.Serve(listener) }()

	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := (&http.Client{Timeout: time.Second}).Get("http://" + listener.Addr().String())
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("active collector request did not start")
	}

	closeContext, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	startedClose := time.Now()
	closeErr := collector.Close(closeContext)
	cancel()
	if closeErr != nil {
		t.Fatalf("bounded collector close failed: %v", closeErr)
	}
	if elapsed := time.Since(startedClose); elapsed > time.Second {
		t.Fatalf("collector close exceeded bounded fallback window: %s", elapsed)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("forced collector close was not idempotent: %v", err)
	}

	close(release)
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("force-closed collector request did not terminate")
	}
}
