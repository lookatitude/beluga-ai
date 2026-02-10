// Package mcp implements the Model Context Protocol (MCP) for tool and resource
// sharing between AI systems. It supports the Streamable HTTP transport per the
// March 2025 MCP specification, using a single HTTP endpoint for JSON-RPC 2.0
// messages.
//
// MCP enables AI applications to expose and consume tools, resources, and prompt
// templates across process and network boundaries. The protocol uses JSON-RPC 2.0
// as its message format, with methods including initialize, tools/list, tools/call,
// resources/list, and prompts/list.
//
// # Server
//
// MCPServer exposes Beluga tools, resources, and prompts to MCP clients. Tools
// registered with the server are automatically converted to MCP tool listings
// and can be invoked remotely via JSON-RPC.
//
//	srv := mcp.NewServer("my-server", "1.0.0")
//	srv.AddTool(myTool)
//	srv.AddResource(mcp.Resource{URI: "file:///data.json", Name: "data"})
//	srv.Serve(ctx, ":8080")
//
// The server can also be used as an http.Handler for integration with existing
// HTTP servers:
//
//	http.Handle("/mcp", srv.Handler())
//
// # Client
//
// MCPClient connects to a remote MCP server and provides methods for
// initialization, tool listing, and tool invocation.
//
//	client := mcp.NewClient("http://localhost:8080/mcp")
//	caps, err := client.Initialize(ctx)
//	tools, err := client.ListTools(ctx)
//	result, err := client.CallTool(ctx, "search", map[string]any{"query": "hello"})
//
// # Bridge Function
//
// FromMCP connects to an MCP server and returns its tools as native tool.Tool
// instances, enabling seamless integration of remote MCP tools into Beluga agents:
//
//	tools, err := mcp.FromMCP(ctx, "http://localhost:8080/mcp")
//	agent := agent.New("assistant", agent.WithTools(tools...))
//
// # Key Types
//
//   - MCPServer — serves Beluga tools/resources/prompts via MCP
//   - MCPClient — connects to remote MCP servers
//   - Request / Response — JSON-RPC 2.0 message types
//   - ToolInfo / ToolCallParams / ToolCallResult — tool operation types
//   - Resource / Prompt — MCP resource and prompt template types
//   - ServerCapabilities — describes server feature support
package mcp
