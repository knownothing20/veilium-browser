package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type memoryBackend struct {
	items     map[string]string
	setErr    error
	getErr    error
	deleteErr error
}

func newMemoryBackend() *memoryBackend                      { return &memoryBackend{items: make(map[string]string)} }
func (b *memoryBackend) key(service, account string) string { return service + "\x00" + account }
func (b *memoryBackend) Set(service, account, secret string) error {
	if b.setErr != nil {
		return b.setErr
	}
	b.items[b.key(service, account)] = secret
	return nil
}
func (b *memoryBackend) Get(service, account string) (string, error) {
	if b.getErr != nil {
		return "", b.getErr
	}
	secret, ok := b.items[b.key(service, account)]
	if !ok {
		return "", ErrSecretNotFound
	}
	return secret, nil
}
func (b *memoryBackend) Delete(service, account string) error {
	if b.deleteErr != nil {
		return b.deleteErr
	}
	key := b.key(service, account)
	if _, ok := b.items[key]; !ok {
		return ErrSecretNotFound
	}
	delete(b.items, key)
	return nil
}

func TestSaveStoresOnlyMetadataOnDisk(t *testing.T) {
	root := t.TempDir()
	backend := newMemoryBackend()
	manager, err := OpenWithBackend(filepath.Join(root, "credentials.json"), backend)
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC) }
	record, err := manager.Save(SaveRequest{Name: "US proxy", Username: "alice", Secret: "top-secret-password"})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID == "" || record.Name != "US proxy" || record.Username != "alice" {
		t.Fatalf("unexpected record: %#v", record)
	}
	data, err := os.ReadFile(filepath.Join(root, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "top-secret-password") {
		t.Fatal("secret was written to credential metadata")
	}
	material, err := manager.Resolve(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if material.Username != "alice" || material.Secret != "top-secret-password" {
		t.Fatalf("unexpected material: %#v", material)
	}
}

func TestUpdateWithoutSecretPreservesVaultValue(t *testing.T) {
	manager, backend := newTestManager(t)
	record, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "one"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := manager.Save(SaveRequest{ID: record.ID, Name: "Proxy renamed", Username: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Proxy renamed" || updated.Username != "bob" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	secret, err := backend.Get(defaultService, accountName(record.ID))
	if err != nil || secret != "one" {
		t.Fatalf("secret changed unexpectedly: %q %v", secret, err)
	}
}

func TestUpdateCanRotateSecret(t *testing.T) {
	manager, backend := newTestManager(t)
	record, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Save(SaveRequest{ID: record.ID, Name: "Proxy", Username: "alice", Secret: "two"}); err != nil {
		t.Fatal(err)
	}
	secret, _ := backend.Get(defaultService, accountName(record.ID))
	if secret != "two" {
		t.Fatalf("expected rotated secret, got %q", secret)
	}
}

func TestCreateRollsBackSecretWhenMetadataPersistenceFails(t *testing.T) {
	manager, backend := newTestManager(t)
	manager.persist = func([]Record) error { return errors.New("disk full") }
	_, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "secret"})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected persistence error, got %v", err)
	}
	if len(manager.List()) != 0 || len(backend.items) != 0 {
		t.Fatalf("create rollback failed: records=%#v secrets=%#v", manager.List(), backend.items)
	}
}

func TestUpdateRollsBackSecretWhenMetadataPersistenceFails(t *testing.T) {
	manager, backend := newTestManager(t)
	record, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "one"})
	if err != nil {
		t.Fatal(err)
	}
	manager.persist = func([]Record) error { return errors.New("disk full") }
	if _, err := manager.Save(SaveRequest{ID: record.ID, Name: "Changed", Username: "bob", Secret: "two"}); err == nil {
		t.Fatal("expected update failure")
	}
	stored, _ := manager.Get(record.ID)
	secret, _ := backend.Get(defaultService, accountName(record.ID))
	if stored.Name != "Proxy" || stored.Username != "alice" || secret != "one" {
		t.Fatalf("update rollback failed: %#v secret=%q", stored, secret)
	}
}

func TestDeleteRollsBackWhenMetadataPersistenceFails(t *testing.T) {
	manager, backend := newTestManager(t)
	record, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "one"})
	if err != nil {
		t.Fatal(err)
	}
	manager.persist = func([]Record) error { return errors.New("disk full") }
	if err := manager.Delete(record.ID); err == nil {
		t.Fatal("expected delete failure")
	}
	if _, err := manager.Get(record.ID); err != nil {
		t.Fatalf("record was not restored: %v", err)
	}
	secret, err := backend.Get(defaultService, accountName(record.ID))
	if err != nil || secret != "one" {
		t.Fatalf("secret was not restored: %q %v", secret, err)
	}
}

func TestVaultFailureDoesNotFallbackToMetadata(t *testing.T) {
	manager, backend := newTestManager(t)
	backend.setErr = errors.New("vault unavailable")
	_, err := manager.Save(SaveRequest{Name: "Proxy", Username: "alice", Secret: "secret"})
	if err == nil || !strings.Contains(err.Error(), "vault unavailable") {
		t.Fatalf("expected vault error, got %v", err)
	}
	if len(manager.List()) != 0 {
		t.Fatal("metadata was created after vault failure")
	}
}

func newTestManager(t *testing.T) (*Manager, *memoryBackend) {
	t.Helper()
	backend := newMemoryBackend()
	manager, err := OpenWithBackend(filepath.Join(t.TempDir(), "credentials.json"), backend)
	if err != nil {
		t.Fatal(err)
	}
	return manager, backend
}
