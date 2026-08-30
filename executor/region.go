package executor

import (
	"fmt"
	"strings"
)

var regionAliases = map[string]string{
	"0": "0", "globe": "0", "global": "0", "international": "0",
	"1": "10", "taiwan": "10", "tw": "10",
	"2": "11", "hongkong": "11", "hk": "11",
	"3": "12", "japan": "12", "jp": "12",
	"4": "13", "korea": "13", "kr": "13",
	"5": "14", "northamerica": "14", "na": "14",
	"6": "15", "southamerica": "15", "sa": "15",
	"7": "16", "europe": "16", "eu": "16",
	"8": "17", "africa": "17", "afr": "17",
	"9": "22", "southeastasia": "22", "sea": "22",
	"10": "18", "oceania": "18", "oce": "18",
	"11": "21", "ai": "21", "artificialintelligence": "21",
	"all": "20", "allplatforms": "20",
}

// ParseRegionSelection converts the newer region-oriented CLI syntax to the
// legacy selection numbers used by this module. The region numbering follows
// MediaUnlockTest (0=global, 1..11=the named regions), while the returned
// selection keeps UnlockTests' existing -f numbering for compatibility.
//
// Multiple values may be separated by commas or whitespace. Duplicate regions
// are removed without changing the order in which they were supplied.
func ParseRegionSelection(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("region selection cannot be empty")
	}

	seen := make(map[string]struct{})
	selection := make([]string, 0)
	for _, segment := range strings.Split(value, ",") {
		fields := strings.Fields(segment)
		for offset := 0; offset < len(fields); {
			mapped, width := matchRegionAlias(fields, offset)
			if width == 0 {
				return "", fmt.Errorf("unknown region %q (use 0-11 or a region name)", fields[offset])
			}
			if _, exists := seen[mapped]; !exists {
				seen[mapped] = struct{}{}
				selection = append(selection, mapped)
			}
			offset += width
		}
	}
	if len(selection) == 0 {
		return "", fmt.Errorf("region selection cannot be empty")
	}
	return strings.Join(selection, " "), nil
}

func matchRegionAlias(fields []string, offset int) (string, int) {
	// Prefer the longest phrase so "South East Asia" and "North America"
	// are consumed as one region while adjacent one-word regions remain valid.
	for width := len(fields) - offset; width > 0; width-- {
		key := normalizeRegionAlias(strings.Join(fields[offset:offset+width], ""))
		if mapped, ok := regionAliases[key]; ok {
			return mapped, width
		}
	}
	return "", 0
}

func normalizeRegionAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
