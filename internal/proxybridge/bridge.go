package proxybridge

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	xproxy "golang.org/x/net/proxy"
)

const (
	maxHeaderBytes = 64 << 10
	connectTimeout = 12 * time.Second
)

type Instance interface {
	URL() string
	Kind() string
	Health(context.Context) error
	Close() error
}

type Factory interface {
	Start(context.Context, string, credential.Material) (Instance, error)
}

type DefaultFactory struct{}

func (DefaultFactory) Start(ctx context.Context, upstream string, material credential.Material) (Instance, error) {
	return Start(ctx, upstream, material)
}

type Bridge struct {
	listener  net.Listener
	server    *http.Server
	transport *http.Transport
	dial      func(context.Context, string) (net.Conn, error)
	url       string
	kind      string
	closeOnce sync.Once
	closeErr  error
}

func Start(ctx context.Context, upstreamRaw string, material credential.Material) (*Bridge, error) {
	upstream, err := parseUpstream(upstreamRaw, material)
	if err != nil {
		return nil, err
	}
	transport, tunnelDialer, kind, err := buildUpstream(upstream, material)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("listen on loopback proxy bridge: %w", err)
	}
	address := listener.Addr().String()
	bridge := &Bridge{
		listener:  listener,
		transport: transport,
		dial:      tunnelDialer,
		url:       "http://" + address,
		kind:      kind,
	}
	bridge.server = &http.Server{
		Handler:           bridge,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	go func() {
		_ = bridge.server.Serve(listener)
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := bridge.Health(healthCtx); err != nil {
		_ = bridge.Close()
		return nil, err
	}
	return bridge, nil
}

func (b *Bridge) URL() string  { return b.url }
func (b *Bridge) Kind() string { return b.kind }

func (b *Bridge) Health(ctx context.Context) error {
	if b == nil || b.listener == nil {
		return fmt.Errorf("proxy bridge is not initialized")
	}
	dialer := net.Dialer{Timeout: 2 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp4", b.listener.Addr().String())
	if err != nil {
		return fmt.Errorf("proxy bridge health check failed: %w", err)
	}
	return connection.Close()
}

func (b *Bridge) Close() error {
	if b == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if b.server != nil {
			if err := b.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				b.closeErr = err
				_ = b.server.Close()
			}
		}
		if b.listener != nil {
			_ = b.listener.Close()
		}
		if b.transport != nil {
			b.transport.CloseIdleConnections()
		}
	})
	return b.closeErr
}

func (b *Bridge) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodConnect {
		b.handleConnect(response, request)
		return
	}
	b.handleHTTP(response, request)
}

func (b *Bridge) handleHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL == nil || request.URL.Host == "" || request.URL.Scheme == "" {
		http.Error(response, "absolute proxy URL required", http.StatusBadRequest)
		return
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		http.Error(response, "unsupported proxy request scheme", http.StatusBadRequest)
		return
	}
	outbound := request.Clone(request.Context())
	outbound.RequestURI = ""
	outbound.URL.User = nil
	outbound.Header = request.Header.Clone()
	stripHopHeaders(outbound.Header)
	outbound.Header.Del("Proxy-Authorization")
	result, err := b.transport.RoundTrip(outbound)
	if err != nil {
		http.Error(response, "upstream proxy request failed", http.StatusBadGateway)
		return
	}
	defer result.Body.Close()
	stripHopHeaders(result.Header)
	for key, values := range result.Header {
		for _, value := range values {
			response.Header().Add(key, value)
		}
	}
	response.WriteHeader(result.StatusCode)
	_, _ = io.Copy(response, result.Body)
}

func (b *Bridge) handleConnect(response http.ResponseWriter, request *http.Request) {
	target, err := canonicalTarget(request.Host)
	if err != nil {
		http.Error(response, "invalid CONNECT target", http.StatusBadRequest)
		return
	}
	upstream, err := b.dial(request.Context(), target)
	if err != nil {
		http.Error(response, "upstream proxy tunnel failed", http.StatusBadGateway)
		return
	}
	hijacker, ok := response.(http.Hijacker)
	if !ok {
		_ = upstream.Close()
		http.Error(response, "proxy hijacking unavailable", http.StatusInternalServerError)
		return
	}
	client, buffered, err := hijacker.Hijack()
	if err != nil {
		_ = upstream.Close()
		return
	}
	if _, err := buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		_ = client.Close()
		_ = upstream.Close()
		return
	}
	if err := buffered.Flush(); err != nil {
		_ = client.Close()
		_ = upstream.Close()
		return
	}
	go relay(client, upstream)
}

