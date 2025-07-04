package formatter

import (
	"fmt"
	"strings"

	"fli/internal/runner"
)

// ANSI color codes.
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
)

// TableFormatter formats query results as an ASCII table.
type TableFormatter struct {
	// MaxWidth limits the width of each column (0 for no limit)
	MaxWidth int
	// ColorizeAction determines if ACCEPT/REJECT actions should be colorized
	ColorizeAction bool
}

// Format converts the query results into a formatted table string.
func (f TableFormatter) Format(results [][]runner.Field, headers []string) string {
	if len(results) == 0 {
		return "No results found"
	}

	// Filter out annotation headers, they will be merged.
	displayHeaders := make([]string, 0, len(headers))
	for _, h := range headers {
		if !strings.HasSuffix(h, "_annotation") {
			displayHeaders = append(displayHeaders, h)
		}
	}

	// Extract values from results
	rows := make([][]string, len(results))
	for i, result := range results {
		row := make([]string, len(displayHeaders))
		// Create a map of fields for easier annotation lookup
		fieldMap := make(map[string]string)
		for _, field := range result {
			fieldMap[field.Name] = field.Value
		}

		for j, header := range displayHeaders {
			// Find the field with matching name (case-insensitive)
			var value, fieldName string
			found := false
			for _, field := range result {
				// Handle timestamp specially
				if header == "timestamp" && (field.Name == "@timestamp" || field.Name == "timestamp") {
					value = field.Value
					fieldName = field.Name
					found = true
					break
				}

				// Try exact match first
				if strings.EqualFold(field.Name, header) {
					value = field.Value
					fieldName = field.Name
					found = true
					break
				}

				// Try partial match (for fields like "interface-id" vs "eni-id")
				if (header == "eni-id" && strings.Contains(field.Name, "interface")) ||
					(header == "interface-id" && (strings.Contains(field.Name, "eni") ||
						field.Name == "interface_id")) {
					value = field.Value
					fieldName = field.Name
					found = true
					break
				}

				// Handle value field for stats queries
				if header == "value" && (field.Name == "value" || field.Name == "bytes" ||
					field.Name == "packets" || field.Name == "count") {
					value = field.Value
					fieldName = field.Name
					found = true
					break
				}

				// Special handling for group-by fields in stats queries
				if field.Name == header ||
					(field.Name == "srcaddr" && header == "srcAddr") ||
					(field.Name == "dstaddr" && header == "dstAddr") ||
					(field.Name == "srcport" && header == "srcPort") ||
					(field.Name == "dstport" && header == "dstPort") {
					value = field.Value
					fieldName = field.Name
					found = true
					break
				}
			}

			// If no match found, leave empty
			if !found {
				row[j] = ""
				continue
			}

			// Check for an annotation and merge it if it exists
			annotation := ""
			if anno, ok := fieldMap[fieldName+"_annotation"]; ok {
				annotation = anno
			} else if anno, ok := fieldMap[header+"_annotation"]; ok {
				// fallback for non-renamed fields
				annotation = anno
			}

			if annotation != "" {
				row[j] = fmt.Sprintf("%s [%s]", value, annotation)
			} else {
				row[j] = value
			}
		}
		rows[i] = row
	}

	// Calculate column widths
	widths := f.calculateColumnWidths(rows, displayHeaders)

	// Build the table
	var sb strings.Builder

	// Write header
	f.writeSeparator(&sb, widths)
	f.writeRow(&sb, displayHeaders, widths, -1) // -1 indicates this is a header row
	f.writeSeparator(&sb, widths)

	// Write data rows
	for _, row := range rows {
		// Find the action column index
		actionIndex := -1
		for i, header := range displayHeaders {
			if strings.EqualFold(header, "action") {
				actionIndex = i
				break
			}
		}

		f.writeRow(&sb, row, widths, actionIndex)
	}

	// Write final separator
	f.writeSeparator(&sb, widths)

	return sb.String()
}

// calculateColumnWidths determines the width needed for each column.
func (f TableFormatter) calculateColumnWidths(rows [][]string, headers []string) []int {
	widths := make([]int, len(headers))

	// Start with header widths
	for i, header := range headers {
		widths[i] = len(header)
	}

	// Check data widths
	for _, row := range rows {
		for i, value := range row {
			if i >= len(widths) {
				continue
			}
			width := len(value)
			if f.MaxWidth > 0 && width > f.MaxWidth {
				width = f.MaxWidth
			}
			if width > widths[i] {
				widths[i] = width
			}
		}
	}

	return widths
}

// writeSeparator writes a horizontal line between rows.
func (f TableFormatter) writeSeparator(sb *strings.Builder, widths []int) {
	sb.WriteString("+")
	for _, width := range widths {
		sb.WriteString(strings.Repeat("-", width+2))
		sb.WriteString("+")
	}
	sb.WriteString("\n")
}

// writeRow writes a single row of data.
// actionIndex is the index of the action column, or -1 if not applicable.
func (f TableFormatter) writeRow(sb *strings.Builder, values []string, widths []int, actionIndex int) {
	sb.WriteString("|")
	for i, value := range values {
		if i >= len(widths) {
			continue
		}

		// Truncate if needed
		if f.MaxWidth > 0 && len(value) > f.MaxWidth {
			value = value[:f.MaxWidth-3] + "..."
		}

		// Start cell
		sb.WriteString(" ")

		// Apply color if this is the action column and colorization is enabled
		if f.ColorizeAction && i == actionIndex {
			switch strings.ToUpper(value) {
			case "ACCEPT":
				sb.WriteString(colorGreen)
				sb.WriteString(value)
				sb.WriteString(colorReset)
			case "REJECT":
				sb.WriteString(colorRed)
				sb.WriteString(value)
				sb.WriteString(colorReset)
			default:
				sb.WriteString(value)
			}
		} else {
			sb.WriteString(value)
		}

		// Pad with spaces
		padding := widths[i] - len(value)
		if padding > 0 {
			sb.WriteString(strings.Repeat(" ", padding))
		}
		sb.WriteString(" |")
	}
	sb.WriteString("\n")
}
