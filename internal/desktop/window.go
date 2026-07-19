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

var windowControllerRegistry sync.Map

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
	// Synthetic test supervisors created before M4.3 do not expose a Browser WebSocket.
	// Production sessions always pass supervisor.validateVersionInfo before becoming ready.
	if strings.TrimSpace(session.WebSocketDebuggerURL) == "" {
		return session, nil
	}
	plan, err := domain.EffectiveWindowPlan(item.Fingerprint)
	if err != nil {
		return session, err
	}
	state := windowStateFor(s)
	state.mu.Lock()
	controller := state.controller
	state.mu.Unlock()
	if controller == nil {
		return session, fmt.Errorf("managed window controller is unavailable")
	}
	windowContext, cancel := context.WithTimeout(nonNilDesktopContext(ctx), 6*time.Second)
	defer cancel()
	observed, err := controller.Apply(windowContext, session.CDPPort, session.WebSocketDebuggerURL, plan)
	if err != nil {
		stopContext, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = s.supervisor.Stop(stopContext, item.ID)
		stopCancel()
		return session, fmt.Errorf("apply managed browser window: %w", err)
	}
	state.mu.Lock()
	state.states[item.ID] = observed
	state.mu.Unlock()
	return session, nil
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
