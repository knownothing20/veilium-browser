package desktop

import (
	"strings"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/adapter"
	"github.com/knownothing20/veilium-browser/internal/kernel"
	"github.com/knownothing20/veilium-browser/internal/portableprofile"
)

func TestPortableKernelMatchingIsExactAndVerified(t *testing.T) {
	requirement := portableprofile.KernelRequirement{
		Provider:  "custom-chromium",
		Version:   "148.0.0",
		SHA256:    strings.Repeat("a", 64),
		SizeBytes: 100,
	}
	record := kernel.Record{
		Provider:  requirement.Provider,
		Version:   requirement.Version,
		SHA256:    requirement.SHA256,
		SizeBytes: requirement.SizeBytes,
		Status:    kernel.StatusVerified,
	}
	if !kernelMatches(record, requirement) {
		t.Fatal("expected exact verified Kernel to match")
	}
	record.Status = kernel.StatusModified
	if kernelMatches(record, requirement) {
		t.Fatal("modified Kernel matched a portable requirement")
	}
	record.Status = kernel.StatusVerified
	record.SHA256 = strings.Repeat("b", 64)
	if kernelMatches(record, requirement) {
		t.Fatal("different Kernel digest matched a portable requirement")
	}
}

func TestPortableAdapterMatchingIsExactAndVerified(t *testing.T) {
	requirement := portableprofile.AdapterRequirement{
		Kind:      "xray",
		Version:   "26.3.27",
		SHA256:    strings.Repeat("c", 64),
		SizeBytes: 200,
	}
	record := adapter.Record{
		Kind:      requirement.Kind,
		Version:   requirement.Version,
		SHA256:    requirement.SHA256,
		SizeBytes: requirement.SizeBytes,
		Status:    adapter.StatusVerified,
	}
	if !adapterMatches(record, requirement) {
		t.Fatal("expected exact verified adapter to match")
	}
	record.Version = "other"
	if adapterMatches(record, requirement) {
		t.Fatal("different adapter version matched a portable requirement")
	}
}
