package localrecovery

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestManifestPersistenceIsStrictAndImmutable(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "snapshots", "snapshot-a", "manifest.json")
	manifest := validManifest(t, "linux")
	digest, err := WriteManifest(manifestPath, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) != 64 {
		t.Fatalf("unexpected manifest digest %q", digest)
	}
	loaded, err := ReadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SnapshotID != manifest.SnapshotID || loaded.TreeDigest != manifest.TreeDigest {
		t.Fatalf("manifest changed after reopen: %#v", loaded)
	}
	if _, err := WriteManifest(manifestPath, manifest); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("immutable manifest replacement was not rejected: %v", err)
	}
}

func TestManifestStrictDecodingRejectsUnknownAndTrailingData(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "manifest.json")
	manifest := validManifest(t, "linux")
	data, err := encodeBounded(manifest, MaxManifestBytes, ErrInvalidManifest)
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.Replace(string(data), "{", `{"unexpected":true,`, 1)
	unknown = strings.Replace(unknown, `\"`, `"`, -1)
	if err := os.WriteFile(filePath, []byte(unknown), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(filePath); err == nil {
		t.Fatal("unknown manifest field was accepted")
	}
	if err := os.WriteFile(filePath, append(data, []byte("\n{}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(filePath); err == nil {
		t.Fatal("trailing manifest data was accepted")
	}
}

func TestManifestFileSafety(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest(t, "linux")
	realPath := filepath.Join(root, "real.json")
	if _, err := WriteManifest(realPath, manifest); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		linkPath := filepath.Join(root, "link.json")
		if err := os.Symlink(realPath, linkPath); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadManifest(linkPath); err == nil {
			t.Fatal("manifest symlink was accepted")
		}
		if err := os.Chmod(realPath, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadManifest(realPath); err == nil {
			t.Fatal("non-private manifest permissions were accepted")
		}
	}
}

func TestCatalogCreateUpdateReopenAndConflict(t *testing.T) {
	root := t.TempDir()
	store, err := OpenCatalogStore(filepath.Join(root, "local-recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return clock }
	record := validCatalogRecord(t)
	created, err := store.Create(record)
	if err != nil {
		t.Fatal(err)
	}
	if created.Revision != 1 || created.Status != SnapshotPending {
		t.Fatalf("unexpected created record: %#v", created)
	}
	if _, err := store.Create(record); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate catalog record was not rejected: %v", err)
	}

	verifiedAt := clock.Add(time.Minute)
	clock = verifiedAt
	created.Status = SnapshotVerified
	created.VerifiedAt = &verifiedAt
	updated, err := store.Update(created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.Status != SnapshotVerified {
		t.Fatalf("unexpected updated record: %#v", updated)
	}
	stale := created
	stale.Status = SnapshotInvalid
	stale.VerifiedAt = nil
	if _, err := store.Update(stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale catalog update was not rejected: %v", err)
	}

	reopened, err := OpenCatalogStore(filepath.Join(root, "local-recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get(record.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 2 || loaded.VerifiedAt == nil {
		t.Fatalf("catalog did not persist verified state: %#v", loaded)
	}
}

func TestCatalogPersistenceFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	store, err := OpenCatalogStore(filepath.Join(root, "local-recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(validCatalogRecord(t))
	if err != nil {
		t.Fatal(err)
	}
	store.write = func(string, []byte) error { return errors.New("simulated write failure") }
	candidate := created
	candidate.Status = SnapshotInvalid
	candidate.VerifiedAt = nil
	if _, err := store.Update(candidate); err == nil {
		t.Fatal("simulated persistence failure was ignored")
	}
	loaded, err := store.Get(created.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != SnapshotPending || loaded.Revision != 1 {
		t.Fatalf("in-memory state changed after failed persistence: %#v", loaded)
	}
	reopened, err := OpenCatalogStore(filepath.Join(root, "local-recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	disk, err := reopened.Get(created.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if disk.Status != SnapshotPending || disk.Revision != 1 {
		t.Fatalf("on-disk state changed after failed persistence: %#v", disk)
	}
}

func TestCatalogRejectsFutureAndDuplicateRecords(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "local-recovery.json")
	record := validCatalogRecord(t)
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	record.SchemaVersion = CatalogSchemaVersion
	record.CreatedAt = now
	record.UpdatedAt = now
	record.Revision = 1
	data, err := encodeBounded(catalogEnvelope{SchemaVersion: CatalogSchemaVersion, Records: []CatalogRecord{record, record}}, MaxCatalogBytes, ErrInvalidRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCatalogStore(filePath); err == nil {
		t.Fatal("duplicate catalog records were accepted")
	}
	data, err = encodeBounded(catalogEnvelope{SchemaVersion: CatalogSchemaVersion + 1}, MaxCatalogBytes, ErrInvalidRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCatalogStore(filePath); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("future catalog version did not fail closed: %v", err)
	}
}

func validCatalogRecord(t *testing.T) CatalogRecord {
	t.Helper()
	manifest := validManifest(t, "linux")
	manifestDigest, err := ComputeManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return CatalogRecord{
		SnapshotID:      manifest.SnapshotID,
		SourceProfileID: manifest.SourceProfileID,
		ManifestRef:     ExpectedManifestRef(manifest.SnapshotID),
		Status:          SnapshotPending,
		ManifestDigest:  manifestDigest,
		TreeDigest:      manifest.TreeDigest,
		FileCount:       manifest.FileCount,
		TotalBytes:      manifest.TotalBytes,
	}
}
