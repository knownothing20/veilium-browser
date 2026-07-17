package adapterruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/knownothing20/veilium-browser/internal/adapter"
)

var ErrProviderUnavailable = errors.New("proxy adapter configuration provider is unavailable")

const ConfigPathToken = "{veilium-config}"

type Request struct {
	Adapter            adapter.Record
	Scheme             string
	ProxyURL           string
	CredentialRef      string
	CredentialUsername string
	CredentialSecret   string
	ProfileID          string
	LocalPort          int
}

type Plan struct {
	Executable   string
	Arguments    []string
	Environment  map[string]string
	Config       []byte
	ConfigFormat string
	LocalScheme  string
}

type Provider interface {
	Kind() string
	Prepare(context.Context, Request) (Plan, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(provider Provider) error {
	if r == nil || provider == nil {
		return fmt.Errorf("adapter provider is required")
	}
	kind := adapter.NormalizeKind(provider.Kind())
	if err := adapter.ValidateKind(kind); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[kind]; exists {
		return fmt.Errorf("adapter provider %q is already registered", kind)
	}
	r.providers[kind] = provider
	return nil
}

func (r *Registry) Prepare(ctx context.Context, request Request) (Plan, error) {
	if r == nil {
		return Plan{}, ErrProviderUnavailable
	}
	if request.Adapter.Status != adapter.StatusVerified {
		return Plan{}, fmt.Errorf("adapter %q is not verified", request.Adapter.Name)
	}
	if !adapter.SupportsScheme(request.Adapter.Kind, request.Scheme) {
		return Plan{}, fmt.Errorf("adapter kind %q does not support scheme %q", request.Adapter.Kind, request.Scheme)
	}
	r.mu.RLock()
	provider := r.providers[adapter.NormalizeKind(request.Adapter.Kind)]
	r.mu.RUnlock()
	if provider == nil {
		return Plan{}, fmt.Errorf("%w for %q", ErrProviderUnavailable, request.Adapter.Kind)
	}
	return provider.Prepare(ctx, request)
}
