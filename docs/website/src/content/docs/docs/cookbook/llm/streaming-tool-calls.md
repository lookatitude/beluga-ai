---
title: "Streaming Tool Call Handler"
description: "Recipe for handling tool calls during streaming LLM responses in Go — detect, accumulate fragments, execute tools, and stream results back with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, streaming tool calls, Go LLM tools, concurrent tool execution, streaming agent, tool call handling, real-time AI"
---

## Problem

You need to handle tool calls that arrive during streaming LLM responses, executing tools as they complete and returning results to the model.

## Solution

Accumulate `schema.ToolCall` fragments from streaming chunks by ID. When `FinishReason` is `"tool_calls"`, all tool calls are complete and can be executed. This works because Beluga AI's streaming interface provides `schema.StreamChunk` values with a `ToolCalls` field that accumulates argument fragments.

## Why This Matters

LLM tool calls during streaming arrive as fragments: the first chunk might contain the tool name and ID, while subsequent chunks contain pieces of the arguments JSON. You cannot execute a tool until you have the complete arguments. This recipe accumulates fragments by call ID, and when `FinishReason` indicates tool call completion, all accumulated calls are executed — potentially concurrently for multiple tool calls.

The `arguments` field accumulates across chunks so JSON that is split at arbitrary byte boundaries by the streaming protocol is correctly reassembled before execution.

## Code Example

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// pendingCall accumulates tool call fragments from the stream.
type pendingCall struct {
	id        string
	name      string
	arguments string // JSON fragments accumulated across chunks.
}

// StreamingToolExecutor collects streaming tool call fragments and executes them.
type StreamingToolExecutor struct {
	registry *tool.Registry
	pending  map[string]*pendingCall // keyed by call ID
}

// NewStreamingToolExecutor creates a tool executor for use with a streaming model.
func NewStreamingToolExecutor(reg *tool.Registry) *StreamingToolExecutor {
	return &StreamingToolExecutor{
		registry: reg,
		pending:  make(map[string]*pendingCall),
	}
}

// ProcessChunk accumulates tool call data from a streaming chunk.
// Returns completed tool results when FinishReason indicates tool_calls.
func (e *StreamingToolExecutor) ProcessChunk(ctx context.Context, chunk schema.StreamChunk) ([]*tool.Result, error) {
	// Accumulate tool call fragments by ID.
	for _, tc := range chunk.ToolCalls {
		if tc.ID != "" {
			if _, exists := e.pending[tc.ID]; !exists {
				e.pending[tc.ID] = &pendingCall{id: tc.ID, name: tc.Name}
			}
		}
		// Accumulate arguments across chunks (fragments arrive incrementally).
		if p, ok := e.pending[tc.ID]; ok {
			p.arguments += tc.Arguments
			if tc.Name != "" {
				p.name = tc.Name
			}
		}
	}

	// Execute all pending tool calls when the finish reason signals completion.
	if chunk.FinishReason != "tool_calls" {
		return nil, nil
	}

	var results []*tool.Result
	for id, call := range e.pending {
		t, err := e.registry.Get(call.name)
		if err != nil {
			slog.Warn("unknown tool in stream", "name", call.name, "id", id)
			results = append(results, tool.ErrorResult(fmt.Errorf("unknown tool %q", call.name)))
			continue
		}

		// Unmarshal the accumulated JSON arguments.
		var args map[string]any
		if call.arguments != "" {
			if err := json.Unmarshal([]byte(call.arguments), &args); err != nil {
				results = append(results, tool.ErrorResult(fmt.Errorf("unmarshal args for %q: %w", call.name, err)))
				continue
			}
		}

		result, err := t.Execute(ctx, args)
		if err != nil {
			results = append(results, tool.ErrorResult(err))
		} else {
			results = append(results, result)
		}
	}

	// Clear pending calls after execution.
	e.pending = make(map[string]*pendingCall)
	return results, nil
}

