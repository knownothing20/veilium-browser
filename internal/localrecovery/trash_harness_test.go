package localrecovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type trashBlockers struct {
	active    bool
	dependent bool
}

type trashHarness struct {
	dataRoot    string
	profileRoot string
	profiles    *profile.Store
	records     *lifecycle.RecordStore
	journal     *lifecycle.Journal
	coordinator *lifecycle.Coordinator
	trash       *TrashStore
	executor    *TrashExecutor
	request     TrashRequest
	blockers    *trashBlockers
}

func newTrashHarness(t *testing.T, state lifecycle.State) trashHarness {
	t.Helper()
	dataRoot := t.TempDir()
	if err := os.Chmod(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	profileRoot := filepath.Join(dataRoot, "profiles", "profile-a")
	if err := os.MkdirAll(filepath.Join(profileRoot, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "sentinel.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "nested", "state.bin"), []byte{1, 2, 3, 4}, 0o600); err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(filepath.Join(dataRoot, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.Create(domain.Profile{
		ID:   "profile-a",
		Name: "Profile A",
		Fingerprint: domain.FingerprintConfig{
			Seed:         "profile-a-seed",
			Platform:     "Windows",
			Brand:        "Chrome",
			Language:     "en-US",
			Timezone:     "UTC",
			ScreenWidth:  1440,
			ScreenHeight: 900,
			WebRTCPolicy: "default",
			CanvasMode:   "stable-noise",
			AudioMode:    "stable-noise",
			FontMode:     "native",
			GPUProfile:   "default",
		},
		UserDataDir: profileRoot,
	}); err != nil {
		t.Fatal(err)
	}
	records, err := lifecycle.OpenRecordStore(filepath.Join(dataRoot, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	record := lifecycle.Record{
		ProfileID:     "profile-a",
		State:         state,
		ManagedDir:    "profiles/profile-a",
		RecoveryCodes: []string{"existing-recovery"},
	}
	if state == lifecycle.StateArchived {
		now := time.Now().UTC().Add(-time.Hour)
		record.ArchivedAt = &now
		record.LimitationCodes = []string{"profile-archived", "archive-origin-available", "existing-limit"}
	} else {
		record.LimitationCodes = []string{"existing-limit"}
	}
	if _, err := records.Create(record); err != nil {
		t.Fatal(err)
	}
	journal, err := lifecycle.OpenJournal(filepath.Join(dataRoot, "lifecycle-operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	blockers := &trashBlockers{}
	coordinator, err := lifecycle.NewCoordinator(records, journal, func(string) (string, bool) {
		return "browser-session-active", blockers.active
	}, func(string) (string, bool) {
		return "protected-dependent-operation", blockers.dependent
	})
	if err != nil {
		t.Fatal(err)
	}
	trashStore, err := OpenTrashStore(filepath.Join(dataRoot, "local-recovery", "trash-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	executor, err := OpenTrashExecutor(dataRoot, records, journal, coordinator, profiles, trashStore)
	if err != nil {
		t.Fatal(err)
	}
	return trashHarness{
		dataRoot:    dataRoot,
		profileRoot: profileRoot,
		profiles:    profiles,
		records:     records,
		journal:     journal,
		coordinator: coordinator,
		trash:       trashStore,
		executor:    executor,
		request: TrashRequest{
			OperationID:        "trash-operation-a",
			ProfileID:          "profile-a",
			IdempotencyKey:     "trash-request-a",
			ApplicationVersion: "0.15.0-dev",
			RetentionDays:      45,
		},
		blockers: blockers,
	}
}

func assertTrashUnlocked(t *testing.T, harness trashHarness) lifecycle.Record {
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

func containsCode(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

type failingTrashRecordStore struct {
	delegate   trashRecordStore
	failUpdate bool
}

func (s *failingTrashRecordStore) Get(id string) (lifecycle.Record, error) {
	return s.delegate.Get(id)
}

func (s *failingTrashRecordStore) Update(record lifecycle.Record) (lifecycle.Record, error) {
	if s.failUpdate {
		return lifecycle.Record{}, errors.New("simulated lifecycle persistence failure")
	}
	return s.delegate.Update(record)
}

func (s *failingTrashRecordStore) AddRecoveryCode(profileID, code string) (lifecycle.Record, bool, error) {
	return s.delegate.AddRecoveryCode(profileID, code)
}

type cancellingTrashJournal struct {
	delegate  *lifecycle.Journal
	triggered bool
}

func (j *cancellingTrashJournal) Get(id string) (lifecycle.Operation, error) {
	if !j.triggered {
		j.triggered = true
		_, _, _ = j.delegate.RequestCancellation(id)
	}
	return j.delegate.Get(id)
}

func (j *cancellingTrashJournal) Update(operation lifecycle.Operation) (lifecycle.Operation, error) {
	return j.delegate.Update(operation)
}

func (j *cancellingTrashJournal) List() []lifecycle.Operation {
	return j.delegate.List()
}

type failingTrashCatalog struct {
	delegate   trashCatalog
	failUpdate bool
	failRemove bool
}

func (s *failingTrashCatalog) List() []TrashRecord {
	return s.delegate.List()
}

func (s *failingTrashCatalog) Get(id string) (TrashRecord, error) {
	return s.delegate.Get(id)
}

func (s *failingTrashCatalog) Create(record TrashRecord) (TrashRecord, error) {
	return s.delegate.Create(record)
}

func (s *failingTrashCatalog) Update(record TrashRecord) (TrashRecord, error) {
	if s.failUpdate {
		return TrashRecord{}, errors.New("simulated trash catalog update failure")
	}
	return s.delegate.Update(record)
}

func (s *failingTrashCatalog) Remove(id string, revision uint64) (TrashRecord, error) {
	if s.failRemove {
		return TrashRecord{}, errors.New("simulated trash catalog removal failure")
	}
	return s.delegate.Remove(id, revision)
}
