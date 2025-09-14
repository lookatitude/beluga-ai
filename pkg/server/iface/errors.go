package iface

import "fmt"

// ServerError represents errors specific to server operations.
// It provides structured error information for programmatic error handling.
type ServerError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *ServerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *ServerError) Unwrap() error {
	return e.Cause
}

// NewServerError creates a new ServerError with the given code and message.
func NewServerError(code, message string, args ...interface{}) *ServerError {
	return &ServerError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with server context.
func WrapError(cause error, code, message string, args ...interface{}) *ServerError {
	return &ServerError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeServerStartFailed = "server_start_failed"
	ErrCodeServerStopFailed  = "server_stop_failed"
	ErrCodeHandlerError      = "handler_error"
	ErrCodeInvalidRequest    = "invalid_request"
	ErrCodeRouteNotFound     = "route_not_found"
	ErrCodeMethodNotAllowed  = "method_not_allowed"
	ErrCodeInternalError     = "internal_error"
	ErrCodeTimeout           = "timeout"
	ErrCodeConnectionFailed  = "connection_failed"
	ErrCodeInvalidParameters = "invalid_parameters"
)

// IsServerError checks if an error is a ServerError with the given code.
func IsServerError(err error, code string) bool {
	var srvErr *ServerError
	if !AsServerError(err, &srvErr) {
		return false
	}
	return srvErr.Code == code
}

// AsServerError attempts to cast an error to ServerError.
func AsServerError(err error, target **ServerError) bool {
	for err != nil {
		if srvErr, ok := err.(*ServerError); ok {
			*target = srvErr
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
