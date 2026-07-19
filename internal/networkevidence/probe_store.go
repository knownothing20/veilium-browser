package networkevidence

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const maxProbeSetBytes = 128 << 10

type ProbeStore struct {
	mu   sync.Mutex
	path string
}

func OpenProbeStore(path string) (*ProbeStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("network probe store path is required")
	}
	if err := preparePrivateDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}
	store := &ProbeStore{path: path}
	if _, exists, err := store.Get(); err != nil {
		return nil, err
	} else if !exists {
		return store, nil
	}
	return store, nil
}

func (store *ProbeStore) Get() (ProbeSet, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	info, err := os.Lstat(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return ProbeSet{}, false, nil
	}
	if err != nil {
		return ProbeSet{}, false, fmt.Errorf("inspect network probe configuration: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxProbeSetBytes {
		return ProbeSet{}, false, fmt.Errorf("network probe configuration must be a bounded regular file")
	}
	file, err := os.Open(store.path)
	if err != nil {
		return ProbeSet{}, false, fmt.Errorf("open network probe configuration: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxProbeSetBytes+1))
	decoder.DisallowUnknownFields()
	var set ProbeSet
	if err := decoder.Decode(&set); err != nil {
		return ProbeSet{}, false, fmt.Errorf("decode network probe configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ProbeSet{}, false, fmt.Errorf("network probe configuration contains trailing data")
	}
	set = NormalizeProbeSet(set)
	if err := set.Validate(); err != nil {
		return ProbeSet{}, false, fmt.Errorf("validate network probe configuration: %w", err)
	}
	return set, true, nil
}

func (store *ProbeStore) Save(set ProbeSet) (ProbeSet, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	set = NormalizeProbeSet(set)
	if err := set.Validate(); err != nil {
		return ProbeSet{}, err
	}
	payload, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return ProbeSet{}, fmt.Errorf("encode network probe configuration: %w", err)
	}
	if len(payload) > maxProbeSetBytes {
		return ProbeSet{}, fmt.Errorf("network probe configuration exceeds %d bytes", maxProbeSetBytes)
	}
	temporary, err := os.OpenFile(store.path+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return ProbeSet{}, fmt.Errorf("create network probe temporary file: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = temporary.Close()
			_ = os.Remove(temporary.Name())
		}
	}()
	if _, err := temporary.Write(payload); err != nil {
		return ProbeSet{}, fmt.Errorf("write network probe temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return ProbeSet{}, fmt.Errorf("sync network probe temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return ProbeSet{}, fmt.Errorf("close network probe temporary file: %w", err)
	}
	if err := os.Rename(temporary.Name(), store.path); err != nil {
		return ProbeSet{}, fmt.Errorf("commit network probe configuration: %w", err)
	}
	cleanup = false
	if err := os.Chmod(store.path, 0o600); err != nil {
		return ProbeSet{}, fmt.Errorf("protect network probe configuration: %w", err)
	}
	return set, nil
}

func (store *ProbeStore) Delete() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	info, err := os.Lstat(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect network probe configuration: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("network probe configuration must be a regular file")
	}
	if err := os.Remove(store.path); err != nil {
		return fmt.Errorf("delete network probe configuration: %w", err)
	}
	return nil
}
