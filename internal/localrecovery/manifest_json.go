package localrecovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func ValidateManifest(manifest LocalSnapshotManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	if err := validateDependencyPlatform(manifest); err != nil {
		return err
	}
	return validateManifestResourceBounds(manifest)
}

func (m LocalSnapshotManifest) MarshalJSON() ([]byte, error) {
	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	type manifestAlias LocalSnapshotManifest
	data, err := json.Marshal(manifestAlias(m))
	if err != nil {
		return nil, fmt.Errorf("encode local recovery manifest: %w", err)
	}
	return data, nil
}

func (m *LocalSnapshotManifest) UnmarshalJSON(data []byte) error {
	if int64(len(data)) > MaxManifestBytes {
		return fmt.Errorf("%w: encoded manifest exceeds %d bytes", ErrInvalidManifest, MaxManifestBytes)
	}
	type manifestAlias LocalSnapshotManifest
	var decoded manifestAlias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return fmt.Errorf("decode local recovery manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%w: manifest contains trailing data", ErrInvalidManifest)
		}
		return fmt.Errorf("decode trailing local recovery manifest data: %w", err)
	}
	candidate := LocalSnapshotManifest(decoded)
	if err := ValidateManifest(candidate); err != nil {
		return err
	}
	*m = candidate
	return nil
}
