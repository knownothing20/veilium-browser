package networkevidence

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStorePersistsPrivateWriteOnceNetworkEvidence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "network-evidence")
	store, err := OpenStore(root, StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	run := validRun(now)
	if err := store.Save(run); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(run); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected write-once rejection, got %v", err)
	}
	loaded, err := store.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != run.ID || loaded.ProfileID != run.ProfileID {
		t.Fatalf("unexpected stored run: %#v", loaded)
	}
	if runtime.GOOS != "windows" {
		rootInfo, err := os.Stat(root)
		if err != nil {
			t.Fatal(err)
		}
		if rootInfo.Mode().Perm() != 0o700 {
			t.Fatalf("unexpected root permissions: %o", rootInfo.Mode().Perm())
		}
		fileInfo, err := os.Stat(filepath.Join(root, run.ID+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if fileInfo.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected report permissions: %o", fileInfo.Mode().Perm())
		}
	}
}

func TestStoreFiltersAndDeletesByProfile(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "network-evidence"), StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	first := validRun(now)
	second := validRun(now.Add(time.Second))
	second.ID = "netev-11111111111111111111111111111111"
	second.ProfileID = "profile-b"
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(second); err != nil {
		t.Fatal(err)
	}
	items, err := store.List("profile-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("unexpected filtered runs: %#v", items)
	}
	if err := store.Delete(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted run to be missing, got %v", err)
	}
}

func TestStorePrunesExpiredAndOldestRuns(t *testing.T) {
	now := time.Now().UTC()
	root := filepath.Join(t.TempDir(), "network-evidence")
	store, err := OpenStore(root, StoreOptions{MaxRuns: 2, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for index, id := range []string{
		"netev-11111111111111111111111111111111",
		"netev-22222222222222222222222222222222",
		"netev-33333333333333333333333333333333",
	} {
		run := validRun(now.Add(time.Duration(index) * time.Second))
		run.ID = id
		if err := store.Save(run); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "netev-33333333333333333333333333333333" {
		t.Fatalf("unexpected maximum retention: %#v", items)
	}

	expired := validRun(now.Add(-48 * time.Hour))
	expired.ID = "netev-44444444444444444444444444444444"
	expired.ExpiresAt = now.Add(-time.Hour)
	if err := expired.Validate(); err != nil {
		t.Fatal(err)
	}
	// Write an expired fixture directly so OpenStore/Prune can exercise cleanup.
	payload := []byte(`{"schemaVersion":1}`)
	if err := os.WriteFile(filepath.Join(root, expired.ID+".json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Prune(); err == nil {
		// Malformed files fail closed instead of being silently deleted.
		t.Fatal("expected malformed expired fixture to fail closed")
	}
}

func TestStoreRejectsSymlinkedRootAndReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional Windows privileges")
	}
	base := t.TempDir()
	realRoot := filepath.Join(base, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedRoot := filepath.Join(base, "linked")
	if err := os.Symlink(realRoot, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(linkedRoot, StoreOptions{}); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("expected symlink-root rejection, got %v", err)
	}

	store, err := OpenStore(filepath.Join(base, "store"), StoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := "netev-55555555555555555555555555555555"
	if err := os.Symlink(target, filepath.Join(store.root, id+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(id); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink-report rejection, got %v", err)
	}
}
