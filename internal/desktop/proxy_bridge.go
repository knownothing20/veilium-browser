package desktop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/proxybridge"
)

type bridgeEntry struct {
	instance proxybridge.Instance
	token    uint64
}

type bridgeRegistry struct {
	mu      sync.Mutex
	factory proxybridge.Factory
	entries map[string]bridgeEntry
	next    uint64
}

var bridgeRegistries sync.Map

func registryFor(service *Service) *bridgeRegistry {
	value, _ := bridgeRegistries.LoadOrStore(service, &bridgeRegistry{
		factory: proxybridge.DefaultFactory{},
		entries: make(map[string]bridgeEntry),
	})
	return value.(*bridgeRegistry)
}

func setProxyBridgeFactory(service *Service, factory proxybridge.Factory) {
	registry := registryFor(service)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.factory = factory
}

func proxyBridgeFactory(service *Service) proxybridge.Factory {
	registry := registryFor(service)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.factory
}

func registerProxyBridge(service *Service, profileID string, instance proxybridge.Instance) uint64 {
	registry := registryFor(service)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.next++
	token := registry.next
	if previous, ok := registry.entries[profileID]; ok {
		_ = previous.instance.Close()
	}
	registry.entries[profileID] = bridgeEntry{instance: instance, token: token}
	return token
}

func watchProxyBridge(service *Service, profileID string, token uint64) {
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if service.supervisor.IsActive(profileID) {
			continue
		}
		_ = releaseProxyBridge(service, profileID, token)
		return
	}
}

func closeProfileProxyBridge(service *Service, profileID string) error {
	registry := registryFor(service)
	registry.mu.Lock()
	entry, ok := registry.entries[profileID]
	if ok {
		delete(registry.entries, profileID)
	}
	registry.mu.Unlock()
	if !ok {
		return nil
	}
	return entry.instance.Close()
}

func releaseProxyBridge(service *Service, profileID string, token uint64) error {
	registry := registryFor(service)
	registry.mu.Lock()
	entry, ok := registry.entries[profileID]
	if !ok || entry.token != token {
		registry.mu.Unlock()
		return nil
	}
	delete(registry.entries, profileID)
	registry.mu.Unlock()
	return entry.instance.Close()
}

func closeAllProxyBridges(service *Service) error {
	value, ok := bridgeRegistries.Load(service)
	if !ok {
		return nil
	}
	registry := value.(*bridgeRegistry)
	registry.mu.Lock()
	entries := make([]bridgeEntry, 0, len(registry.entries))
	for _, entry := range registry.entries {
		entries = append(entries, entry)
	}
	registry.entries = make(map[string]bridgeEntry)
	registry.mu.Unlock()
	bridgeRegistries.Delete(service)
	var failures []error
	for _, entry := range entries {
		if err := entry.instance.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func shutdownRuntimeAndBridges(service *Service, ctx context.Context) error {
	runtimeErr := service.supervisor.Shutdown(ctx)
	bridgeErr := closeAllProxyBridges(service)
	if runtimeErr != nil && bridgeErr != nil {
		return fmt.Errorf("runtime shutdown failed: %v; proxy bridge shutdown failed: %w", runtimeErr, bridgeErr)
	}
	if runtimeErr != nil {
		return runtimeErr
	}
	return bridgeErr
}
