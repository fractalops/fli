// Package querybuilder provides tools for building CloudWatch Logs Insights queries.
package querybuilder

// Schema defines the interface for a specific data source's query dialect,
// providing the necessary validation and query components for the Builder.
type Schema interface {
	// GetParsePattern returns the 'parse' statement pattern for a given log version.
	GetParsePattern(version int) (string, error)
	// ValidateField checks if a field is valid for the given log version.
	ValidateField(field string, version int) error
	// ValidateVersion checks if a version number is supported by the schema.
	ValidateVersion(version int) error
	// GetDefaultVersion returns the default version for the schema.
	GetDefaultVersion() int
	// IsNumeric returns true if the field is of a numeric type.
	IsNumeric(field string) bool
	// GetComputedFieldExpression returns the CloudWatch Logs Insights expression for a computed field.
	// Returns empty string if the field is not a computed field.
	GetComputedFieldExpression(field string, version int) string
}
