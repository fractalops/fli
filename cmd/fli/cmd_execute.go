package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// initExecuteCommand adds the execute command to the root command.
func initExecuteCommand() {
	executeCmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute a query from a YAML configuration file",
		Long: `Execute a query from a YAML configuration file.

Examples:
  # Execute a query from a file
  fli execute -f query.yaml

  # Execute a query from stdin
  fli count srcaddr --filter "dstport=443" --since 1h --dry-run | fli execute -f -`,
		RunE: runExecuteCmd,
	}

	executeCmd.Flags().StringP("file", "f", "", "YAML file containing query configuration (use '-' for stdin)")
	if err := executeCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to mark file flag as required: %v\n", err)
	}

	rootCmd.AddCommand(executeCmd)
}
