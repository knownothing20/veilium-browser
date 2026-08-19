package proxy

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func CheckReachable(ctx context.Context, route Route) error {
	if route.DisplayURL == "direct://" || route.RequiresBridge {
		return nil
	}
	parsed, err := url.Parse(route.BrowserURL)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("proxy route has no reachable host")
	}
	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return fmt.Errorf("proxy route requires an explicit port")
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(parsed.Hostname(), port))
	if err != nil {
		return fmt.Errorf("connect to proxy endpoint: %w", err)
	}
	return connection.Close()
}
