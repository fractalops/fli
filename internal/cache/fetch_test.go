package cache

import (
	"encoding/json"
	"testing"
	"time"
)

const testProviderAWS = "aws"

func TestAWSIPRangesStruct(t *testing.T) {
	// Test that AWSIPRanges struct can be unmarshaled correctly
	jsonData := `{
		"syncToken": "1234567890",
		"createDate": "2024-01-01-00-00-00",
		"prefixes": [
			{
				"ip_prefix": "52.0.0.0/8",
				"region": "us-east-1",
				"service": "EC2",
				"network_border_group": "us-east-1"
			},
			{
				"ip_prefix": "54.0.0.0/8",
				"region": "us-west-2",
				"service": "S3",
				"network_border_group": "us-west-2"
			}
		],
		"ipv6_prefixes": [
			{
				"ipv6_prefix": "2600:1f00::/40",
				"region": "us-east-1",
				"service": "EC2",
				"network_border_group": "us-east-1"
			}
		]
	}`

	var data AWSIPRanges
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal AWS IP ranges: %v", err)
	}

	if len(data.Prefixes) != 2 {
		t.Errorf("Expected 2 IPv4 prefixes, got %d", len(data.Prefixes))
	}
	if len(data.IPv6Prefixes) != 1 {
		t.Errorf("Expected 1 IPv6 prefix, got %d", len(data.IPv6Prefixes))
	}

	// Check first IPv4 prefix
	if data.Prefixes[0].IPPrefix != "52.0.0.0/8" {
		t.Errorf("Expected IP prefix '52.0.0.0/8', got '%s'", data.Prefixes[0].IPPrefix)
	}
	if data.Prefixes[0].Service != "EC2" {
		t.Errorf("Expected service 'EC2', got '%s'", data.Prefixes[0].Service)
	}

	// Check IPv6 prefix
	if data.IPv6Prefixes[0].IPv6Prefix != "2600:1f00::/40" {
		t.Errorf("Expected IPv6 prefix '2600:1f00::/40', got '%s'", data.IPv6Prefixes[0].IPv6Prefix)
	}
}

func TestGCPIPRangesStruct(t *testing.T) {
	// Test that GCPIPRanges struct can be unmarshaled correctly
	jsonData := `{
		"syncToken": "1234567890",
		"prefixes": [
			{
				"ipv4Prefix": "8.8.8.0/24",
				"service": "Google",
				"scope": "global"
			},
			{
				"ipv4Prefix": "172.217.0.0/16",
				"service": "Compute",
				"scope": "us-central1"
			}
		]
	}`

	var data GCPIPRanges
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal GCP IP ranges: %v", err)
	}

	if len(data.Prefixes) != 2 {
		t.Errorf("Expected 2 prefixes, got %d", len(data.Prefixes))
	}

	// Check first prefix
	if data.Prefixes[0].IPPrefix != "8.8.8.0/24" {
		t.Errorf("Expected IPv4 prefix '8.8.8.0/24', got '%s'", data.Prefixes[0].IPPrefix)
	}
	if data.Prefixes[0].Service != "Google" {
		t.Errorf("Expected service 'Google', got '%s'", data.Prefixes[0].Service)
	}

	// Check second prefix
	if data.Prefixes[1].IPPrefix != "172.217.0.0/16" {
		t.Errorf("Expected IPv4 prefix '172.217.0.0/16', got '%s'", data.Prefixes[1].IPPrefix)
	}
	if data.Prefixes[1].Service != "Compute" {
		t.Errorf("Expected service 'Compute', got '%s'", data.Prefixes[1].Service)
	}
}

func TestDigitalOceanIPRangesStruct(t *testing.T) {
	// Test that DigitalOceanIPRanges struct can be unmarshaled correctly
	jsonData := `{
		"links": {
			"self": "https://digitalocean.com/geo/google.json"
		},
		"meta": {
			"total": 2
		},
		"data": [
			{
				"ip_prefix": "192.168.1.0/24",
				"region": "nyc1"
			},
			{
				"ip_prefix": "10.0.0.0/8",
				"region": "sfo2"
			}
		]
	}`

	var data DigitalOceanIPRanges
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal DigitalOcean IP ranges: %v", err)
	}

	if len(data.Data) != 2 {
		t.Errorf("Expected 2 data entries, got %d", len(data.Data))
	}

	// Check first entry
	if data.Data[0].IPPrefix != "192.168.1.0/24" {
		t.Errorf("Expected IP prefix '192.168.1.0/24', got '%s'", data.Data[0].IPPrefix)
	}
	if data.Data[0].Region != "nyc1" {
		t.Errorf("Expected region 'nyc1', got '%s'", data.Data[0].Region)
	}
}

