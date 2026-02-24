---
title: Tools and Model Context Protocol
description: "Create typed Go tools for agents, organize them in registries, and connect to MCP servers for runtime tool discovery and interoperability."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tools, MCP, Model Context Protocol, FuncTool, tool registry, tool calling"
---

LLMs generate text, but to be useful in production systems they need to take actions — query databases, call APIs, read files, send notifications. Tools bridge this gap by giving agents typed, validated functions they can invoke. The LLM sees a tool's name, description, and JSON Schema, decides when to call it, and the framework handles argument parsing, execution, and result delivery.

The `tool` package provides the complete tool system: define tools as typed Go functions with automatic schema generation, organize them in registries, add cross-cutting behavior with middleware, and connect to remote tool servers via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) for runtime discovery.

## The Tool Interface

The `Tool` interface is intentionally minimal — four methods that capture everything an LLM needs to discover and invoke a tool. The `InputSchema` returns a JSON Schema that the LLM uses to generate valid arguments, and `Execute` runs the tool with those arguments. This interface is the foundation for all tool types: local functions, MCP remote tools, and agent handoffs.

```go
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	Execute(ctx context.Context, input map[string]any) (*Result, error)
}
```

| Method | Purpose |
|--------|---------|
| `Name` | Unique identifier shown to the LLM |
| `Description` | Explains when to use this tool |
| `InputSchema` | JSON Schema for input parameters |
| `Execute` | Runs the tool and returns a result |

## Creating Tools with FuncTool

Writing JSON Schema by hand is tedious and error-prone — every field needs a type, description, and validation rules, and any mismatch between the schema and your code causes runtime failures. `FuncTool` eliminates this by deriving the JSON Schema automatically from Go struct tags. You define a typed input struct with `json`, `description`, `required`, and `default` tags, and the framework generates a correct schema at construction time. This approach leverages Go's type system for compile-time safety while producing the JSON Schema that LLMs need for argument generation.

```go
import "github.com/lookatitude/beluga-ai/tool"

// 1. Define the input struct with tags
type WeatherInput struct {
	City    string `json:"city" description:"City name" required:"true"`
	Units   string `json:"units" description:"Temperature units" default:"celsius"`
}

// 2. Create the tool
weather := tool.NewFuncTool("get_weather", "Get current weather for a city",
	func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
		// Call your weather API
		temp, err := fetchWeather(ctx, input.City, input.Units)
		if err != nil {
			return nil, err
		}
		return tool.TextResult(fmt.Sprintf("%s: %d°%s", input.City, temp, input.Units)), nil
	},
)
```

### Struct Tag Reference

| Tag | Purpose | Example |
|-----|---------|---------|
| `json` | JSON field name | `json:"city"` |
| `description` | Field description for LLM | `description:"City name"` |
| `required` | Mark as required | `required:"true"` |
| `default` | Default value | `default:"celsius"` |

### Tool Results

```go
// Text result
return tool.TextResult("The answer is 42"), nil

// Error result (reported to LLM, not a Go error)
return tool.ErrorResult(fmt.Errorf("city not found")), nil

// Multimodal result
return &tool.Result{
	Content: []schema.ContentPart{
		schema.TextPart{Text: "Chart data"},
		schema.ImagePart{URL: "https://example.com/chart.png"},
	},
}, nil
```

## Tool Registry

As the number of tools grows, you need a way to organize, discover, and manage them. The tool registry groups tools into named collections that can be passed to agents, exposed via MCP servers, or composed from multiple sources. Registries support listing, lookup by name, and bulk middleware application.

```go
reg := tool.NewRegistry()
reg.Add(weather)
reg.Add(calculator)
reg.Add(searchTool)

// List all tools
for _, t := range reg.List() {
	fmt.Printf("Tool: %s - %s\n", t.Name(), t.Description())
}

// Get a specific tool
t, ok := reg.Get("get_weather")

// Use with an agent
a := agent.New("assistant",
	agent.WithLLM(model),
	agent.WithTools(reg.List()),
)
```

## Tool Middleware

