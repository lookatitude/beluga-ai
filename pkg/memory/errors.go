// Package memory provides custom error types for package-specific errors.
// It follows the framework's error handling patterns with custom error types
// that preserve error chains and provide context about failed operations.
package memory

import (
	"errors"
	"fmt"
)

// MemoryError represents a memory-specific error with additional context.
type MemoryError struct {
	// Op is the operation that failed
	Op string

	// Err is the underlying error
	Err error

	// Code is a machine-readable error code
	Code string

	// MemoryType indicates which memory implementation was involved
	MemoryType MemoryType

	// Context provides additional context about the error
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *MemoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("memory %s (%s): %v", e.Op, e.MemoryType, e.Err)
	}
	return fmt.Sprintf("memory %s (%s)", e.Op, e.MemoryType)
}

// Unwrap returns the underlying error.
func (e *MemoryError) Unwrap() error {
	return e.Err
}

// Is implements error comparison for specific error codes.
func (e *MemoryError) Is(target error) bool {
	var memErr *MemoryError
	if errors.As(target, &memErr) {
		return e.Code == memErr.Code
	}
	return false
}

// Error codes for common memory operations.
const (
	ErrCodeInvalidConfig   = "invalid_config"
	ErrCodeInvalidInput    = "invalid_input"
	ErrCodeStorageError    = "storage_error"
	ErrCodeRetrievalError  = "retrieval_error"
	ErrCodeTimeout         = "timeout"
	ErrCodeNotFound        = "not_found"
	ErrCodeTypeMismatch    = "type_mismatch"
	ErrCodeSerialization   = "serialization_error"
	ErrCodeDeserialization = "deserialization_error"
	ErrCodeValidation      = "validation_error"
)

// NewMemoryError creates a new MemoryError with the given parameters.
func NewMemoryError(op string, memoryType MemoryType, code string, err error) *MemoryError {
	return &MemoryError{
		Op:         op,
		Err:        err,
		Code:       code,
		MemoryType: memoryType,
		Context:    make(map[string]interface{}),
	}
}

// WithContext adds context information to the error.
func (e *MemoryError) WithContext(key string, value interface{}) *MemoryError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WrapError wraps an existing error with memory-specific context.
func WrapError(err error, op string, memoryType MemoryType, code string) *MemoryError {
	if err == nil {
		return nil
	}
	return NewMemoryError(op, memoryType, code, err)
}

// IsMemoryError checks if an error is a MemoryError with the given code.
func IsMemoryError(err error, code string) bool {
	var memErr *MemoryError
	if errors.As(err, &memErr) {
		return memErr.Code == code
	}
	return false
}

// Common error constructors for frequent error patterns.

// ErrInvalidConfig returns an error for invalid configuration.
func ErrInvalidConfig(memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError("configure", memoryType, ErrCodeInvalidConfig, err)
}

// ErrInvalidInput returns an error for invalid input parameters.
func ErrInvalidInput(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeInvalidInput, err)
}

// ErrStorageError returns an error for storage operation failures.
func ErrStorageError(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeStorageError, err)
}

// ErrRetrievalError returns an error for retrieval operation failures.
func ErrRetrievalError(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeRetrievalError, err)
}

// ErrTimeout returns an error for timeout conditions.
func ErrTimeout(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeTimeout, err)
}

// ErrNotFound returns an error for not found conditions.
func ErrNotFound(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeNotFound, err)
}

// ErrTypeMismatch returns an error for type mismatch conditions.
func ErrTypeMismatch(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeTypeMismatch, err)
}

// ErrSerialization returns an error for serialization failures.
func ErrSerialization(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeSerialization, err)
}

// ErrDeserialization returns an error for deserialization failures.
func ErrDeserialization(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeDeserialization, err)
}

// ErrValidation returns an error for validation failures.
func ErrValidation(op string, memoryType MemoryType, err error) *MemoryError {
	return NewMemoryError(op, memoryType, ErrCodeValidation, err)
}
