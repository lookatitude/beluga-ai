---
title: Tool Recipes
description: "Go recipes for building AI agent tools: wrap Go functions, compose with middleware, connect MCP servers, and integrate external APIs with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go tool recipes, FuncTool, MCP tools, tool middleware, agent tool integration, tool composition, AI tools"
---

Beluga AI's tool system is designed around a small `Tool` interface with three composability mechanisms: the `FuncTool` constructor for wrapping Go functions, middleware (`func(Tool) Tool`) for cross-cutting concerns, and hooks for lightweight lifecycle callbacks. These recipes cover the most common tool patterns you will encounter in production, from wrapping business logic to connecting to remote MCP servers.

## Wrap Any Go Function as a Tool

**Problem:** You have an existing Go function and want to expose it to the LLM as a callable tool, complete with auto-generated JSON Schema.

**Solution:** Use `tool.NewFuncTool` with a typed input struct. Tags on the struct fields generate the schema automatically.

The `FuncTool` constructor bridges the gap between typed Go code and the untyped JSON world of LLM tool calling. When you define a struct with `json`, `description`, and `required` tags, Beluga AI generates a JSON Schema at construction time that the LLM uses to understand what arguments the tool accepts. This approach is safer than defining schemas manually because the schema and the handler always stay in sync -- if you add a field to the struct, it automatically appears in the schema.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/tool"
)

// Define a typed input struct. Tags control the JSON Schema.
type WeatherInput struct {
	City    string `json:"city" description:"City name to get weather for" required:"true"`
	Units   string `json:"units" description:"Temperature units: celsius or fahrenheit" default:"celsius"`
	Days    int    `json:"days" description:"Number of forecast days (1-7)" default:"1"`
}

func main() {
	// NewFuncTool generates JSON Schema from WeatherInput at construction time.
	weather := tool.NewFuncTool("get_weather", "Get current weather and forecast for a city",
		func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
			// Your business logic here -- call a weather API, database, etc.
			forecast := fmt.Sprintf("Weather in %s: 22°%s, sunny, %d-day forecast",
				input.City,
				map[string]string{"celsius": "C", "fahrenheit": "F"}[input.Units],
				input.Days,
			)
			return tool.TextResult(forecast), nil
		},
	)

	// Inspect the auto-generated schema.
	schema, _ := json.MarshalIndent(weather.InputSchema(), "", "  ")
	fmt.Printf("Tool: %s\nDescription: %s\nSchema:\n%s\n\n",
		weather.Name(), weather.Description(), schema)

	// Execute the tool directly.
	result, err := weather.Execute(context.Background(), map[string]any{
		"city":  "Tokyo",
		"units": "celsius",
		"days":  3,
	})
	if err != nil {
		slog.Error("tool failed", "error", err)
		return
	}

	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			fmt.Println("Result:", tp.Text)
		}
	}
}
```

**Supported struct tags:**
- `json:"name"` -- Field name in the schema
- `description:"..."` -- Field description for the LLM (helps the model understand when and how to use each parameter)
- `required:"true"` -- Marks the field as required (the LLM must provide it)
- `default:"value"` -- Default value hint (used when the LLM omits the field)

---

## Connect to MCP Servers

**Problem:** You want to use tools hosted on remote MCP (Model Context Protocol) servers without implementing them locally.

**Solution:** Use `tool.NewMCPClient` to connect to an MCP server and `tool.FromMCP` to import its tools.

MCP is a standardized protocol for exposing tools to LLM-powered applications. Instead of implementing every tool locally, you can connect to MCP servers that host tools as services. This is particularly useful for tools that require specialized infrastructure (databases, search engines, APIs) or when tools are maintained by separate teams. The MCP client handles connection management, tool discovery, argument serialization, and result parsing transparently.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/tool"
)

func main() {
	ctx := context.Background()

	// Create an MCP client with authentication headers.
	client := tool.NewMCPClient("https://mcp.example.com/v1",
		tool.WithSessionID("session-abc-123"),
		tool.WithMCPHeaders(map[string]string{
			"Authorization": "Bearer your-api-token",
		}),
	)

	// Connect and list available tools.
	err := client.Connect(ctx)
	if err != nil {
		slog.Error("MCP connection failed", "error", err)
		return
	}
	defer client.Close(ctx)

	tools, err := client.ListTools(ctx)
	if err != nil {
		slog.Error("list tools failed", "error", err)
		return
	}

	fmt.Printf("Available MCP tools: %d\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s: %s\n", t.Name, t.Description)
	}

	// Execute a remote tool.
	result, err := client.ExecuteTool(ctx, "search", map[string]any{
		"query": "Go concurrency patterns",
	})
	if err != nil {
		slog.Error("tool execution failed", "error", err)
		return
	}
	fmt.Printf("Result: %v\n", result)
}
```

