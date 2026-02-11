# Beluga AI v2 — Go-Native Agentic Framework

## Project Overview

Beluga AI v2 is a ground-up Go framework for building agentic AI systems. It combines production patterns from Google ADK, OpenAI Agents SDK, LangGraph, ByteDance Eino, and LiveKit into a unified framework with streaming-first design, protocol interoperability (MCP + A2A), and pluggable everything.

**Repository**: `github.com/lookatitude/beluga-ai`
**Language**: Go 1.23+ (uses `iter.Seq2[T, error]` for streaming)
**Module path**: `github.com/lookatitude/beluga-ai`

## Architecture Documents

These are the authoritative references. **Read them before making any architectural decisions**:

- `docs/concepts.md` — Architecture & design decisions (the "why")
- `docs/packages.md` — Package layout & interfaces (the "what")
- `docs/providers.md` — Provider categories, extension guide, and discovery patterns
- `docs/architecture.md` — Full architecture document with extensibility patterns (the "how")

## Go Conventions

### Module & Package Rules
- Module path: `github.com/lookatitude/beluga-ai`
- All imports use this module path
- Package names are lowercase, single-word when possible (e.g., `llm`, `agent`, `tool`, `voice`)
- No `pkg/` prefix — packages live at root: `core/`, `schema/`, `llm/`, `agent/`, etc.
- `internal/` for shared utilities not part of public API
- Provider packages nest under their parent: `llm/providers/openai/`, `rag/vectorstore/providers/pgvector/`

### Code Style
- **Interfaces first**: Define the interface, then implementations. Keep interfaces small (1-4 methods).
- **Functional options**: Use `WithX()` pattern for configuration, not builders or config structs alone.
- **Error handling**: Return `(T, error)`. Use typed errors from `core/errors.go` with `ErrorCode`. Always check `IsRetryable()` for LLM/tool errors.
- **Context propagation**: Every public function's first parameter is `context.Context`. No exceptions.
- **Naming**: `New()` for constructors, `Register()` + `New()` + `List()` for registry pattern, `With()` for options.
- **No global state** beyond `init()` registrations. Registry mutations only in `init()`.
- **Embedding over inheritance**: Compose via struct embedding (e.g., `BaseAgent`), not interface hierarchies.
- **Documentation**: Every exported type/func gets a doc comment. Include usage example in package doc.
- **Test files**: `*_test.go` alongside source. Use `internal/testutil/` mocks for integration tests.

### Streaming Pattern
```go
// Primary streaming: iter.Seq2[T, error] (Go 1.23+)
func (a *Agent) Stream(ctx context.Context, input string) iter.Seq2[Event, error] {
    return func(yield func(Event, error) bool) {
        // produce events
    }
}

// Consumers use range:
for event, err := range agent.Stream(ctx, input) {
    if err != nil { break }
    // handle event
}
```

### Registry Pattern (used by ALL extensible packages)
```go
// Every extensible package follows this exact pattern:
var registry = make(map[string]Factory)

func Register(name string, f Factory) { registry[name] = f }  // called in init()
func New(name string, cfg Config) (Interface, error) { ... }   // factory lookup
func List() []string { ... }                                    // discovery

// Provider registration via init():
func init() {
    llm.Register("openai", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

### Middleware Pattern
```go
// Every extensible interface supports: func(T) T
type Middleware func(ChatModel) ChatModel

func ApplyMiddleware(model ChatModel, mws ...Middleware) ChatModel {
    for i := len(mws) - 1; i >= 0; i-- {
        model = mws[i](model)
    }
    return model
}
```

### Hooks Pattern
```go
type Hooks struct {
    OnStart    func(ctx context.Context, input any) error
    OnEnd      func(ctx context.Context, result any, err error)
    OnError    func(ctx context.Context, err error) error
    // ... all fields optional, nil hooks are skipped
}

func ComposeHooks(hooks ...Hooks) Hooks { ... }
```

## Package Layout

```
beluga-ai/
├── core/         # Foundation: Stream, Runnable, Lifecycle, Errors, Tenant
├── schema/       # Shared types: Message, ContentPart, Tool, Document, Event, Session
├── config/       # Configuration: Load[T], Validate, hot-reload
├── o11y/         # Observability: OTel GenAI conventions, slog, adapters
├── llm/          # LLM abstraction: ChatModel, Router, Structured, Context Manager
│   └── providers/  # openai, anthropic, google, ollama, bedrock, groq, ...
├── tool/         # Tool system: Tool, FuncTool, MCP client, registry
├── memory/       # Memory: Core/Recall/Archival/Graph tiers (MemGPT model)
│   └── stores/     # inmemory, redis, postgres, sqlite, neo4j
├── rag/          # RAG pipeline
│   ├── embedding/  # Embedder interface + providers
│   ├── vectorstore/  # VectorStore interface + providers
│   ├── retriever/  # Retriever interface + strategies (hybrid, CRAG, HyDE)
│   ├── loader/     # DocumentLoader implementations
│   └── splitter/   # TextSplitter implementations
├── agent/        # Agent runtime: BaseAgent, Planner, Executor, Handoffs
│   └── workflow/   # SequentialAgent, ParallelAgent, LoopAgent
├── voice/        # Voice pipeline: Frame-based, STT/TTS/S2S
│   ├── stt/providers/
│   ├── tts/providers/
│   ├── s2s/providers/
│   └── transport/
├── orchestration/  # Chain, Graph, Router, Parallel, Supervisor
├── workflow/     # Durable execution engine
├── protocol/     # MCP server/client, A2A server/client, REST
├── guard/        # Safety: input→output→tool pipeline
├── resilience/   # Circuit breaker, hedge, retry, rate limit
├── cache/        # Exact + semantic + prompt cache
├── hitl/         # Human-in-the-loop: confidence-based approval
├── auth/         # RBAC, ABAC, capability-based security
├── eval/         # Evaluation framework: metrics, runner
├── state/        # Shared agent state with Watch
├── prompt/       # Prompt management & versioning
├── server/       # HTTP framework adapters (gin, fiber, echo, chi, grpc)
└── internal/     # syncutil, jsonutil, testutil (mocks for all interfaces)
```

## Key Design Decisions to Enforce

1. **iter.Seq2 for streaming** — NOT channels. Use `iter.Pull()` when pull semantics needed.
2. **Handoffs are tools** — Agent transfers auto-generate `transfer_to_{name}` tools.
3. **MemGPT 3-tier memory** — Core (always in context), Recall (searchable history), Archival (vector + graph).
4. **Guard pipeline is 3-stage** — Input guards → Output guards → Tool guards. Always.
5. **Own durable execution engine** — NOT Temporal as default. Temporal is a provider option.
6. **Frame-based voice** — FrameProcessor interface, NOT monolithic pipeline.
7. **Registry pattern everywhere** — `Register()` + `New()` + `List()` in every extensible package.
8. **OTel GenAI conventions** — Use `gen_ai.*` attribute namespace.
9. **Hybrid search default** — Vector + BM25 + RRF fusion for retrieval.
10. **Prompt cache optimization** — Static content first ordering via PromptBuilder.

## Testing Requirements

- Unit tests for every exported function
- Table-driven tests preferred
- Use `internal/testutil/` mocks — every interface has a mock
- Integration tests use build tags: `//go:build integration`
- Benchmarks for hot paths (streaming, tool execution, retrieval)
- `go vet`, `staticcheck`, `golangci-lint` must pass

