// Package main provides the command-line interface for the fli tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/cobra"

	"fli/internal/aws"
	"fli/internal/cache"
	fliconfig "fli/internal/config"
)

var (
	// Cache-related flags.
	cachePath string
	eniIDs    []string
	allENIs   bool
	verbose   bool

	// Cache-related commands.
	cacheCmd = &cobra.Command{
		Use:   "cache",
		Short: "Cache and annotation operations",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// Initialize cache path early
			return initCachePath()
		},
	}
)

// initCacheCommands initializes all cache-related commands.
func initCacheCommands() {
	// Add common cache flags first
	cacheCmd.PersistentFlags().StringVar(&cachePath, "cache", DefaultCachePath, "Path to cache file")
	cacheCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	// Add cache command to root
	rootCmd.AddCommand(cacheCmd)

	// Cache refresh command
	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh ENI tags in the cache using AWS",
		RunE:  runCacheRefresh,
	}
	refreshCmd.Flags().StringSliceVar(&eniIDs, "eni", nil, "ENI IDs to refresh")
	refreshCmd.Flags().BoolVar(&allENIs, "all", false, "Refresh all ENIs in cache")
	cacheCmd.AddCommand(refreshCmd)

	// Cache list command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List cached items",
		RunE:  runCacheList,
	}
	cacheCmd.AddCommand(listCmd)

	// Cache prefixes command
	prefixesCmd := &cobra.Command{
		Use:   "prefixes",
		Short: "Update cloud provider IP ranges",
		RunE:  runCachePrefixes,
	}
	cacheCmd.AddCommand(prefixesCmd)

	// Cache clean command
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete the cache file",
		RunE:  runCacheClean,
	}
	cacheCmd.AddCommand(cleanCmd)
}

// initCachePath ensures the cache path is properly initialized.
func initCachePath() error {
	// Expand home directory if needed
	if strings.HasPrefix(cachePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		cachePath = strings.Replace(cachePath, "~", home, 1)
	}

	// Ensure cache directory exists
	cacheDir := strings.TrimSuffix(cachePath, "/anno.db")
	if err := os.MkdirAll(cacheDir, fliconfig.DirPermissions); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return nil
}

// runCacheRefresh implements the cache refresh command.
func runCacheRefresh(cmd *cobra.Command, _ []string) error {
	if err := initCachePath(); err != nil {
		return fmt.Errorf("failed to initialize cache path: %w", err)
	}

	if len(eniIDs) == 0 && !allENIs {
		return fmt.Errorf("at least one --eni must be provided, or use --all to refresh all cached ENIs")
	}

	if verbose {
		if _, err := fmt.Fprintf(os.Stdout, "Opening cache at %s...\n", cachePath); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}
	cacheObj, err := cache.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache: %w", err)
	}
	defer func() {
		if closeErr := cacheObj.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close cache: %v\n", closeErr)
		}
	}()

	ctx := cmd.Context()
	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	ec2Svc := awsec2.NewFromConfig(awsCfg)
	ec2Client := aws.NewEC2Client(ec2Svc)

	if allENIs {
		if err := cacheObj.RefreshAllENIs(ctx, ec2Client); err != nil {
			return fmt.Errorf("failed to refresh all ENIs: %w", err)
		}
	} else {
		if err := cacheObj.RefreshENIs(ctx, ec2Client, eniIDs); err != nil {
			return fmt.Errorf("failed to refresh ENIs: %w", err)
		}
	}

	// Whois enrichment for public IPs
	if err := cacheObj.EnrichIPs(); err != nil {
		return fmt.Errorf("failed to enrich IPs: %w", err)
	}
	return nil
}

// runCacheList implements the cache list command.
func runCacheList(cmd *cobra.Command, _ []string) error {
	if err := initCachePath(); err != nil {
		return fmt.Errorf("failed to initialize cache path: %w", err)
	}

	if verbose {
		if _, err := fmt.Fprintf(os.Stdout, "Opening cache at %s...\n", cachePath); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}
	cacheObj, err := cache.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache: %w", err)
	}
	defer func() {
		if closeErr := cacheObj.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close cache: %v\n", closeErr)
		}
	}()

	output, err := cacheObj.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list cache contents: %w", err)
	}
	if _, err := fmt.Fprint(os.Stdout, output); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}
	return nil
}

// runCachePrefixes implements the cache prefixes command.
func runCachePrefixes(_ *cobra.Command, _ []string) error {
	if err := initCachePath(); err != nil {
		return fmt.Errorf("failed to initialize cache path: %w", err)
	}

	if verbose {
		if _, err := fmt.Fprintf(os.Stdout, "Opening cache at %s...\n", cachePath); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}
	cacheObj, err := cache.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache at %s: %w", cachePath, err)
	}
	defer func() {
		if closeErr := cacheObj.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close cache: %v\n", closeErr)
		}
	}()

	return fmt.Errorf("failed to update prefixes: %w", cacheObj.UpdatePrefixes())
}

// runCacheClean implements the cache clean command.
func runCacheClean(_ *cobra.Command, _ []string) error {
	if err := initCachePath(); err != nil {
		return fmt.Errorf("failed to initialize cache path: %w", err)
	}

	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "Deleted cache file at %s\n", cachePath); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}
	return nil
}
