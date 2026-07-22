package portableprofile

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/domain"
)

func TestArtifactRoundTripAndTamperDetection(t *testing.T) {
	artifact, err := Build(BuildInput{
		ApplicationVersion: "0.16.0-dev",
		Profile: domain.Profile{
			ID:          "local-profile-id",
			Name:        "Portable profile",
			Group:       "Research",
			Notes:       "Non-secret settings only",
			Tags:        []string{"work", "Work", "portable"},
			UserDataDir: filepath.Join("private", "profiles", "local-profile-id"),
			Kernel:      domain.KernelRef{ID: "local-kernel-id", Provider: "custom-chromium", Version: "148.0.0", Executable: "C:/private/chrome.exe"},
			Fingerprint: domain.FingerprintConfig{Seed: "source-seed", Platform: "windows", Brand: "Chromium", Language: "en-US", Timezone: "UTC", ScreenWidth: 1920, ScreenHeight: 1080, WebRTCPolicy: "disabled", CanvasMode: "native", AudioMode: "native", FontMode: "native", ClientRectsMode: "native", GPUProfile: "native"},
			Proxy:       domain.ProxyConfig{URL: "socks5://proxy.example:1080", CredentialRef: "local-credential-id", AdapterRef: "local-adapter-id"},
		},
		Kernel:             KernelRequirement{Provider: "custom-chromium", Version: "148.0.0", SHA256: strings.Repeat("a", 64), SizeBytes: 1234},
		Adapter:            &AdapterRequirement{Kind: "xray", Version: "26.3.27", SHA256: strings.Repeat("b", 64), SizeBytes: 5678},
		CredentialRequired: true,
		IdentityMode:       IdentityNew,
		ExportedAt:         time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Payload.Fingerprint.Seed != "" {
		t.Fatal("new identity export retained the source seed")
	}
	data, err := Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"local-profile-id", "local-kernel-id", "local-credential-id", "local-adapter-id", "chrome.exe", "source-seed"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("portable artifact leaked %q", forbidden)
		}
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Payload.Name = "Tampered"
	tampered, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(tampered); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestPortableURLRejectsInlineSecrets(t *testing.T) {
	for _, raw := range []string{
		"http://user:password@proxy.example:8080",
		"socks5://proxy.example:1080?token=secret",
		"vless://proxy.example:443?uuid=secret",
	} {
		if err := validatePortableURL(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestTemplateAlwaysUsesNewIdentity(t *testing.T) {
	payload := Payload{
		Name:         "Template source",
		Fingerprint:  domain.FingerprintConfig{Seed: "must-be-removed", Platform: "windows", Brand: "Chromium", Language: "en-US", Timezone: "UTC", ScreenWidth: 1920, ScreenHeight: 1080, WebRTCPolicy: "disabled", CanvasMode: "native", AudioMode: "native", FontMode: "native", ClientRectsMode: "native", GPUProfile: "native"},
		Kernel:       KernelRequirement{Provider: "custom-chromium", Version: "148.0.0", SHA256: strings.Repeat("a", 64), SizeBytes: 1234},
		IdentityMode: IdentityPreserve,
	}
	template, err := NewTemplate("Research template", payload, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if template.Payload.IdentityMode != IdentityNew || template.Payload.Fingerprint.Seed != "" {
		t.Fatal("template retained reusable identity material")
	}
}
