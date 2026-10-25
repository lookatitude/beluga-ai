// Package composio provides a Composio MCP integration for the Beluga AI
// protocol layer. It connects to the Composio API for tool discovery and
// execution, wrapping Composio tools as native tool.Tool instances.
//
// Composio provides access to hundreds of integrations and actions through
// its unified API, which can be consumed as MCP-compatible tools. Each
// Composio action is wrapped as a Beluga tool.Tool, enabling seamless
// integration with Beluga agents.
//
// # API version
//
// This client targets Composio's legacy v1 REST API (/api/v1/actions).
// Composio's current API is v3; the client has not been migrated, and the
// action listing is not paginated. Tool output is returned as JSON text.
//
// # Usage
//
//	client, err := composio.New(
//	    composio.WithAPIKey("cmp-..."),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	tools, err := client.ListTools(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use Composio tools with a Beluga agent
//	myAgent := agent.New("assistant", agent.WithTools(tools...))
//
// # Configuration
//
// The client is configured via functional options:
//
//   - WithAPIKey(key) — sets the Composio API key (required)
//   - WithBaseURL(url) — overrides the default API base URL
//   - WithTimeout(d) — sets the HTTP client timeout (default: 30s)
//
// # Key Types
//
//   - Client — connects to the Composio API for tool operations
//   - Option — functional option for configuring the Client
package composio
