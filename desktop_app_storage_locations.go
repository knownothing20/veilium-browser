package main

import "github.com/knownothing20/veilium-browser/internal/desktop"

func (a *DesktopApp) GetManagedStorageLocations() desktop.ManagedStorageLocations {
	return a.service.ManagedStorageLocations()
}
