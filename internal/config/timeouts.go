// Package config provides configuration structures and defaults for the application.
package config

import "time"

// Timeouts defines standard timeout values used throughout the application.
type Timeouts struct {
	// Query is the default timeout for CloudWatch Logs Insights queries
	Query time.Duration

	// DefaultSince is the default time window to look back for queries
	DefaultSince time.Duration

	// DB is the timeout for database operations
	DB time.Duration

	// HTTP is the timeout for HTTP requests
	HTTP time.Duration

	// Whois is the timeout for WHOIS lookups
	Whois time.Duration

	// MaxPoll is the maximum interval between query status checks
	MaxPoll time.Duration
}

// DefaultTimeouts returns the default timeout configuration.
func DefaultTimeouts() Timeouts {
	return Timeouts{
		Query:        5 * time.Minute,
		DefaultSince: 5 * time.Minute,
		DB:           30 * time.Second,
		HTTP:         10 * time.Second,
		Whois:        5 * time.Second,
		MaxPoll:      10 * time.Second,
	}
}
