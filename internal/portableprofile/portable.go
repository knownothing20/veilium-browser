package portableprofile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

const (
	SchemaVersion       = 1
	ArtifactKind        = "veilium-portable-profile"
	TemplateCatalogKind = "veilium-profile-templates"
	MaxArtifactBytes    = 1 << 20
	MaxTemplates        = 200
	MaxTags             = 32
	MaxTextBytes         = 16 << 10
)

type IdentityMode string

const (
	IdentityNew      IdentityMode = "new-identity"
	IdentityPreserve IdentityMode = "preserve-identity"
)

type KernelRequirement struct {
	Provider  string `json:"provider"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

type AdapterRequirement struct {
	Kind      string `json:"kind"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

type Payload struct {
	Name               string                `json:"name"`
	Group              string                `json:"group,omitempty"`
	Notes              string                `json:"notes,omitempty"`
	Tags               []string              `json:"tags,omitempty"`
	Fingerprint        domain.FingerprintConfig `json:"fingerprint"`
	ProxyURL           string                `json:"proxyUrl,omitempty"`
	RouteOmitted       bool                  `json:"routeOmitted,omitempty"`
	CredentialRequired bool                  `json:"credentialRequired,omitempty"`
	Kernel             KernelRequirement     `json:"kernel"`
	Adapter            *AdapterRequirement   `json:"adapter,omitempty"`
	IdentityMode       IdentityMode          `json:"identityMode"`
}

type Artifact struct {
	SchemaVersion      int       `json:"schemaVersion"`
	Kind               string    `json:"kind"`
	ApplicationVersion string    `json:"applicationVersion"`
	ExportedAt         time.Time `json:"exportedAt"`
	Payload            Payload   `json:"payload"`
	PayloadSHA256      string    `json:"payloadSha256"`
	Exclusions         []string  `json:"exclusions"`
	Limitations        []string  `json:"limitations,omitempty"`
}

type BuildInput struct {
	ApplicationVersion string
	Profile            domain.Profile
	Kernel             KernelRequirement
	Adapter            *AdapterRequirement
	CredentialRequired bool
	IdentityMode       IdentityMode
	ExportedAt         time.Time
}

type Template struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Payload       Payload   `json:"payload"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type TemplateCatalog struct {
	SchemaVersion int        `json:"schemaVersion"`
	Kind          string     `json:"kind"`
	Templates     []Template `json:"templates"`
}

func Build(input BuildInput) (Artifact, error) {
	mode := input.IdentityMode
	if mode == "" {
		mode = IdentityNew
	}
	if err := validateIdentityMode(mode); err != nil {
		return Artifact{}, err
	}
	payload := Payload{
		Name:               strings.TrimSpace(input.Profile.Name),
		Group:              strings.TrimSpace(input.Profile.Group),
		Notes:              strings.TrimSpace(input.Profile.Notes),
		Tags:               normalizeTags(input.Profile.Tags),
		Fingerprint:        input.Profile.Fingerprint,
		ProxyURL:           strings.TrimSpace(input.Profile.Proxy.URL),
		CredentialRequired: input.CredentialRequired,
		Kernel:             input.Kernel,
		Adapter:            cloneAdapter(input.Adapter),
		IdentityMode:       mode,
	}
	if mode == IdentityNew {
		payload.Fingerprint.Seed = ""
	}
	if err := validatePayload(payload); err != nil {
		return Artifact{}, err
	}
	digest, err := payloadDigest(payload)
	if err != nil {
		return Artifact{}, err
	}
	exportedAt := input.ExportedAt.UTC()
	if exportedAt.IsZero() {
		exportedAt = time.Now().UTC()
	}
	artifact := Artifact{
		SchemaVersion:      SchemaVersion,
		Kind:               ArtifactKind,
		ApplicationVersion: strings.TrimSpace(input.ApplicationVersion),
		ExportedAt:         exportedAt,
		Payload:            payload,
		PayloadSHA256:      digest,
		Exclusions: []string{
			"browser-user-data",
			"cookies-and-site-storage",
			"credential-values-and-local-credential-ids",
			"kernel-and-adapter-binaries",
			"local-profile-id-and-managed-paths",
			"runtime-state-logs-and-temporary-files",
			"browser-and-network-evidence",
		},
		Limitations: []string{
			"Dependencies must be remapped to currently verified local records.",
			"Imported profiles do not inherit health, compatibility, provider trust, or Evidence.",
		},
	}
	if mode == IdentityPreserve {
		artifact.Limitations = append(artifact.Limitations, "Preserved identity material must not be used simultaneously on multiple devices or Profiles.")
	}
	return artifact, nil
}

func Encode(artifact Artifact) ([]byte, error) {
	if err := Validate(artifact); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode portable profile: %w", err)
	}
	data = append(data, '\n')
	if len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("portable profile exceeds %d bytes", MaxArtifactBytes)
	}
	return data, nil
}

func Decode(data []byte) (Artifact, error) {
	if len(data) == 0 {
		return Artifact{}, fmt.Errorf("portable profile is empty")
	}
	if len(data) > MaxArtifactBytes {
		return Artifact{}, fmt.Errorf("portable profile exceeds %d bytes", MaxArtifactBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var artifact Artifact
	if err := decoder.Decode(&artifact); err != nil {
		return Artifact{}, fmt.Errorf("decode portable profile: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Artifact{}, fmt.Errorf("portable profile contains trailing JSON")
		}
		return Artifact{}, fmt.Errorf("decode portable profile trailing data: %w", err)
	}
	if err := Validate(artifact); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func Validate(artifact Artifact) error {
	if artifact.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported portable profile schema %d", artifact.SchemaVersion)
	}
	if artifact.Kind != ArtifactKind {
		return fmt.Errorf("unexpected portable profile kind %q", artifact.Kind)
	}
	if artifact.ExportedAt.IsZero() {
		return fmt.Errorf("portable profile export timestamp is required")
	}
	if len(artifact.ApplicationVersion) > 120 || containsControl(artifact.ApplicationVersion) {
		return fmt.Errorf("portable profile application version is invalid")
	}
	if err := validatePayload(artifact.Payload); err != nil {
		return err
	}
	digest, err := payloadDigest(artifact.Payload)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(artifact.PayloadSHA256), digest) {
		return fmt.Errorf("portable profile payload digest mismatch")
	}
	if len(artifact.Exclusions) == 0 || len(artifact.Exclusions) > 32 {
		return fmt.Errorf("portable profile exclusions are invalid")
	}
	if len(artifact.Limitations) > 32 {
		return fmt.Errorf("portable profile limitations are invalid")
	}
	return nil
}

func Read(path string) (Artifact, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Artifact{}, fmt.Errorf("portable profile path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("inspect portable profile: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("portable profile must be a regular file")
	}
	if info.Size() > MaxArtifactBytes {
		return Artifact{}, fmt.Errorf("portable profile exceeds %d bytes", MaxArtifactBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("read portable profile: %w", err)
	}
	return Decode(data)
}

func Write(path string, artifact Artifact) error {
	data, err := Encode(artifact)
	if err != nil {
		return err
	}
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return fmt.Errorf("portable profile destination is required")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("portable profile destination must be a regular file")
		}
		return fmt.Errorf("portable profile destination already exists")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect portable profile destination: %w", statErr)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create portable profile directory: %w", err)
	}
	temp, err := os.CreateTemp(directory, ".veilium-portable-*.tmp")
	if err != nil {
		return fmt.Errorf("create portable profile staging file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write portable profile staging file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync portable profile staging file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close portable profile staging file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish portable profile: %w", err)
	}
	return nil
}

func NewSeed() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate fingerprint seed: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func NewTemplate(name string, payload Payload, now time.Time) (Template, error) {
	name = strings.TrimSpace(name)
	if err := validateText("template name", name, false, 120); err != nil {
		return Template{}, err
	}
	payload.IdentityMode = IdentityNew
	payload.Fingerprint.Seed = ""
	if err := validatePayload(payload); err != nil {
		return Template{}, err
	}
	id, err := randomID()
	if err != nil {
		return Template{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Template{SchemaVersion: SchemaVersion, ID: id, Name: name, Payload: payload, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
}

func LoadTemplates(path string) (TemplateCatalog, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return TemplateCatalog{SchemaVersion: SchemaVersion, Kind: TemplateCatalogKind, Templates: []Template{}}, nil
	}
	if err != nil {
		return TemplateCatalog{}, fmt.Errorf("read template catalog: %w", err)
	}
	if len(data) > MaxArtifactBytes {
		return TemplateCatalog{}, fmt.Errorf("template catalog exceeds %d bytes", MaxArtifactBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog TemplateCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return TemplateCatalog{}, fmt.Errorf("decode template catalog: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return TemplateCatalog{}, fmt.Errorf("template catalog contains trailing data")
	}
	if err := ValidateTemplates(catalog); err != nil {
		return TemplateCatalog{}, err
	}
	return catalog, nil
}

func SaveTemplates(path string, catalog TemplateCatalog) error {
	catalog.SchemaVersion = SchemaVersion
	catalog.Kind = TemplateCatalogKind
	sort.Slice(catalog.Templates, func(i, j int) bool { return catalog.Templates[i].ID < catalog.Templates[j].ID })
	if err := ValidateTemplates(catalog); err != nil {
		return err
	}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode template catalog: %w", err)
	}
	data = append(data, '\n')
	if len(data) > MaxArtifactBytes {
		return fmt.Errorf("template catalog exceeds %d bytes", MaxArtifactBytes)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create template catalog directory: %w", err)
	}
	temp, err := os.CreateTemp(directory, ".portable-templates-*.tmp")
	if err != nil {
		return fmt.Errorf("create template catalog staging file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	backup := ""
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("template catalog must be a regular file")
		}
		backup = path + ".backup"
		_ = os.Remove(backup)
		if err := os.Rename(path, backup); err != nil {
			return fmt.Errorf("stage previous template catalog: %w", err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect template catalog: %w", statErr)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if backup != "" {
			_ = os.Rename(backup, path)
		}
		return fmt.Errorf("publish template catalog: %w", err)
	}
	if backup != "" {
		_ = os.Remove(backup)
	}
	return nil
}

func ValidateTemplates(catalog TemplateCatalog) error {
	if catalog.SchemaVersion != SchemaVersion || catalog.Kind != TemplateCatalogKind {
		return fmt.Errorf("unsupported template catalog")
	}
	if len(catalog.Templates) > MaxTemplates {
		return fmt.Errorf("template catalog exceeds %d records", MaxTemplates)
	}
	seen := make(map[string]struct{}, len(catalog.Templates))
	for _, item := range catalog.Templates {
		if item.SchemaVersion != SchemaVersion {
			return fmt.Errorf("template %q has unsupported schema", item.ID)
		}
		if err := validateIdentifier("template id", item.ID); err != nil {
			return err
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("duplicate template id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		if err := validateText("template name", item.Name, false, 120); err != nil {
			return err
		}
		if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() || item.UpdatedAt.Before(item.CreatedAt) {
			return fmt.Errorf("template %q timestamps are invalid", item.ID)
		}
		if item.Payload.IdentityMode != IdentityNew || item.Payload.Fingerprint.Seed != "" {
			return fmt.Errorf("template %q must create a new identity", item.ID)
		}
		if err := validatePayload(item.Payload); err != nil {
			return fmt.Errorf("template %q: %w", item.ID, err)
		}
	}
	return nil
}

func validatePayload(payload Payload) error {
	if err := validateText("profile name", payload.Name, false, 120); err != nil {
		return err
	}
	if err := validateText("profile group", payload.Group, true, 120); err != nil {
		return err
	}
	if err := validateText("profile notes", payload.Notes, true, MaxTextBytes); err != nil {
		return err
	}
	if len(payload.Tags) > MaxTags {
		return fmt.Errorf("portable profile has too many tags")
	}
	for _, tag := range payload.Tags {
		if err := validateText("profile tag", tag, false, 80); err != nil {
			return err
		}
	}
	if err := validateRequirement(payload.Kernel.Provider, payload.Kernel.Version, payload.Kernel.SHA256, payload.Kernel.SizeBytes, "kernel"); err != nil {
		return err
	}
	if payload.Adapter != nil {
		if err := validateRequirement(payload.Adapter.Kind, payload.Adapter.Version, payload.Adapter.SHA256, payload.Adapter.SizeBytes, "adapter"); err != nil {
			return err
		}
	}
	if err := validateIdentityMode(payload.IdentityMode); err != nil {
		return err
	}
	if payload.IdentityMode == IdentityNew && payload.Fingerprint.Seed != "" {
		return fmt.Errorf("new-identity payload must not contain a fingerprint seed")
	}
	if payload.RouteOmitted && strings.TrimSpace(payload.ProxyURL) != "" {
		return fmt.Errorf("omitted route cannot include a proxy URL")
	}
	if err := validatePortableURL(payload.ProxyURL); err != nil {
		return err
	}
	return nil
}

func validateRequirement(kind, version, digest string, size int64, label string) error {
	if err := validateText(label+" kind", strings.TrimSpace(kind), false, 120); err != nil {
		return err
	}
	if err := validateText(label+" version", strings.TrimSpace(version), false, 120); err != nil {
		return err
	}
	digest = strings.TrimSpace(digest)
	if len(digest) != 64 {
		return fmt.Errorf("%s sha256 is invalid", label)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("%s sha256 is invalid", label)
	}
	if size <= 0 {
		return fmt.Errorf("%s size is invalid", label)
	}
	return nil
}

func validatePortableURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) > 4096 || containsControl(raw) {
		return fmt.Errorf("portable proxy URL is invalid")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("portable proxy URL is invalid")
	}
	if parsed.User != nil {
		return fmt.Errorf("portable proxy URL must not contain inline credentials")
	}
	for key := range parsed.Query() {
		lower := strings.ToLower(strings.TrimSpace(key))
		for _, fragment := range []string{"password", "passwd", "secret", "token", "credential", "private", "auth", "uuid"} {
			if strings.Contains(lower, fragment) {
				return fmt.Errorf("portable proxy URL contains a secret-like query field %q", key)
			}
		}
	}
	return nil
}

func validateIdentityMode(mode IdentityMode) error {
	if mode != IdentityNew && mode != IdentityPreserve {
		return fmt.Errorf("unsupported identity transfer mode %q", mode)
	}
	return nil
}

func payloadDigest(payload Payload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode portable payload: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, value := range tags {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i]) < strings.ToLower(result[j]) })
	return result
}

func cloneAdapter(input *AdapterRequirement) *AdapterRequirement {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func validateText(label, value string, optional bool, max int) error {
	value = strings.TrimSpace(value)
	if value == "" && !optional {
		return fmt.Errorf("%s is required", label)
	}
	if len(value) > max || containsControl(value) {
		return fmt.Errorf("%s is invalid", label)
	}
	return nil
}

func validateIdentifier(label, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || containsControl(value) || strings.ContainsAny(value, "/\\") {
		return fmt.Errorf("%s is invalid", label)
	}
	return nil
}

func containsControl(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate portable record id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
