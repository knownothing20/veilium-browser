package main

import (
	"context"
	"time"

	"github.com/knownothing20/veilium-browser/internal/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *DesktopApp) BulkUpdateProfileMetadata(request desktop.BulkMetadataUpdateRequest) (desktop.BulkMetadataUpdateResult, error) {
	return a.service.BulkUpdateProfileMetadata(request)
}

func (a *DesktopApp) BulkRefreshProfileHealth(request desktop.BulkHealthRefreshRequest) (desktop.BulkHealthRefreshResult, error) {
	return a.service.BulkRefreshProfileHealth(request)
}

func (a *DesktopApp) BulkApplyProfileLifecycle(request desktop.BulkLifecycleRequest) (desktop.BulkLifecycleResult, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 2*time.Hour)
	defer cancel()
	return a.service.BulkApplyProfileLifecycle(ctx, request)
}

func (a *DesktopApp) BulkExportPortableProfiles(request desktop.BulkPortableExportRequest) (desktop.BulkPortableExportResult, error) {
	return a.service.BulkExportPortableProfiles(request)
}

func (a *DesktopApp) RefreshStorageManagement() (desktop.StorageManagementState, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 30*time.Second)
	defer cancel()
	return a.service.RefreshStorageManagement(ctx)
}

func (a *DesktopApp) ReviewStorageManagement() (desktop.StorageManagementReview, error) {
	ctx, cancel := context.WithTimeout(a.runtimeContext(), 30*time.Second)
	defer cancel()
	return a.service.ReviewStorageManagement(ctx)
}

func (a *DesktopApp) PickOperationReportFile(operationID string) (string, error) {
	name := portableFilename(operationID)
	return wailsruntime.SaveFileDialog(a.runtimeContext(), wailsruntime.SaveDialogOptions{
		Title:           "Export Veilium operation report",
		DefaultFilename: name + ".veilium-operation-report.json",
	})
}

func (a *DesktopApp) ExportLifecycleOperationReport(request desktop.OperationReportExportRequest) (desktop.OperationReportExportResult, error) {
	return a.service.ExportLifecycleOperationReport(request)
}
