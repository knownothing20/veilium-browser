package fingerprint

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ContractSchemaVersion = 2

	ProviderCustom  = "custom-chromium"
	ProviderNative  = "native-chromium"
	ProviderPatched = "patched-chromium"
)

type TrustStatus string

const (
	TrustReviewed TrustStatus = "reviewed"
	TrustCustom   TrustStatus = "custom"
	TrustLegacy   TrustStatus = "legacy"
	TrustDisabled TrustStatus = "disabled"
	TrustInvalid  TrustStatus = "invalid"
)

type CapabilityStatus string

const (
	CapabilityVerified    CapabilityStatus = "verified"
	CapabilityPartial     CapabilityStatus = "partial"
	CapabilityUnsupported CapabilityStatus = "unsupported"
	CapabilityUnverified  CapabilityStatus = "unverified"
	CapabilityFailed      CapabilityStatus = "failed"
)

type CapabilityID string

const (
	CapabilityPlatformOverride    CapabilityID = "platform"
	CapabilityBrandOverride       CapabilityID = "browser-brand"
	CapabilityTimezoneOverride    CapabilityID = "timezone"
	CapabilitySurfaceSeed         CapabilityID = "surface-seed"
	CapabilitySurfaceControls     CapabilityID = "surface-controls"
	CapabilityHardwareConcurrency CapabilityID = "hardware-concurrency"
	CapabilityDeviceMemory        CapabilityID = "device-memory"
	CapabilityCustomGPU           CapabilityID = "custom-gpu"
	CapabilityProxyOnlyWebRTC     CapabilityID = "webrtc-policy"
)

type CapabilityDeclaration struct {
	ID               CapabilityID     `json:"id"`
	Status           CapabilityStatus `json:"status"`
	EvidenceRequired bool             `json:"evidenceRequired"`
	MinMajor         int              `json:"minMajor,omitempty"`
	MaxMajor         int              `json:"maxMajor,omitempty"`
	Limitation       string           `json:"limitation,omitempty"`
}

type ProviderDefinition struct {
	SchemaVersion         int                                    `json:"schemaVersion"`
	Revision              int                                    `json:"revision"`
	ID                    string                                 `json:"id"`
	Name                  string                                 `json:"name"`
	Description           string                                 `json:"description"`
	TrustStatus           TrustStatus                            `json:"trustStatus"`
	SourceURL             string                                 `json:"sourceUrl,omitempty"`
	LicenseSPDX           string                                 `json:"licenseSpdx,omitempty"`
	SupportedOS           []string                               `json:"supportedOs"`
	SupportedArch         []string                               `json:"supportedArch"`
	Versions              []string                               `json:"versions"`
	MinMajor              int                                    `json:"minMajor,omitempty"`
	MaxMajor              int                                    `json:"maxMajor,omitempty"`
	ExpectedExecutable    string                                 `json:"expectedExecutable,omitempty"`
	ProvenanceRequirement string                                 `json:"provenanceRequirement,omitempty"`
	Capabilities          map[CapabilityID]CapabilityDeclaration `json:"capabilities"`
	KnownLimitations      []string                               `json:"knownLimitations,omitempty"`
	PredecessorIDs        []string                               `json:"predecessorIds,omitempty"`
	ReplacementID         string                                 `json:"replacementId,omitempty"`
	DisabledReason        string                                 `json:"disabledReason,omitempty"`
	CreatedAt             string                                 `json:"createdAt"`
	ReviewedAt            string                                 `json:"reviewedAt,omitempty"`
}

type Capabilities struct {
	SchemaVersion int                                    `json:"schemaVersion"`
	Provider      string                                 `json:"provider"`
	Revision      int                                    `json:"revision"`
	TrustStatus   TrustStatus                            `json:"trustStatus"`
	MajorVersion  int                                    `json:"majorVersion"`
	Capabilities  map[CapabilityID]CapabilityDeclaration `json:"capabilities"`
	Limitations   []string                               `json:"limitations,omitempty"`
}

func (c Capabilities) State(id CapabilityID) CapabilityStatus {
	declaration, ok := c.Capabilities[id]
	if !ok {
		return CapabilityUnsupported
	}
	if declaration.MinMajor > 0 && c.MajorVersion < declaration.MinMajor {
		return CapabilityUnsupported
	}
	if declaration.MaxMajor > 0 && c.MajorVersion > declaration.MaxMajor {
		return CapabilityUnsupported
	}
	return declaration.Status
}

