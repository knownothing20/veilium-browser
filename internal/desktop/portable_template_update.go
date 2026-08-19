package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

// PortableTemplateUpdateRequest changes only reusable non-secret template metadata.
// Kernel, adapter, route, credential requirement, and fingerprint settings remain
// the reviewed values already stored in the template catalog.
type PortableTemplateUpdateRequest struct {
	TemplateID  string   `json:"templateId"`
	Name        string   `json:"name"`
	ProfileName string   `json:"profileName"`
	Group       string   `json:"group"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tags"`
}

func (s *Service) UpdatePortableTemplate(request PortableTemplateUpdateRequest) (portableprofile.Template, error) {
	templateID := strings.TrimSpace(request.TemplateID)
	if templateID == "" {
		return portableprofile.Template{}, fmt.Errorf("portable template ID is required")
	}

	catalog, err := portableprofile.LoadTemplates(s.portableTemplatePath())
	if err != nil {
		return portableprofile.Template{}, err
	}

	index := -1
	for position := range catalog.Templates {
		if catalog.Templates[position].ID == templateID {
			index = position
			break
		}
	}
	if index < 0 {
		return portableprofile.Template{}, fmt.Errorf("portable template %q was not found", templateID)
	}

	name := strings.TrimSpace(request.Name)
	if portableTemplateNameConflict(catalog.Templates, templateID, name) {
		return portableprofile.Template{}, fmt.Errorf("template name %q already exists", name)
	}

	updated := catalog.Templates[index]
	updated.Name = name
	updated.Payload.Name = strings.TrimSpace(request.ProfileName)
	updated.Payload.Group = strings.TrimSpace(request.Group)
	updated.Payload.Notes = strings.TrimSpace(request.Notes)
	updated.Payload.Tags = normalizePortableTemplateTags(request.Tags)
	updated.Payload.IdentityMode = portableprofile.IdentityNew
	updated.Payload.Fingerprint.Seed = ""

	now := time.Now().UTC()
	if !now.After(updated.UpdatedAt) {
		now = updated.UpdatedAt.Add(time.Nanosecond)
	}
	updated.UpdatedAt = now
	catalog.Templates[index] = updated

	if err := portableprofile.SaveTemplates(s.portableTemplatePath(), catalog); err != nil {
		return portableprofile.Template{}, fmt.Errorf("update portable template: %w", err)
	}
	return updated, nil
}

func portableTemplateNameConflict(templates []portableprofile.Template, exceptID, name string) bool {
	name = strings.TrimSpace(name)
	for _, item := range templates {
		if item.ID != strings.TrimSpace(exceptID) && strings.EqualFold(strings.TrimSpace(item.Name), name) {
			return true
		}
	}
	return false
}

func normalizePortableTemplateTags(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}
