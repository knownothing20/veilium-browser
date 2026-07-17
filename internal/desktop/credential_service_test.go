package desktop

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/profile"
)

type desktopVaultBackend struct{ items map[string]string }

func newDesktopVaultBackend() *desktopVaultBackend {
	return &desktopVaultBackend{items: make(map[string]string)}
}
func (b *desktopVaultBackend) key(service, account string) string { return service + "\x00" + account }
func (b *desktopVaultBackend) Set(service, account, secret string) error {
	b.items[b.key(service, account)] = secret
	return nil
}
func (b *desktopVaultBackend) Get(service, account string) (string, error) {
	secret, ok := b.items[b.key(service, account)]
	if !ok {
		return "", credential.ErrSecretNotFound
	}
	return secret, nil
}
func (b *desktopVaultBackend) Delete(service, account string) error {
	key := b.key(service, account)
	if _, ok := b.items[key]; !ok {
		return credential.ErrSecretNotFound
	}
	delete(b.items, key)
	return nil
}

func TestCredentialLifecycleAndProfileReferenceProtection(t *testing.T) {
	service, manager := credentialTestService(t)
	record, err := service.SaveCredential(credential.SaveRequest{Name: "US proxy", Username: "alice", Secret: "secret-value"})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Proxy = domain.ProxyConfig{URL: "http://127.0.0.1:8080", CredentialRef: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Proxy.CredentialRef != record.ID {
		t.Fatalf("credential reference was not preserved: %#v", created.Proxy)
	}
	if err := service.DeleteCredential(record.ID); err == nil || !strings.Contains(err.Error(), "used by profile") {
		t.Fatalf("expected in-use protection, got %v", err)
	}
	material, err := manager.Resolve(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if material.Username != "alice" || material.Secret != "secret-value" {
		t.Fatalf("unexpected vault material: %#v", material)
	}
}

func TestProfileRejectsUnknownOrDirectCredentialReference(t *testing.T) {
	service, _ := credentialTestService(t)
	input := validProfile()
	input.Proxy = domain.ProxyConfig{URL: "http://127.0.0.1:8080", CredentialRef: "cred_missing"}
	if _, err := service.CreateProfile(input); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected unknown reference rejection, got %v", err)
	}

	record, err := service.SaveCredential(credential.SaveRequest{Name: "Proxy", Username: "alice", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	input = validProfile()
	input.Proxy = domain.ProxyConfig{URL: "direct://", CredentialRef: record.ID}
	if _, err := service.CreateProfile(input); err == nil || !strings.Contains(err.Error(), "direct connections") {
		t.Fatalf("expected direct credential rejection, got %v", err)
	}
}

func TestBootstrapNeverSerializesCredentialSecret(t *testing.T) {
	service, _ := credentialTestService(t)
	if _, err := service.SaveCredential(credential.SaveRequest{Name: "Proxy", Username: "alice", Secret: "never-serialize-me"}); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(service.Bootstrap())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "never-serialize-me") {
		t.Fatal("bootstrap serialized a credential secret")
	}
}

func TestDeleteCredentialRemovesUnusedRecord(t *testing.T) {
	service, _ := credentialTestService(t)
	record, err := service.SaveCredential(credential.SaveRequest{Name: "Proxy", Username: "alice", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteCredential(record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.credentials.Get(record.ID); !errors.Is(err, credential.ErrNotFound) {
		t.Fatalf("expected deleted credential, got %v", err)
	}
}

func credentialTestService(t *testing.T) (*Service, *credential.Manager) {
	t.Helper()
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	manager, err := credential.OpenWithBackend(filepath.Join(root, "credentials.json"), newDesktopVaultBackend())
	if err != nil {
		t.Fatal(err)
	}
	service, err := newServiceWithCredentials(store, root, newFakeRuntime(), manager)
	if err != nil {
		t.Fatal(err)
	}
	return service, manager
}
