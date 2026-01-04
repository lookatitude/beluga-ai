# Data Model: Real-Time Voice Agent Support

**Date**: 2025-01-27  
**Feature**: Real-Time Voice Agent Support

## Overview

This feature extends existing data structures in `pkg/agents` and `pkg/voice/session` packages. No new top-level entities are created. Instead, existing entities are extended with new fields and capabilities.

## Extended Entities

### 1. Extended Agent Interface (`pkg/agents/iface`)

**Base**: Existing `Agent` interface  
**Extension**: Add streaming execution capabilities

**New Methods**:
```go
// StreamingAgent extends Agent with streaming capabilities
type StreamingAgent interface {
    Agent  // Embed existing Agent interface
    
    // StreamExecute executes agent with streaming LLM responses
    StreamExecute(ctx context.Context, inputs map[string]any) (<-chan AgentStreamChunk, error)
    
    // StreamPlan plans next action with streaming model responses
    StreamPlan(ctx context.Context, intermediateSteps []IntermediateStep, inputs map[string]any) (<-chan AgentStreamChunk, error)
}
```

**New Types**:
```go
// AgentStreamChunk represents a chunk of agent execution output
type AgentStreamChunk struct {
    Content      string              // Text content from LLM
    ToolCalls    []schema.ToolCall   // Tool calls if any
    Action       *AgentAction        // Next action if determined
    Finish       *AgentFinish        // Final result if complete
    Err          error               // Error if occurred
    Metadata     map[string]any      // Additional metadata
}

// StreamingConfig extends AgentConfig with streaming-specific settings
type StreamingConfig struct {
    EnableStreaming     bool          // Enable streaming mode
    ChunkBufferSize     int           // Buffer size for chunks
    SentenceBoundary    bool          // Wait for sentence boundaries before TTS
    InterruptOnNewInput bool          // Allow interruption on new input
    MaxStreamDuration   time.Duration // Maximum stream duration
}
```

**Validation Rules**:
- `ChunkBufferSize` must be > 0 and <= 100
- `MaxStreamDuration` must be > 0
- `EnableStreaming` must be true for streaming methods

**State Transitions**:
- Agent starts in standard mode
- Can transition to streaming mode via `StreamExecute`
- Streaming mode can be interrupted (returns to standard mode)
- Agent can switch between modes during execution

### 2. Voice Session Agent Integration (`pkg/voice/session/internal`)

**Base**: Existing `VoiceSessionImpl` and `AgentIntegration`  
**Extension**: Support full agent instances instead of simple callbacks

**New Types**:
```go
// AgentInstance represents an agent instance integrated with voice session
type AgentInstance struct {
    Agent        iface.StreamingAgent  // The agent instance
    Config       *AgentConfig          // Agent configuration
    Context      *AgentContext        // Conversation context
    State        AgentState           // Current agent state
    mu           sync.RWMutex         // State protection
}

// AgentContext extends voice session context with agent-specific state
type AgentContext struct {
    ConversationHistory []schema.Message    // Message history
    ToolResults        []ToolResult        // Tool execution results
    CurrentPlan        []schema.Step       // Current execution plan
    StreamingActive    bool                // Whether streaming is active
    LastInterruption   time.Time          // Last interruption timestamp
}

// AgentState represents the state of an agent in a voice session
type AgentState string

const (
    AgentStateIdle        AgentState = "idle"
    AgentStateListening   AgentState = "listening"
    AgentStateProcessing  AgentState = "processing"
    AgentStateStreaming   AgentState = "streaming"
    AgentStateExecuting   AgentState = "executing_tool"
    AgentStateSpeaking    AgentState = "speaking"
    AgentStateInterrupted AgentState = "interrupted"
)
```

**Validation Rules**:
- Agent instance must not be nil
- Agent must implement `StreamingAgent` interface
- Context must be initialized before use
- State transitions must be valid (see state machine)

**State Transitions**:
```
Idle → Listening (on user input)
Listening → Processing (on transcript received)
Processing → Streaming (on LLM response start)
Streaming → Speaking (on TTS start)
Streaming → Interrupted (on user interruption)
Speaking → Interrupted (on user interruption)
Interrupted → Processing (on new input)
Executing → Processing (on tool completion)
Any → Idle (on session end)
```

### 3. Streaming Agent Executor (`pkg/agents/internal/executor`)

**Base**: Existing `Executor` interface  
**Extension**: Add streaming execution capabilities

**New Types**:
```go
// StreamingExecutor extends Executor with streaming capabilities
type StreamingExecutor interface {
    Executor  // Embed existing Executor interface
    
    // ExecuteStreamingPlan executes a plan with streaming LLM responses
    ExecuteStreamingPlan(ctx context.Context, agent StreamingAgent, plan []schema.Step) (<-chan ExecutionChunk, error)
}

// ExecutionChunk represents a chunk of execution output
type ExecutionChunk struct {
    Step        schema.Step           // Current step
    Content     string                // Text content
    ToolResult  *ToolExecutionResult  // Tool result if any
    FinalAnswer *schema.FinalAnswer   // Final answer if complete
    Err         error                 // Error if occurred
    Timestamp   time.Time             // Chunk timestamp
}
```

