// Package multimodal provides custom error types for multimodal operations.
package multimodal

import (
	"errors"
	"fmt"
)

// Error codes for multimodal operations.
const (
	ErrCodeProviderNotFound   = "provider_not_found"
	ErrCodeInvalidConfig      = "invalid_config"
	ErrCodeInvalidInput       = "invalid_input"
	ErrCodeInvalidFormat      = "invalid_format"
	ErrCodeProviderError      = "provider_error"
	ErrCodeUnsupportedModality = "unsupported_modality"
	ErrCodeTimeout            = "timeout"
	ErrCodeCancelled          = "cancelled"
	ErrCodeFileNotFound       = "file_not_found"
)

// MultimodalError represents an error that occurred during multimodal operations.
type MultimodalError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *MultimodalError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("multimodal %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("multimodal %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("multimodal %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *MultimodalError) Unwrap() error {
	return e.Err
}

// NewMultimodalError creates a new MultimodalError.
func NewMultimodalError(op, code string, err error) *MultimodalError {
	return &MultimodalError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewMultimodalErrorWithMessage creates a new MultimodalError with a custom message.
func NewMultimodalErrorWithMessage(op, code, message string, err error) *MultimodalError {
	return &MultimodalError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError wraps an error as a MultimodalError.
func WrapError(err error, op, code string) *MultimodalError {
	if err == nil {
		return nil
	}
	return &MultimodalError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// IsMultimodalError checks if an error is a MultimodalError.
func IsMultimodalError(err error) bool {
	var mmErr *MultimodalError
	return errors.As(err, &mmErr)
}

// AsMultimodalError attempts to convert an error to a MultimodalError.
func AsMultimodalError(err error) (*MultimodalError, bool) {
	var mmErr *MultimodalError
	if errors.As(err, &mmErr) {
		return mmErr, true
	}
	return nil, false
}
