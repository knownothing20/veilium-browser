package consistency

import (
	"fmt"
	"strings"
	"time"
)

const (
	SchemaVersion = 1
	RulesRevision = "m4.3-v1"
)

type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthDegraded HealthStatus = "degraded"
	HealthBlocked  HealthStatus = "blocked"
	HealthUnknown  HealthStatus = "unknown"
)

type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckWarning CheckStatus = "warning"
	CheckFailed  CheckStatus = "failed"
	CheckUnknown CheckStatus = "unknown"
)

type WindowSource string

const (
	WindowExplicit       WindowSource = "explicit"
	WindowLegacyFallback WindowSource = "legacy-screen-fallback"
)

type WindowSpec struct {
	Width             int          `json:"width"`
	Height            int          `json:"height"`
	DeviceScaleFactor float64      `json:"deviceScaleFactor"`
	Source            WindowSource `json:"source"`
}

type Check struct {
	ID         string      `json:"id"`
	Status     CheckStatus `json:"status"`
	Expected   string      `json:"expected,omitempty"`
	Observed   string      `json:"observed,omitempty"`
	ReasonCode string      `json:"reasonCode,omitempty"`
	Detail     string      `json:"detail,omitempty"`
}

type ObservationInput struct {
	ID           string `json:"id"`
	Context      string `json:"context"`
	CapabilityID string `json:"capabilityId,omitempty"`
	Status       string `json:"status"`
	Expected     string `json:"expected,omitempty"`
	Observed     string `json:"observed,omitempty"`
	ReasonCode   string `json:"reasonCode,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

type EvidenceInput struct {
	RunID         string             `json:"runId"`
	InputDigest   string             `json:"inputDigest,omitempty"`
	RunStatus     string             `json:"runStatus"`
	FailureCode   string             `json:"failureCode,omitempty"`
	CompletedAt   *time.Time         `json:"completedAt,omitempty"`
	ExpiresAt     time.Time          `json:"expiresAt"`
	Observations  []ObservationInput `json:"observations"`
	Limitations   []string           `json:"limitations,omitempty"`
}

type Result struct {
	SchemaVersion    int          `json:"schemaVersion"`
	RulesRevision    string       `json:"rulesRevision"`
	ProfileID        string       `json:"profileId"`
	InputDigest      string       `json:"inputDigest"`
	EvidenceRunID    string       `json:"evidenceRunId,omitempty"`
	EvidenceFresh    bool         `json:"evidenceFresh"`
	Status           HealthStatus `json:"status"`
	Window           WindowSpec   `json:"window"`
	Checks           []Check      `json:"checks"`
	BlockingReasons  []string     `json:"blockingReasons,omitempty"`
	DegradedReasons  []string     `json:"degradedReasons,omitempty"`
	GeneratedAt      time.Time    `json:"generatedAt"`
}

func (r Result) Validate() error {
	if r.SchemaVersion != SchemaVersion || r.RulesRevision != RulesRevision {
		return fmt.Errorf("unsupported consistency result contract")
	}
	if strings.TrimSpace(r.ProfileID) == "" || strings.TrimSpace(r.InputDigest) == "" {
		return fmt.Errorf("consistency profile and input digest are required")
	}
	if !validHealthStatus(r.Status) {
		return fmt.Errorf("invalid profile health status %q", r.Status)
	}
	if r.GeneratedAt.IsZero() {
		return fmt.Errorf("consistency generation time is required")
	}
	if len(r.Checks) == 0 || len(r.Checks) > 128 {
		return fmt.Errorf("consistency result requires one to 128 checks")
	}
	for _, check := range r.Checks {
		if strings.TrimSpace(check.ID) == "" || !validCheckStatus(check.Status) {
			return fmt.Errorf("invalid consistency check")
		}
	}
	return nil
}

func validHealthStatus(status HealthStatus) bool {
	switch status {
	case HealthHealthy, HealthDegraded, HealthBlocked, HealthUnknown:
		return true
	default:
		return false
	}
}

func validCheckStatus(status CheckStatus) bool {
	switch status {
	case CheckPassed, CheckWarning, CheckFailed, CheckUnknown:
		return true
	default:
		return false
	}
}
