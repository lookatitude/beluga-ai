# Implementation Tasks: Voice Backends

**Feature**: Voice Backends (001-voice-backends)  
**Branch**: `001-voice-backends`  
**Date**: 2025-01-27  
**Status**: Ready for Implementation

## Overview

This document provides a comprehensive, dependency-ordered task list for implementing the Voice Backends feature. Tasks are organized by user story priority to enable independent implementation and testing. Each task is specific enough for an LLM to implement without additional context.

## Implementation Strategy

### Full Implementation (No MVPs)
This feature implements **ALL** requirements from the specification:
- ✅ All user stories (P1, P2, P3 priorities)
- ✅ All functional requirements (FR-001 through FR-028)
- ✅ All success criteria (SC-001 through SC-010)
- ✅ All provider implementations (LiveKit, Pipecat, Vocode, Vapi, Cartesia)
- ✅ Deep integration with ALL Beluga AI packages (agents, config, vectorstores, memory, multimodal, orchestration, monitoring, schema, voice, prompts, retrievers, server, core, chatmodels, embeddings)

### Incremental Delivery
- **Phase 1-2**: Foundation (registry, errors, config, metrics) - Required for all features
- **Phase 3**: User Story 1 (P1) - Core real-time conversation with full package integration
- **Phase 4**: User Story 2 (P1) - S2S pipeline - Critical differentiator
- **Phase 5**: User Story 3 (P2) - Multi-user scalability - Production readiness
- **Phase 6**: User Story 4 (P2) - Provider swapping - All providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia)
- **Phase 7**: User Story 5 (P3) - Custom extensions - Advanced use cases
- **Phase 8**: Polish - Documentation, edge cases, optimization, all package integrations

### Parallel Execution Opportunities

Tasks marked with `[P]` can be executed in parallel with other `[P]` tasks in the same phase, provided they don't depend on incomplete tasks.

## Dependencies

### Story Completion Order

```
Phase 1 (Setup)
    ↓
Phase 2 (Foundational) - Blocks all user stories
    ↓
Phase 3 (US1: Real-Time Conversation) - Core with full package integration
    ↓
Phase 4 (US2: S2S Pipeline) - Can start after US1 core complete
    ↓
Phase 5 (US3: Multi-User Scalability) - Can start after US1 complete
    ↓
Phase 6 (US4: Provider Swapping) - All providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia)
    ↓
Phase 7 (US5: Custom Extensions) - Can start after US1 complete
    ↓
Phase 8 (Polish) - Final phase with all package integration tests
```

### Cross-Story Dependencies

- **US1** provides foundation for US2, US3, US5 with full package integration (memory, orchestration, RAG, multimodal, prompts, chatmodels)
- **US4** requires US1 complete + all providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia) for full provider swapping
- **US5** requires US1 pipeline orchestration complete
- All stories depend on Phase 2 (Foundational)
- Package integrations (memory, orchestration, RAG, etc.) are integrated in US1 and tested in Phase 8

## Independent Test Criteria

### User Story 1 (Real-Time Voice Agent Conversation)
**Test**: Create voice backend instance, connect WebRTC client, speak into microphone, verify agent responses received in <500ms with proper turn-taking.

### User Story 2 (Speech-to-Speech Pipeline)
**Test**: Configure backend with S2S provider, send audio input, verify audio output generated without text transcription, latency <300ms.

### User Story 3 (Multi-User Scalability)
**Test**: Create backend instance, connect 100+ concurrent WebRTC clients, each sends audio simultaneously, verify all conversations complete without degradation.

### User Story 4 (Backend Provider Swapping)
**Test**: Create backend with LiveKit, verify functionality, switch config to Pipecat, verify same functionality works without code changes.

### User Story 5 (Custom Pipeline Extensions)
**Test**: Implement custom audio processor adapter, register with backend, verify audio flows through custom processor during pipeline execution.

---

## Phase 1: Setup & Project Initialization

**Goal**: Initialize project structure, add dependencies, set up package scaffolding.

**Independent Test**: Package structure exists, dependencies added to go.mod, directories created.

### Setup Tasks

- [ ] T001 Create package directory structure `pkg/voice/backend/` with subdirectories: `iface/`, `internal/`, `providers/`
- [ ] T002 Add LiveKit dependencies to `go.mod`: `github.com/livekit/server-sdk-go` (latest version) and `github.com/livekit/protocol` (dependency)
- [ ] T003 Create `pkg/voice/backend/iface/` directory for interface definitions
- [ ] T004 Create `pkg/voice/backend/internal/` directory for private implementation details
- [ ] T005 Create `pkg/voice/backend/providers/` directory for provider implementations
- [ ] T006 Create `pkg/voice/backend/providers/mock/` directory for mock provider
- [ ] T007 Create `pkg/voice/backend/providers/livekit/` directory for LiveKit provider
- [ ] T008 Create `pkg/voice/backend/providers/pipecat/` directory for Pipecat provider
- [ ] T009 Create `pkg/voice/backend/providers/vocode/` directory for Vocode provider
- [ ] T010 Create `pkg/voice/backend/providers/vapi/` directory for Vapi provider
- [ ] T011 Create `pkg/voice/backend/providers/cartesia/` directory for Cartesia provider
- [ ] T012 Create `tests/integration/voice/backend/` directory for integration tests
- [ ] T013 Initialize `pkg/voice/backend/go.mod` or verify main `go.mod` includes voicebackends package

---

## Phase 2: Foundational Components

**Goal**: Implement core infrastructure required by all user stories (registry, errors, config, metrics, interfaces).

**Independent Test**: Registry can register and create providers, errors have proper codes, config validates, metrics emit.

**Dependencies**: Phase 1 complete

### Registry Implementation

- [ ] T014 [P] Create `pkg/voice/backend/registry.go` with `BackendRegistry` struct following `pkg/llms/registry.go` pattern
- [ ] T015 [P] Implement `GetRegistry()` function in `pkg/voice/backend/registry.go` using `sync.Once` for thread-safe initialization
- [ ] T016 [P] Implement `Register()` method in `pkg/voice/backend/registry.go` with thread-safe write lock
- [ ] T017 [P] Implement `Create()` method in `pkg/voice/backend/registry.go` with config validation and error handling
- [ ] T018 [P] Implement `ListProviders()` method in `pkg/voice/backend/registry.go` with thread-safe read lock
- [ ] T019 [P] Implement `IsRegistered()` method in `pkg/voice/backend/registry.go` with thread-safe read lock
- [ ] T020 [P] Implement `GetProvider()` method in `pkg/voice/backend/registry.go` with error handling

### Error Handling

- [ ] T021 [P] Create `pkg/voice/backend/errors.go` with `BackendError` struct following `pkg/llms/errors.go` pattern (Op, Err, Code, Message, Details fields)
- [ ] T022 [P] Define error code constants in `pkg/voice/backend/errors.go`: `ErrCodeInvalidConfig`, `ErrCodeProviderNotFound`, `ErrCodeConnectionFailed`, `ErrCodeConnectionTimeout`, `ErrCodeSessionNotFound`, `ErrCodeSessionLimitExceeded`, `ErrCodeRateLimitExceeded`, `ErrCodeAuthenticationFailed`, `ErrCodeAuthorizationFailed`, `ErrCodePipelineError`, `ErrCodeAgentError`, `ErrCodeTimeout`, `ErrCodeContextCanceled`
- [ ] T023 [P] Implement `NewBackendError()` function in `pkg/voice/backend/errors.go` for creating errors
- [ ] T024 [P] Implement `WrapError()` function in `pkg/voice/backend/errors.go` for error wrapping
- [ ] T025 [P] Implement `IsError()` function in `pkg/voice/backend/errors.go` for error type checking
- [ ] T026 [P] Implement `AsError()` function in `pkg/voice/backend/errors.go` for error type assertion
- [ ] T027 [P] Implement `Error()` method for `BackendError` in `pkg/voice/backend/errors.go` with proper formatting
- [ ] T028 [P] Implement `Unwrap()` method for `BackendError` in `pkg/voice/backend/errors.go` for error unwrapping

