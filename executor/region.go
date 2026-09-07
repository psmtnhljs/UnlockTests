package executor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var regionAliases = map[string]string{
	"globe": "0", "global": "0", "international": "0",
	"taiwan": "10", "tw": "10",
	"hongkong": "11", "hk": "11",
	"japan": "12", "jp": "12",
	"korea": "13", "kr": "13",
	"northamerica": "14", "na": "14",
	"southamerica": "15", "sa": "15",
	"europe": "16", "eu": "16",
	"africa": "17", "afr": "17",
	"oceania": "18", "oce": "18",
	"sport": "19", "sports": "19",
	"all": "20", "allplatforms": "20",
	"ai": "21", "artificialintelligence": "21",
	"southeastasia": "22", "sea": "22",
}

const maxMenuSelection = 22

var numericRangeWhitespace = regexp.MustCompile(`([0-9]+)\s*-\s*([0-9]+)`)

// ParseRegionSelection converts region names and this project's menu numbers
// to the legacy selection string used by this module. Keeping numeric values
// aligned with the interactive menu avoids a second, conflicting numbering
// scheme for the same groups.
//
// Multiple values may be separated by commas or whitespace. Numeric ranges are
// inclusive, so "11-18" selects every menu group from Hong Kong to Oceania.
// Duplicate selections are removed without changing their supplied order.
func ParseRegionSelection(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("region selection cannot be empty")
	}
	value = numericRangeWhitespace.ReplaceAllString(value, "$1-$2")

	seen := make(map[string]struct{})
	selection := make([]string, 0)
	appendSelection := func(mapped string) {
		if _, exists := seen[mapped]; exists {
			return
		}
		seen[mapped] = struct{}{}
		selection = append(selection, mapped)
	}
	for _, segment := range strings.Split(value, ",") {
		fields := strings.Fields(segment)
		for offset := 0; offset < len(fields); {
			if start, end, isRange := menuSelectionRange(fields[offset]); isRange {
				if start > end {
					return "", fmt.Errorf("region range %q must be ascending", fields[offset])
				}
				if end > maxMenuSelection {
					return "", invalidRegionError(fields[offset])
				}
				for current := start; current <= end; current++ {
					appendSelection(strconv.Itoa(current))
				}
				offset++
				continue
			}
			mapped, width := matchRegionAlias(fields, offset)
			if width == 0 {
				return "", invalidRegionError(fields[offset])
			}
			appendSelection(mapped)
			offset += width
		}
	}
	if len(selection) == 0 {
		return "", fmt.Errorf("region selection cannot be empty")
	}
	return strings.Join(selection, " "), nil
}

func matchRegionAlias(fields []string, offset int) (string, int) {
	if selection, ok := menuSelectionNumber(fields[offset]); ok {
		return selection, 1
	}
	if isDecimal(fields[offset]) {
		return "", 0
	}
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

func menuSelectionRange(value string) (int, int, bool) {
	startValue, endValue, found := strings.Cut(value, "-")
	if !found || strings.Contains(endValue, "-") || !isDecimal(startValue) || !isDecimal(endValue) {
		return 0, 0, false
	}
	start, startErr := strconv.Atoi(startValue)
	end, endErr := strconv.Atoi(endValue)
	if startErr != nil || endErr != nil || start < 0 || end < 0 {
		return 0, 0, false
	}
	return start, end, true
}

func menuSelectionNumber(value string) (string, bool) {
	if !isDecimal(value) {
		return "", false
	}
	selection, err := strconv.Atoi(value)
	if err != nil || selection < 0 || selection > maxMenuSelection {
		return "", false
	}
	return strconv.Itoa(selection), true
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func invalidRegionError(value string) error {
	return fmt.Errorf("unknown region %q (use a menu number from 0-22, a region name, or a numeric range)", value)
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
