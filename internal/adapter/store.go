package adapter

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

	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
)

const (
	StatusVerified = "verified"
	StatusModified = "modified"
	StatusMissing  = "missing"
)

var ErrNotFound = errors.New("proxy adapter not found")

type Record struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Kind             string    `json:"kind"`
	Version          string    `json:"version"`
	Executable       string    `json:"executable"`
	SHA256           string    `json:"sha256"`
	SizeBytes        int64     `json:"sizeBytes"`
	LicenseSPDX      string    `json:"licenseSpdx"`
	SourceURL        string    `json:"sourceUrl"`
	Protocols        []string  `json:"protocols"`
	Status           string    `json:"status"`
	ImportedAt       time.Time `json:"importedAt"`
	VerifiedAt       time.Time `json:"verifiedAt"`
	Official         bool      `json:"official"`
	OfficialTag      string    `json:"officialTag,omitempty"`
	OfficialAsset    string    `json:"officialAsset,omitempty"`
	OfficialPlatform string    `json:"officialPlatform,omitempty"`
	OfficialArch     string    `json:"officialArch,omitempty"`
}

type ImportRequest struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Version     string `json:"version"`
	SourcePath  string `json:"sourcePath"`
	LicenseSPDX string `json:"licenseSpdx"`
	SourceURL   string `json:"sourceUrl"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	root  string
	items map[string]Record
}

func Open(path, root string) (*Store, error) {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("adapter store path and root are required")
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
		items = append(items, cloneRecord(item))
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
	item, ok := s.items[strings.TrimSpace(id)]
	if !ok {
		return Record{}, ErrNotFound
	}
	return cloneRecord(item), nil
}

func (s *Store) Import(request ImportRequest) (Record, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Kind = NormalizeKind(request.Kind)
	request.Version = strings.TrimSpace(request.Version)
	request.SourcePath = strings.TrimSpace(request.SourcePath)
	request.LicenseSPDX = strings.TrimSpace(request.LicenseSPDX)
	request.SourceURL = strings.TrimSpace(request.SourceURL)

	if request.Name == "" || len(request.Name) > 120 || strings.ContainsAny(request.Name, "\r\n\x00") {
		return Record{}, fmt.Errorf("adapter name is invalid")
	}
	if request.SourcePath == "" {
		return Record{}, fmt.Errorf("adapter source path is required")
	}
	if err := ValidateKind(request.Kind); err != nil {
		return Record{}, err
	}
	if err := ValidateVersion(request.Version); err != nil {
		return Record{}, err
	}
	if err := ValidateLicenseSPDX(request.LicenseSPDX); err != nil {
		return Record{}, err
	}
	if err := ValidateSourceURL(request.SourceURL); err != nil {
		return Record{}, err
	}

	sourceInfo, err := os.Lstat(request.SourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("inspect adapter source: %w", err)
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return Record{}, fmt.Errorf("adapter source must not be a symbolic link")
	}
	if !sourceInfo.Mode().IsRegular() {
		return Record{}, fmt.Errorf("adapter source must be a regular file")
	}

	source, err := os.Open(request.SourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("open adapter source: %w", err)
	}
	defer source.Close()
	openedInfo, err := source.Stat()
	if err != nil {
		return Record{}, fmt.Errorf("inspect opened adapter source: %w", err)
	}
	if !os.SameFile(sourceInfo, openedInfo) {
		return Record{}, fmt.Errorf("adapter source changed while opening")
	}

	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return Record{}, fmt.Errorf("create adapter root: %w", err)
	}
	temp, err := os.CreateTemp(s.root, ".adapter-import-*")
	if err != nil {
		return Record{}, fmt.Errorf("create adapter import file: %w", err)
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
		return Record{}, fmt.Errorf("protect imported adapter: %w", err)
	}

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(temp, hasher), source)
	if err != nil {
		return Record{}, fmt.Errorf("copy adapter source: %w", err)
	}
	if size == 0 {
		return Record{}, fmt.Errorf("adapter source is empty")
	}
	if err := temp.Sync(); err != nil {
		return Record{}, fmt.Errorf("sync imported adapter: %w", err)
	}
	if err := temp.Close(); err != nil {
		return Record{}, fmt.Errorf("close imported adapter: %w", err)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.SHA256 == digest && existing.Kind == request.Kind && existing.Version == request.Version {
			return cloneRecord(existing), nil
		}
	}

	id, err := recordID(request.Kind, digest)
	if err != nil {
		return Record{}, err
	}
	destinationDir := filepath.Join(s.root, id)
	if err := os.Mkdir(destinationDir, 0o700); err != nil {
		return Record{}, fmt.Errorf("create managed adapter directory: %w", err)
	}
	destination := filepath.Join(destinationDir, safeFilename(filepath.Base(request.SourcePath)))
	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.RemoveAll(destinationDir)
		return Record{}, fmt.Errorf("activate imported adapter: %w", err)
	}
	committed = true

	now := time.Now().UTC()
	record := Record{
		ID: id, Name: request.Name, Kind: request.Kind, Version: request.Version,
		Executable: destination, SHA256: digest, SizeBytes: size,
		LicenseSPDX: request.LicenseSPDX, SourceURL: request.SourceURL,
		Protocols: ProtocolsForKind(request.Kind), Status: StatusVerified,
		ImportedAt: now, VerifiedAt: now,
	}
	applyOfficialIdentity(&record)
	s.items[id] = record
	if err := s.persistLocked(); err != nil {
		delete(s.items, id)
		_ = os.RemoveAll(destinationDir)
		return Record{}, err
	}
	return cloneRecord(record), nil
}

func (s *Store) Verify(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[strings.TrimSpace(id)]
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
	applyOfficialIdentity(&record)
	s.items[record.ID] = record
	if err := s.persistLocked(); err != nil {
		s.items[record.ID] = previous
		return Record{}, err
	}
	return cloneRecord(record), nil
}

func (s *Store) Delete(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[strings.TrimSpace(id)]
	if !ok {
		return Record{}, ErrNotFound
	}
	directory := filepath.Join(s.root, record.ID)
	if !isWithin(s.root, directory) {
		return Record{}, fmt.Errorf("refusing to remove adapter outside managed root")
	}
	trash := filepath.Join(s.root, ".trash-"+record.ID+"-"+time.Now().UTC().Format("20060102150405.000000000"))
	moved := false
	if _, err := os.Stat(directory); err == nil {
		if err := os.Rename(directory, trash); err != nil {
			return Record{}, fmt.Errorf("quarantine managed adapter: %w", err)
		}
		moved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, fmt.Errorf("inspect managed adapter directory: %w", err)
	}

	delete(s.items, record.ID)
	if err := s.persistLocked(); err != nil {
		s.items[record.ID] = record
		if moved {
			_ = os.Rename(trash, directory)
		}
		return Record{}, err
	}
	if moved {
		_ = os.RemoveAll(trash)
	}
	return cloneRecord(record), nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read adapter store: %w", err)
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("decode adapter store: %w", err)
	}
	for _, record := range records {
		if err := ValidateKind(record.Kind); err != nil {
			return err
		}
		if err := ValidateVersion(record.Version); err != nil {
			return err
		}
		if err := ValidateLicenseSPDX(record.LicenseSPDX); err != nil {
			return err
		}
		if err := ValidateSourceURL(record.SourceURL); err != nil {
			return err
		}
		record.Kind = NormalizeKind(record.Kind)
		record.Protocols = ProtocolsForKind(record.Kind)
		applyOfficialIdentity(&record)
		if _, duplicate := s.items[record.ID]; duplicate {
			return fmt.Errorf("adapter store contains duplicate id %q", record.ID)
		}
		s.items[record.ID] = record
	}
	return nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create adapter metadata directory: %w", err)
	}
	records := make([]Record, 0, len(s.items))
	for _, record := range s.items {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode adapter store: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".adapters-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary adapter store: %w", err)
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
		return fmt.Errorf("replace adapter store: %w", err)
	}
	return nil
}

func inspectManagedFile(path string) (string, int64, string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", 0, StatusMissing, nil
	}
	if err != nil {
		return "", 0, "", fmt.Errorf("inspect managed adapter: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, StatusModified, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, "", fmt.Errorf("open managed adapter: %w", err)
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, "", fmt.Errorf("hash managed adapter: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, StatusVerified, nil
}

func recordID(kind, digest string) (string, error) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("generate adapter id: %w", err)
	}
	return fmt.Sprintf("%s-%s-%s", NormalizeKind(kind), digest[:12], hex.EncodeToString(suffix)), nil
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "proxy-adapter"
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

func applyOfficialIdentity(record *Record) {
	if record == nil {
		return
	}
	record.Official = false
	record.OfficialTag = ""
	record.OfficialAsset = ""
	record.OfficialPlatform = ""
	record.OfficialArch = ""
	pin, ok := adapterrelease.MatchExecutable(record.Kind, record.Version, record.SHA256, record.SizeBytes)
	if !ok {
		return
	}
	record.Official = true
	record.OfficialTag = pin.Tag
	record.OfficialAsset = pin.AssetName
	record.OfficialPlatform = pin.Platform
	record.OfficialArch = pin.Arch
	record.LicenseSPDX = pin.LicenseSPDX
	record.SourceURL = pin.AssetURL
}

func cloneRecord(record Record) Record {
	record.Protocols = append([]string(nil), record.Protocols...)
	return record
}
