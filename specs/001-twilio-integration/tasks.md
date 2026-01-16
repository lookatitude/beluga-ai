# Tasks: Twilio API Integration

**Input**: Design documents from `/specs/001-twilio-integration/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Tests are included as they are standard for Beluga AI Framework packages (test_utils.go and advanced_test.go are REQUIRED per beluga-test-standards.mdc).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [Story] Description`

- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions
- **Execution**: All tasks execute sequentially, one at a time

## Path Conventions

- **Go packages**: `pkg/{package_name}/` at repository root
- **Tests**: `tests/integration/` for integration tests
- **Examples**: `examples/voice/twilio/` for example code
- **Documentation**: `docs/providers/twilio.md` for provider documentation

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create directory structure for pkg/voice/providers/twilio/
- [X] T002 Create directory structure for pkg/messaging/ with iface/, internal/, providers/twilio/ subdirectories
- [X] T003 Add Twilio Go SDK v1.29.1 dependency to go.mod
- [X] T004 Create examples/voice/twilio/ directory structure
- [X] T005 Create docs/providers/twilio.md documentation file

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. All files MUST follow Beluga AI Framework constitution requirements (package structure, interfaces, OTEL observability, error handling, configuration, testing per beluga-design-patterns.mdc and beluga-test-standards.mdc).

- [X] T006 Create pkg/messaging/iface/backend.go with ConversationalBackend interface definition
- [X] T007 Create pkg/messaging/iface/types.go with Conversation, Message, Participant, WebhookEvent, HealthStatus types
- [X] T008 Create pkg/messaging/config.go with Config struct, validation, and functional options
- [X] T009 Create pkg/messaging/errors.go with custom error types (Op/Err/Code pattern) and error codes
- [X] T010 Create pkg/messaging/metrics.go with OTEL metrics implementation (counters, histograms) covering FR-034 (metrics for call quality, latency, success rates), FR-035 (tracing for telephony operations), FR-036 (structured logging with appropriate detail levels)
- [X] T011 Create pkg/messaging/registry.go with global registry pattern for provider registration (verify structure matches plan.md: pkg/messaging/registry.go at package root level, following pkg/llms and pkg/embeddings patterns)
- [X] T012 Create pkg/messaging/messaging.go with factory functions and main package interface
- [X] T013 Create pkg/messaging/test_utils.go with AdvancedMockMessaging and testing utilities
- [X] T014 Create pkg/messaging/advanced_test.go with table-driven tests, concurrency tests, benchmarks
- [X] T015 Create pkg/messaging/README.md with complete package documentation (MUST include: package overview, features, quick start, configuration, usage examples, integration patterns, API reference, observability, error handling, testing, and related packages per beluga-design-patterns.mdc)
- [X] T016 Create pkg/voice/providers/twilio/config.go with TwilioConfig struct extending VoiceBackend config
- [X] T017 Create pkg/voice/providers/twilio/errors.go with Twilio-specific error types and error codes
- [X] T018 Create pkg/voice/providers/twilio/metrics.go with OTEL metrics for voice operations covering FR-034 (metrics for call quality, latency, success rates), FR-035 (tracing for telephony operations), FR-036 (structured logging with appropriate detail levels)
- [X] T019 Create pkg/voice/providers/twilio/test_utils.go with AdvancedMockTwilioVoice and testing utilities
- [X] T020 Create pkg/voice/providers/twilio/advanced_test.go with table-driven tests, concurrency tests (including SC-003: 100 concurrent calls), benchmarks
- [X] T020a [Foundation] Design multi-account configuration support in pkg/voice/providers/twilio/config.go and pkg/messaging/config.go for FR-041 (support multiple Twilio accounts/configurations via provider instances with separate credentials)

**Checkpoint**: Foundation ready - user story implementation can now begin sequentially

---

## Phase 3: User Story 1 - Voice-Enabled Interactive Agent (Priority: P1) 🎯 MVP

**Goal**: Enable real-time voice interactions with AI agents via phone calls. System answers calls, processes speech, generates responses, converts to speech, and maintains conversation context.

