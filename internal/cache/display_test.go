package cache

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFormatENITag(t *testing.T) {
	tests := []struct {
		name     string
		tag      *ENITag
		expected string
	}{
		{
			name:     "nil tag",
			tag:      nil,
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      &ENITag{},
			expected: "",
		},
		{
			name: "tag with label only",
			tag: &ENITag{
				ENI:   "eni-123",
				Label: "test-service",
			},
			expected: " (test-service)",
		},
		{
			name: "tag with security groups",
			tag: &ENITag{
				ENI:     "eni-123",
				SGNames: []string{"sg-123", "sg-456"},
			},
			expected: " (SGs: [sg-123 sg-456])",
		},
		{
			name: "tag with private IPs",
			tag: &ENITag{
				ENI:        "eni-123",
				PrivateIPs: []string{"10.0.1.100", "10.0.1.101"},
			},
			expected: " (IPs: [10.0.1.100 10.0.1.101])",
		},
		{
			name: "tag with all fields",
			tag: &ENITag{
				ENI:        "eni-123",
				Label:      "test-service",
				SGNames:    []string{"sg-123"},
				PrivateIPs: []string{"10.0.1.100"},
			},
			expected: " (test-service, SGs: [sg-123], IPs: [10.0.1.100])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatENITag(tt.tag)
			if result != tt.expected {
				t.Errorf("formatENITag() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatIPAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		expected   string
	}{
		{
			name:       "empty annotation",
			annotation: "",
			expected:   "",
		},
		{
			name:       "simple annotation",
			annotation: "AWS",
			expected:   "(AWS)",
		},
		{
			name:       "complex annotation",
			annotation: "AWS (10.0.0.0/8), EC2",
			expected:   "(AWS (10.0.0.0/8), EC2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatIPAnnotation(tt.annotation)
			if result != tt.expected {
				t.Errorf("formatIPAnnotation() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := tmpDir + "/test_cache.db"
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Add test data
	eniTag := ENITag{
		ENI:        "eni-12345678",
		Label:      "test-service",
		SGNames:    []string{"sg-123", "sg-456"},
		PrivateIPs: []string{"10.0.1.100"},
		FirstSeen:  time.Now().Unix(),
	}
	err = cache.UpsertEni(eniTag)
	if err != nil {
		t.Fatalf("Failed to upsert ENI: %v", err)
	}

	ipTag := IPTag{
		Addr: "8.8.8.8",
		Name: "Google DNS",
	}
	err = cache.UpsertIP(ipTag)
	if err != nil {
		t.Fatalf("Failed to upsert IP: %v", err)
	}

	prefixTag := PrefixTag{
		CIDR:    "192.168.1.0/24",
		Cloud:   "AWS",
		Service: "EC2",
		Fetched: time.Now().Unix(),
	}
	err = cache.UpsertPrefix(prefixTag)
	if err != nil {
		t.Fatalf("Failed to upsert prefix: %v", err)
	}

	// Test List output
	output, err := cache.List(context.Background())
	if err != nil {
		t.Fatalf("Failed to list cache: %v", err)
	}

	// Verify output contains expected sections
	if !strings.Contains(output, "ENIs:") {
		t.Error("Output should contain 'ENIs:' section")
	}
	if !strings.Contains(output, "IPs:") {
		t.Error("Output should contain 'IPs:' section")
	}
	if !strings.Contains(output, "Prefixes:") {
		t.Error("Output should contain 'Prefixes:' section")
	}

	// Verify ENI output
	if !strings.Contains(output, "eni-12345678") {
		t.Error("Output should contain ENI ID")
	}
	if !strings.Contains(output, "test-service") {
		t.Error("Output should contain ENI label")
	}

	// Verify IP output
	if !strings.Contains(output, "8.8.8.8") {
		t.Error("Output should contain IP address")
	}
	if !strings.Contains(output, "(Google DNS)") {
		t.Error("Output should contain IP annotation")
	}

	// Verify prefix output
	if !strings.Contains(output, "192.168.1.0/24") {
		t.Error("Output should contain CIDR prefix")
	}
	if !strings.Contains(output, "(AWS (192.168.1.0/24), EC2)") {
		t.Error("Output should contain prefix annotation")
	}
}

func TestListEmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := tmpDir + "/test_cache.db"
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	output, err := cache.List(context.Background())
	if err != nil {
		t.Fatalf("Failed to list empty cache: %v", err)
	}

	// Should contain all sections even when empty
	expectedSections := []string{"ENIs:", "IPs:", "Prefixes:"}
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output should contain '%s' section", section)
		}
	}

	// Should not contain any actual data
	if strings.Contains(output, "eni-") {
		t.Error("Output should not contain ENI data")
	}
	if strings.Contains(output, "8.8.8.8") {
		t.Error("Output should not contain IP data")
	}
	if strings.Contains(output, "192.168.1.0/24") {
		t.Error("Output should not contain prefix data")
	}
}
