package systemproxy

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	SourceWindowsStatic = "windows-static"
	ReasonDisabled      = "disabled"
	ReasonPACOnly       = "pac-only"
	ReasonUnsupported   = "unsupported-platform"
)

// Result is a credential-free snapshot of the current operating-system proxy.
// ProxyURL is always an explicit URL that can be stored by the existing profile
// model; the operating-system setting itself is never referenced or modified.
type Result struct {
	Available bool   `json:"available"`
	ProxyURL  string `json:"proxyUrl,omitempty"`
	Source    string `json:"source"`
	Reason    string `json:"reason,omitempty"`
}

func normalizeStaticProxy(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("the system static proxy address is empty")
	}
	if !strings.Contains(raw, "=") {
		return normalizeEndpoint(raw, "http")
	}

	entries := make(map[string]string)
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", fmt.Errorf("the system static proxy format is unsupported")
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if value != "" {
			entries[key] = value
		}
	}

	httpEndpoint, hasHTTP := entries["http"]
	httpsEndpoint, hasHTTPS := entries["https"]
	if hasHTTP || hasHTTPS {
		// A single profile URL applies to all browser traffic. Only collapse the
		// Windows per-scheme form when HTTP and HTTPS explicitly use the same
		// endpoint; choosing one of two different routes would change semantics.
		if !hasHTTP || !hasHTTPS {
			return "", fmt.Errorf("the system proxy uses a per-scheme route that cannot be represented safely")
		}
		httpURL, err := normalizeEndpoint(httpEndpoint, "http")
		if err != nil {
			return "", err
		}
		httpsURL, err := normalizeEndpoint(httpsEndpoint, "http")
		if err != nil {
			return "", err
		}
		if httpURL != httpsURL {
			return "", fmt.Errorf("the system proxy uses different HTTP and HTTPS endpoints")
		}
		return httpURL, nil
	}

	if endpoint := firstNonEmpty(entries["socks5"], entries["socks"]); endpoint != "" {
		return normalizeEndpoint(endpoint, "socks5")
	}
	return "", fmt.Errorf("the system static proxy format is unsupported")
}

func normalizeEndpoint(raw, defaultScheme string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = defaultScheme + "://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("the system static proxy address is invalid")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("the system proxy contains inline credentials and cannot be imported")
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("the system static proxy address is invalid")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "socks" {
		scheme = "socks5"
	}
	if scheme != "http" && scheme != "https" && scheme != "socks5" {
		return "", fmt.Errorf("the system static proxy scheme is unsupported")
	}
	parsed.Scheme = scheme
	return parsed.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
