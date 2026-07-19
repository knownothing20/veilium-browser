package kernelinstaller

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

const (
	downloadTimeout   = 10 * time.Minute
	maxRedirects      = 3
	maxErrorBodyBytes = 4 << 10
)

type Request struct {
	ProviderID      string `json:"providerId"`
	Version         string `json:"version"`
	LicenseAccepted bool   `json:"licenseAccepted"`
}

type Store interface {
	List() []kernel.Record
	Verify(string) (kernel.Record, error)
	ImportPackage(kernel.PackageImportRequest) (kernel.Record, error)
}

type releaseResolver func(providerID, version, platform, arch string) (kernelrelease.Release, bool)

type Installer struct {
	store       Store
	tempRoot    string
	client      *http.Client
	platform    string
	arch        string
	findRelease releaseResolver
}

func New(store Store, tempRoot string) (*Installer, error) {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: time.Second,
		DisableCompression:    true,
	}
	client := &http.Client{Transport: transport, Timeout: downloadTimeout}
	client.CheckRedirect = secureRedirectPolicy
	return newWithDependencies(store, tempRoot, client, runtime.GOOS, runtime.GOARCH, kernelrelease.Find)
}

func newWithDependencies(store Store, tempRoot string, client *http.Client, platform, arch string, findRelease releaseResolver) (*Installer, error) {
	if store == nil || client == nil || findRelease == nil {
		return nil, fmt.Errorf("official Chromium installer dependencies are required")
	}
	tempRoot = strings.TrimSpace(tempRoot)
	if tempRoot == "" {
		return nil, fmt.Errorf("official Chromium installer temporary root is required")
	}
	if strings.TrimSpace(platform) == "" || strings.TrimSpace(arch) == "" {
		return nil, fmt.Errorf("official Chromium installer platform is required")
	}
	return &Installer{store: store, tempRoot: tempRoot, client: client, platform: platform, arch: arch, findRelease: findRelease}, nil
}

func (installer *Installer) Install(ctx context.Context, request Request) (kernel.Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.ProviderID = strings.TrimSpace(request.ProviderID)
	request.Version = strings.TrimSpace(request.Version)
	if !request.LicenseAccepted {
		return kernel.Record{}, fmt.Errorf("Chromium license and third-party notice acknowledgement is required")
	}
	release, ok := installer.findRelease(request.ProviderID, request.Version, installer.platform, installer.arch)
	if !ok {
		return kernel.Record{}, fmt.Errorf("no pinned reviewed Chromium package is available for %s/%s", installer.platform, installer.arch)
	}
	if existing, found, err := installer.existing(release); err != nil {
		return kernel.Record{}, err
	} else if found {
		return existing, nil
	}
	if err := ensurePrivateDirectory(installer.tempRoot); err != nil {
		return kernel.Record{}, err
	}
	workDir, err := os.MkdirTemp(installer.tempRoot, ".official-chromium-*")
	if err != nil {
		return kernel.Record{}, fmt.Errorf("create official Chromium installer directory: %w", err)
	}
	defer os.RemoveAll(workDir)
	if err := os.Chmod(workDir, 0o700); err != nil {
		return kernel.Record{}, fmt.Errorf("protect official Chromium installer directory: %w", err)
	}
	archivePath := filepath.Join(workDir, release.ArchiveName)
	if err := installer.download(ctx, release, archivePath); err != nil {
		return kernel.Record{}, err
	}
	extractedRoot := filepath.Join(workDir, "extracted")
	if err := extractPinnedPackage(ctx, release, archivePath, extractedRoot); err != nil {
		return kernel.Record{}, err
	}
	record, err := installer.store.ImportPackage(kernel.PackageImportRequest{
		Name: release.Name, Provider: release.ProviderID, Version: release.BrowserVersion,
		SourceRoot: extractedRoot, ExecutablePath: release.ExecutablePath,
		SnapshotRevision: release.SnapshotRevision, ArchiveSHA256: release.ArchiveSHA256,
	})
	if err != nil {
		return kernel.Record{}, fmt.Errorf("import reviewed Chromium package: %w", err)
	}
	if record.Provider != release.ProviderID || record.Version != release.BrowserVersion || record.SnapshotRevision != release.SnapshotRevision || record.ArchiveSHA256 != release.ArchiveSHA256 || record.PackageTreeSHA256 != release.PackageTreeSHA256 {
		return kernel.Record{}, fmt.Errorf("installed Chromium package did not match the embedded reviewed identity")
	}
	return record, nil
}