func (c Capabilities) Supports(id CapabilityID) bool {
	state := c.State(id)
	return state == CapabilityVerified || state == CapabilityPartial
}

func (c Capabilities) IsReviewed() bool { return c.TrustStatus == TrustReviewed }

func Providers() []string {
	definitions := Definitions()
	ids := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		ids = append(ids, definition.ID)
	}
	return ids
}

func Definitions() []ProviderDefinition {
	definitions := []ProviderDefinition{
		customDefinition(),
		legacyNativeDefinition(),
		legacyPatchedDefinition(),
	}
	result := make([]ProviderDefinition, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, cloneDefinition(definition))
	}
	return result
}

func DefinitionFor(provider string) (ProviderDefinition, error) {
	provider = strings.TrimSpace(provider)
	for _, definition := range Definitions() {
		if definition.ID == provider {
			if err := ValidateDefinition(definition); err != nil {
				return ProviderDefinition{}, err
			}
			if definition.TrustStatus == TrustDisabled {
				return ProviderDefinition{}, fmt.Errorf("kernel provider %q is disabled: %s", provider, definition.DisabledReason)
			}
			if definition.TrustStatus == TrustInvalid {
				return ProviderDefinition{}, fmt.Errorf("kernel provider %q is invalid", provider)
			}
			return definition, nil
		}
	}
	return ProviderDefinition{}, fmt.Errorf("unknown kernel provider %q", provider)
}

func For(provider, version string) (Capabilities, error) {
	definition, err := DefinitionFor(provider)
	if err != nil {
		return Capabilities{}, err
	}
	major, err := majorVersion(version)
	if err != nil {
		return Capabilities{}, err
	}
	if !supportsVersion(definition, strings.TrimSpace(version), major) {
		return Capabilities{}, fmt.Errorf("kernel provider %q does not support Chromium %q", definition.ID, version)
	}
	capabilities := make(map[CapabilityID]CapabilityDeclaration, len(definition.Capabilities))
	for id, declaration := range definition.Capabilities {
		capabilities[id] = declaration
	}
	return Capabilities{
		SchemaVersion: definition.SchemaVersion,
		Provider:      definition.ID,
		Revision:      definition.Revision,
		TrustStatus:   definition.TrustStatus,
		MajorVersion:  major,
		Capabilities:  capabilities,
		Limitations:   append([]string(nil), definition.KnownLimitations...),
	}, nil
}

func ValidateDefinition(definition ProviderDefinition) error {
	if definition.SchemaVersion != ContractSchemaVersion {
		return fmt.Errorf("provider %q uses unsupported contract schema %d", definition.ID, definition.SchemaVersion)
	}
	if definition.Revision < 1 {
		return fmt.Errorf("provider %q requires a positive revision", definition.ID)
	}
	if strings.TrimSpace(definition.ID) == "" || strings.TrimSpace(definition.Name) == "" {
		return fmt.Errorf("provider id and name are required")
	}
	if !validTrust(definition.TrustStatus) {
		return fmt.Errorf("provider %q has invalid trust status %q", definition.ID, definition.TrustStatus)
	}
	if definition.TrustStatus == TrustDisabled && strings.TrimSpace(definition.DisabledReason) == "" {
		return fmt.Errorf("disabled provider %q requires a reason", definition.ID)
	}
	if definition.TrustStatus == TrustReviewed {
		if strings.TrimSpace(definition.SourceURL) == "" || strings.TrimSpace(definition.LicenseSPDX) == "" {
			return fmt.Errorf("reviewed provider %q requires source and license", definition.ID)
		}
		if len(definition.SupportedOS) == 0 || len(definition.SupportedArch) == 0 {
			return fmt.Errorf("reviewed provider %q requires platform and architecture support", definition.ID)
		}
		if len(definition.Versions) == 0 && definition.MinMajor == 0 {
			return fmt.Errorf("reviewed provider %q requires version support", definition.ID)
		}
		if strings.TrimSpace(definition.ExpectedExecutable) == "" || strings.TrimSpace(definition.ProvenanceRequirement) == "" {
			return fmt.Errorf("reviewed provider %q requires executable and provenance rules", definition.ID)
		}
	}
	for id, declaration := range definition.Capabilities {
		if declaration.ID != id {
			return fmt.Errorf("provider %q capability key %q does not match declaration %q", definition.ID, id, declaration.ID)
		}
		if !validCapabilityStatus(declaration.Status) {
			return fmt.Errorf("provider %q capability %q has invalid status %q", definition.ID, id, declaration.Status)
		}
		if definition.TrustStatus != TrustReviewed && (declaration.Status == CapabilityVerified || declaration.Status == CapabilityPartial) {
			return fmt.Errorf("provider %q cannot claim %s capability %q without reviewed trust", definition.ID, declaration.Status, id)
		}
	}
	for _, predecessor := range definition.PredecessorIDs {
		if predecessor == definition.ID {
			return fmt.Errorf("provider %q cannot be its own predecessor", definition.ID)
		}
	}
	return nil
}

