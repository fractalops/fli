// Package querybuilder provides functionality to construct CloudWatch Logs Insights queries.
package querybuilder

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

const (
	operatorLike    = "like"
	operatorNotLike = "not like"
)

// InvalidTokenError is returned when a token cannot be parsed.
type InvalidTokenError struct {
	Token  string
	Reason string
}

func (e InvalidTokenError) Error() string {
	return fmt.Sprintf("invalid token %q: %s", e.Token, e.Reason)
}

// Pre-compiled regex patterns for token validation.
var (
	ipPrefixPattern = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){0,3}$`)
)

// Constants for field validation.
const (
	MinPort              = 0
	MaxPort              = 65535
	MaxIPOctet           = 255
	MaxIPParts           = 4
	DefaultSchemaVersion = 2
)

// Error messages for better consistency.
const (
	ErrInvalidFilterClause = "invalid filter clause: %q"
	ErrUnsupportedOperator = "unsupported operator for %s field: %q"
	ErrInvalidPortValue    = "invalid port value: %s"
	ErrPortOutOfRange      = "port out of range: %d"
	ErrInvalidNumericValue = "invalid numeric value for field %s: %s"
	ErrInvalidIPValue      = "invalid IP, CIDR, or prefix value for field %s: %s"
	ErrInvalidCIDRBlock    = "invalid CIDR block: %v"
)

// FieldType represents the type of a field and its supported operators.
type FieldType struct {
	Name           string
	SupportedOps   []string
	ValueValidator func(string) error
	Parser         func(string, string, string) (Expr, error)
}

// FieldRegistry holds the configuration for different field types.
type FieldRegistry struct {
	fields map[string]FieldType
}

// NewFieldRegistry creates a new field registry with default field types.
func NewFieldRegistry() *FieldRegistry {
	registry := &FieldRegistry{
		fields: make(map[string]FieldType),
	}

	// Register default field types
	registry.registerDefaultFields()

	return registry
}

// registerDefaultFields registers the built-in field types.
func (r *FieldRegistry) registerDefaultFields() {
	// IP fields
	ipFields := []string{"srcaddr", "dstaddr", "pkt_srcaddr", "pkt_dstaddr"}
	for _, field := range ipFields {
		r.fields[field] = FieldType{
			Name:         "ip",
			SupportedOps: []string{"=", "!=", "like", "not like"},
			Parser:       parseIPFieldExpr,
		}
	}

	// Port fields
	portFields := []string{"srcport", "dstport"}
	for _, field := range portFields {
		r.fields[field] = FieldType{
			Name:         "port",
			SupportedOps: []string{"=", "!=", ">", "<", ">=", "<="},
			ValueValidator: func(value string) error {
				port, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf(ErrInvalidPortValue, value)
				}
				if port < MinPort || port > MaxPort {
					return fmt.Errorf(ErrPortOutOfRange, port)
				}
				return nil
			},
			Parser: parsePortFieldExpr,
		}
	}

	// Protocol field (supports both numeric and string values)
	r.fields["protocol"] = FieldType{
		Name:         "protocol",
		SupportedOps: []string{"=", "!=", ">", "<", ">=", "<="},
		Parser:       parseProtocolFieldExpr,
	}

	// Numeric fields
	numericFields := []string{"packets", "bytes", "start", "end", "duration"}
	for _, field := range numericFields {
		r.fields[field] = FieldType{
			Name:         "numeric",
			SupportedOps: []string{"=", "!=", ">", "<", ">=", "<="},
			Parser:       parseNumericFieldExpr,
		}
	}
}

// GetFieldType returns the field type for a given field name.
func (r *FieldRegistry) GetFieldType(field string) (FieldType, bool) {
	fieldType, exists := r.fields[field]
	return fieldType, exists
}

// RegisterField registers a new field type.
func (r *FieldRegistry) RegisterField(field string, fieldType FieldType) {
	r.fields[field] = fieldType
}

// Global field registry instance.
var defaultFieldRegistry = NewFieldRegistry()

// OperatorParser provides unified operator handling for different field types.
type OperatorParser struct {
	field string
	value string
}

// NewOperatorParser creates a new operator parser for a field and value.
func NewOperatorParser(field, value string) *OperatorParser {
	return &OperatorParser{
		field: field,
		value: value,
	}
}

// ParseOperator handles the common operator parsing logic with automatic value conversion.
func (op *OperatorParser) ParseOperator(operator string) (Expr, error) {
	// Try to convert value to appropriate type
	convertedValue := op.convertValue()

	switch operator {
	case "=":
		return &Eq{Field: op.field, Value: convertedValue}, nil
	case "!=":
		return &Neq{Field: op.field, Value: convertedValue}, nil
	case ">":
		return &Gt{Field: op.field, Value: convertedValue}, nil
	case "<":
		return &Lt{Field: op.field, Value: convertedValue}, nil
	case ">=":
		return &Gte{Field: op.field, Value: convertedValue}, nil
	case "<=":
		return &Lte{Field: op.field, Value: convertedValue}, nil
	case operatorLike:
		return &Like{Field: op.field, Value: op.value}, nil
	case operatorNotLike:
		return &NotLike{Field: op.field, Value: op.value}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %q", operator)
	}
}

// convertValue attempts to convert the string value to the most appropriate type.
func (op *OperatorParser) convertValue() any {
	// Try to parse as integer first
	if num, err := strconv.Atoi(op.value); err == nil {
		return num
	}

	// Try to parse as float
	if num, err := strconv.ParseFloat(op.value, 64); err == nil {
		return num
	}

	// Return as string if not numeric
	return op.value
}

// splitOnLogical splits s on the given logical operator (case-insensitive, with spaces around)
// respecting parentheses.
func splitOnLogical(s, op string) []string {
	var parts []string
	parenLevel := 0
	lastSplit := 0
	lowerS := strings.ToLower(s)
	lowerOp := " " + op + " "

	for i := range s {
		// Ensure we don't look past the end of the string
		if i+len(lowerOp) > len(s) {
			break
		}

		switch s[i] {
		case '(':
			parenLevel++
		case ')':
			parenLevel--
		}

		// Found the operator at a point where we are not inside parentheses
		if parenLevel == 0 && lowerS[i:i+len(lowerOp)] == lowerOp {
			parts = append(parts, strings.TrimSpace(s[lastSplit:i]))
			lastSplit = i + len(lowerOp)
		}
	}
	// Add the final part of the string
	parts = append(parts, strings.TrimSpace(s[lastSplit:]))
	return parts
}

// isValidIPPrefix checks if the string is a valid IP prefix.
func isValidIPPrefix(prefix string) bool {
	parts := strings.Split(prefix, ".")
	if len(parts) > MaxIPParts {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > MaxIPOctet {
			return false
		}
	}

	return true
}

// parseIPFieldExpr returns the correct Expr for an IP field, operator, and value.
func parseIPFieldExpr(field, op, value string) (Expr, error) {
	// First, check for valid IP operators
	switch op {
	case "=", "!=", operatorLike, operatorNotLike:
	// Continue
	default:
		return nil, fmt.Errorf(ErrUnsupportedOperator, "IP", op)
	}

	if strings.Contains(value, "/") { // CIDR
		if _, err := netip.ParsePrefix(value); err != nil {
			return nil, fmt.Errorf(ErrInvalidCIDRBlock, err)
		}
		switch op {
		case "=", operatorLike:
			return &IsIpv4InSubnet{Field: field, Value: value}, nil
		case "!=", operatorNotLike:
			return &NotExpr{Expr: &IsIpv4InSubnet{Field: field, Value: value}}, nil
		}
	} else if _, err := netip.ParseAddr(value); err == nil { // Full IP
		switch op {
		case "=":
			return &Eq{Field: field, Value: value}, nil
		case "!=":
			return &Neq{Field: field, Value: value}, nil
		case operatorLike: // 'like' on a full IP is just equality
			return &Eq{Field: field, Value: value}, nil
		case operatorNotLike: // 'not like' on a full IP is just non-equality
			return &Neq{Field: field, Value: value}, nil
		}
	} else if ipPrefixPattern.MatchString(value) && isValidIPPrefix(value) { // Prefix
		switch op {
		case "=", operatorLike:
			return &Like{Field: field, Value: value}, nil
		case "!=", operatorNotLike:
			return &NotLike{Field: field, Value: value}, nil
		}
	}
	return nil, fmt.Errorf(ErrInvalidIPValue, field, value)
}

func parsePortFieldExpr(field, op, value string) (Expr, error) {
	// Validate port value
	port, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf(ErrInvalidPortValue, value)
	}
	if port < MinPort || port > MaxPort {
		return nil, fmt.Errorf(ErrPortOutOfRange, port)
	}

	// Use unified operator parser
	parser := NewOperatorParser(field, value)

	// For equality operators, use the numeric value
	if op == "=" || op == "!=" || op == ">=" || op == "<=" {
		return parser.ParseOperator(op)
	}

	// For comparison operators, use the string value
	return parser.ParseOperator(op)
}

// parseNumericFieldExpr returns the correct Expr for a numeric field, operator, and value.
func parseNumericFieldExpr(field, op, value string) (Expr, error) {
	parser := NewOperatorParser(field, value)

	// Try to parse as integer first
	if _, err := strconv.Atoi(value); err == nil {
		// For equality operators, use the numeric value
		if op == "=" || op == "!=" || op == ">=" || op == "<=" {
			return parser.ParseOperator(op)
		}
		// For comparison operators, use the string value
		return parser.ParseOperator(op)
	}

	// Try to parse as float
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		// For equality operators, use the numeric value
		if op == "=" || op == "!=" || op == ">=" || op == "<=" {
			return parser.ParseOperator(op)
		}
		// For comparison operators, use the string value
		return parser.ParseOperator(op)
	}

	return nil, fmt.Errorf(ErrInvalidNumericValue, field, value)
}

// parseProtocolFieldExpr returns the correct Expr for a protocol field, operator, and value.
func parseProtocolFieldExpr(field, op, value string) (Expr, error) {
	// Protocol can be numeric (6, 17) or string (TCP, UDP)

	// Try to parse as integer first
	if num, err := strconv.Atoi(value); err == nil {
		// Use numeric value directly
		parser := NewOperatorParser(field, strconv.Itoa(num))
		return parser.ParseOperator(op)
	}

	// If not numeric, check if it's a known protocol acronym
	protocolMap := map[string]string{
		"tcp":    "6",
		"udp":    "17",
		"icmp":   "1",
		"icmpv6": "58",
		"esp":    "50",
		"ah":     "51",
	}

	if protocolNum, exists := protocolMap[strings.ToLower(value)]; exists {
		// Convert protocol acronym to numeric value and create parser with numeric value
		parser := NewOperatorParser(field, protocolNum)
		return parser.ParseOperator(op)
	}

	// If not a known acronym, treat as string (for custom protocols)
	parser := NewOperatorParser(field, value)
	return parser.ParseOperator(op)
}
