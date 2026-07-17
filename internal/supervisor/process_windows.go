//go:build windows

package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	jobObjectExtendedLimitInformationClass = 9
	jobObjectLimitKillOnJobClose           = 0x00002000
)

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type windowsProcess struct {
	command *exec.Cmd
	log     *os.File
	mu      sync.Mutex
	job     windows.Handle
}

func startPlatformProcess(command *exec.Cmd, logFile *os.File) (Process, error) {
	job, err := createKillOnCloseJob()
	if err != nil {
		return nil, err
	}
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	if err := command.Start(); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	cleanupProcess := func() {
		_ = command.Process.Kill()
		_ = command.Wait()
		_ = windows.CloseHandle(job)
	}
	processHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		uint32(command.Process.Pid),
	)
	if err != nil {
		cleanupProcess()
		return nil, fmt.Errorf("open browser process for Job Object assignment: %w", err)
	}
	assignErr := windows.AssignProcessToJobObject(job, processHandle)
	_ = windows.CloseHandle(processHandle)
	if assignErr != nil {
		cleanupProcess()
		return nil, fmt.Errorf("assign browser process to Windows Job Object: %w", assignErr)
	}
	return &windowsProcess{command: command, log: logFile, job: job}, nil
}

func createKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("create Windows Job Object: %w", err)
	}
	info := jobObjectExtendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	result, err := windows.SetInformationJobObject(
		job,
		jobObjectExtendedLimitInformationClass,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil || result == 0 {
		_ = windows.CloseHandle(job)
		if err == nil {
			err = fmt.Errorf("SetInformationJobObject returned zero")
		}
		return 0, fmt.Errorf("configure Windows Job Object: %w", err)
	}
	return job, nil
}

func (p *windowsProcess) PID() int { return p.command.Process.Pid }

func (p *windowsProcess) Signal(signal os.Signal) error {
	if signal != os.Interrupt {
		return fmt.Errorf("unsupported Windows process-tree signal %v", signal)
	}
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(p.command.Process.Pid))
}

func (p *windowsProcess) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.job == 0 {
		return p.command.Process.Kill()
	}
	return windows.TerminateJobObject(p.job, 1)
}

func (p *windowsProcess) Wait() error {
	err := p.command.Wait()
	p.mu.Lock()
	if p.job != 0 {
		_ = windows.CloseHandle(p.job)
		p.job = 0
	}
	p.mu.Unlock()
	_ = p.log.Close()
	return err
}
