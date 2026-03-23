package flowlog

import (
	"testing"
)

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected int
	}{
		{"empty format is v2", "", 2},
		{"default fields only", "${srcaddr} ${dstaddr} ${srcport} ${dstport}", 2},
		{"v3 field present", "${srcaddr} ${dstaddr} ${vpc-id} ${subnet-id}", 3},
		{"v4 field present", "${srcaddr} ${dstaddr} ${region} ${az-id}", 4},
		{"v5 field present", "${srcaddr} ${dstaddr} ${flow-direction}", 5},
		{"mixed v3 and v5", "${vpc-id} ${flow-direction}", 5},
		{"all fields", "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${vpc-id} ${subnet-id} ${instance-id} ${tcp-flags} ${type} ${pkt-srcaddr} ${pkt-dstaddr} ${region} ${az-id} ${sublocation-type} ${sublocation-id} ${pkt-src-aws-service} ${pkt-dst-aws-service} ${flow-direction} ${traffic-path}", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVersion(tt.format)
			if got != tt.expected {
				t.Errorf("DetectVersion(%q) = %d, want %d", tt.format, got, tt.expected)
			}
		})
	}
}

func TestFormatString(t *testing.T) {
	fields := []string{"srcaddr", "dstaddr", "srcport"}
	got := FormatString(fields)
	expected := "${srcaddr} ${dstaddr} ${srcport}"
	if got != expected {
		t.Errorf("FormatString() = %q, want %q", got, expected)
	}
}

func TestPresetFields(t *testing.T) {
	defaultFields := PresetFields("default")
	if len(defaultFields) != 14 {
		t.Errorf("expected 14 default fields, got %d", len(defaultFields))
	}

	securityFields := PresetFields("security")
	if len(securityFields) != 20 {
		t.Errorf("expected 20 security fields, got %d", len(securityFields))
	}

	fullFields := PresetFields("full")
	if len(fullFields) != 29 {
		t.Errorf("expected 29 full fields, got %d", len(fullFields))
	}
}

func TestPresetVersion(t *testing.T) {
	tests := []struct {
		preset   string
		expected int
	}{
		{"default", 2},
		{"security", 5},
		{"troubleshooting", 4},
		{"full", 5},
		{"unknown", 2},
	}

	for _, tt := range tests {
		got := PresetVersion(tt.preset)
		if got != tt.expected {
			t.Errorf("PresetVersion(%q) = %d, want %d", tt.preset, got, tt.expected)
		}
	}
}

func TestAllFields(t *testing.T) {
	all := AllFields()
	if len(all) != 29 {
		t.Errorf("expected 29 total fields, got %d", len(all))
	}
}