**Validation Rules**:
- Plan must not be empty
- Agent must implement `StreamingAgent`
- Context must not be nil

**State Transitions**:
- Executor starts with first step
- Processes steps sequentially
- Streams content as it arrives
- Can be interrupted at any step
- Completes with final answer or error

### 4. Voice Call Agent Context (`pkg/voice/session/internal`)

**Base**: Existing voice session context  
**Extension**: Add agent-specific state

**New Fields** (added to existing context):
```go
type VoiceCallAgentContext struct {
    // Existing fields from voice session context
    SessionID         string
    UserID            string
    StartTime         time.Time
    LastActivity      time.Time
    
    // New agent-specific fields
    AgentInstance     *AgentInstance      // Agent instance
    ConversationHistory []schema.Message  // Message history
    ToolExecutionResults []ToolResult     // Tool results
    CurrentPlan        []schema.Step      // Current plan
    StreamingState     StreamingState     // Streaming state
}

type StreamingState struct {
    Active           bool              // Whether streaming is active
    CurrentStream    <-chan AgentStreamChunk  // Current stream channel
    Buffer           []AgentStreamChunk       // Buffered chunks
    LastChunkTime    time.Time               // Last chunk timestamp
    Interrupted      bool                     // Whether interrupted
}
```

**Validation Rules**:
- SessionID must be non-empty
- AgentInstance must be set before use
- ConversationHistory must be initialized (can be empty)
- StreamingState must be initialized

**State Transitions**:
- Context created on session start
- ConversationHistory updated on each message
- ToolExecutionResults updated on tool completion
- StreamingState updated during streaming
- Context preserved across interruptions

## Relationships

### Agent ↔ Voice Session
- **Relationship**: One agent instance per voice session
- **Cardinality**: 1:1
- **Lifecycle**: Agent instance created with session, destroyed with session
- **Integration Point**: `pkg/voice/session/internal/agent_integration.go`

### Agent ↔ LLM Provider
- **Relationship**: Agent uses LLM provider for streaming responses
- **Cardinality**: 1:1 (per agent instance)
- **Lifecycle**: LLM connection managed by agent
- **Integration Point**: `pkg/agents` uses `pkg/llms.StreamChat`

### Voice Session ↔ Transport
- **Relationship**: Voice session uses transport for audio I/O
- **Cardinality**: 1:1
- **Lifecycle**: Transport connection managed by session
- **Integration Point**: `pkg/voice/session` uses `pkg/voice/transport`

### Agent ↔ Tools
- **Relationship**: Agent can execute tools during voice calls
- **Cardinality**: 1:many (agent has multiple tools)
- **Lifecycle**: Tools registered with agent, executed on demand
- **Integration Point**: `pkg/agents/tools` registry

## Data Flow

### Streaming Execution Flow
```
User Speech → STT → Transcript → Agent.StreamExecute()
    ↓
Agent.StreamPlan() → LLM.StreamChat() → AgentStreamChunk
    ↓
Sentence Detection → TTS → Audio → Transport → User
```

### Conversation Context Flow
```
AgentStreamChunk → ConversationHistory (append)
Tool Execution → ToolExecutionResults (append)
Final Answer → ConversationHistory (append)
Interruption → Preserve Context → Resume with new input
```

## Constraints

### Performance Constraints
- End-to-end latency must be < 500ms
- Streaming chunks must be processed within 50ms
- Context updates must be atomic (use mutex)
- Conversation history limited to last 100 messages

### Memory Constraints
- Chunk buffer limited to 20 chunks
- Conversation history limited to 100 messages
- Tool results limited to last 50 executions
- Context size monitored via OTEL metrics

### Concurrency Constraints
- One streaming execution per agent instance
- Context updates must be thread-safe
- State transitions must be atomic
- Interruption must be immediate (< 10ms)

## Validation

### Input Validation
- Agent instance must implement `StreamingAgent`
- Config must pass validation rules
- Context must be properly initialized
- State transitions must be valid

### Output Validation
- Stream chunks must have valid structure
- Final answers must match schema
- Errors must follow Op/Err/Code pattern
- Metrics must be recorded for all operations

## Migration Notes

### Backward Compatibility
- Existing `Agent` interface unchanged
- Existing `VoiceSession` interface unchanged
- New interfaces extend existing ones
- Existing code continues to work

### Extension Points
- `Agent` interface can be extended to `StreamingAgent`
- `VoiceSession` can accept `StreamingAgent` instances
- Existing callbacks still supported (deprecated)
- Gradual migration path available

## Testing Considerations

### Unit Test Data
- Mock `StreamingAgent` implementations
- Mock LLM providers with controlled chunk streams
- Mock TTS providers with controlled audio output
- Test state transitions with various scenarios

### Integration Test Data
- Real agent instances with mock LLM providers
- Real voice sessions with mock transport
- End-to-end scenarios with controlled timing
- Performance tests with latency measurement

### Test Coverage Requirements
- 90%+ coverage for all new code
- 100% coverage for state transitions
- 100% coverage for error paths
- Performance benchmarks for critical paths

