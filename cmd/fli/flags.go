package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"fli/internal/config"
)

// CommandFlags holds all the flags for the CLI commands.
type CommandFlags struct {
	// Common flags
	DryRun     bool
	Debug      bool
	UseColor   bool
	NoPtr      bool
	ProtoNames bool

	// Query-specific flags
	Limit    int
	Format   string
	Since    time.Duration // Time window to look back
	Filter   string        // Filter expression
	By       string        // Group by field(s)
	SaveENIs bool          // Save ENIs found in results to the cache
	SaveIPs  bool          // Save public IPs found in results to the cache

	// Metadata flags
	Collection       bool   // Output as a query collection
	QueryName        string // Name for the query
	QueryDescription string // Description for the query
	QueryTags        string // Comma-separated tags for the query

	// AWS-specific flags
	LogGroup     string
	Version      int
	QueryTimeout time.Duration
}

// NewCommandFlags creates a new CommandFlags instance with default values.
func NewCommandFlags() *CommandFlags {
	timeouts := config.DefaultTimeouts()

	flags := &CommandFlags{
		DryRun:           false,
		Debug:            false,
		UseColor:         true,
		NoPtr:            true,
		ProtoNames:       true,
		Limit:            20,
		Format:           "table",
		Since:            timeouts.DefaultSince,
		Filter:           "",
		By:               "",
		SaveENIs:         false,
		SaveIPs:          false,
		LogGroup:         "",
		Version:          2,
		QueryTimeout:     timeouts.Query,
		Collection:       false,
		QueryName:        "",
		QueryDescription: "",
		QueryTags:        "",
	}

	// Load default log group from environment variable
	if envLogGroup := os.Getenv("FLI_LOG_GROUP"); envLogGroup != "" {
		flags.LogGroup = envLogGroup
	}

	return flags
}

// InitDefaults initializes the command flags with default values.
func (f *CommandFlags) InitDefaults(limit int, format string, since time.Duration) {
	f.Limit = limit
	f.Format = format
	f.Since = since
}

// AddCommonFlags adds common flags to a command.
func (f *CommandFlags) AddCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&f.DryRun, "dry-run", false, "Show the query that would be executed without running it")
	cmd.PersistentFlags().StringVarP(&f.LogGroup, "log-group", "l", f.LogGroup, "CloudWatch Logs group containing flow logs")
	cmd.PersistentFlags().IntVarP(&f.Version, "version", "v", f.Version, "VPC Flow Logs format version (2 or 5)")
	cmd.PersistentFlags().BoolVar(&f.UseColor, "color", f.UseColor, "Colorize output (ACCEPT as green, REJECT as red)")
	cmd.PersistentFlags().BoolVar(&f.NoPtr, "no-ptr", f.NoPtr, "Remove @ptr fields from output")
	cmd.PersistentFlags().BoolVar(&f.ProtoNames, "proto-names", f.ProtoNames, "Use protocol names instead of numbers")
	cmd.PersistentFlags().BoolVar(&f.Debug, "debug", f.Debug, "Enable debug output")
}

// AddQueryFlags adds common query flags to a command.
func (f *CommandFlags) AddQueryFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.Limit, "limit", f.Limit, "Maximum number of results to return")
	cmd.Flags().StringVarP(&f.Format, "format", "o", f.Format, "Output format (table, csv, json)")
	cmd.Flags().DurationVarP(&f.Since, "since", "s", f.Since, "Time window to look back (e.g., 5m, 1h, 30s)")
	cmd.Flags().StringVarP(&f.Filter, "filter", "f", f.Filter, "Filter expression (e.g., 'srcaddr=10.0.0.1 and dstport=443')")
	cmd.Flags().StringVar(&f.By, "by", f.By, "Group by field(s), comma-separated if multiple")
	cmd.Flags().BoolVar(&f.SaveENIs, "save-enis", false, "Save ENIs found in results to the cache")
	cmd.Flags().BoolVar(&f.SaveIPs, "save-ips", false, "Save public IPs found in results to the cache")
	cmd.Flags().DurationVarP(&f.QueryTimeout, "timeout", "t", f.QueryTimeout, "Query timeout (e.g., 30s, 5m, 1h)")

	// Metadata flags
	cmd.Flags().BoolVar(&f.Collection, "collection", false, "Output as a query collection")
	cmd.Flags().StringVar(&f.QueryName, "name", "", "Name for the query")
	cmd.Flags().StringVar(&f.QueryDescription, "description", "", "Description for the query")
	cmd.Flags().StringVar(&f.QueryTags, "tags", "", "Comma-separated tags for the query")
}
