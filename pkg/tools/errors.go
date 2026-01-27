// Package tools provides standardized error handling for tool execution.
package tools

import (
	"errors"
	"fmt"
)

// ErrorCode represents standardized error codes for tool operations.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters.
	ErrorCodeInvalidInput ErrorCode = "invalid_input"

	// ErrorCodeInvalidSchema indicates an invalid input schema.
	ErrorCodeInvalidSchema ErrorCode = "invalid_schema"

	// ErrorCodeExecutionFailed indicates tool execution failed.
	ErrorCodeExecutionFailed ErrorCode = "execution_failed"

	// ErrorCodeTimeout indicates a timeout occurred.
	ErrorCodeTimeout ErrorCode = "timeout"

	// ErrorCodeNotFound indicates a tool was not found.
	ErrorCodeNotFound ErrorCode = "not_found"

	// ErrorCodeAlreadyExists indicates a tool already exists.
	ErrorCodeAlreadyExists ErrorCode = "already_exists"

	// ErrorCodeUnsupported indicates an unsupported operation.
	ErrorCodeUnsupported ErrorCode = "unsupported"

	// ErrorCodeRateLimited indicates rate limiting was applied.
	ErrorCodeRateLimited ErrorCode = "rate_limited"

	// ErrorCodePermissionDenied indicates permission was denied.
	ErrorCodePermissionDenied ErrorCode = "permission_denied"

	// ErrorCodeInternalError indicates an internal system error.
	ErrorCodeInternalError ErrorCode = "internal_error"
)

// Static base errors for dynamic error wrapping (err113 compliance).
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidSchema    = errors.New("invalid schema")
	ErrExecutionFailed  = errors.New("execution failed")
	ErrTimeout          = errors.New("operation timed out")
	ErrNotFound         = errors.New("tool not found")
	ErrAlreadyExists    = errors.New("tool already exists")
	ErrUnsupported      = errors.New("unsupported operation")
	ErrRateLimited      = errors.New("rate limited")
	ErrPermissionDenied = errors.New("permission denied")
)

// ToolError represents a standardized error in the tools package.
// It follows the Op/Err/Code pattern used across all Beluga AI packages.
type ToolError struct {
	Err      error
	Context  map[string]any
	Op       string
	Code     ErrorCode
	Message  string
	ToolName string
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	var base string
	if e.ToolName != "" {
		base = fmt.Sprintf("tools[%s] %s", e.ToolName, e.Op)
	} else {
		base = "tools " + e.Op
	}

	if e.Message != "" {
		return fmt.Sprintf("%s: %s (code: %s)", base, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v (code: %s)", base, e.Err, e.Code)
	}
	return fmt.Sprintf("%s: unknown error (code: %s)", base, e.Code)
}

// Unwrap returns the underlying error.
func (e *ToolError) Unwrap() error {
	return e.Err
}

// NewToolError creates a ToolError following the Op/Err/Code pattern.
func NewToolError(op string, code ErrorCode, message string, err error) *ToolError {
	return &ToolError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]any),
	}
}

// NewToolErrorWithName creates a ToolError with the tool name included.
func NewToolErrorWithName(toolName, op string, code ErrorCode, message string, err error) *ToolError {
	e := NewToolError(op, code, message, err)
	e.ToolName = toolName
	return e
}

// NewInvalidInputError creates a new invalid input error.
func NewInvalidInputError(op, message string, err error) *ToolError {
	return NewToolError(op, ErrorCodeInvalidInput, message, err)
}

// NewInvalidSchemaError creates a new invalid schema error.
func NewInvalidSchemaError(op, message string, err error) *ToolError {
	return NewToolError(op, ErrorCodeInvalidSchema, message, err)
}

// NewExecutionError creates a new execution error.
func NewExecutionError(toolName, op, message string, err error) *ToolError {
	return NewToolErrorWithName(toolName, op, ErrorCodeExecutionFailed, message, err)
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(toolName, op, message string) *ToolError {
	return NewToolErrorWithName(toolName, op, ErrorCodeTimeout, message, ErrTimeout)
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(op, toolName string) *ToolError {
	return NewToolError(op, ErrorCodeNotFound, fmt.Sprintf("tool '%s' not found", toolName), ErrNotFound)
}

// NewAlreadyExistsError creates a new already exists error.
func NewAlreadyExistsError(op, toolName string) *ToolError {
	return NewToolError(op, ErrorCodeAlreadyExists, fmt.Sprintf("tool '%s' already exists", toolName), ErrAlreadyExists)
}

// NewUnsupportedError creates a new unsupported operation error.
func NewUnsupportedError(op, message string) *ToolError {
	return NewToolError(op, ErrorCodeUnsupported, message, ErrUnsupported)
}

// NewRateLimitError creates a new rate limit error.
func NewRateLimitError(toolName, op, retryAfter string) *ToolError {
	e := NewToolErrorWithName(toolName, op, ErrorCodeRateLimited, "rate limit exceeded", ErrRateLimited)
	if retryAfter != "" {
		e.Context["retry_after"] = retryAfter
	}
	return e
}

// NewPermissionDeniedError creates a new permission denied error.
func NewPermissionDeniedError(toolName, op, message string) *ToolError {
	return NewToolErrorWithName(toolName, op, ErrorCodePermissionDenied, message, ErrPermissionDenied)
}

// NewInternalError creates a new internal error.
func NewInternalError(op, message string, err error) *ToolError {
	return NewToolError(op, ErrorCodeInternalError, message, err)
}

// AddContext adds context information to a ToolError.
func (e *ToolError) AddContext(key string, value any) *ToolError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// IsToolError checks if an error is a ToolError.
func IsToolError(err error) bool {
	var toolErr *ToolError
	return errors.As(err, &toolErr)
}

// AsToolError attempts to convert an error to a ToolError.
func AsToolError(err error) (*ToolError, bool) {
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return toolErr, true
	}
	return nil, false
}

// GetErrorCode extracts the error code from an error if it's a ToolError.
func GetErrorCode(err error) (ErrorCode, bool) {
	if toolErr, ok := AsToolError(err); ok {
		return toolErr.Code, true
	}
	return "", false
}

// IsErrorCode checks if an error has a specific error code.
func IsErrorCode(err error, code ErrorCode) bool {
	if toolErr, ok := AsToolError(err); ok {
		return toolErr.Code == code
	}
	return false
}

// WrapError wraps an error with additional context.
func WrapError(err error, op, message string) error {
	if err == nil {
		return nil
	}
	return NewToolError(op, ErrorCodeInternalError, message, err)
}
