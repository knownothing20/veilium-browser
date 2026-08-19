package fingerprint

import "github.com/knownothing20/veilium-browser/internal/kernelrelease"

const ProviderOfficial = kernelrelease.ProviderID

func officialDefinition() ProviderDefinition {
	releases, err := kernelrelease.Releases()
	if err != nil {
		return invalidOfficialDefinition()
	}
	providerReleases := make([]kernelrelease.Release, 0, len(releases))
	activeRevision := 0
	for _, release := range releases {
		if release.ProviderID != ProviderOfficial {
			continue
		}
		if release.ProviderRevision > activeRevision {
			activeRevision = release.ProviderRevision
			providerReleases = providerReleases[:0]
		}
		if release.ProviderRevision == activeRevision {
			providerReleases = append(providerReleases, release)
		}
	}
	if len(providerReleases) == 0 {
		return invalidOfficialDefinition()
	}
	release := providerReleases[0]
	versions := make([]string, 0, len(providerReleases))
	supportedOS := make([]string, 0, len(providerReleases))
	supportedArch := make([]string, 0, len(providerReleases))
	for _, candidate := range providerReleases {
		versions = appendUnique(versions, candidate.BrowserVersion)
		supportedOS = appendUnique(supportedOS, candidate.Platform)
		supportedArch = appendUnique(supportedArch, candidate.Arch)
	}
	capabilities := genericCapabilities(CapabilityUnsupported)
	capabilities[CapabilityProxyOnlyWebRTC] = declaration(CapabilityProxyOnlyWebRTC, CapabilityUnsupported, "stock Chromium WebRTC policy is not claimed as a reviewed fingerprint capability")
	return ProviderDefinition{
		SchemaVersion:         ContractSchemaVersion,
		Revision:              activeRevision,
		ID:                    release.ProviderID,
		Name:                  release.Name,
		Description:           "Exact official Chromium Snapshot packages reviewed for generic managed launch and runtime Evidence; advanced fingerprint overrides remain unsupported.",
		TrustStatus:           TrustReviewed,
		SourceURL:             release.SourcePageURL,
		LicenseSPDX:           release.LicenseSPDX,
		SupportedOS:           supportedOS,
		SupportedArch:         supportedArch,
		Versions:              versions,
		ExpectedExecutable:    "chrome.exe",
		ProvenanceRequirement: "exact embedded snapshot revision, archive SHA-256, complete package-tree SHA-256, and executable SHA-256",
		Capabilities:          capabilities,
		KnownLimitations:      append([]string(nil), release.Limitations...),
		CreatedAt:             "2026-07-19T00:00:00Z",
		ReviewedAt:            release.ReviewedAt,
	}
}

func invalidOfficialDefinition() ProviderDefinition {
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

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}
