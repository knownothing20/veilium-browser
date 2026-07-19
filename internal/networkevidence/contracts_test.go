package networkevidence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestProbeSetRequiresExplicitSelfHostableSecureDefinitions(t *testing.T) {
	set := validProbeSet()
	if err := set.Validate(); err != nil {
		t.Fatalf("valid probe set rejected: %v", err)
	}

	thirdPartyHTTP := set
	thirdPartyHTTP.Definitions = append([]ProbeDefinition(nil), set.Definitions...)
	thirdPartyHTTP.Definitions[0].HTTPSURL = "http://probe.example.invalid/ip"
	if err := thirdPartyHTTP.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected third-party HTTP rejection, got %v", err)
	}

	notSelfHostable := set
	notSelfHostable.Definitions = append([]ProbeDefinition(nil), set.Definitions...)
	notSelfHostable.Definitions[0].SelfHostable = false
	if err := notSelfHostable.Validate(); err == nil || !strings.Contains(err.Error(), "self-hostable") {
		t.Fatalf("expected self-hostable requirement, got %v", err)
	}
}

func TestRouteIdentityClassifiesWithoutSerializingRawRoute(t *testing.T) {
	profile := domain.Profile{
		Proxy: domain.ProxyConfig{
			URL:           "vless://server.example.invalid:443/path-with-private-id",
			CredentialRef: "vault-record-a",
			AdapterRef:    "adapter-a",
		},
	}
	identity, err := RouteForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Kind != RouteXray || identity.Scheme != "vless" || !validSHA256(identity.Digest) {
		t.Fatalf("unexpected route identity: %#v", identity)
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"server.example.invalid", "private-id", "vault-record-a", "adapter-a"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("route identity exposed %q: %s", forbidden, text)
		}
	}

	changed := profile
	changed.Proxy.CredentialRef = "vault-record-b"
	other, err := RouteForProfile(changed)
	if err != nil {
		t.Fatal(err)
	}
	if other.Digest == identity.Digest {
		t.Fatal("different selected route references produced the same digest")
	}
}

func TestNetworkEvidenceAllowsOnlyProbeSpecificValues(t *testing.T) {
	now := time.Now().UTC()
	run := validRun(now)
	if err := run.Validate(); err != nil {
		t.Fatalf("valid network evidence rejected: %v", err)
	}

	invalid := run
	invalid.Observations = append([]Observation(nil), run.Observations...)
	invalid.Observations[1].Values = []string{"raw-candidate:arbitrary-page-content"}
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "WebRTC") {
		t.Fatalf("expected allowlist rejection, got %v", err)
	}

	invalidIP := run
	invalidIP.Observations = append([]Observation(nil), run.Observations...)
	invalidIP.Observations[0].Values = []string{"not-an-ip"}
	if err := invalidIP.Validate(); err == nil || !strings.Contains(err.Error(), "normalized IP") {
		t.Fatalf("expected normalized IP rejection, got %v", err)
	}
}

func TestCompatibilityCannotPromoteCustomProviderToVerified(t *testing.T) {
	now := time.Now().UTC()
	entry := validCompatibilityEntry(now)
	entry.ProviderTrust = fingerprint.TrustCustom
	if err := entry.Validate(); err == nil || !strings.Contains(err.Error(), "reviewed Provider") {
		t.Fatalf("expected custom trust rejection, got %v", err)
	}
}

func TestGenerateMatrixMarksExpiredAcceptedEvidenceStale(t *testing.T) {
	now := time.Now().UTC()
	entry := validCompatibilityEntry(now.Add(-2 * time.Hour))
	expires := now.Add(-time.Hour)
	entry.EvidenceExpiresAt = &expires
	matrix, err := GenerateMatrix(now, []CompatibilityEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Entries) != 1 || matrix.Entries[0].Status != CompatibilityStale {
		t.Fatalf("expected stale compatibility, got %#v", matrix.Entries)
	}
	if !containsValue(matrix.Entries[0].Limitations, "accepted network evidence expired") {
		t.Fatalf("missing expiration limitation: %#v", matrix.Entries[0].Limitations)
	}
}

