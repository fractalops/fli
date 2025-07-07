package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/spf13/cobra"

	"fli/internal/formatter"
	"fli/internal/querybuilder"
	"fli/internal/runner"
)

// QueryExecutorInterface defines the interface for query execution.
type QueryExecutorInterface interface {
	ExecuteQuery(ctx context.Context, cmd *cobra.Command, opts []querybuilder.Option, flags *CommandFlags) ([][]interface{}, runner.QueryStatistics, error)
}

// QueryExecutor handles the execution of CloudWatch Logs Insights queries.
type QueryExecutor struct {
	client *cloudwatchlogs.Client
	runner *runner.Runner
}

// NewQueryExecutor creates a new QueryExecutor.
var NewQueryExecutor = func() QueryExecutorInterface {
	return &QueryExecutor{}
}

// ExecuteQuery handles the common query execution flow.
func (e *QueryExecutor) ExecuteQuery(ctx context.Context, _ *cobra.Command, opts []querybuilder.Option, cmdFlags *CommandFlags) ([][]interface{}, runner.QueryStatistics, error) {
	// Calculate time range
	end := time.Now()
	start := end.Add(-cmdFlags.Since)

	// Build query
	schema := &querybuilder.VPCFlowLogsSchema{}
	b, err := querybuilder.New(schema, opts...)
	if err != nil {
		return nil, runner.QueryStatistics{}, fmt.Errorf("failed to build query: %w", err)
	}
	query := b.String()

	// Enhanced dry-run mode - output YAML configuration
	if cmdFlags.DryRun {
		if err := handleDryRunFromQuery(query, opts, cmdFlags); err != nil {
			return nil, runner.QueryStatistics{}, fmt.Errorf("failed to generate dry run output: %w", err)
		}
		return nil, runner.QueryStatistics{}, nil
	}

	// Validate log group
	if cmdFlags.LogGroup == "" {
		return nil, runner.QueryStatistics{}, fmt.Errorf("log group is required")
	}

	// Initialize AWS client if not already initialized
	if e.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, runner.QueryStatistics{}, fmt.Errorf("failed to load AWS config: %w", err)
		}
		e.client = cloudwatchlogs.NewFromConfig(cfg)
	}

	// Initialize runner if not already initialized
	if e.runner == nil {
		e.runner = runner.New(e.client)
	}

	// Execute query
	queryResult, err := e.runner.Run(ctx, cmdFlags.LogGroup, query, start.Unix()*MillisecondsPerSecond, end.Unix()*MillisecondsPerSecond)
	if err != nil {
		return nil, runner.QueryStatistics{}, fmt.Errorf("failed to execute query: %w", err)
	}

	// Convert runner.Field to interface{} for the interface
	interfaceResults := make([][]interface{}, len(queryResult.Results))
	for i, row := range queryResult.Results {
		interfaceResults[i] = make([]interface{}, len(row))
		for j, field := range row {
			interfaceResults[i][j] = field
		}
	}

	return interfaceResults, queryResult.Statistics, nil
}

// handleDryRunFromQuery extracts verb and fields from a query string and handles dry run output.
func handleDryRunFromQuery(query string, _ []querybuilder.Option, cmdFlags *CommandFlags) error {
	// Output YAML configuration with the actual query
	if _, err := fmt.Println("# FLI Query Configuration"); err != nil {
		return fmt.Errorf("failed to write dry run output: %w", err)
	}
	if _, err := fmt.Println("# Save this to a file or pipe to 'fli execute -f -'"); err != nil {
		return fmt.Errorf("failed to write dry run output: %w", err)
	}

	// Use a simple YAML-like output without external dependencies
	output := fmt.Sprintf(`verb: %s
log_group: %s
since: %s
limit: %d
version: %d
format: %s
query_timeout: %s
no_ptr: %t
proto_names: %t
use_color: %t`,
		extractVerbFromQuery(query), cmdFlags.LogGroup, cmdFlags.Since, cmdFlags.Limit,
		cmdFlags.Version, cmdFlags.Format, cmdFlags.QueryTimeout,
		cmdFlags.NoPtr, cmdFlags.ProtoNames, cmdFlags.UseColor)

	if cmdFlags.Filter != "" {
		output += fmt.Sprintf("\nfilter: %s", cmdFlags.Filter)
	}
	if cmdFlags.By != "" {
		output += fmt.Sprintf("\nby: %s", cmdFlags.By)
	}
	output += fmt.Sprintf("\nquery: %s", query)

	if _, err := fmt.Println(output); err != nil {
		return fmt.Errorf("failed to write dry run output: %w", err)
	}
	return nil
}

