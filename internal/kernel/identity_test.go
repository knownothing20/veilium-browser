package kernel

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

func TestBinaryIdentityDoesNotUpgradeCustomIntegrityToReviewed(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 18, 14, 0, 0, 123, time.UTC)
	identity, err := BinaryIdentity(Record{
		ID: "custom-1", Provider: fingerprint.ProviderCustom, Version: "148.0.0",
		Executable: "/managed/chrome", SHA256: strings.Repeat("a", 64), SizeBytes: 10,
		Status: StatusVerified, VerifiedAt: verifiedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProviderTrust != fingerprint.TrustCustom || identity.Reviewed {
		t.Fatalf("unexpected custom identity: %#v", identity)
	}
	if identity.VerificationTimestamp != verifiedAt.Format(time.RFC3339Nano) || len(identity.Limitations) == 0 {
		t.Fatalf("unexpected custom verification metadata: %#v", identity)
	}
}

func TestBinaryIdentityKeepsLegacyRecordLegacy(t *testing.T) {
	identity, err := BinaryIdentity(Record{
		ID: "legacy-1", Provider: fingerprint.ProviderPatched, Version: "148.0.0",
		Executable: "/managed/chrome", SHA256: strings.Repeat("b", 64), SizeBytes: 20,
		Status: StatusVerified,
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProviderTrust != fingerprint.TrustLegacy || identity.Reviewed {
		t.Fatalf("unexpected legacy identity: %#v", identity)
	}
	if identity.VerificationTimestamp != "" || !containsIdentityLimitation(identity.Limitations, "no verification timestamp") {
		t.Fatalf("expected explicit legacy timestamp limitation: %#v", identity)
	}
}

func TestBinaryIdentityRejectsIncompleteDigest(t *testing.T) {
	_, err := BinaryIdentity(Record{Provider: fingerprint.ProviderCustom, Version: "148.0.0", SizeBytes: 10})
	if err == nil || !strings.Contains(err.Error(), "incomplete binary identity") {
		t.Fatalf("expected incomplete identity error, got %v", err)
	}
}

func TestBinaryIdentityRequiresExactReviewedPackageMetadata(t *testing.T) {
	release, ok := kernelrelease.Find(fingerprint.ProviderOfficial, "152.0.7960.0", "windows", "amd64")
	if !ok {
		t.Fatal("reviewed release fixture missing")
	}
	record := Record{
		ID: "official-1", Provider: release.ProviderID, Version: release.BrowserVersion,
		Executable: "/managed/package/chrome-win/chrome.exe",
		SHA256: release.ExecutableSHA256, SizeBytes: release.ExecutableSizeBytes,
		Status: StatusVerified, VerifiedAt: time.Now().UTC(),
		PackageRoot: "/managed/package", PackageTreeSHA256: release.PackageTreeSHA256,
		PackageFileCount: release.PackageFileCount, PackageSizeBytes: release.ExpandedSizeBytes,
		SnapshotRevision: release.SnapshotRevision, ArchiveSHA256: release.ArchiveSHA256,
	}
	identity, err := BinaryIdentity(record)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProviderTrust != fingerprint.TrustReviewed || identity.PackageTreeSHA256 != release.PackageTreeSHA256 || identity.Provenance != release.ArchiveURL {
		t.Fatalf("unexpected reviewed identity: %#v", identity)
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		if !identity.Reviewed {
			t.Fatal("exact Windows reviewed package was not reviewed")
		}
	} else if identity.Reviewed {
		t.Fatal("Windows-only reviewed package inherited trust on another platform")
	}

	record.PackageTreeSHA256 = strings.Repeat("0", 64)
	if _, err := BinaryIdentity(record); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected reviewed package mismatch rejection, got %v", err)
	}
}

func containsIdentityLimitation(limitations []string, fragment string) bool {
	for _, limitation := range limitations {
		if strings.Contains(limitation, fragment) {
			return true
		}
	}
	return false
}
