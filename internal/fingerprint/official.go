package fingerprint

import "github.com/knownothing20/veilium-browser/internal/kernelrelease"

const ProviderOfficial = kernelrelease.ProviderID

func officialDefinition() ProviderDefinition {
	releases, err := kernelrelease.Releases()
	if err != nil || len(releases) != 1 {
		return ProviderDefinition{
			SchemaVersion: ContractSchemaVersion,
			Revision:      1,
			ID:            ProviderOfficial,
			Name:          "Official Chromium Snapshot (invalid catalog)",
			Description:   "The embedded reviewed Chromium catalog could not be loaded.",
			TrustStatus:   TrustInvalid,
			Capabilities:  genericCapabilities(CapabilityUnsupported),
			CreatedAt:     "2026-07-19T00:00:00Z",
		}
	}
	release := releases[0]
	capabilities := genericCapabilities(CapabilityUnsupported)
	capabilities[CapabilityProxyOnlyWebRTC] = declaration(CapabilityProxyOnlyWebRTC, CapabilityUnsupported, "stock Chromium WebRTC policy is not claimed as a reviewed fingerprint capability")
	return ProviderDefinition{
		SchemaVersion:         ContractSchemaVersion,
		Revision:              release.ProviderRevision,
		ID:                    release.ProviderID,
		Name:                  release.Name,
		Description:           "One exact official Chromium Snapshot package reviewed for generic managed launch and runtime Evidence; advanced fingerprint overrides remain unsupported.",
		TrustStatus:           TrustReviewed,
		SourceURL:             release.SourcePageURL,
		LicenseSPDX:           release.LicenseSPDX,
		SupportedOS:           []string{release.Platform},
		SupportedArch:         []string{release.Arch},
		Versions:              []string{release.BrowserVersion},
		ExpectedExecutable:    "chrome.exe",
		ProvenanceRequirement: "exact embedded snapshot revision, archive SHA-256, complete package-tree SHA-256, and executable SHA-256",
		Capabilities:          capabilities,
		KnownLimitations:      append([]string(nil), release.Limitations...),
		CreatedAt:             "2026-07-19T00:00:00Z",
		ReviewedAt:            release.ReviewedAt,
	}
}
