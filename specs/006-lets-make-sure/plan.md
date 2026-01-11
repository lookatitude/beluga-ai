# Implementation Plan: Real-Time Voice Agent Support

**Branch**: `006-lets-make-sure` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-lets-make-sure/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → Found: /home/miguelp/Projects/lookatitude/beluga-ai/specs/006-lets-make-sure/spec.md
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → All clarifications resolved in Session 2025-01-27
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → All checks pass - extending existing packages
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → All research complete
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file
7. Re-evaluate Constitution Check section
   → All checks still pass
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 8. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
This feature extends the Beluga AI Framework to support real-time voice agent interactions with streaming language models. The implementation extends existing `pkg/agents` and `pkg/voice/session` packages to integrate agent instances with voice sessions, enabling ultra-low latency (< 500ms) voice conversations where agents can process streaming LLM responses, execute tools, and maintain conversation context. All extensions follow constitutional principles (ISP, DIP, SRP, composition) and maintain backward compatibility while achieving 90%+ test coverage with comprehensive mocks.

## Technical Context
**Language/Version**: Go 1.24.1 (toolchain go1.24.2)  
**Primary Dependencies**: 
- `go.opentelemetry.io/otel` (v1.38.0) - Metrics, tracing, logging
- `github.com/stretchr/testify` (v1.11.1) - Testing and mocking
- `github.com/gorilla/websocket` (v1.5.3) - WebSocket support
- `github.com/go-playground/validator/v10` (v10.28.0) - Configuration validation
- Existing LLM providers (OpenAI, Anthropic, AWS Bedrock, Ollama) with streaming support
- Existing voice packages (STT, TTS, VAD, Transport, Session)

**Storage**: N/A (in-memory conversation context, no persistent storage required)  
**Testing**: Go testing package (`go test`), testify/assert, testify/mock for mocks  
**Target Platform**: Linux/macOS/Windows (Go cross-platform)  
**Project Type**: single (Go framework/library project)  
**Performance Goals**: 
- End-to-end latency < 500ms (user speech → agent spoken response)
- Streaming response processing with incremental TTS conversion
- Support multiple concurrent voice calls with independent contexts
- Handle backpressure when audio processing cannot keep up

**Constraints**: 
- Must extend existing packages (no new packages unless absolutely necessary)
- Must maintain backward compatibility
- Must follow existing package design patterns (ISP, DIP, SRP, composition)
- Must achieve 90%+ test coverage for all new code
- Must pass all linters and tests
- Must use existing OTEL infrastructure for observability
- Must use existing error handling patterns (Op/Err/Code)

**Scale/Scope**: 
- Extend 2 main packages: `pkg/agents` and `pkg/voice/session`
- Support all existing LLM providers with streaming (OpenAI, Anthropic, AWS Bedrock, Ollama)
- Support all existing transport providers (WebSocket, WebRTC)
- Support all existing audio formats/codecs
- Multiple concurrent voice calls with independent agent instances

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.) - Extending existing packages that already comply
- [x] Multi-provider packages implement global registry pattern - Using existing registries
- [x] All required files present (test_utils.go, advanced_test.go, README.md) - Extending existing files

### Design Principles Compliance  
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method) - Extending existing interfaces with focused additions
- [x] Dependencies injected via constructors (DIP compliance) - Following existing patterns
- [x] Single responsibility per package/struct (SRP compliance) - Extensions maintain single responsibility
- [x] Functional options used for configuration (composition over inheritance) - Using existing functional options patterns

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics) - Using existing OTEL infrastructure
- [x] Structured error handling with Op/Err/Code pattern - Using existing error types
- [x] Comprehensive testing requirements (90%+ coverage, mocks, benchmarks) - All new code will have comprehensive tests
- [x] Integration testing for cross-package interactions - Will add integration tests for agent-voice-session interactions

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*

## Project Structure

