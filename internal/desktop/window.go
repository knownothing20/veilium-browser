package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
	"github.com/knownothing20/veilium-browser/internal/windowcontrol"
)

type managedWindowController interface {
	Apply(context.Context, int, string, domain.WindowPlan) (domain.WindowState, error)
}

type windowControllerState struct {
	mu         sync.Mutex
	controller managedWindowController
	states     map[string]domain.WindowState
}

type windowRuntimeSupervisor struct {
	service *Service
	inner   RuntimeSupervisor
}

var windowControllerRegistry sync.Map

func installWindowSupervisor(service *Service) {
	if service == nil || service.supervisor == nil {
		return
	}
	if _, installed := service.supervisor.(*windowRuntimeSupervisor); installed {
		return
	}
	service.supervisor = &windowRuntimeSupervisor{service: service, inner: service.supervisor}
}

func (runtime *windowRuntimeSupervisor) Start(ctx context.Context, profileID, profileName string, build supervisor.PlanBuilder) (supervisor.Session, error) {
	var windowPlan *domain.WindowPlan
	session, err := runtime.inner.Start(ctx, profileID, profileName, func(port int) (domain.LaunchPlan, error) {
		plan, buildErr := build(port)
		if buildErr == nil && plan.Window != nil {
			captured := *plan.Window
			windowPlan = &captured
		}
		return plan, buildErr
	})
	if err != nil {
		return session, err
	}
	if windowPlan == nil {
		stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = runtime.inner.Stop(stopContext, profileID)
		cancel()
		return session, fmt.Errorf("launch plan did not include a managed window")
	}
	observed, applyErr := runtime.service.applyManagedWindowPlan(ctx, profileID, session, *windowPlan)
	if applyErr != nil {
		stopContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = runtime.inner.Stop(stopContext, profileID)
		cancel()
		return session, applyErr
	}
	if observed.Applied {
		state := windowStateFor(runtime.service)
		state.mu.Lock()
		state.states[profileID] = observed
		state.mu.Unlock()
	}
	return session, nil
}

func (runtime *windowRuntimeSupervisor) Stop(ctx context.Context, profileID string) (supervisor.Session, error) {
	session, err := runtime.inner.Stop(ctx, profileID)
	clearObservedWindow(runtime.service, profileID)
	return session, err
}

func (runtime *windowRuntimeSupervisor) Shutdown(ctx context.Context) error {
	clearAllObservedWindows(runtime.service)
	return runtime.inner.Shutdown(ctx)
}

func (runtime *windowRuntimeSupervisor) List() []supervisor.Session {
	return runtime.inner.List()
}

func (runtime *windowRuntimeSupervisor) IsActive(profileID string) bool {
	return runtime.inner.IsActive(profileID)
}

func windowStateFor(service *Service) *windowControllerState {
	value, _ := windowControllerRegistry.LoadOrStore(service, &windowControllerState{
		controller: windowcontrol.New(),
		states:     make(map[string]domain.WindowState),
	})
	return value.(*windowControllerState)
}

func setWindowController(service *Service, controller managedWindowController) {
	state := windowStateFor(service)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.controller = controller
}

func (s *Service) applyManagedWindow(ctx context.Context, item domain.Profile, session supervisor.Session) (supervisor.Session, error) {
	plan, err := domain.EffectiveWindowPlan(item.Fingerprint)
	if err != nil {
		return session, err
	}
	observed, err := s.applyManagedWindowPlan(ctx, item.ID, session, plan)
	if err != nil {
		return session, err
	}
	if observed.Applied {
		state := windowStateFor(s)
		state.mu.Lock()
		state.states[item.ID] = observed
		state.mu.Unlock()
	}
	return session, nil
}

func (s *Service) applyManagedWindowPlan(ctx context.Context, profileID string, session supervisor.Session, plan domain.WindowPlan) (domain.WindowState, error) {
	// Synthetic supervisors used by unit tests created before M4.3 do not expose
	// a Browser WebSocket. Production sessions must pass supervisor validation
	// before becoming ready and therefore always carry this endpoint.
	if strings.TrimSpace(session.WebSocketDebuggerURL) == "" {
		return domain.WindowState{}, nil
	}
	state := windowStateFor(s)
	state.mu.Lock()
	controller := state.controller
	state.mu.Unlock()
	if controller == nil {
		return domain.WindowState{}, fmt.Errorf("managed window controller is unavailable")
	}
	windowContext, cancel := context.WithTimeout(nonNilDesktopContext(ctx), 6*time.Second)
	defer cancel()
	observed, err := controller.Apply(windowContext, session.CDPPort, session.WebSocketDebuggerURL, plan)
	if err != nil {
		return domain.WindowState{}, fmt.Errorf("apply managed browser window: %w", err)
	}
	return observed, nil
}

func observedWindowFor(service *Service, profileID string) (domain.WindowState, bool) {
	state := windowStateFor(service)
	state.mu.Lock()
	defer state.mu.Unlock()
	observed, ok := state.states[strings.TrimSpace(profileID)]
	return observed, ok
}

func clearObservedWindow(service *Service, profileID string) {
	value, ok := windowControllerRegistry.Load(service)
	if !ok {
		return
	}
	state := value.(*windowControllerState)
	state.mu.Lock()
	delete(state.states, strings.TrimSpace(profileID))
	state.mu.Unlock()
}

func clearAllObservedWindows(service *Service) {
	windowControllerRegistry.Delete(service)
}

func nonNilDesktopContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
