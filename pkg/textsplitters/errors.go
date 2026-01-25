package textsplitters

import (
	"errors"
	"fmt"
)

// Error codes for splitter operations.
const (
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeEmptyInput    = "empty_input"
	ErrCodeNotFound      = "not_found"
	ErrCodeCancelled     = "canceled"
)

// SplitterError represents an error during text splitting.
type SplitterError struct {
	Err     error
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *SplitterError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("textsplitters %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("textsplitters %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("textsplitters %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *SplitterError) Unwrap() error {
	return e.Err
}

// NewSplitterError creates a new SplitterError.
func NewSplitterError(op, code, message string, err error) *SplitterError {
	return &SplitterError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsSplitterError checks if an error is a SplitterError.
func IsSplitterError(err error) bool {
	var splitterErr *SplitterError
	return errors.As(err, &splitterErr)
}

// GetSplitterError extracts a SplitterError from an error if it exists.
func GetSplitterError(err error) *SplitterError {
	var splitterErr *SplitterError
	if errors.As(err, &splitterErr) {
		return splitterErr
	}
	return nil
}
