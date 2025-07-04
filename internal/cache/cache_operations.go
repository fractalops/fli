package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"go.etcd.io/bbolt"
)

// LookupIP searches for an IP address in the cache and returns a formatted
// annotation string if found. Uses a longest-prefix match for CIDR
// blocks and an exact match for specific IP tags.
func (c *Cache) LookupIP(addr netip.Addr) (string, error) {
	var annotation string
	err := c.db.View(func(tx *bbolt.Tx) error {
		ipStr := addr.String()

		// 1. Exact match in IPTags
		ipBucket := tx.Bucket([]byte(bucketIPTags))
		if ipBucket != nil {
			if v := ipBucket.Get([]byte(ipStr)); v != nil {
				var tag IPTag
				if err := json.Unmarshal(v, &tag); err == nil {
					annotation = tag.Name // Exact match found
					return nil            // We're done
				}
			}
		}

		// 2. Longest-prefix match in CIDRTags
		cidrBucket := tx.Bucket([]byte(bucketCIDRTags))
		if cidrBucket == nil {
			return nil // No CIDR tags to check
		}

		var bestTag *PrefixTag
		var bestPrefixLen int
		err := cidrBucket.ForEach(func(k, v []byte) error {
			prefix, err := netip.ParsePrefix(string(k))
			if err != nil {
				return fmt.Errorf("invalid CIDR key %q: %w", string(k), err)
			}
			if prefix.Contains(addr) {
				if plen := prefix.Bits(); plen > bestPrefixLen {
					var tag PrefixTag
					if err := json.Unmarshal(v, &tag); err == nil {
						bestTag = &tag
						bestPrefixLen = plen
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to iterate CIDR bucket: %w", err)
		}

		if bestTag != nil {
			annotation = fmt.Sprintf("%s (%s)", bestTag.Cloud, bestTag.CIDR)
			if bestTag.Service != "" {
				annotation = fmt.Sprintf("%s, %s", annotation, bestTag.Service)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to lookup IP: %w", err)
	}
	return annotation, nil
}

// LookupEni returns the ENITag for the given ENI, if any.
func (c *Cache) LookupEni(ctx context.Context, eni string) (*ENITag, error) {
	var tag ENITag

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
		// Continue with lookup
	}

	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketENITags))
		v := b.Get([]byte(eni))
		if v == nil {
			return nil // Not found is not an error
		}
		return json.Unmarshal(v, &tag)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ENI: %w", err)
	}
	// If we get here with an empty tag, it means not found was returned from the DB.
	if tag.ENI == "" {
		return nil, nil
	}
	return &tag, nil
}

// UpsertEni inserts or updates an ENITag in the cache.
func (c *Cache) UpsertEni(tag ENITag) error {
	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal ENI tag: %w", err)
	}
	err = c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketENITags))
		if b == nil {
			return fmt.Errorf("ENI tag bucket missing")
		}
		return b.Put([]byte(tag.ENI), data)
	})
	if err != nil {
		return fmt.Errorf("failed to update ENI tag: %w", err)
	}
	return nil
}

// UpsertPrefix inserts or updates a PrefixTag in the cache.
func (c *Cache) UpsertPrefix(tag PrefixTag) error {
	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal prefix tag: %w", err)
	}
	err = c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCIDRTags))
		if b == nil {
			return fmt.Errorf("CIDR tag bucket missing")
		}
		return b.Put([]byte(tag.CIDR), data)
	})
	if err != nil {
		return fmt.Errorf("failed to update prefix tag: %w", err)
	}
	return nil
}

// UpsertIP inserts or updates an IPTag in the cache.
func (c *Cache) UpsertIP(tag IPTag) error {
	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal IP tag: %w", err)
	}
	err = c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketIPTags))
		if b == nil {
			return fmt.Errorf("IP tag bucket missing")
		}
		return b.Put([]byte(tag.Addr), data)
	})
	if err != nil {
		return fmt.Errorf("failed to update IP tag: %w", err)
	}
	return nil
}

// UpsertPrefixes inserts or updates multiple PrefixTags in a single transaction.
func (c *Cache) UpsertPrefixes(tags []PrefixTag) error {
	err := c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCIDRTags))
		if b == nil {
			return fmt.Errorf("CIDR tag bucket missing")
		}
		for _, tag := range tags {
			data, err := json.Marshal(tag)
			if err != nil {
				return fmt.Errorf("failed to marshal prefix tag: %w", err)
			}
			if err := b.Put([]byte(tag.CIDR), data); err != nil {
				return fmt.Errorf("failed to put prefix tag: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update prefix tags: %w", err)
	}
	return nil
}

// ListENIs returns all ENI IDs stored in the cache.
func (c *Cache) ListENIs() ([]string, error) {
	var enis []string
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketENITags))
		if b == nil {
			return fmt.Errorf("ENI tag bucket missing")
		}
		return b.ForEach(func(k, _ []byte) error {
			enis = append(enis, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ENIs: %w", err)
	}
	return enis, nil
}

// ListIPs returns all IP addresses with custom tags stored in the cache.
func (c *Cache) ListIPs() ([]string, error) {
	var ips []string
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketIPTags))
		if b == nil {
			return fmt.Errorf("IP tag bucket missing")
		}
		return b.ForEach(func(k, _ []byte) error {
			ips = append(ips, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list IPs: %w", err)
	}
	return ips, nil
}

// ListPrefixes returns all CIDR prefixes stored in the cache.
func (c *Cache) ListPrefixes() ([]string, error) {
	var prefixes []string
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketCIDRTags))
		if b == nil {
			return fmt.Errorf("CIDR tag bucket missing")
		}
		return b.ForEach(func(k, _ []byte) error {
			prefixes = append(prefixes, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list prefixes: %w", err)
	}
	return prefixes, nil
}

// DeleteENI removes an ENI from the cache.
func (c *Cache) DeleteENI(eni string) error {
	err := c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketENITags))
		if b == nil {
			return fmt.Errorf("ENI tag bucket missing")
		}
		return b.Delete([]byte(eni))
	})
	if err != nil {
		return fmt.Errorf("failed to delete ENI: %w", err)
	}
	return nil
}
