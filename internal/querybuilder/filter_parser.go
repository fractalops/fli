// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
	"strings"
)

// ParseFilter parses a filter string into an expression tree.
func ParseFilter(s string) (Expr, error) {
	return ParseFilterWithSchema(s, nil)
}

// ParseFilterWithSchema parses a filter string into an expression tree with schema support for computed fields.
func ParseFilterWithSchema(s string, schema Schema) (Expr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	return parseOrWithSchema(s, schema)
}

func parseOrWithSchema(s string, schema Schema) (Expr, error) {
	parts := splitOnLogical(s, "or")
	if len(parts) == 1 {
		return parseAndWithSchema(s, schema)
	}
	exprs := make([]Expr, len(parts))
	for i, p := range parts {
		expr, err := parseAndWithSchema(p, schema)
		if err != nil {
			return nil, err
		}
		exprs[i] = expr
	}
	orExpr := Or(exprs)
	return &orExpr, nil
}

func parseAndWithSchema(s string, schema Schema) (Expr, error) {
	parts := splitOnLogical(s, "and")
	if len(parts) == 1 {
		return parsePrimaryWithSchema(s, schema)
	}
	exprs := make([]Expr, len(parts))
	for i, p := range parts {
		expr, err := parsePrimaryWithSchema(p, schema)
		if err != nil {
			return nil, err
		}
		exprs[i] = expr
	}
	andExpr := And(exprs)
	return &andExpr, nil
}

func parsePrimaryWithSchema(s string, schema Schema) (Expr, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return ParseFilterWithSchema(s[1:len(s)-1], schema)
	}
	return parseClauseWithSchema(s, schema)
}

// parseClause parses a single filter clause like "field op value".
func parseClauseWithSchema(clause string, schema Schema) (Expr, error) {
	operators := []string{"!=", operatorNotLike, ">=", "<=", ">", "<", "=", operatorLike}
	var op, field, value string

	for _, candidate := range operators {
		// Use case-insensitive search for the operator, ensuring it's surrounded by spaces
		// to avoid matching substrings in field names or values.
		if idx := strings.Index(strings.ToLower(clause), " "+candidate+" "); idx != -1 {
			op = strings.ToLower(candidate)
			field = strings.TrimSpace(clause[:idx])
			value = strings.TrimSpace(clause[idx+len(candidate)+2:])
			break
		}
	}
	// Fallback for operators without spaces (e.g. `srcaddr='1.2.3.4'`)
	if op == "" {
		for _, candidate := range operators {
			if idx := strings.Index(strings.ToLower(clause), candidate); idx != -1 {
				op = strings.ToLower(candidate)
				field = strings.TrimSpace(clause[:idx])
				value = strings.TrimSpace(clause[idx+len(candidate):])
				break
			}
		}
	}

	if op == "" {
		return nil, fmt.Errorf(ErrInvalidFilterClause, clause)
	}
	value = strings.Trim(value, "'\"") // Remove quotes

	if schema != nil {
		if computedExpr := schema.GetComputedFieldExpression(field, DefaultSchemaVersion); computedExpr != "" {
			parser := NewOperatorParser(computedExpr, value)
			return parser.ParseOperator(op)
		}
	}

	if fieldType, exists := defaultFieldRegistry.GetFieldType(field); exists {
		// Only call ValueValidator if it's set
		if fieldType.ValueValidator != nil {
			if err := fieldType.ValueValidator(value); err != nil {
				return nil, err
			}
		}
		return fieldType.Parser(field, op, value)
	}

	// For non-numeric fields, only allow equality and pattern matching operators
	switch op {
	case "=":
		return &Eq{Field: field, Value: value}, nil
	case "!=":
		return &Neq{Field: field, Value: value}, nil
	case operatorLike:
		return &Like{Field: field, Value: value}, nil
	case operatorNotLike:
		return &NotLike{Field: field, Value: value}, nil
	default:
		return nil, fmt.Errorf("unsupported operator for non-numeric field: %q", op)
	}
}

// ValidateFilter recursively checks an Expr for valid fields, operators, and values for the given version.
func ValidateFilter(expr Expr, schema Schema, version int) error {
	if expr == nil {
		return nil
	}

	var validate func(e Expr) error
	validate = func(e Expr) error {
		switch x := e.(type) {
		case *And:
			for _, sub := range *x {
				if err := validate(sub); err != nil {
					return err
				}
			}
			return nil
		case *Or:
			for _, sub := range *x {
				if err := validate(sub); err != nil {
					return err
				}
			}
			return nil
		case *NotExpr:
			return validate(x.Expr)
		case FieldValueExpr:
			// The parser already validated the value (e.g., that a CIDR is valid).
			// We only need to check if the field name itself is valid for the version.
			field := x.GetField()
			return schema.ValidateField(field, version)
		default:
			return fmt.Errorf("unsupported expression type for validation: %T", e)
		}
	}

	return validate(expr)
}
