package formatter

import (
	"strings"

	"fli/internal/runner"
)

const fieldTimestamp = "@timestamp"

// ParseFlowLogMessage parses a VPC Flow Log message into individual fields.
// Format: version account-id interface-id srcaddr dstaddr srcport dstport protocol packets bytes start end action log-status.
func ParseFlowLogMessage(message string) map[string]string {
	fields := strings.Fields(message)
	result := make(map[string]string)

	// Check if we have enough fields for a valid flow log message
	if len(fields) < 14 {
		return result
	}

	// Map fields to their names
	fieldNames := []string{
		"version", "account-id", "interface-id", "srcaddr", "dstaddr",
		"srcport", "dstport", "protocol", "packets", "bytes",
		"start", "end", "action", "log-status",
	}

	for i, name := range fieldNames {
		if i < len(fields) {
			result[name] = fields[i]
		}
	}

	return result
}

// EnrichResultsWithMessageData parses the @message field in results and adds the parsed fields.
func EnrichResultsWithMessageData(results [][]runner.Field) [][]runner.Field {
	if len(results) == 0 {
		return results
	}

	enrichedResults := make([][]runner.Field, len(results))

	for i, row := range results {
		// Find the @message field
		var message string
		var timestamp string

		for _, field := range row {
			if field.Name == "@message" {
				message = field.Value
			}
			if field.Name == fieldTimestamp {
				timestamp = field.Value
			}
		}

		// Create a new row with the original fields plus the parsed fields
		newRow := make([]runner.Field, 0, len(row)+15) // Add extra capacity for parsed fields

		// Add timestamp field first for better ordering
		if timestamp != "" {
			newRow = append(newRow, runner.Field{
				Name:  "timestamp",
				Value: timestamp,
			})
		}

		// Add the original fields (except @timestamp which we already added)
		for _, field := range row {
			if field.Name != fieldTimestamp { // Skip @timestamp as we already added it
				newRow = append(newRow, field)
			}
		}

		// If message found, parse and add its fields
		if message != "" {
			parsedFields := ParseFlowLogMessage(message)

			// Add the parsed fields
			for name, value := range parsedFields {
				// Check if the field already exists
				exists := false
				for _, field := range newRow {
					if field.Name == name {
						exists = true
						break
					}
				}

				// Add the field if it doesn't exist
				if !exists {
					newRow = append(newRow, runner.Field{
						Name:  name,
						Value: value,
					})
				}
			}
		}

		enrichedResults[i] = newRow
	}

	return enrichedResults
}
