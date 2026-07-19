package networkevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/proxy"
)

type routeDigestRecord struct {
	Kind          RouteKind `json:"kind"`
	Scheme        string    `json:"scheme"`
	BridgeKind    string    `json:"bridgeKind,omitempty"`
	ProxyURL      string    `json:"proxyUrl"`
	CredentialRef string    `json:"credentialRef,omitempty"`
	AdapterRef    string    `json:"adapterRef,omitempty"`
}

func RouteForProfile(profile domain.Profile) (RouteIdentity, error) {
	route, err := proxy.Resolve(profile.Proxy.URL, profile.Proxy.CredentialRef)
	if err != nil {
		return RouteIdentity{}, err
	}
	scheme := routeScheme(profile.Proxy.URL)
	kind, err := classifyRoute(route, scheme)
	if err != nil {
		return RouteIdentity{}, err
	}
	record := routeDigestRecord{
		Kind:          kind,
		Scheme:        scheme,
		BridgeKind:    route.BridgeKind,
		ProxyURL:      strings.TrimSpace(profile.Proxy.URL),
		CredentialRef: strings.TrimSpace(profile.Proxy.CredentialRef),
		AdapterRef:    strings.TrimSpace(profile.Proxy.AdapterRef),
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return RouteIdentity{}, fmt.Errorf("encode network route identity: %w", err)
	}
	digest := sha256.Sum256(encoded)
	identity := RouteIdentity{
		Kind:       kind,
		Scheme:     scheme,
		BridgeKind: route.BridgeKind,
		Digest:     hex.EncodeToString(digest[:]),
	}
	return identity, identity.Validate()
}

func classifyRoute(route proxy.Route, scheme string) (RouteKind, error) {
	if route.DisplayURL == "direct://" {
		return RouteDirect, nil
	}
	if route.RequiresBridge {
		switch route.BridgeKind {
		case "local-auth-bridge":
			return RouteLocalAuthBridge, nil
		case "xray":
			return RouteXray, nil
		case "sing-box":
			return RouteSingBox, nil
		default:
			return "", fmt.Errorf("unsupported managed route bridge %q", route.BridgeKind)
		}
	}
	switch scheme {
	case "http":
		return RouteHTTP, nil
	case "https":
		return RouteHTTPS, nil
	case "socks5":
		return RouteSOCKS5, nil
	default:
		return "", fmt.Errorf("unsupported managed route scheme %q", scheme)
	}
}

func routeScheme(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "direct://" {
		return "direct"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Scheme)
}
