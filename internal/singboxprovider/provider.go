package singboxprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
)

const maxSecretBytes = 4096

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Provider struct{}

func New() Provider           { return Provider{} }
func (Provider) Kind() string { return adapter.KindSingBox }

func (Provider) Prepare(_ context.Context, request adapterruntime.Request) (adapterruntime.Plan, error) {
	if request.LocalPort < 1 || request.LocalPort > 65535 {
		return adapterruntime.Plan{}, fmt.Errorf("sing-box loopback port is invalid")
	}
	if err := validateAdapterVersion(request.Adapter.Version); err != nil {
		return adapterruntime.Plan{}, err
	}
	endpoint, err := parseEndpoint(request.ProxyURL, request.Scheme)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	material, err := decodeSecretMaterial(request.CredentialSecret)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	outbound, err := buildOutbound(endpoint, material)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	configuration := map[string]any{
		"log": map[string]any{
			"level":     "warn",
			"timestamp": false,
		},
		"dns": map[string]any{
			"servers":  []any{map[string]any{"type": "local", "tag": "local"}},
			"final":    "local",
			"strategy": "prefer_ipv4",
		},
		"inbounds": []any{map[string]any{
			"type":        "socks",
			"tag":         "veilium-in",
			"listen":      "127.0.0.1",
			"listen_port": request.LocalPort,
		}},
		"outbounds": []any{
			outbound,
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"route": map[string]any{
			"final":                   "proxy",
			"auto_detect_interface":   true,
			"default_domain_resolver": "local",
		},
	}
	encoded, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return adapterruntime.Plan{}, fmt.Errorf("encode private sing-box configuration: %w", err)
	}
	return adapterruntime.Plan{
		Executable:   request.Adapter.Executable,
		Arguments:    []string{"run", "-c", adapterruntime.ConfigPathToken},
		Config:       encoded,
		ConfigFormat: "json",
		LocalScheme:  "socks5",
	}, nil
}

type endpoint struct {
	scheme string
	host   string
	port   int
	query  url.Values
}

func parseEndpoint(raw, expectedScheme string) (endpoint, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return endpoint{}, fmt.Errorf("invalid sing-box proxy URL")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != strings.ToLower(strings.TrimSpace(expectedScheme)) || !adapter.SupportsScheme(adapter.KindSingBox, scheme) {
		return endpoint{}, fmt.Errorf("sing-box provider cannot prepare proxy scheme %q", scheme)
	}
	if parsed.User != nil || parsed.Path != "" || parsed.Opaque != "" || parsed.Fragment != "" {
		return endpoint{}, fmt.Errorf("sing-box proxy URL must not contain inline credentials, paths, or fragments")
	}
	host := strings.TrimSpace(parsed.Hostname())
	port, err := strconv.Atoi(parsed.Port())
	if host == "" || err != nil || port < 1 || port > 65535 || containsControl(host) {
		return endpoint{}, fmt.Errorf("sing-box proxy URL requires an explicit valid host and port")
	}
	query := parsed.Query()
	for key, values := range query {
		if len(values) != 1 {
			return endpoint{}, fmt.Errorf("sing-box proxy option %q must appear exactly once", key)
		}
	}
	if err := validateQueryKeys(scheme, query); err != nil {
		return endpoint{}, err
	}
	return endpoint{scheme: scheme, host: host, port: port, query: query}, nil
}

type secretEnvelope struct {
	Password     string `json:"password"`
	UUID         string `json:"uuid"`
	ObfsPassword string `json:"obfsPassword"`
}

type secretMaterial struct {
	simple       string
	password     string
	uuid         string
	obfsPassword string
}

