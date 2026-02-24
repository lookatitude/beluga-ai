---
title: MCP Server & Client Integration
description: "Expose Beluga AI tools over Model Context Protocol for IDE clients and bots, or consume remote MCP servers as native tools."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "MCP integration, Model Context Protocol, MCP server Go, MCP client, Beluga AI tools, tool interoperability, IDE integration"
---

The Model Context Protocol (MCP) is an open standard for tool interoperability across AI systems. By exposing your tools over MCP, any compatible client -- IDE extensions like Cursor and Claude Code, chatbots, or other agent frameworks -- can discover and invoke them without custom integration code. Conversely, by consuming remote MCP servers, your Beluga agents gain access to tools hosted by other teams or services.

This is particularly valuable in organizations where tool development is distributed across teams. A data engineering team can publish database tools via MCP, and any agent framework in the company can consume them.

## Overview

This integration covers two directions:

1. **MCP Server** — expose your Beluga tools so external MCP clients can call them
2. **MCP Client** — connect to remote MCP servers and use their tools as native Beluga tools

The MCP implementation uses the Streamable HTTP transport (March 2025 spec) with JSON-RPC 2.0 over HTTP.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`go get github.com/lookatitude/beluga-ai`)
- For testing: an MCP client (Cursor, Claude Code, MCP CLI, or `curl`)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Exposing Tools via MCP Server

Create an MCP server that exposes your Beluga tools to external clients.

### Define Tools

Start by creating tools using `FuncTool` or a manual `Tool` implementation.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/protocol/mcp"
    "github.com/lookatitude/beluga-ai/tool"
)

type EchoInput struct {
    Message string `json:"message" description:"The message to echo" required:"true"`
}

type CalcInput struct {
    Expression string `json:"expression" description:"Math expression to evaluate" required:"true"`
}

func main() {
    ctx := context.Background()

    echo := tool.NewFuncTool("echo", "Echoes the input message",
        func(ctx context.Context, input EchoInput) (*tool.Result, error) {
            return tool.TextResult(fmt.Sprintf("Echo: %s", input.Message)), nil
        },
    )

    calc := tool.NewFuncTool("calculate", "Evaluates a math expression",
        func(ctx context.Context, input CalcInput) (*tool.Result, error) {
            // In production, use a safe expression evaluator
            return tool.TextResult(fmt.Sprintf("Result of %s = 42", input.Expression)), nil
        },
    )

    // Create MCP server and register tools
    server := mcp.NewServer("beluga-tools", "1.0.0")
    server.AddTool(echo)
    server.AddTool(calc)

    // Start the server
    log.Println("MCP server listening on :8081")
    if err := server.Serve(ctx, ":8081"); err != nil {
        log.Fatalf("mcp serve: %v", err)
    }
}
```

### Verify with curl

Once the server is running, test tool discovery with a JSON-RPC request:

```bash
curl -X POST http://localhost:8081 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Invoke a tool:

```bash
curl -X POST http://localhost:8081 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"message":"hello"}}}'
```

## Consuming Remote MCP Tools

Use the MCP client to connect to a remote MCP server and wrap its tools as native Beluga `tool.Tool` instances.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    ctx := context.Background()

    // Connect to an MCP server
    client := tool.NewMCPClient("http://localhost:8081",
        tool.WithMCPHeaders(map[string]string{
            "Authorization": "Bearer my-token",
        }),
    )

    if err := client.Connect(ctx); err != nil {
        log.Fatalf("mcp connect: %v", err)
    }

    // List available remote tools
    remoteTools, err := client.ListTools(ctx)
    if err != nil {
        log.Fatalf("list tools: %v", err)
    }

    fmt.Printf("Discovered %d remote tools\n", len(remoteTools))

    // Use remote tools in an agent
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: "your-api-key",
        Model:  "gpt-4o",
    })
    if err != nil {
        log.Fatalf("llm: %v", err)
    }

    a := agent.New("mcp-agent",
        agent.WithLLM(model),
        agent.WithTools(remoteTools),
    )

    result, err := a.Invoke(ctx, "Echo the message 'hello world'")
    if err != nil {
        log.Fatalf("invoke: %v", err)
    }
    fmt.Println(result)
}
```

## Exposing Agent Tools via MCP

Combine an agent's full tool set — including handoff tools — with an MCP server to let external clients access everything the agent can do.

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/protocol/mcp"
    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    ctx := context.Background()

    // Build agent with tools
    a := agent.New("assistant",
        agent.WithTools([]tool.Tool{myTool1, myTool2}),
        // ... other agent options
    )

    // Expose the agent's tools over MCP
    server := mcp.NewServer("agent-tools", "1.0.0")
    for _, t := range a.Tools() {
        server.AddTool(t)
    }

    log.Println("MCP server listening on :8081")
    if err := server.Serve(ctx, ":8081"); err != nil {
        log.Fatalf("mcp serve: %v", err)
    }
}
```

