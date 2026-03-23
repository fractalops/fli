package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"fli/internal/config"
)

// Version information.
var (
	Version   = "0.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Default values.
const (
	defaultLogGroup = ""
	defaultFormat   = "table"
)

// Get default timeouts from config.
var defaultTimeouts = config.DefaultTimeouts()

// Valid format values.
var validFormats = map[string]bool{
	"table": true,
	"csv":   true,
	"json":  true,
}

var (
	// Command flags.
	flags = NewCommandFlags()

	// queryVerbs are the commands that execute a query.
	queryVerbs = []*cobra.Command{
		rawCmd, countCmd, sumCmd, avgCmd, minCmd, maxCmd,
	}
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "fli",
	Short:   "Query VPC Flow Logs using CloudWatch Logs Insights",
	Version: Version,
	Long: `Query VPC Flow Logs using CloudWatch Logs Insights.

Examples:
  # Count flows by source IP
  fli count --by srcaddr --since 1h

  # Sum bytes by destination port
  fli sum bytes --by dstport --since 30m

  # Raw query with filter
  fli raw srcaddr,dstaddr,bytes --filter "bytes > 1000"`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Check for environment variables
		if envLogGroup := os.Getenv("FLI_LOG_GROUP"); envLogGroup != "" && flags.LogGroup == "" {
			flags.LogGroup = envLogGroup
		}

		// Track if --version was explicitly passed before profile resolution
		if cmd.Flags().Changed("version") {
			flags.versionExplicitlySet = true
		}

		// Profile-based resolution (only when log group not already set)
		if flags.LogGroup == "" {
			resolveProfileFlags()
		}

		// Only validate format and version for query commands. We identify query
		// commands by checking for a "query" annotation.
		if cmd.Annotations["query"] == "true" {
			// Ensure log group is available for query commands
			if flags.LogGroup == "" {
				return fmt.Errorf("log group is required. Set it with --log-group, --profile, FLI_LOG_GROUP env, or run \"fli init\"")
			}

			if format := cmd.Flag("format").Value.String(); !validFormats[format] {
				return fmt.Errorf("invalid format %q: must be one of: table, csv, json", format)
			}
			if version := cmd.Flag("version").Value.String(); version != "2" && version != "5" {
				return fmt.Errorf("invalid version %q: must be 2 or 5", version)
			}
		}

		return nil
	},
}

// resolveProfileFlags loads the profile config and sets flags.LogGroup and flags.Version
// from the resolved profile. This is called when --log-group and FLI_LOG_GROUP are not set.
func resolveProfileFlags() {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return
	}

	profileName := cfg.ResolveProfile(flags.Profile)
	profile, ok := cfg.GetProfile(profileName)
	if !ok {
		return
	}

	flags.LogGroup = profile.LogGroup

	// Only override version if it wasn't explicitly set on the command line
	if !flags.versionExplicitlySet {
		flags.Version = profile.Version
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	flags.InitDefaults(DefaultLimit, defaultFormat, defaultTimeouts.DefaultSince)
	flags.AddCommonFlags(rootCmd)

	// Add all commands to the root command
	AddCommands()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// AddCommands adds all the commands to the root command.
func AddCommands() {
	// Add query verbs
	for _, cmd := range queryVerbs {
		cmd.Annotations = map[string]string{"query": "true"}
		flags.AddQueryFlags(cmd)
		setupQueryCommandCompletion(cmd)
		rootCmd.AddCommand(cmd)
	}

	// Add cache commands
	initCacheCommands()

	// Add init, cleanup, and profile commands
	initInitCommand()
	rootCmd.AddCommand(initCmd)

	initCleanupCommand()
	rootCmd.AddCommand(cleanupCmd)

	initProfileCommands()
	rootCmd.AddCommand(profileCmd)

	// Add completion command
	rootCmd.AddCommand(completionCmd)

	// Set up root command completion
	setupRootCommandCompletion(rootCmd)
}
