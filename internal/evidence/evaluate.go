package evidence

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

type Evaluation struct {
	Status       RunStatus
	Observations []Observation
	Limitations  []string
}

func RequestedSurfaces(capabilities fingerprint.Capabilities) []string {
	requested := make([]string, 0, 4)
	if relevantCapability(capabilities.State(fingerprint.CapabilitySurfaceSeed)) || relevantCapability(capabilities.State(fingerprint.CapabilitySurfaceControls)) {
		requested = append(requested, "canvas", "audio", "clientRects")
	}
	if relevantCapability(capabilities.State(fingerprint.CapabilityCustomGPU)) || len(requested) > 0 {
		requested = append(requested, "webgl")
	}
	sort.Strings(requested)
	return requested
}

func Evaluate(profile domain.Profile, capabilities fingerprint.Capabilities, submission BrowserSubmission) (Evaluation, error) {
	submission = normalizeSubmission(submission)
	if err := submission.Validate(); err != nil {
		return Evaluation{}, err
	}
	if capabilities.Provider != profile.Kernel.Provider {
		return Evaluation{}, fmt.Errorf("provider contract %q does not match profile provider %q", capabilities.Provider, profile.Kernel.Provider)
	}
	contexts := make(map[BrowserContext]BrowserSnapshot, len(submission.Contexts))
	for _, snapshot := range submission.Contexts {
		contexts[snapshot.Context] = snapshot
	}
	top, ok := contexts[ContextTopLevel]
	if !ok {
		return Evaluation{}, fmt.Errorf("top-level evidence context is required")
	}

	evaluation := Evaluation{
		Status:      RunPassed,
		Limitations: append([]string(nil), submission.Limitations...),
	}
	for _, snapshot := range submission.Contexts {
		evaluation.Observations = append(evaluation.Observations, evaluateSnapshot(profile, capabilities, snapshot)...)
		evaluation.Limitations = append(evaluation.Limitations, snapshot.Limitations...)
	}
	for _, context := range []BrowserContext{ContextIframe, ContextWorker} {
		snapshot, exists := contexts[context]
		if !exists {
			evaluation.Limitations = append(evaluation.Limitations, string(context)+":missing")
			evaluation.Status = strongerRunStatus(evaluation.Status, RunIncomplete)
			continue
		}
		evaluation.Observations = append(evaluation.Observations, compareContexts(top, snapshot)...)
	}
	evaluation.Observations = append(evaluation.Observations, evaluateSurfaces(capabilities, contexts)...)

	for _, observation := range evaluation.Observations {
		switch observation.Status {
		case ObservationFailed:
			evaluation.Status = RunFailed
		case ObservationUnavailable:
			evaluation.Status = strongerRunStatus(evaluation.Status, RunIncomplete)
		case ObservationPartial:
			evaluation.Status = strongerRunStatus(evaluation.Status, RunPartial)
		}
	}
	if capabilities.TrustStatus != fingerprint.TrustReviewed {
		evaluation.Status = strongerRunStatus(evaluation.Status, RunPartial)
		evaluation.Limitations = append(evaluation.Limitations, "provider-trust:"+string(capabilities.TrustStatus))
	}
	evaluation.Limitations = sortedUnique(evaluation.Limitations)
	return evaluation, nil
}

