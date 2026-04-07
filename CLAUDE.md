# Beluga AI v2 — Go-Native Agentic Framework

**Repository**: `github.com/lookatitude/beluga-ai` | **Language**: Go 1.23+ | **Streaming**: `iter.Seq2[T, error]`

## Architecture References

Read these before architectural decisions:
- `docs/concepts.md` — Design decisions (the "why")
- `docs/packages.md` — Package layout & interfaces (the "what")
- `docs/providers.md` — Provider categories & extension guide
- `docs/architecture.md` — Full architecture with extensibility patterns (the "how")

## Team

| Agent | Role | When to Use |
|-------|------|-------------|
| `architect` | Design, plan, define acceptance criteria | New features, packages, design decisions |
| `researcher` | Investigate topics, return evidence | When Architect needs research before designing |
| `developer` | Code + tests for all packages | Any implementation task |
| `security-reviewer` | Thorough security review | After Developer completes code (2 clean passes required) |
| `qa-engineer` | Validate against acceptance criteria | After Security Review passes |
| `doc-writer` | Package docs, tutorials, API reference | After implementation is approved |

## Workflows

### Feature / Complex Task (use `/plan-feature` then `/implement`)

```
Architect ──→ defines research topics ──→ Researcher investigates each topic
    ↑                                            │
    └──── returns structured findings ───────────┘
    │
    ▼
Architect ──→ designs + plans with acceptance criteria
    │
    ▼
Developer ──→ codes + writes tests ──→ Security Reviewer
    ↑                                       │
    │   ┌── if issues found ────────────────┘
    │   │
    └───┘ (loop until 2 consecutive clean passes)
                                            │
                                            ▼
                                     QA Engineer validates
                                     against acceptance criteria
                                            │
                                     ┌──────┴──────┐
                                     │ PASS → Done  │
                                     │ FAIL → Dev   │
                                     └─────────────┘
```

### Simple Bug Fix / Small Change

Skip Architect and Researcher. Developer fixes + tests → Security Review (2 clean passes) → QA validates.

### Documentation Only

Doc Writer handles directly. No security review needed.

### Code Review (`/review`)

Security Reviewer runs full checklist. Must get 2 consecutive clean passes.

## Go Conventions (Quick Reference)

- **Module**: `github.com/lookatitude/beluga-ai`
- **Packages**: lowercase, single-word, at root (no `pkg/` prefix)
- **Interfaces**: 1-4 methods, define before implementing
- **Config**: `WithX()` functional options
- **Errors**: `(T, error)`, typed errors from `core/errors.go` with `ErrorCode`, check `IsRetryable()`
- **Context**: `context.Context` always first parameter
- **Streaming**: `iter.Seq2[T, error]` — never channels in public API
- **Registry**: `Register()` + `New()` + `List()` in every extensible package (called in `init()`)
- **Middleware**: `func(T) T`, applied outside-in
- **Hooks**: Optional function fields, nil = skip, composable via `ComposeHooks()`
- **Tests**: `*_test.go` alongside source, table-driven, integration tests use `//go:build integration`
- **Deps**: Zero external deps in `core/` and `schema/` beyond stdlib + otel. No circular imports.

## Key Design Decisions

1. `iter.Seq2` for streaming — NOT channels
2. Handoffs are tools — auto-generate `transfer_to_{name}`
3. MemGPT 3-tier memory — Core/Recall/Archival + graph
4. Guard pipeline is 3-stage — Input → Output → Tool
5. Own durable execution engine — Temporal is a provider option
6. Frame-based voice — FrameProcessor interface
7. Registry pattern everywhere
8. OTel GenAI conventions — `gen_ai.*` namespace
9. Hybrid search default — Vector + BM25 + RRF fusion
10. Prompt cache optimization — static content first

## Package Layout

```
beluga-ai/
├── core/           # Stream, Runnable, Lifecycle, Errors, Tenant
├── schema/         # Message, ContentPart, Tool, Document, Event, Session
├── config/         # Load[T], Validate, hot-reload
├── o11y/           # OTel GenAI, slog, adapters
├── llm/            # ChatModel, Router, Structured, Context Manager
│   └── providers/
├── tool/           # Tool, FuncTool, MCP client, registry
├── memory/         # 3-tier + graph (MemGPT model)
│   └── stores/
├── rag/            # embedding/, vectorstore/, retriever/, loader/, splitter/
├── agent/          # BaseAgent, Planner, Executor, Handoffs
│   └── workflow/   # Sequential, Parallel, Loop agents
├── voice/          # Frame-based pipeline, STT/TTS/S2S, transport
├── orchestration/  # Chain, Graph, Router, Parallel, Supervisor
├── workflow/       # Durable execution engine
├── protocol/       # MCP server/client, A2A server/client, REST
├── guard/          # Input→Output→Tool safety pipeline
├── resilience/     # Circuit breaker, hedge, retry, rate limit
├── cache/          # Exact + semantic + prompt cache
├── hitl/           # Human-in-the-loop
├── auth/           # RBAC, ABAC, capability-based
├── eval/           # Metrics, runner
├── state/          # Shared agent state with Watch
├── prompt/         # Prompt management & versioning
├── server/         # HTTP adapters (gin, fiber, echo, chi, grpc)
└── internal/       # syncutil, jsonutil, testutil
```

## Skills

- `go-framework` — Package structure, registries, lifecycle, functional options
- `go-interfaces` — Interface design, hooks, middleware, extension contracts
- `go-testing` — Table-driven tests, stream testing, mocks, benchmarks
- `provider-implementation` — Provider registration, error mapping, streaming
- `streaming-patterns` — iter.Seq2, composition, backpressure, context cancellation
- `doc-writing` — Documentation structure, examples, enterprise standards
- `website-development` — Astro + React + Tailwind for the docs site
