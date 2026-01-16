# Implementation Plan: Twilio API Integration

**Branch**: `001-twilio-integration` | **Date**: 2025-01-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-twilio-integration/spec.md`

## Summary

Integrate Twilio Programmable Voice API and Conversations API into the Beluga AI Framework to enable voice-enabled agents (real-time IVR with LLM-driven responses) and multi-channel conversational agents (SMS/WhatsApp with memory persistence). The integration will create two new provider packages following Beluga's standard patterns: `pkg/voice/providers/twilio` for Voice API integration and a new messaging package structure for Conversations API integration. The implementation will leverage Twilio Go SDK v1.29.1, follow global registry patterns, integrate with existing Beluga packages (agents, memory, orchestration, vectorstores, embeddings), and provide comprehensive OTEL observability.

## Technical Context

**Language/Version**: Go 1.24+  
**Primary Dependencies**: 
- Twilio Go SDK v1.29.1 (github.com/twilio/twilio-go)
- OpenTelemetry (go.opentelemetry.io/otel) for observability
- Existing Beluga packages: pkg/llms, pkg/agents, pkg/memory, pkg/orchestration, pkg/vectorstores, pkg/embeddings, pkg/monitoring, pkg/config, pkg/schema

**Storage**: 
- In-memory session storage for active calls/conversations
- pkg/memory/VectorStoreMemory for conversation history persistence
- pkg/vectorstores for transcription storage and RAG integration

**Testing**: 
- Advanced mocks in test_utils.go following Beluga patterns
- Table-driven tests in advanced_test.go
- Integration tests in tests/integration/voice_twilio_test.go
- Concurrency testing, load testing, benchmarks

**Target Platform**: Linux server (production), local development  
**Project Type**: Backend library/package extension  
**Performance Goals**: 
- Voice: <2s latency from speech completion to agent response (SC-002)
- Messaging: <5s message processing for 95% of messages (SC-005)
- Webhooks: <1s event processing for 99% of events (SC-007)
- Support 100 concurrent voice calls (SC-003)

**Constraints**: 
- Real-time audio streaming via WebSocket (wss://) with mu-law codec
- Webhook endpoints must be publicly accessible or use tunneling
- Rate limits from Twilio API must be respected
- Backward compatibility with existing Beluga packages

**Scale/Scope**: 
- Two new provider packages (voice/twilio, messaging/twilio or voice/conversational/twilio)
- Integration with 9 existing Beluga packages
- Support for Voice API (calls, streaming, transcriptions) and Conversations API (SMS, WhatsApp, multi-channel)
- Implementation effort: 10-18 days for production-ready providers

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Package Structure (MUST) ✅
- [x] Follow standard v2 structure with `iface/`, `internal/`, `providers/` directories
- [x] Include required files: `config.go`, `metrics.go`, `errors.go`, `test_utils.go`, `advanced_test.go`, `README.md`
- [x] Voice provider: `pkg/voice/providers/twilio/` structure
- [x] Messaging provider: New package structure (to be determined in Phase 0 research)

### II. Interface Design (MUST) ✅
- [x] Implement VoiceBackend interface for Voice API integration
- [x] Create new ConversationalBackend interface for Conversations API (following ISP)
- [x] Small, focused interfaces (not "god interfaces")
- [x] Use Dependency Inversion Principle (DIP) with constructor injection

### III. Provider Registry Pattern (MUST) ✅
- [x] Use global registry pattern matching pkg/llms and pkg/embeddings
- [x] Implement `GetRegistry()` function
- [x] Provider registration in `providers/twilio/init.go` files
- [x] Match existing patterns from voice/backend registry

### IV. OTEL Observability (MUST) ✅
- [x] OTEL metrics in `metrics.go` (counters, histograms for latency, throughput)
- [x] OTEL tracing for all public methods with span attributes
- [x] Structured logging with OTEL context (trace IDs, span IDs)
- [x] Record errors with `span.RecordError()` and set status codes

### V. Error Handling (MUST) ✅
- [x] Custom error types with Op, Err, Code, Message fields
- [x] Error codes for common failures (provider_not_found, invalid_config, rate_limit, etc.)
- [x] Context cancellation support
- [x] Map Twilio errors to Beluga's contextual errors

### VI. Configuration (MUST) ✅
- [x] Config struct with mapstructure, yaml, env, validate tags
- [x] Functional options for runtime configuration
- [x] Validation at creation time using validator library
- [x] Extend pkg/config with TwilioConfig struct (AccountSID, AuthToken, PhoneNumber)

### VII. Testing (MUST) ✅
- [x] Table-driven tests in `advanced_test.go`
- [x] Advanced mocks in `test_utils.go` following Beluga patterns
- [x] Benchmarks for performance-critical operations
- [x] Integration tests for cross-package compatibility
- [x] Concurrency testing, load testing

### VIII. Backward Compatibility (MUST) ✅
- [x] No breaking changes to existing APIs
- [x] New functionality is opt-in (provider registration)
- [x] Zero breaking changes to existing Beluga components (SC-013)

**Constitution Compliance**: ✅ All gates pass. Implementation will follow Beluga AI Framework patterns.

## Project Structure

### Documentation (this feature)

```text
specs/001-twilio-integration/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── voice-backend-api.md
│   ├── conversational-backend-api.md
│   └── webhook-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
pkg/
├── voice/
│   ├── providers/
│   │   └── twilio/              # Voice API provider
│   │       ├── iface/           # Provider-specific interfaces (if needed)
│   │       ├── config.go        # Twilio Voice API configuration
│   │       ├── provider.go      # VoiceBackend implementation
│   │       ├── streaming.go     # WebSocket streaming implementation
│   │       ├── webhook.go       # Webhook handling for Voice API
│   │       ├── transcription.go # Transcription integration
│   │       ├── init.go          # Auto-registration
│   │       ├── provider_test.go # Unit tests
│   │       ├── streaming_test.go # Streaming tests
│   │       └── webhook_test.go  # Webhook tests
│   └── backend/                 # Existing voice backend package
│
├── messaging/                    # NEW: Messaging package (or voice/conversational/)
│   ├── iface/                   # ConversationalBackend interface
│   ├── internal/                # Private implementation details
│   ├── providers/
│   │   └── twilio/              # Conversations API provider
│   │       ├── config.go        # Twilio Conversations API configuration
│   │       ├── provider.go      # ConversationalBackend implementation
│   │       ├── webhook.go       # Webhook handling for Conversations API
│   │       ├── init.go          # Auto-registration
│   │       └── provider_test.go # Unit tests
│   ├── config.go                # Messaging package config
│   ├── metrics.go               # OTEL metrics
│   ├── errors.go                # Custom error types
│   ├── messaging.go             # Main interfaces and factory
│   ├── registry.go              # Global registry
│   ├── test_utils.go            # Advanced testing utilities
│   ├── advanced_test.go         # Comprehensive test suites
│   └── README.md                # Package documentation
│
├── [existing packages: llms, agents, memory, orchestration, etc.]

