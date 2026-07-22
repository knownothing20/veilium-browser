package localrecovery

import (
	"strings"
	"testing"
	"time"
)

func TestValidManifest(t *testing.T) {
	manifest := validManifest(t, "linux")
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	digest, err := ComputeManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) != 64 {
		t.Fatalf("unexpected manifest digest %q", digest)
	}
}

func TestManifestRejectsUnsafeAndCollidingPaths(t *testing.T) {
	for _, value := range []string{"../escape", "/absolute", "folder\\file", "folder//file", "folder/./file", "folder/file.", "folder/file:stream"} {
		if err := ValidateRelativePath(value, "linux"); err == nil {
			t.Fatalf("unsafe path %q was accepted", value)
		}
	}
	if err := ValidateRelativePath("CON/data", "windows"); err == nil {
		t.Fatal("reserved Windows path was accepted")
	}

	manifest := validManifest(t, "windows")
	manifest.Files = []FileEntry{
		{Path: "Cache/A", Size: 1, SHA256: strings.Repeat("a", 64)},
		{Path: "cache/a", Size: 1, SHA256: strings.Repeat("b", 64)},
	}
	manifest.FileCount = 2
	manifest.TotalBytes = 2
	digest, err := ComputeTreeDigest("windows", manifest.Files)
	if err == nil {
		t.Fatalf("Windows path collision was accepted with digest %q", digest)
	}
	manifest.TreeDigest = strings.Repeat("0", 64)
	if err := manifest.Validate(); err == nil {
		t.Fatal("Windows path collision manifest was accepted")
	}
}

func TestManifestRejectsContradictorySummaryAndScope(t *testing.T) {
	manifest := validManifest(t, "linux")
	manifest.TotalBytes++
	if err := manifest.Validate(); err == nil {
		t.Fatal("contradictory byte summary was accepted")
	}
	manifest = validManifest(t, "linux")
	manifest.Scope = "portable-definition"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unsupported artifact scope was accepted")
	}
	manifest = validManifest(t, "linux")
	manifest.ParentSnapshotID = "parent"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unapproved incremental snapshot was accepted")
	}
}

func TestManifestRequiresReviewedDependencyIdentity(t *testing.T) {
	manifest := validManifest(t, "linux")
	manifest.Dependencies.Kernel.TrustRequirement = "reviewed"
	manifest.Dependencies.Kernel.ExecutableSHA256 = ""
	if err := manifest.Validate(); err == nil {
		t.Fatal("reviewed dependency without exact identity was accepted")
	}
}

func TestDigestProfileDefinitionIsCanonical(t *testing.T) {
	left, err := DigestProfileDefinition([]byte(`{"name":"Profile","count":1}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := DigestProfileDefinition([]byte("{\n  \"count\": 1,\n  \"name\": \"Profile\"\n}"))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("canonical Profile digests differ: %s != %s", left, right)
	}
	if _, err := DigestProfileDefinition([]byte(`[]`)); err == nil {
		t.Fatal("non-object Profile definition was accepted")
	}
}

func validManifest(t *testing.T, sourceOS string) LocalSnapshotManifest {
	t.Helper()
	entries := []FileEntry{
		{Path: "Default/Cookies", Size: 12, SHA256: strings.Repeat("a", 64)},
		{Path: "Local State", Size: 8, SHA256: strings.Repeat("b", 64)},
	}
	tree, err := ComputeTreeDigest(sourceOS, entries)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := DigestProfileDefinition([]byte(`{"id":"profile-a","name":"Profile A"}`))
	if err != nil {
		t.Fatal(err)
	}
	return LocalSnapshotManifest{
		SchemaVersion:              ManifestSchemaVersion,
		SnapshotID:                 "snapshot-a",
		Scope:                      ScopeLocalFullSnapshot,
		SourceProfileID:            "profile-a",
		SourceProfileName:          "Profile A",
		SourceProfileSchemaVersion: 1,
		SourceApplicationVersion:   "0.15.0-dev",
		SourceOS:                   sourceOS,
		SourceArch:                 "amd64",
		CreatedAt:                  time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		ProfileDefinitionDigest:    profileDigest,
		IncludedRoots:              []string{"browser-data", "profile-definition"},
		TreeDigest:                 tree,
		FileCount:                  int64(len(entries)),
		TotalBytes:                 20,
		Files:                      entries,
		Dependencies: DependencyRequirements{Kernel: KernelRequirement{
			ProviderID:       "custom-chromium",
			ProviderRevision: 1,
			BrowserVersion:   "148.0.0",
			OperatingSystem:  sourceOS,
			Architecture:     "amd64",
			TrustRequirement: "custom",
		}},
		ExcludedData: []string{"adapter-binaries", "browser-evidence", "credential-secrets", "kernel-binaries", "runtime-logs", "runtime-state"},
		Portability:  PortabilitySameUserSameMachine,
	}
}
