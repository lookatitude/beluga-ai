# Implementation Plan: Voice Agents

**Branch**: `004-feature-voice-agents` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/004-feature-voice-agents/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✅ Loaded: /specs/004-feature-voice-agents/spec.md
   → ✅ Clarifications section verified (Session 2025-01-27 with 5 clarifications)
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → ✅ All clarifications resolved (integrated from research.md and user clarifications)
3. Fill the Constitution Check section based on the content of the constitution document.
   → ✅ All checks pass, following framework patterns
4. Evaluate Constitution Check section below
   → ✅ No violations, all patterns align with framework
   → ✅ Progress Tracking: Initial Constitution Check PASS
5. Execute Phase 0 → research.md
   → ✅ Already exists: research.md with all technical decisions
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file
   → ✅ Already exists: data-model.md, contracts/voice-session-api.md, quickstart.md
7. Re-evaluate Constitution Check section
   → ✅ No new violations, design aligns with framework
   → ✅ Progress Tracking: Post-Design Constitution Check PASS
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
   → ✅ Described below
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

**Primary Requirement**: Enable users to have natural voice conversations with Beluga AI agents through real-time speech-to-text, agent processing, and text-to-speech conversion.

**Technical Approach**: 
- Modular voice framework with provider abstraction (STT, TTS, VAD, Turn Detection, Transport, Session Management)
- Integration with existing Beluga AI packages (agents, memory, llms, config, prompts, monitoring)
- Multi-provider support with automatic fallbacks
- Real-time streaming with sub-200ms latency target
- Production-ready features (observability, error handling, circuit breakers)
- **User-facing clarifications integrated**: Silent retry on errors, automatic session timeout, configurable interruption thresholds, configurable preemptive generation, configurable long utterance handling

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- Beluga AI framework packages (agents, llms, memory, config, prompts, monitoring)
- pion/webrtc (WebRTC implementation)
- ONNX runtime (for Silero VAD models)
- Existing framework dependencies (OTEL, zap, testify)

**Storage**: N/A (ephemeral audio streams, conversation history via pkg/memory)  
**Testing**: Go testing package, testify, framework test patterns  
**Target Platform**: Linux server (primary), cross-platform Go support  
**Project Type**: Single (Go framework package)  
**Performance Goals**: 
- Sub-200ms latency for voice interactions
- 100+ concurrent voice sessions per instance
- 1000+ audio chunks per second processing

**Constraints**: 
- Must follow Beluga AI framework package design patterns
- Must integrate with existing packages (no duplication)
- Must support multiple providers with fallbacks
- Must maintain <200ms latency for real-time feel
- **Error handling**: Silent retry with automatic recovery (user may notice brief pause)
- **Session timeout**: Automatic end after inactivity timeout (e.g., 30 seconds)
- **Interruptions**: Configurable threshold (e.g., stop if interruption >2 words)
- **Preemptive generation**: Configurable behavior for interim vs final transcript differences
- **Long utterances**: Configurable chunk size and processing strategy

**Scale/Scope**: 
- 7 core packages (stt, tts, vad, turndetection, transport, session, noise)
- 4+ STT providers (Deepgram, Google, Azure, OpenAI)
- 4+ TTS providers (OpenAI, Google, Azure, ElevenLabs)
- 3+ VAD providers (Silero, Energy, WebRTC)
- Integration with 6+ existing Beluga AI packages

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.)
- [x] Multi-provider packages implement global registry pattern
- [x] All required files present (test_utils.go, advanced_test.go, README.md)

**Details**:
- Each voice sub-package (stt, tts, vad, etc.) follows standard layout
- Global registry pattern for provider management (consistent with llms, embeddings, etc.)
- All required files will be created per package

### Design Principles Compliance  
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method)
- [x] Dependencies injected via constructors (DIP compliance)
- [x] Single responsibility per package/struct (SRP compliance)
- [x] Functional options used for configuration (composition over inheritance)

**Details**:
- STTProvider, TTSProvider, VADProvider follow "er" suffix pattern
- All dependencies injected via constructors (no global state)
- Each package has single responsibility (stt=speech-to-text, tts=text-to-speech, etc.)
- Configuration uses functional options pattern (consistent with framework)

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics)
- [x] Structured error handling with Op/Err/Code pattern
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks)
- [x] Integration testing for cross-package interactions

**Details**:
- All packages use OTEL metrics (via pkg/monitoring patterns)
- Error types follow Op/Err/Code pattern (consistent with framework)
- 100% test coverage requirement (test_utils.go, advanced_test.go)
- Integration tests for voice + agents + memory interactions

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*

## Project Structure

