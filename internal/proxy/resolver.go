package proxy

import (
	"fmt"
	"net/url"
	"strings"
)

type Route struct {
	BrowserURL     string
	DisplayURL     string
	RequiresBridge bool
	BridgeKind     string
	CredentialRef  string
}

func Resolve(raw, credentialRef string) (Route, error) {
	raw = strings.TrimSpace(raw)
	credentialRef = strings.TrimSpace(credentialRef)
	if raw == "" || raw == "direct://" {
		if credentialRef != "" {
			return Route{}, fmt.Errorf("direct connections cannot use a credential reference")
		}
		return Route{BrowserURL: "direct://", DisplayURL: "direct://"}, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Route{}, fmt.Errorf("invalid proxy URL")
	}
	if parsed.User != nil {
		return Route{}, fmt.Errorf("inline proxy credentials are forbidden; use credentialRef")
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https", "socks5":
		if credentialRef != "" {
			return Route{
				DisplayURL:     parsed.String(),
				RequiresBridge: true,
				BridgeKind:     "local-auth-bridge",
				CredentialRef:  credentialRef,
			}, nil
		}
		return Route{BrowserURL: parsed.String(), DisplayURL: parsed.String()}, nil
	case "vmess", "vless", "trojan", "ss", "shadowsocks":
		if credentialRef == "" {
			return Route{}, fmt.Errorf("proxy scheme %q requires credentialRef", scheme)
		}
		return Route{DisplayURL: parsed.String(), RequiresBridge: true, BridgeKind: "xray", CredentialRef: credentialRef}, nil
	case "hysteria2", "tuic", "anytls":
		if credentialRef == "" {
			return Route{}, fmt.Errorf("proxy scheme %q requires credentialRef", scheme)
		}
		return Route{DisplayURL: parsed.String(), RequiresBridge: true, BridgeKind: "sing-box", CredentialRef: credentialRef}, nil
	default:
		return Route{}, fmt.Errorf("unsupported proxy scheme %q", scheme)
	}
}
