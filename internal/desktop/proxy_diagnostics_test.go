package desktop

import (
	"context"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/proxydiagnostics"
)

func TestProxyDiagnosticsResolvesVaultMaterialWithoutChangingProfile(t *testing.T) {
	service, _, _, record := bridgeTestService(t)
	created := createBridgeProfile(t, service, record.ID)
	fake := &capturingDiagnosticRunner{
		report: proxydiagnostics.Report{
			ProfileID:   created.ID,
			ProfileName: created.Name,
			Status:      proxydiagnostics.StatusHealthy,
			ExitIP:      "203.0.113.7",
			StartedAt:   time.Now().UTC(),
			CompletedAt: time.Now().UTC(),
		},
	}
	setProxyDiagnosticsRunner(service, fake)
	t.Cleanup(func() { setProxyDiagnosticsRunner(service, nil) })

	report, err := service.RunProxyDiagnostics(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != proxydiagnostics.StatusHealthy || report.ExitIP != "203.0.113.7" {
		t.Fatalf("unexpected report: %#v", report)
	}
	if fake.request.Profile.ID != created.ID {
		t.Fatalf("unexpected profile: %#v", fake.request.Profile)
	}
	if fake.request.Material != (credential.Material{Username: "alice", Secret: "top-secret"}) {
		t.Fatalf("vault material was not resolved for the diagnostic runner: %#v", fake.request.Material)
	}
	stored, err := service.store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Proxy.CredentialRef != record.ID || stored.Proxy.URL != "http://proxy.example:3128" {
		t.Fatalf("diagnostics mutated the stored profile: %#v", stored.Proxy)
	}
}

type capturingDiagnosticRunner struct {
	request proxydiagnostics.Request
	report  proxydiagnostics.Report
	err     error
}

func (r *capturingDiagnosticRunner) Run(_ context.Context, request proxydiagnostics.Request) (proxydiagnostics.Report, error) {
	r.request = request
	return r.report, r.err
}
