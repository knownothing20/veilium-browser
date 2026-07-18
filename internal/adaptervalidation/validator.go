package adaptervalidation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
	"github.com/knownothing20/veilium-browser/internal/singboxprovider"
	"github.com/knownothing20/veilium-browser/internal/xrayprovider"
)

const (
	commandTimeout = 12 * time.Second
	maxOutputBytes = 64 << 10
)

type Check struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Report struct {
	AdapterID   string    `json:"adapterId"`
	AdapterName string    `json:"adapterName"`
	Kind        string    `json:"kind"`
	Version     string    `json:"version"`
	OfficialTag string    `json:"officialTag"`
	Platform    string    `json:"platform"`
	Arch        string    `json:"arch"`
	Status      string    `json:"status"`
	VersionText string    `json:"versionText"`
	Checks      []Check   `json:"checks"`
	CompletedAt time.Time `json:"completedAt"`
}

type CommandRunner interface {
	Run(context.Context, string, []string) (string, error)
}

type Validator struct {
	runner CommandRunner
	now    func() time.Time
	verify func(adapter.Record, adapterrelease.Pin) error
}

func New() *Validator { return NewWithRunner(execRunner{}, time.Now) }

func NewWithRunner(runner CommandRunner, now func() time.Time) *Validator {
	if now == nil {
		now = time.Now
	}
	return &Validator{runner: runner, now: now, verify: verifyExecutable}
}

