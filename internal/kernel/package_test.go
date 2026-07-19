package kernel

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestInspectPackageTreeIsDeterministicAndDetectsTamper(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	writePackageFixture(t, first, []string{"chrome-win/resources.pak", "chrome-win/chrome.exe"})
	writePackageFixture(t, second, []string{"chrome-win/chrome.exe", "chrome-win/resources.pak"})

	firstTree, err := InspectPackageTree(first)
	if err != nil {
		t.Fatal(err)
	}
	secondTree, err := InspectPackageTree(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstTree != secondTree || firstTree.FileCount != 2 || firstTree.SizeBytes < 1 {
		t.Fatalf("package identity was not deterministic: %#v %#v", firstTree, secondTree)
	}
	if err := os.WriteFile(filepath.Join(second, "chrome-win", "resources.pak"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	tampered, err := InspectPackageTree(second)
	if err != nil {
		t.Fatal(err)
	}
	if tampered.SHA256 == firstTree.SHA256 {
		t.Fatal("package tree digest did not change after tampering")
	}
}

func TestInspectPackageTreeRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated Windows privileges")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "chrome-win"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "chrome-win", "target")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "chrome-win", "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectPackageTree(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected package symlink rejection, got %v", err)
	}
}

func TestReviewedProviderCannotUseSingleFileImport(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chrome.exe")
	if err := os.WriteFile(source, []byte("not-the-reviewed-package"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(root, "kernels.json"), filepath.Join(root, "managed"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Import(ImportRequest{Name: "Invalid reviewed import", Provider: fingerprint.ProviderOfficial, Version: "152.0.7960.0", SourcePath: source})
	if err == nil || !strings.Contains(err.Error(), "pinned package installer") {
		t.Fatalf("expected reviewed single-file import rejection, got %v", err)
	}
}

func writePackageFixture(t *testing.T, root string, order []string) {
	t.Helper()
	for _, relative := range order {
		filename := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(relative), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
