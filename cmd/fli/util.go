package main

import (
	"fmt"
	"os"
	"strings"
)

// expandPath expands a path with ~ to the user's home directory.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}

	return strings.Replace(path, "~", home, 1), nil
}

// parseFields converts a comma-separated field string to a slice of fields.
func parseFields(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	// Join all arguments with spaces, then split by commas
	joined := strings.Join(args, " ")
	fields := strings.Split(joined, ",")

	// Trim whitespace from each field
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}

	return fields
}

// isNumericField returns true if the field is a known numeric field.
func isNumericField(field string) bool {
	numericFields := map[string]bool{
		"bytes":   true,
		"packets": true,
	}
	return numericFields[field]
}