func evaluateSnapshot(profile domain.Profile, capabilities fingerprint.Capabilities, snapshot BrowserSnapshot) []Observation {
	prefix := string(snapshot.Context) + "."
	observations := make([]Observation, 0, 10)
	observations = append(observations,
		matchObservation(prefix+"language", snapshot.Context, profile.Fingerprint.Language, snapshot.Language, "language-mismatch"),
		matchObservation(prefix+"timezone", snapshot.Context, profile.Fingerprint.Timezone, snapshot.Timezone, "timezone-mismatch"),
	)

	platformState := capabilities.State(fingerprint.CapabilityPlatformOverride)
	platformObserved := firstNonEmpty(snapshot.UAPlatform, snapshot.NavigatorPlatform)
	platformMatch := platformMatches(profile.Fingerprint.Platform, snapshot.UAPlatform, snapshot.NavigatorPlatform)
	observations = append(observations, capabilityObservation(prefix+"platform", snapshot.Context, fingerprint.CapabilityPlatformOverride, platformState, profile.Fingerprint.Platform, platformObserved, platformMatch, "platform-mismatch"))

	brandState := capabilities.State(fingerprint.CapabilityBrandOverride)
	brandObserved := strings.Join(snapshot.UABrands, ",")
	if brandObserved == "" {
		brandObserved = snapshot.UserAgent
	}
	brandMatch := browserBrandMatches(profile.Fingerprint.Brand, profile.Kernel.Version, snapshot)
	observations = append(observations, capabilityObservation(prefix+"browserBrand", snapshot.Context, fingerprint.CapabilityBrandOverride, brandState, profile.Fingerprint.Brand+" "+majorVersion(profile.Kernel.Version), brandObserved, brandMatch, "browser-brand-mismatch"))

	if profile.Fingerprint.HardwareConcurrency > 0 {
		observations = append(observations, capabilityObservation(
			prefix+"hardwareConcurrency",
			snapshot.Context,
			fingerprint.CapabilityHardwareConcurrency,
			capabilities.State(fingerprint.CapabilityHardwareConcurrency),
			strconv.Itoa(profile.Fingerprint.HardwareConcurrency),
			strconv.Itoa(snapshot.HardwareConcurrency),
			profile.Fingerprint.HardwareConcurrency == snapshot.HardwareConcurrency,
			"hardware-concurrency-mismatch",
		))
	} else {
		observations = append(observations, Observation{ID: prefix + "hardwareConcurrency", Context: snapshot.Context, Status: ObservationPartial, Observed: strconv.Itoa(snapshot.HardwareConcurrency), ReasonCode: "profile-unconfigured", Detail: "Observed value is recorded but the profile does not declare an override."})
	}

	observations = append(observations, Observation{ID: prefix + "userAgent", Context: snapshot.Context, Status: ObservationPassed, Observed: snapshot.UserAgent})
	if len(snapshot.UABrands) > 0 || snapshot.UAPlatform != "" {
		observations = append(observations, Observation{ID: prefix + "uaClientHints", Context: snapshot.Context, Status: ObservationPassed, Observed: strings.Join(append(snapshot.UABrands, snapshot.UAPlatform), ",")})
	} else {
		observations = append(observations, Observation{ID: prefix + "uaClientHints", Context: snapshot.Context, Status: ObservationUnavailable, ReasonCode: "api-unavailable", Detail: "UA Client Hints were not exposed in this context."})
	}

	if snapshot.Screen != nil {
		expected := fmt.Sprintf("%dx%d", profile.Fingerprint.ScreenWidth, profile.Fingerprint.ScreenHeight)
		observed := fmt.Sprintf("%dx%d", snapshot.Screen.Width, snapshot.Screen.Height)
		observations = append(observations, matchObservation(prefix+"screen", snapshot.Context, expected, observed, "screen-mismatch"))
	} else if snapshot.Context != ContextWorker {
		observations = append(observations, Observation{ID: prefix + "screen", Context: snapshot.Context, Status: ObservationUnavailable, ReasonCode: "api-unavailable"})
	}
	if snapshot.Window != nil {
		observed := fmt.Sprintf("outer=%dx%d inner=%dx%d viewport=%.2fx%.2f@%.4f dpr=%.4f", snapshot.Window.OuterWidth, snapshot.Window.OuterHeight, snapshot.Window.InnerWidth, snapshot.Window.InnerHeight, snapshot.Window.ViewportWidth, snapshot.Window.ViewportHeight, snapshot.Window.ViewportScale, snapshot.Window.DevicePixelRatio)
		observations = append(observations, Observation{ID: prefix + "window", Context: snapshot.Context, Status: ObservationPartial, Observed: observed, ReasonCode: "consistency-policy-deferred", Detail: "M4.2 records window values; final consistency policy is defined in M4.3."})
	}
	if snapshot.WebRTC != nil {
		observed := fmt.Sprintf("available=%t types=%s protocols=%s mdns=%t state=%s", snapshot.WebRTC.Available, strings.Join(snapshot.WebRTC.CandidateTypes, ","), strings.Join(snapshot.WebRTC.Protocols, ","), snapshot.WebRTC.UsesMDNS, snapshot.WebRTC.GatheringState)
		observations = append(observations, Observation{ID: prefix + "webRtcPolicy", Context: snapshot.Context, CapabilityID: fingerprint.CapabilityProxyOnlyWebRTC, Status: ObservationPartial, Expected: profile.Fingerprint.WebRTCPolicy, Observed: observed, ReasonCode: "external-probe-deferred", Detail: "Local indicators do not prove external WebRTC leak behavior; M4.4 owns that probe."})
	} else if snapshot.Context != ContextWorker {
		observations = append(observations, Observation{ID: prefix + "webRtcPolicy", Context: snapshot.Context, CapabilityID: fingerprint.CapabilityProxyOnlyWebRTC, Status: ObservationUnavailable, Expected: profile.Fingerprint.WebRTCPolicy, ReasonCode: "api-unavailable"})
	}
	return observations
}

func compareContexts(top, other BrowserSnapshot) []Observation {
	prefix := string(other.Context) + ".consistency."
	return []Observation{
		matchObservation(prefix+"userAgent", other.Context, top.UserAgent, other.UserAgent, "context-user-agent-mismatch"),
		matchObservation(prefix+"language", other.Context, top.Language, other.Language, "context-language-mismatch"),
		matchObservation(prefix+"timezone", other.Context, top.Timezone, other.Timezone, "context-timezone-mismatch"),
		matchObservation(prefix+"hardwareConcurrency", other.Context, strconv.Itoa(top.HardwareConcurrency), strconv.Itoa(other.HardwareConcurrency), "context-hardware-concurrency-mismatch"),
	}
}

