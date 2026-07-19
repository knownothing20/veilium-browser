package kernelinstaller

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type fakeStore struct {
	records       []kernel.Record
	request       kernel.PackageImportRequest
	verify        kernel.Record
	release       kernelrelease.Release
	sourcePresent bool
}

func (store *fakeStore) List() []kernel.Record                { return append([]kernel.Record(nil), store.records...) }
func (store *fakeStore) Verify(string) (kernel.Record, error) { return store.verify, nil }
func (store *fakeStore) ImportPackage(request kernel.PackageImportRequest) (kernel.Record, error) {
	store.request = request
	_, statErr := os.Stat(filepath.Join(request.SourceRoot, filepath.FromSlash(request.ExecutablePath)))
	store.sourcePresent = statErr == nil
	release := store.release
	return kernel.Record{
		ID: "official-1", Name: release.Name, Provider: release.ProviderID, Version: release.BrowserVersion,
		Status: kernel.StatusVerified, SnapshotRevision: release.SnapshotRevision,
		ArchiveSHA256: release.ArchiveSHA256, PackageTreeSHA256: release.PackageTreeSHA256,
	}, nil
}

func TestInstallVerifiedPackage(t *testing.T) {
	archive, release := buildFixture(t, false)
	store := &fakeStore{release: release}
	installer := testInstaller(t, store, archive, release)
	record, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true})
	if err != nil {
		t.Fatal(err)
	}
	if record.Provider != release.ProviderID || store.request.ExecutablePath != release.ExecutablePath {
		t.Fatalf("unexpected installed record or request: %#v %#v", record, store.request)
	}
	if !store.sourcePresent {
		t.Fatal("installer did not provide the complete extracted package to the store")
	}
}

func TestInstallRequiresLicenseAcknowledgement(t *testing.T) {
	archive, release := buildFixture(t, false)
	installer := testInstaller(t, &fakeStore{}, archive, release)
	_, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion})
	if err == nil || !strings.Contains(err.Error(), "acknowledgement") {
		t.Fatalf("expected license acknowledgement rejection, got %v", err)
	}
}

func TestInstallRejectsArchiveDigestMismatch(t *testing.T) {
	archive, release := buildFixture(t, false)
	release.ArchiveSHA256 = strings.Repeat("0", 64)
	installer := testInstaller(t, &fakeStore{}, archive, release)
	_, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true})
	if err == nil || !strings.Contains(err.Error(), "archive verification") {
		t.Fatalf("expected archive verification failure, got %v", err)
	}
}

func TestInstallRejectsTraversalEntry(t *testing.T) {
	archive, release := buildFixture(t, true)
	installer := testInstaller(t, &fakeStore{}, archive, release)
	_, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}

func TestHealthyExistingInstallAvoidsDownload(t *testing.T) {
	_, release := buildFixture(t, false)
	existing := kernel.Record{ID: "existing", Name: release.Name, Provider: release.ProviderID, Version: release.BrowserVersion, SnapshotRevision: release.SnapshotRevision, Status: kernel.StatusVerified}
	store := &fakeStore{records: []kernel.Record{existing}, verify: existing}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("healthy existing package should not trigger a download")
		return nil, nil
	})}
	installer, err := newWithDependencies(store, t.TempDir(), client, "windows", "amd64", func(_, _, _, _ string) (kernelrelease.Release, bool) { return release, true })
	if err != nil {
		t.Fatal(err)
	}
	record, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true})
	if err != nil || record.ID != existing.ID {
		t.Fatalf("unexpected existing result: %#v %v", record, err)
	}
}

func testInstaller(t *testing.T, store Store, archive []byte, release kernelrelease.Release) *Installer {
	t.Helper()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header),
			Body: io.NopCloser(bytes.NewReader(archive)), ContentLength: int64(len(archive)), Request: request,
		}, nil
	})}
	installer, err := newWithDependencies(store, t.TempDir(), client, "windows", "amd64", func(provider, version, platform, arch string) (kernelrelease.Release, bool) {
		return release, provider == release.ProviderID && version == release.BrowserVersion && platform == release.Platform && arch == release.Arch
	})
	if err != nil {
		t.Fatal(err)
	}
	return installer
}

func buildFixture(t *testing.T, traversal bool) ([]byte, kernelrelease.Release) {
	t.Helper()
	root := t.TempDir()
	packageRoot := filepath.Join(root, "package")
	if err := os.MkdirAll(filepath.Join(packageRoot, "chrome-win"), 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"chrome-win/chrome.exe": []byte("test-browser"), "chrome-win/resources.pak": []byte("resources"),
	}
	for name, data := range files {
		filename := filepath.Join(packageRoot, filepath.FromSlash(name))
		if err := os.WriteFile(filename, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tree, err := kernel.InspectPackageTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, data := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o600)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if traversal {
		header := &zip.FileHeader{Name: "../escape", Method: zip.Store}
		header.SetMode(0o600)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte("escape"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive := buffer.Bytes()
	archiveDigest := sha256.Sum256(archive)
	executableDigest := sha256.Sum256(files["chrome-win/chrome.exe"])
	fileCount := tree.FileCount
	expandedSize := tree.SizeBytes
	if traversal {
		fileCount++
		expandedSize += int64(len("escape"))
	}
	return archive, kernelrelease.Release{
		ProviderID: "test-reviewed-chromium", ProviderRevision: 1, Name: "Test reviewed Chromium",
		BrowserVersion: "152.0.0.0", SnapshotRevision: 123,
		Platform: "windows", Arch: "amd64", ArchiveName: "chrome-win.zip",
		ArchiveURL: "https://commondatastorage.googleapis.com/chromium-browser-snapshots/Win_x64/123/chrome-win.zip",
		ArchiveSizeBytes: int64(len(archive)), ArchiveSHA256: hex.EncodeToString(archiveDigest[:]),
		PackageFileCount: fileCount, ExpandedSizeBytes: expandedSize, PackageTreeSHA256: tree.SHA256,
		ExecutablePath: "chrome-win/chrome.exe", ExecutableSizeBytes: int64(len(files["chrome-win/chrome.exe"])), ExecutableSHA256: hex.EncodeToString(executableDigest[:]),
	}
}
