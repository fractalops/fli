package querybuilder

import (
	"strings"
	"testing"
)

func TestIntegration_BasicCount(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	b, err := New(schema,
		WithVerb(VerbCount),
		WithGroupBy("srcaddr"),
		WithLimit(10),
		WithFilter(&Eq{Field: "interface_id", Value: "eni-123"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := b.String()
	if !strings.Contains(q, "count(*) as flows by srcaddr") {
		t.Errorf("expected count by srcaddr, got: %s", q)
	}
	if !strings.Contains(q, "filter interface_id = 'eni-123'") {
		t.Errorf("expected filter for eni-123, got: %s", q)
	}
}

func TestIntegration_AdvancedFilters(t *testing.T) {
	// This test simulates a more complex, user-driven query
	filterExpr, err := ParseFilter("(srcaddr like '10' or srcaddr like '172.16') and dstport = 443 and action = 'ACCEPT'")
	if err != nil {
		t.Fatalf("failed to parse filter: %v", err)
	}
	schema := &VPCFlowLogsSchema{}
	b, err := New(schema,
		WithVersion(3),
		WithVerb(VerbSum),
		WithFields("bytes"),
		WithGroupBy("vpc_id", "subnet_id"),
		WithFilter(filterExpr),
		WithLimit(10),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path
| filter (srcaddr like '10' or srcaddr like '172.16') and dstport = 443 and action = 'ACCEPT'
| stats sum(bytes) as bytes_sum by vpc_id, subnet_id
| sort bytes_sum desc
| limit 10`
	if clean(b.String()) != clean(expected) {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), expected)
	}
}

func TestIntegration_VersionSpecific(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	b, err := New(schema,
		WithVersion(3),
		WithVerb(VerbSum),
		WithFields("bytes"),
		WithGroupBy("vpc_id", "subnet_id"),
		WithLimit(5),
		WithFilter(&Eq{Field: "srcaddr", Value: "10.0.0.1"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path
| filter srcaddr = '10.0.0.1'
| stats sum(bytes) as bytes_sum by vpc_id, subnet_id
| sort bytes_sum desc
| limit 5`
	if clean(b.String()) != clean(expected) {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), expected)
	}
}

func TestIntegration_InvalidFieldForVersion(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	_, err := New(schema,
		WithVersion(2),
		WithFields("vpc_id"), // vpc_id is not valid for v2
	)
	if err == nil || !strings.Contains(err.Error(), "invalid field") {
		t.Errorf("expected invalid field error, got: %v", err)
	}
}

func TestIntegration_InvalidVersion(t *testing.T) {
	schema := &VPCFlowLogsSchema{}
	_, err := New(schema,
		WithVersion(99),
	)
	if err == nil || !strings.Contains(err.Error(), "invalid flow log version") {
		t.Errorf("expected invalid version error, got: %v", err)
	}
}

// Multi-field aggregation integration tests
func TestIntegration_MultiFieldAggregations(t *testing.T) {
	schema := &VPCFlowLogsSchema{}

	tests := []struct {
		name     string
		options  []Option
		expected string
	}{
		{
			name: "multiple count aggregations",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "dstaddr", Verb: VerbCount},
				),
				WithLimit(10),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats count(srcaddr) as srcaddr_count, count(dstaddr) as dstaddr_count
| sort srcaddr_count desc
| limit 10`,
		},
		{
			name: "multiple sum aggregations with group by",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbSum},
				),
				WithGroupBy("srcaddr", "dstaddr"),
				WithLimit(5),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(bytes) as bytes_sum, sum(packets) as packets_sum by srcaddr, dstaddr
| sort bytes_sum desc
| limit 5`,
		},
		{
			name: "mixed aggregation types with filter",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbAvg},
				),
				WithFilter(&Eq{Field: "action", Value: "ACCEPT"}),
				WithLimit(10),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter action = 'ACCEPT'
| stats count(srcaddr) as srcaddr_count, sum(bytes) as bytes_sum, avg(packets) as packets_avg
| sort srcaddr_count desc
| limit 10`,
		},
		{
			name: "multiple aggregations with computed fields",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "duration", Verb: VerbSum},
					AggregationField{Field: "bytes", Verb: VerbAvg},
				),
				WithGroupBy("srcaddr"),
				WithLimit(5),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| stats sum(end - start) as duration_sum, avg(bytes) as bytes_avg by srcaddr
| sort duration_sum desc
| limit 5`,
		},
		{
			name: "count star with other aggregations",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "*", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbSum},
				),
				WithFilter(&Eq{Field: "protocol", Value: 6}),
				WithLimit(10),
			},
			expected: `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status
| filter protocol = 6
| stats count(*) as flows, sum(bytes) as bytes_sum, sum(packets) as packets_sum
| sort flows desc
| limit 10`,
		},
		{
			name: "version 3 with multiple aggregations",
			options: []Option{
				WithVersion(3),
				WithAggregations(
					AggregationField{Field: "vpc_id", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
				),
				WithGroupBy("vpc_id", "subnet_id"),
				WithLimit(5),
			},
			expected: `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path
| stats count(vpc_id) as vpc_id_count, sum(bytes) as bytes_sum by vpc_id, subnet_id
| sort vpc_id_count desc
| limit 5`,
		},
		{
			name: "complex multi-field query",
			options: []Option{
				WithVersion(3),
				WithAggregations(
					AggregationField{Field: "srcaddr", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbAvg},
					AggregationField{Field: "duration", Verb: VerbMax},
				),
				WithGroupBy("vpc_id", "subnet_id"),
				WithFilter(&And{
					&Eq{Field: "action", Value: "ACCEPT"},
					&Eq{Field: "protocol", Value: 6},
				}),
				WithLimit(10),
			},
			expected: `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path
| filter action = 'ACCEPT' and protocol = 6
| stats count(srcaddr) as srcaddr_count, sum(bytes) as bytes_sum, avg(packets) as packets_avg, max(end - start) as duration_max by vpc_id, subnet_id
| sort srcaddr_count desc
| limit 10`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := New(schema, tt.options...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := clean(b.String())
			expected := clean(tt.expected)
			if got != expected {
				t.Errorf("got:\n%s\nwant:\n%s", got, expected)
			}
		})
	}
}

func TestIntegration_MultiFieldValidation(t *testing.T) {
	schema := &VPCFlowLogsSchema{}

	tests := []struct {
		name           string
		options        []Option
		expectErr      bool
		expectedErrStr string
	}{
		{
			name: "invalid field in multi-aggregation",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "nonexistent", Verb: VerbCount},
				),
			},
			expectErr:      true,
			expectedErrStr: "invalid field 'nonexistent'",
		},
		{
			name: "non-numeric field with numeric verb",
			options: []Option{
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "srcaddr", Verb: VerbSum},
				),
			},
			expectErr:      true,
			expectedErrStr: "field 'srcaddr' must be numeric for verb 'VerbSum'",
		},
		{
			name: "invalid version with multi-aggregation",
			options: []Option{
				WithVersion(99),
				WithAggregations(
					AggregationField{Field: "bytes", Verb: VerbSum},
					AggregationField{Field: "packets", Verb: VerbSum},
				),
			},
			expectErr:      true,
			expectedErrStr: "invalid version 99",
		},
		{
			name: "field not available in version",
			options: []Option{
				WithVersion(2),
				WithAggregations(
					AggregationField{Field: "vpc_id", Verb: VerbCount},
					AggregationField{Field: "bytes", Verb: VerbSum},
				),
			},
			expectErr:      true,
			expectedErrStr: "invalid field 'vpc_id'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(schema, tt.options...)
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
