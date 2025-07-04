package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

// ProviderData represents data fetched from a cloud provider.
type ProviderData struct {
	Provider string
	Data     interface{}
	URL      string
	Fetched  time.Time
}

// FetchResult represents the result of a fetch operation.
type FetchResult struct {
	Provider string
	Data     interface{}
	Error    error
	Duration time.Duration
}

// FetchProvider fetches data from a specific provider.
func (c *Cache) FetchProvider(ctx context.Context, provider string) (*FetchResult, error) {
	url, exists := c.config.ProviderURLs[provider]
	if !exists {
		return nil, NewConfigurationError(fmt.Sprintf("unknown provider: %s", provider), nil)
	}

	c.logger.Info("Fetching data from provider: %s", provider)

	start := time.Now()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewNetworkError("create_request", url, err)
	}

	// Set user agent
	req.Header.Set("User-Agent", c.config.UserAgent)

	// Make request with context
	req = req.WithContext(ctx)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewNetworkError("http_request", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("Failed to close response body: %v", closeErr)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, NewNetworkError("http_status", url,
			fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewNetworkError("read_response", url, err)
	}

	// Parse response based on provider
	var data interface{}
	switch provider {
	case "aws":
		var awsData AWSIPRanges
		if err := json.Unmarshal(body, &awsData); err != nil {
			return nil, NewInvalidDataError("parse_aws_data", provider, "failed to parse AWS data", err)
		}
		data = awsData
	case "gcp", "gcp_legacy":
		var gcpData GCPIPRanges
		if err := json.Unmarshal(body, &gcpData); err != nil {
			return nil, NewInvalidDataError("parse_gcp_data", provider, "failed to parse GCP data", err)
		}
		data = gcpData
	case "cloudflare":
		// Cloudflare returns plain text
		data = string(body)
	case "digitalocean":
		var doData DigitalOceanIPRanges
		if err := json.Unmarshal(body, &doData); err != nil {
			return nil, NewInvalidDataError("parse_do_data", provider, "failed to parse DigitalOcean data", err)
		}
		data = doData
	default:
		return nil, NewConfigurationError(fmt.Sprintf("unsupported provider: %s", provider), nil)
	}

	duration := time.Since(start)
	c.logger.Info("Successfully fetched data from %s in %v", provider, duration)

	return &FetchResult{
		Provider: provider,
		Data:     data,
		Duration: duration,
	}, nil
}

// FetchAllProviders fetches data from all configured providers.
func (c *Cache) FetchAllProviders(ctx context.Context) ([]*FetchResult, error) {
	results := make([]*FetchResult, 0, len(c.config.ProviderURLs))
	for provider := range c.config.ProviderURLs {
		result, err := c.FetchProvider(ctx, provider)
		if err != nil {
			c.logger.Error("Failed to fetch %s: %v", provider, err)
			result = &FetchResult{
				Provider: provider,
				Error:    err,
			}
		}
		results = append(results, result)
	}
	return results, nil
}

// ProcessFetchResult processes a fetch result and converts it to PrefixTags.
func (c *Cache) ProcessFetchResult(result *FetchResult) ([]PrefixTag, error) {
	switch result.Provider {
	case "aws":
		return c.processAWSData(result.Data)
	case "gcp", "gcp_legacy":
		return c.processGCPData(result.Data)
	case "cloudflare":
		return c.processCloudflareData(result.Data)
	case "digitalocean":
		return c.processDigitalOceanData(result.Data)
	default:
		return nil, NewConfigurationError(fmt.Sprintf("unsupported provider: %s", result.Provider), nil)
	}
}

// processAWSData converts AWS data to PrefixTags.
func (c *Cache) processAWSData(data interface{}) ([]PrefixTag, error) {
	awsData, ok := data.(AWSIPRanges)
	if !ok {
		return nil, NewInvalidDataError("process_aws_data", "", "invalid AWS data type", nil)
	}

	tags := make([]PrefixTag, 0, len(awsData.Prefixes)+len(awsData.IPv6Prefixes))

	// Process IPv4 prefixes
	for _, prefix := range awsData.Prefixes {
		if prefix.Service == "AMAZON" { // Skip broad, non-specific ranges
			continue
		}
		tags = append(tags, PrefixTag{
			CIDR:    prefix.IPPrefix,
			Cloud:   "AWS",
			Service: prefix.Service,
		})
	}

	// Process IPv6 prefixes
	for _, prefix := range awsData.IPv6Prefixes {
		if prefix.Service == "AMAZON" {
			continue
		}
		tags = append(tags, PrefixTag{
			CIDR:    prefix.IPv6Prefix,
			Cloud:   "AWS",
			Service: prefix.Service,
		})
	}

	c.logger.Info("Processed %d AWS prefixes", len(tags))
	return tags, nil
}

