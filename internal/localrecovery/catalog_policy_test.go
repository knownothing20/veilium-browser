package localrecovery

import "testing"

func TestCatalogTransitionPolicy(t *testing.T) {
	allowed := [][2]SnapshotStatus{
		{SnapshotPending, SnapshotStaged},
		{SnapshotPending, SnapshotVerified},
		{SnapshotStaged, SnapshotVerified},
		{SnapshotVerified, SnapshotInvalid},
		{SnapshotRecoveryRequired, SnapshotStaged},
	}
	for _, transition := range allowed {
		if !validCatalogTransition(transition[0], transition[1]) {
			t.Fatalf("expected transition %q to %q to be allowed", transition[0], transition[1])
		}
	}
	blocked := [][2]SnapshotStatus{
		{SnapshotInvalid, SnapshotVerified},
		{SnapshotVerified, SnapshotPending},
		{SnapshotStaged, SnapshotPending},
	}
	for _, transition := range blocked {
		if validCatalogTransition(transition[0], transition[1]) {
			t.Fatalf("unexpected transition %q to %q was allowed", transition[0], transition[1])
		}
	}
}
