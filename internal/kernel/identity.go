package kernel

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
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
		SchemaVersion:    BinaryIdentitySchemaVersion,
		ProviderID:       capabilities.Provider,
		ProviderRevision: capabilities.Revision,
		ProviderTrust:    capabilities.TrustStatus,
		BrowserVersion:   record.Version,
		OperatingSystem:  runtime.GOOS,
		Architecture:     runtime.GOARCH,
		ExecutablePath:   record.Executable,
		ExecutableSize:   record.SizeBytes,
		ExecutableSHA256: record.SHA256,
		IntegrityStatus:  record.Status,
		Provenance:       "managed-local-import",
		Reviewed:         capabilities.IsReviewed() && record.Status == StatusVerified,
		Limitations:      append([]string(nil), capabilities.Limitations...),
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
