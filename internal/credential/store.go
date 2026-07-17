package credential

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultService = "Veilium Browser"

var (
	ErrNotFound       = errors.New("credential not found")
	ErrSecretNotFound = errors.New("credential secret not found in operating-system vault")
)

type Backend interface {
	Set(service, account, secret string) error
	Get(service, account string) (string, error)
	Delete(service, account string) error
}

type Record struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SaveRequest struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Secret   string `json:"secret,omitempty"`
}

type Material struct {
	Username string
	Secret   string
}

type Manager struct {
	mu      sync.RWMutex
	path    string
	service string
	backend Backend
	items   map[string]Record
	now     func() time.Time
	persist func([]Record) error
}

func Open(path string) (*Manager, error) {
	return OpenWithBackend(path, keyringBackend{})
}

func OpenWithBackend(path string, backend Backend) (*Manager, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("credential metadata path is required")
	}
	if backend == nil {
		return nil, fmt.Errorf("credential vault backend is required")
	}
	manager := &Manager{
		path:    path,
		service: defaultService,
		backend: backend,
		items:   make(map[string]Record),
		now:     time.Now,
	}
	if err := manager.load(); err != nil {
		return nil, err
	}
	return manager, nil
}

func ProviderName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows Credential Manager"
	case "darwin":
		return "macOS Keychain"
	case "linux", "freebsd", "openbsd", "netbsd":
		return "Secret Service"
	default:
		return "Operating-system keyring"
	}
}

func (m *Manager) List() []Record {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return sortedRecords(m.items)
}

func (m *Manager) Get(id string) (Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.items[strings.TrimSpace(id)]
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}

func (m *Manager) Resolve(id string) (Material, error) {
	record, err := m.Get(id)
	if err != nil {
		return Material{}, err
	}
	secret, err := m.backend.Get(m.service, accountName(record.ID))
	if err != nil {
		return Material{}, fmt.Errorf("read secret from %s: %w", ProviderName(), err)
	}
	return Material{Username: record.Username, Secret: secret}, nil
}

func (m *Manager) Save(request SaveRequest) (Record, error) {
	request.ID = strings.TrimSpace(request.ID)
	request.Name = strings.TrimSpace(request.Name)
	request.Username = strings.TrimSpace(request.Username)
	if request.Name == "" {
		return Record{}, fmt.Errorf("credential name is required")
	}
	if request.Username == "" {
		return Record{}, fmt.Errorf("credential username is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if request.ID == "" {
		return m.createLocked(request)
	}
	return m.updateLocked(request)
}

func (m *Manager) Delete(id string) error {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.items[id]
	if !ok {
		return ErrNotFound
	}

	previousSecret, getErr := m.backend.Get(m.service, accountName(id))
	secretExists := getErr == nil
	if getErr != nil && !errors.Is(getErr, ErrSecretNotFound) {
		return fmt.Errorf("read secret before deletion from %s: %w", ProviderName(), getErr)
	}
	if secretExists {
		if err := m.backend.Delete(m.service, accountName(id)); err != nil && !errors.Is(err, ErrSecretNotFound) {
			return fmt.Errorf("delete secret from %s: %w", ProviderName(), err)
		}
	}

	delete(m.items, id)
	if err := m.persistLocked(); err != nil {
		m.items[id] = record
		if secretExists {
			_ = m.backend.Set(m.service, accountName(id), previousSecret)
		}
		return err
	}
	return nil
}

func (m *Manager) createLocked(request SaveRequest) (Record, error) {
	if request.Secret == "" {
		return Record{}, fmt.Errorf("credential secret is required")
	}
	id, err := newID()
	if err != nil {
		return Record{}, err
	}
	now := m.now().UTC()
	record := Record{ID: id, Name: request.Name, Username: request.Username, CreatedAt: now, UpdatedAt: now}
	if err := m.backend.Set(m.service, accountName(id), request.Secret); err != nil {
		return Record{}, fmt.Errorf("store secret in %s: %w", ProviderName(), err)
	}
	m.items[id] = record
	if err := m.persistLocked(); err != nil {
		delete(m.items, id)
		_ = m.backend.Delete(m.service, accountName(id))
		return Record{}, err
	}
	return record, nil
}

func (m *Manager) updateLocked(request SaveRequest) (Record, error) {
	previous, ok := m.items[request.ID]
	if !ok {
		return Record{}, ErrNotFound
	}
	updated := previous
	updated.Name = request.Name
	updated.Username = request.Username
	updated.UpdatedAt = m.now().UTC()

	var previousSecret string
	var previousSecretExists bool
	secretChanged := request.Secret != ""
	if secretChanged {
		secret, err := m.backend.Get(m.service, accountName(request.ID))
		if err == nil {
			previousSecret, previousSecretExists = secret, true
		} else if !errors.Is(err, ErrSecretNotFound) {
			return Record{}, fmt.Errorf("read existing secret from %s: %w", ProviderName(), err)
		}
		if err := m.backend.Set(m.service, accountName(request.ID), request.Secret); err != nil {
			return Record{}, fmt.Errorf("update secret in %s: %w", ProviderName(), err)
		}
	}

	m.items[request.ID] = updated
	if err := m.persistLocked(); err != nil {
		m.items[request.ID] = previous
		if secretChanged {
			if previousSecretExists {
				_ = m.backend.Set(m.service, accountName(request.ID), previousSecret)
			} else {
				_ = m.backend.Delete(m.service, accountName(request.ID))
			}
		}
		return Record{}, err
	}
	return updated, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read credential metadata: %w", err)
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("decode credential metadata: %w", err)
	}
	for _, record := range records {
		if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Name) == "" || strings.TrimSpace(record.Username) == "" {
			return fmt.Errorf("credential metadata contains an invalid record")
		}
		if _, duplicate := m.items[record.ID]; duplicate {
			return fmt.Errorf("credential metadata contains duplicate id %q", record.ID)
		}
		m.items[record.ID] = record
	}
	return nil
}

func (m *Manager) persistLocked() error {
	records := sortedRecords(m.items)
	if m.persist != nil {
		return m.persist(records)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return fmt.Errorf("create credential metadata directory: %w", err)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential metadata: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(m.path), ".credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary credential metadata: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("protect credential metadata: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write credential metadata: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync credential metadata: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close credential metadata: %w", err)
	}
	if err := os.Rename(tempName, m.path); err != nil {
		return fmt.Errorf("replace credential metadata: %w", err)
	}
	return nil
}

func sortedRecords(items map[string]Record) []Record {
	records := make([]Record, 0, len(items))
	for _, record := range items {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Name == records[j].Name {
			return records[i].ID < records[j].ID
		}
		return strings.ToLower(records[i].Name) < strings.ToLower(records[j].Name)
	})
	return records
}

func accountName(id string) string { return "proxy:" + id }

func newID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate credential id: %w", err)
	}
	return "cred_" + hex.EncodeToString(value), nil
}
