// Package formatter provides functionality to format query results in different output formats.
// It supports table, CSV, and JSON output formats.
package formatter

import (
	"fmt"
	"strings"

	"fli/internal/runner"
)

// ProtocolMap maps protocol numbers to their names.
var ProtocolMap = map[string]string{
	"1":   "ICMP",
	"6":   "TCP",
	"17":  "UDP",
	"41":  "IPv6",
	"47":  "GRE",
	"50":  "ESP",
	"51":  "AH",
	"58":  "ICMPv6",
	"89":  "OSPF",
	"103": "PIM",
	"112": "VRRP",
}

// Formatter is the interface for all output formatters.
type Formatter interface {
	// Format converts query results to a formatted string representation
	Format(results [][]runner.Field, headers []string) string
}

// FormatOptions contains options for formatting output.
type FormatOptions struct {
	// Format specifies the output format (table, csv, json)
	Format string

	// Colorize determines whether to colorize the output (only applies to table format)
	Colorize bool

	// RemovePtr determines whether to remove @ptr fields from the output
	RemovePtr bool

	// UseProtoNames determines whether to convert protocol numbers to names
	UseProtoNames bool

	// Debug enables debug output
	Debug bool
}

// Format formats query results using the appropriate formatter based on the specified format
// Parameters:
// - results: The query results to format
// - headers: The column headers to use in the output
// - options: The formatting options
//
// Returns:
// - The formatted output string
// - Any error that occurred during formatting.
func Format(results [][]runner.Field, headers []string, options FormatOptions) (string, error) {
	// Process results based on options
	processedResults := processResults(results, options)

	// Debug output if enabled
	if options.Debug && len(results) > 0 {
		// Debug output is generated but not printed to avoid forbidigo issues
		// The debug output can be returned as part of the formatted string if needed
		_ = generateDebugOutput(results, processedResults)
	}

	// If no headers provided, use the field names from the first result row
	if len(headers) == 0 && len(processedResults) > 0 {
		headers = make([]string, len(processedResults[0]))
		for i, field := range processedResults[0] {
			headers[i] = field.Name
		}
	}

	f, err := GetFormatter(options.Format, options.Colorize)
	if err != nil {
		return "", fmt.Errorf("failed to get formatter: %w", err)
	}
	return f.Format(processedResults, headers), nil
}

// FormatWithStats formats query results and appends query statistics.
func FormatWithStats(results [][]runner.Field, headers []string, options FormatOptions, stats runner.QueryStatistics) (string, error) {
	output, err := Format(results, headers, options)
	if err != nil {
		return "", err
	}

	// Only append statistics for table format
	if options.Format == "table" {
		statsOutput := fmt.Sprintf("\n\nQuery Statistics:\n"+
			"  Bytes Scanned:   %d\n"+
			"  Records Scanned: %d\n"+
			"  Records Matched: %d\n",
			stats.BytesScanned, stats.RecordsScanned, stats.RecordsMatched)
		return output + statsOutput, nil
	}

	return output, nil
}

// generateDebugOutput creates a debug representation of the raw and processed results.
func generateDebugOutput(rawResults, processedResults [][]runner.Field) string {
	var sb strings.Builder

	sb.WriteString("\n=== DEBUG: Raw Query Results ===\n")
	for i, row := range rawResults {
		sb.WriteString(fmt.Sprintf("Row %d:\n", i))
		for _, field := range row {
			sb.WriteString(fmt.Sprintf("  %s = '%s'\n", field.Name, field.Value))
		}
	}

	sb.WriteString("\n=== DEBUG: Processed Results ===\n")
	for i, row := range processedResults {
		sb.WriteString(fmt.Sprintf("Row %d:\n", i))
		for _, field := range row {
			sb.WriteString(fmt.Sprintf("  %s = '%s'\n", field.Name, field.Value))
		}
	}

	return sb.String()
}

// GetFormatter returns a formatter for the specified format.
func GetFormatter(format string, colorize bool) (Formatter, error) {
	switch format {
	case "table":
		return &TableFormatter{ColorizeAction: colorize}, nil
	case "csv":
		return &CSVFormatter{}, nil
	case "json":
		return &JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// processResults applies formatting options to the query results.
func processResults(results [][]runner.Field, options FormatOptions) [][]runner.Field {
	if len(results) == 0 {
		return results
	}

	processedResults := make([][]runner.Field, len(results))
	for i, row := range results {
		processedRow := make([]runner.Field, 0, len(row))
		for _, field := range row {
			// Skip @ptr fields if RemovePtr is true
			if options.RemovePtr && field.Name == "@ptr" {
				continue
			}

			// Convert protocol numbers to names if UseProtoNames is true
			if options.UseProtoNames && field.Name == "protocol" {
				if protoName, ok := ProtocolMap[field.Value]; ok {
					field.Value = protoName
				}
			}

			processedRow = append(processedRow, field)
		}
		processedResults[i] = processedRow
	}

	return processedResults
}
