---
title: "MCP Providers"
description: "Overview of MCP protocol support, tool discovery, and provider integrations in Beluga AI v2."
---

Beluga AI v2 provides full support for the Model Context Protocol (MCP), enabling agents to discover and use tools from remote MCP servers. The implementation includes an MCP client and server, a server discovery registry, an official Go SDK bridge, and provider integrations for tool platforms.

## Architecture

MCP in Beluga AI spans two packages:

- **`tool/`** -- Tool interface, local registry, and MCP client stub
- **`protocol/mcp/`** -- Full MCP client, server, registry discovery, SDK bridge, and providers

```
┌─────────────┐     JSON-RPC 2.0      ┌─────────────┐
│  MCP Client │ ───────────────────>   │  MCP Server │
│  (Beluga)   │ <───────────────────   │  (Remote)   │
└─────────────┘                        └─────────────┘
       │
       ▼
┌─────────────┐
│ tool.Tool    │  ← Remote tools wrapped as native Tool interface
└─────────────┘
```

## MCP Client

Connect to any MCP server and use its tools as native `tool.Tool` instances:

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp"

// Discover and wrap all tools from an MCP server
tools, err := mcp.FromMCP(ctx, "https://mcp-server.example.com/mcp")
if err != nil {
    log.Fatal(err)
}

for _, t := range tools {
    fmt.Printf("Tool: %s — %s\n", t.Name(), t.Description())
}

// Execute a tool
result, err := tools[0].Execute(ctx, map[string]any{
    "query": "example input",
})
if err != nil {
    log.Fatal(err)
}
```

### Client Methods

The `MCPClient` provides low-level access to the MCP protocol:

```go
client := mcp.NewClient("https://mcp-server.example.com/mcp")

// Initialize the session and discover server capabilities
caps, err := client.Initialize(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Server: %s v%s\n", caps.Tools != nil, caps.Resources != nil)

// List available tools
tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

// Call a specific tool
result, err := client.CallTool(ctx, "get_weather", map[string]any{
    "location": "Paris",
})
if err != nil {
    log.Fatal(err)
}
```

## MCP Server

Expose Beluga tools as an MCP server over HTTP:

```go
import (
    "github.com/lookatitude/beluga-ai/protocol/mcp"
    "github.com/lookatitude/beluga-ai/tool"
)

weatherTool := tool.NewFuncTool("get_weather", "Get current weather",
    func(ctx context.Context, input struct {
        Location string `json:"location" description:"City name"`
    }) (*tool.Result, error) {
        return tool.TextResult(fmt.Sprintf("Sunny in %s", input.Location)), nil
    },
)

server := mcp.NewServer("my-server", "1.0.0")
server.AddTool(weatherTool)

err := server.Serve(ctx, ":8080")
if err != nil {
    log.Fatal(err)
}
```

The server implements JSON-RPC 2.0 over HTTP POST and handles:

- `initialize` -- Returns server capabilities and protocol version
- `tools/list` -- Lists all registered tools with their schemas
- `tools/call` -- Executes a tool and returns the result
- `resources/list` -- Lists registered resources
- `prompts/list` -- Lists registered prompts

## Server Discovery Registry

The MCP registry discovers and aggregates tools from multiple MCP servers:

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp/registry"

reg := registry.New()
reg.AddServer("weather", "https://weather-mcp.example.com/mcp", "weather", "utilities")
reg.AddServer("search", "https://search-mcp.example.com/mcp", "search", "rag")

// Discover all tools across all servers
discovered, err := reg.DiscoverTools(ctx)
if err != nil {
    log.Fatal(err)
}

for _, d := range discovered {
    fmt.Printf("[%s] %s: %s\n", d.ServerName, d.Tool.Name, d.Tool.Description)
}

// Filter servers by tag
weatherServers := reg.ServersByTag("weather")

// Get all tools as a flat slice (ready for agent use)
tools, err := reg.Tools(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Official SDK Bridge

Bridge between Beluga tools and the official MCP Go SDK:

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp/sdk"

// Create an SDK-native MCP server from Beluga tools
sdkServer := sdk.NewServer("my-server", "1.0.0", weatherTool, searchTool)

// Create a client from an SDK transport and get Beluga tools
client, session, err := sdk.NewClient(ctx, transport)
if err != nil {
    log.Fatal(err)
}

tools, err := sdk.FromSession(ctx, session)
if err != nil {
    log.Fatal(err)
}
```

## Tool Interface

All tools from MCP servers are wrapped as native `tool.Tool` instances:

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}
```

This means MCP-sourced tools work identically with agents, middleware, hooks, and all other Beluga components that accept `tool.Tool`.

## Static MCP Registry

For environments where MCP servers are known at compile time, use the static registry in the `tool` package:

```go
import "github.com/lookatitude/beluga-ai/tool"

reg := tool.NewStaticMCPRegistry(
    tool.MCPServerInfo{
        Name: "weather",
        URL:  "https://weather-mcp.example.com/mcp",
    },
    tool.MCPServerInfo{
        Name: "search",
        URL:  "https://search-mcp.example.com/mcp",
    },
)

// Search for servers by name or URL
servers, err := reg.Search(ctx, "weather")
if err != nil {
    log.Fatal(err)
}

// Discover all registered servers
all, err := reg.Discover(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Available Providers

| Provider | Description |
|---|---|
| [Composio](/providers/mcp/composio) | Access 250+ SaaS integrations as MCP tools via the Composio API |
