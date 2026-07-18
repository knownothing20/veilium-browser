package adapterrelease

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const ConfigPathToken = "{config}"

//go:embed releases.json
var releaseFiles embed.FS

var (
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	loadOnce      sync.Once
	loaded        Manifest
	loadErr       error
)

type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Releases      []Release `json:"releases"`
}

type Release struct {
	Kind        string  `json:"kind"`
	Version     string  `json:"version"`
	Tag         string  `json:"tag"`
	Repository  string  `json:"repository"`
	LicenseSPDX string  `json:"licenseSpdx"`
	PublishedAt string  `json:"publishedAt"`
	ReleaseID   int64   `json:"releaseId"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Platform            string   `json:"platform"`
	Arch                string   `json:"arch"`
	Name                string   `json:"name"`
	URL                 string   `json:"url"`
	ArchiveSHA256       string   `json:"archiveSha256"`
	ArchiveSizeBytes    int64    `json:"archiveSizeBytes"`
	ExecutablePath      string   `json:"executablePath"`
	ExecutableSHA256    string   `json:"executableSha256"`
	ExecutableSizeBytes int64    `json:"executableSizeBytes"`
	VersionArgs         []string `json:"versionArgs"`
	CheckArgs           []string `json:"checkArgs"`
}

type Pin struct {
	Kind                string   `json:"kind"`
	Version             string   `json:"version"`
	Tag                 string   `json:"tag"`
	Repository          string   `json:"repository"`
	LicenseSPDX         string   `json:"licenseSpdx"`
	PublishedAt         string   `json:"publishedAt"`
	Platform            string   `json:"platform"`
	Arch                string   `json:"arch"`
	AssetName           string   `json:"assetName"`
	AssetURL            string   `json:"assetUrl"`
	ArchiveSHA256       string   `json:"archiveSha256"`
	ArchiveSizeBytes    int64    `json:"archiveSizeBytes"`
	ExecutablePath      string   `json:"executablePath"`
	ExecutableSHA256    string   `json:"executableSha256"`
	ExecutableSizeBytes int64    `json:"executableSizeBytes"`
	VersionArgs         []string `json:"versionArgs"`
	ConfigurationArgs   []string `json:"configurationArgs"`
}

func Catalog() (Manifest, error) {
	loadOnce.Do(func() {
		data, err := releaseFiles.ReadFile("releases.json")
		if err != nil {
			loadErr = fmt.Errorf("read embedded adapter release manifest: %w", err)
			return
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&loaded); err != nil {
			loadErr = fmt.Errorf("decode embedded adapter release manifest: %w", err)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			loadErr = fmt.Errorf("adapter release manifest contains trailing data")
			return
		}
		loadErr = validate(loaded)
	})
	if loadErr != nil {
		return Manifest{}, loadErr
	}
	return cloneManifest(loaded), nil
}

func Pins() ([]Pin, error) {
	manifest, err := Catalog()
	if err != nil {
		return nil, err
	}
	var pins []Pin
	for _, release := range manifest.Releases {
		for _, asset := range release.Assets {
			pins = append(pins, Pin{
				Kind: release.Kind, Version: release.Version, Tag: release.Tag,
				Repository: release.Repository, LicenseSPDX: release.LicenseSPDX, PublishedAt: release.PublishedAt,
				Platform: asset.Platform, Arch: asset.Arch, AssetName: asset.Name, AssetURL: asset.URL,
				ArchiveSHA256: asset.ArchiveSHA256, ArchiveSizeBytes: asset.ArchiveSizeBytes,
				ExecutablePath: asset.ExecutablePath, ExecutableSHA256: asset.ExecutableSHA256,
				ExecutableSizeBytes: asset.ExecutableSizeBytes,
				VersionArgs:         append([]string(nil), asset.VersionArgs...),
				ConfigurationArgs:   append([]string(nil), asset.CheckArgs...),
			})
		}
	}
	sort.Slice(pins, func(i, j int) bool {
		if pins[i].Kind != pins[j].Kind {
			return pins[i].Kind < pins[j].Kind
		}
		if pins[i].Platform != pins[j].Platform {
			return pins[i].Platform < pins[j].Platform
		}
		return pins[i].Arch < pins[j].Arch
	})
	return pins, nil
}

func Find(kind, version, platform, arch string) (Pin, bool) {
	pins, err := Pins()
	if err != nil {
		return Pin{}, false
	}
	kind = normalizeKind(kind)
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	for _, pin := range pins {
		if pin.Kind == kind && pin.Version == version && pin.Platform == platform && pin.Arch == arch {
			return pin, true
		}
	}
	return Pin{}, false
}

func MatchExecutable(kind, version, digest string, size int64) (Pin, bool) {
	pins, err := Pins()
	if err != nil {
		return Pin{}, false
	}
	kind = normalizeKind(kind)
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	digest = strings.ToLower(strings.TrimSpace(digest))
	for _, pin := range pins {
		if pin.Kind == kind && pin.Version == version && pin.ExecutableSHA256 == digest && pin.ExecutableSizeBytes == size {
			return pin, true
		}
	}
	return Pin{}, false
}

func MaterializeCheckArgs(pin Pin, configPath string) ([]string, error) {
	if strings.TrimSpace(configPath) == "" || strings.ContainsRune(configPath, '\x00') {
		return nil, fmt.Errorf("configuration path is invalid")
	}
	result := make([]string, len(pin.ConfigurationArgs))
	found := false
	for index, value := range pin.ConfigurationArgs {
		if value == ConfigPathToken {
			result[index] = configPath
			found = true
			continue
		}
		if strings.Contains(value, ConfigPathToken) {
			return nil, fmt.Errorf("configuration token must be a complete argument")
		}
		result[index] = value
	}
	if !found {
		return nil, fmt.Errorf("configuration check command does not include a config path")
	}
	return result, nil
}

func validate(manifest Manifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported adapter release manifest schema %d", manifest.SchemaVersion)
	}
	if len(manifest.Releases) == 0 {
		return fmt.Errorf("adapter release manifest is empty")
	}
	seenReleases := map[string]bool{}
	seenAssets := map[string]bool{}
	for _, release := range manifest.Releases {
		release.Kind = normalizeKind(release.Kind)
		if release.Kind != "xray" && release.Kind != "sing-box" {
			return fmt.Errorf("unsupported release kind %q", release.Kind)
		}
		if release.Version == "" || release.Tag != "v"+release.Version || release.Repository == "" || release.LicenseSPDX == "" || release.ReleaseID < 1 {
			return fmt.Errorf("release metadata is incomplete for %q", release.Kind)
		}
		key := release.Kind + "@" + release.Version
		if seenReleases[key] {
			return fmt.Errorf("duplicate adapter release %q", key)
		}
		seenReleases[key] = true
		if len(release.Assets) == 0 {
			return fmt.Errorf("adapter release %q has no assets", key)
		}
		for _, asset := range release.Assets {
			if asset.Platform != "linux" && asset.Platform != "windows" {
				return fmt.Errorf("unsupported adapter platform %q", asset.Platform)
			}
			if asset.Arch != "amd64" {
				return fmt.Errorf("unsupported adapter architecture %q", asset.Arch)
			}
			if asset.Name == "" || asset.ArchiveSizeBytes < 1 || asset.ExecutableSizeBytes < 1 {
				return fmt.Errorf("adapter asset metadata is incomplete")
			}
			if !digestPattern.MatchString(asset.ArchiveSHA256) || !digestPattern.MatchString(asset.ExecutableSHA256) {
				return fmt.Errorf("adapter asset digest is invalid")
			}
			if !safeRelativePath(asset.ExecutablePath) {
				return fmt.Errorf("adapter executable path is unsafe")
			}
			parsed, err := url.Parse(asset.URL)
			expectedPath := "/" + release.Repository + "/releases/download/" + release.Tag + "/" + asset.Name
			if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.EscapedPath() != expectedPath {
				return fmt.Errorf("adapter asset URL does not match the pinned GitHub release")
			}
			assetKey := release.Kind + "/" + asset.Platform + "/" + asset.Arch
			if seenAssets[assetKey] {
				return fmt.Errorf("duplicate adapter asset %q", assetKey)
			}
			seenAssets[assetKey] = true
			if len(asset.VersionArgs) == 0 || len(asset.CheckArgs) == 0 {
				return fmt.Errorf("adapter command contract is incomplete")
			}
			if _, err := MaterializeCheckArgs(Pin{ConfigurationArgs: asset.CheckArgs}, "/private/config.json"); err != nil {
				return err
			}
		}
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

func normalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "singbox" || kind == "sing_box" {
		return "sing-box"
	}
	return kind
}

func cloneManifest(source Manifest) Manifest {
	result := source
	result.Releases = append([]Release(nil), source.Releases...)
	for index := range result.Releases {
		result.Releases[index].Assets = append([]Asset(nil), source.Releases[index].Assets...)
		for assetIndex := range result.Releases[index].Assets {
			asset := &result.Releases[index].Assets[assetIndex]
			asset.VersionArgs = append([]string(nil), asset.VersionArgs...)
			asset.CheckArgs = append([]string(nil), asset.CheckArgs...)
		}
	}
	return result
}
