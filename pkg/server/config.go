// Package server provides configuration structures for server implementations.
package server

import (
	"context"
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
type Option func(*serverOptions)

// serverOptions holds the configuration options for servers
type serverOptions struct {
	config      Config
	restConfig  *RESTConfig
	mcpConfig   *MCPConfig
	logger      Logger
	tracer      Tracer
	meter       Meter
	middlewares []Middleware
	tools       []MCPTool
	resources   []MCPResource
}

// WithConfig sets the base server configuration
func WithConfig(config Config) Option {
	return func(opts *serverOptions) {
		opts.config = config
	}
}

// WithRESTConfig sets REST-specific configuration
func WithRESTConfig(config RESTConfig) Option {
	return func(opts *serverOptions) {
		opts.restConfig = &config
	}
}

// WithMCPConfig sets MCP-specific configuration
func WithMCPConfig(config MCPConfig) Option {
	return func(opts *serverOptions) {
		opts.mcpConfig = &config
	}
}

// WithLogger sets the logger implementation
func WithLogger(logger Logger) Option {
	return func(opts *serverOptions) {
		opts.logger = logger
	}
}

// WithTracer sets the tracer implementation
func WithTracer(tracer Tracer) Option {
	return func(opts *serverOptions) {
		opts.tracer = tracer
	}
}

// WithMeter sets the metrics meter implementation
func WithMeter(meter Meter) Option {
	return func(opts *serverOptions) {
		opts.meter = meter
	}
}

// WithMiddleware adds middleware to the server
func WithMiddleware(middleware Middleware) Option {
	return func(opts *serverOptions) {
		opts.middlewares = append(opts.middlewares, middleware)
	}
}

// WithMCPTool adds an MCP tool to the server
func WithMCPTool(tool MCPTool) Option {
	return func(opts *serverOptions) {
		opts.tools = append(opts.tools, tool)
	}
}

// WithMCPResource adds an MCP resource to the server
func WithMCPResource(resource MCPResource) Option {
	return func(opts *serverOptions) {
		opts.resources = append(opts.resources, resource)
	}
}

// Middleware represents HTTP middleware functions
type Middleware func(handler http.Handler) http.Handler

// Logger represents a logging interface
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Tracer represents a tracing interface
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a tracing span
type Span interface {
	End()
	SetAttributes(attrs ...interface{})
	RecordError(err error)
	SetStatus(code int, msg string)
}

// Meter represents a metrics interface
type Meter = metric.Meter

// Int64Counter represents an integer counter metric
type Int64Counter = metric.Int64Counter

// Float64Histogram represents a float64 histogram metric
type Float64Histogram = metric.Float64Histogram
