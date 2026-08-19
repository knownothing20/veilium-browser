package kernel

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenMigratesLegacyKernelProviderRevisionInMemory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "kernels.json")
	records := []Record{{
		ID: "legacy-kernel", Name: "Legacy kernel", Provider: "custom-chromium", Version: "148.0.0",
		Executable: filepath.Join(root, "legacy.exe"), SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SizeBytes: 1, Status: StatusVerified,
	}}
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, filepath.Join(root, "managed"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get("legacy-kernel")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ProviderRevision != 1 {
		t.Fatalf("legacy Provider revision was not migrated: %#v", loaded)
	}
}

func TestImportVerifyTamperAndDelete(t *testing.T) {
	root := t.TempDir()
	source := testKernelExecutable(t)
	store, err := Open(filepath.Join(root, "kernels.json"), filepath.Join(root, "managed"))
	if err != nil {
		t.Fatal(err)
	}

	record, err := store.Import(ImportRequest{Name: "Test Chromium", Provider: "patched-chromium", Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != StatusVerified || record.SHA256 == "" || record.Executable == source {
		t.Fatalf("unexpected record: %#v", record)
	}
	if len(store.List()) != 1 {
		t.Fatal("expected one registered kernel")
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
	if _, err := os.Stat(filepath.Dir(record.Executable)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("managed directory still exists: %v", err)
	}
}

func TestDuplicateImportIsIdempotent(t *testing.T) {
	root := t.TempDir()
	source := testKernelExecutable(t)
	store, _ := Open(filepath.Join(root, "kernels.json"), filepath.Join(root, "managed"))
	request := ImportRequest{Name: "One", Provider: "native-chromium", Version: "148.0.0", SourcePath: source}
	first, err := store.Import(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Import(ImportRequest{Name: "Two", Provider: request.Provider, Version: request.Version, SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || len(store.List()) != 1 {
		t.Fatalf("duplicate import created another record: %#v %#v", first, second)
	}
}

func TestRejectsSymlinkSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated permissions on Windows")
	}
	root := t.TempDir()
	target := testKernelExecutable(t)
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	store, _ := Open(filepath.Join(root, "kernels.json"), filepath.Join(root, "managed"))
	if _, err := store.Import(ImportRequest{Name: "Link", Provider: "native-chromium", Version: "148.0.0", SourcePath: link}); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
