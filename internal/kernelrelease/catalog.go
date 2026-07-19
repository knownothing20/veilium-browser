package kernelrelease

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
)

const ProviderID = "official-chromium-snapshot-win64"

//go:embed releases.json
var releaseFiles embed.FS

var (
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
	loadOnce sync.Once
	loaded Manifest
	loadErr error
)

type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Releases      []Release `json:"releases"`
}

type Release struct {
	ProviderID           string   `json:"providerId"`
	ProviderRevision     int      `json:"providerRevision"`
	Name                 string   `json:"name"`
	BrowserVersion       string   `json:"browserVersion"`
	SnapshotRevision     int64    `json:"snapshotRevision"`
	SourceProject        string   `json:"sourceProject"`
	SourcePageURL        string   `json:"sourcePageUrl"`
	LicenseSPDX          string   `json:"licenseSpdx"`
	LicenseURL           string   `json:"licenseUrl"`
	ThirdPartyNoticesURL string   `json:"thirdPartyNoticesUrl"`
	ReviewedAt           string   `json:"reviewedAt"`
	Platform             string   `json:"platform"`
	Arch                 string   `json:"arch"`
	ArchiveName          string   `json:"archiveName"`
	ArchiveURL           string   `json:"archiveUrl"`
	ArchiveSizeBytes     int64    `json:"archiveSizeBytes"`
	ArchiveSHA256        string   `json:"archiveSha256"`
	ArchiveEntryCount    int      `json:"archiveEntryCount"`
	ArchiveRoot          string   `json:"archiveRoot"`
	ExecutablePath       string   `json:"executablePath"`
	ExecutableSizeBytes  int64    `json:"executableSizeBytes"`
	ExecutableSHA256     string   `json:"executableSha256"`
	Limitations          []string `json:"limitations"`
}

func Catalog() (Manifest, error) {
	loadOnce.Do(func() {
		data, err := releaseFiles.ReadFile("releases.json")
		if err != nil {
			loadErr = fmt.Errorf("read embedded Chromium release manifest: %w", err)
			return
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&loaded); err != nil {
			loadErr = fmt.Errorf("decode embedded Chromium release manifest: %w", err)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			loadErr = fmt.Errorf("Chromium release manifest contains trailing data")
			return
		}
		loadErr = validate(loaded)
	})
	if loadErr != nil {
		return Manifest{}, loadErr
	}
	return cloneManifest(loaded), nil
}

func Releases() ([]Release, error) {
	manifest, err := Catalog()
	if err != nil {
		return nil, err
	}
	return append([]Release(nil), manifest.Releases...), nil
}

func Find(providerID, version, platform, arch string) (Release, bool) {
	releases, err := Releases()
	if err != nil {
		return Release{}, false
	}
	for _, release := range releases {
		if release.ProviderID == strings.TrimSpace(providerID) && release.BrowserVersion == strings.TrimSpace(version) && release.Platform == strings.TrimSpace(platform) && release.Arch == strings.TrimSpace(arch) {
			return cloneRelease(release), true
		}
	}
	return Release{}, false
}

func MatchExecutable(providerID, version, digest string, size int64) (Release, bool) {
	releases, err := Releases()
	if err != nil {
		return Release{}, false
	}
	digest = strings.ToLower(strings.TrimSpace(digest))
	for _, release := range releases {
		if release.ProviderID == strings.TrimSpace(providerID) && release.BrowserVersion == strings.TrimSpace(version) && release.ExecutableSHA256 == digest && release.ExecutableSizeBytes == size {
			return cloneRelease(release), true
		}
	}
	return Release{}, false
}

func validate(manifest Manifest) error {
	if manifest.SchemaVersion != 1 || len(manifest.Releases) != 1 {
		return fmt.Errorf("reviewed Chromium manifest must contain exactly one schema-v1 release")
	}
	release := manifest.Releases[0]
	if release.ProviderID != ProviderID || release.ProviderRevision < 1 || strings.TrimSpace(release.Name) == "" {
		return fmt.Errorf("reviewed Chromium Provider identity is invalid")
	}
	if !versionPattern.MatchString(release.BrowserVersion) || release.SnapshotRevision < 1 || release.Platform != "windows" || release.Arch != "amd64" {
		return fmt.Errorf("reviewed Chromium version or platform is invalid")
	}
	if release.SourceProject != "The Chromium Project" || release.LicenseSPDX != "BSD-3-Clause" || strings.TrimSpace(release.ReviewedAt) == "" {
		return fmt.Errorf("reviewed Chromium provenance metadata is incomplete")
	}
	if release.ArchiveName != "chrome-win.zip" || release.ArchiveSizeBytes < 50<<20 || release.ArchiveSizeBytes > 500<<20 || release.ArchiveEntryCount < 1 || release.ArchiveEntryCount > 5000 {
		return fmt.Errorf("reviewed Chromium archive metadata is invalid")
	}
	if !digestPattern.MatchString(release.ArchiveSHA256) || !digestPattern.MatchString(release.ExecutableSHA256) || release.ExecutableSizeBytes < 1 {
		return fmt.Errorf("reviewed Chromium digest metadata is invalid")
	}
	if release.ArchiveRoot != "chrome-win" || release.ExecutablePath != "chrome-win/chrome.exe" || !safeRelativePath(release.ExecutablePath) {
		return fmt.Errorf("reviewed Chromium archive layout is invalid")
	}
	expectedArchivePath := fmt.Sprintf("/chromium-browser-snapshots/Win_x64/%d/chrome-win.zip", release.SnapshotRevision)
	if err := validateHTTPSURL(release.ArchiveURL, "commondatastorage.googleapis.com", expectedArchivePath); err != nil {
		return err
	}
	if err := validateHTTPSURL(release.SourcePageURL, "www.chromium.org", "/getting-involved/download-chromium/"); err != nil {
		return err
	}
	license, err := url.Parse(release.LicenseURL)
	if err != nil || license.Scheme != "https" || license.Hostname() != "chromium.googlesource.com" || license.User != nil || license.Fragment != "" || !strings.HasSuffix(license.EscapedPath(), "/LICENSE") {
		return fmt.Errorf("reviewed Chromium license URL is invalid")
	}
	if release.ThirdPartyNoticesURL != "chrome://credits/" || len(release.Limitations) < 3 {
		return fmt.Errorf("reviewed Chromium notices or limitations are incomplete")
	}
	return nil
}

func validateHTTPSURL(raw, host, expectedPath string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != host || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.EscapedPath() != expectedPath {
		return fmt.Errorf("reviewed Chromium URL is invalid")
	}
	return nil
}

func safeRelativePath(value string) bool {
	if value == "" || strings.ContainsRune(value, '\\') || strings.ContainsRune(value, '\x00') || path.IsAbs(value) {
		return false
	}
	clean := path.Clean(value)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && clean == value
}

func cloneManifest(source Manifest) Manifest {
	result := source
	result.Releases = append([]Release(nil), source.Releases...)
	for index := range result.Releases {
		result.Releases[index] = cloneRelease(result.Releases[index])
	}
	return result
}

func cloneRelease(source Release) Release {
	result := source
	result.Limitations = append([]string(nil), source.Limitations...)
	return result
}
