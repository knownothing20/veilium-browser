package localrecovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestRestorePreservesSuccessfulProfileWhenStageCleanupFails(t *testing.T) {
	harness := newRestoreHarness(t, map[string]string{"file.txt": "content"})
	harness.executor.removeStage = func(string, string) error { return errors.New("simulated restore staging cleanup failure") }

	result, err := harness.executor.Restore(context.Background(), harness.request)
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("cleanup failure did not require recovery: %v", err)
	}
	if result.Operation.Status != lifecycle.OperationPartial {
		t.Fatalf("cleanup failure did not preserve partial success: %#v", result.Operation)
	}
	if result.Profile.ID == "" || result.Lifecycle.State != lifecycle.StateDraft {
		t.Fatalf("successful restored Profile was lost: %#v %#v", result.Profile, result.Lifecycle)
	}
	if !containsString(result.Lifecycle.RecoveryCodes, "restore-staging-cleanup-required") {
		t.Fatalf("cleanup recovery code was not recorded: %#v", result.Lifecycle.RecoveryCodes)
	}
	if _, err := os.Stat(filepath.Join(result.Profile.UserDataDir, "file.txt")); err != nil {
		t.Fatalf("restored browser data is missing: %v", err)
	}
	if _, err := os.Stat(restoreStagePath(harness.executor.recoveryRoot, harness.request.OperationID)); err != nil {
		t.Fatalf("failed cleanup staging was not preserved: %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
