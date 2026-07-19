package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/networkevidence"
)

type NetworkProbeConfiguration struct {
	Configured bool                     `json:"configured"`
	ProbeSet   networkevidence.ProbeSet `json:"probeSet"`
}

func (a *DesktopApp) GetNetworkProbeSet() (NetworkProbeConfiguration, error) {
	set, configured, err := a.service.NetworkProbeSet()
	return NetworkProbeConfiguration{Configured: configured, ProbeSet: set}, err
}

func (a *DesktopApp) SaveNetworkProbeSet(set networkevidence.ProbeSet) (networkevidence.ProbeSet, error) {
	return a.service.SaveNetworkProbeSet(set)
}

func (a *DesktopApp) DeleteNetworkProbeSet() error {
	return a.service.DeleteNetworkProbeSet()
}

func (a *DesktopApp) RunNetworkEvidence(profileID string) (networkevidence.Run, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 60*time.Second)
	defer cancel()
	return a.service.RunNetworkEvidence(ctx, profileID)
}

func (a *DesktopApp) CancelNetworkEvidence(profileID string) error {
	return a.service.CancelNetworkEvidence(profileID)
}

func (a *DesktopApp) ListNetworkEvidence(profileID string) ([]networkevidence.Run, error) {
	return a.service.ListNetworkEvidence(profileID)
}

func (a *DesktopApp) GetNetworkEvidence(id string) (networkevidence.Run, error) {
	return a.service.GetNetworkEvidence(id)
}

func (a *DesktopApp) DeleteNetworkEvidence(id string) error {
	return a.service.DeleteNetworkEvidence(id)
}

func (a *DesktopApp) NetworkEvidenceActive(profileID string) bool {
	return a.service.NetworkEvidenceActive(profileID)
}

func (a *DesktopApp) NetworkCompatibilityMatrix() (networkevidence.CompatibilityMatrix, error) {
	return a.service.NetworkCompatibilityMatrix()
}
