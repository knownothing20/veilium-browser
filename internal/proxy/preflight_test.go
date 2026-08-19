package proxy

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestCheckReachableAcceptsListeningProxyEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	route, err := Resolve("http://"+listener.Addr().String(), "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := CheckReachable(ctx, route); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReachableRejectsUnavailableProxyEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	route, err := Resolve("socks5://"+address, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := CheckReachable(ctx, route); err == nil {
		t.Fatal("expected unavailable proxy rejection")
	}
}

func TestCheckReachableSkipsDirectAndManagedBridgeRoutes(t *testing.T) {
	direct, _ := Resolve("direct://", "")
	if err := CheckReachable(context.Background(), direct); err != nil {
		t.Fatal(err)
	}
	bridge, _ := Resolve("vless://proxy.example:443", "credential-1")
	if err := CheckReachable(context.Background(), bridge); err != nil {
		t.Fatal(err)
	}
}
