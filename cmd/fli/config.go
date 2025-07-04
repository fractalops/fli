package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"fli/internal/querybuilder"
)

// QueryConfig represents a single query configuration.
type QueryConfig struct {
	Verb         string        `yaml:"verb"`
	Fields       []string      `yaml:"fields,omitempty"`
	LogGroup     string        `yaml:"log_group"`
	Since        time.Duration `yaml:"since"`
	Filter       string        `yaml:"filter,omitempty"`
	By           string        `yaml:"by,omitempty"`
	Limit        int           `yaml:"limit"`
	Version      int           `yaml:"version"`
	Format       string        `yaml:"format"`
	QueryTimeout time.Duration `yaml:"query_timeout,omitempty"`
	SaveENIs     bool          `yaml:"save_enis,omitempty"`
	SaveIPs      bool          `yaml:"save_ips,omitempty"`
	NoPtr        bool          `yaml:"no_ptr,omitempty"`
	ProtoNames   bool          `yaml:"proto_names,omitempty"`
	UseColor     bool          `yaml:"use_color,omitempty"`
	Name         string        `yaml:"name,omitempty"`
	Description  string        `yaml:"description,omitempty"`
	Tags         []string      `yaml:"tags,omitempty"`
}

// EnhancedQueryConfig represents a query with metadata.
type EnhancedQueryConfig struct {
	Name        string      `yaml:"name,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Tags        []string    `yaml:"tags,omitempty"`
	Config      QueryConfig `yaml:"config"`
}

// QueryCollection represents a collection of queries.
type QueryCollection struct {
	Queries []EnhancedQueryConfig `yaml:"queries"`
}

// handleDryRun outputs the query configuration as YAML without executing the query.
func handleDryRun(_ *cobra.Command, args []string, cmdFlags *CommandFlags) error {
	// If collection flag is set, use the collection handler
	if cmdFlags.Collection {
		return handleDryRunCollection(nil, args, cmdFlags)
	}

	// Parse the verb and fields from args
	if len(args) == 0 {
		return fmt.Errorf("verb is required")
	}

	verb := args[0]
	var fields []string

	if len(args) > 1 {
		fields = parseFields(args[1:])
	}

	// Create config from current command and flags
	config := QueryConfig{
		Verb:         verb,
		Fields:       fields,
		LogGroup:     cmdFlags.LogGroup,
		Since:        cmdFlags.Since,
		Filter:       cmdFlags.Filter,
		By:           cmdFlags.By,
		Limit:        cmdFlags.Limit,
		Version:      cmdFlags.Version,
		Format:       cmdFlags.Format,
		QueryTimeout: cmdFlags.QueryTimeout,
		SaveENIs:     cmdFlags.SaveENIs,
		SaveIPs:      cmdFlags.SaveIPs,
		NoPtr:        cmdFlags.NoPtr,
		ProtoNames:   cmdFlags.ProtoNames,
		UseColor:     cmdFlags.UseColor,
	}

	// Add metadata if provided
	if cmdFlags.QueryName != "" {
		config.Name = cmdFlags.QueryName
	}
	if cmdFlags.QueryDescription != "" {
		config.Description = cmdFlags.QueryDescription
	}
	if cmdFlags.QueryTags != "" {
		config.Tags = strings.Split(cmdFlags.QueryTags, ",")
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	fmt.Println("# FLI Query Configuration")
	fmt.Println("# Save this to a file or pipe to 'fli execute -f -'")
	fmt.Println(string(yamlData))

	return nil
}

// handleDryRunCollection outputs the query configuration as a YAML collection.
func handleDryRunCollection(_ *cobra.Command, args []string, cmdFlags *CommandFlags) error {
	// Parse the verb and fields from args
	if len(args) == 0 {
		return fmt.Errorf("verb is required")
	}

	verb := args[0]
	var fields []string

	if len(args) > 1 {
		fields = parseFields(args[1:])
	}

	// Create config from current command and flags
	config := QueryConfig{
		Verb:         verb,
		Fields:       fields,
		LogGroup:     cmdFlags.LogGroup,
		Since:        cmdFlags.Since,
		Filter:       cmdFlags.Filter,
		By:           cmdFlags.By,
		Limit:        cmdFlags.Limit,
		Version:      cmdFlags.Version,
		Format:       cmdFlags.Format,
		QueryTimeout: cmdFlags.QueryTimeout,
		SaveENIs:     cmdFlags.SaveENIs,
		SaveIPs:      cmdFlags.SaveIPs,
		NoPtr:        cmdFlags.NoPtr,
		ProtoNames:   cmdFlags.ProtoNames,
		UseColor:     cmdFlags.UseColor,
	}

	// Create a collection with a single query
	var tags []string
	if cmdFlags.QueryTags != "" {
		tags = strings.Split(cmdFlags.QueryTags, ",")
	}

	collection := QueryCollection{
		Queries: []EnhancedQueryConfig{
			{
				Name:        cmdFlags.QueryName,
				Description: cmdFlags.QueryDescription,
				Tags:        tags,
				Config:      config,
			},
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(collection)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	fmt.Println("# FLI Query Collection")
	fmt.Println("# Save this to a file or pipe to 'fli execute -f -'")
	fmt.Println(string(yamlData))

	return nil
}

// executeQueryConfig executes a single query configuration.
func executeQueryConfig(cmd *cobra.Command, config QueryConfig) error {
	// Validate required fields
	if config.Verb == "" {
		return fmt.Errorf("verb is required in configuration")
	}
	if config.LogGroup == "" {
		return fmt.Errorf("log_group is required in configuration")
	}

	// Convert back to command arguments and flags
	cmdArgs := []string{config.Verb}
	if len(config.Fields) > 0 {
		cmdArgs = append(cmdArgs, strings.Join(config.Fields, ","))
	}

	// Set flags from config
	cmdFlags := NewCommandFlags()
	cmdFlags.LogGroup = config.LogGroup
	cmdFlags.Since = config.Since
	cmdFlags.Filter = config.Filter
	cmdFlags.By = config.By
	cmdFlags.Limit = config.Limit
	cmdFlags.Version = config.Version
	cmdFlags.Format = config.Format
	cmdFlags.QueryTimeout = config.QueryTimeout
	cmdFlags.SaveENIs = config.SaveENIs
	cmdFlags.SaveIPs = config.SaveIPs
	cmdFlags.NoPtr = config.NoPtr
	cmdFlags.ProtoNames = config.ProtoNames
	cmdFlags.UseColor = config.UseColor

	// Execute the query
	schema := &querybuilder.VPCFlowLogsSchema{}
	opts, err := buildCommandOptions(schema, cmdArgs, cmdFlags)
	if err != nil {
		return fmt.Errorf("failed to build command options: %w", err)
	}

	executor := NewQueryExecutor()
	_, _, err = executor.ExecuteQuery(cmd.Context(), cmd, opts, cmdFlags)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// executeQueryCollection executes a collection of queries.
func executeQueryCollection(cmd *cobra.Command, collection QueryCollection) error {
	for i, query := range collection.Queries {
		// Display query metadata
		fmt.Printf("\n=== Executing Query %d: %s ===\n", i+1, query.Name)

		if query.Description != "" {
			fmt.Printf("Description: %s\n", query.Description)
		}

		if len(query.Tags) > 0 {
			fmt.Printf("Tags: %s\n\n", strings.Join(query.Tags, ", "))
		}

		// Execute the query
		if err := executeQueryConfig(cmd, query.Config); err != nil {
			return fmt.Errorf("failed to execute query %d: %w", i+1, err)
		}
	}
	return nil
}

// runExecuteCmd runs a query from a YAML configuration file or stdin.
func runExecuteCmd(cmd *cobra.Command, _ []string) error {
	filePath, _ := cmd.Flags().GetString("file")

	var yamlData []byte
	var err error

	if filePath == "-" {
		// Read from stdin
		yamlData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		// Read from file
		yamlData, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
	}

	// Try to parse as a collection first
	var collection QueryCollection
	err = yaml.Unmarshal(yamlData, &collection)

	if err == nil && len(collection.Queries) > 0 {
		// This is a collection, execute each query
		return executeQueryCollection(cmd, collection)
	}

	// If not a collection, try to parse as a single query
	var config QueryConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Display metadata if available
	if config.Name != "" {
		fmt.Printf("\n=== Executing Query: %s ===\n", config.Name)

		if config.Description != "" {
			fmt.Printf("Description: %s\n", config.Description)
		}

		if len(config.Tags) > 0 {
			fmt.Printf("Tags: %s\n\n", strings.Join(config.Tags, ", "))
		}
	}

	// Execute the query
	return executeQueryConfig(cmd, config)
}
