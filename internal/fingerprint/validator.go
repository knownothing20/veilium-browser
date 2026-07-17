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
	if fp.GPUProfile == "custom" {
		if strings.TrimSpace(fp.GPUVendor) == "" || strings.TrimSpace(fp.GPURenderer) == "" {
			return nil, fmt.Errorf("custom GPU requires both vendor and renderer")
		}
		if !capabilities.CanSetCustomGPU {
			return nil, fmt.Errorf("kernel %s does not support custom GPU metadata", profile.Kernel.Version)
		}
	}

	if fp.Brand != "Chromium" && !capabilities.CanSetBrand {
		return nil, fmt.Errorf("kernel provider cannot override Chromium brand")
	}
	if fp.Seed != "" && !capabilities.CanSeedSurfaces {
		return nil, fmt.Errorf("kernel provider does not support seeded fingerprint surfaces")
	}
	if fp.DeviceMemoryGB != 0 && !capabilities.CanSetDeviceMemory {
		return nil, fmt.Errorf("kernel provider has no verified device-memory parameter contract")
	}

	warnings := make([]string, 0, 3)
	if profile.Kernel.Provider == ProviderNative {
		warnings = append(warnings, "native Chromium reports the host platform and timezone; configured values are descriptive only")
	}
	if fp.WebRTCPolicy == "default" && profile.Proxy.URL != "" {
		warnings = append(warnings, "proxy is configured while WebRTC is unrestricted")
	}
	if fp.Platform == "macos" && fp.Brand == "Edge" {
		warnings = append(warnings, "Edge on macOS is valid but less common than Chrome")
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