Tool middleware follows the same `func(T) T` composable pattern used for LLM middleware and memory middleware throughout Beluga AI. This consistency means you already know the pattern: each middleware wraps a `Tool`, intercepts `Execute` calls, and delegates to the next layer. Common use cases include logging, timeout enforcement, rate limiting, and metrics collection. Middleware can be applied to individual tools or to all tools in a registry at once.

```go
// Logging middleware
loggingMiddleware := tool.LoggingMiddleware(logger)

// Timeout middleware
timeoutMiddleware := tool.TimeoutMiddleware(5 * time.Second)

// Apply middleware to a tool
wrappedTool := tool.ApplyMiddleware(weather,
	loggingMiddleware,
	timeoutMiddleware,
)

// Apply middleware to all tools in a registry
reg.ApplyMiddleware(loggingMiddleware)
```

### Writing Custom Middleware

Custom middleware follows a closure pattern: the outer function captures configuration, and the inner function wraps the tool. The `MiddlewareFunc` helper creates a tool that delegates `Name`, `Description`, and `InputSchema` to the wrapped tool while intercepting `Execute`. This example demonstrates a rate limiter that uses Go's `rate.Limiter` to throttle tool calls.

```go
func RateLimitMiddleware(rps int) tool.Middleware {
	limiter := rate.NewLimiter(rate.Limit(rps), rps)
	return func(next tool.Tool) tool.Tool {
		return tool.MiddlewareFunc(next, func(ctx context.Context, input map[string]any) (*tool.Result, error) {
			if err := limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limited: %w", err)
			}
			return next.Execute(ctx, input)
		})
	}
}
```

## Tool Hooks

Tool hooks provide lifecycle observation without wrapping. Unlike middleware, which changes execution behavior, hooks are for monitoring and auditing. `BeforeExecute` runs before every tool call (return an error to abort), `AfterExecute` runs after completion regardless of outcome, and `OnError` runs when execution fails. All hook fields are optional — nil hooks are skipped.

```go
hooks := tool.Hooks{
	BeforeExecute: func(ctx context.Context, name string, input map[string]any) error {
		log.Printf("Executing tool: %s with %v", name, input)
		return nil // Return error to abort
	},
	AfterExecute: func(ctx context.Context, name string, result *tool.Result, err error) {
		log.Printf("Tool %s completed: error=%v", name, err)
	},
	OnError: func(ctx context.Context, name string, err error) error {
		log.Printf("Tool %s error: %v", name, err)
		return err
	},
}
```

## MCP Client