func main() {
	ctx := context.Background()

	reg := tool.NewRegistry()
	type CalcInput struct {
		Expression string `json:"expression" description:"Math expression to evaluate" required:"true"`
	}
	if err := reg.Add(tool.NewFuncTool("calculate", "Evaluate a math expression",
		func(ctx context.Context, input CalcInput) (*tool.Result, error) {
			return tool.TextResult("42"), nil
		},
	)); err != nil {
		slog.Error("tool registration failed", "error", err)
		return
	}

	model, err := llm.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	// Bind tool definitions to the model.
	defs := make([]schema.ToolDefinition, 0, len(reg.All()))
	for _, t := range reg.All() {
		defs = append(defs, tool.ToDefinition(t))
	}
	boundModel := model.BindTools(defs)

	msgs := []schema.Message{
		schema.NewHumanMessage("What is 6 * 7?"),
	}

	executor := NewStreamingToolExecutor(reg)

	for chunk, err := range boundModel.Stream(ctx, msgs) {
		if err != nil {
			slog.Error("stream error", "error", err)
			break
		}

		// Print text deltas as they arrive.
		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
		}

		// Process tool call fragments; execute when complete.
		results, err := executor.ProcessChunk(ctx, chunk)
		if err != nil {
			slog.Error("tool execution error", "error", err)
			break
		}
		for _, r := range results {
			data, _ := json.Marshal(r.Content)
			fmt.Printf("\n[tool result: %s]\n", data)
		}
	}
	fmt.Println()
}
```

## Explanation

1. **Fragment accumulation by ID** -- Each tool call has an `ID` that uniquely identifies it across chunks. The executor stores a `pendingCall` per ID and appends argument fragments as they arrive. Without this accumulation, JSON split across chunk boundaries would fail to parse.

2. **Execution on `FinishReason == "tool_calls"`** -- This signals that all tool call arguments are complete. Only at this point is it safe to unmarshal the JSON and execute the tools. Executing before this point risks partial JSON.

3. **Registry-based lookup** -- `registry.Get(name)` returns `(Tool, error)`. Unknown tool names produce an `ErrorResult` that the caller can log or return to the model for self-correction.

4. **Clearing pending state** -- After execution, `pending` is reset so the executor can handle subsequent tool call rounds in a multi-turn agent loop.

## Variations

### Concurrent Tool Execution

When the model requests multiple tools in one response, execute them concurrently:

```go
func (e *StreamingToolExecutor) ProcessChunkConcurrent(ctx context.Context, chunk schema.StreamChunk) ([]*tool.Result, error) {
	// Accumulate fragments (same as above).
	// ...

	if chunk.FinishReason != "tool_calls" {
		return nil, nil
	}

	type result struct {
		r   *tool.Result
		idx int
	}
	calls := make([]*pendingCall, 0, len(e.pending))
	for _, c := range e.pending {
		calls = append(calls, c)
	}

	resultCh := make(chan result, len(calls))
	for i, call := range calls {
		go func(i int, call *pendingCall) {
			t, err := e.registry.Get(call.name)
			if err != nil {
				resultCh <- result{r: tool.ErrorResult(err), idx: i}
				return
			}
			var args map[string]any
			if call.arguments != "" {
				_ = json.Unmarshal([]byte(call.arguments), &args)
			}
			r, execErr := t.Execute(ctx, args)
			if execErr != nil {
				r = tool.ErrorResult(execErr)
			}
			resultCh <- result{r: r, idx: i}
		}(i, call)
	}

	results := make([]*tool.Result, len(calls))
	for range calls {
		res := <-resultCh
		results[res.idx] = res.r
	}
	e.pending = make(map[string]*pendingCall)
	return results, nil
}
```

## Related Recipes

- **[Streaming Metadata](/docs/cookbook/llm/streaming-metadata)** -- Capture token counts and finish reasons from streams
- **[Tool Middleware](/docs/cookbook/agents/tool-recipes)** -- Apply retry and auth middleware to tools
