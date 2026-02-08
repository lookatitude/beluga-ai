// Package core provides the foundational primitives for the Beluga AI framework:
// typed event streams, the Runnable execution interface, batch processing,
// context helpers, multi-tenancy, lifecycle management, and typed errors.
package core

import (
	"errors"
	"fmt"
)

// ErrorCode identifies the category of an error. Downstream packages use these
// codes to decide on retry strategy, alerting, and user-facing messages.
type ErrorCode string

const (
	// ErrRateLimit indicates the upstream provider has throttled the request.
	ErrRateLimit ErrorCode = "rate_limit"

	// ErrAuth indicates an authentication or authorization failure.
	ErrAuth ErrorCode = "auth_error"

	// ErrTimeout indicates the operation exceeded its deadline.
	ErrTimeout ErrorCode = "timeout"

	// ErrInvalidInput indicates the caller supplied malformed or missing input.
	ErrInvalidInput ErrorCode = "invalid_input"

	// ErrToolFailed indicates a tool execution returned an error.
	ErrToolFailed ErrorCode = "tool_failed"

	// ErrProviderDown indicates the upstream provider is unreachable.
	ErrProviderDown ErrorCode = "provider_unavailable"

	// ErrGuardBlocked indicates a guard rejected the request.
	ErrGuardBlocked ErrorCode = "guard_blocked"

	// ErrBudgetExhausted indicates a token or cost budget has been exceeded.
	ErrBudgetExhausted ErrorCode = "budget_exhausted"
)

// retryableCodes is the set of error codes that should be retried.
var retryableCodes = map[ErrorCode]bool{
	ErrRateLimit:    true,
	ErrTimeout:      true,
	ErrProviderDown: true,
}

// Error is a structured error carrying an operation name, error code,
// human-readable message, and an optional wrapped cause.
type Error struct {
	// Op is the operation that failed, e.g. "llm.generate" or "tool.execute".
	Op string

	// Code categorizes the error for programmatic handling.
	Code ErrorCode

	// Message is a human-readable description of what went wrong.
	Message string

	// Err is the underlying cause, if any.
	Err error
}

// NewError creates a new Error with the given operation, code, message, and
// optional cause.
func NewError(op string, code ErrorCode, msg string, cause error) *Error {
	return &Error{
		Op:      op,
		Code:    code,
		Message: msg,
		Err:     cause,
	}
}

// Error returns a string representation of the error including op, code,
// message, and the wrapped cause if present.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s [%s]: %s: %v", e.Op, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Op, e.Code, e.Message)
}

// Unwrap returns the underlying cause so errors.Is and errors.As traverse
// the error chain.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether target matches this error. Two Errors match if they share
// the same Code.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// IsRetryable reports whether err (or any error in its chain) has a retryable
// error code. Retryable codes are rate_limit, timeout, and
// provider_unavailable.
func IsRetryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return retryableCodes[e.Code]
	}
	return false
}
