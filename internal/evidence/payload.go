package evidence

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
)

const PayloadSchemaVersion = 1

type BrowserSubmission struct {
	SchemaVersion int               `json:"schemaVersion"`
	Contexts      []BrowserSnapshot `json:"contexts"`
	Limitations   []string          `json:"limitations,omitempty"`
}

type BrowserSnapshot struct {
	Context             BrowserContext    `json:"context"`
	UserAgent           string            `json:"userAgent"`
	UAPlatform          string            `json:"uaPlatform,omitempty"`
	UABrands            []string          `json:"uaBrands,omitempty"`
	NavigatorPlatform   string            `json:"navigatorPlatform"`
	Language            string            `json:"language"`
	Languages           []string          `json:"languages"`
	Timezone            string            `json:"timezone"`
	HardwareConcurrency int               `json:"hardwareConcurrency,omitempty"`
	Screen              *ScreenSnapshot   `json:"screen,omitempty"`
	Window              *WindowSnapshot   `json:"window,omitempty"`
	WebRTC               *WebRTCSnapshot   `json:"webRtc,omitempty"`
	SurfaceDigests       map[string]string `json:"surfaceDigests,omitempty"`
	Limitations          []string          `json:"limitations,omitempty"`
}

type ScreenSnapshot struct {
	Width       int `json:"width"`
	Height      int `json:"height"`
	AvailWidth  int `json:"availWidth"`
	AvailHeight int `json:"availHeight"`
	ColorDepth  int `json:"colorDepth"`
	PixelDepth  int `json:"pixelDepth"`
}

type WindowSnapshot struct {
	OuterWidth       int     `json:"outerWidth"`
	OuterHeight      int     `json:"outerHeight"`
	InnerWidth       int     `json:"innerWidth"`
	InnerHeight      int     `json:"innerHeight"`
	ViewportWidth    float64 `json:"viewportWidth"`
	ViewportHeight   float64 `json:"viewportHeight"`
	ViewportScale    float64 `json:"viewportScale"`
	DevicePixelRatio float64 `json:"devicePixelRatio"`
}

type WebRTCSnapshot struct {
	Available      bool     `json:"available"`
	CandidateTypes []string `json:"candidateTypes,omitempty"`
	Protocols      []string `json:"protocols,omitempty"`
	UsesMDNS       bool     `json:"usesMdns"`
	GatheringState string   `json:"gatheringState,omitempty"`
}

func (s BrowserSubmission) Validate() error {
	if s.SchemaVersion != PayloadSchemaVersion {
		return fmt.Errorf("unsupported browser evidence payload version %d", s.SchemaVersion)
	}
	if len(s.Contexts) < 1 || len(s.Contexts) > 3 {
		return fmt.Errorf("browser evidence payload must contain one to three contexts")
	}
	seen := make(map[BrowserContext]struct{}, len(s.Contexts))
	for index, snapshot := range s.Contexts {
		if _, exists := seen[snapshot.Context]; exists {
			return fmt.Errorf("duplicate browser context %q", snapshot.Context)
		}
		seen[snapshot.Context] = struct{}{}
		if err := snapshot.Validate(); err != nil {
			return fmt.Errorf("browser context %d: %w", index, err)
		}
	}
	if _, ok := seen[ContextTopLevel]; !ok {
		return fmt.Errorf("top-level browser context is required")
	}
	if len(s.Limitations) > 32 {
		return fmt.Errorf("browser submission contains too many limitations")
	}
	for _, limitation := range s.Limitations {
		if len(limitation) > 512 {
			return fmt.Errorf("browser submission limitation is too long")
		}
	}
	return nil
}