tests/
└── integration/
    ├── voice_twilio_test.go     # Voice API integration tests
    └── messaging_twilio_test.go # Conversations API integration tests

examples/
└── voice/
    └── twilio/                  # Twilio integration examples
        ├── voice_agent/         # Voice-enabled agent example
        │   └── main.go
        ├── messaging_agent/     # Messaging agent example
        │   └── main.go
        └── webhook_server/      # Webhook handling example
            └── main.go

docs/
└── providers/
    └── twilio.md                # Twilio provider documentation
```

**Structure Decision**: 
- Voice API integration extends existing `pkg/voice/backend` with a new Twilio provider following the established provider pattern
- Conversations API integration creates a new `pkg/messaging` package (or `pkg/voice/conversational/` - to be determined in Phase 0 research) with its own interface and registry, following the same patterns as other multi-provider packages
- Both providers use global registry patterns matching pkg/llms and pkg/embeddings
- Integration tests are placed in `tests/integration/` following existing patterns
- Examples follow the existing `examples/voice/` structure

## Complexity Tracking

> **No violations identified** - Implementation follows standard Beluga patterns

## Phase 0: Outline & Research

### Research Tasks

1. **Twilio Go SDK v1.29.1 API Patterns**
   - Research Twilio Go SDK structure and best practices
   - Understand Voice API (2010-04-01) resource patterns (Call, Stream, Transcription)
   - Understand Conversations API (v1) resource patterns (Conversation, Message, Participant)
   - Identify WebSocket streaming patterns for Voice API
   - Document error handling patterns in Twilio SDK

2. **Package Structure Decision: Messaging vs Voice/Conversational**
   - Evaluate whether to create `pkg/messaging` or `pkg/voice/conversational`
   - Consider separation of concerns (voice vs messaging are different domains)
   - Consider existing package structure patterns
   - Make recommendation with rationale

3. **VoiceBackend Interface Integration**
   - Review existing VoiceBackend interface requirements
   - Map Twilio Voice API capabilities to VoiceBackend methods
   - Identify gaps or extensions needed
   - Document integration approach

4. **ConversationalBackend Interface Design**
   - Design new ConversationalBackend interface following ISP
   - Define methods for: CreateConversation, SendMessage, HandleEvent, etc.
   - Ensure interface is provider-agnostic (not Twilio-specific)
   - Document interface design decisions

5. **Webhook Handling Patterns**
   - Research Twilio webhook signature validation
   - Design webhook handler architecture
   - Integrate with pkg/orchestration for event-driven workflows
   - Design webhook endpoint structure

6. **Real-Time Streaming Architecture**
   - Research WebSocket streaming patterns (wss://)
   - Understand mu-law codec requirements
   - Design bidirectional audio streaming
   - Integrate with existing pkg/voice streaming patterns

7. **Memory Integration Patterns**
   - Research how to store conversation history in pkg/memory
   - Design session-to-memory mapping
   - Integrate with VectorStoreMemory for RAG
   - Document memory persistence patterns

8. **Orchestration Integration**
   - Design DAG workflows for call flows (Inbound → Agent → Stream)
   - Integrate webhook events with pkg/orchestration
   - Design event-to-workflow mapping
   - Document orchestration patterns

9. **Transcription and RAG Integration**
   - Research transcription storage patterns
   - Design integration with pkg/vectorstores and pkg/embeddings
   - Design multimodal RAG with transcriptions
   - Document RAG integration approach

10. **Error Handling and Mapping**
    - Research Twilio error types and codes
    - Design error mapping from Twilio to Beluga error types
    - Document error handling patterns
    - Design retry logic for transient failures

### Research Output

**File**: `specs/001-twilio-integration/research.md`

Will contain:
- Decision: Package structure choice (messaging vs conversational)
- Decision: Interface design decisions
- Decision: Webhook architecture
- Decision: Streaming architecture
- Rationale: Why each decision was made
- Alternatives considered: Other approaches evaluated
- Integration patterns: How to integrate with existing packages
- Best practices: Twilio SDK usage patterns

## Phase 1: Design & Contracts

### Data Model

**File**: `specs/001-twilio-integration/data-model.md`

Will define:
- **Call Entity**: Call ID, phone numbers, status, timestamps, duration
- **Message Entity**: Message ID, conversation ID, channel, sender, recipient, content, media, delivery status
- **Conversation Entity**: Conversation ID, participants, channels, creation time, state
- **Transcription Entity**: Transcription ID, call ID, text content, timestamps, confidence scores
- **Webhook Event Entity**: Event type, event data, timestamp, source, signature
- **Voice Session Entity**: Session ID, call ID, agent instance, conversation state, streaming status
- **Messaging Session Entity**: Session ID, conversation ID, agent instance, conversation history, memory state

### API Contracts

**Directory**: `specs/001-twilio-integration/contracts/`

1. **voice-backend-api.md**: VoiceBackend interface contract for Twilio provider
   - StartSession (Create Call)
   - HandleInbound (Webhook parsing + Stream creation)
   - StreamAudio (WebSocket handling)
   - EndSession (Update/Delete Call)
   - Health checks, configuration

2. **conversational-backend-api.md**: ConversationalBackend interface contract
   - CreateConversation
   - SendMessage
   - HandleEvent (Webhook processing)
   - Participant management
   - Conversation lifecycle

3. **webhook-api.md**: Webhook handling contract
   - Webhook endpoint structure
   - Event types (call events, message events)
   - Signature validation
   - Event-to-workflow mapping

### Quickstart Guide

**File**: `specs/001-twilio-integration/quickstart.md`

Will provide:
- Setup instructions (Twilio account, credentials)
- Basic voice agent example
- Basic messaging agent example
- Webhook configuration
- Integration with agents and memory
- Common use cases

### Agent Context Update

After Phase 1 design, run:
```bash
.specify/scripts/bash/update-agent-context.sh cursor-agent
```

This will update the agent-specific context file with new technology from the plan.

## Phase 2: Implementation Planning

*Note: Phase 2 is handled by `/speckit.tasks` command, not `/speckit.plan`*

The implementation will be broken down into tasks covering:
- Package structure setup
- Configuration and registry implementation
- Voice API provider implementation
- Conversations API provider implementation
- Webhook handling
- Streaming implementation
- Memory integration
- Orchestration integration
- Transcription and RAG integration
- Testing (unit, integration, concurrency, load)
- Documentation
- Examples

## Integration Points

### Cross-Package Integration Flow

1. **Voice Pipeline**: 
   - Twilio Call → Webhook → VoiceBackend → STT → Agent → TTS → Stream → Twilio
   - Audio/message → STT/embedding → RAG (pkg/vectorstores) → LLM (pkg/llms) → TTS/response

2. **Messaging Pipeline**:
   - Twilio Message → Webhook → ConversationalBackend → Agent → Response → Twilio
   - Message → Embedding (pkg/embeddings) → Memory (pkg/memory) → Agent (pkg/agents) → Response

3. **Orchestration Integration**:
   - Webhook Event → pkg/orchestration → DAG Workflow → Agent Actions
   - Call flows: Inbound → Agent → Stream (orchestrated via pkg/orchestration)

4. **Memory Integration**:
   - Sessions stored in pkg/memory/VectorStoreMemory
   - Conversation history persisted across sessions
   - Multi-channel context preservation

5. **RAG Integration**:
   - Transcriptions stored in pkg/vectorstores
   - Embeddings via pkg/embeddings
   - Retrieval via pkg/retrievers
   - Multimodal RAG with transcriptions

## Success Criteria Alignment

- **SC-001**: 95% call success rate → Implement robust error handling and retry logic
- **SC-002**: <2s latency → Optimize streaming pipeline, use streaming agents
- **SC-003**: 100 concurrent calls → Design for concurrency, use connection pooling
- **SC-004**: 90% context accuracy → Implement robust memory integration
- **SC-005**: <5s message processing → Optimize message handling pipeline
- **SC-006**: 100% context preservation → Multi-channel memory integration
- **SC-007**: <1s webhook processing → Efficient webhook handlers, async processing
- **SC-008**: <30s transcription storage → Async transcription processing
- **SC-009**: <1s search retrieval → Optimize vector store queries
- **SC-010**: 99.9% uptime → Health checks, error recovery, monitoring
- **SC-011**: 90% error recovery → Comprehensive retry logic, graceful degradation
- **SC-012**: <30min setup → Clear documentation, examples, quickstart
- **SC-013**: Zero breaking changes → Backward compatibility, opt-in features

## Next Steps

1. **Phase 0**: Complete research.md with all decisions and patterns
2. **Phase 1**: Generate data-model.md, contracts/, and quickstart.md
3. **Phase 1**: Update agent context
4. **Phase 2**: Use `/speckit.tasks` to break down implementation into tasks
