# Tasks: Multimodal Models Support

**Input**: Design documents from `/specs/009-multimodal/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Framework requires test_utils.go and advanced_test.go following framework patterns. Integration tests are included for cross-package compatibility.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Package**: `pkg/multimodal/` at repository root
- **Providers**: `pkg/multimodal/providers/{provider}/`
- **Integration tests**: `tests/integration/multimodal/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create package directory structure pkg/multimodal/ following framework v2 standards
- [X] T002 [P] Create iface/ directory with placeholder files pkg/multimodal/iface/model.go, pkg/multimodal/iface/provider.go, pkg/multimodal/iface/content.go
- [X] T003 [P] Create internal/ directory structure pkg/multimodal/internal/ with mock/ subdirectory
- [X] T004 [P] Create providers/ directory structure pkg/multimodal/providers/ for provider implementations
- [X] T005 Create integration test directory tests/integration/multimodal/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 [P] Define MultimodalModel interface in pkg/multimodal/iface/model.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [X] T007 [P] Define MultimodalProvider interface in pkg/multimodal/iface/provider.go with CreateModel, GetName, GetCapabilities, ValidateConfig methods
- [X] T008 [P] Define ContentBlock interface in pkg/multimodal/iface/content.go with GetType, GetData, GetURL, GetFilePath, GetMIMEType, GetSize, GetMetadata methods
- [X] T009 [P] Define Config struct in pkg/multimodal/config.go with mapstructure, yaml, env, validate tags
- [X] T010 [P] Define ModalityCapabilities struct in pkg/multimodal/config.go with Text, Image, Audio, Video support flags and size limits
- [X] T011 [P] Define RoutingConfig struct in pkg/multimodal/config.go with Strategy, provider fields, FallbackToText flag
- [X] T012 [P] Define MultimodalInput struct in pkg/multimodal/multimodal.go with ContentBlocks, Metadata, Format, Routing fields
- [X] T013 [P] Define MultimodalOutput struct in pkg/multimodal/multimodal.go with InputID, ContentBlocks, Metadata, Confidence, Provider, Model fields
- [X] T014 [P] Define MultimodalError type in pkg/multimodal/errors.go with Op, Err, Code, Message fields following framework patterns
- [X] T015 [P] Define error codes in pkg/multimodal/errors.go (ErrCodeProviderNotFound, ErrCodeInvalidConfig, ErrCodeInvalidInput, ErrCodeInvalidFormat, ErrCodeProviderError, ErrCodeUnsupportedModality, ErrCodeTimeout, ErrCodeCancelled, ErrCodeFileNotFound)
- [X] T016 [P] Implement error helper functions in pkg/multimodal/errors.go (NewMultimodalError, WrapError, IsMultimodalError, AsMultimodalError)
- [X] T017 [P] Implement global registry in pkg/multimodal/registry.go with GetRegistry, Register, Create, ListProviders, IsRegistered methods following framework patterns
- [X] T017a [P] Validate registry implementation matches existing framework patterns (compare to pkg/llms/registry.go and pkg/embeddings/registry/registry.go) in pkg/multimodal/registry.go
- [X] T018 [P] Define OTEL metrics in pkg/multimodal/metrics.go with NewMetrics, GetMetrics, RecordProcess, RecordProcessStream, RecordCapabilityCheck methods
- [X] T018a [P] Set up foundational OTEL tracing infrastructure in pkg/multimodal/metrics.go (tracer initialization, span context helpers, structured logging helpers)
- [X] T018b [P] Implement logWithOTELContext helper function in pkg/multimodal/metrics.go for structured logging with OTEL trace/span IDs following framework patterns
- [X] T019 [P] Implement Config validation in pkg/multimodal/config.go with Validate method using validator library
- [X] T020 [P] Implement functional options in pkg/multimodal/config.go (WithProvider, WithModel, WithAPIKey, WithTimeout, WithStreaming, etc.)
- [X] T021 [P] Create test_utils.go in pkg/multimodal/test_utils.go with mock implementations (MockMultimodalModel, MockMultimodalProvider, MockContentBlock) following framework patterns
- [X] T022 Create README.md in pkg/multimodal/README.md with package overview, usage examples, and provider registration guide
- [X] T022a [P] Verify framework package design pattern compliance checklist in Phase 2 (iface/ exists, config.go exists, metrics.go exists, errors.go exists, test_utils.go exists, advanced_test.go will be created) - document any deviations

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Multimodal Input Processing (Priority: P1) 🎯 MVP

