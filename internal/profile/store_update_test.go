package profile

import (
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestUpdatePreservesCreatedAtAndNormalizesTags(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(domain.Profile{Name: "Original", Tags: []string{"B", "a"}})
	if err != nil {
		t.Fatal(err)
	}
	created.Name = "Updated"
	created.Tags = []string{"A", "a", " C "}
	updated, err := store.Update(created)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatal("createdAt changed during update")
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "A" || updated.Tags[1] != "C" {
		t.Fatalf("unexpected tags: %#v", updated.Tags)
	}
}
