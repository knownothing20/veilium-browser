package proxydiagnostics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/proxybridge"
)

const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusFailed   = "failed"

	CheckPass    = "pass"
	CheckWarn    = "warn"
	CheckFail    = "fail"
	CheckSkipped = "skipped"

	defaultProbeURL = "https://api64.ipify.org?format=json"
	maxProbeBytes   = 4096
)

type Check struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	LatencyMS int64  `json:"latencyMs,omitempty"`
}

type Report struct {
	ProfileID          string    `json:"profileId"`
	ProfileName        string    `json:"profileName"`
	Status             string    `json:"status"`
	ProxyDisplay       string    `json:"proxyDisplay"`
	RouteKind          string    `json:"routeKind"`
	BridgeKind         string    `json:"bridgeKind,omitempty"`
	ExitIP             string    `json:"exitIp,omitempty"`
	FirstByteLatencyMS int64     `json:"firstByteLatencyMs,omitempty"`
	TotalLatencyMS     int64     `json:"totalLatencyMs,omitempty"`
	StartedAt          time.Time `json:"startedAt"`
	CompletedAt        time.Time `json:"completedAt"`
	Checks             []Check   `json:"checks"`
	Limitations        []string  `json:"limitations"`
}

type Config struct {
	ProbeURL string
	Timeout  time.Duration
}

func DefaultConfig() Config {
	return Config{ProbeURL: defaultProbeURL, Timeout: 15 * time.Second}
}

type Request struct {
	Profile  domain.Profile
	Material credential.Material
}

type FactoryProvider func() proxybridge.Factory

type Runner struct {
	mu      sync.Mutex
	active  map[string]struct{}
	factory FactoryProvider
	config  Config
	now     func() time.Time
}

func New(factory FactoryProvider, config Config) (*Runner, error) {
	if factory == nil {
		return nil, fmt.Errorf("proxy bridge factory provider is required")
	}
	if strings.TrimSpace(config.ProbeURL) == "" {
		config.ProbeURL = defaultProbeURL
	}
	if config.Timeout <= 0 {
		config.Timeout = 15 * time.Second
	}
	if err := validateProbeURL(config.ProbeURL); err != nil {
		return nil, err
	}
	return &Runner{
		active:  make(map[string]struct{}),
		factory: factory,
		config:  config,
		now:     time.Now,
	}, nil
}

func (r *Runner) reserve(profileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.active[profileID]; exists {
		return fmt.Errorf("proxy diagnostics are already running for this profile")
	}
	r.active[profileID] = struct{}{}
	return nil
}

func (r *Runner) release(profileID string) {
	r.mu.Lock()
	delete(r.active, profileID)
	r.mu.Unlock()
}
