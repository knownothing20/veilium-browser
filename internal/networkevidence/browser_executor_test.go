package networkevidence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

type fakeBrowserCollector struct {
	url        string
	submission BrowserSubmission
	waitErr    error
	closeErr   error
	closed     bool
}

func (collector *fakeBrowserCollector) URL() string { return collector.url }
func (collector *fakeBrowserCollector) Wait(context.Context) (BrowserSubmission, error) {
	return collector.submission, collector.waitErr
}
func (collector *fakeBrowserCollector) Close(context.Context) error {
	collector.closed = true
	return collector.closeErr
}

type fakeBrowserTargetController struct {
	target   evidence.Target
	openErr  error
	closeErr error
	opened   bool
	closed   bool
	url      string
}

func (controller *fakeBrowserTargetController) Open(_ context.Context, port int, rawURL string) (evidence.Target, error) {
	controller.opened = true
	controller.url = rawURL
	if port != 9222 {
		return evidence.Target{}, errors.New("unexpected CDP port")
	}
	return controller.target, controller.openErr
}

func (controller *fakeBrowserTargetController) Close(_ context.Context, port int, targetID string) error {
	controller.closed = true
	if port != 9222 || targetID != controller.target.ID {
		return errors.New("unexpected target close")
	}
	return controller.closeErr
}

func TestBrowserExecutorReturnsValidatedObservations(t *testing.T) {
	now := time.Now().UTC()
	set := validProbeSet()
	collector := &fakeBrowserCollector{
		url: "http://127.0.0.1:45678/run/token",
		submission: BrowserSubmission{
			SchemaVersion: BrowserSubmissionSchemaVersion,
			Observations: []BrowserObservation{
				{ProbeID: "exit", ProbeRevision: 1, ProbeKind: ProbeExitIP, Status: ObservationPassed, Values: []string{"203.0.113.8"}},
				{ProbeID: "stun", ProbeRevision: 1, ProbeKind: ProbeWebRTCSTUN, Status: ObservationPartial, Values: []string{"candidate:host", "protocol:udp", "mdns:true"}},
				{ProbeID: "dns", ProbeRevision: 1, ProbeKind: ProbeDelegatedDNS, Status: ObservationPassed, Values: []string{"seen:true", "resolver-ip:192.0.2.53", "rcode:NOERROR"}},
			},
		},
	}
	target := &fakeBrowserTargetController{target: evidence.Target{ID: "target-a", Type: "page"}}
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{
		CollectorFactory: func(ProbeSet) (BrowserCollector, error) { return collector, nil },
		TargetController: target,
		Now:              func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), ExecutionRequest{
		ProfileID: "profile-a",
		Session: supervisor.Session{ProfileID: "profile-a", State: supervisor.StateReady, CDPPort: 9222, WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test"},
		Route:    RouteIdentity{Kind: RouteDirect, Scheme: "direct", Digest: strings.Repeat("a", 64)},
		ProbeSet: set,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !target.opened || !target.closed || !collector.closed || target.url != collector.url {
		t.Fatalf("executor cleanup did not complete: target=%#v collector=%#v", target, collector)
	}
	if len(result.Observations) != 3 || result.Observations[0].CollectedAt != now {
		t.Fatalf("unexpected executor result: %#v", result)
	}
}

func TestBrowserExecutorClosesCollectorWhenTargetOpenFails(t *testing.T) {
	collector := &fakeBrowserCollector{url: "http://127.0.0.1:45678/run/token"}
	target := &fakeBrowserTargetController{openErr: errors.New("target rejected")}
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{
		CollectorFactory: func(ProbeSet) (BrowserCollector, error) { return collector, nil },
		TargetController: target,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executor.Execute(context.Background(), ExecutionRequest{
		Session:  supervisor.Session{State: supervisor.StateReady, CDPPort: 9222, WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test"},
		ProbeSet: validProbeSet(),
	})
	if err == nil || !collector.closed {
		t.Fatalf("expected target failure and collector cleanup, got %v, closed=%v", err, collector.closed)
	}
}

func TestBrowserExecutorRejectsSubmissionOutsideProbeSet(t *testing.T) {
	collector := &fakeBrowserCollector{
		url: "http://127.0.0.1:45678/run/token",
		submission: BrowserSubmission{
			SchemaVersion: BrowserSubmissionSchemaVersion,
			Observations: []BrowserObservation{{ProbeID: "unknown", ProbeRevision: 1, ProbeKind: ProbeExitIP, Status: ObservationPassed, Values: []string{"203.0.113.8"}}},
		},
	}
	target := &fakeBrowserTargetController{target: evidence.Target{ID: "target-a", Type: "page"}}
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{
		CollectorFactory: func(ProbeSet) (BrowserCollector, error) { return collector, nil },
		TargetController: target,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executor.Execute(context.Background(), ExecutionRequest{
		Session:  supervisor.Session{State: supervisor.StateReady, CDPPort: 9222, WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test"},
		ProbeSet: validProbeSet(),
	})
	if err == nil || !strings.Contains(err.Error(), "selected probe set") {
		t.Fatalf("expected selected-probe-set rejection, got %v", err)
	}
}

func TestBrowserExecutorReportsCleanupLimitations(t *testing.T) {
	collector := &fakeBrowserCollector{
		url:      "http://127.0.0.1:45678/run/token",
		closeErr: errors.New("collector close failed"),
		submission: BrowserSubmission{
			SchemaVersion: BrowserSubmissionSchemaVersion,
			Observations: []BrowserObservation{{ProbeID: "exit", ProbeRevision: 1, ProbeKind: ProbeExitIP, Status: ObservationPassed, Values: []string{"203.0.113.8"}}},
		},
	}
	set := ProbeSet{SchemaVersion: ProbeSchemaVersion, ID: "exit-only", Revision: 1, Definitions: []ProbeDefinition{{
		SchemaVersion: ProbeSchemaVersion, ID: "exit", Revision: 1, Kind: ProbeExitIP,
		HTTPSURL: "https://probe.example.invalid/ip", TimeoutSeconds: 10, MaxResponseBytes: 4096,
		SelfHostable: true, PrivacyNote: "Returns only the request public IP for this test.",
	}}}
	target := &fakeBrowserTargetController{target: evidence.Target{ID: "target-a", Type: "page"}, closeErr: errors.New("target close failed")}
	executor, err := NewBrowserExecutor(BrowserExecutorOptions{
		CollectorFactory: func(ProbeSet) (BrowserCollector, error) { return collector, nil },
		TargetController: target,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), ExecutionRequest{
		Session:  supervisor.Session{State: supervisor.StateReady, CDPPort: 9222, WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test"},
		ProbeSet: set,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Limitations) != 2 {
		t.Fatalf("expected two cleanup limitations, got %#v", result.Limitations)
	}
}