**Quick import with `FromMCP`:**

```go
// One-liner to import all tools from an MCP server.
tools, err := tool.FromMCP(ctx, "https://mcp.example.com/v1",
	tool.WithSessionID("session-123"),
)
if err != nil {
	slog.Error("MCP import failed", "error", err)
	return
}
// Use with an agent:
// agent.New("assistant", agent.WithTools(tools))
```

---

## Tool Middleware for Logging and Auth

**Problem:** You need to add cross-cutting concerns (logging, authorization, rate limiting) to tool execution without modifying each tool.

**Solution:** Use `tool.Middleware` and `tool.ApplyMiddleware` to wrap tools.

Middleware follows Beluga AI's `func(T) T` pattern: a function that takes a `Tool` and returns a new `Tool` with added behavior. This is the same approach used for LLM middleware and agent middleware throughout the framework, creating a consistent extension model. Middleware is applied right-to-left by `ApplyMiddleware`, so the first middleware in the list becomes the outermost wrapper. In the example below, auth checks run first (outermost), then logging wraps the actual execution (innermost).

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/tool"
)

// loggingMiddleware logs every tool execution with timing.
func loggingMiddleware(next tool.Tool) tool.Tool {
	return &loggedTool{inner: next}
}

type loggedTool struct {
	inner tool.Tool
}

func (t *loggedTool) Name() string                  { return t.inner.Name() }
func (t *loggedTool) Description() string            { return t.inner.Description() }
func (t *loggedTool) InputSchema() map[string]any    { return t.inner.InputSchema() }

func (t *loggedTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	start := time.Now()
	slog.Info("tool execution started", "tool", t.inner.Name(), "input", input)

	result, err := t.inner.Execute(ctx, input)
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("tool execution failed",
			"tool", t.inner.Name(),
			"duration", elapsed,
			"error", err,
		)
	} else {
		slog.Info("tool execution completed",
			"tool", t.inner.Name(),
			"duration", elapsed,
			"is_error", result.IsError,
		)
	}

	return result, err
}

// authMiddleware checks that the caller has permission to use a tool.
func authMiddleware(allowedTools map[string]bool) tool.Middleware {
	return func(next tool.Tool) tool.Tool {
		return &authTool{inner: next, allowed: allowedTools}
	}
}

type authTool struct {
	inner   tool.Tool
	allowed map[string]bool
}

func (t *authTool) Name() string                  { return t.inner.Name() }
func (t *authTool) Description() string            { return t.inner.Description() }
func (t *authTool) InputSchema() map[string]any    { return t.inner.InputSchema() }

func (t *authTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	if !t.allowed[t.inner.Name()] {
		return nil, fmt.Errorf("tool %q is not authorized", t.inner.Name())
	}
	return t.inner.Execute(ctx, input)
}

func main() {
	type CalcInput struct {
		Expr string `json:"expression" description:"Math expression" required:"true"`
	}
	calc := tool.NewFuncTool("calculate", "Evaluate math",
		func(ctx context.Context, input CalcInput) (*tool.Result, error) {
			return tool.TextResult("42"), nil
		},
	)

	// Stack middleware: auth check runs first, then logging wraps execution.
	wrapped := tool.ApplyMiddleware(calc,
		loggingMiddleware,
		authMiddleware(map[string]bool{"calculate": true}),
	)

	result, err := wrapped.Execute(context.Background(), map[string]any{
		"expression": "6 * 7",
	})
	if err != nil {
		slog.Error("failed", "error", err)
		return
	}
	fmt.Printf("Result: %v\n", result)
}
```

---

## Tool Hooks for Lifecycle Events

**Problem:** You want to run logic before/after tool execution (validation, metrics, error transformation) without writing full middleware.

**Solution:** Use `tool.WithHooks` for lightweight lifecycle callbacks.

Hooks and middleware serve different purposes. Middleware wraps the entire tool interface and is suited for structural changes (authorization, caching, protocol adaptation). Hooks are lightweight callbacks on specific lifecycle events and are suited for cross-cutting concerns that don't change the tool's behavior (logging, metrics, validation). You can compose multiple hook sets with `tool.ComposeHooks`, and nil hook fields are simply skipped at zero cost.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/tool"
)

func main() {
	var executionCount atomic.Int64

	type PingInput struct {
		Target string `json:"target" description:"Host to ping" required:"true"`
	}
	ping := tool.NewFuncTool("ping", "Ping a host",
		func(ctx context.Context, input PingInput) (*tool.Result, error) {
			return tool.TextResult(fmt.Sprintf("Pong from %s", input.Target)), nil
		},
	)

	// Add hooks for validation, metrics, and error handling.
	hooked := tool.WithHooks(ping, tool.Hooks{
		BeforeExecute: func(ctx context.Context, name string, input map[string]any) error {
			executionCount.Add(1)
			slog.Info("before execute", "tool", name, "count", executionCount.Load())

			// Validate input before execution.
			if target, ok := input["target"].(string); !ok || target == "" {
				return fmt.Errorf("target is required")
			}
			return nil
		},
		AfterExecute: func(ctx context.Context, name string, result *tool.Result, err error) {
			slog.Info("after execute",
				"tool", name,
				"is_error", result != nil && result.IsError,
				"error", err,
			)
		},
		OnError: func(ctx context.Context, name string, err error) error {
			slog.Error("tool error", "tool", name, "error", err)
			// Return the error to propagate, or return nil to suppress.
			return err
		},
	})

	// Hooks compose with ComposeHooks for multiple hook sets.
	result, err := hooked.Execute(context.Background(), map[string]any{
		"target": "example.com",
	})
	if err != nil {
		slog.Error("failed", "error", err)
		return
	}
	fmt.Printf("Result: %v\n", result)
}
```