The [Model Context Protocol (MCP)](https://modelcontextprotocol.io) is an open standard that enables agents to discover and invoke tools hosted on remote servers. This is significant because it decouples tool implementation from agent implementation: a tool server can be written in any language and deployed independently, and agents discover available tools at runtime through the protocol's `ListTools` endpoint. MCP uses Streamable HTTP transport for reliable, bidirectional communication.

Connect to a remote MCP server:

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp"

// Connect to an MCP server
client, err := mcp.NewClient(mcp.ClientConfig{
	ServerURL: "http://localhost:3001/mcp",
	Transport: "streamable-http", // Streamable HTTP transport
})
if err != nil {
	log.Fatal(err)
}
defer client.Close()

// Discover available tools
tools, err := client.ListTools(ctx)
if err != nil {
	log.Fatal(err)
}

for _, t := range tools {
	fmt.Printf("Remote tool: %s - %s\n", t.Name(), t.Description())
}

// Use MCP tools with an agent
a := agent.New("mcp-agent",
	agent.WithLLM(model),
	agent.WithTools(tools),
)
```

### MCP with Multiple Servers

In production, tools often span multiple services: a file operations server, a database query server, a notification server. MCP clients can connect to multiple servers simultaneously, and their tools can be combined into a single tool set for the agent. This architecture scales horizontally — each MCP server is an independent microservice.

```go
// Connect to multiple MCP servers
filesClient, err := mcp.NewClient(mcp.ClientConfig{
	ServerURL: "http://files-server:3001/mcp",
})

dbClient, err := mcp.NewClient(mcp.ClientConfig{
	ServerURL: "http://db-server:3002/mcp",
})

// Combine tools from multiple sources
var allTools []tool.Tool
fileTools, _ := filesClient.ListTools(ctx)
dbTools, _ := dbClient.ListTools(ctx)
allTools = append(allTools, fileTools...)
allTools = append(allTools, dbTools...)

a := agent.New("multi-mcp",
	agent.WithLLM(model),
	agent.WithTools(allTools),
)
```

## MCP Server

You can also expose your Go tools as an MCP server, making them available to any MCP-compatible agent or application. This is useful for building reusable tool services that multiple agents or teams can share. The server handles protocol negotiation, tool discovery, argument validation, and result serialization.

```go
import "github.com/lookatitude/beluga-ai/protocol/mcp"

server := mcp.NewServer(mcp.ServerConfig{
	Name:    "my-tools",
	Version: "1.0.0",
})

// Register tools
server.AddTool(weather)
server.AddTool(calculator)

// Start the server with Streamable HTTP transport
if err := server.ListenAndServe(":3001"); err != nil {
	log.Fatal(err)
}
```

## Building Complex Tools

The `FuncTool` pattern scales to tools of any complexity. The following examples demonstrate common production patterns: a database query tool that returns structured results, and an HTTP API tool that wraps internal services. Both follow the same struct-tag-driven schema generation and typed function signature.

### Database Query Tool

This tool executes read-only SQL queries against a database. The input struct captures the query and parameters, and the function handles row scanning and JSON serialization. Returning an `ErrorResult` (instead of a Go error) reports the problem to the LLM without terminating the agent loop, allowing the LLM to try a different query.

```go
type QueryInput struct {
	SQL    string `json:"sql" description:"SQL query to execute" required:"true"`
	Params []any  `json:"params" description:"Query parameters"`
}

queryTool := tool.NewFuncTool("query_db", "Execute a read-only SQL query",
	func(ctx context.Context, input QueryInput) (*tool.Result, error) {
		rows, err := db.QueryContext(ctx, input.SQL, input.Params...)
		if err != nil {
			return tool.ErrorResult(err), nil
		}
		defer rows.Close()

		var results []map[string]any
		// ... scan rows into results ...

		data, err := json.Marshal(results)
		if err != nil {
			return nil, err
		}
		return tool.TextResult(string(data)), nil
	},
)
```

### HTTP API Tool

This tool wraps an internal HTTP API, giving the agent access to existing services without exposing raw HTTP details. The LLM specifies an endpoint, method, and body, and the tool handles HTTP mechanics.

```go
type APIInput struct {
	Endpoint string            `json:"endpoint" description:"API endpoint path" required:"true"`
	Method   string            `json:"method" description:"HTTP method" default:"GET"`
	Body     map[string]any    `json:"body" description:"Request body"`
}

apiTool := tool.NewFuncTool("call_api", "Call an internal API endpoint",
	func(ctx context.Context, input APIInput) (*tool.Result, error) {
		resp, err := httpClient.Do(ctx, input.Method, input.Endpoint, input.Body)
		if err != nil {
			return tool.ErrorResult(err), nil
		}
		return tool.TextResult(resp.Body), nil
	},
)
```

## Using Tools with Agents

Once tools are defined, passing them to an agent is a single option. The agent's LLM decides when to call tools based on the user's request and the tool descriptions. The framework handles the tool-call loop: the LLM generates tool-call requests, the framework executes them, and the results are fed back to the LLM for the next response.

```go
a := agent.New("assistant",
	agent.WithLLM(model),
	agent.WithTools([]tool.Tool{weather, calculator, queryTool}),
	agent.WithPersona(agent.Persona{
		Role: "data analyst",
		Goal: "answer questions using available tools",
	}),
)

// The agent automatically decides when to call tools
result, err := a.Invoke(ctx, "What's the weather in London and what's 15% of 250?")
```

## Next Steps

- [Building Your First Agent](/docs/guides/first-agent/) — Tools in the agent context
- [Orchestration & Workflows](/docs/guides/orchestration/) — Tools across multi-agent systems
- [Safety & Guards](/docs/guides/safety-and-guards/) — Guard tool execution
- [Monitoring & Observability](/docs/guides/observability/) — Trace tool calls
