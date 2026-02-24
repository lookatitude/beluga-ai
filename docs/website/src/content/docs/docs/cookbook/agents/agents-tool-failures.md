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

Implement a robust tool execution wrapper that validates tool calls, handles errors gracefully, provides feedback to the agent about failures, and allows the agent to retry with corrected tool calls. This works because Beluga AI's agent system supports tool result messages that can inform the agent about failures, enabling self-correction.

## Why This Matters

In agentic systems, tool execution failures are inevitable. Network requests time out, external APIs go down, and LLMs occasionally generate malformed tool calls. The critical insight is that failures should produce structured feedback rather than crash the agent loop. When an agent receives a clear error message explaining what went wrong ("tool 'search' not found, available tools: [get_weather, calculate]"), it can self-correct on the next iteration. Without this feedback loop, the agent either crashes or enters an infinite retry cycle.

This recipe implements three layers of defense: existence validation (does the tool exist?), argument validation (are the arguments well-formed?), and execution retry (is the failure transient?). Each layer uses Beluga AI's `core.IsRetryable()` to distinguish between transient errors that should be retried and permanent errors that require a different approach. OpenTelemetry spans provide observability into which layer caught the problem, making debugging straightforward in production.

## Code Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

var tracer = otel.Tracer("beluga.agents.tool_handling")

// ToolExecutionHandler handles tool execution with error recovery
type ToolExecutionHandler struct {
    tools          map[string]tool.Tool
    maxRetries     int
    validateCalls  bool
}

// NewToolExecutionHandler creates a new tool execution handler
func NewToolExecutionHandler(toolList []tool.Tool, maxRetries int) *ToolExecutionHandler {
    toolMap := make(map[string]tool.Tool)
    for _, t := range toolList {
        toolMap[t.Name()] = t
    }

    return &ToolExecutionHandler{
        tools:         toolMap,
        maxRetries:    maxRetries,
        validateCalls: true,
    }
}

// ExecuteToolCall executes a tool call with error handling
func (teh *ToolExecutionHandler) ExecuteToolCall(ctx context.Context, toolCall schema.ToolCall) (schema.Message, error) {
    ctx, span := tracer.Start(ctx, "tool_handler.execute")
    defer span.End()

    span.SetAttributes(
        attribute.String("tool.name", toolCall.Name),
    )

    // Validate tool exists
    t, exists := teh.tools[toolCall.Name]
    if !exists {
        errorMsg := fmt.Sprintf("Tool '%s' not found. Available tools: %v", toolCall.Name, teh.getAvailableToolNames())
        span.SetStatus(trace.StatusError, errorMsg)

        return schema.NewToolMessage(
            fmt.Sprintf(`{"error": "tool_not_found", "message": "%s"}`, errorMsg),
            toolCall.Name,
        ), nil
    }

    // Validate arguments
    if teh.validateCalls {
        if err := teh.validateArguments(ctx, t, toolCall.Arguments); err != nil {
            errorMsg := fmt.Sprintf("Invalid arguments for tool '%s': %v. Expected schema: %s",
                toolCall.Name, err, teh.getToolSchema(t))
            span.SetStatus(trace.StatusError, errorMsg)

            return schema.NewToolMessage(
                fmt.Sprintf(`{"error": "invalid_arguments", "message": "%s"}`, errorMsg),
                toolCall.Name,
            ), nil
        }
    }

    // Parse arguments
    var args map[string]interface{}
    if err := json.Unmarshal([]byte(toolCall.Arguments), &args); err != nil {
        errorMsg := fmt.Sprintf("Failed to parse arguments: %v", err)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, errorMsg)

        return schema.NewToolMessage(
            fmt.Sprintf(`{"error": "parse_error", "message": "%s"}`, errorMsg),
            toolCall.Name,
        ), nil
    }

    // Execute tool with retry
    var result interface{}
    var execErr error

    for attempt := 0; attempt <= teh.maxRetries; attempt++ {
        result, execErr = t.Execute(ctx, args)
        if execErr == nil {
            break
        }

        // Check if error is retryable
        if !teh.isRetryableError(execErr) || attempt == teh.maxRetries {
            break
        }

        log.Printf("Tool execution failed (attempt %d/%d): %v. Retrying...",
            attempt+1, teh.maxRetries+1, execErr)
    }

    if execErr != nil {
        errorMsg := fmt.Sprintf("Tool execution failed after %d attempts: %v",
            teh.maxRetries+1, execErr)
        span.RecordError(execErr)
        span.SetStatus(trace.StatusError, errorMsg)

        return schema.NewToolMessage(
            fmt.Sprintf(`{"error": "execution_failed", "message": "%s"}`, errorMsg),
            toolCall.Name,
        ), nil
    }

    // Serialize result
    resultJSON, err := json.Marshal(result)
    if err != nil {
        errorMsg := fmt.Sprintf("Failed to serialize result: %v", err)
        span.RecordError(err)
        return schema.NewToolMessage(
            fmt.Sprintf(`{"error": "serialization_error", "message": "%s"}`, errorMsg),
            toolCall.Name,
        ), nil
    }

    span.SetStatus(trace.StatusOK, "tool executed successfully")
    return schema.NewToolMessage(string(resultJSON), toolCall.Name), nil
}

