package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const defaultRunTimeout = 15 * time.Second

var (
	ErrRunActive = errors.New("evidence run is already active for profile")
	ErrNoRun     = errors.New("no active evidence run for profile")
	ErrBrowserExit = errors.New("managed browser session exited during evidence collection")
)

type RunStore interface {
	Save(Run) error
	Get(string) (Run, error)
	List(string) ([]Run, error)
	Delete(string) error
	Retention() time.Duration
}

type CollectorHandle interface {
	URL() string
	Wait(context.Context) (BrowserSubmission, error)
	Close(context.Context) error
}

type CollectorFactory func(CollectorOptions) (CollectorHandle, error)

type TargetController interface {
	Open(context.Context, int, string) (Target, error)
	Close(context.Context, int, string) error
}

type RunRequest struct {
	Profile      domain.Profile
	Kernel       kernel.Record
	Session      supervisor.Session
	Capabilities fingerprint.Capabilities
	SessionReady func() bool
}

type ManagerOptions struct {
	Timeout          time.Duration
	Now              func() time.Time
	CollectorFactory CollectorFactory
	TargetController TargetController
}

type Manager struct {
	store            RunStore
	timeout          time.Duration
	now              func() time.Time
	collectorFactory CollectorFactory
	targetController TargetController

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func NewManager(store RunStore, options ManagerOptions) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("evidence store is required")
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultRunTimeout
	}
	if options.Timeout > 2*time.Minute {
		return nil, fmt.Errorf("evidence run timeout is too large")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.CollectorFactory == nil {
		options.CollectorFactory = func(options CollectorOptions) (CollectorHandle, error) {
			return StartCollector(options)
		}
	}
	if options.TargetController == nil {
		options.TargetController = NewTargetClient()
	}
	return &Manager{
		store:            store,
		timeout:          options.Timeout,
		now:              options.Now,
		collectorFactory: options.CollectorFactory,
		targetController: options.TargetController,
		active:           make(map[string]context.CancelFunc),
	}, nil
}

func (m *Manager) Run(ctx context.Context, request RunRequest) (Run, error) {
	identity, err := validateRunRequest(request)
	if err != nil {
		return Run{}, err
	}
	runID, err := NewRunID()
	if err != nil {
		return Run{}, err
	}
	started := m.now().UTC()
	run := Run{
		SchemaVersion:    SchemaVersion,
		ID:               runID,
		ProfileID:        request.Profile.ID,
		ProfileName:      request.Profile.Name,
		ProviderID:       request.Capabilities.Provider,
		ProviderRevision: request.Capabilities.Revision,
		ProviderTrust:    request.Capabilities.TrustStatus,
		BinaryIdentity:   identity,
		BrowserVersion:   request.Profile.Kernel.Version,
		OperatingSystem:  identity.OperatingSystem,
		Architecture:     identity.Architecture,
		HarnessRevision:  HarnessRevision,
		Status:           RunPending,
		StartedAt:        started,
		ExpiresAt:        started.Add(m.store.Retention()),
	}

	runContext, cancel := context.WithTimeout(nonNilContext(ctx), m.timeout)
	if err := m.activate(request.Profile.ID, cancel); err != nil {
		cancel()
		return Run{}, err
	}
	defer func() {
		cancel()
		m.deactivate(request.Profile.ID)
	}()

	run.Status = RunRunning
	collector, err := m.collectorFactory(CollectorOptions{RequestedSurfaces: RequestedSurfaces(request.Capabilities)})
	if err != nil {
		return m.persistFailure(run, RunFailed, "collector-start-failed", err)
	}
	var target Target
	targetOpened := false
	cleanupLimitations := make([]string, 0, 2)
	defer func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		if targetOpened {
			_ = m.targetController.Close(cleanupContext, request.Session.CDPPort, target.ID)
		}
		_ = collector.Close(cleanupContext)
	}()

	target, err = m.targetController.Open(runContext, request.Session.CDPPort, collector.URL())
	if err != nil {
		closeCollectorWithLimitation(collector, &cleanupLimitations)
		return m.persistFailureWithLimitations(run, RunFailed, "target-open-failed", err, cleanupLimitations)
	}
	targetOpened = true

	submission, collectionErr := waitForSubmission(runContext, collector, request.SessionReady)
	cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := m.targetController.Close(cleanupContext, request.Session.CDPPort, target.ID); err != nil {
		cleanupLimitations = append(cleanupLimitations, "target-close-failed:"+boundedError(err))
	}
	targetOpened = false
	if err := collector.Close(cleanupContext); err != nil {
		cleanupLimitations = append(cleanupLimitations, "collector-close-failed:"+boundedError(err))
	}
	cleanupCancel()

	if collectionErr != nil {
		status, code := classifyCollectionError(collectionErr)
		return m.persistFailureWithLimitations(run, status, code, collectionErr, cleanupLimitations)
	}
	evaluation, err := Evaluate(request.Profile, request.Capabilities, submission)
	if err != nil {
		return m.persistFailureWithLimitations(run, RunFailed, "evaluation-failed", err, cleanupLimitations)
	}
	completed := m.now().UTC()
	run.Status = evaluation.Status
	run.CompletedAt = &completed
	run.Observations = evaluation.Observations
	run.Limitations = sortedUnique(append(evaluation.Limitations, cleanupLimitations...))
	if len(cleanupLimitations) > 0 && run.Status == RunPassed {
		run.Status = RunPartial
	}
	if err := m.store.Save(run); err != nil {
		return run, fmt.Errorf("persist evidence run: %w", err)
	}
	return run, nil
}

