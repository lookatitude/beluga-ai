# Implementation Plan: Speech-to-Speech (S2S) Model Support

**Branch**: `007-speech-s2s` | **Date**: 2026-01-04 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/007-speech-s2s/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✅ Loaded: /specs/007-speech-s2s/spec.md
   → ✅ Clarifications section verified (Session 2026-01-04 with 5 clarifications)
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → ✅ All clarifications resolved (integrated from user clarifications)
3. Fill the Constitution Check section based on the content of the constitution document.
   → ✅ All checks pass, following framework patterns
4. Evaluate Constitution Check section below
   → ✅ No violations, all patterns align with framework
   → ✅ Progress Tracking: Initial Constitution Check PASS
5. Execute Phase 0 → research.md
   → ⏳ To be generated: research.md with all technical decisions
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file
   → ⏳ To be generated: data-model.md, contracts/, quickstart.md
7. Re-evaluate Constitution Check section
   → ⏳ Post-Design Constitution Check (pending Phase 1 completion)
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
   → ✅ Described below
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

**Primary Requirement**: Enable users to have natural, real-time speech conversations with AI applications using Speech-to-Speech (S2S) models integrated into the Voice Agents framework. Users can choose between STT+TTS approach or S2S approach, with S2S providing end-to-end speech processing without explicit intermediate text steps.

**Technical Approach**: 
- S2S package integrated into Voice Agents framework (`pkg/voice/s2s/`)
- Provider abstraction following existing Voice Agents patterns (registry, factory, config, metrics, errors)
- Multi-provider support (Amazon Nova 2 Sonic, Grok Voice Agent, Gemini 2.5 Flash Native Audio, GPT Realtime)
- Session integration allowing S2S as alternative to STT+TTS pipeline
- Configurable reasoning (built-in provider reasoning vs external Beluga AI agents)
- Adaptive latency targets (aim for 200ms, allow up to 2 seconds)
- Comprehensive observability (OTEL metrics, tracing, logging)
- Silent retry with automatic recovery
- **User-facing clarifications integrated**: S2S integrated into Voice Agents, configurable reasoning, silent retry on errors, adaptive latency targets, configurable concurrent session limits per provider

## Technical Context

**Language/Version**: Go 1.24+  
**Primary Dependencies**: 
- Beluga AI framework packages (voice/session, agents, llms, memory, orchestration, config, monitoring)
- AWS SDK for Go (for Amazon Nova 2 Sonic/Bedrock)
- OpenAI Go SDK (github.com/sashabaranov/go-openai for GPT Realtime)
- Google Cloud Speech/Vertex AI SDK (for Gemini 2.5 Flash Native Audio)
- xAI SDK (for Grok Voice Agent, when available)
- Existing framework dependencies (OTEL, zap, testify)

**Storage**: N/A (ephemeral audio streams, conversation history via pkg/memory)  
**Testing**: Go testing package, testify, framework test patterns  
**Target Platform**: Linux server (primary), cross-platform Go support  
**Project Type**: Single (Go framework package - sub-package of Voice Agents)  
**Performance Goals**: 
- Adaptive latency: aim for 200ms (60% of interactions), allow up to 2 seconds (95% of interactions)
- Configurable concurrent sessions: minimum 50+ for basic providers, 100+ for advanced providers
- Scalable horizontally for higher throughput

**Constraints**: 
- Must follow Beluga AI framework package design patterns
- Must integrate with existing Voice Agents session package (no duplication)
- Must support multiple providers with fallbacks
- Must maintain adaptive latency targets (200ms goal, 2s maximum)
- **Error handling**: Silent retry with automatic recovery (user may notice brief pause, no explicit error unless recovery fails)
- **Reasoning mode**: Configurable per provider/conversation (built-in vs external agent)
- **Concurrent limits**: Configurable per provider (different providers have different capabilities)
- Must support both built-in provider reasoning and external Beluga AI agent integration

**Scale/Scope**: 
- 1 S2S package (`pkg/voice/s2s/`)
- 4+ S2S providers (Amazon Nova 2 Sonic, Grok Voice Agent, Gemini 2.5 Flash Native Audio, GPT Realtime)
- Integration with Voice Agents session package (extend existing)
- Integration with 4+ existing Beluga AI packages (agents, memory, orchestration, monitoring)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, registry.go)
- [x] Multi-provider package implements global registry pattern
- [x] All required files present (test_utils.go, advanced_test.go, README.md)

**Details**:
- S2S package (`pkg/voice/s2s/`) follows standard Beluga AI package layout
- Global registry pattern for provider management (consistent with llms, embeddings, voice/stt, voice/tts, etc.)
- All required files will be created per package design patterns:
  - `config.go`: S2SConfig struct with validation
  - `metrics.go`: OTEL metrics implementation
  - `errors.go`: S2SError with Op/Err/Code pattern
  - `registry.go`: Global provider registry
  - `test_utils.go`: AdvancedMockS2SProvider with options
  - `advanced_test.go`: Table-driven tests with concurrency/benchmarks
  - `README.md`: Complete package documentation

