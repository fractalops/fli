package formatter

import (
	"strings"
	"testing"

	"fli/internal/runner"
)

func TestFormat(t *testing.T) {
	// Sample data for testing
	headers := []string{"timestamp", "srcaddr", "dstaddr", "bytes"}
	results := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "srcaddr", Value: "10.0.0.1"},
			{Name: "dstaddr", Value: "10.0.0.2"},
			{Name: "bytes", Value: "1024"},
		},
		{
			{Name: "timestamp", Value: "2023-01-01 12:01:00"},
			{Name: "srcaddr", Value: "10.0.0.3"},
			{Name: "dstaddr", Value: "10.0.0.4"},
			{Name: "bytes", Value: "2048"},
		},
	}

	tests := []struct {
		name       string
		format     string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "table format",
			format:     "table",
			wantPrefix: "+",
			wantErr:    false,
		},
		{
			name:       "csv format",
			format:     "csv",
			wantPrefix: "timestamp,srcaddr,dstaddr,bytes",
			wantErr:    false,
		},
		{
			name:       "json format",
			format:     "json",
			wantPrefix: "[",
			wantErr:    false,
		},
		{
			name:       "unsupported format",
			format:     "xml",
			wantPrefix: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatOptions := FormatOptions{
				Format:   tt.format,
				Colorize: false,
			}
			got, err := Format(results, headers, formatOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("Format() = %v, want prefix %v", got, tt.wantPrefix)
			}
		})
	}
}

func TestTableFormatter(t *testing.T) {
	headers := []string{"timestamp", "srcaddr", "bytes"}
	results := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "srcaddr", Value: "10.0.0.1"},
			{Name: "bytes", Value: "1024"},
		},
	}

	formatter := TableFormatter{MaxWidth: 10, ColorizeAction: false}
	output := formatter.Format(results, headers)

	// Check that output contains expected elements
	if !strings.Contains(output, "timestamp") {
		t.Errorf("TableFormatter.Format() output doesn't contain header: %v", output)
	}
	if !strings.Contains(output, "10.0.0.1") {
		t.Errorf("TableFormatter.Format() output doesn't contain data: %v", output)
	}
	if !strings.Contains(output, "+") {
		t.Errorf("TableFormatter.Format() output doesn't contain table borders: %v", output)
	}

	// Test truncation
	longResults := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "srcaddr", Value: "This is a very long value that should be truncated"},
			{Name: "bytes", Value: "1024"},
		},
	}
	output = formatter.Format(longResults, headers)
	if !strings.Contains(output, "...") {
		t.Errorf("TableFormatter.Format() didn't truncate long value: %v", output)
	}

	// Test empty results
	emptyResults := [][]runner.Field{}
	output = formatter.Format(emptyResults, headers)
	if output != "No results found" {
		t.Errorf("TableFormatter.Format() with empty results = %v, want 'No results found'", output)
	}
}

func TestTableFormatterWithColorization(t *testing.T) {
	headers := []string{"timestamp", "action", "bytes"}
	results := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "action", Value: "ACCEPT"},
			{Name: "bytes", Value: "1024"},
		},
		{
			{Name: "timestamp", Value: "2023-01-01 12:01:00"},
			{Name: "action", Value: "REJECT"},
			{Name: "bytes", Value: "2048"},
		},
	}

	formatter := TableFormatter{MaxWidth: 10, ColorizeAction: true}
	output := formatter.Format(results, headers)

	// Check that output contains color codes
	if !strings.Contains(output, colorGreen) {
		t.Errorf("TableFormatter.Format() with colorization doesn't contain green color code for ACCEPT")
	}
	if !strings.Contains(output, colorRed) {
		t.Errorf("TableFormatter.Format() with colorization doesn't contain red color code for REJECT")
	}
	if !strings.Contains(output, colorReset) {
		t.Errorf("TableFormatter.Format() with colorization doesn't contain color reset code")
	}
}

func TestCSVFormatter(t *testing.T) {
	headers := []string{"timestamp", "srcaddr", "bytes"}
	results := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "srcaddr", Value: "10.0.0.1"},
			{Name: "bytes", Value: "1024"},
		},
	}

	formatter := CSVFormatter{}
	output := formatter.Format(results, headers)

	// Check that output contains expected elements
	expectedFirstLine := "timestamp,srcaddr,bytes"
	if !strings.Contains(output, expectedFirstLine) {
		t.Errorf("CSVFormatter.Format() output doesn't contain headers: %v", output)
	}

	expectedSecondLine := "2023-01-01 12:00:00,10.0.0.1,1024"
	if !strings.Contains(output, expectedSecondLine) {
		t.Errorf("CSVFormatter.Format() output doesn't contain data: %v", output)
	}

	// Test with custom delimiter
	formatter = CSVFormatter{Delimiter: ';'}
	output = formatter.Format(results, headers)
	expectedFirstLineCustom := "timestamp;srcaddr;bytes"
	if !strings.Contains(output, expectedFirstLineCustom) {
		t.Errorf("CSVFormatter.Format() with custom delimiter output doesn't contain headers: %v", output)
	}
}

func TestJSONFormatter(t *testing.T) {
	headers := []string{"timestamp", "srcaddr", "bytes"}
	results := [][]runner.Field{
		{
			{Name: "timestamp", Value: "2023-01-01 12:00:00"},
			{Name: "srcaddr", Value: "10.0.0.1"},
			{Name: "bytes", Value: "1024"},
		},
	}

	// Test with pretty printing
	formatter := JSONFormatter{Pretty: true}
	output := formatter.Format(results, headers)

	// Check that output contains expected elements
	if !strings.Contains(output, "\"timestamp\": \"2023-01-01 12:00:00\"") {
		t.Errorf("JSONFormatter.Format() pretty output doesn't contain formatted data: %v", output)
	}
	if !strings.Contains(output, "\n") {
		t.Errorf("JSONFormatter.Format() pretty output doesn't contain newlines: %v", output)
	}

	// Test without pretty printing
	formatter = JSONFormatter{Pretty: false}
	output = formatter.Format(results, headers)
	if strings.Contains(output, "\n") {
		t.Errorf("JSONFormatter.Format() non-pretty output contains newlines: %v", output)
	}
}
