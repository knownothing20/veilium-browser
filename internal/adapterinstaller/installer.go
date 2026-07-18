package adapterinstaller

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
)

const (
	maxRedirects      = 5
	downloadTimeout   = 5 * time.Minute
	maxErrorBodyBytes = 4 << 10
	copyBufferSize    = 1 << 20
)

type Request struct {
	Kind            string `json:"kind"`
	Version         string `json:"version"`
	LicenseAccepted bool   `json:"licenseAccepted"`
}

type Store interface {
	List() []adapter.Record
	Verify(string) (adapter.Record, error)
	Import(adapter.ImportRequest) (adapter.Record, error)
}

type pinResolver func(kind, version, platform, arch string) (adapterrelease.Pin, bool)

type Installer struct {
	store    Store
	tempRoot string
	client   *http.Client
	platform string
	arch     string
	findPin  pinResolver
}

func New(store Store, tempRoot string) (*Installer, error) {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 20 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
	client := &http.Client{Transport: transport, Timeout: downloadTimeout}
	client.CheckRedirect = secureRedirectPolicy
	return newWithDependencies(store, tempRoot, client, runtime.GOOS, runtime.GOARCH, adapterrelease.Find)
}

func newWithDependencies(store Store, tempRoot string, client *http.Client, platform, arch string, findPin pinResolver) (*Installer, error) {
	if store == nil || client == nil || findPin == nil {
		return nil, fmt.Errorf("official adapter installer dependencies are required")
	}
	tempRoot = strings.TrimSpace(tempRoot)
	if tempRoot == "" {
		return nil, fmt.Errorf("official adapter installer temporary root is required")
	}
	if strings.TrimSpace(platform) == "" || strings.TrimSpace(arch) == "" {
		return nil, fmt.Errorf("official adapter installer platform is required")
	}
	return &Installer{store: store, tempRoot: tempRoot, client: client, platform: platform, arch: arch, findPin: findPin}, nil
}

func (i *Installer) Install(ctx context.Context, request Request) (adapter.Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.Kind = adapter.NormalizeKind(request.Kind)
	request.Version = strings.TrimPrefix(strings.TrimSpace(request.Version), "v")
	if !request.LicenseAccepted {
		return adapter.Record{}, fmt.Errorf("official adapter license acknowledgement is required")
	}
	pin, ok := i.findPin(request.Kind, request.Version, i.platform, i.arch)
	if !ok {
		return adapter.Record{}, fmt.Errorf("no pinned official %s %s build is available for %s/%s", request.Kind, request.Version, i.platform, i.arch)
	}
	if existing, found, err := i.existing(pin); err != nil {
		return adapter.Record{}, err
	} else if found {
		return existing, nil
	}
	if err := ensurePrivateDirectory(i.tempRoot); err != nil {
		return adapter.Record{}, err
	}
	workDir, err := os.MkdirTemp(i.tempRoot, ".official-adapter-*")
	if err != nil {
		return adapter.Record{}, fmt.Errorf("create official adapter installer directory: %w", err)
	}
	defer os.RemoveAll(workDir)
	if err := os.Chmod(workDir, 0o700); err != nil {
		return adapter.Record{}, fmt.Errorf("protect official adapter installer directory: %w", err)
	}
	archivePath := filepath.Join(workDir, "release-asset")
	if err := i.download(ctx, pin, archivePath); err != nil {
		return adapter.Record{}, err
	}
	executablePath := filepath.Join(workDir, safeExecutableName(pin))
	if err := extractPinned(ctx, pin, archivePath, executablePath); err != nil {
		return adapter.Record{}, err
	}
	if err := verifyFile(executablePath, pin.ExecutableSizeBytes, pin.ExecutableSHA256); err != nil {
		return adapter.Record{}, fmt.Errorf("verify extracted official adapter: %w", err)
	}
	if err := os.Chmod(executablePath, 0o700); err != nil {
		return adapter.Record{}, fmt.Errorf("protect extracted official adapter: %w", err)
	}
	record, err := i.store.Import(adapter.ImportRequest{
		Name:        officialName(pin),
		Kind:        pin.Kind,
		Version:     pin.Version,
		SourcePath:  executablePath,
		LicenseSPDX: pin.LicenseSPDX,
		SourceURL:   pin.AssetURL,
	})
	if err != nil {
		return adapter.Record{}, fmt.Errorf("import downloaded official adapter: %w", err)
	}
	if !record.Official || record.OfficialTag != pin.Tag || record.OfficialPlatform != pin.Platform || record.OfficialArch != pin.Arch {
		return adapter.Record{}, fmt.Errorf("installed adapter did not match the embedded official identity")
	}
	return record, nil
}

