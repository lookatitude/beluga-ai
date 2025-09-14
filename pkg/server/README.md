# Server Package

The `server` package provides HTTP API endpoints and MCP (Model Context Protocol) server implementation for the Beluga AI framework. It supports both REST APIs with and without streaming, as well as the Model Context Protocol (MCP) for tool and resource integration.

## Features

- **REST Server**: HTTP API server with streaming support
- **MCP Server**: Model Context Protocol server for AI tool integration
- **Streaming Support**: Real-time streaming responses for long-running operations
- **Observability**: OpenTelemetry integration for metrics, tracing, and structured logging
- **Middleware Support**: Extensible middleware system for authentication, rate limiting, etc.
- **Configuration Management**: Flexible configuration with functional options
- **Error Handling**: Structured error responses with proper HTTP status codes
- **Health Checks**: Built-in health check endpoints

## Quick Start

### REST Server

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/server"
    "github.com/lookatitude/beluga-ai/pkg/server/providers"
)

func main() {
    // Create a REST server
    restProvider, err := providers.NewRESTProvider(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: "localhost",
                Port: 8080,
            },
            APIBasePath: "/api/v1",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Register an agent handler
    agentHandler := &MyAgentHandler{}
    restProvider.RegisterAgentHandler("my-agent", agentHandler)

    // Start the server
    ctx := context.Background()
    if err := restProvider.Start(ctx); err != nil {
        log.Fatal(err)
    }
}

type MyAgentHandler struct{}

func (h *MyAgentHandler) Execute(ctx context.Context, request interface{}) (interface{}, error) {
    // Implement your agent logic here
    return map[string]interface{}{
        "result": "Hello from agent!",
        "timestamp": time.Now(),
    }, nil
}

func (h *MyAgentHandler) GetStatus(ctx context.Context, id string) (interface{}, error) {
    // Implement status checking logic
    return map[string]interface{}{
        "status": "running",
        "id": id,
    }, nil
}
```

### MCP Server

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/server"
    "github.com/lookatitude/beluga-ai/pkg/server/providers"
)

func main() {
    // Create an MCP server
    mcpProvider, err := providers.NewMCPProvider(
        server.WithMCPConfig(server.MCPConfig{
            Config: server.Config{
                Host: "localhost",
                Port: 8081,
            },
            ServerName: "my-mcp-server",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Register tools and resources
    calculator := providers.NewCalculatorTool()
    mcpProvider.RegisterTool(calculator)

    fileResource := providers.NewFileResource(
        "config",
        "Application configuration",
        "/etc/app/config.json",
        "application/json",
    )
    mcpProvider.RegisterResource(fileResource)

    // Start the server
    ctx := context.Background()
    if err := mcpProvider.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Architecture

### Package Structure

```
pkg/server/
├── providers/      # Provider implementations with examples
├── config.go       # Configuration structures
├── metrics.go      # Metrics definitions
├── errors.go       # Error types and handling
├── server.go       # Main package API with interfaces and factory functions
└── server_test.go  # Comprehensive tests

