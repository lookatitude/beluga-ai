// Package memory provides custom error types for package-specific errors.
// It follows the framework's error handling patterns with custom error types
// that preserve error chains and provide context about failed operations.
package memory

import (
	"errors"
	"fmt"
)

// MemoryError represents a memory-specific error with additional context.
// It follows the standard Op/Err/Code pattern used across all Beluga AI packages.
type MemoryError struct {
	Op      string         // operation that failed
	Err     error          // underlying error
	Code    string         // error code for programmatic handling
	Message string         // human-readable message (optional)
	Context map[string]any // additional context (optional)
}

// Error implements the error interface.
func (e *MemoryError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("memory %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("memory %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("memory %s: unknown error (code: %s)", e.Op, e.Code)
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
	ErrCodeMemoryOverflow  = "memory_overflow"
	ErrCodeContextCanceled = "context_canceled"
)

// NewMemoryError creates a new MemoryError following the Op/Err/Code pattern.
func NewMemoryError(op, code string, err error) *MemoryError {
	return &MemoryError{
		Op:      op,
		Code:    code,
		Err:     err,
		Context: make(map[string]any),
	}
}

// NewMemoryErrorWithMessage creates a new MemoryError with a custom message.
func NewMemoryErrorWithMessage(op, code, message string, err error) *MemoryError {
	return &MemoryError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]any),
	}
}

// WithContext adds context information to the error.
func (e *MemoryError) WithContext(key string, value any) *MemoryError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// WrapError wraps an existing error with memory-specific context.
func WrapError(err error, op, code string) *MemoryError {
	if err == nil {
		return nil
	}
	return NewMemoryError(op, code, err)
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
func ErrInvalidConfig(err error) *MemoryError {
	return NewMemoryErrorWithMessage("configure", ErrCodeInvalidConfig, "invalid configuration", err)
}

// ErrInvalidInput returns an error for invalid input parameters.
func ErrInvalidInput(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeInvalidInput, err)
}

// ErrStorageError returns an error for storage operation failures.
func ErrStorageError(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeStorageError, err)
}

// ErrRetrievalError returns an error for retrieval operation failures.
func ErrRetrievalError(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeRetrievalError, err)
}

// ErrTimeout returns an error for timeout conditions.
func ErrTimeout(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeTimeout, err)
}

// ErrNotFound returns an error for not found conditions.
func ErrNotFound(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeNotFound, err)
}

// ErrTypeMismatch returns an error for type mismatch conditions.
func ErrTypeMismatch(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeTypeMismatch, err)
}

// ErrSerialization returns an error for serialization failures.
func ErrSerialization(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeSerialization, err)
}

// ErrDeserialization returns an error for deserialization failures.
func ErrDeserialization(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeDeserialization, err)
}

// ErrValidation returns an error for validation failures.
func ErrValidation(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeValidation, err)
}

// ErrMemoryOverflow returns an error for memory overflow conditions.
func ErrMemoryOverflow(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeMemoryOverflow, err)
}

// ErrContextCanceled returns an error for context cancellation.
func ErrContextCanceled(op string, err error) *MemoryError {
	return NewMemoryError(op, ErrCodeContextCanceled, err)
}
