package kernelrelease

import (
	"strings"
	"testing"
)

func TestCatalogContainsOneExactReviewedSnapshot(t *testing.T) {
	manifest, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Releases) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	release := manifest.Releases[0]
	if release.ProviderID != ProviderID || release.ProviderRevision != 1 || release.BrowserVersion != "152.0.7960.0" || release.SnapshotRevision != 1664436 {
		t.Fatalf("unexpected reviewed release identity: %#v", release)
	}
	if release.Platform != "windows" || release.Arch != "amd64" || release.ArchiveEntryCount != 261 {
		t.Fatalf("unexpected reviewed release platform or layout: %#v", release)
	}
	if !strings.HasSuffix(release.ArchiveURL, "/Win_x64/1664436/chrome-win.zip") || release.ArchiveSizeBytes != 343585547 {
		t.Fatalf("unexpected reviewed archive pin: %#v", release)
	}
	if release.ExecutablePath != "chrome-win/chrome.exe" || release.ExecutableSizeBytes != 2926080 {
		t.Fatalf("unexpected reviewed executable pin: %#v", release)
	}
	if release.PackageFileCount != 261 || release.ExpandedSizeBytes != 814120936 || release.PackageTreeSHA256 != "2c2c4df75cc994b51f24028029057296d3213de4edcada6e698842bd24886e4c" {
		t.Fatalf("unexpected reviewed package-tree pin: %#v", release)
	}
	if release.ThirdPartyNoticesURL != "chrome://credits/" || len(release.Limitations) < 3 {
		t.Fatalf("missing reviewed notices or limitations: %#v", release)
	}
}

func TestFindAndMatchAreExact(t *testing.T) {
	release, ok := Find(ProviderID, "152.0.7960.0", "windows", "amd64")
	if !ok {
		t.Fatal("exact reviewed release was not found")
	}
	if _, ok := Find(ProviderID, "152.0.7960.1", "windows", "amd64"); ok {
		t.Fatal("nearby browser version must not inherit reviewed status")
	}
	if _, ok := Find(ProviderID, release.BrowserVersion, "linux", "amd64"); ok {
		t.Fatal("Linux must not inherit the Windows reviewed release")
	}
	if _, ok := MatchExecutable(ProviderID, release.BrowserVersion, release.ExecutableSHA256, release.ExecutableSizeBytes); !ok {
		t.Fatal("exact executable identity was not matched")
	}
	if _, ok := MatchExecutable(ProviderID, release.BrowserVersion, strings.Repeat("0", 64), release.ExecutableSizeBytes); ok {
		t.Fatal("modified executable digest must not match")
	}
	if _, ok := MatchPackage(ProviderID, release.BrowserVersion, release.ExecutableSHA256, release.ExecutableSizeBytes, release.PackageTreeSHA256, release.PackageFileCount, release.ExpandedSizeBytes); !ok {
		t.Fatal("exact package identity was not matched")
	}
	if _, ok := MatchPackage(ProviderID, release.BrowserVersion, release.ExecutableSHA256, release.ExecutableSizeBytes, strings.Repeat("0", 64), release.PackageFileCount, release.ExpandedSizeBytes); ok {
		t.Fatal("modified package tree must not match")
	}
}

func TestManifestCannotBeMutatedThroughCatalog(t *testing.T) {
	first, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	first.Releases[0].Limitations[0] = "changed"
	second, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	if second.Releases[0].Limitations[0] == "changed" {
		t.Fatal("embedded release catalog was mutated")
	}
}
