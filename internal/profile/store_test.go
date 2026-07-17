package profile

import (
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestStorePersistsProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(domain.Profile{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("profile ID was not generated")
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "A" {
		t.Fatalf("unexpected profile: %+v", loaded)
	}
}
