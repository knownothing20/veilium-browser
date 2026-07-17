package launch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/proxy"
)

type Planner struct{}

func (Planner) Build(profile domain.Profile, remoteDebuggingPort int) (domain.LaunchPlan, error) {
	if profile.Fingerprint.Seed == "" {
		profile.Fingerprint.Seed = deterministicSeed(profile.ID, profile.Name)
	}
	warnings, err := fingerprint.Validate(profile)
	if err != nil {
		return domain.LaunchPlan{}, err
	}
	if strings.TrimSpace(profile.UserDataDir) == "" {
		return domain.LaunchPlan{}, fmt.Errorf("userDataDir is required")
	}
	if remoteDebuggingPort < 0 || remoteDebuggingPort > 65535 {
		return domain.LaunchPlan{}, fmt.Errorf("invalid remote debugging port")
	}

	route, err := proxy.Resolve(profile.Proxy.URL, profile.Proxy.CredentialRef)
	if err != nil {
		return domain.LaunchPlan{}, err
	}
	fpArgs, err := fingerprint.BuildArgs(profile)
	if err != nil {
		return domain.LaunchPlan{}, err
	}

	args := []string{
		"--user-data-dir=" + profile.UserDataDir,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=" + strconv.Itoa(remoteDebuggingPort),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-session-crashed-bubble",
	}
	if route.BrowserURL == "direct://" {
		args = append(args, "--no-proxy-server")
	} else if route.BrowserURL != "" {
		args = append(args, "--proxy-server="+route.BrowserURL)
	} else if route.RequiresBridge {
		warnings = append(warnings, "proxy bridge must be started before launching the browser")
	}
	args = append(args, fpArgs...)

	return domain.LaunchPlan{
		Executable:     profile.Kernel.Executable,
		Args:           args,
		ProxyDisplay:   route.DisplayURL,
		RequiresBridge: route.RequiresBridge,
		BridgeKind:     route.BridgeKind,
		Warnings:       warnings,
	}, nil
}

func deterministicSeed(id, name string) string {
	hash := sha256.Sum256([]byte(id + "\x00" + name))
	return hex.EncodeToString(hash[:8])
}
