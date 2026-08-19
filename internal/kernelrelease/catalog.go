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
	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
	loadOnce       sync.Once
	loaded         Manifest
	loadErr        error
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
	AllowedDownloadHosts []string `json:"allowedDownloadHosts"`
	ArchiveSizeBytes     int64    `json:"archiveSizeBytes"`
	ArchiveSHA256        string   `json:"archiveSha256"`
	ArchiveEntryCount    int      `json:"archiveEntryCount"`
	ArchiveRoot          string   `json:"archiveRoot"`
	ExecutablePath       string   `json:"executablePath"`
	ExecutableSizeBytes  int64    `json:"executableSizeBytes"`
	ExecutableSHA256     string   `json:"executableSha256"`
	PackageFileCount     int      `json:"packageFileCount"`
	ExpandedSizeBytes    int64    `json:"expandedSizeBytes"`
	PackageTreeSHA256    string   `json:"packageTreeSha256"`
	Limitations          []string `json:"limitations"`
}

type Identity struct {
	ProviderID       string `json:"providerId"`
	ProviderRevision int    `json:"providerRevision"`
	BrowserVersion   string `json:"browserVersion"`
	Platform         string `json:"platform"`
	Arch             string `json:"arch"`
}

type providerValidationPolicy struct {
	SourceProject, LicenseSPDX                       string
	Platform, Arch                                   string
	ArchiveName, ArchiveRoot, ExecutablePath         string
	ArchiveHosts                                     []string
	ArchivePath                                      func(Release) string
	SourceHost, SourcePath                           string
	LicenseHost, LicensePathSuffix, ThirdPartyNotice string
	MinimumArchiveSize, MaximumArchiveSize           int64
}

var productionProviderPolicies = map[string]providerValidationPolicy{
	ProviderID: {
		SourceProject: "The Chromium Project", LicenseSPDX: "BSD-3-Clause",
		Platform: "windows", Arch: "amd64",
		ArchiveName: "chrome-win.zip", ArchiveRoot: "chrome-win", ExecutablePath: "chrome-win/chrome.exe",
		ArchiveHosts: []string{"commondatastorage.googleapis.com", "storage.googleapis.com"},
		ArchivePath: func(release Release) string {
			return fmt.Sprintf("/chromium-browser-snapshots/Win_x64/%d/chrome-win.zip", release.SnapshotRevision)
		},
		SourceHost: "www.chromium.org", SourcePath: "/getting-involved/download-chromium/",
		LicenseHost: "chromium.googlesource.com", LicensePathSuffix: "/LICENSE", ThirdPartyNotice: "chrome://credits/",
		MinimumArchiveSize: 50 << 20, MaximumArchiveSize: 500 << 20,
	},
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
	var match Release
	found := false
	for _, release := range releases {
		if release.ProviderID == strings.TrimSpace(providerID) && release.BrowserVersion == strings.TrimSpace(version) && release.Platform == strings.TrimSpace(platform) && release.Arch == strings.TrimSpace(arch) {
			if found {
				return Release{}, false
			}
			match = release
			found = true
		}
	}
	return cloneRelease(match), found
}

func FindExact(identity Identity) (Release, bool) {
	releases, err := Releases()
	if err != nil {
		return Release{}, false
	}
	return findExact(releases, identity)
}

func FindRevision(providerID string, providerRevision int, version, platform, arch string) (Release, bool) {
	if providerRevision == 0 {
		return Find(providerID, version, platform, arch)
	}
	return FindExact(Identity{ProviderID: providerID, ProviderRevision: providerRevision, BrowserVersion: version, Platform: platform, Arch: arch})
}

func findExact(releases []Release, identity Identity) (Release, bool) {
	for _, release := range releases {
		if release.Identity() == normalizeIdentity(identity) {
			return cloneRelease(release), true
		}
	}
	return Release{}, false
}

func (release Release) Identity() Identity {
	return Identity{ProviderID: release.ProviderID, ProviderRevision: release.ProviderRevision, BrowserVersion: release.BrowserVersion, Platform: release.Platform, Arch: release.Arch}
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

func MatchPackage(providerID, version, executableDigest string, executableSize int64, packageDigest string, packageFiles int, packageSize int64) (Release, bool) {
	releases, err := Releases()
	if err != nil {
		return Release{}, false
	}
	executableDigest = strings.ToLower(strings.TrimSpace(executableDigest))
	packageDigest = strings.ToLower(strings.TrimSpace(packageDigest))
	for _, release := range releases {
		if release.ProviderID == strings.TrimSpace(providerID) && release.BrowserVersion == strings.TrimSpace(version) && release.ExecutableSHA256 == executableDigest && release.ExecutableSizeBytes == executableSize && release.PackageTreeSHA256 == packageDigest && release.PackageFileCount == packageFiles && release.ExpandedSizeBytes == packageSize {
			return cloneRelease(release), true
		}
	}
	return Release{}, false
}

func MatchPackageExact(identity Identity, executableDigest string, executableSize int64, packageDigest string, packageFiles int, packageSize int64) (Release, bool) {
	release, ok := FindExact(identity)
	if !ok {
		return Release{}, false
	}
	executableDigest = strings.ToLower(strings.TrimSpace(executableDigest))
	packageDigest = strings.ToLower(strings.TrimSpace(packageDigest))
	if release.ExecutableSHA256 != executableDigest || release.ExecutableSizeBytes != executableSize || release.PackageTreeSHA256 != packageDigest || release.PackageFileCount != packageFiles || release.ExpandedSizeBytes != packageSize {
		return Release{}, false
	}
	return release, true
}

func validate(manifest Manifest) error {
	return validateWithPolicies(manifest, productionProviderPolicies)
}

func validateWithPolicies(manifest Manifest, policies map[string]providerValidationPolicy) error {
	if manifest.SchemaVersion != 1 || len(manifest.Releases) < 1 || len(manifest.Releases) > 128 {
		return fmt.Errorf("reviewed Chromium manifest must contain 1 to 128 schema-v1 releases")
	}
	seen := make(map[Identity]struct{}, len(manifest.Releases))
	for index, release := range manifest.Releases {
		identity := normalizeIdentity(release.Identity())
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf("reviewed Chromium manifest contains duplicate compound identity at release %d", index)
		}
		seen[identity] = struct{}{}
		policy, ok := policies[identity.ProviderID]
		if !ok {
			return fmt.Errorf("reviewed Chromium Provider %q has no validation policy", identity.ProviderID)
		}
		if err := validateRelease(release, policy); err != nil {
			return fmt.Errorf("release %d: %w", index, err)
		}
	}
	return nil
}

