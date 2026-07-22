package localrecovery

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

type snapshotHarness struct {
	dataRoot    string
	profileRoot string
	creator     *SnapshotCreator
	records     *lifecycle.RecordStore
	journal     *lifecycle.Journal
	coordinator *lifecycle.Coordinator
	request     SnapshotRequest
}

func newSnapshotHarness(t *testing.T, files map[string]string) snapshotHarness {
	t.Helper()
	dataRoot := t.TempDir()
	if err := os.Chmod(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	profileRoot := filepath.Join(dataRoot, "profiles", "profile-a")
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for relative, content := range files {
		filePath := filepath.Join(profileRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	records, err := lifecycle.OpenRecordStore(filepath.Join(dataRoot, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := records.Create(lifecycle.Record{
		ProfileID:  "profile-a",
		State:      lifecycle.StateAvailable,
		ManagedDir: "profiles/profile-a",
	}); err != nil {
		t.Fatal(err)
	}
	journal, err := lifecycle.OpenJournal(filepath.Join(dataRoot, "lifecycle-operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := lifecycle.NewCoordinator(records, journal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	creator, err := OpenSnapshotCreator(dataRoot, records, journal, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	creator.space = func(string) (uint64, error) { return math.MaxUint64, nil }
	request := SnapshotRequest{
		OperationID:          "operation-a",
		SnapshotID:           "snapshot-a",
		ProfileID:            "profile-a",
		IdempotencyKey:       "request-a",
		ProfileName:          "Profile A",
		ProfileSchemaVersion: 1,
		ApplicationVersion:   "0.15.0-dev",
		ProfileDefinition:    []byte(`{"id":"profile-a","name":"Profile A"}`),
		Dependencies: DependencyRequirements{Kernel: KernelRequirement{
			ProviderID:       "custom-chromium",
			BrowserVersion:   "148.0.0",
			OperatingSystem:  runtime.GOOS,
			Architecture:     runtime.GOARCH,
			TrustRequirement: "custom",
		}},
		MaxDuration: time.Minute,
	}
	return snapshotHarness{
		dataRoot:    dataRoot,
		profileRoot: profileRoot,
		creator:     creator,
		records:     records,
		journal:     journal,
		coordinator: coordinator,
		request:     request,
	}
}

func assertProfileUnlocked(t *testing.T, harness snapshotHarness) {
	t.Helper()
	record, err := harness.records.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil {
		t.Fatalf("Profile remained locked: %#v", record.Lock)
	}
}
