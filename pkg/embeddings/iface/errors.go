package iface

import "fmt"

// EmbeddingError represents errors specific to embedding operations.
// It provides structured error information for programmatic error handling.
type EmbeddingError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *EmbeddingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *EmbeddingError) Unwrap() error {
	return e.Cause
}

// NewEmbeddingError creates a new EmbeddingError with the given code and message.
func NewEmbeddingError(code, message string, args ...interface{}) *EmbeddingError {
	return &EmbeddingError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with embedding context.
func WrapError(cause error, code, message string, args ...interface{}) *EmbeddingError {
	return &EmbeddingError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeProviderNotFound  = "provider_not_found"
	ErrCodeProviderDisabled  = "provider_disabled"
	ErrCodeEmbeddingFailed   = "embedding_failed"
	ErrCodeConnectionFailed  = "connection_failed"
	ErrCodeInvalidParameters = "invalid_parameters"
)

// IsEmbeddingError checks if an error is an EmbeddingError with the given code.
func IsEmbeddingError(err error, code string) bool {
	var embErr *EmbeddingError
	if !AsEmbeddingError(err, &embErr) {
		return false
	}
	return embErr.Code == code
}

// AsEmbeddingError attempts to cast an error to EmbeddingError.
func AsEmbeddingError(err error, target **EmbeddingError) bool {
	for err != nil {
		if embErr, ok := err.(*EmbeddingError); ok {
			*target = embErr
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
