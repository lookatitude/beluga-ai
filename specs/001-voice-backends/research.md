# Research: Voice Backends Integration

**Date**: 2026-01-11  
**Feature**: Voice Backends (001-voice-backends)  
**Purpose**: Research latest versions, Go SDK availability, and integration patterns for voice backend providers

## Research Questions

1. What are the latest versions of LiveKit, Pipecat, and alternative voice backends?
2. Do these backends have Go SDKs or require HTTP/gRPC integration?
3. What are the best practices for integrating these backends with real-time voice pipelines?
4. How do these backends handle WebRTC, authentication, and session management?
5. What are the integration patterns for STT/TTS/S2S pipelines with these backends?

## Primary Providers

### LiveKit

**Decision**: Use LiveKit server-sdk-go (latest version) for primary backend provider

**Rationale**: 
- LiveKit is open-source, enterprise-grade WebRTC platform ([github.com/livekit/livekit](https://github.com/livekit/livekit))
- Server written in Go using Pion WebRTC implementation
- Widely used by OpenAI, xAI, and other major AI companies
- Supports low-latency (<100ms) real-time audio/video
- Multimodal support (audio + video)
- Self-hostable or cloud-hosted options
- Active Go SDK development with comprehensive server SDK

**Latest Version Research** (as of 2026-01-11):
- **LiveKit Server**: v1.9.10 (Latest release: Jan 1, 2026) - [GitHub Releases](https://github.com/livekit/livekit/releases)
- **LiveKit Go Server SDK**: `github.com/livekit/server-sdk-go` - [GitHub](https://github.com/livekit/server-sdk-go) | [Go Docs](https://pkg.go.dev/github.com/livekit/server-sdk-go)
- **LiveKit Protocol**: `github.com/livekit/protocol` - Protocol buffers and shared types
- **Integration Method**: Direct Go SDK integration via server-sdk-go package

**LiveKit Agents Framework** (Reference for Integration Patterns):
- **Repository**: [github.com/livekit/agents](https://github.com/livekit/agents) - Python/Node.js framework
- **Purpose**: Reference implementation for building voice AI agents with LiveKit
- **Key Concepts** (to replicate in Go with Beluga AI):
  - **Agent**: LLM-based application with defined instructions
  - **AgentSession**: Manages interactions with end-users
  - **Entrypoint**: Starting point for interactive session
  - **Worker**: Coordinates job scheduling and launches agents
- **Features to Integrate**:
  - Semantic turn detection (transformer model for detecting user speech completion)
  - Real-time STT/LLM/TTS pipeline orchestration
  - Multi-modal support (audio + video)
  - Data exchange via RPCs and Data APIs
  - MCP (Model Context Protocol) support for tools
  - Built-in telemetry and observability

**Integration Approach with Beluga AI**:
- **Use LiveKit server-sdk-go** for:
  - Room creation and management
  - Participant management (users and agents)
  - WebRTC signaling and media handling
  - Access token generation (JWT-based authentication)
  - Server APIs for room/participant control
- **Use Beluga AI Framework** for:
  - Agent logic (via `pkg/agents`)
  - Voice pipeline orchestration (STT/TTS/S2S via `pkg/voice`)
  - Turn detection (via `pkg/voice/turndetection`)
  - Agent reasoning and tool calling
- **Architecture Pattern**:
  - LiveKit handles WebRTC infrastructure (rooms, participants, media routing)
  - Beluga AI handles AI pipeline (STT → Agent → TTS/S2S)
  - Bridge between LiveKit participants and Beluga AI voice sessions
  - Replicate LiveKit Agents patterns in Go using Beluga AI components

**Alternatives Considered**:
- Using LiveKit Agents Python framework (not feasible - we need Go integration)
- Direct WebRTC implementation (too complex, LiveKit handles this)
- HTTP-only integration (insufficient for real-time requirements)

**Key Features**:
- WebRTC-based real-time communication (SFU architecture)
- Room-based architecture (matches multi-user requirements)
- JWT-based authentication (aligns with FR-021, FR-022)
- Built-in rate limiting and connection management (aligns with FR-023, FR-024)
- Room state persistence (aligns with FR-025, FR-026)
- Data channels for RPC and custom data exchange
- Webhook support for event handling

### Pipecat

**Decision**: Use Pipecat via Daily.co API or HTTP/gRPC integration (Python framework, Go integration via API)

**Rationale**:
- Open-source Python framework for conversational AI ([github.com/pipecat-ai/pipecat](https://github.com/pipecat-ai/pipecat))
- Supports Daily.co, LiveKit, and Twilio rooms
- Low-cost pipeline orchestration
- Good for custom agent workflows
- Active development community
- Designed for real-time voice AI applications

**Latest Version Research** (as of 2026-01-11):
- **Pipecat**: Latest version from [github.com/pipecat-ai/pipecat](https://github.com/pipecat-ai/pipecat)
- **Daily.co**: Daily.co provides REST API and WebSocket for room management
- **Integration Method**: HTTP REST API + WebSocket integration (Pipecat is Python-based, requires API bridge)

**Architecture**:
- Pipecat is a Python framework that orchestrates STT/TTS/LLM pipelines
- Works with Daily.co rooms for WebRTC infrastructure
- Can also work with LiveKit rooms (alternative to LiveKit server-sdk-go)
- Pipeline: Audio → STT → LLM → TTS → Audio

**Alternatives Considered**:
- Direct Python integration (not feasible in Go codebase)
- Re-implement Pipecat patterns in Go (we're doing this with Beluga AI voice pipeline)
- Use Daily.co directly without Pipecat (simpler but less orchestration features)

**Integration Approach**:
- **Option 1**: Use Daily.co Go SDK (if available) for room management, replicate Pipecat patterns with Beluga AI
- **Option 2**: Use Daily.co HTTP API + WebSocket for room management
- **Option 3**: Use Pipecat as reference, implement similar patterns in Go using Beluga AI components
- Handle session state through Daily.co APIs
- Use Beluga AI voice pipeline instead of Pipecat's Python pipeline

**Key Features**:
- Daily.co room integration (WebRTC infrastructure)
- Pipeline orchestration patterns (reference for Beluga AI implementation)
- Low-cost infrastructure
- Custom agent support
- Real-time audio processing

## Alternative Providers

### Vocode

**Decision**: Support Vocode as alternative provider via HTTP API

**Rationale**:
- Open-source voice agent framework
- Supports phone agents and low-latency TTS/ASR
- Good for telephony integration (aligns with FR-015 extensibility)

**Latest Version Research**:
- **Vocode**: Latest version to be researched
- **Integration Method**: HTTP REST API (Python-based framework)

**Integration Approach**:
- HTTP API integration for agent creation and management
- WebSocket for real-time audio streaming
- Support for telephony hooks (SIP integration)

### Vapi

**Decision**: Support Vapi as alternative provider via HTTP API

**Rationale**:
- Turnkey closed-source platform
- Simple API for voice agents
- $0.05/min pricing model
- Good for rapid prototyping

**Latest Version Research**:
- **Vapi API**: Latest version to be researched
- **Integration Method**: HTTP REST API

**Integration Approach**:
- HTTP API for agent configuration
- WebSocket for real-time audio
- Simple integration, less customizable than LiveKit/Pipecat

### Cartesia

**Decision**: Support Cartesia as alternative provider via HTTP API

**Rationale**:
- Developer-first voice API
- Realistic TTS with multilingual support
- Integrations with LiveKit/Pipecat
- Good for high-quality TTS requirements

**Latest Version Research**:
- **Cartesia API**: Latest version to be researched
- **Integration Method**: HTTP REST API and WebSocket

**Integration Approach**:
- HTTP API for TTS generation
- WebSocket for streaming audio
- Can be used alongside LiveKit/Pipecat for TTS

## Go SDK Availability

### LiveKit Go SDK

**Status**: ✅ Available  
**Package**: `github.com/livekit/server-sdk-go`  
**Documentation**: [pkg.go.dev/github.com/livekit/server-sdk-go](https://pkg.go.dev/github.com/livekit/server-sdk-go)  
**GitHub**: [github.com/livekit/server-sdk-go](https://github.com/livekit/server-sdk-go)  
**Version**: Latest version to be determined from go.mod during implementation  
**Usage**: Direct Go SDK integration for:
- Room creation and management
- Participant management
- Access token generation (JWT)
- Server API calls
- Webhook handling
- Data channel management

**Additional Packages**:
- `github.com/livekit/protocol` - Protocol buffers and shared types
- `github.com/livekit/livekit` - Server implementation (if needed for custom server features)

### Daily.co Go SDK

**Status**: ⚠️ To be researched  
**Package**: To be determined  
**Version**: Latest to be determined  
**Usage**: HTTP/gRPC API integration (may require custom HTTP client)

### Alternative Providers

**Status**: HTTP API integration required  
**Method**: Standard `net/http` and `gorilla/websocket` packages  
**Version**: Latest API versions to be determined during implementation

## Integration Patterns

### WebRTC Integration

**Decision**: Use LiveKit Go SDK for WebRTC, HTTP/gRPC for other providers

**Rationale**:
- LiveKit provides complete WebRTC stack
- Other providers use HTTP/WebSocket (simpler integration)
- Matches existing `pkg/voice/transport` patterns

**Implementation**:
- LiveKit: Use SDK's WebRTC peer connection management
- Pipecat/Daily: Use WebSocket transport (already in `pkg/voice/transport`)
- Other providers: HTTP/WebSocket as per provider APIs

### Authentication Integration

**Decision**: Backend-agnostic hooks pattern (per clarification Q1)

**Rationale**:
- Each backend has different auth mechanisms
- Framework provides hooks for integration
- Matches FR-021, FR-022 requirements

**Implementation**:
- Define `AuthHook` interface in `iface/`
- Providers implement their own auth (LiveKit tokens, Daily API keys, etc.)
- Framework provides hook registration for custom auth policies

### Rate Limiting Integration

**Decision**: Provider-specific with framework fallback (per clarification Q2)

**Rationale**:
- Backends have different rate limiting mechanisms
- Framework provides fallback protection
- Matches FR-023, FR-024 requirements

**Implementation**:
- Define `RateLimiter` interface in `iface/`
- Providers implement their own rate limiting
- Framework provides fallback rate limiter for protection

### Session Persistence

**Decision**: Hybrid model - active sessions persist, completed ephemeral (per clarification Q3)

**Rationale**:
- Active sessions need recovery after restarts
- Completed sessions don't need persistence
- Matches FR-025, FR-026 requirements

**Implementation**:
- Active sessions: Persist to in-memory store with optional external storage hooks
- Completed sessions: No persistence, cleanup on session end
- Use provider's session management where available (LiveKit rooms, Daily calls)

### Data Privacy & Retention

**Decision**: Provider-controlled with framework hooks (per clarification Q4)

**Rationale**:
- Backends handle their own data retention policies
- Framework provides hooks for custom policies
- Matches FR-027, FR-028 requirements

**Implementation**:
- Providers implement their own retention policies
- Framework provides `DataRetentionHook` interface for custom policies
- No framework-level data storage (provider-controlled)

## Pipeline Orchestration

### STT/TTS Pipeline

**Decision**: Integrate with existing `pkg/voice` components

**Rationale**:
- STT, TTS, VAD, turn detection already exist in `pkg/voice`
- Backends orchestrate these components
- Matches FR-005 requirement

**Implementation**:
- Backend providers use `pkg/voice/stt`, `pkg/voice/tts`, `pkg/voice/vad`, etc.
- Pipeline orchestrator in `internal/pipeline_orchestrator.go`
- Configurable pipeline order via `PipelineConfiguration`

### S2S Pipeline

**Decision**: Integrate with `pkg/voice/s2s` for speech-to-speech processing

**Rationale**:
- S2S support exists in `pkg/voice/s2s`
- Backends route audio through S2S providers
- Matches FR-006 requirement

**Implementation**:
- Backend providers use `pkg/voice/s2s` providers
- S2S mode bypasses STT/TTS pipeline
- Agent integration hooks for S2S with reasoning (per User Story 2)

## Agent Integration

**Decision**: Integrate with `pkg/agents` for transcript routing and response generation, replicating LiveKit Agents patterns in Go

**Rationale**:
- Agent framework exists and provides integration points
- Backends route transcripts to agents and receive responses
- Matches FR-009 requirement
- LiveKit Agents framework provides reference patterns for voice AI agents
- Beluga AI agents can replicate LiveKit Agents functionality in Go

**LiveKit Agents Integration Patterns** (to replicate with Beluga AI):
1. **Agent Session Management**:
   - LiveKit Agents: `AgentSession` manages user interactions
   - Beluga AI: `VoiceSession` in `pkg/voice/backend` manages interactions
   - Bridge: Connect LiveKit participant to Beluga AI voice session

2. **Pipeline Orchestration**:
   - LiveKit Agents: STT → LLM → TTS pipeline with plugins
   - Beluga AI: Use `pkg/voice/stt`, `pkg/agents`, `pkg/voice/tts` for same pipeline
   - Bridge: Orchestrate components in `internal/pipeline_orchestrator.go`

3. **Turn Detection**:
   - LiveKit Agents: Semantic turn detection (transformer model)
   - Beluga AI: Use `pkg/voice/turndetection` (silence-based or ONNX-based)
   - Bridge: Integrate turn detection into pipeline

4. **Agent Instructions**:
   - LiveKit Agents: Agents have defined instructions (system prompts)
   - Beluga AI: Agents have instructions via `pkg/agents` configuration
   - Bridge: Map agent instructions to Beluga AI agent config

5. **Tool Calling**:
   - LiveKit Agents: MCP (Model Context Protocol) support for tools
   - Beluga AI: Tool calling via `pkg/agents/tools`
   - Bridge: Use Beluga AI tool calling in voice sessions

6. **Data Exchange**:
   - LiveKit Agents: RPCs and Data APIs for client communication
   - Beluga AI: Use LiveKit data channels for custom data exchange
   - Bridge: Implement data channel handlers for agent tools/responses

**Implementation**:
- Backend providers accept agent callbacks or agent instances
- Transcripts routed to agents via `pkg/agents` interfaces
- Responses converted to audio via TTS or S2S
- Replicate LiveKit Agents patterns:
  - Agent entrypoint pattern (entrypoint → agent session → pipeline)
  - Worker pattern (job scheduling, agent lifecycle)
  - Plugin pattern (STT/TTS/LLM as swappable components)

## Observability Integration

**Decision**: Use `pkg/monitoring` and OTEL patterns from `pkg/llms`

**Rationale**:
- Monitoring package provides OTEL setup
- LLMs package shows OTEL tracing patterns
- Matches FR-012, FR-013 requirements

**Implementation**:
- OTEL metrics in `metrics.go` (latency, concurrency, errors, throughput)
- OTEL tracing in all public methods (following `pkg/llms/tracing.go` patterns)
- Structured logging with `logWithOTELContext` helper

## Configuration Integration

**Decision**: Use `pkg/config` patterns and `github.com/go-playground/validator/v10`

**Rationale**:
- Config package provides validation utilities
- Validator library used across framework
- Matches FR-016 requirement

**Implementation**:
- Config structs with mapstructure, yaml, env, validate tags
- Functional options for runtime configuration
- Validation at creation time

## Testing Patterns

**Decision**: Follow `pkg/llms` and `pkg/embeddings` testing patterns

**Rationale**:
- Existing packages show comprehensive test patterns
- Table-driven tests, mocks, benchmarks
- Matches framework testing requirements

**Implementation**:
- Table-driven tests in `advanced_test.go`
- Mocks in `test_utils.go` (following `AdvancedMock` patterns)
- Benchmarks for latency and concurrency
- Integration tests in `tests/integration/voice/backend/`

## Registry Pattern

**Decision**: Follow `pkg/multimodal/registry` and `pkg/embeddings/registry` patterns

**Rationale**:
- Multimodal and embeddings show clean registry patterns
- Global registry with `GetRegistry()`, `Register()`, `Create()`
- Auto-registration via `init.go` files

**Implementation**:
- `registry.go` with `GetRegistry()` function
- `Register()` and `Create()` methods
- Provider registration in `providers/*/init.go` files
- Thread-safe with `sync.RWMutex`

## Dependencies Summary

### Required Go Packages

- `go.opentelemetry.io/otel` v1.39.0 (already in go.mod)
- `github.com/go-playground/validator/v10` v10.30.1 (already in go.mod)
- `github.com/gorilla/websocket` v1.5.3 (already in go.mod)
- `github.com/livekit/server-sdk-go` (to be added - latest version)
- `github.com/livekit/protocol` (to be added - dependency of server-sdk-go)
- `github.com/pion/webrtc` (dependency of LiveKit, not directly needed)

### Integration Points

- `pkg/voice/stt` - Speech-to-text providers
- `pkg/voice/tts` - Text-to-speech providers
- `pkg/voice/s2s` - Speech-to-speech providers
- `pkg/voice/vad` - Voice activity detection
- `pkg/voice/turndetection` - Turn detection
- `pkg/voice/noise` - Noise cancellation
- `pkg/voice/transport` - Audio transport (WebRTC, WebSocket)
- `pkg/voice/session` - Session management (reference implementation)
- `pkg/agents` - Agent framework integration
- `pkg/multimodal` - Multimodal audio content handling
- `pkg/monitoring` - OTEL setup
- `pkg/config` - Configuration validation
- `pkg/schema` - Voice document and message types

## Open Questions Resolved

1. ✅ **Authentication**: Backend-agnostic with framework hooks (clarified)
2. ✅ **Rate Limiting**: Provider-specific with framework fallback (clarified)
3. ✅ **Session Persistence**: Hybrid - active persist, completed ephemeral (clarified)
4. ✅ **Data Privacy**: Provider-controlled with framework hooks (clarified)
5. ✅ **Go SDK Availability**: LiveKit has Go SDK, others use HTTP/gRPC (researched)
6. ✅ **Integration Patterns**: Follow existing `pkg/voice` and framework patterns (researched)

## Integration Architecture: LiveKit + Beluga AI Agents

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    LiveKit Server (v1.9.10)                  │
│  - WebRTC SFU (Selective Forwarding Unit)                  │
│  - Room Management                                          │
│  - Participant Management                                   │
│  - Media Routing                                            │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ WebRTC / Signaling
                            │
┌─────────────────────────────────────────────────────────────┐
│          Beluga AI Voice Backends Package                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  LiveKit Provider (server-sdk-go)                     │  │
│  │  - Room/Participant Management                        │  │
│  │  - Access Token Generation                            │  │
│  │  - WebRTC Connection Handling                         │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Voice Session Manager                                │  │
│  │  - Session Lifecycle                                  │  │
│  │  - Participant ↔ Session Mapping                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Pipeline Orchestrator                                │  │
│  │  - STT → Agent → TTS Pipeline                        │  │
│  │  - S2S Pipeline                                       │  │
│  │  - Turn Detection                                     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │
┌─────────────────────────────────────────────────────────────┐
│              Beluga AI Framework Components                  │
│  - pkg/voice/stt, tts, s2s, vad, turndetection             │
│  - pkg/agents (Agent framework)                            │
│  - pkg/multimodal (Audio content handling)                  │
│  - pkg/monitoring (OTEL observability)                      │
└─────────────────────────────────────────────────────────────┘
```

### Integration Flow (Replicating LiveKit Agents Patterns)

1. **User Connects to LiveKit Room**:
   - User connects via WebRTC to LiveKit server
   - LiveKit provider creates access token (JWT)
   - User joins room as participant

2. **Agent Joins Room**:
   - Beluga AI backend creates agent participant in LiveKit room
   - Agent participant subscribes to user's audio track
   - Agent participant publishes its own audio track

3. **Audio Processing Pipeline** (Replicating LiveKit Agents):
   - **STT**: User audio → `pkg/voice/stt` → Transcript
   - **Turn Detection**: `pkg/voice/turndetection` → Detect speech completion
   - **Agent Processing**: Transcript → `pkg/agents` → Agent response
   - **TTS**: Agent response → `pkg/voice/tts` → Audio
   - **Audio Output**: Audio → Agent participant track → User

4. **S2S Pipeline** (Alternative):
   - User audio → `pkg/voice/s2s` → Audio output
   - Optional: Route through agent for reasoning before S2S

### Key Integration Points

1. **LiveKit Room Management**:
   ```go
   // Use server-sdk-go for room operations
   import "github.com/livekit/server-sdk-go"
   
   roomService := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
   room, err := roomService.CreateRoom(ctx, &livekit.CreateRoomRequest{
       Name: roomName,
   })
   ```

2. **Participant Management**:
   ```go
   // Create agent participant
   participant, err := roomService.CreateParticipant(ctx, &livekit.CreateParticipantRequest{
       Room: roomName,
       Identity: "agent-1",
   })
   ```

3. **Audio Track Handling**:
   ```go
   // Subscribe to user audio track
   // Process audio through Beluga AI pipeline
   // Publish agent audio track
   ```

4. **Agent Integration**:
   ```go
   // Route transcript to Beluga AI agent
   agent := agents.NewAgent(ctx, agentConfig)
   response, err := agent.Invoke(ctx, transcript)
   ```

## Next Steps

1. ✅ Verify exact LiveKit Go SDK package: `github.com/livekit/server-sdk-go`
2. ✅ Research LiveKit Agents patterns for replication in Go
3. Research Daily.co Go SDK or HTTP API documentation
4. Research Vocode, Vapi, Cartesia API versions and endpoints
5. ✅ Design data model based on entities from spec
6. ✅ Create API contracts for backend interfaces
7. ✅ Design provider factory patterns matching existing packages
8. **NEW**: Design LiveKit-specific integration patterns (room management, participant handling, audio track processing)
9. **NEW**: Design agent session bridge between LiveKit participants and Beluga AI voice sessions
10. **NEW**: Implement LiveKit Agents patterns in Go (entrypoint, worker, plugin architecture)
