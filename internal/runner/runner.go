// Package runner provides functionality to execute CloudWatch Logs Insights queries
// and process their results. It handles query execution, polling, and result retrieval.
package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"fli/internal/config"
)

// Field represents a single field in a query result.
type Field struct {
	// Name is the field name
	Name string

	// Value is the field value as a string
	Value string
}

// QueryStatistics represents statistics about a query execution.
type QueryStatistics struct {
	BytesScanned   int64
	RecordsScanned int64
	RecordsMatched int64
}

// QueryResult contains the results and statistics of a query execution.
type QueryResult struct {
	Results    [][]Field
	Statistics QueryStatistics
}

// CloudWatchLogsClient defines the interface for CloudWatch Logs client operations
// This interface allows for easier testing with mock implementations.
type CloudWatchLogsClient interface {
	StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error)
	GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error)
}

// Runner handles the execution of CloudWatch Logs queries.
type Runner struct {
	// Client is the CloudWatch Logs client used to execute queries
	Client CloudWatchLogsClient

	// PollInterval is the time to wait between query status checks (defaults to 500ms if not set)
	PollInterval time.Duration
}

// New creates a new Runner instance with the given CloudWatch Logs client.
func New(client CloudWatchLogsClient) *Runner {
	return &Runner{
		Client:       client,
		PollInterval: 500 * time.Millisecond,
	}
}

// Run executes a CloudWatch Logs query and returns the results
// Parameters:
// - ctx: Context for the query execution
// - lg: The CloudWatch Logs group name
// - q: The CloudWatch Logs Insights query string
// - start: The start time for the query (Unix timestamp)
// - end: The end time for the query (Unix timestamp)
//
// Returns:
// - A QueryResult containing results and statistics
// - Any error that occurred during query execution.
func (r *Runner) Run(ctx context.Context, lg string, q string, start, end int64) (QueryResult, error) {
	// Start the query
	startResp, err := r.Client.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		LogGroupIdentifiers: []string{lg},
		QueryString:         &q,
		StartTime:           &start,
		EndTime:             &end,
	})
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to start query: %w", err)
	}

	// Wait for query completion
	queryID := startResp.QueryId
	var results [][]Field
	var stats QueryStatistics

	initialPollInterval := r.PollInterval
	if initialPollInterval == 0 {
		initialPollInterval = 500 * time.Millisecond
	}
	pollInterval := initialPollInterval
	timeouts := config.DefaultTimeouts()
	maxPollInterval := timeouts.MaxPoll

	// Track query execution time
	startTime := time.Now()
	longQueryThreshold := 30 * time.Second
	longQueryWarningDisplayed := false

	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return QueryResult{}, fmt.Errorf("query cancelled by context: %w", ctx.Err())
		default:
			// Continue with query
		}

		// Display message for long-running queries
		if time.Since(startTime) > longQueryThreshold && !longQueryWarningDisplayed {
			fmt.Fprintln(os.Stderr, "Query is taking longer than expected. Still waiting for results...")
			longQueryWarningDisplayed = true
		}

		// Check query status
		status, err := r.Client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: queryID,
		})
		if err != nil {
			return QueryResult{}, fmt.Errorf("failed to get query results: %w", err)
		}

		// Extract statistics if available
		if status.Statistics != nil {
			stats.BytesScanned = int64(status.Statistics.BytesScanned)
			stats.RecordsMatched = int64(status.Statistics.RecordsMatched)
			stats.RecordsScanned = int64(status.Statistics.RecordsScanned)
		}

		// Check if query is complete
		switch status.Status {
		case types.QueryStatusComplete:
			// Process results
			results = make([][]Field, len(status.Results))
			for i, row := range status.Results {
				fields := make([]Field, len(row))
				for j, field := range row {
					// Get field value and clean up any @ptr references
					fieldValue := *field.Value
					fieldName := *field.Field

					fields[j] = Field{
						Name:  fieldName,
						Value: fieldValue,
					}
				}
				results[i] = fields
			}
			return QueryResult{
				Results:    results,
				Statistics: stats,
			}, nil

		case types.QueryStatusFailed:
			return QueryResult{}, fmt.Errorf("query execution failed")

		case types.QueryStatusCancelled:
			return QueryResult{}, fmt.Errorf("query was cancelled")

		case types.QueryStatusTimeout:
			return QueryResult{}, fmt.Errorf("query execution timed out")

		case types.QueryStatusUnknown:
			return QueryResult{}, fmt.Errorf("query status is unknown")

		case types.QueryStatusRunning, types.QueryStatusScheduled:
			// Wait before checking again, with exponential back-off
			select {
			case <-ctx.Done():
				return QueryResult{}, fmt.Errorf("query cancelled by context: %w", ctx.Err())
			case <-time.After(pollInterval):
				// Exponential back-off, capped at maxPollInterval
				pollInterval *= 2
				if pollInterval > maxPollInterval {
					pollInterval = maxPollInterval
				}
				continue
			}

		default:
			return QueryResult{}, fmt.Errorf("unknown query status: %s", status.Status)
		}
	}
}
