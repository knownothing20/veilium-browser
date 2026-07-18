package fingerprint

import "fmt"

func ValidateReplacement(current, candidate ProviderDefinition) error {
	if err := ValidateDefinition(current); err != nil {
		return fmt.Errorf("current provider definition: %w", err)
	}
	if err := ValidateDefinition(candidate); err != nil {
		return fmt.Errorf("candidate provider definition: %w", err)
	}
	if candidate.TrustStatus == TrustDisabled || candidate.TrustStatus == TrustInvalid {
		return fmt.Errorf("candidate provider %q is not launchable", candidate.ID)
	}

	if candidate.ID == current.ID {
		if candidate.Revision <= current.Revision {
			return fmt.Errorf("candidate provider revision must advance from %d", current.Revision)
		}
		if current.TrustStatus == TrustReviewed && candidate.TrustStatus == TrustReviewed {
			if candidate.SourceURL != current.SourceURL || candidate.LicenseSPDX != current.LicenseSPDX {
				return fmt.Errorf("reviewed provider %q cannot change source or license without a new provider identity", current.ID)
			}
		}
		return nil
	}

	if !containsProviderID(candidate.PredecessorIDs, current.ID) {
		return fmt.Errorf("candidate provider %q does not explicitly name %q as a predecessor", candidate.ID, current.ID)
	}
	return nil
}

func containsProviderID(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