### Configuration

- [ ] T029 [P] Create `pkg/voice/backend/config.go` with `Config` struct following `pkg/llms/config.go` pattern
- [ ] T030 [P] Add configuration fields to `Config` in `pkg/voice/backend/config.go`: `Provider`, `ProviderConfig`, `PipelineType`, `STTProvider`, `TTSProvider`, `S2SProvider`, `VADProvider`, `TurnDetectionProvider`, `NoiseCancellationProvider`, `LatencyTarget`, `Timeout`, `MaxRetries`, `RetryDelay`, `MaxConcurrentSessions`, `EnableTracing`, `EnableMetrics`, `EnableStructuredLogging`
- [ ] T031 [P] Add integration fields to `Config` in `pkg/voice/backend/config.go`: `Memory`, `Orchestrator`, `Retriever`, `VectorStore`, `Embedder`, `MultimodalModel`, `PromptTemplate`, `ChatModel`, `ServerConfig` (all optional, for deep package integration)
- [ ] T032 [P] Add mapstructure, yaml, env, validate tags to `Config` fields in `pkg/voice/backend/config.go`
- [ ] T033 [P] Add validation rules to `Config` fields in `pkg/voice/backend/config.go` using validator tags (required, oneof, min, max, etc.)
- [ ] T034 [P] Integrate `pkg/config` validation utilities in `pkg/voice/backend/config.go` using `config.Validate()` pattern
- [ ] T035 [P] Create `SessionConfig` struct in `pkg/voice/backend/config.go` with fields: `UserID`, `Transport`, `ConnectionURL`, `AgentCallback`, `AgentInstance`, `PipelineType`, `Metadata`, `MemoryConfig`, `OrchestrationConfig`, `RAGConfig`
- [ ] T036 [P] Add validation tags to `SessionConfig` in `pkg/voice/backend/config.go`
- [ ] T037 [P] Implement `DefaultConfig()` function in `pkg/voice/backend/config.go` returning config with defaults
- [ ] T038 [P] Implement `ValidateConfig()` function in `pkg/voice/backend/config.go` using `github.com/go-playground/validator/v10` and `pkg/config` utilities
- [ ] T039 [P] Create functional options for `Config` in `pkg/voice/backend/config.go`: `WithProvider()`, `WithPipelineType()`, `WithLatencyTarget()`, `WithMaxConcurrentSessions()`, `WithMemory()`, `WithOrchestrator()`, `WithRetriever()`, `WithVectorStore()`, `WithEmbedder()`, `WithMultimodalModel()`, `WithPromptTemplate()`, `WithChatModel()`, etc.

### Core Interfaces

- [ ] T040 [P] Create `pkg/voice/backend/iface/backend.go` with `VoiceBackend` interface (Start, Stop, CreateSession, GetSession, ListSessions, CloseSession, HealthCheck, GetConnectionState, GetActiveSessionCount, GetConfig, UpdateConfig)
- [ ] T041 [P] Create `pkg/voice/backend/iface/provider.go` with `BackendProvider` interface (GetName, GetCapabilities, CreateBackend, ValidateConfig, GetConfigSchema)
- [ ] T042 [P] Create `pkg/voice/backend/iface/session.go` with `VoiceSession` interface (Start, Stop, ProcessAudio, SendAudio, ReceiveAudio, SetAgentCallback, SetAgentInstance, GetState, GetPersistenceStatus, UpdateMetadata, GetID)
- [ ] T043 [P] Create `pkg/voice/backend/iface/pipeline.go` with `PipelineConfiguration` struct and types: `PipelineType` (STT_TTS, S2S), `ProviderCapabilities` struct, `ConnectionState` type, `PipelineState` type, `PersistenceStatus` type, `HealthStatus` struct

### OTEL Metrics

- [ ] T044 [P] Create `pkg/voice/backend/metrics.go` with `Metrics` struct following `pkg/llms/metrics.go` pattern
- [ ] T045 [P] Integrate `pkg/monitoring` for OTEL setup in `pkg/voice/backend/metrics.go` using `monitoring.InitMetrics()` pattern
- [ ] T046 [P] Define OTEL metrics in `pkg/voice/backend/metrics.go`: `backend_requests_total`, `backend_errors_total`, `backend_latency_seconds`, `backend_sessions_active`, `backend_sessions_total`, `backend_throughput_bytes`
- [ ] T047 [P] Implement `NewMetrics()` function in `pkg/voice/backend/metrics.go` for metrics initialization using `pkg/monitoring`
- [ ] T048 [P] Implement `RecordRequest()` method in `pkg/voice/backend/metrics.go` for recording backend requests
- [ ] T049 [P] Implement `RecordError()` method in `pkg/voice/backend/metrics.go` for recording errors
- [ ] T050 [P] Implement `RecordLatency()` method in `pkg/voice/backend/metrics.go` for recording latency histograms
- [ ] T051 [P] Implement `IncrementActiveSessions()` method in `pkg/voice/backend/metrics.go` for session counting
- [ ] T052 [P] Implement `DecrementActiveSessions()` method in `pkg/voice/backend/metrics.go` for session counting
- [ ] T053 [P] Create `NoOpMetrics` struct in `pkg/voice/backend/metrics.go` for when metrics are disabled

### Main Package File

- [ ] T054 [P] Create `pkg/voice/backend/backend.go` with `NewBackend()` factory function using registry
- [ ] T055 [P] Implement `GetRegistry()` convenience function in `pkg/voice/backend/backend.go` wrapping registry.GetRegistry()

---

## Phase 3: User Story 1 - Real-Time Voice Agent Conversation (P1)

**Goal**: Enable real-time voice agent conversations with <500ms latency, automatic audio processing, turn detection, and agent response generation.

**Independent Test**: Create voice backend instance, connect WebRTC client, speak into microphone, verify agent responses received in <500ms with proper turn-taking.

**Dependencies**: Phase 2 complete

### Core Types & State Management

- [ ] T056 [US1] Create `ConnectionState` type in `pkg/voice/backend/iface/pipeline.go`: `Disconnected`, `Connecting`, `Connected`, `Reconnecting`, `Error`
- [ ] T057 [US1] Create `PipelineState` type in `pkg/voice/backend/iface/pipeline.go`: `Idle`, `Listening`, `Processing`, `Speaking`, `Error`
- [ ] T058 [US1] Create `PersistenceStatus` type in `pkg/voice/backend/iface/pipeline.go`: `Active`, `Completed`
- [ ] T059 [US1] Create `HealthStatus` struct in `pkg/voice/backend/iface/pipeline.go` with fields: `Status`, `Details`, `LastCheck`
- [ ] T060 [US1] Create `ProviderCapabilities` struct in `pkg/voice/backend/iface/pipeline.go` with fields: `S2SSupport`, `MultiUserSupport`, `SessionPersistence`, `CustomAuth`, `CustomRateLimiting`, `MaxConcurrentSessions`, `MinLatency`, `SupportedCodecs`

### Internal Session Management