func decodeSecretMaterial(raw string) (secretMaterial, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxSecretBytes || strings.ContainsRune(raw, '\x00') {
		return secretMaterial{}, fmt.Errorf("sing-box credential secret is invalid")
	}
	if !strings.HasPrefix(raw, "{") {
		if containsControl(raw) {
			return secretMaterial{}, fmt.Errorf("sing-box credential secret is invalid")
		}
		return secretMaterial{simple: raw}, nil
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var envelope secretEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return secretMaterial{}, fmt.Errorf("sing-box credential JSON is invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return secretMaterial{}, fmt.Errorf("sing-box credential JSON contains trailing data")
	}
	for _, value := range []string{envelope.Password, envelope.UUID, envelope.ObfsPassword} {
		if containsControl(value) || len(value) > maxSecretBytes {
			return secretMaterial{}, fmt.Errorf("sing-box credential JSON contains an invalid field")
		}
	}
	if envelope.Password == "" && envelope.UUID == "" && envelope.ObfsPassword == "" {
		return secretMaterial{}, fmt.Errorf("sing-box credential JSON is empty")
	}
	return secretMaterial{
		password:     strings.TrimSpace(envelope.Password),
		uuid:         strings.TrimSpace(envelope.UUID),
		obfsPassword: strings.TrimSpace(envelope.ObfsPassword),
	}, nil
}

func buildOutbound(endpoint endpoint, material secretMaterial) (map[string]any, error) {
	tls, err := buildTLS(endpoint)
	if err != nil {
		return nil, err
	}
	outbound := map[string]any{
		"type":        endpoint.scheme,
		"tag":         "proxy",
		"server":      endpoint.host,
		"server_port": endpoint.port,
		"tls":         tls,
	}
	if network := strings.TrimSpace(endpoint.query.Get("network")); network != "" {
		if network != "tcp" && network != "udp" {
			return nil, fmt.Errorf("unsupported sing-box network selection")
		}
		outbound["network"] = network
	}

	switch endpoint.scheme {
	case "hysteria2":
		password := valueOr(material.password, material.simple)
		if password == "" || containsControl(password) {
			return nil, fmt.Errorf("Hysteria2 credential secret must provide a password")
		}
		outbound["password"] = password
		if value := endpoint.query.Get("upMbps"); value != "" {
			number, err := boundedInt(value, 1, 100000)
			if err != nil {
				return nil, fmt.Errorf("invalid Hysteria2 upload bandwidth")
			}
			outbound["up_mbps"] = number
		}
		if value := endpoint.query.Get("downMbps"); value != "" {
			number, err := boundedInt(value, 1, 100000)
			if err != nil {
				return nil, fmt.Errorf("invalid Hysteria2 download bandwidth")
			}
			outbound["down_mbps"] = number
		}
		if obfs := strings.ToLower(strings.TrimSpace(endpoint.query.Get("obfs"))); obfs != "" {
			if obfs != "salamander" {
				return nil, fmt.Errorf("unsupported Hysteria2 obfuscator")
			}
			if material.obfsPassword == "" {
				return nil, fmt.Errorf("Hysteria2 obfuscation requires obfsPassword in the vault secret")
			}
			outbound["obfs"] = map[string]any{"type": obfs, "password": material.obfsPassword}
		} else if material.obfsPassword != "" {
			return nil, fmt.Errorf("Hysteria2 obfsPassword was provided without an obfuscator")
		}
	case "tuic":
		if !uuidPattern.MatchString(material.uuid) || material.password == "" {
			return nil, fmt.Errorf("TUIC credential secret must be JSON with canonical uuid and password")
		}
		outbound["uuid"] = material.uuid
		outbound["password"] = material.password
		congestion := valueOr(strings.ToLower(endpoint.query.Get("congestionControl")), "cubic")
		if !map[string]bool{"cubic": true, "new_reno": true, "bbr": true}[congestion] {
			return nil, fmt.Errorf("unsupported TUIC congestion control")
		}
		outbound["congestion_control"] = congestion
		relay := valueOr(strings.ToLower(endpoint.query.Get("udpRelayMode")), "native")
		if relay != "native" && relay != "quic" {
			return nil, fmt.Errorf("unsupported TUIC UDP relay mode")
		}
		outbound["udp_relay_mode"] = relay
		outbound["zero_rtt_handshake"] = false
	case "anytls":
		password := valueOr(material.password, material.simple)
		if password == "" || containsControl(password) {
			return nil, fmt.Errorf("AnyTLS credential secret must provide a password")
		}
		outbound["password"] = password
		for queryKey, configKey := range map[string]string{
			"idleCheck":   "idle_session_check_interval",
			"idleTimeout": "idle_session_timeout",
		} {
			if value := endpoint.query.Get(queryKey); value != "" {
				if err := validateDuration(value); err != nil {
					return nil, fmt.Errorf("invalid AnyTLS %s", queryKey)
				}
				outbound[configKey] = value
			}
		}
		if value := endpoint.query.Get("minIdle"); value != "" {
			number, err := boundedInt(value, 0, 128)
			if err != nil {
				return nil, fmt.Errorf("invalid AnyTLS minIdle")
			}
			outbound["min_idle_session"] = number
		}
	default:
		return nil, fmt.Errorf("unsupported sing-box proxy scheme")
	}
	return outbound, nil
}

func buildTLS(endpoint endpoint) (map[string]any, error) {
	serverName := strings.TrimSpace(endpoint.query.Get("serverName"))
	sni := strings.TrimSpace(endpoint.query.Get("sni"))
	if serverName != "" && sni != "" {
		return nil, fmt.Errorf("sing-box proxy URL must not set both serverName and sni")
	}
	if serverName == "" {
		serverName = sni
	}
	if serverName == "" && net.ParseIP(endpoint.host) == nil {
		serverName = endpoint.host
	}
	if serverName == "" || containsControl(serverName) {
		return nil, fmt.Errorf("sing-box TLS server name is required")
	}
	tls := map[string]any{
		"enabled":     true,
		"server_name": serverName,
		"insecure":    false,
		"min_version": "1.2",
	}
	if value := strings.TrimSpace(endpoint.query.Get("alpn")); value != "" {
		parts := strings.Split(value, ",")
		if len(parts) > 4 {
			return nil, fmt.Errorf("too many sing-box TLS ALPN values")
		}
		allowed := map[string]bool{"h3": true, "h2": true, "http/1.1": true}
		seen := make(map[string]bool)
		alpn := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !allowed[part] || seen[part] {
				return nil, fmt.Errorf("unsupported sing-box TLS ALPN")
			}
			seen[part] = true
			alpn = append(alpn, part)
		}
		tls["alpn"] = alpn
	}
	return tls, nil
}

