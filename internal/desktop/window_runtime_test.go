package desktop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

type fakeWindowRuntime struct {
	active  bool
	stopped bool
	plan    domain.LaunchPlan
	session supervisor.Session
}

func (runtime *fakeWindowRuntime) Start(_ context.Context, profileID, profileName string, build supervisor.PlanBuilder) (supervisor.Session, error) {
	plan, err := build(9222)
	if err != nil {
		return supervisor.Session{}, err
	}
	runtime.plan = plan
	runtime.active = true
	runtime.session = supervisor.Session{
		ProfileID:            profileID,
		ProfileName:          profileName,
		State:                supervisor.StateReady,
		PID:                  42,
		CDPPort:              9222,
		WebSocketDebuggerURL: "ws://127.0.0.1:9222/devtools/browser/test",
		StartedAt:            time.Now().UTC(),
	}
	return runtime.session, nil
}

func (runtime *fakeWindowRuntime) Stop(_ context.Context, _ string) (supervisor.Session, error) {
	runtime.active = false
	runtime.stopped = true
	runtime.session.State = supervisor.StateExited
	return runtime.session, nil
}

func (runtime *fakeWindowRuntime) Shutdown(context.Context) error {
	runtime.active = false
	return nil
}

func (runtime *fakeWindowRuntime) List() []supervisor.Session {
	if runtime.session.ProfileID == "" {
		return nil
	}
	return []supervisor.Session{runtime.session}
}

func (runtime *fakeWindowRuntime) IsActive(string) bool { return runtime.active }

type fakeManagedWindow struct {
	state domain.WindowState
	err   error
	plan  domain.WindowPlan
}

func (controller *fakeManagedWindow) Apply(_ context.Context, port int, websocketURL string, plan domain.WindowPlan) (domain.WindowState, error) {
	if port != 9222 || websocketURL != "ws://127.0.0.1:9222/devtools/browser/test" {
		return domain.WindowState{}, errors.New("unexpected managed session endpoint")
	}
	controller.plan = plan
	return controller.state, controller.err
}

func TestWindowSupervisorAppliesLaunchPlanAndStoresObservedState(t *testing.T) {
	inner := &fakeWindowRuntime{}
	service := &Service{supervisor: inner}
	installWindowSupervisor(service)
	controller := &fakeManagedWindow{state: domain.WindowState{Width: 1280, Height: 800, Applied: true, State: "normal"}}
	setWindowController(service, controller)

	plan := domain.WindowPlan{Width: 1280, Height: 800, DeviceScaleFactor: 1, Source: domain.WindowSourceExplicit}
	_, err := service.supervisor.Start(context.Background(), "profile-a", "Profile A", func(int) (domain.LaunchPlan, error) {
		return domain.LaunchPlan{Executable: "/managed/chrome", Window: &plan}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if controller.plan != plan {
		t.Fatalf("unexpected applied window plan: %#v", controller.plan)
	}
	observed, ok := observedWindowFor(service, "profile-a")
	if !ok || observed.Width != 1280 || observed.Height != 800 {
		t.Fatalf("expected stored observed window, got %#v, %v", observed, ok)
	}
}

func TestWindowSupervisorStopsBrowserWhenWindowApplicationFails(t *testing.T) {
	inner := &fakeWindowRuntime{}
	service := &Service{supervisor: inner}
	installWindowSupervisor(service)
	setWindowController(service, &fakeManagedWindow{err: errors.New("window rejected")})

	plan := domain.WindowPlan{Width: 1280, Height: 800, DeviceScaleFactor: 1, Source: domain.WindowSourceExplicit}
	_, err := service.supervisor.Start(context.Background(), "profile-a", "Profile A", func(int) (domain.LaunchPlan, error) {
		return domain.LaunchPlan{Executable: "/managed/chrome", Window: &plan}, nil
	})
	if err == nil || !inner.stopped {
		t.Fatalf("expected window failure to stop browser, got %v, stopped=%v", err, inner.stopped)
	}
}

func TestWindowSupervisorStopClearsObservedState(t *testing.T) {
	inner := &fakeWindowRuntime{}
	service := &Service{supervisor: inner}
	installWindowSupervisor(service)
	controller := &fakeManagedWindow{state: domain.WindowState{Width: 1280, Height: 800, Applied: true}}
	setWindowController(service, controller)

	plan := domain.WindowPlan{Width: 1280, Height: 800, DeviceScaleFactor: 1, Source: domain.WindowSourceExplicit}
	_, err := service.supervisor.Start(context.Background(), "profile-a", "Profile A", func(int) (domain.LaunchPlan, error) {
		return domain.LaunchPlan{Executable: "/managed/chrome", Window: &plan}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.supervisor.Stop(context.Background(), "profile-a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := observedWindowFor(service, "profile-a"); ok {
		t.Fatal("observed window state was not cleared")
	}
}
