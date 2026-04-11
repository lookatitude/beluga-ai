# Package Architecture Map

## Core Packages

### core
**Purpose:** Foundational types (Stream, Error, ErrorCode) and streaming primitives.

**Key Types:**
- Stream[T] — iter.Seq2-based range-over-func type for chunk streaming
- Error — structured error with Op, Code, Message, Err fields
- ErrorCode — enum for error classification (rate_limit, timeout, auth_error, etc.)

**Registry:** None (types exported, no registration)

**Dependencies:** Standard library only (context, errors, iter)

**Canonical File:** `core/stream.go`, `core/errors.go`

**Test Coverage:** core/stream_test.go (Stream, MapStream), core/errors_test.go

---

### tool
**Purpose:** Tool interface, execution registry, middleware composition, and hooks.

**Key Types:**
- Tool — interface with Execute, Name, Description, InputSchema
- Registry — thread-safe map-based tool lookup
- Middleware — func(Tool) Tool composition pattern
- Hooks — optional OnStart, OnEnd, OnError func fields with ComposeHooks

**Registry:** tool.Registry (instance-based, not global)

**Dependencies:** context, core (for error handling)

**Canonical File:** `tool/tool.go`, `tool/registry.go`, `tool/middleware.go`, `tool/hooks.go`

**Test Coverage:** tool/tool_test.go (table-driven), tool/middleware_test.go (retry/timeout)

---

### llm
**Purpose:** LLM provider abstraction, registration, and client factory.

**Key Types:**
- Provider — interface for LLM implementations
- Client — LLM chat/completion client
- Factory — func(config) (Client, error) provider factory

**Registry:** Global llm.Register() for provider registration via init()

**Dependencies:** context, core (error handling)

**Canonical File:** `llm/registry.go` (global), `llm/client.go`

**Test Coverage:** llm/client_test.go, llm/providers/*/provider_test.go

---

### guard
**Purpose:** Security guard pipeline (Input → Output → Tool stages) with composable decisions.

**Key Types:**
- Guard — interface with InspectInput, InspectOutput, InspectTool
- GuardResult — Decision (Allow/Review/Block) + Reason
- GuardInput/GuardOutput/GuardTool — stage-specific input types

**Registry:** None (guards passed explicitly or composed)

**Dependencies:** context

**Canonical File:** `guard/guard.go`

**Test Coverage:** guard/guard_test.go (stage ordering, decision blocking)

---

### o11y
**Purpose:** OpenTelemetry instrumentation with GenAI semantic conventions (v1.37+).

**Key Types:**
- Span — wraps otel trace.Span with SetAttributes, RecordError, SetStatus
- Attrs — map[string]any for flexible attribute assignment
- StatusCode — OK/Error for span status mapping

**Registry:** None (global tracer via otel.SetTracerProvider)

**Dependencies:** go.opentelemetry.io/otel, context

**Canonical File:** `o11y/tracer.go`

**Test Coverage:** o11y/tracer_test.go (attribute conversion, status mapping)

---

### memory
**Purpose:** Conversation memory abstraction with MessageStore implementations.

**Key Types:**
- MessageStore — interface for persisting/retrieving conversation messages
- Message — conversation turn with role, content, metadata

**Registry:** None (stores passed explicitly, no global registry)

**Dependencies:** context

**Canonical File:** `memory/store.go`, `memory/stores/inmemory/inmemory.go`

**Test Coverage:** memory/stores/inmemory/inmemory_test.go

---

### protocol
**Purpose:** Wire protocol definitions and message marshaling.

**Key Types:**
- Request/Response — protocol message types
- Marshaler — interface for serialization

**Registry:** None

**Dependencies:** Varies by protocol (gRPC, HTTP, etc.)

**Canonical File:** `protocol/protocol.go`

**Test Coverage:** protocol/protocol_test.go

