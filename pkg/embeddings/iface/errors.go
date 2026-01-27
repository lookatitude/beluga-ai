// Package iface provides error types and codes for embedding operations.
// This file re-exports error codes and provides convenience functions for creating errors.
// The actual error type is defined in the root embeddings package.
package iface

import (
	"errors"
	"fmt"
)

// EmbeddingError represents errors specific to embedding operations.
// It provides structured error information for programmatic error handling.
type EmbeddingError struct {
	Cause   error
	Code    string
	Message string
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
func NewEmbeddingError(code, message string, args ...any) *EmbeddingError {
	return &EmbeddingError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with embedding context.
func WrapError(cause error, code, message string, args ...any) *EmbeddingError {
	return &EmbeddingError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes.
// These are re-exported from the root embeddings package for convenience.
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
	if err == nil {
		return false
	}
	var embErr *EmbeddingError
	if errors.As(err, &embErr) {
		return embErr.Code == code
	}
	return false
}

// AsEmbeddingError attempts to cast an error to EmbeddingError.
func AsEmbeddingError(err error, target **EmbeddingError) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}
