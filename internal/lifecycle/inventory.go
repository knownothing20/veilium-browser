package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type InventoryStatus string

const (
	InventoryPresent    InventoryStatus = "present"
	InventoryMissing    InventoryStatus = "missing"
	InventoryUnsafe     InventoryStatus = "unsafe"
	InventoryIncomplete InventoryStatus = "incomplete"
)

type StorageSummary struct {
	Files int64 `json:"files"`
	Bytes int64 `json:"bytes"`
}

type ProfileStorage struct {
	ProfileID   string          `json:"profileId"`
	ManagedDir  string          `json:"managedDir"`
	Status      InventoryStatus `json:"status"`
	Summary     StorageSummary  `json:"summary"`
	ReasonCode  string          `json:"reasonCode,omitempty"`
	Limitations []string        `json:"limitations,omitempty"`
}

type InventoryFinding struct {
	RelativePath string `json:"relativePath"`
	Kind         string `json:"kind"`
	ReasonCode   string `json:"reasonCode"`
}

type StorageInventory struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	ManagedRoot string             `json:"managedRoot"`
	Profiles    []ProfileStorage   `json:"profiles"`
	Orphans     []InventoryFinding `json:"orphans,omitempty"`
	Unsafe      []InventoryFinding `json:"unsafe,omitempty"`
	Summary     StorageSummary     `json:"summary"`
	Incomplete  bool               `json:"incomplete"`
	Limitations []string           `json:"limitations,omitempty"`
}

type InventoryScanner struct {
	DataRoot string
	MaxFiles int64
	MaxBytes int64
	Now      func() time.Time
}

