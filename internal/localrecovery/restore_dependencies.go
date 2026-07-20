package localrecovery

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/credential"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
)

func (e *RestoreExecutor) resolveDependencies(manifest LocalSnapshotManifest, selection RestoreDependencySelection) (RestoreDependencyResolution, domain.KernelRef, domain.ProxyConfig) {
	resolution := RestoreDependencyResolution{}
	kernelRef := domain.KernelRef{
		Provider: manifest.Dependencies.Kernel.ProviderID,
		Version:  manifest.Dependencies.Kernel.BrowserVersion,
	}
	proxyConfig := domain.ProxyConfig{}

	resolution.Kernel, kernelRef = e.resolveKernel(manifest.Dependencies.Kernel, selection.KernelID)
	if requirement := manifest.Dependencies.Adapter; requirement != nil {
		resolved, adapterID := e.resolveAdapter(*requirement, selection.AdapterID)
		resolution.Adapter = &resolved
		proxyConfig.AdapterRef = adapterID
	}
	if requirement := manifest.Dependencies.Credential; requirement != nil {
		resolved, credentialID := e.resolveCredential(*requirement, selection.CredentialID)
		resolution.Credential = &resolved
		proxyConfig.CredentialRef = credentialID
	}
	resolution.Limitations = dependencyLimitations(resolution)
	return resolution, kernelRef, proxyConfig
}

func (e *RestoreExecutor) resolveKernel(requirement KernelRequirement, selectedID string) (ResolvedDependency, domain.KernelRef) {
	result := ResolvedDependency{Kind: "kernel", Status: DependencyUserActionRequired, ReasonCode: "kernel-selection-required"}
	fallback := domain.KernelRef{Provider: requirement.ProviderID, Version: requirement.BrowserVersion}
	if selectedID == "" {
		return result, fallback
	}
	record, err := e.kernels.Verify(selectedID)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			result.Status = DependencyMissing
			result.ReasonCode = "kernel-record-missing"
		} else {
			result.Status = DependencyUnsupported
			result.ReasonCode = "kernel-record-unavailable"
		}
		return result, fallback
	}
	result.RecordID = record.ID
	if !kernelRequirementMatches(requirement, record) {
		result.Status = DependencyIncompatible
		result.ReasonCode = "kernel-requirement-mismatch"
		return result, fallback
	}
	result.Status = DependencyResolved
	result.ReasonCode = "kernel-resolved"
	return result, domain.KernelRef{
		ID:         record.ID,
		Provider:   record.Provider,
		Version:    record.Version,
		Executable: record.Executable,
	}
}

func kernelRequirementMatches(requirement KernelRequirement, record kernel.Record) bool {
	if record.Status != kernel.StatusVerified || record.Provider != requirement.ProviderID || record.Version != requirement.BrowserVersion {
		return false
	}
	if requirement.OperatingSystem != runtime.GOOS || requirement.Architecture != runtime.GOARCH {
		return false
	}
	if requirement.ExecutableSHA256 != "" && !strings.EqualFold(record.SHA256, requirement.ExecutableSHA256) {
		return false
	}
	if requirement.PackageTreeSHA256 != "" && !strings.EqualFold(record.PackageTreeSHA256, requirement.PackageTreeSHA256) {
		return false
	}
	capabilities, err := fingerprint.For(record.Provider, record.Version)
	if err != nil {
		return false
	}
	if requirement.TrustRequirement == "reviewed" && !capabilities.IsReviewed() {
		return false
	}
	// The existing fingerprint package does not expose a reviewed, stable
	// capability-ID lookup contract for restore. Do not guess. A requirement
	// with explicit capability IDs remains unresolved until the later Desktop
	// validation boundary can confirm it through the current Provider contract.
	if len(requirement.Capabilities) != 0 {
		return false
	}
	return true
}

