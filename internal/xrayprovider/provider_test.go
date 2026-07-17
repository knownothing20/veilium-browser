package xrayprovider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
)

const testUUID = "5783a3e7-e373-51cd-8642-c83782b807c5"

func TestPrepareVLESSTLSRaw(t *testing.T) {
	plan := preparePlan(t,
		"vless",
		"vless://server.example:443?security=tls&type=raw&sni=server.example&alpn=h2%2Chttp%2F1.1&fp=chrome&encryption=none&flow=xtls-rprx-vision",
		"identity",
		testUUID,
	)
	if strings.Contains(strings.Join(plan.Arguments, " "), testUUID) {
		t.Fatal("VLESS UUID leaked into process arguments")
	}
	config := decodeConfig(t, plan.Config)
	outbound := proxyOutbound(t, config)
	settings := object(t, outbound["settings"])
	if settings["id"] != testUUID || settings["encryption"] != "none" || settings["flow"] != "xtls-rprx-vision" {
		t.Fatalf("unexpected VLESS settings: %#v", settings)
	}
	stream := object(t, outbound["streamSettings"])
	if stream["method"] != "raw" || stream["security"] != "tls" {
		t.Fatalf("unexpected VLESS stream: %#v", stream)
	}
	tlsSettings := object(t, stream["tlsSettings"])
	if tlsSettings["serverName"] != "server.example" || tlsSettings["allowInsecure"] != false || tlsSettings["minVersion"] != "1.2" {
		t.Fatalf("unexpected TLS settings: %#v", tlsSettings)
	}
	alpn := array(t, tlsSettings["alpn"])
	if len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Fatalf("unexpected ALPN: %#v", alpn)
	}
}

func TestPrepareVMessWebSocketTLS(t *testing.T) {
	plan := preparePlan(t,
		"vmess",
		"vmess://server.example:443?security=tls&type=ws&sni=server.example&path=%2Fsocket&host=cdn.example&cipher=auto",
		"identity",
		testUUID,
	)
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	if outbound["protocol"] != "vmess" {
		t.Fatalf("unexpected protocol: %#v", outbound)
	}
	settings := object(t, outbound["settings"])
	if settings["id"] != testUUID || settings["security"] != "auto" {
		t.Fatalf("unexpected VMess settings: %#v", settings)
	}
	stream := object(t, outbound["streamSettings"])
	if stream["method"] != "websocket" || stream["security"] != "tls" {
		t.Fatalf("unexpected VMess stream: %#v", stream)
	}
	ws := object(t, stream["wsSettings"])
	if ws["path"] != "/socket" || ws["host"] != "cdn.example" {
		t.Fatalf("unexpected WebSocket settings: %#v", ws)
	}
}

func TestPrepareTrojanGRPCTLS(t *testing.T) {
	plan := preparePlan(t,
		"trojan",
		"trojan://server.example:443?security=tls&type=grpc&sni=server.example&serviceName=veilium&authority=grpc.example&alpn=h2",
		"identity",
		"trojan-secret",
	)
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	settings := object(t, outbound["settings"])
	if settings["password"] != "trojan-secret" {
		t.Fatalf("unexpected Trojan settings: %#v", settings)
	}
	stream := object(t, outbound["streamSettings"])
	grpc := object(t, stream["grpcSettings"])
	if grpc["serviceName"] != "veilium" || grpc["authority"] != "grpc.example" || grpc["multiMode"] != false {
		t.Fatalf("unexpected gRPC settings: %#v", grpc)
	}
}

func TestPrepareShadowsocksUsesVaultMetadataForMethod(t *testing.T) {
	plan := preparePlan(t,
		"ss",
		"ss://server.example:8388",
		"aes-256-gcm",
		"shadowsocks-secret",
	)
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	if outbound["protocol"] != "shadowsocks" {
		t.Fatalf("unexpected protocol: %#v", outbound)
	}
	settings := object(t, outbound["settings"])
	if settings["method"] != "aes-256-gcm" || settings["password"] != "shadowsocks-secret" {
		t.Fatalf("unexpected Shadowsocks settings: %#v", settings)
	}
	if _, exists := outbound["streamSettings"]; exists {
		t.Fatal("Shadowsocks unexpectedly received transport settings")
	}
}

