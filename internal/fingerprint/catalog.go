package fingerprint

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ProviderNative  = "native-chromium"
	ProviderPatched = "patched-chromium"
)

type Capabilities struct {
	Provider                string `json:"provider"`
	MajorVersion            int    `json:"majorVersion"`
	CanSetPlatform          bool   `json:"canSetPlatform"`
	CanSetBrand             bool   `json:"canSetBrand"`
	CanSetTimezone          bool   `json:"canSetTimezone"`
	CanSeedSurfaces         bool   `json:"canSeedSurfaces"`
	CanDisableSurfaces      bool   `json:"canDisableSurfaces"`
	CanSetHardwareThreads   bool   `json:"canSetHardwareThreads"`
	CanSetDeviceMemory      bool   `json:"canSetDeviceMemory"`
	CanSetCustomGPU         bool   `json:"canSetCustomGpu"`
	SupportsProxyOnlyWebRTC bool   `json:"supportsProxyOnlyWebRtc"`
}

func Providers() []string {
	return []string{ProviderNative, ProviderPatched}
}

func For(provider, version string) (Capabilities, error) {
	major, err := majorVersion(version)
	if err != nil {
		return Capabilities{}, err
	}

	switch provider {
	case ProviderNative:
		return Capabilities{
			Provider:                provider,
			MajorVersion:            major,
			SupportsProxyOnlyWebRTC: true,
		}, nil
	case ProviderPatched:
		if major < 131 {
			return Capabilities{}, fmt.Errorf("patched-chromium requires Chromium 131 or newer")
		}
		return Capabilities{
			Provider:                provider,
			MajorVersion:            major,
			CanSetPlatform:          true,
			CanSetBrand:             true,
			CanSetTimezone:          true,
			CanSeedSurfaces:         true,
			CanDisableSurfaces:      major >= 144,
			CanSetHardwareThreads:   true,
			CanSetDeviceMemory:      false,
			CanSetCustomGPU:         major >= 139 && major < 144,
			SupportsProxyOnlyWebRTC: true,
		}, nil
	default:
		return Capabilities{}, fmt.Errorf("unknown kernel provider %q", provider)
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
