# Research: Twilio API Integration

**Date**: 2025-01-07  
**Feature**: Twilio API Integration (001-twilio-integration)  
**Purpose**: Resolve all technical decisions and patterns for implementation

## 1. Twilio Go SDK v1.29.1 API Patterns

### Decision: Use Twilio Go SDK v1.29.1 with Semantic Versioning

**Rationale**:
- Official Twilio Go SDK provides type-safe API access
- v1.29.1 (released January 7, 2026) includes latest API features
- Semantic versioning ensures compatibility
- SDK handles HTTP client, authentication, and request/response parsing

**Key Patterns**:
- **Client Initialization**: `twilio.NewRestClientWithParams(twilio.ClientParams{Username: accountSID, Password: authToken})`
- **Resource Access**: Client provides access to API resources (e.g., `client.Api.V2010.Calls`, `client.Conversations.V1.Conversations`)
- **Error Handling**: SDK returns `twilio.Error` types with status codes and messages
- **Context Support**: All API calls accept `context.Context` for cancellation/timeouts

**Voice API Resources (2010-04-01)**:
- `Call`: Create, fetch, list, update, delete calls
- `Stream`: Real-time audio streaming via WebSocket (wss://)
- `Transcription`: Real-time STT via `<Gather>` or POST creations

**Conversations API Resources (v1)**:
- `Conversation`: Create, fetch, list, update, delete conversations
- `Message`: Send/receive messages with media support (MMS)
- `Participant`: Add/remove participants with identity and messaging bindings
- `Webhook`: Configure webhooks for events (onMessageAdded, onDeliveryUpdated, onTypingStarted)

**Alternatives Considered**:
- Direct HTTP client implementation (rejected: more error-prone, no type safety)
- Older SDK versions (rejected: missing latest features)

## 2. Package Structure Decision: Messaging vs Voice/Conversational

### Decision: Create `pkg/messaging` as a separate package

**Rationale**:
- **Separation of Concerns**: Voice and messaging are distinct domains with different use cases
- **Interface Clarity**: VoiceBackend handles real-time audio streaming; messaging handles text/media messages
- **Scalability**: Separate packages allow independent evolution and provider additions
- **Consistency**: Follows pattern of other domain-specific packages (pkg/llms, pkg/embeddings)
- **Provider Flexibility**: Messaging package can support other providers (e.g., Slack, Discord) beyond Twilio

**Package Structure**:
```
pkg/messaging/
├── iface/              # ConversationalBackend interface
├── internal/           # Private implementation
├── providers/
│   └── twilio/         # Twilio Conversations API provider
├── config.go           # Messaging package config
├── metrics.go          # OTEL metrics
├── errors.go           # Custom error types
├── messaging.go        # Main interfaces and factory
├── registry.go         # Global registry
├── test_utils.go       # Advanced testing utilities
├── advanced_test.go    # Comprehensive test suites
└── README.md           # Package documentation
```

**Alternatives Considered**:
- `pkg/voice/conversational/` (rejected: voice and messaging are conceptually different)
- Extend existing `pkg/voice/backend` (rejected: violates SRP, messaging doesn't need voice infrastructure)
- Single `pkg/telephony/` package (rejected: voice and messaging have different patterns and use cases)

## 3. VoiceBackend Interface Integration

### Decision: Implement VoiceBackend interface with Twilio-specific provider

**Rationale**:
- Existing `pkg/voice/backend/iface.VoiceBackend` interface provides standard contract
- Twilio provider implements all required methods
- Follows established provider pattern (LiveKit, Pipecat, etc.)

**Interface Mapping**:
- `Start()` → Initialize Twilio client, validate configuration
- `Stop()` → Gracefully shutdown, complete in-flight calls
- `CreateSession()` → Create Twilio Call resource, establish WebSocket stream
- `GetSession()` → Retrieve active call session by call SID
- `ListSessions()` → List all active calls
- `CloseSession()` → Update/delete Twilio Call resource
- `HealthCheck()` → Verify Twilio API connectivity
- `GetConnectionState()` → Return connection state (connected, disconnected, error)
- `GetActiveSessionCount()` → Return count of active calls
- `GetConfig()` → Return Twilio configuration
- `UpdateConfig()` → Update configuration with validation

**Twilio-Specific Extensions**:
- Webhook handling for inbound calls
- WebSocket stream management for real-time audio
- Call status callback integration
- Transcription resource management

**Integration Approach**:
- Provider implements `VoiceBackend` interface
- Uses Twilio SDK for API calls
- Manages WebSocket connections for streaming
- Integrates with pkg/voice/session for agent interaction

## 4. ConversationalBackend Interface Design

### Decision: Create new ConversationalBackend interface following ISP

**Rationale**:
- Messaging is a distinct domain from voice
- Interface should be provider-agnostic (not Twilio-specific)
- Follows Interface Segregation Principle (small, focused interface)
- Enables future providers (Slack, Discord, etc.)

**Interface Design**:
```go
type ConversationalBackend interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Conversation Management
    CreateConversation(ctx context.Context, config *ConversationConfig) (Conversation, error)
    GetConversation(ctx context.Context, conversationID string) (Conversation, error)
    ListConversations(ctx context.Context) ([]Conversation, error)
    CloseConversation(ctx context.Context, conversationID string) error
    
    // Message Operations
    SendMessage(ctx context.Context, conversationID string, message *Message) error
    ReceiveMessages(ctx context.Context, conversationID string) (<-chan *Message, error)
    
    // Participant Management
    AddParticipant(ctx context.Context, conversationID string, participant *Participant) error
    RemoveParticipant(ctx context.Context, conversationID string, participantID string) error
    
    // Webhook Handling
    HandleWebhook(ctx context.Context, event *WebhookEvent) error
    
    // Health & Status
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    GetConfig() *Config
}
```

**Key Principles**:
- Small, focused methods (ISP)
- Provider-agnostic types (Conversation, Message, Participant)
- Context support for cancellation/timeouts
- Error handling via custom error types

**Alternatives Considered**:
- Single large interface with all methods (rejected: violates ISP)
- Twilio-specific interface (rejected: not provider-agnostic)
- Extend VoiceBackend (rejected: messaging doesn't need voice infrastructure)

## 5. Webhook Handling Patterns

### Decision: HTTP endpoint handlers with signature validation and orchestration integration

**Rationale**:
- Twilio requires publicly accessible webhook endpoints
- Signature validation ensures webhook authenticity (security requirement FR-028)
- Integration with pkg/orchestration enables event-driven workflows (FR-026)
- Follows existing pkg/server patterns for HTTP handling

**Architecture**:
1. **Webhook Endpoints**: HTTP POST endpoints in pkg/server or dedicated webhook handler
2. **Signature Validation**: Validate Twilio signature using auth token
3. **Event Parsing**: Parse webhook payload into structured events
4. **Orchestration Integration**: Trigger pkg/orchestration workflows based on event type
5. **Error Handling**: Retry logic for failed webhook processing (FR-027)

**Webhook Event Types**:
- **Voice API**: CallStatusCallback (initiated, answered, completed, failed), Stream events, Transcription events
- **Conversations API**: onMessageAdded, onDeliveryUpdated, onTypingStarted, onParticipantAdded, onParticipantRemoved

**Integration Pattern**:
```go
// Webhook handler receives event
func (p *TwilioProvider) HandleWebhook(ctx context.Context, event *WebhookEvent) error {
    // Validate signature
    if err := p.validateSignature(event); err != nil {
        return err
    }
    
    // Parse event
    parsedEvent := p.parseEvent(event)
    
    // Trigger orchestration workflow
    if p.orchestrator != nil {
        return p.orchestrator.TriggerWorkflow(ctx, parsedEvent)
    }
    
    // Handle event directly
    return p.handleEvent(ctx, parsedEvent)
}
```

**Alternatives Considered**:
- Polling for events (rejected: inefficient, not real-time)
- WebSocket for webhooks (rejected: Twilio uses HTTP webhooks)
- No signature validation (rejected: security risk)

## 6. Real-Time Streaming Architecture

### Decision: WebSocket (wss://) streaming with mu-law codec and bidirectional audio

**Rationale**:
- Twilio Media Streams use WebSocket (wss://) for real-time audio
- Mu-law codec is standard for telephony (PCMU)
- Bidirectional streaming enables real-time conversation
- Low latency requirement (<2s) requires streaming architecture

**Architecture**:
1. **Stream Creation**: Create Twilio Media Stream via API
2. **WebSocket Connection**: Establish wss:// connection to Twilio
3. **Audio Encoding**: Encode audio to mu-law (PCMU) format
4. **Bidirectional Streaming**: 
   - Receive audio from Twilio → STT → Agent → TTS → Send audio to Twilio
   - Stream audio chunks in real-time
5. **Stream Management**: Handle connection lifecycle, reconnection, errors

**Integration with pkg/voice**:
- Use existing STT/TTS providers from pkg/voice
- Integrate with pkg/agents for streaming agent responses
- Use pkg/voice/session for session management

**Streaming Flow**:
```
Twilio Call → Media Stream (WebSocket) → Audio Chunks → STT → Transcript → Agent → Response → TTS → Audio Chunks → Media Stream → Twilio Call
```

**Codec Requirements**:
- **Input**: Mu-law (PCMU) from Twilio
- **Output**: Mu-law (PCMU) to Twilio
- **Internal**: May use different format for STT/TTS processing

**Alternatives Considered**:
- HTTP polling for audio (rejected: too high latency)
- One-way streaming (rejected: need bidirectional for conversation)
- Different codec (rejected: Twilio requires mu-law for Media Streams)

## 7. Memory Integration Patterns

### Decision: Use pkg/memory/VectorStoreMemory for conversation persistence

**Rationale**:
- VectorStoreMemory provides persistent storage with semantic search
- Enables RAG integration with transcriptions
- Supports multi-channel context preservation (FR-016)
- Follows existing Beluga memory patterns

**Storage Pattern**:
1. **Session-to-Memory Mapping**: Map call/conversation sessions to memory instances
2. **Conversation History**: Store messages/transcripts in memory
3. **Context Retrieval**: Retrieve conversation history for agent context
4. **Multi-Channel**: Use participant identity to link conversations across channels

**Memory Integration**:
```go
// Create memory instance for session
memory, err := memory.NewMemory(memory.MemoryTypeVectorStore, memory.WithVectorStore(vectorStore))

// Store conversation history
memory.SaveContext(ctx, map[string]any{
    "messages": conversationHistory,
    "session_id": sessionID,
})

// Load context for agent
context, err := memory.LoadMemoryVariables(ctx, map[string]string{})
```

**Session Persistence**:
- **Voice Sessions**: Store transcripts and call metadata
- **Messaging Sessions**: Store message history and participant info
- **Cross-Session**: Link sessions by participant identity for multi-channel support

**Alternatives Considered**:
- In-memory only (rejected: doesn't meet FR-015 persistence requirement)
- Database storage (rejected: VectorStoreMemory provides RAG capabilities)
- Separate storage per provider (rejected: want unified memory interface)

## 8. Orchestration Integration

### Decision: DAG workflows for call flows with webhook event triggers

**Rationale**:
- pkg/orchestration provides workflow capabilities (FR-026)
- DAG pattern enables complex call flows (Inbound → Agent → Stream)
- Webhook events trigger workflows for event-driven architecture
- Supports complex multi-step processes

**Workflow Patterns**:
1. **Call Flow DAG**: Inbound Call → Agent Setup → Stream Creation → Conversation Loop → Call End
2. **Message Flow DAG**: Message Received → Agent Processing → Response Generation → Message Send
3. **Event-Driven**: Webhook Event → Workflow Trigger → Agent Actions → Response

**Integration Pattern**:
```go
// Create workflow for call flow
workflow, err := orchestrator.CreateGraph(
    orchestration.WithGraphNode("inbound", handleInboundCall),
    orchestration.WithGraphNode("agent", setupAgent),
    orchestration.WithGraphNode("stream", createStream),
    orchestration.WithGraphEdge("inbound", "agent"),
    orchestration.WithGraphEdge("agent", "stream"),
)

// Trigger workflow from webhook
func (p *TwilioProvider) HandleWebhook(ctx context.Context, event *WebhookEvent) error {
    if event.Type == "call.answered" {
        return p.orchestrator.Invoke(ctx, workflow, event.Data)
    }
    return nil
}
```

**Event-to-Workflow Mapping**:
- Call events → Call flow workflows
- Message events → Message processing workflows
- Transcription events → RAG workflows

**Alternatives Considered**:
- Direct function calls (rejected: doesn't support complex flows)
- Sequential chains only (rejected: DAG enables parallel processing)
- No orchestration (rejected: doesn't meet FR-026 requirement)

## 9. Transcription and RAG Integration

### Decision: Store transcriptions in pkg/vectorstores with embeddings for RAG

**Rationale**:
- Vector stores enable semantic search of transcriptions (FR-031, FR-032)
- Embeddings enable multimodal RAG (FR-033)
- Integration with existing RAG pipeline (pkg/vectorstores, pkg/embeddings, pkg/retrievers)
- Supports transcription search and retrieval

**Storage Pattern**:
1. **Transcription Storage**: Store transcriptions as documents in vector store
2. **Embedding Generation**: Generate embeddings via pkg/embeddings
3. **Vector Storage**: Store embeddings in pkg/vectorstores
4. **Retrieval**: Use pkg/retrievers for semantic search

**RAG Integration Flow**:
```
Call Transcription → Embedding (pkg/embeddings) → Vector Store (pkg/vectorstores) → Retrieval (pkg/retrievers) → Agent Context → LLM Response
```

**Multimodal RAG**:
- Combine transcriptions with other data sources (documents, images)
- Use pkg/multimodal for multimodal model integration
- Enable cross-modal retrieval

**Integration Pattern**:
```go
// Store transcription
transcriptionDoc := schema.NewDocument(transcriptionText, schema.WithMetadata(map[string]any{
    "call_sid": callSID,
    "timestamp": timestamp,
}))

// Generate embedding
embedding, err := embedder.Embed(ctx, []string{transcriptionText})

// Store in vector store
err = vectorStore.AddDocuments(ctx, []schema.Document{transcriptionDoc}, embedding)

// Retrieve for RAG
results, err := retriever.GetRelevantDocuments(ctx, query)
```

**Alternatives Considered**:
- Plain text storage (rejected: doesn't enable semantic search)
- Separate transcription database (rejected: want unified RAG pipeline)
- No RAG integration (rejected: doesn't meet FR-032, FR-033 requirements)

## 10. Error Handling and Mapping

### Decision: Map Twilio errors to Beluga error types with contextual wrapping

**Rationale**:
- Beluga uses custom error types with Op/Err/Code pattern (Constitution V)
- Twilio SDK returns twilio.Error types
- Need consistent error handling across providers
- Enable programmatic error handling via error codes

**Error Mapping**:
```go
// Twilio error codes to Beluga error codes
const (
    ErrCodeTwilioRateLimit     = "twilio_rate_limit"
    ErrCodeTwilioInvalidConfig = "twilio_invalid_config"
    ErrCodeTwilioNetworkError  = "twilio_network_error"
    ErrCodeTwilioTimeout       = "twilio_timeout"
    ErrCodeTwilioAuthError     = "twilio_auth_error"
)

// Map Twilio error to Beluga error
func mapTwilioError(op string, err error) *PackageError {
    if twilioErr, ok := err.(*twilio.Error); ok {
        code := mapTwilioErrorCode(twilioErr.Code)
        return NewError(op, err, code)
    }
    return NewError(op, err, ErrCodeUnknown)
}
```

**Error Handling Patterns**:
- **Rate Limits**: Return ErrCodeRateLimit, implement retry with backoff
- **Network Errors**: Return ErrCodeNetworkError, implement retry logic
- **Timeouts**: Return ErrCodeTimeout, respect context cancellation
- **Auth Errors**: Return ErrCodeAuthError, don't retry
- **Invalid Config**: Return ErrCodeInvalidConfig, don't retry

**Retry Logic**:
- Transient errors (rate limit, network): Retry with exponential backoff
- Permanent errors (auth, invalid config): Don't retry
- Context cancellation: Respect immediately

**Alternatives Considered**:
- Pass through Twilio errors directly (rejected: not consistent with Beluga patterns)
- Generic error wrapping only (rejected: need error codes for programmatic handling)
- No retry logic (rejected: doesn't meet SC-011 error recovery requirement)

## Summary of Decisions

| Decision Area | Decision | Rationale |
|--------------|----------|-----------|
| SDK Usage | Twilio Go SDK v1.29.1 | Official SDK, type safety, latest features |
| Package Structure | `pkg/messaging` separate package | Separation of concerns, scalability |
| VoiceBackend | Implement existing interface | Follows established provider pattern |
| ConversationalBackend | New interface following ISP | Provider-agnostic, focused interface |
| Webhook Handling | HTTP endpoints with signature validation | Security, orchestration integration |
| Streaming | WebSocket (wss://) with mu-law codec | Real-time, bidirectional, low latency |
| Memory | pkg/memory/VectorStoreMemory | Persistent, RAG-capable, multi-channel |
| Orchestration | DAG workflows with event triggers | Complex flows, event-driven architecture |
| Transcription/RAG | Vector stores with embeddings | Semantic search, multimodal RAG |
| Error Handling | Map to Beluga error types | Consistency, programmatic handling |

## Implementation Readiness

✅ **All research tasks complete**:
- SDK patterns understood
- Package structure decided
- Interfaces designed
- Integration patterns defined
- Error handling approach determined

**Next Steps**: Proceed to Phase 1 (Design & Contracts)
