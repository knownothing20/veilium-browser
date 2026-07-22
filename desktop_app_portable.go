package main

import (
	"path/filepath"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/desktop"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *DesktopApp) PickPortableExportFile(profileName string) (string, error) {
	name := portableFilename(profileName)
	return runtime.SaveFileDialog(a.runtimeContext(), runtime.SaveDialogOptions{
		Title:           "Export portable Veilium Profile",
		DefaultFilename: name + ".veilium-profile.json",
	})
}

func (a *DesktopApp) PickPortableImportFile() (string, error) {
	return runtime.OpenFileDialog(a.runtimeContext(), runtime.OpenDialogOptions{
		Title: "Import portable Veilium Profile",
	})
}

func (a *DesktopApp) ExportPortableProfile(request desktop.PortableExportRequest) (desktop.PortableExportResult, error) {
	return a.service.ExportPortableProfile(request)
}

func (a *DesktopApp) PreviewPortableImport(path string) (desktop.PortableImportPreview, error) {
	return a.service.PreviewPortableImport(path)
}

func (a *DesktopApp) ImportPortableProfile(request desktop.PortableImportRequest) (desktop.PortableImportResult, error) {
	return a.service.ImportPortableProfile(request)
}

func (a *DesktopApp) ListPortableTemplates() ([]portableprofile.Template, error) {
	return a.service.ListPortableTemplates()
}

func (a *DesktopApp) CreatePortableTemplate(request desktop.PortableTemplateCreateRequest) (portableprofile.Template, error) {
	return a.service.CreatePortableTemplate(request)
}

func (a *DesktopApp) DeletePortableTemplate(templateID string) error {
	return a.service.DeletePortableTemplate(templateID)
}

func (a *DesktopApp) ApplyPortableTemplate(request desktop.PortableTemplateApplyRequest) (desktop.PortableImportResult, error) {
	return a.service.ApplyPortableTemplate(request)
}

func portableFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "veilium-profile"
	}
	value = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00', '\r', '\n':
			return '-'
		default:
			return r
		}
	}, value)
	value = strings.Trim(strings.TrimSpace(value), ".")
	if value == "" {
		return "veilium-profile"
	}
	if len(value) > 80 {
		value = value[:80]
	}
	return filepath.Base(value)
}
