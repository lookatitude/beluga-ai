# Agents Tool Registry and Custom Tools

**What you will build:** An agent that uses the **tool registry** and **custom tools** from `pkg/agents/tools`. You'll create tools (embedding `BaseTool` or implementing `Tool`), register them with `InMemoryToolRegistry`, pass them to `NewBaseAgent`, and run `Execute` or `StreamExecute`.

## Learning Objectives

- Create custom tools with `tools.BaseTool` and `Definition` / `Execute`
- Use `tools.NewInMemoryToolRegistry` and register tools
- Wire the registry (or a tool list) into `agents.NewBaseAgent`
- Run `Execute` and optionally `StreamExecute` with tool calls

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- [First Agent](../../getting-started/03-first-agent.md) and [Working with Tools](../../getting-started/04-working-with-tools.md)

## Step 1: Define a Custom Tool

Implement `Tool` (or embed `BaseTool` and override `Execute`):
```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

type EchoTool struct {
	tools.BaseTool
}

func NewEchoTool() *EchoTool {
	t := &EchoTool{}
	t.SetName("echo")
	t.SetDescription("Echoes the input back")
	t.SetInputSchema(map[string]any{"type": "object", "properties": map[string]any{"input": map[string]any{"type": "string"}}})
	return t
}

func (t *EchoTool) Execute(ctx context.Context, input any) (any, error) {
	return fmt.Sprintf("Echo: %v", input), nil
}

func (t *EchoTool) Definition() tools.ToolDefinition {
	return t.BaseTool.Definition()
}

## Step 2: Create a Tool Registry and Register Tools
	registry := tools.NewInMemoryToolRegistry()
	_ = registry.RegisterTool(NewEchoTool())
	// Add more tools
	names := registry.ListTools()
	var agentTools []tools.Tool
	for _, n := range names {
		t, _ := registry.GetTool(n)
		agentTools = append(agentTools, t)
	}

## Step 3: Create an Agent and Execute
	import "github.com/lookatitude/beluga-ai/pkg/agents"

	// llm implements llms.LLMCaller or use a chat model
	var llm any
	agent, err := agents.NewBaseAgent("my-agent", llm, agentTools)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
	result, err := agent.Execute(ctx, inputs)
```

## Step 4: StreamExecute (Optional)

If your agent and LLM support streaming, use `StreamExecute` and process chunks (e.g. content, tool calls).

## Verification

1. Register at least one tool and create an agent.
2. Run `Execute` with input that triggers a tool call.
3. Assert the result (or streamed chunks) include the tool output.

## Next Steps

- **[MCP Tools Integration](../../integrations/agents/agents-mcp-tools-integration.md)** — Expose tools via MCP server.
- **[Custom Tools Registry](../../integrations/agents/agents-custom-tools-registry.md)** — More registry patterns.
- **[Multi-Agent Orchestration](./agents-multi-agent-orchestration.md)** — Coordinate multiple agents.
