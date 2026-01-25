package documentloaders

import (
	"errors"
	"fmt"
)

// Error codes for loader operations.
const (
	ErrCodeIOError       = "io_error"
	ErrCodeNotFound      = "not_found"
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeCycleDetected = "cycle_detected"
	ErrCodeFileTooLarge  = "file_too_large"
	ErrCodeBinaryFile    = "binary_file"
	ErrCodeCancelled     = "canceled"
)

// LoaderError represents an error during document loading.
type LoaderError struct {
	Err     error
	Op      string
	Code    string
	Path    string
	Message string
}

// Error implements the error interface.
func (e *LoaderError) Error() string {
	if e.Message != "" {
		if e.Path != "" {
			return fmt.Sprintf("documentloaders %s [%s]: %s (code: %s)", e.Op, e.Path, e.Message, e.Code)
		}
		return fmt.Sprintf("documentloaders %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		if e.Path != "" {
			return fmt.Sprintf("documentloaders %s [%s]: %v (code: %s)", e.Op, e.Path, e.Err, e.Code)
		}
		return fmt.Sprintf("documentloaders %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("documentloaders %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *LoaderError) Unwrap() error {
	return e.Err
}

// NewLoaderError creates a new LoaderError.
func NewLoaderError(op, code, path, message string, err error) *LoaderError {
	return &LoaderError{
		Op:      op,
		Code:    code,
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// IsLoaderError checks if an error is a LoaderError.
func IsLoaderError(err error) bool {
	var loaderErr *LoaderError
	return errors.As(err, &loaderErr)
}

// GetLoaderError extracts a LoaderError from an error if it exists.
func GetLoaderError(err error) *LoaderError {
	var loaderErr *LoaderError
	if errors.As(err, &loaderErr) {
		return loaderErr
	}
	return nil
}
