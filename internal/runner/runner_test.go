// Package runner provides functionality to execute CloudWatch Logs Insights queries
package runner

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// mockCloudWatchLogsClient implements the CloudWatch Logs client interface for testing
type mockCloudWatchLogsClient struct {
	StartQueryFunc      func(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error)
	GetQueryResultsFunc func(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error)
}

func (m *mockCloudWatchLogsClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	return m.StartQueryFunc(ctx, params, optFns...)
}

func (m *mockCloudWatchLogsClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	return m.GetQueryResultsFunc(ctx, params, optFns...)
}

func TestRunnerRun(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		logGroup       string
		startTime      int64
		endTime        int64
		mockStartResp  *cloudwatchlogs.StartQueryOutput
		mockStartErr   error
		mockResultResp *cloudwatchlogs.GetQueryResultsOutput
		mockResultErr  error
		want           QueryResult
		wantErr        bool
	}{
		{
			name:      "successful query",
			query:     "stats count(*) by srcaddr",
			logGroup:  "/aws/vpc/flowlogs",
			startTime: 1609459200, // 2021-01-01 00:00:00
			endTime:   1609545600, // 2021-01-02 00:00:00
			mockStartResp: &cloudwatchlogs.StartQueryOutput{
				QueryId: stringPtr("query-123"),
			},
			mockStartErr: nil,
			mockResultResp: &cloudwatchlogs.GetQueryResultsOutput{
				Status: types.QueryStatusComplete,
				Results: [][]types.ResultField{
					{
						{Field: stringPtr("srcaddr"), Value: stringPtr("10.0.0.1")},
						{Field: stringPtr("count"), Value: stringPtr("42")},
					},
					{
						{Field: stringPtr("srcaddr"), Value: stringPtr("10.0.0.2")},
						{Field: stringPtr("count"), Value: stringPtr("17")},
					},
				},
				Statistics: &types.QueryStatistics{
					BytesScanned:   1024.0,
					RecordsMatched: 59.0,
					RecordsScanned: 100.0,
				},
			},
			mockResultErr: nil,
			want: QueryResult{
				Results: [][]Field{
					{
						{Name: "srcaddr", Value: "10.0.0.1"},
						{Name: "count", Value: "42"},
					},
					{
						{Name: "srcaddr", Value: "10.0.0.2"},
						{Name: "count", Value: "17"},
					},
				},
				Statistics: QueryStatistics{
					BytesScanned:   1024,
					RecordsMatched: 59,
					RecordsScanned: 100,
				},
			},
			wantErr: false,
		},
		{
			name:          "start query error",
			query:         "stats count(*) by srcaddr",
			logGroup:      "/aws/vpc/flowlogs",
			startTime:     1609459200,
			endTime:       1609545600,
			mockStartResp: nil,
			mockStartErr:  fmt.Errorf("start query error"),
			mockResultResp: &cloudwatchlogs.GetQueryResultsOutput{
				Status: types.QueryStatusComplete,
				Results: [][]types.ResultField{
					{
						{Field: stringPtr("srcaddr"), Value: stringPtr("10.0.0.1")},
						{Field: stringPtr("count"), Value: stringPtr("42")},
					},
				},
			},
			mockResultErr: nil,
			want:          QueryResult{},
			wantErr:       true,
		},
		{
			name:      "get results error",
			query:     "stats count(*) by srcaddr",
			logGroup:  "/aws/vpc/flowlogs",
			startTime: 1609459200,
			endTime:   1609545600,
			mockStartResp: &cloudwatchlogs.StartQueryOutput{
				QueryId: stringPtr("query-123"),
			},
			mockStartErr:   nil,
			mockResultResp: nil,
			mockResultErr:  fmt.Errorf("get results error"),
			want:           QueryResult{},
			wantErr:        true,
		},
		{
			name:      "query failed status",
			query:     "stats count(*) by srcaddr",
			logGroup:  "/aws/vpc/flowlogs",
			startTime: 1609459200,
			endTime:   1609545600,
			mockStartResp: &cloudwatchlogs.StartQueryOutput{
				QueryId: stringPtr("query-123"),
			},
			mockStartErr: nil,
			mockResultResp: &cloudwatchlogs.GetQueryResultsOutput{
				Status: types.QueryStatusFailed,
			},
			mockResultErr: nil,
			want:          QueryResult{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockCloudWatchLogsClient{
				StartQueryFunc: func(_ context.Context, _ *cloudwatchlogs.StartQueryInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
					return tt.mockStartResp, tt.mockStartErr
				},
				GetQueryResultsFunc: func(_ context.Context, _ *cloudwatchlogs.GetQueryResultsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
					return tt.mockResultResp, tt.mockResultErr
				},
			}

			// Create runner with mock client
			r := &Runner{
				Client:       mockClient,
				PollInterval: 1 * time.Millisecond, // Use short poll interval for tests
			}

			// Run the query
			got, err := r.Run(context.Background(), tt.logGroup, tt.query, tt.startTime, tt.endTime)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Runner.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, don't check the results
			if tt.wantErr {
				return
			}

			// Check results
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Runner.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for creating pointers to primitives
func stringPtr(s string) *string {
	return &s
}
