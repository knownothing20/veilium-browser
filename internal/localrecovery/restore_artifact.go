package localrecovery

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

type verifiedRestoreSource struct {
	Catalog         CatalogRecord
	Manifest        LocalSnapshotManifest
	ManifestDigest  string
	SnapshotPath    string
	BrowserDataPath string
	ProfileData     []byte
	SourceProfile   domain.Profile
	Plan            snapshotPlan
}

func (e *RestoreExecutor) verifyRestoreSource(ctx context.Context, request RestoreRequest) (verifiedRestoreSource, error) {
	if err := e.checkCancellation(ctx, request.OperationID); err != nil {
		return verifiedRestoreSource{}, err
	}
	catalog, err := e.catalog.Get(request.SnapshotID)
	if err != nil {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot catalog: %v", ErrSnapshotUnavailable, err)
	}
	if catalog.Status != SnapshotVerified {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot status is %s", ErrSnapshotUnavailable, catalog.Status)
	}
	snapshotPath := snapshotFinalPath(e.recoveryRoot, request.SnapshotID)
	if err := inspectDirectoryTree(filepath.Join(e.recoveryRoot, snapshotFinalDirectory), snapshotPath); err != nil {
		return verifiedRestoreSource{}, err
	}
	manifestPath := filepath.Join(snapshotPath, manifestFileName)
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return verifiedRestoreSource{}, err
	}
	if manifest.SnapshotID != request.SnapshotID || manifest.SourceProfileID != catalog.SourceProfileID {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot catalog and manifest disagree", ErrSnapshotUnavailable)
	}
	if manifest.SourceOS != runtime.GOOS || manifest.SourceArch != runtime.GOARCH || manifest.Portability != PortabilitySameUserSameMachine {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot is not applicable to this machine", ErrSnapshotUnavailable)
	}
	manifestDigest, err := ComputeManifestDigest(manifest)
	if err != nil {
		return verifiedRestoreSource{}, err
	}
	if manifestDigest != catalog.ManifestDigest || manifest.TreeDigest != catalog.TreeDigest || manifest.FileCount != catalog.FileCount || manifest.TotalBytes != catalog.TotalBytes {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot catalog identity is stale or contradictory", ErrSnapshotUnavailable)
	}
	verified, err := verifyStagedSnapshot(snapshotPath, manifest)
	if err != nil {
		return verifiedRestoreSource{}, fmt.Errorf("%w: snapshot verification failed: %v", ErrSnapshotUnavailable, err)
	}
	profileData, err := readBoundedFile(filepath.Join(snapshotPath, profileDefinitionName), MaxProfileDefinitionBytes)
	if err != nil {
		return verifiedRestoreSource{}, err
	}
	sourceProfile, err := decodeSnapshotProfile(profileData, verified)
	if err != nil {
		return verifiedRestoreSource{}, err
	}
	plan, err := e.planRestoreFiles(ctx, request.OperationID, snapshotPath, verified)
	if err != nil {
		return verifiedRestoreSource{}, err
	}
	return verifiedRestoreSource{
		Catalog:         catalog,
		Manifest:        verified,
		ManifestDigest:  manifestDigest,
		SnapshotPath:    snapshotPath,
		BrowserDataPath: filepath.Join(snapshotPath, browserDataDirectory),
		ProfileData:     profileData,
		SourceProfile:   sourceProfile,
		Plan:            plan,
	}, nil
}

func (e *RestoreExecutor) planRestoreFiles(ctx context.Context, operationID, snapshotPath string, manifest LocalSnapshotManifest) (snapshotPlan, error) {
	browserRoot := filepath.Join(snapshotPath, browserDataDirectory)
	plan := snapshotPlan{SourceRoot: browserRoot, Files: make([]plannedSnapshotFile, 0, len(manifest.Files)), TotalBytes: manifest.TotalBytes}
	for _, entry := range manifest.Files {
		if err := e.checkCancellation(ctx, operationID); err != nil {
			return snapshotPlan{}, err
		}
		sourcePath := filepath.Join(browserRoot, filepath.FromSlash(entry.Path))
		if !pathContainedBy(sourcePath, browserRoot) {
			return snapshotPlan{}, fmt.Errorf("%w: snapshot entry escapes browser-data", ErrSnapshotUnavailable)
		}
		if err := inspectDirectoryTree(browserRoot, filepath.Dir(sourcePath)); err != nil {
			return snapshotPlan{}, err
		}
		token, err := inspectRegularPath(sourcePath)
		if err != nil {
			return snapshotPlan{}, err
		}
		if token.Size != entry.Size {
			return snapshotPlan{}, fmt.Errorf("%w: snapshot file size changed for %q", ErrSnapshotUnavailable, entry.Path)
		}
		digest, size, err := hashStableRegularFile(sourcePath)
		if err != nil {
			return snapshotPlan{}, err
		}
		if size != entry.Size || digest != entry.SHA256 {
			return snapshotPlan{}, fmt.Errorf("%w: snapshot file digest changed for %q", ErrSnapshotUnavailable, entry.Path)
		}
		plan.Files = append(plan.Files, plannedSnapshotFile{
			RelativePath: entry.Path,
			SourcePath:   sourcePath,
			Size:         entry.Size,
			Token:        token,
		})
	}
	required, err := requiredSnapshotSpace(plan.TotalBytes, len(manifest.Files)*128)
	if err != nil {
		return snapshotPlan{}, err
	}
	available, err := e.space(e.profilesRoot)
	if err != nil {
		return snapshotPlan{}, err
	}
	if available < required {
		return snapshotPlan{}, fmt.Errorf("%w: restore requires %d bytes but only %d are available", ErrInsufficientSpace, required, available)
	}
	return plan, nil
}
