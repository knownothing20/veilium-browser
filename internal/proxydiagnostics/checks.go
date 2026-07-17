package proxydiagnostics

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/proxy"
)

func dnsCheck(route proxy.Route) Check {
	if route.DisplayURL == "direct://" {
		return Check{
			ID: "dns_route", Label: "DNS route", Status: CheckSkipped,
			Detail: "Direct baseline uses the operating-system resolver.",
		}
	}
	return Check{
		ID: "dns_route", Label: "DNS route", Status: CheckPass,
		Detail: "Target hostnames are passed to the upstream HTTP/HTTPS/SOCKS5 proxy instead of being resolved by the diagnostic client.",
	}
}

func webRTCCheck(profile domain.Profile, route proxy.Route) Check {
	if route.DisplayURL == "direct://" {
		return Check{
			ID: "webrtc_policy", Label: "WebRTC policy", Status: CheckSkipped,
			Detail: "Direct baseline has no proxy-leak comparison.",
		}
	}
	switch profile.Fingerprint.WebRTCPolicy {
	case "disabled":
		return Check{ID: "webrtc_policy", Label: "WebRTC policy", Status: CheckPass, Detail: "WebRTC is disabled for this profile."}
	case "proxy-only":
		return Check{ID: "webrtc_policy", Label: "WebRTC policy", Status: CheckPass, Detail: "Profile policy restricts WebRTC to the proxy route."}
	default:
		return Check{ID: "webrtc_policy", Label: "WebRTC policy", Status: CheckWarn, Detail: "WebRTC uses the default network policy and may expose a local or direct candidate."}
	}
}

func finish(report Report, completedAt time.Time) Report {
	report.CompletedAt = completedAt
	report.Status = StatusHealthy
	for _, check := range report.Checks {
		if check.Status == CheckFail {
			report.Status = StatusFailed
			return report
		}
		if check.Status == CheckWarn {
			report.Status = StatusDegraded
		}
	}
	return report
}

func routeKind(route proxy.Route) string {
	if route.DisplayURL == "direct://" {
		return "direct"
	}
	if route.RequiresBridge {
		return route.BridgeKind
	}
	parsed, err := url.Parse(route.BrowserURL)
	if err != nil {
		return "proxy"
	}
	return strings.ToLower(parsed.Scheme)
}

func routeDetail(route proxy.Route) string {
	if route.DisplayURL == "direct://" {
		return "Direct connection baseline."
	}
	if route.RequiresBridge {
		return fmt.Sprintf("Credential-backed proxy through %s.", route.BridgeKind)
	}
	return "Unauthenticated proxy route."
}

func validateProbeURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid proxy diagnostic probe URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("proxy diagnostic probe URL must not contain credentials")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("proxy diagnostic probe must use HTTPS unless it is a loopback test endpoint")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sanitize(value string, material credential.Material) string {
	for _, secret := range []string{material.Secret, material.Username} {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	return value
}