**Goal**: Process multimodal inputs (text+image, text+audio, text+video) through a unified interface that routes content to appropriate providers

**Independent Test**: Create multimodal messages (text+image), pass them through the multimodal interface, verify content is correctly routed to appropriate providers (LLM for reasoning, embeddings for vectors). System handles mixed content types and falls back gracefully when providers don't support specific modalities.

### Implementation for User Story 1

- [X] T023 [US1] Implement ContentBlock struct in pkg/multimodal/multimodal.go with Type, Data, URL, FilePath, MIMEType, Format, Size, Metadata fields
- [X] T024 [US1] Implement NewContentBlock factory function in pkg/multimodal/multimodal.go for creating content blocks from data
- [X] T025 [US1] Implement NewContentBlockFromURL factory function in pkg/multimodal/multimodal.go for creating content blocks from URLs
- [X] T026 [US1] Implement NewContentBlockFromFile factory function in pkg/multimodal/multimodal.go for creating content blocks from file paths
- [X] T027 [US1] Implement content block validation in pkg/multimodal/multimodal.go (validate type, data source, MIME type, size)
- [X] T028 [US1] Implement NewMultimodalInput factory function in pkg/multimodal/multimodal.go with functional options (WithRouting, WithMetadata, WithFormat)
- [X] T029 [US1] Implement input validation in pkg/multimodal/multimodal.go (validate content blocks, format, routing)
- [X] T030 [US1] Implement format normalizer in pkg/multimodal/internal/normalizer.go to convert between base64, URLs, and file paths
- [X] T031 [US1] Implement content router in pkg/multimodal/internal/router.go to route content blocks to appropriate providers based on capabilities
- [X] T032 [US1] Implement capability detection in pkg/multimodal/internal/router.go to check provider support for modalities
- [X] T033 [US1] Implement fallback logic in pkg/multimodal/internal/router.go for graceful fallback to text-only when modality not supported
- [X] T034 [US1] Implement BaseMultimodalModel in pkg/multimodal/internal/model.go with Process method that uses router for content routing (routing logic only, output generation handled in T043)
- [X] T035 [US1] Integrate OTEL tracing in pkg/multimodal/internal/model.go for Process method with span attributes
- [X] T036 [US1] Integrate OTEL metrics in pkg/multimodal/internal/model.go for Process method recording latency and success
- [X] T037 [US1] Implement structured logging in pkg/multimodal/internal/model.go with OTEL context (trace IDs, span IDs)
- [X] T038 [US1] Implement NewMultimodalModel factory function in pkg/multimodal/factory.go that creates model instances via registry
- [X] T039 [US1] Add unit tests in pkg/multimodal/multimodal_test.go for content block creation and validation
- [X] T040 [US1] Add unit tests in pkg/multimodal/multimodal_test.go for multimodal input creation and validation
- [X] T041 [US1] Add advanced tests in pkg/multimodal/advanced_test.go for content routing scenarios (text+image, text+audio, text+video, fallback cases)

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently - users can create multimodal inputs and route them to providers

---

## Phase 4: User Story 2 - Multimodal Reasoning & Generation (Priority: P1)

**Goal**: Perform multimodal reasoning (visual question answering, image captioning) and generation (text-to-image, speech-to-text chains) through agents

**Independent Test**: Create an agent with multimodal capabilities, provide it with an image and a question, verify the agent can reason about the image and generate appropriate responses. System supports both input reasoning (understanding multimodal content) and output generation (creating multimodal content).

### Implementation for User Story 2

