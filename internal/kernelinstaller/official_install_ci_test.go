package kernelinstaller

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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

type ciArchiveFileIdentity struct {
	path   string
	size   int64
	digest string
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
	archiveTree, err := inspectCIArchivePackageTree(archive)
	if err != nil {
		t.Fatalf("inspect reviewed Chromium archive package tree: %v", err)
	}
	expectedTree := kernel.PackageTreeIdentity{
		SHA256:    release.PackageTreeSHA256,
		FileCount: release.PackageFileCount,
		SizeBytes: release.ExpandedSizeBytes,
	}
	diagnostic, err := json.MarshalIndent(map[string]kernel.PackageTreeIdentity{
		"actualArchiveTree": archiveTree,
		"expectedPin":       expectedTree,
	}, "", "  ")
	if err != nil {
		t.Fatalf("encode reviewed Chromium tree diagnostic: %v", err)
	}
	if err := os.WriteFile(resultPath, diagnostic, 0o600); err != nil {
		t.Fatalf("write reviewed Chromium tree diagnostic: %v", err)
	}
	if archiveTree != expectedTree {
		t.Fatalf("reviewed Chromium archive package tree mismatch: actual=%#v expected=%#v", archiveTree, expectedTree)
	}
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

func inspectCIArchivePackageTree(archivePath string) (kernel.PackageTreeIdentity, error) {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return kernel.PackageTreeIdentity{}, err
	}
	defer archive.Close()

	files := make([]ciArchiveFileIdentity, 0, len(archive.File))
	var total int64
	for _, entry := range archive.File {
		name, isDirectory, err := validateArchiveEntry(entry)
		if err != nil {
			return kernel.PackageTreeIdentity{}, err
		}
		if isDirectory {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return kernel.PackageTreeIdentity{}, err
		}
		hasher := sha256.New()
		size, copyErr := io.Copy(hasher, reader)
		closeErr := reader.Close()
		if copyErr != nil {
			return kernel.PackageTreeIdentity{}, copyErr
		}
		if closeErr != nil {
			return kernel.PackageTreeIdentity{}, closeErr
		}
		if size != int64(entry.UncompressedSize64) {
			return kernel.PackageTreeIdentity{}, fmt.Errorf("archive entry size changed while hashing %q", name)
		}
		files = append(files, ciArchiveFileIdentity{
			path:   name,
			size:   size,
			digest: hex.EncodeToString(hasher.Sum(nil)),
		})
		total += size
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
	return kernel.PackageTreeIdentity{
		SHA256:    hex.EncodeToString(treeHasher.Sum(nil)),
		FileCount: len(files),
		SizeBytes: total,
	}, nil
}
