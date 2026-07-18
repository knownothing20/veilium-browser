package consistency

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

type EvaluationInput struct {
	Profile         domain.Profile
	Capabilities    fingerprint.Capabilities
	BinaryIdentity  kernel.ProviderBinaryIdentity
	RuntimeOS       string
	RuntimeArch     string
	HarnessRevision string
	Evidence        *EvidenceInput
	Now             time.Time
}

func Evaluate(input EvaluationInput) (Result, error) {
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	input.Now = input.Now.UTC()
	window, preflightChecks, err := Preflight(input.Profile, input.Capabilities, input.RuntimeOS)
	if err != nil {
		return Result{}, err
	}
	digest, err := InputDigest(DigestInput{
		Profile:         input.Profile,
		Capabilities:    input.Capabilities,
		BinaryIdentity:  input.BinaryIdentity,
		RuntimeOS:       input.RuntimeOS,
		RuntimeArch:     input.RuntimeArch,
		HarnessRevision: input.HarnessRevision,
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{
		SchemaVersion: SchemaVersion,
		RulesRevision: RulesRevision,
		ProfileID:     input.Profile.ID,
		InputDigest:   digest,
		Status:        HealthUnknown,
		Window:        window,
		Checks:        append([]Check(nil), preflightChecks...),
		GeneratedAt:   input.Now,
	}
	collectReasons(&result)
	if hasFailedCheck(result.Checks) {
		result.Status = HealthBlocked
		return result, result.Validate()
	}

	if input.Evidence == nil {
		result.Checks = append(result.Checks, unknown("evidence.freshness", digest, "none", "evidence-unavailable", "No applicable real-browser evidence run exists."))
		finishStatus(&result, input.Capabilities)
		return result, result.Validate()
	}
	result.EvidenceRunID = input.Evidence.RunID
	if input.Evidence.CompletedAt == nil || (!input.Evidence.ExpiresAt.IsZero() && !input.Evidence.ExpiresAt.After(input.Now)) {
		result.Checks = append(result.Checks, unknown("evidence.freshness", digest, input.Evidence.InputDigest, "evidence-expired-or-incomplete", "The latest evidence run is incomplete or expired."))
		finishStatus(&result, input.Capabilities)
		return result, result.Validate()
	}
	if strings.TrimSpace(input.Evidence.InputDigest) == "" || input.Evidence.InputDigest != digest {
		result.Checks = append(result.Checks, unknown("evidence.freshness", digest, input.Evidence.InputDigest, "evidence-stale", "Profile, Provider, binary, runtime, harness, or consistency inputs changed after evidence collection."))
		finishStatus(&result, input.Capabilities)
		return result, result.Validate()
	}
	result.EvidenceFresh = true
	result.Checks = append(result.Checks, passed("evidence.freshness", digest, input.Evidence.InputDigest))
	appendEvidenceChecks(&result, input.Profile, input.Evidence)
	result.DegradedReasons = append(result.DegradedReasons, input.Evidence.Limitations...)
	collectReasons(&result)
	finishStatus(&result, input.Capabilities)
	return result, result.Validate()
}

func appendEvidenceChecks(result *Result, profile domain.Profile, evidence *EvidenceInput) {
	for _, observation := range evidence.Observations {
		if observation.Context != "top-level" && !strings.HasPrefix(observation.ID, "context.") {
			continue
		}
		switch observation.ID {
		case "top-level.screen":
			result.Checks = append(result.Checks, screenCheck(profile, observation))
		case "top-level.window":
			observed, err := ParseObservedWindow(observation.Observed)
			if err != nil {
				result.Checks = append(result.Checks, failed("window.observation", fmt.Sprintf("%dx%d", result.Window.Width, result.Window.Height), observation.Observed, "invalid-window-observation", err.Error()))
				continue
			}
			result.Checks = append(result.Checks, WindowChecks(result.Window, observed)...)
		case "top-level.platform", "top-level.brand", "top-level.language", "top-level.timezone", "top-level.hardwareConcurrency":
			result.Checks = append(result.Checks, convertObservation(observation))
		default:
			if strings.HasPrefix(observation.ID, "context.") {
				result.Checks = append(result.Checks, convertObservation(observation))
			}
		}
	}
	if evidence.RunStatus == "failed" && !hasFailedCheck(result.Checks) {
		result.Checks = append(result.Checks, unknown("evidence.run", "completed observations", evidence.FailureCode, "evidence-run-failed", "The evidence run failed without an applicable identity mismatch."))
	}
}

func screenCheck(profile domain.Profile, observation ObservationInput) Check {
	var width, height int
	count, err := fmt.Sscanf(strings.TrimSpace(observation.Observed), "%dx%d", &width, &height)
	expected := fmt.Sprintf("%dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight)
	if err != nil || count != 2 {
		return failed("screen.observation", expected, observation.Observed, "invalid-screen-observation", "The real-browser screen observation could not be parsed.")
	}
	observed := fmt.Sprintf("%dx%d", width, height)
	if withinInt(width, profile.Fingerprint.ScreenWidth, 1) && withinInt(height, profile.Fingerprint.ScreenHeight, 1) {
		return passed("screen.observation", expected, observed)
	}
	return failed("screen.observation", expected, observed, "screen-mismatch", "Observed screen dimensions do not match the profile declaration.")
}

func convertObservation(observation ObservationInput) Check {
	id := "evidence." + observation.ID
	switch observation.Status {
	case "passed":
		return passed(id, observation.Expected, observation.Observed)
	case "failed":
		return failed(id, observation.Expected, observation.Observed, firstNonEmpty(observation.ReasonCode, "evidence-mismatch"), observation.Detail)
	case "partial":
		return warning(id, observation.Expected, observation.Observed, firstNonEmpty(observation.ReasonCode, "evidence-partial"), observation.Detail)
	default:
		return unknown(id, observation.Expected, observation.Observed, firstNonEmpty(observation.ReasonCode, "evidence-unavailable"), observation.Detail)
	}
}

func finishStatus(result *Result, capabilities fingerprint.Capabilities) {
	collectReasons(result)
	if hasFailedCheck(result.Checks) {
		result.Status = HealthBlocked
		return
	}
	if capabilities.TrustStatus != fingerprint.TrustReviewed || len(result.DegradedReasons) > 0 {
		result.Status = HealthDegraded
		return
	}
	if !result.EvidenceFresh || hasUnknownCheck(result.Checks) {
		result.Status = HealthUnknown
		return
	}
	result.Status = HealthHealthy
}

func collectReasons(result *Result) {
	for _, check := range result.Checks {
		message := check.ReasonCode
		if strings.TrimSpace(message) == "" {
			continue
		}
		if check.Status == CheckFailed {
			result.BlockingReasons = append(result.BlockingReasons, message)
		} else if check.Status == CheckWarning || check.Status == CheckUnknown {
			result.DegradedReasons = append(result.DegradedReasons, message)
		}
	}
	result.BlockingReasons = sortedUnique(result.BlockingReasons)
	result.DegradedReasons = sortedUnique(result.DegradedReasons)
}

func hasFailedCheck(checks []Check) bool {
	for _, check := range checks {
		if check.Status == CheckFailed {
			return true
		}
	}
	return false
}

func hasUnknownCheck(checks []Check) bool {
	for _, check := range checks {
		if check.Status == CheckUnknown {
			return true
		}
	}
	return false
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
