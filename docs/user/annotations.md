# Annotations and Caching

FLI includes a powerful annotation system that enriches VPC Flow Log data with meaningful context. This document explains how to use and manage annotations.

## Overview

The annotation system automatically adds human-readable labels to:
- ENIs (Elastic Network Interfaces)
- IP addresses
- CIDR blocks

These annotations appear inline in query results, making it easier to understand your network traffic.

## Cache Location

Annotations are stored in a local cache file:
- Default location: `~/.fli/cache/anno.db`
- Can be configured via environment variables

## Annotation Types

### 1. ENI Annotations
- Labels ENIs based on security group names
- Example: `eni-01234567 (api-server-sg)`
- Includes:
  - Security group names
  - Private IP mappings
  - First seen timestamp

### 2. IP Annotations
- Labels IP addresses with meaningful context
- Examples:
  - `1.1.1.1 (CLOUDFLARE)`
  - `13.32.0.1 (AWS-CLOUDFRONT)`
  - `34.56.78.90 (US)`
- Sources:
  - WHOIS data
  - Cloud provider information
  - Manual annotations

### 3. CIDR Block Annotations
- Labels IP ranges with provider and service info
- Examples:
  - AWS service ranges
  - GCP service ranges
  - Cloudflare ranges

## Commands

### Refresh Cache
```bash
# Refresh specific ENIs
fli cache refresh --eni eni-01234567

# Refresh all cached ENIs
fli cache refresh --all

# Cache results from queries
fli query raw --save-enis --save-ips
```

### Manage Cloud Provider Prefixes
```bash
# Fetch and cache cloud provider IP ranges
fli cache prefixes

# Supported providers:
# - AWS
# - GCP
# - Cloudflare
```

### View Cache Contents
```bash
# List all cached items
fli cache list

# Clean cache
fli cache clean
```

## Automatic Enrichment

When running queries with `--save-enis` or `--save-ips` flags:
1. New ENIs and IPs are automatically added to cache
2. Results are enriched with cached annotations
3. Public IPs are enriched with WHOIS data

Example output:
```
| srcaddr                    | dstaddr                    | action |
|---------------------------|----------------------------|--------|
| 10.0.1.10 (api-server)    | 1.1.1.1 (CLOUDFLARE)      | ACCEPT |
| 172.16.0.5 (worker-node)  | 3.5.6.7 (AWS-CLOUDFRONT)  | ACCEPT |
```

## Performance Considerations

- Cache lookups add minimal overhead (<50µs per flow)
- Batch operations optimize performance
- WHOIS lookups only performed during cache refresh
- Efficient longest-prefix matching for CIDR blocks

## Cache Persistence

- Cache survives across tool restarts
- Uses BoltDB for reliable storage
- ACID compliant
- Automatic parent directory creation

## Limitations

1. **Initial Setup Required**
   - Run `fli cache prefixes` to populate cloud ranges
   - Run `fli cache refresh` to populate ENI info

2. **Network Dependencies**
   - WHOIS enrichment requires internet access
   - Cloud prefix fetching requires internet access

3. **Storage Growth**
   - Cache file grows with number of annotations
   - Use `fli cache clean` to reset if needed

## Best Practices

1. **Regular Updates**
   ```bash
   # Update cloud prefixes weekly
   fli cache prefixes
   
   # Refresh ENI cache after infrastructure changes
   fli cache refresh --all
   ```

2. **Query Enrichment**
   ```bash
   # Use save flags for enriched results
   fli raw --save-enis --save-ips
   ```

3. **Cache Management**
   - Periodically review cache contents
   - Clean and refresh if annotations become stale
   - Back up cache file if needed 