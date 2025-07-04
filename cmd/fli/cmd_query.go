package main

import (
	"github.com/spf13/cobra"

	"fli/internal/querybuilder"
)

// Subcommands for main verbs.
var rawCmd = &cobra.Command{
	Use:   "raw [field...]",
	Short: "Raw query with specified fields",
	Long: `Raw query that returns the specified fields from VPC Flow Logs.

Examples:
  # Show source and destination addresses
  fli raw srcaddr,dstaddr --since 10m

  # Show rejected traffic
  fli raw srcaddr,dstaddr,dstport --filter "action = 'REJECT'" --since 1h`,
	RunE: runVerb(querybuilder.VerbRaw),
}

var countCmd = &cobra.Command{
	Use:   "count [field...]",
	Short: "Count flows, optionally grouped by fields",
	Long: `Count the number of flows, optionally grouped by specified fields.

Examples:
  # Count all flows in the last hour
  fli count --since 1h

  # Count flows by source address
  fli count --by srcaddr --since 1h

  # Count flows with a specific filter
  fli count srcaddr --filter "dstport = 443"`,
	RunE: runVerb(querybuilder.VerbCount),
}

var sumCmd = &cobra.Command{
	Use:   "sum <field...>",
	Short: "Sum numeric fields, grouped by optional fields",
	Long: `Sum numeric fields (e.g., bytes, packets), optionally grouped by specified fields.

Examples:
  # Sum bytes by source address
  fli sum bytes --by srcaddr --since 1h

  # Sum packets for HTTPS traffic
  fli sum packets --filter "dstport = 443" --since 1h`,
	RunE: runVerb(querybuilder.VerbSum),
}

var avgCmd = &cobra.Command{
	Use:   "avg <field...>",
	Short: "Average numeric fields, grouped by optional fields",
	Long: `Calculate the average of numeric fields (e.g., bytes, packets), optionally grouped by specified fields.

Examples:
  # Average bytes by source address
  fli avg bytes --by srcaddr --since 1h

  # Average packets for HTTPS traffic
  fli avg packets --filter "dstport = 443" --since 1h`,
	RunE: runVerb(querybuilder.VerbAvg),
}

var minCmd = &cobra.Command{
	Use:   "min <field...>",
	Short: "Find minimum of numeric fields, grouped by optional fields",
	Long: `Find the minimum value of numeric fields (e.g., bytes, packets), optionally grouped by specified fields.

Examples:
  # Minimum bytes by source address
  fli min bytes --by srcaddr --since 1h

  # Minimum packets for HTTPS traffic
  fli min packets --filter "dstport = 443" --since 1h`,
	RunE: runVerb(querybuilder.VerbMin),
}

var maxCmd = &cobra.Command{
	Use:   "max <field...>",
	Short: "Find maximum of numeric fields, grouped by optional fields",
	Long: `Find the maximum value of numeric fields (e.g., bytes, packets), optionally grouped by specified fields.

Examples:
  # Maximum bytes by source address
  fli max bytes --by srcaddr --since 1h

  # Maximum packets for HTTPS traffic
  fli max packets --filter "dstport = 443" --since 1h`,
	RunE: runVerb(querybuilder.VerbMax),
}
