package singboxprovider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
)

const testUUID = "2dd61d93-75d8-4da4-ac0e-6aece7eac365"

func TestPrepareHysteria2WithObfuscation(t *testing.T) {
	plan := preparePlan(t,
		"hysteria2",
		"hysteria2://hy.example:443?sni=hy.example&alpn=h3&upMbps=50&downMbps=200&obfs=salamander&network=udp",
		`{"password":"hy-secret","obfsPassword":"obfs-secret"}`,
	)
	if strings.Contains(strings.Join(plan.Arguments, " "), "hy-secret") {
		t.Fatal("Hysteria2 password leaked into process arguments")
	}
	config := decodeConfig(t, plan.Config)
	outbound := proxyOutbound(t, config)
	if outbound["type"] != "hysteria2" || outbound["password"] != "hy-secret" || outbound["up_mbps"] != float64(50) || outbound["down_mbps"] != float64(200) {
		t.Fatalf("unexpected Hysteria2 outbound: %#v", outbound)
	}
	obfs := object(t, outbound["obfs"])
	if obfs["type"] != "salamander" || obfs["password"] != "obfs-secret" {
		t.Fatalf("unexpected Hysteria2 obfs: %#v", obfs)
	}
	assertLoopbackConfig(t, config)
}

func TestPrepareTUIC(t *testing.T) {
	plan := preparePlan(t,
		"tuic",
		"tuic://tuic.example:443?serverName=tuic.example&alpn=h3&congestionControl=bbr&udpRelayMode=quic&network=udp",
		`{"uuid":"`+testUUID+`","password":"tuic-secret"}`,
	)
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	if outbound["type"] != "tuic" || outbound["uuid"] != testUUID || outbound["password"] != "tuic-secret" {
		t.Fatalf("unexpected TUIC outbound: %#v", outbound)
	}
	if outbound["congestion_control"] != "bbr" || outbound["udp_relay_mode"] != "quic" || outbound["zero_rtt_handshake"] != false {
		t.Fatalf("unexpected TUIC controls: %#v", outbound)
	}
}

func TestPrepareAnyTLS(t *testing.T) {
	plan := preparePlan(t,
		"anytls",
		"anytls://any.example:443?sni=any.example&alpn=h2,http%2F1.1&idleCheck=30s&idleTimeout=45s&minIdle=3",
		"anytls-secret",
	)
	outbound := proxyOutbound(t, decodeConfig(t, plan.Config))
	if outbound["type"] != "anytls" || outbound["password"] != "anytls-secret" {
		t.Fatalf("unexpected AnyTLS outbound: %#v", outbound)
	}
	if outbound["idle_session_check_interval"] != "30s" || outbound["idle_session_timeout"] != "45s" || outbound["min_idle_session"] != float64(3) {
		t.Fatalf("unexpected AnyTLS session settings: %#v", outbound)
	}
}

func TestProviderRejectsUnsafeOrUnsupportedInputs(t *testing.T) {
	tests := []struct {
		name    string
		scheme  string
		url     string
		secret  string
		version string
		want    string
	}{
		{name: "inline user info", scheme: "hysteria2", url: "hysteria2://user@server.example:443?sni=server.example", secret: "secret", version: "1.12.0", want: "inline credentials"},
		{name: "unknown option", scheme: "hysteria2", url: "hysteria2://server.example:443?sni=server.example&unknown=value", secret: "secret", version: "1.12.0", want: "unsupported sing-box proxy option"},
		{name: "duplicate option", scheme: "hysteria2", url: "hysteria2://server.example:443?sni=a&sni=b", secret: "secret", version: "1.12.0", want: "exactly once"},
		{name: "ip without sni", scheme: "hysteria2", url: "hysteria2://192.0.2.10:443", secret: "secret", version: "1.12.0", want: "server name is required"},
		{name: "insecure option", scheme: "tuic", url: "tuic://server.example:443?sni=server.example&insecure=true", secret: `{"uuid":"` + testUUID + `","password":"secret"}`, version: "1.12.0", want: "unsupported sing-box proxy option"},
		{name: "tuic simple secret", scheme: "tuic", url: "tuic://server.example:443?sni=server.example", secret: "secret", version: "1.12.0", want: "canonical uuid"},
		{name: "old adapter", scheme: "anytls", url: "anytls://server.example:443?sni=server.example", secret: "secret", version: "1.11.9", want: "1.12.0 or newer"},
		{name: "bad duration", scheme: "anytls", url: "anytls://server.example:443?sni=server.example&idleCheck=5d", secret: "secret", version: "1.12.0", want: "invalid AnyTLS idleCheck"},
		{name: "secret JSON unknown field", scheme: "hysteria2", url: "hysteria2://server.example:443?sni=server.example", secret: `{"password":"secret","extra":"value"}`, version: "1.12.0", want: "credential JSON is invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().Prepare(context.Background(), adapterruntime.Request{
				Adapter: adapter.Record{Kind: adapter.KindSingBox, Version: test.version, Status: adapter.StatusVerified, Executable: "/managed/sing-box"},
				Scheme:  test.scheme, ProxyURL: test.url, CredentialSecret: test.secret,
				ProfileID: "profile-a", LocalPort: 19080,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func preparePlan(t *testing.T, scheme, rawURL, secret string) adapterruntime.Plan {
	t.Helper()
	plan, err := New().Prepare(context.Background(), adapterruntime.Request{
		Adapter: adapter.Record{
			ID: "singbox-a", Name: "sing-box", Kind: adapter.KindSingBox, Version: "1.12.0",
			Status: adapter.StatusVerified, Executable: "/managed/sing-box",
		},
		Scheme: scheme, ProxyURL: rawURL, CredentialSecret: secret,
		ProfileID: "profile-a", LocalPort: 19080,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable != "/managed/sing-box" || plan.LocalScheme != "socks5" || plan.ConfigFormat != "json" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if strings.Join(plan.Arguments, " ") != "run -c "+adapterruntime.ConfigPathToken {
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
	if len(outbounds) == 0 {
		t.Fatal("configuration has no outbound")
	}
	return object(t, outbounds[0])
}

func assertLoopbackConfig(t *testing.T, config map[string]any) {
	t.Helper()
	inbounds := array(t, config["inbounds"])
	inbound := object(t, inbounds[0])
	if inbound["type"] != "socks" || inbound["listen"] != "127.0.0.1" || inbound["listen_port"] != float64(19080) {
		t.Fatalf("unexpected loopback inbound: %#v", inbound)
	}
	route := object(t, config["route"])
	if route["final"] != "proxy" || route["default_domain_resolver"] != "local" {
		t.Fatalf("unexpected route: %#v", route)
	}
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
