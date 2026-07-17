package main

import (
	"context"

	"github.com/knownothing20/veilium-browser/internal/desktop"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

type DesktopApp struct {
	ctx     context.Context
	service *desktop.Service
}

func NewDesktopApp(service *desktop.Service) *DesktopApp {
	return &DesktopApp{service: service}
}

func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *DesktopApp) Bootstrap() desktop.Bootstrap {
	return a.service.Bootstrap()
}

func (a *DesktopApp) ListProfiles() []domain.Profile {
	return a.service.ListProfiles()
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

func (a *DesktopApp) DeleteProfile(id string) error {
	return a.service.DeleteProfile(id)
}

func (a *DesktopApp) BuildLaunchPlan(request desktop.LaunchPlanRequest) (domain.LaunchPlan, error) {
	return a.service.BuildLaunchPlan(request)
}