func (i *Installer) existing(pin adapterrelease.Pin) (adapter.Record, bool, error) {
	for _, record := range i.store.List() {
		if !record.Official || record.Kind != pin.Kind || record.Version != pin.Version || record.OfficialPlatform != pin.Platform || record.OfficialArch != pin.Arch {
			continue
		}
		verified, err := i.store.Verify(record.ID)
		if err != nil {
			return adapter.Record{}, false, fmt.Errorf("verify existing official adapter: %w", err)
		}
		if verified.Status != adapter.StatusVerified {
			return adapter.Record{}, false, fmt.Errorf("existing official adapter %q is %s; remove or repair it before reinstalling", verified.Name, verified.Status)
		}
		return verified, true, nil
	}
	return adapter.Record{}, false, nil
}

func (i *Installer) download(ctx context.Context, pin adapterrelease.Pin, destination string) error {
	parsed, err := url.Parse(pin.AssetURL)
	if err != nil || !allowedReleaseURL(parsed) {
		return fmt.Errorf("embedded official adapter URL is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pin.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("create official adapter download request: %w", err)
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "Veilium-Official-Adapter-Installer/1")
	response, err := i.client.Do(request)
	if err != nil {
		return fmt.Errorf("download official adapter: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxErrorBodyBytes))
		return fmt.Errorf("download official adapter: unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > pin.ArchiveSizeBytes && response.ContentLength >= 0 {
		return fmt.Errorf("download official adapter: response exceeds pinned archive size")
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create official adapter archive: %w", err)
	}
	hasher := sha256.New()
	written, copyErr := copyContext(ctx, io.MultiWriter(file, hasher), io.LimitReader(response.Body, pin.ArchiveSizeBytes+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("download official adapter archive: %w", copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync official adapter archive: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close official adapter archive: %w", closeErr)
	}
	if written != pin.ArchiveSizeBytes || hex.EncodeToString(hasher.Sum(nil)) != pin.ArchiveSHA256 {
		return fmt.Errorf("downloaded official adapter failed pinned archive verification")
	}
	return nil
}

func secureRedirectPolicy(request *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("official adapter download exceeded redirect limit")
	}
	if !allowedDownloadURL(request.URL) {
		return fmt.Errorf("official adapter download redirected outside approved GitHub hosts")
	}
	request.Header.Del("Authorization")
	request.Header.Del("Cookie")
	return nil
}

func allowedReleaseURL(value *url.URL) bool {
	return value != nil && value.Scheme == "https" && value.Hostname() == "github.com" && value.User == nil && value.Fragment == "" && strings.Contains(value.EscapedPath(), "/releases/download/")
}

func allowedDownloadURL(value *url.URL) bool {
	if value == nil || value.Scheme != "https" || value.User != nil || value.Fragment != "" {
		return false
	}
	host := strings.ToLower(value.Hostname())
	return host == "github.com" || host == "release-assets.githubusercontent.com" || host == "objects.githubusercontent.com"
}

func extractPinned(ctx context.Context, pin adapterrelease.Pin, archivePath, destination string) error {
	switch {
	case strings.HasSuffix(strings.ToLower(pin.AssetName), ".zip"):
		return extractZip(ctx, pin, archivePath, destination)
	case strings.HasSuffix(strings.ToLower(pin.AssetName), ".tar.gz"):
		return extractTarGzip(ctx, pin, archivePath, destination)
	default:
		return fmt.Errorf("unsupported official adapter archive format")
	}
}

func extractZip(ctx context.Context, pin adapterrelease.Pin, archivePath, destination string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open official adapter ZIP archive: %w", err)
	}
	defer archive.Close()
	var match *zip.File
	for _, entry := range archive.File {
		if err := validateArchivePath(entry.Name); err != nil {
			return err
		}
		mode := entry.Mode()
		if mode&os.ModeSymlink != 0 || mode&(os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
			return fmt.Errorf("official adapter ZIP contains an unsupported linked or special entry")
		}
		if entry.Name == pin.ExecutablePath {
			if match != nil {
				return fmt.Errorf("official adapter ZIP contains duplicate executable entries")
			}
			if entry.FileInfo().IsDir() || !mode.IsRegular() {
				return fmt.Errorf("official adapter executable is not a regular ZIP entry")
			}
			match = entry
		}
	}
	if match == nil {
		return fmt.Errorf("official adapter executable is missing from ZIP archive")
	}
	if int64(match.UncompressedSize64) != pin.ExecutableSizeBytes {
		return fmt.Errorf("official adapter ZIP executable size does not match the pin")
	}
	reader, err := match.Open()
	if err != nil {
		return fmt.Errorf("open official adapter ZIP executable: %w", err)
	}
	defer reader.Close()
	return writeExtracted(ctx, destination, reader, pin.ExecutableSizeBytes)
}

func extractTarGzip(ctx context.Context, pin adapterrelease.Pin, archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open official adapter tar archive: %w", err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open official adapter gzip stream: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	found := false
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read official adapter tar archive: %w", err)
		}
		if err := validateArchivePath(header.Name); err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir, tar.TypeReg, tar.TypeRegA:
		default:
			return fmt.Errorf("official adapter tar archive contains a linked or special entry")
		}
		if header.Name != pin.ExecutablePath {
			continue
		}
		if found {
			return fmt.Errorf("official adapter tar archive contains duplicate executable entries")
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("official adapter executable is not a regular tar entry")
		}
		if header.Size != pin.ExecutableSizeBytes {
			return fmt.Errorf("official adapter tar executable size does not match the pin")
		}
		if err := writeExtracted(ctx, destination, tarReader, pin.ExecutableSizeBytes); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return fmt.Errorf("official adapter executable is missing from tar archive")
	}
	return nil
}

