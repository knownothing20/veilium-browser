package localrecovery

func validCatalogTransition(from, to SnapshotStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case SnapshotPending:
		return to == SnapshotStaged || to == SnapshotVerified || to == SnapshotInvalid || to == SnapshotRecoveryRequired
	case SnapshotStaged:
		return to == SnapshotVerified || to == SnapshotInvalid || to == SnapshotRecoveryRequired
	case SnapshotVerified:
		return to == SnapshotInvalid || to == SnapshotRecoveryRequired
	case SnapshotRecoveryRequired:
		return to == SnapshotStaged || to == SnapshotInvalid
	default:
		return false
	}
}
