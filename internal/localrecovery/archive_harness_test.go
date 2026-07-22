package localrecovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

type archiveBlockers struct {
	active    bool
	dependent bool
}

type archiveHarness struct {
	dataRoot    string
	profileRoot string
	records     *lifecycle.RecordStore
	journal     *lifecycle.Journal
	coordinator *lifecycle.Coordinator
	executor    *ArchiveExecutor
	request     ArchiveRequest
	blockers    *archiveBlockers
}

func newArchiveHarness(t *testing.T, state lifecycle.State, limitations []string) archiveHarness {
	t.Helper()
	dataRoot := t.TempDir()
	if err := os.Chmod(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	profileRoot := filepath.Join(dataRoot, "profiles", "profile-a")
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "sentinel.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	records, err := lifecycle.OpenRecordStore(filepath.Join(dataRoot, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	record := lifecycle.Record{
		ProfileID:       "profile-a",
		State:           state,
		ManagedDir:      "profiles/profile-a",
		LimitationCodes: append([]string(nil), limitations...),
	}
	if _, err := records.Create(record); err != nil {
		t.Fatal(err)
	}
	journal, err := lifecycle.OpenJournal(filepath.Join(dataRoot, "lifecycle-operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	blockers := &archiveBlockers{}
	coordinator, err := lifecycle.NewCoordinator(records, journal, func(string) (string, bool) {
		return "browser-session-active", blockers.active
	}, func(string) (string, bool) {
		return "protected-dependent-operation", blockers.dependent
	})
	if err != nil {
		t.Fatal(err)
	}
	executor, err := OpenArchiveExecutor(dataRoot, records, journal, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	return archiveHarness{
		dataRoot:    dataRoot,
		profileRoot: profileRoot,
		records:     records,
		journal:     journal,
		coordinator: coordinator,
		executor:    executor,
		request: ArchiveRequest{
			OperationID:        "archive-operation-a",
			ProfileID:          "profile-a",
			IdempotencyKey:     "archive-request-a",
			ApplicationVersion: "0.15.0-dev",
		},
		blockers: blockers,
	}
}

func assertArchiveUnlocked(t *testing.T, harness archiveHarness) lifecycle.Record {
	t.Helper()
	record, err := harness.records.Get(harness.request.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Lock != nil {
		t.Fatalf("lifecycle record remained locked: %#v", record.Lock)
	}
	return record
}

type failingArchiveRecordStore struct {
	delegate   archiveRecordStore
	failUpdate bool
}

func (s *failingArchiveRecordStore) Get(id string) (lifecycle.Record, error) {
	return s.delegate.Get(id)
}

func (s *failingArchiveRecordStore) Update(record lifecycle.Record) (lifecycle.Record, error) {
	if s.failUpdate {
		return lifecycle.Record{}, errors.New("simulated lifecycle persistence failure")
	}
	return s.delegate.Update(record)
}

func (s *failingArchiveRecordStore) AddRecoveryCode(profileID, code string) (lifecycle.Record, bool, error) {
	return s.delegate.AddRecoveryCode(profileID, code)
}

type cancellingArchiveJournal struct {
	delegate  *lifecycle.Journal
	triggered bool
}

func (j *cancellingArchiveJournal) Get(id string) (lifecycle.Operation, error) {
	if !j.triggered {
		j.triggered = true
		_, _, _ = j.delegate.RequestCancellation(id)
	}
	return j.delegate.Get(id)
}

func (j *cancellingArchiveJournal) Update(operation lifecycle.Operation) (lifecycle.Operation, error) {
	return j.delegate.Update(operation)
}

type failingArchiveCoordinator struct {
	delegate archiveCoordinator
	finish   error
}

func (c *failingArchiveCoordinator) Begin(operation lifecycle.Operation) (lifecycle.Operation, bool, error) {
	return c.delegate.Begin(operation)
}

func (c *failingArchiveCoordinator) Finish(string, lifecycle.OperationStatus, []lifecycle.OperationItemResult, []string, []string) (lifecycle.Operation, error) {
	return lifecycle.Operation{}, c.finish
}
