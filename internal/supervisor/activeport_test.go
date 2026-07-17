package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDevToolsActivePortDiscoveryRemovesStaleFileAndReadsFreshPort(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, devToolsActivePortFilename)
	if err := os.WriteFile(path, []byte("9221\n/devtools/browser/stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	discovery := DevToolsActivePortDiscovery{Interval: time.Millisecond}
	if err := discovery.Prepare(directory); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale file remains: %v", err)
	}
	go func() {
		time.Sleep(5 * time.Millisecond)
		_ = os.WriteFile(path, []byte("9333\n/devtools/browser/fresh\n"), 0o600)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	port, err := discovery.Wait(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	if port != 9333 {
		t.Fatalf("unexpected port %d", port)
	}
}

func TestDevToolsActivePortRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated Windows permissions")
	}
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	link := filepath.Join(directory, devToolsActivePortFilename)
	if err := os.WriteFile(target, []byte("9222\n/devtools/browser/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readDevToolsActivePort(link); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
