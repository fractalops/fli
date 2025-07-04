package cache

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

// IPAnnotator provides efficient IP address annotation using CIDR prefix matching.
type IPAnnotator struct {
	mu      sync.RWMutex
	root    *annotatorNode
	entries map[string]*PrefixTag // CIDR -> PrefixTag for quick lookups
}

type annotatorNode struct {
	children map[byte]*annotatorNode
	prefix   *PrefixTag
}

// NewIPAnnotator creates a new IP annotator for efficient CIDR lookups.
func NewIPAnnotator() *IPAnnotator {
	return &IPAnnotator{
		root:    &annotatorNode{children: make(map[byte]*annotatorNode)},
		entries: make(map[string]*PrefixTag),
	}
}

// Insert adds a CIDR prefix to the annotator.
func (ia *IPAnnotator) Insert(prefix *PrefixTag) error {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	// Parse the CIDR
	parsed, err := netip.ParsePrefix(prefix.CIDR)
	if err != nil {
		return NewValidationError("insert_prefix", prefix.CIDR, "invalid CIDR format")
	}

	// Store in quick lookup map
	ia.entries[prefix.CIDR] = prefix

	// Build annotator path from IP bytes
	addr := parsed.Addr()
	bytes := addr.AsSlice()
	current := ia.root
	for i, b := range bytes {
		if current.children == nil {
			current.children = make(map[byte]*annotatorNode)
		}

		if current.children[b] == nil {
			current.children[b] = &annotatorNode{}
		}
		current = current.children[b]

		// Store prefix at this level if it's the most specific match so far
		if i >= parsed.Bits()-1 {
			current.prefix = prefix
		}
	}

	return nil
}

// Lookup finds the longest matching prefix for an IP address.
func (ia *IPAnnotator) Lookup(addr netip.Addr) *PrefixTag {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	bytes := addr.AsSlice()
	current := ia.root
	var bestMatch *PrefixTag

	for _, b := range bytes {
		if current.children == nil {
			break
		}

		child, exists := current.children[b]
		if !exists {
			break
		}

		if child.prefix != nil {
			bestMatch = child.prefix
		}
		current = child
	}

	return bestMatch
}

// Remove removes a CIDR prefix from the annotator.
func (ia *IPAnnotator) Remove(cidr string) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	delete(ia.entries, cidr)
	// Note: Full annotator cleanup would be more complex
	// For now, we just remove from the quick lookup map
}

// GetAll returns all prefixes in the annotator.
func (ia *IPAnnotator) GetAll() []*PrefixTag {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	result := make([]*PrefixTag, 0, len(ia.entries))
	for _, prefix := range ia.entries {
		result = append(result, prefix)
	}
	return result
}

// Metrics provides metrics about the cache usage.
type Metrics struct {
	ENICount    int64
	IPCount     int64
	PrefixCount int64
	LookupCount int64
	HitCount    int64
	MissCount   int64
	LastReset   int64
}

// AnnotatedCache extends Cache with efficient IP annotation capabilities.
type AnnotatedCache struct {
	*Cache
	ipAnnotator *IPAnnotator
	metrics     *Metrics
	metricsMu   sync.RWMutex
}

// NewAnnotatedCache creates a cache with efficient IP annotation capabilities.
func NewAnnotatedCache(cache *Cache) *AnnotatedCache {
	return &AnnotatedCache{
		Cache:       cache,
		ipAnnotator: NewIPAnnotator(),
		metrics:     &Metrics{},
	}
}

// BuildAnnotator rebuilds the IP annotator from the database.
func (ac *AnnotatedCache) BuildAnnotator() error {
	prefixes, err := ac.ListPrefixes()
	if err != nil {
		return err
	}

	// Clear existing annotator.
	ac.ipAnnotator = NewIPAnnotator()

	// Rebuild annotator
	for _, cidr := range prefixes {
		addr, err := netip.ParseAddr(cidr)
		if err != nil {
			continue // Skip invalid prefixes
		}

		// Get the prefix tag
		annotation, err := ac.LookupIP(addr)
		if err != nil {
			continue
		}

		// Parse annotation to extract prefix info
		// This is a simplified version - in practice you'd want to store
		// the actual PrefixTag objects in the annotator
		if annotation != "" {
			// Create a minimal PrefixTag for the annotator
			prefixTag := &PrefixTag{
				CIDR: cidr,
			}
			if err := ac.ipAnnotator.Insert(prefixTag); err != nil {
				// Log the error but continue processing other prefixes
				// This is a non-critical error in annotator building
				continue
			}
		}
	}

	return nil
}

// LookupIP uses the IP annotator for efficient lookups.
func (ac *AnnotatedCache) LookupIP(addr netip.Addr) (string, error) {
	ac.metricsMu.Lock()
	ac.metrics.LookupCount++
	ac.metricsMu.Unlock()

	// First check exact IP match
	ipStr := addr.String()
	err := ac.db.View(func(tx *bbolt.Tx) error {
		ipBucket := tx.Bucket([]byte(bucketIPTags))
		if ipBucket != nil {
			if v := ipBucket.Get([]byte(ipStr)); v != nil {
				var tag IPTag
				if err := json.Unmarshal(v, &tag); err == nil {
					ac.metricsMu.Lock()
					ac.metrics.HitCount++
					ac.metricsMu.Unlock()
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to lookup IP in database: %w", err)
	}

	// Then check IP annotator
	prefixMatch := ac.ipAnnotator.Lookup(addr)
	if prefixMatch != nil {
		ac.metricsMu.Lock()
		ac.metrics.HitCount++
		ac.metricsMu.Unlock()

		annotation := fmt.Sprintf("%s (%s)", prefixMatch.Cloud, prefixMatch.CIDR)
		if prefixMatch.Service != "" {
			annotation = fmt.Sprintf("%s, %s", annotation, prefixMatch.Service)
		}
		return annotation, nil
	}

	ac.metricsMu.Lock()
	ac.metrics.MissCount++
	ac.metricsMu.Unlock()
	return "", nil
}

// GetMetrics returns current cache metrics.
func (ac *AnnotatedCache) GetMetrics() Metrics {
	ac.metricsMu.RLock()
	defer ac.metricsMu.RUnlock()

	// Get current counts from database
	enis, _ := ac.ListENIs()
	ips, _ := ac.ListIPs()
	prefixes, _ := ac.ListPrefixes()

	metrics := *ac.metrics
	metrics.ENICount = int64(len(enis))
	metrics.IPCount = int64(len(ips))
	metrics.PrefixCount = int64(len(prefixes))

	return metrics
}

// ResetMetrics resets the metrics.
func (ac *AnnotatedCache) ResetMetrics() {
	ac.metricsMu.Lock()
	defer ac.metricsMu.Unlock()

	ac.metrics = &Metrics{
		LastReset: time.Now().Unix(),
	}
}