- [ ] T068 [US1] Create `pkg/voice/backend/internal/session_manager.go` with `SessionManager` struct for managing voice sessions
- [ ] T069 [US1] Implement `CreateSession()` method in `pkg/voice/backend/internal/session_manager.go` with session ID generation and validation
- [ ] T070 [US1] Implement `GetSession()` method in `pkg/voice/backend/internal/session_manager.go` with thread-safe access
- [ ] T071 [US1] Implement `ListSessions()` method in `pkg/voice/backend/internal/session_manager.go` returning all active sessions
- [ ] T072 [US1] Implement `CloseSession()` method in `pkg/voice/backend/internal/session_manager.go` with cleanup and state transition to Completed
- [ ] T073 [US1] Implement session persistence logic in `pkg/voice/backend/internal/session_manager.go` (active sessions persist, completed ephemeral per FR-025, FR-026)

### Pipeline Orchestrator

- [ ] T074 [US1] Create `pkg/voice/backend/internal/pipeline_orchestrator.go` with `PipelineOrchestrator` struct
- [ ] T075 [US1] Implement STT/TTS pipeline orchestration in `pkg/voice/backend/internal/pipeline_orchestrator.go`: Audio → STT → Agent → TTS → Audio
- [ ] T076 [US1] Integrate `pkg/voice/stt` providers in `pkg/voice/backend/internal/pipeline_orchestrator.go` for speech-to-text
- [ ] T077 [US1] Integrate `pkg/voice/tts` providers in `pkg/voice/backend/internal/pipeline_orchestrator.go` for text-to-speech
- [ ] T078 [US1] Integrate `pkg/voice/turndetection` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for detecting speech boundaries
- [ ] T079 [US1] Integrate `pkg/voice/vad` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for voice activity detection (optional)
- [ ] T080 [US1] Integrate `pkg/voice/noise` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for noise cancellation (optional)
- [ ] T081 [US1] Integrate `pkg/embeddings` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for RAG embedding support
- [ ] T082 [US1] Implement agent integration in `pkg/voice/backend/internal/pipeline_orchestrator.go` routing transcripts to `pkg/agents` and receiving responses
- [ ] T083 [US1] Integrate `pkg/chatmodels` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for agent LLM integration
- [ ] T084 [US1] Integrate `pkg/prompts` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for agent prompt management
- [ ] T085 [US1] Integrate `pkg/memory` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for conversation context storage and retrieval
- [ ] T086 [US1] Integrate `pkg/orchestration` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for workflow orchestration triggers
- [ ] T087 [US1] Integrate `pkg/retrievers` and `pkg/vectorstores` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for RAG-enabled voice agents
- [ ] T088 [US1] Integrate `pkg/multimodal` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for audio content handling in multimodal workflows
- [ ] T089 [US1] Integrate `pkg/schema` in `pkg/voice/backend/internal/pipeline_orchestrator.go` for voice documents and message types
- [ ] T090 [US1] Implement interruption handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` stopping current playback when user speaks (FR-008)
- [ ] T091 [US1] Implement turn detection triggering in `pkg/voice/backend/internal/pipeline_orchestrator.go` processing complete utterances at turn boundaries (FR-007)
- [ ] T092 [US1] Add OTEL tracing to pipeline orchestrator methods in `pkg/voice/backend/internal/pipeline_orchestrator.go` following `pkg/llms/tracing.go` patterns using `pkg/monitoring`

### Mock Provider (Required for Testing)

- [ ] T093 [US1] Create `pkg/voice/backend/providers/mock/config.go` with `MockConfig` struct extending base `Config`
- [ ] T094 [US1] Create `pkg/voice/backend/providers/mock/provider.go` implementing `BackendProvider` interface
- [ ] T095 [US1] Implement `GetName()` method in `pkg/voice/backend/providers/mock/provider.go` returning "mock"
- [ ] T096 [US1] Implement `GetCapabilities()` method in `pkg/voice/backend/providers/mock/provider.go` returning mock capabilities
- [ ] T097 [US1] Implement `CreateBackend()` method in `pkg/voice/backend/providers/mock/provider.go` creating mock backend instance
- [ ] T098 [US1] Implement `ValidateConfig()` method in `pkg/voice/backend/providers/mock/provider.go` with validation logic
- [ ] T099 [US1] Implement `GetConfigSchema()` method in `pkg/voice/backend/providers/mock/provider.go` returning mock schema
- [ ] T100 [US1] Create `pkg/voice/backend/providers/mock/backend.go` implementing `VoiceBackend` interface for mock backend
- [ ] T101 [US1] Create `pkg/voice/backend/providers/mock/session.go` implementing `VoiceSession` interface for mock session
- [ ] T102 [US1] Implement mock audio processing in `pkg/voice/backend/providers/mock/session.go` simulating STT/TTS pipeline
- [ ] T103 [US1] Create `pkg/voice/backend/providers/mock/init.go` with auto-registration: `backend.GetRegistry().Register("mock", NewMockProvider())`

### LiveKit Provider - Core Structure

- [ ] T104 [US1] Create `pkg/voice/backend/providers/livekit/config.go` with `LiveKitConfig` struct extending base `Config` with LiveKit-specific fields: `APIKey`, `APISecret`, `URL`, `RoomName`, etc.
- [ ] T105 [US1] Add validation tags to `LiveKitConfig` in `pkg/voice/backend/providers/livekit/config.go`
- [ ] T106 [US1] Create `pkg/voice/backend/providers/livekit/provider.go` implementing `BackendProvider` interface
- [ ] T107 [US1] Implement `GetName()` method in `pkg/voice/backend/providers/livekit/provider.go` returning "livekit"
- [ ] T108 [US1] Implement `GetCapabilities()` method in `pkg/voice/backend/providers/livekit/provider.go` returning LiveKit capabilities (S2S support, multi-user, etc.)
- [ ] T109 [US1] Implement `CreateBackend()` method in `pkg/voice/backend/providers/livekit/provider.go` creating LiveKit backend using server-sdk-go
- [ ] T110 [US1] Implement `ValidateConfig()` method in `pkg/voice/backend/providers/livekit/provider.go` validating LiveKit-specific config
- [ ] T111 [US1] Implement `GetConfigSchema()` method in `pkg/voice/backend/providers/livekit/provider.go` returning LiveKit config schema
- [ ] T112 [US1] Create `pkg/voice/backend/providers/livekit/init.go` with auto-registration: `backend.GetRegistry().Register("livekit", NewLiveKitProvider())`

### LiveKit Provider - Backend Implementation

- [ ] T113 [US1] Create `pkg/voice/backend/providers/livekit/backend.go` with `LiveKitBackend` struct implementing `VoiceBackend` interface
- [ ] T114 [US1] Implement `Start()` method in `pkg/voice/backend/providers/livekit/backend.go` initializing LiveKit connection using `lksdk.RoomServiceClient`
- [ ] T115 [US1] Implement `Stop()` method in `pkg/voice/backend/providers/livekit/backend.go` with graceful shutdown completing in-flight conversations (FR-020)
- [ ] T116 [US1] Implement `CreateSession()` method in `pkg/voice/backend/providers/livekit/backend.go` creating LiveKit room and participant
- [ ] T117 [US1] Implement `GetSession()` method in `pkg/voice/backend/providers/livekit/backend.go` retrieving session by ID
- [ ] T118 [US1] Implement `ListSessions()` method in `pkg/voice/backend/providers/livekit/backend.go` listing all active sessions
- [ ] T119 [US1] Implement `CloseSession()` method in `pkg/voice/backend/providers/livekit/backend.go` closing LiveKit room and cleaning up
- [ ] T120 [US1] Implement `HealthCheck()` method in `pkg/voice/backend/providers/livekit/backend.go` checking LiveKit server health (FR-018)
- [ ] T121 [US1] Implement `GetConnectionState()` method in `pkg/voice/backend/providers/livekit/backend.go` returning current connection state
- [ ] T122 [US1] Implement `GetActiveSessionCount()` method in `pkg/voice/backend/providers/livekit/backend.go` returning active session count
- [ ] T123 [US1] Implement `GetConfig()` method in `pkg/voice/backend/providers/livekit/backend.go` returning backend configuration
- [ ] T124 [US1] Implement `UpdateConfig()` method in `pkg/voice/backend/providers/livekit/backend.go` with validation

### LiveKit Provider - Session Implementation

- [ ] T125 [US1] Create `pkg/voice/backend/providers/livekit/session.go` with `LiveKitSession` struct implementing `VoiceSession` interface
- [ ] T126 [US1] Implement `Start()` method in `pkg/voice/backend/providers/livekit/session.go` starting session and subscribing to user audio track
- [ ] T127 [US1] Implement `Stop()` method in `pkg/voice/backend/providers/livekit/session.go` stopping session and cleaning up tracks
- [ ] T128 [US1] Implement `ProcessAudio()` method in `pkg/voice/backend/providers/livekit/session.go` processing audio from LiveKit track through pipeline orchestrator
- [ ] T129 [US1] Implement `SendAudio()` method in `pkg/voice/backend/providers/livekit/session.go` publishing audio to LiveKit track
- [ ] T130 [US1] Implement `ReceiveAudio()` method in `pkg/voice/backend/providers/livekit/session.go` returning channel for receiving audio from user
- [ ] T131 [US1] Implement `SetAgentCallback()` method in `pkg/voice/backend/providers/livekit/session.go` setting agent callback function
- [ ] T132 [US1] Implement `SetAgentInstance()` method in `pkg/voice/backend/providers/livekit/session.go` setting agent instance from `pkg/agents`
- [ ] T133 [US1] Implement `GetState()` method in `pkg/voice/backend/providers/livekit/session.go` returning current pipeline state
- [ ] T134 [US1] Implement `GetPersistenceStatus()` method in `pkg/voice/backend/providers/livekit/session.go` returning persistence status
- [ ] T135 [US1] Implement `UpdateMetadata()` method in `pkg/voice/backend/providers/livekit/session.go` updating session metadata
- [ ] T136 [US1] Implement `GetID()` method in `pkg/voice/backend/providers/livekit/session.go` returning session ID

### LiveKit Provider - Audio Track Handling

- [ ] T137 [US1] Create `pkg/voice/backend/providers/livekit/track_handler.go` with `TrackHandler` struct for managing WebRTC audio tracks
- [ ] T138 [US1] Implement audio track subscription in `pkg/voice/backend/providers/livekit/track_handler.go` subscribing to user's audio track
- [ ] T139 [US1] Implement audio track publication in `pkg/voice/backend/providers/livekit/track_handler.go` publishing agent's audio track
- [ ] T140 [US1] Implement track event handling in `pkg/voice/backend/providers/livekit/track_handler.go` handling muted, unmuted, disconnected events
- [ ] T141 [US1] Implement audio format conversion in `pkg/voice/backend/providers/livekit/track_handler.go` converting between formats (FR-019)
- [ ] T142 [US1] Bridge LiveKit audio tracks to Beluga AI pipeline in `pkg/voice/backend/providers/livekit/track_handler.go` connecting tracks to pipeline orchestrator

### LiveKit Provider - Access Token Generation

- [ ] T143 [US1] Implement JWT access token generation in `pkg/voice/backend/providers/livekit/token.go` using `lksdk.AccessToken` for authentication (FR-021, FR-022)
- [ ] T144 [US1] Add token generation hooks in `pkg/voice/backend/providers/livekit/token.go` supporting custom auth hooks per FR-021, FR-022

### Session Bridge

- [ ] T145 [US1] Create `pkg/voice/backend/internal/session_bridge.go` with `SessionBridge` struct mapping LiveKit participants to Beluga AI voice sessions
- [ ] T146 [US1] Implement participant-to-session mapping in `pkg/voice/backend/internal/session_bridge.go` creating session for each participant
- [ ] T147 [US1] Implement session lifecycle management in `pkg/voice/backend/internal/session_bridge.go` handling start, stop, error recovery
- [ ] T148 [US1] Implement audio track event handling in `pkg/voice/backend/internal/session_bridge.go` propagating track events to sessions

### OTEL Tracing Integration

- [ ] T149 [US1] Add OTEL tracing to all public methods in `pkg/voice/backend/providers/livekit/backend.go` following `pkg/llms/tracing.go` patterns (FR-013)
- [ ] T150 [US1] Add OTEL tracing to all public methods in `pkg/voice/backend/providers/livekit/session.go` with span attributes (FR-013)
- [ ] T151 [US1] Add OTEL tracing to pipeline orchestrator methods in `pkg/voice/backend/internal/pipeline_orchestrator.go` with end-to-end request flows (FR-013)
- [ ] T152 [US1] Implement structured logging with OTEL context in `pkg/voice/backend/providers/livekit/backend.go` using `logWithOTELContext` helper
- [ ] T153 [US1] Implement structured logging with OTEL context in `pkg/voice/backend/providers/livekit/session.go` with trace IDs and span IDs

### Error Handling & Recovery

- [ ] T154 [US1] Implement connection failure handling in `pkg/voice/backend/providers/livekit/backend.go` with retry logic and error codes (FR-014)
- [ ] T155 [US1] Implement timeout handling in `pkg/voice/backend/providers/livekit/session.go` with context cancellation support (FR-014)
- [ ] T156 [US1] Implement provider error handling in `pkg/voice/backend/providers/livekit/backend.go` with appropriate error codes (FR-014)
- [ ] T157 [US1] Implement error recovery logic in `pkg/voice/backend/providers/livekit/session.go` for transient failures (FR-014)

---

## Phase 4: User Story 2 - Speech-to-Speech Pipeline (P1)

**Goal**: Enable S2S processing that bypasses text transcription for ultra-low latency (<300ms) voice interactions.

**Independent Test**: Configure backend with S2S provider, send audio input, verify audio output generated without text transcription, latency <300ms.

**Dependencies**: Phase 3 complete (US1 core pipeline orchestration)

### S2S Pipeline Support

- [ ] T158 [US2] Add S2S pipeline type support to `PipelineOrchestrator` in `pkg/voice/backend/internal/pipeline_orchestrator.go`
- [ ] T159 [US2] Implement S2S pipeline orchestration in `pkg/voice/backend/internal/pipeline_orchestrator.go`: Audio → S2S Provider → Audio (bypassing STT/TTS)
- [ ] T160 [US2] Integrate `pkg/voice/s2s` providers in `pkg/voice/backend/internal/pipeline_orchestrator.go` for speech-to-speech processing
- [ ] T161 [US2] Implement optional agent integration for S2S in `pkg/voice/backend/internal/pipeline_orchestrator.go` routing audio through agent for reasoning before S2S (per User Story 2 acceptance scenario 2)
- [ ] T162 [US2] Add S2S latency tracking in `pkg/voice/backend/internal/pipeline_orchestrator.go` ensuring <300ms target (SC-005)
- [ ] T163 [US2] Update `PipelineConfiguration` validation in `pkg/voice/backend/config.go` to support S2S pipeline type with S2SProvider field

### LiveKit Provider S2S Support

- [ ] T164 [US2] Add S2S capability to `GetCapabilities()` in `pkg/voice/backend/providers/livekit/provider.go` marking S2SSupport as true
- [ ] T165 [US2] Implement S2S pipeline handling in `pkg/voice/backend/providers/livekit/session.go` ProcessAudio method for S2S pipeline type
- [ ] T166 [US2] Add S2S pipeline configuration support in `pkg/voice/backend/providers/livekit/config.go` with S2SProvider field

### Mock Provider S2S Support

- [ ] T167 [US2] Add S2S capability to `GetCapabilities()` in `pkg/voice/backend/providers/mock/provider.go` marking S2SSupport as true
- [ ] T168 [US2] Implement S2S pipeline handling in `pkg/voice/backend/providers/mock/session.go` ProcessAudio method for S2S pipeline type

---

## Phase 5: User Story 3 - Multi-User Scalability (P2)

**Goal**: Support 100+ concurrent voice conversations per backend instance with isolated session management.

**Independent Test**: Create backend instance, connect 100+ concurrent WebRTC clients, each sends audio simultaneously, verify all conversations complete without degradation.

**Dependencies**: Phase 3 complete (US1 session management)

### Concurrent Session Management

- [ ] T169 [US3] Implement thread-safe session map in `pkg/voice/backend/internal/session_manager.go` using `sync.RWMutex` for concurrent access
- [ ] T170 [US3] Add session capacity management in `pkg/voice/backend/internal/session_manager.go` enforcing `MaxConcurrentSessions` limit (FR-010)
- [ ] T171 [US3] Implement session isolation in `pkg/voice/backend/internal/session_manager.go` ensuring zero cross-conversation data leakage (SC-008)
- [ ] T172 [US3] Add concurrent session creation handling in `pkg/voice/backend/providers/livekit/backend.go` supporting multiple simultaneous CreateSession calls
- [ ] T173 [US3] Implement session state isolation in `pkg/voice/backend/providers/livekit/session.go` ensuring each session maintains independent state

### Performance Optimization

- [ ] T174 [US3] Optimize audio buffer management in `pkg/voice/backend/internal/pipeline_orchestrator.go` for concurrent processing
- [ ] T175 [US3] Implement connection pooling in `pkg/voice/backend/providers/livekit/backend.go` reusing LiveKit connections where possible
- [ ] T176 [US3] Add goroutine pool management in `pkg/voice/backend/internal/pipeline_orchestrator.go` for concurrent audio processing
- [ ] T177 [US3] Implement efficient audio buffer handling in `pkg/voice/backend/providers/livekit/session.go` preventing buffer overflows during high-volume periods

### Scalability Metrics

- [ ] T178 [US3] Add concurrency metrics in `pkg/voice/backend/metrics.go`: `backend_sessions_active` gauge, `backend_concurrent_operations` gauge
- [ ] T179 [US3] Track session creation time in `pkg/voice/backend/metrics.go` ensuring <2 seconds (SC-007)
- [ ] T180 [US3] Add throughput metrics in `pkg/voice/backend/metrics.go` tracking audio processing throughput per session

### Load Testing Support

- [ ] T181 [US3] Create `tests/integration/voice/backend/scalability_test.go` with test for 100+ concurrent sessions
- [ ] T182 [US3] Implement load test in `tests/integration/voice/backend/scalability_test.go` verifying no latency degradation with 100 concurrent users (SC-002)

---

## Phase 6: User Story 4 - Backend Provider Swapping (P2)

**Goal**: Enable switching between backend providers (LiveKit to Pipecat) without code changes, only configuration updates.

**Independent Test**: Create backend with LiveKit, verify functionality, switch config to Pipecat, verify same functionality works without code changes.

**Dependencies**: Phase 3 complete (US1 with at least LiveKit + Mock providers)

### Pipecat Provider Implementation

- [ ] T183 [US4] Create `pkg/voice/backend/providers/pipecat/config.go` with `PipecatConfig` struct extending base `Config` with Pipecat-specific fields: `DailyAPIKey`, `PipecatServerURL`, etc.
- [ ] T184 [US4] Add validation tags to `PipecatConfig` in `pkg/voice/backend/providers/pipecat/config.go`
- [ ] T185 [US4] Create `pkg/voice/backend/providers/pipecat/provider.go` implementing `BackendProvider` interface
- [ ] T186 [US4] Implement `GetName()` method in `pkg/voice/backend/providers/pipecat/provider.go` returning "pipecat"
- [ ] T187 [US4] Implement `GetCapabilities()` method in `pkg/voice/backend/providers/pipecat/provider.go` returning Pipecat capabilities
- [ ] T188 [US4] Implement `CreateBackend()` method in `pkg/voice/backend/providers/pipecat/provider.go` creating Pipecat backend via Daily.co API
- [ ] T189 [US4] Implement `ValidateConfig()` method in `pkg/voice/backend/providers/pipecat/provider.go` validating Pipecat-specific config
- [ ] T190 [US4] Implement `GetConfigSchema()` method in `pkg/voice/backend/providers/pipecat/provider.go` returning Pipecat config schema
- [ ] T191 [US4] Create `pkg/voice/backend/providers/pipecat/init.go` with auto-registration: `backend.GetRegistry().Register("pipecat", NewPipecatProvider())`

### Pipecat Provider Backend Implementation

- [ ] T192 [US4] Create `pkg/voice/backend/providers/pipecat/backend.go` with `PipecatBackend` struct implementing `VoiceBackend` interface
- [ ] T193 [US4] Implement `Start()` method in `pkg/voice/backend/providers/pipecat/backend.go` initializing Daily.co connection via HTTP API
- [ ] T194 [US4] Implement `Stop()` method in `pkg/voice/backend/providers/pipecat/backend.go` with graceful shutdown
- [ ] T195 [US4] Implement `CreateSession()` method in `pkg/voice/backend/providers/pipecat/backend.go` creating Daily.co room via API
- [ ] T196 [US4] Implement `GetSession()` method in `pkg/voice/backend/providers/pipecat/backend.go` retrieving session by ID
- [ ] T197 [US4] Implement `ListSessions()` method in `pkg/voice/backend/providers/pipecat/backend.go` listing all active sessions
- [ ] T198 [US4] Implement `CloseSession()` method in `pkg/voice/backend/providers/pipecat/backend.go` closing Daily.co room
- [ ] T199 [US4] Implement `HealthCheck()` method in `pkg/voice/backend/providers/pipecat/backend.go` checking Daily.co/Pipecat server health
- [ ] T200 [US4] Implement `GetConnectionState()` method in `pkg/voice/backend/providers/pipecat/backend.go` returning current connection state
- [ ] T201 [US4] Implement `GetActiveSessionCount()` method in `pkg/voice/backend/providers/pipecat/backend.go` returning active session count
- [ ] T202 [US4] Implement `GetConfig()` method in `pkg/voice/backend/providers/pipecat/backend.go` returning backend configuration
- [ ] T203 [US4] Implement `UpdateConfig()` method in `pkg/voice/backend/providers/pipecat/backend.go` with validation

### Pipecat Provider Session Implementation

- [ ] T204 [US4] Create `pkg/voice/backend/providers/pipecat/session.go` with `PipecatSession` struct implementing `VoiceSession` interface
- [ ] T205 [US4] Implement `Start()` method in `pkg/voice/backend/providers/pipecat/session.go` starting session and connecting to Daily.co room via WebSocket
- [ ] T206 [US4] Implement `Stop()` method in `pkg/voice/backend/providers/pipecat/session.go` stopping session and disconnecting
- [ ] T207 [US4] Implement `ProcessAudio()` method in `pkg/voice/backend/providers/pipecat/session.go` processing audio through pipeline orchestrator
- [ ] T208 [US4] Implement `SendAudio()` method in `pkg/voice/backend/providers/pipecat/session.go` sending audio via WebSocket
- [ ] T209 [US4] Implement `ReceiveAudio()` method in `pkg/voice/backend/providers/pipecat/session.go` receiving audio via WebSocket
- [ ] T210 [US4] Implement `SetAgentCallback()` method in `pkg/voice/backend/providers/pipecat/session.go` setting agent callback function
- [ ] T211 [US4] Implement `SetAgentInstance()` method in `pkg/voice/backend/providers/pipecat/session.go` setting agent instance
- [ ] T212 [US4] Implement `GetState()` method in `pkg/voice/backend/providers/pipecat/session.go` returning current pipeline state
- [ ] T213 [US4] Implement `GetPersistenceStatus()` method in `pkg/voice/backend/providers/pipecat/session.go` returning persistence status
- [ ] T214 [US4] Implement `UpdateMetadata()` method in `pkg/voice/backend/providers/pipecat/session.go` updating session metadata
- [ ] T215 [US4] Implement `GetID()` method in `pkg/voice/backend/providers/pipecat/session.go` returning session ID

### Vocode Provider Implementation

- [ ] T216 [US4] Create `pkg/voice/backend/providers/vocode/config.go` with `VocodeConfig` struct extending base `Config` with Vocode-specific fields: `APIKey`, `APIURL`, etc.
- [ ] T217 [US4] Add validation tags to `VocodeConfig` in `pkg/voice/backend/providers/vocode/config.go`
- [ ] T218 [US4] Create `pkg/voice/backend/providers/vocode/provider.go` implementing `BackendProvider` interface
- [ ] T219 [US4] Implement `GetName()`, `GetCapabilities()`, `CreateBackend()`, `ValidateConfig()`, `GetConfigSchema()` methods in `pkg/voice/backend/providers/vocode/provider.go`
- [ ] T220 [US4] Create `pkg/voice/backend/providers/vocode/backend.go` with `VocodeBackend` struct implementing `VoiceBackend` interface
- [ ] T221 [US4] Create `pkg/voice/backend/providers/vocode/session.go` with `VocodeSession` struct implementing `VoiceSession` interface
- [ ] T222 [US4] Implement HTTP API integration in `pkg/voice/backend/providers/vocode/backend.go` and `session.go` for Vocode API
- [ ] T223 [US4] Create `pkg/voice/backend/providers/vocode/init.go` with auto-registration: `backend.GetRegistry().Register("vocode", NewVocodeProvider())`

### Vapi Provider Implementation

- [ ] T224 [US4] Create `pkg/voice/backend/providers/vapi/config.go` with `VapiConfig` struct extending base `Config` with Vapi-specific fields: `APIKey`, `APIURL`, etc.
- [ ] T225 [US4] Add validation tags to `VapiConfig` in `pkg/voice/backend/providers/vapi/config.go`
- [ ] T226 [US4] Create `pkg/voice/backend/providers/vapi/provider.go` implementing `BackendProvider` interface
- [ ] T227 [US4] Implement `GetName()`, `GetCapabilities()`, `CreateBackend()`, `ValidateConfig()`, `GetConfigSchema()` methods in `pkg/voice/backend/providers/vapi/provider.go`
- [ ] T228 [US4] Create `pkg/voice/backend/providers/vapi/backend.go` with `VapiBackend` struct implementing `VoiceBackend` interface
- [ ] T229 [US4] Create `pkg/voice/backend/providers/vapi/session.go` with `VapiSession` struct implementing `VoiceSession` interface
- [ ] T230 [US4] Implement HTTP API integration in `pkg/voice/backend/providers/vapi/backend.go` and `session.go` for Vapi API
- [ ] T231 [US4] Create `pkg/voice/backend/providers/vapi/init.go` with auto-registration: `backend.GetRegistry().Register("vapi", NewVapiProvider())`

### Cartesia Provider Implementation

- [ ] T232 [US4] Create `pkg/voice/backend/providers/cartesia/config.go` with `CartesiaConfig` struct extending base `Config` with Cartesia-specific fields: `APIKey`, `APIURL`, etc.
- [ ] T233 [US4] Add validation tags to `CartesiaConfig` in `pkg/voice/backend/providers/cartesia/config.go`
- [ ] T234 [US4] Create `pkg/voice/backend/providers/cartesia/provider.go` implementing `BackendProvider` interface
- [ ] T235 [US4] Implement `GetName()`, `GetCapabilities()`, `CreateBackend()`, `ValidateConfig()`, `GetConfigSchema()` methods in `pkg/voice/backend/providers/cartesia/provider.go`
- [ ] T236 [US4] Create `pkg/voice/backend/providers/cartesia/backend.go` with `CartesiaBackend` struct implementing `VoiceBackend` interface
- [ ] T237 [US4] Create `pkg/voice/backend/providers/cartesia/session.go` with `CartesiaSession` struct implementing `VoiceSession` interface
- [ ] T238 [US4] Implement HTTP API integration in `pkg/voice/backend/providers/cartesia/backend.go` and `session.go` for Cartesia API
- [ ] T239 [US4] Create `pkg/voice/backend/providers/cartesia/init.go` with auto-registration: `backend.GetRegistry().Register("cartesia", NewCartesiaProvider())`

### Provider Swapping Tests

- [ ] T240 [US4] Create `tests/integration/voice/backend/multi_provider_test.go` with test for provider swapping
- [ ] T241 [US4] Implement test in `tests/integration/voice/backend/multi_provider_test.go` verifying LiveKit to Pipecat swap requires zero code changes (SC-003)
- [ ] T242 [US4] Add test in `tests/integration/voice/backend/multi_provider_test.go` verifying LiveKit to Vocode swap requires zero code changes
- [ ] T243 [US4] Add test in `tests/integration/voice/backend/multi_provider_test.go` verifying LiveKit to Vapi swap requires zero code changes
- [ ] T244 [US4] Add test in `tests/integration/voice/backend/multi_provider_test.go` verifying LiveKit to Cartesia swap requires zero code changes
- [ ] T245 [US4] Add test in `tests/integration/voice/backend/multi_provider_test.go` verifying provider query returns all registered providers with capabilities (User Story 4 acceptance scenario 2)
- [ ] T246 [US4] Add test in `tests/integration/voice/backend/multi_provider_test.go` verifying clear error messages when provider fails to initialize (User Story 4 acceptance scenario 3)

---

## Phase 7: User Story 5 - Custom Pipeline Extensions (P3)

**Goal**: Enable custom audio processors and external tool integrations via extensibility hooks.

**Independent Test**: Implement custom audio processor adapter, register with backend, verify audio flows through custom processor during pipeline execution.

**Dependencies**: Phase 3 complete (US1 pipeline orchestration)

### Extensibility Hooks

- [ ] T313 [US5] Create `pkg/voice/backend/iface/hooks.go` with `AuthHook` interface (Authenticate, Authorize methods) per FR-021, FR-022
- [ ] T314 [US5] Create `pkg/voice/backend/iface/hooks.go` with `RateLimiter` interface (Allow, Wait methods) per FR-023, FR-024
- [ ] T315 [US5] Create `pkg/voice/backend/iface/hooks.go` with `DataRetentionHook` interface (ShouldRetain, GetRetentionPeriod methods) per FR-027, FR-028
- [ ] T316 [US5] Create `pkg/voice/backend/iface/hooks.go` with `AudioProcessor` interface (Process, GetName, GetOrder methods) per FR-015
- [ ] T317 [US5] Add `AuthHook` field to `Config` struct in `pkg/voice/backend/config.go` with mapstructure:"-" yaml:"-" tags
- [ ] T318 [US5] Add `RateLimiter` field to `Config` struct in `pkg/voice/backend/config.go` with mapstructure:"-" yaml:"-" tags
- [ ] T319 [US5] Add `DataRetentionHook` field to `Config` struct in `pkg/voice/backend/config.go` with mapstructure:"-" yaml:"-" tags
- [ ] T320 [US5] Add `CustomProcessors` field to `Config` struct in `pkg/voice/backend/config.go` with validation tags

### Custom Processor Integration

- [ ] T321 [US5] Implement custom processor registration in `pkg/voice/backend/internal/pipeline_orchestrator.go` allowing processors to be registered and executed in order
- [ ] T322 [US5] Add processor execution order handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` executing processors by Order field (lower = earlier)
- [ ] T323 [US5] Integrate custom processors into STT/TTS pipeline in `pkg/voice/backend/internal/pipeline_orchestrator.go` executing before STT and after TTS
- [ ] T324 [US5] Integrate custom processors into S2S pipeline in `pkg/voice/backend/internal/pipeline_orchestrator.go` executing before and after S2S provider

