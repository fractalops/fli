package cache

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	config := DefaultConfig().WithCachePath(cachePath)

	cache, err := OpenWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Verify the cache was created
	if cache.db == nil {
		t.Fatal("Cache database is nil")
	}

	// Close the first cache before opening the second one
	if err := cache.Close(); err != nil {
		t.Logf("Warning: failed to close cache: %v", err)
	}

	// Test opening the same cache again (should work)
	cache2, err := OpenWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to reopen cache: %v", err)
	}
	defer func() {
		if closeErr := cache2.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache2: %v", closeErr)
		}
	}()
}

func TestOpenWithInvalidPath(t *testing.T) {
	// Test opening cache with invalid path (should fail gracefully)
	_, err := Open("/invalid/path/test.db")
	if err == nil {
		t.Fatal("Expected error when opening cache with invalid path")
	}
}

func TestENIOperations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Test data
	testENI := ENITag{
		ENI:        "eni-12345678",
		Label:      "test-service",
		SGNames:    []string{"sg-123", "sg-456"},
		PrivateIPs: []string{"10.0.1.100", "10.0.1.101"},
		FirstSeen:  time.Now().Unix(),
	}

	// Test UpsertEni
	err = cache.UpsertEni(testENI)
	if err != nil {
		t.Fatalf("Failed to upsert ENI: %v", err)
	}

	// Test LookupEni - found
	found, err := cache.LookupEni(context.Background(), testENI.ENI)
	if err != nil {
		t.Fatalf("Failed to lookup ENI: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find ENI, got nil")
	}
	if found.ENI != testENI.ENI {
		t.Errorf("Expected ENI %s, got %s", testENI.ENI, found.ENI)
	}
	if found.Label != testENI.Label {
		t.Errorf("Expected label %s, got %s", testENI.Label, found.Label)
	}

	// Test LookupEni - not found
	notFound, err := cache.LookupEni(context.Background(), "eni-nonexistent")
	if err != nil {
		t.Fatalf("Failed to lookup non-existent ENI: %v", err)
	}
	if notFound != nil {
		t.Fatal("Expected nil for non-existent ENI")
	}

	// Test ListENIs
	enis, err := cache.ListENIs()
	if err != nil {
		t.Fatalf("Failed to list ENIs: %v", err)
	}
	if len(enis) != 1 {
		t.Errorf("Expected 1 ENI, got %d", len(enis))
	}
	if enis[0] != testENI.ENI {
		t.Errorf("Expected ENI %s, got %s", testENI.ENI, enis[0])
	}
}

func TestPrefixOperations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Test data
	testPrefix := PrefixTag{
		CIDR:    "192.168.1.0/24",
		Cloud:   "AWS",
		Service: "EC2",
		Fetched: time.Now().Unix(),
	}

	// Test UpsertPrefix
	err = cache.UpsertPrefix(testPrefix)
	if err != nil {
		t.Fatalf("Failed to upsert prefix: %v", err)
	}

	// Test UpsertPrefixes (batch)
	prefixes := []PrefixTag{
		{CIDR: "10.0.0.0/16", Cloud: "GCP", Service: "Compute"},
		{CIDR: "172.16.0.0/12", Cloud: "Azure", Service: "VM"},
	}
	err = cache.UpsertPrefixes(prefixes)
	if err != nil {
		t.Fatalf("Failed to upsert prefixes: %v", err)
	}

	// Test ListPrefixes
	allPrefixes, err := cache.ListPrefixes()
	if err != nil {
		t.Fatalf("Failed to list prefixes: %v", err)
	}
	if len(allPrefixes) != 3 {
		t.Errorf("Expected 3 prefixes, got %d", len(allPrefixes))
	}

	// Test IP lookup with prefix match
	addr, _ := netip.ParseAddr("192.168.1.100")
	annotation, err := cache.LookupIP(addr)
	if err != nil {
		t.Fatalf("Failed to lookup IP: %v", err)
	}
	if annotation == "" {
		t.Fatal("Expected annotation for IP in prefix range")
	}
	if annotation != "AWS (192.168.1.0/24), EC2" {
		t.Errorf("Expected 'AWS (192.168.1.0/24), EC2', got '%s'", annotation)
	}
}

func TestIPOperations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Test data
	testIP := IPTag{
		Addr: "8.8.8.8",
		Name: "Google DNS",
	}

	// Test UpsertIP
	err = cache.UpsertIP(testIP)
	if err != nil {
		t.Fatalf("Failed to upsert IP: %v", err)
	}

	// Test ListIPs
	ips, err := cache.ListIPs()
	if err != nil {
		t.Fatalf("Failed to list IPs: %v", err)
	}
	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}
	if ips[0] != testIP.Addr {
		t.Errorf("Expected IP %s, got %s", testIP.Addr, ips[0])
	}

	// Test IP lookup - exact match
	addr, _ := netip.ParseAddr("8.8.8.8")
	annotation, err := cache.LookupIP(addr)
	if err != nil {
		t.Fatalf("Failed to lookup IP: %v", err)
	}
	if annotation != "Google DNS" {
		t.Errorf("Expected 'Google DNS', got '%s'", annotation)
	}

	// Test IP lookup - not found
	addr2, _ := netip.ParseAddr("1.1.1.1")
	annotation2, err := cache.LookupIP(addr2)
	if err != nil {
		t.Fatalf("Failed to lookup non-existent IP: %v", err)
	}
	if annotation2 != "" {
		t.Errorf("Expected empty annotation, got '%s'", annotation2)
	}
}

