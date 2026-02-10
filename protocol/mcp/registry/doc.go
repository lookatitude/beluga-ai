// Package registry provides MCP server discovery and tool aggregation for the
// Beluga AI framework. It discovers MCP servers from registry endpoints or
// static configuration, lists their available tools, and makes them accessible
// as native tool.Tool instances.
//
// The registry maintains a list of known MCP servers and provides methods to
// connect to them, initialize MCP sessions, and aggregate their tools into a
// unified collection. It is safe for concurrent use.
//
// # Basic Usage
//
//	reg := registry.New()
//	reg.AddServer("search", "http://localhost:8081/mcp")
//	reg.AddServer("code", "http://localhost:8082/mcp")
//
//	tools, err := reg.DiscoverTools(ctx)
//	// tools contains all tools from all registered MCP servers
//
// # Server Management
//
// Servers can be added, removed, and filtered by tags:
//
//	reg.AddServer("search", "http://search:8080/mcp", "production", "search")
//	reg.AddServer("dev-tools", "http://dev:8080/mcp", "development")
//
//	prodServers := reg.ServersByTag("production")
//	reg.RemoveServer("dev-tools")
//
// # Selective Discovery
//
// Tools can be discovered from all servers or from a specific server:
//
//	// All servers
//	allTools, err := reg.DiscoverTools(ctx)
//
//	// Specific server
//	searchTools, err := reg.DiscoverToolsFromServer(ctx, "search")
//
// # Key Types
//
//   - Registry — manages MCP server discovery and tool aggregation
//   - ServerEntry — describes a registered MCP server (name, URL, tags)
//   - DiscoveredTool — wraps a tool.Tool with server provenance metadata
//   - MCPClientInterface — abstracts MCP client operations for testability
package registry
