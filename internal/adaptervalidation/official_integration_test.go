package adaptervalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
)

func TestOfficialAdapterBinaries(t *testing.T) {
	tests := []struct{ kind, version, env string }{
		{adapter.KindXray, "26.3.27", "VEILIUM_XRAY_BINARY"},
		{adapter.KindSingBox, "1.13.12", "VEILIUM_SINGBOX_BINARY"},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			path := os.Getenv(test.env)
			if path == "" {
				t.Skipf("%s is not configured", test.env)
			}
			digest, size := fileDigest(t, path)
			pin, ok := adapterrelease.MatchExecutable(test.kind, test.version, digest, size)
			if !ok {
				t.Fatalf("%s does not match the embedded official release pin", path)
			}
			report, err := New().Validate(context.Background(), adapter.Record{
				ID: test.kind + "-official", Name: filepath.Base(path), Kind: test.kind, Version: test.version,
				Executable: path, SHA256: digest, SizeBytes: size, Status: adapter.StatusVerified,
				Official: true, OfficialTag: pin.Tag, OfficialAsset: pin.AssetName,
				OfficialPlatform: pin.Platform, OfficialArch: pin.Arch,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "passed" {
				t.Fatalf("unexpected report: %#v", report)
			}
		})
	}
}

func fileDigest(t *testing.T, path string) (string, int64) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), size
}
