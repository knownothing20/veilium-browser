package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedStorageLocationsReportsFixedManagedPaths(t *testing.T) {
	root := t.TempDir()
	profilesDir := filepath.Join(root, "profiles")
	if err := os.MkdirAll(profilesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "kernels"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lifecycle.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "adapters"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := &Service{dataRoot: root, profilesDir: profilesDir}
	result := service.ManagedStorageLocations()
	if result.DataRoot != filepath.Clean(root) {
		t.Fatalf("data root = %q, want %q", result.DataRoot, filepath.Clean(root))
	}
	if result.GeneratedAt.IsZero() {
		t.Fatal("generated timestamp is required")
	}
	if len(result.Locations) != 13 {
		t.Fatalf("locations = %d, want 13", len(result.Locations))
	}
	if len(result.Limitations) < 3 {
		t.Fatalf("limitations = %#v", result.Limitations)
	}

	locations := make(map[string]ManagedStorageLocation, len(result.Locations))
	for _, item := range result.Locations {
		if _, exists := locations[item.ID]; exists {
			t.Fatalf("duplicate location id %q", item.ID)
		}
		locations[item.ID] = item
		if item.Path == "" || item.Label == "" || item.Description == "" {
			t.Fatalf("incomplete location: %#v", item)
		}
	}
	if got := locations["data-root"].Status; got != managedStoragePresent {
		t.Fatalf("data root status = %q", got)
	}
	if got := locations["profile-data"].Status; got != managedStoragePresent {
		t.Fatalf("profile data status = %q", got)
	}
	if got := locations["kernel-packages"].Status; got != managedStoragePresent {
		t.Fatalf("kernel status = %q", got)
	}
	if got := locations["adapter-packages"].Status; got != managedStorageUnexpected {
		t.Fatalf("adapter status = %q", got)
	}
	if got := locations["lifecycle-records"].Status; got != managedStoragePresent {
		t.Fatalf("lifecycle status = %q", got)
	}
	if got := locations["portable-templates"].Status; got != managedStorageMissing {
		t.Fatalf("template status = %q", got)
	}
}

func TestInspectManagedStorageLocationRejectsLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result := inspectManagedStorageLocation(managedStorageSpec{
		id: "linked", label: "Linked", path: link, kind: "directory", description: "test",
	}, "")
	if result.Status != managedStorageLink || result.ReasonCode != "managed-location-is-link" {
		t.Fatalf("linked result = %#v", result)
	}
}

func TestSameManagedStorageVolumeIsCaseInsensitiveAndNonEmpty(t *testing.T) {
	if !sameManagedStorageVolume("C:", "c:\\") {
		t.Fatal("expected Windows volume comparison to be case-insensitive")
	}
	if sameManagedStorageVolume("", "C:") {
		t.Fatal("empty volume must not match")
	}
	if sameManagedStorageVolume("C:", "D:") {
		t.Fatal("different volumes must not match")
	}
}
