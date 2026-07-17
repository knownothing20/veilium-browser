package proxydiagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/proxybridge"
)

func TestAuthenticatedRouteProducesHealthyReportWithoutSecrets(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ip":"203.0.113.42"}`))
	}))
	defer proxyServer.Close()

	factory := &fakeFactory{
		instance: &fakeBridge{url: proxyServer.URL, kind: "http-auth"},
		expected: credential.Material{Username: "alice", Secret: "top-secret"},
	}
	runner, err := New(func() proxybridge.Factory { return factory }, Config{ProbeURL: proxyServer.URL, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	report, err := runner.Run(context.Background(), Request{
		Profile: diagnosticProfile("proxy-only"),
		Material: credential.Material{
			Username: "alice",
			Secret:   "top-secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusHealthy || report.ExitIP != "203.0.113.42" || report.BridgeKind != "http-auth" {
		t.Fatalf("unexpected report: %#v", report)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"alice", "top-secret"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("credential material leaked into report: %s", encoded)
		}
	}
	if factory.instance.closeCount() != 1 {
		t.Fatalf("temporary diagnostic bridge was not closed: %d", factory.instance.closeCount())
	}
}

func TestDefaultWebRTCPPolicyDegradesOtherwiseHealthyProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ip":"198.51.100.11"}`))
	}))
	defer server.Close()

	factory := &fakeFactory{
		instance: &fakeBridge{url: server.URL, kind: "socks5-auth"},
		expected: credential.Material{Username: "u", Secret: "p"},
	}
	runner, _ := New(func() proxybridge.Factory { return factory }, Config{ProbeURL: server.URL, Timeout: time.Second})
	report, err := runner.Run(context.Background(), Request{
		Profile:  diagnosticProfile("default"),
		Material: credential.Material{Username: "u", Secret: "p"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusDegraded {
		t.Fatalf("expected degraded report, got %#v", report)
	}
	if checkStatus(report, "webrtc_policy") != CheckWarn {
		t.Fatalf("expected WebRTC warning: %#v", report.Checks)
	}
}

func TestInvalidExitIPFailsReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ip":"not-an-ip"}`))
	}))
	defer server.Close()

	runner, _ := New(func() proxybridge.Factory { return &fakeFactory{} }, Config{ProbeURL: server.URL, Timeout: time.Second})
	profile := diagnosticProfile("disabled")
	profile.Proxy = domain.ProxyConfig{URL: "direct://"}
	report, err := runner.Run(context.Background(), Request{Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusFailed || checkStatus(report, "exit_ip") != CheckFail {
		t.Fatalf("expected failed report: %#v", report)
	}
}

func TestDuplicateRunIsRejected(t *testing.T) {
	gate := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		<-gate
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ip":"192.0.2.8"}`))
	}))
	defer server.Close()

	runner, _ := New(func() proxybridge.Factory { return &fakeFactory{} }, Config{ProbeURL: server.URL, Timeout: time.Second})
	profile := diagnosticProfile("disabled")
	profile.Proxy = domain.ProxyConfig{URL: "direct://"}

	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), Request{Profile: profile})
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		runner.mu.Lock()
		_, active := runner.active[profile.ID]
		runner.mu.Unlock()
		if active {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := runner.Run(context.Background(), Request{Profile: profile}); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected duplicate diagnostic rejection, got %v", err)
	}
	close(gate)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestProbeURLRejectsNonLoopbackHTTP(t *testing.T) {
	_, err := New(func() proxybridge.Factory { return &fakeFactory{} }, Config{ProbeURL: "http://example.com/ip"})
	if err == nil {
		t.Fatal("expected insecure probe URL rejection")
	}
}

func diagnosticProfile(policy string) domain.Profile {
	return domain.Profile{
		ID:   "profile-a",
		Name: "Profile A",
		Fingerprint: domain.FingerprintConfig{
			WebRTCPolicy: policy,
		},
		Proxy: domain.ProxyConfig{
			URL:           "http://proxy.example:3128",
			CredentialRef: "cred-a",
		},
	}
}

func checkStatus(report Report, id string) string {
	for _, check := range report.Checks {
		if check.ID == id {
			return check.Status
		}
	}
	return ""
}

type fakeFactory struct {
	instance *fakeBridge
	expected credential.Material
}

func (f *fakeFactory) Start(_ context.Context, _ string, material credential.Material) (proxybridge.Instance, error) {
	if f.instance == nil {
		return nil, errors.New("unexpected bridge start")
	}
	if material != f.expected {
		return nil, errors.New("unexpected credential material")
	}
	return f.instance, nil
}

type fakeBridge struct {
	mu     sync.Mutex
	url    string
	kind   string
	closed int
}

func (b *fakeBridge) URL() string                  { return b.url }
func (b *fakeBridge) Kind() string                 { return b.kind }
func (b *fakeBridge) Health(context.Context) error { return nil }
func (b *fakeBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed++
	return nil
}
func (b *fakeBridge) closeCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}
