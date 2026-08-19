package kernel

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const (
	launchProbeTimeout = 15 * time.Second
	maxProbeOutput     = 1 << 20
)

func probeManagedExecutable(executable string) error {
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return fmt.Errorf("managed executable path is empty")
	}
	info, err := os.Lstat(executable)
	if err != nil {
		return fmt.Errorf("inspect managed executable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("managed executable is not a regular file")
	}

	probeRoot, err := os.MkdirTemp("", "veilium-kernel-probe-*")
	if err != nil {
		return fmt.Errorf("create launch probe root: %w", err)
	}
	defer os.RemoveAll(probeRoot)
	if err := os.Chmod(probeRoot, 0o700); err != nil {
		return fmt.Errorf("protect launch probe root: %w", err)
	}
	profileDir := filepath.Join(probeRoot, "profile")
	logDir := filepath.Join(probeRoot, "logs")
	if err := os.Mkdir(profileDir, 0o700); err != nil {
		return fmt.Errorf("create launch probe profile: %w", err)
	}

	runtime, err := supervisor.New(logDir)
	if err != nil {
		return fmt.Errorf("create launch probe runtime: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), launchProbeTimeout)
	defer cancel()
	session, err := runtime.Start(ctx, "kernel-probe", "Kernel launch probe", func(port int) (domain.LaunchPlan, error) {
		return domain.LaunchPlan{
			Executable: executable,
			Args: []string{
				"--headless=new",
				"--disable-background-networking",
				"--disable-component-update",
				"--disable-default-apps",
				"--disable-gpu",
				"--disable-sync",
				"--metrics-recording-only",
				"--no-default-browser-check",
				"--no-first-run",
				"--no-pings",
				"--no-proxy-server",
				"--remote-debugging-address=127.0.0.1",
				fmt.Sprintf("--remote-debugging-port=%d", port),
				"--safebrowsing-disable-auto-update",
				"--user-data-dir=" + profileDir,
				"about:blank",
			},
		}, nil
	})
	if err != nil {
		return fmt.Errorf("managed Chromium did not reach loopback CDP readiness: %w%s", err, probeLogDetail(session.LogPath))
	}

	stopContext, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	stopped, stopErr := runtime.Stop(stopContext, session.ProfileID)
	stopCancel()
	if stopErr != nil {
		_ = runtime.Shutdown(context.Background())
		return fmt.Errorf("stop managed Chromium launch probe: %w%s", stopErr, probeLogDetail(session.LogPath))
	}
	if runtime.IsActive(session.ProfileID) {
		_ = runtime.Shutdown(context.Background())
		return fmt.Errorf("managed Chromium launch probe process tree is still active%s", probeLogDetail(session.LogPath))
	}
	if stopped.State != supervisor.StateExited {
		return fmt.Errorf("managed Chromium launch probe exited in state %s%s", stopped.State, probeLogDetail(session.LogPath))
	}
	return nil
}

func probeLogDetail(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProbeOutput+1))
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(data))
	if len(value) > 600 {
		value = value[:600] + "…"
	}
	if value == "" {
		return ""
	}
	return ": " + value
}
