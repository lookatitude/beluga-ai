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
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
	"github.com/lookatitude/beluga-ai/pkg/server/providers/mcp"
	"github.com/lookatitude/beluga-ai/pkg/server/providers/rest"
)

// NewRESTServer creates a new REST server instance with the provided options.
// It implements the RESTServer interface and provides HTTP endpoints with streaming support.
//
// This factory function creates the REST server implementation directly.
func NewRESTServer(opts ...iface.Option) (RESTServer, error) {
	return rest.NewServer(opts...)
}

// NewMCPServer creates a new MCP server instance with the provided options.
// It implements the MCPServer interface and provides MCP protocol support for tools and resources.
//
// This factory function creates the MCP server implementation directly.
func NewMCPServer(opts ...iface.Option) (MCPServer, error) {
	return mcp.NewServer(opts...)
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
