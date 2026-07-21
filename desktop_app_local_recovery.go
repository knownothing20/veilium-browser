package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/desktop"
)

func (a *DesktopApp) LocalRecoveryState() (desktop.LocalRecoveryBootstrap, error) {
	return a.service.LocalRecoveryState()
}

func (a *DesktopApp) LocalRecoveryPreflight(profileID string) (desktop.LocalRecoveryPreflight, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 10*time.Second)
	defer cancel()
	return a.service.LocalRecoveryPreflight(ctx, profileID)
}

func (a *DesktopApp) ListLocalSnapshots() (any, error) {
	return a.service.ListLocalSnapshots()
}

func (a *DesktopApp) GetLocalSnapshot(snapshotID string) (desktop.LocalSnapshotDetail, error) {
	return a.service.GetLocalSnapshot(snapshotID)
}

func (a *DesktopApp) ListLocalTrash() (any, error) {
	return a.service.ListLocalTrash()
}

func (a *DesktopApp) RefreshLocalRecovery() (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 10*time.Second)
	defer cancel()
	return a.service.RefreshLocalRecovery(ctx)
}

func (a *DesktopApp) CreateLocalSnapshot(request desktop.CreateLocalSnapshotRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	_, err := a.service.CreateLocalSnapshot(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) RestoreLocalSnapshot(request desktop.RestoreLocalSnapshotRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	_, err := a.service.RestoreLocalSnapshot(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) ArchiveProfile(request desktop.ArchiveProfileRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Minute)
	defer cancel()
	_, err := a.service.ArchiveProfile(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) UnarchiveProfile(request desktop.ArchiveProfileRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Minute)
	defer cancel()
	_, err := a.service.UnarchiveProfile(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) TrashProfile(request desktop.TrashProfileRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	_, err := a.service.TrashProfile(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) RestoreTrashedProfile(request desktop.TrashProfileActionRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	_, err := a.service.RestoreTrashedProfile(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) PermanentlyDeleteTrashedProfile(request desktop.TrashProfileActionRequest) (desktop.LocalRecoveryBootstrap, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	_, err := a.service.PermanentlyDeleteTrashedProfile(ctx, request)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}

func (a *DesktopApp) CancelLocalRecoveryOperation(operationID string) (desktop.LocalRecoveryBootstrap, error) {
	_, err := a.service.CancelLocalRecoveryOperation(operationID)
	state, stateErr := a.service.LocalRecoveryState()
	if err != nil {
		return state, err
	}
	return state, stateErr
}
