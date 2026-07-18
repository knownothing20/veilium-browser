package evidence

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestCustomProviderEvidenceCannotBecomePassed(t *testing.T) {
	profile := evidenceProfile()
	capabilities, err := fingerprint.For(fingerprint.ProviderCustom, "148.0.0")
	if err != nil {
		t.Fatal(err)
	}
	submission := fullMatchingSubmission()
	evaluation, err := Evaluate(profile, capabilities, submission)
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Status != RunPartial {
		t.Fatalf("custom provider evidence became %s", evaluation.Status)
	}
	if !containsValue(evaluation.Limitations, "provider-trust:custom") {
		t.Fatalf("missing custom trust limitation: %#v", evaluation.Limitations)
	}
}

func TestCrossContextConflictFailsRun(t *testing.T) {
	profile := evidenceProfile()
	capabilities, _ := fingerprint.For(fingerprint.ProviderCustom, "148.0.0")
	submission := fullMatchingSubmission()
	for index := range submission.Contexts {
		if submission.Contexts[index].Context == ContextWorker {
			submission.Contexts[index].UserAgent = "Different browser"
		}
	}
	evaluation, err := Evaluate(profile, capabilities, submission)
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Status != RunFailed {
		t.Fatalf("expected failed context conflict, got %s", evaluation.Status)
	}
	if !hasObservation(evaluation.Observations, "worker.consistency.userAgent", ObservationFailed) {
		t.Fatalf("missing failed worker consistency observation: %#v", evaluation.Observations)
	}
}

func TestMissingContextIsIncomplete(t *testing.T) {
	profile := evidenceProfile()
	capabilities, _ := fingerprint.For(fingerprint.ProviderCustom, "148.0.0")
	evaluation, err := Evaluate(profile, capabilities, validBrowserSubmission())
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Status != RunIncomplete {
		t.Fatalf("expected incomplete evidence, got %s", evaluation.Status)
	}
	if !containsValue(evaluation.Limitations, "iframe:missing") || !containsValue(evaluation.Limitations, "worker:missing") {
		t.Fatalf("missing context limitations: %#v", evaluation.Limitations)
	}
}

func TestLegacySurfaceEvidenceCanMatchWithoutCreatingReviewedTrust(t *testing.T) {
	profile := evidenceProfile()
	profile.Kernel.Provider = fingerprint.ProviderPatched
	capabilities, err := fingerprint.For(fingerprint.ProviderPatched, "148.0.0")
	if err != nil {
		t.Fatal(err)
	}
	requested := RequestedSurfaces(capabilities)
	if len(requested) != 4 {
		t.Fatalf("expected four relevant legacy surfaces, got %#v", requested)
	}
	submission := fullMatchingSubmission()
	for index := range submission.Contexts {
		if submission.Contexts[index].Context == ContextTopLevel || submission.Contexts[index].Context == ContextIframe {
			submission.Contexts[index].SurfaceDigests = map[string]string{
				"canvas":      strings.Repeat("a", 64),
				"webgl":       strings.Repeat("b", 64),
				"audio":       strings.Repeat("c", 64),
				"clientRects": strings.Repeat("d", 64),
			}
		}
	}
	evaluation, err := Evaluate(profile, capabilities, submission)
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Status != RunPartial {
		t.Fatalf("legacy provider surface evidence became %s", evaluation.Status)
	}
	if !hasObservation(evaluation.Observations, "surface.canvas", ObservationPassed) {
		t.Fatalf("matching surface observation missing: %#v", evaluation.Observations)
	}
}

func TestEvaluateRejectsProviderMismatch(t *testing.T) {
	profile := evidenceProfile()
	capabilities, _ := fingerprint.For(fingerprint.ProviderPatched, "148.0.0")
	if _, err := Evaluate(profile, capabilities, fullMatchingSubmission()); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected provider mismatch, got %v", err)
	}
}

func evidenceProfile() domain.Profile {
	return domain.Profile{
		ID:   "profile-1",
		Name: "Profile One",
		Kernel: domain.KernelRef{
			Provider:   fingerprint.ProviderCustom,
			Version:    "148.0.0",
			Executable: "/managed/chrome",
		},
		Fingerprint: domain.FingerprintConfig{
			Platform:            "windows",
			Brand:               "Chromium",
			Language:            "en-US",
			Timezone:            "America/Los_Angeles",
			ScreenWidth:         1920,
			ScreenHeight:        1080,
			HardwareConcurrency: 8,
			WebRTCPolicy:        "proxy-only",
			CanvasMode:          "native",
			AudioMode:           "native",
			FontMode:            "native",
			ClientRectsMode:     "native",
			GPUProfile:          "auto",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
}

func fullMatchingSubmission() BrowserSubmission {
	top := validBrowserSubmission().Contexts[0]
	frame := top
	frame.Context = ContextIframe
	worker := top
	worker.Context = ContextWorker
	worker.Screen = nil
	worker.Window = nil
	worker.WebRTC = nil
	worker.SurfaceDigests = nil
	return BrowserSubmission{SchemaVersion: PayloadSchemaVersion, Contexts: []BrowserSnapshot{top, frame, worker}}
}

func hasObservation(observations []Observation, id string, status ObservationStatus) bool {
	for _, observation := range observations {
		if observation.ID == id && observation.Status == status {
			return true
		}
	}
	return false
}

func containsValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
