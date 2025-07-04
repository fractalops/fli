package querybuilder

import (
	"strings"
	"testing"
)

func clean(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected string
	}{
		{
			name:    "default builder",
			options: []Option{},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with filter",
			options: []Option{
				WithFilter(&Eq{Field: "action", Value: "ACCEPT"}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter action = 'ACCEPT'
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with multiple filters",
			options: []Option{
				WithFilter(&And{
					&Eq{Field: "action", Value: "ACCEPT"},
					&Eq{Field: "protocol", Value: "6"},
				}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter action = 'ACCEPT' and protocol = '6'
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with or filter",
			options: []Option{
				WithFilter(&Or{
					&Eq{Field: "action", Value: "ACCEPT"},
					&Eq{Field: "action", Value: "REJECT"},
				}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter (action = 'ACCEPT' or action = 'REJECT')
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with like filter",
			options: []Option{
				WithFilter(&Like{Field: "srcaddr", Value: "10"}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter srcaddr like '10'
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with custom limit",
			options: []Option{
				WithLimit(50),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(*) as flows
| sort flows desc
| limit 50`,
		},
		{
			name: "with complex filter combination",
			options: []Option{
				WithFilter(&And{
					&Eq{Field: "action", Value: "ACCEPT"},
					&Or{
						&Eq{Field: "protocol", Value: "6"},
						&Eq{Field: "protocol", Value: "17"},
					},
				}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter action = 'ACCEPT' and (protocol = '6' or protocol = '17')
| stats count(*) as flows
| sort flows desc
| limit 100`,
		},
		{
			name: "with fields",
			options: []Option{
				WithFields("srcaddr", "dstaddr"),
				WithVerb(VerbRaw),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| fields srcaddr, dstaddr
| limit 100`,
		},
		{
			name: "count with fields",
			options: []Option{
				WithFields("srcaddr"),
				WithVerb(VerbCount),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(srcaddr) as srcaddr_count
| sort srcaddr_count desc
| limit 100`,
		},
		{
			name: "count with fields and group by",
			options: []Option{
				WithFields("bytes"),
				WithGroupBy("srcaddr"),
				WithVerb(VerbCount),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(bytes) as bytes_count by srcaddr
| sort bytes_count desc
| limit 100`,
		},
		{
			name: "sum with fields",
			options: []Option{
				WithVerb(VerbSum),
				WithFields("bytes"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(bytes) as bytes_sum
| sort bytes_sum desc
| limit 100`,
		},
		{
			name: "avg with fields",
			options: []Option{
				WithVerb(VerbAvg),
				WithFields("bytes"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats avg(bytes) as bytes_avg
| sort bytes_avg desc
| limit 100`,
		},
		{
			name: "min with fields",
			options: []Option{
				WithVerb(VerbMin),
				WithFields("bytes"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats min(bytes) as bytes_min
| sort bytes_min desc
| limit 100`,
		},
		{
			name: "max with fields",
			options: []Option{
				WithVerb(VerbMax),
				WithFields("bytes"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats max(bytes) as bytes_max
| sort bytes_max desc
| limit 100`,
		},
		{
			name: "duration computed field with sum",
			options: []Option{
				WithVerb(VerbSum),
				WithFields("duration"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(end - start) as duration_sum
| sort duration_sum desc
| limit 100`,
		},
		{
			name: "duration computed field with avg",
			options: []Option{
				WithVerb(VerbAvg),
				WithFields("duration"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats avg(end - start) as duration_avg
| sort duration_avg desc
| limit 100`,
		},
		{
			name: "duration computed field with raw verb",
			options: []Option{
				WithVerb(VerbRaw),
				WithFields("duration"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| fields end - start as duration
| limit 100`,
		},
		{
			name: "duration computed field with group by",
			options: []Option{
				WithVerb(VerbAvg),
				WithFields("duration"),
				WithGroupBy("srcaddr"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats avg(end - start) as duration_avg by srcaddr
| sort duration_avg desc
| limit 100`,
		},
		{
			name: "duration computed field in group by",
			options: []Option{
				WithVerb(VerbSum),
				WithFields("bytes"),
				WithGroupBy("srcaddr", "duration"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(bytes) as bytes_sum by srcaddr, end - start as duration
| sort bytes_sum desc
| limit 100`,
		},
		// Multi-field aggregation tests
		{
			name: "multiple count aggregations",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "dstaddr", Verb: VerbCount},
				),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(srcaddr) as srcaddr_count, count(dstaddr) as dstaddr_count
| sort srcaddr_count desc
| limit 100`,
		},
		{
			name: "multiple sum aggregations",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbSum},
				),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(bytes) as bytes_sum, sum(packets) as packets_sum
| sort bytes_sum desc
| limit 100`,
		},
		{
			name: "mixed aggregation types",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbAvg},
				),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(srcaddr) as srcaddr_count, sum(bytes) as bytes_sum, avg(packets) as packets_avg
| sort srcaddr_count desc
| limit 100`,
		},
		{
			name: "multiple aggregations with group by",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbSum},
				),
				WithGroupBy("srcaddr", "dstaddr"),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(bytes) as bytes_sum, sum(packets) as packets_sum by srcaddr, dstaddr
| sort bytes_sum desc
| limit 100`,
		},
		{
			name: "multiple aggregations with computed fields",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "duration", Verb: VerbSum},
					AggregationField{Field: "bytes", Verb: VerbAvg},
				),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(end - start) as duration_sum, avg(bytes) as bytes_avg
| sort duration_sum desc
| limit 100`,
		},
		{
			name: "count star with other aggregations",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "*", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
				),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(*) as flows, sum(bytes) as bytes_sum
| sort flows desc
| limit 100`,
		},
		{
			name: "multiple aggregations with filters",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
				),
				WithFilter(&Eq{Field: "action", Value: "ACCEPT"}),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter action = 'ACCEPT'
| stats count(srcaddr) as srcaddr_count, sum(bytes) as bytes_sum
| sort srcaddr_count desc
| limit 100`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &VPCFlowLogsSchema{}
			b, err := New(schema, tt.options...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := clean(b.String())
			expected := clean(tt.expected)
			if got != expected {
				t.Errorf("New() = %v, want %v", got, expected)
			}
		})
	}
}

// TestWithAggregations tests the WithAggregations function specifically
func TestWithAggregations(t *testing.T) {
	schema := &VPCFlowLogsSchema{}

	tests := []struct {
		name           string
		aggregations   []AggregationField
		expectErr      bool
		expectedErrStr string
	}{
		{
			name: "valid single aggregation",
			aggregations: []AggregationField{
				{Field: "bytes", Verb: VerbSum},
			},
			expectErr: false,
		},
		{
			name: "valid multiple aggregations",
			aggregations: []AggregationField{
				{Field: "srcaddr", Verb: VerbCount},
				{Field: "bytes", Verb: VerbSum},
				{Field: "packets", Verb: VerbAvg},
			},
			expectErr: false,
		},
		{
			name: "invalid field for numeric verb",
			aggregations: []AggregationField{
				{Field: "srcaddr", Verb: VerbSum},
			},
			expectErr:      true,
			expectedErrStr: "field 'srcaddr' must be numeric for verb 'VerbSum'",
		},
		{
			name: "invalid field",
			aggregations: []AggregationField{
				{Field: "nonexistent", Verb: VerbCount},
			},
			expectErr:      true,
			expectedErrStr: "invalid field 'nonexistent'",
		},
		{
			name: "count with non-numeric field is valid",
			aggregations: []AggregationField{
				{Field: "srcaddr", Verb: VerbCount},
			},
			expectErr: false,
		},
		{
			name: "mixed valid and invalid",
			aggregations: []AggregationField{
				{Field: "bytes", Verb: VerbSum},
				{Field: "srcaddr", Verb: VerbSum}, // Invalid: non-numeric field with sum
			},
			expectErr:      true,
			expectedErrStr: "field 'srcaddr' must be numeric for verb 'VerbSum'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(schema, WithAggregations(tt.aggregations...))
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, but got none")
				}
				if tt.expectedErrStr != "" && !strings.Contains(err.Error(), tt.expectedErrStr) {
					t.Errorf("expected error string '%s', but got '%s'", tt.expectedErrStr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestAggregationFieldGetAlias tests the getAlias method
func TestAggregationFieldGetAlias(t *testing.T) {
	tests := []struct {
		name     string
		field    AggregationField
		expected string
	}{
		{
			name:     "count star",
			field:    AggregationField{Field: "*", Verb: VerbCount},
			expected: "flows",
		},
		{
			name:     "count field",
			field:    AggregationField{Field: "srcaddr", Verb: VerbCount},
			expected: "srcaddr_count",
		},
		{
			name:     "sum field",
			field:    AggregationField{Field: "bytes", Verb: VerbSum},
			expected: "bytes_sum",
		},
		{
			name:     "avg field",
			field:    AggregationField{Field: "packets", Verb: VerbAvg},
			expected: "packets_avg",
		},
		{
			name:     "min field",
			field:    AggregationField{Field: "duration", Verb: VerbMin},
			expected: "duration_min",
		},
		{
			name:     "max field",
			field:    AggregationField{Field: "bytes", Verb: VerbMax},
			expected: "bytes_max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.field.getAlias()
			if got != tt.expected {
				t.Errorf("getAlias() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	cases := []struct {
		field string
		want  bool
	}{
		{"srcport", true},
		{"dstport", true},
		{"protocol", true},
		{"packets", true},
		{"bytes", true},
		{"start", true},
		{"end", true},
		{"duration", true},
		{"srcaddr", false},
		{"dstaddr", false},
		{"action", false},
		{"unknown", false},
	}
	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			if got := schema.IsNumeric(c.field); got != c.want {
				t.Errorf("IsNumeric(%q) = %v, want %v", c.field, got, c.want)
			}
		})
	}
}
