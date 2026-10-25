package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound ErrorCode = "not_found"

	// ErrCanceled indicates the operation was canceled by its caller (for
	// example via context cancellation). It is not retryable: the caller
	// asked for the work to stop.
	ErrCanceled ErrorCode = "canceled"

	// ErrInternal indicates a failure that carries no more specific
	// classification, such as a child agent or callback returning a plain
	// error. It is not retryable, because nothing indicates that repeating
	// the operation would succeed.
	ErrInternal ErrorCode = "internal_error"
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

// Errorf returns a new Error with the given code and a formatted message. It is
// the typed-error analogue of fmt.Errorf: if the format string contains %w
// and a wrapped error is supplied, that error is preserved as the cause and
// participates in errors.Is / errors.As / IsRetryable traversal. Multiple %w
// verbs are supported; every wrapped error stays reachable through
// errors.Is / errors.As.
//
// The formatted message already contains the text of any wrapped cause, so
// Error does not print the cause a second time.
//
// Errorf always uses the supplied code, even when the wrapped cause is itself
// an *Error with a different code. To keep the cause's classification (the
// usual intent when propagating an error from a lower layer) use [Wrapf].
//
// This is the primary entry point for capability-layer code that needs to
// return a typed error without constructing an Error literal.
func Errorf(code ErrorCode, format string, args ...any) *Error {
	// Delegate formatting to fmt.Errorf so we inherit its %w handling.
	inner := fmt.Errorf(format, args...)
	return &Error{
		Code:    code,
		Message: inner.Error(),
		Err:     wrappedCause(inner),
	}
}

// Wrapf is like [Errorf] but preserves the classification of the wrapped
// cause. The code of the returned Error is chosen as follows:
//
//   - if a %w operand (or anything in its chain) is an *Error, its Code is
//     kept, so a guard_blocked, auth_error or budget_exhausted failure is not
//     silently turned into a retryable code by an outer layer;
//   - otherwise, if the cause is context.Canceled the code is [ErrCanceled],
//     and if it is context.DeadlineExceeded the code is [ErrTimeout];
//   - otherwise fallback is used.
//
// Use Wrapf whenever an error returned by another component (an agent, a
// model, a tool, a store, a hook) is propagated with extra context.
func Wrapf(fallback ErrorCode, format string, args ...any) *Error {
	e := Errorf(fallback, format, args...)
	e.Code = CodeOf(e.Err, fallback)
	return e
}

// CodeOf returns the classification of err: the Code of the first *Error in
// its chain, [ErrCanceled] for context.Canceled, [ErrTimeout] for
// context.DeadlineExceeded, and fallback for anything else (including nil).
func CodeOf(err error, fallback ErrorCode) ErrorCode {
	if err == nil {
		return fallback
	}
	var ce *Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	switch {
	case errors.Is(err, context.Canceled):
		return ErrCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrTimeout
	}
	return fallback
}

// wrappedCause returns the error(s) wrapped by a fmt.Errorf result: the single
// cause for one %w, the fmt wrapper itself for several %w (so that
// errors.Is/As still reach every cause), and nil when nothing was wrapped.
func wrappedCause(inner error) error {
	if multi, ok := inner.(interface{ Unwrap() []error }); ok {
		if len(multi.Unwrap()) > 0 {
			return inner
		}
		return nil
	}
	return errors.Unwrap(inner)
}

// Error returns a string representation of the error including op, code,
// message, and the wrapped cause if present. The op segment is omitted when
// Op is empty, and the cause is omitted when the message already contains it
// (as it always does for errors built by Errorf and Wrapf).
func (e *Error) Error() string {
	var b strings.Builder
	if e.Op != "" {
		b.WriteString(e.Op)
		b.WriteByte(' ')
	}
	b.WriteByte('[')
	b.WriteString(string(e.Code))
	b.WriteString("]: ")
	b.WriteString(e.Message)
	if e.Err != nil {
		if cause := e.Err.Error(); !strings.Contains(e.Message, cause) {
			b.WriteString(": ")
			b.WriteString(cause)
		}
	}
	return b.String()
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
