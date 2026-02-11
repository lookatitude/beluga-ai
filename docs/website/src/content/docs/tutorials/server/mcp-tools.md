---
title: Building an MCP Tool Server
description: Expose Beluga AI tools to external agents and IDEs via the Model Context Protocol using Streamable HTTP transport.
---

The Model Context Protocol (MCP) is an open standard for how AI models interact with external tools and context. Instead of building custom integrations for every platform, build one MCP server that makes your Go tools available to Claude Desktop, Cursor, and any MCP-compatible client. This approach follows the same principle as Beluga AI's registry pattern -- define capabilities once, expose them through a standard interface, and let clients discover them dynamically.

## What You Will Build

An MCP server that exposes Beluga `Tool` implementations, MCP resources, and prompt templates via Streamable HTTP transport. You will register tools, add resources, and configure the server for IDE integration.

## Prerequisites

- Familiarity with the `tool` package and `FuncTool`
- Basic understanding of JSON-RPC 2.0

## Core Concepts

### MCP Server

The `protocol/mcp` package provides `MCPServer`, which processes JSON-RPC 2.0 requests at a single HTTP endpoint. JSON-RPC was chosen because the MCP specification requires it -- using a single endpoint with method-based dispatch simplifies deployment and firewall configuration compared to REST-style multiple endpoints. The server exposes three capability types:

- **Tools** -- Callable functions the model can invoke. These map directly to Beluga's `tool.Tool` interface, so any tool you build for agent use is automatically MCP-compatible.
- **Resources** -- Read-only context data such as database schemas, API documentation, or configuration files. Resources give models access to reference material without tool execution.
- **Prompts** -- Reusable prompt templates with parameterized arguments. Clients can list available prompts and fill in arguments, enabling consistent prompt engineering across tools.

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp"

server := mcp.NewServer("my-tools", "1.0.0")
```

## Step 1: Define Tools

Create tools using the `tool.FuncTool` pattern with typed input structs. The `FuncTool` uses Go's reflection to automatically generate a JSON Schema from the struct's field tags (`json`, `description`, `required`). This schema is returned to MCP clients via the `tools/list` method, enabling models to understand the tool's input format without manual schema authoring. The typed input struct approach eliminates runtime type assertion errors because the framework handles deserialization before calling your function.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/protocol/mcp"
    "github.com/lookatitude/beluga-ai/tool"
)

type WeatherInput struct {
    City    string `json:"city" description:"City name" required:"true"`
    Country string `json:"country" description:"Country code (e.g., US, UK)" required:"false"`
}

func main() {
    weatherTool := tool.NewFuncTool(
        "get_weather",
        "Get the current weather for a city",
        func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
            // In production, call a weather API.
            return tool.TextResult(fmt.Sprintf(
                "Weather in %s: 22C, partly cloudy", input.City,
            )), nil
        },
    )

    calculatorTool := tool.NewFuncTool(
        "calculate",
        "Evaluate a math expression",
        func(ctx context.Context, input struct {
            Expression string `json:"expression" description:"Math expression to evaluate" required:"true"`
        }) (*tool.Result, error) {
            return tool.TextResult("Result: 42"), nil
        },
    )

    // Continue to Step 2...
    _ = weatherTool
    _ = calculatorTool
}
```

## Step 2: Create the MCP Server

Register tools with the MCP server. The `AddTool` method accepts any value implementing the `tool.Tool` interface, which means tools built for agent use work without modification. The server internally maps each tool's `Name()` to its `Schema()` for the `tools/list` response and delegates to `Execute()` for `tools/call` requests.

```go
func buildServer() *mcp.MCPServer {
    server := mcp.NewServer("beluga-tools", "1.0.0")

    weatherTool := tool.NewFuncTool(
        "get_weather",
        "Get the current weather for a city",
        func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
            return tool.TextResult(fmt.Sprintf(
                "Weather in %s: 22C, partly cloudy", input.City,
            )), nil
        },
    )

    calculatorTool := tool.NewFuncTool(
        "calculate",
        "Evaluate a math expression",
        func(ctx context.Context, input struct {
            Expression string `json:"expression" description:"Math expression" required:"true"`
        }) (*tool.Result, error) {
            return tool.TextResult("42"), nil
        },
    )

    server.AddTool(weatherTool)
    server.AddTool(calculatorTool)

    return server
}
```