func (m *Manager) Cancel(profileID string) error {
	profileID = strings.TrimSpace(profileID)
	m.mu.Lock()
	cancel, ok := m.active[profileID]
	m.mu.Unlock()
	if !ok {
		return ErrNoRun
	}
	cancel()
	return nil
}

func (m *Manager) List(profileID string) ([]Run, error) { return m.store.List(profileID) }
func (m *Manager) Get(id string) (Run, error)            { return m.store.Get(id) }
func (m *Manager) Delete(id string) error                { return m.store.Delete(id) }

func (m *Manager) IsActive(profileID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.active[strings.TrimSpace(profileID)]
	return ok
}

func (m *Manager) activate(profileID string, cancel context.CancelFunc) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return fmt.Errorf("profile id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.active[profileID]; exists {
		return ErrRunActive
	}
	m.active[profileID] = cancel
	return nil
}

func (m *Manager) deactivate(profileID string) {
	m.mu.Lock()
	delete(m.active, strings.TrimSpace(profileID))
	m.mu.Unlock()
}

func (m *Manager) persistFailure(run Run, status RunStatus, code string, cause error) (Run, error) {
	return m.persistFailureWithLimitations(run, status, code, cause, nil)
}

func (m *Manager) persistFailureWithLimitations(run Run, status RunStatus, code string, cause error, limitations []string) (Run, error) {
	completed := m.now().UTC()
	run.Status = status
	run.CompletedAt = &completed
	run.FailureCode = code
	run.FailureDetail = boundedError(cause)
	run.Limitations = sortedUnique(limitations)
	if err := m.store.Save(run); err != nil {
		return run, fmt.Errorf("%s; persist evidence failure: %w", boundedError(cause), err)
	}
	return run, cause
}

func validateRunRequest(request RunRequest) (kernel.ProviderBinaryIdentity, error) {
	if strings.TrimSpace(request.Profile.ID) == "" || strings.TrimSpace(request.Profile.Name) == "" {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("evidence profile id and name are required")
	}
	if strings.TrimSpace(request.Profile.Kernel.ID) == "" || request.Profile.Kernel.ID != request.Kernel.ID {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("evidence requires the profile's exact managed kernel")
	}
	if request.Kernel.Status != kernel.StatusVerified {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("evidence kernel failed integrity verification: %s", request.Kernel.Status)
	}
	if request.Profile.Kernel.Provider != request.Kernel.Provider || request.Profile.Kernel.Version != request.Kernel.Version || request.Profile.Kernel.Executable != request.Kernel.Executable {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("profile kernel reference does not match the verified managed kernel")
	}
	if request.Session.ProfileID != request.Profile.ID || request.Session.State != supervisor.StateReady || request.Session.CDPPort < 1 {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("evidence requires a ready managed browser session")
	}
	if request.Capabilities.Provider != request.Profile.Kernel.Provider || request.Capabilities.MajorVersion != parseMajor(request.Profile.Kernel.Version) {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("provider capabilities do not match the profile kernel")
	}
	identity, err := kernel.BinaryIdentity(request.Kernel)
	if err != nil {
		return kernel.ProviderBinaryIdentity{}, err
	}
	if identity.ProviderID != request.Capabilities.Provider || identity.ProviderRevision != request.Capabilities.Revision || identity.ProviderTrust != request.Capabilities.TrustStatus {
		return kernel.ProviderBinaryIdentity{}, fmt.Errorf("provider binary identity does not match the active provider contract")
	}
	return identity, nil
}

func waitForSubmission(ctx context.Context, collector CollectorHandle, sessionReady func() bool) (BrowserSubmission, error) {
	waitContext, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		submission BrowserSubmission
		err        error
	}
	results := make(chan result, 1)
	go func() {
		submission, err := collector.Wait(waitContext)
		results <- result{submission: submission, err: err}
	}()
	if sessionReady == nil {
		value := <-results
		return value.submission, value.err
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case value := <-results:
			return value.submission, value.err
		case <-ticker.C:
			if !sessionReady() {
				cancel()
				return BrowserSubmission{}, ErrBrowserExit
			}
		case <-ctx.Done():
			cancel()
			return BrowserSubmission{}, ctx.Err()
		}
	}
}

func classifyCollectionError(err error) (RunStatus, string) {
	switch {
	case errors.Is(err, context.Canceled):
		return RunCancelled, "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return RunIncomplete, "collection-timeout"
	case errors.Is(err, ErrBrowserExit):
		return RunIncomplete, "browser-exited"
	default:
		return RunFailed, "collection-failed"
	}
}

func closeCollectorWithLimitation(collector CollectorHandle, limitations *[]string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := collector.Close(ctx); err != nil {
		*limitations = append(*limitations, "collector-close-failed:"+boundedError(err))
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

func parseMajor(version string) int {
	major, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	value := 0
	_, _ = fmt.Sscanf(major, "%d", &value)
	return value
}
