package localrecovery

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ManifestSchemaVersion = 1
	CatalogSchemaVersion  = 1

	MaxManifestBytes          = 4 << 20
	MaxCatalogBytes           = 8 << 20
	MaxProfileDefinitionBytes = 2 << 20
	MaxSnapshots              = 4096
	MaxFiles                  = 100000
	MaxFileBytes              = int64(16 << 30)
	MaxTotalBytes             = int64(64 << 30)
	MaxPathLength             = 4096
	MaxPathSegmentLength      = 255
	MaxIdentifierLength       = 256
	MaxTextLength             = 512
	MaxCodes                  = 64
	MaxCapabilities           = 64
)

var (
	ErrNotFound           = errors.New("local recovery record not found")
	ErrAlreadyExists      = errors.New("local recovery record already exists")
	ErrConflict           = errors.New("local recovery revision conflict")
	ErrUnsupportedVersion = errors.New("unsupported local recovery schema version")
	ErrInvalidManifest    = errors.New("invalid local recovery manifest")
	ErrInvalidRecord      = errors.New("invalid local recovery record")
)

type ArtifactScope string

const ScopeLocalFullSnapshot ArtifactScope = "local-full-snapshot"

type PortabilityClass string

const PortabilitySameUserSameMachine PortabilityClass = "same-user-same-machine"

type SnapshotStatus string

const (
	SnapshotPending          SnapshotStatus = "pending"
	SnapshotStaged           SnapshotStatus = "staged"
	SnapshotVerified         SnapshotStatus = "verified"
	SnapshotInvalid          SnapshotStatus = "invalid"
	SnapshotRecoveryRequired SnapshotStatus = "recovery-required"
)

func (s SnapshotStatus) Valid() bool {
	switch s {
	case SnapshotPending, SnapshotStaged, SnapshotVerified, SnapshotInvalid, SnapshotRecoveryRequired:
		return true
	default:
		return false
	}
}

type FileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func (e FileEntry) Validate(sourceOS string) error {
	if err := ValidateRelativePath(e.Path, sourceOS); err != nil {
		return err
	}
	if e.Size < 0 || e.Size > MaxFileBytes {
		return fmt.Errorf("%w: file %q size is outside bounds", ErrInvalidManifest, e.Path)
	}
	if err := validateDigest("file sha256", e.SHA256, ErrInvalidManifest); err != nil {
		return err
	}
	return nil
}