func (v *Validator) Validate(ctx context.Context, record adapter.Record) (Report, error) {
	if v == nil || v.runner == nil {
		return Report{}, fmt.Errorf("adapter validator is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if record.Status != adapter.StatusVerified {
		return Report{}, fmt.Errorf("adapter %q is not integrity verified", record.Name)
	}
	pin, ok := adapterrelease.Find(record.Kind, record.Version, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return Report{}, fmt.Errorf("no official %s %s validation pin exists for %s/%s", record.Kind, record.Version, runtime.GOOS, runtime.GOARCH)
	}
	if v.verify == nil {
		v.verify = verifyExecutable
	}
	if err := v.verify(record, pin); err != nil {
		return Report{}, err
	}

	report := Report{
		AdapterID: record.ID, AdapterName: record.Name, Kind: record.Kind, Version: record.Version,
		OfficialTag: pin.Tag, Platform: pin.Platform, Arch: pin.Arch, Status: "passed",
		CompletedAt: v.now().UTC(),
	}
	versionCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	versionText, err := v.runner.Run(versionCtx, record.Executable, pin.VersionArgs)
	cancel()
	if err != nil {
		return Report{}, fmt.Errorf("official %s version command failed: %w", record.Kind, err)
	}
	versionText = sanitizeVersionText(versionText)
	if !strings.Contains(versionText, pin.Version) {
		return Report{}, fmt.Errorf("official %s version output does not contain %s", record.Kind, pin.Version)
	}
	report.VersionText = versionText
	report.Checks = append(report.Checks, Check{ID: "binary-version", Label: "Official binary version", Status: "pass", Detail: record.Kind + " " + pin.Version + " matched the embedded release pin."})

	samples, err := sampleConfigurations(record)
	if err != nil {
		return Report{}, err
	}
	directory, err := os.MkdirTemp("", "veilium-adapter-validation-*")
	if err != nil {
		return Report{}, fmt.Errorf("create adapter validation directory: %w", err)
	}
	defer os.RemoveAll(directory)
	if err := os.Chmod(directory, 0o700); err != nil {
		return Report{}, fmt.Errorf("protect adapter validation directory: %w", err)
	}
	for _, sample := range samples {
		configPath := filepath.Join(directory, sample.ID+".json")
		if err := writePrivateFile(configPath, sample.Config); err != nil {
			return Report{}, err
		}
		args, err := adapterrelease.MaterializeCheckArgs(pin, configPath)
		if err != nil {
			return Report{}, err
		}
		checkCtx, cancel := context.WithTimeout(ctx, commandTimeout)
		_, runErr := v.runner.Run(checkCtx, record.Executable, args)
		cancel()
		_ = os.Remove(configPath)
		if runErr != nil {
			return Report{}, fmt.Errorf("official %s rejected generated %s configuration: %w", record.Kind, sample.Label, runErr)
		}
		report.Checks = append(report.Checks, Check{ID: sample.ID, Label: sample.Label, Status: "pass", Detail: "The pinned official binary accepted Veilium's generated configuration."})
	}
	return report, nil
}

type sample struct {
	ID, Label string
	Config    []byte
}

func sampleConfigurations(record adapter.Record) ([]sample, error) {
	switch adapter.NormalizeKind(record.Kind) {
	case adapter.KindXray:
		provider := xrayprovider.New()
		inputs := []struct{ id, label, scheme, rawURL, username, secret string }{
			{"xray-vless", "VLESS TLS", "vless", "vless://server.example:443?security=tls&type=raw&sni=server.example&alpn=h2%2Chttp%2F1.1&fp=chrome&encryption=none&flow=xtls-rprx-vision", "identity", "5783a3e7-e373-51cd-8642-c83782b807c5"},
			{"xray-vmess", "VMess WebSocket TLS", "vmess", "vmess://server.example:443?security=tls&type=ws&sni=server.example&path=%2Fsocket&host=cdn.example&cipher=auto", "identity", "5783a3e7-e373-51cd-8642-c83782b807c5"},
			{"xray-trojan", "Trojan gRPC TLS", "trojan", "trojan://server.example:443?security=tls&type=grpc&sni=server.example&serviceName=veilium&authority=grpc.example&alpn=h2", "identity", "trojan-secret"},
			{"xray-shadowsocks", "Shadowsocks", "ss", "ss://server.example:8388", "aes-256-gcm", "shadowsocks-secret"},
		}
		result := make([]sample, 0, len(inputs))
		for _, input := range inputs {
			plan, err := provider.Prepare(context.Background(), adapterruntime.Request{Adapter: record, Scheme: input.scheme, ProxyURL: input.rawURL, CredentialUsername: input.username, CredentialSecret: input.secret, ProfileID: "official-validation", LocalPort: 19080})
			if err != nil {
				return nil, fmt.Errorf("generate %s validation configuration: %w", input.label, err)
			}
			result = append(result, sample{ID: input.id, Label: input.label, Config: plan.Config})
		}
		return result, nil
	case adapter.KindSingBox:
		provider := singboxprovider.New()
		inputs := []struct{ id, label, scheme, rawURL, secret string }{
			{"sing-hysteria2", "Hysteria2", "hysteria2", "hysteria2://hy.example:443?sni=hy.example&alpn=h3&upMbps=50&downMbps=200&obfs=salamander&network=udp", `{"password":"hy-secret","obfsPassword":"obfs-secret"}`},
			{"sing-tuic", "TUIC", "tuic", "tuic://tuic.example:443?serverName=tuic.example&alpn=h3&congestionControl=bbr&udpRelayMode=quic&network=udp", `{"uuid":"2dd61d93-75d8-4da4-ac0e-6aece7eac365","password":"tuic-secret"}`},
			{"sing-anytls", "AnyTLS", "anytls", "anytls://any.example:443?sni=any.example&alpn=h2,http%2F1.1&idleCheck=30s&idleTimeout=45s&minIdle=3", "anytls-secret"},
		}
		result := make([]sample, 0, len(inputs))
		for _, input := range inputs {
			plan, err := provider.Prepare(context.Background(), adapterruntime.Request{Adapter: record, Scheme: input.scheme, ProxyURL: input.rawURL, CredentialSecret: input.secret, ProfileID: "official-validation", LocalPort: 19080})
			if err != nil {
				return nil, fmt.Errorf("generate %s validation configuration: %w", input.label, err)
			}
			result = append(result, sample{ID: input.id, Label: input.label, Config: plan.Config})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported adapter kind %q", record.Kind)
	}
}

func verifyExecutable(record adapter.Record, pin adapterrelease.Pin) error {
	info, err := os.Lstat(record.Executable)
	if err != nil {
		return fmt.Errorf("inspect managed adapter executable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("managed adapter executable must be a regular file")
	}
	file, err := os.Open(record.Executable)
	if err != nil {
		return fmt.Errorf("open managed adapter executable: %w", err)
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf("hash managed adapter executable: %w", err)
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if digest != pin.ExecutableSHA256 || size != pin.ExecutableSizeBytes || record.SHA256 != digest || record.SizeBytes != size {
		return fmt.Errorf("managed adapter does not match the pinned official %s asset", pin.Tag)
	}
	return nil
}

func writePrivateFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create private validation configuration: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write private validation configuration: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync private validation configuration: %w", err)
	}
	return file.Close()
}

func sanitizeVersionText(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\x00", ""))
	if len(value) > 512 {
		value = value[:512]
	}
	return value
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, executable string, args []string) (string, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdin = nil
	var output limitedBuffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if ctx.Err() != nil {
		return output.String(), fmt.Errorf("command timed out: %w", ctx.Err())
	}
	if err != nil {
		return output.String(), fmt.Errorf("command exited unsuccessfully: %w", err)
	}
	return output.String(), nil
}

type limitedBuffer struct{ buffer bytes.Buffer }

func (w *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := maxOutputBytes - w.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = w.buffer.Write(data)
	}
	return original, nil
}
func (w *limitedBuffer) String() string { return w.buffer.String() }
