---
title: "Handling Tool Failures & Hallucinations"
description: "Recipe for robust tool execution in Go agents — validate tool calls, handle errors gracefully, and enable LLM self-correction with Beluga AI middleware."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go agent recipe, tool failure handling, LLM hallucination, agent self-correction, tool validation, error recovery"
---

## Problem

You need to handle cases where agents call tools that don't exist, provide invalid arguments, or when the LLM hallucinates tool calls that aren't actually available, without breaking the agent's execution flow.

## Solution

Implement a robust tool execution wrapper that validates tool calls, handles errors gracefully, provides feedback to the agent about failures, and allows the agent to retry with corrected tool calls. This works because Beluga AI's agent hooks intercept every tool call before execution, enabling validation and self-correction.

## Why This Matters

In agentic systems, tool execution failures are inevitable. Network requests time out, external APIs go down, and LLMs occasionally generate malformed tool calls. The critical insight is that failures should produce structured feedback rather than crash the agent loop. When an agent receives a clear error message explaining what went wrong, it can self-correct on the next iteration.

This recipe implements two layers of defense: existence validation (does the tool exist?) through `OnToolCall` hooks, and retry logic through `tool.WithRetry` middleware. Each layer uses `core.IsRetryable()` to distinguish between transient errors that should be retried and permanent errors that require a different approach.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

// buildResilientRegistry creates a tool registry where every tool is wrapped
// with retry middleware for transient failures.
func buildResilientRegistry(maxRetries int, tools ...tool.Tool) (*tool.Registry, error) {
	reg := tool.NewRegistry()
	for _, t := range tools {
		// Wrap each tool with retry middleware.
		resilient := tool.ApplyMiddleware(t, tool.WithRetry(maxRetries))
		if err := reg.Add(resilient); err != nil {
			return nil, fmt.Errorf("register tool %q: %w", t.Name(), err)
		}
	}
	return reg, nil
}

func main() {
	ctx := context.Background()

	// Define tools.
	type SearchInput struct {
		Query string `json:"query" description:"Search query" required:"true"`
	}
	search := tool.NewFuncTool("search", "Search the web",
		func(ctx context.Context, input SearchInput) (*tool.Result, error) {
			// May fail transiently due to network issues.
			return tool.TextResult("results for: " + input.Query), nil
		},
	)

	type WeatherInput struct {
		City string `json:"city" description:"City name" required:"true"`
	}
	weather := tool.NewFuncTool("get_weather", "Get current weather",
		func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
			return tool.TextResult(fmt.Sprintf("Weather in %s: sunny, 22°C", input.City)), nil
		},
	)

	// Build registry with retry middleware applied to every tool.
	reg, err := buildResilientRegistry(3, search, weather)
	if err != nil {
		slog.Error("registry setup failed", "error", err)
		return
	}

	// Use OnToolCall hook to validate that the tool exists before execution.
	// This catches hallucinated tool names before they cause confusing errors.
	a := agent.New("assistant",
		agent.WithTools(reg.All()),
		agent.WithHooks(agent.Hooks{
			OnToolCall: func(ctx context.Context, call agent.ToolCallInfo) error {
				if _, err := reg.Get(call.Name); err != nil {
					// Return error to agent; LLM sees this and can self-correct.
					available := reg.List()
					return fmt.Errorf("tool %q not found; available tools: %v", call.Name, available)
				}
				return nil
			},
			OnToolResult: func(ctx context.Context, call agent.ToolCallInfo, result *tool.Result) error {
				if result != nil && result.IsError {
					slog.Warn("tool returned error result",
						"tool", call.Name,
						"call_id", call.CallID,
					)
				}
				return nil
			},
		}),
	)

	result, err := a.Invoke(ctx, "Search for Go concurrency patterns and get Tokyo weather")
	if err != nil {
		// Check if retryable before giving up.
		if core.IsRetryable(err) {
			slog.Warn("retryable error from agent", "error", err)
		} else {
			slog.Error("agent failed with permanent error", "error", err)
		}
		return
	}
	fmt.Println(result)
}
```

## Explanation

1. **Registry as the source of truth** -- The `OnToolCall` hook looks up the called tool in the same registry that was passed to the agent. If the LLM hallucinates a tool name, the hook returns a descriptive error that the agent loop feeds back to the LLM, enabling self-correction.

2. **Retry middleware on tools** -- `tool.WithRetry(n)` wraps each tool so transient errors (network timeouts, 5xx responses) are retried automatically. The retry logic calls `core.IsRetryable()` internally to avoid retrying permanent failures like authentication errors.

3. **Structured error feedback** -- By returning an error from `OnToolCall`, you give the agent a clear error message including the list of valid tool names. This is more helpful than a raw "tool not found" panic and allows the LLM to adjust its next action.

4. **OnToolResult for observability** -- The `OnToolResult` hook runs after every successful tool execution and receives the `tool.Result`. Use it to log domain errors (`result.IsError == true`) for debugging without blocking execution.

## Testing

```go
func TestResilientRegistry_RejectsUnknownTool(t *testing.T) {
	reg, err := buildResilientRegistry(1)
	if err != nil {
		t.Fatal(err)
	}

	_, lookupErr := reg.Get("nonexistent_tool")
	if lookupErr == nil {
		t.Error("expected error for unknown tool, got nil")
	}
}
```

## Variations

### Tool Call Sanitization

Sanitize tool call arguments before validation to prevent injection:

```go
OnToolCall: func(ctx context.Context, call agent.ToolCallInfo) error {
	// Validate argument JSON length to prevent oversized payloads.
	const maxArgLen = 4096
	if len(call.Arguments) > maxArgLen {
		return fmt.Errorf("tool %q: arguments too large (%d bytes, max %d)",
			call.Name, len(call.Arguments), maxArgLen)
	}
	return nil
},
```

### Tool Result Caching

Cache deterministic tool results to avoid redundant network calls:

```go
func cachedTool(inner tool.Tool, cache map[string]*tool.Result) tool.Tool {
	return tool.ApplyMiddleware(inner, func(next tool.Tool) tool.Tool {
		return &cachingWrapper{inner: next, cache: cache}
	})
}
```

## Related Recipes

- **[Streaming Tool Calls](/docs/cookbook/llm/streaming-tool-calls)** -- Handle tools in streaming
- **[Parallel Step Execution](/docs/cookbook/agents/agents-parallel-execution)** -- Execute steps in parallel
