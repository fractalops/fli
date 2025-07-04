package cache

import (
	"context"
	"fmt"
	"testing"

	"fli/internal/aws"
)

// mockENITagProvider implements ENITagProvider for testing
type mockENITagProvider struct {
	tags map[string]aws.ENITag
	err  error
}

func (m *mockENITagProvider) GetENITag(ctx context.Context, eniID string) (aws.ENITag, error) {
	if m.err != nil {
		return aws.ENITag{}, m.err
	}
	if tag, exists := m.tags[eniID]; exists {
		return tag, nil
	}
	return aws.ENITag{}, nil
}

func TestRefreshENIs(t *testing.T) {
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

	// Create mock provider
	mockProvider := &mockENITagProvider{
		tags: map[string]aws.ENITag{
			"eni-123": {
				ENI:        "eni-123",
				Label:      "test-service",
				SGNames:    []string{"sg-123", "sg-456"},
				PrivateIPs: []string{"10.0.1.100", "10.0.1.101"},
			},
			"eni-456": {
				ENI:        "eni-456",
				Label:      "another-service",
				SGNames:    []string{"sg-789"},
				PrivateIPs: []string{"10.0.2.100"},
			},
		},
	}

	// Test refreshing ENIs
	enis := []string{"eni-123", "eni-456"}
	err = cache.RefreshENIs(context.Background(), mockProvider, enis)
	if err != nil {
		t.Fatalf("Failed to refresh ENIs: %v", err)
	}

	// Verify ENIs were added to cache
	for _, eniID := range enis {
		tag, err := cache.LookupEni(context.Background(), eniID)
		if err != nil {
			t.Fatalf("Failed to lookup ENI %s: %v", eniID, err)
		}
		if tag == nil {
			t.Fatalf("Expected to find ENI %s in cache", eniID)
		}

		// Verify the tag was converted correctly
		expectedTag := mockProvider.tags[eniID]
		if tag.ENI != expectedTag.ENI {
			t.Errorf("Expected ENI %s, got %s", expectedTag.ENI, tag.ENI)
		}
		if tag.Label != expectedTag.Label {
			t.Errorf("Expected label %s, got %s", expectedTag.Label, tag.Label)
		}
		if len(tag.SGNames) != len(expectedTag.SGNames) {
			t.Errorf("Expected %d SG names, got %d", len(expectedTag.SGNames), len(tag.SGNames))
		}
		if len(tag.PrivateIPs) != len(expectedTag.PrivateIPs) {
			t.Errorf("Expected %d private IPs, got %d", len(expectedTag.PrivateIPs), len(tag.PrivateIPs))
		}
		if tag.FirstSeen == 0 {
			t.Error("Expected FirstSeen to be set")
		}
	}
}

func TestRefreshENIsWithError(t *testing.T) {
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

	// Create mock provider that returns an error
	mockProvider := &mockENITagProvider{
		err: context.DeadlineExceeded,
	}

	// Test refreshing ENIs with error
	enis := []string{"eni-123"}
	err = cache.RefreshENIs(context.Background(), mockProvider, enis)
	if err != nil {
		t.Fatalf("RefreshENIs should not return error when provider fails: %v", err)
	}

	// Verify no ENIs were added to cache
	tag, err := cache.LookupEni(context.Background(), "eni-123")
	if err != nil {
		t.Fatalf("Failed to lookup ENI: %v", err)
	}
	if tag != nil {
		t.Error("Expected ENI to not be in cache due to provider error")
	}
}

