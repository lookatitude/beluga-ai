# Research: Real-Time Voice Agent Support

**Date**: 2025-01-27  
**Feature**: Real-Time Voice Agent Support  
**Status**: Complete

## Research Objectives

1. Streaming agent execution patterns with voice session integration
2. Incremental TTS conversion from streaming LLM responses
3. Interruption handling for streaming agent responses
4. Backpressure handling in real-time audio processing
5. Conversation context management across streaming interactions

## Research Findings

### 1. Streaming Agent Execution Patterns

**Decision**: Use channel-based streaming with context cancellation for interruption support

**Rationale**: 
- Go's channel model provides natural backpressure handling
- Context cancellation enables clean interruption of streaming operations
- Existing `StreamChat` interface already returns `<-chan AIMessageChunk`
- Pattern aligns with existing LLM streaming implementations

**Alternatives Considered**:
- Callback-based streaming: Rejected - harder to manage lifecycle and cancellation
- Buffer-based streaming: Rejected - adds latency, defeats purpose of streaming
- Event-driven streaming: Rejected - more complex, doesn't match existing patterns

**Implementation Pattern**:
```go
// Agent streams responses via channel
chunkChan, err := agent.StreamExecute(ctx, input)
for chunk := range chunkChan {
    // Process chunk immediately
    // Convert to TTS and send to voice session
    // Handle interruption via ctx.Done()
}
```

### 2. Incremental TTS Conversion from Streaming LLM Responses

**Decision**: Convert LLM chunks to TTS incrementally using sentence/word boundary detection

**Rationale**:
- Reduces latency by starting TTS before complete response
- Natural conversation flow (agent speaks as it thinks)
- Existing TTS providers support streaming input
- Sentence boundary detection prevents mid-word TTS breaks

**Alternatives Considered**:
- Wait for complete response: Rejected - violates < 500ms latency requirement
- Character-by-character TTS: Rejected - produces unnatural speech, high TTS API calls
- Fixed-size chunk TTS: Rejected - may break mid-word, unnatural pauses

**Implementation Pattern**:
- Buffer LLM chunks until sentence boundary detected (period, exclamation, question mark)
- Send complete sentences to TTS provider
- Stream TTS audio to voice session as it's generated
- Handle partial sentences at end of response

### 3. Interruption Handling for Streaming Agent Responses

**Decision**: Use context cancellation with graceful cleanup and state preservation

**Rationale**:
- Context cancellation provides clean interruption mechanism
- Preserves conversation context for resumption
- Aligns with Go best practices
- Existing voice session has interruption handling infrastructure

**Alternatives Considered**:
- Force stop without cleanup: Rejected - resource leaks, poor UX
- Queue interruptions: Rejected - adds complexity, may cause confusion
- Ignore interruptions: Rejected - violates requirement FR-005

**Implementation Pattern**:
- Monitor `ctx.Done()` in streaming loops
- On interruption: cancel streaming, stop TTS generation, preserve conversation state
- Process new input immediately after cleanup
- Resume conversation with preserved context

### 4. Backpressure Handling in Real-Time Audio Processing

**Decision**: Use buffered channels with size limits and drop-oldest strategy for overflow

**Rationale**:
- Buffered channels provide natural backpressure
- Drop-oldest prevents memory growth
- Size limits prevent unbounded buffering
- Pattern matches existing audio processing infrastructure

**Alternatives Considered**:
- Blocking on full buffer: Rejected - causes latency spikes, violates < 500ms requirement
- Drop-newest: Rejected - may lose important audio chunks
- Unbounded buffering: Rejected - memory growth, eventual OOM

**Implementation Pattern**:
- Use buffered channels (size: 10-20 chunks)
- Monitor channel capacity
- On overflow: drop oldest chunk, log warning, record metric
- Provide backpressure metrics for monitoring

### 5. Conversation Context Management Across Streaming Interactions

**Decision**: Extend existing voice session context with agent-specific state management

**Rationale**:
- Reuses existing session context infrastructure
- Maintains conversation history for agent planning
- Preserves tool execution results for agent decision-making
- Aligns with existing memory integration patterns