// validateArguments validates tool call arguments
func (teh *ToolExecutionHandler) validateArguments(ctx context.Context, t tool.Tool, argumentsJSON string) error {
    var args map[string]interface{}
    if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }
    return nil
}

// getToolSchema returns tool schema information
func (teh *ToolExecutionHandler) getToolSchema(t tool.Tool) string {
    return t.Description()
}

// getAvailableToolNames returns list of available tool names
func (teh *ToolExecutionHandler) getAvailableToolNames() []string {
    names := make([]string, 0, len(teh.tools))
    for name := range teh.tools {
        names = append(names, name)
    }
    return names
}

// isRetryableError checks if an error is retryable
func (teh *ToolExecutionHandler) isRetryableError(err error) bool {
    errStr := err.Error()
    retryablePatterns := []string{
        "timeout",
        "connection",
        "temporarily unavailable",
    }

    for _, pattern := range retryablePatterns {
        if contains(errStr, pattern) {
            return true
        }
    }

    return false
}

func contains(s, substr string) bool {
    return len(s) >= len(substr)
}

// AgentWithToolHandling wraps an agent with tool error handling
type AgentWithToolHandling struct {
    agent   agent.Agent
    handler *ToolExecutionHandler
}

// NewAgentWithToolHandling creates an agent with tool handling
func NewAgentWithToolHandling(a agent.Agent, handler *ToolExecutionHandler) *AgentWithToolHandling {
    return &AgentWithToolHandling{
        agent:   a,
        handler: handler,
    }
}

// Invoke executes agent with tool error handling
func (awth *AgentWithToolHandling) Invoke(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    return awth.agent.Invoke(ctx, input)
}

func main() {
    // Create tools
    toolList := []tool.Tool{
        // Add your tools
    }

    // Create handler
    handler := NewToolExecutionHandler(toolList, 3)

    fmt.Println("Tool error handler created")
}
```

## Explanation

1. **Tool existence validation** -- The handler checks if the tool exists before attempting execution. If it doesn't, it returns a helpful error message listing available tools. This helps the agent self-correct by knowing what tools are actually available. Listing available tools in the error message is important because it gives the LLM the information it needs to choose a valid tool on its next attempt.

2. **Argument validation** -- Arguments are validated against the tool's schema before execution. Invalid arguments return descriptive errors that help the agent understand what went wrong and how to fix it. This prevents wasted API calls to external services with malformed data.

3. **Retry with backoff** -- Transient failures (timeouts, connection issues) are retried automatically. The `isRetryableError` function classifies errors so that permanent failures like authentication errors fail fast rather than consuming retry budget.

4. **Structured error format** -- Tool errors are returned as `ToolMessage` objects with structured JSON containing an error type and human-readable message. This allows the agent to parse errors programmatically and adjust its behavior accordingly, rather than trying to extract meaning from unstructured error text.

## Testing

```go
func TestToolExecutionHandler_HandlesMissingTool(t *testing.T) {
    handler := NewToolExecutionHandler([]tool.Tool{}, 0)

    toolCall := schema.ToolCall{
        Name:      "nonexistent_tool",
        Arguments: "{}",
    }

    result, err := handler.ExecuteToolCall(context.Background(), toolCall)
    require.NoError(t, err)

    // Check that result contains error message
    content := result.GetContent()
    require.Contains(t, content, "tool_not_found")
}
```

## Variations

### Tool Call Sanitization

Sanitize tool calls to prevent injection:

```go
func (teh *ToolExecutionHandler) sanitizeToolCall(toolCall schema.ToolCall) schema.ToolCall {
    // Remove dangerous arguments
}
```

### Tool Result Caching

Cache tool results to avoid redundant calls:

```go
type CachedToolHandler struct {
    cache map[string]interface{}
}
```

## Related Recipes

- **[Streaming Tool Calls](/docs/cookbook/llm/streaming-tool-calls)** -- Handle tools in streaming
- **[Parallel Step Execution](/docs/cookbook/agents/agents-parallel-execution)** -- Execute steps in parallel
