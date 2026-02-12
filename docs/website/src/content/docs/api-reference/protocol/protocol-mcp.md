---
title: "MCP API — Server, Client, SDK, Registry"
description: "MCP protocol API reference for Beluga AI. Model Context Protocol server/client, tool sharing, Composio integration, and official SDK bridge."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "MCP API, Model Context Protocol, JSON-RPC, tool sharing, MCP server, MCP client, Composio, Beluga AI, Go, reference"
---

## mcp

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp"
```

Package mcp implements the Model Context Protocol (MCP) for tool and resource
sharing between AI systems. It supports the Streamable HTTP transport per the
March 2025 MCP specification, using a single HTTP endpoint for JSON-RPC 2.0
messages.

MCP enables AI applications to expose and consume tools, resources, and prompt
templates across process and network boundaries. The protocol uses JSON-RPC 2.0
as its message format, with methods including initialize, tools/list, tools/call,
resources/list, and prompts/list.

## Server

MCPServer exposes Beluga tools, resources, and prompts to MCP clients. Tools
registered with the server are automatically converted to MCP tool listings
and can be invoked remotely via JSON-RPC.

```go
srv := mcp.NewServer("my-server", "1.0.0")
srv.AddTool(myTool)
srv.AddResource(mcp.Resource{URI: "file:///data.json", Name: "data"})
srv.Serve(ctx, ":8080")
```

The server can also be used as an http.Handler for integration with existing
HTTP servers:

```go
http.Handle("/mcp", srv.Handler())
```

## Client

MCPClient connects to a remote MCP server and provides methods for
initialization, tool listing, and tool invocation.

```go
client := mcp.NewClient("http://localhost:8080/mcp")
caps, err := client.Initialize(ctx)
tools, err := client.ListTools(ctx)
result, err := client.CallTool(ctx, "search", map[string]any{"query": "hello"})
```

## Bridge Function

FromMCP connects to an MCP server and returns its tools as native tool.Tool
instances, enabling seamless integration of remote MCP tools into Beluga agents:

```go
tools, err := mcp.FromMCP(ctx, "http://localhost:8080/mcp")
agent := agent.New("assistant", agent.WithTools(tools...))
```

## Key Types

- MCPServer — serves Beluga tools/resources/prompts via MCP
- MCPClient — connects to remote MCP servers
- Request / Response — JSON-RPC 2.0 message types
- ToolInfo / ToolCallParams / ToolCallResult — tool operation types
- Resource / Prompt — MCP resource and prompt template types
- ServerCapabilities — describes server feature support

---

## composio

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp/providers/composio"
```

Package composio provides a Composio MCP integration for the Beluga AI
protocol layer. It connects to the Composio API for tool discovery and
execution, wrapping Composio tools as native tool.Tool instances.

Composio provides access to hundreds of integrations and actions through
its unified API, which can be consumed as MCP-compatible tools. Each
Composio action is wrapped as a Beluga tool.Tool, enabling seamless
integration with Beluga agents.

## Usage

```go
client, err := composio.New(
    composio.WithAPIKey("cmp-..."),
)
if err != nil {
    log.Fatal(err)
}

tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

// Use Composio tools with a Beluga agent
myAgent := agent.New("assistant", agent.WithTools(tools...))
```

## Configuration

The client is configured via functional options:

- WithAPIKey(key) — sets the Composio API key (required)
- WithBaseURL(url) — overrides the default API base URL
- WithTimeout(d) — sets the HTTP client timeout (default: 30s)

## Key Types

- Client — connects to the Composio API for tool operations
- Option — functional option for configuring the Client

---

## registry

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp/registry"
```

Package registry provides MCP server discovery and tool aggregation for the
Beluga AI framework. It discovers MCP servers from registry endpoints or
static configuration, lists their available tools, and makes them accessible
as native tool.Tool instances.

The registry maintains a list of known MCP servers and provides methods to
connect to them, initialize MCP sessions, and aggregate their tools into a
unified collection. It is safe for concurrent use.

## Basic Usage

```go
reg := registry.New()
reg.AddServer("search", "http://localhost:8081/mcp")
reg.AddServer("code", "http://localhost:8082/mcp")

tools, err := reg.DiscoverTools(ctx)
// tools contains all tools from all registered MCP servers
```

## Server Management

Servers can be added, removed, and filtered by tags:

```go
reg.AddServer("search", "http://search:8080/mcp", "production", "search")
reg.AddServer("dev-tools", "http://dev:8080/mcp", "development")

prodServers := reg.ServersByTag("production")
reg.RemoveServer("dev-tools")
```

## Selective Discovery

Tools can be discovered from all servers or from a specific server:

```go
// All servers
allTools, err := reg.DiscoverTools(ctx)

// Specific server
searchTools, err := reg.DiscoverToolsFromServer(ctx, "search")
```

## Key Types

- Registry — manages MCP server discovery and tool aggregation
- ServerEntry — describes a registered MCP server (name, URL, tags)
- DiscoveredTool — wraps a tool.Tool with server provenance metadata
- MCPClientInterface — abstracts MCP client operations for testability

---

## sdk

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp/sdk"
```

Package sdk provides integration between the official MCP Go SDK
(github.com/modelcontextprotocol/go-sdk) and Beluga's MCP protocol layer.
It bridges Beluga's tool.Tool interface with the SDK's server and client,
enabling exposure of Beluga tools via the official MCP SDK and consumption
of remote MCP tools as native Beluga tools.

This package is useful when you need full compliance with the official MCP
SDK behavior, including support for advanced features like transport
negotiation and session management provided by the SDK.

## Server

NewServer creates an MCP server using the official SDK and registers Beluga
tools. Each tool.Tool is exposed as an MCP tool with its name, description,
and input schema.

```go
srv := sdk.NewServer("my-server", "1.0.0", searchTool, calcTool)
// srv is an *sdkmcp.Server from the official SDK
```

## Client

NewClient creates an MCP client using the official SDK and connects it to
a server via a transport. FromSession lists tools from the connected session
and returns them as native Beluga tool.Tool instances.

```go
client, session, err := sdk.NewClient(ctx, transport)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

tools, err := sdk.FromSession(ctx, session)
// tools are native tool.Tool instances backed by the remote MCP server
```

## Type Conversions

The package handles bidirectional conversion between Beluga and SDK types:
- tool.Tool to MCP SDK tool definitions (server side)
- MCP SDK CallToolResult to tool.Result (client side)
- schema.TextPart to/from sdkmcp.TextContent
