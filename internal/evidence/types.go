package evidence

import (
	"fmt"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

const (
	SchemaVersion   = 1
	HarnessRevision = "m4.2-v1"
)

type RunStatus string

const (
	RunPending    RunStatus = "pending"
	RunRunning    RunStatus = "running"
	RunPassed     RunStatus = "passed"
	RunPartial    RunStatus = "partial"
	RunFailed     RunStatus = "failed"
	RunCancelled  RunStatus = "cancelled"
	RunIncomplete RunStatus = "incomplete"
)

type ObservationStatus string

const (
	ObservationPassed      ObservationStatus = "passed"
	ObservationPartial     ObservationStatus = "partial"
	ObservationFailed      ObservationStatus = "failed"
	ObservationUnavailable ObservationStatus = "unavailable"
	ObservationSkipped     ObservationStatus = "skipped"
)

type BrowserContext string

const (
	ContextTopLevel BrowserContext = "top-level"
	ContextIframe   BrowserContext = "iframe"
	ContextWorker   BrowserContext = "worker"
)

type Observation struct {
	ID           string                    `json:"id"`
	Context      BrowserContext            `json:"context"`
	CapabilityID fingerprint.CapabilityID  `json:"capabilityId,omitempty"`
	Status       ObservationStatus         `json:"status"`
	Expected     string                    `json:"expected,omitempty"`
	Observed     string                    `json:"observed,omitempty"`
	ReasonCode   string                    `json:"reasonCode,omitempty"`
	Detail       string                    `json:"detail,omitempty"`
}

type Run struct {
	SchemaVersion      int                           `json:"schemaVersion"`
	ID                 string                        `json:"id"`
	ProfileID          string                        `json:"profileId"`
	ProfileName        string                        `json:"profileName"`
	ProviderID         string                        `json:"providerId"`
	ProviderRevision   int                           `json:"providerRevision"`
	ProviderTrust      fingerprint.TrustStatus       `json:"providerTrust"`
	BinaryIdentity     kernel.ProviderBinaryIdentity `json:"binaryIdentity"`
	BrowserVersion     string                        `json:"browserVersion"`
	OperatingSystem    string                        `json:"operatingSystem"`
	Architecture       string                        `json:"architecture"`
	HarnessRevision    string                        `json:"harnessRevision"`
	Status             RunStatus                     `json:"status"`
	StartedAt          time.Time                     `json:"startedAt"`
	CompletedAt        *time.Time                    `json:"completedAt,omitempty"`
	ExpiresAt          time.Time                     `json:"expiresAt"`
	Observations       []Observation                 `json:"observations"`
	Limitations        []string                      `json:"limitations,omitempty"`
	FailureCode        string                        `json:"failureCode,omitempty"`
	FailureDetail      string                        `json:"failureDetail,omitempty"`
}

func (r Run) Validate() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported evidence schema version %d", r.SchemaVersion)
	}
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.ProfileID) == "" || strings.TrimSpace(r.ProfileName) == "" {
		return fmt.Errorf("evidence run id, profile id, and profile name are required")
	}
	if strings.TrimSpace(r.ProviderID) == "" || r.ProviderRevision < 1 {
		return fmt.Errorf("evidence provider id and revision are required")
	}
	if strings.TrimSpace(r.BrowserVersion) == "" || strings.TrimSpace(r.OperatingSystem) == "" || strings.TrimSpace(r.Architecture) == "" {
		return fmt.Errorf("evidence browser and platform identity are required")
	}
	if r.HarnessRevision != HarnessRevision {
		return fmt.Errorf("unsupported evidence harness revision %q", r.HarnessRevision)
	}
	if !validRunStatus(r.Status) {
		return fmt.Errorf("invalid evidence run status %q", r.Status)
	}
	if r.StartedAt.IsZero() || r.ExpiresAt.IsZero() || !r.ExpiresAt.After(r.StartedAt) {
		return fmt.Errorf("evidence timestamps are invalid")
	}
	if terminalRunStatus(r.Status) && r.CompletedAt == nil {
		return fmt.Errorf("terminal evidence run %q requires completion time", r.Status)
	}
	if r.Status == RunFailed && strings.TrimSpace(r.FailureCode) == "" {
		return fmt.Errorf("failed evidence run requires failure code")
	}
	if len(r.Observations) > 128 {
		return fmt.Errorf("evidence run contains too many observations")
	}
	for index, observation := range r.Observations {
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("observation %d: %w", index, err)
		}
	}
	if len(r.Limitations) > 64 {
		return fmt.Errorf("evidence run contains too many limitations")
	}
	return nil
}

func (o Observation) Validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("observation id is required")
	}
	if o.Context != ContextTopLevel && o.Context != ContextIframe && o.Context != ContextWorker {
		return fmt.Errorf("invalid browser context %q", o.Context)
	}
	if !validObservationStatus(o.Status) {
		return fmt.Errorf("invalid observation status %q", o.Status)
	}
	for label, value := range map[string]string{
		"expected": o.Expected,
		"observed": o.Observed,
		"reason":   o.ReasonCode,
		"detail":   o.Detail,
	} {
		if len(value) > 4096 {
			return fmt.Errorf("%s value exceeds the evidence limit", label)
		}
	}
	return nil
}

func validRunStatus(status RunStatus) bool {
	switch status {
	case RunPending, RunRunning, RunPassed, RunPartial, RunFailed, RunCancelled, RunIncomplete:
		return true
	default:
		return false
	}
}

func terminalRunStatus(status RunStatus) bool {
	switch status {
	case RunPassed, RunPartial, RunFailed, RunCancelled, RunIncomplete:
		return true
	default:
		return false
	}
}

func validObservationStatus(status ObservationStatus) bool {
	switch status {
	case ObservationPassed, ObservationPartial, ObservationFailed, ObservationUnavailable, ObservationSkipped:
		return true
	default:
		return false
	}
}
