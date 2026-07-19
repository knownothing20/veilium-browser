package desktop

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/consistency"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

func (s *Service) validateProfileConsistency(item domain.Profile) error {
	capabilities, err := fingerprint.For(item.Kernel.Provider, item.Kernel.Version)
	if err != nil {
		return err
	}
	_, checks, err := consistency.Preflight(item, capabilities, runtime.GOOS)
	if err != nil {
		return err
	}
	failures := make([]string, 0, 4)
	for _, check := range checks {
		if check.Status != consistency.CheckFailed {
			continue
		}
		detail := strings.TrimSpace(check.Detail)
		if detail == "" {
			detail = strings.TrimSpace(check.ReasonCode)
		}
		if detail == "" {
			detail = "consistency check failed"
		}
		failures = append(failures, check.ID+": "+detail)
	}
	if len(failures) > 0 {
		return fmt.Errorf("profile consistency blocked: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *Service) ProfileConsistency(profileID string) (consistency.Result, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return consistency.Result{}, fmt.Errorf("profile id is required")
	}
	item, err := s.store.Get(profileID)
	if err != nil {
		return consistency.Result{}, err
	}
	if strings.TrimSpace(item.Kernel.ID) == "" {
		return consistency.Result{}, fmt.Errorf("profile %q must use a managed kernel before health can be evaluated", item.Name)
	}
	record, err := s.kernels.Verify(item.Kernel.ID)
	if err != nil {
		return consistency.Result{}, err
	}
	if record.Status != kernel.StatusVerified {
		return consistency.Result{}, fmt.Errorf("kernel %q failed integrity verification: %s", record.Name, record.Status)
	}
	if err := s.resolveKernel(&item); err != nil {
		return consistency.Result{}, err
	}
	capabilities, err := fingerprint.For(item.Kernel.Provider, item.Kernel.Version)
	if err != nil {
		return consistency.Result{}, err
	}
	identity, err := kernel.BinaryIdentity(record)
	if err != nil {
		return consistency.Result{}, err
	}
	var latest *consistency.EvidenceInput
	runs, err := s.ListEvidence(item.ID)
	if err != nil {
		return consistency.Result{}, err
	}
	for _, run := range runs {
		if run.CompletedAt == nil {
			continue
		}
		converted := evidenceForConsistency(run)
		latest = &converted
		break
	}
	return consistency.Evaluate(consistency.EvaluationInput{
		Profile:         item,
		Capabilities:    capabilities,
		BinaryIdentity:  identity,
		RuntimeOS:       runtime.GOOS,
		RuntimeArch:     runtime.GOARCH,
		HarnessRevision: evidence.HarnessRevision,
		Evidence:        latest,
		Now:             time.Now().UTC(),
	})
}

func evidenceForConsistency(run evidence.Run) consistency.EvidenceInput {
	observations := make([]consistency.ObservationInput, 0, len(run.Observations))
	for _, observation := range run.Observations {
		observations = append(observations, consistency.ObservationInput{
			ID:           observation.ID,
			Context:      string(observation.Context),
			CapabilityID: string(observation.CapabilityID),
			Status:       string(observation.Status),
			Expected:     observation.Expected,
			Observed:     observation.Observed,
			ReasonCode:   observation.ReasonCode,
			Detail:       observation.Detail,
		})
	}
	return consistency.EvidenceInput{
		RunID:        run.ID,
		InputDigest:  run.ConsistencyInputDigest,
		RunStatus:    string(run.Status),
		FailureCode:  run.FailureCode,
		CompletedAt:  run.CompletedAt,
		ExpiresAt:    run.ExpiresAt,
		Observations: observations,
		Limitations:  append([]string(nil), run.Limitations...),
	}
}
