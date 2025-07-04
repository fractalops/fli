package formatter

import (
	"encoding/csv"
	"strings"

	"fli/internal/runner"
)

// CSVFormatter formats query results as CSV.
type CSVFormatter struct {
	// Delimiter is the character used to separate fields
	Delimiter rune
}

// Format converts the query results to CSV format.
func (f CSVFormatter) Format(results [][]runner.Field, headers []string) string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Set delimiter if specified (default is comma)
	if f.Delimiter != 0 {
		writer.Comma = f.Delimiter
	}

	// Write headers
	if err := writer.Write(headers); err != nil {
		// If we can't write headers, return an error message
		return "Error: failed to write CSV headers"
	}

	// Write data rows
	for _, row := range results {
		values := make([]string, len(row))
		for i, field := range row {
			values[i] = field.Value
		}
		if err := writer.Write(values); err != nil {
			// If we can't write a row, continue with the next one
			// This is a non-critical error in CSV formatting
			continue
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		// If we can't flush, return an error message
		return "Error: failed to write CSV data"
	}
	return sb.String()
}
