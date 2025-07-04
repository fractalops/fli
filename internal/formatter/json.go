package formatter

import (
	"encoding/json"

	"fli/internal/runner"
)

// JSONFormatter formats query results as JSON.
type JSONFormatter struct {
	// Pretty determines if the JSON should be pretty-printed
	Pretty bool
}

// Format converts the query results to JSON format.
func (f JSONFormatter) Format(results [][]runner.Field, headers []string) string {
	// Convert results to a more JSON-friendly structure
	jsonData := make([]map[string]string, 0, len(results))

	for _, row := range results {
		rowMap := make(map[string]string)
		for i, field := range row {
			if i < len(headers) {
				rowMap[headers[i]] = field.Value
			}
		}
		jsonData = append(jsonData, rowMap)
	}

	var bytes []byte
	var err error

	if f.Pretty {
		bytes, err = json.MarshalIndent(jsonData, "", "  ")
	} else {
		bytes, err = json.Marshal(jsonData)
	}

	if err != nil {
		return "{\"error\": \"Failed to format as JSON\"}"
	}

	return string(bytes)
}
