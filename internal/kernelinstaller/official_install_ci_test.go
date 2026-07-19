package kernelinstaller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

type ciInstallResult struct {
	KernelID          string `json:"kernelId"`
	Executable        string `json:"executable"`
	ProviderID        string `json:"providerId"`
	ProviderRevision  int    `json:"providerRevision"`
	BrowserVersion    string `json:"browserVersion"`
	SnapshotRevision  int64  `json:"snapshotRevision"`
	ArchiveSHA256     string `json:"archiveSha256"`
	ExecutableSHA256  string `json:"executableSha256"`
	PackageTreeSHA256 string `json:"packageTreeSha256"`
	PackageFileCount  int    `json:"packageFileCount"`
	PackageSizeBytes  int64  `json:"packageSizeBytes"`
}

type ciArchiveTransport struct {
	path string
	url  string
}

func (transport ciArchiveTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || request.URL.String() != transport.url {
		return nil, fmt.Errorf("unexpected reviewed Chromium request")
	}
	file, err := os.Open(transport.path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: file, ContentLength: info.Size(), Request: request}, nil
}

func TestReviewedChromiumInstallForCI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("reviewed Chromium package is Windows amd64 only")
	}
	archive := strings.TrimSpace(os.Getenv("VEILIUM_REVIEWED_CHROMIUM_ARCHIVE"))
	workDir := strings.TrimSpace(os.Getenv("VEILIUM_REVIEWED_CHROMIUM_WORKDIR"))
	resultPath := strings.TrimSpace(os.Getenv("VEILIUM_REVIEWED_CHROMIUM_RESULT"))
	if archive == "" || workDir == "" || resultPath == "" {
		t.Skip("reviewed Chromium CI paths are not configured")
	}
	releases, err := kernelrelease.Releases()
	if err != nil || len(releases) != 1 {
		t.Fatalf("load reviewed release: %#v %v", releases, err)
	}
	release := releases[0]
	_ = os.RemoveAll(workDir)
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := kernel.Open(filepath.Join(workDir, "kernels.json"), filepath.Join(workDir, "kernels"))
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: ciArchiveTransport{path: archive, url: release.ArchiveURL}}
	installer, err := newWithDependencies(store, filepath.Join(workDir, "installer"), client, "windows", "amd64", kernelrelease.Find)
	if err != nil {
		t.Fatal(err)
	}
	record, err := installer.Install(context.Background(), Request{ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true})
	if err != nil {
		t.Fatal(err)
	}
	record, err = store.Verify(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := kernel.BinaryIdentity(record)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != kernel.StatusVerified || !identity.Reviewed || identity.PackageTreeSHA256 != release.PackageTreeSHA256 {
		t.Fatalf("unexpected reviewed identity: record=%#v identity=%#v", record, identity)
	}
	result := ciInstallResult{KernelID: record.ID, Executable: record.Executable, ProviderID: identity.ProviderID, ProviderRevision: identity.ProviderRevision, BrowserVersion: identity.BrowserVersion, SnapshotRevision: identity.SnapshotRevision, ArchiveSHA256: identity.ArchiveSHA256, ExecutableSHA256: identity.ExecutableSHA256, PackageTreeSHA256: identity.PackageTreeSHA256, PackageFileCount: identity.PackageFileCount, PackageSizeBytes: identity.PackageSizeBytes}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
