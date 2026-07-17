package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/proxydiagnostics"
)

func (a *DesktopApp) RunProxyDiagnostics(profileID string) (proxydiagnostics.Report, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 20*time.Second)
	defer cancel()
	return a.service.RunProxyDiagnostics(ctx, profileID)
}