func parseUpstream(raw string, material credential.Material) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if strings.TrimSpace(material.Username) == "" || material.Secret == "" {
		return nil, fmt.Errorf("proxy credential material is incomplete")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid authenticated upstream proxy URL")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("upstream proxy URL must not contain inline credentials")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	switch parsed.Scheme {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("authenticated proxy bridge does not support scheme %q", parsed.Scheme)
	}
	if _, _, err := net.SplitHostPort(parsed.Host); err != nil {
		return nil, fmt.Errorf("upstream proxy must include an explicit port")
	}
	return parsed, nil
}

func buildUpstream(upstream *url.URL, material credential.Material) (*http.Transport, func(context.Context, string) (net.Conn, error), string, error) {
	baseDialer := &net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          32,
		IdleConnTimeout:       45 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if upstream.Scheme == "socks5" {
		dialer, err := xproxy.SOCKS5("tcp", upstream.Host, &xproxy.Auth{User: material.Username, Password: material.Secret}, baseDialer)
		if err != nil {
			return nil, nil, "", fmt.Errorf("create SOCKS5 upstream dialer: %w", err)
		}
		dialContext := contextDialer(dialer)
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialContext(ctx, address)
		}
		return transport, dialContext, "socks5-auth", nil
	}
	authenticated := *upstream
	authenticated.User = url.UserPassword(material.Username, material.Secret)
	transport.Proxy = http.ProxyURL(&authenticated)
	dialTunnel := func(ctx context.Context, target string) (net.Conn, error) {
		return connectThroughHTTPProxy(ctx, upstream, material, target)
	}
	return transport, dialTunnel, upstream.Scheme + "-auth", nil
}

func contextDialer(dialer xproxy.Dialer) func(context.Context, string) (net.Conn, error) {
	if contextual, ok := dialer.(xproxy.ContextDialer); ok {
		return func(ctx context.Context, address string) (net.Conn, error) {
			return contextual.DialContext(ctx, "tcp", address)
		}
	}
	return func(ctx context.Context, address string) (net.Conn, error) {
		type result struct {
			connection net.Conn
			err        error
		}
		channel := make(chan result, 1)
		go func() {
			connection, err := dialer.Dial("tcp", address)
			channel <- result{connection: connection, err: err}
		}()
		select {
		case value := <-channel:
			return value.connection, value.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func connectThroughHTTPProxy(ctx context.Context, upstream *url.URL, material credential.Material, target string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", upstream.Host)
	if err != nil {
		return nil, fmt.Errorf("connect to upstream proxy: %w", err)
	}
	if upstream.Scheme == "https" {
		tlsConnection := tls.Client(connection, &tls.Config{ServerName: upstream.Hostname(), MinVersion: tls.VersionTLS12})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("authenticate upstream proxy TLS: %w", err)
		}
		connection = tlsConnection
	}
	request := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: target},
		Host:   target,
		Header: make(http.Header),
	}
	token := base64.StdEncoding.EncodeToString([]byte(material.Username + ":" + material.Secret))
	request.Header.Set("Proxy-Authorization", "Basic "+token)
	request.Header.Set("User-Agent", "Veilium")
	if err := request.Write(connection); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("write upstream CONNECT request: %w", err)
	}
	result, err := http.ReadResponse(bufio.NewReader(connection), request)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("read upstream CONNECT response: %w", err)
	}
	if result.Body != nil {
		_ = result.Body.Close()
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		_ = connection.Close()
		return nil, fmt.Errorf("upstream proxy rejected CONNECT with status %d", result.StatusCode)
	}
	return connection, nil
}

func canonicalTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("invalid target")
	}
	if _, _, err := net.SplitHostPort(value); err == nil {
		return value, nil
	}
	if strings.Contains(value, ":") {
		return "", fmt.Errorf("invalid target")
	}
	return net.JoinHostPort(value, "443"), nil
}

func stripHopHeaders(header http.Header) {
	for _, value := range header.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			header.Del(strings.TrimSpace(token))
		}
	}
	for _, name := range []string{"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"} {
		header.Del(name)
	}
}

func relay(left, right net.Conn) {
	defer left.Close()
	defer right.Close()
	done := make(chan struct{}, 2)
	copySide := func(destination, source net.Conn) {
		_, _ = io.Copy(destination, source)
		if closer, ok := destination.(interface{ CloseWrite() error }); ok {
			_ = closer.CloseWrite()
		}
		done <- struct{}{}
	}
	go copySide(left, right)
	go copySide(right, left)
	<-done
}
