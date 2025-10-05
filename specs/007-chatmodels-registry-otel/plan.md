
# Implementation Plan: ChatModels Global Registry & OTEL Integration

**Branch**: `007-chatmodels-registry-otel` | **Date**: October 5, 2025 | **Spec**: specs/007-chatmodels-registry-otel/spec.md
**Input**: Feature specification from `/specs/007-chatmodels-registry-otel/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Implement global registry and OpenTelemetry integration for the ChatModels package following reference implementations from pkg/config/registry.go ConfigProviderRegistry and pkg/core/di.go Container patterns. The implementation will provide thread-safe provider management with comprehensive observability, maintaining framework consistency while enabling seamless provider registration, discovery, and monitoring of chat model operations with <1ms registry resolution, <10ms provider creation, and <5% OTEL overhead.

## Technical Context
**Language/Version**: Go 1.21+ (following Beluga AI Framework standards)
**Primary Dependencies**: OpenTelemetry libraries, sync primitives, existing chatmodels package dependencies
**Storage**: In-memory registry with optional provider caching, following config package patterns
**Testing**: Go testing framework with table-driven tests, advanced mock utilities following pkg/core/di.go patterns
**Target Platform**: Linux/macOS/Windows server environments supporting Beluga AI Framework
**Project Type**: Multi-provider framework package extending chatmodels with registry and observability
**Performance Goals**: <1ms registry resolution, <10ms provider creation, <5% OTEL overhead (matching pkg/core/di.go performance)
**Constraints**: Must follow Beluga AI Framework constitution v1.0.0, thread-safe operations, no global state except registry singleton, maintain existing chatmodels API compatibility
**Scale/Scope**: Support 10+ chat model providers, 1000+ concurrent operations, production-grade observability for the 'chatmodels' package

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.) - extending existing chatmodels package structure
- [x] Multi-provider packages implement global registry pattern - following ConfigProviderRegistry from config package
- [x] All required files present (test_utils.go, advanced_test.go, README.md) - extending existing chatmodels testing infrastructure

### Design Principles Compliance
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method) - following core.iface patterns
- [x] Dependencies injected via constructors (DIP compliance) - using pkg/core/di.go Container pattern
- [x] Single responsibility per package/struct (SRP compliance) - registry, providers, metrics separated
- [x] Functional options used for configuration (composition over inheritance) - following pkg/core/di.go options pattern

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics) - following pkg/core/metrics.go pattern exactly
- [x] Structured error handling with Op/Err/Code pattern - following pkg/core/errors.go pattern
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks) - extending existing chatmodels testing
- [x] Integration testing for cross-package interactions - following framework testing standards

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*
*Status: PASS - Implementation follows established reference patterns from pkg/config/registry.go and pkg/core/di.go*

## Project Structure

### Documentation (this feature)
```
specs/007-chatmodels-registry-otel/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (extending existing chatmodels package)
```
pkg/chatmodels/
├── iface/                    # Interfaces (following framework ISP)
│   ├── chatmodel.go         # Existing: Core chat model interfaces
│   └── provider.go          # NEW: Provider interface (following config.iface.Provider)
├── internal/                 # Private implementations
│   ├── mock/               # Existing: Mock implementations for testing
│   └── registry/           # NEW: Internal registry implementation details
├── providers/               # Provider implementations
│   ├── openai/            # Existing: OpenAI provider implementation
│   └── registry.go         # NEW: Global registry facade (following pkg/config/registry.go)
├── config.go               # Existing: Enhanced with provider registry config
├── metrics.go              # NEW: OTEL metrics (following pkg/core/metrics.go)
├── errors.go               # Existing: Enhanced with registry error patterns
├── registry.go             # NEW: Public registry API (following config pattern)
├── di.go                   # NEW: Dependency injection (following pkg/core/di.go)
├── chatmodels.go           # Existing: Enhanced with registry integration
├── test_utils.go           # Existing: Enhanced with registry testing utilities
├── advanced_test.go        # Existing: Enhanced with registry integration tests
└── README.md               # Existing: Updated with registry documentation
```

**Structure Decision**: Extending existing chatmodels package structure while maintaining backward compatibility. New files added following Beluga AI Framework constitution v1.0.0 standards. Registry functionality added through new files (registry.go, di.go, iface/provider.go) that integrate with existing chatmodels API without breaking changes. Reference implementations: pkg/config/registry.go, pkg/core/di.go, pkg/core/metrics.go.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** - RESOLVED via reference implementations:
   - Registry pattern: Follow config.ConfigProviderRegistry with RegisterGlobal/NewRegistryProvider
   - OTEL integration: Follow pkg/core/di.go Container pattern with metrics/tracing/logging
   - Provider interface: Follow config.iface.Provider pattern
   - Error handling: Follow config.errors.go Op/Err/Code pattern
   - Performance targets: <1ms registry, <10ms creation, <5% OTEL overhead (matching core.di.go)

