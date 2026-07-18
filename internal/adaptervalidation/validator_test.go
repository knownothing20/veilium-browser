package adaptervalidation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/adapterrelease"
)

type fakeRunner struct {
	calls  [][]string
	failAt int
	kind   string
}

func (r *fakeRunner) Run(_ context.Context, executable string, args []string) (string, error) {
	call := append([]string{executable}, args...)
	r.calls = append(r.calls, call)
	if r.failAt > 0 && len(r.calls) == r.failAt {
		return "sensitive output should not be returned", errors.New("exit 1")
	}
	if len(args) == 1 && args[0] == "version" {
		if r.kind == adapter.KindSingBox {
			return "sing-box version 1.13.12", nil
		}
		return "Xray 26.3.27 official", nil
	}
	return "configuration accepted", nil
}

func TestValidatorRunsPinnedXrayChecks(t *testing.T) {
	runner := &fakeRunner{kind: adapter.KindXray}
	validator := NewWithRunner(runner, func() time.Time { return time.Unix(100, 0) })
	validator.verify = func(adapter.Record, adapterrelease.Pin) error { return nil }
	report, err := validator.Validate(context.Background(), adapter.Record{
		ID: "xray-a", Name: "Xray", Kind: adapter.KindXray, Version: "26.3.27",
		Status: adapter.StatusVerified, Executable: "/managed/xray",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || len(report.Checks) != 5 || len(runner.calls) != 5 {
		t.Fatalf("unexpected report: %#v calls=%#v", report, runner.calls)
	}
	for _, call := range runner.calls[1:] {
		if strings.Contains(strings.Join(call, " "), "{config}") {
			t.Fatalf("config token was not materialized: %#v", call)
		}
	}
}

func TestValidatorRunsPinnedSingBoxChecks(t *testing.T) {
	runner := &fakeRunner{kind: adapter.KindSingBox}
	validator := NewWithRunner(runner, time.Now)
	validator.verify = func(adapter.Record, adapterrelease.Pin) error { return nil }
	report, err := validator.Validate(context.Background(), adapter.Record{
		ID: "sing-a", Name: "sing-box", Kind: adapter.KindSingBox, Version: "1.13.12",
		Status: adapter.StatusVerified, Executable: "/managed/sing-box",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Checks) != 4 || len(runner.calls) != 4 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestValidatorDoesNotReturnCommandOutputOnConfigurationFailure(t *testing.T) {
	runner := &fakeRunner{kind: adapter.KindXray, failAt: 2}
	validator := NewWithRunner(runner, time.Now)
	validator.verify = func(adapter.Record, adapterrelease.Pin) error { return nil }
	_, err := validator.Validate(context.Background(), adapter.Record{
		ID: "xray-a", Name: "Xray", Kind: adapter.KindXray, Version: "26.3.27",
		Status: adapter.StatusVerified, Executable: "/managed/xray",
	})
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if strings.Contains(err.Error(), "sensitive output") {
		t.Fatalf("command output leaked through error: %v", err)
	}
}
