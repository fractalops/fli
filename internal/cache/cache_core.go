// Package cache provides functionality for caching and annotating network data.
package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// ENITag stores ENI annotation info.
type ENITag struct {
	ENI        string   // eni-abc…
	Label      string   // "catalogue-api" (from SG name) or "unknown"
	SGNames    []string // ["cart-api-sg", "bastion-sg"]
	FirstSeen  int64
	PrivateIPs []string // Private IPs attached to this ENI
	Name       string
}

// PrefixTag stores CIDR annotation info.
type PrefixTag struct {
	CIDR    string // "13.32.0.0/15"
	Cloud   string // "AWS" | "AZURE" | "GCP"
	Service string // Optional ("CLOUDFRONT", "EC2", …)
	Fetched int64
}

// IPTag stores IP annotation info.
type IPTag struct {
	Addr string
	Name string
}

// Cache wraps BoltDB and provides annotation lookups.
type Cache struct {
	db          *bbolt.DB
	config      *Config
	httpClient  HTTPClient
	whoisClient WhoisClient
	logger      Logger
	fileSystem  FileSystem
}

const (
	bucketENITags  = "eni_tags"
	bucketCIDRTags = "cidr_tags"
	bucketIPTags   = "ip_tags"
)

// Open opens or creates the cache at the given path. It ensures the parent
// directory and necessary database buckets exist.
func Open(path string) (*Cache, error) {
	return OpenWithConfig(DefaultConfig().WithCachePath(path))
}

// OpenWithConfig opens or creates the cache with the given configuration.
func OpenWithConfig(config *Config) (*Cache, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create default dependencies if not provided
	httpClient := NewDefaultHTTPClient(config.HTTPTimeout)
	whoisClient := NewDefaultWhoisClient(config.WhoisTimeout)
	logger := NewDefaultLogger(config.EnableLogging)
	fileSystem := NewDefaultFileSystem()

	return OpenWithDependencies(config, httpClient, whoisClient, logger, fileSystem)
}

// OpenWithDependencies opens or creates the cache with custom dependencies.
func OpenWithDependencies(
	config *Config,
	httpClient HTTPClient,
	whoisClient WhoisClient,
	logger Logger,
	fileSystem FileSystem,
) (*Cache, error) {
	if config == nil {
		return nil, NewConfigurationError("config cannot be nil", nil)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(config.CachePath)
	if err := fileSystem.MkdirAll(dir, 0o755); err != nil {
		return nil, NewConfigurationError("failed to create cache directory", err)
	}

	// Open BoltDB
	db, err := bbolt.Open(config.CachePath, 0o600, &bbolt.Options{
		Timeout: config.DBTimeout,
	})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// Log the close error but return the original error
			fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", closeErr)
		}
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ensure buckets exist
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketENITags)); err != nil {
			return NewDatabaseError("create_bucket", bucketENITags, err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketCIDRTags)); err != nil {
			return NewDatabaseError("create_bucket", bucketCIDRTags, err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketIPTags)); err != nil {
			return NewDatabaseError("create_bucket", bucketIPTags, err)
		}
		return nil
	})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// Log the close error but return the original error
			fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", closeErr)
		}
		return nil, fmt.Errorf("failed to initialize database buckets: %w", err)
	}

	return &Cache{
		db:          db,
		config:      config,
		httpClient:  httpClient,
		whoisClient: whoisClient,
		logger:      logger,
		fileSystem:  fileSystem,
	}, nil
}

// Close closes the underlying BoltDB database.
func (c *Cache) Close() error {
	if c.db != nil {
		err := c.db.Close()
		if err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}
