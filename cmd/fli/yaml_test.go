package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"fli/internal/querybuilder"
	"fli/internal/runner"
)

// Mock QueryExecutor for testing
type MockQueryExecutor struct{}

func (m *MockQueryExecutor) ExecuteQuery(ctx context.Context, cmd *cobra.Command, opts []querybuilder.Option, flags *CommandFlags) ([][]interface{}, runner.QueryStatistics, error) {
	return nil, runner.QueryStatistics{}, nil
}

func TestDryRunYAMLOutput(t *testing.T) {
	// Save original stdout and restore it after the test
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up test flags
	flags := NewCommandFlags()
	flags.LogGroup = "/aws/vpc/flow-logs"
	flags.Since = 1 * time.Hour
	flags.Filter = "dstport=443"
	flags.By = "srcaddr"
	flags.Limit = 50
	flags.Version = 5
	flags.Format = "json"
	flags.QueryTimeout = 2 * time.Minute
	flags.SaveENIs = true
	flags.SaveIPs = false
	flags.NoPtr = true
	flags.ProtoNames = true
	flags.UseColor = false

	// Create a mock command
	cmd := &cobra.Command{Use: "test"}
	args := []string{"count", "srcaddr,dstaddr"}

	// Call the function that handles dry run
	err := handleDryRun(cmd, args, flags)
	if err != nil {
		t.Fatalf("handleDryRun failed: %v", err)
	}

	// Close writer and read the output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = originalStdout
	output := buf.String()

	// Verify the output is valid YAML
	var config QueryConfig
	yamlContent := strings.Join(strings.Split(output, "\n")[2:], "\n") // Skip the comment lines
	err = yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		t.Fatalf("Failed to parse YAML output: %v\nOutput was: %s", err, output)
	}

	// Verify the content matches our input
	if config.Verb != "count" {
		t.Errorf("Expected verb 'count', got '%s'", config.Verb)
	}
	if len(config.Fields) != 2 || config.Fields[0] != "srcaddr" || config.Fields[1] != "dstaddr" {
		t.Errorf("Expected fields [srcaddr dstaddr], got %v", config.Fields)
	}
	if config.LogGroup != "/aws/vpc/flow-logs" {
		t.Errorf("Expected log group '/aws/vpc/flow-logs', got '%s'", config.LogGroup)
	}
	if config.Since != 1*time.Hour {
		t.Errorf("Expected since 1h, got %v", config.Since)
	}
	if config.Filter != "dstport=443" {
		t.Errorf("Expected filter 'dstport=443', got '%s'", config.Filter)
	}
	if config.By != "srcaddr" {
		t.Errorf("Expected by 'srcaddr', got '%s'", config.By)
	}
	if config.Limit != 50 {
		t.Errorf("Expected limit 50, got %d", config.Limit)
	}
	if config.Version != 5 {
		t.Errorf("Expected version 5, got %d", config.Version)
	}
	if config.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", config.Format)
	}
	if config.QueryTimeout != 2*time.Minute {
		t.Errorf("Expected query timeout 2m, got %v", config.QueryTimeout)
	}
	if !config.SaveENIs {
		t.Errorf("Expected SaveENIs to be true")
	}
	if config.SaveIPs {
		t.Errorf("Expected SaveIPs to be false")
	}
}

func TestExecuteFromYAML(t *testing.T) {
	// Create a test YAML file
	yamlContent := `
verb: count
fields:
  - srcaddr
  - dstaddr
log_group: /aws/vpc/flow-logs
since: 1h0m0s
filter: dstport=443
by: srcaddr
limit: 50
version: 5
format: json
query_timeout: 2m0s
save_enis: true
no_ptr: true
proto_names: true
use_color: false
`
	tmpfile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a mock command
	cmd := &cobra.Command{Use: "execute"}
	cmd.Flags().String("file", tmpfile.Name(), "")

	// Save the original NewQueryExecutor and replace it with our mock
	originalNewQueryExecutor := NewQueryExecutor
	NewQueryExecutor = func() QueryExecutorInterface {
		return &MockQueryExecutor{}
	}
	defer func() { NewQueryExecutor = originalNewQueryExecutor }()

	// Call the function that handles execution from YAML
	err = runExecuteCmd(cmd, []string{})
	if err != nil {
		t.Fatalf("runExecuteCmd failed: %v", err)
	}
}

func TestExecuteFromStdin(t *testing.T) {
	// Save original stdin and restore it after the test
	originalStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write YAML to stdin
	yamlContent := `
verb: sum
fields:
  - bytes
log_group: /aws/vpc/flow-logs
since: 30m0s
filter: srcaddr=10.0.0.1
limit: 10
version: 2
`
	go func() {
		w.Write([]byte(yamlContent))
		w.Close()
	}()

	// Create a mock command
	cmd := &cobra.Command{Use: "execute"}
	cmd.Flags().String("file", "-", "")

	// Save the original NewQueryExecutor and replace it with our mock
	originalNewQueryExecutor := NewQueryExecutor
	NewQueryExecutor = func() QueryExecutorInterface {
		return &MockQueryExecutor{}
	}
	defer func() { NewQueryExecutor = originalNewQueryExecutor }()

	// Call the function that handles execution from YAML
	err := runExecuteCmd(cmd, []string{})
	if err != nil {
		t.Fatalf("runExecuteCmd failed: %v", err)
	}
	os.Stdin = originalStdin
}

func TestExecuteValidation(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError string
	}{
		{
			name: "missing verb",
			yamlContent: `
fields:
  - bytes
log_group: /aws/vpc/flow-logs
`,
			expectError: "verb is required",
		},
		{
			name: "missing log group",
			yamlContent: `
verb: count
fields:
  - srcaddr
`,
			expectError: "log_group is required",
		},
		{
			name: "invalid YAML",
			yamlContent: `
verb: count
fields: [
  - this is invalid
`,
			expectError: "failed to parse configuration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temp file with the test YAML
			tmpfile, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tc.yamlContent)); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Create a mock command
			cmd := &cobra.Command{Use: "execute"}
			cmd.Flags().String("file", tmpfile.Name(), "")

			// Call the function that handles execution from YAML
			err = runExecuteCmd(cmd, []string{})
			if err == nil {
				t.Fatalf("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tc.expectError) {
				t.Errorf("Expected error containing '%s', got '%s'", tc.expectError, err.Error())
			}
		})
	}
}