// processGCPData converts GCP data to PrefixTags.
func (c *Cache) processGCPData(data interface{}) ([]PrefixTag, error) {
	gcpData, ok := data.(GCPIPRanges)
	if !ok {
		return nil, NewInvalidDataError("process_gcp_data", "", "invalid GCP data type", nil)
	}

	tags := make([]PrefixTag, 0, len(gcpData.Prefixes))

	for _, prefix := range gcpData.Prefixes {
		if cidr := prefix.IPPrefix; cidr != "" {
			tags = append(tags, PrefixTag{
				CIDR:    cidr,
				Cloud:   "GCP",
				Service: prefix.Service,
			})
		}
	}

	c.logger.Info("Processed %d GCP prefixes", len(tags))
	return tags, nil
}

// processCloudflareData converts Cloudflare data to PrefixTags.
func (c *Cache) processCloudflareData(data interface{}) ([]PrefixTag, error) {
	body, ok := data.(string)
	if !ok {
		return nil, NewInvalidDataError("process_cloudflare_data", "", "invalid Cloudflare data type", nil)
	}

	lines := strings.Split(body, "\n")
	tags := make([]PrefixTag, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tags = append(tags, PrefixTag{
			CIDR:  line,
			Cloud: "Cloudflare",
		})
	}

	c.logger.Info("Processed %d Cloudflare prefixes", len(tags))
	return tags, nil
}

// processDigitalOceanData converts DigitalOcean data to PrefixTags.
func (c *Cache) processDigitalOceanData(data interface{}) ([]PrefixTag, error) {
	doData, ok := data.(DigitalOceanIPRanges)
	if !ok {
		return nil, NewInvalidDataError("process_do_data", "", "invalid DigitalOcean data type", nil)
	}

	tags := make([]PrefixTag, 0, len(doData.Data))

	for _, item := range doData.Data {
		tags = append(tags, PrefixTag{
			CIDR:  item.IPPrefix,
			Cloud: "DigitalOcean",
		})
	}

	c.logger.Info("Processed %d DigitalOcean prefixes", len(tags))
	return tags, nil
}

// UpdatePrefixes fetches and updates all provider prefixes.
func (c *Cache) UpdatePrefixes() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout*2)
	defer cancel()

	c.logger.Info("Starting prefix update from all providers")

	results, err := c.FetchAllProviders(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch providers: %w", err)
	}

	var allTags []PrefixTag
	for _, result := range results {
		if result.Error != nil {
			c.logger.Error("Skipping %s due to error: %v", result.Provider, result.Error)
			continue
		}

		tags, err := c.ProcessFetchResult(result)
		if err != nil {
			c.logger.Error("Failed to process %s data: %v", result.Provider, err)
			continue
		}

		allTags = append(allTags, tags...)
	}

	// Insert all tags in a single transaction for efficiency
	if err := c.insertPrefixes(allTags); err != nil {
		return fmt.Errorf("failed to update prefixes: %w", err)
	}

	c.logger.Info("Successfully updated %d prefixes from %d providers", len(allTags), len(results))
	return nil
}

// insertPrefixes efficiently inserts multiple prefixes in a single transaction.
func (c *Cache) insertPrefixes(tags []PrefixTag) error {
	err := c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCIDRTags))
		if bucket == nil {
			return NewDatabaseError("get_bucket", bucketCIDRTags, nil)
		}

		for _, tag := range tags {
			data, err := json.Marshal(tag)
			if err != nil {
				return NewInvalidDataError("marshal_prefix", tag.CIDR, "failed to marshal prefix tag", err)
			}

			if err := bucket.Put([]byte(tag.CIDR), data); err != nil {
				return NewDatabaseError("put_prefix", tag.CIDR, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	return nil
}

// AWSIPRanges represents AWS IP ranges data.
type AWSIPRanges struct {
	SyncToken  string `json:"syncToken"`
	CreateDate string `json:"createDate"`
	Prefixes   []struct {
		IPPrefix           string `json:"ip_prefix"`
		Region             string `json:"region"`
		Service            string `json:"service"`
		NetworkBorderGroup string `json:"network_border_group"`
	} `json:"prefixes"`
	IPv6Prefixes []struct {
		IPv6Prefix         string `json:"ipv6_prefix"`
		Region             string `json:"region"`
		Service            string `json:"service"`
		NetworkBorderGroup string `json:"network_border_group"`
	} `json:"ipv6_prefixes"`
}

// GCPIPRanges represents GCP IP ranges data.
type GCPIPRanges struct {
	SyncToken string `json:"syncToken"`
	Prefixes  []struct {
		IPPrefix string `json:"ipv4Prefix"`
		Service  string `json:"service"`
		Scope    string `json:"scope"`
	} `json:"prefixes"`
}

// DigitalOceanIPRanges represents DigitalOcean IP ranges data.
type DigitalOceanIPRanges struct {
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
	Data []struct {
		IPPrefix string `json:"ip_prefix"`
		Region   string `json:"region"`
	} `json:"data"`
}