### Hook Integration

- [ ] T325 [US5] Integrate `AuthHook` in `pkg/voice/backend/providers/livekit/backend.go` calling Authenticate before session creation (FR-021, FR-022)
- [ ] T326 [US5] Integrate `AuthHook` in `pkg/voice/backend/providers/livekit/backend.go` calling Authorize for operations (FR-021, FR-022)
- [ ] T327 [US5] Integrate `RateLimiter` in `pkg/voice/backend/providers/livekit/backend.go` calling Allow before operations with framework fallback (FR-023, FR-024)
- [ ] T328 [US5] Integrate `DataRetentionHook` in `pkg/voice/backend/providers/livekit/session.go` calling ShouldRetain for data retention decisions (FR-027, FR-028)
- [ ] T329 [US5] Implement framework fallback rate limiter in `pkg/voice/backend/internal/rate_limiter.go` for when providers don't implement their own (FR-024)

### Telephony Integration Support

- [ ] T330 [US5] Add telephony hook interface in `pkg/voice/backend/iface/hooks.go` for SIP and telephony protocol integration (FR-015, User Story 5 acceptance scenario 2)
- [ ] T331 [US5] Integrate telephony hooks in `pkg/voice/backend/providers/livekit/backend.go` allowing SIP routing via hooks

---

