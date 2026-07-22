package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestRestoreCreatesNewDraftIdentity(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{
		"Default/Preferences": `{"theme":"dark"}`,
		"Local State":         `{"browser":"state"}`,
	})
	var last RestoreProgress
	harness.executor.SetProgressCallback(func(progress RestoreProgress) { last = progress })

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if err != nil {
		t.Fatal(err)
	}
	expectedID := restoreDestinationID(harness.request.IdempotencyKey, harness.request.SnapshotID)
	if result.Operation.Status != lifecycle.OperationCompleted || result.Operation.ApplicationVersion != harness.request.ApplicationVersion {
		t.Fatalf("restore operation did not complete truthfully: %#v", result.Operation)
	}
	if result.Profile.ID != expectedID || result.Profile.ID == harness.source.ID {
		t.Fatalf("restore reused the source identity: %#v", result.Profile)
	}
	if result.Profile.Fingerprint.Seed == "" || result.Profile.Fingerprint.Seed == harness.source.Fingerprint.Seed {
		t.Fatalf("restore reused or omitted the fingerprint seed: %#v", result.Profile.Fingerprint)
	}
	if result.Profile.Kernel.ID != "" || result.Profile.Kernel.Executable != "" {
		t.Fatalf("restore copied local Kernel identity: %#v", result.Profile.Kernel)
	}
	if result.Profile.Proxy.CredentialRef != "" || result.Profile.Proxy.AdapterRef != "" {
		t.Fatalf("restore copied local dependency references: %#v", result.Profile.Proxy)
	}
	if result.Profile.UserDataDir != filepath.Join(harness.executor.profilesRoot, expectedID) {
		t.Fatalf("restore uses the wrong managed directory: %q", result.Profile.UserDataDir)
	}
	if result.Lifecycle.State != lifecycle.StateDraft || result.Lifecycle.SourceID != harness.request.SnapshotID || result.Lifecycle.ManagedDir != restoreManagedRef(expectedID) || result.Lifecycle.Lock != nil {
		t.Fatalf("restore lifecycle state is not a limited unlocked draft: %#v", result.Lifecycle)
	}
	if result.Dependencies.Kernel.Status != DependencyUserActionRequired {
		t.Fatalf("unselected Kernel was promoted: %#v", result.Dependencies.Kernel)
	}
	if last.Stage != RestoreStageFinished || last.FilesProcessed != 2 {
		t.Fatalf("unexpected final restore progress: %#v", last)
	}
	if _, err := os.Stat(restoreStagePath(harness.executor.recoveryRoot, harness.request.OperationID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restore staging remained after success: %v", err)
	}

	for relative, expected := range map[string]string{
		"Default/Preferences": `{"theme":"dark"}`,
		"Local State":         `{"browser":"state"}`,
	} {
		data, err := os.ReadFile(filepath.Join(result.Profile.UserDataDir, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != expected {
			t.Fatalf("restored file %q changed", relative)
		}
	}
}
