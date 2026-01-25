package markdown

import (
	"fmt"
)

// Error codes (duplicated from parent package to avoid import cycle).
const (
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeEmptyInput    = "empty_input"
	ErrCodeCancelled     = "canceled"
)

// SplitterError represents an error during text splitting (duplicated to avoid import cycle).
type SplitterError struct {
	Err     error
	Op      string
	Code    string
	Message string
}

func (e *SplitterError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("textsplitters %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("textsplitters %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("textsplitters %s: unknown error (code: %s)", e.Op, e.Code)
}

// newSplitterError creates a SplitterError without importing the parent package.
func newSplitterError(op, code, message string, err error) *SplitterError {
	return &SplitterError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