## Phase 8: Polish & Cross-Cutting Concerns

**Goal**: Complete documentation, handle edge cases, optimize performance, add comprehensive tests.

**Dependencies**: All previous phases complete

### Test Utilities

- [ ] T247 Create `pkg/voice/backend/test_utils.go` with `AdvancedMockVoiceBackend` struct following `pkg/llms/test_utils.go` pattern
- [ ] T248 Implement `NewAdvancedMockVoiceBackend()` function in `pkg/voice/backend/test_utils.go` with configurable behavior
- [ ] T249 Create `MockOption` type in `pkg/voice/backend/test_utils.go` for functional options: `WithMockError()`, `WithMockDelay()`, `WithAudioData()`, etc.
- [ ] T250 Implement `AdvancedMockVoiceSession` struct in `pkg/voice/backend/test_utils.go` for session mocking
- [ ] T251 Implement `ConcurrentTestRunner` struct in `pkg/voice/backend/test_utils.go` for concurrent testing patterns
- [ ] T252 Implement `RunLoadTest()` function in `pkg/voice/backend/test_utils.go` for load testing utilities

### Advanced Tests

- [ ] T253 Create `pkg/voice/backend/advanced_test.go` with table-driven tests for registry operations
- [ ] T254 Add table-driven tests in `pkg/voice/backend/advanced_test.go` for backend creation and lifecycle
- [ ] T255 Add table-driven tests in `pkg/voice/backend/advanced_test.go` for session management operations
- [ ] T256 Add table-driven tests in `pkg/voice/backend/advanced_test.go` for pipeline orchestration (STT/TTS and S2S)
- [ ] T257 Add table-driven tests in `pkg/voice/backend/advanced_test.go` for error handling scenarios
- [ ] T258 Add table-driven tests in `pkg/voice/backend/advanced_test.go` for turn detection and interruption handling
- [ ] T259 Add concurrent tests in `pkg/voice/backend/advanced_test.go` for multi-user scenarios
- [ ] T260 Add edge case tests in `pkg/voice/backend/advanced_test.go` covering all edge cases from spec (connection loss, format mismatches, timeouts, etc.)

