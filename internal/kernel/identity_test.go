package kernel

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestBinaryIdentityDoesNotUpgradeCustomIntegrityToReviewed(t *testing.T) {
	identity, err := BinaryIdentity(Record{
		ID:         "custom-1",
		Provider:   fingerprint.ProviderCustom,
		Version:    "148.0.0",
		Executable: "/managed/chrome",
		SHA256:     strings.Repeat("a", 64),
		SizeBytes:  10,
		Status:     StatusVerified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProviderTrust != fingerprint.TrustCustom {
		t.Fatalf("expected custom trust, got %s", identity.ProviderTrust)
	}
	if identity.Reviewed {
		t.Fatal("integrity-verified custom binary was silently upgraded to reviewed")
	}
	if len(identity.Limitations) == 0 {
		t.Fatal("expected trust limitation")
	}
}

func TestBinaryIdentityKeepsLegacyRecordLegacy(t *testing.T) {
	identity, err := BinaryIdentity(Record{
		ID:         "legacy-1",
		Provider:   fingerprint.ProviderPatched,
		Version:    "148.0.0",
		Executable: "/managed/chrome",
		SHA256:     strings.Repeat("b", 64),
		SizeBytes:  20,
		Status:     StatusVerified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProviderTrust != fingerprint.TrustLegacy || identity.Reviewed {
		t.Fatalf("unexpected legacy identity: %#v", identity)
	}
}

func TestBinaryIdentityRejectsIncompleteDigest(t *testing.T) {
	_, err := BinaryIdentity(Record{Provider: fingerprint.ProviderCustom, Version: "148.0.0", SizeBytes: 10})
	if err == nil || !strings.Contains(err.Error(), "incomplete binary identity") {
		t.Fatalf("expected incomplete identity error, got %v", err)
	}
}
