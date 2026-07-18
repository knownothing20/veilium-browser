package evidence

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTargetClientOpensAndClosesControlledPage(t *testing.T) {
	var openedURL string
	var closedID string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/json/new":
			if request.Method != http.MethodPut {
				http.Error(writer, "wrong method", http.StatusMethodNotAllowed)
				return
			}
			decoded, err := url.QueryUnescape(request.URL.RawQuery)
			if err != nil {
				http.Error(writer, "bad query", http.StatusBadRequest)
				return
			}
			openedURL = decoded
			port := request.Host[strings.LastIndex(request.Host, ":")+1:]
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"id":"target_1","type":"page","title":"","url":%q,"webSocketDebuggerUrl":"ws://127.0.0.1:%s/devtools/page/target_1","devtoolsFrontendUrl":"/devtools/inspector.html"}`, decoded, port)
		case strings.HasPrefix(request.URL.Path, "/json/close/"):
			closedID = strings.TrimPrefix(request.URL.Path, "/json/close/")
			writer.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(writer, "Target is closing")
		default:
			http.NotFound(writer, request)
		}
	}))
	server.Listener, _ = net.Listen("tcp4", "127.0.0.1:0")
	server.Start()
	defer server.Close()
	port := server.Listener.Addr().(*net.TCPAddr).Port
	client, err := NewTargetClientWithHTTP(server.Client())
	if err != nil {
		t.Fatal(err)
	}
	controlledURL := "http://127.0.0.1:45678/run/" + strings.Repeat("a", 64)
	target, err := client.Open(context.Background(), port, controlledURL)
	if err != nil {
		t.Fatal(err)
	}
	if target.ID != "target_1" || openedURL != controlledURL {
		t.Fatalf("unexpected target %#v opened=%q", target, openedURL)
	}
	if err := client.Close(context.Background(), port, target.ID); err != nil {
		t.Fatal(err)
	}
	if closedID != target.ID {
		t.Fatalf("target close was not issued: %q", closedID)
	}
}

func TestTargetClientRejectsExternalControlledURL(t *testing.T) {
	client := NewTargetClient()
	for _, raw := range []string{
		"https://127.0.0.1:1234/run/token",
		"http://example.com:1234/run/token",
		"http://127.0.0.1:1234/other/token",
		"http://user@127.0.0.1:1234/run/token",
	} {
		if _, err := client.Open(context.Background(), 9222, raw); err == nil {
			t.Fatalf("unsafe controlled URL was accepted: %s", raw)
		}
	}
}

func TestTargetClientRejectsRedirectAndOversizedResponse(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"redirect": func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, "/other", http.StatusFound)
		},
		"oversized": func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"target_1","type":"page","padding":"`+strings.Repeat("x", maxTargetResponseBytes)+`"}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(handler)
			server.Listener, _ = net.Listen("tcp4", "127.0.0.1:0")
			server.Start()
			defer server.Close()
			client, err := NewTargetClientWithHTTP(server.Client())
			if err != nil {
				t.Fatal(err)
			}
			port := server.Listener.Addr().(*net.TCPAddr).Port
			_, err = client.Open(context.Background(), port, "http://127.0.0.1:45678/run/"+strings.Repeat("a", 64))
			if err == nil {
				t.Fatalf("expected %s response rejection", name)
			}
		})
	}
}

func TestValidateTargetRejectsNonLoopbackWebSocket(t *testing.T) {
	port := 9222
	for _, target := range []Target{
		{ID: "target_1", Type: "worker"},
		{ID: "../target", Type: "page"},
		{ID: "target_1", Type: "page", WebSocketDebuggerURL: "wss://127.0.0.1:" + strconv.Itoa(port) + "/devtools/page/target_1"},
		{ID: "target_1", Type: "page", WebSocketDebuggerURL: "ws://192.0.2.1:" + strconv.Itoa(port) + "/devtools/page/target_1"},
		{ID: "target_1", Type: "page", WebSocketDebuggerURL: "ws://127.0.0.1:9333/devtools/page/target_1"},
	} {
		if err := validateTarget(target, port); err == nil {
			t.Fatalf("unsafe target was accepted: %#v", target)
		}
	}
}

func TestTargetClientHonorsContextCancellation(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	server.Listener, _ = net.Listen("tcp4", "127.0.0.1:0")
	server.Start()
	defer server.Close()
	client, err := NewTargetClientWithHTTP(server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	port := server.Listener.Addr().(*net.TCPAddr).Port
	if _, err := client.Open(ctx, port, "http://127.0.0.1:45678/run/"+strings.Repeat("a", 64)); err == nil {
		t.Fatal("cancelled target request succeeded")
	}
}
