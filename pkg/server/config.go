// Package server provides configuration structures for server implementations.
package server

import (
	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Re-export types from iface to maintain backward compatibility.
type (
	Config           = iface.Config
	RESTConfig       = iface.RESTConfig
	MCPConfig        = iface.MCPConfig
	Option           = iface.Option
	Middleware       = iface.Middleware
	Logger           = iface.Logger
	Tracer           = iface.Tracer
	Span             = iface.Span
	Meter            = iface.Meter
	Int64Counter     = iface.Int64Counter
	Float64Histogram = iface.Float64Histogram
	MCPTool          = iface.MCPTool
	MCPResource      = iface.MCPResource
	StreamingHandler = iface.StreamingHandler
	Server           = iface.Server
	RESTServer       = iface.RESTServer
	MCPServer        = iface.MCPServer
	ErrorCode        = iface.ErrorCode
	ServerError      = iface.ServerError
)

const (
	ErrCodeInvalidRequest   = iface.ErrCodeInvalidRequest
	ErrCodeMethodNotAllowed = iface.ErrCodeMethodNotAllowed
	ErrCodeNotFound         = iface.ErrCodeNotFound
	ErrCodeInternalError    = iface.ErrCodeInternalError
	ErrCodeTimeout          = iface.ErrCodeTimeout
	ErrCodeRateLimited      = iface.ErrCodeRateLimited
	ErrCodeUnauthorized     = iface.ErrCodeUnauthorized
	ErrCodeForbidden        = iface.ErrCodeForbidden
	ErrCodeToolNotFound     = iface.ErrCodeToolNotFound
	ErrCodeResourceNotFound = iface.ErrCodeResourceNotFound
	ErrCodeToolExecution    = iface.ErrCodeToolExecution
	ErrCodeResourceRead     = iface.ErrCodeResourceRead
	ErrCodeInvalidToolInput = iface.ErrCodeInvalidToolInput
	ErrCodeMCPProtocol      = iface.ErrCodeMCPProtocol
	ErrCodeServerStartup    = iface.ErrCodeServerStartup
	ErrCodeServerShutdown   = iface.ErrCodeServerShutdown
	ErrCodeConfigValidation = iface.ErrCodeConfigValidation
)

// Re-export functions from iface.
var (
	WithConfig               = iface.WithConfig
	WithRESTConfig           = iface.WithRESTConfig
	WithMCPConfig            = iface.WithMCPConfig
	WithLogger               = iface.WithLogger
	WithTracer               = iface.WithTracer
	WithMeter                = iface.WithMeter
	WithMiddleware           = iface.WithMiddleware
	WithMCPTool              = iface.WithMCPTool
	WithMCPResource          = iface.WithMCPResource
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
