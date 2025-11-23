# Research: Voice Agents Feature

**Date**: 2025-01-27  
**Feature**: Voice Agents Framework  
**Status**: Complete

## Overview

This document consolidates research findings to resolve all NEEDS CLARIFICATION markers from the feature specification, based on the BELUGA_AI_VOICE_AGENTS_PROPOSAL.md and existing Beluga AI framework patterns.

---

## 1. Performance Targets

### Decision: Latency and Throughput Requirements

**Latency Target**: Sub-200ms for voice interactions (as specified in proposal success criteria)

**Rationale**:
- The proposal explicitly states "Sub-200ms latency for voice interactions" as a success criterion
- This aligns with industry standards for real-time voice AI (typically 150-300ms for perceived real-time)
- Allows for network round-trip, STT processing, agent inference, and TTS generation

**Concurrent Sessions**: 
- Initial target: Support 100+ concurrent voice sessions per instance
- Scale horizontally for higher throughput
- Use connection pooling and resource management

**Throughput**:
- Target: Process 1000+ audio chunks per second per instance
- Use buffering and batching where appropriate
- Implement backpressure handling

**Alternatives Considered**:
- 100ms target: Too aggressive, would require significant optimization overhead
- 500ms target: Too permissive, would feel laggy to users
- 200ms chosen as balance between quality and feasibility

---

## 2. Authentication and Authorization

### Decision: Integration with Existing Beluga AI Auth

**Authentication Method**: Defer to application-level authentication

**Rationale**:
- Voice sessions are transport-layer concerns (WebRTC, WebSocket)
- Beluga AI framework doesn't mandate auth patterns (follows DIP)
- Voice package should accept authenticated contexts from callers
- Application layer handles auth tokens, API keys, user sessions

**Authorization**:
- Voice package provides hooks for authorization callbacks
- Session creation accepts authorization context
- Provider API keys managed via config package (existing pattern)

**Implementation**:
- Use `context.Context` for auth propagation (standard Go pattern)
- Voice session config accepts optional `AuthChecker` interface
- Integrate with existing `pkg/config` for credential management

**Alternatives Considered**:
- Built-in JWT validation: Too opinionated, violates DIP
- OAuth2 integration: Overkill for framework-level package
- Context-based approach chosen for flexibility

---

## 3. Data Retention and Privacy

### Decision: Framework-Agnostic with Memory Integration

**Retention Policy**: Delegate to Beluga AI memory package

**Rationale**:
- Voice package integrates with existing `pkg/memory` package
- Memory package already handles retention policies
- Voice sessions use memory for conversation history
- Application layer controls retention via memory configuration

**Privacy**:
- Audio data not persisted by voice package (streaming only)
- Transcripts stored via memory package (existing patterns)
- Provider API keys managed securely via config package
- No voice data logging by default (configurable)

**Implementation**:
- Voice session uses `memory.Memory` interface for history
- Audio streams are ephemeral (not stored)
- Transcripts follow memory package retention policies
- Observability respects privacy settings (configurable)

**Alternatives Considered**:
- Built-in retention: Duplicates memory package functionality
- No retention: Loses conversation context
- Memory integration chosen for consistency

---

## 4. Security Requirements

### Decision: TLS/DTLS for Voice Data Transmission

**Encryption in Transit**: 
- WebRTC uses DTLS (mandatory for WebRTC)
- WebSocket connections use WSS (TLS)
- REST API calls use HTTPS
- All provider API calls use TLS

**Rationale**:
- WebRTC standard requires DTLS for media encryption
- Industry standard for real-time voice communication
- Provider APIs (Deepgram, OpenAI, etc.) require HTTPS
- No additional encryption needed at application layer

**API Key Security**:
- Use existing `pkg/config` for secure credential management
- Support environment variables, secret managers
- No hardcoded keys in code
- Follow existing config package patterns

**Input Validation**:
- Validate all audio format inputs
- Sanitize transcript text before agent processing
- Rate limiting via circuit breakers (existing pattern)
- Input size limits to prevent DoS

**Implementation**:
- WebRTC transport enforces DTLS (standard)
- WebSocket wrapper enforces WSS in production
- Config package handles API key security
- Input validation in each provider

**Alternatives Considered**:
- Application-layer encryption: Redundant with DTLS/TLS
- Custom encryption: Non-standard, maintenance burden
- Standard protocols chosen for interoperability

---

## 5. Language Support

### Decision: Multi-Language Support with Provider Abstraction

**Initial Languages**: English (en) as primary, with extensibility for others

**Rationale**:
- Proposal specifies multiple providers (Deepgram, Google, Azure, OpenAI)
- Each provider supports multiple languages
- Language detection available in most providers
- Framework should support any language providers support

**Implementation**:
- Language specified in STT/TTS config (ISO 639-1 codes)
- Auto-detection available via provider APIs
- Language passed through to providers
- No language-specific logic in framework (provider handles it)

**Provider Language Support**:
- Deepgram: 30+ languages
- Google Cloud: 100+ languages  
- Azure: 100+ languages
- OpenAI Whisper: 50+ languages

**Future Extensibility**:
- Easy to add new languages via provider configuration
- No framework changes needed for new languages
- Language-specific features (accents, dialects) handled by providers

**Alternatives Considered**:
- English-only: Too restrictive for framework
- Built-in language detection: Duplicates provider functionality
- Provider-based approach chosen for flexibility

---

## 6. Integration with Existing Packages

### Decision: Leverage Existing Beluga AI Packages

