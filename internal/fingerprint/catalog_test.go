package fingerprint

import (
	"strings"
	"testing"
)

func TestLegacyProvidersNeverBecomeReviewed(t *testing.T) {
	for _, id := range []string{ProviderNative, ProviderPatched} {
		capabilities, err := For(id, "148.0.0")
		if err != nil {
			t.Fatalf("resolve %s: %v", id, err)
		}
		if capabilities.TrustStatus != TrustLegacy {
			t.Fatalf("expected legacy trust for %s, got %s", id, capabilities.TrustStatus)
		}
		for capabilityID, declaration := range capabilities.Capabilities {
			if declaration.Status == CapabilityVerified || declaration.Status == CapabilityPartial {
				t.Fatalf("legacy provider %s silently claimed %s for %s", id, declaration.Status, capabilityID)
			}
		}
	}
}

func TestCustomProviderExposesOnlyGenericUnverifiedPath(t *testing.T) {
	capabilities, err := For(ProviderCustom, "148.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.TrustStatus != TrustCustom {
		t.Fatalf("expected custom trust, got %s", capabilities.TrustStatus)
	}
	if capabilities.State(CapabilitySurfaceSeed) != CapabilityUnsupported {
		t.Fatalf("custom surface seed should be unsupported, got %s", capabilities.State(CapabilitySurfaceSeed))
	}
	if capabilities.State(CapabilityProxyOnlyWebRTC) != CapabilityUnverified {
		t.Fatalf("custom WebRTC policy should remain unverified, got %s", capabilities.State(CapabilityProxyOnlyWebRTC))
	}
}

func TestReviewedDefinitionRequiresProvenanceAndCanRepresentCandidate(t *testing.T) {
	definition := ProviderDefinition{
		SchemaVersion:         ContractSchemaVersion,
		Revision:              1,
		ID:                    "reviewed-candidate",
		Name:                  "Reviewed candidate",
		TrustStatus:           TrustReviewed,
		SourceURL:             "https://example.invalid/browser",
		LicenseSPDX:           "BSD-3-Clause",
		SupportedOS:           []string{"windows"},
		SupportedArch:         []string{"amd64"},
		Versions:              []string{"148.0.0"},
		ExpectedExecutable:    "chrome.exe",
		ProvenanceRequirement: "exact pinned digest",
		Capabilities: map[CapabilityID]CapabilityDeclaration{
			CapabilityPlatformOverride: declaration(CapabilityPlatformOverride, CapabilityUnverified, "real-browser evidence pending"),
		},
		CreatedAt: "2026-07-18T00:00:00Z",
	}
	if err := ValidateDefinition(definition); err != nil {
		t.Fatalf("valid reviewed candidate was rejected: %v", err)
	}

	definition.LicenseSPDX = ""
	if err := ValidateDefinition(definition); err == nil || !strings.Contains(err.Error(), "source and license") {
		t.Fatalf("expected missing reviewed license rejection, got %v", err)
	}
}

func TestCustomDefinitionCannotManufactureVerifiedCapability(t *testing.T) {
	definition := customDefinition()
	definition.Capabilities[CapabilityBrandOverride] = declaration(CapabilityBrandOverride, CapabilityVerified, "user claimed")
	if err := ValidateDefinition(definition); err == nil || !strings.Contains(err.Error(), "cannot claim") {
		t.Fatalf("expected custom reviewed-claim rejection, got %v", err)
	}
}

func TestDisabledDefinitionRequiresReasonAndFailsClosed(t *testing.T) {
	definition := customDefinition()
	definition.ID = "disabled-candidate"
	definition.TrustStatus = TrustDisabled
	definition.DisabledReason = ""
	if err := ValidateDefinition(definition); err == nil || !strings.Contains(err.Error(), "requires a reason") {
		t.Fatalf("expected disabled reason error, got %v", err)
	}
}
