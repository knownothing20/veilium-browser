package desktop

import (
	"context"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterinstaller"
)

type fakeOfficialInstaller struct {
	request adapterinstaller.Request
	record  adapter.Record
	err     error
}

func (i *fakeOfficialInstaller) Install(_ context.Context, request adapterinstaller.Request) (adapter.Record, error) {
	i.request = request
	return i.record, i.err
}

func TestInstallOfficialAdapterDelegatesToInstaller(t *testing.T) {
	service, _, _ := adapterTestService(t)
	installer := &fakeOfficialInstaller{record: adapter.Record{ID: "official", Kind: adapter.KindXray, Version: "26.3.27", Official: true}}
	service.adapterInstaller = installer
	request := adapterinstaller.Request{Kind: adapter.KindXray, Version: "26.3.27", LicenseAccepted: true}
	record, err := service.InstallOfficialAdapter(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "official" || installer.request != request {
		t.Fatalf("unexpected installer delegation: record=%#v request=%#v", record, installer.request)
	}
}

func TestInstallOfficialAdapterFailsClosedWithoutInstaller(t *testing.T) {
	service, _, _ := adapterTestService(t)
	service.adapterInstaller = nil
	_, err := service.InstallOfficialAdapter(context.Background(), adapterinstaller.Request{Kind: adapter.KindXray, Version: "26.3.27", LicenseAccepted: true})
	if err == nil {
		t.Fatalf("expected installer unavailable error, got %v", err)
	}
}
