# pkg/tools

The tools package provides implementations for tools that can be used by agents in the Beluga AI Framework.

## Overview

This package contains:

- **Base tool types** - `BaseTool` struct and common tool utilities
- **Tool registry** - Registry for managing tool instances
- **Built-in tools** - Ready-to-use tool implementations

## Sub-packages

| Package | Description |
|---------|-------------|
| `tools/api` | HTTP API request tool |
| `tools/gofunc` | Wrapper for Go functions as tools |
| `tools/mcp` | Minecraft server tools (ping, RCON) |
| `tools/shell` | Shell command execution tool |
| `tools/providers` | Built-in tool providers (echo, calculator) |

## Usage

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/tools"
    "github.com/lookatitude/beluga-ai/pkg/tools/api"
    "github.com/lookatitude/beluga-ai/pkg/tools/shell"
)

// Create an API tool
apiTool, err := api.NewAPITool(nil, 30*time.Second)

// Create a shell tool
shellTool, err := shell.NewShellTool(10*time.Second)

// Use with an agent
agent := agents.NewAgent(
    agents.WithTools(apiTool, shellTool),
)
```

## Creating Custom Tools

Embed `tools.BaseTool` to create custom tools:

```go
type MyTool struct {
    tools.BaseTool
}

func NewMyTool() *MyTool {
    t := &MyTool{}
    t.SetName("my_tool")
    t.SetDescription("Does something useful")
    t.SetInputSchema(map[string]any{
        "type": "object",
        "properties": map[string]any{
            "input": map[string]any{"type": "string"},
        },
    })
    return t
}

func (t *MyTool) Execute(ctx context.Context, input any) (any, error) {
    // Implementation
    return "result", nil
}
```

## Migration from pkg/agents/tools

If you were using `pkg/agents/tools`, update your imports:

```go
// Old
import "github.com/lookatitude/beluga-ai/pkg/agents/tools"

// New
import "github.com/lookatitude/beluga-ai/pkg/tools"
```

The API is identical - only the import path has changed.
