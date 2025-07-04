package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// TestDryRunYAMLCollectionOutput tests the enhanced dry run output with metadata and collection support
func TestDryRunYAMLCollectionOutput(t *testing.T) {
	// Save original stdout and restore it after the test
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up test flags
	flags := NewCommandFlags()
	flags.LogGroup = "/aws/vpc/flow-logs"
	flags.Since = 3600000000000 // 1 hour in nanoseconds
	flags.Filter = "dstport=443"
	flags.By = "srcaddr"
	flags.Limit = 50
	flags.Version = 5
	flags.Format = "json"
	flags.QueryTimeout = 120000000000 // 2 minutes in nanoseconds
	flags.SaveENIs = true
	flags.SaveIPs = false
	flags.NoPtr = true
	flags.ProtoNames = true
	flags.UseColor = false

	// Add metadata flags
	flags.Collection = true
	flags.QueryName = "HTTPS Traffic Analysis"
	flags.QueryDescription = "Analyzes HTTPS traffic by source IP"
	flags.QueryTags = "security,https,traffic"

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
	var collection QueryCollection
	yamlContent := strings.Join(strings.Split(output, "\n")[2:], "\n") // Skip the comment lines
	err = yaml.Unmarshal([]byte(yamlContent), &collection)
	if err != nil {
		t.Fatalf("Failed to parse YAML output: %v\nOutput was: %s", err, output)
	}

	// Verify the collection has one query
	if len(collection.Queries) != 1 {
		t.Fatalf("Expected 1 query in collection, got %d", len(collection.Queries))
	}

	// Verify the metadata
	query := collection.Queries[0]
	if query.Name != "HTTPS Traffic Analysis" {
		t.Errorf("Expected name 'HTTPS Traffic Analysis', got '%s'", query.Name)
	}
	if query.Description != "Analyzes HTTPS traffic by source IP" {
		t.Errorf("Expected description 'Analyzes HTTPS traffic by source IP', got '%s'", query.Description)
	}
	if len(query.Tags) != 3 || query.Tags[0] != "security" || query.Tags[1] != "https" || query.Tags[2] != "traffic" {
		t.Errorf("Expected tags [security https traffic], got %v", query.Tags)
	}

	// Verify the query configuration
	config := query.Config
	if config.Verb != "count" {
		t.Errorf("Expected verb 'count', got '%s'", config.Verb)
	}
	if len(config.Fields) != 2 || config.Fields[0] != "srcaddr" || config.Fields[1] != "dstaddr" {
		t.Errorf("Expected fields [srcaddr dstaddr], got %v", config.Fields)
	}
	if config.LogGroup != "/aws/vpc/flow-logs" {
		t.Errorf("Expected log group '/aws/vpc/flow-logs', got '%s'", config.LogGroup)
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
	if !config.SaveENIs {
		t.Errorf("Expected SaveENIs to be true")
	}
	if config.SaveIPs {
		t.Errorf("Expected SaveIPs to be false")
	}
}
