//go:build windows

package systemproxy

import (
	"net/url"
	"os"
	"testing"
)

func TestCurrentWindowsStaticProxyOptIn(t *testing.T) {
	if os.Getenv("VEILIUM_TEST_WINDOWS_SYSTEM_PROXY") != "1" {
		t.Skip("set VEILIUM_TEST_WINDOWS_SYSTEM_PROXY=1 to inspect the current user's static proxy")
	}
	result, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available {
		t.Fatalf("Windows static proxy is unavailable (reason %q)", result.Reason)
	}
	parsed, err := url.Parse(result.ProxyURL)
	if err != nil || parsed.Host == "" {
		t.Fatal("system proxy result is not a valid explicit URL")
	}
	if parsed.User != nil {
		t.Fatal("system proxy result must not expose inline credentials")
	}
}
