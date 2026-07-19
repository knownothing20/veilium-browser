package networkevidence

import (
	"testing"
	"time"
)

func TestReconcileBlocksWebRTCExitMismatch(t *testing.T) {
	now := time.Now().UTC()
	items := ReconcileObservations([]Observation{
		{ID: "exit-ip", ProbeKind: ProbeExitIP, ProbeID: "exit", ProbeRevision: 1, Status: ObservationPassed, Values: []string{"203.0.113.8"}, CollectedAt: now},
		{ID: "webrtc-stun", ProbeKind: ProbeWebRTCSTUN, ProbeID: "stun", ProbeRevision: 1, Status: ObservationPassed, Values: []string{"candidate:srflx", "public-ip:198.51.100.9", "protocol:udp"}, CollectedAt: now},
	})
	if items[1].Status != ObservationFailed || items[1].ReasonCode != "webrtc-exit-ip-mismatch" || items[1].Expected != "203.0.113.8" {
		t.Fatalf("expected WebRTC mismatch failure, got %#v", items[1])
	}
}

func TestReconcileAcceptsMatchingWebRTCAndDegradesUnseenDNS(t *testing.T) {
	now := time.Now().UTC()
	items := ReconcileObservations([]Observation{
		{ID: "exit-ip", ProbeKind: ProbeExitIP, ProbeID: "exit", ProbeRevision: 1, Status: ObservationPassed, Values: []string{"203.0.113.8"}, CollectedAt: now},
		{ID: "webrtc-stun", ProbeKind: ProbeWebRTCSTUN, ProbeID: "stun", ProbeRevision: 1, Status: ObservationPassed, Values: []string{"candidate:srflx", "public-ip:203.0.113.8", "protocol:udp"}, CollectedAt: now},
		{ID: "delegated-dns", ProbeKind: ProbeDelegatedDNS, ProbeID: "dns", ProbeRevision: 1, Status: ObservationPassed, Values: []string{"seen:false", "rcode:NOERROR"}, CollectedAt: now},
	})
	if items[1].Status != ObservationPassed {
		t.Fatalf("expected matching WebRTC pass, got %#v", items[1])
	}
	if items[2].Status != ObservationPartial || items[2].ReasonCode != "dns-query-not-seen" {
		t.Fatalf("expected unseen DNS degradation, got %#v", items[2])
	}
}
