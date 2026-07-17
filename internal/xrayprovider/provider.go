package xrayprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
)

const maxSecretBytes = 4096

var (
	uuidPattern        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	shortIDPattern     = regexp.MustCompile(`^(?:[0-9a-fA-F]{2}){0,8}$`)
	realityKeyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{20,128}$`)
	allowedFingerprint = map[string]bool{
		"chrome": true, "firefox": true, "safari": true, "ios": true,
		"android": true, "edge": true, "360": true, "qq": true,
		"random": true, "randomized": true,
	}
	allowedShadowsocks = map[string]bool{
		"2022-blake3-aes-128-gcm":       true,
		"2022-blake3-aes-256-gcm":       true,
		"2022-blake3-chacha20-poly1305": true,
		"aes-256-gcm":                   true,
		"aes-128-gcm":                   true,
		"chacha20-poly1305":             true,
		"chacha20-ietf-poly1305":        true,
		"xchacha20-poly1305":            true,
		"xchacha20-ietf-poly1305":       true,
	}
)

type Provider struct{}

func New() Provider           { return Provider{} }
func (Provider) Kind() string { return adapter.KindXray }

func (Provider) Prepare(_ context.Context, request adapterruntime.Request) (adapterruntime.Plan, error) {
	if request.LocalPort < 1 || request.LocalPort > 65535 {
		return adapterruntime.Plan{}, fmt.Errorf("Xray loopback port is invalid")
	}
	endpoint, err := parseEndpoint(request.ProxyURL, request.Scheme)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	material, err := decodeSecretMaterial(request.CredentialSecret)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	outbound, err := buildOutbound(endpoint, request.CredentialUsername, material)
	if err != nil {
		return adapterruntime.Plan{}, err
	}
	configuration := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
			"access":   "none",
		},
		"inbounds": []any{map[string]any{
			"tag":      "veilium-in",
			"listen":   "127.0.0.1",
			"port":     request.LocalPort,
			"protocol": "socks",
			"settings": map[string]any{
				"auth": "noauth",
				"udp":  true,
				"ip":   "127.0.0.1",
			},
		}},
		"outbounds": []any{
			outbound,
			map[string]any{"tag": "direct", "protocol": "freedom"},
			map[string]any{"tag": "block", "protocol": "blackhole"},
		},
		"routing": map[string]any{
			"domainStrategy": "AsIs",
			"rules": []any{map[string]any{
				"type":        "field",
				"inboundTag":  []string{"veilium-in"},
				"outboundTag": "proxy",
			}},
		},
	}
	encoded, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return adapterruntime.Plan{}, fmt.Errorf("encode private Xray configuration: %w", err)
	}
	return adapterruntime.Plan{
		Executable:   request.Adapter.Executable,
		Arguments:    []string{"run", "-config", adapterruntime.ConfigPathToken},
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
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return endpoint{}, fmt.Errorf("invalid advanced proxy URL")
	}
	scheme := normalizeScheme(parsed.Scheme)
	expectedScheme = normalizeScheme(expectedScheme)
	if scheme != expectedScheme || !adapter.SupportsScheme(adapter.KindXray, scheme) {
		return endpoint{}, fmt.Errorf("Xray provider cannot prepare proxy scheme %q", scheme)
	}
	if parsed.User != nil || parsed.Path != "" || parsed.Opaque != "" || parsed.Fragment != "" {
		return endpoint{}, fmt.Errorf("advanced proxy URL must not contain inline credentials, paths, or fragments")
	}
	host := strings.TrimSpace(parsed.Hostname())
	port, err := strconv.Atoi(parsed.Port())
	if host == "" || err != nil || port < 1 || port > 65535 || containsControl(host) {
		return endpoint{}, fmt.Errorf("advanced proxy URL requires an explicit valid host and port")
	}
	return endpoint{scheme: scheme, host: host, port: port, query: parsed.Query()}, nil
}

type secretEnvelope struct {
	ID              string `json:"id"`
	Password        string `json:"password"`
	RealityPassword string `json:"realityPassword"`
}

type secretMaterial struct {
	simple          string
	id              string
	password        string
	realityPassword string
}

func decodeSecretMaterial(raw string) (secretMaterial, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxSecretBytes || strings.ContainsRune(raw, '\x00') {
		return secretMaterial{}, fmt.Errorf("advanced proxy credential secret is invalid")
	}
	if !strings.HasPrefix(raw, "{") {
		if containsControl(raw) {
			return secretMaterial{}, fmt.Errorf("advanced proxy credential secret is invalid")
		}
		return secretMaterial{simple: raw}, nil
	}

	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var envelope secretEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return secretMaterial{}, fmt.Errorf("advanced proxy credential JSON is invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return secretMaterial{}, fmt.Errorf("advanced proxy credential JSON contains trailing data")
	}
	for _, value := range []string{envelope.ID, envelope.Password, envelope.RealityPassword} {
		if containsControl(value) || len(value) > maxSecretBytes {
			return secretMaterial{}, fmt.Errorf("advanced proxy credential JSON contains an invalid field")
		}
	}
	if envelope.ID == "" && envelope.Password == "" && envelope.RealityPassword == "" {
		return secretMaterial{}, fmt.Errorf("advanced proxy credential JSON is empty")
	}
	return secretMaterial{
		id:              strings.TrimSpace(envelope.ID),
		password:        strings.TrimSpace(envelope.Password),
		realityPassword: strings.TrimSpace(envelope.RealityPassword),
	}, nil
}

func buildOutbound(endpoint endpoint, username string, material secretMaterial) (map[string]any, error) {
	username = strings.TrimSpace(username)
	var settings map[string]any

	switch endpoint.scheme {
	case "vless":
		id := valueOr(material.id, material.simple)
		if !uuidPattern.MatchString(id) {
			return nil, fmt.Errorf("VLESS credential secret must provide a canonical UUID")
		}
		encryption := valueOr(endpoint.query.Get("encryption"), "none")
		if encryption != "none" {
			return nil, fmt.Errorf("VLESS Encryption is not accepted by this provider yet")
		}
		settings = map[string]any{
			"address":    endpoint.host,
			"port":       endpoint.port,
			"id":         id,
			"encryption": encryption,
		}
		if flow := strings.TrimSpace(endpoint.query.Get("flow")); flow != "" {
			if flow != "xtls-rprx-vision" && flow != "xtls-rprx-vision-udp443" {
				return nil, fmt.Errorf("unsupported VLESS flow")
			}
			settings["flow"] = flow
		}
	case "vmess":
		id := valueOr(material.id, material.simple)
		if !uuidPattern.MatchString(id) {
			return nil, fmt.Errorf("VMess credential secret must provide a canonical UUID")
		}
		cipher := strings.ToLower(valueOr(endpoint.query.Get("cipher"), "auto"))
		if !map[string]bool{"auto": true, "aes-128-gcm": true, "chacha20-poly1305": true, "none": true, "zero": true}[cipher] {
			return nil, fmt.Errorf("unsupported VMess cipher")
		}
		settings = map[string]any{
			"address":  endpoint.host,
			"port":     endpoint.port,
			"id":       id,
			"security": cipher,
		}
	case "trojan":
		password := valueOr(material.password, material.simple)
		if password == "" || containsControl(password) {
			return nil, fmt.Errorf("Trojan credential secret must provide a password")
		}
		settings = map[string]any{
			"address":  endpoint.host,
			"port":     endpoint.port,
			"password": password,
		}
	case "ss":
		password := valueOr(material.password, material.simple)
		if password == "" || containsControl(password) {
			return nil, fmt.Errorf("Shadowsocks credential secret must provide a password")
		}
		method := strings.ToLower(strings.TrimSpace(endpoint.query.Get("method")))
		if method == "" {
			method = strings.ToLower(username)
		}
		if !allowedShadowsocks[method] {
			return nil, fmt.Errorf("unsupported Shadowsocks encryption method")
		}
		settings = map[string]any{
			"address":  endpoint.host,
			"port":     endpoint.port,
			"method":   method,
			"password": password,
		}
	default:
		return nil, fmt.Errorf("unsupported Xray proxy scheme")
	}

	stream, err := buildStreamSettings(endpoint, material.realityPassword)
	if err != nil {
		return nil, err
	}
	outbound := map[string]any{
		"tag":      "proxy",
		"protocol": canonicalProtocol(endpoint.scheme),
		"settings": settings,
	}
	if len(stream) > 0 {
		outbound["streamSettings"] = stream
	}
	return outbound, nil
}

func buildStreamSettings(endpoint endpoint, realityPassword string) (map[string]any, error) {
	allowed := map[string]bool{
		"security":    true,
		"type":        true,
		"sni":         true,
		"alpn":        true,
		"fp":          true,
		"flow":        true,
		"encryption":  true,
		"path":        true,
		"host":        true,
		"serviceName": true,
		"authority":   true,
		"shortId":     true,
		"spiderX":     true,
		"method":      true,
		"cipher":      true,
	}
	for key, values := range endpoint.query {
		if !allowed[key] {
			return nil, fmt.Errorf("unsupported advanced proxy option %q", key)
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("advanced proxy option %q must appear once", key)
		}
	}
	if endpoint.query.Has("password") || endpoint.query.Has("publicKey") {
		return nil, fmt.Errorf("REALITY server key must be stored in the operating-system credential secret")
	}

	if endpoint.scheme == "ss" {
		for key := range endpoint.query {
			if key != "method" {
				return nil, fmt.Errorf("Shadowsocks URL does not accept transport option %q", key)
			}
		}
		return nil, nil
	}
	if endpoint.scheme != "vless" && (endpoint.query.Has("flow") || endpoint.query.Has("encryption")) {
		return nil, fmt.Errorf("flow and encryption options are valid only for VLESS")
	}
	if endpoint.scheme != "vmess" && endpoint.query.Has("cipher") {
		return nil, fmt.Errorf("cipher option is valid only for VMess")
	}
	if endpoint.query.Has("method") {
		return nil, fmt.Errorf("method option is valid only for Shadowsocks")
	}

	methodInput := strings.ToLower(valueOr(endpoint.query.Get("type"), "raw"))
	method := ""
	switch methodInput {
	case "tcp", "raw":
		method = "raw"
	case "ws", "websocket":
		method = "websocket"
	case "grpc":
		method = "grpc"
	default:
		return nil, fmt.Errorf("unsupported Xray transport method")
	}

	security := strings.ToLower(strings.TrimSpace(endpoint.query.Get("security")))
	if security == "" {
		if endpoint.scheme == "vmess" {
			security = "none"
		} else {
			return nil, fmt.Errorf("%s requires explicit transport security", strings.ToUpper(endpoint.scheme))
		}
	}
	if security != "none" && security != "tls" && security != "reality" {
		return nil, fmt.Errorf("unsupported Xray transport security")
	}
	if endpoint.scheme == "trojan" && security == "none" {
		return nil, fmt.Errorf("Trojan requires TLS or REALITY transport security")
	}
	if endpoint.scheme == "vless" && security == "none" {
		return nil, fmt.Errorf("VLESS without transport security is not accepted")
	}
	if endpoint.scheme == "vmess" && security == "reality" {
		return nil, fmt.Errorf("VMess REALITY is not accepted by this provider")
	}
	if security == "reality" && method != "raw" && method != "grpc" {
		return nil, fmt.Errorf("REALITY is accepted only with raw or gRPC transport")
	}

	if method != "websocket" && (endpoint.query.Has("path") || endpoint.query.Has("host")) {
		return nil, fmt.Errorf("path and host options require WebSocket transport")
	}
	if method != "grpc" && (endpoint.query.Has("serviceName") || endpoint.query.Has("authority")) {
		return nil, fmt.Errorf("serviceName and authority options require gRPC transport")
	}
	if security != "reality" && (endpoint.query.Has("shortId") || endpoint.query.Has("spiderX")) {
		return nil, fmt.Errorf("shortId and spiderX options require REALITY")
	}
	if security == "none" && (endpoint.query.Has("sni") || endpoint.query.Has("alpn") || endpoint.query.Has("fp")) {
		return nil, fmt.Errorf("sni, alpn, and fp options require TLS or REALITY")
	}

	stream := map[string]any{"method": method, "security": security}
	switch method {
	case "websocket":
		path := valueOr(endpoint.query.Get("path"), "/")
		if !strings.HasPrefix(path, "/") || containsControl(path) {
			return nil, fmt.Errorf("invalid WebSocket path")
		}
		settings := map[string]any{"path": path}
		if host := strings.TrimSpace(endpoint.query.Get("host")); host != "" {
			if containsControl(host) {
				return nil, fmt.Errorf("invalid WebSocket host")
			}
			settings["host"] = host
		}
		stream["wsSettings"] = settings
	case "grpc":
		serviceName := strings.TrimSpace(endpoint.query.Get("serviceName"))
		if serviceName == "" || containsControl(serviceName) {
			return nil, fmt.Errorf("gRPC transport requires a serviceName")
		}
		settings := map[string]any{"serviceName": serviceName, "multiMode": false}
		if authority := strings.TrimSpace(endpoint.query.Get("authority")); authority != "" {
			if containsControl(authority) {
				return nil, fmt.Errorf("invalid gRPC authority")
			}
			settings["authority"] = authority
		}
		stream["grpcSettings"] = settings
	}

	if security == "tls" {
		serverName := valueOr(endpoint.query.Get("sni"), endpoint.host)
		if containsControl(serverName) {
			return nil, fmt.Errorf("invalid TLS server name")
		}
		fingerprint, err := parseFingerprint(endpoint.query.Get("fp"))
		if err != nil {
			return nil, err
		}
		tlsSettings := map[string]any{
			"serverName":    serverName,
			"allowInsecure": false,
			"minVersion":    "1.2",
			"fingerprint":   fingerprint,
		}
		alpn, err := parseALPN(endpoint.query.Get("alpn"))
		if err != nil {
			return nil, err
		}
		if len(alpn) > 0 {
			tlsSettings["alpn"] = alpn
		}
		stream["tlsSettings"] = tlsSettings
	}
	if security == "reality" {
		serverName := strings.TrimSpace(endpoint.query.Get("sni"))
		shortID := strings.TrimSpace(endpoint.query.Get("shortId"))
		fingerprint, err := parseFingerprint(endpoint.query.Get("fp"))
		if err != nil {
			return nil, err
		}
		if serverName == "" || containsControl(serverName) {
			return nil, fmt.Errorf("REALITY requires a valid sni")
		}
		if !realityKeyPattern.MatchString(realityPassword) {
			return nil, fmt.Errorf("REALITY credential secret must provide a valid realityPassword")
		}
		if !shortIDPattern.MatchString(shortID) {
			return nil, fmt.Errorf("REALITY shortId must contain an even number of hexadecimal characters")
		}
		reality := map[string]any{
			"serverName":  serverName,
			"fingerprint": fingerprint,
			"password":    realityPassword,
			"shortId":     shortID,
		}
		if spiderX := strings.TrimSpace(endpoint.query.Get("spiderX")); spiderX != "" {
			if !strings.HasPrefix(spiderX, "/") || containsControl(spiderX) {
				return nil, fmt.Errorf("invalid REALITY spiderX")
			}
			reality["spiderX"] = spiderX
		}
		stream["realitySettings"] = reality
	}
	return stream, nil
}

func canonicalProtocol(scheme string) string {
	if scheme == "ss" {
		return "shadowsocks"
	}
	return scheme
}

func parseFingerprint(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "chrome"
	}
	if !allowedFingerprint[value] {
		return "", fmt.Errorf("unsupported TLS fingerprint")
	}
	return value, nil
}

func parseALPN(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		if part != "h2" && part != "http/1.1" {
			return nil, fmt.Errorf("unsupported TLS ALPN value")
		}
		seen[part] = true
		result = append(result, part)
	}
	return result, nil
}

func normalizeScheme(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "shadowsocks" {
		return "ss"
	}
	return value
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func containsControl(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