**Independent Test**: Make a phone call to configured number, speak to agent, verify agent responds appropriately with <2s latency. Test delivers working voice agent handling real phone calls.

### Implementation for User Story 1

- [X] T021 [US1] Create pkg/voice/providers/twilio/provider.go with TwilioProvider struct implementing VoiceBackend interface
- [X] T022 [US1] Implement Start() method in pkg/voice/providers/twilio/provider.go (validate config, initialize Twilio client, verify connectivity)
- [X] T023 [US1] Implement Stop() method in pkg/voice/providers/twilio/provider.go (graceful shutdown, complete calls, close streams)
- [X] T024 [US1] Implement CreateSession() method in pkg/voice/providers/twilio/provider.go (create Twilio Call resource, establish WebSocket stream)
- [X] T025 [US1] Implement GetSession() method in pkg/voice/providers/twilio/provider.go (lookup session by ID)
- [X] T026 [US1] Implement ListSessions() method in pkg/voice/providers/twilio/provider.go (return active sessions)
- [X] T027 [US1] Implement CloseSession() method in pkg/voice/providers/twilio/provider.go (update/delete call, close stream, cleanup). Handle edge case: call drops mid-conversation (save partial conversation state, allow resumption if customer calls back per spec.md edge cases)
- [X] T028 [US1] Implement HealthCheck() method in pkg/voice/providers/twilio/provider.go (verify API connectivity, check sessions)
- [X] T029 [US1] Implement GetConnectionState() method in pkg/voice/providers/twilio/provider.go (return connection state)
- [X] T030 [US1] Implement GetActiveSessionCount() method in pkg/voice/providers/twilio/provider.go (return active session count)
- [X] T031 [US1] Implement GetConfig() and UpdateConfig() methods in pkg/voice/providers/twilio/provider.go
- [X] T032 [US1] Create pkg/voice/providers/twilio/streaming.go with StreamAudio() method and WebSocket streaming implementation
- [X] T033 [US1] Implement bidirectional audio streaming in pkg/voice/providers/twilio/streaming.go (mu-law codec, WebSocket handling). Handle edge case: network failures during call (detect failures, attempt reconnection, provide fallback behavior per spec.md edge cases)
- [X] T034 [US1] Create pkg/voice/providers/twilio/webhook.go with HandleInboundCall() method and webhook signature validation
- [X] T035 [US1] Implement webhook parsing for inbound calls in pkg/voice/providers/twilio/webhook.go
- [X] T036 [US1] Create pkg/voice/providers/twilio/session.go with VoiceSession implementation managing call state and agent integration
- [X] T037 [US1] Integrate STT/TTS providers from pkg/voice in pkg/voice/providers/twilio/session.go for speech processing
- [X] T038 [US1] Integrate pkg/agents in pkg/voice/providers/twilio/session.go for agent responses during calls
- [X] T039 [US1] Create pkg/voice/providers/twilio/init.go with provider auto-registration in global registry
- [X] T040 [US1] Create pkg/voice/providers/twilio/provider_test.go with unit tests for provider methods
- [X] T041 [US1] Create pkg/voice/providers/twilio/streaming_test.go with tests for WebSocket streaming
- [X] T042 [US1] Create pkg/voice/providers/twilio/webhook_test.go with tests for webhook handling
- [X] T043 [US1] Create examples/voice/twilio/voice_agent/main.go with basic voice agent example
- [X] T044 [US1] Create tests/integration/voice_twilio_test.go with integration tests for voice agent functionality
- [X] T044a [US1] Implement latency measurement and validation in tests/integration/voice_twilio_test.go to verify FR-009 (<2s from speech completion to agent audio response start, measured from customer speech end to agent audio begin)

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently. Voice calls can be made, agent responds with <2s latency, context is maintained.

---

## Phase 4: User Story 2 - Multi-Channel Messaging Agent with Memory (Priority: P1)

**Goal**: Enable SMS/WhatsApp messaging agents that remember previous conversations, maintain context across messages, and provide consistent responses across channels.

**Independent Test**: Send SMS/WhatsApp messages to configured number, have multi-turn conversation, verify agent remembers previous messages and maintains context. Test delivers working messaging agent with persistent memory.

