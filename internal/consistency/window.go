package consistency

import (
	"fmt"
	"math"
	"strings"
)

type ObservedWindow struct {
	OuterWidth       int
	OuterHeight      int
	InnerWidth       int
	InnerHeight      int
	ViewportWidth    float64
	ViewportHeight   float64
	ViewportScale    float64
	DevicePixelRatio float64
}

func ParseObservedWindow(value string) (ObservedWindow, error) {
	var observed ObservedWindow
	count, err := fmt.Sscanf(
		strings.TrimSpace(value),
		"outer=%dx%d inner=%dx%d viewport=%fx%f@%f dpr=%f",
		&observed.OuterWidth,
		&observed.OuterHeight,
		&observed.InnerWidth,
		&observed.InnerHeight,
		&observed.ViewportWidth,
		&observed.ViewportHeight,
		&observed.ViewportScale,
		&observed.DevicePixelRatio,
	)
	if err != nil || count != 8 {
		return ObservedWindow{}, fmt.Errorf("parse observed window geometry")
	}
	if observed.OuterWidth < 1 || observed.OuterHeight < 1 || observed.InnerWidth < 1 || observed.InnerHeight < 1 {
		return ObservedWindow{}, fmt.Errorf("observed window dimensions are invalid")
	}
	if observed.InnerWidth > observed.OuterWidth || observed.InnerHeight > observed.OuterHeight {
		return ObservedWindow{}, fmt.Errorf("observed inner window exceeds outer window")
	}
	for _, value := range []float64{observed.ViewportWidth, observed.ViewportHeight, observed.ViewportScale, observed.DevicePixelRatio} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
			return ObservedWindow{}, fmt.Errorf("observed window contains invalid scale or viewport values")
		}
	}
	return observed, nil
}

func WindowChecks(expected WindowSpec, observed ObservedWindow) []Check {
	checks := make([]Check, 0, 4)
	outerExpected := fmt.Sprintf("%dx%d", expected.Width, expected.Height)
	outerObserved := fmt.Sprintf("%dx%d", observed.OuterWidth, observed.OuterHeight)
	if withinInt(observed.OuterWidth, expected.Width, 2) && withinInt(observed.OuterHeight, expected.Height, 2) {
		checks = append(checks, passed("window.outer", outerExpected, outerObserved))
	} else {
		checks = append(checks, failed("window.outer", outerExpected, outerObserved, "outer-window-mismatch", "The real outer browser window does not match the controlled WindowPlan tolerance."))
	}

	viewportObserved := fmt.Sprintf("%.2fx%.2f", observed.ViewportWidth, observed.ViewportHeight)
	if observed.ViewportWidth <= float64(observed.InnerWidth)+2 && observed.ViewportHeight <= float64(observed.InnerHeight)+2 {
		checks = append(checks, passed("window.viewport", "within inner window", viewportObserved))
	} else {
		checks = append(checks, failed("window.viewport", "within inner window", viewportObserved, "viewport-exceeds-inner-window", "The visual viewport cannot exceed the inner browser window."))
	}

	dprExpected := fmt.Sprintf("%.4f", expected.DeviceScaleFactor)
	dprObserved := fmt.Sprintf("%.4f", observed.DevicePixelRatio)
	if withinFloat(observed.DevicePixelRatio, expected.DeviceScaleFactor, 0.05) {
		checks = append(checks, passed("window.dpr", dprExpected, dprObserved))
	} else {
		checks = append(checks, failed("window.dpr", dprExpected, dprObserved, "device-scale-factor-mismatch", "Observed device-pixel ratio differs from the controlled WindowPlan."))
	}

	if observed.ViewportScale >= 0.5 && observed.ViewportScale <= 4 {
		checks = append(checks, passed("window.viewport-scale", "0.5 to 4.0", fmt.Sprintf("%.4f", observed.ViewportScale)))
	} else {
		checks = append(checks, failed("window.viewport-scale", "0.5 to 4.0", fmt.Sprintf("%.4f", observed.ViewportScale), "viewport-scale-out-of-range", "Visual viewport scale is outside the supported range."))
	}
	return checks
}

func withinInt(observed, expected, tolerance int) bool {
	difference := observed - expected
	if difference < 0 {
		difference = -difference
	}
	return difference <= tolerance
}

func withinFloat(observed, expected, tolerance float64) bool {
	return math.Abs(observed-expected) <= tolerance
}
