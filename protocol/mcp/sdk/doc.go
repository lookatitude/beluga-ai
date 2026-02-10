// Package sdk provides integration between the official MCP Go SDK
// (github.com/modelcontextprotocol/go-sdk) and Beluga's MCP protocol layer.
// It bridges Beluga's tool.Tool interface with the SDK's server and client,
// enabling exposure of Beluga tools via the official MCP SDK and consumption
// of remote MCP tools as native Beluga tools.
//
// This package is useful when you need full compliance with the official MCP
// SDK behavior, including support for advanced features like transport
// negotiation and session management provided by the SDK.
//
// # Server
//
// NewServer creates an MCP server using the official SDK and registers Beluga
// tools. Each tool.Tool is exposed as an MCP tool with its name, description,
// and input schema.
//
//	srv := sdk.NewServer("my-server", "1.0.0", searchTool, calcTool)
//	// srv is an *sdkmcp.Server from the official SDK
//
// # Client
//
// NewClient creates an MCP client using the official SDK and connects it to
// a server via a transport. FromSession lists tools from the connected session
// and returns them as native Beluga tool.Tool instances.
//
//	client, session, err := sdk.NewClient(ctx, transport)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer session.Close()
//
//	tools, err := sdk.FromSession(ctx, session)
//	// tools are native tool.Tool instances backed by the remote MCP server
//
// # Type Conversions
//
// The package handles bidirectional conversion between Beluga and SDK types:
//   - tool.Tool to MCP SDK tool definitions (server side)
//   - MCP SDK CallToolResult to tool.Result (client side)
//   - schema.TextPart to/from sdkmcp.TextContent
package sdk
