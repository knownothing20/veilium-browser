package networkevidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const defaultRunTimeout = 45 * time.Second

var (
	ErrRunActive        = errors.New("network evidence run is already active for profile")
	ErrNoActiveRun      = errors.New("no active network evidence run for profile")
	ErrSessionExited    = errors.New("managed browser session exited during network evidence")
	ErrProbeUnavailable = errors.New("network evidence probe is unavailable")
)

type RunStore interface {
	Save(Run) error
	Get(string) (Run, error)
	List(string) ([]Run, error)
	Delete(string) error
	Retention() time.Duration
}

type ExecutionRequest struct {
	ProfileID string
	Session   supervisor.Session
	Route     RouteIdentity
	ProbeSet  ProbeSet
}

type ExecutionResult struct {
	Observations []Observation
	Limitations  []string
}

type Executor interface {
	Execute(context.Context, ExecutionRequest) (ExecutionResult, error)
}

type RunRequest struct {
	Profile      domain.Profile
	BaseEvidence evidence.Run
	Session      supervisor.Session
	ProbeSet     ProbeSet
	SessionReady func() bool
}

type ManagerOptions struct {
	Timeout time.Duration
	Now     func() time.Time
}

type Manager struct {
	store    RunStore
	executor Executor
	timeout  time.Duration
	now      func() time.Time

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func NewManager(store RunStore, executor Executor, options ManagerOptions) (*Manager, error) {
	if store == nil || executor == nil {
		return nil, fmt.Errorf("network evidence store and executor are required")
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultRunTimeout
	}
	if options.Timeout > 2*time.Minute {
		return nil, fmt.Errorf("network evidence timeout is too large")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Manager{
		store: store, executor: executor, timeout: options.Timeout, now: options.Now,
		active: make(map[string]context.CancelFunc),
	}, nil
}

func (manager *Manager) Run(ctx context.Context, request RunRequest) (Run, error) {
	route, binaryDigest, err := validateRunRequest(request, manager.now().UTC())
	if err != nil {
		return Run{}, err
	}
	id, err := NewRunID()
	if err != nil {
		return Run{}, err
	}
	started := manager.now().UTC()
	expires := started.Add(manager.store.Retention())
	if request.BaseEvidence.ExpiresAt.Before(expires) {
		expires = request.BaseEvidence.ExpiresAt
	}
	run := Run{
		SchemaVersion:          SchemaVersion,
		ID:                     id,
		EvidenceRunID:          request.BaseEvidence.ID,
		ProfileID:              request.Profile.ID,
		ProviderID:             request.BaseEvidence.ProviderID,
		ProviderRevision:       request.BaseEvidence.ProviderRevision,
		BrowserVersion:         request.BaseEvidence.BrowserVersion,
		OperatingSystem:        request.BaseEvidence.OperatingSystem,
		Architecture:           request.BaseEvidence.Architecture,
		BinaryIdentityDigest:   binaryDigest,
		ConsistencyInputDigest: request.BaseEvidence.ConsistencyInputDigest,
		Route:                  route,
		ProbeSetID:             request.ProbeSet.ID,
		ProbeSetRevision:       request.ProbeSet.Revision,
		Status:                 RunPending,
		StartedAt:              started,
		ExpiresAt:              expires,
	}

	runContext, cancel := context.WithTimeout(nonNilContext(ctx), manager.timeout)
	if err := manager.activate(request.Profile.ID, cancel); err != nil {
		cancel()
		return Run{}, err
	}
	defer func() {
		cancel()
		manager.deactivate(request.Profile.ID)
	}()

	run.Status = RunRunning
	result, executionErr := manager.waitForExecution(runContext, ExecutionRequest{
		ProfileID: request.Profile.ID,
		Session:   request.Session,
		Route:     route,
		ProbeSet:  request.ProbeSet,
	}, request.SessionReady)
	if executionErr != nil {
		status, code := classifyExecutionError(executionErr)
		return manager.persistFailure(run, status, code, executionErr)
	}
	if err := validateExecutionResult(result, request.ProbeSet); err != nil {
		return manager.persistFailure(run, RunFailed, "invalid-probe-result", err)
	}
	completed := manager.now().UTC()
	run.Status = deriveRunStatus(result.Observations)
	run.CompletedAt = &completed
	run.Observations = result.Observations
	run.Limitations = sortedUnique(result.Limitations)
	if run.Status == RunUnavailable && len(run.Limitations) == 0 {
		run.Limitations = []string{"all configured network probes were unavailable"}
	}
	run = Normalize(run)
	if err := manager.store.Save(run); err != nil {
		return run, fmt.Errorf("persist network evidence run: %w", err)
	}
	return run, nil
}

func (manager *Manager) Cancel(profileID string) error {
	profileID = strings.TrimSpace(profileID)
	manager.mu.Lock()
	cancel, exists := manager.active[profileID]
	manager.mu.Unlock()
	if !exists {
		return ErrNoActiveRun
	}
	cancel()
	return nil
}

func (manager *Manager) Shutdown() {
	manager.mu.Lock()
	cancellations := make([]context.CancelFunc, 0, len(manager.active))
	for _, cancel := range manager.active {
		cancellations = append(cancellations, cancel)
	}
	manager.mu.Unlock()
	for _, cancel := range cancellations {
		cancel()
	}
}

func (manager *Manager) IsActive(profileID string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	_, exists := manager.active[strings.TrimSpace(profileID)]
	return exists
}

func (manager *Manager) List(profileID string) ([]Run, error) {
	return manager.store.List(strings.TrimSpace(profileID))
}
func (manager *Manager) Get(id string) (Run, error) { return manager.store.Get(strings.TrimSpace(id)) }
func (manager *Manager) Delete(id string) error     { return manager.store.Delete(strings.TrimSpace(id)) }

func (manager *Manager) activate(profileID string, cancel context.CancelFunc) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return fmt.Errorf("network evidence profile id is required")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if _, exists := manager.active[profileID]; exists {
		return ErrRunActive
	}
	manager.active[profileID] = cancel
	return nil
}

func (manager *Manager) deactivate(profileID string) {
	manager.mu.Lock()
	delete(manager.active, strings.TrimSpace(profileID))
	manager.mu.Unlock()
}

func (manager *Manager) waitForExecution(ctx context.Context, request ExecutionRequest, sessionReady func() bool) (ExecutionResult, error) {
	type response struct {
		result ExecutionResult
		err    error
	}
	responses := make(chan response, 1)
	go func() {
		result, err := manager.executor.Execute(ctx, request)
		responses <- response{result: result, err: err}
	}()
	if sessionReady == nil {
		select {
		case response := <-responses:
			return response.result, response.err
		case <-ctx.Done():
			return ExecutionResult{}, ctx.Err()
		}
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case response := <-responses:
			return response.result, response.err
		case <-ticker.C:
			if !sessionReady() {
				return ExecutionResult{}, ErrSessionExited
			}
		case <-ctx.Done():
			return ExecutionResult{}, ctx.Err()
		}
	}
}