func TestGenerateMatrixRejectsDuplicateExactCombination(t *testing.T) {
	now := time.Now().UTC()
	entry := validCompatibilityEntry(now)
	if _, err := GenerateMatrix(now, []CompatibilityEntry{entry, entry}); err == nil || !strings.Contains(err.Error(), "duplicate exact") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func validProbeSet() ProbeSet {
	return ProbeSet{
		SchemaVersion: ProbeSchemaVersion,
		ID:            "local-test-probes",
		Revision:      1,
		Definitions: []ProbeDefinition{
			{
				SchemaVersion: ProbeSchemaVersion, ID: "exit", Revision: 1, Kind: ProbeExitIP,
				HTTPSURL: "https://probe.example.invalid/ip", TimeoutSeconds: 10, MaxResponseBytes: 4096,
				SelfHostable: true, PrivacyNote: "Returns only the request public IP for this test.",
			},
			{
				SchemaVersion: ProbeSchemaVersion, ID: "stun", Revision: 1, Kind: ProbeWebRTCSTUN,
				STUNServer: "stun:stun.example.invalid:3478", TimeoutSeconds: 10,
				SelfHostable: true, PrivacyNote: "Receives only the bounded STUN exchange for this run.",
			},
			{
				SchemaVersion: ProbeSchemaVersion, ID: "dns", Revision: 1, Kind: ProbeDelegatedDNS,
				DNSZone: "probe.example.invalid", TimeoutSeconds: 10,
				SelfHostable: true, PrivacyNote: "Records only the one-time delegated query result.",
			},
		},
	}
}

func validRun(now time.Time) Run {
	completed := now.Add(time.Second)
	return Run{
		SchemaVersion:          SchemaVersion,
		ID:                     "netev-0123456789abcdef0123456789abcdef",
		EvidenceRunID:          "evidence-a",
		ProfileID:              "profile-a",
		ProviderID:             "reviewed-a",
		ProviderRevision:       1,
		BrowserVersion:         "148.0.0",
		OperatingSystem:        "windows",
		Architecture:           "amd64",
		BinaryIdentityDigest:   strings.Repeat("a", 64),
		ConsistencyInputDigest: strings.Repeat("b", 64),
		Route:                  RouteIdentity{Kind: RouteDirect, Scheme: "direct", Digest: strings.Repeat("c", 64)},
		ProbeSetID:             "local-test-probes", ProbeSetRevision: 1,
		Status: RunPassed, StartedAt: now, CompletedAt: &completed, ExpiresAt: now.Add(24 * time.Hour),
		Observations: []Observation{
			{
				ID: "exit-ip", ProbeKind: ProbeExitIP, ProbeID: "exit", ProbeRevision: 1,
				Status: ObservationPassed, Values: []string{"203.0.113.8"}, CollectedAt: completed,
			},
			{
				ID: "webrtc", ProbeKind: ProbeWebRTCSTUN, ProbeID: "stun", ProbeRevision: 1,
				Status: ObservationPassed, Values: []string{"candidate:srflx", "protocol:udp", "public-ip:203.0.113.8"}, CollectedAt: completed,
			},
			{
				ID: "dns", ProbeKind: ProbeDelegatedDNS, ProbeID: "dns", ProbeRevision: 1,
				Status: ObservationPassed, Values: []string{"seen:true", "resolver-ip:192.0.2.53", "rcode:NOERROR"}, CollectedAt: completed,
			},
		},
	}
}

func validCompatibilityEntry(reviewedAt time.Time) CompatibilityEntry {
	expires := reviewedAt.Add(24 * time.Hour)
	return CompatibilityEntry{
		SchemaVersion: MatrixSchemaVersion,
		ProviderID:    "reviewed-a", ProviderRevision: 1, ProviderTrust: fingerprint.TrustReviewed,
		BrowserVersion: "148.0.0", OperatingSystem: "windows", Architecture: "amd64",
		BinaryIdentityDigest: strings.Repeat("d", 64), CapabilityID: "network.route",
		Status: CompatibilityVerified, ProbeSetID: "local-test-probes", ProbeSetRevision: 1,
		NetworkEvidenceIDs: []string{"netev-a"}, ReviewedAt: &reviewedAt, EvidenceExpiresAt: &expires,
	}
}

func containsValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
