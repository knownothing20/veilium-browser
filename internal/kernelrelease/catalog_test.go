package kernelrelease

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/second-provider.json
var secondProviderFixture []byte

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
	if len(release.AllowedDownloadHosts) != 2 || release.AllowedDownloadHosts[0] != "commondatastorage.googleapis.com" {
		t.Fatalf("unexpected reviewed download-host policy: %#v", release.AllowedDownloadHosts)
	}
	if release.ExecutablePath != "chrome-win/chrome.exe" || release.ExecutableSizeBytes != 2926080 {
		t.Fatalf("unexpected reviewed executable pin: %#v", release)
	}
	if release.PackageFileCount != 261 || release.ExpandedSizeBytes != 814120936 || release.PackageTreeSHA256 != "312cb62d6bfab56ecfa52c4e8047dd33c05a1c17c7e44bc2afd9be436854a8dc" {
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
	first.Releases[0].AllowedDownloadHosts[0] = "changed.invalid"
	second, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	if second.Releases[0].Limitations[0] == "changed" {
		t.Fatal("embedded release catalog was mutated")
	}
	if second.Releases[0].AllowedDownloadHosts[0] == "changed.invalid" {
		t.Fatal("embedded download-host policy was mutated")
	}
}

func TestMultiProviderManifestUsesCompoundIdentityAndIndependentPolicy(t *testing.T) {
	manifest, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(secondProviderFixture))
	decoder.DisallowUnknownFields()
	var second Release
	if err := decoder.Decode(&second); err != nil {
		t.Fatal(err)
	}
	manifest.Releases = append(manifest.Releases, second)
	policies := make(map[string]providerValidationPolicy, len(productionProviderPolicies)+1)
	for id, policy := range productionProviderPolicies {
		policies[id] = policy
	}
	policies[second.ProviderID] = providerValidationPolicy{
		SourceProject: second.SourceProject, LicenseSPDX: second.LicenseSPDX,
		Platform: "windows", Arch: "amd64",
		ArchiveName: second.ArchiveName, ArchiveRoot: second.ArchiveRoot, ExecutablePath: second.ExecutablePath,
		ArchiveHosts: []string{"example.invalid"}, ArchivePath: func(Release) string { return "/downloads/9001/test-win.zip" },
		SourceHost: "example.invalid", SourcePath: "/provider/",
		LicenseHost: "example.invalid", LicensePathSuffix: "/LICENSE", ThirdPartyNotice: "https://example.invalid/notices/NOTICE",
		MinimumArchiveSize: 1, MaximumArchiveSize: 1 << 20,
	}
	if err := validateWithPolicies(manifest, policies); err != nil {
		t.Fatalf("second Provider did not coexist: %v", err)
	}
	found, ok := findExact(manifest.Releases, second.Identity())
	if !ok || found.ProviderRevision != 7 || found.ExecutablePath != "test-win/test-browser.exe" {
		t.Fatalf("compound lookup changed the second Provider: %#v", found)
	}
	duplicate := manifest
	duplicate.Releases = append(append([]Release(nil), manifest.Releases...), second)
	if err := validateWithPolicies(duplicate, policies); err == nil || !strings.Contains(err.Error(), "duplicate compound identity") {
		t.Fatalf("duplicate compound identity was not rejected: %v", err)
	}
	wrongPolicy := manifest
	wrongPolicy.Releases = append([]Release(nil), manifest.Releases...)
	wrongPolicy.Releases[1].SourceProject = "The Chromium Project"
	if err := validateWithPolicies(wrongPolicy, policies); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("cross-Provider provenance was accepted: %v", err)
	}
}
