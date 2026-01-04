# Contract: Voice Session Agent Integration

**Package**: `pkg/voice/session`  
**Type**: Integration Contract  
**Status**: Design Phase

## Integration Point

Voice sessions must accept agent instances (not just callbacks) and integrate them with session lifecycle.

## Interface Extensions

```go
// VoiceSessionOption extends existing options
type VoiceSessionOption func(*VoiceOptions)

// WithAgentInstance sets an agent instance for the voice session
func WithAgentInstance(agent iface.StreamingAgent, config *AgentConfig) VoiceSessionOption

// VoiceSession interface extension (conceptual, not breaking change)
// Existing methods unchanged, new behavior when agent instance is provided
```

## Contract Requirements

### Agent Instance Integration
- Agent instance must implement `StreamingAgent` interface
- Agent instance must be set before session starts
- Agent instance lifecycle tied to session lifecycle
- Agent instance must be cleaned up on session end

### Streaming Integration
- When agent instance provided, use streaming execution instead of callback
- Stream chunks from agent to TTS conversion
- Handle interruptions by cancelling agent stream
- Preserve conversation context across interruptions

### Conversation Context
- Maintain conversation history in session context
- Update history with each agent response
- Include tool execution results in context
- Preserve context across interruptions and reconnections

### Error Handling
- Agent errors must not crash voice session
- Agent errors must be logged and reported via metrics
- Session must handle agent unavailability gracefully
- Session must recover from agent errors

## Behavior Contract

### Session Start
1. Validate agent instance (if provided)
2. Initialize agent context
3. Start agent in idle state
4. Ready to receive user input

### User Input Processing
1. Receive audio from transport
2. Convert to transcript via STT
3. If agent instance: call `StreamExecute` with transcript
4. If callback: use existing callback mechanism (backward compatibility)

### Streaming Response Handling
1. Receive chunks from agent stream
2. Detect sentence boundaries
3. Convert sentences to TTS
4. Stream audio to transport
5. Update conversation context

### Interruption Handling
1. Detect new user input during streaming
2. Cancel agent stream context
3. Stop TTS generation
4. Preserve conversation context
5. Process new input immediately

### Session End
1. Cancel any active agent streams
2. Clean up agent resources
3. Finalize conversation context
4. Close agent instance (if managed by session)

## Performance Contract

### Latency Requirements
- End-to-end latency < 500ms (user speech → agent spoken response)
- Agent stream start < 50ms after transcript received
- First TTS audio < 200ms after first agent chunk
- Interruption response < 10ms

### Throughput Requirements
- Support multiple concurrent sessions
- Each session independent (no interference)
- Handle backpressure gracefully
- No memory leaks over long sessions

## Error Contract

### Error Codes
- `ErrCodeAgentNotSet`: Agent instance required but not provided
- `ErrCodeAgentInvalid`: Agent instance doesn't implement required interface
- `ErrCodeStreamError`: Error during agent streaming
- `ErrCodeContextError`: Error managing conversation context
- `ErrCodeInterruptionError`: Error during interruption handling

### Error Handling
- Errors must follow Op/Err/Code pattern
- Errors must be logged with context
- Errors must be reported via OTEL metrics
- Errors must not crash session (graceful degradation)

## Test Contract

### Must Pass Tests
1. **Agent Integration**: Session accepts agent instance, uses it for responses
2. **Streaming**: Agent stream chunks converted to TTS and sent to user
3. **Interruption**: New input interrupts agent stream, processes new input
4. **Context Preservation**: Conversation context preserved across interruptions
5. **Error Recovery**: Agent errors don't crash session, session recovers
6. **Backward Compatibility**: Sessions without agent instance still work (callback mode)
7. **Concurrent Sessions**: Multiple sessions with different agents work independently
8. **Performance**: End-to-end latency < 500ms

### Must Fail Tests (No Implementation Yet)
- All tests must fail initially (TDD approach)
- Tests validate contract, not implementation
- Implementation must make tests pass

## Migration Path

### Backward Compatibility
- Existing callback-based sessions continue to work
- New agent instance option is additive (doesn't break existing code)
- Gradual migration: can use callbacks or agent instances
- Callback mode deprecated but supported

### Migration Steps
1. Add `WithAgentInstance` option (non-breaking)
2. Update internal implementation to support both modes
3. Add tests for agent instance mode
4. Document migration guide
5. Eventually deprecate callback mode (future release)