func (manager *Manager) persistFailure(run Run, status RunStatus, code string, cause error) (Run, error) {
	completed := manager.now().UTC()
	run.Status = status
	run.CompletedAt = &completed
	run.FailureCode = code
	run.FailureDetail = boundedError(cause)
	if status == RunUnavailable {
		run.Limitations = []string{"configured network evidence probes were unavailable"}
	}
	if err := manager.store.Save(run); err != nil {
		return run, fmt.Errorf("%s; persist network evidence failure: %w", boundedError(cause), err)
	}
	return run, cause
}

func validateRunRequest(request RunRequest, now time.Time) (RouteIdentity, string, error) {
	if strings.TrimSpace(request.Profile.ID) == "" || strings.TrimSpace(request.Profile.Kernel.ID) == "" {
		return RouteIdentity{}, "", fmt.Errorf("network evidence requires a managed profile and kernel")
	}
	if err := request.ProbeSet.Validate(); err != nil {
		return RouteIdentity{}, "", err
	}
	base := request.BaseEvidence
	if err := base.Validate(); err != nil {
		return RouteIdentity{}, "", fmt.Errorf("validate base evidence: %w", err)
	}
	if base.ProfileID != request.Profile.ID || base.ProviderID != request.Profile.Kernel.Provider || base.BrowserVersion != request.Profile.Kernel.Version {
		return RouteIdentity{}, "", fmt.Errorf("base evidence does not match the selected profile and kernel")
	}
	if base.Status != evidence.RunPassed && base.Status != evidence.RunPartial {
		return RouteIdentity{}, "", fmt.Errorf("network evidence requires a passed or partial base browser evidence run")
	}
	if base.CompletedAt == nil || !base.ExpiresAt.After(now) || !validSHA256(base.ConsistencyInputDigest) {
		return RouteIdentity{}, "", fmt.Errorf("base evidence is incomplete, expired, or lacks M4.3 consistency metadata")
	}
	if request.Session.ProfileID != request.Profile.ID || request.Session.State != supervisor.StateReady || request.Session.CDPPort < 1 || strings.TrimSpace(request.Session.WebSocketDebuggerURL) == "" {
		return RouteIdentity{}, "", fmt.Errorf("network evidence requires the selected ready managed browser session")
	}
	route, err := RouteForProfile(request.Profile)
	if err != nil {
		return RouteIdentity{}, "", err
	}
	binaryDigest, err := digestBinaryIdentity(base)
	if err != nil {
		return RouteIdentity{}, "", err
	}
	return route, binaryDigest, nil
}

