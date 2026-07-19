package desktop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestCreateUpdateCloneLifecycle(t *testing.T) {
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.UserDataDir == "" {
		t.Fatalf("expected generated identity and data directory: %#v", created)
	}
	created.Group = "Work"
	created.Tags = []string{"Commerce", "commerce", " US "}
	updated, err := service.UpdateProfile(created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Group != "Work" || len(updated.Tags) != 2 {
		t.Fatalf("unexpected update result: %#v", updated)
	}
	cloned, err := service.CloneProfile(updated.ID, "Store B")
	if err != nil {
		t.Fatal(err)
	}
	if cloned.ID == updated.ID || cloned.UserDataDir == updated.UserDataDir {
		t.Fatalf("clone did not receive isolated identity: %#v", cloned)
	}
	if len(service.ListProfiles()) != 2 {
		t.Fatalf("expected two profiles")
	}
}

func TestCapabilitiesRejectUnknownProvider(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, _ := NewService(store, root)
	if _, err := service.Capabilities("unknown", "148.0.0"); err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestKernelRegistryProtectsProfilesAndLaunchPlans(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	service, _ := NewService(store, root)
	record, err := service.ImportKernel(kernel.ImportRequest{Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Kernel.Executable != record.Executable || created.Kernel.Provider != record.Provider {
		t.Fatalf("kernel was not resolved: %#v", created.Kernel)
	}
	if err := service.DeleteKernel(record.ID); err == nil || !strings.Contains(err.Error(), "used by profile") {
		t.Fatalf("expected in-use protection, got %v", err)
	}
	if err := os.WriteFile(record.Executable, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BuildLaunchPlan(LaunchPlanRequest{ProfileID: created.ID}); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity failure, got %v", err)
	}
}

func TestStartProfileRequiresManagedVerifiedKernelAndLocksMutations(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	runtime := newFakeRuntime()
	service, err := newService(store, root, runtime)
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.ImportKernel(kernel.ImportRequest{Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.StartProfile(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.State != supervisor.StateReady || session.CDPPort != 9222 {
		t.Fatalf("unexpected runtime session: %#v", session)
	}
	if _, err := os.Stat(created.UserDataDir); err != nil {
		t.Fatalf("managed profile directory was not created: %v", err)
	}
	joined := strings.Join(runtime.plan.Args, " ")
	if !strings.Contains(joined, "--remote-debugging-port=9222") || runtime.plan.Executable != record.Executable {
		t.Fatalf("unexpected start plan: %#v", runtime.plan)
	}
	created.Notes = "unsafe live edit"
	if _, err := service.UpdateProfile(created); err == nil || !strings.Contains(err.Error(), "while its browser is running") {
		t.Fatalf("expected live edit rejection, got %v", err)
	}
	if err := service.DeleteProfile(created.ID); err == nil || !strings.Contains(err.Error(), "while its browser is running") {
		t.Fatalf("expected live delete rejection, got %v", err)
	}
}

func TestStartProfileRejectsLegacyAndUnmanagedProfiles(t *testing.T) {
	root := t.TempDir()
	store, _ := profile.Open(filepath.Join(root, "profiles.json"))
	runtime := newFakeRuntime()
	service, _ := newService(store, root, runtime)

	legacy, err := service.CreateProfile(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartProfile(context.Background(), legacy.ID); err == nil || !strings.Contains(err.Error(), "registered kernel") {
		t.Fatalf("expected registered kernel requirement, got %v", err)
	}

	source := filepath.Join(root, "chrome-test")
	if err := os.WriteFile(source, []byte("verified-browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	record, err := service.ImportKernel(kernel.ImportRequest{Name: "Verified Chromium", Provider: fingerprint.ProviderPatched, Version: "148.0.0", SourcePath: source})
	if err != nil {
		t.Fatal(err)
	}
	input := validProfile()
	input.Kernel = domain.KernelRef{ID: record.ID}
	input.UserDataDir = filepath.Join(root, "outside")
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartProfile(context.Background(), created.ID); err == nil || !strings.Contains(err.Error(), "unmanaged") {
		t.Fatalf("expected unmanaged directory rejection, got %v", err)
	}
}

func validProfile() domain.Profile {
	return domain.Profile{
		Name:   "Store A",
		Kernel: domain.KernelRef{Provider: fingerprint.ProviderCustom, Version: "148.0.0", Executable: `C:\Browsers\Chromium\chrome.exe`},
		Fingerprint: domain.FingerprintConfig{
			Platform: runtime.GOOS, Brand: "Chromium", Language: "en-US", Timezone: "America/Los_Angeles",
			ScreenWidth: 1920, ScreenHeight: 1080,
			WebRTCPolicy: "proxy-only", CanvasMode: "native", AudioMode: "native",
			FontMode: "native", ClientRectsMode: "native", GPUProfile: "auto",
		},
		Proxy: domain.ProxyConfig{URL: "direct://"},
	}
}

type fakeRuntime struct {
	active   map[string]bool
	sessions map[string]supervisor.Session
	plan     domain.LaunchPlan
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{active: make(map[string]bool), sessions: make(map[string]supervisor.Session)}
}

func (r *fakeRuntime) Start(_ context.Context, id, name string, build supervisor.PlanBuilder) (supervisor.Session, error) {
	plan, err := build(9222)
	if err != nil {
		return supervisor.Session{}, err
	}
	r.plan = plan
	r.active[id] = true
	session := supervisor.Session{ProfileID: id, ProfileName: name, State: supervisor.StateReady, PID: 123, CDPPort: 9222, CDPURL: "http://127.0.0.1:9222", StartedAt: time.Now().UTC()}
	r.sessions[id] = session
	return session, nil
}
func (r *fakeRuntime) Stop(_ context.Context, id string) (supervisor.Session, error) {
	session, ok := r.sessions[id]
	if !ok {
		return supervisor.Session{}, errors.New("not found")
	}
	r.active[id] = false
	session.State = supervisor.StateExited
	r.sessions[id] = session
	return session, nil
}
func (r *fakeRuntime) Shutdown(context.Context) error { return nil }
func (r *fakeRuntime) List() []supervisor.Session {
	items := make([]supervisor.Session, 0, len(r.sessions))
	for _, item := range r.sessions {
		items = append(items, item)
	}
	return items
}
func (r *fakeRuntime) IsActive(id string) bool { return r.active[id] }