### Documentation (this feature)
```
specs/004-feature-voice-agents/
├── plan.md              # This file (/plan command output) ✅
├── research.md          # Phase 0 output (/plan command) ✅
├── data-model.md        # Phase 1 output (/plan command) ✅
├── quickstart.md         # Phase 1 output (/plan command) ✅
├── contracts/           # Phase 1 output (/plan command) ✅
│   └── voice-session-api.md
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/voice/
├── iface/                    # Core interfaces
│   ├── stt.go                # STTProvider interface
│   ├── tts.go                # TTSProvider interface
│   ├── vad.go                # VADProvider interface
│   ├── turndetection.go      # TurnDetector interface
│   ├── transport.go          # Transport interface
│   └── session.go            # VoiceSession interface
├── internal/                 # Private implementations
│   ├── audio/                # Audio format utilities
│   └── utils/                # Shared utilities
├── providers/                # Provider implementations
│   ├── stt/                  # STT providers
│   │   ├── deepgram/
│   │   ├── google/
│   │   ├── azure/
│   │   └── openai/
│   ├── tts/                  # TTS providers
│   │   ├── openai/
│   │   ├── google/
│   │   ├── azure/
│   │   └── elevenlabs/
│   ├── vad/                  # VAD providers
│   │   ├── silero/
│   │   ├── energy/
│   │   └── webrtc/
│   └── turndetection/        # Turn detection providers
│       ├── onnx/
│       └── heuristic/
├── stt/                      # STT package (main)
│   ├── iface/
│   ├── internal/
│   ├── providers/
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── registry.go
│   ├── stt.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
├── tts/                      # TTS package (main)
│   ├── iface/
│   ├── internal/
│   ├── providers/
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── registry.go
│   ├── tts.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
├── vad/                      # VAD package (main)
│   ├── iface/
│   ├── internal/
│   ├── providers/
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── registry.go
│   ├── vad.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
├── turndetection/            # Turn detection package
│   ├── iface/
│   ├── internal/
│   ├── providers/
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── registry.go
│   ├── turndetection.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
├── transport/                # Transport package
│   ├── iface/
│   ├── internal/
│   ├── webrtc/               # WebRTC transport
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── transport.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
├── session/                  # Session management package
│   ├── iface/
│   ├── internal/
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── session.go
│   ├── test_utils.go
│   ├── advanced_test.go
│   └── README.md
└── noise/                    # Noise cancellation package
    ├── iface/
    ├── internal/
    ├── providers/
    ├── config.go
    ├── metrics.go
    ├── errors.go
    ├── registry.go
    ├── noise.go
    ├── test_utils.go
    ├── advanced_test.go
    └── README.md
```

**Structure Decision**: Single Go framework package with modular sub-packages. Each sub-package (stt, tts, vad, etc.) is independently usable and follows the standard Beluga AI package structure. Provider implementations are organized under providers/ subdirectories. This structure enables:
- Independent use of each component (e.g., use STT without TTS)
- Clear separation of concerns (SRP compliance)
- Easy extension with new providers
- Consistent with existing framework packages (llms, embeddings, etc.)

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - ✅ Latency targets: Resolved (sub-200ms from proposal)
   - ✅ Authentication: Resolved (context-based, defer to application)
   - ✅ Data retention: Resolved (via pkg/memory integration)
   - ✅ Performance targets: Resolved (100+ concurrent sessions)
   - ✅ Security: Resolved (TLS/DTLS standard protocols)
   - ✅ Language support: Resolved (multi-language via providers)
   - ✅ Error handling: Resolved (silent retry with automatic recovery)
   - ✅ Session timeout: Resolved (automatic end after inactivity)
   - ✅ Interruptions: Resolved (configurable threshold)
   - ✅ Preemptive generation: Resolved (configurable behavior)
   - ✅ Long utterances: Resolved (configurable chunk size and strategy)

2. **Generate and dispatch research agents**:
   - ✅ Researched: Performance targets and latency requirements
   - ✅ Researched: Integration patterns with existing packages
   - ✅ Researched: Provider abstraction patterns
   - ✅ Researched: Error handling and observability patterns
   - ✅ Researched: Testing strategies and requirements
   - ✅ Researched: User-facing behavior clarifications

3. **Consolidate findings** in `research.md` using format:
   - ✅ Decision: [what was chosen]
   - ✅ Rationale: [why chosen]
   - ✅ Alternatives considered: [what else evaluated]

**Output**: ✅ research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete ✅*

1. **Extract entities from feature spec** → `data-model.md`:
   - ✅ VoiceSession: Session lifecycle, state machine, relationships
   - ✅ AudioStream: Input/output streams, format validation
   - ✅ Transcript: STT results, word timestamps, confidence
   - ✅ AgentResponse: Agent output, streaming support
   - ✅ VoiceSessionConfig: Configuration structure
   - ✅ VoiceOptions: Behavior configuration (including new clarifications)
   - ✅ AudioFormat: Format specifications
   - ✅ Validation rules and state transitions defined

