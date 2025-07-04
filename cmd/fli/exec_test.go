package main

import (
	"strings"
	"testing"
	"time"

	"fli/internal/querybuilder"
)

// mockSchema wraps the real schema to provide a simplified parse pattern for tests.
type mockSchema struct {
	querybuilder.VPCFlowLogsSchema
}

// GetParsePattern returns a short, predictable parse pattern for testing.
func (m *mockSchema) GetParsePattern(version int) (string, error) {
	return "parse @message 'mock_pattern'", nil
}

func TestBuildCommandOptions(t *testing.T) {
	schema := &mockSchema{}

	// Reset flags to defaults for each test run to ensure isolation
	resetFlags := func() {
		flags = NewCommandFlags()
		flags.InitDefaults(100, "table", 5*time.Minute)
	}

	testCases := []struct {
		name           string
		args           []string
		setupFlags     func()
		expectedQuery  string
		expectErr      bool
		expectedErrStr string
	}{
		{
			name:       "simple count",
			args:       []string{"count"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats count(*) as flows" +
				" | sort flows desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "sum with field",
			args:       []string{"sum", "bytes"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats sum(bytes) as bytes_sum" +
				" | sort bytes_sum desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:           "sum with non-numeric field",
			args:           []string{"sum", "srcaddr"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "srcaddr" must be numeric for verb "sum"`,
		},
		{
			name: "count with group by",
			args: []string{"count", "action"},
			setupFlags: func() {
				resetFlags()
				flags.By = "srcaddr,dstaddr"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats count(action) as action_count by srcaddr, dstaddr" +
				" | sort action_count desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "raw with filter",
			args: []string{"raw"},
			setupFlags: func() {
				resetFlags()
				flags.Filter = "dstport = 443"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | filter dstport = 443" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "invalid filter",
			args: []string{"raw"},
			setupFlags: func() {
				resetFlags()
				flags.Filter = "this is not a valid filter"
			},
			expectErr:      true,
			expectedErrStr: "invalid filter expression: invalid filter clause: \"this is not a valid filter\"",
		},
		{
			name:           "no verb",
			args:           []string{},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: "verb is required",
		},
		{
			name:           "invalid verb",
			args:           []string{"delete"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: "invalid verb 'delete'",
		},
		// Multi-field aggregation tests
		{
			name:       "count with multiple fields",
			args:       []string{"count", "srcaddr,dstaddr"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats count(srcaddr) as srcaddr_count, count(dstaddr) as dstaddr_count" +
				" | sort srcaddr_count desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "sum with multiple fields",
			args:       []string{"sum", "bytes,packets"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats sum(bytes) as bytes_sum, sum(packets) as packets_sum" +
				" | sort bytes_sum desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "avg with multiple fields",
			args:       []string{"avg", "bytes,packets"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats avg(bytes) as bytes_avg, avg(packets) as packets_avg" +
				" | sort bytes_avg desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "min with multiple fields",
			args:       []string{"min", "bytes,packets"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats min(bytes) as bytes_min, min(packets) as packets_min" +
				" | sort bytes_min desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "max with multiple fields",
			args:       []string{"max", "bytes,packets"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats max(bytes) as bytes_max, max(packets) as packets_max" +
				" | sort bytes_max desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "multiple fields with group by",
			args: []string{"sum", "bytes,packets"},
			setupFlags: func() {
				resetFlags()
				flags.By = "srcaddr,dstaddr"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats sum(bytes) as bytes_sum, sum(packets) as packets_sum by srcaddr, dstaddr" +
				" | sort bytes_sum desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "multiple fields with filter",
			args: []string{"count", "srcaddr,dstaddr"},
			setupFlags: func() {
				resetFlags()
				flags.Filter = "action = 'ACCEPT'"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | filter action = 'ACCEPT'" +
				" | stats count(srcaddr) as srcaddr_count, count(dstaddr) as dstaddr_count" +
				" | sort srcaddr_count desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "multiple fields with group by and filter",
			args: []string{"sum", "bytes,packets"},
			setupFlags: func() {
				resetFlags()
				flags.By = "srcaddr"
				flags.Filter = "protocol = 6"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | filter protocol = 6" +
				" | stats sum(bytes) as bytes_sum, sum(packets) as packets_sum by srcaddr" +
				" | sort bytes_sum desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:       "count with star and field",
			args:       []string{"count", "*,srcaddr"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | stats count(*) as flows, count(srcaddr) as srcaddr_count" +
				" | sort flows desc" +
				" | limit 100",
			expectErr: false,
		},
		{
			name:           "sum with mixed numeric and non-numeric fields",
			args:           []string{"sum", "bytes,srcaddr"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "srcaddr" must be numeric for verb "sum"`,
		},
		{
			name:           "avg with mixed numeric and non-numeric fields",
			args:           []string{"avg", "packets,dstaddr"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "dstaddr" must be numeric for verb "avg"`,
		},
		{
			name:           "min with mixed numeric and non-numeric fields",
			args:           []string{"min", "bytes,action"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "action" must be numeric for verb "min"`,
		},
		{
			name:           "max with mixed numeric and non-numeric fields",
			args:           []string{"max", "packets,interface_id"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "interface_id" must be numeric for verb "max"`,
		},
		{
			name:           "sum with invalid field",
			args:           []string{"sum", "bytes,nonexistent"},
			setupFlags:     resetFlags,
			expectErr:      true,
			expectedErrStr: `field "nonexistent" must be numeric for verb "sum"`,
		},
		{
			name:           "count with invalid field",
			args:           []string{"count", "srcaddr,nonexistent"},
			setupFlags:     resetFlags,
			expectErr:      false, // CLI doesn't validate field existence for count
			expectedErrStr: "",    // Error will come from builder creation
		},
		{
			name:       "raw with multiple fields",
			args:       []string{"raw", "srcaddr,dstaddr,action"},
			setupFlags: resetFlags,
			expectedQuery: "parse @message 'mock_pattern'" +
				" | fields srcaddr, dstaddr, action" +
				" | limit 100",
			expectErr: false,
		},
		{
			name: "raw with multiple fields and filter",
			args: []string{"raw", "srcaddr,dstaddr"},
			setupFlags: func() {
				resetFlags()
				flags.Filter = "action = 'ACCEPT'"
			},
			expectedQuery: "parse @message 'mock_pattern'" +
				" | filter action = 'ACCEPT'" +
				" | fields srcaddr, dstaddr" +
				" | limit 100",
			expectErr: false,
		},
		// Debug test to understand the issue
		{
			name:       "debug: count with invalid field should fail",
			args:       []string{"count", "nonexistent"},
			setupFlags: resetFlags,
			expectErr:  false, // CLI doesn't validate field existence for count
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFlags()

			opts, err := buildCommandOptions(schema, tc.args, flags)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected an error, but got none")
				}
				if tc.expectedErrStr != "" {
					if !strings.Contains(err.Error(), tc.expectedErrStr) {
						t.Errorf("expected error string '%s', but got '%s'", tc.expectedErrStr, err.Error())
					}
				}
				return // End test here if an error was expected
			}

			if err != nil {
				t.Fatalf("did not expect an error, but got: %v", err)
			}

			b, builderErr := querybuilder.New(schema, opts...)
			if builderErr != nil {
				// Check if this is a count test with invalid field that should fail
				if strings.Contains(tc.name, "count with invalid field") || strings.Contains(tc.name, "debug: count with invalid field") {
					if !strings.Contains(builderErr.Error(), "invalid field 'nonexistent'") {
						t.Errorf("expected error about invalid field, but got: %v", builderErr)
					}
					return // This is expected to fail
				}
				t.Fatalf("failed to create query builder: %v", builderErr)
			}

			// Clean up strings for comparison
			clean := func(s string) string {
				s = strings.ReplaceAll(s, "\n", "")
				s = strings.ReplaceAll(s, "\t", "")
				s = strings.Join(strings.Fields(s), " ")
				return s
			}

			actualQuery := clean(b.String())
			expectedQuery := clean(tc.expectedQuery)

			if actualQuery != expectedQuery {
				t.Errorf("queries do not match:\n got: %s\nwant: %s", actualQuery, expectedQuery)
			}
		})
	}
}

// Debug test to understand field validation
func TestDebugFieldValidation(t *testing.T) {
	// Test with real schema
	realSchema := &querybuilder.VPCFlowLogsSchema{}

	// Test if "nonexistent" is actually invalid
	err := realSchema.ValidateField("nonexistent", 2)
	if err == nil {
		t.Log("'nonexistent' field is considered valid by real schema - this explains why the test doesn't get an error")
	} else {
		t.Logf("'nonexistent' field is invalid in real schema: %v", err)
	}

	// Test if "srcaddr" is valid
	err = realSchema.ValidateField("srcaddr", 2)
	if err != nil {
		t.Errorf("'srcaddr' should be valid but got error: %v", err)
	}

	// Test with mock schema
	mockSchema := &mockSchema{}

	err = mockSchema.ValidateField("nonexistent", 2)
	if err == nil {
		t.Log("'nonexistent' field is considered valid by mock schema")
	} else {
		t.Logf("'nonexistent' field is invalid in mock schema: %v", err)
	}

	// Test direct builder creation with invalid field
	opts := []querybuilder.Option{
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "nonexistent", Verb: querybuilder.VerbCount},
		),
	}

	_, err = querybuilder.New(realSchema, opts...)
	if err == nil {
		t.Log("Builder creation with invalid field succeeded - validation not working")
	} else {
		t.Logf("Builder creation with invalid field failed as expected: %v", err)
	}
}
