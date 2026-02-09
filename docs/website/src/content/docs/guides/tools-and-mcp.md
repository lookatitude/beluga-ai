---
title: Tools & MCP
description: Create tools for agents, build tool registries, and connect to MCP servers for remote tool discovery.
---

The `tool` package provides the tool system that lets agents interact with the outside world. Define tools as typed Go functions, organize them in registries, and connect to remote tool servers via the Model Context Protocol (MCP).

## The Tool Interface

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

`FuncTool` wraps any typed Go function as a `Tool`, automatically generating the JSON Schema from struct tags:

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

Organize tools into registries for modular management:

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

Wrap tools with cross-cutting behavior:

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

Monitor tool execution lifecycle:

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

Connect to remote tool servers using the [Model Context Protocol](https://modelcontextprotocol.io):

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

Expose your tools as an MCP server:

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

### Database Query Tool

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

- [Building Your First Agent](/guides/first-agent/) — Tools in the agent context
- [Orchestration & Workflows](/guides/orchestration/) — Tools across multi-agent systems
- [Safety & Guards](/guides/safety-and-guards/) — Guard tool execution
- [Monitoring & Observability](/guides/observability/) — Trace tool calls
