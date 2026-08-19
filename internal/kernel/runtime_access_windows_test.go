//go:build windows

package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareManagedPackageRuntimeAccess(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "chrome.exe")
	if err := os.WriteFile(executable, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := prepareManagedPackageRuntimeAccess(root); err != nil {
		t.Fatalf("prepare package runtime access: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime-download.bin"), []byte("must fail"), 0o600); err == nil {
		t.Fatal("managed package directory remained writable")
	}
	if err := os.WriteFile(executable, []byte("must fail"), 0o600); err == nil {
		t.Fatal("managed package file remained writable")
	}
	if err := releaseManagedPackageRuntimeAccess(root); err != nil {
		t.Fatalf("release package runtime access: %v", err)
	}
	if err := os.WriteFile(executable, []byte("unlocked"), 0o600); err != nil {
		t.Fatalf("managed package did not become writable for uninstall: %v", err)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatalf("managed package file could not be deleted during uninstall: %v", err)
	}
}

func TestPrepareManagedPackageRuntimeAccessRejectsUnsafeRoots(t *testing.T) {
	if err := prepareManagedPackageRuntimeAccess(""); err == nil {
		t.Fatal("expected an empty package root to be rejected")
	}
	if err := prepareManagedPackageRuntimeAccess("relative-package"); err == nil {
		t.Fatal("expected a relative package root to be rejected")
	}
	file := filepath.Join(t.TempDir(), "package.bin")
	if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := prepareManagedPackageRuntimeAccess(file); err == nil {
		t.Fatal("expected a file package root to be rejected")
	}
}

func TestRealReviewedChromiumRuntimeAccessAndProbe(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("VEILIUM_REAL_REVIEWED_CHROMIUM_PACKAGE"))
	if root == "" {
		t.Skip("VEILIUM_REAL_REVIEWED_CHROMIUM_PACKAGE is not configured")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := prepareManagedPackageRuntimeAccess(root); err != nil {
		t.Fatalf("prepare real reviewed Chromium runtime access: %v", err)
	}
	if err := probeManagedExecutable(filepath.Join(root, "chrome.exe")); err != nil {
		t.Fatalf("probe real reviewed Chromium: %v", err)
	}
}
