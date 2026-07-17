package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

var ErrNotFound = errors.New("profile not found")

type Store struct {
	mu       sync.RWMutex
	path     string
	profiles map[string]domain.Profile
}

func Open(path string) (*Store, error) {
	store := &Store{path: path, profiles: make(map[string]domain.Profile)}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) List() []domain.Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Profile, 0, len(s.profiles))
	for _, item := range s.profiles {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items
}

func (s *Store) Get(id string) (domain.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.profiles[id]
	if !ok {
		return domain.Profile{}, ErrNotFound
	}
	return item, nil
}

func (s *Store) Create(input domain.Profile) (domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(input.ID) == "" {
		id, err := newID()
		if err != nil {
			return domain.Profile{}, err
		}
		input.ID = id
	}
	if _, exists := s.profiles[input.ID]; exists {
		return domain.Profile{}, fmt.Errorf("profile %q already exists", input.ID)
	}
	now := time.Now().UTC()
	input.CreatedAt = now
	input.UpdatedAt = now
	s.profiles[input.ID] = input
	if err := s.persistLocked(); err != nil {
		delete(s.profiles, input.ID)
		return domain.Profile{}, err
	}
	return input, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, exists := s.profiles[id]
	if !exists {
		return ErrNotFound
	}
	delete(s.profiles, id)
	if err := s.persistLocked(); err != nil {
		s.profiles[id] = old
		return err
	}
	return nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read profile store: %w", err)
	}
	var items []domain.Profile
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("decode profile store: %w", err)
	}
	for _, item := range items {
		s.profiles[item.ID] = item
	}
	return nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}
	items := make([]domain.Profile, 0, len(s.profiles))
	for _, item := range s.profiles {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile store: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".profiles-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary profile store: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, s.path); err != nil {
		return fmt.Errorf("replace profile store: %w", err)
	}
	return nil
}

func newID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate profile id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
