---
title: "Handling Tool Failures & Hallucinations"
package: "agents"
category: "resilience"
complexity: "advanced"
---

# Handling Tool Failures & Hallucinations

## Problem

You need to handle cases where agents call tools that don't exist, provide invalid arguments, or when the LLM hallucinates tool calls that aren't actually available, without breaking the agent's execution flow.

## Solution

Implement a robust tool execution wrapper that validates tool calls, handles errors gracefully, provides feedback to the agent about failures, and allows the agent to retry with corrected tool calls. This works because Beluga AI's agent system supports tool result messages that can inform the agent about failures, enabling self-correction.

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
    
    "github.com/lookatitude/beluga-ai/pkg/agents/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

var tracer = otel.Tracer("beluga.agents.tool_handling")

// ToolExecutionHandler handles tool execution with error recovery
type ToolExecutionHandler struct {
    tools          map[string]tools.Tool
    maxRetries     int
    validateCalls  bool
}

// NewToolExecutionHandler creates a new tool execution handler
func NewToolExecutionHandler(toolList []tools.Tool, maxRetries int) *ToolExecutionHandler {
    toolMap := make(map[string]tools.Tool)
    for _, tool := range toolList {
        toolMap[tool.Name()] = tool
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
    tool, exists := teh.tools[toolCall.Name]
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
        if err := teh.validateArguments(ctx, tool, toolCall.Arguments); err != nil {
            errorMsg := fmt.Sprintf("Invalid arguments for tool '%s': %v. Expected schema: %s", 
                toolCall.Name, err, teh.getToolSchema(tool))
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
        result, execErr = tool.Execute(ctx, args)
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
func (teh *ToolExecutionHandler) validateArguments(ctx context.Context, tool tools.Tool, argumentsJSON string) error {
    var args map[string]interface{}
    if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }

    // Get tool description to check required arguments
    // This is a simplified validation - in practice, use tool schemas
    return nil
}

// getToolSchema returns tool schema information
func (teh *ToolExecutionHandler) getToolSchema(tool tools.Tool) string {
    // Return tool schema/description for error messages
    return tool.Description()
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
    // Network errors, timeouts are retryable
    // Validation errors, not found errors are not
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
    agent  iface.Agent
    handler *ToolExecutionHandler
}

// NewAgentWithToolHandling creates an agent with tool handling
func NewAgentWithToolHandling(agent iface.Agent, handler *ToolExecutionHandler) *AgentWithToolHandling {
    return &AgentWithToolHandling{
        agent:   agent,
        handler: handler,
    }
}

// Invoke executes agent with tool error handling
func (awth *AgentWithToolHandling) Invoke(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Intercept tool calls and handle errors
    // This is a simplified version - in practice, you'd hook into the agent's tool execution

    return awth.agent.Invoke(ctx, input)
}

func main() {
    ctx := context.Background()

    // Create tools
    toolList := []tools.Tool{
        // Add your tools
    }
    
    // Create handler
    handler := NewToolExecutionHandler(toolList, 3)
    
    // Use with agent
    // agent := yourAgent
    // agentWithHandling := NewAgentWithToolHandling(agent, handler)
    fmt.Println("Tool error handler created")
}
```

## Explanation

Let's break down what's happening:

1. **Tool existence validation** - Notice how we check if the tool exists before attempting execution. If it doesn't, we return a helpful error message listing available tools. This helps the agent self-correct by knowing what tools are actually available.

2. **Argument validation** - We validate arguments before execution. Invalid arguments return descriptive errors that help the agent understand what went wrong and how to fix it.

3. **Error message format** - Tool errors are returned as ToolMessage objects with structured JSON. This allows the agent to parse errors programmatically and adjust its behavior accordingly.

```go
**Key insight:** Always return structured error messages that the agent can understand and act upon. Generic errors like "failed" don't help the agent learn and self-correct.

## Testing

```
Here's how to test this solution:
```go
func TestToolExecutionHandler_HandlesMissingTool(t *testing.T) {
    handler := NewToolExecutionHandler([]tools.Tool{}, 0)
    
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

## Variations

### Tool Call Sanitization

Sanitize tool calls to prevent injection:
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

- **[LLMs Streaming Tool Logic Handler](./llms-streaming-tool-logic-handler.md)** - Handle tools in streaming
- **[Agents Parallel Step Execution](./agents-parallel-step-execution.md)** - Execute steps in parallel
- **[Agents Package Guide](../guides/agent-types.md)** - For a deeper understanding of agents
