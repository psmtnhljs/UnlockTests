package executor

import "testing"

func TestParseRegionSelectionMapsMenuNumbers(t *testing.T) {
	got, err := ParseRegionSelection("0, 11,sea,21,11")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0 11 22 21" {
		t.Fatalf("ParseRegionSelection() = %q, want %q", got, "0 11 22 21")
	}
}

func TestParseRegionSelectionAcceptsNamesAndRejectsUnknown(t *testing.T) {
	for _, input := range []string{"globe", "Taiwan", "hong-kong", "Hong Kong", "North America", "South America", "south-east-asia", "South East Asia", "all-platforms", "oce", "sport", "ai"} {
		if _, err := ParseRegionSelection(input); err != nil {
			t.Errorf("ParseRegionSelection(%q) returned error: %v", input, err)
		}
	}
	if _, err := ParseRegionSelection(""); err == nil {
		t.Fatal("expected empty region selection to fail")
	}
	if _, err := ParseRegionSelection("23"); err == nil {
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

func TestParseRegionSelectionExpandsNumericRanges(t *testing.T) {
	for input, want := range map[string]string{
		"11-18":          "11 12 13 14 15 16 17 18",
		"11 - 18":        "11 12 13 14 15 16 17 18",
		"10,12-14,14,21": "10 12 13 14 21",
		"11-21":          "11 12 13 14 15 16 17 18 19 20 21",
		"Globe,hongkong": "0 11",
		"Globe,11-12,AI": "0 11 12 21",
	} {
		got, err := ParseRegionSelection(input)
		if err != nil {
			t.Errorf("ParseRegionSelection(%q) returned error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseRegionSelection(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseRegionSelectionRejectsInvalidRanges(t *testing.T) {
	for _, input := range []string{"21-11", "11-23", "999999999999999999999-1"} {
		if _, err := ParseRegionSelection(input); err == nil {
			t.Errorf("expected ParseRegionSelection(%q) to fail", input)
		}
	}
}
