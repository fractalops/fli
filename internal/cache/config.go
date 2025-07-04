package cache

import (
	"time"

	"fli/internal/config"
)

// Config holds all configuration for the cache package.
type Config struct {
	// Database settings
	CachePath string
	DBTimeout time.Duration

	// HTTP client settings
	HTTPTimeout time.Duration
	UserAgent   string

	// Whois settings
	WhoisTimeout time.Duration

	// Provider URLs
	ProviderURLs map[string]string

	// Feature flags
	EnableWhoisEnrichment bool
	EnableLogging         bool
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	timeouts := config.DefaultTimeouts()

	return &Config{
		CachePath:             "cache.db",
		DBTimeout:             timeouts.DB,
		HTTPTimeout:           timeouts.HTTP,
		UserAgent:             "fli-cache/1.0",
		WhoisTimeout:          timeouts.Whois,
		EnableWhoisEnrichment: true,
		EnableLogging:         true,
		ProviderURLs: map[string]string{
			"aws":          "https://ip-ranges.amazonaws.com/ip-ranges.json",
			"gcp":          "https://www.gstatic.com/ipranges/cloud.json",
			"gcp_legacy":   "https://www.gstatic.com/ipranges/goog.json",
			"cloudflare":   "https://www.cloudflare.com/ips-v4",
			"digitalocean": "https://digitalocean.com/geo/google.json",
		},
	}
}

// WithCachePath sets the cache database path.
func (c *Config) WithCachePath(path string) *Config {
	c.CachePath = path
	return c
}

// WithDBTimeout sets the database timeout.
func (c *Config) WithDBTimeout(timeout time.Duration) *Config {
	c.DBTimeout = timeout
	return c
}

// WithHTTPTimeout sets the HTTP client timeout.
func (c *Config) WithHTTPTimeout(timeout time.Duration) *Config {
	c.HTTPTimeout = timeout
	return c
}

// WithWhoisTimeout sets the whois lookup timeout.
func (c *Config) WithWhoisTimeout(timeout time.Duration) *Config {
	c.WhoisTimeout = timeout
	return c
}

// WithProviderURL sets a custom URL for a specific provider.
func (c *Config) WithProviderURL(provider, url string) *Config {
	if c.ProviderURLs == nil {
		c.ProviderURLs = make(map[string]string)
	}
	c.ProviderURLs[provider] = url
	return c
}

// WithWhoisEnrichment enables or disables whois enrichment.
func (c *Config) WithWhoisEnrichment(enabled bool) *Config {
	c.EnableWhoisEnrichment = enabled
	return c
}

// WithLogging enables or disables logging.
func (c *Config) WithLogging(enabled bool) *Config {
	c.EnableLogging = enabled
	return c
}
