//go:build !windows

package supervisor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type unixProcess struct {
	command *exec.Cmd
	log     *os.File
	pgid    int
}

func startPlatformProcess(command *exec.Cmd, logFile *os.File) (Process, error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return nil, err
	}
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, fmt.Errorf("resolve browser process group: %w", err)
	}
	return &unixProcess{command: command, log: logFile, pgid: pgid}, nil
}

func (p *unixProcess) PID() int { return p.command.Process.Pid }

func (p *unixProcess) Signal(signal os.Signal) error {
	syscallSignal, ok := signal.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported process-group signal %T", signal)
	}
	return ignoreMissingProcessGroup(syscall.Kill(-p.pgid, syscallSignal))
}

func (p *unixProcess) Kill() error {
	return ignoreMissingProcessGroup(syscall.Kill(-p.pgid, syscall.SIGKILL))
}

func (p *unixProcess) Wait() error {
	err := p.command.Wait()
	// Chromium descendants may outlive a parent that exits or crashes. The
	// process group remains owned by this runtime session, so clean it before
	// releasing the log file and reporting the session as exited.
	_ = ignoreMissingProcessGroup(syscall.Kill(-p.pgid, syscall.SIGKILL))
	_ = p.log.Close()
	return err
}

func ignoreMissingProcessGroup(err error) error {
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
