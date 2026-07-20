package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func inventoryRecord(id string, now time.Time) Record {
	return NewCompatibilityRecord(id, "profiles/"+id, now)
}

func TestInventoryReportsPresentMissingOrphanAndUnsafeEntries(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	present := filepath.Join(root, "profiles", "profile-a")
	if err := os.MkdirAll(present, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(present, "Preferences"), []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "profiles", "orphan"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profiles", "unexpected-file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	scanner, err := NewInventoryScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	scanner.Now = func() time.Time { return now }
	report, err := scanner.Scan(context.Background(), []Record{inventoryRecord("profile-a", now), inventoryRecord("profile-b", now)})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Profiles) != 2 {
		t.Fatalf("unexpected profiles: %+v", report.Profiles)
	}
	byID := map[string]ProfileStorage{}
	for _, profile := range report.Profiles {
		byID[profile.ProfileID] = profile
	}
	if byID["profile-a"].Status != InventoryPresent || byID["profile-a"].Summary.Files != 1 || byID["profile-a"].Summary.Bytes != 3 {
		t.Fatalf("unexpected present result: %+v", byID["profile-a"])
	}
	if byID["profile-b"].Status != InventoryMissing || byID["profile-b"].ReasonCode != "managed-directory-missing" {
		t.Fatalf("unexpected missing result: %+v", byID["profile-b"])
	}
	if len(report.Orphans) != 1 || report.Orphans[0].RelativePath != "profiles/orphan" {
		t.Fatalf("unexpected orphans: %+v", report.Orphans)
	}
	if len(report.Unsafe) != 1 || report.Unsafe[0].RelativePath != "profiles/unexpected-file" {
		t.Fatalf("unexpected unsafe entries: %+v", report.Unsafe)
	}
	if report.ManagedRoot != "." {
		t.Fatalf("absolute root leaked: %q", report.ManagedRoot)
	}
}

func TestInventoryRejectsDuplicateManagedDirectoryIdentity(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	first := inventoryRecord("profile-a", now)
	second := inventoryRecord("profile-b", now)
	second.ManagedDir = first.ManagedDir
	scanner, err := NewInventoryScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Scan(context.Background(), []Record{first, second}); err == nil || !strings.Contains(err.Error(), "duplicate managed") {
		t.Fatalf("expected duplicate managed directory rejection, got %v", err)
	}
}

func TestInventoryDoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows reparse behavior is covered by platform CI")
	}
	root := t.TempDir()
	outside := t.TempDir()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	profileDir := filepath.Join(root, "profiles", "profile-a")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("do-not-read"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(profileDir, "linked-secret")); err != nil {
		t.Fatal(err)
	}

	scanner, err := NewInventoryScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := scanner.Scan(context.Background(), []Record{inventoryRecord("profile-a", now)})
	if err != nil {
		t.Fatal(err)
	}
	profile := report.Profiles[0]
	if profile.Status != InventoryUnsafe || profile.ReasonCode != "unsafe-link-or-reparse" {
		t.Fatalf("expected unsafe link report, got %+v", profile)
	}
	if profile.Summary.Bytes != 0 {
		t.Fatalf("symlink target was counted/read: %+v", profile.Summary)
	}
}

func TestInventoryBoundsAndCancellationAreIncompleteNotSuccessful(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	profileDir := filepath.Join(root, "profiles", "profile-a")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "one"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "two"), []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}

	scanner, err := NewInventoryScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	scanner.MaxFiles = 1
	report, err := scanner.Scan(context.Background(), []Record{inventoryRecord("profile-a", now)})
	if err != nil {
		t.Fatal(err)
	}
	if report.Profiles[0].Status != InventoryIncomplete || report.Profiles[0].ReasonCode != "inventory-bound-exceeded" || !report.Incomplete {
		t.Fatalf("bound exhaustion reported as complete: %+v", report)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scanner.MaxFiles = 100
	cancelled, err := scanner.Scan(ctx, []Record{inventoryRecord("profile-a", now)})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Profiles[0].Status != InventoryIncomplete || cancelled.Profiles[0].ReasonCode != "inventory-cancelled" || !cancelled.Incomplete {
		t.Fatalf("cancellation reported as complete: %+v", cancelled)
	}
}

func TestInventoryRejectsSymlinkManagedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows reparse behavior is covered by platform CI")
	}
	realRoot := t.TempDir()
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "managed")
	if err := os.Symlink(realRoot, link); err != nil {
		t.Fatal(err)
	}
	scanner, err := NewInventoryScanner(link)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Scan(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "managed data root") {
		t.Fatalf("expected symlink root rejection, got %v", err)
	}
}
