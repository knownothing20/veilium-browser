package consistency

import (
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

func TestLegacyWindowFallbackIsDegradedWithoutEvidence(t *testing.T) {
	profile := testProfile()
	capabilities, err := fingerprint.For(fingerprint.ProviderCustom, profile.Kernel.Version)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Evaluate(EvaluationInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: testIdentity(capabilities),
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != HealthDegraded {
		t.Fatalf("expected degraded custom profile, got %s", result.Status)
	}
	if !contains(result.DegradedReasons, "legacy-window-fallback") || !contains(result.DegradedReasons, "evidence-unavailable") {
		t.Fatalf("missing compatibility reasons: %#v", result.DegradedReasons)
	}
}

func TestCustomProviderCannotDeclareAnotherHostPlatform(t *testing.T) {
	profile := testProfile()
	profile.Fingerprint.Platform = "linux"
	capabilities, _ := fingerprint.For(fingerprint.ProviderCustom, profile.Kernel.Version)
	result, err := Evaluate(EvaluationInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: testIdentity(capabilities),
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != HealthBlocked || !contains(result.BlockingReasons, "host-platform-mismatch") {
		t.Fatalf("expected blocked platform mismatch: %#v", result)
	}
}

func TestWindowCannotExceedDeclaredScreen(t *testing.T) {
	profile := testProfile()
	profile.Fingerprint.WindowWidth = 2200
	profile.Fingerprint.WindowHeight = 900
	profile.Fingerprint.DeviceScaleFactor = 1
	capabilities, _ := fingerprint.For(fingerprint.ProviderCustom, profile.Kernel.Version)
	result, err := Evaluate(EvaluationInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: testIdentity(capabilities),
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != HealthBlocked || !contains(result.BlockingReasons, "window-exceeds-screen") {
		t.Fatalf("expected oversized window block: %#v", result)
	}
}

func TestStaleEvidenceCannotRemainHealthy(t *testing.T) {
	profile := testProfile()
	profile.Fingerprint.WindowWidth = 1280
	profile.Fingerprint.WindowHeight = 800
	profile.Fingerprint.DeviceScaleFactor = 1
	capabilities := reviewedCapabilities()
	identity := testIdentity(capabilities)
	completed := time.Now().UTC().Add(-time.Minute)
	result, err := Evaluate(EvaluationInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: identity,
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1", Now: time.Now().UTC(),
		Evidence: &EvidenceInput{RunID: "old", InputDigest: strings.Repeat("a", 64), RunStatus: "passed", CompletedAt: &completed, ExpiresAt: completed.Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != HealthUnknown || result.EvidenceFresh {
		t.Fatalf("stale reviewed evidence must be unknown: %#v", result)
	}
}

func TestFreshReviewedWindowEvidenceCanBeHealthy(t *testing.T) {
	profile := testProfile()
	profile.Fingerprint.WindowWidth = 1280
	profile.Fingerprint.WindowHeight = 800
	profile.Fingerprint.DeviceScaleFactor = 1
	capabilities := reviewedCapabilities()
	identity := testIdentity(capabilities)
	digest, err := InputDigest(DigestInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: identity,
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	completed := time.Now().UTC().Add(-time.Second)
	evidence := &EvidenceInput{
		RunID: "fresh", InputDigest: digest, RunStatus: "passed", CompletedAt: &completed, ExpiresAt: completed.Add(time.Hour),
		Observations: []ObservationInput{
			{ID: "top-level.screen", Context: "top-level", Status: "partial", Observed: "1920x1080"},
			{ID: "top-level.window", Context: "top-level", Status: "partial", Observed: "outer=1280x800 inner=1264x720 viewport=1264.00x720.00@1.0000 dpr=1.0000"},
			{ID: "top-level.platform", Context: "top-level", Status: "passed", Expected: "windows", Observed: "windows"},
			{ID: "top-level.brand", Context: "top-level", Status: "passed", Expected: "Chromium", Observed: "Chromium"},
			{ID: "top-level.language", Context: "top-level", Status: "passed", Expected: "en-US", Observed: "en-US"},
			{ID: "top-level.timezone", Context: "top-level", Status: "passed", Expected: "UTC", Observed: "UTC"},
			{ID: "context.language", Context: "top-level", Status: "passed"},
		},
	}
	result, err := Evaluate(EvaluationInput{
		Profile: profile, Capabilities: capabilities, BinaryIdentity: identity,
		RuntimeOS: "windows", RuntimeArch: "amd64", HarnessRevision: "m4.2-v1", Now: time.Now().UTC(), Evidence: evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != HealthHealthy || !result.EvidenceFresh {
		t.Fatalf("expected healthy reviewed profile: %#v", result)
	}
}

func testProfile() domain.Profile {
	return domain.Profile{
		ID: "profile-a", Name: "Profile A",
		Kernel: domain.KernelRef{ID: "kernel-a", Provider: fingerprint.ProviderCustom, Version: "148.0.0", Executable: "/managed/chrome"},
		Fingerprint: domain.FingerprintConfig{
			Platform: "windows", Brand: "Chromium", Language: "en-US", Timezone: "UTC",
			ScreenWidth: 1920, ScreenHeight: 1080, WebRTCPolicy: "proxy-only",
			CanvasMode: "native", AudioMode: "native", FontMode: "native", ClientRectsMode: "native", GPUProfile: "native",
		},
	}
}

func reviewedCapabilities() fingerprint.Capabilities {
	return fingerprint.Capabilities{
		SchemaVersion: fingerprint.ContractSchemaVersion,
		Provider: "reviewed-test", Revision: 1, TrustStatus: fingerprint.TrustReviewed, MajorVersion: 148,
		Capabilities: map[fingerprint.CapabilityID]fingerprint.CapabilityDeclaration{
			fingerprint.CapabilityPlatformOverride: {ID: fingerprint.CapabilityPlatformOverride, Status: fingerprint.CapabilityVerified, EvidenceRequired: true},
		},
	}
}

func testIdentity(capabilities fingerprint.Capabilities) kernel.ProviderBinaryIdentity {
	return kernel.ProviderBinaryIdentity{
		SchemaVersion: kernel.BinaryIdentitySchemaVersion,
		ProviderID: capabilities.Provider, ProviderRevision: capabilities.Revision, ProviderTrust: capabilities.TrustStatus,
		BrowserVersion: "148.0.0", OperatingSystem: "windows", Architecture: "amd64",
		ExecutablePath: "/managed/chrome", ExecutableSize: 10, ExecutableSHA256: strings.Repeat("b", 64),
		IntegrityStatus: kernel.StatusVerified, Provenance: "test", Reviewed: capabilities.IsReviewed(),
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
