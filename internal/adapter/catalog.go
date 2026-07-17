package adapter

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	KindXray    = "xray"
	KindSingBox = "sing-box"
)

var (
	licensePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+-]{1,79}$`)
	kindProtocols  = map[string][]string{
		KindXray:    {"vmess", "vless", "trojan", "ss", "shadowsocks"},
		KindSingBox: {"hysteria2", "tuic", "anytls"},
	}
)

func SupportedKinds() []string {
	return []string{KindXray, KindSingBox}
}

func ProtocolsForKind(kind string) []string {
	protocols := append([]string(nil), kindProtocols[NormalizeKind(kind)]...)
	sort.Strings(protocols)
	return protocols
}

func NormalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "singbox" || kind == "sing_box" {
		return KindSingBox
	}
	return kind
}

func ValidateKind(kind string) error {
	kind = NormalizeKind(kind)
	if _, ok := kindProtocols[kind]; !ok {
		return fmt.Errorf("unsupported proxy adapter kind %q", kind)
	}
	return nil
}

func RequiredKindForScheme(scheme string) (string, error) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	for kind, protocols := range kindProtocols {
		for _, protocol := range protocols {
			if scheme == protocol {
				return kind, nil
			}
		}
	}
	return "", fmt.Errorf("proxy scheme %q does not use a managed adapter", scheme)
}

func SupportsScheme(kind, scheme string) bool {
	required, err := RequiredKindForScheme(scheme)
	return err == nil && NormalizeKind(kind) == required
}

func ValidateLicenseSPDX(value string) error {
	value = strings.TrimSpace(value)
	if !licensePattern.MatchString(value) {
		return fmt.Errorf("license must be a short SPDX identifier")
	}
	return nil
}

func ValidateSourceURL(value string) error {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("source URL must be an HTTPS URL without credentials")
	}
	return nil
}

func ValidateVersion(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("adapter version is invalid")
	}
	return nil
}
