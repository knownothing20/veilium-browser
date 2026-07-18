package desktop

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/knownothing20/veilium-browser/internal/consistency"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func (s *Service) validateProfileConsistency(item domain.Profile) error {
	capabilities, err := fingerprint.For(item.Kernel.Provider, item.Kernel.Version)
	if err != nil {
		return err
	}
	_, checks, err := consistency.Preflight(item, capabilities, runtime.GOOS)
	if err != nil {
		return err
	}
	failures := make([]string, 0, 4)
	for _, check := range checks {
		if check.Status != consistency.CheckFailed {
			continue
		}
		detail := strings.TrimSpace(check.Detail)
		if detail == "" {
			detail = strings.TrimSpace(check.ReasonCode)
		}
		if detail == "" {
			detail = "consistency check failed"
		}
		failures = append(failures, check.ID+": "+detail)
	}
	if len(failures) > 0 {
		return fmt.Errorf("profile consistency blocked: %s", strings.Join(failures, "; "))
	}
	return nil
}
