# Cache Package

The cache package provides functionality for caching and annotating network data, particularly for VPC Flow Logs analysis.

## Components

- **Core Cache**: Basic cache operations and management
- **Annotation**: IP and ENI annotation functionality
- **WHOIS**: WHOIS lookup and provider identification
- **Cloud Providers**: Fetching and processing cloud provider IP ranges
- **Display**: Formatting and displaying cache contents

## Files

- `cache_core.go` - Core cache initialization and management
- `cache_operations.go` - Cache read/write operations
- `annotator.go` - IP annotation functionality
- `cloud_fetch.go` - Cloud provider IP range fetching
- `display.go` - Output formatting for cache contents
- `errors.go` - Error types and handling
- `interfaces.go` - Interface definitions for external dependencies
- `refresh.go` - Cache refresh operations
- `whois.go` - WHOIS lookup functionality
- `config.go` - Configuration handling

## Usage

```go
// Open a cache
cache, err := cache.Open("/path/to/cache.db")
if err != nil {
    // Handle error
}
defer cache.Close()

// Refresh ENIs
err = cache.RefreshENIs(ctx, ec2Client, []string{"eni-1234567890"})

// Enrich IPs with WHOIS data
err = cache.EnrichIPs()

// Get annotations for an IP
annotation, err := cache.GetIPAnnotation("10.0.0.1")
```

## Testing

Each component has corresponding test files that demonstrate usage and verify functionality.