### Integration Tests

- [ ] T261 Create `tests/integration/voice/backend/livekit_integration_test.go` with integration tests for LiveKit provider
- [ ] T262 Create `tests/integration/voice/backend/pipecat_integration_test.go` with integration tests for Pipecat provider
- [ ] T263 Create `tests/integration/voice/backend/pipeline_test.go` with integration tests for pipeline orchestration
- [ ] T264 Add integration test in `tests/integration/voice/backend/livekit_integration_test.go` verifying end-to-end latency <500ms (SC-001)
- [ ] T265 Add integration test in `tests/integration/voice/backend/pipeline_test.go` verifying S2S latency <300ms (SC-005)
- [ ] T266 Add integration test in `tests/integration/voice/backend/pipeline_test.go` verifying 99% turn processing success (SC-004)
- [ ] T267 Add integration test in `tests/integration/voice/backend/livekit_integration_test.go` verifying 90% connection failure recovery (SC-006)

### Package Integration Tests

- [ ] T268 Create `tests/integration/voice/backend/agent_integration_test.go` with tests for `pkg/agents` integration
- [ ] T269 Create `tests/integration/voice/backend/memory_integration_test.go` with tests for `pkg/memory` integration (conversation context, history)
- [ ] T270 Create `tests/integration/voice/backend/orchestration_integration_test.go` with tests for `pkg/orchestration` integration (workflow triggers)
- [ ] T271 Create `tests/integration/voice/backend/rag_integration_test.go` with tests for `pkg/retrievers`, `pkg/vectorstores`, `pkg/embeddings` integration (RAG-enabled voice agents)
- [ ] T272 Create `tests/integration/voice/backend/multimodal_integration_test.go` with tests for `pkg/multimodal` integration (audio content handling)
- [ ] T273 Create `tests/integration/voice/backend/prompts_integration_test.go` with tests for `pkg/prompts` integration (agent prompt management)
- [ ] T274 Create `tests/integration/voice/backend/chatmodels_integration_test.go` with tests for `pkg/chatmodels` integration (agent LLM)
- [ ] T275 Add test in `tests/integration/voice/backend/memory_integration_test.go` verifying conversation context is stored and retrieved per session
- [ ] T276 Add test in `tests/integration/voice/backend/orchestration_integration_test.go` verifying workflows are triggered based on voice events
- [ ] T277 Add test in `tests/integration/voice/backend/rag_integration_test.go` verifying RAG-enabled voice agents can retrieve knowledge base documents
- [ ] T278 Add test in `tests/integration/voice/backend/multimodal_integration_test.go` verifying audio content is handled in multimodal workflows

