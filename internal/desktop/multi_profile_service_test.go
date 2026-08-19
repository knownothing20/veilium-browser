package desktop

import (
	"reflect"
	"testing"

	"github.com/knownothing20/veilium-browser/internal/domain"
	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestNormalizeBulkProfileIDsSortsAndDeduplicates(t *testing.T) {
	got, err := normalizeBulkProfileIDs([]string{" beta ", "alpha", "beta", ""})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeBulkProfileIDs() = %#v, want %#v", got, want)
	}
}

func TestNormalizeBulkMetadataMutationRejectsContradictoryTags(t *testing.T) {
	_, err := normalizeBulkMetadataMutation(BulkMetadataUpdateRequest{
		AddTags:    []string{"QA"},
		RemoveTags: []string{"qa"},
	})
	if err == nil {
		t.Fatal("expected contradictory tag mutation to fail")
	}
}

func TestApplyBulkMetadataMutationPreservesUnrelatedProfileState(t *testing.T) {
	input := domain.Profile{
		ID:    "profile-1",
		Name:  "Profile One",
		Group: "Old",
		Notes: "keep this",
		Tags:  []string{"Keep", "Remove"},
	}
	mutation, err := normalizeBulkMetadataMutation(BulkMetadataUpdateRequest{
		SetGroup:   true,
		Group:      "New Group",
		AddTags:    []string{"Added", "keep"},
		RemoveTags: []string{"remove"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := applyBulkMetadataMutation(input, mutation)
	if got.Group != "New Group" {
		t.Fatalf("group = %q, want New Group", got.Group)
	}
	if got.Notes != input.Notes || got.Name != input.Name || got.ID != input.ID {
		t.Fatal("bulk metadata mutation changed unrelated Profile fields")
	}
	wantTags := []string{"Added", "Keep"}
	if !reflect.DeepEqual(got.Tags, wantTags) {
		t.Fatalf("tags = %#v, want %#v", got.Tags, wantTags)
	}
}

func TestBulkOperationStatusDerivesAggregateResult(t *testing.T) {
	tests := []struct {
		name  string
		items []lifecycle.OperationItemResult
		want  lifecycle.OperationStatus
	}{
		{name: "complete", items: []lifecycle.OperationItemResult{{Status: lifecycle.ItemSucceeded}, {Status: lifecycle.ItemSucceeded}}, want: lifecycle.OperationCompleted},
		{name: "partial", items: []lifecycle.OperationItemResult{{Status: lifecycle.ItemSucceeded}, {Status: lifecycle.ItemSkipped}}, want: lifecycle.OperationPartial},
		{name: "cancelled", items: []lifecycle.OperationItemResult{{Status: lifecycle.ItemCancelled}, {Status: lifecycle.ItemCancelled}}, want: lifecycle.OperationCancelled},
		{name: "failed", items: []lifecycle.OperationItemResult{{Status: lifecycle.ItemFailed}, {Status: lifecycle.ItemSkipped}}, want: lifecycle.OperationFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := bulkOperationStatus(test.items); got != test.want {
				t.Fatalf("bulkOperationStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