### Implementation for User Story 2

- [X] T045 [US2] Create pkg/messaging/providers/twilio/provider.go with TwilioProvider struct implementing ConversationalBackend interface
- [X] T046 [US2] Implement Start() method in pkg/messaging/providers/twilio/provider.go (validate config, initialize Twilio client, verify connectivity)
- [X] T047 [US2] Implement Stop() method in pkg/messaging/providers/twilio/provider.go (graceful shutdown, complete conversations, close channels)
- [X] T048 [US2] Implement CreateConversation() method in pkg/messaging/providers/twilio/provider.go (create Twilio Conversation resource, setup session)
- [X] T049 [US2] Implement GetConversation() method in pkg/messaging/providers/twilio/provider.go (fetch conversation from Twilio API)
- [X] T050 [US2] Implement ListConversations() method in pkg/messaging/providers/twilio/provider.go (fetch conversations with pagination)
- [X] T051 [US2] Implement CloseConversation() method in pkg/messaging/providers/twilio/provider.go (update state to closed, cleanup)
- [X] T052 [US2] Implement SendMessage() method in pkg/messaging/providers/twilio/provider.go (create Message resource, support text and media). Handle edge cases: unsupported media types (graceful error messages per spec.md), multiple simultaneous messages (process in order, maintain conversation coherence per spec.md)
- [X] T053 [US2] Implement ReceiveMessages() method in pkg/messaging/providers/twilio/provider.go (create message channel, listen for webhook events)
- [X] T054 [US2] Implement AddParticipant() method in pkg/messaging/providers/twilio/provider.go (create Participant resource, setup bindings)
- [X] T055 [US2] Implement RemoveParticipant() method in pkg/messaging/providers/twilio/provider.go (delete Participant resource)
- [X] T056 [US2] Implement HealthCheck() method in pkg/messaging/providers/twilio/provider.go (verify API connectivity, check conversations)
- [X] T057 [US2] Implement GetConfig() method in pkg/messaging/providers/twilio/provider.go
- [X] T058 [US2] Create pkg/messaging/providers/twilio/webhook.go with HandleWebhook() method and webhook signature validation
- [X] T059 [US2] Implement webhook event parsing for Conversations API in pkg/messaging/providers/twilio/webhook.go (message.added, conversation.created, etc.). Handle edge case: webhook delivery failures (retry with exponential backoff, log failures appropriately per spec.md edge cases)
- [X] T060 [US2] Create pkg/messaging/providers/twilio/session.go with MessagingSession implementation managing conversation state
- [X] T061 [US2] Integrate pkg/agents in pkg/messaging/providers/twilio/session.go for agent message processing
- [X] T062 [US2] Integrate pkg/memory/VectorStoreMemory in pkg/messaging/providers/twilio/session.go for conversation history persistence. Handle edge case: very long conversations exceeding memory limits (summarize or truncate appropriately while maintaining key context per spec.md edge cases)
- [X] T063 [US2] Implement multi-channel context preservation in pkg/messaging/providers/twilio/session.go (link sessions by participant identity)
- [X] T064 [US2] Create pkg/messaging/providers/twilio/config.go with TwilioConfig struct for Conversations API
- [X] T065 [US2] Create pkg/messaging/providers/twilio/init.go with provider auto-registration in global registry
- [X] T066 [US2] Create pkg/messaging/providers/twilio/provider_test.go with unit tests for provider methods
- [X] T067 [US2] Create pkg/messaging/providers/twilio/webhook_test.go with tests for webhook handling
- [X] T068 [US2] Create examples/voice/twilio/messaging_agent/main.go with basic messaging agent example
- [X] T069 [US2] Create tests/integration/messaging_twilio_test.go with integration tests for messaging agent with memory

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently. Messaging agent handles SMS/WhatsApp, maintains context across sessions, preserves memory across channels.

---

## Phase 5: User Story 3 - Event-Driven Workflow Orchestration (Priority: P2)

**Goal**: Enable complex workflows triggered by telephony events (call answered, message received) that execute different actions based on event types.

