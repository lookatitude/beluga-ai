// Package embeddings provides custom error types for the embeddings package.
package embeddings

import (
	"errors"
	"fmt"
)

// Error codes for embedding operations.
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeEmbeddingFailed   = "embedding_failed"
	ErrCodeProviderNotFound  = "provider_not_found"
	ErrCodeProviderError     = "provider_error"
	ErrCodeNetworkError      = "network_error"
	ErrCodeTimeout           = "timeout"
	ErrCodeRateLimit         = "rate_limit"
	ErrCodeAuthentication    = "authentication_error"
	ErrCodeInvalidDimension  = "invalid_dimension"
	ErrCodeBatchSizeExceeded = "batch_size_exceeded"
	ErrCodeContextCanceled   = "context_canceled"
	ErrCodeContextTimeout    = "context_timeout"
)

// EmbeddingError represents an error that occurred during embedding operations.
type EmbeddingError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *EmbeddingError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("embeddings %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("embeddings %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("embeddings %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *EmbeddingError) Unwrap() error {
	return e.Err
}

// NewEmbeddingError creates a new EmbeddingError.
func NewEmbeddingError(op, code string, err error) *EmbeddingError {
	return &EmbeddingError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewEmbeddingErrorWithMessage creates a new EmbeddingError with a custom message.
func NewEmbeddingErrorWithMessage(op, code, message string, err error) *EmbeddingError {
	return &EmbeddingError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsEmbeddingError checks if an error is an EmbeddingError.
func IsEmbeddingError(err error) bool {
	var embeddingErr *EmbeddingError
	return errors.As(err, &embeddingErr)
}

// AsEmbeddingError attempts to convert an error to an EmbeddingError.
func AsEmbeddingError(err error) (*EmbeddingError, bool) {
	var embeddingErr *EmbeddingError
	if errors.As(err, &embeddingErr) {
		return embeddingErr, true
	}
	return nil, false
}
