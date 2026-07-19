package kernel

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

const BinaryIdentitySchemaVersion = 1

type ProviderBinaryIdentity struct {
	SchemaVersion         int                     `json:"schemaVersion"`
	ProviderID            string                  `json:"providerId"`
	ProviderRevision      int                     `json:"providerRevision"`
	ProviderTrust         fingerprint.TrustStatus `json:"providerTrust"`
	BrowserVersion        string                  `json:"browserVersion"`
	OperatingSystem       string                  `json:"operatingSystem"`
	Architecture          string                  `json:"architecture"`
	ExecutablePath        string                  `json:"executablePath"`
	ExecutableSize        int64                   `json:"executableSize"`
	ExecutableSHA256      string                  `json:"executableSha256"`
	PackageRoot           string                  `json:"packageRoot,omitempty"`
	PackageTreeSHA256     string                  `json:"packageTreeSha256,omitempty"`
	PackageFileCount      int                     `json:"packageFileCount,omitempty"`
	PackageSizeBytes      int64                   `json:"packageSizeBytes,omitempty"`
	SnapshotRevision      int64                   `json:"snapshotRevision,omitempty"`
	ArchiveSHA256         string                  `json:"archiveSha256,omitempty"`
	IntegrityStatus       string                  `json:"integrityStatus"`
	VerificationTimestamp string                  `json:"verificationTimestamp,omitempty"`
	Provenance            string                  `json:"provenance"`
	Reviewed              bool                    `json:"reviewed"`
	Limitations           []string                `json:"limitations,omitempty"`
}

func BinaryIdentity(record Record) (ProviderBinaryIdentity, error) {
	if strings.TrimSpace(record.Provider) == "" || strings.TrimSpace(record.Version) == "" {
		return ProviderBinaryIdentity{}, fmt.Errorf("kernel record requires provider and version")
	}
	capabilities, err := fingerprint.For(record.Provider, record.Version)
	if err != nil {
		return ProviderBinaryIdentity{}, err
	}
	if strings.TrimSpace(record.SHA256) == "" || record.SizeBytes < 1 {
		return ProviderBinaryIdentity{}, fmt.Errorf("kernel record %q has incomplete binary identity", record.ID)
	}
	identity := ProviderBinaryIdentity{
		SchemaVersion: BinaryIdentitySchemaVersion, ProviderID: capabilities.Provider,
		ProviderRevision: capabilities.Revision, ProviderTrust: capabilities.TrustStatus,
		BrowserVersion: record.Version, OperatingSystem: runtime.GOOS, Architecture: runtime.GOARCH,
		ExecutablePath: record.Executable, ExecutableSize: record.SizeBytes, ExecutableSHA256: record.SHA256,
		PackageRoot: record.PackageRoot, PackageTreeSHA256: record.PackageTreeSHA256,
		PackageFileCount: record.PackageFileCount, PackageSizeBytes: record.PackageSizeBytes,
		SnapshotRevision: record.SnapshotRevision, ArchiveSHA256: record.ArchiveSHA256,
		IntegrityStatus: record.Status, Provenance: "managed-local-import",
		Limitations: append([]string(nil), capabilities.Limitations...),
	}
	if capabilities.IsReviewed() {
		release, ok := kernelrelease.MatchPackage(record.Provider, record.Version, record.SHA256, record.SizeBytes, record.PackageTreeSHA256, record.PackageFileCount, record.PackageSizeBytes)
		if !ok || record.SnapshotRevision != release.SnapshotRevision || record.ArchiveSHA256 != release.ArchiveSHA256 || strings.TrimSpace(record.PackageRoot) == "" {
			return ProviderBinaryIdentity{}, fmt.Errorf("reviewed kernel record %q does not match the embedded package identity", record.ID)
		}
		identity.Provenance = release.ArchiveURL
		identity.Reviewed = record.Status == StatusVerified && runtime.GOOS == release.Platform && runtime.GOARCH == release.Arch
		if runtime.GOOS != release.Platform || runtime.GOARCH != release.Arch {
			identity.Limitations = append(identity.Limitations, "reviewed package trust applies only on "+release.Platform+"/"+release.Arch)
		}
	}
	if !record.VerifiedAt.IsZero() {
		identity.VerificationTimestamp = record.VerifiedAt.UTC().Format(time.RFC3339Nano)
	} else {
		identity.Limitations = append(identity.Limitations, "legacy kernel record has no verification timestamp")
	}
	if !identity.Reviewed {
		identity.Limitations = append(identity.Limitations, "binary integrity does not establish reviewed provider trust")
	}
	return identity, nil
}
