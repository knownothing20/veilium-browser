package fingerprint

import "testing"

func TestOfficialSnapshotProviderIsReviewedButDoesNotInventOverrides(t *testing.T) {
	capabilities, err := For(ProviderOfficial, "152.0.7960.0")
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.TrustStatus != TrustReviewed || capabilities.Revision != 1 {
		t.Fatalf("unexpected reviewed Provider identity: %#v", capabilities)
	}
	for id, declaration := range capabilities.Capabilities {
		if declaration.Status != CapabilityUnsupported {
			t.Fatalf("stock Chromium capability %s was optimistically claimed as %s", id, declaration.Status)
		}
	}
	if _, err := For(ProviderOfficial, "152.0.7960.1"); err == nil {
		t.Fatal("nearby Chromium version inherited exact reviewed trust")
	}
}
