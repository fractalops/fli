package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `Generate shell completion script for fli.

To load completions:

Bash:
  $ source <(fli completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ fli completion bash > /etc/bash_completion.d/fli
  # macOS:
  $ fli completion bash > /usr/local/etc/bash_completion.d/fli

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ fli completion zsh > "${fpath[1]}/_fli"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ fli completion fish | source

  # To load completions for each session, execute once:
  $ fli completion fish > ~/.config/fish/completions/fli.fish

PowerShell:
  PS> fli completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> fli completion powershell > fli.ps1
  # and source this file from your PowerShell profile.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	DisableFlagParsing:    true,
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating bash completion: %v\n", err)
				os.Exit(1)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating zsh completion: %v\n", err)
				os.Exit(1)
			}
		case "fish":
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating fish completion: %v\n", err)
				os.Exit(1)
			}
		case "powershell":
			if err := cmd.Root().GenPowerShellCompletion(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating powershell completion: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// fieldCompletion provides dynamic field completion based on the selected version.
func fieldCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get version from flags (default to 2 if not set)
	version := 2
	if versionFlag := cmd.Flag("version"); versionFlag != nil {
		if v, err := strconv.Atoi(versionFlag.Value.String()); err == nil {
			version = v
		}
	}

	// Get fields for the version
	fields := getFieldsForVersion(version)

	// Filter fields that match the toComplete prefix
	var matches []string
	for _, field := range fields {
		if strings.HasPrefix(field, toComplete) {
			matches = append(matches, field)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

// getFieldsForVersion returns the list of valid fields for a given VPC Flow Logs version.
func getFieldsForVersion(version int) []string {
	// Get fields based on version
	switch version {
	case 2:
		return []string{
			"version", "account_id", "interface_id", "srcaddr", "dstaddr",
			"srcport", "dstport", "protocol", "packets", "bytes",
			"start", "end", "action", "log_status", "duration",
		}
	case 3, 5:
		return []string{
			"version", "account_id", "interface_id", "srcaddr", "dstaddr",
			"srcport", "dstport", "protocol", "packets", "bytes",
			"start", "end", "action", "log_status", "vpc_id", "subnet_id",
			"instance_id", "tcp_flags", "type", "pkt_srcaddr", "pkt_dstaddr",
			"region", "az_id", "sublocation_type", "sublocation_id",
			"pkt_src_aws_service", "pkt_dst_aws_service", "flow_direction",
			"traffic_path", "duration",
		}
	default:
		// Return v2 fields as fallback
		return []string{
			"version", "account_id", "interface_id", "srcaddr", "dstaddr",
			"srcport", "dstport", "protocol", "packets", "bytes",
			"start", "end", "action", "log_status", "duration",
		}
	}
}

// formatCompletion provides completion for output format options.
func formatCompletion(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"table", "csv", "json"}
	var matches []string
	for _, format := range formats {
		if strings.HasPrefix(format, toComplete) {
			matches = append(matches, format)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

// timeCompletion provides completion for common time ranges.
func timeCompletion(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	times := []string{
		"5m", "10m", "15m", "30m",
		"1h", "2h", "6h", "12h",
		"1d", "2d", "7d", "30d",
	}
	var matches []string
	for _, time := range times {
		if strings.HasPrefix(time, toComplete) {
			matches = append(matches, time)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

// versionCompletion provides completion for VPC Flow Logs versions.
func versionCompletion(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	versions := []string{"2", "5"}
	var matches []string
	for _, version := range versions {
		if strings.HasPrefix(version, toComplete) {
			matches = append(matches, version)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

// setupQueryCommandCompletion configures completion for query commands.
func setupQueryCommandCompletion(cmd *cobra.Command) {
	// Set up field completion for positional arguments
	cmd.ValidArgsFunction = fieldCompletion

	// Set up flag completion
	err := cmd.RegisterFlagCompletionFunc("format", formatCompletion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up format completion: %v\n", err)
	}
	err = cmd.RegisterFlagCompletionFunc("since", timeCompletion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up since completion: %v\n", err)
	}
	err = cmd.RegisterFlagCompletionFunc("by", fieldCompletion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up by completion: %v\n", err)
	}
	// Simple field completion for filter instead of complex parsing
	err = cmd.RegisterFlagCompletionFunc("filter", fieldCompletion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up filter completion: %v\n", err)
	}
}

// setupRootCommandCompletion configures completion for the root command.
func setupRootCommandCompletion(cmd *cobra.Command) {
	// Set up flag completion for persistent flags
	// Only register completion for flags that exist
	if cmd.Flags().Lookup("version") != nil {
		err := cmd.RegisterFlagCompletionFunc("version", versionCompletion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up version completion: %v\n", err)
		}
	}
}
