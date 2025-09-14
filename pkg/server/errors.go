// Package server provides custom error types for server implementations.
package server

import (
	"fmt"
	"net/http"
)

// ErrorCode represents different types of server errors
type ErrorCode string

const (
	// HTTP error codes
	ErrCodeInvalidRequest   ErrorCode = "invalid_request"
	ErrCodeMethodNotAllowed ErrorCode = "method_not_allowed"
	ErrCodeNotFound         ErrorCode = "not_found"
	ErrCodeInternalError    ErrorCode = "internal_error"
	ErrCodeTimeout          ErrorCode = "timeout"
	ErrCodeRateLimited      ErrorCode = "rate_limited"
	ErrCodeUnauthorized     ErrorCode = "unauthorized"
	ErrCodeForbidden        ErrorCode = "forbidden"

	// MCP error codes
	ErrCodeToolNotFound     ErrorCode = "tool_not_found"
	ErrCodeResourceNotFound ErrorCode = "resource_not_found"
	ErrCodeToolExecution    ErrorCode = "tool_execution_error"
	ErrCodeResourceRead     ErrorCode = "resource_read_error"
	ErrCodeInvalidToolInput ErrorCode = "invalid_tool_input"
	ErrCodeMCPProtocol      ErrorCode = "mcp_protocol_error"

	// Server error codes
	ErrCodeServerStartup    ErrorCode = "server_startup_error"
	ErrCodeServerShutdown   ErrorCode = "server_shutdown_error"
	ErrCodeConfigValidation ErrorCode = "config_validation_error"
)

// ServerError represents a structured server error
type ServerError struct {
	Code      ErrorCode   `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Operation string      `json:"operation,omitempty"`
	Err       error       `json:"-"`
}

// Error implements the error interface
func (e *ServerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Operation, e.Message, e.Err.Error())
	}
	if e.Operation != "" {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *ServerError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the appropriate HTTP status code for this error
func (e *ServerError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeInvalidRequest, ErrCodeInvalidToolInput:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound, ErrCodeToolNotFound, ErrCodeResourceNotFound:
		return http.StatusNotFound
	case ErrCodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeInternalError, ErrCodeToolExecution, ErrCodeResourceRead,
		ErrCodeServerStartup, ErrCodeServerShutdown, ErrCodeConfigValidation,
		ErrCodeMCPProtocol:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewInvalidRequestError creates a new invalid request error
func NewInvalidRequestError(operation string, message string, details interface{}) *ServerError {
	return &ServerError{
		Code:      ErrCodeInvalidRequest,
		Message:   message,
		Details:   details,
		Operation: operation,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(operation string, resource string) *ServerError {
	return &ServerError{
		Code:      ErrCodeNotFound,
		Message:   fmt.Sprintf("%s not found", resource),
		Operation: operation,
	}
}

// NewInternalError creates a new internal server error
func NewInternalError(operation string, err error) *ServerError {
	return &ServerError{
		Code:      ErrCodeInternalError,
		Message:   "internal server error",
		Operation: operation,
		Err:       err,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation string) *ServerError {
	return &ServerError{
		Code:      ErrCodeTimeout,
		Message:   "request timeout",
		Operation: operation,
	}
}

// NewToolNotFoundError creates a new tool not found error
func NewToolNotFoundError(toolName string) *ServerError {
	return &ServerError{
		Code:      ErrCodeToolNotFound,
		Message:   fmt.Sprintf("tool '%s' not found", toolName),
		Operation: "tool_execution",
	}
}

// NewResourceNotFoundError creates a new resource not found error
func NewResourceNotFoundError(resourceURI string) *ServerError {
	return &ServerError{
		Code:      ErrCodeResourceNotFound,
		Message:   fmt.Sprintf("resource '%s' not found", resourceURI),
		Operation: "resource_read",
	}
}

// NewToolExecutionError creates a new tool execution error
func NewToolExecutionError(toolName string, err error) *ServerError {
	return &ServerError{
		Code:      ErrCodeToolExecution,
		Message:   fmt.Sprintf("failed to execute tool '%s'", toolName),
		Operation: "tool_execution",
		Err:       err,
	}
}

// NewResourceReadError creates a new resource read error
func NewResourceReadError(resourceURI string, err error) *ServerError {
	return &ServerError{
		Code:      ErrCodeResourceRead,
		Message:   fmt.Sprintf("failed to read resource '%s'", resourceURI),
		Operation: "resource_read",
		Err:       err,
	}
}

// NewInvalidToolInputError creates a new invalid tool input error
func NewInvalidToolInputError(toolName string, details interface{}) *ServerError {
	return &ServerError{
		Code:      ErrCodeInvalidToolInput,
		Message:   fmt.Sprintf("invalid input for tool '%s'", toolName),
		Details:   details,
		Operation: "tool_execution",
	}
}

// NewConfigValidationError creates a new configuration validation error
func NewConfigValidationError(field string, reason string) *ServerError {
	return &ServerError{
		Code:      ErrCodeConfigValidation,
		Message:   fmt.Sprintf("configuration validation failed for field '%s': %s", field, reason),
		Details:   map[string]string{"field": field, "reason": reason},
		Operation: "config_validation",
	}
}

// NewMCPProtocolError creates a new MCP protocol error
func NewMCPProtocolError(operation string, err error) *ServerError {
	return &ServerError{
		Code:      ErrCodeMCPProtocol,
		Message:   "MCP protocol error",
		Operation: operation,
		Err:       err,
	}
}
