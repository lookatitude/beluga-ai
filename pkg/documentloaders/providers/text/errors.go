package text

import (
	"fmt"
)

// Error codes (duplicated from parent package to avoid import cycle).
const (
	ErrCodeIOError       = "io_error"
	ErrCodeNotFound      = "not_found"
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeFileTooLarge  = "file_too_large"
	ErrCodeCancelled     = "canceled"
)

// LoaderError represents an error during document loading (duplicated to avoid import cycle).
type LoaderError struct {
	Err     error
	Op      string
	Code    string
	Path    string
	Message string
}

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

// newLoaderError creates a LoaderError without importing the parent package.
func newLoaderError(op, code, path, message string, err error) *LoaderError {
	return &LoaderError{
		Op:      op,
		Code:    code,
		Path:    path,
		Message: message,
		Err:     err,
	}
}