## Dependency Rules

- `core/` and `schema/` — ZERO external deps beyond stdlib + otel
- Provider packages may import provider SDKs
- No circular imports — dependency flows downward through layers
- Prefer stdlib where possible (e.g., `slog` for logging, `net/http` for transports)

## Personas

| Persona | Role | Agent(s) |
|---------|------|----------|
| **Architect** | Oversee architecture, define patterns, create plans, delegate to Team lead | `architect` |
| **Researcher** | Research topics, gather info, return structured findings to Architect | `researcher` |
| **Team lead** | Break plans into tasks, run develop/test/review loops per task | `team-lead` |
| **Developer** | Go + distributed systems + AI; implement and test per framework patterns | `core-implementer`, `llm-implementer`, `agent-implementer`, `rag-implementer`, `voice-implementer`, `tool-implementer`, `protocol-implementer`, `infra-implementer` |
| **Test developer** | Same as Developer; write tests implementations must pass | `test-writer` |
| **Reviewer** | Same as Developer; review code, provide suggestions | `reviewer` |
| **Doc writer** | Write package docs, tutorials, API reference | `doc-writer` |
| **Website developer** | Astro + React + Tailwind specialist; build and adjust the documentation website UI, components, layouts, and styling | `website-developer` |

### Task to Persona Mapping

- Design / architecture / patterns --> **Architect** (`architect`)
- Research / gather info --> **Researcher** (`researcher`)
- Break plan into tasks, manage develop/test/review loop --> **Team lead** (`team-lead`)
- Implement core, schema, config, o11y --> **Developer** (`core-implementer`)
- Implement llm, providers, router --> **Developer** (`llm-implementer`)
- Implement agent, planners, handoffs --> **Developer** (`agent-implementer`)
- Implement rag pipeline --> **Developer** (`rag-implementer`)
- Implement voice pipeline --> **Developer** (`voice-implementer`)
- Implement tool system, MCP client --> **Developer** (`tool-implementer`)
- Implement protocols, server adapters --> **Developer** (`protocol-implementer`)
- Implement guard, resilience, cache, auth, workflow, eval, state, prompt --> **Developer** (`infra-implementer`)
- Write tests / mocks / benchmarks --> **Test developer** (`test-writer`)
- Review code --> **Reviewer** (`reviewer`)
- Write documentation --> **Doc writer** (`doc-writer`)
- Build / adjust website UI, components, layouts, styling --> **Website developer** (`website-developer`)

### Workflow

```
Architect -> (optionally) Researcher -> Architect produces plan -> Team lead breaks into tasks
-> Per task: Developer implements -> Test developer writes tests -> Reviewer reviews
-> If pass: next task. If fail: iterate. -> All tasks done: conclude.
```

## Skills

- `go-framework` — Package structure, registries, lifecycle, functional options
- `go-interfaces` — Interface design, hooks, middleware, extension contract
- `go-testing` — Table-driven tests, stream testing, mocks, benchmarks
- `provider-implementation` — Provider registration, error mapping, streaming, testing
- `streaming-patterns` — iter.Seq2, composition, backpressure, context cancellation
- `doc-writing` — Documentation structure, examples, enterprise standards
- `website-development` — Astro, React, Tailwind CSS; website components, layouts, styling

## Patterns (quick reference)

- **Registry**: `Register()` + `New()` + `List()` in every extensible package. See `go-framework` skill.
- **Middleware**: `func(T) T` — composable, applied outside-in. See `go-interfaces` skill.
- **Hooks**: Optional function fields, nil = skip, composable via `ComposeHooks()`. See `go-interfaces` skill.
- **Streaming**: `iter.Seq2[T, error]` for public API, never channels. See `streaming-patterns` skill.
- **Options**: `WithX()` functional options for configuration. See `go-framework` skill.
- **Errors**: `core.Error` with Op/Code/Message/Err. Check `IsRetryable()`. See CLAUDE.md conventions.
