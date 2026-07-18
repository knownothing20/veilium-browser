package consistency

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

const (
	minimumWindowWidth  = 400
	minimumWindowHeight = 300
	minimumScaleFactor  = 0.5
	maximumScaleFactor  = 4.0
)

func EffectiveWindow(profile domain.Profile) (WindowSpec, error) {
	fingerprintConfig := profile.Fingerprint
	width := fingerprintConfig.WindowWidth
	height := fingerprintConfig.WindowHeight
	source := WindowExplicit
	if width == 0 && height == 0 {
		width = fingerprintConfig.ScreenWidth
		height = fingerprintConfig.ScreenHeight
		source = WindowLegacyFallback
	} else if width == 0 || height == 0 {
		return WindowSpec{}, fmt.Errorf("window width and height must both be configured")
	}
	scale := fingerprintConfig.DeviceScaleFactor
	if scale == 0 {
		scale = 1
	}
	return WindowSpec{Width: width, Height: height, DeviceScaleFactor: scale, Source: source}, nil
}

func Preflight(profile domain.Profile, capabilities fingerprint.Capabilities, runtimeOS string) (WindowSpec, []Check, error) {
	window, err := EffectiveWindow(profile)
	if err != nil {
		return WindowSpec{}, nil, err
	}
	checks := make([]Check, 0, 8)

	if capabilities.Provider != profile.Kernel.Provider {
		checks = append(checks, failed("provider", capabilities.Provider, profile.Kernel.Provider, "provider-contract-mismatch", "The active Provider contract does not match the profile kernel."))
	} else if capabilities.TrustStatus == fingerprint.TrustDisabled || capabilities.TrustStatus == fingerprint.TrustInvalid {
		checks = append(checks, failed("provider", "launchable Provider", string(capabilities.TrustStatus), "provider-not-launchable", "Disabled or invalid Providers cannot launch."))
	} else if capabilities.TrustStatus != fingerprint.TrustReviewed {
		checks = append(checks, warning("provider", "reviewed", string(capabilities.TrustStatus), "provider-unreviewed", "Custom and legacy Providers cannot produce healthy reviewed identity status."))
	} else {
		checks = append(checks, passed("provider", "reviewed", "reviewed"))
	}

	normalizedRuntimeOS := profilePlatform(runtimeOS)
	if normalizedRuntimeOS == "" {
		checks = append(checks, unknown("platform", profile.Fingerprint.Platform, runtimeOS, "runtime-platform-unknown", "Runtime platform is not recognized by the M4.3 policy."))
	} else if profile.Fingerprint.Platform == "macos" && normalizedRuntimeOS != "macos" {
		checks = append(checks, failed("platform", normalizedRuntimeOS, profile.Fingerprint.Platform, "macos-path-unclaimed", "macOS identity remains unclaimed without a real macOS validation path."))
	} else if capabilities.Supports(fingerprint.CapabilityPlatformOverride) {
		checks = append(checks, passed("platform", profile.Fingerprint.Platform, profile.Fingerprint.Platform))
	} else if profile.Fingerprint.Platform != normalizedRuntimeOS {
		checks = append(checks, failed("platform", normalizedRuntimeOS, profile.Fingerprint.Platform, "host-platform-mismatch", "The selected Provider cannot override platform, so the profile must match the host operating system."))
	} else {
		checks = append(checks, passed("platform", normalizedRuntimeOS, profile.Fingerprint.Platform))
	}

	if profile.Fingerprint.ScreenWidth < 800 || profile.Fingerprint.ScreenHeight < 600 {
		checks = append(checks, failed("screen", "at least 800x600", fmt.Sprintf("%dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight), "screen-too-small", "Declared screen dimensions are below the supported minimum."))
	} else {
		checks = append(checks, passed("screen", fmt.Sprintf("%dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight), fmt.Sprintf("%dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight)))
	}

	windowObserved := fmt.Sprintf("%dx%d", window.Width, window.Height)
	switch {
	case window.Width < minimumWindowWidth || window.Height < minimumWindowHeight:
		checks = append(checks, failed("window", fmt.Sprintf("at least %dx%d", minimumWindowWidth, minimumWindowHeight), windowObserved, "window-too-small", "Window dimensions are below the controlled-browser minimum."))
	case window.Width > profile.Fingerprint.ScreenWidth || window.Height > profile.Fingerprint.ScreenHeight:
		checks = append(checks, failed("window", fmt.Sprintf("within %dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight), windowObserved, "window-exceeds-screen", "The controlled window cannot exceed the declared screen dimensions."))
	case window.Source == WindowLegacyFallback:
		checks = append(checks, warning("window", "explicit window dimensions", windowObserved, "legacy-window-fallback", "This profile predates explicit window fields. Existing screen dimensions are used without rewriting the profile."))
	default:
		checks = append(checks, passed("window", windowObserved, windowObserved))
	}

	if math.IsNaN(window.DeviceScaleFactor) || math.IsInf(window.DeviceScaleFactor, 0) || window.DeviceScaleFactor < minimumScaleFactor || window.DeviceScaleFactor > maximumScaleFactor {
		checks = append(checks, failed("device-scale-factor", fmt.Sprintf("%.1f to %.1f", minimumScaleFactor, maximumScaleFactor), fmt.Sprintf("%.4f", window.DeviceScaleFactor), "invalid-device-scale-factor", "Device scale factor is outside the supported range."))
	} else if profile.Fingerprint.DeviceScaleFactor == 0 {
		checks = append(checks, warning("device-scale-factor", "explicit value", fmt.Sprintf("%.4f", window.DeviceScaleFactor), "native-scale-fallback", "No explicit DPR is stored. The launch uses a compatibility default of 1.0."))
	} else {
		checks = append(checks, passed("device-scale-factor", fmt.Sprintf("%.4f", window.DeviceScaleFactor), fmt.Sprintf("%.4f", window.DeviceScaleFactor)))
	}

	return window, checks, nil
}

func CurrentRuntimeOS() string { return runtime.GOOS }

func profilePlatform(runtimeOS string) string {
	switch strings.ToLower(strings.TrimSpace(runtimeOS)) {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin", "macos":
		return "macos"
	default:
		return ""
	}
}

func passed(id, expected, observed string) Check {
	return Check{ID: id, Status: CheckPassed, Expected: expected, Observed: observed}
}

func warning(id, expected, observed, reason, detail string) Check {
	return Check{ID: id, Status: CheckWarning, Expected: expected, Observed: observed, ReasonCode: reason, Detail: detail}
}

func failed(id, expected, observed, reason, detail string) Check {
	return Check{ID: id, Status: CheckFailed, Expected: expected, Observed: observed, ReasonCode: reason, Detail: detail}
}

func unknown(id, expected, observed, reason, detail string) Check {
	return Check{ID: id, Status: CheckUnknown, Expected: expected, Observed: observed, ReasonCode: reason, Detail: detail}
}
