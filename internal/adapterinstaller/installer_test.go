package adapterinstaller

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
)

type fakeStore struct {
	mu      sync.Mutex
	records []adapter.Record
	imports []adapter.ImportRequest
}

func (s *fakeStore) List() []adapter.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]adapter.Record(nil), s.records...)
}

func (s *fakeStore) Verify(id string) (adapter.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range s.records {
		if record.ID == id {
			return record, nil
		}
	}
	return adapter.Record{}, adapter.ErrNotFound
}

func (s *fakeStore) Import(request adapter.ImportRequest) (adapter.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imports = append(s.imports, request)
	data, err := os.ReadFile(request.SourcePath)
	if err != nil {
		return adapter.Record{}, err
	}
	digest := sha256.Sum256(data)
	record := adapter.Record{
		ID: "official-1", Name: request.Name, Kind: request.Kind, Version: request.Version,
		Executable: request.SourcePath, SHA256: hex.EncodeToString(digest[:]), SizeBytes: int64(len(data)),
		LicenseSPDX: request.LicenseSPDX, SourceURL: request.SourceURL, Status: adapter.StatusVerified,
		Official: true, OfficialTag: "v" + request.Version, OfficialPlatform: "linux", OfficialArch: "amd64",
	}
	s.records = append(s.records, record)
	return record, nil
}

func TestInstallPinnedZip(t *testing.T) {
	executable := []byte("official-xray-binary")
	archive := zipArchive(t, "xray", executable, false)
	pin := testPin("xray", "26.3.27", "official.zip", "xray", archive, executable)
	server := newPinnedServer(t, pin.AssetURL, archive)
	defer server.Close()
	pin.AssetURL = "https://github.com/XTLS/Xray-core/releases/download/v26.3.27/official.zip"

	store := &fakeStore{}
	installer := newTestInstaller(t, store, server, pin)
	record, err := installer.Install(context.Background(), Request{Kind: "xray", Version: "26.3.27", LicenseAccepted: true})
	if err != nil {
		t.Fatal(err)
	}
	if !record.Official || len(store.imports) != 1 {
		t.Fatalf("unexpected installation result: %#v imports=%d", record, len(store.imports))
	}
	request := store.imports[0]
	if request.SourceURL != pin.AssetURL || request.LicenseSPDX != pin.LicenseSPDX || request.Name != "Xray v26.3.27 (official)" {
		t.Fatalf("canonical import metadata was not used: %#v", request)
	}
}

func TestInstallPinnedTarGzip(t *testing.T) {
	executable := []byte("official-sing-box-binary")
	member := "sing-box-1.13.12-linux-amd64/sing-box"
	archive := tarGzipArchive(t, member, executable, false)
	pin := testPin("sing-box", "1.13.12", "official.tar.gz", member, archive, executable)
	server := newPinnedServer(t, pin.AssetURL, archive)
	defer server.Close()
	pin.AssetURL = "https://github.com/SagerNet/sing-box/releases/download/v1.13.12/official.tar.gz"

	store := &fakeStore{}
	installer := newTestInstaller(t, store, server, pin)
	if _, err := installer.Install(context.Background(), Request{Kind: "sing-box", Version: "1.13.12", LicenseAccepted: true}); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRequiresLicenseAcknowledgement(t *testing.T) {
	store := &fakeStore{}
	installer, err := newWithDependencies(store, t.TempDir(), &http.Client{}, "linux", "amd64", func(string, string, string, string) (adapterrelease.Pin, bool) {
		return adapterrelease.Pin{}, false
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), Request{Kind: "xray", Version: "26.3.27"}); err == nil || !strings.Contains(err.Error(), "acknowledgement") {
		t.Fatalf("expected license acknowledgement failure, got %v", err)
	}
}

func TestInstallRejectsArchiveDigestMismatch(t *testing.T) {
	executable := []byte("binary")
	archive := zipArchive(t, "xray", executable, false)
	pin := testPin("xray", "26.3.27", "official.zip", "xray", archive, executable)
	pin.ArchiveSHA256 = strings.Repeat("0", 64)
	server := newPinnedServer(t, pin.AssetURL, archive)
	defer server.Close()
	pin.AssetURL = "https://github.com/XTLS/Xray-core/releases/download/v26.3.27/official.zip"
	installer := newTestInstaller(t, &fakeStore{}, server, pin)
	if _, err := installer.Install(context.Background(), Request{Kind: "xray", Version: pin.Version, LicenseAccepted: true}); err == nil || !strings.Contains(err.Error(), "archive verification") {
		t.Fatalf("expected archive verification failure, got %v", err)
	}
}

func TestInstallRejectsUnsafeArchiveEntry(t *testing.T) {
	executable := []byte("binary")
	archive := zipArchive(t, "xray", executable, true)
	pin := testPin("xray", "26.3.27", "official.zip", "xray", archive, executable)
	server := newPinnedServer(t, pin.AssetURL, archive)
	defer server.Close()
	pin.AssetURL = "https://github.com/XTLS/Xray-core/releases/download/v26.3.27/official.zip"
	installer := newTestInstaller(t, &fakeStore{}, server, pin)
	if _, err := installer.Install(context.Background(), Request{Kind: "xray", Version: pin.Version, LicenseAccepted: true}); err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("expected unsafe archive failure, got %v", err)
	}
}

