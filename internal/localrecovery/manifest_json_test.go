package localrecovery

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestManifestJSONRejectsDependencyPlatformMismatch(t *testing.T) {
	manifest := validManifest(t, "linux")
	manifest.Dependencies.Kernel.OperatingSystem = "windows"
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("dependency platform mismatch was accepted")
	}
	if _, err := json.Marshal(manifest); err == nil {
		t.Fatal("JSON encoding bypassed manifest validation")
	}
}

func TestManifestJSONRoundTripIsStrict(t *testing.T) {
	manifest := validManifest(t, "windows")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var loaded LocalSnapshotManifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.SnapshotID != manifest.SnapshotID || loaded.TreeDigest != manifest.TreeDigest {
		t.Fatalf("manifest changed during JSON round trip: %#v", loaded)
	}
	unknown := strings.Replace(string(data), "{", `{"unexpected":true,`, 1)
	unknown = strings.Replace(unknown, `\"`, `"`, -1)
	if err := json.Unmarshal([]byte(unknown), &loaded); err == nil {
		t.Fatal("manifest decoder accepted an unknown field")
	}
}
