package kernel

import (
	"crypto/rand"
	"crypto/sha256"
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

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

const (
	StatusVerified = "verified"
	StatusModified = "modified"
	StatusMissing  = "missing"
)

var ErrNotFound = errors.New("kernel not found")

type Record struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Provider          string    `json:"provider"`
	Version           string    `json:"version"`
	Executable        string    `json:"executable"`
	SHA256            string    `json:"sha256"`
	SizeBytes         int64     `json:"sizeBytes"`
	Status            string    `json:"status"`
	ImportedAt        time.Time `json:"importedAt"`
	VerifiedAt        time.Time `json:"verifiedAt"`
	PackageRoot       string    `json:"packageRoot,omitempty"`
	PackageTreeSHA256 string    `json:"packageTreeSha256,omitempty"`
	PackageFileCount  int       `json:"packageFileCount,omitempty"`
	PackageSizeBytes  int64     `json:"packageSizeBytes,omitempty"`
	SnapshotRevision  int64     `json:"snapshotRevision,omitempty"`
	ArchiveSHA256     string    `json:"archiveSha256,omitempty"`
}

type ImportRequest struct {
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	Version    string `json:"version"`
	SourcePath string `json:"sourcePath"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	root  string
	items map[string]Record
}

func Open(path, root string) (*Store, error) {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("kernel store path and root are required")
	}
	store := &Store{path: path, root: root, items: make(map[string]Record)}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Record, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ImportedAt.Equal(items[j].ImportedAt) {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return items[i].ImportedAt.After(items[j].ImportedAt)
	})
	return items
}

func (s *Store) Get(id string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	return item, nil
}

func (s *Store) Import(request ImportRequest) (Record, error) {
	name := strings.TrimSpace(request.Name)
	sourcePath := strings.TrimSpace(request.SourcePath)
	if name == "" {
		return Record{}, fmt.Errorf("kernel name is required")
	}
	if sourcePath == "" {
		return Record{}, fmt.Errorf("kernel source path is required")
	}
	capabilities, err := fingerprint.For(strings.TrimSpace(request.Provider), strings.TrimSpace(request.Version))
	if err != nil {
		return Record{}, err
	}
	if capabilities.IsReviewed() {
		return Record{}, fmt.Errorf("reviewed kernel providers require the pinned package installer")
	}

	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("inspect kernel source: %w", err)
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return Record{}, fmt.Errorf("kernel source must not be a symbolic link")
	}
	if !sourceInfo.Mode().IsRegular() {
		return Record{}, fmt.Errorf("kernel source must be a regular file")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("open kernel source: %w", err)
	}
	defer source.Close()
	openedInfo, err := source.Stat()
	if err != nil {
		return Record{}, fmt.Errorf("inspect opened kernel source: %w", err)
	}
	if !os.SameFile(sourceInfo, openedInfo) {
		return Record{}, fmt.Errorf("kernel source changed while opening")
	}

	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return Record{}, fmt.Errorf("create kernel root: %w", err)
	}
	temp, err := os.CreateTemp(s.root, ".kernel-import-*")
	if err != nil {
		return Record{}, fmt.Errorf("create kernel import file: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o700); err != nil {
		return Record{}, fmt.Errorf("protect imported kernel: %w", err)
	}

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(temp, hasher), source)
	if err != nil {
		return Record{}, fmt.Errorf("copy kernel source: %w", err)
	}
	if size == 0 {
		return Record{}, fmt.Errorf("kernel source is empty")
	}
	if err := temp.Sync(); err != nil {
		return Record{}, fmt.Errorf("sync imported kernel: %w", err)
	}
	if err := temp.Close(); err != nil {
		return Record{}, fmt.Errorf("close imported kernel: %w", err)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	provider := strings.TrimSpace(request.Provider)
	version := strings.TrimSpace(request.Version)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.SHA256 == digest && existing.Provider == provider && existing.Version == version {
			return existing, nil
		}
	}

	id, err := recordID(provider, capabilities.MajorVersion, digest)
	if err != nil {
		return Record{}, err
	}
	destinationDir := filepath.Join(s.root, id)
	if err := os.Mkdir(destinationDir, 0o700); err != nil {
		return Record{}, fmt.Errorf("create managed kernel directory: %w", err)
	}
	destination := filepath.Join(destinationDir, safeFilename(filepath.Base(sourcePath)))
	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.RemoveAll(destinationDir)
		return Record{}, fmt.Errorf("activate imported kernel: %w", err)
	}
	committed = true

	now := time.Now().UTC()
	record := Record{
		ID: id, Name: name, Provider: provider, Version: version,
		Executable: destination, SHA256: digest, SizeBytes: size,
		Status: StatusVerified, ImportedAt: now, VerifiedAt: now,
	}
	s.items[id] = record
	if err := s.persistLocked(); err != nil {
		delete(s.items, id)
		_ = os.RemoveAll(destinationDir)
		return Record{}, err
	}
	return record, nil
}

func (s *Store) Verify(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	previous := record
	digest, size, status, err := inspectManagedFile(record.Executable)
	if err != nil {
		return Record{}, err
	}
	record.Status = status
	record.VerifiedAt = time.Now().UTC()
	if status == StatusVerified && (digest != record.SHA256 || size != record.SizeBytes) {
		record.Status = StatusModified
	}
	if record.Status == StatusVerified && record.PackageRoot != "" {
		packageStatus, packageErr := verifyPackageRecord(record, s.root)
		if packageErr != nil {
			return Record{}, packageErr
		}
		if packageStatus != StatusVerified {
			record.Status = packageStatus
		}
	}
	s.items[id] = record
	if err := s.persistLocked(); err != nil {
		s.items[id] = previous
		return Record{}, err
	}
	return record, nil
}

func (s *Store) Delete(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	directory := filepath.Join(s.root, id)
	if !isWithin(s.root, directory) {
		return Record{}, fmt.Errorf("refusing to remove kernel outside managed root")
	}
	trash := filepath.Join(s.root, ".trash-"+id+"-"+time.Now().UTC().Format("20060102150405.000000000"))
	moved := false
	if _, err := os.Stat(directory); err == nil {
		if err := os.Rename(directory, trash); err != nil {
			return Record{}, fmt.Errorf("quarantine managed kernel: %w", err)
		}
		moved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, fmt.Errorf("inspect managed kernel directory: %w", err)
	}

	delete(s.items, id)
	if err := s.persistLocked(); err != nil {
		s.items[id] = record
		if moved {
			_ = os.Rename(trash, directory)
		}
		return Record{}, err
	}
	if moved {
		_ = os.RemoveAll(trash)
	}
	return record, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read kernel store: %w", err)
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("decode kernel store: %w", err)
	}
	for _, record := range records {
		s.items[record.ID] = record
	}
	return nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create kernel metadata directory: %w", err)
	}
	records := make([]Record, 0, len(s.items))
	for _, record := range s.items {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode kernel store: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".kernels-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary kernel store: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, s.path); err != nil {
		return fmt.Errorf("replace kernel store: %w", err)
	}
	return nil
}

func inspectManagedFile(path string) (string, int64, string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", 0, StatusMissing, nil
	}
	if err != nil {
		return "", 0, "", fmt.Errorf("inspect managed kernel: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, StatusModified, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, "", fmt.Errorf("open managed kernel: %w", err)
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, "", fmt.Errorf("hash managed kernel: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, StatusVerified, nil
}

func recordID(provider string, major int, digest string) (string, error) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("generate kernel id: %w", err)
	}
	provider = strings.NewReplacer("_", "-", "/", "-").Replace(strings.ToLower(provider))
	return fmt.Sprintf("%s-%d-%s-%s", provider, major, digest[:12], hex.EncodeToString(suffix)), nil
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "chromium-kernel"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', 0:
			return '_'
		default:
			return r
		}
	}, name)
}

func isWithin(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
