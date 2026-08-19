//go:build windows

package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/evidence"
	"github.com/knownothing20/veilium-browser/internal/fingerprint"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/kernelinstaller"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
)

func TestRealReviewedKernelLifecycleOptIn(t *testing.T) {
	if os.Getenv("VEILIUM_TEST_REAL_REVIEWED_KERNEL") != "1" {
		t.Skip("set VEILIUM_TEST_REAL_REVIEWED_KERNEL=1 to exercise the pinned reviewed Chromium")
	}
	root := t.TempDir()
	store, err := profile.Open(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = service.Shutdown(ctx)
	})

	release, ok := kernelrelease.Find(kernelrelease.ProviderID, "152.0.7960.0", "windows", "amd64")
	if !ok {
		t.Fatal("pinned reviewed Chromium release is unavailable")
	}
	var record kernel.Record
	if cachedPackage := strings.TrimSpace(os.Getenv("VEILIUM_REAL_REVIEWED_CHROMIUM_PACKAGE")); cachedPackage != "" {
		cachedPackage, err = filepath.Abs(cachedPackage)
		if err != nil {
			t.Fatal(err)
		}
		record, err = service.kernels.ImportPackage(kernel.PackageImportRequest{
			Name: "Reviewed Chromium lifecycle cache", Provider: release.ProviderID, Version: release.BrowserVersion,
			SourceRoot: cachedPackage, ExecutablePath: release.ExecutablePath,
			SnapshotRevision: release.SnapshotRevision, ArchiveSHA256: release.ArchiveSHA256,
		})
	} else {
		installContext, installCancel := context.WithTimeout(context.Background(), 12*time.Minute)
		record, err = service.InstallOfficialKernel(installContext, kernelinstaller.Request{
			ProviderID: release.ProviderID, Version: release.BrowserVersion, LicenseAccepted: true,
		})
		installCancel()
	}
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != kernel.StatusVerified || record.Provider != kernelrelease.ProviderID || record.PackageTreeSHA256 == "" {
		t.Fatalf("reviewed package was not installed with exact verified identity: %#v", record)
	}

	input := validProfile()
	input.Name = "Reviewed Chromium lifecycle"
	input.Kernel = domain.KernelRef{ID: record.ID}
	screenWidth, screenHeight := 1920, 1080
	if configuredScreen := strings.TrimSpace(os.Getenv("VEILIUM_REAL_SCREEN_SIZE")); configuredScreen != "" {
		if count, scanErr := fmt.Sscanf(configuredScreen, "%dx%d", &screenWidth, &screenHeight); scanErr != nil || count != 2 || screenWidth < 800 || screenWidth > 7680 || screenHeight < 600 || screenHeight > 4320 {
			t.Fatalf("invalid VEILIUM_REAL_SCREEN_SIZE %q", configuredScreen)
		}
	}
	input.Fingerprint = domain.FingerprintConfig{
		Platform: runtimePlatformForTest(), Brand: "Chromium", Language: "zh-CN", Timezone: "Asia/Hong_Kong",
		ScreenWidth: screenWidth, ScreenHeight: screenHeight, WindowWidth: 1280, WindowHeight: 800,
		DeviceScaleFactor: 1, WebRTCPolicy: "default", CanvasMode: "native", AudioMode: "native",
		FontMode: "native", ClientRectsMode: "native", GPUProfile: "native",
	}
	input.Proxy = domain.ProxyConfig{URL: "direct://"}
	created, err := service.CreateProfile(input)
	if err != nil {
		t.Fatal(err)
	}

	startContext, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	session, err := service.StartProfile(startContext, created.ID)
	startCancel()
	if err != nil {
		t.Fatal(err)
	}
	if session.State != supervisor.StateReady || session.PID < 1 || session.CDPPort < 1 || session.WebSocketDebuggerURL == "" {
		t.Fatalf("reviewed browser did not become ready: %#v", session)
	}

	evidenceContext, evidenceCancel := context.WithTimeout(context.Background(), 45*time.Second)
	run, err := service.RunEvidence(evidenceContext, created.ID)
	evidenceCancel()
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != evidence.RunPassed && run.Status != evidence.RunPartial {
		t.Fatalf("real-browser evidence did not complete acceptably: %#v", run)
	}
	if run.ProviderID != fingerprint.ProviderOfficial || run.BinaryIdentity.PackageTreeSHA256 != record.PackageTreeSHA256 {
		t.Fatalf("evidence was not bound to the exact reviewed package: %#v", run)
	}

	stopContext, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
	stopped, err := service.StopProfile(stopContext, created.ID)
	stopCancel()
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != supervisor.StateExited || stopped.ExitedAt == nil || service.IsProfileActive(created.ID) {
		t.Fatalf("reviewed browser process tree did not stop cleanly: %#v", stopped)
	}
	reverified, err := service.VerifyKernel(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reverified.Status != kernel.StatusVerified || reverified.PackageTreeSHA256 != record.PackageTreeSHA256 || reverified.PackageFileCount != record.PackageFileCount || reverified.PackageSizeBytes != record.PackageSizeBytes {
		t.Fatalf("reviewed package identity changed after the real lifecycle: before=%#v after=%#v", record, reverified)
	}
}

func runtimePlatformForTest() string { return "windows" }