- [X] T042 [US2] Implement MultimodalOutput struct validation in pkg/multimodal/multimodal.go (validate content blocks, confidence scores)
- [X] T043 [US2] Implement output generation logic in pkg/multimodal/internal/model.go for Process method to generate MultimodalOutput (extends T034's routing with actual output generation)
- [X] T044 [US2] Implement reasoning pipeline in pkg/multimodal/internal/model.go that processes multimodal inputs and generates reasoning outputs
- [X] T045 [US2] Implement generation pipeline in pkg/multimodal/internal/model.go that generates multimodal outputs from text instructions
- [X] T046 [US2] Integrate with existing LLM providers in pkg/multimodal/internal/model.go for text processing in multimodal workflows
- [X] T047 [US2] Integrate with existing embedding providers in pkg/multimodal/internal/model.go for multimodal vector generation
- [X] T048 [US2] Integrate with existing voice providers in pkg/multimodal/internal/model.go for audio processing (placeholder - voice integration structure ready)
- [X] T049 [US2] Implement multimodal chain support in pkg/multimodal/internal/model.go for chaining operations (image → text → image)
- [X] T050 [US2] Implement output formatting in pkg/multimodal/internal/model.go to ensure outputs are properly formatted for subsequent operations
- [X] T051 [US2] Add OTEL tracing for reasoning operations in pkg/multimodal/internal/model.go with reasoning-specific attributes
- [X] T052 [US2] Add OTEL metrics for generation operations in pkg/multimodal/metrics.go with generation-specific counters and histograms
- [X] T053 [US2] Add unit tests in pkg/multimodal/multimodal_test.go for output generation and validation
- [X] T054 [US2] Add advanced tests in pkg/multimodal/advanced_test.go for reasoning scenarios (visual QA, image captioning)
- [X] T055 [US2] Add advanced tests in pkg/multimodal/advanced_test.go for generation scenarios (text-to-image, speech-to-text chains)
- [X] T056 [US2] Add advanced tests in pkg/multimodal/advanced_test.go for multimodal chain scenarios

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - users can process multimodal inputs and perform reasoning/generation

---

## Phase 5: User Story 3 - Multimodal RAG Integration (Priority: P2)

**Goal**: Perform retrieval-augmented generation with multimodal data (retrieve/embed images/videos via retrievers/vectorstores, fuse with text for agent decisions)

**Independent Test**: Store multimodal documents (images with text) in a vector store, perform multimodal search queries, verify retrieved multimodal content is properly fused with text for agent reasoning. System supports multimodal embeddings, multimodal vector storage, and multimodal retrieval.

### Implementation for User Story 3

- [X] T057 [US3] Implement multimodal embedding integration in pkg/multimodal/internal/model.go to use MultimodalEmbedder interface from embeddings package
- [X] T058 [US3] Implement document embedding generation in pkg/multimodal/internal/model.go for multimodal documents (text+images)
- [X] T059 [US3] Implement query embedding generation in pkg/multimodal/internal/model.go for multimodal queries (text+image)
- [X] T060 [US3] Integrate with vectorstores package in pkg/multimodal/internal/model.go to store multimodal vectors
- [X] T061 [US3] Implement multimodal retrieval in pkg/multimodal/internal/model.go to search with multimodal queries
- [X] T062 [US3] Implement content fusion logic in pkg/multimodal/internal/model.go to fuse multimodal and text content for agent reasoning
- [X] T063 [US3] Implement context preservation in pkg/multimodal/internal/model.go to maintain context across text and multimodal modalities in RAG workflows
- [X] T064 [US3] Add integration tests in tests/integration/multimodal/rag_test.go for storing multimodal documents
- [X] T065 [US3] Add integration tests in tests/integration/multimodal/rag_test.go for multimodal search queries
- [X] T066 [US3] Add integration tests in tests/integration/multimodal/rag_test.go for content fusion and context preservation
- [X] T067 [US3] Add advanced tests in pkg/multimodal/advanced_test.go for multimodal RAG scenarios with various content types

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently - users can perform multimodal RAG workflows

---

## Phase 6: User Story 4 - Real-Time Multimodal Streaming (Priority: P2)

**Goal**: Process multimodal data in real-time (live video analysis in agents, streaming audio/video processing) with low latency for interactive workflows

**Independent Test**: Set up a streaming multimodal workflow (live video feed), verify chunks are processed as they arrive, confirm latency is acceptable for interactive use. System supports streaming inputs and outputs for multimodal data.

### Implementation for User Story 4

- [X] T068 [US4] Implement ProcessStream method in pkg/multimodal/internal/model.go with streaming support for video/audio inputs
- [X] T069 [US4] Implement chunking logic in pkg/multimodal/internal/model.go for video (1MB chunks) and audio (64KB chunks)
- [X] T070 [US4] Implement incremental result streaming in pkg/multimodal/internal/model.go that sends results as chunks arrive
- [X] T071 [US4] Implement streaming output channel in pkg/multimodal/internal/model.go with proper error handling and context cancellation
- [X] T072 [US4] Implement interruption handling in pkg/multimodal/internal/model.go for context switching when new input arrives
- [X] T073 [US4] Implement state management in pkg/multimodal/internal/model.go for streaming operations
- [X] T074 [US4] Add latency monitoring in pkg/multimodal/metrics.go for streaming operations (voice <500ms, video <1s)
- [X] T075 [US4] Add OTEL tracing for streaming operations in pkg/multimodal/internal/model.go with streaming-specific attributes
- [X] T076 [US4] Add integration tests in tests/integration/multimodal/streaming_test.go for streaming video input
- [X] T077 [US4] Add integration tests in tests/integration/multimodal/streaming_test.go for streaming audio input
- [X] T078 [US4] Add integration tests in tests/integration/multimodal/streaming_test.go for streaming output and incremental results
- [X] T079 [US4] Add integration tests in tests/integration/multimodal/streaming_test.go for interruption and context switching
- [X] T080 [US4] Add benchmarks in pkg/multimodal/advanced_test.go for streaming latency (voice, video)

**Checkpoint**: At this point, User Stories 1, 2, 3, AND 4 should all work independently - users can process multimodal data in real-time with streaming

---

## Phase 7: User Story 5 - Multimodal Agent Extensions (Priority: P3)

**Goal**: Extend agents with multimodal capabilities (voice-enabled ReAct loops, handling image inputs in orchestration graphs)

**Independent Test**: Create a multimodal agent (ReAct agent that processes images), provide multimodal inputs, verify the agent can reason, plan, and execute with multimodal data. System integrates multimodal capabilities into existing agent patterns.

### Implementation for User Story 5

- [X] T081 [US5] Integrate with agents package in pkg/multimodal/internal/agent_integration.go to enable multimodal agent capabilities
- [X] T082 [US5] Implement multimodal message handling in pkg/multimodal/internal/agent_integration.go for ImageMessage, VideoMessage, VoiceDocument from schema package - add explicit integration test in tests/integration/multimodal/schema_integration_test.go to verify compatibility with schema package types
- [X] T083 [US5] Implement ReAct loop integration in pkg/multimodal/internal/agent_integration.go for voice-enabled ReAct loops
- [X] T084 [US5] Implement orchestration graph integration in pkg/multimodal/internal/agent_integration.go for handling image inputs in orchestration graphs
- [X] T085 [US5] Implement tool integration in pkg/multimodal/internal/agent_integration.go for tools that require multimodal inputs/outputs
- [X] T086 [US5] Implement agent-to-agent communication in pkg/multimodal/internal/agent_integration.go to preserve multimodal data
- [X] T087 [US5] Implement context preservation in pkg/multimodal/internal/agent_integration.go to maintain multimodal context throughout agent workflows
- [X] T088 [US5] Add integration tests in tests/integration/multimodal/agent_test.go for ReAct agent with multimodal inputs
- [X] T089 [US5] Add integration tests in tests/integration/multimodal/agent_test.go for orchestration graphs with multimodal processing
- [X] T090 [US5] Add integration tests in tests/integration/multimodal/agent_test.go for tool integration with multimodal data
- [X] T091 [US5] Add integration tests in tests/integration/multimodal/agent_test.go for agent-to-agent communication with multimodal data
- [X] T092 [US5] Add advanced tests in pkg/multimodal/advanced_test.go for multimodal agent workflows

**Checkpoint**: At this point, all user stories should work independently - users can extend agents with multimodal capabilities

---

## Phase 8: Provider Implementations

**Purpose**: Implement provider adapters for multimodal models

**Note**: Provider implementations can be done in parallel and are independent of each other. They build on the foundational infrastructure.

### OpenAI Provider

- [X] T093 [P] Implement OpenAI provider in pkg/multimodal/providers/openai/provider.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [X] T094 [P] Implement OpenAI-specific config in pkg/multimodal/providers/openai/config.go with API key, model, base URL, timeout settings
- [X] T095 [P] Implement OpenAI capability detection in pkg/multimodal/providers/openai/provider.go (supports text, image, audio, video)
- [X] T096 [P] Register OpenAI provider in pkg/multimodal/providers/openai/init.go using global registry
- [X] T096a [P] Add unit tests for OpenAI provider in pkg/multimodal/providers/openai/provider_test.go (Process, ProcessStream, GetCapabilities, SupportsModality, error handling)

### Google Provider

- [ ] T097 [P] Implement Google provider in pkg/multimodal/providers/google/provider.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [ ] T098 [P] Implement Google-specific config in pkg/multimodal/providers/google/config.go with API key, model, base URL, timeout settings
- [ ] T099 [P] Implement Google capability detection in pkg/multimodal/providers/google/provider.go (supports text, image, audio, video)
- [ ] T100 [P] Register Google provider in pkg/multimodal/providers/google/init.go using global registry
- [ ] T100a [P] Add unit tests for Google provider in pkg/multimodal/providers/google/provider_test.go (Process, ProcessStream, GetCapabilities, SupportsModality, error handling)

### Anthropic Provider

- [ ] T101 [P] Implement Anthropic provider in pkg/multimodal/providers/anthropic/provider.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [ ] T102 [P] Implement Anthropic-specific config in pkg/multimodal/providers/anthropic/config.go with API key, model, base URL, timeout settings
- [ ] T103 [P] Implement Anthropic capability detection in pkg/multimodal/providers/anthropic/provider.go (supports text, image, audio, video)
- [ ] T104 [P] Register Anthropic provider in pkg/multimodal/providers/anthropic/init.go using global registry
- [ ] T104a [P] Add unit tests for Anthropic provider in pkg/multimodal/providers/anthropic/provider_test.go (Process, ProcessStream, GetCapabilities, SupportsModality, error handling)

### xAI Provider

- [ ] T105 [P] Implement xAI provider in pkg/multimodal/providers/xai/provider.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [ ] T106 [P] Implement xAI-specific config in pkg/multimodal/providers/xai/config.go with API key, model, base URL, timeout settings
- [ ] T107 [P] Implement xAI capability detection in pkg/multimodal/providers/xai/provider.go (supports text, image, audio, video)
- [ ] T108 [P] Register xAI provider in pkg/multimodal/providers/xai/init.go using global registry
- [ ] T108a [P] Add unit tests for xAI provider in pkg/multimodal/providers/xai/provider_test.go (Process, ProcessStream, GetCapabilities, SupportsModality, error handling)

### Open-Source Providers (Qwen, Pixtral, Phi, DeepSeek, Gemma)

- [ ] T109 [P] Implement Qwen provider in pkg/multimodal/providers/qwen/provider.go with Process, ProcessStream, GetCapabilities, SupportsModality methods
- [ ] T110 [P] Implement Qwen-specific config in pkg/multimodal/providers/qwen/config.go
- [ ] T111 [P] Register Qwen provider in pkg/multimodal/providers/qwen/init.go using global registry
- [ ] T111a [P] Add unit tests for Qwen provider in pkg/multimodal/providers/qwen/provider_test.go (Process, ProcessStream, GetCapabilities, SupportsModality, error handling)
- [ ] T112 [P] Implement at least one additional open-source provider (Pixtral, Phi, DeepSeek, or Gemma) following the same pattern
- [ ] T112a [P] Add unit tests for the additional open-source provider following the same test pattern

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T113 [P] Update package README.md in pkg/multimodal/README.md with comprehensive usage examples and provider documentation
- [ ] T114 [P] Add godoc comments to all public interfaces and functions in pkg/multimodal/ following framework documentation standards
- [ ] T115 [P] Code cleanup and refactoring across pkg/multimodal/ to ensure consistency with framework patterns
- [ ] T116 [P] Performance optimization in pkg/multimodal/internal/model.go for content routing and processing
- [ ] T117 [P] Performance optimization in pkg/multimodal/internal/normalizer.go for format conversion
- [ ] T118 [P] Add comprehensive error handling edge cases in pkg/multimodal/errors.go for all error scenarios
- [ ] T119 [P] Add benchmarks in pkg/multimodal/advanced_test.go for performance-critical operations (routing, normalization, processing)
- [ ] T120 [P] Validate quickstart.md examples in specs/009-multimodal/quickstart.md work with implementation
- [ ] T121 [P] Add integration tests in tests/integration/multimodal/ for cross-package compatibility (schema, embeddings, vectorstores, agents, orchestration)
- [ ] T122 [P] Verify backward compatibility with text-only workflows in pkg/multimodal/ (ensure no breaking changes) - add explicit integration test in tests/integration/multimodal/backward_compatibility_test.go
- [ ] T123 [P] Add health check support in pkg/multimodal/ if applicable following framework patterns
- [ ] T124 [P] Add comprehensive examples in examples/multimodal/ directory demonstrating all user stories (US1: input processing, US2: reasoning/generation, US3: RAG, US4: streaming, US5: agent extensions)
- [ ] T125 [P] Validate framework package design pattern compliance in pkg/multimodal/ - verify all required files exist (iface/, config.go, metrics.go, errors.go, test_utils.go, advanced_test.go) and follow framework standards

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (US1 → US2 → US3 → US4 → US5)
- **Provider Implementations (Phase 8)**: Can start after Foundational phase, can be done in parallel
- **Polish (Phase 9)**: Depends on all desired user stories and providers being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Depends on US1 for input processing infrastructure
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 and US2 for processing and reasoning
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 for input processing, can be parallel with US2/US3
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - Depends on US1 and US2 for processing and reasoning, can be parallel with US3/US4

### Within Each User Story

- Core interfaces and structs before implementations
- Internal implementations before provider integrations
- Provider integrations before agent/orchestration integrations
- Core implementation before integration tests
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, User Stories 1 and 2 can start (US1 first, then US2)
- User Stories 3, 4, and 5 can run in parallel after US1 and US2 are complete
- All provider implementations marked [P] can run in parallel (Phase 8)
- All Polish tasks marked [P] can run in parallel (Phase 9)
- Models and interfaces within a story marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all foundational interface definitions together:
Task: "Define MultimodalModel interface in pkg/multimodal/iface/model.go"
Task: "Define MultimodalProvider interface in pkg/multimodal/iface/provider.go"
Task: "Define ContentBlock interface in pkg/multimodal/iface/content.go"

# Launch all struct definitions together:
Task: "Define Config struct in pkg/multimodal/config.go"
Task: "Define ModalityCapabilities struct in pkg/multimodal/config.go"
Task: "Define RoutingConfig struct in pkg/multimodal/config.go"
Task: "Define MultimodalInput struct in pkg/multimodal/multimodal.go"
Task: "Define MultimodalOutput struct in pkg/multimodal/multimodal.go"
```

---

## Parallel Example: Provider Implementations

```bash
# Launch all provider implementations together (after foundational phase):
Task: "Implement OpenAI provider in pkg/multimodal/providers/openai/provider.go"
Task: "Implement Google provider in pkg/multimodal/providers/google/provider.go"
Task: "Implement Anthropic provider in pkg/multimodal/providers/anthropic/provider.go"
Task: "Implement xAI provider in pkg/multimodal/providers/xai/provider.go"
Task: "Implement Qwen provider in pkg/multimodal/providers/qwen/provider.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Multimodal Input Processing)
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Add User Story 4 → Test independently → Deploy/Demo
6. Add User Story 5 → Test independently → Deploy/Demo
7. Add Provider Implementations → Test independently → Deploy/Demo
8. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (blocks others)
   - Developer B: Prepares provider implementations
3. Once User Story 1 is done:
   - Developer A: User Story 2
   - Developer B: User Story 3
   - Developer C: User Story 4
   - Developer D: Provider implementations
4. Once User Stories 2-4 are done:
   - Developer A: User Story 5
   - Developer B: Integration tests
   - Developer C: Polish and documentation
5. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Provider implementations can be done incrementally (start with OpenAI, add others as needed)
- Framework patterns must be followed: iface/, config.go, metrics.go, errors.go, test_utils.go, advanced_test.go
- All public methods must include OTEL tracing and metrics
- All errors must use custom error types with codes
- All configuration must use struct tags and validation

---

## Task Summary

- **Total Tasks**: 135
- **Setup Tasks**: 5 (Phase 1)
- **Foundational Tasks**: 21 (Phase 2, includes T017a, T018a, T018b, T022a)
- **User Story 1 Tasks**: 19 (Phase 3)
- **User Story 2 Tasks**: 15 (Phase 4)
- **User Story 3 Tasks**: 11 (Phase 5)
- **User Story 4 Tasks**: 13 (Phase 6)
- **User Story 5 Tasks**: 12 (Phase 7)
- **Provider Implementation Tasks**: 26 (Phase 8, includes provider test tasks T096a, T100a, T104a, T108a, T111a, T112a)
- **Polish Tasks**: 13 (Phase 9, includes T125)

**Suggested MVP Scope**: Phases 1-3 (Setup + Foundational + User Story 1) = 41 tasks

**Parallel Opportunities**: 
- 17 foundational tasks can run in parallel
- 5 provider implementations can run in parallel
- User Stories 3, 4, 5 can run in parallel after US1 and US2
- 12 polish tasks can run in parallel