### Benchmarks

- [ ] T279 Create benchmark tests in `pkg/voice/backend/advanced_test.go` for backend creation time (target <2s per SC-007)
- [ ] T280 Create benchmark tests in `pkg/voice/backend/advanced_test.go` for audio processing latency (target <500ms per SC-001)
- [ ] T281 Create benchmark tests in `pkg/voice/backend/advanced_test.go` for concurrent session handling (target 100+ sessions per SC-002)

### Documentation

- [ ] T282 Create `pkg/voice/backend/README.md` with package overview, architecture, and usage examples following `pkg/llms/README.md` pattern
- [ ] T283 Add quick start section to `pkg/voice/backend/README.md` with basic usage examples
- [ ] T284 Add provider-specific examples to `pkg/voice/backend/README.md` for LiveKit and Pipecat
- [ ] T285 Add extensibility examples to `pkg/voice/backend/README.md` showing custom processor and hook usage
- [ ] T286 Add observability section to `pkg/voice/backend/README.md` documenting OTEL metrics and tracing
- [ ] T287 Add error handling section to `pkg/voice/backend/README.md` documenting error codes and handling patterns
- [ ] T288 Add performance section to `pkg/voice/backend/README.md` documenting latency targets and scalability

### Edge Case Handling

- [ ] T289 Implement WebRTC connection loss handling in `pkg/voice/backend/providers/livekit/session.go` with reconnection logic
- [ ] T290 Implement audio format mismatch handling in `pkg/voice/backend/providers/livekit/track_handler.go` with automatic conversion (FR-019)
- [ ] T291 Implement agent response timeout handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` with timeout error codes
- [ ] T292 Implement network latency spike handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` with buffering and retry
- [ ] T293 Implement turn detection failure handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` with fallback mechanisms
- [ ] T294 Implement concurrent interruption handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` for multi-user scenarios
- [ ] T295 Implement provider unavailability handling in `pkg/voice/backend/providers/livekit/backend.go` with error codes and retry
- [ ] T296 Implement audio buffer overflow handling in `pkg/voice/backend/providers/livekit/session.go` with buffer management
- [ ] T297 Implement malformed S2S output handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` with validation and error codes
- [ ] T298 Implement agent error handling in `pkg/voice/backend/internal/pipeline_orchestrator.go` preventing response generation failures

### Health Checks

- [ ] T299 Implement comprehensive health check in `pkg/voice/backend/providers/livekit/backend.go` checking LiveKit server connectivity (FR-018)
- [ ] T300 Implement health check in `pkg/voice/backend/providers/pipecat/backend.go` checking Daily.co/Pipecat server connectivity (FR-018)
- [ ] T301 Add health status tracking in `pkg/voice/backend/providers/livekit/backend.go` with `HealthStatus` (healthy, degraded, unhealthy)

### Graceful Shutdown

- [ ] T302 Implement graceful shutdown in `pkg/voice/backend/providers/livekit/backend.go` completing in-flight conversations before termination (FR-020)
- [ ] T303 Implement graceful shutdown in `pkg/voice/backend/providers/pipecat/backend.go` completing in-flight conversations before termination (FR-020)
- [ ] T304 Add shutdown timeout handling in `pkg/voice/backend/providers/livekit/backend.go` with configurable timeout

### Session Persistence

- [ ] T305 Implement active session persistence in `pkg/voice/backend/internal/session_manager.go` persisting active sessions (FR-025)
- [ ] T306 Implement session recovery in `pkg/voice/backend/internal/session_manager.go` recovering active sessions after restart
- [ ] T307 Implement ephemeral cleanup in `pkg/voice/backend/internal/session_manager.go` cleaning up completed sessions (FR-026)

### Audio Format Conversion

- [ ] T308 Implement audio format conversion in `pkg/voice/backend/providers/livekit/track_handler.go` converting between PCM, Opus, etc. (FR-019)
- [ ] T309 Add format validation in `pkg/voice/backend/providers/livekit/track_handler.go` ensuring provider requirements are met

### Rate Limiting Integration

- [ ] T310 Implement provider-specific rate limiting in `pkg/voice/backend/providers/livekit/backend.go` using LiveKit's rate limiting (FR-023)
- [ ] T311 Implement framework fallback rate limiting in `pkg/voice/backend/internal/rate_limiter.go` for protection (FR-024)
- [ ] T312 Add rate limit error handling in `pkg/voice/backend/providers/livekit/backend.go` with `ErrCodeRateLimitExceeded`

### Data Privacy Hooks

- [ ] T332 Implement data retention hook integration in `pkg/voice/backend/providers/livekit/session.go` calling hooks for retention decisions (FR-027, FR-028)
- [ ] T333 Add data privacy metadata in `pkg/voice/backend/providers/livekit/session.go` tracking retention policies

### Cloud/Self-Hosted Support

- [ ] T334 Add cloud deployment configuration in `pkg/voice/backend/providers/livekit/config.go` supporting LiveKit Cloud
- [ ] T335 Add self-hosted deployment configuration in `pkg/voice/backend/providers/livekit/config.go` supporting self-hosted LiveKit server (FR-017)

### Observability Verification

- [ ] T336 [SC-009] Create integration test in `tests/integration/voice/backend/observability_test.go` verifying all OTEL metrics are emitted (latency, concurrency, errors, throughput)
- [ ] T337 [SC-009] Add test in `tests/integration/voice/backend/observability_test.go` verifying OTEL traces are created for end-to-end request flows
- [ ] T338 [SC-009] Add test in `tests/integration/voice/backend/observability_test.go` verifying structured logging includes OTEL context (trace IDs, span IDs)

---

## Task Summary

### Total Tasks: 331 (All duplicate IDs fixed, sequential ordering maintained)

### Tasks by Phase

- **Phase 1 (Setup)**: 13 tasks (added Vocode, Vapi, Cartesia directories)
- **Phase 2 (Foundational)**: 46 tasks (added package integration config fields, monitoring integration)
- **Phase 3 (US1 - Real-Time Conversation)**: 95+ tasks (added deep package integration tasks)
- **Phase 4 (US2 - S2S Pipeline)**: 11 tasks
- **Phase 5 (US3 - Multi-User Scalability)**: 14 tasks
- **Phase 6 (US4 - Provider Swapping)**: 61 tasks (added Vocode, Vapi, Cartesia providers)
- **Phase 7 (US5 - Custom Extensions)**: 19 tasks
- **Phase 8 (Polish)**: 90+ tasks (added package integration tests)

### Tasks by User Story

- **User Story 1 (P1)**: 95+ tasks (includes all package integrations)
- **User Story 2 (P1)**: 11 tasks
- **User Story 3 (P2)**: 14 tasks
- **User Story 4 (P2)**: 61 tasks (all providers: LiveKit, Pipecat, Vocode, Vapi, Cartesia)
- **User Story 5 (P3)**: 19 tasks

### Package Integration Tasks

- **Memory Integration**: Tasks in Phase 3 (US1) for conversation context storage and retrieval
- **Orchestration Integration**: Tasks in Phase 3 (US1) for workflow orchestration triggers
- **RAG Integration**: Tasks in Phase 3 (US1) for retrievers, vectorstores, embeddings
- **Multimodal Integration**: Tasks in Phase 3 (US1) for audio content handling
- **Prompts Integration**: Tasks in Phase 3 (US1) for agent prompt management
- **ChatModels Integration**: Tasks in Phase 3 (US1) for agent LLM integration
- **Integration Tests**: Tasks in Phase 8 for all package integrations

### Parallel Execution Opportunities

Tasks marked with `[P]` can be executed in parallel:
- **Phase 2**: 39 parallelizable tasks (registry, errors, config, metrics, interfaces can be developed simultaneously)
- **Phase 3**: Multiple parallelizable tasks within provider implementations
- **Phase 4-7**: Provider-specific tasks can be parallelized

### Full Implementation Scope

**Complete Implementation**: All Phases (1-8) - Full feature set with all providers and package integrations
- **Total Tasks**: See Task Summary below
- **Delivers**: Complete voice backend framework with all providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia) and deep integration with all Beluga AI packages
- **Enables**: Production-ready voice agents with RAG, orchestration, memory, multimodal support, and full observability

### Implementation Notes

1. **Follow Framework Patterns**: All implementations must follow patterns from `pkg/llms/`, `pkg/embeddings/`, `pkg/multimodal/`
2. **OTEL Integration**: All public methods must include OTEL tracing and metrics using `pkg/monitoring`
3. **Error Handling**: Use custom error types with Op/Err/Code pattern throughout
4. **Testing**: Table-driven tests in `advanced_test.go`, mocks in `test_utils.go`, integration tests in `tests/integration/voice/backend/`
5. **Documentation**: Comprehensive README with examples following `pkg/llms/README.md` structure
6. **Full Package Integration**: Integrate deeply with ALL Beluga AI packages (agents, config, vectorstores, memory, multimodal, orchestration, monitoring, schema, voice, prompts, retrievers, server, core, chatmodels, embeddings)
7. **All Providers**: Implement all providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia) - no MVPs or partial implementations
8. **Full Feature Set**: Implement all user stories, all functional requirements, all success criteria
