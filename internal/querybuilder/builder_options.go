// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
)

// Option is a function that configures a Builder.
type Option func(*Builder) error

// WithVerb sets the query verb (legacy support).
func WithVerb(v Verb) Option {
	return func(b *Builder) error {
		if v == VerbRaw {
			b.handleRawVerb()
		} else {
			b.handleAggregationVerb(v)
		}
		return nil
	}
}

// WithFields sets the fields to select (legacy support).
func WithFields(fields ...string) Option {
	return func(b *Builder) error {
		// Always validate fields first
		for _, field := range fields {
			if err := b.schema.ValidateField(field, b.version); err != nil {
				return fmt.Errorf("invalid field '%s': %w", field, err)
			}
		}

		// Check if we have a raw verb (no aggregations)
		if len(b.aggregations) == 0 {
			// This is a raw verb, set the fields
			b.fields = fields
		} else {
			// For other verbs, store fields for potential raw verb use
			b.pendingFields = fields
			// Update the aggregation field (take first field)
			if len(fields) > 0 {
				b.aggregations[0].Field = fields[0]
			}
		}
		return nil
	}
}

// WithAggregations sets multiple aggregation fields.
func WithAggregations(aggregations ...AggregationField) Option {
	return func(b *Builder) error {
		for _, agg := range aggregations {
			if err := b.schema.ValidateField(agg.Field, b.version); err != nil {
				return fmt.Errorf("invalid field '%s': %w", agg.Field, err)
			}
			if agg.Verb != VerbCount && !b.schema.IsNumeric(agg.Field) {
				return fmt.Errorf("field '%s' must be numeric for verb '%s'", agg.Field, agg.Verb)
			}
		}
		b.aggregations = aggregations
		return nil
	}
}

// WithGroupBy sets the group by fields.
func WithGroupBy(fields ...string) Option {
	return func(b *Builder) error {
		for _, field := range fields {
			if err := b.schema.ValidateField(field, b.version); err != nil {
				return fmt.Errorf("invalid group by field '%s': %w", field, err)
			}
		}
		b.groupBy = fields
		return nil
	}
}

// WithLimit sets the result limit.
func WithLimit(n int) Option {
	return func(b *Builder) error {
		if n < 0 {
			return fmt.Errorf("limit must be non-negative")
		}
		b.limit = n
		return nil
	}
}

// WithFilter adds a filter expression.
func WithFilter(e Expr) Option {
	return func(b *Builder) error {
		if err := ValidateFilter(e, b.schema, b.version); err != nil {
			return err
		}
		b.filters = append(b.filters, e)
		return nil
	}
}

// WithVersion sets the flow log version.
func WithVersion(v int) Option {
	return func(b *Builder) error {
		if err := b.schema.ValidateVersion(v); err != nil {
			return fmt.Errorf("invalid version %d: %w", v, err)
		}
		b.version = v
		return nil
	}
}