func TestLookupIPWithPrefixMatch(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Add multiple overlapping prefixes
	prefixes := []PrefixTag{
		{CIDR: "10.0.0.0/8", Cloud: "AWS", Service: "VPC"},
		{CIDR: "10.1.0.0/16", Cloud: "GCP", Service: "Compute"},
		{CIDR: "10.1.1.0/24", Cloud: "Azure", Service: "VM"},
	}

	for _, prefix := range prefixes {
		err = cache.UpsertPrefix(prefix)
		if err != nil {
			t.Fatalf("Failed to upsert prefix %s: %v", prefix.CIDR, err)
		}
	}

	// Test longest prefix match
	addr, _ := netip.ParseAddr("10.1.1.100")
	annotation, err := cache.LookupIP(addr)
	if err != nil {
		t.Fatalf("Failed to lookup IP: %v", err)
	}
	// Should match the most specific prefix (10.1.1.0/24)
	expected := "Azure (10.1.1.0/24), VM"
	if annotation != expected {
		t.Errorf("Expected '%s', got '%s'", expected, annotation)
	}
}

func TestLookupIPExactMatchTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Add a prefix that would match
	prefix := PrefixTag{
		CIDR:    "10.0.0.0/8",
		Cloud:   "AWS",
		Service: "VPC",
	}
	err = cache.UpsertPrefix(prefix)
	if err != nil {
		t.Fatalf("Failed to upsert prefix: %v", err)
	}

	// Add an exact IP match
	ipTag := IPTag{
		Addr: "10.0.0.1",
		Name: "Exact Match",
	}
	err = cache.UpsertIP(ipTag)
	if err != nil {
		t.Fatalf("Failed to upsert IP: %v", err)
	}

	// Test that exact match takes precedence
	addr, _ := netip.ParseAddr("10.0.0.1")
	annotation, err := cache.LookupIP(addr)
	if err != nil {
		t.Fatalf("Failed to lookup IP: %v", err)
	}
	if annotation != "Exact Match" {
		t.Errorf("Expected 'Exact Match', got '%s'", annotation)
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}

	// Test closing
	err = cache.Close()
	if err != nil {
		t.Fatalf("Failed to close cache: %v", err)
	}

	// Test closing again (should not error)
	err = cache.Close()
	if err != nil {
		t.Fatalf("Failed to close cache twice: %v", err)
	}
}

func TestEmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Test empty lists
	enis, err := cache.ListENIs()
	if err != nil {
		t.Fatalf("Failed to list empty ENIs: %v", err)
	}
	if len(enis) != 0 {
		t.Errorf("Expected 0 ENIs, got %d", len(enis))
	}

	ips, err := cache.ListIPs()
	if err != nil {
		t.Fatalf("Failed to list empty IPs: %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Expected 0 IPs, got %d", len(ips))
	}

	prefixes, err := cache.ListPrefixes()
	if err != nil {
		t.Fatalf("Failed to list empty prefixes: %v", err)
	}
	if len(prefixes) != 0 {
		t.Errorf("Expected 0 prefixes, got %d", len(prefixes))
	}
}

func TestDeleteENI(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.db")
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Warning: failed to close cache: %v", closeErr)
		}
	}()

	// Test data
	testENI := ENITag{
		ENI:        "eni-12345678",
		Label:      "test-service",
		SGNames:    []string{"sg-123", "sg-456"},
		PrivateIPs: []string{"10.0.1.100", "10.0.1.101"},
		FirstSeen:  time.Now().Unix(),
	}

	// Add ENI to cache
	err = cache.UpsertEni(testENI)
	if err != nil {
		t.Fatalf("Failed to upsert ENI: %v", err)
	}

	// Verify ENI exists
	found, err := cache.LookupEni(context.Background(), testENI.ENI)
	if err != nil {
		t.Fatalf("Failed to lookup ENI: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find ENI before deletion")
	}

	// Delete ENI
	err = cache.DeleteENI(testENI.ENI)
	if err != nil {
		t.Fatalf("Failed to delete ENI: %v", err)
	}

	// Verify ENI no longer exists
	found, err = cache.LookupEni(context.Background(), testENI.ENI)
	if err != nil {
		t.Fatalf("Failed to lookup ENI after deletion: %v", err)
	}
	if found != nil {
		t.Fatal("Expected ENI to be deleted")
	}

	// Test deleting non-existent ENI (should not error)
	err = cache.DeleteENI("eni-nonexistent")
	if err != nil {
		t.Fatalf("Failed to delete non-existent ENI: %v", err)
	}
}
