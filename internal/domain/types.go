package domain

import "time"

type Profile struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Group       string            `json:"group,omitempty"`
	Notes       string            `json:"notes,omitempty"`
	Kernel      KernelRef         `json:"kernel"`
	Fingerprint FingerprintConfig `json:"fingerprint"`
	Proxy       ProxyConfig       `json:"proxy"`
	UserDataDir string            `json:"userDataDir"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type KernelRef struct {
	ID         string `json:"id,omitempty"`
	Provider   string `json:"provider"`
	Version    string `json:"version"`
	Executable string `json:"executable"`
}

type FingerprintConfig struct {
	Seed                string `json:"seed,omitempty"`
	Platform            string `json:"platform"`
	Brand               string `json:"brand"`
	Language            string `json:"language"`
	Timezone            string `json:"timezone"`
	ScreenWidth         int    `json:"screenWidth"`
	ScreenHeight        int    `json:"screenHeight"`
	HardwareConcurrency int    `json:"hardwareConcurrency,omitempty"`
	DeviceMemoryGB      int    `json:"deviceMemoryGb,omitempty"`
	WebRTCPolicy        string `json:"webrtcPolicy"`
	CanvasMode          string `json:"canvasMode"`
	AudioMode           string `json:"audioMode"`
	FontMode            string `json:"fontMode"`
	ClientRectsMode     string `json:"clientRectsMode"`
	GPUProfile          string `json:"gpuProfile"`
	GPUVendor           string `json:"gpuVendor,omitempty"`
	GPURenderer         string `json:"gpuRenderer,omitempty"`
}

type ProxyConfig struct {
	URL           string `json:"url,omitempty"`
	CredentialRef string `json:"credentialRef,omitempty"`
	AdapterRef    string `json:"adapterRef,omitempty"`
}

type LaunchPlan struct {
	Executable     string            `json:"executable"`
	Args           []string          `json:"args"`
	Environment    map[string]string `json:"environment,omitempty"`
	ProxyDisplay   string            `json:"proxyDisplay"`
	RequiresBridge bool              `json:"requiresBridge"`
	BridgeKind     string            `json:"bridgeKind,omitempty"`
	Warnings       []string          `json:"warnings,omitempty"`
}
