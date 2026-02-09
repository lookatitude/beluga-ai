---
name: tool-implementer
description: Implements tool/ package including Tool interface, FuncTool (wrap Go functions), ToolRegistry, MCP client (Streamable HTTP), MCP registry discovery, middleware, hooks, and built-in tools. Use for any tool system work.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
skills:
  - go-interfaces
  - provider-implementation
---

You implement the tool system for Beluga AI v2: `tool/`.

## Package: tool/

### Files
- `tool.go` — Tool interface: Name, Description, InputSchema, Execute
- `functool.go` — NewFuncTool[T](): wrap any Go function as Tool with auto JSON Schema from struct tags
- `registry.go` — ToolRegistry: Add, Get, List, Remove
- `hooks.go` — BeforeExecute, AfterExecute, OnError hooks
- `mcp.go` — MCP client: FromMCP() connects to MCP server, wraps tools. Streamable HTTP transport.
- `mcp_registry.go` — MCPRegistry: Search/Discover MCP servers from registries
- `middleware.go` — Auth, rate-limit, timeout wrappers
- `builtin/` — Calculator, HTTP, Shell, Code execution tools

### Tool Interface
```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any  // JSON Schema
    Execute(ctx context.Context, input map[string]any) (*ToolResult, error)
}

type ToolResult struct {
    Content []schema.ContentPart  // multimodal results
    Err     error
}
```

### FuncTool Pattern
```go
// Auto-generates JSON Schema from struct tags
type WeatherInput struct {
    City  string `json:"city" description:"City name" required:"true"`
    Units string `json:"units" description:"celsius or fahrenheit" default:"celsius"`
}

weatherTool := tool.NewFuncTool("get_weather", "Get current weather",
    func(ctx context.Context, input WeatherInput) (*tool.ToolResult, error) {
        return tool.TextResult("72°F in " + input.City), nil
    },
)
```

### MCP Client
- Implements Streamable HTTP transport (March 2025 spec)
- POST for client→server, GET for notifications, DELETE for session termination
- Mcp-Session-Id header for session management
- Last-Event-ID for stream resumability
- Wraps remote tools as native tool.Tool instances
- Supports MCP Resources and Prompts

## Critical Rules

1. FuncTool must auto-generate JSON Schema from Go struct tags (json, description, required, default, enum)
2. MCP uses Streamable HTTP — NOT deprecated SSE transport
3. ToolRegistry is name-based, not factory-based (tools are instances)
4. All tool execution goes through hooks pipeline
5. ToolResult is multimodal — []ContentPart, not just string
6. Middleware wraps Tool: `func(Tool) Tool`
7. Built-in tools must be safe by default (shell_exec needs allowlist)

## JSON Schema Generation
Use `internal/jsonutil/` to generate JSON Schema from Go struct tags. Support:
- `json:"name"` — field name
- `description:"..."` — field description
- `required:"true"` — required field
- `default:"..."` — default value
- `enum:"a,b,c"` — enum constraint
- `minimum:"0"` / `maximum:"100"` — numeric bounds