2. **Generate API contracts** from functional requirements:
   - ✅ VoiceSession interface contract (contracts/voice-session-api.md)
   - ✅ Method signatures, preconditions, postconditions
   - ✅ Error codes and error handling
   - ✅ Observability requirements
   - ✅ Performance contracts

3. **Generate contract tests** from contracts:
   - ⏳ To be created in Phase 2 (/tasks command)
   - Will include: Unit tests for all interface methods
   - Will include: State machine transition tests
   - Will include: Error condition tests

4. **Extract test scenarios** from user stories:
   - ✅ Quickstart guide includes validation steps
   - ✅ Integration test scenarios defined in quickstart.md
   - ⏳ Detailed integration tests to be created in Phase 2

5. **Update agent file incrementally** (O(1) operation):
   - ✅ Already executed: `.specify/scripts/bash/update-agent-context.sh cursor`
   - ✅ Added: Voice agents framework context
   - ✅ Preserved: Existing manual additions

**Output**: ✅ data-model.md, ✅ contracts/voice-session-api.md, ⏳ contract tests (Phase 2), ✅ quickstart.md, ✅ agent file (updated)

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs:
  - **From data-model.md**: Entity creation tasks for each core entity
  - **From contracts/**: Interface implementation tasks, contract test tasks
  - **From quickstart.md**: Integration test tasks, example implementation tasks
  - **From research.md**: Provider implementation tasks, integration tasks
  - **From clarifications**: Configuration tasks for error handling, timeout, interruptions, preemptive generation, long utterances
- Each contract method → contract test task [P]
- Each entity → model/struct creation task [P] 
- Each user story → integration test task
- Implementation tasks to make tests pass (TDD approach)

**Ordering Strategy**:
- **Foundation First**: Core interfaces and types (iface/, config.go, errors.go)
- **Providers Next**: STT, TTS, VAD providers (can be parallel [P])
- **Integration**: Session management, transport
- **Testing**: Contract tests, integration tests, benchmarks
- **Documentation**: README.md, examples
- TDD order: Tests before implementation 
- Dependency order: Interfaces → Providers → Session → Integration
- Mark [P] for parallel execution (independent files/packages)

**Task Categories**:
1. **Core Infrastructure** (5-7 tasks):
   - Interface definitions (iface/)
   - Configuration structs (config.go) - including clarification-based configs
   - Error types (errors.go)
   - Metrics setup (metrics.go)
   - Registry pattern (registry.go)

2. **STT Package** (8-10 tasks):
   - STT interface and base
   - Deepgram provider
   - Google provider
   - Azure provider
   - OpenAI provider
   - Tests and benchmarks

3. **TTS Package** (8-10 tasks):
   - TTS interface and base
   - OpenAI provider
   - Google provider
   - Azure provider
   - ElevenLabs provider
   - Tests and benchmarks

4. **VAD Package** (6-8 tasks):
   - VAD interface and base
   - Silero provider
   - Energy provider
   - WebRTC provider
   - Tests and benchmarks

5. **Turn Detection Package** (4-6 tasks):
   - TurnDetector interface
   - ONNX provider
   - Heuristic provider
   - Tests

6. **Transport Package** (6-8 tasks):
   - Transport interface
   - WebRTC implementation
   - Connection management
   - Tests

7. **Session Package** (10-12 tasks):
   - VoiceSession interface
   - Session lifecycle
   - State management
   - Integration with providers
   - **Error handling implementation** (silent retry)
   - **Timeout handling** (automatic session end)
   - **Interruption handling** (configurable threshold)
   - **Preemptive generation** (configurable behavior)
   - **Long utterance handling** (configurable chunking)
   - Tests

8. **Noise Cancellation Package** (4-6 tasks):
   - NoiseCancellation interface
   - Spectral subtraction
   - RNNoise integration
   - Tests

9. **Integration** (6-8 tasks):
   - Integration with pkg/agents
   - Integration with pkg/memory
   - Integration with pkg/config
   - End-to-end tests
   - Quickstart example

10. **Documentation & Polish** (4-6 tasks):
    - README.md for each package
    - API documentation
    - Usage examples
    - Performance tuning guide

**Estimated Output**: 65-85 numbered, ordered tasks in tasks.md (increased due to clarification-based features)

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No violations - all design decisions align with Beluga AI framework constitution.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command) ✅
- [x] Phase 1: Design complete (/plan command) ✅
- [x] Phase 2: Task planning complete (/plan command - describe approach only) ✅
- [x] Phase 3: Tasks generated (/tasks command) ✅ - tasks.md created
- [x] Phase 4: Implementation complete ✅ - All tasks implemented
- [x] Phase 5: Validation passed ✅ - Tests passing, validation complete

**Gate Status**:
- [x] Initial Constitution Check: PASS ✅
- [x] Post-Design Constitution Check: PASS ✅
- [x] All NEEDS CLARIFICATION resolved ✅
- [x] Complexity deviations documented ✅ (none)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
