package networkevidence

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProbeStorePersistsAndDeletesExplicitSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "network-probes.json")
	store, err := OpenProbeStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists, err := store.Get(); err != nil || exists {
		t.Fatalf("expected empty store, exists=%v err=%v", exists, err)
	}
	saved, err := store.Save(validProbeSet())
	if err != nil {
		t.Fatal(err)
	}
	loaded, exists, err := store.Get()
	if err != nil || !exists || loaded.ID != saved.ID || len(loaded.Definitions) != 3 {
		t.Fatalf("unexpected stored ProbeSet: %#v exists=%v err=%v", loaded, exists, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected ProbeSet permissions: %o", info.Mode().Perm())
		}
	}
	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := store.Get(); err != nil || exists {
		t.Fatalf("expected deleted ProbeSet, exists=%v err=%v", exists, err)
	}
}

func TestProbeStoreRejectsSymlinkedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional Windows privileges")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "network-probes.json")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	store, err := OpenProbeStore(path)
	if err == nil || store != nil {
		t.Fatalf("expected symlink rejection, store=%#v err=%v", store, err)
	}
}
