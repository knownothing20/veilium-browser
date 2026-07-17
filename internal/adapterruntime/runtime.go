package adapterruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

const (
	maxConfigBytes = 1 << 20
	startTimeout   = 10 * time.Second
	stopTimeout    = 3 * time.Second
)

type Instance interface {
	URL() string
	Kind() string
	Health(context.Context) error
	Done() <-chan struct{}
	Close() error
}

type Factory interface {
	Start(context.Context, Request) (Instance, error)
}

type ProcessStarter interface {
	Start(domain.LaunchPlan, string) (supervisor.Process, error)
}

type Manager struct {
	root     string
	registry *Registry
	starter  ProcessStarter
}

type managedProcessStarter struct{}

func (managedProcessStarter) Start(plan domain.LaunchPlan, logPath string) (supervisor.Process, error) {
	return supervisor.StartManagedProcess(plan, logPath)
}

func NewManager(root string, registry *Registry) (*Manager, error) {
	return NewManagerWithStarter(root, registry, managedProcessStarter{})
}

func NewManagerWithStarter(root string, registry *Registry, starter ProcessStarter) (*Manager, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("adapter runtime root is required")
	}
	if registry == nil || starter == nil {
		return nil, fmt.Errorf("adapter runtime dependencies are required")
	}
	return &Manager{root: root, registry: registry, starter: starter}, nil
}

func (m *Manager) Start(ctx context.Context, request Request) (Instance, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(request.ProfileID) == "" {
		return nil, fmt.Errorf("adapter runtime profile id is required")
	}
	if err := ensurePrivateDirectory(m.root); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		port, err := allocateLoopbackPort()
		if err != nil {
			return nil, err
		}
		request.LocalPort = port
		plan, err := m.registry.Prepare(ctx, request)
		if err != nil {
			return nil, err
		}
		instance, err := m.startPlan(ctx, request, plan, port)
		if err == nil {
			return instance, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("start adapter runtime after retries: %w", lastErr)
}

func (m *Manager) startPlan(ctx context.Context, request Request, plan Plan, port int) (*processInstance, error) {
	if err := validatePlan(request, plan); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(m.root, safeID(request.ProfileID)+"-*")
	if err != nil {
		return nil, fmt.Errorf("create adapter runtime directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if err := os.Chmod(directory, 0o700); err != nil {
		cleanup()
		return nil, fmt.Errorf("protect adapter runtime directory: %w", err)
	}
	configPath := filepath.Join(directory, "config.json")
	if err := writePrivateFile(configPath, plan.Config, 0o600); err != nil {
		cleanup()
		return nil, err
	}
	arguments, err := materializeArguments(plan.Arguments, configPath)
	if err != nil {
		cleanup()
		return nil, err
	}
	logPath := filepath.Join(directory, "runtime.log")
	process, err := m.starter.Start(domain.LaunchPlan{
		Executable:  plan.Executable,
		Args:        arguments,
		Environment: cloneEnvironment(plan.Environment),
	}, logPath)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("start managed adapter process: %w", err)
	}
	instance := &processInstance{
		process: process,
		url:     plan.LocalScheme + "://127.0.0.1:" + strconv.Itoa(port),
		kind:    request.Adapter.Kind,
		address: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		dir:     directory,
		done:    make(chan struct{}),
	}
	go instance.wait()

	healthCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()
	if err := instance.waitHealthy(healthCtx); err != nil {
		_ = instance.Close()
		return nil, err
	}
	return instance, nil
}

func validatePlan(request Request, plan Plan) error {
	if strings.TrimSpace(plan.Executable) == "" || plan.Executable != request.Adapter.Executable {
		return fmt.Errorf("adapter provider must use the verified managed executable")
	}
	if plan.ConfigFormat != "json" || len(plan.Config) == 0 || len(plan.Config) > maxConfigBytes {
		return fmt.Errorf("adapter provider returned an invalid private configuration")
	}
	if plan.LocalScheme != "socks5" {
		return fmt.Errorf("adapter provider must expose a loopback SOCKS5 endpoint")
	}
	foundToken := false
	for _, argument := range plan.Arguments {
		if strings.ContainsRune(argument, '\x00') {
			return fmt.Errorf("adapter argument contains an invalid null byte")
		}
		if argument == ConfigPathToken {
			foundToken = true
		}
	}
	if !foundToken {
		return fmt.Errorf("adapter provider did not request a private config path")
	}
	for key, value := range plan.Environment {
		if strings.ContainsAny(key, "=\x00\r\n") || strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("adapter environment contains an invalid entry")
		}
	}
	return nil
}

