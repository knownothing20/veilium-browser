package main

import (
	"context"

	"github.com/knownothing20/veilium-browser/internal/desktop"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
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
func (a *DesktopApp) BuildLaunchPlan(request desktop.LaunchPlanRequest) (domain.LaunchPlan, error) {
	return a.service.BuildLaunchPlan(request)
}
func (a *DesktopApp) PickKernelExecutable() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select Chromium executable"})
}
func (a *DesktopApp) ImportKernel(request kernel.ImportRequest) (kernel.Record, error) {
	return a.service.ImportKernel(request)
}
func (a *DesktopApp) VerifyKernel(id string) (kernel.Record, error) {
	return a.service.VerifyKernel(id)
}
func (a *DesktopApp) DeleteKernel(id string) error { return a.service.DeleteKernel(id) }
