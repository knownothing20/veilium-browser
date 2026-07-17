//go:build windows

package supervisor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"golang.org/x/sys/windows"
)

const windowsHelperModeKey = "VEILIUM_JOB_HELPER_MODE"

func TestWindowsJobObjectKillsChildTree(t *testing.T) {
	gatePath := filepath.Join(t.TempDir(), "gate")
	childPIDPath := filepath.Join(t.TempDir(), "child.pid")
	logPath := filepath.Join(t.TempDir(), "runtime.log")
	plan := domain.LaunchPlan{
		Executable: os.Args[0],
		Args:       []string{"-test.run=^TestWindowsJobObjectHelper$"},
		Environment: map[string]string{
			windowsHelperModeKey:    "parent",
			"VEILIUM_JOB_GATE":      gatePath,
			"VEILIUM_JOB_CHILD_PID": childPIDPath,
		},
	}
	process, err := (execRunner{}).Start(plan, logPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gatePath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	childPID := waitWindowsPIDFile(t, childPIDPath)
	if err := process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = process.Wait()
	waitWindowsProcessExit(t, childPID)
}

func TestWindowsJobObjectHelper(t *testing.T) {
	mode := os.Getenv(windowsHelperModeKey)
	if mode == "" {
		return
	}
	if mode == "child" {
		for {
			time.Sleep(time.Second)
		}
	}
	gatePath := os.Getenv("VEILIUM_JOB_GATE")
	childPIDPath := os.Getenv("VEILIUM_JOB_CHILD_PID")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(gatePath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestWindowsJobObjectHelper$")
	command.Env = replaceWindowsTestEnv(os.Environ(), windowsHelperModeKey, "child")
	if err := command.Start(); err != nil {
		os.Exit(21)
	}
	if err := os.WriteFile(childPIDPath, []byte(strconv.Itoa(command.Process.Pid)), 0o600); err != nil {
		os.Exit(22)
	}
	_ = command.Wait()
	os.Exit(0)
}

func replaceWindowsTestEnv(environment []string, key, value string) []string {
	prefix := strings.ToUpper(key) + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(strings.ToUpper(entry), prefix) {
			result = append(result, entry)
		}
	}
	return append(result, key+"="+value)
}

func waitWindowsPIDFile(t *testing.T, path string) uint32 {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr == nil && pid > 0 {
				return uint32(pid)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("child pid file %s was not written", path)
	return 0
}

func waitWindowsProcessExit(t *testing.T, pid uint32) {
	t.Helper()
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return
	}
	if err != nil {
		t.Fatalf("open child process %d: %v", pid, err)
	}
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, 5000)
	if err != nil {
		t.Fatalf("wait for child process %d: %v", pid, err)
	}
	if result != windows.WAIT_OBJECT_0 {
		t.Fatalf("child process %d remained alive, wait result %#x", pid, result)
	}
}