func validateRelease(release Release, policy providerValidationPolicy) error {
	if strings.TrimSpace(release.ProviderID) == "" || release.ProviderID != strings.TrimSpace(release.ProviderID) || release.ProviderRevision < 1 || strings.TrimSpace(release.Name) == "" {
		return fmt.Errorf("reviewed Chromium Provider identity is invalid")
	}
	if !versionPattern.MatchString(release.BrowserVersion) || release.SnapshotRevision < 1 || release.Platform != policy.Platform || release.Arch != policy.Arch {
		return fmt.Errorf("reviewed Chromium version or platform is invalid")
	}
	if release.SourceProject != policy.SourceProject || release.LicenseSPDX != policy.LicenseSPDX || strings.TrimSpace(release.ReviewedAt) == "" {
		return fmt.Errorf("reviewed Chromium provenance metadata is incomplete")
	}
	if release.ArchiveName != policy.ArchiveName || release.ArchiveSizeBytes < policy.MinimumArchiveSize || release.ArchiveSizeBytes > policy.MaximumArchiveSize || release.ArchiveEntryCount < 1 || release.ArchiveEntryCount > 5000 {
		return fmt.Errorf("reviewed Chromium archive metadata is invalid")
	}
	if !digestPattern.MatchString(release.ArchiveSHA256) || !digestPattern.MatchString(release.ExecutableSHA256) || !digestPattern.MatchString(release.PackageTreeSHA256) || release.ExecutableSizeBytes < 1 {
		return fmt.Errorf("reviewed Chromium digest metadata is invalid")
	}
	if release.PackageFileCount != release.ArchiveEntryCount || release.ExpandedSizeBytes < release.ExecutableSizeBytes {
		return fmt.Errorf("reviewed Chromium package-tree metadata is invalid")
	}
	if release.ArchiveRoot != policy.ArchiveRoot || release.ExecutablePath != policy.ExecutablePath || !safeRelativePath(release.ExecutablePath) || !strings.HasPrefix(release.ExecutablePath, release.ArchiveRoot+"/") {
		return fmt.Errorf("reviewed Chromium archive layout is invalid")
	}
	if len(policy.ArchiveHosts) == 0 || policy.ArchivePath == nil || !equalStrings(release.AllowedDownloadHosts, policy.ArchiveHosts) {
		return fmt.Errorf("reviewed Chromium download-host policy is invalid")
	}
	if err := validateHTTPSURL(release.ArchiveURL, policy.ArchiveHosts[0], policy.ArchivePath(release)); err != nil {
		return err
	}
	if err := validateHTTPSURL(release.SourcePageURL, policy.SourceHost, policy.SourcePath); err != nil {
		return err
	}
	license, err := url.Parse(release.LicenseURL)
	if err != nil || license.Scheme != "https" || license.Hostname() != policy.LicenseHost || license.User != nil || license.RawQuery != "" || license.Fragment != "" || !strings.HasSuffix(license.EscapedPath(), policy.LicensePathSuffix) {
		return fmt.Errorf("reviewed Chromium license URL is invalid")
	}
	if release.ThirdPartyNoticesURL != policy.ThirdPartyNotice || len(release.Limitations) < 3 {
		return fmt.Errorf("reviewed Chromium notices or limitations are incomplete")
	}
	return nil
}

func normalizeIdentity(identity Identity) Identity {
	identity.ProviderID = strings.TrimSpace(identity.ProviderID)
	identity.BrowserVersion = strings.TrimSpace(identity.BrowserVersion)
	identity.Platform = strings.TrimSpace(identity.Platform)
	identity.Arch = strings.TrimSpace(identity.Arch)
	return identity
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.TrimSpace(left[index]) != strings.TrimSpace(right[index]) {
			return false
		}
	}
	return true
}

func DownloadURLAllowed(release Release, candidate *url.URL) bool {
	if candidate == nil || candidate.String() != release.ArchiveURL || candidate.Scheme != "https" || candidate.User != nil || candidate.RawQuery != "" || candidate.Fragment != "" || (candidate.Port() != "" && candidate.Port() != "443") {
		return false
	}
	for _, host := range release.AllowedDownloadHosts {
		if strings.EqualFold(candidate.Hostname(), strings.TrimSpace(host)) {
			return true
		}
	}
	return false
}

func RedirectURLAllowed(release Release, candidate *url.URL) bool {
	if candidate == nil || candidate.Scheme != "https" || candidate.User != nil || candidate.Fragment != "" || (candidate.Port() != "" && candidate.Port() != "443") {
		return false
	}
	for _, host := range release.AllowedDownloadHosts {
		if strings.EqualFold(candidate.Hostname(), strings.TrimSpace(host)) {
			return true
		}
	}
	return false
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
	result.AllowedDownloadHosts = append([]string(nil), source.AllowedDownloadHosts...)
	result.Limitations = append([]string(nil), source.Limitations...)
	return result
}
