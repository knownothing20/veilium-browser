package desktop

import (
	"testing"

	"github.com/knownothing20/veilium-browser/internal/fingerprint"
)

func TestProviderDescriptorsAreDerivedForMultipleContracts(t *testing.T) {
	definitions := fingerprint.Definitions()
	if len(definitions) == 0 {
		t.Fatal("stock Provider contract is missing")
	}
	stock := definitions[0]
	second := stock
	second.ID = "test-reviewed-provider"
	second.Revision = 7
	second.Name = "Second reviewed Provider fixture"
	second.Description = "Test-only descriptor fixture."
	second.Versions = []string{"152.1.2.3"}
	second.PredecessorIDs = nil
	second.ReplacementID = ""

	descriptors := providerCatalogFromContracts([]fingerprint.ProviderDefinition{stock, second})
	if len(descriptors) != 2 {
		t.Fatalf("multiple Provider descriptors were not preserved: %#v", descriptors)
	}
	if descriptors[0].ID != stock.ID || len(descriptors[0].Samples) == 0 || descriptors[0].Samples[0].Revision != stock.Revision {
		t.Fatalf("stock Provider descriptor changed: %#v", descriptors[0])
	}
	if descriptors[1].ID != second.ID || descriptors[1].Name != second.Name || len(descriptors[1].Samples) != 1 || descriptors[1].Samples[0].Provider != second.ID || descriptors[1].Samples[0].Revision != second.Revision || descriptors[1].Samples[0].TrustStatus != fingerprint.TrustReviewed {
		t.Fatalf("second Provider descriptor was not backend-derived: %#v", descriptors[1])
	}
}