type KernelRequirement struct {
	ProviderID       string   `json:"providerId"`
	ProviderRevision int      `json:"providerRevision,omitempty"`
	BrowserVersion   string   `json:"browserVersion"`
	OperatingSystem  string   `json:"operatingSystem"`
	Architecture     string   `json:"architecture"`
	TrustRequirement string   `json:"trustRequirement"`
	ExecutableSHA256 string   `json:"executableSha256,omitempty"`
	PackageTreeSHA256 string   `json:"packageTreeSha256,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	Limitations      []string `json:"limitations,omitempty"`
}

func (r KernelRequirement) Validate() error {
	if err := validateIdentifier("kernel provider id", r.ProviderID, ErrInvalidManifest); err != nil {
		return err
	}
	if r.ProviderRevision < 0 {
		return fmt.Errorf("%w: kernel provider revision cannot be negative", ErrInvalidManifest)
	}
	if err := validateText("browser version", r.BrowserVersion, true, ErrInvalidManifest); err != nil {
		return err
	}
	if !validOS(r.OperatingSystem) || !validArch(r.Architecture) {
		return fmt.Errorf("%w: unsupported kernel platform %q/%q", ErrInvalidManifest, r.OperatingSystem, r.Architecture)
	}
	switch r.TrustRequirement {
	case "reviewed", "custom", "legacy":
	default:
		return fmt.Errorf("%w: unsupported kernel trust requirement %q", ErrInvalidManifest, r.TrustRequirement)
	}
	if r.TrustRequirement == "reviewed" && r.ExecutableSHA256 == "" && r.PackageTreeSHA256 == "" {
		return fmt.Errorf("%w: reviewed kernel requirement needs an exact identity", ErrInvalidManifest)
	}
	if r.ExecutableSHA256 != "" {
		if err := validateDigest("kernel executable sha256", r.ExecutableSHA256, ErrInvalidManifest); err != nil {
			return err
		}
	}
	if r.PackageTreeSHA256 != "" {
		if err := validateDigest("kernel package tree sha256", r.PackageTreeSHA256, ErrInvalidManifest); err != nil {
			return err
		}
	}
	if err := validateCodes("kernel capabilities", r.Capabilities, MaxCapabilities, ErrInvalidManifest); err != nil {
		return err
	}
	return validateCodes("kernel limitations", r.Limitations, MaxCodes, ErrInvalidManifest)
}

type AdapterRequirement struct {
	Kind             string   `json:"kind"`
	Version          string   `json:"version"`
	Official         bool     `json:"official"`
	ExecutableSHA256 string   `json:"executableSha256,omitempty"`
	Scheme           string   `json:"scheme"`
	OperatingSystem  string   `json:"operatingSystem"`
	Architecture     string   `json:"architecture"`
	Limitations      []string `json:"limitations,omitempty"`
}

func (r AdapterRequirement) Validate() error {
	switch r.Kind {
	case "xray", "sing-box":
	default:
		return fmt.Errorf("%w: unsupported adapter kind %q", ErrInvalidManifest, r.Kind)
	}
	if err := validateText("adapter version", r.Version, true, ErrInvalidManifest); err != nil {
		return err
	}
	if err := validateText("adapter scheme", r.Scheme, true, ErrInvalidManifest); err != nil {
		return err
	}
	if !validOS(r.OperatingSystem) || !validArch(r.Architecture) {
		return fmt.Errorf("%w: unsupported adapter platform %q/%q", ErrInvalidManifest, r.OperatingSystem, r.Architecture)
	}
	if r.Official && r.ExecutableSHA256 == "" {
		return fmt.Errorf("%w: official adapter requirement needs an exact identity", ErrInvalidManifest)
	}
	if r.ExecutableSHA256 != "" {
		if err := validateDigest("adapter executable sha256", r.ExecutableSHA256, ErrInvalidManifest); err != nil {
			return err
		}
	}
	return validateCodes("adapter limitations", r.Limitations, MaxCodes, ErrInvalidManifest)
}

type CredentialRequirement struct {
	PlaceholderID    string `json:"placeholderId"`
	Authentication   string `json:"authentication"`
	Label            string `json:"label,omitempty"`
	RequiresUsername bool   `json:"requiresUsername"`
	RequiresSecret   bool   `json:"requiresSecret"`
}

func (r CredentialRequirement) Validate() error {
	if err := validateIdentifier("credential placeholder id", r.PlaceholderID, ErrInvalidManifest); err != nil {
		return err
	}
	if err := validateText("credential authentication", r.Authentication, true, ErrInvalidManifest); err != nil {
		return err
	}
	if err := validateText("credential label", r.Label, false, ErrInvalidManifest); err != nil {
		return err
	}
	if !r.RequiresUsername && !r.RequiresSecret {
		return fmt.Errorf("%w: credential requirement must require user input", ErrInvalidManifest)
	}
	return nil
}

type DependencyRequirements struct {
	Kernel     KernelRequirement     `json:"kernel"`
	Adapter    *AdapterRequirement   `json:"adapter,omitempty"`
	Credential *CredentialRequirement `json:"credential,omitempty"`
}

func (r DependencyRequirements) Validate() error {
	if err := r.Kernel.Validate(); err != nil {
		return err
	}
	if r.Adapter != nil {
		if err := r.Adapter.Validate(); err != nil {
			return err
		}
	}
	if r.Credential != nil {
		if err := r.Credential.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type LocalSnapshotManifest struct {
	SchemaVersion               int                    `json:"schemaVersion"`
	SnapshotID                  string                 `json:"snapshotId"`
	Scope                       ArtifactScope          `json:"scope"`
	SourceProfileID             string                 `json:"sourceProfileId"`
	SourceProfileName           string                 `json:"sourceProfileName"`
	SourceProfileSchemaVersion  int                    `json:"sourceProfileSchemaVersion"`
	SourceApplicationVersion    string                 `json:"sourceApplicationVersion"`
	SourceOS                    string                 `json:"sourceOs"`
	SourceArch                  string                 `json:"sourceArch"`
	CreatedAt                   time.Time              `json:"createdAt"`
	ProfileDefinitionDigest     string                 `json:"profileDefinitionDigest"`
	IncludedRoots               []string               `json:"includedRoots"`
	TreeDigest                  string                 `json:"treeDigest"`
	FileCount                   int64                  `json:"fileCount"`
	TotalBytes                  int64                  `json:"totalBytes"`
	Files                       []FileEntry            `json:"files"`
	Dependencies                DependencyRequirements `json:"dependencies"`
	ExcludedData                []string               `json:"excludedData"`
	Portability                 PortabilityClass       `json:"portability"`
	Limitations                 []string               `json:"limitations,omitempty"`
	ParentSnapshotID            string                 `json:"parentSnapshotId,omitempty"`
}

func (m LocalSnapshotManifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("%w: manifest version %d", ErrUnsupportedVersion, m.SchemaVersion)
	}
	if err := validateIdentifier("snapshot id", m.SnapshotID, ErrInvalidManifest); err != nil {
		return err
	}
	if m.Scope != ScopeLocalFullSnapshot {
		return fmt.Errorf("%w: unsupported artifact scope %q", ErrInvalidManifest, m.Scope)
	}
	if err := validateIdentifier("source profile id", m.SourceProfileID, ErrInvalidManifest); err != nil {
		return err
	}
	if err := validateText("source profile name", m.SourceProfileName, true, ErrInvalidManifest); err != nil {
		return err
	}
	if m.SourceProfileSchemaVersion <= 0 {
		return fmt.Errorf("%w: source Profile schema version must be positive", ErrInvalidManifest)
	}
	if err := validateText("source application version", m.SourceApplicationVersion, true, ErrInvalidManifest); err != nil {
		return err
	}
	if !validOS(m.SourceOS) || !validArch(m.SourceArch) {
		return fmt.Errorf("%w: unsupported source platform %q/%q", ErrInvalidManifest, m.SourceOS, m.SourceArch)
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("%w: creation timestamp is required", ErrInvalidManifest)
	}
	if _, offset := m.CreatedAt.Zone(); offset != 0 {
		return fmt.Errorf("%w: creation timestamp must use UTC", ErrInvalidManifest)
	}
	if err := validateDigest("Profile definition digest", m.ProfileDefinitionDigest, ErrInvalidManifest); err != nil {
		return err
	}
	if len(m.IncludedRoots) != 2 || m.IncludedRoots[0] != "browser-data" || m.IncludedRoots[1] != "profile-definition" {
		return fmt.Errorf("%w: included roots must be browser-data and profile-definition", ErrInvalidManifest)
	}
	if m.FileCount != int64(len(m.Files)) || m.FileCount < 0 || m.FileCount > MaxFiles {
		return fmt.Errorf("%w: file count does not match bounded entries", ErrInvalidManifest)
	}
	var total int64
	seen := make(map[string]string, len(m.Files))
	previous := ""
	for _, entry := range m.Files {
		if err := entry.Validate(m.SourceOS); err != nil {
			return err
		}
		if previous != "" && entry.Path <= previous {
			return fmt.Errorf("%w: file entries must be strictly sorted", ErrInvalidManifest)
		}
		previous = entry.Path
		key := canonicalPathKey(entry.Path, m.SourceOS)
		if existing, ok := seen[key]; ok {
			return fmt.Errorf("%w: path collision between %q and %q", ErrInvalidManifest, existing, entry.Path)
		}
		seen[key] = entry.Path
		if entry.Size > MaxTotalBytes-total {
			return fmt.Errorf("%w: total bytes overflow bounds", ErrInvalidManifest)
		}
		total += entry.Size
	}
	if total != m.TotalBytes || m.TotalBytes < 0 || m.TotalBytes > MaxTotalBytes {
		return fmt.Errorf("%w: total bytes do not match bounded entries", ErrInvalidManifest)
	}
	digest, err := ComputeTreeDigest(m.SourceOS, m.Files)
	if err != nil {
		return err
	}
	if m.TreeDigest != digest {
		return fmt.Errorf("%w: tree digest does not match file entries", ErrInvalidManifest)
	}
	if err := m.Dependencies.Validate(); err != nil {
		return err
	}
	if err := validateRequiredExclusions(m.ExcludedData); err != nil {
		return err
	}
	if m.Portability != PortabilitySameUserSameMachine {
		return fmt.Errorf("%w: unsupported portability class %q", ErrInvalidManifest, m.Portability)
	}
	if err := validateCodes("manifest limitations", m.Limitations, MaxCodes, ErrInvalidManifest); err != nil {
		return err
	}
	if m.ParentSnapshotID != "" {
		return fmt.Errorf("%w: incremental snapshots are not approved", ErrInvalidManifest)
	}
	return nil
}

type CatalogRecord struct {
	SchemaVersion  int            `json:"schemaVersion"`
	SnapshotID     string         `json:"snapshotId"`
	SourceProfileID string        `json:"sourceProfileId"`
	ManifestRef    string         `json:"manifestRef"`
	Status         SnapshotStatus `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	VerifiedAt     *time.Time     `json:"verifiedAt,omitempty"`
	ManifestDigest string         `json:"manifestDigest"`
	TreeDigest     string         `json:"treeDigest"`
	FileCount      int64          `json:"fileCount"`
	TotalBytes     int64          `json:"totalBytes"`
	Limitations    []string       `json:"limitations,omitempty"`
	Revision       uint64         `json:"revision"`
}

