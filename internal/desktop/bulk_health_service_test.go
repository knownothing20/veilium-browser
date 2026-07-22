package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

func TestBulkRefreshProfileHealthReportsReadyAndReusesOperation(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	record, err := service.ImportKernel(kernel.ImportRequest{
		Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(created.UserDataDir, 0o700); err != nil {
		t.Fatal(err)
	}

	request := BulkHealthRefreshRequest{ProfileIDs: []string{created.ID}, IdempotencyKey: "health-refresh-1"}
	first, err := service.BulkRefreshProfileHealth(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Operation.Status != lifecycle.OperationCompleted || len(first.Reports) != 1 {
		t.Fatalf("unexpected first health refresh: %#v", first)
	}
	if first.Reports[0].Status != ProfileHealthReady {
		t.Fatalf("health status = %q, want %q: %#v", first.Reports[0].Status, ProfileHealthReady, first.Reports[0].Checks)
	}

	second, err := service.BulkRefreshProfileHealth(request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Operation.ID != first.Operation.ID {
		t.Fatalf("idempotent retry created operation %q, want %q", second.Operation.ID, first.Operation.ID)
	}
	if len(second.Reports) != 1 || second.Reports[0].Status != ProfileHealthReady {
		t.Fatalf("idempotent retry lost health result: %#v", second)
	}
	if operations := service.ListLifecycleOperations(); len(operations) != 1 {
		t.Fatalf("idempotent retry created %d operations, want 1", len(operations))
	}
}

func TestBulkRefreshProfileHealthReportsBlockedForMissingManagedKernel(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := newService(store, root, newFakeRuntime())
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(created.UserDataDir, 0o700); err != nil {
		t.Fatal(err)
	}

	result, err := service.BulkRefreshProfileHealth(BulkHealthRefreshRequest{
		ProfileIDs: []string{created.ID}, IdempotencyKey: "health-refresh-missing-kernel",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation.Status != lifecycle.OperationCompleted || len(result.Reports) != 1 {
		t.Fatalf("unexpected health refresh: %#v", result)
	}
	report := result.Reports[0]
	if report.Status != ProfileHealthBlocked {
		t.Fatalf("health status = %q, want %q", report.Status, ProfileHealthBlocked)
	}
	if checkStatus(report.Checks, "kernel") != HealthCheckFail {
		t.Fatalf("missing managed Kernel was not reported: %#v", report.Checks)
	}
	if len(result.Operation.Items) != 1 || result.Operation.Items[0].Status != lifecycle.ItemSucceeded {
		t.Fatalf("completed assessment should remain a successful operation item: %#v", result.Operation.Items)
	}
}

func TestHealthCheckEncodingRoundTripsDeterministically(t *testing.T) {
	input := []ProfileHealthCheck{
		newHealthCheck("lifecycle", HealthCheckPass),
		newHealthCheck("kernel", HealthCheckFail),
		newHealthCheck("managed-data", HealthCheckWarning),
	}
	got := decodeHealthChecks(encodeHealthChecks(input))
	if checkStatus(got, "lifecycle") != HealthCheckPass ||
		checkStatus(got, "kernel") != HealthCheckFail ||
		checkStatus(got, "managed-data") != HealthCheckWarning {
		t.Fatalf("health checks did not round-trip: %#v", got)
	}
	if deriveProfileHealthStatus(got) != ProfileHealthBlocked {
		t.Fatalf("decoded checks did not preserve aggregate status: %#v", got)
	}
}

func checkStatus(checks []ProfileHealthCheck, id string) HealthCheckStatus {
	for _, check := range checks {
		if check.ID == id {
			return check.Status
		}
	}
	return ""
}