func NewInventoryScanner(dataRoot string) (*InventoryScanner, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return nil, fmt.Errorf("managed data root is required")
	}
	root, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve managed data root: %w", err)
	}
	return &InventoryScanner{
		DataRoot: filepath.Clean(root),
		MaxFiles: 100000,
		MaxBytes: 64 << 30,
		Now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *InventoryScanner) Scan(ctx context.Context, records []Record) (StorageInventory, error) {
	report := StorageInventory{
		GeneratedAt: s.Now().UTC(),
		ManagedRoot: ".",
		Profiles:    make([]ProfileStorage, 0, len(records)),
	}
	if s.MaxFiles <= 0 || s.MaxBytes <= 0 {
		return report, fmt.Errorf("%w: inventory bounds must be positive", ErrInvalidRecord)
	}
	rootInfo, err := os.Lstat(s.DataRoot)
	if err != nil {
		return report, fmt.Errorf("inspect managed data root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return report, fmt.Errorf("%w: managed data root is unsafe", ErrInvalidRecord)
	}
	if unsafe, err := pathHasReparsePoint(s.DataRoot); err != nil {
		return report, fmt.Errorf("inspect managed data root: %w", err)
	} else if unsafe {
		return report, fmt.Errorf("%w: managed data root is a reparse point", ErrInvalidRecord)
	}

	expected := make(map[string]string, len(records))
	for _, record := range records {
		if err := record.Validate(); err != nil {
			return report, err
		}
		if existingProfileID, exists := expected[record.ManagedDir]; exists {
			return report, fmt.Errorf("%w: duplicate managed directory %q for profiles %q and %q", ErrInvalidRecord, record.ManagedDir, existingProfileID, record.ProfileID)
		}
		expected[record.ManagedDir] = record.ProfileID
		profile := ProfileStorage{ProfileID: record.ProfileID, ManagedDir: record.ManagedDir}
		absolute, err := s.resolve(record.ManagedDir)
		if err != nil {
			profile.Status = InventoryUnsafe
			profile.ReasonCode = "managed-path-escape"
			report.Unsafe = append(report.Unsafe, InventoryFinding{RelativePath: record.ManagedDir, Kind: "profile", ReasonCode: profile.ReasonCode})
			report.Profiles = append(report.Profiles, profile)
			continue
		}
		summary, status, reason, limitations, err := s.scanPath(ctx, absolute)
		if err != nil {
			return report, err
		}
		profile.Status = status
		profile.ReasonCode = reason
		profile.Summary = summary
		profile.Limitations = limitations
		report.Summary.Files += summary.Files
		report.Summary.Bytes += summary.Bytes
		if status == InventoryUnsafe {
			report.Unsafe = append(report.Unsafe, InventoryFinding{RelativePath: record.ManagedDir, Kind: "profile", ReasonCode: reason})
		}
		if status == InventoryIncomplete {
			report.Incomplete = true
		}
		report.Profiles = append(report.Profiles, profile)
	}

	profilesRoot, err := s.resolve("profiles")
	if err != nil {
		return report, err
	}
	if profilesInfo, statErr := os.Lstat(profilesRoot); statErr == nil {
		unsafe, reparseErr := pathHasReparsePoint(profilesRoot)
		if reparseErr != nil || profilesInfo.Mode()&os.ModeSymlink != 0 || unsafe || !profilesInfo.IsDir() {
			return report, fmt.Errorf("%w: managed profiles root is unsafe", ErrInvalidRecord)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return report, fmt.Errorf("inspect managed profiles root: %w", statErr)
	}
	entries, err := os.ReadDir(profilesRoot)
	if errors.Is(err, os.ErrNotExist) {
		entries = nil
	} else if err != nil {
		return report, fmt.Errorf("read managed profiles root: %w", err)
	}
	for _, entry := range entries {
		relative := filepath.ToSlash(filepath.Join("profiles", entry.Name()))
		if _, exists := expected[relative]; exists {
			continue
		}
		absolute := filepath.Join(profilesRoot, entry.Name())
		info, err := os.Lstat(absolute)
		if err != nil {
			report.Unsafe = append(report.Unsafe, InventoryFinding{RelativePath: relative, Kind: "uninspectable", ReasonCode: "lstat-failed"})
			continue
		}
		unsafe, reparseErr := pathHasReparsePoint(absolute)
		if reparseErr != nil || info.Mode()&os.ModeSymlink != 0 || unsafe || !info.IsDir() {
			reason := "unsafe-orphan-entry"
			if !info.IsDir() {
				reason = "orphan-not-directory"
			}
			report.Unsafe = append(report.Unsafe, InventoryFinding{RelativePath: relative, Kind: "orphan", ReasonCode: reason})
			continue
		}
		report.Orphans = append(report.Orphans, InventoryFinding{RelativePath: relative, Kind: "directory", ReasonCode: "unregistered-profile-directory"})
	}

	sort.Slice(report.Profiles, func(i, j int) bool { return report.Profiles[i].ProfileID < report.Profiles[j].ProfileID })
	sort.Slice(report.Orphans, func(i, j int) bool { return report.Orphans[i].RelativePath < report.Orphans[j].RelativePath })
	sort.Slice(report.Unsafe, func(i, j int) bool { return report.Unsafe[i].RelativePath < report.Unsafe[j].RelativePath })
	return report, nil
}

func (s *InventoryScanner) resolve(relative string) (string, error) {
	if err := validateManagedRelativePath(relative); err != nil && relative != "profiles" {
		return "", err
	}
	candidate := filepath.Clean(filepath.Join(s.DataRoot, filepath.FromSlash(relative)))
	prefix := s.DataRoot + string(filepath.Separator)
	if candidate != s.DataRoot && !strings.HasPrefix(candidate, prefix) {
		return "", fmt.Errorf("%w: path escaped managed root", ErrInvalidRecord)
	}
	return candidate, nil
}

func (s *InventoryScanner) scanPath(ctx context.Context, path string) (StorageSummary, InventoryStatus, string, []string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return StorageSummary{}, InventoryMissing, "managed-directory-missing", nil, nil
	}
	if err != nil {
		return StorageSummary{}, InventoryUnsafe, "lstat-failed", nil, nil
	}
	unsafe, err := pathHasReparsePoint(path)
	if err != nil {
		return StorageSummary{}, InventoryUnsafe, "reparse-inspection-failed", nil, nil
	}
	if info.Mode()&os.ModeSymlink != 0 || unsafe || !info.IsDir() {
		return StorageSummary{}, InventoryUnsafe, "managed-entry-unsafe", nil, nil
	}

	summary := StorageSummary{}
	status := InventoryPresent
	reason := ""
	limitations := []string(nil)
	err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			status = InventoryIncomplete
			reason = "walk-error"
			limitations = appendUnique(limitations, "inventory-walk-incomplete")
			return filepath.SkipDir
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if current == path {
			return nil
		}
		info, err := os.Lstat(current)
		if err != nil {
			status = InventoryIncomplete
			reason = "lstat-failed"
			limitations = appendUnique(limitations, "inventory-lstat-incomplete")
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		reparse, err := pathHasReparsePoint(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || reparse {
			status = InventoryUnsafe
			reason = "unsafe-link-or-reparse"
			limitations = appendUnique(limitations, "unsafe-entry-present")
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			status = InventoryUnsafe
			reason = "special-file-present"
			limitations = appendUnique(limitations, "unsupported-special-file")
			return nil
		}
		summary.Files++
		summary.Bytes += info.Size()
		if summary.Files > s.MaxFiles || summary.Bytes > s.MaxBytes {
			status = InventoryIncomplete
			reason = "inventory-bound-exceeded"
			limitations = appendUnique(limitations, "inventory-bound-exceeded")
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return summary, InventoryIncomplete, "inventory-cancelled", appendUnique(limitations, "inventory-cancelled"), nil
		}
		return summary, InventoryIncomplete, "inventory-walk-failed", limitations, fmt.Errorf("scan managed profile directory: %w", err)
	}
	return summary, status, reason, limitations, nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
