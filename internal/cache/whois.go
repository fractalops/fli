package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	"go.etcd.io/bbolt"
)

// whoisProvider represents a known provider in whois data.
type whoisProvider struct {
	searchTerms []string
	name        string
}

var knownProviders = []whoisProvider{
	{[]string{"cloudflare"}, "CLOUDFLARE"},
	{[]string{"digitalocean"}, "DIGITALOCEAN"},
	{[]string{"amazon", "aws"}, "AMAZON"},
	{[]string{"akamai"}, "AKAMAI"},
	{[]string{"ripe"}, "RIPE"},
	{[]string{"shodan"}, "SHODAN"},
	{[]string{"censys"}, "CENSYS"},
	{[]string{"hurricane electric"}, "HE"},
	{[]string{"apnic"}, "APNIC"},
}

// extractWhoisSummary tries to extract a short alias/code from whois text.
func extractWhoisSummary(whoisText string) string {
	lines := strings.Split(whoisText, "\n")
	for _, line := range lines {
		low := strings.ToLower(line)

		// Check for known providers
		for _, provider := range knownProviders {
			for _, term := range provider.searchTerms {
				if strings.Contains(low, term) {
					return provider.name
				}
			}
		}

		// Try to extract country code
		if strings.Contains(low, "country:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				code := strings.TrimSpace(parts[1])
				if len(code) > 0 && len(code) <= 3 {
					return strings.ToUpper(code)
				}
			}
		}

		// Try to extract organization
		if strings.Contains(low, "org:") || strings.Contains(low, "organization") {
			if org := extractOrganization(line); org != "" {
				return org
			}
		}
	}

	// Fallback: return first non-empty word in any line
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			words := strings.Fields(line)
			if len(words) > 0 {
				return words[0]
			}
		}
	}
	return "whois"
}

// extractOrganization extracts organization information from a whois line.
func extractOrganization(line string) string {
	parts := strings.Split(line, ":")
	if len(parts) <= 1 {
		return ""
	}

	org := strings.TrimSpace(parts[1])
	if org == "" {
		return ""
	}

	words := strings.Fields(org)
	if len(words) == 0 {
		return ""
	}

	return words[0]
}

// EnrichIPs performs whois enrichment for public IPs in the cache.
func (c *Cache) EnrichIPs() error {
	ips, err := c.ListIPs()
	if err != nil {
		return fmt.Errorf("failed to list IPs: %w", err)
	}
	for i, ip := range ips {
		addr, err := netip.ParseAddr(ip)
		if err != nil || addr.IsPrivate() {
			continue
		}

		// Check if the IP already has an annotation from a cloud prefix.
		// If it does, we can skip the expensive whois lookup.
		annotation, err := c.LookupIP(addr)
		if err != nil {
			log.Printf("Warning: failed to lookup IP %s: %v", ip, err)
			continue
		}

		if annotation == "" {
			// No existing annotation, let's try to enrich it.
			log.Printf("Enriching public IP %s (%d/%d)...", ip, i+1, len(ips))
			whoisInfo, err := whois.Whois(ip)
			if err == nil {
				label := extractWhoisSummary(whoisInfo)
				if err := c.UpsertIP(IPTag{Addr: ip, Name: label}); err != nil {
					log.Printf("Warning: failed to upsert IP %s: %v", ip, err)
				}
			} else {
				log.Printf("Warning: whois lookup failed for %s: %v", ip, err)
			}
		}
	}
	return nil
}

// WhoisResult represents the result of a whois lookup.
type WhoisResult struct {
	IP       string
	ASN      string
	Org      string
	Country  string
	Error    error
	Duration time.Duration
}

// EnrichIP performs a whois lookup for an IP address and stores the result.
func (c *Cache) EnrichIP(ip string) (*WhoisResult, error) {
	if !c.config.EnableWhoisEnrichment {
		return nil, NewConfigurationError("whois enrichment is disabled", nil)
	}

	c.logger.Debug("Enriching IP with whois data: %s", ip)

	start := time.Now()

	// Perform whois lookup
	whoisData, err := c.whoisClient.Lookup(ip)
	if err != nil {
		return nil, NewWhoisError(ip, err)
	}

	duration := time.Since(start)

	// Parse whois data
	result := c.parseWhoisData(ip, whoisData)
	result.Duration = duration

	// Store the result
	if err := c.storeWhoisResult(result); err != nil {
		c.logger.Error("Failed to store whois result for %s: %v", ip, err)
		// Don't return error here as the lookup was successful
	}

	c.logger.Debug("Enriched IP %s in %v", ip, duration)
	return result, nil
}

