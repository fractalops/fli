// Package querybuilder provides tools for building CloudWatch Logs Insights queries.
package querybuilder

import "fmt"

// VPCFlowLogsSchema implements the Schema interface for VPC Flow Logs.
type VPCFlowLogsSchema struct{}

// Constants for VPC Flow Logs.
const (
	// DefaultVersion is the default VPC Flow Log version to use.
	DefaultVersion = 2
	// ParsePatternV2 is the parse pattern for VPC Flow Logs version 2.
	ParsePatternV2 = `parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status`
	// ParsePatternV3 is the parse pattern for VPC Flow Logs version 3.
	ParsePatternV3 = `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path`
	// ParsePatternV5 is the parse pattern for VPC Flow Logs version 5.
	ParsePatternV5 = `parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path`
)

// versionFields maps flow log versions to their valid fields.
var versionFields = map[int][]string{
	2: {
		"version", "account_id", "interface_id", "srcaddr", "dstaddr",
		"srcport", "dstport", "protocol", "packets", "bytes",
		"start", "end", "action", "log_status",
	},
	3: {
		"version", "account_id", "interface_id", "srcaddr", "dstaddr",
		"srcport", "dstport", "protocol", "packets", "bytes",
		"start", "end", "action", "log_status", "vpc_id", "subnet_id",
		"instance_id", "tcp_flags", "type", "pkt_srcaddr", "pkt_dstaddr",
		"region", "az_id", "sublocation_type", "sublocation_id",
		"pkt_src_aws_service", "pkt_dst_aws_service", "flow_direction",
		"traffic_path",
	},
	5: {
		"version", "account_id", "interface_id", "srcaddr", "dstaddr",
		"srcport", "dstport", "protocol", "packets", "bytes",
		"start", "end", "action", "log_status", "vpc_id", "subnet_id",
		"instance_id", "tcp_flags", "type", "pkt_srcaddr", "pkt_dstaddr",
		"region", "az_id", "sublocation_type", "sublocation_id",
		"pkt_src_aws_service", "pkt_dst_aws_service", "flow_direction",
		"traffic_path",
	},
}

// GetParsePattern returns the 'parse' statement pattern for a given log version.
func (s *VPCFlowLogsSchema) GetParsePattern(version int) (string, error) {
	switch version {
	case 2:
		return ParsePatternV2, nil
	case 3:
		return ParsePatternV3, nil
	case 5:
		return ParsePatternV5, nil
	default:
		return "", fmt.Errorf("unsupported VPC Flow Log version for parse pattern: %d", version)
	}
}

// ValidateField checks if a field is valid for the given log version.
func (s *VPCFlowLogsSchema) ValidateField(field string, version int) error {
	validFields, ok := versionFields[version]
	if !ok {
		return fmt.Errorf("invalid flow log version: %d", version)
	}

	// Allow computed fields.
	if field == "duration" || field == "*" {
		return nil
	}

	for _, f := range validFields {
		if f == field {
			return nil
		}
	}
	return fmt.Errorf("invalid field '%s' for version %d", field, version)
}

// ValidateVersion checks if a version number is supported by the schema.
func (s *VPCFlowLogsSchema) ValidateVersion(version int) error {
	if _, ok := versionFields[version]; !ok {
		return fmt.Errorf("invalid flow log version: %d", version)
	}
	return nil
}

// GetDefaultVersion returns the default version for the schema.
func (s *VPCFlowLogsSchema) GetDefaultVersion() int {
	return DefaultVersion
}

// IsNumeric returns true if the field is known to be numeric.
func (s *VPCFlowLogsSchema) IsNumeric(field string) bool {
	numericFields := map[string]bool{
		"srcport":  true,
		"dstport":  true,
		"protocol": true,
		"packets":  true,
		"bytes":    true,
		"start":    true,
		"end":      true,
		"duration": true, // This is a computed field.
	}
	return numericFields[field]
}

// GetComputedFieldExpression returns the CloudWatch Logs Insights expression for a computed field.
// Returns empty string if the field is not a computed field.
func (s *VPCFlowLogsSchema) GetComputedFieldExpression(field string, _ int) string {
	switch field {
	case "duration":
		// duration = end - start (in seconds).
		return "end - start"
	default:
		return ""
	}
}
