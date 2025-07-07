// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
	"strings"
)

// AggregationField represents a field with its aggregation verb.
type AggregationField struct {
	Field string
	Verb  Verb
}

// getAlias returns the alias for this aggregation field.
func (af AggregationField) getAlias() string {
	statFn := verbToStat[af.Verb]
	// Special case for count(*) - use "flows" alias
	if af.Field == "*" && af.Verb == VerbCount {
		return "flows"
	}
	return fmt.Sprintf("%s_%s", af.Field, statFn)
}

// Builder constructs CloudWatch Logs Insights queries.
type Builder struct {
	aggregations  []AggregationField
	fields        []string // For raw verb
	pendingFields []string // Fields set by WithFields but not yet used
	groupBy       []string
	limit         int
	filters       []Expr
	version       int
	schema        Schema
}

// New creates a new Builder with the given options.
func New(schema Schema, opts ...Option) (*Builder, error) {
	b := &Builder{
		schema:  schema,
		limit:   100,
		version: schema.GetDefaultVersion(),
		// Default to count aggregation
		aggregations: []AggregationField{{Field: "*", Verb: VerbCount}},
	}
	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// handleRawVerb sets up the builder for raw verb operations.
func (b *Builder) handleRawVerb() {
	// For raw verb, clear aggregations and set up for fields
	b.aggregations = nil
	// Use pending fields if available, otherwise use default
	if len(b.pendingFields) > 0 {
		b.fields = b.pendingFields
		b.pendingFields = nil
	} else if len(b.fields) == 0 {
		b.fields = []string{"*"}
	}
}

// handleAggregationVerb sets up the builder for aggregation verb operations.
func (b *Builder) handleAggregationVerb(v Verb) {
	// For aggregation verbs, create or update aggregation
	if len(b.aggregations) == 0 {
		// Create new aggregation
		b.aggregations = []AggregationField{{Field: "*", Verb: v}}
	} else {
		// Update existing aggregation verb
		b.aggregations[0].Verb = v
	}
}

var verbToStat = map[Verb]string{
	VerbCount: "count",
	VerbSum:   "sum",
	VerbAvg:   "avg",
	VerbMin:   "min",
	VerbMax:   "max",
}

// String returns the query string.
func (b Builder) String() string {
	// Build the query string from the components.
	var parts []string

	// Start with the 'parse' statement.
	parsePattern, err := b.schema.GetParsePattern(b.version)
	if err != nil {
		// This should not happen if validation is done in New().
		// Return an empty string or handle error appropriately.
		return ""
	}
	parts = append(parts, parsePattern)

	// Add filter expression.
	if len(b.filters) > 0 {
		parts = append(parts, "filter "+And(b.filters).String())
	}

	// Add 'stats' for aggregate functions or 'fields' for raw verb.
	if len(b.aggregations) > 0 {
		// This is an aggregation verb
		statsClause, sortClause := b.buildStatsAndSortClauses()
		parts = append(parts, statsClause)
		parts = append(parts, sortClause)
	} else if len(b.fields) > 0 && b.fields[0] != "*" {
		// This is a raw verb with specific fields (not "*")
		// Use display clause instead of fields clause to avoid conflicts
		displayClause := b.buildDisplayClause()
		if displayClause != "" {
			parts = append(parts, displayClause)
		}
	}

	// Add 'limit'.
	if b.limit > 0 {
		parts = append(parts, fmt.Sprintf("limit %d", b.limit))
	}

	return strings.Join(parts, " | ")
}

// buildStatsAndSortClauses constructs the 'stats' and 'sort' parts of the query.
// It returns two strings: the stats clause and the sort clause.
func (b *Builder) buildStatsAndSortClauses() (string, string) {
	if len(b.aggregations) == 0 {
		return "", ""
	}

	// Build stats clause for multiple aggregations
	var stats []string
	for _, agg := range b.aggregations {
		statFn := verbToStat[agg.Verb]
		alias := agg.getAlias()

		// Handle computed fields
		computedExpr := b.schema.GetComputedFieldExpression(agg.Field, b.version)
		if computedExpr != "" {
			// Use the computed field expression
			stats = append(stats, fmt.Sprintf("%s(%s) as %s", statFn, computedExpr, alias))
		} else {
			// Use the field name directly
			stats = append(stats, fmt.Sprintf("%s(%s) as %s", statFn, agg.Field, alias))
		}
	}

	statsClause := "stats " + strings.Join(stats, ", ")

	// Add grouping if specified
	if len(b.groupBy) > 0 {
		groupByExpressions := b.buildGroupByExpressions()
		var sb strings.Builder
		sb.WriteString(statsClause)
		sb.WriteString(" by ")
		sb.WriteString(groupByExpressions)
		statsClause = sb.String()
	}

	// Sort by first aggregation field (primary field sorting)
	primaryAlias := b.aggregations[0].getAlias()
	sortClause := "sort " + primaryAlias + " desc"

	return statsClause, sortClause
}

// buildGroupByExpressions constructs the group by expressions, handling computed fields.
func (b *Builder) buildGroupByExpressions() string {
	if len(b.groupBy) == 0 {
		return ""
	}

	var groupByExpressions []string
	for _, field := range b.groupBy {
		// Check if this is a computed field
		computedExpr := b.schema.GetComputedFieldExpression(field, b.version)
		if computedExpr != "" {
			// Use the computed field expression with an alias
			groupByExpressions = append(groupByExpressions, fmt.Sprintf("%s as %s", computedExpr, field))
		} else {
			// Use the field name directly
			groupByExpressions = append(groupByExpressions, field)
		}
	}

	return strings.Join(groupByExpressions, ", ")
}

// buildDisplayClause constructs the 'display' clause for the raw verb.
// It handles computed fields by using their expressions.
func (b *Builder) buildDisplayClause() string {
	if len(b.fields) == 0 {
		return ""
	}

	var fieldExpressions []string
	for _, field := range b.fields {
		// Check if this is a computed field
		computedExpr := b.schema.GetComputedFieldExpression(field, b.version)
		if computedExpr != "" {
			// Use the computed field expression with an alias
			fieldExpressions = append(fieldExpressions, fmt.Sprintf("%s as %s", computedExpr, field))
		} else {
			// Use the field name directly
			fieldExpressions = append(fieldExpressions, field)
		}
	}

	return "display " + strings.Join(fieldExpressions, ", ")
}
