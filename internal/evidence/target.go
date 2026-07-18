package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxTargetResponseBytes = 128 << 10

var targetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type Target struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type TargetClient struct {
	client *http.Client
}

func NewTargetClient() *TargetClient {
	return &TargetClient{client: &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			Proxy:               nil,
			DisableKeepAlives:   true,
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
		},
		CheckRedirect: rejectTargetRedirect,
	}}
}

func NewTargetClientWithHTTP(client *http.Client) (*TargetClient, error) {
	if client == nil {
		return nil, fmt.Errorf("CDP target HTTP client is required")
	}
	copy := *client
	copy.CheckRedirect = rejectTargetRedirect
	return &TargetClient{client: &copy}, nil
}

func (c *TargetClient) Open(ctx context.Context, port int, controlledURL string) (Target, error) {
	if c == nil || c.client == nil {
		return Target{}, fmt.Errorf("CDP target client is unavailable")
	}
	if port < 1 || port > 65535 {
		return Target{}, fmt.Errorf("invalid CDP port %d", port)
	}
	if err := validateControlledURL(controlledURL); err != nil {
		return Target{}, err
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/new?%s", port, url.QueryEscape(controlledURL))
	request, err := http.NewRequestWithContext(nonNilContext(ctx), http.MethodPut, endpoint, nil)
	if err != nil {
		return Target{}, fmt.Errorf("create CDP target request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return Target{}, fmt.Errorf("open CDP evidence target: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return Target{}, fmt.Errorf("CDP target endpoint returned %s", response.Status)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxTargetResponseBytes+1))
	if err != nil {
		return Target{}, fmt.Errorf("read CDP target response: %w", err)
	}
	if len(payload) > maxTargetResponseBytes {
		return Target{}, fmt.Errorf("CDP target response exceeds %d bytes", maxTargetResponseBytes)
	}
	var target Target
	if err := json.Unmarshal(payload, &target); err != nil {
		return Target{}, fmt.Errorf("decode CDP target response: %w", err)
	}
	if err := validateTarget(target, port); err != nil {
		return Target{}, err
	}
	return target, nil
}

func (c *TargetClient) Close(ctx context.Context, port int, targetID string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("CDP target client is unavailable")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid CDP port %d", port)
	}
	if !targetIDPattern.MatchString(targetID) {
		return fmt.Errorf("invalid CDP target id")
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/close/%s", port, targetID)
	request, err := http.NewRequestWithContext(nonNilContext(ctx), http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create CDP close request: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("close CDP evidence target: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("CDP close endpoint returned %s", response.Status)
	}
	return nil
}

func validateControlledURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("parse controlled evidence URL: %w", err)
	}
	if parsed.Scheme != "http" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("controlled evidence URL must be plain loopback HTTP")
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		return fmt.Errorf("controlled evidence URL requires an explicit loopback port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("controlled evidence URL must use a loopback IP")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("controlled evidence URL has an invalid port")
	}
	if !strings.HasPrefix(parsed.EscapedPath(), "/run/") || len(parsed.EscapedPath()) > 256 {
		return fmt.Errorf("controlled evidence URL has an invalid path")
	}
	return nil
}

func validateTarget(target Target, port int) error {
	if !targetIDPattern.MatchString(target.ID) {
		return fmt.Errorf("CDP target response has an invalid id")
	}
	if target.Type != "page" {
		return fmt.Errorf("CDP target response has unexpected type %q", target.Type)
	}
	if strings.TrimSpace(target.WebSocketDebuggerURL) != "" {
		parsed, err := url.Parse(target.WebSocketDebuggerURL)
		if err != nil {
			return fmt.Errorf("parse CDP target websocket URL: %w", err)
		}
		if parsed.Scheme != "ws" {
			return fmt.Errorf("CDP target websocket must use ws")
		}
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("CDP target websocket is not loopback-only")
		}
		if parsed.Port() != strconv.Itoa(port) {
			return fmt.Errorf("CDP target websocket does not use the selected port")
		}
	}
	return nil
}

func rejectTargetRedirect(*http.Request, []*http.Request) error {
	return errors.New("CDP target endpoint must not redirect")
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