### Design Principles Compliance  
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method, noun for multi-method)
- [x] Dependencies injected via constructors (DIP compliance)
- [x] Single responsibility per package/struct (SRP compliance)
- [x] Functional options used for configuration (composition over inheritance)

**Details**:
- S2SProvider interface follows "Provider" noun pattern (consistent with STTProvider, TTSProvider)
- StreamingS2SProvider interface for streaming capabilities (segregation)
- All dependencies injected via constructors (no global state)
- Package has single responsibility: S2S processing (integrated into Voice Agents)
- Configuration uses functional options pattern (consistent with framework)
- Providers organized under `pkg/voice/s2s/providers/{provider_name}/`

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics)
- [x] Structured error handling with Op/Err/Code pattern
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks)
- [x] Integration testing for cross-package interactions

**Details**:
- All packages use OTEL metrics (via pkg/monitoring patterns)
- Error types follow Op/Err/Code pattern (consistent with framework)
- 100% test coverage requirement (test_utils.go, advanced_test.go)
- Integration tests for S2S + agents + memory + orchestration interactions
- Cross-package integration tests in `tests/integration/voice/s2s/`
- Benchmarks for latency, throughput, concurrent sessions

*Reference: Constitution v1.0.0 - See `docs/package_design_patterns.md`*

## Project Structure

### Documentation (this feature)
```
specs/007-speech-s2s/
├── plan.md              # This file (/speckit.plan command output) ✅
├── research.md          # Phase 0 output (/speckit.plan command) ⏳
├── data-model.md        # Phase 1 output (/speckit.plan command) ⏳
├── quickstart.md        # Phase 1 output (/speckit.plan command) ⏳
├── contracts/           # Phase 1 output (/speckit.plan command) ⏳
│   └── s2s-provider-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan) ✅
```

### Source Code (repository root)
```
pkg/voice/
├── iface/                    # Shared interfaces (existing)
│   └── ...                   # Existing interfaces
├── s2s/                      # NEW: S2S package
│   ├── iface/                # S2S-specific interfaces
│   │   ├── s2s.go            # S2SProvider interface
│   │   └── streaming.go      # StreamingS2SProvider interface
│   ├── internal/             # Private implementations
│   │   ├── types.go          # AudioInput, AudioOutput, ConversationContext
│   │   └── options.go        # STSOptions, functional options
│   ├── providers/            # Provider implementations
│   │   ├── amazon_nova/      # Amazon Nova 2 Sonic
│   │   │   ├── config.go
│   │   │   ├── provider.go
│   │   │   ├── streaming.go
│   │   │   └── provider_test.go
│   │   ├── grok/             # Grok Voice Agent
│   │   │   ├── config.go
│   │   │   ├── provider.go
│   │   │   ├── streaming.go
│   │   │   └── provider_test.go
│   │   ├── gemini/           # Gemini 2.5 Flash Native Audio
│   │   │   ├── config.go
│   │   │   ├── provider.go
│   │   │   ├── streaming.go
│   │   │   └── provider_test.go
│   │   └── openai_realtime/  # GPT Realtime
│   │       ├── config.go
│   │       ├── provider.go
│   │       ├── streaming.go
│   │       └── provider_test.go
│   ├── config.go             # S2SConfig, validation, options
│   ├── errors.go             # S2SError, error codes
│   ├── metrics.go            # OTEL metrics implementation
│   ├── registry.go           # Global provider registry
│   ├── s2s.go                # Factory function NewProvider
│   ├── fallback.go           # ProviderFallback, automatic fallback
│   ├── provider_manager.go   # ProviderManager for multiple providers
│   ├── reasoning.go          # ReasoningMode enum, configuration
│   ├── health.go             # Health checks
│   ├── test_utils.go         # AdvancedMockS2SProvider, testing utilities
│   ├── advanced_test.go      # Table-driven tests, benchmarks
│   └── README.md             # Package documentation
├── session/                  # EXTENDED: Session package (existing)
│   ├── types.go              # Add S2SProvider field to VoiceOptions
│   ├── options.go            # Add WithS2SProvider option
│   ├── session.go            # Update NewVoiceSession validation
│   └── internal/             # EXTENDED: Internal implementations
│       ├── s2s_integration.go        # NEW: S2SIntegration struct
│       ├── s2s_agent_integration.go  # NEW: S2S agent integration
│       ├── s2s_integration_test.go   # NEW: Integration tests
│       ├── s2s_agent_integration_test.go  # NEW: Agent integration tests
│       ├── s2s_memory_test.go        # NEW: Memory integration tests
│       └── s2s_orchestration_test.go # NEW: Orchestration integration tests
└── ...                       # Other voice sub-packages (existing)

examples/voice/
└── s2s/                      # NEW: S2S examples
    ├── basic_conversation.go
    ├── multi_provider.go
    ├── agent_integration.go
    └── config_example.yaml

tests/integration/voice/
└── s2s/                      # NEW: S2S integration tests
    ├── basic_conversation_test.go
    ├── multi_provider_test.go
    ├── agent_integration_test.go
    ├── memory_integration_test.go
    ├── orchestration_integration_test.go
    ├── cross_package_llms_test.go
    ├── cross_package_orchestration_test.go
    ├── observability_test.go
    └── end_to_end_test.go
```

