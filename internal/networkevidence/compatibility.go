package networkevidence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

type CompatibilityStatus string

const (
	CompatibilityVerified    CompatibilityStatus = "verified"
	CompatibilityPartial     CompatibilityStatus = "partial"
	CompatibilityUnsupported CompatibilityStatus = "unsupported"
	CompatibilityFailed      CompatibilityStatus = "failed"
	CompatibilityStale       CompatibilityStatus = "stale"
	CompatibilityUntested    CompatibilityStatus = "untested"
)

type CompatibilityEntry struct {
	SchemaVersion        int                     `json:"schemaVersion"`
	ProviderID           string                  `json:"providerId"`
	ProviderRevision     int                     `json:"providerRevision"`
	ProviderTrust        fingerprint.TrustStatus `json:"providerTrust"`
	BrowserVersion       string                  `json:"browserVersion"`
	OperatingSystem      string                  `json:"operatingSystem"`
	Architecture         string                  `json:"architecture"`
	BinaryIdentityDigest string                  `json:"binaryIdentityDigest"`
	CapabilityID         string                  `json:"capabilityId"`
	Status               CompatibilityStatus     `json:"status"`
	ProbeSetID           string                  `json:"probeSetId,omitempty"`
	ProbeSetRevision     int                     `json:"probeSetRevision,omitempty"`
	NetworkEvidenceIDs   []string                `json:"networkEvidenceIds,omitempty"`
	ReviewedAt           *time.Time              `json:"reviewedAt,omitempty"`
	EvidenceExpiresAt    *time.Time              `json:"evidenceExpiresAt,omitempty"`
	Limitations          []string                `json:"limitations,omitempty"`
}

type CompatibilityMatrix struct {
	SchemaVersion int                  `json:"schemaVersion"`
	GeneratedAt   time.Time            `json:"generatedAt"`
	Entries       []CompatibilityEntry `json:"entries"`
}

func (entry CompatibilityEntry) Validate() error {
	if entry.SchemaVersion != MatrixSchemaVersion {
		return fmt.Errorf("unsupported compatibility-entry schema %d", entry.SchemaVersion)
	}
	for label, value := range map[string]string{
		"provider id": entry.ProviderID, "browser version": entry.BrowserVersion,
		"operating system": entry.OperatingSystem, "architecture": entry.Architecture,
		"capability id": entry.CapabilityID,
	} {
		if strings.TrimSpace(value) == "" || len(value) > 256 {
			return fmt.Errorf("compatibility %s is invalid", label)
		}
	}
	if entry.ProviderRevision < 1 || !validCompatibilityStatus(entry.Status) {
		return fmt.Errorf("compatibility revision or status is invalid")
	}
	if !validSHA256(entry.BinaryIdentityDigest) {
		return fmt.Errorf("compatibility binary identity digest is invalid")
	}
	if len(entry.NetworkEvidenceIDs) > 32 || len(entry.Limitations) > 64 {
		return fmt.Errorf("compatibility entry exceeds bounded lists")
	}
	for _, id := range entry.NetworkEvidenceIDs {
		if strings.TrimSpace(id) == "" || len(id) > 256 {
			return fmt.Errorf("compatibility evidence reference is invalid")
		}
	}
	for _, limitation := range entry.Limitations {
		if len(strings.TrimSpace(limitation)) > 512 {
			return fmt.Errorf("compatibility limitation is too long")
		}
	}

	if entry.Status == CompatibilityVerified || entry.Status == CompatibilityPartial {
		if entry.ProviderTrust != fingerprint.TrustReviewed {
			return fmt.Errorf("verified or partial compatibility requires reviewed Provider trust")
		}
		if len(entry.NetworkEvidenceIDs) == 0 || entry.ReviewedAt == nil || entry.ReviewedAt.IsZero() {
			return fmt.Errorf("verified or partial compatibility requires accepted evidence and review time")
		}
		if strings.TrimSpace(entry.ProbeSetID) == "" || entry.ProbeSetRevision < 1 {
			return fmt.Errorf("network compatibility requires an exact probe-set identity")
		}
	}
	if entry.Status == CompatibilityUntested && len(entry.NetworkEvidenceIDs) != 0 {
		return fmt.Errorf("untested compatibility cannot reference accepted evidence")
	}
	if entry.EvidenceExpiresAt != nil && entry.ReviewedAt != nil && !entry.EvidenceExpiresAt.After(*entry.ReviewedAt) {
		return fmt.Errorf("compatibility evidence expiration is invalid")
	}
	return nil
}

func GenerateMatrix(now time.Time, entries []CompatibilityEntry) (CompatibilityMatrix, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	result := CompatibilityMatrix{
		SchemaVersion: MatrixSchemaVersion,
		GeneratedAt:   now,
		Entries:       make([]CompatibilityEntry, 0, len(entries)),
	}
	seen := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		entry.NetworkEvidenceIDs = sortedUnique(entry.NetworkEvidenceIDs)
		entry.Limitations = sortedUnique(entry.Limitations)
		if entry.EvidenceExpiresAt != nil && !entry.EvidenceExpiresAt.After(now) && (entry.Status == CompatibilityVerified || entry.Status == CompatibilityPartial) {
			entry.Status = CompatibilityStale
			entry.Limitations = sortedUnique(append(entry.Limitations, "accepted network evidence expired"))
		}
		if err := entry.Validate(); err != nil {
			return CompatibilityMatrix{}, fmt.Errorf("compatibility entry %d: %w", index, err)
		}
		key := compatibilityKey(entry)
		if _, exists := seen[key]; exists {
			return CompatibilityMatrix{}, fmt.Errorf("duplicate exact compatibility combination %q", key)
		}
		seen[key] = struct{}{}
		result.Entries = append(result.Entries, entry)
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		return compatibilityKey(result.Entries[i]) < compatibilityKey(result.Entries[j])
	})
	return result, result.Validate()
}

func (matrix CompatibilityMatrix) Validate() error {
	if matrix.SchemaVersion != MatrixSchemaVersion || matrix.GeneratedAt.IsZero() {
		return fmt.Errorf("compatibility matrix metadata is invalid")
	}
	if len(matrix.Entries) > 10000 {
		return fmt.Errorf("compatibility matrix is too large")
	}
	for index, entry := range matrix.Entries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("compatibility matrix entry %d: %w", index, err)
		}
	}
	return nil
}

func compatibilityKey(entry CompatibilityEntry) string {
	return strings.Join([]string{
		entry.ProviderID,
		fmt.Sprintf("%d", entry.ProviderRevision),
		entry.BrowserVersion,
		strings.ToLower(entry.OperatingSystem),
		strings.ToLower(entry.Architecture),
		entry.BinaryIdentityDigest,
		entry.CapabilityID,
	}, "|")
}

func validCompatibilityStatus(status CompatibilityStatus) bool {
	switch status {
	case CompatibilityVerified, CompatibilityPartial, CompatibilityUnsupported, CompatibilityFailed, CompatibilityStale, CompatibilityUntested:
		return true
	default:
		return false
	}
}