// extractVerbFromQuery extracts just the verb from a query string.
func extractVerbFromQuery(query string) string {
	if strings.Contains(query, "stats") {
		// This is an aggregation query
		return extractAggregationVerb(query)
	}
	// This is a raw query
	return "raw"
}

// extractAggregationVerb determines the aggregation verb from a query string.
func extractAggregationVerb(query string) string {
	if strings.Contains(query, "count(") {
		return "count"
	} else if strings.Contains(query, "sum(") {
		return "sum"
	} else if strings.Contains(query, "avg(") {
		return "avg"
	} else if strings.Contains(query, "min(") {
		return "min"
	} else if strings.Contains(query, "max(") {
		return "max"
	}
	return ""
}

// runVerb executes a query based on the verb and flags.
func runVerb(verb querybuilder.Verb) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Get command flags
		cmdFlags := flags // Use the global flags for now, but pass it as a parameter

		verbStr := strings.ToLower(strings.TrimPrefix(verb.String(), "Verb"))
		allArgs := append([]string{verbStr}, args...)
		schema := &querybuilder.VPCFlowLogsSchema{}
		opts, err := buildCommandOptions(schema, allArgs, cmdFlags)
		if err != nil {
			return err
		}

		// Regular single query execution
		results, stats, err := executeQuery(cmd.Context(), cmd, opts, cmdFlags)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		// If this is a dry run, we're done
		if cmdFlags.DryRun {
			return nil
		}

		// Convert interface{} results back to runner.Field
		fieldResults := make([][]runner.Field, len(results))
		for i, row := range results {
			fieldResults[i] = make([]runner.Field, len(row))
			for j, field := range row {
				if f, ok := field.(runner.Field); ok {
					fieldResults[i][j] = f
				}
			}
		}

		// Enrich results with message data
		enrichedResults := formatter.EnrichResultsWithMessageData(fieldResults)

		// Automatically enrich with annotations if the cache exists.
		cachePath, err := expandPath(DefaultCachePath)
		if err != nil {
			// This is unlikely, but handle it. Don't annotate.
			fmt.Fprintf(os.Stderr, "Warning: could not expand cache path: %v\n", err)
		} else {
			// Attempt to annotate. If it fails, print a warning and continue.
			annotatedResults, err := formatter.EnrichResultsWithAnnotations(enrichedResults, cachePath)
			if err != nil {
				// Non-fatal error, just print to stderr and continue
				fmt.Fprintf(os.Stderr, "Warning: Failed to enrich results with annotations: %v\n", err)
			} else {
				// If successful, use the annotated results.
				enrichedResults = annotatedResults
			}
		}

		// Handle cases where there are no results to display
		if len(enrichedResults) == 0 {
			if !cmdFlags.DryRun {
				if _, err := fmt.Fprintln(os.Stdout, "No results found."); err != nil {
					return fmt.Errorf("failed to write to stdout: %w", err)
				}
			}
			return nil
		}

		// Build headers from enriched results
		headers := make([]string, 0, len(enrichedResults[0]))
		for _, field := range enrichedResults[0] {
			if field.Name != "@ptr" {
				headers = append(headers, field.Name)
			}
		}

		// Format options
		formatOptions := formatter.FormatOptions{
			Format:        cmdFlags.Format,
			Colorize:      cmdFlags.UseColor,
			UseProtoNames: cmdFlags.ProtoNames,
			Debug:         cmdFlags.Debug,
		}

		// Format the results with statistics
		output, err := formatter.FormatWithStats(enrichedResults, headers, formatOptions, stats)
		if err != nil {
			return fmt.Errorf("failed to format results: %w", err)
		}

		if _, err := fmt.Fprint(os.Stdout, output); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
		return nil
	}
}

// For testing.
var executeQuery = func(ctx context.Context, cmd *cobra.Command, opts []querybuilder.Option, flags *CommandFlags) ([][]interface{}, runner.QueryStatistics, error) {
	executor := NewQueryExecutor()
	return executor.ExecuteQuery(ctx, cmd, opts, flags)
}
