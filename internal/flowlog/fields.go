// Package flowlog provides VPC Flow Log field definitions, version detection, and format presets.
package flowlog

import (
	"fmt"
	"strings"
)

// Flow log field names.
const (
	FieldVersion          = "version"
	FieldAccountID        = "account-id"
	FieldInterfaceID      = "interface-id"
	FieldSrcAddr          = "srcaddr"
	FieldDstAddr          = "dstaddr"
	FieldSrcPort          = "srcport"
	FieldDstPort          = "dstport"
	FieldProtocol         = "protocol"
	FieldPackets          = "packets"
	FieldBytes            = "bytes"
	FieldStart            = "start"
	FieldEnd              = "end"
	FieldAction           = "action"
	FieldLogStatus        = "log-status"
	FieldVpcID            = "vpc-id"
	FieldSubnetID         = "subnet-id"
	FieldInstanceID       = "instance-id"
	FieldTCPFlags         = "tcp-flags"
	FieldType             = "type"
	FieldPktSrcAddr       = "pkt-srcaddr"
	FieldPktDstAddr       = "pkt-dstaddr"
	FieldRegion           = "region"
	FieldAZID             = "az-id"
	FieldSublocationType  = "sublocation-type"
	FieldSublocationID    = "sublocation-id"
	FieldPktSrcAWSService = "pkt-src-aws-service"
	FieldPktDstAWSService = "pkt-dst-aws-service"
	FieldFlowDirection    = "flow-direction"
	FieldTrafficPath      = "traffic-path"
)

// V2Fields are the default flow log fields (version 2).
var V2Fields = []string{
	FieldVersion, FieldAccountID, FieldInterfaceID,
	FieldSrcAddr, FieldDstAddr, FieldSrcPort, FieldDstPort,
	FieldProtocol, FieldPackets, FieldBytes,
	FieldStart, FieldEnd, FieldAction, FieldLogStatus,
}

// V3Fields are the fields added in version 3.
var V3Fields = []string{
	FieldVpcID, FieldSubnetID, FieldInstanceID,
	FieldTCPFlags, FieldType, FieldPktSrcAddr, FieldPktDstAddr,
}

// V4Fields are the fields added in version 4.
var V4Fields = []string{
	FieldRegion, FieldAZID, FieldSublocationType, FieldSublocationID,
}

// V5Fields are the fields added in version 5.
var V5Fields = []string{
	FieldPktSrcAWSService, FieldPktDstAWSService,
	FieldFlowDirection, FieldTrafficPath,
}

// AllFields returns all fields across all versions.
func AllFields() []string {
	all := make([]string, 0, len(V2Fields)+len(V3Fields)+len(V4Fields)+len(V5Fields))
	all = append(all, V2Fields...)
	all = append(all, V3Fields...)
	all = append(all, V4Fields...)
	all = append(all, V5Fields...)
	return all
}

// Preset names.
const (
	PresetDefault         = "default"
	PresetSecurity        = "security"
	PresetTroubleshooting = "troubleshooting"
	PresetFull            = "full"
	PresetCustom          = "custom"
)

// PresetFields returns the fields for a given preset name.
func PresetFields(preset string) []string {
	switch preset {
	case PresetDefault:
		return V2Fields
	case PresetSecurity:
		fields := make([]string, 0, len(V2Fields)+6)
		fields = append(fields, V2Fields...)
		fields = append(fields, FieldTCPFlags, FieldPktSrcAddr, FieldPktDstAddr,
			FieldFlowDirection, FieldPktSrcAWSService, FieldPktDstAWSService)
		return fields
	case PresetTroubleshooting:
		fields := make([]string, 0, len(V2Fields)+len(V3Fields)+len(V4Fields))
		fields = append(fields, V2Fields...)
		fields = append(fields, V3Fields...)
		fields = append(fields, V4Fields...)
		return fields
	case PresetFull:
		return AllFields()
	default:
		return V2Fields
	}
}

// PresetVersion returns the minimum version required for a preset.
func PresetVersion(preset string) int {
	switch preset {
	case PresetSecurity, PresetFull:
		return 5
	case PresetTroubleshooting:
		return 4
	default:
		return 2
	}
}

// FormatString builds a flow log format string from field names.
// Example output: "${srcaddr} ${dstaddr} ${srcport}".
func FormatString(fields []string) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprintf("${%s}", f)
	}
	return strings.Join(parts, " ")
}

// Version detection marker sets.
var (
	v5Markers = toSet(V5Fields)
	v4Markers = toSet(V4Fields)
	v3Markers = toSet(V3Fields)
)

// DetectVersion determines the flow log version from a LogFormat string.
// An empty format string indicates the default v2 format.
func DetectVersion(logFormat string) int {
	if logFormat == "" {
		return 2
	}

	fields := parseFormatFields(logFormat)

	for _, f := range fields {
		if v5Markers[f] {
			return 5
		}
	}
	for _, f := range fields {
		if v4Markers[f] {
			return 4
		}
	}
	for _, f := range fields {
		if v3Markers[f] {
			return 3
		}
	}
	return 2
}

// parseFormatFields extracts field names from a format string like "${srcaddr} ${dstaddr}".
func parseFormatFields(format string) []string {
	var fields []string
	for _, token := range strings.Fields(format) {
		token = strings.TrimPrefix(token, "${")
		token = strings.TrimSuffix(token, "}")
		if token != "" {
			fields = append(fields, token)
		}
	}
	return fields
}

func toSet(slice []string) map[string]bool {
	m := make(map[string]bool, len(slice))
	for _, s := range slice {
		m[s] = true
	}
	return m
}
