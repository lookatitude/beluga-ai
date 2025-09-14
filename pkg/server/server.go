// Package server provides HTTP API endpoints and MCP server implementation for the Beluga AI framework.
// It supports both REST APIs with and without streaming, as well as the Model Context Protocol (MCP).
//
// The package provides:
// - REST server with streaming capabilities
// - MCP server for tool and resource integration
// - Comprehensive observability with OpenTelemetry
// - Structured error handling
// - Configurable middleware and handlers
//
// Example usage:
//
//	// Create a REST server
//	restServer, err := server.NewRESTServer(
//		server.WithRESTConfig(server.RESTConfig{
//			Config: server.Config{
//				Host: "localhost",
//				Port: 8080,
//			},
//			APIBasePath: "/api/v1",
//		}),
//		server.WithLogger(logger),
//		server.WithTracer(tracer),
//		server.WithMeter(meter),
//	)
//
//	// Register a streaming handler
//	restServer.RegisterHandler("agents", agentHandler)
//
//	// Start the server
//	ctx := context.Background()
//	if err := restServer.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//
// Example MCP server usage:
//
//	// Create an MCP server
//	mcpServer, err := server.NewMCPServer(
//		server.WithMCPConfig(server.MCPConfig{
//			Config: server.Config{
//				Host: "localhost",
//				Port: 8081,
//			},
//			ServerName: "beluga-mcp",
//		}),
//	)
//
//	// Register tools and resources
//	mcpServer.RegisterTool(calculatorTool)
//	mcpServer.RegisterResource(documentResource)
//
//	// Start the server
//	if err := mcpServer.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

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
	RegisterMiddleware(middleware func(http.Handler) http.Handler)
	// GetMux returns the underlying HTTP router for advanced customization
	GetMux() interface{}
}

// StreamingHandler handles HTTP requests that may involve streaming responses
type StreamingHandler interface {
	// HandleStreaming processes requests that may return streaming responses
	HandleStreaming(w http.ResponseWriter, r *http.Request) error
	// HandleNonStreaming processes regular HTTP requests
	HandleNonStreaming(w http.ResponseWriter, r *http.Request) error
}

// MCPTool represents a tool that can be invoked via MCP
type MCPTool = iface.MCPTool

// MCPResource represents a resource that can be accessed via MCP
type MCPResource = iface.MCPResource

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

// NewRESTServer creates a new REST server instance with the provided options.
// It implements the RESTServer interface and provides HTTP endpoints with streaming support.
//
// Note: This is a factory function that should be implemented by specific providers.
// Use providers.NewRESTProvider() for a concrete implementation.
func NewRESTServer(opts ...Option) (RESTServer, error) {
	panic("NewRESTServer is not implemented in the base server package. Use providers.NewRESTProvider() instead.")
}

// NewMCPServer creates a new MCP server instance with the provided options.
// It implements the MCPServer interface and provides MCP protocol support for tools and resources.
//
// Note: This is a factory function that should be implemented by specific providers.
// Use providers.NewMCPProvider() for a concrete implementation.
func NewMCPServer(opts ...Option) (MCPServer, error) {
	panic("NewMCPServer is not implemented in the base server package. Use providers.NewMCPProvider() instead.")
}

// Default configurations

// DefaultRESTConfig returns a default REST server configuration
func DefaultRESTConfig() RESTConfig {
	return RESTConfig{
		Config: Config{
			Host:            "localhost",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxHeaderBytes:  1 << 20, // 1MB
			EnableCORS:      true,
			CORSOrigins:     []string{"*"},
			EnableMetrics:   true,
			EnableTracing:   true,
			LogLevel:        "info",
			ShutdownTimeout: 30 * time.Second,
		},
		APIBasePath:       "/api/v1",
		EnableStreaming:   true,
		MaxRequestSize:    10 << 20, // 10MB
		RateLimitRequests: 1000,
		EnableRateLimit:   true,
	}
}

// DefaultMCPConfig returns a default MCP server configuration
func DefaultMCPConfig() MCPConfig {
	return MCPConfig{
		Config: Config{
			Host:            "localhost",
			Port:            8081,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxHeaderBytes:  1 << 20, // 1MB
			EnableMetrics:   true,
			EnableTracing:   true,
			LogLevel:        "info",
			ShutdownTimeout: 30 * time.Second,
		},
		ServerName:            "beluga-mcp-server",
		ServerVersion:         "1.0.0",
		ProtocolVersion:       "2024-11-05",
		MaxConcurrentRequests: 10,
		RequestTimeout:        60 * time.Second,
	}
}

// Common middleware functions

// CORSMiddleware returns a CORS middleware function
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if !allowed {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware returns a logging middleware function
func LoggingMiddleware(logger Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware returns a panic recovery middleware function
func RecoveryMiddleware(logger Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered", "error", err, "path", r.URL.Path)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
