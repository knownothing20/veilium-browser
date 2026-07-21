package localrecovery

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type memoryCredentialBackend struct {
	values   map[string]string
	getCalls int
}

func (b *memoryCredentialBackend) Set(service, account, value string) error {
	if b.values == nil {
		b.values = make(map[string]string)
	}
	b.values[service+"\x00"+account] = value
	return nil
}

func (b *memoryCredentialBackend) Get(service, account string) (string, error) {
	b.getCalls++
	value, exists := b.values[service+"\x00"+account]
	if !exists {
		return "", credential.ErrSecretNotFound
	}
	return value, nil
}

func (b *memoryCredentialBackend) Delete(service, account string) error {
	key := service + "\x00" + account
	if _, exists := b.values[key]; !exists {
		return credential.ErrSecretNotFound
	}
	delete(b.values, key)
	return nil
}

type restoreHarness struct {
	snapshot          snapshotHarness
	snapshotResult    SnapshotResult
	profiles          *profile.Store
	executor          *RestoreExecutor
	source            domain.Profile
	request           RestoreRequest
	credentialBackend *memoryCredentialBackend
	credentials       *credential.Manager
}

func newRestoreHarness(t *testing.T, files map[string]string) restoreHarness {
	t.Helper()
	base := newSnapshotHarness(t, files)

	profiles, err := profile.Open(filepath.Join(base.dataRoot, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := profiles.Create(domain.Profile{
		ID:    "profile-a",
		Name:  "Source Profile",
		Group: "Source Group",
		Notes: "source metadata remains unchanged",
		Kernel: domain.KernelRef{
			ID:         "source-kernel-record",
			Provider:   "custom-chromium",
			Version:    "148.0.0",
			Executable: filepath.Join(base.dataRoot, "source-kernel", "chrome"),
		},
		Fingerprint: domain.FingerprintConfig{
			Seed:         "source-fingerprint-seed",
			Platform:     runtime.GOOS,
			Brand:        "Chrome",
			Language:     "en-US",
			Timezone:     "UTC",
			ScreenWidth:  1440,
			ScreenHeight: 900,
			WebRTCPolicy: "default",
			CanvasMode:   "stable-noise",
			AudioMode:    "stable-noise",
			FontMode:     "native",
			GPUProfile:   "default",
		},
		Proxy: domain.ProxyConfig{
			URL:           "http://127.0.0.1:8080",
			CredentialRef: "source-credential-record",
			AdapterRef:    "source-adapter-record",
		},
		UserDataDir: base.profileRoot,
		Tags:        []string{"source"},
	})
	if err != nil {
		t.Fatal(err)
	}
	profileDefinition, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	base.request.OperationID = "snapshot-operation-a"
	base.request.IdempotencyKey = "snapshot-request-a"
	base.request.ProfileName = source.Name
	base.request.ProfileDefinition = profileDefinition
	base.request.Dependencies = DependencyRequirements{Kernel: KernelRequirement{
		ProviderID:       source.Kernel.Provider,
		BrowserVersion:   source.Kernel.Version,
		OperatingSystem:  runtime.GOOS,
		Architecture:     runtime.GOARCH,
		TrustRequirement: "custom",
	}}
	snapshotResult, err := base.creator.Create(context.Background(), base.request)
	if err != nil {
		t.Fatal(err)
	}

	kernels, err := kernel.Open(filepath.Join(base.dataRoot, "kernels.json"), filepath.Join(base.dataRoot, "kernels"))
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := adapter.Open(filepath.Join(base.dataRoot, "adapters.json"), filepath.Join(base.dataRoot, "adapters"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &memoryCredentialBackend{values: make(map[string]string)}
	credentials, err := credential.OpenWithBackend(filepath.Join(base.dataRoot, "credentials.json"), backend)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := OpenRestoreExecutor(base.dataRoot, profiles, base.records, base.journal, base.coordinator, kernels, adapters, credentials)
	if err != nil {
		t.Fatal(err)
	}
	executor.space = func(string) (uint64, error) { return math.MaxUint64, nil }

	request := RestoreRequest{
		OperationID:        "restore-operation-a",
		SnapshotID:         base.request.SnapshotID,
		IdempotencyKey:     "restore-request-a",
		ApplicationVersion: "0.15.0-dev",
		Name:               "Restored Profile",
		MaxDuration:        time.Minute,
	}
	return restoreHarness{
		snapshot:          base,
		snapshotResult:    snapshotResult,
		profiles:          profiles,
		executor:          executor,
		source:            source,
		request:           request,
		credentialBackend: backend,
		credentials:       credentials,
	}
}

func assertRestoreRolledBack(t *testing.T, harness restoreHarness) {
	t.Helper()
	destinationID := restoreDestinationID(harness.request.IdempotencyKey, harness.request.SnapshotID)
	if _, err := harness.profiles.Get(destinationID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("restored Profile metadata remains after rollback: %v", err)
	}
	if _, err := harness.snapshot.records.Get(destinationID); !errors.Is(err, lifecycle.ErrNotFound) {
		t.Fatalf("restore lifecycle reservation remains after rollback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(harness.executor.profilesRoot, destinationID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restored browser data remains after rollback: %v", err)
	}
}