func (r CatalogRecord) Validate() error {
	if r.SchemaVersion != CatalogSchemaVersion {
		return fmt.Errorf("%w: catalog record version %d", ErrUnsupportedVersion, r.SchemaVersion)
	}
	if err := validateIdentifier("snapshot id", r.SnapshotID, ErrInvalidRecord); err != nil {
		return err
	}
	if err := validateIdentifier("source profile id", r.SourceProfileID, ErrInvalidRecord); err != nil {
		return err
	}
	if r.ManifestRef != ExpectedManifestRef(r.SnapshotID) {
		return fmt.Errorf("%w: manifest reference is not canonical", ErrInvalidRecord)
	}
	if !r.Status.Valid() {
		return fmt.Errorf("%w: unsupported snapshot status %q", ErrInvalidRecord, r.Status)
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: invalid catalog timestamps", ErrInvalidRecord)
	}
	if (r.Status == SnapshotVerified) != (r.VerifiedAt != nil) {
		return fmt.Errorf("%w: verified status and timestamp disagree", ErrInvalidRecord)
	}
	if r.VerifiedAt != nil && r.VerifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: verification precedes creation", ErrInvalidRecord)
	}
	if err := validateDigest("manifest digest", r.ManifestDigest, ErrInvalidRecord); err != nil {
		return err
	}
	if err := validateDigest("tree digest", r.TreeDigest, ErrInvalidRecord); err != nil {
		return err
	}
	if r.FileCount < 0 || r.FileCount > MaxFiles || r.TotalBytes < 0 || r.TotalBytes > MaxTotalBytes {
		return fmt.Errorf("%w: catalog size summary is outside bounds", ErrInvalidRecord)
	}
	if err := validateCodes("catalog limitations", r.Limitations, MaxCodes, ErrInvalidRecord); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: catalog revision must be positive", ErrInvalidRecord)
	}
	return nil
}

