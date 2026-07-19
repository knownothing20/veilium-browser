package networkevidence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/evidence"
)

type BrowserCollector interface {
	URL() string
	Wait(context.Context) (BrowserSubmission, error)
	Close(context.Context) error
}

type BrowserCollectorFactory func(ProbeSet) (BrowserCollector, error)

type BrowserTargetController interface {
	Open(context.Context, int, string) (evidence.Target, error)
	Close(context.Context, int, string) error
}

type BrowserExecutorOptions struct {
	CollectorFactory BrowserCollectorFactory
	TargetController BrowserTargetController
	Now              func() time.Time
	CleanupTimeout   time.Duration
}

type BrowserExecutor struct {
	collectorFactory BrowserCollectorFactory
	targetController BrowserTargetController
	now              func() time.Time
	cleanupTimeout   time.Duration
}

func NewBrowserExecutor(options BrowserExecutorOptions) (*BrowserExecutor, error) {
	if options.CollectorFactory == nil {
		return nil, fmt.Errorf("browser network collector factory is required")
	}
	if options.TargetController == nil {
		options.TargetController = evidence.NewTargetClient()
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.CleanupTimeout <= 0 {
		options.CleanupTimeout = 3 * time.Second
	}
	if options.CleanupTimeout > 30*time.Second {
		return nil, fmt.Errorf("browser network cleanup timeout is too large")
	}
	return &BrowserExecutor{
		collectorFactory: options.CollectorFactory,
		targetController: options.TargetController,
		now:              options.Now,
		cleanupTimeout:   options.CleanupTimeout,
	}, nil
}

func (executor *BrowserExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if executor == nil || executor.collectorFactory == nil || executor.targetController == nil {
		return ExecutionResult{}, fmt.Errorf("browser network executor is unavailable")
	}
	if err := request.ProbeSet.Validate(); err != nil {
		return ExecutionResult{}, err
	}
	if request.Session.CDPPort < 1 || strings.TrimSpace(request.Session.WebSocketDebuggerURL) == "" {
		return ExecutionResult{}, fmt.Errorf("browser network executor requires a ready managed CDP session")
	}
	collector, err := executor.collectorFactory(request.ProbeSet)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("start browser network collector: %w", err)
	}
	cleanupLimitations := make([]string, 0, 2)
	target, targetErr := executor.targetController.Open(nonNilContext(ctx), request.Session.CDPPort, collector.URL())
	if targetErr != nil {
		closeCollector(executor.cleanupTimeout, collector, &cleanupLimitations)
		return ExecutionResult{Limitations: cleanupLimitations}, fmt.Errorf("open controlled browser network target: %w", targetErr)
	}

	submission, collectionErr := collector.Wait(nonNilContext(ctx))
	cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), executor.cleanupTimeout)
	if err := executor.targetController.Close(cleanupContext, request.Session.CDPPort, target.ID); err != nil {
		cleanupLimitations = append(cleanupLimitations, "target-close-failed:"+boundedError(err))
	}
	if err := collector.Close(cleanupContext); err != nil {
		cleanupLimitations = append(cleanupLimitations, "collector-close-failed:"+boundedError(err))
	}
	cleanupCancel()
	if collectionErr != nil {
		return ExecutionResult{Limitations: sortedUnique(cleanupLimitations)}, collectionErr
	}
	submission = NormalizeBrowserSubmission(submission)
	if err := submission.Validate(request.ProbeSet); err != nil {
		return ExecutionResult{Limitations: sortedUnique(cleanupLimitations)}, fmt.Errorf("validate controlled browser network submission: %w", err)
	}
	collectedAt := executor.now().UTC()
	observations := make([]Observation, 0, len(submission.Observations))
	for _, browserObservation := range submission.Observations {
		observations = append(observations, browserObservation.Observation(collectedAt))
	}
	limitations := sortedUnique(append(submission.Limitations, cleanupLimitations...))
	return ExecutionResult{Observations: observations, Limitations: limitations}, nil
}

func closeCollector(timeout time.Duration, collector BrowserCollector, limitations *[]string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := collector.Close(ctx); err != nil {
		*limitations = append(*limitations, "collector-close-failed:"+boundedError(err))
	}
}