**Alternatives Considered**:
- Separate context management: Rejected - duplicates functionality, breaks integration
- Stateless agent execution: Rejected - violates requirement FR-004 (conversation context)
- External context storage: Rejected - adds latency, complexity, not needed for in-memory context

**Implementation Pattern**:
- Extend `VoiceCallAgentContext` with:
  - Conversation history (messages)
  - Agent state (current plan, tool execution results)
  - Streaming state (active streams, interruption flags)
- Store in voice session internal state
- Persist across interruptions and reconnections

## Integration Patterns

### Agent-Voice Session Integration

**Pattern**: Agent instance passed to voice session, session manages agent lifecycle

**Rationale**:
- Voice session controls audio flow and timing
- Agent provides intelligence and decision-making
- Clear separation of concerns (SRP)
- Enables multiple agents per session (future extension)

### Streaming Executor Pattern

**Pattern**: Separate streaming executor that wraps standard executor with streaming capabilities

**Rationale**:
- Maintains backward compatibility (standard executor unchanged)
- Clear separation: standard vs streaming execution
- Easier testing (mock standard executor, test streaming wrapper)
- Follows composition over inheritance

### Error Propagation Pattern

**Pattern**: Use existing Op/Err/Code error pattern, wrap LLM/transport errors appropriately

**Rationale**:
- Consistent error handling across framework
- Enables programmatic error handling
- Preserves error chains for debugging
- Aligns with constitutional requirements

## Performance Considerations

### Latency Optimization
- Start TTS conversion as soon as sentence boundary detected
- Use async TTS generation (don't block on TTS)
- Pipeline: LLM streaming → sentence detection → TTS → audio output
- Minimize buffering between stages

### Memory Management
- Limit conversation history size (keep last N messages)
- Use streaming channels instead of accumulating full responses
- Clean up cancelled streams promptly
- Monitor memory usage via OTEL metrics

### Concurrency
- Each voice call runs in separate goroutine
- Agent execution uses context for cancellation
- TTS generation can be async (non-blocking)
- Use sync primitives for shared state (conversation context)

## Testing Strategy

### Unit Tests
- Mock all external dependencies (LLM, TTS, STT, Transport)
- Test streaming executor with various chunk patterns
- Test interruption handling with context cancellation
- Test backpressure scenarios (channel overflow)
- Test error propagation and recovery

### Integration Tests
- End-to-end voice call with real agent
- Multiple concurrent calls
- Interruption scenarios
- Error recovery scenarios
- Performance tests (latency measurement)

### Coverage Goals
- 90%+ coverage for all new code
- 100% coverage for error paths
- 100% coverage for interruption handling
- Performance benchmarks for latency-critical paths

## Dependencies on Existing Infrastructure

### Required Existing Components
- `pkg/llms`: StreamChat interface, existing providers
- `pkg/voice/session`: Session management, interruption handling
- `pkg/voice/transport`: WebSocket/WebRTC providers
- `pkg/voice/stt`: Speech-to-text providers
- `pkg/voice/tts`: Text-to-speech providers
- `pkg/agents`: Base agent interfaces, executor patterns

### Extension Points
- `pkg/agents/iface.Agent`: Add streaming methods
- `pkg/agents/internal/executor`: Add streaming executor
- `pkg/voice/session/internal/agent_integration`: Complete implementation
- `pkg/voice/session/internal/streaming_agent`: Complete implementation

## Open Questions Resolved

1. **Q**: How to handle partial LLM responses for TTS?  
   **A**: Use sentence boundary detection, buffer until complete sentence

2. **Q**: How to maintain conversation context during streaming?  
   **A**: Extend voice session context with agent-specific state

3. **Q**: How to handle tool execution during streaming?  
   **A**: Pause streaming, execute tool, resume with tool results in context

4. **Q**: How to measure end-to-end latency?  
   **A**: Use OTEL metrics with timestamps at each stage (STT start, LLM start, TTS start, audio output)

5. **Q**: How to test streaming behavior?  
   **A**: Use mock channels, test with various chunk patterns, measure latency in tests

## Conclusion

All research objectives completed. Implementation approach defined with clear patterns, rationale, and alternatives. Ready to proceed to Phase 1 (Design & Contracts).

