// Package iface provides shared types and interfaces for the server package.
// This file contains option types to break circular dependencies.
package iface

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Config contains the base configuration for all server types
type Config struct {
	// Host is the server host (default: "localhost")
	Host string `mapstructure:"host" yaml:"host" env:"HOST" default:"localhost"`
	// Port is the server port (default: 8080)
	Port int `mapstructure:"port" yaml:"port" env:"PORT" default:"8080"`
	// ReadTimeout is the maximum duration for reading the entire request (default: 30s)
	ReadTimeout time.Duration `mapstructure:"read_timeout" yaml:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	// WriteTimeout is the maximum duration before timing out writes of the response (default: 30s)
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
	// IdleTimeout is the maximum amount of time to wait for the next request (default: 120s)
	IdleTimeout time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout" env:"IDLE_TIMEOUT" default:"120s"`
	// MaxHeaderBytes is the maximum number of bytes the server will read parsing the request header (default: 1MB)
	MaxHeaderBytes int `mapstructure:"max_header_bytes" yaml:"max_header_bytes" env:"MAX_HEADER_BYTES" default:"1048576"`
	// EnableCORS enables CORS headers (default: true)
	EnableCORS bool `mapstructure:"enable_cors" yaml:"enable_cors" env:"ENABLE_CORS" default:"true"`
	// CORSOrigins is the list of allowed CORS origins (default: ["*"])
	CORSOrigins []string `mapstructure:"cors_origins" yaml:"cors_origins" env:"cors_origins" default:"[\"*\"]"`
	// EnableMetrics enables OpenTelemetry metrics (default: true)
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"ENABLE_METRICS" default:"true"`
	// EnableTracing enables OpenTelemetry tracing (default: true)
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enable_tracing" env:"ENABLE_TRACING" default:"true"`
	// LogLevel sets the logging level (default: "info")
	LogLevel string `mapstructure:"log_level" yaml:"log_level" env:"LOG_LEVEL" default:"info"`
	// ShutdownTimeout is the timeout for graceful shutdown (default: 30s)
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"30s"`
}

// RESTConfig extends Config with REST-specific configuration
type RESTConfig struct {
	Config
	// APIBasePath is the base path for API endpoints (default: "/api/v1")
	APIBasePath string `mapstructure:"api_base_path" yaml:"api_base_path" env:"API_BASE_PATH" default:"/api/v1"`
	// EnableStreaming enables streaming responses (default: true)
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" env:"ENABLE_STREAMING" default:"true"`
	// MaxRequestSize is the maximum request body size in bytes (default: 10MB)
	MaxRequestSize int64 `mapstructure:"max_request_size" yaml:"max_request_size" env:"MAX_REQUEST_SIZE" default:"10485760"`
	// RateLimitRequests is the number of requests allowed per minute (default: 1000)
	RateLimitRequests int `mapstructure:"rate_limit_requests" yaml:"rate_limit_requests" env:"RATE_LIMIT_REQUESTS" default:"1000"`
	// EnableRateLimit enables rate limiting (default: true)
	EnableRateLimit bool `mapstructure:"enable_rate_limit" yaml:"enable_rate_limit" env:"ENABLE_RATE_LIMIT" default:"true"`
}

// MCPConfig extends Config with MCP-specific configuration
type MCPConfig struct {
	Config
	// ServerName is the name of the MCP server (required)
	ServerName string `mapstructure:"server_name" yaml:"server_name" env:"SERVER_NAME" validate:"required"`
	// ServerVersion is the version of the MCP server (default: "1.0.0")
	ServerVersion string `mapstructure:"server_version" yaml:"server_version" env:"SERVER_VERSION" default:"1.0.0"`
	// ProtocolVersion is the MCP protocol version (default: "2024-11-05")
	ProtocolVersion string `mapstructure:"protocol_version" yaml:"protocol_version" env:"PROTOCOL_VERSION" default:"2024-11-05"`
	// MaxConcurrentRequests is the maximum number of concurrent requests (default: 10)
	MaxConcurrentRequests int `mapstructure:"max_concurrent_requests" yaml:"max_concurrent_requests" env:"MAX_CONCURRENT_REQUESTS" default:"10"`
	// RequestTimeout is the timeout for individual tool/resource requests (default: 60s)
	RequestTimeout time.Duration `mapstructure:"request_timeout" yaml:"request_timeout" env:"REQUEST_TIMEOUT" default:"60s"`
}

