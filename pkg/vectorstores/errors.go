// Package vectorstores provides custom error types for the vectorstores package.
package vectorstores

import (
	"errors"
	"fmt"
)

// Error codes for vector store operations.
const (
	ErrCodeInvalidConfig      = "invalid_config"
	ErrCodeInvalidInput       = "invalid_input"
	ErrCodeStorageFailed      = "storage_failed"
	ErrCodeRetrievalFailed    = "retrieval_failed"
	ErrCodeProviderNotFound   = "provider_not_found"
	ErrCodeProviderError      = "provider_error"
	ErrCodeNetworkError       = "network_error"
	ErrCodeTimeout            = "timeout"
	ErrCodeConnectionFailed   = "connection_failed"
	ErrCodeIndexNotFound      = "index_not_found"
	ErrCodeIndexCreationFailed = "index_creation_failed"
	ErrCodeDocumentNotFound    = "document_not_found"
	ErrCodeEmbeddingFailed     = "embedding_failed"
	ErrCodeInvalidVector       = "invalid_vector"
	ErrCodeInvalidDimension    = "invalid_dimension"
	ErrCodeSearchFailed        = "search_failed"
	ErrCodeContextCanceled     = "context_canceled"
	ErrCodeContextTimeout      = "context_timeout"
)

// VectorStoreError represents an error that occurred during vector store operations.
type VectorStoreError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *VectorStoreError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("vectorstores %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("vectorstores %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("vectorstores %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *VectorStoreError) Unwrap() error {
	return e.Err
}

// NewVectorStoreError creates a new VectorStoreError.
func NewVectorStoreError(op, code string, err error) *VectorStoreError {
	return &VectorStoreError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewVectorStoreErrorWithMessage creates a new VectorStoreError with a custom message.
func NewVectorStoreErrorWithMessage(op, code, message string, err error) *VectorStoreError {
	return &VectorStoreError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError wraps an error as a VectorStoreError.
func WrapError(err error, op, code string) *VectorStoreError {
	if err == nil {
		return nil
	}
	return &VectorStoreError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// IsVectorStoreError checks if an error is a VectorStoreError.
func IsVectorStoreError(err error) bool {
	var vsErr *VectorStoreError
	return errors.As(err, &vsErr)
}

// AsVectorStoreError attempts to convert an error to a VectorStoreError.
func AsVectorStoreError(err error) (*VectorStoreError, bool) {
	var vsErr *VectorStoreError
	if errors.As(err, &vsErr) {
		return vsErr, true
	}
	return nil, false
}
