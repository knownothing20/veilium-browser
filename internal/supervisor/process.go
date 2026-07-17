package supervisor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

type execRunner struct{}

func (execRunner) Start(plan domain.LaunchPlan, logPath string) (Process, error) {
	info, err := os.Lstat(plan.Executable)
	if err != nil {
		return nil, fmt.Errorf("inspect browser executable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("browser executable must be a regular managed file")
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open runtime log: %w", err)
	}
	command := exec.Command(plan.Executable, plan.Args...)
	command.Stdin = nil
	command.Stdout = logFile
	command.Stderr = logFile
	command.Env = os.Environ()
	for key, value := range plan.Environment {
		command.Env = append(command.Env, key+"="+value)
	}
	process, err := startPlatformProcess(command, logFile)
	if err != nil {
		_ = logFile.Close()
		_ = os.Remove(logPath)
		return nil, err
	}
	return process, nil
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}