**Independent Test**: Configure webhook handlers for telephony events, trigger events (calls, messages), verify workflows execute correctly. Test delivers event-driven automation capabilities.

### Implementation for User Story 3

- [X] T070 [US3] Create pkg/voice/providers/twilio/orchestration.go with webhook-to-workflow mapping for voice events
- [X] T071 [US3] Integrate pkg/orchestration in pkg/voice/providers/twilio/orchestration.go for DAG workflow creation (Inbound → Agent → Stream)
- [X] T072 [US3] Implement workflow trigger from call.answered event in pkg/voice/providers/twilio/orchestration.go
- [X] T073 [US3] Create pkg/messaging/providers/twilio/orchestration.go with webhook-to-workflow mapping for messaging events
- [X] T074 [US3] Integrate pkg/orchestration in pkg/messaging/providers/twilio/orchestration.go for message processing workflows
- [X] T075 [US3] Implement workflow trigger from message.added event in pkg/messaging/providers/twilio/orchestration.go
- [X] T076 [US3] Create pkg/voice/providers/twilio/webhook_handlers.go with handlers for call.status, stream.event, transcription.completed events
- [X] T077 [US3] Create pkg/messaging/providers/twilio/webhook_handlers.go with handlers for conversation.*, message.*, participant.*, typing.* events
- [X] T078 [US3] Implement event-to-workflow routing logic in pkg/voice/providers/twilio/webhook_handlers.go
- [X] T079 [US3] Implement event-to-workflow routing logic in pkg/messaging/providers/twilio/webhook_handlers.go
- [X] T080 [US3] Create examples/voice/twilio/webhook_server/main.go with basic webhook endpoint setup example (minimal HTTP server with signature validation)
- [X] T081 [US3] Create tests/integration/orchestration_twilio_test.go with integration tests for event-driven workflows
- [X] T082 [US3] Add orchestration integration tests to pkg/voice/providers/twilio/advanced_test.go
- [X] T083 [US3] Add orchestration integration tests to pkg/messaging/providers/twilio/advanced_test.go

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently. Webhook events trigger orchestration workflows correctly.

---

## Phase 6: User Story 4 - Transcription and Multimodal RAG Integration (Priority: P2)

**Goal**: Store call transcriptions in searchable knowledge base and enable RAG retrieval of relevant information from past conversations.

**Independent Test**: Make calls generating transcriptions, store transcriptions, query knowledge base to retrieve relevant information. Test delivers searchable conversation history.

### Implementation for User Story 4

- [X] T084 [US4] Create pkg/voice/providers/twilio/transcription.go with transcription resource management (FR-030: store call transcriptions for later retrieval)
- [X] T085 [US4] Implement transcription retrieval from Twilio API in pkg/voice/providers/twilio/transcription.go (FR-030: store call transcriptions)
- [X] T086 [US4] Implement transcription storage in pkg/vectorstores in pkg/voice/providers/twilio/transcription.go (create Document, store with metadata) (FR-030: store call transcriptions for later retrieval)
- [X] T087 [US4] Integrate pkg/embeddings in pkg/voice/providers/twilio/transcription.go for embedding generation (FR-032: RAG integration, FR-033: multimodal RAG)
- [X] T088 [US4] Implement embedding generation for transcriptions in pkg/voice/providers/twilio/transcription.go (FR-032: RAG integration, FR-033: multimodal RAG)
- [X] T089 [US4] Integrate pkg/retrievers in pkg/voice/providers/twilio/transcription.go for semantic search (FR-031: enable searching stored transcriptions, FR-032: RAG integration)
- [X] T090 [US4] Implement transcription search functionality in pkg/voice/providers/twilio/transcription.go (query vector store, retrieve relevant transcriptions) (FR-031: enable searching stored transcriptions to find relevant conversation content, FR-032: RAG integration)
- [X] T091 [US4] Integrate transcription RAG in pkg/voice/providers/twilio/session.go (retrieve relevant transcriptions for agent context) (FR-032: integrate transcriptions with retrieval-augmented generation capabilities)
- [X] T092 [US4] Implement multimodal RAG integration in pkg/voice/providers/twilio/transcription.go (combine transcriptions with other data sources) (FR-033: support multimodal RAG where transcriptions can be combined with other data sources)
- [X] T093 [US4] Create pkg/voice/providers/twilio/transcription_test.go with tests for transcription storage and retrieval
- [X] T094 [US4] Create tests/integration/rag_twilio_test.go with integration tests for transcription RAG
- [X] T095 [US4] Add RAG integration example to examples/voice/twilio/voice_agent/main.go
- [X] T096 [US4] Add transcription processing to webhook handlers in pkg/voice/providers/twilio/webhook_handlers.go (transcription.completed event)