func validateQueryKeys(scheme string, query url.Values) error {
	allowed := map[string]bool{"serverName": true, "sni": true, "alpn": true}
	switch scheme {
	case "hysteria2":
		for _, key := range []string{"network", "upMbps", "downMbps", "obfs"} {
			allowed[key] = true
		}
	case "tuic":
		for _, key := range []string{"network", "congestionControl", "udpRelayMode"} {
			allowed[key] = true
		}
	case "anytls":
		for _, key := range []string{"idleCheck", "idleTimeout", "minIdle"} {
			allowed[key] = true
		}
	default:
		return fmt.Errorf("unsupported sing-box proxy scheme")
	}
	for key := range query {
		if !allowed[key] {
			return fmt.Errorf("unsupported sing-box proxy option %q", key)
		}
	}
	return nil
}

func validateAdapterVersion(value string) error {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return fmt.Errorf("sing-box adapter version must be a semantic version")
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil || major < 1 || (major == 1 && minor < 12) {
		return fmt.Errorf("sing-box provider requires adapter version 1.12.0 or newer")
	}
	return nil
}

func boundedInt(value string, minimum, maximum int) (int, error) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < minimum || number > maximum {
		return 0, fmt.Errorf("integer is outside the accepted range")
	}
	return number, nil
}

func validateDuration(value string) error {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration < time.Second || duration > time.Hour {
		return fmt.Errorf("duration is outside the accepted range")
	}
	return nil
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fallback)
}

func containsControl(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