func TestRefreshENIsWithPartialError(t *testing.T) {
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

	// Create mock provider with mixed success/failure
	mockProvider := &mockENITagProvider{
		tags: map[string]aws.ENITag{
			"eni-123": {
				ENI:   "eni-123",
				Label: "success",
			},
		},
	}

	// Test refreshing ENIs where one succeeds and one fails
	enis := []string{"eni-123", "eni-456"} // eni-456 not in mock
	err = cache.RefreshENIs(context.Background(), mockProvider, enis)
	if err != nil {
		t.Fatalf("RefreshENIs should not return error for partial failure: %v", err)
	}

	// Verify successful ENI was added
	tag, err := cache.LookupEni(context.Background(), "eni-123")
	if err != nil {
		t.Fatalf("Failed to lookup successful ENI: %v", err)
	}
	if tag == nil {
		t.Error("Expected successful ENI to be in cache")
	}

	// Verify failed ENI was not added
	tag, err = cache.LookupEni(context.Background(), "eni-456")
	if err != nil {
		t.Fatalf("Failed to lookup failed ENI: %v", err)
	}
	if tag != nil {
		t.Error("Expected failed ENI to not be in cache")
	}
}

func TestRefreshAllENIs(t *testing.T) {
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

	// Add some ENIs to the cache first
	existingENI := ENITag{
		ENI:   "eni-existing",
		Label: "old-label",
	}
	err = cache.UpsertEni(existingENI)
	if err != nil {
		t.Fatalf("Failed to add existing ENI: %v", err)
	}

	// Create mock provider
	mockProvider := &mockENITagProvider{
		tags: map[string]aws.ENITag{
			"eni-existing": {
				ENI:   "eni-existing",
				Label: "new-label",
			},
		},
	}

	// Test refreshing all ENIs
	err = cache.RefreshAllENIs(context.Background(), mockProvider)
	if err != nil {
		t.Fatalf("Failed to refresh all ENIs: %v", err)
	}

	// Verify the ENI was updated
	tag, err := cache.LookupEni(context.Background(), "eni-existing")
	if err != nil {
		t.Fatalf("Failed to lookup refreshed ENI: %v", err)
	}
	if tag == nil {
		t.Fatal("Expected to find refreshed ENI in cache")
	}
	if tag.Label != "new-label" {
		t.Errorf("Expected updated label 'new-label', got '%s'", tag.Label)
	}
}

func TestRefreshAllENIsEmpty(t *testing.T) {
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

	// Create mock provider
	mockProvider := &mockENITagProvider{
		tags: map[string]aws.ENITag{},
	}

	// Test refreshing all ENIs when cache is empty
	err = cache.RefreshAllENIs(context.Background(), mockProvider)
	if err != nil {
		t.Fatalf("Failed to refresh all ENIs when empty: %v", err)
	}

	// Verify no ENIs were added
	enis, err := cache.ListENIs()
	if err != nil {
		t.Fatalf("Failed to list ENIs: %v", err)
	}
	if len(enis) != 0 {
		t.Errorf("Expected 0 ENIs, got %d", len(enis))
	}
}

func TestRefreshENIsWithENINotFound(t *testing.T) {
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

	// Add an ENI to the cache first
	existingENI := ENITag{
		ENI:   "eni-nonexistent",
		Label: "old-label",
	}
	err = cache.UpsertEni(existingENI)
	if err != nil {
		t.Fatalf("Failed to add existing ENI: %v", err)
	}

	// Create mock provider that returns ENI not found error
	mockProvider := &mockENITagProvider{
		err: fmt.Errorf("operation error EC2: DescribeNetworkInterfaces, https response error StatusCode: 400, RequestID: fc1dac8f-f5e9-4e44-88ab-ae3f95e33c2c, api error InvalidNetworkInterfaceID.NotFound: The networkInterface ID 'eni-nonexistent' does not exist"),
	}

	// Test refreshing ENIs where one no longer exists
	enis := []string{"eni-nonexistent"}
	err = cache.RefreshENIs(context.Background(), mockProvider, enis)
	if err != nil {
		t.Fatalf("RefreshENIs should not return error for ENI not found: %v", err)
	}

	// Verify the ENI was removed from cache
	tag, err := cache.LookupEni(context.Background(), "eni-nonexistent")
	if err != nil {
		t.Fatalf("Failed to lookup removed ENI: %v", err)
	}
	if tag != nil {
		t.Error("Expected ENI to be removed from cache due to not found error")
	}
}