**Checkpoint**: At this point, all user stories should be independently functional. Transcriptions are stored, searchable, and used for RAG in agent responses.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T097 Update docs/providers/twilio.md with complete provider documentation
- [X] T098 Add comprehensive error handling and retry logic for transient failures across all providers (FR-037: handle errors gracefully with appropriate error messages and recovery). Include edge case handling: call drops (save partial state, allow resumption), network failures (detect, reconnect, fallback), rate limits (queue requests, clear errors), unsupported media (graceful error messages), long conversations (summarize/truncate while maintaining key context), webhook delivery failures (retry with exponential backoff)
- [X] T099 Implement connection pooling and resource management optimizations
- [X] T100 Add performance optimizations and latency measurement for requirements (FR-009: <2s voice from speech completion to audio response start, SC-003: 100 concurrent calls without degradation, SC-005: <5s messaging, SC-007: <1s webhooks). Include latency metrics in OTEL, concurrent call load testing, and validation in tests.
- [X] T101 Add comprehensive OTEL tracing spans to all public methods in both providers (FR-035: tracing for telephony operations to enable debugging, with span attributes for call quality, latency, success rates)
- [X] T102 Add structured logging with OTEL context (trace IDs, span IDs) throughout (FR-036: log all telephony events with appropriate detail levels - DEBUG for detailed operations, INFO for events, WARN for errors, ERROR for failures)
- [X] T103 Implement rate limit handling and backoff strategies for Twilio API (handle rate limits: respect rate limits, queue requests appropriately, provide clear error messages per spec.md edge cases)
- [X] T104 Add health check endpoints and monitoring integration
- [X] T105 Enhance examples/voice/twilio/webhook_server/main.go with complete webhook server example (production-ready with error handling, orchestration integration, observability, and all event types)
- [X] T106 Validate quickstart.md examples work correctly
- [X] T107 Run all quality checks (make fmt-check, make lint, make vet, make security, make test)
- [X] T108 Update pkg/voice/providers/twilio/README.md with provider-specific documentation
- [X] T109 Update pkg/messaging/README.md with complete package documentation
- [X] T110 Add integration tests for cross-package compatibility (voice + agents, messaging + memory, etc.)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories execute sequentially in priority order (P1 → P2)
  - Complete one story before starting the next
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories (independent messaging)
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 and US2 for webhook events
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 for transcriptions

### Within Each User Story

- Core provider implementation before extensions
- Webhook handling after core provider methods
- Session management after provider setup
- Integration with other packages (agents, memory) after core functionality
- Tests after implementation
- Examples after implementation and tests

### Execution Order

- **All tasks execute sequentially**: One task at a time, in the order listed
- **Complete each task fully** before moving to the next
- **No parallel execution**: Agents implement one step at a time
- **Follow task order**: Tasks are ordered to respect dependencies and logical flow

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Voice-Enabled Agent)
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Add User Story 4 → Test independently → Deploy/Demo
6. Each story adds value without breaking previous stories

### Sequential Execution Strategy

All tasks execute one at a time in order:

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T020)
3. Complete Phase 3: User Story 1 (T021-T044)
4. Complete Phase 4: User Story 2 (T045-T069)
5. Complete Phase 5: User Story 3 (T070-T083)
6. Complete Phase 6: User Story 4 (T084-T096)
7. Complete Phase 7: Polish (T097-T110)

---

## Notes

