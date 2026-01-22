# Custom Tools and Tool Registry

Welcome, colleague! In this guide we'll implement **custom tools** and use the **tool registry** with Beluga AI agents. You'll define tools (embedding `BaseTool` or implementing `Tool`), register them with `InMemoryToolRegistry`, and pass them to `NewBaseAgent` for execution.

## What you will build

You will create one or more custom tools, register them in a tool registry, and create an agent that uses those tools. You'll run `Execute` (or `StreamExecute`) and verify that the agent can invoke your tools.

## Learning Objectives

- ✅ Implement the `Tool` interface (or embed `BaseTool`)
- ✅ Use `ToolDefinition`, `Name`, `Description`, `InputSchema`, and `Execute`
- ✅ Register tools with `InMemoryToolRegistry` and `List` / `Get`
- ✅ Create an agent with the tool list and run `Execute`

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- LLM or chat model for the agent

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Implement a Custom Tool

Embed `BaseTool` and override `Execute` and `Definition`:
```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
)

type GreetTool struct {
	tools.BaseTool
}

func NewGreetTool() *GreetTool {
	t := &GreetTool{}
	t.SetName("greet")
	t.SetDescription("Greets the user by name. Input: name (string).")
	t.SetInputSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
		"required": []string{"name"},
	})
	return t
}

func (t *GreetTool) Execute(ctx context.Context, input any) (any, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("greet: expected map, got %T", input)
	}
	name, _ := m["name"].(string)
	if name == "" {
		name = "Guest"
	}
	return fmt.Sprintf("Hello, %s!", name), nil
}

func (t *GreetTool) Definition() tools.ToolDefinition {
	return t.BaseTool.Definition()
}

## Step 3: Register Tools and Create Agent
	registry := tools.NewInMemoryToolRegistry()
	_ = registry.RegisterTool(NewGreetTool())
	// optionally more tools
	names := registry.ListTools()
	var agentTools []tools.Tool
	for _, n := range names {
		t, err := registry.GetTool(n)
		if err != nil {
			log.Fatalf("get tool %s: %v", n, err)
		}
		agentTools = append(agentTools, t)
	}

	llm := // ... your LLM or chat model
	agent, err := agents.NewBaseAgent("my-agent", llm, agentTools)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}

	inputs := map[string]any{"messages": []schema.Message{...}}
	result, err := agent.Execute(ctx, inputs)
```

## Step 4: Verify Tool Invocation

Run the agent with input that triggers a tool call (e.g. "Say hello to Alice"). Inspect the result or streamed chunks to confirm the tool was invoked and its output used.

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Registry | In-memory vs custom | `NewInMemoryToolRegistry` |
| Tools | Registered tools | Application-defined |
| Agent config | Timeout, retries, etc. | `agents.Config` |

## Common Issues

### "Tool not found" or "Execute not implemented"

**Problem**: Tool not registered or `Execute` not overridden.

**Solution**: Ensure you call `registry.Register(tool)` and that your tool implements `Execute`. Use `BaseTool` and override only what you need.

### "Input schema mismatch"

**Problem**: Agent or LLM sends input that doesn't match `InputSchema`.

**Solution**: Define `InputSchema` to match what the model produces. Validate and normalize in `Execute` if needed.

### "Agent ignores tools"

**Problem**: LLM not instructed to use tools or tools not bound.

**Solution**: Use a chat model that supports tool binding (`BindTools`). Ensure system prompt or config encourages tool use.

## Production Considerations

- **Validation**: Validate tool input in `Execute` and return clear errors.
- **Timeouts**: Use `context.WithTimeout` in `Execute` for external calls.
- **Observability**: Add OTEL spans around tool execution and registry ops.

## Next Steps

- **[MCP Tools Integration](./agents-mcp-tools-integration.md)** — Expose tools via MCP.
- **[Agents Tool Registry Tutorial](../../../tutorials/higher-level/agents-tools-registry.md)** — Tutorial.
- **[Working with Tools](../../../getting-started/04-working-with-tools.md)** — Getting started.
