package main

import (
	"fmt"
	"strings"

	"fli/internal/querybuilder"
)

// buildCommandOptions builds the query options based on command flags.
func buildCommandOptions(schema querybuilder.Schema, args []string, cmdFlags *CommandFlags) ([]querybuilder.Option, error) {
	var opts []querybuilder.Option

	// Add version
	opts = append(opts, querybuilder.WithVersion(cmdFlags.Version))

	// Parse verb from first argument
	if len(args) < 1 {
		return nil, fmt.Errorf("verb is required")
	}
	verb, err := querybuilder.ParseVerb(args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid verb '%s': %w", args[0], err)
	}

	// Add limit
	opts = append(opts, querybuilder.WithLimit(cmdFlags.Limit))

	// Handle raw verb separately
	if verb == querybuilder.VerbRaw {
		rawOpts := buildRawVerbOptions(args)
		opts = append(opts, rawOpts...)
	} else {
		// Handle aggregation verbs
		aggOpts, err := buildAggregationVerbOptions(schema, args, verb)
		if err != nil {
			return nil, fmt.Errorf("failed to build aggregation options: %w", err)
		}
		opts = append(opts, aggOpts...)
	}

	// Add group by if --by is set
	if cmdFlags.By != "" {
		groupFields := strings.Split(cmdFlags.By, ",")
		opts = append(opts, querybuilder.WithGroupBy(groupFields...))
	}

	// Add filter if --filter is set
	if cmdFlags.Filter != "" {
		// Parse the filter expression using the querybuilder's parser with schema support
		filterExpr, err := querybuilder.ParseFilterWithSchema(cmdFlags.Filter, schema)
		if err != nil {
			return nil, fmt.Errorf("invalid filter expression: %w", err)
		}
		opts = append(opts, querybuilder.WithFilter(filterExpr))
	}

	return opts, nil
}

// buildRawVerbOptions builds options for the raw verb.
func buildRawVerbOptions(args []string) []querybuilder.Option {
	var opts []querybuilder.Option
	opts = append(opts, querybuilder.WithVerb(querybuilder.VerbRaw))

	// Handle fields for raw verb
	if len(args) > 1 {
		fields := parseFields(args[1:])
		if len(fields) > 0 {
			opts = append(opts, querybuilder.WithFields(fields...))
		}
	}

	return opts
}

// buildAggregationVerbOptions builds options for aggregation verbs.
func buildAggregationVerbOptions(_ querybuilder.Schema, args []string, verb querybuilder.Verb) ([]querybuilder.Option, error) {
	opts := []querybuilder.Option{querybuilder.WithVerb(verb)}

	// If no additional arguments, just return the verb option
	if len(args) <= 1 {
		return opts, nil
	}

	// Parse fields from arguments
	fields := parseFields(args[1:])
	if len(fields) == 0 {
		return opts, nil
	}

	// For count verb with no fields, return just the verb option
	if verb == querybuilder.VerbCount && len(fields) == 0 {
		return opts, nil
	}

	// Create and add aggregations
	return addAggregationsToOptions(opts, fields, verb)
}

// addAggregationsToOptions creates aggregations for fields and adds them to options.
func addAggregationsToOptions(opts []querybuilder.Option, fields []string, verb querybuilder.Verb) ([]querybuilder.Option, error) {
	// Create aggregations for each field
	aggregations, err := createAggregationsForFields(fields, verb)
	if err != nil {
		return nil, err
	}

	// Add aggregations to options
	return append(opts, querybuilder.WithAggregations(aggregations...)), nil
}

// createAggregationsForFields creates aggregation fields for the given fields and verb.
func createAggregationsForFields(fields []string, verb querybuilder.Verb) ([]querybuilder.AggregationField, error) {
	aggregations := make([]querybuilder.AggregationField, 0, len(fields))

	for _, field := range fields {
		// For non-count verbs, validate that fields are numeric
		if err := validateFieldForVerb(field, verb); err != nil {
			return nil, fmt.Errorf("field validation failed: %w", err)
		}

		aggregations = append(aggregations, querybuilder.AggregationField{
			Verb:  verb,
			Field: field,
		})
	}

	return aggregations, nil
}

// validateFieldForVerb validates that a field is appropriate for the given verb.
func validateFieldForVerb(field string, verb querybuilder.Verb) error {
	// Count verb can use any field
	if verb == querybuilder.VerbCount || field == "*" {
		return nil
	}

	// For other verbs, validate that fields are numeric
	if !isNumericField(field) {
		verbStr := strings.ToLower(strings.TrimPrefix(verb.String(), "Verb"))
		return fmt.Errorf("field %q must be numeric for verb %q", field, verbStr)
	}

	return nil
}
