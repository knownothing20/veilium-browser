package networkevidence

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxBrowserSubmissionBytes = 128 << 10

type browserSubmissionResult struct {
	submission BrowserSubmission
	err        error
}

type Collector struct {
	listener net.Listener
	server   *http.Server
	origin   string
	token    string
	dnsToken string
	probeSet ProbeSet

	mu       sync.Mutex
	consumed bool
	closed   bool

	result   chan browserSubmissionResult
	serveErr chan error
}

func StartCollector(set ProbeSet) (*Collector, error) {
	set = NormalizeProbeSet(set)
	if err := set.Validate(); err != nil {
		return nil, err
	}
	token, err := randomCollectorHex(32)
	if err != nil {
		return nil, err
	}
	dnsToken, err := randomCollectorHex(16)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start loopback network evidence collector: %w", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !address.IP.IsLoopback() {
		_ = listener.Close()
		return nil, fmt.Errorf("network evidence collector did not bind to loopback")
	}
	collector := &Collector{
		listener: listener,
		origin:   "http://" + listener.Addr().String(),
		token:    token,
		dnsToken: dnsToken,
		probeSet: set,
		result:   make(chan browserSubmissionResult, 1),
		serveErr: make(chan error, 1),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /run/"+token, collector.handleRun)
	mux.HandleFunc("GET /script/"+token+".js", collector.handleScript)
	mux.HandleFunc("GET /config/"+token+".json", collector.handleConfig)
	mux.HandleFunc("POST /submit/"+token, collector.handleSubmit)
	collector.server = &http.Server{
		Handler:           collector.loopbackOnly(mux),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		err := collector.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case collector.serveErr <- err:
			default:
			}
		}
	}()
	return collector, nil
}

func (collector *Collector) URL() string {
	return collector.origin + "/run/" + collector.token
}

func (collector *Collector) Wait(ctx context.Context) (BrowserSubmission, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case result := <-collector.result:
		return result.submission, result.err
	case err := <-collector.serveErr:
		return BrowserSubmission{}, fmt.Errorf("network evidence collector stopped: %w", err)
	case <-ctx.Done():
		return BrowserSubmission{}, ctx.Err()
	}
}

func (collector *Collector) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	collector.mu.Lock()
	if collector.closed {
		collector.mu.Unlock()
		return nil
	}
	collector.closed = true
	collector.mu.Unlock()
	if err := collector.server.Shutdown(ctx); err != nil {
		_ = collector.listener.Close()
		return fmt.Errorf("stop network evidence collector: %w", err)
	}
	return nil
}

func (collector *Collector) loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		if err != nil {
			http.Error(writer, "invalid remote address", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(writer, "loopback only", http.StatusForbidden)
			return
		}
		if request.Host != collector.listener.Addr().String() {
			http.Error(writer, "invalid host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (collector *Collector) handleRun(writer http.ResponseWriter, _ *http.Request) {
	setNetworkEvidenceHeaders(writer.Header())
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Content-Security-Policy", collector.contentSecurityPolicy())
	_, _ = io.WriteString(writer, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Veilium network evidence</title></head><body><main id=\"status\">Collecting controlled network evidence…</main><script src=\"/script/"+collector.token+".js\"></script></body></html>")
}

func (collector *Collector) handleScript(writer http.ResponseWriter, _ *http.Request) {
	setNetworkEvidenceHeaders(writer.Header())
	writer.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	_, _ = io.WriteString(writer, browserProbeScript)
}

func (collector *Collector) handleConfig(writer http.ResponseWriter, _ *http.Request) {
	setNetworkEvidenceHeaders(writer.Header())
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	config := browserProbeConfig{
		SchemaVersion: BrowserSubmissionSchemaVersion,
		SubmitURL:     collector.origin + "/submit/" + collector.token,
		DNSToken:      collector.dnsToken,
		Definitions:   browserProbeDefinitions(collector.probeSet),
	}
	if err := json.NewEncoder(writer).Encode(config); err != nil {
		collector.finish(BrowserSubmission{}, fmt.Errorf("encode network evidence browser config: %w", err))
	}
}

func (collector *Collector) handleSubmit(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != collector.origin {
		http.Error(writer, "invalid origin", http.StatusForbidden)
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(writer, "application/json required", http.StatusUnsupportedMediaType)
		return
	}
	collector.mu.Lock()
	if collector.consumed || collector.closed {
		collector.mu.Unlock()
		http.Error(writer, "submission already consumed", http.StatusGone)
		return
	}
	collector.consumed = true
	collector.mu.Unlock()

	reader := http.MaxBytesReader(writer, request.Body, maxBrowserSubmissionBytes)
	defer request.Body.Close()
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var submission BrowserSubmission
	if err := decoder.Decode(&submission); err != nil {
		collector.finish(BrowserSubmission{}, fmt.Errorf("decode browser network submission: %w", err))
		http.Error(writer, "invalid network evidence submission", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		collector.finish(BrowserSubmission{}, fmt.Errorf("browser network submission contains trailing data"))
		http.Error(writer, "invalid network evidence submission", http.StatusBadRequest)
		return
	}
	submission = NormalizeBrowserSubmission(submission)
	if err := submission.Validate(collector.probeSet); err != nil {
		collector.finish(BrowserSubmission{}, fmt.Errorf("validate browser network submission: %w", err))
		http.Error(writer, "invalid network evidence submission", http.StatusBadRequest)
		return
	}
	collector.finish(submission, nil)
	writer.WriteHeader(http.StatusNoContent)
}

func (collector *Collector) finish(submission BrowserSubmission, err error) {
	select {
	case collector.result <- browserSubmissionResult{submission: submission, err: err}:
	default:
	}
}

func (collector *Collector) contentSecurityPolicy() string {
	connectSources := []string{"'self'"}
	imageSources := []string{"'self'"}
	for _, definition := range collector.probeSet.Definitions {
		for _, raw := range []string{definition.HTTPSURL, definition.DNSResultURL} {
			if origin := probeOrigin(raw); origin != "" {
				connectSources = append(connectSources, origin)
			}
		}
		if definition.Kind == ProbeDelegatedDNS {
			zone := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(definition.DNSZone)), ".")
			if zone != "" {
				imageSources = append(imageSources, "https://*."+zone)
				connectSources = append(connectSources, "https://*."+zone)
			}
		}
	}
	connectSources = uniqueStrings(connectSources)
	imageSources = uniqueStrings(imageSources)
	return "default-src 'none'; script-src 'self'; connect-src " + strings.Join(connectSources, " ") + "; img-src " + strings.Join(imageSources, " ") + "; style-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
}

func probeOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func setNetworkEvidenceHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), bluetooth=(), clipboard-read=(), clipboard-write=()")
}

func randomCollectorHex(byteCount int) (string, error) {
	buffer := make([]byte, byteCount)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate network evidence collector token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
