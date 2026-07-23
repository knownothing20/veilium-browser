package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	managedStoragePresent    = "present"
	managedStorageMissing    = "missing"
	managedStorageUnexpected = "unexpected-entry"
	managedStorageLink       = "unsafe-link"
	managedStorageUnreadable = "unavailable"
)

type ManagedStorageLocation struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Path           string `json:"path"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	ReasonCode     string `json:"reasonCode,omitempty"`
	Description    string `json:"description"`
	Volume         string `json:"volume,omitempty"`
	OnSystemVolume bool   `json:"onSystemVolume"`
}

type ManagedStorageLocations struct {
	DataRoot       string                   `json:"dataRoot"`
	DataVolume     string                   `json:"dataVolume,omitempty"`
	SystemVolume   string                   `json:"systemVolume,omitempty"`
	OnSystemVolume bool                     `json:"onSystemVolume"`
	GeneratedAt    time.Time                `json:"generatedAt"`
	Locations      []ManagedStorageLocation `json:"locations"`
	Limitations    []string                 `json:"limitations"`
}

type managedStorageSpec struct {
	id          string
	label       string
	path        string
	kind        string
	description string
}

func (s *Service) ManagedStorageLocations() ManagedStorageLocations {
	dataRoot := filepath.Clean(s.dataRoot)
	dataVolume := managedStorageVolume(dataRoot)
	systemVolume := managedSystemVolume()
	specs := []managedStorageSpec{
		{id: "data-root", label: "Veilium data root", path: dataRoot, kind: "directory", description: "Parent boundary for Veilium-managed application data."},
		{id: "profile-data", label: "Profile browser data", path: s.profilesDir, kind: "directory", description: "One isolated managed user-data directory per Profile."},
		{id: "kernel-packages", label: "Managed browser Kernels", path: filepath.Join(dataRoot, "kernels"), kind: "directory", description: "Integrity-recorded Chromium executables and complete reviewed packages."},
		{id: "kernel-installer", label: "Kernel installer workspace", path: filepath.Join(dataRoot, "kernel-installer"), kind: "directory", description: "Private download and staging boundary for reviewed Kernel installation."},
		{id: "adapter-packages", label: "Managed proxy adapters", path: filepath.Join(dataRoot, "adapters"), kind: "directory", description: "Integrity-recorded Xray and sing-box executables."},
		{id: "adapter-installer", label: "Adapter installer workspace", path: filepath.Join(dataRoot, "adapter-installer"), kind: "directory", description: "Private download and staging boundary for reviewed adapter installation."},
		{id: "adapter-runtime", label: "Adapter runtime", path: filepath.Join(dataRoot, "adapter-runtime"), kind: "directory", description: "Private per-session adapter configurations and runtime state."},
		{id: "runtime-logs", label: "Runtime logs", path: filepath.Join(dataRoot, "runtime-logs"), kind: "directory", description: "Private logs produced by Veilium-managed browser sessions."},
		{id: "local-recovery", label: "Snapshots and recoverable trash", path: filepath.Join(dataRoot, "local-recovery"), kind: "directory", description: "Verified local snapshots, retained trash, catalogs, staging, and reconciliation state."},
		{id: "lifecycle-records", label: "Lifecycle records", path: filepath.Join(dataRoot, "lifecycle.json"), kind: "file", description: "Authoritative Profile lifecycle states and locks."},
		{id: "lifecycle-journal", label: "Lifecycle operation history", path: filepath.Join(dataRoot, "lifecycle-operations.json"), kind: "file", description: "Authoritative Phase 5 operation journal and per-item results."},
		{id: "credential-metadata", label: "Credential metadata", path: filepath.Join(dataRoot, "credentials.json"), kind: "file", description: "Non-secret credential references; secret values remain in the operating-system vault."},
		{id: "portable-templates", label: "Portable templates", path: filepath.Join(dataRoot, "portable-templates.json"), kind: "file", description: "Reusable non-secret Profile template catalog."},
	}

	locations := make([]ManagedStorageLocation, 0, len(specs))
	for _, spec := range specs {
		locations = append(locations, inspectManagedStorageLocation(spec, systemVolume))
	}
	limitations := []string{
		"This view lists only fixed Veilium-managed locations and does not browse arbitrary filesystem content.",
		"It does not move data, change the data root, create links, clean storage, or delete files.",
		"Credential values remain in the operating-system vault and are not present in the credential metadata file.",
	}
	if runtime.GOOS == "windows" && systemVolume == "" {
		limitations = append(limitations, "The Windows system volume could not be determined from the current process environment.")
	}
	return ManagedStorageLocations{
		DataRoot:       dataRoot,
		DataVolume:     dataVolume,
		SystemVolume:   systemVolume,
		OnSystemVolume: sameManagedStorageVolume(dataVolume, systemVolume),
		GeneratedAt:    time.Now().UTC(),
		Locations:      locations,
		Limitations:    limitations,
	}
}

func inspectManagedStorageLocation(spec managedStorageSpec, systemVolume string) ManagedStorageLocation {
	path := filepath.Clean(spec.path)
	result := ManagedStorageLocation{
		ID: spec.id, Label: spec.label, Path: path, Kind: spec.kind, Description: spec.description,
		Volume: managedStorageVolume(path),
	}
	result.OnSystemVolume = sameManagedStorageVolume(result.Volume, systemVolume)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		result.Status = managedStorageMissing
		result.ReasonCode = "managed-location-not-created"
		return result
	}
	if err != nil {
		result.Status = managedStorageUnreadable
		result.ReasonCode = "managed-location-inspection-failed"
		return result
	}
	if info.Mode()&os.ModeSymlink != 0 {
		result.Status = managedStorageLink
		result.ReasonCode = "managed-location-is-link"
		return result
	}
	if spec.kind == "directory" && !info.IsDir() {
		result.Status = managedStorageUnexpected
		result.ReasonCode = "managed-directory-is-not-directory"
		return result
	}
	if spec.kind == "file" && !info.Mode().IsRegular() {
		result.Status = managedStorageUnexpected
		result.ReasonCode = "managed-file-is-not-regular"
		return result
	}
	result.Status = managedStoragePresent
	return result
}

func managedSystemVolume() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	value := strings.TrimSpace(os.Getenv("SystemDrive"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("SystemRoot"))
	}
	return managedStorageVolume(value)
}

func managedStorageVolume(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return ""
	}
	if volume := filepath.VolumeName(path); volume != "" {
		return strings.TrimRight(volume, `\/`)
	}
	if filepath.IsAbs(path) {
		return string(filepath.Separator)
	}
	return ""
}

func sameManagedStorageVolume(left, right string) bool {
	left = strings.TrimRight(strings.TrimSpace(left), `\/`)
	right = strings.TrimRight(strings.TrimSpace(right), `\/`)
	return left != "" && right != "" && strings.EqualFold(left, right)
}
