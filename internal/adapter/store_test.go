package adapter

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestImportVerifyTamperAndDelete(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "xray-test")
	if err := os.WriteFile(source, []byte("adapter-binary-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(root, "adapters.json"), filepath.Join(root, "managed"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Import(ImportRequest{
		Name: "Xray local", Kind: KindXray, Version: "25.1.1", SourcePath: source,
		LicenseSPDX: "MPL-2.0", SourceURL: "https://example.test/xray",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != StatusVerified || record.SHA256 == "" || record.Executable == source {
		t.Fatalf("unexpected record: %#v", record)
	}
	if !SupportsScheme(record.Kind, "vless") || SupportsScheme(record.Kind, "tuic") {
		t.Fatalf("unexpected capabilities: %#v", record.Protocols)
	}
	verified, err := store.Verify(record.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("verification failed: %#v %v", verified, err)
	}
	if err := os.WriteFile(record.Executable, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	modified, err := store.Verify(record.ID)
	if err != nil || modified.Status != StatusModified {
		t.Fatalf("expected modified status: %#v %v", modified, err)
	}
	if _, err := store.Delete(record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(record.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing record, got %v", err)
	}
}

func TestDuplicateImportIsIdempotent(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "sing-box-test")
	if err := os.WriteFile(source, []byte("same-adapter"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, _ := Open(filepath.Join(root, "adapters.json"), filepath.Join(root, "managed"))
	request := ImportRequest{Name: "One", Kind: KindSingBox, Version: "1.12.0", SourcePath: source, LicenseSPDX: "GPL-3.0-or-later", SourceURL: "https://example.test/sing-box"}
	first, err := store.Import(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Name = "Two"
	second, err := store.Import(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || len(store.List()) != 1 {
		t.Fatalf("duplicate import created another record: %#v %#v", first, second)
	}
}

func TestRejectsInvalidMetadataAndSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, _ := Open(filepath.Join(root, "adapters.json"), filepath.Join(root, "managed"))
	base := ImportRequest{Name: "Adapter", Kind: KindXray, Version: "1", SourcePath: target, LicenseSPDX: "MPL-2.0", SourceURL: "https://example.test/source"}

	invalid := base
	invalid.Kind = "unknown"
	if _, err := store.Import(invalid); err == nil {
		t.Fatal("expected kind rejection")
	}
	invalid = base
	invalid.LicenseSPDX = "not a license value with spaces"
	if _, err := store.Import(invalid); err == nil {
		t.Fatal("expected license rejection")
	}
	invalid = base
	invalid.SourceURL = "http://example.test/source"
	if _, err := store.Import(invalid); err == nil {
		t.Fatal("expected insecure source URL rejection")
	}

	if runtime.GOOS == "windows" {
		return
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	invalid = base
	invalid.SourcePath = link
	if _, err := store.Import(invalid); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
