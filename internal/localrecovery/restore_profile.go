package localrecovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func decodeSnapshotProfile(data []byte, manifest LocalSnapshotManifest) (domain.Profile, error) {
	if err := validateProfileDefinitionExclusions(data); err != nil {
		return domain.Profile{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var source domain.Profile
	if err := decoder.Decode(&source); err != nil {
		return domain.Profile{}, fmt.Errorf("%w: decode snapshot Profile definition: %v", ErrInvalidManifest, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.Profile{}, fmt.Errorf("%w: snapshot Profile definition contains trailing data", ErrInvalidManifest)
	}
	if strings.TrimSpace(source.ID) != manifest.SourceProfileID {
		return domain.Profile{}, fmt.Errorf("%w: snapshot Profile id contradicts the manifest", ErrInvalidManifest)
	}
	if strings.TrimSpace(source.Name) == "" {
		return domain.Profile{}, fmt.Errorf("%w: snapshot Profile name is missing", ErrInvalidManifest)
	}
	return source, nil
}

func buildRestoredProfile(source domain.Profile, request RestoreRequest, destinationID, seed, userDataDir string, kernelRef domain.KernelRef, resolvedProxy domain.ProxyConfig) domain.Profile {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = strings.TrimSpace(source.Name) + " Restored"
	}
	restored := source
	restored.ID = destinationID
	restored.Name = name
	restored.UserDataDir = filepath.Clean(userDataDir)
	restored.CreatedAt = time.Time{}
	restored.UpdatedAt = time.Time{}
	restored.Fingerprint.Seed = seed
	restored.Kernel = kernelRef
	restored.Proxy.CredentialRef = resolvedProxy.CredentialRef
	restored.Proxy.AdapterRef = resolvedProxy.AdapterRef
	return restored
}
