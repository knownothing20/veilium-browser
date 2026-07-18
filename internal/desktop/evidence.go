package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

type evidenceServiceState struct {
	once    sync.Once
	manager *evidence.Manager
	err     error
}

var evidenceServiceRegistry sync.Map

func evidenceManagerFor(service *Service) (*evidence.Manager, error) {
	if service == nil {
		return nil, fmt.Errorf("desktop service is required")
	}
	candidate := &evidenceServiceState{}
	loaded, _ := evidenceServiceRegistry.LoadOrStore(service, candidate)
	state := loaded.(*evidenceServiceState)
	state.once.Do(func() {
		store, err := evidence.OpenStore(filepath.Join(service.dataRoot, "evidence"), evidence.StoreOptions{})
		if err != nil {
			state.err = err
			return
		}
		state.manager, state.err = evidence.NewManager(store, evidence.ManagerOptions{})
	})
	return state.manager, state.err
}

func (s *Service) RunEvidence(ctx context.Context, profileID string) (evidence.Run, error) {
	manager, err := evidenceManagerFor(s)
	if err != nil {
		return evidence.Run{}, err
	}
	profileID = strings.TrimSpace(profileID)
	item, err := s.store.Get(profileID)
	if err != nil {
		return evidence.Run{}, err
	}
	if strings.TrimSpace(item.Kernel.ID) == "" {
		return evidence.Run{}, fmt.Errorf("profile %q must use a managed kernel before evidence can run", item.Name)
	}
	record, err := s.kernels.Verify(item.Kernel.ID)
	if err != nil {
		return evidence.Run{}, err
	}
	if err := s.resolveKernel(&item); err != nil {
		return evidence.Run{}, err
	}
	capabilities, err := fingerprint.For(item.Kernel.Provider, item.Kernel.Version)
	if err != nil {
		return evidence.Run{}, err
	}
	session, ok := readyEvidenceSession(s.supervisor.List(), item.ID)
	if !ok {
		return evidence.Run{}, fmt.Errorf("profile %q requires a ready managed browser session", item.Name)
	}
	request := evidence.RunRequest{
		Profile:      item,
		Kernel:       record,
		Session:      session,
		Capabilities: capabilities,
		SessionReady: func() bool {
			current, exists := readyEvidenceSession(s.supervisor.List(), item.ID)
			return exists && sameEvidenceSession(session, current)
		},
	}
	return manager.Run(ctx, request)
}

func (s *Service) CancelEvidence(profileID string) error {
	manager, err := evidenceManagerFor(s)
	if err != nil {
		return err
	}
	return manager.Cancel(profileID)
}

func (s *Service) ListEvidence(profileID string) ([]evidence.Run, error) {
	manager, err := evidenceManagerFor(s)
	if err != nil {
		return nil, err
	}
	return manager.List(strings.TrimSpace(profileID))
}

func (s *Service) GetEvidence(id string) (evidence.Run, error) {
	manager, err := evidenceManagerFor(s)
	if err != nil {
		return evidence.Run{}, err
	}
	return manager.Get(strings.TrimSpace(id))
}

func (s *Service) DeleteEvidence(id string) error {
	manager, err := evidenceManagerFor(s)
	if err != nil {
		return err
	}
	return manager.Delete(strings.TrimSpace(id))
}

func (s *Service) EvidenceActive(profileID string) bool {
	manager, err := evidenceManagerFor(s)
	return err == nil && manager.IsActive(profileID)
}

func shutdownEvidenceRuntimes(service *Service) {
	loaded, ok := evidenceServiceRegistry.Load(service)
	if !ok {
		return
	}
	state := loaded.(*evidenceServiceState)
	if state.manager != nil {
		state.manager.Shutdown()
	}
	evidenceServiceRegistry.Delete(service)
}

func readyEvidenceSession(sessions []supervisor.Session, profileID string) (supervisor.Session, bool) {
	for _, session := range sessions {
		if session.ProfileID == profileID && session.State == supervisor.StateReady && session.CDPPort > 0 {
			return session, true
		}
	}
	return supervisor.Session{}, false
}

func sameEvidenceSession(expected, current supervisor.Session) bool {
	return current.ProfileID == expected.ProfileID &&
		current.State == supervisor.StateReady &&
		current.PID == expected.PID &&
		current.CDPPort == expected.CDPPort &&
		current.StartedAt.Equal(expected.StartedAt)
}
