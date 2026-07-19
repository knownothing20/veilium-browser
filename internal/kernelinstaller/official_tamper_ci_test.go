package kernelinstaller

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/kernel"
)

func TestReviewedChromiumPackageTamperForCI(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("reviewed Chromium package is Windows amd64 only")
	}
	workDir := strings.TrimSpace(os.Getenv("VEILIUM_REVIEWED_CHROMIUM_WORKDIR"))
	resultPath := strings.TrimSpace(os.Getenv("VEILIUM_REVIEWED_CHROMIUM_RESULT"))
	if workDir == "" || resultPath == "" {
		t.Skip("reviewed Chromium CI paths are not configured")
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result ciInstallResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	store, err := kernel.Open(filepath.Join(workDir, "kernels.json"), filepath.Join(workDir, "kernels"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Get(result.KernelID)
	if err != nil {
		t.Fatal(err)
	}
	dependency, err := firstPackageDependency(record.PackageRoot, record.Executable)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(dependency, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := io.WriteString(file, "tamper")
	closeErr := file.Close()
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	verified, err := store.Verify(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Status != kernel.StatusModified {
		t.Fatalf("dependency tamper remained trusted: %#v", verified)
	}
}

func firstPackageDependency(root, executable string) (string, error) {
	var selected string
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if selected != "" || entry.IsDir() || current == executable {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			selected = current
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", fmt.Errorf("reviewed Chromium package has no dependency file")
	}
	return selected, nil
}
