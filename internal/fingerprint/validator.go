package fingerprint

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

var languagePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)

func Validate(profile domain.Profile) ([]string, error) {
	if strings.TrimSpace(profile.Name) == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	if strings.TrimSpace(profile.Kernel.Executable) == "" {
		return nil, fmt.Errorf("kernel executable is required")
	}
	capabilities, err := For(profile.Kernel.Provider, profile.Kernel.Version)
	if err != nil {
		return nil, err
	}

	fp := profile.Fingerprint
	if !oneOf(fp.Platform, "windows", "linux", "macos") {
		return nil, fmt.Errorf("unsupported platform %q", fp.Platform)
	}
	if !oneOf(fp.Brand, "Chromium", "Chrome", "Edge", "Opera", "Vivaldi") {
		return nil, fmt.Errorf("unsupported Chromium brand %q", fp.Brand)
	}
	if !languagePattern.MatchString(fp.Language) {
		return nil, fmt.Errorf("language must look like en-US or zh-CN")
	}
	if _, err := time.LoadLocation(fp.Timezone); err != nil {
		return nil, fmt.Errorf("invalid IANA timezone %q", fp.Timezone)
	}
	if fp.ScreenWidth < 800 || fp.ScreenWidth > 7680 || fp.ScreenHeight < 600 || fp.ScreenHeight > 4320 {
		return nil, fmt.Errorf("screen dimensions are outside the supported range")
	}
	if fp.HardwareConcurrency != 0 && (fp.HardwareConcurrency < 2 || fp.HardwareConcurrency > 128) {
		return nil, fmt.Errorf("hardwareConcurrency must be between 2 and 128")
	}
	if fp.DeviceMemoryGB != 0 && !oneOfInt(fp.DeviceMemoryGB, 2, 4, 8, 16, 32, 64) {
		return nil, fmt.Errorf("deviceMemoryGb must be one of 2, 4, 8, 16, 32, 64")
	}
	if !oneOf(fp.WebRTCPolicy, "default", "proxy-only", "disabled") {
		return nil, fmt.Errorf("unsupported WebRTC policy %q", fp.WebRTCPolicy)
	}
	for label, mode := range map[string]string{
		"canvasMode":      fp.CanvasMode,
		"audioMode":       fp.AudioMode,
		"fontMode":        fp.FontMode,
		"clientRectsMode": fp.ClientRectsMode,
	} {
		if !oneOf(mode, "seeded", "native") {
			return nil, fmt.Errorf("%s must be seeded or native", label)
		}
	}
	if !oneOf(fp.GPUProfile, "auto", "native", "custom") {
		return nil, fmt.Errorf("gpuProfile must be auto, native, or custom")
	}

	if fp.Brand != "Chromium" {
		if err := requireVerifiedCapability(capabilities, CapabilityBrandOverride, "browser-brand override"); err != nil {
			return nil, err
		}
	}
	if fp.Seed != "" || usesSeededSurfaces(fp) {
		if strings.TrimSpace(fp.Seed) == "" {
			return nil, fmt.Errorf("seeded fingerprint surfaces require a non-empty seed")
		}
		if err := requireVerifiedCapability(capabilities, CapabilitySurfaceSeed, "seeded fingerprint surfaces"); err != nil {
			return nil, err
		}
	}
	if fp.HardwareConcurrency != 0 {
		if err := requireVerifiedCapability(capabilities, CapabilityHardwareConcurrency, "hardware-concurrency override"); err != nil {
			return nil, err
		}
	}
	if fp.DeviceMemoryGB != 0 {
		if err := requireVerifiedCapability(capabilities, CapabilityDeviceMemory, "device-memory override"); err != nil {
			return nil, err
		}
	}
	if fp.GPUProfile == "custom" {
		if strings.TrimSpace(fp.GPUVendor) == "" || strings.TrimSpace(fp.GPURenderer) == "" {
			return nil, fmt.Errorf("custom GPU requires both vendor and renderer")
		}
		if err := requireVerifiedCapability(capabilities, CapabilityCustomGPU, "custom GPU metadata"); err != nil {
			return nil, err
		}
	}

	warnings := make([]string, 0, 6)
	if capabilities.TrustStatus == TrustCustom {
		warnings = append(warnings, "custom Chromium may launch with generic settings but has no Veilium-reviewed fingerprint claims")
	}
	if capabilities.TrustStatus == TrustLegacy {
		warnings = append(warnings, "legacy provider record remains readable but is not silently upgraded to reviewed status")
	}
	if !capabilities.Supports(CapabilityPlatformOverride) {
		warnings = append(warnings, "configured platform is descriptive because the selected provider cannot apply a reviewed platform override")
	}
	if !capabilities.Supports(CapabilityTimezoneOverride) {
		warnings = append(warnings, "configured timezone is descriptive because the selected provider cannot apply a reviewed timezone override")
	}
	if fp.WebRTCPolicy != "default" && capabilities.State(CapabilityProxyOnlyWebRTC) == CapabilityUnverified {
		warnings = append(warnings, "WebRTC command-line policy is configured but remains unverified until the Phase 4 browser evidence harness runs")
	}
	if fp.WebRTCPolicy == "default" && profile.Proxy.URL != "" && profile.Proxy.URL != "direct://" {
		warnings = append(warnings, "proxy is configured while WebRTC is unrestricted")
	}
	if fp.Platform == "macos" {
		warnings = append(warnings, "macOS capability support remains unclaimed until a real macOS validation path exists")
	}
	return warnings, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func oneOfInt(value int, allowed ...int) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
