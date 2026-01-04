# Contract: Streaming Agent Interface

**Package**: `pkg/agents/iface`  
**Type**: Interface Contract  
**Status**: Design Phase

## Interface Definition

```go
// StreamingAgent extends Agent with streaming execution capabilities
type StreamingAgent interface {
    Agent  // Embed existing Agent interface
    
    // StreamExecute executes agent with streaming LLM responses
    // Returns a channel of AgentStreamChunk that will be closed when execution completes
    StreamExecute(ctx context.Context, inputs map[string]any) (<-chan AgentStreamChunk, error)
    
    // StreamPlan plans next action with streaming model responses
    // Returns a channel of AgentStreamChunk that will be closed when planning completes
    StreamPlan(ctx context.Context, intermediateSteps []IntermediateStep, inputs map[string]any) (<-chan AgentStreamChunk, error)
}
```

## Type Definitions

```go
// AgentStreamChunk represents a chunk of agent execution output
type AgentStreamChunk struct {
    Content      string              // Text content from LLM (can be partial)
    ToolCalls    []schema.ToolCall   // Tool calls if any (may be partial)
    Action       *AgentAction        // Next action if determined
    Finish       *AgentFinish        // Final result if complete
    Err          error               // Error if occurred (stream ends on error)
    Metadata     map[string]any      // Additional metadata (latency, timestamps, etc.)
}
```

## Contract Requirements

### Input Requirements
- `ctx` must not be nil
- `inputs` must contain all required input variables (as defined by `InputVariables()`)
- Context cancellation must be respected (stream must close on `ctx.Done()`)

### Output Requirements
- Channel must be non-nil if error is nil
- Channel must be closed when stream completes (success or error)
- At least one chunk must be sent before closing (unless error occurs immediately)
- Final chunk must have either `Finish` set or `Err` set
- Chunks must be sent in order

### Behavior Requirements
- Streaming must start immediately (no blocking on first chunk)
- Chunks must be sent as soon as available from LLM
- Tool calls must be sent as soon as detected
- Interruption via context cancellation must be handled gracefully
- Error handling must follow Op/Err/Code pattern

### Performance Requirements
- First chunk must arrive within 200ms of call
- Subsequent chunks must arrive within 100ms of previous chunk
- Stream must complete within reasonable time (no infinite streams)
- Memory usage must be bounded (no unbounded buffering)

## Error Codes

- `ErrCodeStreamingNotSupported`: Agent doesn't support streaming
- `ErrCodeInvalidInput`: Input validation failed
- `ErrCodeStreamInterrupted`: Stream interrupted by context cancellation
- `ErrCodeStreamError`: Error during streaming execution
- `ErrCodeLLMError`: Error from LLM provider

## Test Contract

### Must Pass Tests
1. **Basic Streaming**: StreamExecute returns channel, chunks arrive, channel closes
2. **Context Cancellation**: Cancelling context closes channel gracefully
3. **Error Handling**: Errors are sent as final chunk with Err set
4. **Tool Calls**: Tool calls are included in chunks when detected
5. **Final Answer**: Final chunk has Finish set with return values
6. **Input Validation**: Invalid inputs return error before streaming starts
7. **Concurrent Calls**: Multiple StreamExecute calls work independently
8. **Performance**: First chunk arrives within 200ms

### Must Fail Tests (No Implementation Yet)
- All tests must fail initially (TDD approach)
- Tests validate contract, not implementation
- Implementation must make tests pass

## Implementation Notes

- Implementations must use existing `StreamChat` from `pkg/llms`
- Implementations must handle backpressure (don't block on channel send)
- Implementations must clean up resources on context cancellation
- Implementations must record OTEL metrics for streaming operations