func ExpectedManifestRef(snapshotID string) string {
	return path.Join("snapshots", strings.TrimSpace(snapshotID), "manifest.json")
}

func ValidateRelativePath(value, sourceOS string) error {
	if value == "" || strings.TrimSpace(value) != value || len(value) > MaxPathLength || !utf8.ValidString(value) {
		return fmt.Errorf("%w: invalid relative path", ErrInvalidManifest)
	}
	if strings.Contains(value, "\\") || strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || path.IsAbs(value) || path.Clean(value) != value {
		return fmt.Errorf("%w: path %q is not canonical relative form", ErrInvalidManifest, value)
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return fmt.Errorf("%w: path %q contains control characters", ErrInvalidManifest, value)
		}
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || len(segment) > MaxPathSegmentLength || strings.Contains(segment, ":") || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return fmt.Errorf("%w: path %q contains an unsafe segment", ErrInvalidManifest, value)
		}
		if sourceOS == "windows" && windowsReservedName(segment) {
			return fmt.Errorf("%w: path %q uses a reserved Windows name", ErrInvalidManifest, value)
		}
	}
	return nil
}

func canonicalPathKey(value, sourceOS string) string {
	if sourceOS == "windows" {
		return strings.ToLower(value)
	}
	return value
}

func windowsReservedName(segment string) bool {
	base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func validateIdentifier(label, value string, base error) error {
	if value == "" || strings.TrimSpace(value) != value || len(value) > MaxIdentifierLength {
		return fmt.Errorf("%w: invalid %s", base, label)
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') && !(char >= '0' && char <= '9') && char != '-' && char != '_' && char != '.' {
			return fmt.Errorf("%w: invalid %s", base, label)
		}
	}
	return nil
}

