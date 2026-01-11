// Package schema provides custom error types for the schema package.
package schema

import (
	"errors"
	"fmt"
)

// Error codes for schema operations.
const (
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeValidationFailed  = "validation_failed"
	ErrCodeInvalidMessage    = "invalid_message"
	ErrCodeInvalidDocument   = "invalid_document"
	ErrCodeInvalidRole        = "invalid_role"
	ErrCodeInvalidContent     = "invalid_content"
	ErrCodeSerializationError = "serialization_error"
	ErrCodeDeserializationError = "deserialization_error"
	ErrCodeContextRequired    = "context_required"
)

// SchemaError represents an error that occurred during schema operations.
type SchemaError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *SchemaError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("schema %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("schema %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("schema %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *SchemaError) Unwrap() error {
	return e.Err
}

// NewSchemaError creates a new SchemaError.
func NewSchemaError(op, code string, err error) *SchemaError {
	return &SchemaError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewSchemaErrorWithMessage creates a new SchemaError with a custom message.
func NewSchemaErrorWithMessage(op, code, message string, err error) *SchemaError {
	return &SchemaError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsSchemaError checks if an error is a SchemaError.
func IsSchemaError(err error) bool {
	var schemaErr *SchemaError
	return errors.As(err, &schemaErr)
}

// AsSchemaError attempts to convert an error to a SchemaError.
func AsSchemaError(err error) (*SchemaError, bool) {
	var schemaErr *SchemaError
	if errors.As(err, &schemaErr) {
		return schemaErr, true
	}
	return nil, false
}