// Option represents a functional option for configuring servers
type Option func(*ServerOptions)

// ServerOptions holds the configuration options for servers
type ServerOptions struct {
	Config      Config
	RESTConfig  *RESTConfig
	MCPConfig   *MCPConfig
	Logger      Logger
	Tracer      Tracer
	Meter       Meter
	Middlewares []Middleware
	Tools       []MCPTool
	Resources   []MCPResource
}

// Middleware represents HTTP middleware functions
type Middleware func(handler http.Handler) http.Handler

// Meter represents a metrics interface
type Meter = metric.Meter

// Int64Counter represents an integer counter metric
type Int64Counter = metric.Int64Counter

// Float64Histogram represents a float64 histogram metric
type Float64Histogram = metric.Float64Histogram

// WithConfig sets the base server configuration
func WithConfig(config Config) Option {
	return func(opts *ServerOptions) {
		opts.Config = config
	}
}

// WithRESTConfig sets REST-specific configuration
func WithRESTConfig(config RESTConfig) Option {
	return func(opts *ServerOptions) {
		opts.RESTConfig = &config
	}
}

// WithMCPConfig sets MCP-specific configuration
func WithMCPConfig(config MCPConfig) Option {
	return func(opts *ServerOptions) {
		opts.MCPConfig = &config
	}
}

// WithLogger sets the logger implementation
func WithLogger(logger Logger) Option {
	return func(opts *ServerOptions) {
		opts.Logger = logger
	}
}

// WithTracer sets the tracer implementation
func WithTracer(tracer Tracer) Option {
	return func(opts *ServerOptions) {
		opts.Tracer = tracer
	}
}

// WithMeter sets the metrics meter implementation
func WithMeter(meter Meter) Option {
	return func(opts *ServerOptions) {
		opts.Meter = meter
	}
}

// WithMiddleware adds middleware to the server
func WithMiddleware(middleware Middleware) Option {
	return func(opts *ServerOptions) {
		opts.Middlewares = append(opts.Middlewares, middleware)
	}
}

// WithMCPTool adds an MCP tool to the server
func WithMCPTool(tool MCPTool) Option {
	return func(opts *ServerOptions) {
		opts.Tools = append(opts.Tools, tool)
	}
}

// WithMCPResource adds an MCP resource to the server
func WithMCPResource(resource MCPResource) Option {
	return func(opts *ServerOptions) {
		opts.Resources = append(opts.Resources, resource)
	}
}

// StreamingHandler handles HTTP requests that may involve streaming responses
type StreamingHandler interface {
	// HandleStreaming processes requests that may return streaming responses
	HandleStreaming(w http.ResponseWriter, r *http.Request) error
	// HandleNonStreaming processes regular HTTP requests
	HandleNonStreaming(w http.ResponseWriter, r *http.Request) error
}

// Server represents a generic server interface that can be implemented
// by different server types (REST, MCP, etc.)
type Server interface {
	// Start begins serving requests and blocks until the context is cancelled
	Start(ctx context.Context) error
	// Stop gracefully shuts down the server
	Stop(ctx context.Context) error
	// IsHealthy returns true if the server is healthy and ready to serve requests
	IsHealthy(ctx context.Context) bool
}

// RESTServer extends Server with REST-specific functionality
type RESTServer interface {
	Server
	// RegisterHandler registers a streaming handler for a specific resource
	RegisterHandler(resource string, handler StreamingHandler)
	// RegisterHTTPHandler registers an HTTP handler for a specific method and path
	RegisterHTTPHandler(method, path string, handler http.HandlerFunc)
	// RegisterMiddleware adds middleware to the request processing pipeline
	RegisterMiddleware(middleware Middleware)
	// GetMux returns the underlying HTTP router for advanced customization
	GetMux() interface{}
}

// MCPServer extends Server with MCP-specific functionality
type MCPServer interface {
	Server
	// RegisterTool registers a tool with the MCP server
	RegisterTool(tool MCPTool) error
	// RegisterResource registers a resource with the MCP server
	RegisterResource(resource MCPResource) error
	// ListTools returns all registered tools
	ListTools(ctx context.Context) ([]MCPTool, error)
	// ListResources returns all registered resources
	ListResources(ctx context.Context) ([]MCPResource, error)
	// CallTool executes a tool by name
	CallTool(ctx context.Context, name string, input map[string]interface{}) (interface{}, error)
}

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

