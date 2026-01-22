# Building an MCP Server for Tools

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement the Model Context Protocol (MCP) to expose your Beluga AI tools to other agents and IDEs like Cursor. You'll learn how to adapt existing tools to the MCP standard and run a dedicated MCP server.

## Learning Objectives
- ✅ Understand MCP (Model Context Protocol)
- ✅ Expose Beluga Tools as MCP Resources/Tools
- ✅ Run an MCP Server

## Introduction
Welcome, colleague! MCP is quickly becoming the industry standard for how AI models interact with the outside world. Instead of building custom integrations for every platform, let's build one MCP server that makes our Go tools instantly available to Claude, Cursor, and beyond.

## Prerequisites

- [Working with Tools](../../getting-started/04-working-with-tools.md)
- `pkg/server` package

## What is MCP?

MCP is a standard for exposing context (resources) and capabilities (tools) to LLMs. Instead of proprietary APIs, you build one MCP server that works with Claude Desktop, Cursor, and Beluga Agents.

## Step 1: Define Tools

Standard Beluga Tools.
```text
go
go
weatherTool := tools.NewWeatherTool(apiKey)
dbTool := tools.NewDatabaseTool(db)
```

## Step 2: Create MCP Server

Beluga provides helpers to adapt `Tool` interface to MCP.
```go
import (
    "github.com/lookatitude/beluga-ai/pkg/server/mcp"
)
go
func main() {
    // Create server
    s := mcp.NewServer("my-tools-server", "1.0.0")

    

    // Register tools
    s.RegisterTool(weatherTool)
    s.RegisterTool(dbTool)
    
    // Start (usually via Stdio for local integration)
    s.ServeStdio()
}
```

## Step 3: Resources (Context)

MCP also supports "Resources" (read-only data).
s.RegisterResource("config://db", func() string \{
```text
    return "Database Schema..."
})


## Step 4: Connecting to Cursor/Claude

Configure the client to run your binary.

**claude_desktop_config.json**:{
  "mcpServers": {
    "my-beluga-tools": {
      "command": "/path/to/your/server_binary"
    }
  }
}
```

## Verification

1. Build your server.
2. Add to Cursor/Claude config.
3. Open the AI pane.
4. Ask "What's the weather?" -> Verify it calls your Go tool.

## Next Steps

- **[Deploying via REST](./server-rest-deployment.md)** - Alternative deployment
- **[Working with Tools](../../getting-started/04-working-with-tools.md)** - Build more tools
