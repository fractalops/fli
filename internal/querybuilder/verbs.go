// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
	"strings"
)

// Verb represents a CloudWatch Logs Insights query verb.
//
// The stringer tool generates the String() method for this type in verb_string.go.
// When adding new verbs, run 'go generate' to update the String() method.
//
//go:generate stringer -type=Verb
type Verb int

const (
	// VerbRaw returns raw log entries without aggregation.
	VerbRaw Verb = iota
	// VerbCount counts the number of log entries.
	VerbCount
	// VerbSum calculates the sum of a numeric field.
	VerbSum
	// VerbAvg calculates the average of a numeric field.
	VerbAvg
	// VerbMin finds the minimum value of a numeric field.
	VerbMin
	// VerbMax finds the maximum value of a numeric field.
	VerbMax
)

// ParseVerb converts a string to a Verb.
// This function complements the auto-generated String() method in verb_string.go
// by providing the reverse operation: converting a string to a Verb.
func ParseVerb(s string) (Verb, error) {
	switch strings.ToLower(s) {
	case "raw":
		return VerbRaw, nil
	case "count":
		return VerbCount, nil
	case "sum":
		return VerbSum, nil
	case "avg":
		return VerbAvg, nil
	case "min":
		return VerbMin, nil
	case "max":
		return VerbMax, nil
	default:
		return VerbRaw, fmt.Errorf("unknown verb: %s", s)
	}
}
