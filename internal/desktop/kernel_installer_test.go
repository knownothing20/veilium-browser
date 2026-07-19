package desktop

import (
	"context"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/kernelinstaller"
	"github.com/knownothing20/veilium-browser/internal/kernelrelease"
)

type fakeKernelInstaller struct {
	request kernelinstaller.Request
	record  kernel.Record
}

func (installer *fakeKernelInstaller) Install(_ context.Context, request kernelinstaller.Request) (kernel.Record, error) {
	installer.request = request
	return installer.record, nil
}

func TestInstallOfficialKernelDelegates(t *testing.T) {
	service, _, _ := adapterTestService(t)
	installer := &fakeKernelInstaller{record: kernel.Record{ID: "official", Provider: kernelrelease.ProviderID, Status: kernel.StatusVerified}}
	service.kernelInstaller = installer
	request := kernelinstaller.Request{ProviderID: kernelrelease.ProviderID, Version: "152.0.7960.0", LicenseAccepted: true}
	record, err := service.InstallOfficialKernel(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "official" || installer.request != request {
		t.Fatalf("unexpected installer delegation: %#v %#v", record, installer.request)
	}
}

func TestInstallOfficialKernelFailsClosed(t *testing.T) {
	service, _, _ := adapterTestService(t)
	service.kernelInstaller = nil
	_, err := service.InstallOfficialKernel(context.Background(), kernelinstaller.Request{ProviderID: kernelrelease.ProviderID, Version: "152.0.7960.0", LicenseAccepted: true})
	if err == nil {
		t.Fatal("expected unavailable installer error")
	}
}
