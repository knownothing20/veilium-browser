package adapterruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
)

func TestPrepareRequiresRegisteredProvider(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Prepare(context.Background(), Request{
		Adapter: adapter.Record{Name: "Xray", Kind: adapter.KindXray, Status: adapter.StatusVerified},
		Scheme:  "vless",
	})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected unavailable provider, got %v", err)
	}
}

func TestPrepareRejectsMismatchedKind(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Prepare(context.Background(), Request{
		Adapter: adapter.Record{Name: "sing-box", Kind: adapter.KindSingBox, Status: adapter.StatusVerified},
		Scheme:  "vless",
	})
	if err == nil {
		t.Fatal("expected mismatch rejection")
	}
}
