package desktop

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

func TestUpdatePortableTemplatePreservesIdentityAndDependencies(t *testing.T) {
	service := &Service{dataRoot: t.TempDir()}
	createdAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	template := portableTemplateFixture(t, "Original template", createdAt)
	catalog := portableprofile.TemplateCatalog{
		SchemaVersion: portableprofile.SchemaVersion,
		Kind:          portableprofile.TemplateCatalogKind,
		Templates:     []portableprofile.Template{template},
	}
	if err := portableprofile.SaveTemplates(service.portableTemplatePath(), catalog); err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdatePortableTemplate(PortableTemplateUpdateRequest{
		TemplateID:  template.ID,
		Name:        "  Updated template  ",
		ProfileName: "  Updated Profile  ",
		Group:       "  Research  ",
		Notes:       "  Reusable non-secret defaults  ",
		Tags:        []string{"Beta", "alpha", "ALPHA", ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != template.ID || !updated.CreatedAt.Equal(template.CreatedAt) {
		t.Fatalf("template identity changed: %#v", updated)
	}
	if !updated.UpdatedAt.After(template.UpdatedAt) {
		t.Fatalf("updated timestamp did not advance: %s <= %s", updated.UpdatedAt, template.UpdatedAt)
	}
	if updated.Name != "Updated template" || updated.Payload.Name != "Updated Profile" {
		t.Fatalf("updated names = %q / %q", updated.Name, updated.Payload.Name)
	}
	if updated.Payload.Group != "Research" || updated.Payload.Notes != "Reusable non-secret defaults" {
		t.Fatalf("updated metadata = %#v", updated.Payload)
	}
	if want := []string{"alpha", "Beta"}; !reflect.DeepEqual(updated.Payload.Tags, want) {
		t.Fatalf("updated tags = %#v, want %#v", updated.Payload.Tags, want)
	}
	if updated.Payload.IdentityMode != portableprofile.IdentityNew || updated.Payload.Fingerprint.Seed != "" {
		t.Fatal("updated template retained reusable identity material")
	}
	if updated.Payload.Kernel != template.Payload.Kernel || !reflect.DeepEqual(updated.Payload.Adapter, template.Payload.Adapter) {
		t.Fatal("template dependency requirements changed during metadata update")
	}
	if updated.Payload.ProxyURL != template.Payload.ProxyURL || updated.Payload.CredentialRequired != template.Payload.CredentialRequired {
		t.Fatal("template route requirements changed during metadata update")
	}

	reloaded, err := portableprofile.LoadTemplates(service.portableTemplatePath())
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Templates) != 1 || !reflect.DeepEqual(reloaded.Templates[0], updated) {
		t.Fatalf("persisted template = %#v, want %#v", reloaded.Templates, updated)
	}
}

func TestUpdatePortableTemplateRejectsNameConflictWithoutMutation(t *testing.T) {
	service := &Service{dataRoot: t.TempDir()}
	first := portableTemplateFixture(t, "First", time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC))
	second := portableTemplateFixture(t, "Second", time.Date(2026, 7, 22, 13, 0, 0, 0, time.UTC))
	catalog := portableprofile.TemplateCatalog{
		SchemaVersion: portableprofile.SchemaVersion,
		Kind:          portableprofile.TemplateCatalogKind,
		Templates:     []portableprofile.Template{first, second},
	}
	if err := portableprofile.SaveTemplates(service.portableTemplatePath(), catalog); err != nil {
		t.Fatal(err)
	}

	_, err := service.UpdatePortableTemplate(PortableTemplateUpdateRequest{
		TemplateID:  second.ID,
		Name:        "FIRST",
		ProfileName: second.Payload.Name,
		Group:       second.Payload.Group,
		Notes:       second.Payload.Notes,
		Tags:        second.Payload.Tags,
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected name conflict, got %v", err)
	}

	reloaded, err := portableprofile.LoadTemplates(service.portableTemplatePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range reloaded.Templates {
		if item.ID == second.ID && !reflect.DeepEqual(item, second) {
			t.Fatalf("conflicting update mutated template: %#v", item)
		}
	}
}

func TestUpdatePortableTemplateRejectsTooManyTagsWithoutMutation(t *testing.T) {
	service := &Service{dataRoot: t.TempDir()}
	template := portableTemplateFixture(t, "Bounded", time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC))
	catalog := portableprofile.TemplateCatalog{
		SchemaVersion: portableprofile.SchemaVersion,
		Kind:          portableprofile.TemplateCatalogKind,
		Templates:     []portableprofile.Template{template},
	}
	if err := portableprofile.SaveTemplates(service.portableTemplatePath(), catalog); err != nil {
		t.Fatal(err)
	}
	tags := make([]string, portableprofile.MaxTags+1)
	for index := range tags {
		tags[index] = "tag-" + time.Unix(int64(index), 0).UTC().Format("150405")
	}

	if _, err := service.UpdatePortableTemplate(PortableTemplateUpdateRequest{
		TemplateID:  template.ID,
		Name:        template.Name,
		ProfileName: template.Payload.Name,
		Tags:        tags,
	}); err == nil {
		t.Fatal("over-bound template tags were accepted")
	}
	reloaded, err := portableprofile.LoadTemplates(service.portableTemplatePath())
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Templates) != 1 || !reflect.DeepEqual(reloaded.Templates[0], template) {
		t.Fatal("failed bounded update changed the persisted template")
	}
}

func portableTemplateFixture(t *testing.T, name string, now time.Time) portableprofile.Template {
	t.Helper()
	template, err := portableprofile.NewTemplate(name, portableprofile.Payload{
		Name:               name + " Profile",
		Group:              "Original group",
		Notes:              "Original notes",
		Tags:               []string{"original"},
		Fingerprint:        domain.FingerprintConfig{Seed: "must-be-removed", Platform: "windows", Brand: "Chromium"},
		ProxyURL:           "socks5://proxy.example:1080",
		CredentialRequired: true,
		Kernel: portableprofile.KernelRequirement{
			Provider:  "custom-chromium",
			Version:   "148.0.0",
			SHA256:    strings.Repeat("a", 64),
			SizeBytes: 1234,
		},
		Adapter: &portableprofile.AdapterRequirement{
			Kind:      "xray",
			Version:   "26.3.27",
			SHA256:    strings.Repeat("b", 64),
			SizeBytes: 5678,
		},
		IdentityMode: portableprofile.IdentityPreserve,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return template
}
