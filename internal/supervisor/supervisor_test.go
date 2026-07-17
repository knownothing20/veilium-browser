package supervisor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

type fakeProcess struct {
	mu          sync.Mutex
	pid         int
	wait        chan error
	signalCount int
	killCount   int
}

func newFakeProcess(pid int) *fakeProcess {
	return &fakeProcess{pid: pid, wait: make(chan error, 1)}
}

func (p *fakeProcess) PID() int { return p.pid }
func (p *fakeProcess) Wait() error {
	return <-p.wait
}
func (p *fakeProcess) Signal(os.Signal) error {
	p.mu.Lock()
	p.signalCount++
	p.mu.Unlock()
	return nil
}
func (p *fakeProcess) Kill() error {
	p.mu.Lock()
	p.killCount++
	p.mu.Unlock()
	select {
	case p.wait <- errors.New("killed"):
	default:
	}
	return nil
}
func (p *fakeProcess) exit(err error) {
	select {
	case p.wait <- err:
	default:
	}
}

type fakeRunner struct {
	process *fakeProcess
	plan    domain.LaunchPlan
	logPath string
	starts  int
}

func (r *fakeRunner) Start(plan domain.LaunchPlan, logPath string) (Process, error) {
	r.starts++
	r.plan = plan
	r.logPath = logPath
	return r.process, nil
}

type fakeProber struct {
	version VersionInfo
	err     error
	wait    chan struct{}
}

func (p *fakeProber) Wait(ctx context.Context, _ int) (VersionInfo, error) {
	if p.wait != nil {
		select {
		case <-p.wait:
		case <-ctx.Done():
			return VersionInfo{}, ctx.Err()
		}
	}
	return p.version, p.err
}

type fixedPorts struct{ port int }

func (p fixedPorts) Allocate() (int, error) { return p.port, nil }

func TestStartBecomesReadyAndUsesExactLoopbackPort(t *testing.T) {
	process := newFakeProcess(321)
	runner := &fakeRunner{process: process}
	supervisor := newTestSupervisor(t, runner, &fakeProber{version: VersionInfo{
		Browser:              "Chrome/148.0.0.0",
		WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test",
	}})

	session, err := supervisor.Start(context.Background(), "profile-a", "Profile A", safePlan)
	if err != nil {
		t.Fatal(err)
	}
	if session.State != StateReady || session.PID != 321 || session.CDPPort != 9222 {
		t.Fatalf("unexpected session: %#v", session)
	}
	joined := strings.Join(runner.plan.Args, " ")
	if !strings.Contains(joined, "--remote-debugging-address=127.0.0.1") || !strings.Contains(joined, "--remote-debugging-port=9222") {
		t.Fatalf("unsafe launch plan: %s", joined)
	}
	if filepath.Dir(runner.logPath) != supervisor.logDir {
		t.Fatalf("log path escaped runtime directory: %s", runner.logPath)
	}
	process.exit(nil)
	waitForState(t, supervisor, "profile-a", StateExited)
}

func TestConcurrentDuplicateStartIsBlocked(t *testing.T) {
	process := newFakeProcess(322)
	gate := make(chan struct{})
	runner := &fakeRunner{process: process}
	supervisor := newTestSupervisor(t, runner, &fakeProber{
		version: VersionInfo{Browser: "Chrome/148", WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test"},
		wait:    gate,
	})

	firstDone := make(chan error, 1)
	go func() {
		_, err := supervisor.Start(context.Background(), "profile-a", "Profile A", safePlan)
		firstDone <- err
	}()
	waitUntil(t, func() bool { return supervisor.IsActive("profile-a") })
	if _, err := supervisor.Start(context.Background(), "profile-a", "Profile A", safePlan); err == nil {
		t.Fatal("expected duplicate start rejection")
	}
	close(gate)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if runner.starts != 1 {
		t.Fatalf("expected one process start, got %d", runner.starts)
	}
	process.exit(nil)
}

func TestStopSignalsAndWaitsForExit(t *testing.T) {
	process := newFakeProcess(323)
	runner := &fakeRunner{process: process}
	supervisor := newTestSupervisor(t, runner, &fakeProber{version: VersionInfo{
		Browser:              "Chrome/148",
		WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test",
	}})
	if _, err := supervisor.Start(context.Background(), "profile-a", "Profile A", safePlan); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(10 * time.Millisecond)
		process.exit(nil)
	}()
	session, err := supervisor.Stop(context.Background(), "profile-a")
	if err != nil {
		t.Fatal(err)
	}
	if session.State != StateExited {
		t.Fatalf("unexpected stop state: %#v", session)
	}
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.signalCount != 1 || process.killCount != 0 {
		t.Fatalf("unexpected stop calls: signal=%d kill=%d", process.signalCount, process.killCount)
	}
}

