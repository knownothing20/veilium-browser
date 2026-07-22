package desktop

import (
	"reflect"
	"testing"
	"time"

	"github.com/knownothing20/veilium-browser/internal/lifecycle"
)

func TestPreparePortableDraftRecordPreservesOwnedLockAndAddsLimitations(t *testing.T) {
	record := lifecycle.Record{
		ProfileID: "profile-1",
		State: lifecycle.StateAvailable,
		Lock: &lifecycle.OperationLock{OperationID: "operation-1", AcquiredAt: time.Now().UTC()},
		LimitationCodes: []string{"existing"},
	}
	got, err := preparePortableDraftRecord(
		record,
		"operation-1",
		"portable-source",
		"portable-import-validation-required",
		"existing",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != lifecycle.StateDraft || got.SourceID != "portable-source" {
		t.Fatalf("draft record = %#v", got)
	}
	if got.Lock == nil || got.Lock.OperationID != "operation-1" {
		t.Fatal("portable draft preparation changed lock ownership")
	}
	want := []string{"existing", "portable-import-validation-required"}
	if !reflect.DeepEqual(got.LimitationCodes, want) {
		t.Fatalf("limitations = %#v, want %#v", got.LimitationCodes, want)
	}
}

func TestPreparePortableDraftRecordRejectsForeignLock(t *testing.T) {
	record := lifecycle.Record{
		ProfileID: "profile-1",
		State: lifecycle.StateAvailable,
		Lock: &lifecycle.OperationLock{OperationID: "other-operation", AcquiredAt: time.Now().UTC()},
	}
	if _, err := preparePortableDraftRecord(record, "operation-1", "source"); err == nil {
		t.Fatal("foreign lifecycle lock was accepted")
	}
}
