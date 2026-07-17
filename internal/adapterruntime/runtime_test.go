package adapterruntime_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterruntime"
	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/supervisor"
	"github.com/knownothing20/veilium-browser/internal/xrayprovider"
)

func TestManagerStartsHealthySOCKSAndRemovesPrivateRuntimeFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "adapter-runtime")
	registry := adapterruntime.NewRegistry()
	if err := registry.Register(xrayprovider.New()); err != nil {
		t.Fatal(err)
	}
	starter := &fakeStarter{}
	manager, err := adapterruntime.NewManagerWithStarter(root, registry, starter)
	if err != nil {
		t.Fatal(err)
	}

	instance, err := manager.Start(context.Background(), adapterruntime.Request{
		Adapter:            adapter.Record{ID: "xray-a", Name: "Xray", Kind: adapter.KindXray, Status: adapter.StatusVerified, Executable: filepath.Join(root, "xray")},
		Scheme:             "vless",
		ProxyURL:           "vless://server.example:443?security=tls&sni=server.example&encryption=none",
		CredentialUsername: "identity",
		CredentialSecret:   "5783a3e7-e373-51cd-8642-c83782b807c5",
		ProfileID:          "profile-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(instance.URL(), "socks5://127.0.0.1:") || instance.Kind() != adapter.KindXray {
		t.Fatalf("unexpected instance: url=%s kind=%s", instance.URL(), instance.Kind())
	}
	if err := instance.Health(context.Background()); err != nil {
		t.Fatal(err)
	}

	starter.mu.Lock()
	configPath := starter.configPath
	configMode := starter.configMode
	directoryMode := starter.directoryMode
	arguments := append([]string(nil), starter.arguments...)
	starter.mu.Unlock()
	if runtime.GOOS != "windows" && (configMode.Perm() != 0o600 || directoryMode.Perm() != 0o700) {
		t.Fatalf("unsafe runtime permissions: config=%o directory=%o", configMode.Perm(), directoryMode.Perm())
	}
	if strings.Contains(strings.Join(arguments, " "), "5783a3e7-e373-51cd-8642-c83782b807c5") {
		t.Fatal("credential leaked into adapter arguments")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("private config disappeared before runtime close: %v", err)
	}

	if err := instance.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-instance.Done():
	case <-time.After(time.Second):
		t.Fatal("adapter process did not stop")
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private runtime directory remained after close: %v", err)
	}
}

func TestManagerCleansRuntimeDirectoryWhenProcessStartFails(t *testing.T) {
	root := filepath.Join(t.TempDir(), "adapter-runtime")
	registry := adapterruntime.NewRegistry()
	if err := registry.Register(xrayprovider.New()); err != nil {
		t.Fatal(err)
	}
	manager, err := adapterruntime.NewManagerWithStarter(root, registry, failingStarter{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Start(context.Background(), adapterruntime.Request{
		Adapter:          adapter.Record{ID: "xray-a", Name: "Xray", Kind: adapter.KindXray, Status: adapter.StatusVerified, Executable: filepath.Join(root, "xray")},
		Scheme:           "trojan",
		ProxyURL:         "trojan://server.example:443?security=tls&sni=server.example",
		CredentialSecret: "top-secret",
		ProfileID:        "profile-a",
	})
	if err == nil || !strings.Contains(err.Error(), "start adapter runtime after retries") {
		t.Fatalf("expected managed start failure, got %v", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary runtime entries remained: %#v", entries)
	}
}

type fakeStarter struct {
	mu            sync.Mutex
	configPath    string
	configMode    os.FileMode
	directoryMode os.FileMode
	arguments     []string
}

func (s *fakeStarter) Start(plan domain.LaunchPlan, _ string) (supervisor.Process, error) {
	configPath := ""
	for index := range plan.Args {
		if index > 0 && plan.Args[index-1] == "-config" {
			configPath = plan.Args[index]
			break
		}
	}
	if configPath == "" {
		return nil, errors.New("config argument was not materialized")
	}
	configInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}
	directoryInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	port, err := configInboundPort(data)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	process := newFakeProcess(listener)
	go process.serve()

	s.mu.Lock()
	s.configPath = configPath
	s.configMode = configInfo.Mode()
	s.directoryMode = directoryInfo.Mode()
	s.arguments = append([]string(nil), plan.Args...)
	s.mu.Unlock()
	return process, nil
}

func configInboundPort(data []byte) (int, error) {
	var config struct {
		Inbounds []struct {
			Port int `json:"port"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}
	if len(config.Inbounds) == 0 || config.Inbounds[0].Port < 1 {
		return 0, errors.New("config has no loopback inbound port")
	}
	return config.Inbounds[0].Port, nil
}

type fakeProcess struct {
	listener net.Listener
	done     chan error
	once     sync.Once
}

func newFakeProcess(listener net.Listener) *fakeProcess {
	return &fakeProcess{listener: listener, done: make(chan error, 1)}
}

func (p *fakeProcess) PID() int               { return 4242 }
func (p *fakeProcess) Wait() error            { return <-p.done }
func (p *fakeProcess) Signal(os.Signal) error { p.stop(nil); return nil }
func (p *fakeProcess) Kill() error            { p.stop(errors.New("killed")); return nil }

func (p *fakeProcess) stop(err error) {
	p.once.Do(func() {
		_ = p.listener.Close()
		p.done <- err
	})
}

func (p *fakeProcess) serve() {
	for {
		connection, err := p.listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(time.Second))
			reader := bufio.NewReader(connection)
			version, err := reader.ReadByte()
			if err != nil || version != 5 {
				return
			}
			count, err := reader.ReadByte()
			if err != nil {
				return
			}
			methods := make([]byte, int(count))
			if _, err := reader.Read(methods); err != nil {
				return
			}
			_, _ = connection.Write([]byte{5, 0})
		}()
	}
}

type failingStarter struct{}

func (failingStarter) Start(domain.LaunchPlan, string) (supervisor.Process, error) {
	return nil, errors.New("process unavailable")
}