func (installer *Installer) existing(release kernelrelease.Release) (kernel.Record, bool, error) {
	for _, record := range installer.store.List() {
		if record.Provider != release.ProviderID || record.Version != release.BrowserVersion || record.SnapshotRevision != release.SnapshotRevision {
			continue
		}
		verified, err := installer.store.Verify(record.ID)
		if err != nil {
			return kernel.Record{}, false, fmt.Errorf("verify existing reviewed Chromium package: %w", err)
		}
		if verified.Status != kernel.StatusVerified {
			return kernel.Record{}, false, fmt.Errorf("existing reviewed Chromium package %q is %s; remove or repair it before reinstalling", verified.Name, verified.Status)
		}
		return verified, true, nil
	}
	return kernel.Record{}, false, nil
}

func (installer *Installer) download(ctx context.Context, release kernelrelease.Release, destination string) error {
	parsed, err := url.Parse(release.ArchiveURL)
	if err != nil || !allowedPinnedURL(parsed, release) {
		return fmt.Errorf("embedded official Chromium URL is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.ArchiveURL, nil)
	if err != nil {
		return fmt.Errorf("create official Chromium download request: %w", err)
	}
	request.Header.Set("Accept", "application/zip")
	request.Header.Set("User-Agent", "Veilium-Official-Chromium-Installer/1")
	response, err := installer.client.Do(request)
	if err != nil {
		return fmt.Errorf("download official Chromium: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxErrorBodyBytes))
		return fmt.Errorf("download official Chromium: unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength >= 0 && response.ContentLength != release.ArchiveSizeBytes {
		return fmt.Errorf("download official Chromium: response size does not match the pin")
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create official Chromium archive: %w", err)
	}
	hasher := sha256.New()
	written, copyErr := copyContext(ctx, io.MultiWriter(file, hasher), io.LimitReader(response.Body, release.ArchiveSizeBytes+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("download official Chromium archive: %w", copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync official Chromium archive: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close official Chromium archive: %w", closeErr)
	}
	if written != release.ArchiveSizeBytes || hex.EncodeToString(hasher.Sum(nil)) != release.ArchiveSHA256 {
		return fmt.Errorf("downloaded official Chromium failed pinned archive verification")
	}
	return nil
}

func extractPinnedPackage(ctx context.Context, release kernelrelease.Release, archivePath, destinationRoot string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open official Chromium ZIP archive: %w", err)
	}
	defer archive.Close()
	seen := make(map[string]struct{}, release.PackageFileCount)
	var files int
	var expanded int64
	for _, entry := range archive.File {
		name, isDirectory, err := validateArchiveEntry(entry)
		if err != nil {
			return err
		}
		if isDirectory {
			continue
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("official Chromium ZIP contains duplicate path %q", name)
		}
		seen[name] = struct{}{}
		files++
		expanded += int64(entry.UncompressedSize64)
		if files > release.PackageFileCount || expanded > release.ExpandedSizeBytes {
			return fmt.Errorf("official Chromium ZIP exceeds the pinned package bounds")
		}
	}
	if files != release.PackageFileCount || expanded != release.ExpandedSizeBytes {
		return fmt.Errorf("official Chromium ZIP package size or file count does not match the pin")
	}
	if err := os.Mkdir(destinationRoot, 0o700); err != nil {
		return fmt.Errorf("create official Chromium extraction root: %w", err)
	}
	for _, entry := range archive.File {
		name, isDirectory, err := validateArchiveEntry(entry)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, filepath.FromSlash(name))
		if !within(destinationRoot, destination) {
			return fmt.Errorf("official Chromium ZIP entry escaped the extraction root")
		}
		if isDirectory {
			if err := os.MkdirAll(destination, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		reader, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open official Chromium ZIP entry: %w", err)
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = reader.Close()
			return err
		}
		written, copyErr := copyContext(ctx, output, io.LimitReader(reader, int64(entry.UncompressedSize64)+1))
		syncErr := output.Sync()
		closeOutputErr := output.Close()
		closeReaderErr := reader.Close()
		if copyErr != nil {
			return copyErr
		}
		if written != int64(entry.UncompressedSize64) {
			return fmt.Errorf("official Chromium ZIP entry size changed during extraction")
		}
		if syncErr != nil {
			return syncErr
		}
		if closeOutputErr != nil {
			return closeOutputErr
		}
		if closeReaderErr != nil {
			return closeReaderErr
		}
	}
	tree, err := kernel.InspectPackageTree(destinationRoot)
	if err != nil {
		return fmt.Errorf("inspect extracted official Chromium package: %w", err)
	}
	if tree.SHA256 != release.PackageTreeSHA256 || tree.FileCount != release.PackageFileCount || tree.SizeBytes != release.ExpandedSizeBytes {
		return fmt.Errorf("extracted official Chromium package tree does not match the pin")
	}
	executable := filepath.Join(destinationRoot, filepath.FromSlash(release.ExecutablePath))
	if err := verifyFile(executable, release.ExecutableSizeBytes, release.ExecutableSHA256); err != nil {
		return fmt.Errorf("verify extracted official Chromium executable: %w", err)
	}
	if err := os.Chmod(executable, 0o700); err != nil {
		return fmt.Errorf("protect official Chromium executable: %w", err)
	}
	return nil
}

func validateArchiveEntry(entry *zip.File) (string, bool, error) {
	if entry == nil || strings.ContainsRune(entry.Name, 0) || strings.ContainsRune(entry.Name, '\\') || strings.HasPrefix(entry.Name, "/") {
		return "", false, fmt.Errorf("official Chromium ZIP contains an unsafe path")
	}
	isDirectory := entry.FileInfo().IsDir() || strings.HasSuffix(entry.Name, "/")
	name := strings.TrimSuffix(entry.Name, "/")
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != name || (clean != "chrome-win" && !strings.HasPrefix(clean, "chrome-win/")) {
		return "", false, fmt.Errorf("official Chromium ZIP contains an unsafe or unexpected path %q", entry.Name)
	}
	mode := entry.Mode()
	if mode&os.ModeSymlink != 0 || mode&(os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
		return "", false, fmt.Errorf("official Chromium ZIP contains a linked or special entry")
	}
	if !isDirectory && !mode.IsRegular() {
		return "", false, fmt.Errorf("official Chromium ZIP contains a non-regular file")
	}
	return clean, isDirectory, nil
}

func verifyFile(filename string, expectedSize int64, expectedDigest string) error {
	info, err := os.Lstat(filename)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != expectedSize {
		return fmt.Errorf("file identity is invalid")
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if hex.EncodeToString(hasher.Sum(nil)) != expectedDigest {
		return fmt.Errorf("file digest does not match the pin")
	}
	return nil
}

func allowedPinnedURL(value *url.URL, release kernelrelease.Release) bool {
	if value == nil || value.String() != release.ArchiveURL || value.Scheme != "https" || value.Hostname() != "commondatastorage.googleapis.com" || value.User != nil || value.RawQuery != "" || value.Fragment != "" {
		return false
	}
	expected := fmt.Sprintf("/chromium-browser-snapshots/Win_x64/%d/chrome-win.zip", release.SnapshotRevision)
	return value.EscapedPath() == expected
}

func secureRedirectPolicy(request *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("official Chromium download exceeded redirect limit")
	}
	if request.URL == nil || request.URL.Scheme != "https" || request.URL.User != nil || request.URL.Fragment != "" {
		return fmt.Errorf("official Chromium download redirected to an unsafe URL")
	}
	host := strings.ToLower(request.URL.Hostname())
	if host != "commondatastorage.googleapis.com" && host != "storage.googleapis.com" {
		return fmt.Errorf("official Chromium download redirected outside approved Google storage hosts")
	}
	request.Header.Del("Authorization")
	request.Header.Del("Cookie")
	return nil
}

func ensurePrivateDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create official Chromium temporary root: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect official Chromium temporary root: %w", err)
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("official Chromium temporary root must be a real directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("protect official Chromium temporary root: %w", err)
	}
	return nil
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 1<<20)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			outputCount, writeErr := destination.Write(buffer[:count])
			written += int64(outputCount)
			if writeErr != nil {
				return written, writeErr
			}
			if outputCount != count {
				return written, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}

func within(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbs, candidateAbs)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
