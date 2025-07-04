// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
	"strings"
)

// Expr represents a query expression.
type Expr interface {
	String() string
}

// FieldValueExpr is implemented by expressions with a Field and Value
// Used for generic validation and processing.
type FieldValueExpr interface {
	Expr
	GetField() string
	GetValue() any
}

// Eq represents an equality comparison.
type Eq struct {
	Field string
	Value any
}

func (e Eq) String() string {
	return fmt.Sprintf("%s = %s", formatField(e.Field), quote(e.Value))
}

// GetField returns the field name for the equality expression.
func (e Eq) GetField() string { return e.Field }

// GetValue returns the value for the equality expression.
func (e Eq) GetValue() any { return e.Value }

// Neq represents a non-equality comparison.
type Neq struct {
	Field string
	Value any
}

func (e Neq) String() string {
	return fmt.Sprintf("%s != %s", formatField(e.Field), quote(e.Value))
}

// GetField returns the field name for the non-equality expression.
func (e Neq) GetField() string { return e.Field }

// GetValue returns the value for the non-equality expression.
func (e Neq) GetValue() any { return e.Value }

// Gt represents a "greater than" comparison.
type Gt struct {
	Field string
	Value any
}

func (g Gt) String() string {
	return fmt.Sprintf("%s > %s", formatField(g.Field), quote(g.Value))
}

// GetField returns the field name for the greater than expression.
func (g Gt) GetField() string { return g.Field }

// GetValue returns the value for the greater than expression.
func (g Gt) GetValue() any { return g.Value }

// Lt represents a "less than" comparison.
type Lt struct {
	Field string
	Value any
}

func (l Lt) String() string {
	return fmt.Sprintf("%s < %s", formatField(l.Field), quote(l.Value))
}

// GetField returns the field name for the less than expression.
func (l Lt) GetField() string { return l.Field }

// GetValue returns the value for the less than expression.
func (l Lt) GetValue() any { return l.Value }

// Like represents a pattern match.
type Like struct {
	Field string
	Value string
}

func (e Like) String() string {
	return fmt.Sprintf("%s like %s", e.Field, quote(e.Value))
}

// GetField returns the field name for the like expression.
func (e Like) GetField() string { return e.Field }

// GetValue returns the value for the like expression.
func (e Like) GetValue() any { return e.Value }

// NotLike represents a negative pattern match.
type NotLike struct {
	Field string
	Value string
}

func (e NotLike) String() string {
	return fmt.Sprintf("%s not like %s", e.Field, quote(e.Value))
}

// GetField returns the field name for the not like expression.
func (e NotLike) GetField() string { return e.Field }

// GetValue returns the value for the not like expression.
func (e NotLike) GetValue() any { return e.Value }

// Gte represents a "greater than or equal to" comparison.
type Gte struct {
	Field string
	Value any
}

func (e Gte) String() string {
	return fmt.Sprintf("%s >= %s", formatField(e.Field), quote(e.Value))
}

// GetField returns the field name for the greater than or equal expression.
func (e Gte) GetField() string { return e.Field }

// GetValue returns the value for the greater than or equal expression.
func (e Gte) GetValue() any { return e.Value }

// Lte represents a "less than or equal to" comparison.
type Lte struct {
	Field string
	Value any
}

func (e Lte) String() string {
	return fmt.Sprintf("%s <= %s", formatField(e.Field), quote(e.Value))
}

// GetField returns the field name for the less than or equal expression.
func (e Lte) GetField() string { return e.Field }

// GetValue returns the value for the less than or equal expression.
func (e Lte) GetValue() any { return e.Value }

// And represents a conjunction of expressions.
type And []Expr

func (e And) String() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, len(e))
	for i, expr := range e {
		parts[i] = expr.String()
	}
	return strings.Join(parts, " and ")
}

// Or represents a disjunction of expressions.
type Or []Expr

func (e Or) String() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return "(" + e[0].String() + ")"
	}
	parts := make([]string, len(e))
	for i, expr := range e {
		parts[i] = expr.String()
	}
	return "(" + strings.Join(parts, " or ") + ")"
}

// NotExpr represents a logical NOT operation on an expression.
type NotExpr struct {
	Expr
}

func (e NotExpr) String() string {
	return fmt.Sprintf("not %s", e.Expr.String())
}

// quote returns a properly quoted value for CloudWatch Logs Insights.
func quote(v any) string {
	switch x := v.(type) {
	case string:
		// Escape single quotes and wrap in single quotes
		return "'" + strings.ReplaceAll(x, "'", "\\'") + "'"
	case int, int64, float64:
		// Numbers don't need quotes
		return fmt.Sprint(x)
	default:
		// Default to string representation with quotes
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "\\'") + "'"
	}
}

// isComputedField returns true if the field contains operators or spaces that indicate it's a computed expression.
func isComputedField(field string) bool {
	// Check for common operators and spaces that indicate a computed expression
	return strings.Contains(field, " ") || strings.Contains(field, "-") ||
		strings.Contains(field, "/") || strings.Contains(field, "*") ||
		strings.Contains(field, "+") || strings.Contains(field, "(") ||
		strings.Contains(field, ")")
}

// formatField returns the field with parentheses if it's a computed expression.
func formatField(field string) string {
	if isComputedField(field) {
		return "(" + field + ")"
	}
	return field
}

// IsIpv4InSubnet represents a CIDR block membership check.
// It generates a CloudWatch Logs Insights expression that checks if an IP address
// is within a CIDR block using the isIpv4InSubnet function.
// Example: IsIpv4InSubnet{Field: "srcaddr", Value: "10.0.0.0/24"} generates:
// isIpv4InSubnet(srcaddr, '10.0.0.0/24').
type IsIpv4InSubnet struct {
	Field string
	Value string
}

func (e IsIpv4InSubnet) String() string {
	return fmt.Sprintf("isIpv4InSubnet(%s, '%s')", e.Field, e.Value)
}

// GetField returns the field name for the IPv4 subnet check expression.
func (e IsIpv4InSubnet) GetField() string { return e.Field }

// GetValue returns the value for the IPv4 subnet check expression.
func (e IsIpv4InSubnet) GetValue() any { return e.Value }
