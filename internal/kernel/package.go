package kernel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

const maxManagedPackageFiles = 5000

type PackageImportRequest struct {
	Name             string `json:"name"`
	Provider         string `json:"provider"`
	Version          string `json:"version"`
	SourceRoot       string `json:"sourceRoot"`
	ExecutablePath   string `json:"executablePath"`
	SnapshotRevision int64  `json:"snapshotRevision"`
	ArchiveSHA256    string `json:"archiveSha256"`
}

type PackageTreeIdentity struct {
	SHA256    string `json:"sha256"`
	FileCount int    `json:"fileCount"`
	SizeBytes int64  `json:"sizeBytes"`
}

type packageFileIdentity struct {
	path   string
	size   int64
	digest string
}

func (s *Store) ImportPackage(request PackageImportRequest) (Record, error) {
	name := strings.TrimSpace(request.Name)
	provider := strings.TrimSpace(request.Provider)
	version := strings.TrimSpace(request.Version)
	sourceRoot := strings.TrimSpace(request.SourceRoot)
	executablePath := strings.TrimSpace(request.ExecutablePath)
	if name == "" || sourceRoot == "" {
		return Record{}, fmt.Errorf("kernel package name and source root are required")
	}
	capabilities, err := fingerprint.For(provider, version)
	if err != nil {
		return Record{}, err
	}
	if !capabilities.IsReviewed() {
		return Record{}, fmt.Errorf("package imports are reserved for reviewed kernel providers")
	}
	release, ok := kernelrelease.Find(provider, version, "windows", "amd64")
	if !ok {
		return Record{}, fmt.Errorf("no exact reviewed package release exists for %s %s", provider, version)
	}
	if request.SnapshotRevision != release.SnapshotRevision || strings.ToLower(strings.TrimSpace(request.ArchiveSHA256)) != release.ArchiveSHA256 {
		return Record{}, fmt.Errorf("kernel package release identity does not match the embedded pin")
	}
	if executablePath != release.ExecutablePath || !safePackagePath(executablePath) {
		return Record{}, fmt.Errorf("kernel package executable path does not match the embedded pin")
	}

	tree, err := InspectPackageTree(sourceRoot)
	if err != nil {
		return Record{}, err
	}
	if tree.SHA256 != release.PackageTreeSHA256 || tree.FileCount != release.PackageFileCount || tree.SizeBytes != release.ExpandedSizeBytes {
		return Record{}, fmt.Errorf("kernel package tree does not match the embedded pin")
	}
	executableSource := filepath.Join(sourceRoot, filepath.FromSlash(executablePath))
	digest, size, status, err := inspectManagedFile(executableSource)
	if err != nil {
		return Record{}, fmt.Errorf("inspect reviewed Chromium executable: %w", err)
	}
	if status != StatusVerified || digest != release.ExecutableSHA256 || size != release.ExecutableSizeBytes {
		return Record{}, fmt.Errorf("kernel package executable does not match the embedded pin")
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return Record{}, fmt.Errorf("create kernel root: %w", err)
	}
	staging, err := os.MkdirTemp(s.root, ".kernel-package-*")
	if err != nil {
		return Record{}, fmt.Errorf("create kernel package staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		return Record{}, fmt.Errorf("protect kernel package staging directory: %w", err)
	}
	stagedPackage := filepath.Join(staging, "package")
	if err := copyPackage(sourceRoot, stagedPackage, executablePath); err != nil {
		return Record{}, err
	}
	stagedTree, err := InspectPackageTree(stagedPackage)
	if err != nil {
		return Record{}, fmt.Errorf("verify copied kernel package: %w", err)
	}
	if stagedTree != tree {
		return Record{}, fmt.Errorf("copied kernel package identity changed during import")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.Provider == provider && existing.Version == version && existing.SHA256 == digest && existing.PackageTreeSHA256 == tree.SHA256 {
			verified, verifyErr := verifyPackageRecord(existing, s.root)
			if verifyErr != nil {
				return Record{}, verifyErr
			}
			if verified != StatusVerified {
				return Record{}, fmt.Errorf("existing reviewed kernel package %q is %s; remove or repair it before reinstalling", existing.Name, verified)
			}
			return existing, nil
		}
	}
	id, err := recordID(provider, capabilities.MajorVersion, digest)
	if err != nil {
		return Record{}, err
	}
	destinationDir := filepath.Join(s.root, id)
	if err := os.Rename(staging, destinationDir); err != nil {
		return Record{}, fmt.Errorf("activate reviewed kernel package: %w", err)
	}
	committed = true
	packageRoot := filepath.Join(destinationDir, "package")
	now := time.Now().UTC()
	record := Record{
		ID: id, Name: name, Provider: provider, Version: version,
		Executable: filepath.Join(packageRoot, filepath.FromSlash(executablePath)),
		SHA256:     digest, SizeBytes: size, Status: StatusVerified,
		ImportedAt: now, VerifiedAt: now,
		PackageRoot: packageRoot, PackageTreeSHA256: tree.SHA256,
		PackageFileCount: tree.FileCount, PackageSizeBytes: tree.SizeBytes,
		SnapshotRevision: release.SnapshotRevision, ArchiveSHA256: release.ArchiveSHA256,
	}
	s.items[id] = record
	if err := s.persistLocked(); err != nil {
		delete(s.items, id)
		_ = os.RemoveAll(destinationDir)
		return Record{}, err
	}
	return record, nil
}

func InspectPackageTree(root string) (PackageTreeIdentity, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return PackageTreeIdentity{}, fmt.Errorf("kernel package root is required")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return PackageTreeIdentity{}, fmt.Errorf("inspect kernel package root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return PackageTreeIdentity{}, fmt.Errorf("kernel package root must be a real directory")
	}
	files := make([]packageFileIdentity, 0, 300)
	var total int64
	err = filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		entryInfo, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("kernel package contains a symbolic link")
		}
		if entry.IsDir() {
			if !entryInfo.IsDir() {
				return fmt.Errorf("kernel package contains an invalid directory entry")
			}
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("kernel package contains a special file")
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !safePackagePath(relative) {
			return fmt.Errorf("kernel package contains an unsafe path %q", relative)
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		openedInfo, statErr := file.Stat()
		if statErr != nil {
			_ = file.Close()
			return statErr
		}
		if !os.SameFile(entryInfo, openedInfo) {
			_ = file.Close()
			return fmt.Errorf("kernel package file changed while opening")
		}
		hasher := sha256.New()
		size, copyErr := io.Copy(hasher, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files = append(files, packageFileIdentity{path: relative, size: size, digest: hex.EncodeToString(hasher.Sum(nil))})
		total += size
		if len(files) > maxManagedPackageFiles {
			return fmt.Errorf("kernel package contains too many files")
		}
		return nil
	})
	if err != nil {
		return PackageTreeIdentity{}, fmt.Errorf("inspect kernel package tree: %w", err)
	}
	if len(files) == 0 {
		return PackageTreeIdentity{}, fmt.Errorf("kernel package is empty")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	treeHasher := sha256.New()
	for _, file := range files {
		_, _ = io.WriteString(treeHasher, file.path)
		_, _ = treeHasher.Write([]byte{0})
		_, _ = io.WriteString(treeHasher, strconv.FormatInt(file.size, 10))
		_, _ = treeHasher.Write([]byte{0})
		_, _ = io.WriteString(treeHasher, file.digest)
		_, _ = treeHasher.Write([]byte{'\n'})
	}
	return PackageTreeIdentity{SHA256: hex.EncodeToString(treeHasher.Sum(nil)), FileCount: len(files), SizeBytes: total}, nil
}

func verifyPackageRecord(record Record, managedRoot string) (string, error) {
	if !isWithin(managedRoot, record.PackageRoot) {
		return StatusModified, nil
	}
	info, err := os.Lstat(record.PackageRoot)
	if os.IsNotExist(err) {
		return StatusMissing, nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect managed kernel package: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return StatusModified, nil
	}
	tree, err := InspectPackageTree(record.PackageRoot)
	if err != nil {
		return StatusModified, nil
	}
	if tree.SHA256 != record.PackageTreeSHA256 || tree.FileCount != record.PackageFileCount || tree.SizeBytes != record.PackageSizeBytes {
		return StatusModified, nil
	}
	release, ok := kernelrelease.MatchPackage(record.Provider, record.Version, record.SHA256, record.SizeBytes, tree.SHA256, tree.FileCount, tree.SizeBytes)
	if !ok || record.SnapshotRevision != release.SnapshotRevision || record.ArchiveSHA256 != release.ArchiveSHA256 {
		return StatusModified, nil
	}
	relativeExecutable, err := filepath.Rel(record.PackageRoot, record.Executable)
	if err != nil || filepath.ToSlash(relativeExecutable) != release.ExecutablePath || !isWithin(record.PackageRoot, record.Executable) {
		return StatusModified, nil
	}
	return StatusVerified, nil
}

func copyPackage(sourceRoot, destinationRoot, executablePath string) error {
	if err := os.Mkdir(destinationRoot, 0o700); err != nil {
		return fmt.Errorf("create managed package root: %w", err)
	}
	return filepath.WalkDir(sourceRoot, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == sourceRoot {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, current)
		if err != nil {
			return err
		}
		relativeSlash := filepath.ToSlash(relative)
		if !safePackagePath(relativeSlash) {
			return fmt.Errorf("kernel package contains an unsafe path %q", relativeSlash)
		}
		sourceInfo, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if sourceInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("kernel package contains a symbolic link")
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			if !sourceInfo.IsDir() {
				return fmt.Errorf("kernel package contains an invalid directory")
			}
			return os.Mkdir(destination, 0o700)
		}
		if !sourceInfo.Mode().IsRegular() {
			return fmt.Errorf("kernel package contains a special file")
		}
		input, err := os.Open(current)
		if err != nil {
			return err
		}
		openedInfo, err := input.Stat()
		if err != nil {
			_ = input.Close()
			return err
		}
		if !os.SameFile(sourceInfo, openedInfo) {
			_ = input.Close()
			return fmt.Errorf("kernel package file changed while opening")
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		syncErr := output.Sync()
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		if closeOutputErr != nil {
			return closeOutputErr
		}
		if closeInputErr != nil {
			return closeInputErr
		}
		if relativeSlash == executablePath {
			if err := os.Chmod(destination, 0o700); err != nil {
				return err
			}
		}
		return nil
	})
}

func safePackagePath(value string) bool {
	if value == "" || strings.ContainsRune(value, '\\') || strings.ContainsRune(value, 0) || path.IsAbs(value) {
		return false
	}
	clean := path.Clean(value)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && clean == value
}
