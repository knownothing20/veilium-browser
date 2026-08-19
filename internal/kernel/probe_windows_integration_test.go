//go:build windows

package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealPackagedChromiumSingleFileImportFailsClosed(t *testing.T) {
	source := strings.TrimSpace(os.Getenv("VEILIUM_REAL_PACKAGED_CHROMIUM_EXE"))
	if source == "" {
		t.Skip("VEILIUM_REAL_PACKAGED_CHROMIUM_EXE is not configured")
	}
	root := t.TempDir()
	store, err := Open(filepath.Join(root, "kernels.json"), filepath.Join(root, "managed"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Import(ImportRequest{
		Name: "Real packaged Chromium single-file check", Provider: "custom-chromium", Version: "148.0.0", SourcePath: source,
	})
	if err == nil || !strings.Contains(err.Error(), "not launchable") {
		t.Fatalf("expected fail-closed single-file import, got %v", err)
	}
}
