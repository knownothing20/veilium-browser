package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type failingRestoreProfileStore struct {
	delegate   restoreProfileStore
	failCreate bool
	failDelete bool
}

func (s *failingRestoreProfileStore) Get(id string) (domain.Profile, error) {
	return s.delegate.Get(id)
}

func (s *failingRestoreProfileStore) Create(value domain.Profile) (domain.Profile, error) {
	if s.failCreate {
		return domain.Profile{}, errors.New("simulated Profile persistence failure")
	}
	return s.delegate.Create(value)
}

func (s *failingRestoreProfileStore) Delete(id string) error {
	if s.failDelete {
		return errors.New("simulated Profile rollback failure")
	}
	return s.delegate.Delete(id)
}

func TestRestoreRollsBackActivationRenameFailure(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "content"})
	harness.executor.rename = func(string, string) error { return errors.New("simulated activation failure") }

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if err == nil || errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("activation failure should roll back cleanly: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("activation failure produced wrong operation state: %#v", result.Operation)
	}
	assertRestoreRolledBack(t, harness)
}

func TestRestoreRollsBackProfilePersistenceFailure(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "content"})
	harness.executor.profiles = &failingRestoreProfileStore{delegate: harness.profiles, failCreate: true}

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if err == nil || errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("Profile persistence failure should roll back cleanly: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("Profile persistence failure produced wrong operation state: %#v", result.Operation)
	}
	assertRestoreRolledBack(t, harness)
}

func TestRestoreDoesNotOverwriteExistingTarget(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "content"})
	destinationID := restoreDestinationID(harness.request.IdempotencyKey, harness.request.SnapshotID)
	finalPath := filepath.Join(harness.executor.profilesRoot, destinationID)
	if err := os.Mkdir(finalPath, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(finalPath, "existing.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := harness.profiles.Create(domain.Profile{
		ID:          destinationID,
		Name:        "Existing Profile",
		UserDataDir: finalPath,
		Fingerprint: domain.FingerprintConfig{Seed: "existing-seed"},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if !errors.Is(err, lifecycle.ErrConflict) {
		t.Fatalf("existing target was not rejected as a conflict: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationFailed {
		t.Fatalf("target conflict produced wrong operation state: %#v", result.Operation)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing target data changed: %q %v", data, err)
	}
	existing, err := harness.profiles.Get(destinationID)
	if err != nil || existing.Name != "Existing Profile" || existing.Fingerprint.Seed != "existing-seed" {
		t.Fatalf("existing Profile metadata changed: %#v %v", existing, err)
	}
	if _, err := harness.snapshot.records.Get(destinationID); !errors.Is(err, lifecycle.ErrNotFound) {
		t.Fatalf("target conflict left a lifecycle reservation: %v", err)
	}
	if _, err := harness.profiles.Get(harness.source.ID); errors.Is(err, profile.ErrNotFound) {
		t.Fatal("source Profile was removed")
	}
}