---

## Dynamic Tool Injection

**Problem:** Available tools should change based on the conversation state, user permissions, or agent context. You can't define all tools at agent creation time.

**Solution:** Use a dynamic registry that resolves tools at execution time.

Static tool sets work for simple agents, but production systems often need tools that vary per request. An admin user might have access to destructive operations that a regular user should never see. A conversation about billing might need payment tools that aren't relevant during technical support. A dynamic registry backed by `sync.RWMutex` allows safe concurrent access from multiple goroutines while tools are added or removed at runtime.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lookatitude/beluga-ai/tool"
)

// DynamicRegistry allows adding and removing tools at runtime.
type DynamicRegistry struct {
	mu    sync.RWMutex
	tools map[string]tool.Tool
}

func NewDynamicRegistry() *DynamicRegistry {
	return &DynamicRegistry{tools: make(map[string]tool.Tool)}
}

func (r *DynamicRegistry) Add(t tool.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *DynamicRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

func (r *DynamicRegistry) Get(name string) (tool.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *DynamicRegistry) All() []tool.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]tool.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// FilterByPermission returns only tools the user is allowed to access.
func (r *DynamicRegistry) FilterByPermission(userRole string) []tool.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allowed []tool.Tool
	for _, t := range r.tools {
		// Check tool metadata for role requirements.
		allowed = append(allowed, t)
	}
	return allowed
}

func main() {
	reg := NewDynamicRegistry()

	type SearchInput struct {
		Query string `json:"query" required:"true"`
	}
	reg.Add(tool.NewFuncTool("search", "Search documents",
		func(ctx context.Context, input SearchInput) (*tool.Result, error) {
			return tool.TextResult("results"), nil
		},
	))

	fmt.Printf("Available tools: %d\n", len(reg.All()))

	// Add a tool dynamically based on conversation state.
	type AdminInput struct {
		Action string `json:"action" required:"true"`
	}
	reg.Add(tool.NewFuncTool("admin_action", "Perform admin action",
		func(ctx context.Context, input AdminInput) (*tool.Result, error) {
			return tool.TextResult("action completed"), nil
		},
	))

	fmt.Printf("Available tools after injection: %d\n", len(reg.All()))

	// Remove tool when no longer needed.
	reg.Remove("admin_action")
	fmt.Printf("Available tools after removal: %d\n", len(reg.All()))
}
```

---

## MCP Server Registry and Discovery

**Problem:** You have multiple MCP servers and need to discover and route to the right one based on the tool being requested.

**Solution:** Use `tool.StaticMCPRegistry` for fixed server lists, or implement `tool.MCPRegistry` for dynamic discovery.

When your application connects to multiple MCP servers, you need a way to discover which server provides which tool. The `StaticMCPRegistry` holds a fixed list of servers and supports search by name. For more dynamic environments (service meshes, Kubernetes), implement the `MCPRegistry` interface with custom discovery logic.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/tool"
)

func main() {
	ctx := context.Background()

	// Create a static registry of known MCP servers.
	registry := tool.NewStaticMCPRegistry(
		tool.MCPServerInfo{
			Name:      "weather-tools",
			URL:       "https://weather.mcp.example.com",
			Transport: "streamable-http",
		},
		tool.MCPServerInfo{
			Name:      "search-tools",
			URL:       "https://search.mcp.example.com",
			Transport: "streamable-http",
		},
		tool.MCPServerInfo{
			Name:      "calendar-tools",
			URL:       "https://calendar.mcp.example.com",
			Transport: "streamable-http",
		},
	)

	// Discover all servers.
	servers, err := registry.Discover(ctx)
	if err != nil {
		slog.Error("discovery failed", "error", err)
		return
	}
	fmt.Printf("Found %d MCP servers\n", len(servers))

	// Search for specific servers by name (case-insensitive).
	results, err := registry.Search(ctx, "weather")
	if err != nil {
		slog.Error("search failed", "error", err)
		return
	}
	fmt.Printf("Matching 'weather': %d servers\n", len(results))
	for _, s := range results {
		fmt.Printf("  - %s (%s)\n", s.Name, s.URL)
	}
}
```