### Documentation (this feature)
```
specs/006-lets-make-sure/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
├── agents/                    # Extending this package
│   ├── iface/
│   │   └── agent.go          # Add streaming interfaces
│   ├── internal/
│   │   ├── base/
│   │   │   └── agent.go      # Extend for streaming support
│   │   └── executor/
│   │       └── executor.go   # Add streaming executor
│   ├── config.go             # Extend config for voice integration
│   ├── metrics.go            # Add voice-specific metrics
│   ├── errors.go             # Add voice-specific error codes
│   └── test_utils.go         # Add voice session mocks
│
└── voice/
    └── session/              # Extending this package
        ├── iface/
        │   └── session.go    # Extend for agent instance support
        ├── internal/
        │   ├── agent_integration.go      # Enhance existing placeholder
        │   ├── streaming_agent.go       # Complete implementation
        │   └── voice_agent_executor.go   # New: agent executor integration
        ├── config.go         # Extend for agent configuration
        ├── metrics.go        # Add agent-specific metrics
        └── test_utils.go    # Add agent mocks

tests/
└── integration/
    └── agents_voice_test.go  # New: agent-voice integration tests
```

**Structure Decision**: Single Go framework project. Extending existing packages `pkg/agents` and `pkg/voice/session` with new interfaces, internal implementations, and comprehensive test coverage. No new packages required as functionality logically belongs to existing packages.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - All clarifications resolved in spec (latency, providers, protocols, formats, duration)
   - Research: Best practices for streaming agent execution with voice sessions
   - Research: Patterns for incremental TTS conversion from streaming LLM responses
   - Research: Interruption handling patterns for streaming responses
   - Research: Backpressure handling in audio processing pipelines

2. **Generate and dispatch research agents**:
   - Task: "Research streaming agent execution patterns with voice session integration"
   - Task: "Find best practices for incremental TTS conversion from streaming LLM chunks"
   - Task: "Research interruption handling for streaming agent responses in voice calls"
   - Task: "Find patterns for backpressure handling in real-time audio processing"
   - Task: "Research conversation context management across streaming interactions"

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all research findings

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Extended Agent Interface (streaming capabilities)
   - Voice Session Agent Integration (agent instance management)
   - Streaming Agent Executor (streaming response handling)
   - Voice Call Agent Context (conversation state, history, tool results)

2. **Generate API contracts** from functional requirements:
   - Agent streaming interface contracts
   - Voice session agent integration contracts
   - Streaming executor contracts
   - Configuration contracts

3. **Generate contract tests** from contracts:
   - Interface compliance tests
   - Integration contract tests
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - 6 acceptance scenarios → integration test scenarios
   - Quickstart test = primary user story validation steps

5. **Update agent file incrementally**:
   - Run `.specify/scripts/bash/update-agent-context.sh cursor`

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Break down into small, focused tasks with detailed goals
- Each interface extension → interface definition task [P]
- Each internal implementation → implementation task with tests
- Each integration point → integration test task
- Configuration extensions → config task [P]
- Metrics extensions → metrics task [P]
- Error handling extensions → error codes task [P]
- Mock creation → mock task [P]
- Test coverage → comprehensive test tasks for each component

**Ordering Strategy**:
- TDD order: Tests before implementation
- Dependency order: Interfaces → Config → Errors → Metrics → Implementation → Integration
- Mark [P] for parallel execution (independent files)
- Group related tasks (e.g., all pkg/agents extensions together, then pkg/voice/session)

**Task Breakdown Details**:
- **Interface Tasks**: Define streaming interfaces, agent integration interfaces (with tests)
- **Config Tasks**: Extend config structs, add validation, add functional options (with tests)
- **Error Tasks**: Add error codes, error constructors, error tests
- **Metrics Tasks**: Add metrics definitions, metric recording, metric tests
- **Implementation Tasks**: Streaming executor, agent integration, voice session extensions (with unit tests)
- **Mock Tasks**: Create mocks for all external dependencies (transport, LLM, TTS, STT)
- **Integration Tasks**: Cross-package integration tests, end-to-end scenarios
- **Documentation Tasks**: Update README files, add examples
- **Coverage Tasks**: Ensure 90%+ coverage for all new code, add missing tests

**Estimated Output**: 60-80+ numbered, ordered tasks in tasks.md (detailed breakdown for 90%+ coverage)

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation, coverage verification)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No violations - all extensions follow existing patterns and constitutional requirements.

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command) - tasks.md created
- [x] Phase 4: Implementation complete - All tasks implemented
- [x] Phase 5: Validation passed - Tests passing, validation complete

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
