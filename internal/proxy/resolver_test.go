package proxy

import (
	"strings"
	"testing"
)

func TestRejectsInlineProxyCredentials(t *testing.T) {
	_, err := Resolve("socks5://user:secret@127.0.0.1:1080", "")
	if err == nil || !strings.Contains(err.Error(), "inline proxy credentials") {
		t.Fatalf("expected inline credential rejection, got %v", err)
	}
}

func TestCredentialReferenceRequiresLocalBridge(t *testing.T) {
	route, err := Resolve("socks5://127.0.0.1:1080", "os-vault://proxy/account-a")
	if err != nil {
		t.Fatal(err)
	}
	if !route.RequiresBridge || route.BridgeKind != "local-auth-bridge" {
		t.Fatalf("unexpected route: %+v", route)
	}
	if route.CredentialRef == "" {
		t.Fatal("credential reference was lost")
	}
}

func TestAdvancedProtocolUsesDedicatedBridge(t *testing.T) {
	route, err := Resolve("vless://example.com:443", "os-vault://proxy/vless-a")
	if err != nil {
		t.Fatal(err)
	}
	if route.BridgeKind != "xray" {
		t.Fatalf("expected xray bridge, got %+v", route)
	}
}
