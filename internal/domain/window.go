package domain

import "fmt"

const (
	WindowSourceExplicit       = "explicit"
	WindowSourceLegacyFallback = "legacy-screen-fallback"
)

func EffectiveWindowPlan(config FingerprintConfig) (WindowPlan, error) {
	width := config.WindowWidth
	height := config.WindowHeight
	source := WindowSourceExplicit
	if width == 0 && height == 0 {
		width = config.ScreenWidth
		height = config.ScreenHeight
		source = WindowSourceLegacyFallback
	} else if width == 0 || height == 0 {
		return WindowPlan{}, fmt.Errorf("window width and height must both be configured")
	}
	scale := config.DeviceScaleFactor
	if scale == 0 {
		scale = 1
	}
	return WindowPlan{Width: width, Height: height, DeviceScaleFactor: scale, Source: source}, nil
}
