package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseCLIStructuredOptions(t *testing.T) {
	opts, err := parseCLI([]string{"--structured", "-m", "6", "-f", "0", "-conc", "3", "--timeout", "1s"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if !opts.jsonOutput || opts.mode != 6 || opts.selection != "0" || opts.concurrency != 3 || opts.timeout != time.Second {
		t.Fatalf("unexpected options: %#v", opts)
	}
}

func TestHelpRetainsLegacyFlags(t *testing.T) {
	var output bytes.Buffer
	newFlagSet(&cliOptions{}, &output).PrintDefaults()
	for _, legacy := range []string{"-b", "-f string", "-h", "-I string", "-L string", "-m int", "-region string", "-s", "-table", "-test string", "-v"} {
		if !strings.Contains(output.String(), legacy) {
			t.Fatalf("help is missing legacy flag %q: %s", legacy, output.String())
		}
	}
}

func TestParseCLIRegionSelectionUsesMenuNumbers(t *testing.T) {
	opts, err := parseCLI([]string{"-region", "0,11,sea", "-table", "-m", "4", "-timeout", "3s"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if !opts.table || !opts.regionSet || opts.selection != "0 11 22" || opts.timeout != 3*time.Second {
		t.Fatalf("unexpected region/table options: %#v", opts)
	}
}

func TestParseCLIRegionSelectionAcceptsMultipleMenuNumbers(t *testing.T) {
	opts, err := parseCLI([]string{"-region", "11,21", "-table"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if opts.selection != "11 21" || !opts.regionSet || !opts.table {
		t.Fatalf("unexpected menu-number options: %#v", opts)
	}
}

func TestParseCLIRegionSelectionAcceptsUnquotedValues(t *testing.T) {
	opts, err := parseCLI([]string{"-region", "0", "11", "-table"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if opts.selection != "0 11" || !opts.regionSet || !opts.table {
		t.Fatalf("unexpected unquoted region options: %#v", opts)
	}
}

func TestParseCLIRegionSelectionAcceptsEqualsAndMultiWordNames(t *testing.T) {
	opts, err := parseCLI([]string{"-region=North", "America", "-json"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if opts.selection != "14" || !opts.jsonOutput {
		t.Fatalf("unexpected region options: %#v", opts)
	}
}

func TestParseCLIRegionSelectionAcceptsNumericRanges(t *testing.T) {
	opts, err := parseCLI([]string{"-region", "11-18", "-table"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if opts.selection != "11 12 13 14 15 16 17 18" || !opts.regionSet || !opts.table {
		t.Fatalf("unexpected range options: %#v", opts)
	}
}

func TestParseCLIRejectsRegionConflicts(t *testing.T) {
	for _, args := range [][]string{
		{"-region", "0", "-f", "0"},
		{"-region", "0", "-test", "Netflix"},
		{"-json", "-table"},
		{"-table", "-json"},
		{"-region", "unknown"},
		{"-region", "11-23"},
	} {
		if _, err := parseCLI(args); err == nil {
			t.Fatalf("expected arguments %v to be rejected", args)
		}
	}
}