// EnrichIPsBatch performs whois lookups for multiple IP addresses.
func (c *Cache) EnrichIPsBatch(ips []string) ([]*WhoisResult, error) {
	if !c.config.EnableWhoisEnrichment {
		return nil, NewConfigurationError("whois enrichment is disabled", nil)
	}

	c.logger.Info("Enriching %d IPs with whois data", len(ips))

	results := make([]*WhoisResult, 0, len(ips))
	var errors []error
	var mu sync.Mutex // Mutex to protect results and errors slices

	// Process IPs with rate limiting to avoid overwhelming whois servers
	semaphore := make(chan struct{}, 5) // Limit concurrent lookups
	resultsChan := make(chan *WhoisResult, len(ips))

	// Use WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup

	// Start workers
	for _, ip := range ips {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result, err := c.EnrichIP(ipAddr)
			if err != nil {
				result = &WhoisResult{
					IP:    ipAddr,
					Error: err,
				}
			}
			resultsChan <- result
		}(ip)
	}

	// Close resultsChan when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		mu.Lock()
		results = append(results, result)
		if result.Error != nil {
			errors = append(errors, result.Error)
		}
		mu.Unlock()
	}

	c.logger.Info("Completed whois enrichment: %d successful, %d failed",
		len(results)-len(errors), len(errors))

	return results, nil
}

// EnrichIPsInBatches efficiently enriches multiple IPs with rate limiting.
func (c *Cache) EnrichIPsInBatches(ctx context.Context, ips []string, batchSize int) error {
	if !c.config.EnableWhoisEnrichment {
		return NewConfigurationError("whois enrichment is disabled", nil)
	}

	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	c.logger.Info("Starting batch whois enrichment for %d IPs (batch size: %d)", len(ips), batchSize)

	for i := 0; i < len(ips); i += batchSize {
		end := i + batchSize
		if end > len(ips) {
			end = len(ips)
		}

		batch := ips[i:end]
		c.logger.Debug("Processing batch %d-%d of %d", i+1, end, len(ips))

		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during whois enrichment: %w", ctx.Err())
		default:
		}

		results, err := c.EnrichIPsBatch(batch)
		if err != nil {
			c.logger.Error("Batch enrichment failed: %v", err)
			// Continue with next batch
		}

		// Log batch results
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}
		c.logger.Debug("Batch completed: %d/%d successful", successCount, len(batch))

		// Rate limiting between batches
		if end < len(ips) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	c.logger.Info("Completed batch whois enrichment")
	return nil
}

// parseWhoisData extracts useful information from whois response.
func (c *Cache) parseWhoisData(ip, whoisData string) *WhoisResult {
	result := &WhoisResult{
		IP: ip,
	}

	lines := strings.Split(whoisData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "%") {
			continue
		}

		// Extract ASN
		if strings.HasPrefix(strings.ToUpper(line), "ORIGIN:") {
			result.ASN = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(line), "ORIGIN:"))
		}

		// Extract organization
		if strings.HasPrefix(strings.ToUpper(line), "ORG:") {
			result.Org = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(line), "ORG:"))
		}

		// Extract country
		if strings.HasPrefix(strings.ToUpper(line), "COUNTRY:") {
			result.Country = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(line), "COUNTRY:"))
		}
	}

	return result
}

// storeWhoisResult stores the whois result in the database.
func (c *Cache) storeWhoisResult(result *WhoisResult) error {
	if result.Error != nil {
		return fmt.Errorf("cannot store failed lookup for %s: %w", result.IP, result.Error)
	}

	// Create a summary label from the whois data
	label := extractWhoisSummary(fmt.Sprintf("ASN: %s\nORG: %s\nCOUNTRY: %s",
		result.ASN, result.Org, result.Country))

	// Create IP tag with whois data
	ipTag := IPTag{
		Addr: result.IP,
		Name: label,
	}

	return c.UpsertIP(ipTag)
}

// GetWhoisInfo retrieves stored whois information for an IP.
func (c *Cache) GetWhoisInfo(ip string) (*WhoisResult, error) {
	err := c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketIPTags))
		if bucket == nil {
			return NewDatabaseError("get_bucket", bucketIPTags, nil)
		}

		data := bucket.Get([]byte(ip))
		if data == nil {
			return NewNotFoundError("get_whois_info", ip)
		}

		var ipTag IPTag
		if err := json.Unmarshal(data, &ipTag); err != nil {
			return NewInvalidDataError("unmarshal_whois", ip, "failed to unmarshal IP tag", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get whois info: %w", err)
	}

	return &WhoisResult{
		IP: ip,
	}, nil
}
