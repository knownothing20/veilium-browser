package windowcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/knownothing20/veilium-browser/internal/domain"
)

const (
	maxTargetListBytes = 128 << 10
	maxCDPMessageBytes  = 128 << 10
	maxSkippedMessages  = 32
)

type Controller struct {
	httpClient *http.Client
	dialer     *websocket.Dialer
	timeout    time.Duration
}

type targetInfo struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type cdpRequest struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type cdpResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type windowResult struct {
	WindowID int          `json:"windowId"`
	Bounds   windowBounds `json:"bounds"`
}

type windowBounds struct {
	Left        int    `json:"left,omitempty"`
	Top         int    `json:"top,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	WindowState string `json:"windowState,omitempty"`
}

func New() *Controller {
	return &Controller{
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("CDP window endpoint must not redirect")
			},
		},
		dialer:  &websocket.Dialer{Proxy: nil, HandshakeTimeout: 3 * time.Second},
		timeout: 4 * time.Second,
	}
}

func NewWithClients(httpClient *http.Client, dialer *websocket.Dialer, timeout time.Duration) (*Controller, error) {
	if httpClient == nil || dialer == nil {
		return nil, fmt.Errorf("window controller clients are required")
	}
	if timeout <= 0 || timeout > 30*time.Second {
		return nil, fmt.Errorf("window controller timeout is invalid")
	}
	return &Controller{httpClient: httpClient, dialer: dialer, timeout: timeout}, nil
}

func (c *Controller) Apply(ctx context.Context, cdpPort int, browserWebSocket string, plan domain.WindowPlan) (domain.WindowState, error) {
	if c == nil || c.httpClient == nil || c.dialer == nil {
		return domain.WindowState{}, fmt.Errorf("window controller is unavailable")
	}
	if cdpPort < 1 || cdpPort > 65535 {
		return domain.WindowState{}, fmt.Errorf("invalid CDP port %d", cdpPort)
	}
	if plan.Width < 320 || plan.Width > 16384 || plan.Height < 240 || plan.Height > 16384 {
		return domain.WindowState{}, fmt.Errorf("window plan dimensions are outside the controlled range")
	}
	if plan.DeviceScaleFactor < 0.5 || plan.DeviceScaleFactor > 8 {
		return domain.WindowState{}, fmt.Errorf("window plan device scale factor is outside the controlled range")
	}
	if err := validateBrowserWebSocket(browserWebSocket, cdpPort); err != nil {
		return domain.WindowState{}, err
	}
	ctx, cancel := context.WithTimeout(nonNilContext(ctx), c.timeout)
	defer cancel()
	targetID, err := c.firstPageTarget(ctx, cdpPort)
	if err != nil {
		return domain.WindowState{}, err
	}
	connection, response, err := c.dialer.DialContext(ctx, browserWebSocket, http.Header{"Origin": []string{"http://127.0.0.1"}})
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return domain.WindowState{}, fmt.Errorf("connect browser CDP window controller: %w", err)
	}
	defer connection.Close()
	connection.SetReadLimit(maxCDPMessageBytes)
	deadline, ok := ctx.Deadline()
	if ok {
		_ = connection.SetReadDeadline(deadline)
		_ = connection.SetWriteDeadline(deadline)
	}

	var current windowResult
	if err := call(connection, 1, "Browser.getWindowForTarget", map[string]any{"targetId": targetID}, &current); err != nil {
		return domain.WindowState{}, err
	}
	if current.WindowID < 1 {
		return domain.WindowState{}, fmt.Errorf("browser returned an invalid window id")
	}
	if err := call(connection, 2, "Browser.setWindowBounds", map[string]any{
		"windowId": current.WindowID,
		"bounds": map[string]any{
			"windowState": "normal",
			"width":       plan.Width,
			"height":      plan.Height,
		},
	}, nil); err != nil {
		return domain.WindowState{}, err
	}
	var observed struct {
		Bounds windowBounds `json:"bounds"`
	}
	if err := call(connection, 3, "Browser.getWindowBounds", map[string]any{"windowId": current.WindowID}, &observed); err != nil {
		return domain.WindowState{}, err
	}
	if observed.Bounds.Width < 1 || observed.Bounds.Height < 1 {
		return domain.WindowState{}, fmt.Errorf("browser returned invalid controlled window bounds")
	}
	return domain.WindowState{
		Width:             observed.Bounds.Width,
		Height:            observed.Bounds.Height,
		Left:              observed.Bounds.Left,
		Top:               observed.Bounds.Top,
		DeviceScaleFactor: plan.DeviceScaleFactor,
		State:             observed.Bounds.WindowState,
		Applied:           true,
	}, nil
}

func (c *Controller) firstPageTarget(ctx context.Context, port int) (string, error) {
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create CDP target-list request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("read CDP target list: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("CDP target-list endpoint returned %s", response.Status)
	}
	var targets []targetInfo
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxTargetListBytes+1))
	if err := decoder.Decode(&targets); err != nil {
		return "", fmt.Errorf("decode CDP target list: %w", err)
	}
	for _, target := range targets {
		if target.Type == "page" && validTargetID(target.ID) {
			return target.ID, nil
		}
	}
	return "", fmt.Errorf("no controlled page target is available")
}

func call(connection *websocket.Conn, id int, method string, params map[string]any, output any) error {
	if connection == nil || id < 1 || !allowedMethod(method) {
		return fmt.Errorf("invalid controlled CDP window request")
	}
	if err := connection.WriteJSON(cdpRequest{ID: id, Method: method, Params: params}); err != nil {
		return fmt.Errorf("write %s request: %w", method, err)
	}
	for skipped := 0; skipped <= maxSkippedMessages; skipped++ {
		_, payload, err := connection.ReadMessage()
		if err != nil {
			return fmt.Errorf("read %s response: %w", method, err)
		}
		if len(payload) > maxCDPMessageBytes {
			return fmt.Errorf("%s response exceeded the controlled limit", method)
		}
		var response cdpResponse
		if err := json.Unmarshal(payload, &response); err != nil {
			return fmt.Errorf("decode %s response: %w", method, err)
		}
		if response.ID != id {
			continue
		}
		if response.Error != nil {
			return fmt.Errorf("%s failed: %s", method, bounded(response.Error.Message, 512))
		}
		if output != nil && len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, output); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
		}
		return nil
	}
	return fmt.Errorf("%s response was not received within the controlled message limit", method)
}

func validateBrowserWebSocket(raw string, port int) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "ws" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("browser CDP websocket URL is invalid")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("browser CDP websocket is not loopback-only")
	}
	value, err := strconv.Atoi(parsed.Port())
	if err != nil || value != port {
		return fmt.Errorf("browser CDP websocket does not use the selected port")
	}
	if !strings.HasPrefix(parsed.EscapedPath(), "/devtools/browser/") {
		return fmt.Errorf("browser CDP websocket path is invalid")
	}
	return nil
}

func allowedMethod(method string) bool {
	switch method {
	case "Browser.getWindowForTarget", "Browser.setWindowBounds", "Browser.getWindowBounds":
		return true
	default:
		return false
	}
}

func validTargetID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
