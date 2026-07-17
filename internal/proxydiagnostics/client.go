package proxydiagnostics

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

func diagnosticClient(proxyURL *url.URL) *http.Client {
	transport := baseTransport()
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return clientWithTransport(transport)
}

func baseTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       15 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 12 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
}

func clientWithTransport(transport *http.Transport) *http.Client {
	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("diagnostic endpoint redirects are disabled")
		},
	}
}

type probeResult struct {
	IP         string
	StatusCode int
	FirstByte  time.Duration
	Total      time.Duration
}

func runProbe(
	ctx context.Context,
	client *http.Client,
	endpoint string,
) (probeResult, error) {
	started := time.Now()
	firstByteAt := time.Time{}
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			if firstByteAt.IsZero() {
				firstByteAt = time.Now()
			}
		},
	}
	request, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return probeResult{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Veilium-Proxy-Diagnostics/1")
	response, err := client.Do(request)
	if err != nil {
		return probeResult{}, fmt.Errorf("proxy probe failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(
			io.Discard,
			io.LimitReader(response.Body, maxProbeBytes),
		)
		return probeResult{}, fmt.Errorf(
			"proxy probe returned HTTP %d",
			response.StatusCode,
		)
	}
	body, err := io.ReadAll(
		io.LimitReader(response.Body, maxProbeBytes+1),
	)
	if err != nil {
		return probeResult{}, fmt.Errorf(
			"read proxy probe response: %w",
			err,
		)
	}
	if len(body) > maxProbeBytes {
		return probeResult{}, fmt.Errorf(
			"proxy probe response exceeded the size limit",
		)
	}
	var payload struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return probeResult{}, fmt.Errorf(
			"decode exit-IP response: %w",
			err,
		)
	}
	ip := net.ParseIP(strings.TrimSpace(payload.IP))
	if ip == nil {
		return probeResult{}, fmt.Errorf(
			"exit-IP response did not contain a valid address",
		)
	}
	if firstByteAt.IsZero() {
		firstByteAt = time.Now()
	}
	return probeResult{
		IP:         ip.String(),
		StatusCode: response.StatusCode,
		FirstByte:  firstByteAt.Sub(started),
		Total:      time.Since(started),
	}, nil
}