func writeExtracted(ctx context.Context, destination string, source io.Reader, expected int64) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return fmt.Errorf("create extracted official adapter: %w", err)
	}
	written, copyErr := copyContext(ctx, file, io.LimitReader(source, expected+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("extract official adapter executable: %w", copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync extracted official adapter: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close extracted official adapter: %w", closeErr)
	}
	if written != expected {
		return fmt.Errorf("extracted official adapter size does not match the pin")
	}
	return nil
}

func validateArchivePath(value string) error {
	if value == "" || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") {
		return fmt.Errorf("official adapter archive contains an unsafe path")
	}
	trimmed := strings.TrimSuffix(value, "/")
	if trimmed == "" {
		return fmt.Errorf("official adapter archive contains an unsafe path")
	}
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return fmt.Errorf("official adapter archive contains an unsafe path")
	}
	return nil
}

func verifyFile(path string, expectedSize int64, expectedDigest string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != expectedSize {
		return fmt.Errorf("file size or type does not match the pin")
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if hex.EncodeToString(hasher.Sum(nil)) != expectedDigest {
		return fmt.Errorf("file SHA-256 does not match the pin")
	}
	return nil
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, copyBufferSize)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create official adapter installer root: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect official adapter installer root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("official adapter installer root must be a real directory")
	}
	return os.Chmod(path, 0o700)
}

func safeExecutableName(pin adapterrelease.Pin) string {
	if pin.Platform == "windows" {
		return strings.ReplaceAll(pin.Kind, "-", "_") + ".exe"
	}
	return strings.ReplaceAll(pin.Kind, "-", "_")
}

func officialName(pin adapterrelease.Pin) string {
	if pin.Kind == adapter.KindXray {
		return "Xray " + pin.Tag + " (official)"
	}
	return "sing-box " + pin.Tag + " (official)"
}