## Advanced Topics

### Using the MCP Server as an HTTP Handler

Embed the MCP endpoint into an existing HTTP server instead of running standalone.

```go
server := mcp.NewServer("beluga-tools", "1.0.0")
server.AddTool(myTool)

mux := http.NewServeMux()
mux.Handle("/mcp", server.Handler())
mux.HandleFunc("/health", healthHandler)

log.Fatal(http.ListenAndServe(":8080", mux))
```

### Session Management

MCP supports session-based connections via the `Mcp-Session-Id` header. When consuming remote tools, pass the session ID to maintain state across requests.

```go
client := tool.NewMCPClient("http://localhost:8081",
    tool.WithSessionID("session-abc-123"),
)
```

### Stream Resumability

Use `Last-Event-ID` to resume interrupted server-sent event streams.

```go
client := tool.NewMCPClient("http://localhost:8081",
    tool.WithLastEventID("evt-42"),
)
```

### Combining Local and Remote Tools

Merge tools from multiple sources — local tools, MCP remote tools, and registry tools — into a single agent.

```go
var allTools []tool.Tool

// Local tools
allTools = append(allTools, localTool1, localTool2)

// Remote MCP tools
remoteTools, err := mcpClient.ListTools(ctx)
if err != nil {
    log.Fatalf("list remote tools: %v", err)
}
allTools = append(allTools, remoteTools...)

// Registry tools
for _, name := range reg.List() {
    t, err := reg.Get(name)
    if err != nil {
        log.Fatalf("get tool %s: %v", name, err)
    }
    allTools = append(allTools, t)
}

a := agent.New("multi-source-agent",
    agent.WithLLM(model),
    agent.WithTools(allTools),
)
```

## Configuration Reference

| Component | API | Description |
|-----------|-----|-------------|
| `mcp.NewServer` | `NewServer(name, version string)` | Create an MCP server |
| `MCPServer.AddTool` | `AddTool(t tool.Tool)` | Register a tool with the server |
| `MCPServer.Serve` | `Serve(ctx, addr string) error` | Start the server on a TCP address |
| `MCPServer.Handler` | `Handler() http.Handler` | Get an HTTP handler for embedding |
| `tool.NewMCPClient` | `NewMCPClient(url string, opts...)` | Create an MCP client |
| `MCPClient.Connect` | `Connect(ctx) error` | Establish a session |
| `MCPClient.ListTools` | `ListTools(ctx) ([]Tool, error)` | Discover remote tools |

## Troubleshooting

### MCP server does not start

**Problem**: Port already in use or configuration error.

**Solution**: Check that no other process is bound to the port. Verify the address format (e.g., `:8081` or `localhost:8081`).

### Client does not see tools

**Problem**: Tools not registered before calling `Serve`.

**Solution**: Register all tools with `server.AddTool()` before starting the server. Check server logs for registration errors.

### Tool call fails from client

**Problem**: Input format or schema mismatch between client and server.

**Solution**: Ensure the client sends arguments matching the tool's `InputSchema`. Use `tools/list` to inspect the schema the server advertises. Validate and log tool inputs on the server side.

### Authentication errors

**Problem**: Client receives 401/403 when connecting.

**Solution**: Pass the required authentication headers via `tool.WithMCPHeaders()`. When running the MCP server in production, add TLS and token validation middleware to the HTTP handler.

## Related Resources

- [Custom Tools and Tool Registry](/integrations/agents-tools-registry) — Building and registering tools
- [LLM Providers](/integrations/llm-providers) — Supported LLM providers for agents
- [Tool System API Reference](/api-reference/tool) — Complete tool package documentation