func TestProcessAWSData(t *testing.T) {
	cache := &Cache{
		logger: NewDefaultLogger(true),
	}

	awsData := AWSIPRanges{
		Prefixes: []struct {
			IPPrefix           string `json:"ip_prefix"`
			Region             string `json:"region"`
			Service            string `json:"service"`
			NetworkBorderGroup string `json:"network_border_group"`
		}{
			{IPPrefix: "52.0.0.0/8", Region: "us-east-1", Service: "EC2"},
			{IPPrefix: "54.0.0.0/8", Region: "us-west-2", Service: "S3"},
		},
		IPv6Prefixes: []struct {
			IPv6Prefix         string `json:"ipv6_prefix"`
			Region             string `json:"region"`
			Service            string `json:"service"`
			NetworkBorderGroup string `json:"network_border_group"`
		}{
			{IPv6Prefix: "2600:1f00::/40", Region: "us-east-1", Service: "EC2"},
		},
	}

	tags, err := cache.processAWSData(awsData)
	if err != nil {
		t.Fatalf("Failed to process AWS data: %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// Check first tag
	if tags[0].CIDR != "52.0.0.0/8" {
		t.Errorf("Expected CIDR '52.0.0.0/8', got '%s'", tags[0].CIDR)
	}
	if tags[0].Cloud != "AWS" {
		t.Errorf("Expected cloud 'AWS', got '%s'", tags[0].Cloud)
	}
	if tags[0].Service != "EC2" {
		t.Errorf("Expected service 'EC2', got '%s'", tags[0].Service)
	}
}

func TestProcessGCPData(t *testing.T) {
	cache := &Cache{
		logger: NewDefaultLogger(true),
	}

	gcpData := GCPIPRanges{
		Prefixes: []struct {
			IPPrefix string `json:"ipv4Prefix"`
			Service  string `json:"service"`
			Scope    string `json:"scope"`
		}{
			{IPPrefix: "8.8.8.0/24", Service: "Google", Scope: "global"},
			{IPPrefix: "172.217.0.0/16", Service: "Compute", Scope: "us-central1"},
		},
	}

	tags, err := cache.processGCPData(gcpData)
	if err != nil {
		t.Fatalf("Failed to process GCP data: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// Check first tag
	if tags[0].CIDR != "8.8.8.0/24" {
		t.Errorf("Expected CIDR '8.8.8.0/24', got '%s'", tags[0].CIDR)
	}
	if tags[0].Cloud != "GCP" {
		t.Errorf("Expected cloud 'GCP', got '%s'", tags[0].Cloud)
	}
	if tags[0].Service != "Google" {
		t.Errorf("Expected service 'Google', got '%s'", tags[0].Service)
	}
}

func TestProcessCloudflareData(t *testing.T) {
	cache := &Cache{
		logger: NewDefaultLogger(true),
	}

	cloudflareData := "173.245.48.0/20\n103.21.244.0/22\n103.22.200.0/22"

	tags, err := cache.processCloudflareData(cloudflareData)
	if err != nil {
		t.Fatalf("Failed to process Cloudflare data: %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// Check first tag
	if tags[0].CIDR != "173.245.48.0/20" {
		t.Errorf("Expected CIDR '173.245.48.0/20', got '%s'", tags[0].CIDR)
	}
	if tags[0].Cloud != "Cloudflare" {
		t.Errorf("Expected cloud 'Cloudflare', got '%s'", tags[0].Cloud)
	}
}

func TestProcessDigitalOceanData(t *testing.T) {
	cache := &Cache{
		logger: NewDefaultLogger(true),
	}

	doData := DigitalOceanIPRanges{
		Data: []struct {
			IPPrefix string `json:"ip_prefix"`
			Region   string `json:"region"`
		}{
			{IPPrefix: "192.168.1.0/24", Region: "nyc1"},
			{IPPrefix: "10.0.0.0/8", Region: "sfo2"},
		},
	}

	tags, err := cache.processDigitalOceanData(doData)
	if err != nil {
		t.Fatalf("Failed to process DigitalOcean data: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// Check first tag
	if tags[0].CIDR != "192.168.1.0/24" {
		t.Errorf("Expected CIDR '192.168.1.0/24', got '%s'", tags[0].CIDR)
	}
	if tags[0].Cloud != "DigitalOcean" {
		t.Errorf("Expected cloud 'DigitalOcean', got '%s'", tags[0].Cloud)
	}
}

func TestFetchResult(t *testing.T) {
	result := &FetchResult{
		Provider: testProviderAWS,
		Data:     "test data",
		Duration: 100 * time.Millisecond,
	}

	if result.Provider != testProviderAWS {
		t.Errorf("Expected provider '%s', got '%s'", testProviderAWS, result.Provider)
	}
	if result.Data != "test data" {
		t.Errorf("Expected data 'test data', got '%v'", result.Data)
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", result.Duration)
	}
}

// Note: Tests for FetchProvider, FetchAllProviders, and UpdatePrefixes
// are not included here because they would require real HTTP calls which
// can fail due to network issues or rate limiting. In a real testing
// environment, you would:
// 1. Mock the HTTP client
// 2. Use httptest.Server to create a test server
// 3. Use integration tests with controlled test data
// 4. Test the parsing logic separately from the network calls