**Structure Decision**: Single Go framework package structure. S2S is integrated as a sub-package of Voice Agents (`pkg/voice/s2s/`), following the same pattern as `pkg/voice/stt/` and `pkg/voice/tts/`. Session package is extended to support S2S providers alongside STT+TTS. This maintains consistency with existing Voice Agents architecture while adding S2S as a provider option.

## Phase 0: Research & Technical Decisions

### Research Tasks

1. **S2S Provider API Patterns**
   - Research Amazon Nova 2 Sonic API (AWS Bedrock/Nova APIs)
   - Research Grok Voice Agent API (xAI platform)
   - Research Gemini 2.5 Flash Native Audio API (Google Cloud/Vertex AI)
   - Research GPT Realtime API (OpenAI)
   - Document API patterns, authentication, streaming capabilities

2. **Provider SDK Integration**
   - Identify Go SDKs for each provider
   - Document SDK capabilities and limitations
   - Plan adapter patterns for unified interface

3. **Voice Agents Session Integration Pattern**
   - Analyze existing session package architecture
   - Design integration pattern for S2S alongside STT+TTS
   - Plan provider selection logic (either STT+TTS or S2S)

4. **Reasoning Mode Implementation**
   - Design pattern for built-in vs external reasoning
   - Plan agent integration when external reasoning selected
   - Document provider-specific reasoning capabilities

5. **Fallback and Resilience Patterns**
   - Design fallback strategy between S2S providers
   - Plan silent retry logic with automatic recovery
   - Design circuit breaker patterns

**Output**: `research.md` with all technical decisions documented

## Phase 1: Design & Contracts

### Data Model

Extract entities from spec:
- **SpeechConversation**: Session state, configuration, provider info, lifecycle
- **AudioInput**: Audio data, format, metadata (timestamp, language, quality), streaming info
- **AudioOutput**: Audio data, format, metadata (timestamp, provider, voice characteristics), streaming info
- **ConversationContext**: Conversation history, user preferences, agent state, memory references
- **S2SProviderConfiguration**: Provider selection, authentication, provider-specific options, fallback settings

**Output**: `data-model.md` with entity definitions, relationships, validation rules

### API Contracts

Generate contracts from functional requirements:
- S2SProvider interface contract (Process method, streaming)
- Session integration contract (S2S provider option)
- Agent integration contract (reasoning modes)

**Output**: `contracts/s2s-provider-api.md` with interface definitions

### Quickstart Guide

Generate quickstart scenarios based on user stories:
- Basic S2S conversation (User Story 1)
- Multi-provider configuration (User Story 2)
- Agent integration (User Story 3)
- Observability setup (User Story 4)

**Output**: `quickstart.md` with step-by-step examples

## Phase 2: Task Generation Approach

**Command**: `/speckit.tasks` will generate tasks.md

**Approach**:
1. Organize tasks by user story priority (P1, P2, P3)
2. Follow Beluga AI package design patterns:
   - Phase 1: Setup (directory structure, dependencies)
   - Phase 2: Foundational (interfaces, config, errors, metrics, registry, test utils)
   - Phase 3+: User story phases (US1, US2, US3, US4)
   - Final Phase: Polish (documentation, examples, integration tests)
3. Ensure all required files are covered:
   - config.go, metrics.go, errors.go, registry.go
   - test_utils.go, advanced_test.go
   - README.md
   - Provider implementations
   - Session integration
4. Include comprehensive testing:
   - Unit tests for all components
   - Integration tests for cross-package interactions
   - Benchmarks for performance
   - End-to-end tests for complete flows
5. Include documentation tasks:
   - Package README
   - Examples
   - Integration guides
   - Configuration documentation

**Expected Task Count**: ~100-110 tasks organized across phases

## Implementation Phases

### Phase 0: Research (plan.md scope)
- Generate research.md with technical decisions
- Document provider APIs and SDKs
- Design integration patterns

### Phase 1: Design (plan.md scope)
- Generate data-model.md
- Generate contracts/
- Generate quickstart.md
- Update agent context files

### Phase 2: Task Generation (/tasks command)
- Generate tasks.md organized by user story
- Map all requirements to tasks
- Define dependencies and parallel opportunities

### Phase 3: Implementation
- Execute tasks from tasks.md
- Follow TDD approach where applicable
- Implement providers one by one
- Integrate with session package

### Phase 4: Testing & Documentation
- Comprehensive testing (unit, integration, benchmarks)
- Complete documentation (README, examples, guides)
- Code quality checks (linters, coverage)
- Integration validation

## Complexity Tracking

> **No violations detected** - All patterns align with framework standards

## Progress Tracking

### Initial Constitution Check
- ✅ Package Structure Compliance: PASS
- ✅ Design Principles Compliance: PASS
- ✅ Observability & Quality Standards: PASS

### Post-Design Constitution Check
- ⏳ Pending Phase 1 completion (will re-check after data-model.md, contracts/, quickstart.md generated)

---

**Status**: Plan ready for Phase 0 and Phase 1 execution. All clarifications resolved, constitution checks passed, structure defined.
