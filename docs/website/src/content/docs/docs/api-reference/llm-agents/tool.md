---
title: "Tool API — FuncTool, Registry, MCP Client"
description: "Tool package API reference for Beluga AI. Tool interface, generic FuncTool, thread-safe Registry, MCP client integration, middleware, and hooks."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "tool API, Tool interface, FuncTool, Registry, MCP client, middleware, hooks, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/tool"
```

Package tool provides the tool system for the Beluga AI framework.

It defines the `Tool` interface, a type-safe `FuncTool` wrapper using generics,
a thread-safe tool `Registry`, `Middleware` composition, lifecycle `Hooks`,
an `MCPClient` for connecting to remote MCP tool servers, and an `MCPRegistry`
for MCP server discovery.

## Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}
```

Tools have a name (used by the LLM to select them), a description (provided
to the LLM as context), a JSON Schema for input validation, and an `Execute`
method that performs the tool's action.

## FuncTool

`FuncTool[I]` wraps a typed Go function as a `Tool` using generics. It
automatically generates a JSON Schema from the input struct's field tags:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/tool"
)

type SearchInput struct {
    Query string `json:"query" description:"Search query" required:"true"`
    Limit int    `json:"limit" description:"Max results" default:"10"`
}

func main() {
    search := tool.NewFuncTool("search", "Search the web",
        func(ctx context.Context, input SearchInput) (*tool.Result, error) {
            results := doSearch(ctx, input.Query, input.Limit)
            return tool.TextResult(results), nil
        },
    )

    result, err := search.Execute(context.Background(), map[string]any{
        "query": "Go generics",
        "limit": 5,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Content)
}
```

The input struct supports `json`, `description`, `required`, and `default`
tags recognized by the internal `jsonutil.GenerateSchema` function.

`NewFuncTool[I any](name, description string, fn func(ctx context.Context, input I) (*Result, error)) *FuncTool[I]`

## Results

`Result` holds multimodal output from tool execution:

```go
type Result struct {
    Content []schema.ContentPart
    IsError bool
}
```

Convenience constructors:

```go
result := tool.TextResult("The answer is 42")
errResult := tool.ErrorResult(fmt.Errorf("not found"))
```

Use `ToDefinition` to convert a `Tool` to a `schema.ToolDefinition` for binding
to an LLM provider:

```go
def := tool.ToDefinition(myTool) // returns schema.ToolDefinition
```

## Registry

`Registry` is a thread-safe, name-based collection of tools:

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    reg := tool.NewRegistry()

    if err := reg.Add(search); err != nil {
        log.Fatal(err)
    }

    t, err := reg.Get("search")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(t.Name())

    names := reg.List()       // sorted tool names
    all := reg.All()          // sorted tool instances
    defs := reg.Definitions() // []map[string]any for each tool
}
```

`Registry` methods:

| Method | Signature | Description |
|---|---|---|
| `Add` | `Add(t Tool) error` | Register a tool. Errors if name already registered. |
| `Get` | `Get(name string) (Tool, error)` | Look up a tool by name. |
| `List` | `List() []string` | Sorted tool names. |
| `All` | `All() []Tool` | Sorted tool instances. |
| `Remove` | `Remove(name string) error` | Unregister a tool by name. |
| `Definitions` | `Definitions() []map[string]any` | Tool definitions as raw maps, sorted by name. |

## MCP Client

`MCPClient` connects to an MCP (Model Context Protocol) server using the
Streamable HTTP transport (March 2025 spec). It wraps remote tools as native
`Tool` instances.

The convenience function `FromMCP` connects, discovers tools, and returns
them along with the client for later cleanup:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    ctx := context.Background()

    tools, client, err := tool.FromMCP(ctx, "https://mcp.example.com/tools",
        tool.WithSessionID("session-1"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(ctx)

    for _, t := range tools {
        fmt.Println(t.Name())
    }
}
```

`FromMCP` returns `([]Tool, *MCPClient, error)`. Always call `client.Close(ctx)`
when done to send the DELETE session-termination request.

`MCPOption` values for `FromMCP` and `NewMCPClient`:

| Option | Description |
|---|---|
| `WithSessionID(id string)` | Set the `Mcp-Session-Id` header. |
| `WithMCPHeaders(headers map[string]string)` | Additional HTTP headers. |
| `WithHTTPClient(c *http.Client)` | Custom HTTP client (default: 30s timeout). |

For lower-level control, use `MCPClient` directly:

```go
client := tool.NewMCPClient("https://mcp.example.com", tool.WithSessionID("s1"))
if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer client.Close(ctx)

tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

result, err := client.ExecuteTool(ctx, "my-tool", map[string]any{"key": "value"})
if err != nil {
    log.Fatal(err)
}
```

Transport protocol: POST for requests, GET for notifications, DELETE for
session termination. `Mcp-Session-Id` header is used for session management.

## MCP Registry

`MCPRegistry` provides discovery of MCP servers. `StaticMCPRegistry` is backed
by a fixed list of servers:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    ctx := context.Background()

    registry := tool.NewStaticMCPRegistry(
        tool.MCPServerInfo{Name: "code-tools", URL: "https://mcp.example.com/code"},
        tool.MCPServerInfo{Name: "search-tools", URL: "https://mcp.example.com/search"},
    )

    // Case-insensitive substring search on server name
    servers, err := registry.Search(ctx, "code")
    if err != nil {
        log.Fatal(err)
    }
    for _, s := range servers {
        fmt.Printf("%s: %s\n", s.Name, s.URL)
    }

    all, err := registry.Discover(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%d servers\n", len(all))
}
```

`MCPRegistry` is an interface with `Search(ctx, query) ([]MCPServerInfo, error)`
and `Discover(ctx) ([]MCPServerInfo, error)`.

`MCPServerInfo` fields: `Name string`, `URL string`, `Tools []schema.ToolDefinition`,
`Transport string`.

## Middleware

`Middleware` wraps a `Tool` to add cross-cutting behavior. Built-in middleware:

- `WithTimeout(d time.Duration) Middleware` — cancels execution after `d` and returns `core.ErrTimeout`.
- `WithRetry(maxAttempts int) Middleware` — retries on retryable errors (via `core.IsRetryable`).

Applied via `ApplyMiddleware`. The first middleware in the list is the
outermost wrapper and executes first:

```go
package main

import (
    "time"

    "github.com/lookatitude/beluga-ai/tool"
)

func withResilience(t tool.Tool) tool.Tool {
    return tool.ApplyMiddleware(t,
        tool.WithTimeout(30*time.Second),
        tool.WithRetry(3),
    )
}
```

## Hooks

`Hooks` provides lifecycle callbacks around tool execution. All fields are
optional; nil hooks are skipped. Compose multiple `Hooks` values with
`ComposeHooks`, or wrap a tool with `WithHooks`:

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/tool"
)

func withLogging(t tool.Tool) tool.Tool {
    hooks := tool.Hooks{
        BeforeExecute: func(ctx context.Context, name string, input map[string]any) error {
            log.Printf("executing tool: %s", name)
            return nil
        },
        AfterExecute: func(ctx context.Context, name string, result *tool.Result, err error) {
            log.Printf("tool %s finished: err=%v", name, err)
        },
    }
    return tool.WithHooks(t, hooks)
}
```

`Hooks` fields:

| Field | Signature | Description |
|---|---|---|
| `BeforeExecute` | `func(ctx, name string, input map[string]any) error` | Called before Execute. Returning an error aborts execution. |
| `AfterExecute` | `func(ctx, name string, result *Result, err error)` | Called after Execute (success or failure). |
| `OnError` | `func(ctx, name string, err error) error` | Called on error. Returning nil suppresses the error. |

## Related

- [`agent`](/docs/api-reference/llm-agents/agent) — Agent uses tools via WithTools and handoffs
- [`llm`](/docs/api-reference/llm-agents/llm) — BindTools attaches tool definitions to a ChatModel
- [`core`](/docs/api-reference/foundation/core) — IsRetryable, typed errors used by middleware
