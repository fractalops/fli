package cache

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of cache error.
type ErrorType int

const (
	// ErrorTypeNotFound indicates a resource was not found.
	ErrorTypeNotFound ErrorType = iota
	// ErrorTypeInvalidData indicates invalid data was provided.
	ErrorTypeInvalidData
	// ErrorTypeDatabase indicates a database operation failed.
	ErrorTypeDatabase
	// ErrorTypeNetwork indicates a network operation failed.
	ErrorTypeNetwork
	// ErrorTypeConfiguration indicates a configuration error.
	ErrorTypeConfiguration
	// ErrorTypeWhois indicates a whois lookup failed.
	ErrorTypeWhois
	// ErrorTypeValidation indicates a validation error.
	ErrorTypeValidation
)

// Error represents a cache-specific error.
type Error struct {
	Type    ErrorType
	Op      string
	Key     string
	Message string
	Err     error
}

func (e *Error) Error() string {
	var parts []string

	// Add operation context
	if e.Op != "" {
		parts = append(parts, fmt.Sprintf("operation: %s", e.Op))
	}

	// Add key context
	if e.Key != "" {
		parts = append(parts, fmt.Sprintf("key: %s", e.Key))
	}

	// Add message
	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	// Add underlying error
	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Err))
	}

	return fmt.Sprintf("cache error [%s]: %s", e.typeString(), strings.Join(parts, ", "))
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) typeString() string {
	switch e.Type {
	case ErrorTypeNotFound:
		return "not_found"
	case ErrorTypeInvalidData:
		return "invalid_data"
	case ErrorTypeDatabase:
		return "database"
	case ErrorTypeNetwork:
		return "network"
	case ErrorTypeConfiguration:
		return "configuration"
	case ErrorTypeWhois:
		return "whois"
	case ErrorTypeValidation:
		return "validation"
	default:
		return "unknown"
	}
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(op, key string) *Error {
	return &Error{
		Type:    ErrorTypeNotFound,
		Op:      op,
		Key:     key,
		Message: "resource not found",
	}
}

// NewInvalidDataError creates a new invalid data error.
func NewInvalidDataError(op, key, message string, err error) *Error {
	return &Error{
		Type:    ErrorTypeInvalidData,
		Op:      op,
		Key:     key,
		Message: message,
		Err:     err,
	}
}

// NewDatabaseError creates a new database error.
func NewDatabaseError(op, key string, err error) *Error {
	return &Error{
		Type:    ErrorTypeDatabase,
		Op:      op,
		Key:     key,
		Message: "database operation failed",
		Err:     err,
	}
}

// NewNetworkError creates a new network error.
func NewNetworkError(op, url string, err error) *Error {
	return &Error{
		Type:    ErrorTypeNetwork,
		Op:      op,
		Key:     url,
		Message: "network request failed",
		Err:     err,
	}
}

// NewConfigurationError creates a new configuration error.
func NewConfigurationError(message string, err error) *Error {
	return &Error{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Err:     err,
	}
}

// NewWhoisError creates a new whois error.
func NewWhoisError(ip string, err error) *Error {
	return &Error{
		Type:    ErrorTypeWhois,
		Op:      "whois_lookup",
		Key:     ip,
		Message: "whois lookup failed",
		Err:     err,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(op, key, message string) *Error {
	return &Error{
		Type:    ErrorTypeValidation,
		Op:      op,
		Key:     key,
		Message: message,
	}
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if cacheErr, ok := err.(*Error); ok {
		return cacheErr.Type == ErrorTypeNotFound
	}
	return false
}

// IsNetworkError checks if an error is a network error.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if cacheErr, ok := err.(*Error); ok {
		return cacheErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsDatabaseError checks if an error is a database error.
func IsDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	if cacheErr, ok := err.(*Error); ok {
		return cacheErr.Type == ErrorTypeDatabase
	}
	return false
}
