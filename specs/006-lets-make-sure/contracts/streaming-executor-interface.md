# Contract: Streaming Executor Interface

**Package**: `pkg/agents/internal/executor`  
**Type**: Interface Contract  
**Status**: Design Phase

## Interface Definition

```go
// StreamingExecutor extends Executor with streaming execution capabilities
type StreamingExecutor interface {
    Executor  // Embed existing Executor interface
    
    // ExecuteStreamingPlan executes a plan with streaming LLM responses
    // Returns a channel of ExecutionChunk that will be closed when execution completes
    ExecuteStreamingPlan(ctx context.Context, agent StreamingAgent, plan []schema.Step) (<-chan ExecutionChunk, error)
}
```

## Type Definitions

```go
// ExecutionChunk represents a chunk of execution output
type ExecutionChunk struct {
    Step        schema.Step           // Current step being executed
    Content     string                // Text content from this step
    ToolResult  *ToolExecutionResult  // Tool result if step executed tool
    FinalAnswer *schema.FinalAnswer   // Final answer if execution complete
    Err         error                 // Error if occurred (execution ends on error)
    Timestamp   time.Time             // Chunk timestamp for latency measurement
}

// ToolExecutionResult represents the result of tool execution
type ToolExecutionResult struct {
    ToolName    string                // Name of tool executed
    Input       map[string]any        // Tool input
    Output      map[string]any        // Tool output
    Duration    time.Duration         // Execution duration
    Err         error                 // Error if tool execution failed
}
```

## Contract Requirements

### Input Requirements
- `ctx` must not be nil
- `agent` must implement `StreamingAgent` interface
- `plan` must not be empty
- Plan steps must be valid (as defined by schema)

### Output Requirements
- Channel must be non-nil if error is nil
- Channel must be closed when execution completes (success or error)
- At least one chunk must be sent (for first step)
- Final chunk must have either `FinalAnswer` set or `Err` set
- Chunks must be sent in order (one per step)

### Behavior Requirements
- Execute steps sequentially (one at a time)
- Stream content from each step as it arrives
- Execute tools when step requires tool execution
- Include tool results in chunks
- Handle interruptions via context cancellation
- Error handling must follow Op/Err/Code pattern

### Performance Requirements
- First chunk must arrive within 200ms of call
- Step execution must not block on streaming
- Tool execution must complete within reasonable time
- Memory usage must be bounded

## Execution Flow

```
ExecuteStreamingPlan called
    ↓
For each step in plan:
    ↓
    Call agent.StreamPlan() for this step
    ↓
    Stream chunks from agent
    ↓
    If step requires tool:
        Execute tool
        Include tool result in chunk
    ↓
    Send chunk with step info
    ↓
    If final step:
        Send chunk with FinalAnswer
        Close channel
    ↓
    If error:
        Send chunk with Err
        Close channel
```

## Error Codes

- `ErrCodeInvalidPlan`: Plan validation failed
- `ErrCodeAgentNotStreaming`: Agent doesn't support streaming
- `ErrCodeStepExecutionError`: Error executing a step
- `ErrCodeToolExecutionError`: Error executing a tool
- `ErrCodeStreamInterrupted`: Execution interrupted by context cancellation
- `ErrCodeTimeout`: Execution exceeded timeout

## Test Contract

### Must Pass Tests
1. **Basic Execution**: ExecuteStreamingPlan returns channel, chunks arrive, channel closes
2. **Step Execution**: Each step produces a chunk with step info
3. **Tool Execution**: Tool execution included in chunks with results
4. **Final Answer**: Final chunk has FinalAnswer set
5. **Error Handling**: Errors are sent as final chunk with Err set
6. **Context Cancellation**: Cancelling context closes channel gracefully
7. **Sequential Execution**: Steps executed in order
8. **Performance**: First chunk arrives within 200ms

### Must Fail Tests (No Implementation Yet)
- All tests must fail initially (TDD approach)
- Tests validate contract, not implementation
- Implementation must make tests pass

## Implementation Notes

- Must use existing `Executor` implementation as base
- Must integrate with existing tool execution framework
- Must use existing error handling patterns
- Must record OTEL metrics for execution operations
- Must handle backpressure (don't block on channel send)