## Step 3: Add Resources

Resources provide read-only context to the model. Unlike tools, resources are not callable -- they are static data that the client can fetch to augment the model's context window. This is useful for providing database schemas, API documentation, or configuration details that help the model make better tool calls. Resources use URIs for identification, following the MCP specification's resource addressing scheme.

```go
func addResources(server *mcp.MCPServer) {
    server.AddResource(mcp.Resource{
        URI:         "resource://db-schema",
        Name:        "Database Schema",
        Description: "The current database schema for the application",
        MimeType:    "text/plain",
    })

    server.AddResource(mcp.Resource{
        URI:         "resource://api-docs",
        Name:        "API Documentation",
        Description: "REST API endpoint documentation",
        MimeType:    "text/markdown",
    })
}
```

## Step 4: Add Prompt Templates

Register reusable prompt templates that clients can use. Prompt templates enable consistent prompt engineering across different clients -- rather than each client constructing its own prompt, it can list available templates and fill in the arguments. The `Required` field on arguments tells the client which parameters must be provided, paralleling the `required` tag on `FuncTool` input structs.

```go
func addPrompts(server *mcp.MCPServer) {
    server.AddPrompt(mcp.Prompt{
        Name:        "analyze-data",
        Description: "Analyze a dataset and provide insights",
        Arguments: []mcp.PromptArgument{
            {Name: "dataset", Description: "Name of the dataset to analyze", Required: true},
            {Name: "format", Description: "Output format (summary, detailed, csv)", Required: false},
        },
    })
}
```

## Step 5: Serve via HTTP

Start the MCP server on an HTTP endpoint. The `Serve` method binds to the given address and handles the MCP protocol lifecycle, including capability negotiation during the `initialize` handshake. The server uses Streamable HTTP transport (protocol version `2025-03-26`), which supports both request-response and server-initiated events over a single HTTP connection.

```go
func main() {
    server := buildServer()
    addResources(server)
    addPrompts(server)

    ctx := context.Background()
    addr := ":3000"
    fmt.Printf("MCP server listening on %s\n", addr)

    if err := server.Serve(ctx, addr); err != nil {
        fmt.Printf("server error: %v\n", err)
        os.Exit(1)
    }
}
```

Alternatively, use the server as an `http.Handler` with your own HTTP server or router. This is useful when you want to add custom middleware (authentication, CORS, rate limiting) or serve additional endpoints alongside the MCP endpoint.

```go
func main() {
    server := buildServer()

    mux := http.NewServeMux()
    mux.Handle("/mcp", server.Handler())
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    http.ListenAndServe(":3000", mux)
}
```

## Step 6: Configure Claude Desktop or Cursor

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
    "mcpServers": {
        "beluga-tools": {
            "url": "http://localhost:3000/mcp"
        }
    }
}
```

### Cursor

Add to your Cursor MCP configuration:

```json
{
    "mcpServers": {
        "beluga-tools": {
            "url": "http://localhost:3000/mcp"
        }
    }
}
```

## MCP Protocol Details

The server handles these JSON-RPC 2.0 methods:

| Method | Description |
|--------|-------------|
| `initialize` | Exchanges capabilities between client and server |
| `tools/list` | Lists all registered tools with their schemas |
| `tools/call` | Executes a tool with given arguments |
| `resources/list` | Lists all registered resources |
| `prompts/list` | Lists all registered prompt templates |

The protocol version is `2025-03-26` (Streamable HTTP transport).

## Verification

1. Build and run the MCP server.
2. Send an `initialize` request to confirm the server responds with capabilities.
3. Send a `tools/list` request. Verify your tools appear with correct schemas.
4. Send a `tools/call` request for `get_weather`. Verify the tool executes and returns results.
5. Configure Claude Desktop or Cursor and verify the tools appear in the AI interface.

## Next Steps

- [REST Deployment](/tutorials/server/rest-deployment) -- Alternative REST API deployment for direct client access
- [Content Moderation](/tutorials/safety/content-moderation) -- Add safety guards to tool execution
