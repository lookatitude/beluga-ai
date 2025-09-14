package iface

import "fmt"

// SchemaError represents errors specific to schema operations.
// It provides structured error information for programmatic error handling.
type SchemaError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *SchemaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// NewSchemaError creates a new SchemaError with the given code and message.
func NewSchemaError(code, message string, args ...interface{}) *SchemaError {
	return &SchemaError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with schema context.
func WrapError(cause error, code, message string, args ...interface{}) *SchemaError {
	return &SchemaError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig         = "invalid_config"
	ErrCodeValidationFailed      = "validation_failed"
	ErrCodeInvalidMessage        = "invalid_message"
	ErrCodeInvalidDocument       = "invalid_document"
	ErrCodeSerializationFailed   = "serialization_failed"
	ErrCodeDeserializationFailed = "deserialization_failed"
	ErrCodeTypeConversionFailed  = "type_conversion_failed"
	ErrCodeSchemaMismatch        = "schema_mismatch"
	ErrCodeInvalidParameters     = "invalid_parameters"
)

// IsSchemaError checks if an error is a SchemaError with the given code.
func IsSchemaError(err error, code string) bool {
	var schErr *SchemaError
	if !AsSchemaError(err, &schErr) {
		return false
	}
	return schErr.Code == code
}

// AsSchemaError attempts to cast an error to SchemaError.
func AsSchemaError(err error, target **SchemaError) bool {
	for err != nil {
		if schErr, ok := err.(*SchemaError); ok {
			*target = schErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