# Related packages:
├── pkg/restserver/  # REST server implementation
└── pkg/mcpserver/   # MCP server implementation
```

### Interfaces

#### Server Interface

```go
type Server interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsHealthy(ctx context.Context) bool
}
```

#### REST Server Interface

```go
type RESTServer interface {
    Server
    RegisterHandler(method, path string, handler http.HandlerFunc)
    RegisterMiddleware(middleware func(http.Handler) http.Handler)
    GetMux() interface{}
}
```

#### MCP Server Interface

```go
type MCPServer interface {
    Server
    RegisterTool(tool MCPTool) error
    RegisterResource(resource MCPResource) error
    ListTools(ctx context.Context) ([]MCPTool, error)
    ListResources(ctx context.Context) ([]MCPResource, error)
    CallTool(ctx context.Context, name string, input map[string]interface{}) (interface{}, error)
}
```

## Configuration

### REST Configuration

```go
type RESTConfig struct {
    Config
    APIBasePath     string        // Base path for API endpoints (default: "/api/v1")
    EnableStreaming bool          // Enable streaming responses (default: true)
    MaxRequestSize  int64         // Maximum request body size (default: 10MB)
    RateLimitRequests int         // Requests per minute (default: 1000)
    EnableRateLimit bool          // Enable rate limiting (default: true)
}
```

### MCP Configuration

```go
type MCPConfig struct {
    Config
    ServerName         string        // MCP server name (required)
    ServerVersion      string        // Server version (default: "1.0.0")
    ProtocolVersion    string        // MCP protocol version (default: "2024-11-05")
    MaxConcurrentRequests int        // Max concurrent requests (default: 10)
    RequestTimeout     time.Duration // Request timeout (default: 60s)
}
```

### Functional Options

The package uses functional options for flexible configuration:

```go
server, err := server.NewRESTServer(
    server.WithRESTConfig(restConfig),
    server.WithLogger(logger),
    server.WithTracer(tracer),
    server.WithMeter(meter),
    server.WithMiddleware(server.CORSMiddleware([]string{"*"})),
)
```

## REST API Endpoints

### Agent Endpoints

- `POST /api/v1/agents/{name}/execute` - Execute an agent
- `GET /api/v1/agents/{name}/status` - Get agent status
- `GET /api/v1/agents` - List all agents

### Chain Endpoints

- `POST /api/v1/chains/{name}/execute` - Execute a chain
- `GET /api/v1/chains/{name}/status` - Get chain status
- `GET /api/v1/chains` - List all chains

### Streaming Endpoints

- `GET /api/v1/agents/{name}/stream` - Stream agent execution
- `POST /api/v1/chains/{name}/stream` - Stream chain execution

### Health Check

- `GET /health` - Server health check

## MCP Protocol Support

The MCP server implements the Model Context Protocol for AI tool integration:

### Supported Methods

- `initialize` - Initialize the MCP connection
- `tools/list` - List available tools
- `tools/call` - Execute a tool
- `resources/list` - List available resources
- `resources/read` - Read a resource

### Tool Interface

```go
type MCPTool interface {
    Name() string
    Description() string
    InputSchema() map[string]interface{}
    Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}
```

### Resource Interface

```go
type MCPResource interface {
    URI() string
    Name() string
    Description() string
    MimeType() string
    Read(ctx context.Context) ([]byte, error)
}
```

## Observability

### Metrics

The package provides comprehensive metrics using OpenTelemetry:

- HTTP request metrics (count, duration, status codes)
- MCP tool call metrics (count, duration, success rate)
- MCP resource read metrics
- Server health and uptime metrics

### Tracing

All operations are traced with OpenTelemetry:

- HTTP request tracing
- MCP tool execution tracing
- Resource read tracing
- Error tracing with proper status codes

### Logging

Structured logging with configurable levels:

- Request/response logging
- Error logging with context
- Performance logging
- Health check logging

## Middleware

### Built-in Middleware

- **CORS Middleware**: Handles cross-origin requests
- **Rate Limiting**: Prevents abuse with configurable limits
- **Logging**: Request/response logging
- **Metrics**: Automatic metrics collection
- **Tracing**: Distributed tracing support
- **Recovery**: Panic recovery with proper error responses

### Custom Middleware

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Authentication logic
        if !isAuthenticated(r) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Register middleware
server.RegisterMiddleware(AuthMiddleware)
```

## Error Handling

The package provides structured error handling:

```go
// Custom error types
err := server.NewInvalidRequestError("operation", "message", details)
err := server.NewToolNotFoundError("tool_name")
err := server.NewInternalError("operation", underlyingError)

// HTTP status mapping
statusCode := err.HTTPStatus() // Returns appropriate HTTP status code
```

## Examples

### Complete REST Server Example

See `providers/rest.go` for a complete example of REST server usage with agent and chain handlers.

### Complete MCP Server Example

See `providers/mcp.go` for examples of MCP tools and resources including:
- Calculator tool
- File resource
- Text resource

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/server/...

# Run with coverage
go test -cover ./pkg/server/...

# Run benchmarks
go test -bench=. ./pkg/server/...
```

### Test Coverage

- Unit tests for all major components
- Integration tests for server startup/shutdown
- Benchmark tests for performance
- Mock implementations for testing
- Error handling tests

## Dependencies

- `github.com/gorilla/mux` - HTTP router
- `go.opentelemetry.io/otel` - Observability
- Standard library HTTP packages

## Contributing

1. Follow the package design patterns from the framework guidelines
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Ensure backward compatibility for interface changes

## License

This package is part of the Beluga AI Framework and follows the same license terms.
