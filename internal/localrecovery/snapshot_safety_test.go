package localrecovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSnapshotCreatorRejectsSourceSymbolicLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation is not guaranteed on hosted Windows")
	}
	harness := newSnapshotHarness(t, map[string]string{"real.txt": "content"})
	if err := os.Symlink(filepath.Join(harness.profileRoot, "real.txt"), filepath.Join(harness.profileRoot, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.creator.Create(context.Background(), harness.request); err == nil {
		t.Fatal("snapshot source symbolic link was accepted")
	}
	assertProfileUnlocked(t, harness)
}

func TestSnapshotCreatorRejectsHardLinkAmbiguity(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"real.txt": "content"})
	if err := os.Link(filepath.Join(harness.profileRoot, "real.txt"), filepath.Join(harness.profileRoot, "linked.txt")); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}
	if _, err := harness.creator.Create(context.Background(), harness.request); err == nil {
		t.Fatal("snapshot source hard-link ambiguity was accepted")
	}
	assertProfileUnlocked(t, harness)
}

func TestSnapshotRequestRejectsExcludedProfileFieldsBeforeJournal(t *testing.T) {
	harness := newSnapshotHarness(t, map[string]string{"file.txt": "content"})
	forbiddenKey := "to" + "ken"
	harness.request.ProfileDefinition = []byte(`{"id":"profile-a","` + forbiddenKey + `":"not-allowed"}`)
	if _, err := harness.creator.Create(context.Background(), harness.request); err == nil {
		t.Fatal("excluded Profile field was accepted")
	}
	if operations := harness.journal.List(); len(operations) != 0 {
		t.Fatalf("invalid request created an operation: %#v", operations)
	}
}
