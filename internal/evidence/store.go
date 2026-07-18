package evidence

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetention = 30 * 24 * time.Hour
	defaultMaxRuns   = 100
	maxReportBytes   = 1 << 20
)

var (
	ErrNotFound = errors.New("evidence run not found")
	ErrExpired  = errors.New("evidence run expired")
)

type StoreOptions struct {
	Retention time.Duration
	MaxRuns   int
	Now       func() time.Time
}

type Store struct {
	mu        sync.Mutex
	root      string
	retention time.Duration
	maxRuns   int
	now       func() time.Time
}

func OpenStore(root string, options StoreOptions) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("evidence storage root is required")
	}
	if options.Retention <= 0 {
		options.Retention = defaultRetention
	}
	if options.MaxRuns <= 0 {
		options.MaxRuns = defaultMaxRuns
	}
	if options.MaxRuns > 1000 {
		return nil, fmt.Errorf("evidence maximum run count is too large")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if err := preparePrivateDirectory(root); err != nil {
		return nil, err
	}
	store := &Store{root: root, retention: options.Retention, maxRuns: options.MaxRuns, now: options.Now}
	if _, err := store.Prune(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewRunID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate evidence run id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func (s *Store) Retention() time.Duration { return s.retention }

func (s *Store) Save(run Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := run.Validate(); err != nil {
		return err
	}
	if !validRunID(run.ID) {
		return fmt.Errorf("invalid evidence run id")
	}
	payload, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("encode evidence run: %w", err)
	}
	if len(payload) > maxReportBytes {
		return fmt.Errorf("evidence report exceeds %d bytes", maxReportBytes)
	}
	if err := s.pruneLocked(); err != nil {
		return err
	}
	path := s.path(run.ID)
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("evidence run %q already exists", run.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect evidence destination: %w", err)
	}
	temporary, err := os.OpenFile(filepath.Join(s.root, "."+run.ID+".tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create evidence temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(payload); err != nil {
		return fmt.Errorf("write evidence temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync evidence temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close evidence temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("commit evidence run: %w", err)
	}
	cleanup = false
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect evidence run: %w", err)
	}
	return s.enforceMaximumLocked()
}

func (s *Store) Get(id string) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validRunID(id) {
		return Run{}, ErrNotFound
	}
	run, err := s.readLocked(s.path(id))
	if err != nil {
		return Run{}, err
	}
	if !run.ExpiresAt.After(s.now().UTC()) {
		return Run{}, ErrExpired
	}
	return run, nil
}

func (s *Store) List(profileID string) ([]Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.pruneLocked(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("list evidence runs: %w", err)
	}
	items := make([]Run, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		run, err := s.readLocked(filepath.Join(s.root, entry.Name()))
		if err != nil {
			return nil, err
		}
		if profileID == "" || run.ProfileID == profileID {
			items = append(items, run)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.After(items[j].StartedAt) })
	return items, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validRunID(id) {
		return ErrNotFound
	}
	path := s.path(id)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect evidence run: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("evidence run must be a regular file")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete evidence run: %w", err)
	}
	return nil
}

func (s *Store) Prune() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before, err := s.countLocked()
	if err != nil {
		return 0, err
	}
	if err := s.pruneLocked(); err != nil {
		return 0, err
	}
	if err := s.enforceMaximumLocked(); err != nil {
		return 0, err
	}
	after, err := s.countLocked()
	if err != nil {
		return 0, err
	}
	return before - after, nil
}

func (s *Store) readLocked(path string) (Run, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("inspect evidence run: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Run{}, fmt.Errorf("evidence run must be a regular file")
	}
	if info.Size() < 1 || info.Size() > maxReportBytes {
		return Run{}, fmt.Errorf("evidence run has invalid size")
	}
	file, err := os.Open(path)
	if err != nil {
		return Run{}, fmt.Errorf("open evidence run: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxReportBytes+1))
	decoder.DisallowUnknownFields()
	var run Run
	if err := decoder.Decode(&run); err != nil {
		return Run{}, fmt.Errorf("decode evidence run: %w", err)
	}
	if err := run.Validate(); err != nil {
		return Run{}, fmt.Errorf("validate evidence run: %w", err)
	}
	return run, nil
}

func (s *Store) pruneLocked() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("list evidence runs: %w", err)
	}
	now := s.now().UTC()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.root, entry.Name())
		run, err := s.readLocked(path)
		if err != nil {
			return err
		}
		if !run.ExpiresAt.After(now) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("prune expired evidence run: %w", err)
			}
		}
	}
	return nil
}

func (s *Store) enforceMaximumLocked() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("list evidence runs: %w", err)
	}
	type storedRun struct {
		path string
		run  Run
	}
	items := make([]storedRun, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.root, entry.Name())
		run, err := s.readLocked(path)
		if err != nil {
			return err
		}
		items = append(items, storedRun{path: path, run: run})
	}
	if len(items) <= s.maxRuns {
		return nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].run.StartedAt.Before(items[j].run.StartedAt) })
	for _, item := range items[:len(items)-s.maxRuns] {
		if err := os.Remove(item.path); err != nil {
			return fmt.Errorf("remove old evidence run: %w", err)
		}
	}
	return nil
}

func (s *Store) countLocked() (int, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return 0, fmt.Errorf("list evidence runs: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}
	return count, nil
}

func (s *Store) path(id string) string { return filepath.Join(s.root, id+".json") }

func preparePrivateDirectory(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("evidence storage root must be a real directory")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect evidence storage root: %w", err)
	} else if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create evidence storage root: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("protect evidence storage root: %w", err)
	}
	return nil
}

func validRunID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}
