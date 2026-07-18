package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/evidence"
)

func (a *DesktopApp) RunEvidence(profileID string) (evidence.Run, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 25*time.Second)
	defer cancel()
	return a.service.RunEvidence(ctx, profileID)
}

func (a *DesktopApp) CancelEvidence(profileID string) error {
	return a.service.CancelEvidence(profileID)
}

func (a *DesktopApp) ListEvidence(profileID string) ([]evidence.Run, error) {
	return a.service.ListEvidence(profileID)
}

func (a *DesktopApp) GetEvidence(id string) (evidence.Run, error) {
	return a.service.GetEvidence(id)
}

func (a *DesktopApp) DeleteEvidence(id string) error {
	return a.service.DeleteEvidence(id)
}

func (a *DesktopApp) EvidenceActive(profileID string) bool {
	return a.service.EvidenceActive(profileID)
}
