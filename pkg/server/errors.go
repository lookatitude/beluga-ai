// Package server provides custom error types for the server package.
package server

import (
	"errors"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Re-export error types from iface for consistency with other packages.
type (
	ErrorCode   = iface.ErrorCode
	ServerError = iface.ServerError
)

// Re-export error code constants from iface.
const (
	ErrCodeInvalidRequest   = iface.ErrCodeInvalidRequest
	ErrCodeMethodNotAllowed = iface.ErrCodeMethodNotAllowed
	ErrCodeNotFound         = iface.ErrCodeNotFound
	ErrCodeInternalError    = iface.ErrCodeInternalError
	ErrCodeTimeout          = iface.ErrCodeTimeout
	ErrCodeRateLimited      = iface.ErrCodeRateLimited
	ErrCodeUnauthorized     = iface.ErrCodeUnauthorized
	ErrCodeForbidden        = iface.ErrCodeForbidden

	// MCP error codes.
	ErrCodeToolNotFound     = iface.ErrCodeToolNotFound
	ErrCodeResourceNotFound = iface.ErrCodeResourceNotFound
	ErrCodeToolExecution    = iface.ErrCodeToolExecution
	ErrCodeResourceRead     = iface.ErrCodeResourceRead
	ErrCodeInvalidToolInput = iface.ErrCodeInvalidToolInput
	ErrCodeMCPProtocol      = iface.ErrCodeMCPProtocol

	// Server error codes.
	ErrCodeServerStartup    = iface.ErrCodeServerStartup
	ErrCodeServerShutdown   = iface.ErrCodeServerShutdown
	ErrCodeConfigValidation = iface.ErrCodeConfigValidation
)

// Re-export error constructor functions from iface.
var (
	NewInvalidRequestError   = iface.NewInvalidRequestError
	NewNotFoundError         = iface.NewNotFoundError
	NewInternalError         = iface.NewInternalError
	NewTimeoutError          = iface.NewTimeoutError
	NewToolNotFoundError     = iface.NewToolNotFoundError
	NewResourceNotFoundError = iface.NewResourceNotFoundError
	NewToolExecutionError    = iface.NewToolExecutionError
	NewResourceReadError     = iface.NewResourceReadError
	NewInvalidToolInputError = iface.NewInvalidToolInputError
	NewConfigValidationError = iface.NewConfigValidationError
	NewMCPProtocolError      = iface.NewMCPProtocolError
)

// IsServerError checks if an error is a ServerError.
func IsServerError(err error) bool {
	var serverErr *ServerError
	return errors.As(err, &serverErr)
}

// AsServerError attempts to convert an error to a ServerError.
func AsServerError(err error) (*ServerError, bool) {
	var serverErr *ServerError
	if errors.As(err, &serverErr) {
		return serverErr, true
	}
	return nil, false
}