func customDefinition() ProviderDefinition {
	return ProviderDefinition{
		SchemaVersion: ContractSchemaVersion,
		Revision:      1,
		ID:            ProviderCustom,
		Name:          "Custom local Chromium",
		Description:   "Locally imported Chromium with generic launch support and no Veilium-reviewed fingerprint claims.",
		TrustStatus:   TrustCustom,
		SupportedOS:   []string{"windows", "linux"},
		SupportedArch: []string{"amd64", "arm64"},
		Versions:      []string{"148.0.0", "144.0.0", "142.0.0"},
		MinMajor:      1,
		Capabilities:  genericCapabilities(CapabilityUnsupported),
		KnownLimitations: []string{
			"custom binaries may launch but do not receive reviewed fingerprint capability claims",
			"language, window size, and WebRTC command-line policy remain unverified until real-browser evidence exists",
		},
		CreatedAt: "2026-07-18T00:00:00Z",
	}
}

func legacyNativeDefinition() ProviderDefinition {
	definition := customDefinition()
	definition.ID = ProviderNative
	definition.Name = "Legacy native Chromium"
	definition.Description = "Compatibility definition for records created before Provider Contract v2."
	definition.TrustStatus = TrustLegacy
	definition.ReplacementID = ProviderCustom
	definition.Capabilities[CapabilityProxyOnlyWebRTC] = declaration(CapabilityProxyOnlyWebRTC, CapabilityUnverified, "native Chromium WebRTC policy has not completed the Phase 4 evidence chain")
	definition.KnownLimitations = append(definition.KnownLimitations, "legacy records are never silently upgraded to reviewed status")
	return definition
}

func legacyPatchedDefinition() ProviderDefinition {
	definition := customDefinition()
	definition.ID = ProviderPatched
	definition.Name = "Legacy patched Chromium"
	definition.Description = "Compatibility definition for pre-v2 patched Chromium records; former boolean claims are retained only as unverified history."
	definition.TrustStatus = TrustLegacy
	definition.MinMajor = 131
	definition.ReplacementID = ""
	definition.Capabilities = map[CapabilityID]CapabilityDeclaration{
		CapabilityPlatformOverride:    declaration(CapabilityPlatformOverride, CapabilityUnverified, "legacy command-line declaration lacks reviewed real-browser evidence"),
		CapabilityBrandOverride:       declaration(CapabilityBrandOverride, CapabilityUnverified, "legacy command-line declaration lacks reviewed real-browser evidence"),
		CapabilityTimezoneOverride:    declaration(CapabilityTimezoneOverride, CapabilityUnverified, "legacy command-line declaration lacks reviewed real-browser evidence"),
		CapabilitySurfaceSeed:         declaration(CapabilitySurfaceSeed, CapabilityUnverified, "legacy surface-seed claim is not reviewed"),
		CapabilitySurfaceControls:     declaration(CapabilitySurfaceControls, CapabilityUnverified, "legacy surface-control claim is not reviewed"),
		CapabilityHardwareConcurrency: declaration(CapabilityHardwareConcurrency, CapabilityUnverified, "legacy CPU override claim is not reviewed"),
		CapabilityDeviceMemory:        declaration(CapabilityDeviceMemory, CapabilityUnsupported, "no verified device-memory parameter contract exists"),
		CapabilityCustomGPU:           declaration(CapabilityCustomGPU, CapabilityUnverified, "legacy GPU metadata claim is not reviewed"),
		CapabilityProxyOnlyWebRTC:     declaration(CapabilityProxyOnlyWebRTC, CapabilityUnverified, "WebRTC policy requires real-browser evidence"),
	}
	definition.KnownLimitations = []string{
		"legacy patched Chromium records may remain readable but advanced fingerprint settings are blocked until a reviewed provider replaces them",
		"the provider name alone cannot establish source, license, binary provenance, or reviewed capability support",
	}
	return definition
}

