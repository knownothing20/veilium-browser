//go:build windows

package systemproxy

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsInternetSettings = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// Current reads the current user's Windows static proxy configuration. It does
// not read proxy credentials, follow PAC/WPAD, or mutate any system setting.
func Current() (Result, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsInternetSettings, registry.QUERY_VALUE)
	if err != nil {
		return Result{}, fmt.Errorf("read Windows proxy settings: %w", err)
	}
	defer key.Close()

	enabled, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return Result{}, fmt.Errorf("read Windows static proxy state: %w", err)
	}
	if err != nil || enabled == 0 {
		reason := ReasonDisabled
		if hasRegistryValue(key, "AutoConfigURL") {
			reason = ReasonPACOnly
		}
		return Result{Source: SourceWindowsStatic, Reason: reason}, nil
	}

	raw, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return Result{}, fmt.Errorf("read Windows static proxy address: %w", err)
	}
	proxyURL, err := normalizeStaticProxy(raw)
	if err != nil {
		return Result{}, err
	}
	return Result{Available: true, ProxyURL: proxyURL, Source: SourceWindowsStatic}, nil
}

func hasRegistryValue(key registry.Key, target string) bool {
	names, err := key.ReadValueNames(0)
	if err != nil {
		return false
	}
	for _, name := range names {
		if strings.EqualFold(name, target) {
			return true
		}
	}
	return false
}
