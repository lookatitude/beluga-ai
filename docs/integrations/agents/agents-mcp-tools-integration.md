# Agents and MCP Server Integration

Welcome, colleague! In this guide we'll integrate Beluga AI **agents** with the **MCP (Model Context Protocol) server**. You'll expose agent tools as MCP tools, start the MCP server, and let MCP clients (e.g. IDEs, bots) discover and invoke those tools.

## What you will build

You will create an MCP server that exposes your agent's tools, optionally run an agent (or use a standalone tool registry), and connect MCP clients to it. This allows tools like Cursor, Claude Code, or other MCP clients to call your Beluga tools over the protocol.

## Learning Objectives

- ✅ Create an MCP server with `pkg/server` MCP provider
- ✅ Register agent tools (or tool registry) as MCP tools
- ✅ Start the server and verify tool discovery
- ✅ Invoke tools from an MCP client

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- MCP client (e.g. Cursor, MCP CLI) for testing

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Create Tools and Registry

Use `pkg/agents/tools` to define tools and `InMemoryToolRegistry`:
```text
go
go
	registry := tools.NewInMemoryToolRegistry()
	_ = registry.RegisterTool(NewEchoTool())
	// ... more tools
	names := registry.ListTools()
	var agentTools []tools.Tool
	for _, n := range names {
		t, _ := registry.GetTool(n)
		agentTools = append(agentTools, t)
	}
```

## Step 3: Configure MCP Server with Tools

The MCP server in `pkg/server` accepts MCP-specific config. Wire your tool list (or an agent that provides tools) into the MCP server configuration. Refer to `pkg/server/providers/mcp` and `MCPConfig` for host, port, and tool wiring.

```
	import (
		"github.com/lookatitude/beluga-ai/pkg/server"
	)

go
```go
	opts := []server.Option{
		server.WithMCPConfig(server.MCPConfig{
			Host: "localhost",
			Port: 8081,
			// Tool registration: pass agentTools or adapter
		}),
	}
	mcpSrv, err := server.NewMCPServer(opts...)
	if err != nil {
		log.Fatalf("mcp server: %v", err)
	}
	_ = mcpSrv
```

## Step 4: Start Server and Verify

Start the MCP server, then use an MCP client to list tools and call them. Ensure the client connects to the configured host/port.

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Host | MCP server bind address | localhost |
| Port | MCP server port | 8081 |
| Tools | Tool list or adapter | Application-defined |

## Common Issues

### "MCP server won't start"

**Problem**: Port in use or config validation fails.

**Solution**: Check host/port and that MCP config is valid. Ensure no other process uses the port.

### "Client doesn't see my tools"

**Problem**: Tools not registered or not exposed via MCP adapter.

**Solution**: Verify tool registration and that the MCP provider maps them to the protocol. Check server logs.

### "Tool call fails from client"

**Problem**: Input format or schema mismatch.

**Solution**: Align tool `InputSchema` with what the client sends. Validate and log tool inputs.

## Production Considerations

- **Auth**: Secure MCP endpoint (e.g. TLS, auth token) when exposed.
- **Rate limiting**: Limit tool invocations per client.
- **Observability**: Log and metric MCP requests and tool calls.

## Next Steps

- **[Custom Tools Registry](./agents-custom-tools-registry.md)** — Registry patterns and custom tools.
- **[Agents Tool Registry Tutorial](../../../tutorials/higher-level/agents-tools-registry.md)** — Tools and registry basics.
- **[Server MCP](../../../api/packages/server.md)** — MCP server API reference.
