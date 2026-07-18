package evidence

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxSubmissionBytes = 256 << 10

type CollectorOptions struct {
	RequestedSurfaces []string
}

type submissionResult struct {
	submission BrowserSubmission
	err        error
}

type Collector struct {
	listener net.Listener
	server   *http.Server
	origin   string
	token    string
	nonce    string
	surfaces []string

	mu       sync.Mutex
	consumed bool
	closed   bool

	result   chan submissionResult
	serveErr chan error
}

func StartCollector(options CollectorOptions) (*Collector, error) {
	surfaces, err := normalizeRequestedSurfaces(options.RequestedSurfaces)
	if err != nil {
		return nil, err
	}
	token, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	nonce, err := randomNonce()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start loopback evidence collector: %w", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !address.IP.IsLoopback() {
		_ = listener.Close()
		return nil, fmt.Errorf("evidence collector did not bind to loopback")
	}
	collector := &Collector{
		listener: listener,
		origin:   "http://" + listener.Addr().String(),
		token:    token,
		nonce:    nonce,
		surfaces: surfaces,
		result:   make(chan submissionResult, 1),
		serveErr: make(chan error, 1),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /run/"+token, collector.handleRun)
	mux.HandleFunc("GET /frame/"+token, collector.handleFrame)
	mux.HandleFunc("GET /worker/"+token+".js", collector.handleWorker)
	mux.HandleFunc("POST /submit/"+token, collector.handleSubmit)
	collector.server = &http.Server{
		Handler:           collector.loopbackOnly(mux),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       4 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       4 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		err := collector.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			collector.serveErr <- err
		}
	}()
	return collector, nil
}

func (c *Collector) URL() string { return c.origin + "/run/" + c.token }

func (c *Collector) Origin() string { return c.origin }

func (c *Collector) Wait(ctx context.Context) (BrowserSubmission, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case result := <-c.result:
		return result.submission, result.err
	case err := <-c.serveErr:
		return BrowserSubmission{}, fmt.Errorf("evidence collector stopped: %w", err)
	case <-ctx.Done():
		return BrowserSubmission{}, ctx.Err()
	}
}

func (c *Collector) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	if err := c.server.Shutdown(ctx); err != nil {
		_ = c.listener.Close()
		return fmt.Errorf("stop evidence collector: %w", err)
	}
	return nil
}

func (c *Collector) loopbackOnly(next http.Handler) http.Handler {
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
		if request.Host != c.listener.Addr().String() {
			http.Error(writer, "invalid host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (c *Collector) handleRun(writer http.ResponseWriter, _ *http.Request) {
	c.writeHTML(writer, renderRunPage(c.nonce, c.surfaces, c.origin, c.token))
}

func (c *Collector) handleFrame(writer http.ResponseWriter, _ *http.Request) {
	c.writeHTML(writer, renderFramePage(c.nonce, c.surfaces, c.origin))
}

func (c *Collector) handleWorker(writer http.ResponseWriter, _ *http.Request) {
	setEvidenceHeaders(writer.Header())
	writer.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	_, _ = io.WriteString(writer, renderWorkerScript())
}

func (c *Collector) writeHTML(writer http.ResponseWriter, body string) {
	setEvidenceHeaders(writer.Header())
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'nonce-"+c.nonce+"'; style-src 'nonce-"+c.nonce+"'; connect-src 'self'; frame-src 'self'; worker-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	_, _ = io.WriteString(writer, body)
}

func (c *Collector) handleSubmit(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != c.origin {
		http.Error(writer, "invalid origin", http.StatusForbidden)
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(writer, "application/json required", http.StatusUnsupportedMediaType)
		return
	}
	c.mu.Lock()
	if c.consumed || c.closed {
		c.mu.Unlock()
		http.Error(writer, "submission already consumed", http.StatusGone)
		return
	}
	c.consumed = true
	c.mu.Unlock()

	reader := http.MaxBytesReader(writer, request.Body, maxSubmissionBytes)
	defer request.Body.Close()
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var submission BrowserSubmission
	if err := decoder.Decode(&submission); err != nil {
		c.finish(BrowserSubmission{}, fmt.Errorf("decode browser evidence submission: %w", err))
		http.Error(writer, "invalid evidence submission", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		c.finish(BrowserSubmission{}, fmt.Errorf("browser evidence submission contains trailing data"))
		http.Error(writer, "invalid evidence submission", http.StatusBadRequest)
		return
	}
	submission = normalizeSubmission(submission)
	if err := submission.Validate(); err != nil {
		c.finish(BrowserSubmission{}, fmt.Errorf("validate browser evidence submission: %w", err))
		http.Error(writer, "invalid evidence submission", http.StatusBadRequest)
		return
	}
	c.finish(submission, nil)
	writer.WriteHeader(http.StatusNoContent)
}

func (c *Collector) finish(submission BrowserSubmission, err error) {
	select {
	case c.result <- submissionResult{submission: submission, err: err}:
	default:
	}
}

func setEvidenceHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), bluetooth=(), clipboard-read=(), clipboard-write=()")
}

func normalizeRequestedSurfaces(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !allowedSurface(value) {
			return nil, fmt.Errorf("unsupported requested surface %q", value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return sortedUnique(result), nil
}

func randomHex(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate evidence token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func randomNonce() (string, error) {
	buffer := make([]byte, 18)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate evidence CSP nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
