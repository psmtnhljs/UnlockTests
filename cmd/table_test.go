package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oneclickvirt/UnlockTests/executor"
	"github.com/oneclickvirt/UnlockTests/model"
)

func TestTableRowsSeparatesVersionsAndKeepsDetails(t *testing.T) {
	results := []executor.StructuredResult{
		{Name: "Netflix", IPVersion: "ipv4", Status: model.StatusYes, Region: "us"},
		{Name: "Gemini", IPVersion: "ipv4", Status: model.StatusRestricted},
		{Name: "Grok", IPVersion: "ipv4", Status: model.StatusBanned},
		{Name: "DeepSeek", IPVersion: "ipv6", Status: model.StatusTimeout},
	}
	headings, values := tableRows(results, 4)
	if want := []string{"Netflix", "Gemini", "Grok"}; !reflect.DeepEqual(headings, want) {
		t.Fatalf("headings = %#v, want %#v", headings, want)
	}
	if want := []string{"US", "RESTRICTED", "BANNED"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("values = %#v, want %#v", values, want)
	}
	_, values = tableRows(results, "ipv6")
	if want := []string{"TIMEOUT"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("IPv6 values = %#v, want %#v", values, want)
	}
}

func TestTableResultValuePreservesFailureClasses(t *testing.T) {
	tests := map[string]string{
		model.StatusYes:         "YES",
		model.StatusNo:          "NO",
		model.StatusRestricted:  "RESTRICTED",
		model.StatusBanned:      "BANNED",
		model.StatusNetworkErr:  "ERR",
		model.StatusErr:         "ERR",
		model.StatusUnexpected:  "ERR",
		model.StatusTimeout:     "TIMEOUT",
		model.StatusRateLimited: "RATE LIMITED",
	}
	for status, want := range tests {
		if got := tableResultValue(executor.StructuredResult{Status: status}); got != want {
			t.Errorf("tableResultValue(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestServiceResultTableHasSixAlignedColumns(t *testing.T) {
	rows := serviceResultTable([]string{"Amazon", "Apple", "Bilibili", "Netflix"}, []string{"SG", "US", "NO", "JP"})
	widths := serviceResultWidths(rows)
	if got, want := tableBorder(widths, "┌", "┬", "┐"), "┌──────────┬────────┬──────────┬────────┬──────────┬────────┐"; got != want {
		t.Fatalf("top border = %q, want %q", got, want)
	}
	if got, want := tableRow(rows[1], widths), "│  Amazon  │   SG   │  Apple   │   US   │ Bilibili │   NO   │"; got != want {
		t.Fatalf("first data row = %q, want %q", got, want)
	}
	if got, want := tableRow(rows[2], widths), "│ Netflix  │   JP   │          │        │          │        │"; got != want {
		t.Fatalf("padded final row = %q, want %q", got, want)
	}
}

func TestCompactServiceName(t *testing.T) {
	for input, want := range map[string]string{
		"Amazon Prime Video": "Amazon",
		"Google Play Store":  "Google Play",
		"Youtube CDN":        "YouTube CDN",
		"Youtube Premium":    "YouTube Premium",
		"Netflix":            "Netflix",
	} {
		if got := compactServiceName(input); got != want {
			t.Errorf("compactServiceName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderStructuredTableIncludesBothIPVersions(t *testing.T) {
	output := renderStructuredTable([]executor.StructuredResult{
		{Name: "Netflix", IPVersion: "ipv4", Status: model.StatusYes, Region: "us"},
		{Name: "Netflix", IPVersion: "ipv6", Status: model.StatusNo},
	})
	if !strings.Contains(output, "IPv4:") || !strings.Contains(output, "IPv6:") || !strings.Contains(output, "US") || !strings.Contains(output, "NO") {
		t.Fatalf("unexpected table output: %q", output)
	}
}
