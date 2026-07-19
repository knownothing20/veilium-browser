package desktop

import (
	"testing"

	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

func TestBootstrapExposesOneExactReviewedChromiumPin(t *testing.T) {
	service, _, _ := adapterTestService(t)
	pins := service.Bootstrap().KernelPins
	if len(pins) != 1 {
		t.Fatalf("expected one reviewed Chromium pin, got %#v", pins)
	}
	pin := pins[0]
	if pin.ProviderID != kernelrelease.ProviderID || pin.BrowserVersion != "152.0.7960.0" || pin.SnapshotRevision != 1664436 {
		t.Fatalf("unexpected reviewed Chromium pin: %#v", pin)
	}
	if pin.Platform != "windows" || pin.Arch != "amd64" || pin.PackageTreeSHA256 == "" {
		t.Fatalf("reviewed Chromium pin has invalid scope: %#v", pin)
	}
}
