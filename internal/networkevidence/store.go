package networkevidence

import (
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
	ErrNotFound = errors.New("network evidence run not found")
	ErrExpired  = errors.New("network evidence run expired")
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
		return nil, fmt.Errorf("network evidence storage root is required")
	}
	if options.Retention <= 0 {
		options.Retention = defaultRetention
	}
	if options.MaxRuns <= 0 {
		options.MaxRuns = defaultMaxRuns
	}
	if options.MaxRuns > 1000 {
		return nil, fmt.Errorf("network evidence maximum run count is too large")
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

func (store *Store) Retention() time.Duration { return store.retention }

func (store *Store) Save(run Run) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	run = Normalize(run)
	if err := run.Validate(); err != nil {
		return err
	}
	if !validRunID(run.ID) {
		return fmt.Errorf("invalid network evidence run id")
	}
	payload, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("encode network evidence run: %w", err)
	}
	if len(payload) > maxReportBytes {
		return fmt.Errorf("network evidence report exceeds %d bytes", maxReportBytes)
	}
	if err := store.pruneLocked(); err != nil {
		return err
	}
	path := store.path(run.ID)
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("network evidence run %q already exists", run.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect network evidence destination: %w", err)
	}
	temporary, err := os.OpenFile(filepath.Join(store.root, "."+run.ID+".tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create network evidence temporary file: %w", err)
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
		return fmt.Errorf("write network evidence temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync network evidence temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close network evidence temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("commit network evidence run: %w", err)
	}
	cleanup = false
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect network evidence run: %w", err)
	}
	return store.enforceMaximumLocked()
}

func (store *Store) Get(id string) (Run, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !validRunID(id) {
		return Run{}, ErrNotFound
	}
	run, err := store.readLocked(store.path(id))
	if err != nil {
		return Run{}, err
	}
	if !run.ExpiresAt.After(store.now().UTC()) {
		return Run{}, ErrExpired
	}
	return run, nil
}

func (store *Store) List(profileID string) ([]Run, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.pruneLocked(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return nil, fmt.Errorf("list network evidence runs: %w", err)
	}
	items := make([]Run, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		run, err := store.readLocked(filepath.Join(store.root, entry.Name()))
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

func (store *Store) Delete(id string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !validRunID(id) {
		return ErrNotFound
	}
	path := store.path(id)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect network evidence run: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("network evidence run must be a regular file")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete network evidence run: %w", err)
	}
	return nil
}

func (store *Store) Prune() (int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	before, err := store.countLocked()
	if err != nil {
		return 0, err
	}
	if err := store.pruneLocked(); err != nil {
		return 0, err
	}
	if err := store.enforceMaximumLocked(); err != nil {
		return 0, err
	}
	after, err := store.countLocked()
	if err != nil {
		return 0, err
	}
	return before - after, nil
}

func (store *Store) readLocked(path string) (Run, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("inspect network evidence run: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Run{}, fmt.Errorf("network evidence run must be a regular file")
	}
	if info.Size() < 1 || info.Size() > maxReportBytes {
		return Run{}, fmt.Errorf("network evidence run has invalid size")
	}
	file, err := os.Open(path)
	if err != nil {
		return Run{}, fmt.Errorf("open network evidence run: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxReportBytes+1))
	decoder.DisallowUnknownFields()
	var run Run
	if err := decoder.Decode(&run); err != nil {
		return Run{}, fmt.Errorf("decode network evidence run: %w", err)
	}
	if err := run.Validate(); err != nil {
		return Run{}, fmt.Errorf("validate network evidence run: %w", err)
	}
	return run, nil
}

func (store *Store) pruneLocked() error {
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return fmt.Errorf("list network evidence runs: %w", err)
	}
	now := store.now().UTC()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(store.root, entry.Name())
		run, err := store.readLocked(path)
		if err != nil {
			return err
		}
		if !run.ExpiresAt.After(now) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("prune expired network evidence run: %w", err)
			}
		}
	}
	return nil
}

func (store *Store) enforceMaximumLocked() error {
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return fmt.Errorf("list network evidence runs: %w", err)
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
		path := filepath.Join(store.root, entry.Name())
		run, err := store.readLocked(path)
		if err != nil {
			return err
		}
		items = append(items, storedRun{path: path, run: run})
	}
	if len(items) <= store.maxRuns {
		return nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].run.StartedAt.Before(items[j].run.StartedAt) })
	for _, item := range items[:len(items)-store.maxRuns] {
		if err := os.Remove(item.path); err != nil {
			return fmt.Errorf("remove old network evidence run: %w", err)
		}
	}
	return nil
}

func (store *Store) countLocked() (int, error) {
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return 0, fmt.Errorf("list network evidence runs: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}
	return count, nil
}

func (store *Store) path(id string) string { return filepath.Join(store.root, id+".json") }

func preparePrivateDirectory(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("network evidence storage root must be a real directory")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect network evidence storage root: %w", err)
	} else if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create network evidence storage root: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("protect network evidence storage root: %w", err)
	}
	return nil
}

func validRunID(id string) bool {
	if !strings.HasPrefix(id, "netev-") || len(id) != len("netev-")+32 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(id, "netev-"))
	return err == nil
}
