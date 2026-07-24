package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

func TestSafeBulkPortableFilenameIsDeterministicAndDistinct(t *testing.T) {
	first := safeBulkPortableFilename(`CON:<QA>/Profile`, "profile-a")
	second := safeBulkPortableFilename(`CON:<QA>/Profile`, "profile-b")
	if first == second {
		t.Fatal("different Profile IDs produced the same export filename")
	}
	if strings.ContainsAny(first, `<>:"/\|?*`) {
		t.Fatalf("unsafe export filename %q", first)
	}
	if !strings.HasSuffix(first, ".veilium-profile.json") {
		t.Fatalf("export filename %q does not use the portable suffix", first)
	}
	if got := safeBulkPortableFilename(`CON:<QA>/Profile`, "profile-a"); got != first {
		t.Fatalf("filename is not deterministic: %q != %q", got, first)
	}
}

func TestInspectBulkExportDirectoryRequiresExistingRegularDirectory(t *testing.T) {
	root := t.TempDir()
	got, err := inspectBulkExportDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("directory = %q, want %q", got, want)
	}

	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectBulkExportDirectory(file); err == nil {
		t.Fatal("regular file was accepted as a bulk export directory")
	}
}

func TestEnsureBulkPortableTargetAvailableRejectsExistingTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.veilium-profile.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureBulkPortableTargetAvailable(path); err == nil {
		t.Fatal("existing export target was accepted")
	}
}

func TestNormalizeBulkPortableIdentityMode(t *testing.T) {
	mode, err := normalizeBulkPortableIdentityMode("")
	if err != nil || mode != portableprofile.IdentityNew {
		t.Fatalf("default mode = %q, err = %v", mode, err)
	}
	if _, err := normalizeBulkPortableIdentityMode("unknown"); err == nil {
		t.Fatal("unsupported identity mode was accepted")
	}
}