func digestBinaryIdentity(base evidence.Run) (string, error) {
	encoded, err := json.Marshal(base.BinaryIdentity)
	if err != nil {
		return "", fmt.Errorf("encode binary identity digest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateExecutionResult(result ExecutionResult, set ProbeSet) error {
	definitions := make(map[string]ProbeDefinition, len(set.Definitions))
	for _, definition := range set.Definitions {
		definitions[fmt.Sprintf("%s@%d", definition.ID, definition.Revision)] = definition
	}
	if len(result.Observations) > maxObservations || len(result.Limitations) > maxLimitations {
		return fmt.Errorf("network probe result exceeds bounded lists")
	}
	for index, observation := range result.Observations {
		definition, exists := definitions[fmt.Sprintf("%s@%d", observation.ProbeID, observation.ProbeRevision)]
		if !exists || definition.Kind != observation.ProbeKind {
			return fmt.Errorf("observation %d does not match the selected probe set", index)
		}
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("observation %d: %w", index, err)
		}
	}
	return nil
}

func deriveRunStatus(observations []Observation) RunStatus {
	if len(observations) == 0 {
		return RunUnavailable
	}
	passed := 0
	limited := 0
	for _, observation := range observations {
		switch observation.Status {
		case ObservationFailed:
			return RunFailed
		case ObservationPassed:
			passed++
		case ObservationPartial, ObservationUnavailable, ObservationSkipped:
			limited++
		}
	}
	if passed == 0 {
		return RunUnavailable
	}
	if limited > 0 {
		return RunPartial
	}
	return RunPassed
}

func classifyExecutionError(err error) (RunStatus, string) {
	switch {
	case errors.Is(err, context.Canceled):
		return RunCancelled, "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return RunIncomplete, "probe-timeout"
	case errors.Is(err, ErrSessionExited):
		return RunIncomplete, "browser-exited"
	case errors.Is(err, ErrProbeUnavailable):
		return RunUnavailable, "probe-unavailable"
	default:
		return RunFailed, "probe-execution-failed"
	}
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimSpace(err.Error())
	if len(value) > 1024 {
		return value[:1024]
	}
	return value
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
