package supervisor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

type State string

const (
	StateStarting State = "starting"
	StateReady    State = "ready"
	StateStopping State = "stopping"
	StateExited   State = "exited"
	StateFailed   State = "failed"
)

type Session struct {
	ProfileID            string     `json:"profileId"`
	ProfileName          string     `json:"profileName"`
	State                State      `json:"state"`
	PID                  int        `json:"pid"`
	CDPPort              int        `json:"cdpPort"`
	CDPURL               string     `json:"cdpUrl"`
	WebSocketDebuggerURL string     `json:"webSocketDebuggerUrl,omitempty"`
	Browser              string     `json:"browser,omitempty"`
	StartedAt            time.Time  `json:"startedAt"`
	ReadyAt              *time.Time `json:"readyAt,omitempty"`
	ExitedAt             *time.Time `json:"exitedAt,omitempty"`
	ExitCode             *int       `json:"exitCode,omitempty"`
	LastError            string     `json:"lastError,omitempty"`
	LogPath              string     `json:"logPath"`
}

type VersionInfo struct {
	Browser              string `json:"Browser"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type PlanBuilder func(cdpPort int) (domain.LaunchPlan, error)

type Process interface {
	PID() int
	Wait() error
	Signal(os.Signal) error
	Kill() error
}

type Runner interface {
	Start(plan domain.LaunchPlan, logPath string) (Process, error)
}

type Prober interface {
	Wait(context.Context, int) (VersionInfo, error)
}

type PortAllocator interface {
	Allocate() (int, error)
}

type Dependencies struct {
	Runner       Runner
	Prober       Prober
	Ports        PortAllocator
	Now          func() time.Time
	ReadyTimeout time.Duration
	StopTimeout  time.Duration
}

type managedSession struct {
	snapshot Session
	process  Process
	done     chan struct{}
}

type Supervisor struct {
	mu           sync.RWMutex
	sessions     map[string]*managedSession
	starting     map[string]struct{}
	runner       Runner
	prober       Prober
	ports        PortAllocator
	now          func() time.Time
	logDir       string
	readyTimeout time.Duration
	stopTimeout  time.Duration
}

var environmentKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func New(logDir string) (*Supervisor, error) {
	client := &http.Client{
		Timeout:   750 * time.Millisecond,
		Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("CDP readiness endpoint must not redirect")
		},
	}
	return NewWithDependencies(logDir, Dependencies{
		Runner:       execRunner{},
		Prober:       HTTPProber{Client: client, Interval: 150 * time.Millisecond},
		Ports:        localPortAllocator{},
		Now:          time.Now,
		ReadyTimeout: 12 * time.Second,
		StopTimeout:  3 * time.Second,
	})
}

func NewWithDependencies(logDir string, dependencies Dependencies) (*Supervisor, error) {
	if strings.TrimSpace(logDir) == "" {
		return nil, fmt.Errorf("runtime log directory is required")
	}
	if dependencies.Runner == nil || dependencies.Prober == nil || dependencies.Ports == nil {
		return nil, fmt.Errorf("runtime dependencies are required")
	}
	if dependencies.Now == nil {
		dependencies.Now = time.Now
	}
	if dependencies.ReadyTimeout <= 0 {
		dependencies.ReadyTimeout = 12 * time.Second
	}
	if dependencies.StopTimeout <= 0 {
		dependencies.StopTimeout = 3 * time.Second
	}
	return &Supervisor{
		sessions:     make(map[string]*managedSession),
		starting:     make(map[string]struct{}),
		runner:       dependencies.Runner,
		prober:       dependencies.Prober,
		ports:        dependencies.Ports,
		now:          dependencies.Now,
		logDir:       logDir,
		readyTimeout: dependencies.ReadyTimeout,
		stopTimeout:  dependencies.StopTimeout,
	}, nil
}

func (s *Supervisor) Start(ctx context.Context, profileID, profileName string, build PlanBuilder) (Session, error) {
	profileID = strings.TrimSpace(profileID)
	profileName = strings.TrimSpace(profileName)
	if profileID == "" || profileName == "" {
		return Session{}, fmt.Errorf("profile id and name are required")
	}
	if build == nil {
		return Session{}, fmt.Errorf("launch plan builder is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := s.reserveStart(profileID, profileName); err != nil {
		if snapshot, getErr := s.Get(profileID); getErr == nil {
			return snapshot, err
		}
		return Session{}, err
	}
	reserved := true
	defer func() {
		if reserved {
			s.releaseStart(profileID)
		}
	}()

	port, err := s.ports.Allocate()
	if err != nil {
		return Session{}, fmt.Errorf("allocate loopback CDP port: %w", err)
	}
	plan, err := build(port)
	if err != nil {
		return Session{}, fmt.Errorf("build browser launch plan: %w", err)
	}
	if err := validatePlan(plan, port); err != nil {
		return Session{}, err
	}
	if err := os.MkdirAll(s.logDir, 0o700); err != nil {
		return Session{}, fmt.Errorf("create runtime log directory: %w", err)
	}
	startedAt := s.now().UTC()
	logPath := filepath.Join(s.logDir, fmt.Sprintf("%s-%s.log", safeID(profileID), startedAt.Format("20060102T150405.000000000Z")))
	process, err := s.runner.Start(plan, logPath)
	if err != nil {
		return Session{}, fmt.Errorf("start browser process: %w", err)
	}

	managed := &managedSession{
		snapshot: Session{
			ProfileID:   profileID,
			ProfileName: profileName,
			State:       StateStarting,
			PID:         process.PID(),
			CDPPort:     port,
			CDPURL:      fmt.Sprintf("http://127.0.0.1:%d", port),
			StartedAt:   startedAt,
			LogPath:     logPath,
		},
		process: process,
		done:    make(chan struct{}),
	}
	s.mu.Lock()
	delete(s.starting, profileID)
	s.sessions[profileID] = managed
	s.mu.Unlock()
	reserved = false
	go s.wait(managed)

	readyContext, cancel := context.WithTimeout(ctx, s.readyTimeout)
	defer cancel()
	resultChannel := make(chan probeResult, 1)
	go func() {
		version, probeErr := s.prober.Wait(readyContext, port)
		resultChannel <- probeResult{version: version, err: probeErr}
	}()

	select {
	case result := <-resultChannel:
		if result.err != nil {
			failure := fmt.Errorf("CDP readiness check failed: %w", result.err)
			s.failStart(profileID, failure)
			return s.snapshotWithError(profileID, failure)
		}
		if err := validateVersionInfo(result.version, port); err != nil {
			s.failStart(profileID, err)
			return s.snapshotWithError(profileID, err)
		}
		readyAt := s.now().UTC()
		s.mu.Lock()
		current, ok := s.sessions[profileID]
		if !ok || current != managed || current.snapshot.State != StateStarting {
			var snapshot Session
			if ok {
				snapshot = current.snapshot
			}
			s.mu.Unlock()
			return snapshot, fmt.Errorf("browser exited or was stopped before CDP became ready")
		}
		current.snapshot.State = StateReady
		current.snapshot.ReadyAt = &readyAt
		current.snapshot.Browser = result.version.Browser
		current.snapshot.WebSocketDebuggerURL = result.version.WebSocketDebuggerURL
		snapshot := current.snapshot
		s.mu.Unlock()
		return snapshot, nil
	case <-managed.done:
		return s.snapshotWithError(profileID, fmt.Errorf("browser exited before CDP became ready"))
	case <-readyContext.Done():
		failure := fmt.Errorf("CDP readiness timed out: %w", readyContext.Err())
		s.failStart(profileID, failure)
		return s.snapshotWithError(profileID, failure)
	}
}

func (s *Supervisor) Stop(ctx context.Context, profileID string) (Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	managed, ok := s.sessions[profileID]
	if !ok {
		s.mu.Unlock()
		return Session{}, fmt.Errorf("runtime session not found")
	}
	if !managedIsActive(managed) {
		snapshot := managed.snapshot
		s.mu.Unlock()
		return snapshot, nil
	}
	managed.snapshot.State = StateStopping
	s.mu.Unlock()

	if err := managed.process.Signal(os.Interrupt); err != nil {
		_ = managed.process.Kill()
	}
	timer := time.NewTimer(s.stopTimeout)
	defer timer.Stop()
	select {
	case <-managed.done:
		return s.Get(profileID)
	case <-timer.C:
		_ = managed.process.Kill()
		select {
		case <-managed.done:
			return s.Get(profileID)
		case <-ctx.Done():
			return s.snapshotWithError(profileID, fmt.Errorf("stop browser process: %w", ctx.Err()))
		}
	case <-ctx.Done():
		_ = managed.process.Kill()
		return s.snapshotWithError(profileID, fmt.Errorf("stop browser process: %w", ctx.Err()))
	}
}

func (s *Supervisor) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var failures []string
	for _, session := range s.List() {
		if !sessionIsActive(session) {
			continue
		}
		if _, err := s.Stop(ctx, session.ProfileID); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("stop browser sessions: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *Supervisor) List() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		items = append(items, session.snapshot)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.After(items[j].StartedAt) })
	return items
}

func (s *Supervisor) Get(profileID string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	managed, ok := s.sessions[profileID]
	if !ok {
		return Session{}, fmt.Errorf("runtime session not found")
	}
	return managed.snapshot, nil
}

func (s *Supervisor) IsActive(profileID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, starting := s.starting[profileID]; starting {
		return true
	}
	managed, ok := s.sessions[profileID]
	return ok && managedIsActive(managed)
}

type probeResult struct {
	version VersionInfo
	err     error
}

func (s *Supervisor) reserveStart(profileID, profileName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, starting := s.starting[profileID]; starting {
		return fmt.Errorf("profile %q is already starting", profileName)
	}
	if existing, ok := s.sessions[profileID]; ok && managedIsActive(existing) {
		return fmt.Errorf("profile %q is already running", profileName)
	}
	s.starting[profileID] = struct{}{}
	return nil
}

func (s *Supervisor) releaseStart(profileID string) {
	s.mu.Lock()
	delete(s.starting, profileID)
	s.mu.Unlock()
}

func (s *Supervisor) failStart(profileID string, failure error) {
	s.mu.Lock()
	managed, ok := s.sessions[profileID]
	if ok && managed.snapshot.State == StateStarting {
		managed.snapshot.State = StateFailed
		managed.snapshot.LastError = failure.Error()
	}
	s.mu.Unlock()
	if ok {
		_ = managed.process.Kill()
	}
}

func (s *Supervisor) snapshotWithError(profileID string, failure error) (Session, error) {
	snapshot, err := s.Get(profileID)
	if err != nil {
		return Session{}, failure
	}
	return snapshot, failure
}

func (s *Supervisor) wait(managed *managedSession) {
	err := managed.process.Wait()
	exitedAt := s.now().UTC()
	code := processExitCode(err)

	s.mu.Lock()
	current, ok := s.sessions[managed.snapshot.ProfileID]
	if ok && current == managed {
		current.snapshot.ExitedAt = &exitedAt
		current.snapshot.ExitCode = &code
		switch current.snapshot.State {
		case StateStopping:
			current.snapshot.State = StateExited
		case StateFailed:
			// Preserve the readiness or validation error that caused the kill.
		default:
			if err != nil {
				current.snapshot.State = StateFailed
				current.snapshot.LastError = err.Error()
			} else {
				current.snapshot.State = StateExited
			}
		}
	}
	s.mu.Unlock()
	close(managed.done)
}

func validatePlan(plan domain.LaunchPlan, port int) error {
	if strings.TrimSpace(plan.Executable) == "" {
		return fmt.Errorf("launch plan executable is required")
	}
	if strings.ContainsRune(plan.Executable, '\x00') {
		return fmt.Errorf("launch plan executable contains an invalid null byte")
	}
	if plan.RequiresBridge {
		return fmt.Errorf("proxy bridge %q is not available yet", plan.BridgeKind)
	}
	requiredAddress := false
	requiredPort := false
	for _, argument := range plan.Args {
		if strings.ContainsRune(argument, '\x00') {
			return fmt.Errorf("launch argument contains an invalid null byte")
		}
		if strings.HasPrefix(argument, "--remote-debugging-address=") {
			requiredAddress = argument == "--remote-debugging-address=127.0.0.1"
		}
		if strings.HasPrefix(argument, "--remote-debugging-port=") {
			requiredPort = argument == "--remote-debugging-port="+strconv.Itoa(port)
		}
	}
	if !requiredAddress || !requiredPort {
		return fmt.Errorf("launch plan must bind CDP to the allocated loopback port")
	}
	for key, value := range plan.Environment {
		if !environmentKeyPattern.MatchString(key) || strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("launch environment contains an invalid entry")
		}
	}
	return nil
}

func validateVersionInfo(version VersionInfo, port int) error {
	if strings.TrimSpace(version.Browser) == "" {
		return fmt.Errorf("CDP readiness response did not identify the browser")
	}
	parsed, err := url.Parse(version.WebSocketDebuggerURL)
	if err != nil || parsed.Scheme != "ws" {
		return fmt.Errorf("CDP websocket URL is invalid")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("CDP websocket URL is not loopback-only")
	}
	webSocketPort, err := strconv.Atoi(parsed.Port())
	if err != nil || webSocketPort != port {
		return fmt.Errorf("CDP websocket URL does not use the allocated port")
	}
	return nil
}

func sessionIsActive(session Session) bool {
	if session.State == StateStarting || session.State == StateReady || session.State == StateStopping {
		return true
	}
	return session.State == StateFailed && session.ExitedAt == nil
}

func managedIsActive(session *managedSession) bool {
	return session != nil && sessionIsActive(session.snapshot)
}

func safeID(value string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
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

type execRunner struct{}

type execProcess struct {
	command *exec.Cmd
	log     *os.File
}

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
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		_ = os.Remove(logPath)
		return nil, err
	}
	return &execProcess{command: command, log: logFile}, nil
}

func (p *execProcess) PID() int                      { return p.command.Process.Pid }
func (p *execProcess) Signal(signal os.Signal) error { return p.command.Process.Signal(signal) }
func (p *execProcess) Kill() error                   { return p.command.Process.Kill() }
func (p *execProcess) Wait() error {
	err := p.command.Wait()
	_ = p.log.Close()
	return err
}

type localPortAllocator struct{}

func (localPortAllocator) Allocate() (int, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP == nil || !address.IP.IsLoopback() {
		return 0, fmt.Errorf("allocated CDP listener is not loopback-only")
	}
	return address.Port, nil
}
