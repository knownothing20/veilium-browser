package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/desktop"
)

func (a *DesktopApp) BulkUpdateProfileMetadata(request desktop.BulkMetadataUpdateRequest) (desktop.BulkMetadataUpdateResult, error) {
	return a.service.BulkUpdateProfileMetadata(request)
}

func (a *DesktopApp) BulkRefreshProfileHealth(request desktop.BulkHealthRefreshRequest) (desktop.BulkHealthRefreshResult, error) {
	return a.service.BulkRefreshProfileHealth(request)
}

func (a *DesktopApp) RefreshStorageManagement() (desktop.StorageManagementState, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 30*time.Second)
	defer cancel()
	return a.service.RefreshStorageManagement(ctx)
}
