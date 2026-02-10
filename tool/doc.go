// Package tool provides the tool system for the Beluga AI framework.
//
// It defines the [Tool] interface, a type-safe [FuncTool] wrapper using generics,
// a thread-safe tool [Registry], [Middleware] composition, lifecycle [Hooks],
// an [MCPClient] for connecting to remote MCP tool servers, and an [MCPRegistry]
// for MCP server discovery.
//
// # Tool Interface
//
// The Tool interface is the core abstraction for all tools:
//
//	type Tool interface {
//	    Name() string
//	    Description() string
//	    InputSchema() map[string]any
//	    Execute(ctx context.Context, input map[string]any) (*Result, error)
//	}
//
// Tools have a name (used by the LLM to select them), a description (provided
// to the LLM as context), a JSON Schema for input validation, and an Execute
// method that performs the tool's action.
//
// # FuncTool
//
// [FuncTool] wraps a typed Go function as a Tool using generics. It
// automatically generates a JSON Schema from the input struct's field tags:
//
//	type SearchInput struct {
//	    Query string `json:"query" description:"Search query" required:"true"`
//	    Limit int    `json:"limit" description:"Max results" default:"10"`
//	}
//
//	search := tool.NewFuncTool("search", "Search the web",
//	    func(ctx context.Context, input SearchInput) (*tool.Result, error) {
//	        results := doSearch(ctx, input.Query, input.Limit)
//	        return tool.TextResult(results), nil
//	    },
//	)
//
// The input struct supports json, description, required, and default tags
// recognized by the internal jsonutil.GenerateSchema function.
//
// # Registry
//
// [Registry] is a thread-safe, name-based collection of tools. Tools are
// registered as instances and looked up by name:
//
//	reg := tool.NewRegistry()
//	if err := reg.Add(search); err != nil {
//	    log.Fatal(err)
//	}
//
//	t, err := reg.Get("search")
//	names := reg.List()     // sorted tool names
//	all := reg.All()        // sorted tool instances
//	defs := reg.Definitions() // for LLM binding
//
// # Results
//
// [Result] holds multimodal output from tool execution. Content parts support
// text, images, and other types via schema.ContentPart. Convenience constructors:
//
//	result := tool.TextResult("The answer is 42")
//	errResult := tool.ErrorResult(fmt.Errorf("not found"))
//
// Use [ToDefinition] to convert a Tool to a schema.ToolDefinition for LLM providers.
//
// # MCP Client
//
// [MCPClient] connects to an MCP (Model Context Protocol) server using the
// Streamable HTTP transport. It wraps remote tools as native Tool instances:
//
//	tools, err := tool.FromMCP(ctx, "https://mcp.example.com/tools",
//	    tool.WithSessionID("session-1"),
//	)
//
// The transport protocol uses POST for requests, GET for notifications,
// DELETE for session termination, and Mcp-Session-Id for session management.
//
// # MCP Registry
//
// [MCPRegistry] provides discovery of MCP servers. The [StaticMCPRegistry]
// implementation is backed by a fixed list of servers:
//
//	registry := tool.NewStaticMCPRegistry(
//	    tool.MCPServerInfo{Name: "code-tools", URL: "https://mcp.example.com/code"},
//	    tool.MCPServerInfo{Name: "search-tools", URL: "https://mcp.example.com/search"},
//	)
//
//	servers, err := registry.Search(ctx, "code")
//
// # Middleware
//
// [Middleware] wraps a Tool to add cross-cutting behavior. Built-in middleware
// includes [WithTimeout] and [WithRetry]. Applied via [ApplyMiddleware]:
//
//	wrapped := tool.ApplyMiddleware(myTool,
//	    tool.WithTimeout(30 * time.Second),
//	    tool.WithRetry(3),
//	)
//
// # Hooks
//
// [Hooks] provide lifecycle callbacks around tool execution. Compose multiple
// hooks with [ComposeHooks], or wrap a tool with hooks using [WithHooks]:
//
//	hooks := tool.Hooks{
//	    BeforeExecute: func(ctx context.Context, name string, input map[string]any) error {
//	        log.Printf("Executing tool: %s", name)
//	        return nil
//	    },
//	}
//	hooked := tool.WithHooks(myTool, hooks)
package tool