func evaluateSurfaces(capabilities fingerprint.Capabilities, contexts map[BrowserContext]BrowserSnapshot) []Observation {
	requested := RequestedSurfaces(capabilities)
	if len(requested) == 0 {
		return nil
	}
	top := contexts[ContextTopLevel]
	frame, hasFrame := contexts[ContextIframe]
	observations := make([]Observation, 0, len(requested))
	for _, surface := range requested {
		capabilityID := fingerprint.CapabilitySurfaceSeed
		if surface == "webgl" {
			capabilityID = fingerprint.CapabilityCustomGPU
		}
		topDigest := top.SurfaceDigests[surface]
		frameDigest := frame.SurfaceDigests[surface]
		observation := Observation{ID: "surface." + surface, Context: ContextTopLevel, CapabilityID: capabilityID, Expected: topDigest, Observed: frameDigest}
		switch {
		case topDigest == "":
			observation.Status = ObservationUnavailable
			observation.ReasonCode = "top-level-surface-unavailable"
		case !hasFrame || frameDigest == "":
			observation.Status = ObservationUnavailable
			observation.ReasonCode = "iframe-surface-unavailable"
		case topDigest != frameDigest:
			observation.Status = ObservationFailed
			observation.ReasonCode = "surface-context-mismatch"
		default:
			observation.Status = ObservationPassed
		}
		observations = append(observations, observation)
	}
	return observations
}

func capabilityObservation(id string, context BrowserContext, capabilityID fingerprint.CapabilityID, state fingerprint.CapabilityStatus, expected, observed string, matches bool, mismatchCode string) Observation {
	observation := Observation{ID: id, Context: context, CapabilityID: capabilityID, Expected: expected, Observed: observed}
	if !relevantCapability(state) {
		observation.Status = ObservationPartial
		observation.ReasonCode = "provider-capability-" + string(state)
		observation.Detail = "Observed value cannot establish support because the provider capability is not reviewed."
		return observation
	}
	if matches {
		observation.Status = ObservationPassed
		if state == fingerprint.CapabilityPartial {
			observation.Status = ObservationPartial
			observation.ReasonCode = "provider-capability-partial"
		}
		return observation
	}
	observation.Status = ObservationFailed
	observation.ReasonCode = mismatchCode
	return observation
}

func matchObservation(id string, context BrowserContext, expected, observed, mismatchCode string) Observation {
	observation := Observation{ID: id, Context: context, Expected: expected, Observed: observed}
	if expected == observed {
		observation.Status = ObservationPassed
	} else {
		observation.Status = ObservationFailed
		observation.ReasonCode = mismatchCode
	}
	return observation
}

func relevantCapability(state fingerprint.CapabilityStatus) bool {
	return state == fingerprint.CapabilityVerified || state == fingerprint.CapabilityPartial || state == fingerprint.CapabilityUnverified
}

func strongerRunStatus(current, candidate RunStatus) RunStatus {
	rank := map[RunStatus]int{RunPassed: 0, RunPartial: 1, RunIncomplete: 2, RunFailed: 3}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func platformMatches(expected, uaPlatform, navigatorPlatform string) bool {
	value := strings.ToLower(uaPlatform + " " + navigatorPlatform)
	switch expected {
	case "windows":
		return strings.Contains(value, "windows") || strings.Contains(value, "win32") || strings.Contains(value, "win64")
	case "linux":
		return strings.Contains(value, "linux")
	case "macos":
		return strings.Contains(value, "mac")
	default:
		return false
	}
}

func browserBrandMatches(expected, version string, snapshot BrowserSnapshot) bool {
	major := majorVersion(version)
	brands := strings.ToLower(strings.Join(snapshot.UABrands, " "))
	userAgent := strings.ToLower(snapshot.UserAgent)
	versionMatch := major == "" || strings.Contains(brands, "/"+major) || strings.Contains(userAgent, "/"+major)
	if !versionMatch {
		return false
	}
	switch expected {
	case "Chromium":
		return strings.Contains(brands, "chromium") || strings.Contains(userAgent, "chrome/")
	case "Chrome":
		return strings.Contains(brands, "google chrome") || strings.Contains(userAgent, "chrome/")
	case "Edge":
		return strings.Contains(brands, "microsoft edge") || strings.Contains(userAgent, "edg/")
	case "Opera":
		return strings.Contains(brands, "opera") || strings.Contains(userAgent, "opr/")
	case "Vivaldi":
		return strings.Contains(brands, "vivaldi") || strings.Contains(userAgent, "vivaldi/")
	default:
		return false
	}
}

func majorVersion(version string) string {
	major, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	return major
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
