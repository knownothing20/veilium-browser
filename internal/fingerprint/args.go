package fingerprint

import (
	"fmt"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func BuildArgs(profile domain.Profile) ([]string, error) {
	capabilities, err := For(profile.Kernel.Provider, profile.Kernel.Version)
	if err != nil {
		return nil, err
	}
	fp := profile.Fingerprint
	args := []string{
		fmt.Sprintf("--lang=%s", fp.Language),
		fmt.Sprintf("--window-size=%d,%d", fp.ScreenWidth, fp.ScreenHeight),
	}

	if usesSeededSurfaces(fp) {
		if err := requireVerifiedCapability(capabilities, CapabilitySurfaceSeed, "seeded fingerprint surfaces"); err != nil {
			return nil, err
		}
		args = append(args, fmt.Sprintf("--fingerprint=%s", fp.Seed))
	}
	if capabilities.Supports(CapabilityPlatformOverride) {
		args = append(args, fmt.Sprintf("--fingerprint-platform=%s", fp.Platform))
	}
	if fp.Brand != "Chromium" {
		if err := requireVerifiedCapability(capabilities, CapabilityBrandOverride, "browser-brand override"); err != nil {
			return nil, err
		}
		args = append(args, fmt.Sprintf("--fingerprint-brand=%s", fp.Brand))
	}
	if capabilities.Supports(CapabilityTimezoneOverride) {
		args = append(args, fmt.Sprintf("--timezone=%s", fp.Timezone))
	}
	if fp.HardwareConcurrency > 0 {
		if err := requireVerifiedCapability(capabilities, CapabilityHardwareConcurrency, "hardware-concurrency override"); err != nil {
			return nil, err
		}
		args = append(args, fmt.Sprintf("--fingerprint-hardware-concurrency=%d", fp.HardwareConcurrency))
	}
	if fp.DeviceMemoryGB > 0 {
		if err := requireVerifiedCapability(capabilities, CapabilityDeviceMemory, "device-memory override"); err != nil {
			return nil, err
		}
	}
	if fp.WebRTCPolicy == "proxy-only" {
		args = append(args, "--disable-non-proxied-udp")
	} else if fp.WebRTCPolicy == "disabled" {
		args = append(args, "--webrtc-ip-handling-policy=disable_non_proxied_udp")
	}

	if capabilities.Supports(CapabilitySurfaceControls) {
		disabled := make([]string, 0, 5)
		if fp.FontMode == "native" {
			disabled = append(disabled, "font")
		}
		if fp.AudioMode == "native" {
			disabled = append(disabled, "audio")
		}
		if fp.CanvasMode == "native" {
			disabled = append(disabled, "canvas")
		}
		if fp.ClientRectsMode == "native" {
			disabled = append(disabled, "clientrects")
		}
		if fp.GPUProfile == "native" {
			disabled = append(disabled, "gpu")
		}
		if len(disabled) > 0 {
			args = append(args, "--disable-spoofing="+strings.Join(disabled, ","))
		}
	}
	if fp.GPUProfile == "custom" {
		if err := requireVerifiedCapability(capabilities, CapabilityCustomGPU, "custom GPU metadata"); err != nil {
			return nil, err
		}
		args = append(args,
			"--fingerprint-gpu-vendor="+fp.GPUVendor,
			"--fingerprint-gpu-renderer="+fp.GPURenderer,
		)
	}
	return args, nil
}

func requireVerifiedCapability(capabilities Capabilities, id CapabilityID, label string) error {
	state := capabilities.State(id)
	if state == CapabilityVerified || state == CapabilityPartial {
		return nil
	}
	return fmt.Errorf("kernel provider %q cannot apply %s: capability is %s", capabilities.Provider, label, state)
}

func usesSeededSurfaces(fp domain.FingerprintConfig) bool {
	return fp.CanvasMode == "seeded" || fp.AudioMode == "seeded" || fp.FontMode == "seeded" || fp.ClientRectsMode == "seeded"
}