func TestPrepareVLESSRealityUsesSecretEnvelope(t *testing.T) {
	realityPassword := "V8TnE2e7L9HqYlqFv2dPUz5BVQeRrJ70x4o9H5gC_uQ"
	secret := `{"id":"` + testUUID + `","realityPassword":"` + realityPassword + `"}`
	plan := preparePlan(t,
		"vless",
		"vless://server.example:443?security=reality&type=raw&sni=www.example.com&fp=chrome&shortId=aabb&spiderX=%2Fnews&encryption=none&flow=xtls-rprx-vision",
		"identity",
		secret,
	)
	if strings.Contains(strings.Join(plan.Arguments, " "), realityPassword) || strings.Contains(strings.Join(plan.Arguments, " "), testUUID) {
		t.Fatal("REALITY material leaked into process arguments")
	}
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	settings := object(t, outbound["settings"])
	if settings["id"] != testUUID {
		t.Fatalf("unexpected VLESS id: %#v", settings)
	}
	stream := object(t, outbound["streamSettings"])
	reality := object(t, stream["realitySettings"])
	if reality["password"] != realityPassword || reality["serverName"] != "www.example.com" || reality["shortId"] != "aabb" || reality["spiderX"] != "/news" {
		t.Fatalf("unexpected REALITY settings: %#v", reality)
	}
}

func TestProviderRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		url    string
		secret string
		want   string
	}{
		{name: "inline user info", scheme: "vless", url: "vless://user@server.example:443?security=tls", secret: testUUID, want: "inline credentials"},
		{name: "unknown option", scheme: "vless", url: "vless://server.example:443?security=tls&unknown=value", secret: testUUID, want: "unsupported advanced proxy option"},
		{name: "invalid alpn", scheme: "vless", url: "vless://server.example:443?security=tls&alpn=h3", secret: testUUID, want: "unsupported TLS ALPN"},
		{name: "vless no security", scheme: "vless", url: "vless://server.example:443", secret: testUUID, want: "explicit transport security"},
		{name: "vmess reality", scheme: "vmess", url: "vmess://server.example:443?security=reality&sni=server.example&shortId=aa", secret: `{"id":"` + testUUID + `","realityPassword":"V8TnE2e7L9HqYlqFv2dPUz5BVQeRrJ70x4o9H5gC_uQ"}`, want: "VMess REALITY"},
		{name: "reality key in URL", scheme: "vless", url: "vless://server.example:443?security=reality&sni=server.example&publicKey=secret", secret: testUUID, want: "unsupported advanced proxy option"},
		{name: "secret JSON unknown field", scheme: "vless", url: "vless://server.example:443?security=tls", secret: `{"id":"` + testUUID + `","unexpected":"value"}`, want: "credential JSON is invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().Prepare(context.Background(), adapterruntime.Request{
				Adapter: adapter.Record{Kind: adapter.KindXray, Status: adapter.StatusVerified, Executable: "/managed/xray"},
				Scheme:  test.scheme, ProxyURL: test.url, CredentialUsername: "identity", CredentialSecret: test.secret,
				ProfileID: "profile-a", LocalPort: 19080,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func preparePlan(t *testing.T, scheme, rawURL, username, secret string) adapterruntime.Plan {
	t.Helper()
	plan, err := New().Prepare(context.Background(), adapterruntime.Request{
		Adapter: adapter.Record{
			ID: "xray-a", Name: "Xray", Kind: adapter.KindXray,
			Status: adapter.StatusVerified, Executable: "/managed/xray",
		},
		Scheme: scheme, ProxyURL: rawURL, CredentialUsername: username, CredentialSecret: secret,
		ProfileID: "profile-a", LocalPort: 19080,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable != "/managed/xray" || plan.LocalScheme != "socks5" || plan.ConfigFormat != "json" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if strings.Join(plan.Arguments, " ") != "run -config "+adapterruntime.ConfigPathToken {
		t.Fatalf("unexpected arguments: %#v", plan.Arguments)
	}
	return plan
}

func decodeConfig(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config
}

func proxyOutbound(t *testing.T, config map[string]any) map[string]any {
	t.Helper()
	outbounds := array(t, config["outbounds"])
	if len(outbounds) < 1 {
		t.Fatal("configuration has no outbound")
	}
	return object(t, outbounds[0])
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	return result
}

func array(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", value)
	}
	return result
}
