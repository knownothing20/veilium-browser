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

	if capabilities.CanSeedSurfaces {
		args = append(args, fmt.Sprintf("--fingerprint=%s", fp.Seed))
	}
	if capabilities.CanSetPlatform {
		args = append(args, fmt.Sprintf("--fingerprint-platform=%s", fp.Platform))
	}
	if capabilities.CanSetBrand {
		args = append(args, fmt.Sprintf("--fingerprint-brand=%s", fp.Brand))
	}
	if capabilities.CanSetTimezone {
		args = append(args, fmt.Sprintf("--timezone=%s", fp.Timezone))
	}
	if capabilities.CanSetHardwareThreads && fp.HardwareConcurrency > 0 {
		args = append(args, fmt.Sprintf("--fingerprint-hardware-concurrency=%d", fp.HardwareConcurrency))
	}
	if fp.WebRTCPolicy == "proxy-only" {
		args = append(args, "--disable-non-proxied-udp")
	} else if fp.WebRTCPolicy == "disabled" {
		args = append(args, "--webrtc-ip-handling-policy=disable_non_proxied_udp")
	}

	if capabilities.CanDisableSurfaces {
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
	if capabilities.CanSetCustomGPU && fp.GPUProfile == "custom" {
		args = append(args,
			"--fingerprint-gpu-vendor="+fp.GPUVendor,
			"--fingerprint-gpu-renderer="+fp.GPURenderer,
		)
	}
	return args, nil
}
