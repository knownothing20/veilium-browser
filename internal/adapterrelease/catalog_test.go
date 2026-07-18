package adapterrelease

import "testing"

func TestEmbeddedCatalogIsValidAndFindsPinnedExecutables(t *testing.T) {
	manifest, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Releases) != 2 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	pins, err := Pins()
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 4 {
		t.Fatalf("expected four platform pins, got %d", len(pins))
	}
	pin, ok := MatchExecutable("xray", "26.3.27", "8255dd939c34cf966cc91517b6324dd3c8d0bcf49ffac8beca049a38c46845ed", 36577406)
	if !ok || pin.Platform != "linux" || pin.AssetName != "Xray-linux-64.zip" {
		t.Fatalf("unexpected pin: %#v", pin)
	}
}

func TestMaterializeCheckArgsRequiresExactToken(t *testing.T) {
	pin := Pin{ConfigurationArgs: []string{"check", "-c", ConfigPathToken}}
	args, err := MaterializeCheckArgs(pin, "/tmp/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 3 || args[2] != "/tmp/config.json" {
		t.Fatalf("unexpected args: %#v", args)
	}
	if _, err := MaterializeCheckArgs(Pin{ConfigurationArgs: []string{"--config=" + ConfigPathToken}}, "/tmp/config.json"); err == nil {
		t.Fatal("expected embedded token rejection")
	}
}
