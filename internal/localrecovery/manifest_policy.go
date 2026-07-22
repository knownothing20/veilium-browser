package localrecovery

import "fmt"

const estimatedManifestBaseBytes int64 = 96 << 10

func validateManifestResourceBounds(manifest LocalSnapshotManifest) error {
	estimated := estimatedManifestBaseBytes
	for _, entry := range manifest.Files {
		estimated += int64(len(entry.Path) + len(entry.SHA256) + 96)
		if estimated > MaxManifestBytes {
			return fmt.Errorf("%w: manifest entry set exceeds the encoded-size budget", ErrInvalidManifest)
		}
	}
	return nil
}

func validateDependencyPlatform(manifest LocalSnapshotManifest) error {
	kernel := manifest.Dependencies.Kernel
	if kernel.OperatingSystem != manifest.SourceOS || kernel.Architecture != manifest.SourceArch {
		return fmt.Errorf("%w: kernel requirement platform contradicts the snapshot source", ErrInvalidManifest)
	}
	if adapter := manifest.Dependencies.Adapter; adapter != nil {
		if adapter.OperatingSystem != manifest.SourceOS || adapter.Architecture != manifest.SourceArch {
			return fmt.Errorf("%w: adapter requirement platform contradicts the snapshot source", ErrInvalidManifest)
		}
	}
	return nil
}
