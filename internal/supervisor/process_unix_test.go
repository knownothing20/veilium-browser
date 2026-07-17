//go:build !windows

package supervisor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

const unixHelperModeKey = "VEILIUM_UNIX_HELPER_MODE"

func TestUnixProcessGroupKillStopsChildTree(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	logPath := filepath.Join(t.TempDir(), "runtime.log")
	process := startUnixHelper(t, pidFile, logPath, "parent-wait")
	_ = waitPIDFile(t, pidFile)
	groupID := process.PID()
	if err := process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = process.Wait()
	waitForUnixTargetExit(t, -groupID)
}

func TestUnixParentExitCleansRemainingChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	logPath := filepath.Join(t.TempDir(), "runtime.log")
	process := startUnixHelper(t, pidFile, logPath, "parent-exit")
	childPID := waitPIDFile(t, pidFile)
	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
	waitForUnixTargetExit(t, childPID)
}

func startUnixHelper(t *testing.T, pidFile, logPath, mode string) Process {
	t.Helper()
	plan := domain.LaunchPlan{
		Executable: os.Args[0],
		Args:       []string{"-test.run=^TestUnixProcessGroupHelper$"},
		Environment: map[string]string{
			unixHelperModeKey:        mode,
			"VEILIUM_UNIX_CHILD_PID": pidFile,
		},
	}
	process, err := (execRunner{}).Start(plan, logPath)
	if err != nil {
		t.Fatal(err)
	}
	return process
}

func TestUnixProcessGroupHelper(t *testing.T) {
	mode := os.Getenv(unixHelperModeKey)
	if mode == "" {
		return
	}
	if mode == "child" {
		for {
			time.Sleep(time.Second)
		}
	}
	childPIDPath := os.Getenv("VEILIUM_UNIX_CHILD_PID")
	command := exec.Command(os.Args[0], "-test.run=^TestUnixProcessGroupHelper$")
	command.Env = replaceUnixTestEnv(os.Environ(), unixHelperModeKey, "child")
	if err := command.Start(); err != nil {
		os.Exit(21)
	}
	if err := os.WriteFile(childPIDPath, []byte(strconv.Itoa(command.Process.Pid)), 0o600); err != nil {
		os.Exit(22)
	}
	if mode == "parent-exit" {
		os.Exit(0)
	}
	_ = command.Wait()
	os.Exit(0)
}

func replaceUnixTestEnv(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func waitPIDFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("pid file %s was not written", path)
	return 0
}

func waitForUnixTargetExit(t *testing.T, target int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(target, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("unix process target %d still exists", target)
}
