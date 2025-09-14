package vectorstores

import "fmt"

// VectorStoreError represents errors specific to vector store operations.
// It provides structured error information for programmatic error handling.
type VectorStoreError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *VectorStoreError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *VectorStoreError) Unwrap() error {
	return e.Cause
}

// NewVectorStoreError creates a new VectorStoreError with the given code and message.
func NewVectorStoreError(code, message string, args ...interface{}) *VectorStoreError {
	return &VectorStoreError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with vector store context.
func WrapError(cause error, code, message string, args ...interface{}) *VectorStoreError {
	return &VectorStoreError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeUnknownProvider      = "unknown_provider"
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeConnectionFailed     = "connection_failed"
	ErrCodeEmbeddingFailed      = "embedding_failed"
	ErrCodeStorageFailed        = "storage_failed"
	ErrCodeRetrievalFailed      = "retrieval_failed"
	ErrCodeInvalidParameters    = "invalid_parameters"
	ErrCodeNotFound             = "not_found"
	ErrCodeDuplicateID          = "duplicate_id"
	ErrCodeUnsupportedOperation = "unsupported_operation"
)

// IsVectorStoreError checks if an error is a VectorStoreError with the given code.
func IsVectorStoreError(err error, code string) bool {
	var vsErr *VectorStoreError
	if !AsVectorStoreError(err, &vsErr) {
		return false
	}
	return vsErr.Code == code
}

// AsVectorStoreError attempts to cast an error to VectorStoreError.
func AsVectorStoreError(err error, target **VectorStoreError) bool {
	for err != nil {
		if vsErr, ok := err.(*VectorStoreError); ok {
			*target = vsErr
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