---

## Tool Result Error Handling

**Problem:** You need to distinguish between tool execution errors (infrastructure failures) and tool result errors (the tool ran but the operation failed).

**Solution:** Use `tool.ErrorResult` for domain errors and Go errors for infrastructure failures.

This distinction matters because the two error types require different handling. Infrastructure errors (network timeouts, connection refused) are transient and should be retried. Domain errors (resource not found, permission denied) are permanent and the LLM should know about them so it can adjust its approach. Returning infrastructure errors as Go `error` values triggers retry middleware and error hooks. Returning domain errors via `tool.ErrorResult` produces a `ToolMessage` with `IsError: true` that the LLM sees as a tool result and can reason about.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/lookatitude/beluga-ai/tool"
)

func main() {
	type DeleteInput struct {
		ID string `json:"id" description:"Resource ID to delete" required:"true"`
	}

	deleteTool := tool.NewFuncTool("delete_resource", "Delete a resource by ID",
		func(ctx context.Context, input DeleteInput) (*tool.Result, error) {
			// Infrastructure errors -- return Go error.
			// These trigger retry logic and error hooks.
			resp, err := http.Get("https://api.example.com/resources/" + input.ID)
			if err != nil {
				return nil, fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			// Domain errors -- return ErrorResult.
			// The LLM sees this as a tool result and can react accordingly.
			switch resp.StatusCode {
			case http.StatusNotFound:
				return tool.ErrorResult(fmt.Errorf("resource %s not found", input.ID)), nil
			case http.StatusForbidden:
				return tool.ErrorResult(fmt.Errorf("not authorized to delete %s", input.ID)), nil
			case http.StatusOK:
				return tool.TextResult(fmt.Sprintf("Deleted resource %s", input.ID)), nil
			default:
				return tool.ErrorResult(fmt.Errorf("unexpected status: %d", resp.StatusCode)), nil
			}
		},
	)

	result, err := deleteTool.Execute(context.Background(), map[string]any{
		"id": "res-123",
	})
	if err != nil {
		// Infrastructure error -- log and possibly retry.
		slog.Error("tool infrastructure error", "error", err)
		return
	}

	if result.IsError {
		// Domain error -- the LLM will see this and adapt.
		fmt.Println("Tool reported an error (LLM will see this):")
	} else {
		fmt.Println("Tool succeeded:")
	}
	fmt.Printf("  Content: %v\n", result.Content)
}
```

---

## Tool Registry with the Standard Pattern

**Problem:** You want a centralized place to register, discover, and look up tools by name.

**Solution:** Use `tool.Registry`, which provides thread-safe tool management.

The registry pattern (`Register()` + `New()` + `List()`) is used throughout Beluga AI for all extensible types: LLM providers, memory stores, vector stores, and tools. The tool registry follows this same convention, providing a consistent API for tool management. Thread safety via `sync.RWMutex` allows registering tools during initialization (typically in `init()` functions) while concurrent agents look up tools at execution time.

```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/tool"
)

func main() {
	reg := tool.NewRegistry()

	// Register multiple tools.
	type CalcInput struct {
		Expr string `json:"expression" required:"true"`
	}
	reg.Add(tool.NewFuncTool("calculate", "Evaluate math expressions",
		func(ctx context.Context, input CalcInput) (*tool.Result, error) {
			return tool.TextResult("42"), nil
		},
	))

	type SearchInput struct {
		Query string `json:"query" required:"true"`
	}
	reg.Add(tool.NewFuncTool("search", "Search the web",
		func(ctx context.Context, input SearchInput) (*tool.Result, error) {
			return tool.TextResult("results for: " + input.Query), nil
		},
	))

	// Look up a specific tool.
	if t, ok := reg.Get("calculate"); ok {
		fmt.Printf("Found: %s -- %s\n", t.Name(), t.Description())
	}

	// List all registered tools.
	fmt.Printf("Registered tools: %d\n", len(reg.All()))
	for _, t := range reg.All() {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}

	// Convert to definitions for LLM binding.
	defs := make([]interface{}, 0)
	for _, t := range reg.All() {
		defs = append(defs, tool.ToDefinition(t))
	}
	fmt.Printf("Tool definitions for LLM: %d\n", len(defs))
}
```