- **Sequential Execution**: All tasks execute one at a time, in order
- **Complete Each Task**: Finish each task fully before moving to the next
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- All packages MUST follow Beluga AI Framework patterns (beluga-design-patterns.mdc)
- All packages MUST include test_utils.go and advanced_test.go (beluga-test-standards.mdc)
- All code MUST pass quality checks (beluga-quality-standards.mdc)
- OTEL observability is MANDATORY for all packages
- Error handling MUST follow Op/Err/Code pattern
- Configuration MUST use mapstructure, yaml, env, validate tags
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, skipping tasks, cross-story dependencies that break independence

---

## Coverage Verification

### Observability Requirements (FR-034-FR-037) ✅
- **FR-034 (Metrics)**: Covered by T010, T018 (OTEL metrics for call quality, latency, success rates)
- **FR-035 (Tracing)**: Covered by T101 (OTEL tracing spans for all public methods)
- **FR-036 (Logging)**: Covered by T102 (structured logging with OTEL context, appropriate detail levels)
- **FR-037 (Error Handling)**: Covered by T009, T017, T098 (comprehensive error handling with graceful recovery)

### Edge Case Handling ✅
All edge cases from spec.md are explicitly covered:
- **Call drops mid-conversation**: T027 (save partial state, allow resumption)
- **Network failures during call**: T033 (detect, reconnect, fallback)
- **Multiple simultaneous messages**: T052 (process in order, maintain coherence)
- **Rate limits**: T098, T103 (queue requests, clear errors, backoff strategies)
- **Unsupported media types**: T052 (graceful error messages)
- **Long conversations exceeding memory**: T062 (summarize/truncate while maintaining context)
- **Webhook delivery failures**: T059, T098 (retry with exponential backoff, log failures)

### Registry Structure Verification ✅
- **pkg/messaging/registry.go**: T011 creates registry at package root level, matching plan.md structure and following pkg/llms/pkg/embeddings patterns
- **pkg/voice/providers/twilio/init.go**: T039 handles provider auto-registration in existing voice backend registry

### Transcription/RAG Integration (FR-030-FR-033) ✅
- **FR-030 (Store transcriptions)**: Covered by T084, T085, T086 (transcription resource management, retrieval, storage in vectorstores)
- **FR-031 (Search transcriptions)**: Covered by T089, T090 (retriever integration, semantic search functionality)
- **FR-032 (RAG integration)**: Covered by T087, T088, T091 (embedding generation, RAG integration in session)
- **FR-033 (Multimodal RAG)**: Covered by T092 (multimodal RAG combining transcriptions with other data sources)

---

## Task Summary

- **Total Tasks**: 112
- **Phase 1 (Setup)**: 5 tasks
- **Phase 2 (Foundational)**: 16 tasks (includes T020a for multi-account support)
- **Phase 3 (User Story 1 - Voice)**: 25 tasks (includes T044a for latency measurement)
- **Phase 4 (User Story 2 - Messaging)**: 25 tasks
- **Phase 5 (User Story 3 - Orchestration)**: 14 tasks
- **Phase 6 (User Story 4 - RAG)**: 13 tasks
- **Phase 7 (Polish)**: 14 tasks

### Execution Strategy

- **Sequential Execution**: All 112 tasks execute one at a time, in order
- **No Parallel Execution**: Agents implement one step at a time
- **Task Order**: Tasks are ordered to respect dependencies and logical flow
- **Completion Required**: Each task must be fully completed before proceeding

### Independent Test Criteria

- **User Story 1**: Make phone call, verify agent responds with <2s latency, context maintained
- **User Story 2**: Send SMS/WhatsApp messages, verify agent remembers context, multi-channel support
- **User Story 3**: Configure webhooks, trigger events, verify workflows execute
- **User Story 4**: Make calls, store transcriptions, query knowledge base, verify RAG retrieval

### Suggested MVP Scope

**MVP = Phase 1 + Phase 2 + Phase 3 (User Story 1)**
- Total: 46 tasks (includes T020a and T044a)
- Delivers: Working voice-enabled agent that can handle phone calls with AI responses
- Can be tested independently and deployed