func validateText(label, value string, required bool, base error) error {
	if strings.TrimSpace(value) != value || len(value) > MaxTextLength || !utf8.ValidString(value) || (required && value == "") {
		return fmt.Errorf("%w: invalid %s", base, label)
	}
	for _, char := range value {
		if char == 0 || (char < 0x20 && char != '\t') || char == 0x7f {
			return fmt.Errorf("%w: invalid %s", base, label)
		}
	}
	return nil
}

func validateDigest(label, value string, base error) error {
	if len(value) != 64 {
		return fmt.Errorf("%w: invalid %s", base, label)
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') {
			return fmt.Errorf("%w: invalid %s", base, label)
		}
	}
	return nil
}

func validateCodes(label string, values []string, maximum int, base error) error {
	if len(values) > maximum || !sort.StringsAreSorted(values) {
		return fmt.Errorf("%w: %s must be bounded and sorted", base, label)
	}
	for index, value := range values {
		if err := validateText(label, value, true, base); err != nil {
			return err
		}
		if index > 0 && value == values[index-1] {
			return fmt.Errorf("%w: duplicate %s", base, label)
		}
	}
	return nil
}

func validateRequiredExclusions(values []string) error {
	required := []string{"adapter-binaries", "browser-evidence", "credential-secrets", "kernel-binaries", "runtime-logs", "runtime-state"}
	if len(values) < len(required) || !sort.StringsAreSorted(values) {
		return fmt.Errorf("%w: excluded data must be sorted and complete", ErrInvalidManifest)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateText("excluded data code", value, true, ErrInvalidManifest); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate excluded data code", ErrInvalidManifest)
		}
		seen[value] = struct{}{}
	}
	for _, value := range required {
		if _, exists := seen[value]; !exists {
			return fmt.Errorf("%w: required exclusion %q is missing", ErrInvalidManifest, value)
		}
	}
	return nil
}

func validOS(value string) bool { return value == "windows" || value == "linux" }
func validArch(value string) bool { return value == "amd64" || value == "arm64" }
