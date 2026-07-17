package proxybridge

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
)

func TestHTTPBridgeForwardsAuthenticatedHTTPAndCONNECT(t *testing.T) {
	echo := startEchoServer(t)
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Proxy-Authorization") != expectedAuth {
			http.Error(response, "authentication required", http.StatusProxyAuthRequired)
			return
		}
		if request.Method != http.MethodConnect {
			response.Header().Set("X-Upstream-Authenticated", "yes")
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte("authenticated-http"))
			return
		}
		target, err := net.Dial("tcp", request.Host)
		if err != nil {
			http.Error(response, "dial failed", http.StatusBadGateway)
			return
		}
		hijacker := response.(http.Hijacker)
		client, buffered, err := hijacker.Hijack()
		if err != nil {
			_ = target.Close()
			return
		}
		_, _ = buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		_ = buffered.Flush()
		go relay(client, target)
	}))
	defer upstream.Close()

	bridge, err := Start(context.Background(), upstream.URL, credential.Material{Username: "alice", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	assertLoopbackBridge(t, bridge)

	bridgeURL, _ := url.Parse(bridge.URL())
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(bridgeURL)}, Timeout: 5 * time.Second}
	result, err := client.Get("http://example.test/resource")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(result.Body)
	_ = result.Body.Close()
	if result.StatusCode != http.StatusOK || string(body) != "authenticated-http" || result.Header.Get("X-Upstream-Authenticated") != "yes" {
		t.Fatalf("unexpected HTTP bridge response: status=%d body=%q", result.StatusCode, body)
	}

	connection, err := net.DialTimeout("tcp", strings.TrimPrefix(bridge.URL(), "http://"), 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_, _ = fmt.Fprintf(connection, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", echo, echo)
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("CONNECT failed: %s", response.Status)
	}
	assertEcho(t, connection)
}

func TestSOCKS5BridgeUsesUsernamePasswordAuthentication(t *testing.T) {
	echo := startEchoServer(t)
	upstream := startSOCKS5Server(t, "bob", "password")
	bridge, err := Start(context.Background(), "socks5://"+upstream, credential.Material{Username: "bob", Secret: "password"})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	if bridge.Kind() != "socks5-auth" {
		t.Fatalf("unexpected bridge kind: %s", bridge.Kind())
	}
	connection, err := net.DialTimeout("tcp", strings.TrimPrefix(bridge.URL(), "http://"), 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_, _ = fmt.Fprintf(connection, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", echo, echo)
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("SOCKS5 CONNECT failed: %s", response.Status)
	}
	assertEcho(t, connection)
}

func TestBridgeRejectsInlineCredentialsAndUnsupportedSchemes(t *testing.T) {
	for _, raw := range []string{"http://user:pass@127.0.0.1:8080", "vmess://127.0.0.1:8080", "http://127.0.0.1"} {
		if _, err := Start(context.Background(), raw, credential.Material{Username: "user", Secret: "secret"}); err == nil {
			t.Fatalf("expected rejection for %s", raw)
		}
	}
	if _, err := Start(context.Background(), "http://127.0.0.1:8080", credential.Material{}); err == nil {
		t.Fatal("expected incomplete credential rejection")
	}
}

func assertLoopbackBridge(t *testing.T, bridge *Bridge) {
	t.Helper()
	parsed, err := url.Parse(bridge.URL())
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("bridge is not loopback-only: %s", bridge.URL())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bridge.Health(ctx); err != nil {
		t.Fatal(err)
	}
}

func assertEcho(t *testing.T, connection net.Conn) {
	t.Helper()
	_ = connection.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := connection.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(connection, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("unexpected echo: %q", buffer)
	}
}

func startEchoServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer connection.Close()
				_, _ = io.Copy(connection, connection)
			}()
		}
	}()
	return listener.Addr().String()
}

func startSOCKS5Server(t *testing.T, username, password string) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go serveSOCKS5(connection, username, password)
		}
	}()
	return listener.Addr().String()
}

func serveSOCKS5(connection net.Conn, username, password string) {
	defer connection.Close()
	reader := bufio.NewReader(connection)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil || header[0] != 5 {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}
	_, _ = connection.Write([]byte{5, 2})
	authHeader := make([]byte, 2)
	if _, err := io.ReadFull(reader, authHeader); err != nil || authHeader[0] != 1 {
		return
	}
	user := make([]byte, int(authHeader[1]))
	if _, err := io.ReadFull(reader, user); err != nil {
		return
	}
	length, err := reader.ReadByte()
	if err != nil {
		return
	}
	secret := make([]byte, int(length))
	if _, err := io.ReadFull(reader, secret); err != nil {
		return
	}
	if string(user) != username || string(secret) != password {
		_, _ = connection.Write([]byte{1, 1})
		return
	}
	_, _ = connection.Write([]byte{1, 0})
	request := make([]byte, 4)
	if _, err := io.ReadFull(reader, request); err != nil || request[0] != 5 || request[1] != 1 {
		return
	}
	host, err := readSOCKSAddress(reader, request[3])
	if err != nil {
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	port := int(portBytes[0])<<8 | int(portBytes[1])
	target, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 5*time.Second)
	if err != nil {
		_, _ = connection.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()
	_, _ = connection.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
	relay(connection, target)
}

func readSOCKSAddress(reader *bufio.Reader, addressType byte) (string, error) {
	switch addressType {
	case 1:
		value := make([]byte, 4)
		_, err := io.ReadFull(reader, value)
		return net.IP(value).String(), err
	case 3:
		length, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		value := make([]byte, int(length))
		_, err = io.ReadFull(reader, value)
		return string(value), err
	case 4:
		value := make([]byte, 16)
		_, err := io.ReadFull(reader, value)
		return net.IP(value).String(), err
	default:
		return "", fmt.Errorf("unsupported address type")
	}
}
