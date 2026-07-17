package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/desktop"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DesktopApp struct {
	ctx     context.Context
	service *desktop.Service
}

func NewDesktopApp(service *desktop.Service) *DesktopApp { return &DesktopApp{service: service} }
func (a *DesktopApp) startup(ctx context.Context)        { a.ctx = ctx }
func (a *DesktopApp) Bootstrap() desktop.Bootstrap       { return a.service.Bootstrap() }
func (a *DesktopApp) ListProfiles() []domain.Profile     { return a.service.ListProfiles() }
func (a *DesktopApp) ListKernels() []kernel.Record       { return a.service.ListKernels() }
func (a *DesktopApp) ListAdapters() []adapter.Record     { return a.service.ListAdapters() }
func (a *DesktopApp) ListSessions() []supervisor.Session { return a.service.ListSessions() }
func (a *DesktopApp) ListCredentials() []credential.Record {
	return a.service.ListCredentials()
}
func (a *DesktopApp) Capabilities(provider, version string) (fingerprint.Capabilities, error) {
	return a.service.Capabilities(provider, version)
}
func (a *DesktopApp) CreateProfile(input domain.Profile) (domain.Profile, error) {
	return a.service.CreateProfile(input)
}
func (a *DesktopApp) UpdateProfile(input domain.Profile) (domain.Profile, error) {
	return a.service.UpdateProfile(input)
}
func (a *DesktopApp) CloneProfile(id, name string) (domain.Profile, error) {
	return a.service.CloneProfile(id, name)
}
func (a *DesktopApp) DeleteProfile(id string) error { return a.service.DeleteProfile(id) }
func (a *DesktopApp) SaveCredential(request credential.SaveRequest) (credential.Record, error) {
	return a.service.SaveCredential(request)
}
func (a *DesktopApp) DeleteCredential(id string) error { return a.service.DeleteCredential(id) }
func (a *DesktopApp) BuildLaunchPlan(request desktop.LaunchPlanRequest) (domain.LaunchPlan, error) {
	return a.service.BuildLaunchPlan(request)
}
func (a *DesktopApp) StartProfile(profileID string) (supervisor.Session, error) {
	return a.service.StartProfile(a.runtimeContext(), profileID)
}
func (a *DesktopApp) StopProfile(profileID string) (supervisor.Session, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 5*time.Second)
	defer cancel()
	return a.service.StopProfile(ctx, profileID)
}
func (a *DesktopApp) PickKernelExecutable() (string, error) {
	return runtime.OpenFileDialog(a.runtimeContext(), runtime.OpenDialogOptions{Title: "Select Chromium executable"})
}
func (a *DesktopApp) ImportKernel(request kernel.ImportRequest) (kernel.Record, error) {
	return a.service.ImportKernel(request)
}
func (a *DesktopApp) VerifyKernel(id string) (kernel.Record, error) {
	return a.service.VerifyKernel(id)
}
func (a *DesktopApp) DeleteKernel(id string) error { return a.service.DeleteKernel(id) }
func (a *DesktopApp) PickAdapterExecutable() (string, error) {
	return runtime.OpenFileDialog(a.runtimeContext(), runtime.OpenDialogOptions{Title: "Select Xray or sing-box executable"})
}
func (a *DesktopApp) ImportAdapter(request adapter.ImportRequest) (adapter.Record, error) {
	return a.service.ImportAdapter(request)
}
func (a *DesktopApp) VerifyAdapter(id string) (adapter.Record, error) {
	return a.service.VerifyAdapter(id)
}
func (a *DesktopApp) DeleteAdapter(id string) error { return a.service.DeleteAdapter(id) }
func (a *DesktopApp) shutdown(ctx context.Context) {
	shutdownContext, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	_ = a.service.Shutdown(shutdownContext)
}
func (a *DesktopApp) runtimeContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