func TestInstallReturnsHealthyExistingRecordWithoutDownload(t *testing.T) {
	pin := adapterrelease.Pin{Kind: "xray", Version: "26.3.27", Tag: "v26.3.27", Platform: "linux", Arch: "amd64"}
	store := &fakeStore{records: []adapter.Record{{
		ID: "existing", Name: "Xray", Kind: "xray", Version: "26.3.27", Status: adapter.StatusVerified,
		Official: true, OfficialTag: pin.Tag, OfficialPlatform: pin.Platform, OfficialArch: pin.Arch,
	}}}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network must not be used")
	})}
	installer, err := newWithDependencies(store, t.TempDir(), client, "linux", "amd64", func(string, string, string, string) (adapterrelease.Pin, bool) { return pin, true })
	if err != nil {
		t.Fatal(err)
	}
	record, err := installer.Install(context.Background(), Request{Kind: "xray", Version: pin.Version, LicenseAccepted: true})
	if err != nil || record.ID != "existing" {
		t.Fatalf("unexpected existing result %#v %v", record, err)
	}
}

func TestInstallHonorsCancellation(t *testing.T) {
	pin := testPin("xray", "26.3.27", "official.zip", "xray", []byte("placeholder"), []byte("binary"))
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})}
	installer, err := newWithDependencies(&fakeStore{}, t.TempDir(), client, "linux", "amd64", func(string, string, string, string) (adapterrelease.Pin, bool) { return pin, true })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := installer.Install(ctx, Request{Kind: "xray", Version: pin.Version, LicenseAccepted: true}); err == nil || !strings.Contains(err.Error(), "context deadline") {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestSecureRedirectPolicy(t *testing.T) {
	allowed, _ := http.NewRequest(http.MethodGet, "https://release-assets.githubusercontent.com/github-production-release-asset/1", nil)
	if err := secureRedirectPolicy(allowed, []*http.Request{{}}); err != nil {
		t.Fatalf("expected GitHub CDN redirect to pass: %v", err)
	}
	blocked, _ := http.NewRequest(http.MethodGet, "https://example.com/asset.zip", nil)
	if err := secureRedirectPolicy(blocked, []*http.Request{{}}); err == nil {
		t.Fatal("expected foreign redirect rejection")
	}
}

func TestValidateArchivePathAllowsDirectoryEntries(t *testing.T) {
	for _, value := range []string{"folder/", "folder/subfolder/", "folder/file"} {
		if err := validateArchivePath(value); err != nil {
			t.Fatalf("expected %q to be accepted: %v", value, err)
		}
	}
	for _, value := range []string{"../escape", "folder/../escape", "/absolute", `folder\file`, "./file"} {
		if err := validateArchivePath(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func newTestInstaller(t *testing.T, store Store, server *httptest.Server, pin adapterrelease.Pin) *Installer {
	t.Helper()
	serverURL, _ := url.Parse(server.URL)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // test server only
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", serverURL.Host)
		},
	}
	client := &http.Client{Transport: transport, Timeout: time.Second}
	installer, err := newWithDependencies(store, t.TempDir(), client, pin.Platform, pin.Arch, func(kind, version, platform, arch string) (adapterrelease.Pin, bool) {
		return pin, kind == pin.Kind && version == pin.Version && platform == pin.Platform && arch == pin.Arch
	})
	if err != nil {
		t.Fatal(err)
	}
	return installer
}

func newPinnedServer(t *testing.T, _ string, payload []byte) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = response.Write(payload)
	}))
}

func testPin(kind, version, assetName, executablePath string, archive, executable []byte) adapterrelease.Pin {
	archiveDigest := sha256.Sum256(archive)
	executableDigest := sha256.Sum256(executable)
	license := "MPL-2.0"
	repository := "XTLS/Xray-core"
	if kind == "sing-box" {
		license = "GPL-3.0-or-later"
		repository = "SagerNet/sing-box"
	}
	return adapterrelease.Pin{
		Kind: kind, Version: version, Tag: "v" + version, Repository: repository, LicenseSPDX: license,
		Platform: "linux", Arch: "amd64", AssetName: assetName,
		AssetURL:      "https://github.com/" + repository + "/releases/download/v" + version + "/" + assetName,
		ArchiveSHA256: hex.EncodeToString(archiveDigest[:]), ArchiveSizeBytes: int64(len(archive)),
		ExecutablePath: executablePath, ExecutableSHA256: hex.EncodeToString(executableDigest[:]), ExecutableSizeBytes: int64(len(executable)),
	}
}

func zipArchive(t *testing.T, path string, executable []byte, unsafe bool) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(executable); err != nil {
		t.Fatal(err)
	}
	if unsafe {
		entry, err = writer.Create("../escape")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte("escape"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func tarGzipArchive(t *testing.T, path string, executable []byte, linked bool) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: path, Mode: 0o700, Size: int64(len(executable)), Typeflag: tar.TypeReg}
	if linked {
		header.Typeflag = tar.TypeSymlink
		header.Linkname = "../../escape"
		header.Size = 0
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if !linked {
		if _, err := tarWriter.Write(executable); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