func (e *RestoreExecutor) resolveAdapter(requirement AdapterRequirement, selectedID string) (ResolvedDependency, string) {
	result := ResolvedDependency{Kind: "adapter", Status: DependencyUserActionRequired, ReasonCode: "adapter-selection-required"}
	if selectedID == "" {
		return result, ""
	}
	record, err := e.adapters.Verify(selectedID)
	if err != nil {
		if errors.Is(err, adapter.ErrNotFound) {
			result.Status = DependencyMissing
			result.ReasonCode = "adapter-record-missing"
		} else {
			result.Status = DependencyUnsupported
			result.ReasonCode = "adapter-record-unavailable"
		}
		return result, ""
	}
	result.RecordID = record.ID
	if !adapterRequirementMatches(requirement, record) {
		result.Status = DependencyIncompatible
		result.ReasonCode = "adapter-requirement-mismatch"
		return result, ""
	}
	result.Status = DependencyResolved
	result.ReasonCode = "adapter-resolved"
	return result, record.ID
}

func adapterRequirementMatches(requirement AdapterRequirement, record adapter.Record) bool {
	if record.Status != adapter.StatusVerified || record.Kind != requirement.Kind || record.Version != requirement.Version || record.Official != requirement.Official {
		return false
	}
	if requirement.OperatingSystem != runtime.GOOS || requirement.Architecture != runtime.GOARCH {
		return false
	}
	if requirement.ExecutableSHA256 != "" && !strings.EqualFold(record.SHA256, requirement.ExecutableSHA256) {
		return false
	}
	if requirement.Official {
		if record.OfficialPlatform != "" && record.OfficialPlatform != runtime.GOOS {
			return false
		}
		if record.OfficialArch != "" && record.OfficialArch != runtime.GOARCH {
			return false
		}
	}
	for _, protocol := range record.Protocols {
		if strings.EqualFold(protocol, requirement.Scheme) {
			return true
		}
	}
	return false
}

func (e *RestoreExecutor) resolveCredential(requirement CredentialRequirement, selectedID string) (ResolvedDependency, string) {
	result := ResolvedDependency{Kind: "credential", Status: DependencyUserActionRequired, ReasonCode: "credential-selection-required"}
	if selectedID == "" {
		return result, ""
	}
	record, err := e.credentials.Get(selectedID)
	if err != nil {
		if errors.Is(err, credential.ErrNotFound) {
			result.Status = DependencyMissing
			result.ReasonCode = "credential-record-missing"
		} else {
			result.Status = DependencyUnsupported
			result.ReasonCode = "credential-record-unavailable"
		}
		return result, ""
	}
	result.RecordID = record.ID
	if requirement.RequiresUsername && strings.TrimSpace(record.Username) == "" {
		result.Status = DependencyIncompatible
		result.ReasonCode = "credential-username-missing"
		return result, ""
	}
	if requirement.RequiresSecret {
		// Metadata existence cannot prove that current vault material exists.
		// Stage 3 never reads or copies the secret. Keep the dependency limited
		// until the current Desktop validation path can verify the vault entry.
		result.Status = DependencyUserActionRequired
		result.ReasonCode = "credential-secret-verification-required"
		return result, ""
	}
	result.Status = DependencyResolved
	result.ReasonCode = "credential-metadata-resolved"
	return result, record.ID
}

func dependencyLimitations(resolution RestoreDependencyResolution) []string {
	limitations := []string{"restore-current-validation-required", "restore-evidence-not-applicable", "restore-lifecycle-draft"}
	for _, dependency := range dependencyResolutionItems(resolution) {
		if dependency.Status != DependencyResolved {
			limitations = append(limitations, dependency.ReasonCode)
		}
	}
	sort.Strings(limitations)
	return uniqueStrings(limitations)
}

func dependencyResolutionItems(resolution RestoreDependencyResolution) []ResolvedDependency {
	items := []ResolvedDependency{resolution.Kernel}
	if resolution.Adapter != nil {
		items = append(items, *resolution.Adapter)
	}
	if resolution.Credential != nil {
		items = append(items, *resolution.Credential)
	}
	return items
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || (len(result) > 0 && result[len(result)-1] == value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func (r RestoreDependencyResolution) Validate() error {
	for _, item := range dependencyResolutionItems(r) {
		if item.Kind == "" || item.Status == "" || item.ReasonCode == "" {
			return fmt.Errorf("%w: incomplete restore dependency resolution", ErrDependencyMismatch)
		}
	}
	return nil
}