func genericCapabilities(defaultStatus CapabilityStatus) map[CapabilityID]CapabilityDeclaration {
	return map[CapabilityID]CapabilityDeclaration{
		CapabilityPlatformOverride:    declaration(CapabilityPlatformOverride, defaultStatus, "custom Chromium reports its own platform"),
		CapabilityBrandOverride:       declaration(CapabilityBrandOverride, defaultStatus, "custom Chromium reports its own browser brand"),
		CapabilityTimezoneOverride:    declaration(CapabilityTimezoneOverride, defaultStatus, "custom Chromium uses the host or browser-configured timezone"),
		CapabilitySurfaceSeed:         declaration(CapabilitySurfaceSeed, defaultStatus, "no reviewed surface-seed contract exists"),
		CapabilitySurfaceControls:     declaration(CapabilitySurfaceControls, defaultStatus, "no reviewed surface-control contract exists"),
		CapabilityHardwareConcurrency: declaration(CapabilityHardwareConcurrency, defaultStatus, "no reviewed CPU override contract exists"),
		CapabilityDeviceMemory:        declaration(CapabilityDeviceMemory, defaultStatus, "no reviewed device-memory contract exists"),
		CapabilityCustomGPU:           declaration(CapabilityCustomGPU, defaultStatus, "no reviewed GPU override contract exists"),
		CapabilityProxyOnlyWebRTC:     declaration(CapabilityProxyOnlyWebRTC, CapabilityUnverified, "standard command-line policy is available but not yet verified by the Phase 4 browser harness"),
	}
}

func declaration(id CapabilityID, status CapabilityStatus, limitation string) CapabilityDeclaration {
	return CapabilityDeclaration{ID: id, Status: status, EvidenceRequired: status != CapabilityUnsupported, Limitation: limitation}
}

func supportsVersion(definition ProviderDefinition, version string, major int) bool {
	if definition.MinMajor > 0 && major < definition.MinMajor {
		return false
	}
	if definition.MaxMajor > 0 && major > definition.MaxMajor {
		return false
	}
	if len(definition.Versions) == 0 {
		return true
	}
	for _, supported := range definition.Versions {
		if supported == version {
			return true
		}
		supportedMajor, err := majorVersion(supported)
		if err == nil && supportedMajor == major {
			return true
		}
	}
	return definition.TrustStatus != TrustReviewed
}

func cloneDefinition(definition ProviderDefinition) ProviderDefinition {
	clone := definition
	clone.SupportedOS = append([]string(nil), definition.SupportedOS...)
	clone.SupportedArch = append([]string(nil), definition.SupportedArch...)
	clone.Versions = append([]string(nil), definition.Versions...)
	clone.KnownLimitations = append([]string(nil), definition.KnownLimitations...)
	clone.PredecessorIDs = append([]string(nil), definition.PredecessorIDs...)
	clone.Capabilities = make(map[CapabilityID]CapabilityDeclaration, len(definition.Capabilities))
	for id, declaration := range definition.Capabilities {
		clone.Capabilities[id] = declaration
	}
	return clone
}

func SortedCapabilityIDs(capabilities map[CapabilityID]CapabilityDeclaration) []CapabilityID {
	ids := make([]CapabilityID, 0, len(capabilities))
	for id := range capabilities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func validTrust(value TrustStatus) bool {
	switch value {
	case TrustReviewed, TrustCustom, TrustLegacy, TrustDisabled, TrustInvalid:
		return true
	default:
		return false
	}
}

func validCapabilityStatus(value CapabilityStatus) bool {
	switch value {
	case CapabilityVerified, CapabilityPartial, CapabilityUnsupported, CapabilityUnverified, CapabilityFailed:
		return true
	default:
		return false
	}
}

func majorVersion(version string) (int, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return 0, fmt.Errorf("kernel version is required")
	}
	first, _, _ := strings.Cut(version, ".")
	major, err := strconv.Atoi(first)
	if err != nil || major < 1 {
		return 0, fmt.Errorf("invalid kernel version %q", version)
	}
	return major, nil
}

func parseReviewTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
