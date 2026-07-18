package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

func TestStoreSavesAndLoadsPrivateEvidence(t *testing.T) {
	now := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "evidence")
	store, err := OpenStore(root, StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	run := testRun("00000000000000000000000000000001", now)
	if err := store.Save(run); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ProfileID != run.ProfileID || loaded.Status != RunPassed || len(loaded.Observations) != 1 {
		t.Fatalf("unexpected loaded evidence: %#v", loaded)
	}
	if runtime.GOOS != "windows" {
		rootInfo, err := os.Stat(root)
		if err != nil {
			t.Fatal(err)
		}
		if rootInfo.Mode().Perm() != 0o700 {
			t.Fatalf("unexpected evidence directory permissions %o", rootInfo.Mode().Perm())
		}
		fileInfo, err := os.Stat(filepath.Join(root, run.ID+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if fileInfo.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected evidence file permissions %o", fileInfo.Mode().Perm())
		}
	}
}

func TestStorePrunesExpiredAndLimitsCount(t *testing.T) {
	now := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)
	store, err := OpenStore(filepath.Join(t.TempDir(), "evidence"), StoreOptions{
		Retention: 24 * time.Hour,
		MaxRuns:   2,
		Now:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	first := testRun("00000000000000000000000000000001", now.Add(-3*time.Hour))
	second := testRun("00000000000000000000000000000002", now.Add(-2*time.Hour))
	third := testRun("00000000000000000000000000000003", now.Add(-time.Hour))
	for _, run := range []Run{first, second, third} {
		if err := store.Save(run); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != third.ID || items[1].ID != second.ID {
		t.Fatalf("unexpected retained evidence: %#v", items)
	}
	if _, err := store.Get(first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected oldest evidence to be removed, got %v", err)
	}

	expired := testRun("00000000000000000000000000000004", now.Add(-48*time.Hour))
	expired.ExpiresAt = now.Add(-time.Hour)
	if err := store.Save(expired); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(expired.ID); !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrExpired) {
		t.Fatalf("expected expired evidence to be unavailable, got %v", err)
	}
}

func TestRunValidationRejectsUnboundedOrIncompleteEvidence(t *testing.T) {
	now := time.Now().UTC()
	run := testRun("00000000000000000000000000000001", now)
	run.Observations[0].Observed = strings.Repeat("x", 4097)
	if err := run.Validate(); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected observation bound error, got %v", err)
	}

	run = testRun("00000000000000000000000000000002", now)
	run.Status = RunFailed
	run.FailureCode = ""
	if err := run.Validate(); err == nil || !strings.Contains(err.Error(), "failure code") {
		t.Fatalf("expected failed-run code error, got %v", err)
	}
}

func TestStoreRejectsDuplicateAndSymlinkedRoot(t *testing.T) {
	now := time.Now().UTC()
	store, err := OpenStore(filepath.Join(t.TempDir(), "evidence"), StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	run := testRun("00000000000000000000000000000001", now)
	if err := store.Save(run); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(run); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	if runtime.GOOS != "windows" {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(root, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenStore(link, StoreOptions{}); err == nil || !strings.Contains(err.Error(), "real directory") {
			t.Fatalf("expected symlink root rejection, got %v", err)
		}
	}
}

func testRun(id string, started time.Time) Run {
	completed := started.Add(time.Second)
	return Run{
		SchemaVersion:    SchemaVersion,
		ID:               id,
		ProfileID:        "profile-1",
		ProfileName:      "Profile One",
		ProviderID:       fingerprint.ProviderCustom,
		ProviderRevision: 1,
		ProviderTrust:    fingerprint.TrustCustom,
		BinaryIdentity: kernel.ProviderBinaryIdentity{
			SchemaVersion:         kernel.BinaryIdentitySchemaVersion,
			ProviderID:            fingerprint.ProviderCustom,
			ProviderRevision:      1,
			ProviderTrust:         fingerprint.TrustCustom,
			BrowserVersion:        "148.0.0",
			OperatingSystem:       runtime.GOOS,
			Architecture:          runtime.GOARCH,
			ExecutablePath:        "/managed/chrome",
			ExecutableSize:        10,
			ExecutableSHA256:      strings.Repeat("a", 64),
			IntegrityStatus:       kernel.StatusVerified,
			VerificationTimestamp: started.Format(time.RFC3339Nano),
			Provenance:            "managed-local-import",
		},
		BrowserVersion:  "148.0.0",
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
		HarnessRevision: HarnessRevision,
		Status:          RunPassed,
		StartedAt:       started,
		CompletedAt:     &completed,
		ExpiresAt:       started.Add(24 * time.Hour),
		Observations: []Observation{{
			ID:       "navigator.userAgent",
			Context:  ContextTopLevel,
			Status:   ObservationPassed,
			Expected: "Chromium",
			Observed: "Chromium/148",
		}},
	}
}
