package proxydiagnostics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/proxy"
	"github.com/knownothing20/veilium-browser/internal/proxybridge"
	xproxy "golang.org/x/net/proxy"
)

func (r *Runner) Run(ctx context.Context, request Request) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	profileID := strings.TrimSpace(request.Profile.ID)
	profileName := strings.TrimSpace(request.Profile.Name)
	if profileID == "" || profileName == "" {
		return Report{}, fmt.Errorf("profile id and name are required")
	}
	if err := r.reserve(profileID); err != nil {
		return Report{}, err
	}
	defer r.release(profileID)

	report := Report{
		ProfileID:   profileID,
		ProfileName: profileName,
		Status:      StatusHealthy,
		StartedAt:   r.now().UTC(),
		Limitations: []string{
			"DNS is assessed from the proxy protocol path; Veilium does not run a third-party DNS-leak domain in this check.",
			"WebRTC is a profile-policy audit, not a live browser STUN packet capture.",
		},
	}
	route, err := proxy.Resolve(request.Profile.Proxy.URL, request.Profile.Proxy.CredentialRef)
	if err != nil {
		return Report{}, err
	}
	report.ProxyDisplay = route.DisplayURL
	report.RouteKind = routeKind(route)
	report.Checks = append(report.Checks, Check{
		ID: "route", Label: "Route configuration", Status: CheckPass, Detail: routeDetail(route),
	})

	client, bridge, bridgeCheck, err := r.clientFor(ctx, route, request.Material)
	if bridgeCheck.ID != "" {
		report.Checks = append(report.Checks, bridgeCheck)
	}
	if bridge != nil {
		defer bridge.Close()
		report.BridgeKind = bridge.Kind()
	}
	if err != nil {
		report.Checks = append(report.Checks, Check{
			ID: "connectivity", Label: "Proxy connectivity", Status: CheckFail,
			Detail: sanitize(err.Error(), request.Material),
		})
		report.Checks = append(report.Checks, dnsCheck(route), webRTCCheck(request.Profile, route))
		return finish(report, r.now().UTC()), nil
	}

	probeContext, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()
	probe, err := runProbe(probeContext, client, r.config.ProbeURL)
	if err != nil {
		report.Checks = append(report.Checks,
			Check{
				ID: "connectivity", Label: "Proxy connectivity", Status: CheckFail,
				Detail: sanitize(err.Error(), request.Material),
			},
			Check{
				ID: "exit_ip", Label: "Exit IP", Status: CheckFail,
				Detail: "The public exit address could not be verified.",
			},
		)
	} else {
		report.ExitIP = probe.IP
		report.FirstByteLatencyMS = probe.FirstByte.Milliseconds()
		report.TotalLatencyMS = probe.Total.Milliseconds()
		report.Checks = append(report.Checks,
			Check{
				ID: "connectivity", Label: "Proxy connectivity", Status: CheckPass,
				Detail:    fmt.Sprintf("HTTPS probe completed with status %d.", probe.StatusCode),
				LatencyMS: report.TotalLatencyMS,
			},
			Check{
				ID: "exit_ip", Label: "Exit IP", Status: CheckPass,
				Detail: "Public address observed through the selected route: " + probe.IP,
			},
		)
	}
	report.Checks = append(report.Checks, dnsCheck(route), webRTCCheck(request.Profile, route))
	return finish(report, r.now().UTC()), nil
}

func (r *Runner) clientFor(
	ctx context.Context,
	route proxy.Route,
	material credential.Material,
) (*http.Client, proxybridge.Instance, Check, error) {
	if route.DisplayURL == "direct://" {
		return diagnosticClient(nil), nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckSkipped,
			Detail: "Direct baseline does not use a proxy bridge.",
		}, nil
	}
	if route.RequiresBridge {
		return r.bridgedClient(ctx, route, material)
	}

	parsed, err := url.Parse(route.BrowserURL)
	if err != nil {
		return nil, nil, Check{}, err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return diagnosticClient(parsed), nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckSkipped,
			Detail: "Unauthenticated HTTP proxy is tested through the native proxy route.",
		}, nil
	case "socks5":
		dialer, err := xproxy.SOCKS5(
			"tcp",
			parsed.Host,
			nil,
			&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second},
		)
		if err != nil {
			return nil, nil, Check{}, err
		}
		transport := baseTransport()
		if contextual, ok := dialer.(xproxy.ContextDialer); ok {
			transport.DialContext = contextual.DialContext
		} else {
			transport.DialContext = func(
				_ context.Context,
				network,
				address string,
			) (net.Conn, error) {
				return dialer.Dial(network, address)
			}
		}
		return clientWithTransport(transport), nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckSkipped,
			Detail: "Unauthenticated SOCKS5 proxy is tested through its native tunnel.",
		}, nil
	default:
		return nil, nil, Check{}, fmt.Errorf(
			"unsupported diagnostic proxy scheme %q",
			parsed.Scheme,
		)
	}
}

func (r *Runner) bridgedClient(
	ctx context.Context,
	route proxy.Route,
	material credential.Material,
) (*http.Client, proxybridge.Instance, Check, error) {
	if route.BridgeKind != "local-auth-bridge" {
		err := fmt.Errorf("proxy adapter %q is not available", route.BridgeKind)
		return nil, nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckFail,
			Detail: fmt.Sprintf("Proxy adapter %q is not available.", route.BridgeKind),
		}, err
	}

	started := time.Now()
	bridge, err := r.factory().Start(ctx, route.DisplayURL, material)
	if err != nil {
		return nil, nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckFail,
			Detail: sanitize(
				"Authenticated loopback bridge failed: "+err.Error(),
				material,
			),
			LatencyMS: time.Since(started).Milliseconds(),
		}, err
	}
	bridgeURL, err := url.Parse(bridge.URL())
	if err != nil || bridgeURL.Scheme != "http" || !isLoopbackHost(bridgeURL.Hostname()) {
		_ = bridge.Close()
		return nil, nil, Check{
			ID: "bridge", Label: "Local bridge", Status: CheckFail,
			Detail: "The diagnostic bridge did not expose a valid loopback HTTP endpoint.",
		}, fmt.Errorf("invalid loopback bridge URL")
	}
	return diagnosticClient(bridgeURL), bridge, Check{
		ID: "bridge", Label: "Local bridge", Status: CheckPass,
		Detail:    "Vault-backed credentials are isolated behind an ephemeral IPv4 loopback listener.",
		LatencyMS: time.Since(started).Milliseconds(),
	}, nil
}