2. **Reference Implementation Analysis** (completed):
   ```
   Core Package Analysis (pkg/core/di.go, pkg/core/metrics.go):
     ✓ DI Container pattern with OTEL components and functional options
     ✓ Metrics collection for Runnable operations with standard OTEL instruments
     ✓ Tracing and logging interfaces with context propagation
     ✓ NoOpMetrics for testing scenarios with graceful degradation

   Config Package Analysis (pkg/config/registry.go, pkg/core/errors.go):
     ✓ Global registry with thread-safe operations using sync.RWMutex
     ✓ ProviderCreator functions and metadata management for capability enumeration
     ✓ ProviderOptions structure for configuration with validation
     ✓ Structured error handling with operation context and error codes
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: Follow reference implementations from config and core packages exactly
   - Rationale: Maintains framework consistency and leverages proven production patterns
   - Alternatives considered: Custom implementations (rejected for consistency and reduced risk)

**Output**: research.md documenting reference implementation analysis and pattern adoption decisions

## Phase 1: Design & Contracts
*Prerequisites: research.md complete (reference implementations analyzed)*

1. **Extract entities from feature spec** → `data-model.md`:
   - Global Registry: Following config.ConfigProviderRegistry structure with ProviderCreator functions
   - Provider Interface: Following config.iface.Provider pattern with CreateChatModel method
   - OTEL Components: Following pkg/core/di.go Container with Logger, TracerProvider interfaces
   - Provider Metadata: Following config.ProviderMetadata with capabilities and requirements
   - Provider Options: Following config.ProviderOptions with validation rules
   - Error Types: Following config.errors.go ChatModelError with Op/Err/Code pattern

2. **Generate API contracts** from functional requirements:
   - Registry API: RegisterGlobal, NewRegistryProvider, ListProviders following config patterns
   - Provider API: CreateChatModel, HealthCheck, GetModelInfo following iface.Provider
   - OTEL API: Metrics recording, tracing spans, structured logging following pkg/core/di.go
   - Output OpenAPI schema to `/contracts/` following framework contract standards

3. **Generate contract tests** from contracts:
   - Registry contract tests: Provider registration, discovery, creation following config test patterns
   - OTEL contract tests: Metrics collection, tracing propagation, logging correlation following pkg/core test patterns
   - Provider contract tests: Chat model operations, health checks following existing chatmodels tests
   - Tests must fail (no implementation yet) but follow framework testing standards

4. **Extract test scenarios** from user stories:
   - Registry management scenarios: Provider registration and discovery following config acceptance tests
   - Observability scenarios: Metrics collection and tracing following pkg/core integration tests
   - Provider integration scenarios: Chat model creation and operation extending existing chatmodels tests
   - Quickstart test = end-to-end registry and OTEL integration validation

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh cursor`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - Add OTEL integration patterns from pkg/core/di.go reference implementation
   - Add registry patterns from pkg/config/registry.go reference implementation
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs following reference implementation patterns
- Registry contracts → Registry implementation tasks [P] (following pkg/config/registry.go ConfigProviderRegistry pattern exactly)
- OTEL contracts → Observability implementation tasks [P] (following pkg/core/di.go Container pattern exactly)
- Provider contracts → Provider interface tasks [P] (following config.iface.Provider pattern exactly)
- Integration contracts → End-to-end integration tasks (registry + OTEL + existing chatmodels)
- Each failing contract test → Implementation task to make it pass following reference patterns

**Ordering Strategy**:
- Foundation first: Provider interface, Registry core, OTEL core [sequential - establish contracts]
- Implementation parallel: Registry features, OTEL features, Provider implementations [P - independent components]
- Integration last: End-to-end scenarios combining registry, OTEL, and existing chatmodels functionality
- TDD approach: Contract tests first, then implementation to make them pass following reference patterns
- Mark [P] for parallel execution (components that don't depend on each other)

**Estimated Output**: 20-25 numbered, ordered tasks in tasks.md focusing on:
- 6-8 Registry implementation tasks (following pkg/config/registry.go ConfigProviderRegistry pattern)
- 6-8 OTEL integration tasks (following pkg/core/di.go Container pattern)
- 4-6 Provider interface and enhancement tasks (following config.iface.Provider pattern)
- 4-6 Integration and testing tasks (extending existing chatmodels functionality)

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Post-Implementation Workflow (mandatory commit, push, PR, merge to develop process)
**Phase 6**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command) - Reference implementations analyzed
- [ ] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - approach described)
- [x] Phase 3: Tasks generated (/tasks command) - 35 tasks created with TDD approach
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS - Following reference patterns
- [ ] Post-Design Constitution Check: PENDING
- [x] All NEEDS CLARIFICATION resolved - Via reference implementation analysis and clarification sessions
- [x] Complexity deviations documented - None required (following established patterns)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