func (s BrowserSnapshot) Validate() error {
	if s.Context != ContextTopLevel && s.Context != ContextIframe && s.Context != ContextWorker {
		return fmt.Errorf("invalid browser context %q", s.Context)
	}
	for label, value := range map[string]string{
		"userAgent":         s.UserAgent,
		"uaPlatform":        s.UAPlatform,
		"navigatorPlatform": s.NavigatorPlatform,
		"language":          s.Language,
		"timezone":          s.Timezone,
	} {
		if len(value) > 1024 {
			return fmt.Errorf("%s exceeds the browser evidence limit", label)
		}
	}
	if strings.TrimSpace(s.UserAgent) == "" || strings.TrimSpace(s.Language) == "" || strings.TrimSpace(s.Timezone) == "" {
		return fmt.Errorf("user agent, language, and timezone are required")
	}
	if len(s.Languages) < 1 || len(s.Languages) > 16 {
		return fmt.Errorf("languages must contain one to sixteen values")
	}
	if len(s.UABrands) > 16 {
		return fmt.Errorf("UA brands contain too many values")
	}
	for _, values := range [][]string{s.Languages, s.UABrands, s.Limitations} {
		for _, value := range values {
			if len(value) > 512 {
				return fmt.Errorf("browser evidence list value is too long")
			}
		}
	}
	if s.HardwareConcurrency < 0 || s.HardwareConcurrency > 1024 {
		return fmt.Errorf("hardware concurrency is outside the evidence range")
	}
	if s.Screen != nil {
		if err := s.Screen.Validate(); err != nil {
			return err
		}
	}
	if s.Window != nil {
		if err := s.Window.Validate(); err != nil {
			return err
		}
	}
	if s.WebRTC != nil {
		if err := s.WebRTC.Validate(); err != nil {
			return err
		}
	}
	if len(s.SurfaceDigests) > 8 {
		return fmt.Errorf("too many surface digests")
	}
	for name, digest := range s.SurfaceDigests {
		if !allowedSurface(name) || !validSHA256(digest) {
			return fmt.Errorf("invalid surface digest %q", name)
		}
	}
	return nil
}

func (s ScreenSnapshot) Validate() error {
	for label, value := range map[string]int{
		"screen width": s.Width, "screen height": s.Height,
		"available width": s.AvailWidth, "available height": s.AvailHeight,
	} {
		if value < 1 || value > 16384 {
			return fmt.Errorf("%s is outside the evidence range", label)
		}
	}
	if s.ColorDepth < 1 || s.ColorDepth > 128 || s.PixelDepth < 1 || s.PixelDepth > 128 {
		return fmt.Errorf("screen color depth is outside the evidence range")
	}
	return nil
}

func (w WindowSnapshot) Validate() error {
	for label, value := range map[string]int{
		"outer width": w.OuterWidth, "outer height": w.OuterHeight,
		"inner width": w.InnerWidth, "inner height": w.InnerHeight,
	} {
		if value < 0 || value > 16384 {
			return fmt.Errorf("%s is outside the evidence range", label)
		}
	}
	for label, value := range map[string]float64{
		"viewport width": w.ViewportWidth, "viewport height": w.ViewportHeight,
		"viewport scale": w.ViewportScale, "device pixel ratio": w.DevicePixelRatio,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 128 {
			return fmt.Errorf("%s is outside the evidence range", label)
		}
	}
	return nil
}

func (w WebRTCSnapshot) Validate() error {
	if len(w.CandidateTypes) > 8 || len(w.Protocols) > 8 {
		return fmt.Errorf("WebRTC evidence contains too many values")
	}
	for _, candidateType := range w.CandidateTypes {
		if candidateType != "host" && candidateType != "srflx" && candidateType != "prflx" && candidateType != "relay" {
			return fmt.Errorf("unsupported WebRTC candidate type %q", candidateType)
		}
	}
	for _, protocol := range w.Protocols {
		if protocol != "udp" && protocol != "tcp" {
			return fmt.Errorf("unsupported WebRTC protocol %q", protocol)
		}
	}
	if len(w.GatheringState) > 64 {
		return fmt.Errorf("WebRTC gathering state is too long")
	}
	return nil
}

func normalizeSubmission(submission BrowserSubmission) BrowserSubmission {
	for index := range submission.Contexts {
		snapshot := &submission.Contexts[index]
		snapshot.Languages = sortedUnique(snapshot.Languages)
		snapshot.UABrands = sortedUnique(snapshot.UABrands)
		snapshot.Limitations = sortedUnique(snapshot.Limitations)
		if snapshot.WebRTC != nil {
			snapshot.WebRTC.CandidateTypes = sortedUnique(snapshot.WebRTC.CandidateTypes)
			snapshot.WebRTC.Protocols = sortedUnique(snapshot.WebRTC.Protocols)
		}
		for name, digest := range snapshot.SurfaceDigests {
			snapshot.SurfaceDigests[name] = strings.ToLower(strings.TrimSpace(digest))
		}
	}
	submission.Limitations = sortedUnique(submission.Limitations)
	sort.Slice(submission.Contexts, func(i, j int) bool {
		return browserContextRank(submission.Contexts[i].Context) < browserContextRank(submission.Contexts[j].Context)
	})
	return submission
}

func browserContextRank(context BrowserContext) int {
	switch context {
	case ContextTopLevel:
		return 0
	case ContextIframe:
		return 1
	case ContextWorker:
		return 2
	default:
		return 3
	}
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func allowedSurface(name string) bool {
	switch name {
	case "canvas", "webgl", "audio", "clientRects":
		return true
	default:
		return false
	}
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
