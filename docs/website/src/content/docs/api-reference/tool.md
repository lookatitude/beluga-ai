---
title: Tool Package API
description: API documentation for the tool system and MCP client.
---

```go
import "github.com/lookatitude/beluga-ai/tool"
```

Package tool provides the `Tool` interface, type-safe `FuncTool` wrapper, thread-safe registry, middleware, hooks, and MCP client for remote tools.

## Quick Start

```go
// Define typed input
type CalcInput struct {
    Expression string `json:"expression" description:"Math expression" required:"true"`
}

// Create FuncTool (auto-generates JSON Schema)
calc := tool.NewFuncTool("calculate", "Evaluate math expressions",
    func(ctx context.Context, input CalcInput) (*tool.Result, error) {
        result := evaluate(input.Expression)
        return tool.TextResult(fmt.Sprintf("%d", result)), nil
    },
)

// Use in agent
agent := agent.New("assistant",
    agent.WithTools(calc),
)
```

## Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}
```

## FuncTool

Type-safe tool wrapper:

```go
type SearchInput struct {
    Query string `json:"query" description:"Search query" required:"true"`
    Limit int    `json:"limit" description:"Max results" default:"10" min:"1" max:"100"`
}

search := tool.NewFuncTool("search", "Search the web",
    func(ctx context.Context, input SearchInput) (*tool.Result, error) {
        results := searchWeb(input.Query, input.Limit)
        return tool.TextResult(results), nil
    },
)

// Input schema is auto-generated from struct tags
schema := search.InputSchema()
```

## Result Types

```go
// Text result
return tool.TextResult("Success")

// Error result
return tool.ErrorResult(fmt.Errorf("failed"))

// Multimodal result
return &tool.Result{
    Content: []schema.ContentPart{
        schema.TextPart{Text: "Here's the image:"},
        schema.ImagePart{Data: imgBytes, MimeType: "image/png"},
    },
}
```

## Registry

Thread-safe tool collection:

```go
reg := tool.NewRegistry()
reg.Add(searchTool)
reg.Add(calcTool)

// Get by name
t, err := reg.Get("search")

// List all
names := reg.List()

// Get all tools
tools := reg.All()

// Remove
reg.Remove("search")
```

## Middleware

Add retry, timeout, etc.:

```go
wrapped := tool.ApplyMiddleware(myTool,
    tool.WithRetry(3),
    tool.WithTimeout(5*time.Second),
)
```

## Hooks

Inject callbacks:

```go
wrapped := tool.WithHooks(myTool, tool.Hooks{
    BeforeExecute: func(ctx context.Context, toolName string, input map[string]any) error {
        log.Printf("Calling %s with %+v", toolName, input)
        return nil
    },
    AfterExecute: func(ctx context.Context, toolName string, result *tool.Result, err error) {
        if err != nil {
            log.Printf("Tool %s failed: %v", toolName, err)
        }
    },
})
```

## MCP Client

Connect to Model Context Protocol servers:

```go
// Connect to MCP server
tools, err := tool.FromMCP(ctx, "http://localhost:8080",
    tool.WithSessionID("session-123"),
)

// Use remote tools like local tools
for _, t := range tools {
    result, err := t.Execute(ctx, map[string]any{
        "query": "search term",
    })
}
```

### Manual MCP Client

```go
client := tool.NewMCPClient("http://localhost:8080",
    tool.WithSessionID("session-123"),
)

if err := client.Connect(ctx); err != nil {
    return err
}
defer client.Close(ctx)

tools, err := client.ListTools(ctx)
result, err := client.ExecuteTool(ctx, "search", map[string]any{
    "query": "Go programming",
})
```

## MCP Registry

Discover MCP servers:

```go
registry := tool.NewStaticMCPRegistry(
    tool.MCPServerInfo{
        Name:      "web-search",
        URL:       "http://localhost:8080",
        Transport: "streamable-http",
        Tools:     []schema.ToolDefinition{...},
    },
)

servers, err := registry.Discover(ctx)
results, err := registry.Search(ctx, "search")
```

## Convert to ToolDefinition

For LLM tool binding:

```go
def := tool.ToDefinition(myTool)
// Returns schema.ToolDefinition

model = model.BindTools([]schema.ToolDefinition{def})
```

## Example: Multi-Step Tool

```go
type CodeGenInput struct {
    Language string `json:"language" required:"true"`
    Task     string `json:"task" required:"true"`
}

codegen := tool.NewFuncTool("generate_code", "Generate code",
    func(ctx context.Context, input CodeGenInput) (*tool.Result, error) {
        // Generate code
        code := generateCode(input.Language, input.Task)

        // Test code
        if err := testCode(code); err != nil {
            return tool.ErrorResult(err)
        }

        return tool.TextResult(code), nil
    },
)
```

## See Also

- [Agent Package](./agent.md) for tool usage in agents
- [Guard Package](./guard.md) for tool input validation
- [Schema Package](./schema.md) for tool definitions
