package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/oneclickvirt/UnlockTests/executor"
	"github.com/oneclickvirt/UnlockTests/model"
)

const (
	tableIPv4 = "ipv4"
	tableIPv6 = "ipv6"
	tableAuto = "auto"
)

// renderStructuredTable renders compact tables without making any network
// calls. Results are already ordered by the structured executor, so the
// renderer only separates IPv4 and IPv6 runs and keeps that order intact.
func renderStructuredTable(results []executor.StructuredResult) string {
	var builder strings.Builder
	for _, version := range []string{tableIPv4, tableIPv6, tableAuto} {
		headings, values := tableRows(results, version)
		if len(headings) == 0 {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		label := map[string]string{tableIPv4: "IPv4", tableIPv6: "IPv6", tableAuto: "Auto"}[version]
		builder.WriteString(label)
		builder.WriteString(":\n")
		rows := serviceResultTable(headings, values)
		widths := serviceResultWidths(rows)
		builder.WriteString(tableBorder(widths, "┌", "┬", "┐"))
		builder.WriteByte('\n')
		for index, row := range rows {
			builder.WriteString(tableRow(row, widths))
			builder.WriteByte('\n')
			if index == len(rows)-1 {
				builder.WriteString(tableBorder(widths, "└", "┴", "┘"))
			} else {
				builder.WriteString(tableBorder(widths, "├", "┼", "┤"))
			}
			if index != len(rows)-1 {
				builder.WriteByte('\n')
			}
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

// formatStructuredTable is kept as a descriptive alias for callers that use
// the formatter as a pure function.
func formatStructuredTable(results []executor.StructuredResult) string {
	return renderStructuredTable(results)
}

// tableRows extracts one IP-version's service and result cells. The second
// argument accepts either the canonical string ("ipv4", "ipv6", "auto") or
// an integer (4, 6, 0), which keeps the helper convenient for legacy callers.
func tableRows(results []executor.StructuredResult, ipVersion any) ([]string, []string) {
	version := normalizeTableIPVersion(ipVersion)
	headings := make([]string, 0, len(results))
	values := make([]string, 0, len(results))
	for _, result := range results {
		if normalizeTableIPVersion(result.IPVersion) != version || strings.TrimSpace(result.Name) == "" {
			continue
		}
		headings = append(headings, tableCell(compactServiceName(result.Name)))
		values = append(values, tableCell(tableResultValue(result)))
	}
	return headings, values
}

func normalizeTableIPVersion(value any) string {
	switch typed := value.(type) {
	case int:
		switch typed {
		case 4:
			return tableIPv4
		case 6:
			return tableIPv6
		case 0:
			return tableAuto
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "4", "v4", "ip4", "ipv4", "tcp4":
			return tableIPv4
		case "6", "v6", "ip6", "ipv6", "tcp6":
			return tableIPv6
		case "", "0", "auto", "both", "dual", "dualstack":
			return tableAuto
		}
	}
	return ""
}

func compactServiceName(name string) string {
	switch {
	case strings.EqualFold(name, "Amazon Prime Video"):
		return "Amazon"
	case strings.EqualFold(name, "Google Play Store"):
		return "Google Play"
	case strings.EqualFold(name, "Spotify Registration"):
		return "Spotify"
	case strings.EqualFold(name, "Wikipedia Editability"):
		return "Wikipedia"
	case strings.EqualFold(name, "Youtube CDN"):
		return "YouTube CDN"
	case strings.EqualFold(name, "Youtube Premium"):
		return "YouTube Premium"
	}
	return name
}

func serviceResultTable(services, results []string) [][]string {
	const pairsPerRow = 3
	rows := [][]string{{"Service", "Result", "Service", "Result", "Service", "Result"}}
	for start := 0; start < len(services); start += pairsPerRow {
		row := make([]string, pairsPerRow*2)
		for pair := 0; pair < pairsPerRow; pair++ {
			index := start + pair
			if index >= len(services) {
				break
			}
			row[pair*2] = services[index]
			if index < len(results) {
				row[pair*2+1] = results[index]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func serviceResultWidths(rows [][]string) []int {
	const columnCount = 6
	serviceWidth := tableCellWidth("Service")
	resultWidth := tableCellWidth("Result")
	for _, row := range rows {
		for column, cell := range row {
			if column%2 == 0 {
				serviceWidth = max(serviceWidth, tableCellWidth(cell))
			} else {
				resultWidth = max(resultWidth, tableCellWidth(cell))
			}
		}
	}
	widths := make([]int, columnCount)
	for column := range widths {
		if column%2 == 0 {
			widths[column] = serviceWidth
		} else {
			widths[column] = resultWidth
		}
	}
	return widths
}

func tableRow(cells []string, widths []int) string {
	padded := make([]string, len(widths))
	for index, width := range widths {
		cell := ""
		if index < len(cells) {
			cell = cells[index]
		}
		padding := max(width-tableCellWidth(cell), 0)
		leftPadding := padding / 2
		rightPadding := padding - leftPadding
		padded[index] = strings.Repeat(" ", leftPadding) + cell + strings.Repeat(" ", rightPadding)
	}
	return "│ " + strings.Join(padded, " │ ") + " │"
}

func tableBorder(widths []int, left, middle, right string) string {
	sections := make([]string, len(widths))
	for index, width := range widths {
		sections[index] = strings.Repeat("─", max(width, 0)+2)
	}
	return left + strings.Join(sections, middle) + right
}

var ansiColorPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func tableCellWidth(value string) int {
	return runewidth.StringWidth(ansiColorPattern.ReplaceAllString(value, ""))
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "\n", " ")
}

// tableResultValue deliberately keeps the distinction between provider
// outcomes. In particular, a WAF ban or a restricted service must not appear
// as an ordinary NO result in compact output.
func tableResultValue(value any) string {
	status, region := "", ""
	switch result := value.(type) {
	case executor.StructuredResult:
		status, region = result.Status, result.Region
	case model.Result:
		status, region = result.Status, result.Region
	case *model.Result:
		if result != nil {
			status, region = result.Status, result.Region
		}
	}
	switch status {
	case model.StatusYes:
		if strings.TrimSpace(region) != "" {
			return strings.ToUpper(strings.TrimSpace(region))
		}
		return "YES"
	case model.StatusNo:
		return "NO"
	case model.StatusRestricted:
		return "RESTRICTED"
	case model.StatusBanned:
		return "BANNED"
	case model.StatusNetworkErr, model.StatusDNSFailed, model.StatusErr, model.StatusUnexpected:
		return "ERR"
	case model.StatusTimeout:
		return "TIMEOUT"
	case model.StatusNoIPv6:
		return "N/A"
	case model.StatusRateLimited:
		return "RATE LIMITED"
	case model.StatusCDNRelay:
		return "CDN"
	case model.PrintHead:
		return ""
	default:
		if status == "" {
			return "UNKNOWN"
		}
		return "UNKNOWN"
	}
}

// printStructuredTable is useful for the command path and keeps formatting
// separate from execution.
func printStructuredTable(results []executor.StructuredResult) {
	fmt.Print(renderStructuredTable(results))
}
