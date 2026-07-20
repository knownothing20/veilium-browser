package localrecovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

func ComputeTreeDigest(sourceOS string, entries []FileEntry) (string, error) {
	if !validOS(sourceOS) {
		return "", fmt.Errorf("%w: unsupported source operating system %q", ErrInvalidManifest, sourceOS)
	}
	ordered := append([]FileEntry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	seen := make(map[string]string, len(ordered))
	digest := sha256.New()
	for _, entry := range ordered {
		if err := entry.Validate(sourceOS); err != nil {
			return "", err
		}
		key := canonicalPathKey(entry.Path, sourceOS)
		if existing, exists := seen[key]; exists {
			return "", fmt.Errorf("%w: path collision between %q and %q", ErrInvalidManifest, existing, entry.Path)
		}
		seen[key] = entry.Path
		_, _ = digest.Write([]byte(entry.Path))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(strconv.FormatInt(entry.Size, 10)))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(entry.SHA256))
		_, _ = digest.Write([]byte{'\n'})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func ComputeManifestDigest(manifest LocalSnapshotManifest) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode local recovery manifest for digest: %w", err)
	}
	if len(data) > MaxManifestBytes {
		return "", fmt.Errorf("%w: encoded manifest exceeds %d bytes", ErrInvalidManifest, MaxManifestBytes)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func DigestProfileDefinition(data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxProfileDefinitionBytes {
		return "", fmt.Errorf("%w: Profile definition size is outside bounds", ErrInvalidManifest)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return "", fmt.Errorf("%w: Profile definition is not valid JSON: %v", ErrInvalidManifest, err)
	}
	if _, ok := value.(map[string]any); !ok {
		return "", fmt.Errorf("%w: Profile definition must be a JSON object", ErrInvalidManifest)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("canonicalize Profile definition: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
