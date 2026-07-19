package networkevidence

import (
	"fmt"
	"strings"
	"time"
)

const BrowserSubmissionSchemaVersion = 1

type BrowserSubmission struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Observations  []BrowserObservation `json:"observations"`
	Limitations   []string             `json:"limitations,omitempty"`
}

type BrowserObservation struct {
	ProbeID      string            `json:"probeId"`
	ProbeRevision int              `json:"probeRevision"`
	ProbeKind    ProbeKind         `json:"probeKind"`
	Status       ObservationStatus `json:"status"`
	Values       []string          `json:"values,omitempty"`
	ReasonCode   string            `json:"reasonCode,omitempty"`
	Detail       string            `json:"detail,omitempty"`
}

func (submission BrowserSubmission) Validate(set ProbeSet) error {
	if submission.SchemaVersion != BrowserSubmissionSchemaVersion {
		return fmt.Errorf("unsupported browser network submission schema %d", submission.SchemaVersion)
	}
	if err := set.Validate(); err != nil {
		return err
	}
	if len(submission.Observations) < 1 || len(submission.Observations) > len(set.Definitions) {
		return fmt.Errorf("browser network submission has an invalid observation count")
	}
	definitions := make(map[string]ProbeDefinition, len(set.Definitions))
	for _, definition := range set.Definitions {
		definitions[probeKey(definition.ID, definition.Revision)] = definition
	}
	seen := make(map[string]struct{}, len(submission.Observations))
	now := time.Now().UTC()
	for index, browserObservation := range submission.Observations {
		key := probeKey(browserObservation.ProbeID, browserObservation.ProbeRevision)
		definition, exists := definitions[key]
		if !exists || definition.Kind != browserObservation.ProbeKind {
			return fmt.Errorf("browser observation %d is not part of the selected probe set", index)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate browser network observation %q", key)
		}
		seen[key] = struct{}{}
		observation := browserObservation.Observation(now)
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("browser observation %d: %w", index, err)
		}
	}
	if len(submission.Limitations) > maxLimitations {
		return fmt.Errorf("browser network submission has too many limitations")
	}
	for _, limitation := range submission.Limitations {
		if len(strings.TrimSpace(limitation)) > 512 {
			return fmt.Errorf("browser network limitation is too long")
		}
	}
	return nil
}

func (observation BrowserObservation) Observation(collectedAt time.Time) Observation {
	return Observation{
		ID:            string(observation.ProbeKind),
		ProbeKind:     observation.ProbeKind,
		ProbeID:       strings.TrimSpace(observation.ProbeID),
		ProbeRevision: observation.ProbeRevision,
		Status:        observation.Status,
		Values:        sortedUnique(observation.Values),
		ReasonCode:    strings.TrimSpace(observation.ReasonCode),
		Detail:        strings.TrimSpace(observation.Detail),
		CollectedAt:   collectedAt.UTC(),
	}
}

func NormalizeBrowserSubmission(submission BrowserSubmission) BrowserSubmission {
	for index := range submission.Observations {
		submission.Observations[index].Values = sortedUnique(submission.Observations[index].Values)
	}
	submission.Limitations = sortedUnique(submission.Limitations)
	return submission
}

func probeKey(id string, revision int) string {
	return fmt.Sprintf("%s@%d", strings.TrimSpace(id), revision)
}
