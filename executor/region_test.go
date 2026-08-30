package executor

import "testing"

func TestParseRegionSelectionMapsUpstreamNumbers(t *testing.T) {
	got, err := ParseRegionSelection("0, 1,sea,11,1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0 10 22 21" {
		t.Fatalf("ParseRegionSelection() = %q, want %q", got, "0 10 22 21")
	}
}

func TestParseRegionSelectionAcceptsNamesAndRejectsUnknown(t *testing.T) {
	for _, input := range []string{"globe", "Taiwan", "hong-kong", "Hong Kong", "North America", "South America", "south-east-asia", "South East Asia", "all-platforms", "oce", "ai"} {
		if _, err := ParseRegionSelection(input); err != nil {
			t.Errorf("ParseRegionSelection(%q) returned error: %v", input, err)
		}
	}
	if _, err := ParseRegionSelection(""); err == nil {
		t.Fatal("expected empty region selection to fail")
	}
	if _, err := ParseRegionSelection("12"); err == nil {
		t.Fatal("expected unknown region number to fail")
	}
}

func TestParseRegionSelectionGreedilyConsumesMultipleWordNames(t *testing.T) {
	got, err := ParseRegionSelection("North America South East Asia Europe")
	if err != nil {
		t.Fatal(err)
	}
	if got != "14 22 16" {
		t.Fatalf("ParseRegionSelection() = %q, want %q", got, "14 22 16")
	}
}
