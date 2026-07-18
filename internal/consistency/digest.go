package consistency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

type DigestInput struct {
	Profile         domain.Profile
	Capabilities    fingerprint.Capabilities
	BinaryIdentity  kernel.ProviderBinaryIdentity
	RuntimeOS       string
	RuntimeArch     string
	HarnessRevision string
}

type digestRecord struct {
	RulesRevision    string                   `json:"rulesRevision"`
	ProfileID        string                   `json:"profileId"`
	KernelID         string                   `json:"kernelId"`
	Provider         string                   `json:"provider"`
	ProviderRevision int                      `json:"providerRevision"`
	ProviderTrust    fingerprint.TrustStatus  `json:"providerTrust"`
	KernelVersion    string                   `json:"kernelVersion"`
	BinarySHA256     string                   `json:"binarySha256"`
	BinarySize       int64                    `json:"binarySize"`
	BinaryIntegrity  string                   `json:"binaryIntegrity"`
	RuntimeOS        string                   `json:"runtimeOs"`
	RuntimeArch      string                   `json:"runtimeArch"`
	HarnessRevision  string                   `json:"harnessRevision"`
	Fingerprint      domain.FingerprintConfig `json:"fingerprint"`
	Window           WindowSpec               `json:"window"`
}

func InputDigest(input DigestInput) (string, error) {
	if strings.TrimSpace(input.Profile.ID) == "" {
		return "", fmt.Errorf("profile id is required for consistency digest")
	}
	window, err := EffectiveWindow(input.Profile)
	if err != nil {
		return "", err
	}
	record := digestRecord{
		RulesRevision:    RulesRevision,
		ProfileID:        input.Profile.ID,
		KernelID:         input.Profile.Kernel.ID,
		Provider:         input.Capabilities.Provider,
		ProviderRevision: input.Capabilities.Revision,
		ProviderTrust:    input.Capabilities.TrustStatus,
		KernelVersion:    input.Profile.Kernel.Version,
		BinarySHA256:     input.BinaryIdentity.ExecutableSHA256,
		BinarySize:       input.BinaryIdentity.ExecutableSize,
		BinaryIntegrity:  input.BinaryIdentity.IntegrityStatus,
		RuntimeOS:        strings.ToLower(strings.TrimSpace(input.RuntimeOS)),
		RuntimeArch:      strings.ToLower(strings.TrimSpace(input.RuntimeArch)),
		HarnessRevision:  strings.TrimSpace(input.HarnessRevision),
		Fingerprint:      input.Profile.Fingerprint,
		Window:           window,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("encode consistency digest input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