**Config Package** (`pkg/config`):
- Use for provider configuration (API keys, endpoints, timeouts)
- Follow existing config patterns (YAML, env vars, viper)
- Provider-specific configs extend base config

**LLMs Package** (`pkg/llms`):
- Voice agents use existing LLM interfaces
- No changes needed to LLM package
- Voice session wraps LLM calls

**Prompts Package** (`pkg/prompts`):
- Use for formatting agent prompts from transcripts
- Integrate with existing prompt templates
- Support voice-specific prompt formatting

**Memory Package** (`pkg/memory`):
- Use for conversation history management
- Voice sessions save transcripts to memory
- Retrieve context from memory for agent calls

**Monitoring Package** (`pkg/monitoring`):
- Use for observability (metrics, tracing, logging)
- Follow existing OTEL patterns
- Voice-specific metrics extend base metrics

**Agents Package** (`pkg/agents`):
- Voice sessions integrate with existing agent interfaces
- Use `iface.CompositeAgent` for agent interactions
- Support all existing agent types (ReAct, etc.)

**Rationale**:
- Follows DIP (depend on abstractions)
- Reuses existing, tested infrastructure
- Maintains consistency across framework
- Reduces code duplication

---

## 7. Package Structure Decisions

### Decision: Multi-Provider Package Structure

**Package Layout** (following constitution):
```
pkg/voice/
├── iface/                    # Interfaces (STTProvider, TTSProvider, VADProvider, etc.)
├── internal/                 # Private implementations
├── providers/                # Provider implementations
│   ├── stt/                  # STT providers (deepgram, google, azure, openai)
│   ├── tts/                  # TTS providers (openai, google, azure, elevenlabs)
│   ├── vad/                  # VAD providers (silero, energy, webrtc)
│   └── ...
├── stt/                      # STT package (main)
├── tts/                      # TTS package (main)
├── vad/                      # VAD package (main)
├── turndetection/            # Turn detection package
├── transport/               # Transport package (WebRTC)
├── session/                  # Session management package
├── noise/                    # Noise cancellation package
├── config.go                 # Configuration structs
├── metrics.go                # OTEL metrics
├── errors.go                 # Error types
├── registry.go               # Global provider registry
├── test_utils.go             # Testing utilities
├── advanced_test.go         # Comprehensive tests
└── README.md                 # Documentation
```

**Rationale**:
- Follows mandatory package structure from constitution
- Each sub-package (stt, tts, vad) is independently usable
- Provider implementations in providers/ subdirectories
- Global registry pattern for multi-provider support

---

## 8. Technology Choices

### Decision: Go Standard Library + Existing Dependencies

**Core Dependencies**:
- `context` (standard library) for cancellation
- `sync` (standard library) for concurrency
- Existing Beluga AI packages (config, llms, memory, etc.)

**Audio Processing**:
- WebRTC: Use `pion/webrtc` (industry standard Go WebRTC library)
- Audio formats: Standard PCM, Opus codec support
- ONNX runtime: For Silero VAD and turn detection models

**Observability**:
- OpenTelemetry (already in framework)
- Structured logging (zap, already in framework)
- Metrics via OTEL (already in framework)

**Testing**:
- `testing` (standard library)
- `testify` (already in framework)
- Mock generation (already in framework)

**Rationale**:
- Minimize new dependencies
- Use existing framework infrastructure
- Follow Go best practices
- Leverage proven libraries (pion/webrtc)

**Alternatives Considered**:
- Custom WebRTC implementation: Too complex, maintenance burden
- Different audio libraries: pion/webrtc is standard
- Custom observability: Framework already has OTEL

---

## 9. Error Handling Patterns

### Decision: Follow Existing Op/Err/Code Pattern

**Error Structure**:
```go
type VoiceError struct {
    Op   string // operation name
    Err  error  // underlying error
    Code string // error code (e.g., "stt_provider_failed", "tts_timeout")
}
```

**Error Codes**:
- Provider-specific codes (e.g., `stt_deepgram_connection_failed`)
- Generic codes (e.g., `stt_timeout`, `tts_invalid_config`)
- Follow existing error code patterns from other packages

**Error Handling**:
- All public methods return errors
- Errors preserve context via error wrapping
- Retryable vs non-retryable errors (for circuit breakers)
- Error codes for programmatic handling

**Rationale**:
- Consistency with existing framework packages
- Enables proper error handling and observability
- Supports retry logic and circuit breakers
- Follows Go error handling best practices

---

## 10. Testing Strategy

### Decision: Comprehensive Testing Following Framework Patterns

**Unit Tests**:
- 100% coverage of public methods (framework requirement)
- Table-driven tests (framework pattern)
- Mock providers for testing (test_utils.go pattern)

**Integration Tests**:
- End-to-end voice session tests
- Provider integration tests (with test API keys)
- Cross-package integration (voice + agents + memory)

**Performance Tests**:
- Benchmark tests for critical paths (VAD, audio processing)
- Latency measurements (target <200ms)
- Concurrent session stress tests

**Contract Tests**:
- Interface compliance tests
- Provider contract validation
- API contract tests

**Rationale**:
- Follows framework testing requirements
- Ensures quality and reliability
- Validates performance targets
- Maintains framework standards

---

## Summary

All NEEDS CLARIFICATION markers have been resolved based on:
1. BELUGA_AI_VOICE_AGENTS_PROPOSAL.md technical specifications
2. Existing Beluga AI framework patterns and packages
3. Industry best practices for voice AI systems
4. Go language best practices

The implementation can proceed with clear technical decisions for all aspects of the voice agents feature.

