package fingerprint

import (
	"strings"
	"testing"
)

func TestReplacementRequiresExplicitPredecessorForNewProviderID(t *testing.T) {
	current := customDefinition()
	candidate := customDefinition()
	candidate.ID = "custom-chromium-v2"
	candidate.Revision = 1
	if err := ValidateReplacement(current, candidate); err == nil || !strings.Contains(err.Error(), "does not explicitly name") {
		t.Fatalf("expected predecessor error, got %v", err)
	}
	candidate.PredecessorIDs = []string{current.ID}
	if err := ValidateReplacement(current, candidate); err != nil {
		t.Fatalf("valid replacement rejected: %v", err)
	}
}

func TestSameProviderIdentityRequiresHigherRevision(t *testing.T) {
	current := customDefinition()
	candidate := customDefinition()
	candidate.Revision = current.Revision
	if err := ValidateReplacement(current, candidate); err == nil || !strings.Contains(err.Error(), "must advance") {
		t.Fatalf("expected revision error, got %v", err)
	}
	candidate.Revision = current.Revision + 1
	if err := ValidateReplacement(current, candidate); err != nil {
		t.Fatalf("valid same-provider revision rejected: %v", err)
	}
}

func TestSameReviewedIdentityCannotSilentlyChangeSource(t *testing.T) {
	current := reviewedFixture("reviewed-one", 1)
	candidate := reviewedFixture("reviewed-one", 2)
	candidate.SourceURL = "https://other.example.invalid/browser"
	if err := ValidateReplacement(current, candidate); err == nil || !strings.Contains(err.Error(), "new provider identity") {
		t.Fatalf("expected reviewed source-change rejection, got %v", err)
	}
}

func TestNewReviewedIdentityMayUseNewSourceWithExplicitPredecessor(t *testing.T) {
	current := reviewedFixture("reviewed-one", 1)
	candidate := reviewedFixture("reviewed-two", 1)
	candidate.SourceURL = "https://other.example.invalid/browser"
	candidate.PredecessorIDs = []string{current.ID}
	if err := ValidateReplacement(current, candidate); err != nil {
		t.Fatalf("new reviewed provider identity was rejected: %v", err)
	}
}

func reviewedFixture(id string, revision int) ProviderDefinition {
	return ProviderDefinition{
		SchemaVersion:         ContractSchemaVersion,
		Revision:              revision,
		ID:                    id,
		Name:                  id,
		TrustStatus:           TrustReviewed,
		SourceURL:             "https://example.invalid/browser",
		LicenseSPDX:           "BSD-3-Clause",
		SupportedOS:           []string{"windows"},
		SupportedArch:         []string{"amd64"},
		Versions:              []string{"148.0.0"},
		ExpectedExecutable:    "chrome.exe",
		ProvenanceRequirement: "exact pinned digest",
		Capabilities: map[CapabilityID]CapabilityDeclaration{
			CapabilityPlatformOverride: declaration(CapabilityPlatformOverride, CapabilityUnverified, "evidence pending"),
		},
		CreatedAt: "2026-07-18T00:00:00Z",
	}
}
