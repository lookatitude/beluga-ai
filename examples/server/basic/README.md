# Server Basic Example

This example demonstrates how to use the Server package for creating REST and MCP servers.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating REST server configuration
2. Creating a REST server instance
3. Creating MCP server configuration
4. Creating an MCP server instance

## REST Server Configuration

- `Host`: Server hostname (default: "localhost")
- `Port`: Server port (default: 8080)
- `APIBasePath`: Base path for API endpoints
- `CORS`: CORS configuration
- `Timeouts`: Read, write, and idle timeouts

## MCP Server Configuration

- `Host`: Server hostname (default: "localhost")
- `Port`: Server port (default: 8081)
- `ServerName`: MCP server name
- `ProtocolVersion`: MCP protocol version
- `MaxConcurrentRequests`: Maximum concurrent requests

## Starting Servers

To start servers in production:

```go
// Start REST server
go func() {
    if err := restServer.Start(ctx); err != nil {
        log.Fatal(err)
    }
}()

// Start MCP server
go func() {
    if err := mcpServer.Start(ctx); err != nil {
        log.Fatal(err)
    }
}()
```

## See Also

- [Server Package Documentation](../../../pkg/server/README.md)
- [Integration Examples](../../integration/full_stack/main.go)