func materializeArguments(arguments []string, configPath string) ([]string, error) {
	result := make([]string, len(arguments))
	for index, argument := range arguments {
		if argument == ConfigPathToken {
			result[index] = configPath
			continue
		}
		if strings.Contains(argument, ConfigPathToken) {
			return nil, fmt.Errorf("config path token must be a complete argument")
		}
		result[index] = argument
	}
	return result, nil
}

func writePrivateFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create private adapter config: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write private adapter config: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync private adapter config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close private adapter config: %w", err)
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create adapter runtime root: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect adapter runtime root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("adapter runtime root must be a real directory")
	}
	return os.Chmod(path, 0o700)
}

func allocateLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate adapter loopback port: %w", err)
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.Port < 1 {
		return 0, fmt.Errorf("adapter loopback allocation returned an invalid port")
	}
	return address.Port, nil
}

func cloneEnvironment(environment map[string]string) map[string]string {
	if len(environment) == 0 {
		return nil
	}
	result := make(map[string]string, len(environment))
	for key, value := range environment {
		result[key] = value
	}
	return result
}

type processInstance struct {
	process supervisor.Process
	url     string
	kind    string
	address string
	dir     string
	done    chan struct{}

	mu        sync.Mutex
	waitErr   error
	closeOnce sync.Once
	closeErr  error
}

func (i *processInstance) URL() string           { return i.url }
func (i *processInstance) Kind() string          { return i.kind }
func (i *processInstance) Done() <-chan struct{} { return i.done }

func (i *processInstance) Health(ctx context.Context) error {
	connection, err := (&net.Dialer{Timeout: 750 * time.Millisecond}).DialContext(ctx, "tcp4", i.address)
	if err != nil {
		return fmt.Errorf("adapter SOCKS5 listener is unavailable: %w", err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(time.Second))
	if _, err := connection.Write([]byte{5, 1, 0}); err != nil {
		return fmt.Errorf("write adapter SOCKS5 greeting: %w", err)
	}
	response := make([]byte, 2)
	if _, err := io.ReadFull(connection, response); err != nil {
		return fmt.Errorf("read adapter SOCKS5 greeting: %w", err)
	}
	if response[0] != 5 || response[1] != 0 {
		return fmt.Errorf("adapter endpoint did not accept SOCKS5 no-auth negotiation")
	}
	return nil
}

func (i *processInstance) waitHealthy(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		healthCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := i.Health(healthCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		select {
		case <-i.done:
			i.mu.Lock()
			waitErr := i.waitErr
			i.mu.Unlock()
			if waitErr == nil {
				waitErr = errors.New("adapter process exited")
			}
			return fmt.Errorf("adapter process exited before SOCKS5 readiness: %w", waitErr)
		case <-ctx.Done():
			return fmt.Errorf("adapter SOCKS5 readiness timed out: %v: %w", lastErr, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (i *processInstance) wait() {
	err := i.process.Wait()
	i.mu.Lock()
	i.waitErr = err
	i.mu.Unlock()
	close(i.done)
	i.cleanup()
}

func (i *processInstance) Close() error {
	if i == nil {
		return nil
	}
	i.closeOnce.Do(func() {
		select {
		case <-i.done:
			i.cleanup()
			return
		default:
		}
		if err := i.process.Signal(os.Interrupt); err != nil {
			_ = i.process.Kill()
		}
		timer := time.NewTimer(stopTimeout)
		defer timer.Stop()
		select {
		case <-i.done:
		case <-timer.C:
			if err := i.process.Kill(); err != nil {
				i.closeErr = err
			}
			<-i.done
		}
		i.cleanup()
	})
	return i.closeErr
}

func (i *processInstance) cleanup() {
	if i.dir != "" {
		_ = os.RemoveAll(i.dir)
	}
}

func safeID(value string) string {
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
	if result == "" {
		return "profile"
	}
	return result
}
