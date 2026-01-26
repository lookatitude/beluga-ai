// Package safety provides error types for safety validation.
package safety

import (
	"errors"
	"fmt"
)

// ErrorCode represents safety error classifications.
type ErrorCode string

// Error codes for the safety package.
const (
	ErrCodeUnsafeContent     ErrorCode = "UNSAFE_CONTENT"
	ErrCodeCheckFailed       ErrorCode = "CHECK_FAILED"
	ErrCodeHighRisk          ErrorCode = "HIGH_RISK"
	ErrCodeInvalidConfig     ErrorCode = "INVALID_CONFIG"
	ErrCodePatternError      ErrorCode = "PATTERN_ERROR"
	ErrCodeContextCancelled  ErrorCode = "CONTEXT_CANCELLED"
)

// Common safety-related errors.
var (
	// ErrUnsafe is returned when content fails safety validation.
	ErrUnsafe = errors.New("content failed safety validation")

	// ErrUnsafeContent is returned when content contains unsafe material.
	ErrUnsafeContent = errors.New("content contains unsafe material")

	// ErrSafetyCheckFailed is returned when the safety check process fails.
	ErrSafetyCheckFailed = errors.New("safety check process failed")

	// ErrHighRiskContent is returned when content has a high safety risk score.
	ErrHighRiskContent = errors.New("content has high safety risk")
)

// Error represents a safety package error with operation context.
type Error struct {
	Op   string    // Operation that failed
	Err  error     // Underlying error
	Code ErrorCode // Error classification
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("safety %s: %s: %v", e.Op, e.Code, e.Err)
	}
	return fmt.Sprintf("safety %s: %s", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new safety error.
func NewError(op string, code ErrorCode, err error) *Error {
	return &Error{
		Op:   op,
		Err:  err,
		Code: code,
	}
}

// NewUnsafeContentError creates an error for unsafe content.
func NewUnsafeContentError(op string, err error) *Error {
	return NewError(op, ErrCodeUnsafeContent, err)
}

// NewCheckFailedError creates an error for check failures.
func NewCheckFailedError(op string, err error) *Error {
	return NewError(op, ErrCodeCheckFailed, err)
}

// NewHighRiskError creates an error for high risk content.
func NewHighRiskError(op string, err error) *Error {
	return NewError(op, ErrCodeHighRisk, err)
}

// NewInvalidConfigError creates an error for invalid configuration.
func NewInvalidConfigError(op string, err error) *Error {
	return NewError(op, ErrCodeInvalidConfig, err)
}

// IsSafetyError checks if an error is a safety error with the given code.
func IsSafetyError(err error, code ErrorCode) bool {
	var safetyErr *Error
	if errors.As(err, &safetyErr) {
		return safetyErr.Code == code
	}
	return false
}

// GetSafetyError extracts a safety error from an error chain.
func GetSafetyError(err error) *Error {
	var safetyErr *Error
	if errors.As(err, &safetyErr) {
		return safetyErr
	}
	return nil
}

// GetErrorCode extracts the error code from a safety error.
func GetErrorCode(err error) (ErrorCode, bool) {
	if safetyErr := GetSafetyError(err); safetyErr != nil {
		return safetyErr.Code, true
	}
	return "", false
}