func TestProbeFailureKillsProcessAndPreservesFailure(t *testing.T) {
	process := newFakeProcess(324)
	runner := &fakeRunner{process: process}
	supervisor := newTestSupervisor(t, runner, &fakeProber{err: errors.New("not ready")})

	session, err := supervisor.Start(context.Background(), "profile-a", "Profile A", safePlan)
	if err == nil || !strings.Contains(err.Error(), "readiness") {
		t.Fatalf("expected readiness error, got %v", err)
	}
	if session.State != StateFailed || !strings.Contains(session.LastError, "not ready") {
		t.Fatalf("failure was not preserved: %#v", session)
	}
	waitForState(t, supervisor, "profile-a", StateFailed)
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.killCount != 1 {
		t.Fatalf("expected process kill, got %d", process.killCount)
	}
}

func TestRejectsBridgeAndNonLoopbackCDPBeforeReady(t *testing.T) {
	process := newFakeProcess(325)
	runner := &fakeRunner{process: process}
	supervisor := newTestSupervisor(t, runner, &fakeProber{})
	_, err := supervisor.Start(context.Background(), "profile-a", "Profile A", func(port int) (domain.LaunchPlan, error) {
		plan, buildErr := safePlan(port)
		plan.RequiresBridge = true
		plan.BridgeKind = "xray"
		return plan, buildErr
	})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("expected bridge rejection, got %v", err)
	}
	if runner.starts != 0 {
		t.Fatal("process started despite bridge rejection")
	}

	runner.process = newFakeProcess(326)
	supervisor.prober = &fakeProber{version: VersionInfo{
		Browser:              "Chrome/148",
		WebSocketDebuggerURL: "ws://192.168.1.4:9222/devtools/browser/test",
	}}
	session, err := supervisor.Start(context.Background(), "profile-b", "Profile B", safePlan)
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected non-loopback rejection, got %v", err)
	}
	if session.State != StateFailed {
		t.Fatalf("unexpected state: %#v", session)
	}
}

func TestValidatePlanRejectsEnvironmentInjection(t *testing.T) {
	plan, _ := safePlan(9222)
	plan.Environment = map[string]string{"BAD=KEY": "value"}
	if err := validatePlan(plan, 9222); err == nil {
		t.Fatal("expected invalid environment rejection")
	}
}

func newTestSupervisor(t *testing.T, runner Runner, prober Prober) *Supervisor {
	t.Helper()
	supervisor, err := NewWithDependencies(t.TempDir(), Dependencies{
		Runner:       runner,
		Prober:       prober,
		Ports:        fixedPorts{port: 9222},
		Now:          time.Now,
		ReadyTimeout: 500 * time.Millisecond,
		StopTimeout:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return supervisor
}

func safePlan(port int) (domain.LaunchPlan, error) {
	return domain.LaunchPlan{
		Executable: "/managed/chrome",
		Args: []string{
			"--remote-debugging-address=127.0.0.1",
			fmt.Sprintf("--remote-debugging-port=%d", port),
		},
	}, nil
}

func waitForState(t *testing.T, supervisor *Supervisor, profileID string, expected State) {
	t.Helper()
	waitUntil(t, func() bool {
		session, err := supervisor.Get(profileID)
		return err == nil && session.State == expected
	})
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met before deadline")
}
