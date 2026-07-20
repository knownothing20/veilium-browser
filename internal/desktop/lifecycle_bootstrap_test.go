package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestServiceCreatesMissingLifecycleDataRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "new", "veilium-data")
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	if service == nil {
		t.Fatal("service was not created")
	}
	info, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("lifecycle data root is unsafe: %v", info.Mode())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("new lifecycle data root is not private: %o", info.Mode().Perm())
	}
}

func TestServiceRejectsSymlinkLifecycleDataRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows reparse handling is covered by lifecycle inventory tests")
	}
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "linked-root")
	if err := os.Symlink(target, root); err != nil {
		t.Fatal(err)
	}
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newService(store, root, newFakeRuntime()); err == nil {
		t.Fatal("expected symlink lifecycle data root rejection")
	}
}
