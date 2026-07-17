package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/proxy"
	"github.com/knownothing20/veilium-browser/internal/proxybridge"
	"github.com/knownothing20/veilium-browser/internal/proxydiagnostics"
)

type proxyDiagnosticRunner interface {
	Run(context.Context, proxydiagnostics.Request) (proxydiagnostics.Report, error)
}

var proxyDiagnosticRunners sync.Map

func proxyDiagnosticsRunnerFor(service *Service) (proxyDiagnosticRunner, error) {
	if value, ok := proxyDiagnosticRunners.Load(service); ok {
		return value.(proxyDiagnosticRunner), nil
	}
	runner, err := proxydiagnostics.New(
		func() proxybridge.Factory { return proxyBridgeFactory(service) },
		proxydiagnostics.DefaultConfig(),
	)
	if err != nil {
		return nil, err
	}
	actual, _ := proxyDiagnosticRunners.LoadOrStore(service, runner)
	return actual.(proxyDiagnosticRunner), nil
}

func setProxyDiagnosticsRunner(service *Service, runner proxyDiagnosticRunner) {
	if runner == nil {
		proxyDiagnosticRunners.Delete(service)
		return
	}
	proxyDiagnosticRunners.Store(service, runner)
}

func (s *Service) RunProxyDiagnostics(ctx context.Context, profileID string) (proxydiagnostics.Report, error) {
	item, err := s.store.Get(strings.TrimSpace(profileID))
	if err != nil {
		return proxydiagnostics.Report{}, err
	}
	if err := s.validateProxy(item); err != nil {
		return proxydiagnostics.Report{}, err
	}
	route, err := proxy.Resolve(item.Proxy.URL, item.Proxy.CredentialRef)
	if err != nil {
		return proxydiagnostics.Report{}, err
	}
	material := credential.Material{}
	if strings.TrimSpace(route.CredentialRef) != "" {
		material, err = s.credentials.Resolve(route.CredentialRef)
		if err != nil {
			return proxydiagnostics.Report{}, fmt.Errorf("resolve proxy credential for diagnostics: %w", err)
		}
	}
	runner, err := proxyDiagnosticsRunnerFor(s)
	if err != nil {
		return proxydiagnostics.Report{}, err
	}
	return runner.Run(ctx, proxydiagnostics.Request{Profile: item, Material: material})
}
