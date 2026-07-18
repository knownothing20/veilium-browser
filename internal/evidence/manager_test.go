package evidence

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestManagerRunsEvaluatesPersistsAndCleansUp(t *testing.T) {
	now := time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)
	store := newTestEvidenceStore(t, now)
	collector := &fakeCollector{submission: fullMatchingSubmission(), url: controlledTestURL()}
	targets := &fakeTargetController{target: Target{ID: "target_1", Type: "page"}}
	manager, err := NewManager(store, ManagerOptions{
		Timeout: 2 * time.Second,
		Now:     func() time.Time { return now },
		CollectorFactory: func(CollectorOptions) (CollectorHandle, error) {
			return collector, nil
		},
		TargetController: targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := manager.Run(context.Background(), validRunRequest())
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunPartial {
		t.Fatalf("custom provider evidence should be partial, got %s", run.Status)
	}
	if collector.closeCount != 1 || targets.closeCount != 1 {
		t.Fatalf("temporary resources were not closed: collector=%d target=%d", collector.closeCount, targets.closeCount)
	}
	loaded, err := store.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ProviderTrust != fingerprint.TrustCustom || len(loaded.Observations) == 0 {
		t.Fatalf("unexpected persisted run: %#v", loaded)
	}
}

func TestManagerPreventsDuplicateRunAndPersistsCancellation(t *testing.T) {
	now := time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)
	store := newTestEvidenceStore(t, now)
	collector := &fakeCollector{url: controlledTestURL(), block: true}
	manager, err := NewManager(store, ManagerOptions{
		Timeout: time.Second,
		Now:     func() time.Time { return now },
		CollectorFactory: func(CollectorOptions) (CollectorHandle, error) {
			return collector, nil
		},
		TargetController: &fakeTargetController{target: Target{ID: "target_1", Type: "page"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := validRunRequest()
	result := make(chan error, 1)
	go func() {
		_, runErr := manager.Run(context.Background(), request)
		result <- runErr
	}()
	waitUntil(t, func() bool { return manager.IsActive(request.Profile.ID) })
	if _, err := manager.Run(context.Background(), request); !errors.Is(err, ErrRunActive) {
		t.Fatalf("expected duplicate run rejection, got %v", err)
	}
	if err := manager.Cancel(request.Profile.ID); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancelled run error, got %v", err)
	}
	items, err := store.List(request.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != RunCancelled || items[0].FailureCode != "cancelled" {
		t.Fatalf("cancelled run was not persisted: %#v", items)
	}
	if err := manager.Cancel(request.Profile.ID); !errors.Is(err, ErrNoRun) {
		t.Fatalf("expected no-active-run error, got %v", err)
	}
}

func TestManagerDetectsBrowserExitAsIncomplete(t *testing.T) {
	now := time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)
	store := newTestEvidenceStore(t, now)
	collector := &fakeCollector{url: controlledTestURL(), block: true}
	request := validRunRequest()
	request.SessionReady = func() bool { return false }
	manager, err := NewManager(store, ManagerOptions{
		Timeout: time.Second,
		Now:     func() time.Time { return now },
		CollectorFactory: func(CollectorOptions) (CollectorHandle, error) {
			return collector, nil
		},
		TargetController: &fakeTargetController{target: Target{ID: "target_1", Type: "page"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := manager.Run(context.Background(), request)
	if !errors.Is(err, ErrBrowserExit) {
		t.Fatalf("expected browser-exit error, got %v", err)
	}
	if run.Status != RunIncomplete || run.FailureCode != "browser-exited" {
		t.Fatalf("unexpected browser-exit run: %#v", run)
	}
}

func TestManagerPersistsTargetOpenFailure(t *testing.T) {
	now := time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)
	store := newTestEvidenceStore(t, now)
	collector := &fakeCollector{url: controlledTestURL()}
	targets := &fakeTargetController{openErr: errors.New("CDP refused target")}
	manager, err := NewManager(store, ManagerOptions{
		Timeout: time.Second,
		Now:     func() time.Time { return now },
		CollectorFactory: func(CollectorOptions) (CollectorHandle, error) {
			return collector, nil
		},
		TargetController: targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := manager.Run(context.Background(), validRunRequest())
	if err == nil || !strings.Contains(err.Error(), "CDP refused") {
		t.Fatalf("expected target open error, got %v", err)
	}
	if run.Status != RunFailed || run.FailureCode != "target-open-failed" {
		t.Fatalf("target failure was not recorded: %#v", run)
	}
	if collector.closeCount == 0 {
		t.Fatal("collector was not closed after target failure")
	}
}

func TestManagerRejectsUnmanagedOrMismatchedRequest(t *testing.T) {
	store := newTestEvidenceStore(t, time.Now().UTC())
	manager, err := NewManager(store, ManagerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	request := validRunRequest()
	request.Profile.Kernel.ID = "other-kernel"
	if _, err := manager.Run(context.Background(), request); err == nil || !strings.Contains(err.Error(), "exact managed kernel") {
		t.Fatalf("expected managed-kernel rejection, got %v", err)
	}
	request = validRunRequest()
	request.Session.State = supervisor.StateExited
	if _, err := manager.Run(context.Background(), request); err == nil || !strings.Contains(err.Error(), "ready managed") {
		t.Fatalf("expected ready-session rejection, got %v", err)
	}
}

func newTestEvidenceStore(t *testing.T, now time.Time) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "evidence"), StoreOptions{
		Retention: 24 * time.Hour,
		Now:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func validRunRequest() RunRequest {
	profile := evidenceProfile()
	profile.Kernel.ID = "kernel-1"
	record := kernel.Record{
		ID:         "kernel-1",
		Name:       "Custom Chromium",
		Provider:   fingerprint.ProviderCustom,
		Version:    "148.0.0",
		Executable: "/managed/chrome",
		SHA256:     strings.Repeat("a", 64),
		SizeBytes:  100,
		Status:     kernel.StatusVerified,
		VerifiedAt: time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC),
	}
	capabilities, _ := fingerprint.For(fingerprint.ProviderCustom, "148.0.0")
	return RunRequest{
		Profile:      profile,
		Kernel:       record,
		Session:      supervisor.Session{ProfileID: profile.ID, ProfileName: profile.Name, State: supervisor.StateReady, CDPPort: 9222},
		Capabilities: capabilities,
		SessionReady: func() bool { return true },
	}
}

type fakeCollector struct {
	mu         sync.Mutex
	url        string
	submission BrowserSubmission
	waitErr    error
	block      bool
	closeCount int
}

func (c *fakeCollector) URL() string { return c.url }
func (c *fakeCollector) Wait(ctx context.Context) (BrowserSubmission, error) {
	if c.block {
		<-ctx.Done()
		return BrowserSubmission{}, ctx.Err()
	}
	return c.submission, c.waitErr
}
func (c *fakeCollector) Close(context.Context) error {
	c.mu.Lock()
	c.closeCount++
	c.mu.Unlock()
	return nil
}

type fakeTargetController struct {
	target     Target
	openErr    error
	closeErr   error
	closeCount int
}

func (c *fakeTargetController) Open(context.Context, int, string) (Target, error) {
	return c.target, c.openErr
}
func (c *fakeTargetController) Close(context.Context, int, string) error {
	c.closeCount++
	return c.closeErr
}

func controlledTestURL() string {
	return "http://127.0.0.1:45678/run/" + strings.Repeat("a", 64)
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
