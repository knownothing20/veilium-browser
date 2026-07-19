package windowcontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestApplyUsesOnlyControlledWindowCommands(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	var mu sync.Mutex
	methods := make([]string, 0, 3)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `[{"id":"page-1","type":"page"}]`)
	})
	mux.HandleFunc("/devtools/browser/test", func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		for count := 0; count < 3; count++ {
			var message cdpRequest
			if err := connection.ReadJSON(&message); err != nil {
				return
			}
			mu.Lock()
			methods = append(methods, message.Method)
			mu.Unlock()
			var result any = map[string]any{}
			switch message.Method {
			case "Browser.getWindowForTarget":
				result = map[string]any{"windowId": 7, "bounds": map[string]any{"width": 900, "height": 700, "windowState": "normal"}}
			case "Browser.getWindowBounds":
				result = map[string]any{"bounds": map[string]any{"left": 20, "top": 30, "width": 1280, "height": 800, "windowState": "normal"}}
			}
			encoded, _ := json.Marshal(result)
			if err := connection.WriteJSON(cdpResponse{ID: message.ID, Result: encoded}); err != nil {
				return
			}
		}
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	controller, err := NewWithClients(&http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}, &websocket.Dialer{Proxy: nil, HandshakeTimeout: time.Second}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	state, err := controller.Apply(context.Background(), port, "ws://127.0.0.1:"+strconv.Itoa(port)+"/devtools/browser/test", domain.WindowPlan{Width: 1280, Height: 800, DeviceScaleFactor: 1, Source: domain.WindowSourceExplicit})
	if err != nil {
		t.Fatal(err)
	}
	if !state.Applied || state.Width != 1280 || state.Height != 800 || state.Left != 20 || state.Top != 30 {
		t.Fatalf("unexpected window state: %#v", state)
	}
	mu.Lock()
	defer mu.Unlock()
	expected := []string{"Browser.getWindowForTarget", "Browser.setWindowBounds", "Browser.getWindowBounds"}
	if len(methods) != len(expected) {
		t.Fatalf("unexpected commands: %#v", methods)
	}
	for index := range expected {
		if methods[index] != expected[index] {
			t.Fatalf("unexpected command order: %#v", methods)
		}
	}
}

func TestApplyRejectsNonLoopbackWebSocket(t *testing.T) {
	controller := New()
	_, err := controller.Apply(context.Background(), 9222, "ws://example.com:9222/devtools/browser/test", domain.WindowPlan{Width: 1280, Height: 800, DeviceScaleFactor: 1})
	if err == nil {
		t.Fatal("expected non-loopback websocket rejection")
	}
}

func TestApplyRejectsUnboundedWindowPlan(t *testing.T) {
	controller := New()
	_, err := controller.Apply(context.Background(), 9222, "ws://127.0.0.1:9222/devtools/browser/test", domain.WindowPlan{Width: 20000, Height: 800, DeviceScaleFactor: 1})
	if err == nil {
		t.Fatal("expected oversized window rejection")
	}
}
